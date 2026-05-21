package integration_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInvites_CreateInvite_AsAdmin(t *testing.T) {
	srv := newTestServer(t)
	admin := registerAndLogin(t, srv, "Admin")
	makeAdmin(t, admin.ID)
	enableApiV2(t, admin.ID)

	// Create group
	groupRes := apiCall(t, srv, http.MethodPost, "/api/v2/groups", admin.Token, map[string]any{
		"name": "Test Group",
	})
	groupID := groupRes.Body["id"].(string)

	// POST /api/v2/invites to create invite
	res := apiCall(t, srv, http.MethodPost, "/api/v2/invites", admin.Token, map[string]any{
		"group_id": groupID,
	})
	assert.Equal(t, http.StatusCreated, res.Code)
	assert.Contains(t, res.Body, "token")
}

func TestInvites_CreateInvite_Forbidden(t *testing.T) {
	srv := newTestServer(t)
	player := registerAndLogin(t, srv, "Player")
	enableApiV2(t, player.ID)

	admin := registerAndLogin(t, srv, "Admin")
	makeAdmin(t, admin.ID)
	enableApiV2(t, admin.ID)

	// Create group as admin
	groupRes := apiCall(t, srv, http.MethodPost, "/api/v2/groups", admin.Token, map[string]any{
		"name": "Test Group",
	})
	groupID := groupRes.Body["id"].(string)

	// Regular player tries to create invite
	res := apiCall(t, srv, http.MethodPost, "/api/v2/invites", player.Token, map[string]any{
		"group_id": groupID,
	})
	assert.Equal(t, http.StatusForbidden, res.Code)
}

func TestInvites_GetInviteInfo_ValidToken(t *testing.T) {
	srv := newTestServer(t)
	admin := registerAndLogin(t, srv, "Admin")
	makeAdmin(t, admin.ID)
	enableApiV2(t, admin.ID)

	// Create group and invite
	groupRes := apiCall(t, srv, http.MethodPost, "/api/v2/groups", admin.Token, map[string]any{
		"name": "Test Group",
	})
	groupID := groupRes.Body["id"].(string)

	inviteRes := apiCall(t, srv, http.MethodPost, "/api/v2/invites", admin.Token, map[string]any{
		"group_id": groupID,
	})
	require.Equal(t, http.StatusCreated, inviteRes.Code)
	token := inviteRes.Body["token"].(string)

	// GET /api/v2/invites/{token}
	res := apiCall(t, srv, http.MethodGet, "/api/v2/invites/"+token, "", nil)
	assert.Equal(t, http.StatusOK, res.Code)
}

func TestInvites_GetInviteInfo_InvalidToken(t *testing.T) {
	srv := newTestServer(t)

	// GET /api/v2/invites/{token} with invalid token
	res := apiCall(t, srv, http.MethodGet, "/api/v2/invites/invalid-token", "", nil)
	assert.Equal(t, http.StatusNotFound, res.Code)
}

func TestInvites_CheckInvite_ValidToken(t *testing.T) {
	srv := newTestServer(t)
	admin := registerAndLogin(t, srv, "Admin")
	makeAdmin(t, admin.ID)
	enableApiV2(t, admin.ID)

	newPlayer := registerAndLogin(t, srv, "New Player")
	enableApiV2(t, newPlayer.ID)

	// Create group and invite
	groupRes := apiCall(t, srv, http.MethodPost, "/api/v2/groups", admin.Token, map[string]any{
		"name": "Test Group",
	})
	groupID := groupRes.Body["id"].(string)

	inviteRes := apiCall(t, srv, http.MethodPost, "/api/v2/invites", admin.Token, map[string]any{
		"group_id": groupID,
	})
	require.Equal(t, http.StatusCreated, inviteRes.Code)
	token := inviteRes.Body["token"].(string)

	// GET /api/v2/invites/{token}/check
	res := apiCall(t, srv, http.MethodGet, "/api/v2/invites/"+token+"/check", newPlayer.Token, nil)
	assert.Equal(t, http.StatusOK, res.Code)
}

func TestInvites_AcceptInvite_Success(t *testing.T) {
	srv := newTestServer(t)
	admin := registerAndLogin(t, srv, "Admin")
	makeAdmin(t, admin.ID)
	enableApiV2(t, admin.ID)

	newPlayer := registerAndLogin(t, srv, "New Player")
	enableApiV2(t, newPlayer.ID)

	// Create group and invite
	groupRes := apiCall(t, srv, http.MethodPost, "/api/v2/groups", admin.Token, map[string]any{
		"name": "Test Group",
	})
	groupID := groupRes.Body["id"].(string)

	inviteRes := apiCall(t, srv, http.MethodPost, "/api/v2/invites", admin.Token, map[string]any{
		"group_id": groupID,
	})
	require.Equal(t, http.StatusCreated, inviteRes.Code)
	token := inviteRes.Body["token"].(string)

	// POST /api/v2/invites/{token}/accept
	res := apiCall(t, srv, http.MethodPost, "/api/v2/invites/"+token+"/accept", newPlayer.Token, nil)
	assert.True(t, res.Code == http.StatusOK || res.Code == http.StatusCreated)
}

func TestInvites_AcceptInvite_Twice(t *testing.T) {
	srv := newTestServer(t)
	admin := registerAndLogin(t, srv, "Admin")
	makeAdmin(t, admin.ID)
	enableApiV2(t, admin.ID)

	newPlayer := registerAndLogin(t, srv, "New Player")
	enableApiV2(t, newPlayer.ID)

	// Create group and invite
	groupRes := apiCall(t, srv, http.MethodPost, "/api/v2/groups", admin.Token, map[string]any{
		"name": "Test Group",
	})
	groupID := groupRes.Body["id"].(string)

	inviteRes := apiCall(t, srv, http.MethodPost, "/api/v2/invites", admin.Token, map[string]any{
		"group_id": groupID,
	})
	require.Equal(t, http.StatusCreated, inviteRes.Code)
	token := inviteRes.Body["token"].(string)

	// Accept first time
	res1 := apiCall(t, srv, http.MethodPost, "/api/v2/invites/"+token+"/accept", newPlayer.Token, nil)
	assert.True(t, res1.Code == http.StatusOK || res1.Code == http.StatusCreated)

	// Try to accept second time
	res2 := apiCall(t, srv, http.MethodPost, "/api/v2/invites/"+token+"/accept", newPlayer.Token, nil)
	// Should return conflict since already accepted
	assert.True(t, res2.Code == http.StatusConflict || res2.Code == http.StatusBadRequest)
}
