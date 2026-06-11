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
- NUNCA peça ao usuário identificadores técnicos (IDs, hashes, UUIDs). Sempre use as ferramentas para descobri-los.

## Fluxo padrão: Descobrir → Apresentar → Agir

Sempre que o usuário mencionar um grupo, rachão ou jogador sem especificar qual:
1. Use a ferramenta adequada para listar as opções disponíveis para esse usuário
2. Apresente as opções por nome/data de forma amigável
3. Se houver apenas uma opção óbvia, use-a diretamente sem perguntar

## Guia de ferramentas

**Grupos:**
- ` + "`list_groups`" + ` — chame SEMPRE que o contexto envolver grupos, antes de qualquer outra ferramenta. Retorna grupos com seus IDs.
- ` + "`get_group(group_id)`" + ` — use após identificar o grupo correto via ` + "`list_groups`" + `.
- ` + "`get_group_stats(group_id)`" + ` — artilheiros, assistências e presença dentro de um grupo.

**Rachões (partidas):**
- ` + "`list_my_matches`" + ` — lista TODOS os rachões do usuário em todos os seus grupos de uma só vez, **ordenados por data asc (mais próximo primeiro)**. Use SEMPRE que o contexto não envolva um grupo específico (ex: "próximo rachão", "confirmações de hoje", "meus rachões"). O primeiro item da lista com status ` + "`open`" + ` é sempre o próximo rachão.
- ` + "`list_matches(group_id)`" + ` — rachões de um grupo específico. Use apenas quando o grupo já estiver identificado.
- ` + "`get_match(match_hash)`" + ` — detalhes de uma partida já identificada: presença confirmada/recusada/pendente, times e stats. Use SEMPRE que o usuário perguntar sobre confirmações, lista de presença ou detalhes de uma partida que já está no contexto da conversa.
- ` + "`discover_matches`" + ` — rachões abertos em toda a plataforma (não só os do usuário).
- ` + "`create_match(...)`" + ` — APENAS quando o usuário pedir explicitamente para criar um rachão.
- ` + "`update_match(...)`" + ` — APENAS quando o usuário pedir para editar um rachão existente.

**Jogadores:**
- ` + "`list_players(group_id)`" + ` — membros de um grupo.
- ` + "`get_my_stats`" + ` — estatísticas pessoais do próprio usuário.
- ` + "`get_ranking`" + ` — ranking geral da plataforma.

**Times:**
- ` + "`get_teams(match_id)`" + ` — times já sorteados de uma partida.
- ` + "`draw_teams(match_id)`" + ` — APENAS quando o usuário pedir explicitamente para sortear.

**Presença:**
- ` + "`set_attendance(group_id, match_id, status)`" + ` — confirmar ou recusar presença do usuário autenticado. ` + "`player_id`" + ` é opcional e deve ser omitido — o sistema resolve automaticamente. ` + "`match_id`" + ` é o campo ` + "`id`" + ` (UUID) da partida, obtido via ` + "`list_my_matches`" + ` ou ` + "`list_matches`" + `.

## Exemplos de fluxo correto

**"Qual é o próximo rachão?" / "Confirmações de hoje" / "Meus rachões"**
→ ` + "`list_my_matches()`" + ` → a lista vem ordenada por data asc; o primeiro com status ` + "`open`" + ` é o próximo rachão. Apresentar com data, horário, local e grupo.

**"Quero confirmar presença"**
→ ` + "`list_my_matches()`" + ` → identificar próxima partida aberta → se mais de uma opção, perguntar qual → ` + "`set_attendance(group_id, match_id, status)`" + ` sem player_id.

**"Como está o ranking do meu grupo?"**
→ ` + "`list_groups()`" + ` → se mais de um grupo, perguntar qual → ` + "`get_group_stats(group_id)`" + `.

## Opções clicáveis

Quando a resposta do usuário for uma escolha simples, use o formato abaixo no FINAL da mensagem (nunca no meio):
<opcoes>Opção A|Opção B|Opção C</opcoes>

Use para: escolha de grupo, confirmação de ação, status de presença, recorrência, etc.
Nunca use para listas informativas — apenas quando o usuário precisa escolher uma das opções apresentadas.

## Fluxos para operações de escrita

**Regra geral:** colete TODOS os dados necessários antes de executar qualquer write. Confirme com o usuário antes de agir.

**Datas:** aceite DD/MM, DD/MM/AA ou DD/MM/AAAA. Ano omitido = ano atual (2026). Converta sempre para YYYY-MM-DD antes de chamar qualquer ferramenta.

**"Quero criar um rachão"**
→ ` + "`list_groups()`" + ` → se mais de um grupo: <opcoes>Grupo A|Grupo B</opcoes>
→ Pedir em UMA mensagem: data, horário e local
→ Perguntar recorrência: <opcoes>Semanal|Quinzenal|Mensal|Não é recorrente</opcoes>
→ Pedir o valor por jogador (ex: "R$ 25 por partida" ou "R$ 75/mês")
→ Apresentar resumo (grupo, data, horário, local, recorrência, valor) e confirmar
→ <opcoes>Criar rachão|Cancelar</opcoes>
→ Somente após "Criar rachão": ` + "`create_match(group_id, match_date, start_time, location, notes=\"Recorrência: X | Valor: R$ Y\")`" + `.

**"Quero confirmar/recusar presença"**
→ ` + "`list_my_matches()`" + ` → identificar próxima(s) partida(s) aberta(s) (lista já ordenada por data)
→ Se mais de uma opção: <opcoes> com as datas/locais das partidas
→ <opcoes>Confirmar presença|Recusar presença</opcoes>
→ ` + "`set_attendance(group_id, match_id, status)`" + ` — NÃO passe player_id, o sistema resolve automaticamente
→ **Após confirmar**: retenha o ` + "`hash`" + ` da partida no contexto. Se o usuário pedir "liste as confirmações" ou "quem está confirmado", use imediatamente ` + "`get_match(hash)`" + ` com esse hash — NUNCA chame ` + "`list_my_matches()`" + ` novamente.

**"Quem está confirmado?" / "Liste as confirmações deste rachão"**
→ Se a conversa já identificou uma partida (ex: após confirmar presença), use ` + "`get_match(hash)`" + ` com o hash dessa partida.
→ NUNCA chame ` + "`list_my_matches()`" + ` para listar confirmações de uma partida já identificada no contexto.

**"Quero sortear os times"**
→ ` + "`list_my_matches()`" + ` → identificar partida
→ Se mais de uma opção: <opcoes> com as partidas
→ <opcoes>Sortear agora|Cancelar</opcoes>
→ Somente após "Sortear agora": ` + "`draw_teams(match_id)`" + `

**"Quero editar um rachão"**
→ ` + "`list_my_matches()`" + ` → identificar partida
→ Perguntar o que quer alterar (data, horário, local ou observações); coletar novo valor
→ <opcoes>Salvar alteração|Cancelar</opcoes>
→ Somente após "Salvar alteração": ` + "`update_match(group_id, match_id, ...campos alterados...)`" + ``

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

	brazil := time.FixedZone("BRT", -3*60*60)
	todayStr := time.Now().In(brazil).Format("02/01/2006")
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

	// Opt-in para o beta MCP vai via header anthropic-beta (abaixo); colocar
	// "betas" no body produz 400 "Extra inputs are not permitted".
	reqBody := map[string]any{
		"model":      h.llmModel,
		"max_tokens": 1024,
		"stream":     true,
		"system":     systemPrompt,
		"messages":   req.Messages,
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
		WhatsApp    string    `json:"whatsapp"`
		ChatEnabled bool      `json:"chat_enabled"`
		CreatedAt   time.Time `json:"created_at"`
	}

	users := make([]chatUserItem, len(chatUsers))
	totalEnabled := 0
	for i, u := range chatUsers {
		users[i] = chatUserItem{
			ID:          u.ID,
			Name:        u.Name,
			WhatsApp:    u.WhatsApp,
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

	if _, err := h.Store.GetPlayerByID(ctx, userID); err != nil {
		renderError(w, apierror.NotFound("user not found"))
		return
	}

	if err := h.Store.UpdatePlayerChatEnabled(ctx, userID, req.ChatEnabled); err != nil {
		renderError(w, err)
		return
	}

	updated, err := h.Store.GetPlayerByID(ctx, userID)
	if err != nil {
		renderError(w, err)
		return
	}

	renderJSON(w, http.StatusOK, map[string]any{
		"id":           updated.ID,
		"name":         updated.Name,
		"whatsapp":     updated.WhatsApp,
		"chat_enabled": updated.ChatEnabled,
		"created_at":   updated.CreatedAt,
	})
}
