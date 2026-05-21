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
		"whatsapp": "+5511999990001",
		"email":    "test@example.com",
	})
	assert.True(t, res.Code == http.StatusOK || res.Code == http.StatusCreated || res.Code == http.StatusConflict,
		"Expected 200/201/409, got %d", res.Code)
}

func TestBeta_AndroidSignup_MissingField(t *testing.T) {
	srv := newTestServer(t)

	// POST without whatsapp field
	res := apiCall(t, srv, http.MethodPost, "/api/v2/beta/android-signup", "", map[string]any{
		"email": "test@example.com",
	})
	assert.Equal(t, http.StatusUnprocessableEntity, res.Code)
}

func TestBeta_AndroidSignup_InvalidEmail(t *testing.T) {
	srv := newTestServer(t)

	// POST with invalid email format
	res := apiCall(t, srv, http.MethodPost, "/api/v2/beta/android-signup", "", map[string]any{
		"whatsapp": "+5511999990001",
		"email":    "not-an-email",
	})
	// Should reject invalid email
	assert.Equal(t, http.StatusUnprocessableEntity, res.Code)
}

func TestBeta_AndroidSignup_InvalidWhatsApp(t *testing.T) {
	srv := newTestServer(t)

	// POST with invalid WhatsApp format (missing +)
	res := apiCall(t, srv, http.MethodPost, "/api/v2/beta/android-signup", "", map[string]any{
		"whatsapp": "5511999990001",
		"email":    "test@example.com",
	})
	// Should reject invalid WhatsApp format
	assert.Equal(t, http.StatusUnprocessableEntity, res.Code)
}

func TestBeta_AndroidSignup_Duplicate(t *testing.T) {
	srv := newTestServer(t)

	whatsapp := "+5511999990002"
	email := "duplicate@example.com"

	// First signup
	res1 := apiCall(t, srv, http.MethodPost, "/api/v2/beta/android-signup", "", map[string]any{
		"whatsapp": whatsapp,
		"email":    email,
	})
	assert.True(t, res1.Code == http.StatusOK || res1.Code == http.StatusCreated)

	// Second signup with same WhatsApp
	res2 := apiCall(t, srv, http.MethodPost, "/api/v2/beta/android-signup", "", map[string]any{
		"whatsapp": whatsapp,
		"email":    "other@example.com",
	})
	// Should either return conflict or success (depending on implementation)
	assert.True(t, res2.Code == http.StatusConflict || res2.Code == http.StatusOK)
}
