package integration_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPush_GetVapidPublicKey_NoAuth(t *testing.T) {
	srv := newTestServer(t)

	// GET /api/v2/push/vapid-public-key without auth (public endpoint)
	res := apiCall(t, srv, http.MethodGet, "/api/v2/push/vapid-public-key", "", nil)
	assert.Equal(t, http.StatusOK, res.Code)
	// Body should contain key string or empty if not configured
	assert.NotNil(t, res.Body)
}

func TestPush_Subscribe_ValidPayload(t *testing.T) {
	srv := newTestServer(t)
	p := registerAndLogin(t, srv, "Player")
	enableApiV2(t, p.ID)

	// POST /api/v2/push/subscribe with valid payload
	res := apiCall(t, srv, http.MethodPost, "/api/v2/push/subscribe", p.Token, map[string]any{
		"endpoint": "https://example.com/push/abc123",
		"auth_key": "auth_secret_key",
		"p256_key": "p256_secret_key",
	})
	// Should succeed or return 201/200
	assert.True(t, res.Code == http.StatusOK || res.Code == http.StatusCreated)
}

func TestPush_Subscribe_MissingEndpoint(t *testing.T) {
	srv := newTestServer(t)
	p := registerAndLogin(t, srv, "Player")
	enableApiV2(t, p.ID)

	// POST without endpoint field
	res := apiCall(t, srv, http.MethodPost, "/api/v2/push/subscribe", p.Token, map[string]any{
		"auth_key": "auth_secret_key",
		"p256_key": "p256_secret_key",
	})
	assert.Equal(t, http.StatusUnprocessableEntity, res.Code)
}

func TestPush_Subscribe_InvalidPayload(t *testing.T) {
	srv := newTestServer(t)
	p := registerAndLogin(t, srv, "Player")
	enableApiV2(t, p.ID)

	// POST with invalid/empty fields
	res := apiCall(t, srv, http.MethodPost, "/api/v2/push/subscribe", p.Token, map[string]any{
		"endpoint": "",
		"auth_key": "",
		"p256_key": "",
	})
	assert.Equal(t, http.StatusUnprocessableEntity, res.Code)
}

func TestPush_Unsubscribe_AfterSubscribe(t *testing.T) {
	srv := newTestServer(t)
	p := registerAndLogin(t, srv, "Player")
	enableApiV2(t, p.ID)

	// Subscribe first
	subRes := apiCall(t, srv, http.MethodPost, "/api/v2/push/subscribe", p.Token, map[string]any{
		"endpoint": "https://example.com/push/abc123",
		"auth_key": "auth_secret_key",
		"p256_key": "p256_secret_key",
	})
	assert.True(t, subRes.Code == http.StatusOK || subRes.Code == http.StatusCreated)

	// DELETE /api/v2/push/subscribe to unsubscribe
	unsubRes := apiCall(t, srv, http.MethodDelete, "/api/v2/push/subscribe", p.Token, nil)
	assert.True(t, unsubRes.Code == http.StatusNoContent || unsubRes.Code == http.StatusOK)
}

func TestPush_Unsubscribe_WithoutSubscription(t *testing.T) {
	srv := newTestServer(t)
	p := registerAndLogin(t, srv, "Player")
	enableApiV2(t, p.ID)

	// DELETE without prior subscription
	res := apiCall(t, srv, http.MethodDelete, "/api/v2/push/subscribe", p.Token, nil)
	// Should return 404 or 204 (depending on implementation)
	assert.True(t, res.Code == http.StatusNotFound || res.Code == http.StatusNoContent)
}
