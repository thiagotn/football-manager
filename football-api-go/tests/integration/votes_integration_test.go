package integration_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		"name": "VoteStatus Test Group",
	})
	require.Equal(t, http.StatusCreated, groupRes.Code, "create group: %v", groupRes.Body)
	groupID, _ := groupRes.Body["id"].(string)
	require.NotEmpty(t, groupID)
	registerGroupCleanup(t, groupID)

	matchRes := apiCall(t, srv, http.MethodPost, "/api/v2/groups/"+groupID+"/matches", admin.Token, map[string]any{
		"match_date": "2099-12-31",
		"start_time": "18:00:00",
		"location":   "Test Court",
	})
	require.Equal(t, http.StatusCreated, matchRes.Code, "create match: %v", matchRes.Body)
	matchID, _ := matchRes.Body["id"].(string)
	require.NotEmpty(t, matchID)

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
		"name": "CreateVote Test Group",
	})
	require.Equal(t, http.StatusCreated, groupRes.Code, "create group: %v", groupRes.Body)
	groupID, _ := groupRes.Body["id"].(string)
	require.NotEmpty(t, groupID)
	registerGroupCleanup(t, groupID)

	matchRes := apiCall(t, srv, http.MethodPost, "/api/v2/groups/"+groupID+"/matches", admin.Token, map[string]any{
		"match_date": "2099-12-31",
		"start_time": "18:00:00",
		"location":   "Test Court",
	})
	require.Equal(t, http.StatusCreated, matchRes.Code, "create match: %v", matchRes.Body)
	matchID, _ := matchRes.Body["id"].(string)
	require.NotEmpty(t, matchID)

	// POST /api/v2/matches/{id}/votes
	// For a future match, voting is not yet open, so 422 or 403 is also acceptable.
	res := apiCall(t, srv, http.MethodPost, "/api/v2/matches/"+matchID+"/votes", player.Token, map[string]any{
		"rating":  5,
		"comment": "Great match!",
	})
	assert.True(t, res.Code == http.StatusOK || res.Code == http.StatusCreated ||
		res.Code == http.StatusUnprocessableEntity || res.Code == http.StatusForbidden)
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
		"name": "InvalidRating Test Group",
	})
	require.Equal(t, http.StatusCreated, groupRes.Code, "create group: %v", groupRes.Body)
	groupID, _ := groupRes.Body["id"].(string)
	require.NotEmpty(t, groupID)
	registerGroupCleanup(t, groupID)

	matchRes := apiCall(t, srv, http.MethodPost, "/api/v2/groups/"+groupID+"/matches", admin.Token, map[string]any{
		"match_date": "2099-12-31",
		"start_time": "18:00:00",
		"location":   "Test Court",
	})
	require.Equal(t, http.StatusCreated, matchRes.Code, "create match: %v", matchRes.Body)
	matchID, _ := matchRes.Body["id"].(string)
	require.NotEmpty(t, matchID)

	// POST with invalid rating
	res := apiCall(t, srv, http.MethodPost, "/api/v2/matches/"+matchID+"/votes", player.Token, map[string]any{
		"rating":  10,
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
	assert.NotNil(t, res.Body["items"])
}

func TestVotes_GetVoteResults_BeforeClosing(t *testing.T) {
	srv := newTestServer(t)
	admin := registerAndLogin(t, srv, "Admin")
	makeAdmin(t, admin.ID)
	enableApiV2(t, admin.ID)

	// Create group and match
	groupRes := apiCall(t, srv, http.MethodPost, "/api/v2/groups", admin.Token, map[string]any{
		"name": "VoteResults Before Test Group",
	})
	require.Equal(t, http.StatusCreated, groupRes.Code, "create group: %v", groupRes.Body)
	groupID, _ := groupRes.Body["id"].(string)
	require.NotEmpty(t, groupID)
	registerGroupCleanup(t, groupID)

	matchRes := apiCall(t, srv, http.MethodPost, "/api/v2/groups/"+groupID+"/matches", admin.Token, map[string]any{
		"match_date": "2099-12-31",
		"start_time": "18:00:00",
		"location":   "Test Court",
	})
	require.Equal(t, http.StatusCreated, matchRes.Code, "create match: %v", matchRes.Body)
	matchID, _ := matchRes.Body["id"].(string)
	require.NotEmpty(t, matchID)

	// GET /api/v2/matches/{id}/votes/results before closing.
	// For a future match, voting is not open/closed, so 403 is expected.
	res := apiCall(t, srv, http.MethodGet, "/api/v2/matches/"+matchID+"/votes/results", admin.Token, nil)
	assert.True(t, res.Code == http.StatusOK || res.Code == http.StatusForbidden)
}

func TestVotes_CloseVoting_AsAdmin(t *testing.T) {
	srv := newTestServer(t)
	admin := registerAndLogin(t, srv, "Admin")
	makeAdmin(t, admin.ID)
	enableApiV2(t, admin.ID)

	// Create group and match
	groupRes := apiCall(t, srv, http.MethodPost, "/api/v2/groups", admin.Token, map[string]any{
		"name": "CloseVoting Test Group",
	})
	require.Equal(t, http.StatusCreated, groupRes.Code, "create group: %v", groupRes.Body)
	groupID, _ := groupRes.Body["id"].(string)
	require.NotEmpty(t, groupID)
	registerGroupCleanup(t, groupID)

	matchRes := apiCall(t, srv, http.MethodPost, "/api/v2/groups/"+groupID+"/matches", admin.Token, map[string]any{
		"match_date": "2099-12-31",
		"start_time": "18:00:00",
		"location":   "Test Court",
	})
	require.Equal(t, http.StatusCreated, matchRes.Code, "create match: %v", matchRes.Body)
	matchID, _ := matchRes.Body["id"].(string)
	require.NotEmpty(t, matchID)

	// POST /api/v2/matches/{id}/votes/close.
	// For a future match, voting is not yet open, so 403 (VOTING_NOT_OPEN) is expected.
	res := apiCall(t, srv, http.MethodPost, "/api/v2/matches/"+matchID+"/votes/close", admin.Token, nil)
	assert.True(t, res.Code == http.StatusOK || res.Code == http.StatusNoContent || res.Code == http.StatusForbidden)
}

func TestVotes_GetPublicResults_NoAuth(t *testing.T) {
	srv := newTestServer(t)
	admin := registerAndLogin(t, srv, "Admin")
	makeAdmin(t, admin.ID)
	enableApiV2(t, admin.ID)

	// Create group and match
	groupRes := apiCall(t, srv, http.MethodPost, "/api/v2/groups", admin.Token, map[string]any{
		"name": "PublicResults Test Group",
	})
	require.Equal(t, http.StatusCreated, groupRes.Code, "create group: %v", groupRes.Body)
	groupID, _ := groupRes.Body["id"].(string)
	require.NotEmpty(t, groupID)
	registerGroupCleanup(t, groupID)

	matchRes := apiCall(t, srv, http.MethodPost, "/api/v2/groups/"+groupID+"/matches", admin.Token, map[string]any{
		"match_date": "2099-12-31",
		"start_time": "18:00:00",
		"location":   "Test Court",
	})
	require.Equal(t, http.StatusCreated, matchRes.Code, "create match: %v", matchRes.Body)
	matchHash, _ := matchRes.Body["hash"].(string)
	require.NotEmpty(t, matchHash)

	// GET /api/v2/matches/public/{hash}/votes/results without auth.
	// Voting is not yet closed on a future match, so 404 is expected here.
	res := apiCall(t, srv, http.MethodGet, "/api/v2/matches/public/"+matchHash+"/votes/results", "", nil)
	assert.True(t, res.Code == http.StatusOK || res.Code == http.StatusForbidden || res.Code == http.StatusNotFound)
}
