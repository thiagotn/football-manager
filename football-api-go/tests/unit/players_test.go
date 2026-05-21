package unit_test

import (
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/thiagotn/football-manager/football-api-go/internal/db"
	"github.com/thiagotn/football-manager/football-api-go/internal/handlers"
	"github.com/thiagotn/football-manager/football-api-go/internal/middleware"
)

// playersRouter sets up a chi router with player routes and an injected player.
// pool is nil — only tests that return before any DB call are valid here.
func playersRouter(player *db.Player) http.Handler {
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := middleware.InjectPlayerForTest(req.Context(), player)
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	r.Mount("/players", handlers.NewPlayerHandler(nil, nil).Routes())
	return r
}

// ── createPlayer ──────────────────────────────────────────────────────────────

func TestCreatePlayer_NonAdminForbidden(t *testing.T) {
	r := playersRouter(fakePlayer())
	w := postJSON(r, "/players", `{"name":"Zico","whatsapp":"+5511999990000","password":"senha123"}`)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestCreatePlayer_MalformedBody(t *testing.T) {
	r := playersRouter(fakePlayer(asAdmin()))
	w := postJSON(r, "/players", `{bad json}`)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestCreatePlayer_NameTooShort(t *testing.T) {
	r := playersRouter(fakePlayer(asAdmin()))
	w := postJSON(r, "/players", `{"name":"X","whatsapp":"+5511999990000","password":"senha123"}`)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestCreatePlayer_PasswordTooShort(t *testing.T) {
	r := playersRouter(fakePlayer(asAdmin()))
	w := postJSON(r, "/players", `{"name":"Zico","whatsapp":"+5511999990000","password":"123"}`)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

// ── getPlayer ─────────────────────────────────────────────────────────────────

func TestGetPlayer_InvalidUUID(t *testing.T) {
	r := playersRouter(fakePlayer())
	w := doRequest(r, http.MethodGet, "/players/not-a-uuid", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetPlayer_NonAdminAccessOtherPlayer(t *testing.T) {
	// Non-admin player trying to access a different player's profile → 403
	player := fakePlayer()
	r := playersRouter(player)
	otherID := uuid.New().String()
	w := doRequest(r, http.MethodGet, "/players/"+otherID, "")
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// ── updatePlayer ──────────────────────────────────────────────────────────────

func TestUpdatePlayer_InvalidUUID(t *testing.T) {
	r := playersRouter(fakePlayer())
	w := doRequest(r, http.MethodPatch, "/players/not-a-uuid", `{"name":"New"}`)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ── resetPassword ─────────────────────────────────────────────────────────────

func TestResetPassword_NonAdminForbidden(t *testing.T) {
	player := fakePlayer()
	r := playersRouter(player)
	targetID := uuid.New().String()
	w := postJSON(r, "/players/"+targetID+"/reset-password", `{"new_password":"newpass123"}`)
	assert.Equal(t, http.StatusForbidden, w.Code)
}
