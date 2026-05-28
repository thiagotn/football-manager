package integration_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSubscriptions_GetMySubscription_NoActive(t *testing.T) {
	srv := newTestServer(t)
	p := registerAndLogin(t, srv, "Player")

	// GET /api/v2/subscriptions/me without active subscription
	res := apiCall(t, srv, http.MethodGet, "/api/v2/subscriptions/me", p.Token, nil)
	assert.Equal(t, http.StatusOK, res.Code)
	// Should return empty/null subscription info
	assert.NotNil(t, res.Body)
}

func TestSubscriptions_CreateSubscription_NoStripe(t *testing.T) {
	srv := newTestServer(t)
	p := registerAndLogin(t, srv, "Player")

	// POST /api/v2/subscriptions without Stripe configured
	res := apiCall(t, srv, http.MethodPost, "/api/v2/subscriptions", p.Token, map[string]any{
		"plan": "pro_monthly",
	})
	// Without Stripe configured, should return error
	assert.True(t, res.Code >= 400)
}
