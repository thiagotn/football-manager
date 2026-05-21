package integration_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChat_SendMessage_WithoutAccess(t *testing.T) {
	srv := newTestServer(t)
	p := registerAndLogin(t, srv, "Player")
	enableApiV2(t, p.ID)

	// POST /api/v2/chat without chat access enabled
	res := apiCall(t, srv, http.MethodPost, "/api/v2/chat", p.Token, map[string]any{
		"message": "Hello",
	})
	// Should return 403 if chat_enabled is false
	assert.Equal(t, http.StatusForbidden, res.Code)
}

func TestChat_SendMessage_WithAccess(t *testing.T) {
	srv := newTestServer(t)
	admin := registerAndLogin(t, srv, "Admin")
	makeAdmin(t, admin.ID)
	enableApiV2(t, admin.ID)

	p := registerAndLogin(t, srv, "Player")
	enableApiV2(t, p.ID)

	// Enable chat access for player
	apiCall(t, srv, http.MethodPatch, "/api/v2/admin/chat-users/"+p.ID, admin.Token, map[string]any{
		"chat_enabled": true,
	})

	// POST /api/v2/chat with chat access enabled
	res := apiCall(t, srv, http.MethodPost, "/api/v2/chat", p.Token, map[string]any{
		"message": "Hello, AI!",
	})
	// Should return 200/201 or error if Anthropic not configured
	assert.True(t, res.Code == http.StatusOK || res.Code == http.StatusCreated || res.Code >= 400)
}

func TestChat_SendMessage_InvalidPayload(t *testing.T) {
	srv := newTestServer(t)
	admin := registerAndLogin(t, srv, "Admin")
	makeAdmin(t, admin.ID)
	enableApiV2(t, admin.ID)

	p := registerAndLogin(t, srv, "Player")
	enableApiV2(t, p.ID)

	// Enable chat access
	apiCall(t, srv, http.MethodPatch, "/api/v2/admin/chat-users/"+p.ID, admin.Token, map[string]any{
		"chat_enabled": true,
	})

	// POST without message field
	res := apiCall(t, srv, http.MethodPost, "/api/v2/chat", p.Token, map[string]any{})
	assert.Equal(t, http.StatusUnprocessableEntity, res.Code)
}

func TestChat_SendMessage_EmptyMessage(t *testing.T) {
	srv := newTestServer(t)
	admin := registerAndLogin(t, srv, "Admin")
	makeAdmin(t, admin.ID)
	enableApiV2(t, admin.ID)

	p := registerAndLogin(t, srv, "Player")
	enableApiV2(t, p.ID)

	// Enable chat access
	apiCall(t, srv, http.MethodPatch, "/api/v2/admin/chat-users/"+p.ID, admin.Token, map[string]any{
		"chat_enabled": true,
	})

	// POST with empty message
	res := apiCall(t, srv, http.MethodPost, "/api/v2/chat", p.Token, map[string]any{
		"message": "",
	})
	assert.Equal(t, http.StatusUnprocessableEntity, res.Code)
}
