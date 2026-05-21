package integration_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRanking_GetRanking_NoAuth(t *testing.T) {
	srv := newTestServer(t)

	res := apiCall(t, srv, http.MethodGet, "/api/v2/ranking", "", nil)
	assert.Equal(t, http.StatusOK, res.Code)

	// Should return ranking array (may be empty in new DB)
	assert.Contains(t, res.Body, "items")
}

func TestRanking_GetRanking_WithPlayers(t *testing.T) {
	srv := newTestServer(t)

	// Register and login a player
	p := registerAndLogin(t, srv, "Test Player")
	enableApiV2(t, p.ID)

	res := apiCall(t, srv, http.MethodGet, "/api/v2/ranking", p.Token, nil)
	assert.Equal(t, http.StatusOK, res.Code)

	assert.Contains(t, res.Body, "items")
	assert.Contains(t, res.Body, "month")
	assert.Contains(t, res.Body, "year")
}

func TestRanking_GetRanking_WithGroupFilter(t *testing.T) {
	srv := newTestServer(t)
	p := registerAndLogin(t, srv, "Test Player")
	enableApiV2(t, p.ID)

	// Create a group
	groupRes := apiCall(t, srv, http.MethodPost, "/api/v2/groups", p.Token, map[string]any{
		"name": "Test Group",
	})
	assert.Equal(t, http.StatusCreated, groupRes.Code)

	groupID := groupRes.Body["id"].(string)

	// Get ranking for that group
	res := apiCall(t, srv, http.MethodGet, "/api/v2/ranking?groupId="+groupID, p.Token, nil)
	assert.Equal(t, http.StatusOK, res.Code)

	assert.Contains(t, res.Body, "items")
}

func TestRanking_GetRanking_InvalidGroupID(t *testing.T) {
	srv := newTestServer(t)
	p := registerAndLogin(t, srv, "Test Player")
	enableApiV2(t, p.ID)

	// Invalid UUID format
	res := apiCall(t, srv, http.MethodGet, "/api/v2/ranking?groupId=not-a-uuid", p.Token, nil)
	assert.Equal(t, http.StatusUnprocessableEntity, res.Code)
}
