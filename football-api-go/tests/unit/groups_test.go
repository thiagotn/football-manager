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

// groupsRouter sets up a chi router with groups routes and an injected player.
// pool is nil — only tests that return before any DB call are valid here.
func groupsRouter(player *db.Player) http.Handler {
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := middleware.InjectPlayerForTest(req.Context(), player)
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	r.Mount("/groups", handlers.NewGroupHandler(nil).Routes())
	return r
}

// ── createGroup ───────────────────────────────────────────────────────────────

func TestCreateGroup_MalformedBody(t *testing.T) {
	r := groupsRouter(fakePlayer(asAdmin()))
	w := postJSON(r, "/groups", `{bad json}`)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestCreateGroup_EmptyName(t *testing.T) {
	r := groupsRouter(fakePlayer(asAdmin()))
	w := postJSON(r, "/groups", `{"name":""}`)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestCreateGroup_NameTooShort(t *testing.T) {
	r := groupsRouter(fakePlayer(asAdmin()))
	w := postJSON(r, "/groups", `{"name":"x"}`)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

// ── getGroup / updateGroup / deleteGroup ──────────────────────────────────────

func TestGetGroup_InvalidUUID(t *testing.T) {
	r := groupsRouter(fakePlayer())
	w := doRequest(r, http.MethodGet, "/groups/not-a-uuid", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateGroup_InvalidUUID(t *testing.T) {
	r := groupsRouter(fakePlayer())
	w := doRequest(r, http.MethodPatch, "/groups/not-a-uuid", `{"name":"New Name"}`)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteGroup_InvalidUUID(t *testing.T) {
	r := groupsRouter(fakePlayer(asAdmin()))
	w := doRequest(r, http.MethodDelete, "/groups/not-a-uuid", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ── members ───────────────────────────────────────────────────────────────────

func TestListMembers_InvalidGroupUUID(t *testing.T) {
	r := groupsRouter(fakePlayer())
	w := doRequest(r, http.MethodGet, "/groups/not-a-uuid/members", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAddMember_InvalidGroupUUID(t *testing.T) {
	r := groupsRouter(fakePlayer(asAdmin()))
	w := postJSON(r, "/groups/not-a-uuid/members",
		`{"player_id":"`+uuid.New().String()+`","role":"member"}`)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateMember_InvalidGroupUUID(t *testing.T) {
	r := groupsRouter(fakePlayer(asAdmin()))
	playerID := uuid.New().String()
	w := doRequest(r, http.MethodPatch,
		"/groups/not-a-uuid/members/"+playerID, `{"role":"admin"}`)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRemoveMember_InvalidGroupUUID(t *testing.T) {
	r := groupsRouter(fakePlayer(asAdmin()))
	playerID := uuid.New().String()
	w := doRequest(r, http.MethodDelete,
		"/groups/not-a-uuid/members/"+playerID, "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ── groupStats ────────────────────────────────────────────────────────────────

func TestGroupStats_InvalidUUID(t *testing.T) {
	r := groupsRouter(fakePlayer())
	w := doRequest(r, http.MethodGet, "/groups/bad-id/stats", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}
