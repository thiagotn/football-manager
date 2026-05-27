package handlers

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thiagotn/football-manager/football-api-go/internal/apierror"
	"github.com/thiagotn/football-manager/football-api-go/internal/db"
	"github.com/thiagotn/football-manager/football-api-go/internal/middleware"
)

type PushStore interface {
	UpsertPushSubscription(ctx context.Context, playerID uuid.UUID, endpoint, p256dh, auth string, userAgent *string) error
	DeletePushSubscriptions(ctx context.Context, playerID uuid.UUID) error
}

type pgPushStore struct {
	pool *pgxpool.Pool
}

func (s *pgPushStore) UpsertPushSubscription(ctx context.Context, playerID uuid.UUID, endpoint, p256dh, auth string, userAgent *string) error {
	return db.UpsertPushSubscription(ctx, s.pool, playerID, endpoint, p256dh, auth, userAgent)
}

func (s *pgPushStore) DeletePushSubscriptions(ctx context.Context, playerID uuid.UUID) error {
	return db.DeletePushSubscriptions(ctx, s.pool, playerID)
}

type PushHandler struct {
	Store          PushStore
	VapidPublicKey string
}

func NewPushHandler(pool *pgxpool.Pool, vapidPublicKey string) *PushHandler {
	return &PushHandler{Store: &pgPushStore{pool: pool}, VapidPublicKey: vapidPublicKey}
}

func (h *PushHandler) GetVapidKey(w http.ResponseWriter, r *http.Request) {
	renderJSON(w, http.StatusOK, map[string]string{"public_key": h.VapidPublicKey})
}

func (h *PushHandler) Subscribe(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())
	if player == nil {
		renderError(w, apierror.Unauthorized())
		return
	}

	var body struct {
		Endpoint string `json:"endpoint"`
		Keys     struct {
			P256dh string `json:"p256dh"`
			Auth   string `json:"auth"`
		} `json:"keys"`
		UserAgent *string `json:"user_agent"`
	}
	if err := decodeJSON(r, &body); err != nil {
		renderError(w, err)
		return
	}
	if body.Endpoint == "" || body.Keys.P256dh == "" || body.Keys.Auth == "" {
		renderError(w, apierror.Unprocessable("endpoint and keys (p256dh, auth) are required"))
		return
	}

	if err := h.Store.UpsertPushSubscription(r.Context(), player.ID, body.Endpoint, body.Keys.P256dh, body.Keys.Auth, body.UserAgent); err != nil {
		renderError(w, apierror.Internal("failed to subscribe"))
		return
	}

	renderJSON(w, http.StatusOK, map[string]string{"status": "subscribed"})
}

func (h *PushHandler) Unsubscribe(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())
	if player == nil {
		renderError(w, apierror.Unauthorized())
		return
	}

	if err := h.Store.DeletePushSubscriptions(r.Context(), player.ID); err != nil {
		renderError(w, apierror.Internal("failed to unsubscribe"))
		return
	}

	noContent(w)
}
