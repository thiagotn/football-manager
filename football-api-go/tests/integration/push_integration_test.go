package integration_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPush_GetVapidKey_NoAuth(t *testing.T) {
	srv := newTestServer(t)

	res := apiCall(t, srv, http.MethodGet, "/api/v2/push/vapid-public-key", "", nil)
	assert.Equal(t, http.StatusOK, res.Code)

	// VAPID key may be empty in test (not configured)
	assert.Contains(t, res.Body, "public_key")
}

func TestPush_Subscribe_ValidPayload(t *testing.T) {
	srv := newTestServer(t)
	p := registerAndLogin(t, srv, "Test Player")
	enableApiV2(t, p.ID)

	payload := map[string]any{
		"endpoint": "https://example.com/push",
		"keys": map[string]string{
			"p256dh": "valid-p256dh-key",
			"auth":   "valid-auth-key",
		},
	}

	res := apiCall(t, srv, http.MethodPost, "/api/v2/push/subscribe", p.Token, payload)
	assert.True(t, res.Code == http.StatusOK || res.Code == http.StatusCreated)

	assert.NotEmpty(t, res.Body["id"])
}

func TestPush_Subscribe_InvalidPayload(t *testing.T) {
	srv := newTestServer(t)
	p := registerAndLogin(t, srv, "Test Player")
	enableApiV2(t, p.ID)

	// Missing required fields
	payload := map[string]any{
		"endpoint": "https://example.com/push",
		// missing keys
	}

	res := apiCall(t, srv, http.MethodPost, "/api/v2/push/subscribe", p.Token, payload)
	assert.Equal(t, http.StatusUnprocessableEntity, res.Code)
}

func TestPush_Subscribe_RequiresAuth(t *testing.T) {
	srv := newTestServer(t)

	payload := map[string]any{
		"endpoint": "https://example.com/push",
		"keys": map[string]string{
			"p256dh": "key",
			"auth":   "key",
		},
	}

	res := apiCall(t, srv, http.MethodPost, "/api/v2/push/subscribe", "", payload)
	assert.Equal(t, http.StatusUnauthorized, res.Code)
}

func TestPush_Unsubscribe_NoSubscription(t *testing.T) {
	srv := newTestServer(t)
	p := registerAndLogin(t, srv, "Test Player")
	enableApiV2(t, p.ID)

	res := apiCall(t, srv, http.MethodDelete, "/api/v2/push/subscribe", p.Token, nil)
	// May return 404 or 204 depending on implementation
	assert.True(t, res.Code == http.StatusNotFound || res.Code == http.StatusNoContent)
}

func TestPush_Unsubscribe_AfterSubscribe(t *testing.T) {
	srv := newTestServer(t)
	p := registerAndLogin(t, srv, "Test Player")
	enableApiV2(t, p.ID)

	// Subscribe first
	payload := map[string]any{
		"endpoint": "https://example.com/push",
		"keys": map[string]string{
			"p256dh": "key1",
			"auth":   "key1",
		},
	}
	apiCall(t, srv, http.MethodPost, "/api/v2/push/subscribe", p.Token, payload)

	// Then unsubscribe
	res := apiCall(t, srv, http.MethodDelete, "/api/v2/push/subscribe", p.Token, nil)
	assert.True(t, res.Code == http.StatusOK || res.Code == http.StatusNoContent)
}
