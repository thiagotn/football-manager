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

	// Create group with unique name
	groupRes := apiCall(t, srv, http.MethodPost, "/api/v2/groups", admin.Token, map[string]any{
		"name": "Invites Test Group " + admin.ID,
	})
	assert.Equal(t, http.StatusCreated, groupRes.Code)
	groupID, ok := groupRes.Body["id"].(string)
	assert.True(t, ok)

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

	admin := registerAndLogin(t, srv, "Admin")
	makeAdmin(t, admin.ID)

	// Create group as admin with unique name
	groupRes := apiCall(t, srv, http.MethodPost, "/api/v2/groups", admin.Token, map[string]any{
		"name": "Invites Forbidden Test " + admin.ID,
	})
	assert.Equal(t, http.StatusCreated, groupRes.Code)
	groupID, ok := groupRes.Body["id"].(string)
	assert.True(t, ok)

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

	// Create group and invite with unique name
	groupRes := apiCall(t, srv, http.MethodPost, "/api/v2/groups", admin.Token, map[string]any{
		"name": "Invites Info Test " + admin.ID,
	})
	assert.Equal(t, http.StatusCreated, groupRes.Code)
	groupID, ok := groupRes.Body["id"].(string)
	assert.True(t, ok)

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

	newPlayer := registerAndLogin(t, srv, "New Player")

	// Create group and invite with unique name
	groupRes := apiCall(t, srv, http.MethodPost, "/api/v2/groups", admin.Token, map[string]any{
		"name": "Invites Info Test " + admin.ID,
	})
	assert.Equal(t, http.StatusCreated, groupRes.Code)
	groupID, ok := groupRes.Body["id"].(string)
	assert.True(t, ok)

	inviteRes := apiCall(t, srv, http.MethodPost, "/api/v2/invites", admin.Token, map[string]any{
		"group_id": groupID,
	})
	require.Equal(t, http.StatusCreated, inviteRes.Code)
	token := inviteRes.Body["token"].(string)

	// GET /api/v2/invites/{token}/check
	res := apiCall(t, srv, http.MethodGet, "/api/v2/invites/"+token+"/check?whatsapp="+newPlayer.WhatsApp, newPlayer.Token, nil)
	assert.Equal(t, http.StatusOK, res.Code)
}

func TestInvites_AcceptInvite_Success(t *testing.T) {
	srv := newTestServer(t)
	admin := registerAndLogin(t, srv, "Admin")
	makeAdmin(t, admin.ID)

	newPlayer := registerAndLogin(t, srv, "New Player")

	// Create group and invite with unique name
	groupRes := apiCall(t, srv, http.MethodPost, "/api/v2/groups", admin.Token, map[string]any{
		"name": "Invites Info Test " + admin.ID,
	})
	assert.Equal(t, http.StatusCreated, groupRes.Code)
	groupID, ok := groupRes.Body["id"].(string)
	assert.True(t, ok)

	inviteRes := apiCall(t, srv, http.MethodPost, "/api/v2/invites", admin.Token, map[string]any{
		"group_id": groupID,
	})
	require.Equal(t, http.StatusCreated, inviteRes.Code)
	token := inviteRes.Body["token"].(string)

	// POST /api/v2/invites/{token}/accept
	res := apiCall(t, srv, http.MethodPost, "/api/v2/invites/"+token+"/accept", "", map[string]any{
		"whatsapp": newPlayer.WhatsApp,
		"password": newPlayer.Password,
	})
	assert.True(t, res.Code == http.StatusOK || res.Code == http.StatusCreated)
}

func TestInvites_AcceptInvite_Twice(t *testing.T) {
	srv := newTestServer(t)
	admin := registerAndLogin(t, srv, "Admin")
	makeAdmin(t, admin.ID)

	newPlayer := registerAndLogin(t, srv, "New Player")

	// Create group and invite with unique name
	groupRes := apiCall(t, srv, http.MethodPost, "/api/v2/groups", admin.Token, map[string]any{
		"name": "Invites Info Test " + admin.ID,
	})
	assert.Equal(t, http.StatusCreated, groupRes.Code)
	groupID, ok := groupRes.Body["id"].(string)
	assert.True(t, ok)

	inviteRes := apiCall(t, srv, http.MethodPost, "/api/v2/invites", admin.Token, map[string]any{
		"group_id": groupID,
	})
	require.Equal(t, http.StatusCreated, inviteRes.Code)
	token := inviteRes.Body["token"].(string)

	// Accept first time
	res1 := apiCall(t, srv, http.MethodPost, "/api/v2/invites/"+token+"/accept", "", map[string]any{
		"whatsapp": newPlayer.WhatsApp,
		"password": newPlayer.Password,
	})
	assert.True(t, res1.Code == http.StatusOK || res1.Code == http.StatusCreated)

	// Try to accept second time
	res2 := apiCall(t, srv, http.MethodPost, "/api/v2/invites/"+token+"/accept", "", map[string]any{
		"whatsapp": newPlayer.WhatsApp,
		"password": newPlayer.Password,
	})
	// Should fail — invite already used (404 = invalid/used, 409 = already member, 400 = bad request)
	assert.True(t, res2.Code == http.StatusConflict || res2.Code == http.StatusBadRequest || res2.Code == http.StatusNotFound)
}
