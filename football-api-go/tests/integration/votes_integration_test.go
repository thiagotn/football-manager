package integration_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVotes_GetVoteStatus_Before(t *testing.T) {
	srv := newTestServer(t)
	admin := registerAndLogin(t, srv, "Admin")
	makeAdmin(t, admin.ID)
	enableApiV2(t, admin.ID)

	player := registerAndLogin(t, srv, "Player")
	enableApiV2(t, player.ID)

	// Create group and match
	groupRes := apiCall(t, srv, http.MethodPost, "/api/v2/groups", admin.Token, map[string]any{
		"name": "Test Group",
	})
	groupID := groupRes.Body["id"].(string)

	matchRes := apiCall(t, srv, http.MethodPost, "/api/v2/groups/"+groupID+"/matches", admin.Token, map[string]any{
		"matchDate": "2099-12-31",
		"startTime": "18:00:00",
		"location":  "Test Court",
	})
	matchID := matchRes.Body["id"].(string)

	// GET /api/v2/matches/{id}/votes/status
	res := apiCall(t, srv, http.MethodGet, "/api/v2/matches/"+matchID+"/votes/status", player.Token, nil)
	assert.Equal(t, http.StatusOK, res.Code)
}

func TestVotes_CreateVote_Success(t *testing.T) {
	srv := newTestServer(t)
	admin := registerAndLogin(t, srv, "Admin")
	makeAdmin(t, admin.ID)
	enableApiV2(t, admin.ID)

	player := registerAndLogin(t, srv, "Player")
	enableApiV2(t, player.ID)

	// Create group and match
	groupRes := apiCall(t, srv, http.MethodPost, "/api/v2/groups", admin.Token, map[string]any{
		"name": "Test Group",
	})
	groupID := groupRes.Body["id"].(string)

	matchRes := apiCall(t, srv, http.MethodPost, "/api/v2/groups/"+groupID+"/matches", admin.Token, map[string]any{
		"matchDate": "2099-12-31",
		"startTime": "18:00:00",
		"location":  "Test Court",
	})
	matchID := matchRes.Body["id"].(string)

	// POST /api/v2/matches/{id}/votes
	res := apiCall(t, srv, http.MethodPost, "/api/v2/matches/"+matchID+"/votes", player.Token, map[string]any{
		"rating": 5,
		"comment": "Great match!",
	})
	assert.True(t, res.Code == http.StatusOK || res.Code == http.StatusCreated)
}

func TestVotes_CreateVote_InvalidRating(t *testing.T) {
	srv := newTestServer(t)
	admin := registerAndLogin(t, srv, "Admin")
	makeAdmin(t, admin.ID)
	enableApiV2(t, admin.ID)

	player := registerAndLogin(t, srv, "Player")
	enableApiV2(t, player.ID)

	// Create group and match
	groupRes := apiCall(t, srv, http.MethodPost, "/api/v2/groups", admin.Token, map[string]any{
		"name": "Test Group",
	})
	groupID := groupRes.Body["id"].(string)

	matchRes := apiCall(t, srv, http.MethodPost, "/api/v2/groups/"+groupID+"/matches", admin.Token, map[string]any{
		"matchDate": "2099-12-31",
		"startTime": "18:00:00",
		"location":  "Test Court",
	})
	matchID := matchRes.Body["id"].(string)

	// POST with invalid rating
	res := apiCall(t, srv, http.MethodPost, "/api/v2/matches/"+matchID+"/votes", player.Token, map[string]any{
		"rating": 10,
		"comment": "Invalid",
	})
	assert.Equal(t, http.StatusUnprocessableEntity, res.Code)
}

func TestVotes_GetPendingVotes(t *testing.T) {
	srv := newTestServer(t)
	p := registerAndLogin(t, srv, "Player")
	enableApiV2(t, p.ID)

	// GET /api/v2/votes/pending
	res := apiCall(t, srv, http.MethodGet, "/api/v2/votes/pending", p.Token, nil)
	assert.Equal(t, http.StatusOK, res.Code)
	assert.NotNil(t, res.List)
}

func TestVotes_GetVoteResults_BeforeClosing(t *testing.T) {
	srv := newTestServer(t)
	admin := registerAndLogin(t, srv, "Admin")
	makeAdmin(t, admin.ID)
	enableApiV2(t, admin.ID)

	// Create group and match
	groupRes := apiCall(t, srv, http.MethodPost, "/api/v2/groups", admin.Token, map[string]any{
		"name": "Test Group",
	})
	groupID := groupRes.Body["id"].(string)

	matchRes := apiCall(t, srv, http.MethodPost, "/api/v2/groups/"+groupID+"/matches", admin.Token, map[string]any{
		"matchDate": "2099-12-31",
		"startTime": "18:00:00",
		"location":  "Test Court",
	})
	matchID := matchRes.Body["id"].(string)

	// GET /api/v2/matches/{id}/votes/results before closing
	res := apiCall(t, srv, http.MethodGet, "/api/v2/matches/"+matchID+"/votes/results", admin.Token, nil)
	// Might return 403 if not time to see results, or 200 with partial data
	assert.True(t, res.Code == http.StatusOK || res.Code == http.StatusForbidden)
}

func TestVotes_CloseVoting_AsAdmin(t *testing.T) {
	srv := newTestServer(t)
	admin := registerAndLogin(t, srv, "Admin")
	makeAdmin(t, admin.ID)
	enableApiV2(t, admin.ID)

	// Create group and match
	groupRes := apiCall(t, srv, http.MethodPost, "/api/v2/groups", admin.Token, map[string]any{
		"name": "Test Group",
	})
	groupID := groupRes.Body["id"].(string)

	matchRes := apiCall(t, srv, http.MethodPost, "/api/v2/groups/"+groupID+"/matches", admin.Token, map[string]any{
		"matchDate": "2099-12-31",
		"startTime": "18:00:00",
		"location":  "Test Court",
	})
	matchID := matchRes.Body["id"].(string)

	// POST /api/v2/matches/{id}/votes/close
	res := apiCall(t, srv, http.MethodPost, "/api/v2/matches/"+matchID+"/votes/close", admin.Token, nil)
	assert.True(t, res.Code == http.StatusOK || res.Code == http.StatusNoContent)
}

func TestVotes_GetPublicResults_NoAuth(t *testing.T) {
	srv := newTestServer(t)
	admin := registerAndLogin(t, srv, "Admin")
	makeAdmin(t, admin.ID)
	enableApiV2(t, admin.ID)

	// Create group and match
	groupRes := apiCall(t, srv, http.MethodPost, "/api/v2/groups", admin.Token, map[string]any{
		"name": "Test Group",
	})
	groupID := groupRes.Body["id"].(string)

	matchRes := apiCall(t, srv, http.MethodPost, "/api/v2/groups/"+groupID+"/matches", admin.Token, map[string]any{
		"matchDate": "2099-12-31",
		"startTime": "18:00:00",
		"location":  "Test Court",
	})
	matchBody := matchRes.Body
	matchHash := matchBody["hash"].(string)

	// GET /api/v2/matches/public/{hash}/votes/results without auth
	res := apiCall(t, srv, http.MethodGet, "/api/v2/matches/public/"+matchHash+"/votes/results", "", nil)
	assert.True(t, res.Code == http.StatusOK || res.Code == http.StatusForbidden)
}
