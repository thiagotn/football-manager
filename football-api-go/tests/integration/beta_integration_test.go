package integration_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBeta_AndroidSignup_ValidPayload(t *testing.T) {
	srv := newTestServer(t)

	// POST /api/v2/beta/android-signup with valid payload (public endpoint)
	res := apiCall(t, srv, http.MethodPost, "/api/v2/beta/android-signup", "", map[string]any{
		"google_email": "test@example.com",
	})
	assert.True(t, res.Code == http.StatusCreated,
		"Expected 201, got %d", res.Code)
}

func TestBeta_AndroidSignup_MissingField(t *testing.T) {
	srv := newTestServer(t)

	// POST without google_email field
	res := apiCall(t, srv, http.MethodPost, "/api/v2/beta/android-signup", "", map[string]any{})
	assert.Equal(t, http.StatusUnprocessableEntity, res.Code)
}

func TestBeta_AndroidSignup_InvalidEmail(t *testing.T) {
	srv := newTestServer(t)

	// POST with invalid email format
	res := apiCall(t, srv, http.MethodPost, "/api/v2/beta/android-signup", "", map[string]any{
		"google_email": "not-an-email",
	})
	// Should reject invalid email
	assert.Equal(t, http.StatusUnprocessableEntity, res.Code)
}

func TestBeta_AndroidSignup_EmptyEmail(t *testing.T) {
	srv := newTestServer(t)

	// POST with empty google_email
	res := apiCall(t, srv, http.MethodPost, "/api/v2/beta/android-signup", "", map[string]any{
		"google_email": "",
	})
	// Should reject empty email
	assert.Equal(t, http.StatusUnprocessableEntity, res.Code)
}

func TestBeta_AndroidSignup_Duplicate(t *testing.T) {
	srv := newTestServer(t)

	email := "duplicate@example.com"

	// First signup
	res1 := apiCall(t, srv, http.MethodPost, "/api/v2/beta/android-signup", "", map[string]any{
		"google_email": email,
	})
	assert.Equal(t, http.StatusCreated, res1.Code)

	// Second signup with same email — should succeed (endpoint doesn't reject duplicates)
	res2 := apiCall(t, srv, http.MethodPost, "/api/v2/beta/android-signup", "", map[string]any{
		"google_email": email,
	})
	assert.Equal(t, http.StatusCreated, res2.Code)
}
