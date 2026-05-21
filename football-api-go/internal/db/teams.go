package db

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MatchTeam is a team record in a match.
type MatchTeam struct {
	ID        uuid.UUID `json:"id"`
	MatchID   uuid.UUID `json:"match_id"`
	Name      string    `json:"name"`
	Color     *string   `json:"color"`
	Position  int       `json:"position"`
	CreatedAt time.Time `json:"created_at"`
}

// MatchTeamPlayer is a player assigned to a team.
type MatchTeamPlayer struct {
	TeamID     uuid.UUID `json:"team_id"`
	PlayerID   uuid.UUID `json:"player_id"`
	IsReserve  bool      `json:"is_reserve"`
	Name       string    `json:"name"`
	Nickname   *string   `json:"nickname"`
	AvatarURL  *string   `json:"avatar_url"`
	SkillStars int       `json:"skill_stars"`
	Position   string    `json:"position"`
}

// PlayerForDraw holds the data needed by the team builder algorithm.
type PlayerForDraw struct {
	PlayerID   uuid.UUID `json:"player_id"`
	Name       string    `json:"name"`
	Nickname   *string   `json:"nickname"`
	AvatarURL  *string   `json:"avatar_url"`
	SkillStars int       `json:"skill_stars"`
	Position   string    `json:"position"`
}

// MatchTeamWithPlayers is a team with its player list for API responses.
type MatchTeamWithPlayers struct {
	MatchTeam
	Players    []MatchTeamPlayer `json:"players"`
	SkillTotal int               `json:"skill_total"`
}

func GetConfirmedPlayersForMatch(ctx context.Context, pool *pgxpool.Pool, matchID uuid.UUID) ([]PlayerForDraw, error) {
	rows, err := pool.Query(ctx, `
		SELECT
			p.id, p.name, p.nickname, p.avatar_url,
			COALESCE(gm.skill_stars, 2),
			COALESCE(gm.position, 'mei')
		FROM attendances a
		JOIN players p ON p.id = a.player_id
		JOIN matches m ON m.id = a.match_id
		LEFT JOIN group_members gm ON gm.player_id = a.player_id AND gm.group_id = m.group_id
		WHERE a.match_id = $1
		  AND a.status = 'confirmed'
		  AND p.role = 'player'
		ORDER BY gm.skill_stars DESC NULLS LAST`, matchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var players []PlayerForDraw
	for rows.Next() {
		var p PlayerForDraw
		if err := rows.Scan(&p.PlayerID, &p.Name, &p.Nickname, &p.AvatarURL, &p.SkillStars, &p.Position); err != nil {
			return nil, err
		}
		players = append(players, p)
	}
	return players, rows.Err()
}

func DeleteTeamsByMatch(ctx context.Context, pool *pgxpool.Pool, matchID uuid.UUID) error {
	_, err := pool.Exec(ctx, `DELETE FROM match_teams WHERE match_id = $1`, matchID)
	return err
}

func CreateTeam(ctx context.Context, pool *pgxpool.Pool, matchID uuid.UUID, name string, color *string, position int) (*MatchTeam, error) {
	var t MatchTeam
	err := pool.QueryRow(ctx, `
		INSERT INTO match_teams (match_id, name, color, position)
		VALUES ($1, $2, $3, $4)
		RETURNING id, match_id, name, color, position, created_at`,
		matchID, name, color, position).
		Scan(&t.ID, &t.MatchID, &t.Name, &t.Color, &t.Position, &t.CreatedAt)
	return &t, err
}

func AddPlayerToTeam(ctx context.Context, pool *pgxpool.Pool, teamID, playerID uuid.UUID, isReserve bool) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO match_team_players (team_id, player_id, is_reserve)
		VALUES ($1, $2, $3)
		ON CONFLICT (team_id, player_id) DO NOTHING`,
		teamID, playerID, isReserve)
	return err
}

func GetTeamsForMatch(ctx context.Context, pool *pgxpool.Pool, matchID uuid.UUID) ([]MatchTeamWithPlayers, error) {
	// Fetch teams
	teamRows, err := pool.Query(ctx, `
		SELECT id, match_id, name, color, position, created_at
		FROM match_teams
		WHERE match_id = $1
		ORDER BY position`, matchID)
	if err != nil {
		return nil, err
	}
	defer teamRows.Close()

	var teams []MatchTeamWithPlayers
	for teamRows.Next() {
		var t MatchTeamWithPlayers
		if err := teamRows.Scan(&t.ID, &t.MatchID, &t.Name, &t.Color, &t.Position, &t.CreatedAt); err != nil {
			return nil, err
		}
		t.Players = []MatchTeamPlayer{}
		teams = append(teams, t)
	}
	if err := teamRows.Err(); err != nil {
		return nil, err
	}
	if len(teams) == 0 {
		return teams, nil
	}

	// Fetch players for all teams in one query
	playerRows, err := pool.Query(ctx, `
		SELECT
			tp.team_id, tp.player_id, tp.is_reserve,
			p.name, p.nickname, p.avatar_url,
			COALESCE(gm.skill_stars, 2),
			COALESCE(gm.position, 'mei')
		FROM match_team_players tp
		JOIN players p ON p.id = tp.player_id
		JOIN match_teams t ON t.id = tp.team_id
		JOIN matches m ON m.id = t.match_id
		LEFT JOIN group_members gm ON gm.player_id = tp.player_id AND gm.group_id = m.group_id
		WHERE t.match_id = $1
		ORDER BY tp.is_reserve, gm.skill_stars DESC NULLS LAST`, matchID)
	if err != nil {
		return nil, err
	}
	defer playerRows.Close()

	// Map team ID → index
	teamIdx := make(map[uuid.UUID]int, len(teams))
	for i, t := range teams {
		teamIdx[t.ID] = i
	}

	for playerRows.Next() {
		var tp MatchTeamPlayer
		var teamID uuid.UUID
		if err := playerRows.Scan(
			&teamID, &tp.PlayerID, &tp.IsReserve,
			&tp.Name, &tp.Nickname, &tp.AvatarURL,
			&tp.SkillStars, &tp.Position,
		); err != nil {
			return nil, err
		}
		tp.TeamID = teamID
		if idx, ok := teamIdx[teamID]; ok {
			teams[idx].Players = append(teams[idx].Players, tp)
			teams[idx].SkillTotal += tp.SkillStars
		}
	}
	return teams, playerRows.Err()
}
