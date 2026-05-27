package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thiagotn/football-manager/football-api-go/internal/apierror"
	"github.com/thiagotn/football-manager/football-api-go/internal/db"
	"github.com/thiagotn/football-manager/football-api-go/internal/middleware"
)

type MCPTokenStore interface {
	GenerateMCPToken() (raw, hash, prefix string, err error)
	CreateMCPToken(ctx context.Context, params db.CreateMCPTokenParams) (*db.MCPToken, error)
	ListMCPTokens(ctx context.Context, playerID uuid.UUID) ([]db.MCPToken, error)
	GetMCPToken(ctx context.Context, tokenID uuid.UUID) (*db.MCPToken, error)
	RevokeMCPToken(ctx context.Context, tokenID uuid.UUID) error
}

type pgMCPTokenStore struct {
	pool *pgxpool.Pool
}

func (s *pgMCPTokenStore) GenerateMCPToken() (raw, hash, prefix string, err error) {
	return db.GenerateMCPToken()
}

func (s *pgMCPTokenStore) CreateMCPToken(ctx context.Context, params db.CreateMCPTokenParams) (*db.MCPToken, error) {
	return db.CreateMCPToken(ctx, s.pool, params)
}

func (s *pgMCPTokenStore) ListMCPTokens(ctx context.Context, playerID uuid.UUID) ([]db.MCPToken, error) {
	return db.ListMCPTokens(ctx, s.pool, playerID)
}

func (s *pgMCPTokenStore) GetMCPToken(ctx context.Context, tokenID uuid.UUID) (*db.MCPToken, error) {
	return db.GetMCPToken(ctx, s.pool, tokenID)
}

func (s *pgMCPTokenStore) RevokeMCPToken(ctx context.Context, tokenID uuid.UUID) error {
	return db.RevokeMCPToken(ctx, s.pool, tokenID)
}

type MCPTokenHandler struct {
	Store MCPTokenStore
}

func NewMCPTokenHandler(pool *pgxpool.Pool) *MCPTokenHandler {
	return &MCPTokenHandler{Store: &pgMCPTokenStore{pool: pool}}
}

func (h *MCPTokenHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.createToken)
	r.Get("/", h.listTokens)
	r.Delete("/{tokenID}", h.revokeToken)
	return r
}

func (h *MCPTokenHandler) createToken(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())
	if player == nil {
		renderError(w, apierror.Unauthorized())
		return
	}

	var body struct {
		Name      string  `json:"name"`
		ExpiresIn *string `json:"expires_in"`
	}
	if err := decodeJSON(r, &body); err != nil {
		renderError(w, err)
		return
	}
	if body.Name == "" {
		renderError(w, apierror.Unprocessable("name is required"))
		return
	}

	var expiresAt *time.Time
	if body.ExpiresIn != nil {
		now := time.Now().UTC()
		switch *body.ExpiresIn {
		case "24h":
			t := now.Add(24 * time.Hour)
			expiresAt = &t
		case "7d":
			t := now.Add(7 * 24 * time.Hour)
			expiresAt = &t
		default:
			renderError(w, apierror.Unprocessable("expires_in must be '24h', '7d', or null"))
			return
		}
	}

	raw, hash, prefix, err := h.Store.GenerateMCPToken()
	if err != nil {
		renderError(w, apierror.Internal("failed to generate token"))
		return
	}

	token, err := h.Store.CreateMCPToken(r.Context(), db.CreateMCPTokenParams{
		PlayerID:  player.ID,
		Name:      body.Name,
		TokenHash: hash,
		Prefix:    prefix,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		renderError(w, apierror.Internal("failed to create token"))
		return
	}

	renderJSON(w, http.StatusCreated, map[string]any{
		"id":           token.ID,
		"name":         token.Name,
		"token":        raw,
		"token_prefix": token.TokenPrefix,
		"expires_at":   token.ExpiresAt,
		"created_at":   token.CreatedAt,
	})
}

func (h *MCPTokenHandler) listTokens(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())
	if player == nil {
		renderError(w, apierror.Unauthorized())
		return
	}

	tokens, err := h.Store.ListMCPTokens(r.Context(), player.ID)
	if err != nil {
		renderError(w, apierror.Internal("failed to list tokens"))
		return
	}

	now := time.Now().UTC()
	type tokenResp struct {
		ID          uuid.UUID  `json:"id"`
		Name        string     `json:"name"`
		TokenPrefix string     `json:"token_prefix"`
		ExpiresAt   *time.Time `json:"expires_at"`
		CreatedAt   time.Time  `json:"created_at"`
		LastUsedAt  *time.Time `json:"last_used_at"`
		IsExpired   bool       `json:"is_expired"`
	}

	resp := make([]tokenResp, len(tokens))
	for i, t := range tokens {
		resp[i] = tokenResp{
			ID:          t.ID,
			Name:        t.Name,
			TokenPrefix: t.TokenPrefix,
			ExpiresAt:   t.ExpiresAt,
			CreatedAt:   t.CreatedAt,
			LastUsedAt:  t.LastUsedAt,
			IsExpired:   t.ExpiresAt != nil && t.ExpiresAt.Before(now),
		}
	}
	renderJSON(w, http.StatusOK, resp)
}

func (h *MCPTokenHandler) revokeToken(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())
	if player == nil {
		renderError(w, apierror.Unauthorized())
		return
	}

	tokenIDStr := chi.URLParam(r, "tokenID")
	tokenID, err := uuid.Parse(tokenIDStr)
	if err != nil {
		renderError(w, apierror.NotFound("token not found"))
		return
	}

	token, err := h.Store.GetMCPToken(r.Context(), tokenID)
	if err != nil {
		renderError(w, apierror.NotFound("token not found"))
		return
	}
	if token.PlayerID != player.ID {
		renderError(w, apierror.Forbidden("cannot revoke another player's token"))
		return
	}

	if err := h.Store.RevokeMCPToken(r.Context(), tokenID); err != nil {
		renderError(w, apierror.Internal("failed to revoke token"))
		return
	}

	noContent(w)
}
