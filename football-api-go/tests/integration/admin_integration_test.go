package integration_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAdmin_GetStats_AsAdmin(t *testing.T) {
	srv := newTestServer(t)
	admin := registerAndLogin(t, srv, "Admin")
	makeAdmin(t, admin.ID)

	// GET /api/v2/admin/stats
	res := apiCall(t, srv, http.MethodGet, "/api/v2/admin/stats", admin.Token, nil)
	assert.Equal(t, http.StatusOK, res.Code)
	assert.NotNil(t, res.Body)
}

func TestAdmin_GetStats_Forbidden(t *testing.T) {
	srv := newTestServer(t)
	player := registerAndLogin(t, srv, "Player")

	// GET /api/v2/admin/stats as regular player
	res := apiCall(t, srv, http.MethodGet, "/api/v2/admin/stats", player.Token, nil)
	assert.Equal(t, http.StatusForbidden, res.Code)
}

func TestAdmin_ListPlayers_AsAdmin(t *testing.T) {
	srv := newTestServer(t)
	admin := registerAndLogin(t, srv, "Admin")
	makeAdmin(t, admin.ID)

	// GET /api/v2/admin/players
	res := apiCall(t, srv, http.MethodGet, "/api/v2/admin/players", admin.Token, nil)
	assert.Equal(t, http.StatusOK, res.Code)
	assert.NotNil(t, res.Body)
	assert.NotNil(t, res.Body["items"])
}

func TestAdmin_ListGroups_AsAdmin(t *testing.T) {
	srv := newTestServer(t)
	admin := registerAndLogin(t, srv, "Admin")
	makeAdmin(t, admin.ID)

	// GET /api/v2/admin/groups
	res := apiCall(t, srv, http.MethodGet, "/api/v2/admin/groups", admin.Token, nil)
	assert.Equal(t, http.StatusOK, res.Code)
	assert.NotNil(t, res.Body)
	assert.NotNil(t, res.Body["items"])
}

func TestAdmin_ListMatches_AsAdmin(t *testing.T) {
	srv := newTestServer(t)
	admin := registerAndLogin(t, srv, "Admin")
	makeAdmin(t, admin.ID)

	// GET /api/v2/admin/matches
	res := apiCall(t, srv, http.MethodGet, "/api/v2/admin/matches", admin.Token, nil)
	assert.Equal(t, http.StatusOK, res.Code)
	assert.NotNil(t, res.Body)
	assert.NotNil(t, res.Body["items"])
}

func TestAdmin_ListBetaSignups_AsAdmin(t *testing.T) {
	srv := newTestServer(t)
	admin := registerAndLogin(t, srv, "Admin")
	makeAdmin(t, admin.ID)

	// GET /api/v2/admin/beta-signups
	res := apiCall(t, srv, http.MethodGet, "/api/v2/admin/beta-signups", admin.Token, nil)
	assert.Equal(t, http.StatusOK, res.Code)
	assert.NotNil(t, res.Body)
	assert.NotNil(t, res.Body["items"])
}

func TestAdmin_GetChatUsers_AsAdmin(t *testing.T) {
	srv := newTestServer(t)
	admin := registerAndLogin(t, srv, "Admin")
	makeAdmin(t, admin.ID)

	// GET /api/v2/admin/chat-users
	res := apiCall(t, srv, http.MethodGet, "/api/v2/admin/chat-users", admin.Token, nil)
	assert.Equal(t, http.StatusOK, res.Code)
	assert.NotNil(t, res.Body)
	assert.NotNil(t, res.Body["users"])
}

func TestAdmin_ToggleChatAccess_AsAdmin(t *testing.T) {
	srv := newTestServer(t)
	admin := registerAndLogin(t, srv, "Admin")
	makeAdmin(t, admin.ID)

	player := registerAndLogin(t, srv, "Player")

	// PATCH /api/v2/admin/chat-users/{id}
	res := apiCall(t, srv, http.MethodPatch, "/api/v2/admin/chat-users/"+player.ID, admin.Token, map[string]any{
		"chat_enabled": true,
	})
	assert.Equal(t, http.StatusOK, res.Code)
}
