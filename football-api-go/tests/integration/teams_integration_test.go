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
		"name": "Teams Test Group " + p.ID,
	})
	assert.Equal(t, http.StatusCreated, groupRes.Code)
	groupID, _ := groupRes.Body["id"].(string)
	assert.NotEmpty(t, groupID)

	matchRes := apiCall(t, srv, http.MethodPost, "/api/v2/groups/"+groupID+"/matches", p.Token, map[string]any{
		"match_date": "2099-12-31",
		"start_time": "18:00:00",
		"location":   "Test Court",
	})
	assert.Equal(t, http.StatusCreated, matchRes.Code)
	matchID, _ := matchRes.Body["id"].(string)
	assert.NotEmpty(t, matchID)

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
		"name": "Teams Draw Success Group " + admin.ID,
	})
	assert.Equal(t, http.StatusCreated, groupRes.Code)
	groupID, _ := groupRes.Body["id"].(string)
	assert.NotEmpty(t, groupID)

	// Add 4 non-admin members (need nTeams=ceil(4/2)=2 with players_per_team=1)
	player1 := registerAndLogin(t, srv, "Player 1")
	enableApiV2(t, player1.ID)
	player2 := registerAndLogin(t, srv, "Player 2")
	enableApiV2(t, player2.ID)
	player3 := registerAndLogin(t, srv, "Player 3")
	enableApiV2(t, player3.ID)
	player4 := registerAndLogin(t, srv, "Player 4")
	enableApiV2(t, player4.ID)

	for _, p := range []string{player1.ID, player2.ID, player3.ID, player4.ID} {
		apiCall(t, srv, http.MethodPost, "/api/v2/groups/"+groupID+"/members", admin.Token, map[string]any{
			"player_id": p,
			"role":      "member",
		})
	}

	// Create match with players_per_team set (required by DrawTeams)
	matchRes := apiCall(t, srv, http.MethodPost, "/api/v2/groups/"+groupID+"/matches", admin.Token, map[string]any{
		"match_date":      "2099-12-31",
		"start_time":      "18:00:00",
		"location":        "Test Court",
		"players_per_team": 1,
	})
	assert.Equal(t, http.StatusCreated, matchRes.Code)
	matchID, _ := matchRes.Body["id"].(string)
	assert.NotEmpty(t, matchID)

	// Confirm attendance for all 4 players
	for _, p := range []testPlayer{player1, player2, player3, player4} {
		apiCall(t, srv, http.MethodPost, "/api/v2/groups/"+groupID+"/matches/"+matchID+"/attendance", p.Token, map[string]any{
			"player_id": p.ID,
			"status":    "confirmed",
		})
	}

	// Draw teams — handler returns 201
	res := apiCall(t, srv, http.MethodPost, "/api/v2/matches/"+matchID+"/teams", admin.Token, nil)
	assert.Equal(t, http.StatusCreated, res.Code)

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
		"name": "Teams GetTeams Group " + admin.ID,
	})
	assert.Equal(t, http.StatusCreated, groupRes.Code)
	groupID, _ := groupRes.Body["id"].(string)
	assert.NotEmpty(t, groupID)

	matchRes := apiCall(t, srv, http.MethodPost, "/api/v2/groups/"+groupID+"/matches", admin.Token, map[string]any{
		"match_date": "2099-12-31",
		"start_time": "18:00:00",
		"location":   "Test Court",
	})
	assert.Equal(t, http.StatusCreated, matchRes.Code)
	matchID, _ := matchRes.Body["id"].(string)
	assert.NotEmpty(t, matchID)

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
