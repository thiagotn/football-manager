package handlers

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thiagotn/football-manager/football-api-go/internal/apierror"
	"github.com/thiagotn/football-manager/football-api-go/internal/db"
	"github.com/thiagotn/football-manager/football-api-go/internal/middleware"
)

type pushHandler struct {
	pool           *pgxpool.Pool
	vapidPublicKey string
}

func NewPushHandler(pool *pgxpool.Pool, vapidPublicKey string) *pushHandler {
	return &pushHandler{pool: pool, vapidPublicKey: vapidPublicKey}
}

func (h *pushHandler) GetVapidKey(w http.ResponseWriter, r *http.Request) {
	renderJSON(w, http.StatusOK, map[string]string{"public_key": h.vapidPublicKey})
}

func (h *pushHandler) Subscribe(w http.ResponseWriter, r *http.Request) {
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

	if err := db.UpsertPushSubscription(r.Context(), h.pool, player.ID, body.Endpoint, body.Keys.P256dh, body.Keys.Auth, body.UserAgent); err != nil {
		renderError(w, apierror.Internal("failed to subscribe"))
		return
	}

	renderJSON(w, http.StatusOK, map[string]string{"status": "subscribed"})
}

func (h *pushHandler) Unsubscribe(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())
	if player == nil {
		renderError(w, apierror.Unauthorized())
		return
	}

	if err := db.DeletePushSubscriptions(r.Context(), h.pool, player.ID); err != nil {
		renderError(w, apierror.Internal("failed to unsubscribe"))
		return
	}

	noContent(w)
}
