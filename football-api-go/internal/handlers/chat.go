package handlers

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thiagotn/football-manager/football-api-go/internal/apierror"
	"github.com/thiagotn/football-manager/football-api-go/internal/db"
	"github.com/thiagotn/football-manager/football-api-go/internal/middleware"
)

const anthropicMessagesURL = "https://api.anthropic.com/v1/messages"

const chatSystemPrompt = `Você é o assistente oficial do rachao.app, uma plataforma para organização de peladas e rachões no Brasil.

Seu papel é ajudar usuários com dúvidas sobre funcionalidades, fluxos, pagamentos, convites, confirmações de presença e configurações do app.

## Regras gerais
- Responda APENAS sobre o rachao.app e suas funcionalidades.
- Se perguntado sobre qualquer outro assunto, decline educadamente e redirecione para tópicos do app.
- Seja direto, amigável e use linguagem informal brasileira.
- Nunca invente funcionalidades que não existem no app.
- NUNCA peça ao usuário identificadores técnicos (IDs, hashes, UUIDs). Sempre use as ferramentas para descobri-los.`

type chatHandler struct {
	pool          *pgxpool.Pool
	anthropicKey  string
	llmModel      string
	chatRateLimit int
}

func NewChatHandler(pool *pgxpool.Pool, anthropicKey, llmModel string, chatRateLimit int) *chatHandler {
	return &chatHandler{
		pool:          pool,
		anthropicKey:  anthropicKey,
		llmModel:      llmModel,
		chatRateLimit: chatRateLimit,
	}
}

// ── Request types ─────────────────────────────────────────────────────────────

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Messages []chatMessage `json:"messages"`
}

type chatAccessUpdate struct {
	ChatEnabled bool `json:"chat_enabled"`
}

// ── Public routes ─────────────────────────────────────────────────────────────

func (h *chatHandler) ListChatUsers(w http.ResponseWriter, r *http.Request) {
	h.listChatUsers(w, r)
}

func (h *chatHandler) UpdateChatAccess(w http.ResponseWriter, r *http.Request) {
	h.updateChatAccess(w, r)
}

// ── Handlers ──────────────────────────────────────────────────────────────────

func (h *chatHandler) Chat(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())

	if !player.ChatEnabled {
		renderError(w, apierror.Forbidden("chat access not enabled for this user"))
		return
	}

	allowed, err := db.CheckAndIncrementChatRateLimit(r.Context(), h.pool, player.ID, h.chatRateLimit)
	if err != nil {
		renderError(w, err)
		return
	}
	if !allowed {
		renderError(w, apierror.TooManyRequests())
		return
	}

	var req chatRequest
	if err := decodeJSON(r, &req); err != nil {
		renderError(w, err)
		return
	}
	if len(req.Messages) == 0 {
		renderError(w, apierror.Unprocessable("messages must not be empty"))
		return
	}

	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	token = strings.TrimSpace(token)

	brazilOffset := -3 * time.Hour
	todayStr := time.Now().UTC().Add(-brazilOffset).Format("02/01/2006")
	systemPrompt := fmt.Sprintf("Hoje é %s (horário de Brasília).\n\n%s", todayStr, chatSystemPrompt)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	flusher, canFlush := w.(http.Flusher)

	emit := func(data string) {
		fmt.Fprintf(w, "data: %s\n\n", data) //nolint:errcheck
		if canFlush {
			flusher.Flush()
		}
	}

	emitError := func(msg string) {
		b, _ := json.Marshal(map[string]string{"error": msg})
		emit(string(b))
	}

	if h.anthropicKey == "" {
		emitError("assistente não configurado. Contate o administrador.")
		return
	}

	reqBody := map[string]any{
		"model":      h.llmModel,
		"max_tokens": 1024,
		"stream":     true,
		"system":     systemPrompt,
		"messages":   req.Messages,
		"betas":      []string{"mcp-client-2025-04-04"},
		"mcp_servers": []map[string]any{
			{
				"type":                "url",
				"url":                 "https://mcp.rachao.app/mcp",
				"name":                "rachao",
				"authorization_token": token,
			},
		},
	}

	bodyBytes, _ := json.Marshal(reqBody)
	anthropicReq, err := http.NewRequestWithContext(r.Context(), http.MethodPost, anthropicMessagesURL, bytes.NewReader(bodyBytes))
	if err != nil {
		emitError("erro interno. Tente novamente.")
		return
	}
	anthropicReq.Header.Set("x-api-key", h.anthropicKey)
	anthropicReq.Header.Set("anthropic-version", "2023-06-01")
	anthropicReq.Header.Set("anthropic-beta", "mcp-client-2025-04-04")
	anthropicReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(anthropicReq)
	if err != nil {
		emitError("erro ao conectar com o assistente. Tente novamente.")
		return
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		emitError("erro ao conectar com o assistente. Tente novamente.")
		return
	}

	log.Printf("chat_request player_id=%s model=%s", player.ID, h.llmModel)

	// Parse SSE stream from Anthropic
	lastBlockWasTool := false
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" || data == "" {
			continue
		}

		var event map[string]any
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		eventType, _ := event["type"].(string)
		switch eventType {
		case "content_block_start":
			block, _ := event["content_block"].(map[string]any)
			if block != nil {
				blockType, _ := block["type"].(string)
				if blockType == "text" && lastBlockWasTool {
					b, _ := json.Marshal(map[string]string{"text": "\n\n"})
					emit(string(b))
				}
				lastBlockWasTool = blockType != "text"
			}
		case "content_block_delta":
			delta, _ := event["delta"].(map[string]any)
			if delta != nil {
				if dt, _ := delta["type"].(string); dt == "text_delta" {
					if text, _ := delta["text"].(string); text != "" {
						b, _ := json.Marshal(map[string]string{"text": text})
						emit(string(b))
					}
				}
			}
		case "message_stop":
			emit("[DONE]")
			return
		case "error":
			emitError("erro ao conectar com o assistente. Tente novamente.")
			return
		}
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		emitError("erro interno. Tente novamente.")
		return
	}
	emit("[DONE]")
}

func (h *chatHandler) listChatUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := h.pool.Query(r.Context(), `
		SELECT id, name, nickname, avatar_url, chat_enabled, api_v2_enabled, created_at
		FROM players
		WHERE role = 'player'
		ORDER BY created_at DESC`)
	if err != nil {
		renderError(w, err)
		return
	}
	defer rows.Close()

	type chatUserItem struct {
		ID           uuid.UUID   `json:"id"`
		Name         string      `json:"name"`
		Nickname     *string     `json:"nickname"`
		AvatarURL    *string     `json:"avatar_url"`
		ChatEnabled  bool        `json:"chat_enabled"`
		ApiV2Enabled bool        `json:"api_v2_enabled"`
		CreatedAt    interface{} `json:"created_at"`
	}

	users := make([]chatUserItem, 0)
	for rows.Next() {
		var u chatUserItem
		if err := rows.Scan(&u.ID, &u.Name, &u.Nickname, &u.AvatarURL,
			&u.ChatEnabled, &u.ApiV2Enabled, &u.CreatedAt); err != nil {
			renderError(w, err)
			return
		}
		users = append(users, u)
	}

	totalEnabled := 0
	for _, u := range users {
		if u.ChatEnabled {
			totalEnabled++
		}
	}

	renderJSON(w, http.StatusOK, map[string]any{
		"users":         users,
		"total_enabled": totalEnabled,
	})
}

func (h *chatHandler) updateChatAccess(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		renderError(w, apierror.NotFound("user not found"))
		return
	}

	var req chatAccessUpdate
	if err := decodeJSON(r, &req); err != nil {
		renderError(w, err)
		return
	}

	player, err := db.GetPlayerByID(ctx, h.pool, userID)
	if err != nil {
		renderError(w, apierror.NotFound("user not found"))
		return
	}

	if err := db.UpdatePlayerChatEnabled(ctx, h.pool, userID, req.ChatEnabled); err != nil {
		renderError(w, err)
		return
	}

	renderJSON(w, http.StatusOK, map[string]any{
		"id":           player.ID,
		"name":         player.Name,
		"chat_enabled": req.ChatEnabled,
	})
}
