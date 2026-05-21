package integration_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTeams_DrawTeams_NoConfirmedPlayers(t *testing.T) {
	srv := newTestServer(t)
	p := registerAndLogin(t, srv, "Admin")
	makeAdmin(t, p.ID)
	enableApiV2(t, p.ID)

	// Create group and match
	groupRes := apiCall(t, srv, http.MethodPost, "/api/v2/groups", p.Token, map[string]any{
		"name": "Test Group",
	})
	groupBody := groupRes.Body
	groupID := groupBody["id"].(string)

	matchRes := apiCall(t, srv, http.MethodPost, "/api/v2/groups/"+groupID+"/matches", p.Token, map[string]any{
		"matchDate": "2099-12-31",
		"startTime": "18:00:00",
		"location":  "Test Court",
	})
	matchBody := matchRes.Body
	matchID := matchBody["id"].(string)

	// Try to draw teams with no confirmed players
	res := apiCall(t, srv, http.MethodPost, "/api/v2/matches/"+matchID+"/teams", p.Token, nil)
	// Should fail — no confirmed players
	assert.Equal(t, http.StatusUnprocessableEntity, res.Code)
}

func TestTeams_DrawTeams_Success(t *testing.T) {
	srv := newTestServer(t)

	// Create admin
	admin := registerAndLogin(t, srv, "Admin")
	makeAdmin(t, admin.ID)
	enableApiV2(t, admin.ID)

	// Create group
	groupRes := apiCall(t, srv, http.MethodPost, "/api/v2/groups", admin.Token, map[string]any{
		"name": "Test Group",
	})
	groupBody := groupRes.Body
	groupID := groupBody["id"].(string)

	// Add members
	player1 := registerAndLogin(t, srv, "Player 1")
	enableApiV2(t, player1.ID)
	player2 := registerAndLogin(t, srv, "Player 2")
	enableApiV2(t, player2.ID)

	apiCall(t, srv, http.MethodPost, "/api/v2/groups/"+groupID+"/members", admin.Token, map[string]any{
		"playerID": player1.ID,
		"position": "ZAG",
	})
	apiCall(t, srv, http.MethodPost, "/api/v2/groups/"+groupID+"/members", admin.Token, map[string]any{
		"playerID": player2.ID,
		"position": "MEI",
	})

	// Create match
	matchRes := apiCall(t, srv, http.MethodPost, "/api/v2/groups/"+groupID+"/matches", admin.Token, map[string]any{
		"matchDate": "2099-12-31",
		"startTime": "18:00:00",
		"location":  "Test Court",
	})
	matchBody := matchRes.Body
	matchID := matchBody["id"].(string)

	// Confirm attendance
	apiCall(t, srv, http.MethodPatch, "/api/v2/groups/"+groupID+"/matches/"+matchID+"/attendance", player1.Token, map[string]any{
		"status": "confirmed",
	})
	apiCall(t, srv, http.MethodPatch, "/api/v2/groups/"+groupID+"/matches/"+matchID+"/attendance", player2.Token, map[string]any{
		"status": "confirmed",
	})

	// Draw teams
	res := apiCall(t, srv, http.MethodPost, "/api/v2/matches/"+matchID+"/teams", admin.Token, nil)
	assert.Equal(t, http.StatusOK, res.Code)

	teamBody := res.Body
	assert.Contains(t, teamBody, "teams")
}

func TestTeams_GetTeams_NoAuth(t *testing.T) {
	srv := newTestServer(t)

	admin := registerAndLogin(t, srv, "Admin")
	makeAdmin(t, admin.ID)
	enableApiV2(t, admin.ID)

	// Create group and match
	groupRes := apiCall(t, srv, http.MethodPost, "/api/v2/groups", admin.Token, map[string]any{
		"name": "Test Group",
	})
	groupBody := groupRes.Body
	groupID := groupBody["id"].(string)

	matchRes := apiCall(t, srv, http.MethodPost, "/api/v2/groups/"+groupID+"/matches", admin.Token, map[string]any{
		"matchDate": "2099-12-31",
		"startTime": "18:00:00",
		"location":  "Test Court",
	})
	matchBody := matchRes.Body
	matchID := matchBody["id"].(string)

	// Get teams without auth (public endpoint)
	res := apiCall(t, srv, http.MethodGet, "/api/v2/matches/"+matchID+"/teams", "", nil)
	// Should work even without auth
	assert.True(t, res.Code == http.StatusOK || res.Code == http.StatusNotFound)
}

func TestTeams_GetTeams_InvalidMatchID(t *testing.T) {
	srv := newTestServer(t)

	res := apiCall(t, srv, http.MethodGet, "/api/v2/matches/invalid-uuid/teams", "", nil)
	assert.Equal(t, http.StatusUnprocessableEntity, res.Code)
}
