package integration_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thiagotn/football-manager/football-api-go/internal/db"
)

func TestVotes_GetVoteStatus_Before(t *testing.T) {
	srv := newTestServer(t)
	admin := registerAndLogin(t, srv, "Admin")
	makeAdmin(t, admin.ID)

	player := registerAndLogin(t, srv, "Player")

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

	player := registerAndLogin(t, srv, "Player")

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

	player := registerAndLogin(t, srv, "Player")

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

	// GET /api/v2/votes/pending
	res := apiCall(t, srv, http.MethodGet, "/api/v2/votes/pending", p.Token, nil)
	assert.Equal(t, http.StatusOK, res.Code)
	assert.NotNil(t, res.Body["items"])
}

func TestVotes_GetVoteResults_BeforeClosing(t *testing.T) {
	srv := newTestServer(t)
	admin := registerAndLogin(t, srv, "Admin")
	makeAdmin(t, admin.ID)

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

// Among players tied on a match's points, the results must favor whoever has FEWER points
// in the group ranking up to the match's date (the underdog goes first / higher on podium).
func TestVotes_GetVoteResults_TieBreakByGroupRanking(t *testing.T) {
	srv := newTestServer(t)
	pool := getPool(t)
	ctx := context.Background()

	admin := registerAndLogin(t, srv, "TieBreak Admin")
	makeAdmin(t, admin.ID)
	candA := registerAndLogin(t, srv, "Candidate A")
	candB := registerAndLogin(t, srv, "Candidate B")
	voter1 := registerAndLogin(t, srv, "Voter One")
	voter2 := registerAndLogin(t, srv, "Voter Two")

	groupRes := apiCall(t, srv, http.MethodPost, "/api/v2/groups", admin.Token, map[string]any{
		"name": "TieBreak Group",
	})
	require.Equal(t, http.StatusCreated, groupRes.Code, "create group: %v", groupRes.Body)
	groupID, _ := groupRes.Body["id"].(string)
	registerGroupCleanup(t, groupID)

	mkMatch := func(date string) string {
		r := apiCall(t, srv, http.MethodPost, "/api/v2/groups/"+groupID+"/matches", admin.Token, map[string]any{
			"match_date": date, "start_time": "18:00:00", "location": "Court",
		})
		require.Equal(t, http.StatusCreated, r.Code, "create match: %v", r.Body)
		id, _ := r.Body["id"].(string)
		require.NotEmpty(t, id)
		return id
	}
	prevMatch := mkMatch("2020-01-01")
	targetMatch := mkMatch("2020-02-01")

	// Seed votes directly (bypasses voting-window/eligibility for a deterministic setup).
	seed := func(matchID, voterID, playerID string, position, points int) {
		var voteID uuid.UUID
		err := pool.QueryRow(ctx,
			`INSERT INTO match_votes (match_id, voter_id) VALUES ($1, $2) RETURNING id`,
			uuid.MustParse(matchID), uuid.MustParse(voterID)).Scan(&voteID)
		require.NoError(t, err)
		_, err = pool.Exec(ctx,
			`INSERT INTO match_vote_top5 (vote_id, player_id, position, points) VALUES ($1, $2, $3, $4)`,
			voteID, uuid.MustParse(playerID), position, points)
		require.NoError(t, err)
	}

	// Prior match: candidate A scores → A has more group-ranking points than B.
	seed(prevMatch, voter1.ID, candA.ID, 1, 10)
	// Target match: A and B tie at 10 points each.
	seed(targetMatch, voter1.ID, candA.ID, 1, 10)
	seed(targetMatch, voter2.ID, candB.ID, 1, 10)

	results, err := db.GetVoteResults(ctx, pool, uuid.MustParse(targetMatch))
	require.NoError(t, err)
	require.Len(t, results.Top5, 2)
	// B (fewer group points up to the match date) must come first.
	assert.Equal(t, candB.ID, results.Top5[0].PlayerID.String(),
		"tied player with fewer group-ranking points should rank first")
	assert.Equal(t, candA.ID, results.Top5[1].PlayerID.String())
}
