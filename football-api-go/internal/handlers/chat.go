package handlers

import (
	"bufio"
	"bytes"
	"context"
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

type ChatStore interface {
	CheckAndIncrementChatRateLimit(ctx context.Context, playerID uuid.UUID, limit int) (bool, error)
	GetPlayerByID(ctx context.Context, playerID uuid.UUID) (*db.Player, error)
	UpdatePlayerChatEnabled(ctx context.Context, playerID uuid.UUID, enabled bool) error
	ListChatUsers(ctx context.Context) ([]db.ChatUser, error)
}

type pgChatStore struct {
	pool *pgxpool.Pool
}

func (s *pgChatStore) CheckAndIncrementChatRateLimit(ctx context.Context, playerID uuid.UUID, limit int) (bool, error) {
	return db.CheckAndIncrementChatRateLimit(ctx, s.pool, playerID, limit)
}

func (s *pgChatStore) GetPlayerByID(ctx context.Context, playerID uuid.UUID) (*db.Player, error) {
	return db.GetPlayerByID(ctx, s.pool, playerID)
}

func (s *pgChatStore) UpdatePlayerChatEnabled(ctx context.Context, playerID uuid.UUID, enabled bool) error {
	return db.UpdatePlayerChatEnabled(ctx, s.pool, playerID, enabled)
}

func (s *pgChatStore) ListChatUsers(ctx context.Context) ([]db.ChatUser, error) {
	return db.ListChatUsers(ctx, s.pool)
}

type ChatHandler struct {
	Store         ChatStore
	anthropicKey  string
	llmModel      string
	chatRateLimit int
}

func NewChatHandler(pool *pgxpool.Pool, anthropicKey, llmModel string, chatRateLimit int) *ChatHandler {
	return &ChatHandler{
		Store:         &pgChatStore{pool: pool},
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

func (h *ChatHandler) ListChatUsers(w http.ResponseWriter, r *http.Request) {
	h.listChatUsers(w, r)
}

func (h *ChatHandler) UpdateChatAccess(w http.ResponseWriter, r *http.Request) {
	h.updateChatAccess(w, r)
}

// ── Handlers ──────────────────────────────────────────────────────────────────

func (h *ChatHandler) Chat(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())

	if !player.ChatEnabled {
		renderError(w, apierror.Forbidden("chat access not enabled for this user"))
		return
	}

	allowed, err := h.Store.CheckAndIncrementChatRateLimit(r.Context(), player.ID, h.chatRateLimit)
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
		log.Printf("chat_anthropic_dial_failed player_id=%s err=%v", player.ID, err)
		emitError("erro ao conectar com o assistente. Tente novamente.")
		return
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		// Lê (até 1KB) e loga o body do Anthropic pra diagnóstico. Sem isso o
		// handler ficava mudo: 200 com mensagem genérica e nenhuma pista da causa.
		bodyPreview, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		log.Printf("chat_anthropic_non_ok player_id=%s status=%d body=%q model=%s",
			player.ID, resp.StatusCode, string(bodyPreview), h.llmModel)
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
			// Loga o conteúdo do evento de erro do Anthropic pra diagnóstico
			// (ex.: erro do servidor MCP propagado, modelo inválido, rate limit etc.).
			errPayload, _ := event["error"]
			log.Printf("chat_anthropic_stream_error player_id=%s err=%v", player.ID, errPayload)
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

func (h *ChatHandler) listChatUsers(w http.ResponseWriter, r *http.Request) {
	chatUsers, err := h.Store.ListChatUsers(r.Context())
	if err != nil {
		renderError(w, err)
		return
	}

	type chatUserItem struct {
		ID          uuid.UUID `json:"id"`
		Name        string    `json:"name"`
		Nickname    *string   `json:"nickname"`
		AvatarURL   *string   `json:"avatar_url"`
		ChatEnabled bool      `json:"chat_enabled"`
		CreatedAt   time.Time `json:"created_at"`
	}

	users := make([]chatUserItem, len(chatUsers))
	totalEnabled := 0
	for i, u := range chatUsers {
		users[i] = chatUserItem{
			ID:          u.ID,
			Name:        u.Name,
			Nickname:    u.Nickname,
			AvatarURL:   u.AvatarURL,
			ChatEnabled: u.ChatEnabled,
			CreatedAt:   u.CreatedAt,
		}
		if u.ChatEnabled {
			totalEnabled++
		}
	}

	renderJSON(w, http.StatusOK, map[string]any{
		"users":         users,
		"total_enabled": totalEnabled,
	})
}

func (h *ChatHandler) updateChatAccess(w http.ResponseWriter, r *http.Request) {
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

	player, err := h.Store.GetPlayerByID(ctx, userID)
	if err != nil {
		renderError(w, apierror.NotFound("user not found"))
		return
	}

	if err := h.Store.UpdatePlayerChatEnabled(ctx, userID, req.ChatEnabled); err != nil {
		renderError(w, err)
		return
	}

	renderJSON(w, http.StatusOK, map[string]any{
		"id":           player.ID,
		"name":         player.Name,
		"chat_enabled": req.ChatEnabled,
	})
}
