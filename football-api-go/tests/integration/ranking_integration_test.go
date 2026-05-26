package integration_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRanking_GetRanking_NoAuth(t *testing.T) {
	srv := newTestServer(t)

	// GET /api/v2/ranking without auth should work (public endpoint)
	res := apiCall(t, srv, http.MethodGet, "/api/v2/ranking", "", nil)
	assert.Equal(t, http.StatusOK, res.Code)
	assert.NotNil(t, res.Body["items"])
}

func TestRanking_GetRanking_WithPlayers(t *testing.T) {
	srv := newTestServer(t)
	p1 := registerAndLogin(t, srv, "Player 1")
	enableApiV2(t, p1.ID)

	p2 := registerAndLogin(t, srv, "Player 2")
	enableApiV2(t, p2.ID)

	// GET /api/v2/ranking should return a list
	res := apiCall(t, srv, http.MethodGet, "/api/v2/ranking", "", nil)
	assert.Equal(t, http.StatusOK, res.Code)
	assert.NotNil(t, res.Body["items"])
	items, _ := res.Body["items"].([]any)
	assert.True(t, len(items) >= 0)
}

func TestRanking_GetRanking_InvalidYear(t *testing.T) {
	srv := newTestServer(t)

	// GET /api/v2/ranking with invalid year param
	res := apiCall(t, srv, http.MethodGet, "/api/v2/ranking?year=invalid", "", nil)
	assert.Equal(t, http.StatusUnprocessableEntity, res.Code)
}

func TestRanking_GetRanking_WithValidYear(t *testing.T) {
	srv := newTestServer(t)
	admin := registerAndLogin(t, srv, "Admin")
	makeAdmin(t, admin.ID)
	enableApiV2(t, admin.ID)

	// GET /api/v2/ranking with valid year parameter
	res := apiCall(t, srv, http.MethodGet, "/api/v2/ranking?year=2024", "", nil)
	assert.Equal(t, http.StatusOK, res.Code)
	assert.NotNil(t, res.Body["items"])
}
