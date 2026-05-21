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

// matchesRouter sets up a chi router mirroring the main router's group-match routes.
// pool is nil — only tests that return before any DB call are valid here.
func matchesRouter(player *db.Player) http.Handler {
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := middleware.InjectPlayerForTest(req.Context(), player)
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	h := handlers.NewMatchHandler(nil)
	r.Route("/groups/{groupID}/matches", func(r chi.Router) {
		r.Mount("/", h.GroupMatchRoutes())
	})
	return r
}

func groupMatchURL(groupID, matchID string) string {
	return "/groups/" + groupID + "/matches/" + matchID
}

// ── setAttendance ─────────────────────────────────────────────────────────────

func TestSetAttendance_AdminForbidden(t *testing.T) {
	admin := fakePlayer(asAdmin())
	r := matchesRouter(admin)

	url := groupMatchURL(uuid.New().String(), uuid.New().String()) + "/attendance"
	w := postJSON(r, url, `{"player_id":"`+uuid.New().String()+`","status":"confirmed"}`)
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "admins cannot set attendance")
}

func TestSetAttendance_InvalidGroupID(t *testing.T) {
	player := fakePlayer()
	r := matchesRouter(player)

	url := groupMatchURL("not-a-uuid", uuid.New().String()) + "/attendance"
	w := postJSON(r, url, `{"player_id":"`+uuid.New().String()+`","status":"confirmed"}`)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSetAttendance_InvalidMatchID(t *testing.T) {
	player := fakePlayer()
	r := matchesRouter(player)

	url := groupMatchURL(uuid.New().String(), "not-a-uuid") + "/attendance"
	w := postJSON(r, url, `{"player_id":"`+uuid.New().String()+`","status":"confirmed"}`)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSetAttendance_InvalidStatus(t *testing.T) {
	// Non-admin, valid UUIDs in URL, bad status → 422 before any DB call
	player := fakePlayer()
	r := matchesRouter(player)

	url := groupMatchURL(uuid.New().String(), uuid.New().String()) + "/attendance"
	w := postJSON(r, url, `{"player_id":"`+uuid.New().String()+`","status":"maybe"}`)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "invalid status")
}

func TestSetAttendance_MalformedBody(t *testing.T) {
	player := fakePlayer()
	r := matchesRouter(player)

	url := groupMatchURL(uuid.New().String(), uuid.New().String()) + "/attendance"
	w := postJSON(r, url, `{bad json}`)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

// ── createMatch ───────────────────────────────────────────────────────────────

func TestCreateMatch_MalformedBody(t *testing.T) {
	// Admin player → skips group-member DB check, fails on bad JSON
	admin := fakePlayer(asAdmin())
	r := matchesRouter(admin)

	groupID := uuid.New().String()
	w := postJSON(r, "/groups/"+groupID+"/matches", `{bad json}`)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestCreateMatch_MissingRequiredFields(t *testing.T) {
	// Admin player, valid body but missing match_date/start_time/location → 422
	admin := fakePlayer(asAdmin())
	r := matchesRouter(admin)

	groupID := uuid.New().String()
	w := postJSON(r, "/groups/"+groupID+"/matches", `{"location":"Quadra A"}`)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "required")
}

func TestCreateMatch_InvalidGroupID(t *testing.T) {
	admin := fakePlayer(asAdmin())
	r := matchesRouter(admin)

	w := postJSON(r, "/groups/not-a-uuid/matches",
		`{"match_date":"2026-01-01","start_time":"20:00","location":"Quadra"}`)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ── getMatch / updateMatch / deleteMatch ──────────────────────────────────────

func TestGetMatch_InvalidGroupID(t *testing.T) {
	player := fakePlayer()
	r := matchesRouter(player)

	w := doRequest(r, http.MethodGet, "/groups/bad/matches/"+uuid.New().String(), "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetMatch_InvalidMatchID(t *testing.T) {
	player := fakePlayer()
	r := matchesRouter(player)

	w := doRequest(r, http.MethodGet,
		"/groups/"+uuid.New().String()+"/matches/bad-match-id", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateMatch_MalformedBody(t *testing.T) {
	// Admin player → skips group-member DB check, fails on bad JSON
	admin := fakePlayer(asAdmin())
	r := matchesRouter(admin)

	url := groupMatchURL(uuid.New().String(), uuid.New().String())
	w := doRequest(r, http.MethodPatch, url, `{bad}`)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestDeleteMatch_InvalidGroupID(t *testing.T) {
	admin := fakePlayer(asAdmin())
	r := matchesRouter(admin)

	w := doRequest(r, http.MethodDelete,
		"/groups/bad/matches/"+uuid.New().String(), "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}
