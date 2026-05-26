package integration_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPlayers_GetMe_Success(t *testing.T) {
	srv := newTestServer(t)
	p := registerAndLogin(t, srv, "Test Player")
	enableApiV2(t, p.ID)

	// GET /api/v2/players/me
	res := apiCall(t, srv, http.MethodGet, "/api/v2/players/me", p.Token, nil)
	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, p.ID, res.Body["id"])
}

func TestPlayers_GetMe_NoAuth(t *testing.T) {
	srv := newTestServer(t)

	// GET /api/v2/players/me without auth
	res := apiCall(t, srv, http.MethodGet, "/api/v2/players/me", "", nil)
	assert.Equal(t, http.StatusUnauthorized, res.Code)
}

func TestPlayers_GetPlayerByID_Success(t *testing.T) {
	srv := newTestServer(t)
	p := registerAndLogin(t, srv, "Test Player")
	enableApiV2(t, p.ID)

	// GET /api/v2/players/{id} — player fetching their own profile
	res := apiCall(t, srv, http.MethodGet, "/api/v2/players/"+p.ID, p.Token, nil)
	assert.Equal(t, http.StatusOK, res.Code)
}

func TestPlayers_GetPlayerByID_NotFound(t *testing.T) {
	srv := newTestServer(t)
	admin := registerAndLogin(t, srv, "Admin")
	makeAdmin(t, admin.ID)
	enableApiV2(t, admin.ID)

	// GET /api/v2/players/{id} for non-existent player (admin can access any player)
	res := apiCall(t, srv, http.MethodGet, "/api/v2/players/00000000-0000-0000-0000-000000000000", admin.Token, nil)
	assert.Equal(t, http.StatusNotFound, res.Code)
}

func TestPlayers_GetPlayerByID_InvalidUUID(t *testing.T) {
	srv := newTestServer(t)
	p := registerAndLogin(t, srv, "Test Player")
	enableApiV2(t, p.ID)

	// GET /api/v2/players/{id} with invalid UUID
	res := apiCall(t, srv, http.MethodGet, "/api/v2/players/invalid-uuid", p.Token, nil)
	assert.Equal(t, http.StatusUnprocessableEntity, res.Code)
}

func TestPlayers_UpdatePlayer_Success(t *testing.T) {
	srv := newTestServer(t)
	p := registerAndLogin(t, srv, "Test Player")
	enableApiV2(t, p.ID)

	// PATCH /api/v2/players/{id}
	res := apiCall(t, srv, http.MethodPatch, "/api/v2/players/"+p.ID, p.Token, map[string]any{
		"name": "Updated Name",
	})
	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, "Updated Name", res.Body["name"])
}

func TestPlayers_UpdatePlayer_Forbidden(t *testing.T) {
	srv := newTestServer(t)
	p1 := registerAndLogin(t, srv, "Player 1")
	enableApiV2(t, p1.ID)
	p2 := registerAndLogin(t, srv, "Player 2")
	enableApiV2(t, p2.ID)

	// Player 2 tries to update Player 1's profile
	res := apiCall(t, srv, http.MethodPatch, "/api/v2/players/"+p1.ID, p2.Token, map[string]any{
		"name": "Hacked Name",
	})
	assert.Equal(t, http.StatusForbidden, res.Code)
}

func TestPlayers_ListPlayers_AdminOnly(t *testing.T) {
	srv := newTestServer(t)
	p := registerAndLogin(t, srv, "Player")
	enableApiV2(t, p.ID)

	// Regular player tries to list players
	res := apiCall(t, srv, http.MethodGet, "/api/v2/players", p.Token, nil)
	assert.Equal(t, http.StatusForbidden, res.Code)
}

func TestPlayers_ListPlayers_AsAdmin(t *testing.T) {
	srv := newTestServer(t)
	admin := registerAndLogin(t, srv, "Admin")
	makeAdmin(t, admin.ID)
	enableApiV2(t, admin.ID)

	// Admin lists players
	res := apiCall(t, srv, http.MethodGet, "/api/v2/players", admin.Token, nil)
	assert.Equal(t, http.StatusOK, res.Code)
	assert.NotNil(t, res.List)
}

func TestPlayers_ResetPassword_AsAdmin(t *testing.T) {
	srv := newTestServer(t)
	admin := registerAndLogin(t, srv, "Admin")
	makeAdmin(t, admin.ID)
	enableApiV2(t, admin.ID)

	player := registerAndLogin(t, srv, "Player")

	// Admin resets player's password
	res := apiCall(t, srv, http.MethodPost, "/api/v2/players/"+player.ID+"/reset-password", admin.Token, map[string]any{
		"new_password": "newpassword123",
	})
	assert.True(t, res.Code == http.StatusOK || res.Code == http.StatusNoContent)
}

func TestPlayers_ResetPassword_Forbidden(t *testing.T) {
	srv := newTestServer(t)
	p1 := registerAndLogin(t, srv, "Player 1")
	enableApiV2(t, p1.ID)
	p2 := registerAndLogin(t, srv, "Player 2")

	// Player 1 tries to reset Player 2's password
	res := apiCall(t, srv, http.MethodPost, "/api/v2/players/"+p2.ID+"/reset-password", p1.Token, map[string]any{
		"new_password": "newpassword123",
	})
	assert.Equal(t, http.StatusForbidden, res.Code)
}

func TestPlayers_GetPlayerStats_Success(t *testing.T) {
	srv := newTestServer(t)
	p := registerAndLogin(t, srv, "Player")
	enableApiV2(t, p.ID)

	// GET /api/v2/players/me/stats
	res := apiCall(t, srv, http.MethodGet, "/api/v2/players/me/stats", p.Token, nil)
	assert.Equal(t, http.StatusOK, res.Code)
	assert.NotNil(t, res.Body)
}

func TestPlayers_GetPlayerMatches_Success(t *testing.T) {
	srv := newTestServer(t)
	p := registerAndLogin(t, srv, "Player")
	enableApiV2(t, p.ID)

	// GET /api/v2/players/me/matches
	res := apiCall(t, srv, http.MethodGet, "/api/v2/players/me/matches", p.Token, nil)
	assert.Equal(t, http.StatusOK, res.Code)
	assert.NotNil(t, res.List)
}
