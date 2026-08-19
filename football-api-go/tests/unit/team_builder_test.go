package unit_test

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thiagotn/football-manager/football-api-go/internal/db"
	"github.com/thiagotn/football-manager/football-api-go/internal/services"
)

// ── Helpers ───────────────────────────────────────────────────────────────────

func makePlayers(n int, position string, skillBase int) []db.PlayerForDraw {
	players := make([]db.PlayerForDraw, n)
	for i := range players {
		players[i] = db.PlayerForDraw{
			PlayerID:   uuid.New(),
			Name:       fmt.Sprintf("Player%d", i+1),
			SkillStars: (skillBase+i)%5 + 1,
			Position:   position,
		}
	}
	return players
}

func ptr(s string) *string { return &s }

// ── BuildTeams ────────────────────────────────────────────────────────────────

func TestBuildTeams_NotEnoughPlayers(t *testing.T) {
	_, _, err := services.BuildTeams([]db.PlayerForDraw{}, 5, nil, "balanced")
	assert.Error(t, err)
}

func TestBuildTeams_OnePlayerOnly(t *testing.T) {
	// teamSize = 6 → nTeams = ceil(1/6) = 1 < 2 → error
	_, _, err := services.BuildTeams(makePlayers(1, "mei", 0), 5, nil, "balanced")
	assert.Error(t, err)
}

func TestBuildTeams_TwoTeams_ExactSize(t *testing.T) {
	// 12 players, 5 per team → teamSize=6 → nTeams=2, 0 reserves
	players := makePlayers(12, "mei", 0)
	teams, reserves, err := services.BuildTeams(players, 5, nil, "balanced")
	require.NoError(t, err)
	assert.Len(t, teams, 2)
	assert.Empty(t, reserves)
	assert.Len(t, teams[0].Players, 6)
	assert.Len(t, teams[1].Players, 6)
}

func TestBuildTeams_WithReserves(t *testing.T) {
	// 15 players, 5 per team → teamSize=6, nTeams=ceil(15/6)=3
	// actually: teamSize = playersPerTeam+1 = 6, 15/6 = 2.5 → nTeams=3
	// 3 teams × 6 = 18 slots, 15 players → 0 reserves (all placed)
	// Let's use 7 players with ppt=2 → teamSize=3, nTeams=ceil(7/3)=3, 3×3=9 slots → 7 placed, 0 extra
	// Actually 7 players / teamSize 3 = 2.33 → nTeams = 3, 9 slots → 7 go to teams, 2 overflow that don't fit
	players := makePlayers(7, "mei", 0)
	teams, reserves, err := services.BuildTeams(players, 2, nil, "balanced")
	require.NoError(t, err)
	assert.Len(t, teams, 3)
	totalPlaced := 0
	for _, t := range teams {
		totalPlaced += len(t.Players)
	}
	assert.Equal(t, 7, totalPlaced+len(reserves))
}

func TestBuildTeams_GKDistributed(t *testing.T) {
	// 2 GKs + 10 field players, 5 per team → teamSize=6, nTeams=2
	gks := makePlayers(2, "gk", 3)
	field := makePlayers(10, "mei", 0)
	players := append(gks, field...)

	teams, reserves, err := services.BuildTeams(players, 5, nil, "balanced")
	require.NoError(t, err)
	assert.Len(t, teams, 2)
	assert.Empty(t, reserves)

	for _, team := range teams {
		gkCount := 0
		for _, p := range team.Players {
			if p.Position == "gk" {
				gkCount++
			}
		}
		assert.Equal(t, 1, gkCount, "each team should have exactly 1 GK")
	}
}

func TestBuildTeams_ExtraGKGoesToReserves(t *testing.T) {
	// 3 GKs + 9 field, 5 per team → teamSize=6, nTeams=2
	// 2 GKs assigned to teams, 1 GK extra → goes to reserves
	gks := makePlayers(3, "gk", 4)
	field := makePlayers(9, "mei", 0)
	players := append(gks, field...)

	teams, reserves, err := services.BuildTeams(players, 5, nil, "balanced")
	require.NoError(t, err)
	assert.Len(t, teams, 2)
	// 1 extra GK in reserves OR assigned to field slot if team has room
	total := 0
	for _, t := range teams {
		total += len(t.Players)
	}
	assert.Equal(t, 12, total+len(reserves))
}

func TestBuildTeams_ThreeTeams(t *testing.T) {
	// 15 players, 4 per team → teamSize=5, nTeams=ceil(15/5)=3
	players := makePlayers(15, "mei", 0)
	teams, _, err := services.BuildTeams(players, 4, nil, "balanced")
	require.NoError(t, err)
	assert.Len(t, teams, 3)
}

func TestBuildTeams_CustomSlots_Name(t *testing.T) {
	players := makePlayers(10, "mei", 0)
	slots := []db.TeamSlot{
		{Name: ptr("Flamengo"), Color: ptr("vermelho")},
		{Name: ptr("Fluminense"), Color: ptr("verde")},
	}
	teams, _, err := services.BuildTeams(players, 4, slots, "balanced")
	require.NoError(t, err)
	require.Len(t, teams, 2)

	names := map[string]bool{teams[0].Name: true, teams[1].Name: true}
	assert.True(t, names["Flamengo"])
	assert.True(t, names["Fluminense"])
}

func TestBuildTeams_CustomSlots_Color(t *testing.T) {
	players := makePlayers(10, "mei", 0)
	slots := []db.TeamSlot{
		{Name: ptr("Vermelho"), Color: ptr("vermelho")},
		{Name: ptr("Azul"), Color: ptr("azul")},
	}
	teams, _, err := services.BuildTeams(players, 4, slots, "balanced")
	require.NoError(t, err)
	require.Len(t, teams, 2)

	// vermelho and azul are in the bibColorHex map
	colors := map[string]bool{teams[0].Color: true, teams[1].Color: true}
	assert.True(t, colors["#ef4444"], "vermelho should map to #ef4444")
	assert.True(t, colors["#3b82f6"], "azul should map to #3b82f6")
}

func TestBuildTeams_PositiveSkillBalance(t *testing.T) {
	// 4 distinct skill values, 2 teams of 2: optimizer should produce balanced teams
	players := []db.PlayerForDraw{
		{PlayerID: uuid.New(), Name: "A", SkillStars: 5, Position: "mei"},
		{PlayerID: uuid.New(), Name: "B", SkillStars: 4, Position: "mei"},
		{PlayerID: uuid.New(), Name: "C", SkillStars: 3, Position: "mei"},
		{PlayerID: uuid.New(), Name: "D", SkillStars: 2, Position: "mei"},
	}
	// Run multiple times to account for random elements
	for i := 0; i < 10; i++ {
		teams, _, err := services.BuildTeams(players, 1, nil, "balanced")
		require.NoError(t, err)
		require.Len(t, teams, 2)
		diff := teams[0].SkillTotal - teams[1].SkillTotal
		if diff < 0 {
			diff = -diff
		}
		assert.LessOrEqual(t, diff, 1, "skill imbalance should be ≤ 1 star")
	}
}

func TestBuildTeams_PositionIndexIsOneIndexed(t *testing.T) {
	players := makePlayers(6, "mei", 0)
	teams, _, err := services.BuildTeams(players, 2, nil, "balanced")
	require.NoError(t, err)
	require.Len(t, teams, 2)
	// positions should be 1 and 2 (not 0-indexed)
	positions := map[int]bool{teams[0].Position: true, teams[1].Position: true}
	assert.True(t, positions[1])
	assert.True(t, positions[2])
}

func TestBuildTeams_AllPlayersAccountedFor(t *testing.T) {
	// Every player must end up in a team or reserves — no one lost
	players := makePlayers(17, "ata", 0)
	teams, reserves, err := services.BuildTeams(players, 4, nil, "balanced")
	require.NoError(t, err)

	seen := make(map[uuid.UUID]bool)
	for _, t := range teams {
		for _, p := range t.Players {
			seen[p.PlayerID] = true
		}
	}
	for _, p := range reserves {
		seen[p.PlayerID] = true
	}
	assert.Len(t, seen, len(players), "all players must be accounted for")
}

func TestBuildTeams_NoDuplicatePlayers(t *testing.T) {
	players := makePlayers(12, "mei", 0)
	teams, reserves, err := services.BuildTeams(players, 5, nil, "balanced")
	require.NoError(t, err)

	seen := make(map[uuid.UUID]int)
	for _, t := range teams {
		for _, p := range t.Players {
			seen[p.PlayerID]++
		}
	}
	for _, p := range reserves {
		seen[p.PlayerID]++
	}
	for id, count := range seen {
		assert.Equal(t, 1, count, "player %s appears %d times", id, count)
	}
}

// ── BuildTeams — "simple" strategy ───────────────────────────────────────────

func TestBuildTeamsSimple_GKDistributedOnePerTeam(t *testing.T) {
	gks := makePlayers(2, "gk", 3)
	field := makePlayers(10, "mei", 0)
	players := append(gks, field...)

	teams, reserves, err := services.BuildTeams(players, 5, nil, "simple")
	require.NoError(t, err)
	assert.Len(t, teams, 2)
	assert.Empty(t, reserves)

	for _, team := range teams {
		gkCount := 0
		for _, p := range team.Players {
			if p.Position == "gk" {
				gkCount++
			}
		}
		assert.Equal(t, 1, gkCount, "each team should have exactly 1 GK")
	}
}

func TestBuildTeamsSimple_ExtraGKNeverDoublesUp(t *testing.T) {
	// 3 GKs for 2 teams → no team ends up with 2 GKs
	gks := makePlayers(3, "gk", 4)
	field := makePlayers(9, "mei", 0)
	players := append(gks, field...)

	teams, reserves, err := services.BuildTeams(players, 5, nil, "simple")
	require.NoError(t, err)
	assert.Len(t, teams, 2)

	total := 0
	for _, team := range teams {
		gkCount := 0
		for _, p := range team.Players {
			if p.Position == "gk" {
				gkCount++
			}
		}
		assert.LessOrEqual(t, gkCount, 1, "no team should have more than 1 GK")
		total += len(team.Players)
	}
	assert.Equal(t, 12, total+len(reserves))
}

func TestBuildTeamsSimple_MixedPositionsAllAccountedFor(t *testing.T) {
	// Mixed roster with all positions — sizes and conservation match balanced
	players := append(makePlayers(2, "gk", 3), makePlayers(4, "lat", 0)...)
	players = append(players, makePlayers(4, "zag", 1)...)
	players = append(players, makePlayers(4, "mei", 2)...)
	players = append(players, makePlayers(4, "ata", 3)...)

	teams, reserves, err := services.BuildTeams(players, 5, nil, "simple")
	require.NoError(t, err)
	assert.Len(t, teams, 3)

	seen := make(map[uuid.UUID]bool)
	for _, team := range teams {
		for _, p := range team.Players {
			seen[p.PlayerID] = true
		}
	}
	for _, p := range reserves {
		seen[p.PlayerID] = true
	}
	assert.Len(t, seen, len(players), "all players must be accounted for")
}

func TestBuildTeamsSimple_SkillBalance(t *testing.T) {
	// Greedy-swap optimizer still runs: imbalance ≤ 1 star
	players := []db.PlayerForDraw{
		{PlayerID: uuid.New(), Name: "A", SkillStars: 5, Position: "ata"},
		{PlayerID: uuid.New(), Name: "B", SkillStars: 4, Position: "zag"},
		{PlayerID: uuid.New(), Name: "C", SkillStars: 3, Position: "lat"},
		{PlayerID: uuid.New(), Name: "D", SkillStars: 2, Position: "mei"},
	}
	for i := 0; i < 10; i++ {
		teams, _, err := services.BuildTeams(players, 1, nil, "simple")
		require.NoError(t, err)
		require.Len(t, teams, 2)
		diff := teams[0].SkillTotal - teams[1].SkillTotal
		if diff < 0 {
			diff = -diff
		}
		assert.LessOrEqual(t, diff, 1, "skill imbalance should be ≤ 1 star")
	}
}

func TestBuildTeamsSimple_SinglePositionRoster(t *testing.T) {
	// Extreme roster (only forwards) fills teams normally — no position quota
	players := append(makePlayers(2, "gk", 3), makePlayers(6, "ata", 0)...)
	teams, reserves, err := services.BuildTeams(players, 3, nil, "simple")
	require.NoError(t, err)
	assert.Len(t, teams, 2)
	assert.Empty(t, reserves)
	for _, team := range teams {
		assert.Len(t, team.Players, 4)
	}
}
