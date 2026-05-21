package integration_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBeta_AndroidSignup_NoAuth(t *testing.T) {
	srv := newTestServer(t)

	payload := map[string]any{
		"email": "user@example.com",
	}

	res := apiCall(t, srv, http.MethodPost, "/api/v2/beta/android-signup", "", payload)
	// Public route, should work without auth
	assert.True(t, res.Code == http.StatusOK || res.Code == http.StatusCreated)
}

func TestBeta_AndroidSignup_WithAuth(t *testing.T) {
	srv := newTestServer(t)
	p := registerAndLogin(t, srv, "Test Player")

	payload := map[string]any{
		"email": p.WhatsApp + "@example.com",
	}

	res := apiCall(t, srv, http.MethodPost, "/api/v2/beta/android-signup", p.Token, payload)
	assert.True(t, res.Code == http.StatusOK || res.Code == http.StatusCreated)

	body := res.Body
	// Should return some confirmation
	assert.NotNil(t, body)
}

func TestBeta_AndroidSignup_InvalidPayload(t *testing.T) {
	srv := newTestServer(t)

	payload := map[string]any{
		// missing email
	}

	res := apiCall(t, srv, http.MethodPost, "/api/v2/beta/android-signup", "", payload)
	assert.Equal(t, http.StatusUnprocessableEntity, res.Code)
}

func TestBeta_AndroidSignup_InvalidEmail(t *testing.T) {
	srv := newTestServer(t)

	payload := map[string]any{
		"email": "not-an-email",
	}

	res := apiCall(t, srv, http.MethodPost, "/api/v2/beta/android-signup", "", payload)
	assert.Equal(t, http.StatusUnprocessableEntity, res.Code)
}

func TestBeta_AndroidSignup_DuplicateSignup(t *testing.T) {
	srv := newTestServer(t)

	email := "test@example.com"
	payload := map[string]any{
		"email": email,
	}

	// First signup
	res1 := apiCall(t, srv, http.MethodPost, "/api/v2/beta/android-signup", "", payload)
	assert.True(t, res1.Code == http.StatusOK || res1.Code == http.StatusCreated)

	// Second signup with same email
	res2 := apiCall(t, srv, http.MethodPost, "/api/v2/beta/android-signup", "", payload)
	// May return 200 (idempotent) or 409 (conflict)
	assert.True(t, res2.Code == http.StatusOK || res2.Code == http.StatusCreated || res2.Code == http.StatusConflict)
}
