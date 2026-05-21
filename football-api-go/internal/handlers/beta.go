package handlers

import (
	"net/http"
	"regexp"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thiagotn/football-manager/football-api-go/internal/apierror"
	"github.com/thiagotn/football-manager/football-api-go/internal/db"
	"github.com/thiagotn/football-manager/football-api-go/internal/middleware"
)

var emailRegex = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

type betaHandler struct {
	pool *pgxpool.Pool
}

func NewBetaHandler(pool *pgxpool.Pool) *betaHandler {
	return &betaHandler{pool: pool}
}

func (h *betaHandler) AndroidSignup(w http.ResponseWriter, r *http.Request) {
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
	var err error
	if player != nil {
		id := player.ID
		err = db.InsertAndroidBetaSignup(r.Context(), h.pool, body.GoogleEmail, &id)
	} else {
		err = db.InsertAndroidBetaSignup(r.Context(), h.pool, body.GoogleEmail, nil)
	}
	if err != nil {
		renderError(w, apierror.Internal("failed to register signup"))
		return
	}

	renderJSON(w, http.StatusCreated, map[string]string{"status": "ok"})
}
