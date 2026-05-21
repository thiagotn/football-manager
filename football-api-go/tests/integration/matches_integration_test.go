package integration_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatches_CreateAndGet(t *testing.T) {
	srv := newTestServer(t)
	player := registerAndLogin(t, srv, "Match Player")
	enableApiV2(t, player.ID)

	r := apiCall(t, srv, http.MethodPost, "/api/v2/groups", player.Token,
		map[string]any{"name": "Match Test Group"})
	require.Equal(t, http.StatusCreated, r.Code)
	groupID, _ := r.Body["id"].(string)
	registerGroupCleanup(t, groupID)

	r = apiCall(t, srv, http.MethodPost,
		"/api/v2/groups/"+groupID+"/matches", player.Token,
		map[string]any{
			"match_date": "2030-01-15",
			"start_time": "20:00",
			"location":   "Quadra da Esquina",
		})
	require.Equal(t, http.StatusCreated, r.Code, "create match: %v", r.Body)
	matchID, _ := r.Body["id"].(string)
	require.NotEmpty(t, matchID)

	r = apiCall(t, srv, http.MethodGet,
		"/api/v2/groups/"+groupID+"/matches/"+matchID, player.Token, nil)
	require.Equal(t, http.StatusOK, r.Code)
	assert.Equal(t, "Quadra da Esquina", r.Body["location"])
}

func TestMatches_PublicMatchByHash(t *testing.T) {
	srv := newTestServer(t)
	player := registerAndLogin(t, srv, "Hash Match Player")
	enableApiV2(t, player.ID)

	r := apiCall(t, srv, http.MethodPost, "/api/v2/groups", player.Token,
		map[string]any{"name": "Public Match Group"})
	require.Equal(t, http.StatusCreated, r.Code)
	groupID, _ := r.Body["id"].(string)
	registerGroupCleanup(t, groupID)

	r = apiCall(t, srv, http.MethodPost,
		"/api/v2/groups/"+groupID+"/matches", player.Token,
		map[string]any{"match_date": "2030-02-20", "start_time": "19:00", "location": "Arena Test"})
	require.Equal(t, http.StatusCreated, r.Code)
	hash, _ := r.Body["hash"].(string)
	require.NotEmpty(t, hash)

	// Public route — no auth required
	r = apiCall(t, srv, http.MethodGet, "/api/v2/matches/public/"+hash, "", nil)
	require.Equal(t, http.StatusOK, r.Code)
	assert.Equal(t, "Arena Test", r.Body["location"])
}

func TestMatches_SetAttendance_Confirmed(t *testing.T) {
	srv := newTestServer(t)
	player := registerAndLogin(t, srv, "Attendance Player")
	enableApiV2(t, player.ID)

	r := apiCall(t, srv, http.MethodPost, "/api/v2/groups", player.Token,
		map[string]any{"name": "Attendance Group"})
	require.Equal(t, http.StatusCreated, r.Code)
	groupID, _ := r.Body["id"].(string)
	registerGroupCleanup(t, groupID)

	r = apiCall(t, srv, http.MethodPost,
		"/api/v2/groups/"+groupID+"/matches", player.Token,
		map[string]any{"match_date": "2030-03-10", "start_time": "18:00", "location": "Quadra Teste"})
	require.Equal(t, http.StatusCreated, r.Code)
	matchID, _ := r.Body["id"].(string)

	playerUUID, err := uuid.Parse(player.ID)
	require.NoError(t, err)

	r = apiCall(t, srv, http.MethodPost,
		"/api/v2/groups/"+groupID+"/matches/"+matchID+"/attendance", player.Token,
		map[string]any{"player_id": playerUUID.String(), "status": "confirmed"})
	require.Equal(t, http.StatusOK, r.Code)
	assert.Equal(t, "confirmed", r.Body["status"])
}

func TestMatches_ListGroupMatches(t *testing.T) {
	srv := newTestServer(t)
	player := registerAndLogin(t, srv, "List Matches Player")
	enableApiV2(t, player.ID)

	r := apiCall(t, srv, http.MethodPost, "/api/v2/groups", player.Token,
		map[string]any{"name": "List Matches Group"})
	require.Equal(t, http.StatusCreated, r.Code)
	groupID, _ := r.Body["id"].(string)
	registerGroupCleanup(t, groupID)

	for _, loc := range []string{"Quadra A", "Quadra B"} {
		r = apiCall(t, srv, http.MethodPost,
			"/api/v2/groups/"+groupID+"/matches", player.Token,
			map[string]any{"match_date": "2030-04-01", "start_time": "20:00", "location": loc})
		require.Equal(t, http.StatusCreated, r.Code)
	}

	r = apiCall(t, srv, http.MethodGet,
		"/api/v2/groups/"+groupID+"/matches", player.Token, nil)
	require.Equal(t, http.StatusOK, r.Code)
	assert.Len(t, r.List, 2)
}

func TestMatches_Discover_Public(t *testing.T) {
	srv := newTestServer(t)
	r := apiCall(t, srv, http.MethodGet, "/api/v2/matches/discover", "", nil)
	assert.Equal(t, http.StatusOK, r.Code)
}

func TestMatches_PublicStats_EmptyWhenNoStats(t *testing.T) {
	srv := newTestServer(t)
	player := registerAndLogin(t, srv, "Stats Player 2")
	enableApiV2(t, player.ID)

	r := apiCall(t, srv, http.MethodPost, "/api/v2/groups", player.Token,
		map[string]any{"name": "Stats Group 2"})
	require.Equal(t, http.StatusCreated, r.Code)
	groupID, _ := r.Body["id"].(string)
	registerGroupCleanup(t, groupID)

	r = apiCall(t, srv, http.MethodPost,
		"/api/v2/groups/"+groupID+"/matches", player.Token,
		map[string]any{"match_date": "2030-05-01", "start_time": "20:00", "location": "Quadra Stats"})
	require.Equal(t, http.StatusCreated, r.Code)
	hash, _ := r.Body["hash"].(string)

	r = apiCall(t, srv, http.MethodGet, "/api/v2/matches/public/"+hash+"/player-stats", "", nil)
	require.Equal(t, http.StatusOK, r.Code)
	registered, _ := r.Body["registered"].(bool)
	assert.False(t, registered)
}

func TestMatches_GetPublicMatch_NotFound(t *testing.T) {
	srv := newTestServer(t)
	r := apiCall(t, srv, http.MethodGet, "/api/v2/matches/public/nonexistent-hash", "", nil)
	assert.Equal(t, http.StatusNotFound, r.Code)
}
