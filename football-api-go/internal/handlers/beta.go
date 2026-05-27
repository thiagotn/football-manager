package handlers

import (
	"context"
	"net/http"
	"regexp"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thiagotn/football-manager/football-api-go/internal/apierror"
	"github.com/thiagotn/football-manager/football-api-go/internal/db"
	"github.com/thiagotn/football-manager/football-api-go/internal/middleware"
)

var emailRegex = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

type BetaStore interface {
	InsertAndroidBetaSignup(ctx context.Context, email string, playerID *uuid.UUID) error
}

type pgBetaStore struct {
	pool *pgxpool.Pool
}

func (s *pgBetaStore) InsertAndroidBetaSignup(ctx context.Context, email string, playerID *uuid.UUID) error {
	return db.InsertAndroidBetaSignup(ctx, s.pool, email, playerID)
}

type BetaHandler struct {
	Store BetaStore
}

func NewBetaHandler(pool *pgxpool.Pool) *BetaHandler {
	return &BetaHandler{Store: &pgBetaStore{pool: pool}}
}

func (h *BetaHandler) AndroidSignup(w http.ResponseWriter, r *http.Request) {
	var body struct {
		GoogleEmail string `json:"google_email"`
	}
	if err := decodeJSON(r, &body); err != nil {
		renderError(w, err)
		return
	}

	if body.GoogleEmail == "" || len(body.GoogleEmail) > 254 || !emailRegex.MatchString(body.GoogleEmail) {
		renderError(w, apierror.Unprocessable("invalid email address"))
		return
	}

	player := middleware.PlayerFromCtx(r.Context())
	var playerID *uuid.UUID
	if player != nil {
		playerID = &player.ID
	}

	if err := h.Store.InsertAndroidBetaSignup(r.Context(), body.GoogleEmail, playerID); err != nil {
		renderError(w, apierror.Internal("failed to register signup"))
		return
	}

	renderJSON(w, http.StatusCreated, map[string]string{"status": "ok"})
}
