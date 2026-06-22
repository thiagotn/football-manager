package db

import (
	"context"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Match represents a football match.
type Match struct {
	ID                   uuid.UUID `json:"id"`
	GroupID              uuid.UUID `json:"group_id"`
	Number               int       `json:"number"`
	Hash                 string    `json:"hash"`
	MatchDate            string    `json:"match_date"` // "YYYY-MM-DD"
	StartTime            string    `json:"start_time"` // "HH:MM:SS"
	EndTime              *string   `json:"end_time"`   // nullable
	Location             string    `json:"location"`
	Address              *string   `json:"address"`
	CourtType            *string   `json:"court_type"`
	PlayersPerTeam       *int      `json:"players_per_team"`
	MaxPlayers           *int      `json:"max_players"`
	Notes                *string   `json:"notes"`
	Status               string    `json:"status"`
	VoteOpenDelayMinutes int       `json:"vote_open_delay_minutes"`
	VoteDurationHours    int       `json:"vote_duration_hours"`
	VoteNotified         bool      `json:"-"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// AttendanceWithPlayer is an attendance record joined with player info.
type AttendanceWithPlayer struct {
	ID              uuid.UUID `json:"id"`
	MatchID         uuid.UUID `json:"match_id"`
	PlayerID        uuid.UUID `json:"player_id"`
	Status          string    `json:"status"`
	UpdatedAt       time.Time `json:"updated_at"`
	PlayerName      string    `json:"player_name"`
	PlayerNickname  *string   `json:"player_nickname"`
	PlayerAvatarURL *string   `json:"player_avatar_url"`
	PlayerRole      string    `json:"player_role"`
	Position        string    `json:"position"`       // from group_members
	GroupNickname   *string   `json:"group_nickname"` // from group_members
}

// DiscoverMatch is a match row for the discover feed.
type DiscoverMatch struct {
	Match
	GroupName      string `json:"group_name"`
	GroupTimezone  string `json:"group_timezone"`
	ConfirmedCount int    `json:"confirmed_count"`
}

// MatchPlayerStat is a record of goals/assists for a player in a match.
type MatchPlayerStat struct {
	ID         uuid.UUID `json:"id"`
	MatchID    uuid.UUID `json:"match_id"`
	PlayerID   uuid.UUID `json:"player_id"`
	Goals      int       `json:"goals"`
	Assists    int       `json:"assists"`
	PlayerName string    `json:"player_name"`
	AvatarURL  *string   `json:"avatar_url"`
}

const matchCols = `
	m.id, m.group_id, m.number, m.hash,
	m.match_date::TEXT, m.start_time::TEXT, m.end_time::TEXT,
	m.location, m.address, m.court_type::TEXT,
	m.players_per_team, m.max_players, m.notes,
	m.status::TEXT, m.vote_open_delay_minutes, m.vote_duration_hours, m.vote_notified,
	m.created_at, m.updated_at`

// matchReturnCols is the same as matchCols but without the "m." alias prefix,
// for use in INSERT...RETURNING clauses where no table alias is available.
const matchReturnCols = `
	id, group_id, number, hash,
	match_date::TEXT, start_time::TEXT, end_time::TEXT,
	location, address, court_type::TEXT,
	players_per_team, max_players, notes,
	status::TEXT, vote_open_delay_minutes, vote_duration_hours, vote_notified,
	created_at, updated_at`

func scanMatch(scanFn func(dest ...any) error) (*Match, error) {
	var m Match
	err := scanFn(
		&m.ID, &m.GroupID, &m.Number, &m.Hash,
		&m.MatchDate, &m.StartTime, &m.EndTime,
		&m.Location, &m.Address, &m.CourtType,
		&m.PlayersPerTeam, &m.MaxPlayers, &m.Notes,
		&m.Status, &m.VoteOpenDelayMinutes, &m.VoteDurationHours, &m.VoteNotified,
		&m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &m, nil
}

func GetMatchesByGroup(ctx context.Context, pool *pgxpool.Pool, groupID uuid.UUID) ([]Match, error) {
	rows, err := pool.Query(ctx, `
		SELECT `+matchCols+`
		FROM matches m
		WHERE m.group_id = $1
		ORDER BY m.match_date DESC, m.start_time DESC`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	matches := make([]Match, 0)
	for rows.Next() {
		match, err := scanMatch(rows.Scan)
		if err != nil {
			return nil, err
		}
		matches = append(matches, *match)
	}
	return matches, rows.Err()
}

func GetMatchByID(ctx context.Context, pool *pgxpool.Pool, matchID uuid.UUID) (*Match, error) {
	row := pool.QueryRow(ctx, `SELECT `+matchCols+` FROM matches m WHERE m.id = $1`, matchID)
	return scanMatch(row.Scan)
}

func GetMatchByHash(ctx context.Context, pool *pgxpool.Pool, hash string) (*Match, error) {
	row := pool.QueryRow(ctx, `SELECT `+matchCols+` FROM matches m WHERE m.hash = $1`, hash)
	return scanMatch(row.Scan)
}

// MatchWithGroupName bundles a Match with the parent group's basic fields,
// fetched in a single round-trip.
type MatchWithGroupName struct {
	Match
	GroupName           string
	GroupTimezone       string
	GroupPerMatchAmount *float64
	GroupMonthlyAmount  *float64
	GroupIsPublic       bool
	GroupVotingEnabled  bool
}

// GetMatchByHashWithGroup fetches a match by hash AND the group's
// name/timezone/pricing/visibility in a single SQL query. Avoids the extra
// round-trip of calling GetMatchByHash followed by GetGroupByID.
func GetMatchByHashWithGroup(ctx context.Context, pool *pgxpool.Pool, hash string) (*MatchWithGroupName, error) {
	row := pool.QueryRow(ctx, `
		SELECT `+matchCols+`,
		       g.name,
		       g.timezone,
		       g.per_match_amount::FLOAT8,
		       g.monthly_amount::FLOAT8,
		       g.is_public,
		       g.voting_enabled
		FROM matches m
		JOIN groups g ON g.id = m.group_id
		WHERE m.hash = $1`, hash)
	var m MatchWithGroupName
	err := row.Scan(
		&m.ID, &m.GroupID, &m.Number, &m.Hash,
		&m.MatchDate, &m.StartTime, &m.EndTime,
		&m.Location, &m.Address, &m.CourtType,
		&m.PlayersPerTeam, &m.MaxPlayers, &m.Notes,
		&m.Status, &m.VoteOpenDelayMinutes, &m.VoteDurationHours, &m.VoteNotified,
		&m.CreatedAt, &m.UpdatedAt,
		&m.GroupName,
		&m.GroupTimezone,
		&m.GroupPerMatchAmount,
		&m.GroupMonthlyAmount,
		&m.GroupIsPublic,
		&m.GroupVotingEnabled,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &m, nil
}

type CreateMatchParams struct {
	GroupID              uuid.UUID
	Hash                 string
	Number               int
	MatchDate            string
	StartTime            string
	EndTime              *string
	Location             string
	Address              *string
	CourtType            *string
	PlayersPerTeam       *int
	MaxPlayers           *int
	Notes                *string
	CreatedByID          uuid.UUID
	VoteOpenDelayMinutes int
	VoteDurationHours    int
}

func CreateMatch(ctx context.Context, pool *pgxpool.Pool, p CreateMatchParams) (*Match, error) {
	row := pool.QueryRow(ctx, `
		INSERT INTO matches
			(group_id, hash, number, match_date, start_time, end_time,
			 location, address, court_type, players_per_team, max_players, notes, created_by_id,
			 vote_open_delay_minutes, vote_duration_hours)
		VALUES
			($1,$2,$3,$4::DATE,$5::TIME,$6::TIME,
			 $7,$8,$9::court_type,$10,$11,$12,$13,
			 $14,$15)
		RETURNING `+matchReturnCols,
		p.GroupID, p.Hash, p.Number, p.MatchDate, p.StartTime, p.EndTime,
		p.Location, p.Address, p.CourtType, p.PlayersPerTeam, p.MaxPlayers, p.Notes, p.CreatedByID,
		p.VoteOpenDelayMinutes, p.VoteDurationHours,
	)
	return scanMatch(row.Scan)
}

type UpdateMatchParams struct {
	MatchDate      *string
	StartTime      *string
	EndTime        *string
	Location       *string
	Address        *string
	CourtType      *string
	PlayersPerTeam *int
	MaxPlayers     *int
	Notes          *string
	Status         *string
}

func UpdateMatch(ctx context.Context, pool *pgxpool.Pool, matchID uuid.UUID, p UpdateMatchParams) (*Match, error) {
	m, err := GetMatchByID(ctx, pool, matchID)
	if err != nil {
		return nil, err
	}
	if p.MatchDate != nil {
		m.MatchDate = *p.MatchDate
	}
	if p.StartTime != nil {
		m.StartTime = *p.StartTime
	}
	if p.EndTime != nil {
		m.EndTime = p.EndTime
	}
	if p.Location != nil {
		m.Location = *p.Location
	}
	if p.Address != nil {
		m.Address = p.Address
	}
	if p.CourtType != nil {
		m.CourtType = p.CourtType
	}
	if p.PlayersPerTeam != nil {
		m.PlayersPerTeam = p.PlayersPerTeam
	}
	if p.MaxPlayers != nil {
		m.MaxPlayers = p.MaxPlayers
	}
	if p.Notes != nil {
		m.Notes = p.Notes
	}
	if p.Status != nil {
		m.Status = *p.Status
	}
	row := pool.QueryRow(ctx, `
		UPDATE matches SET
			match_date=$1::DATE, start_time=$2::TIME, end_time=$3::TIME,
			location=$4, address=$5, court_type=$6::court_type,
			players_per_team=$7, max_players=$8, notes=$9, status=$10
		WHERE id=$11
		RETURNING `+matchReturnCols,
		m.MatchDate, m.StartTime, m.EndTime,
		m.Location, m.Address, m.CourtType,
		m.PlayersPerTeam, m.MaxPlayers, m.Notes, m.Status, matchID,
	)
	return scanMatch(row.Scan)
}

func DeleteMatch(ctx context.Context, pool *pgxpool.Pool, matchID uuid.UUID) error {
	_, err := pool.Exec(ctx, `DELETE FROM matches WHERE id = $1`, matchID)
	return err
}

func NextMatchNumber(ctx context.Context, pool *pgxpool.Pool, groupID uuid.UUID) (int, error) {
	var n int
	err := pool.QueryRow(ctx,
		`SELECT COALESCE(MAX(number), 0) + 1 FROM matches WHERE group_id = $1`,
		groupID).Scan(&n)
	return n, err
}

func GetDiscoverMatches(ctx context.Context, pool *pgxpool.Pool, playerID *uuid.UUID, limit, offset int) ([]DiscoverMatch, error) {
	query := `
		SELECT
			m.id, m.group_id, m.number, m.hash,
			m.match_date::TEXT, m.start_time::TEXT, m.end_time::TEXT,
			m.location, m.address, m.court_type::TEXT,
			m.players_per_team, m.max_players, m.notes,
			m.status::TEXT, m.vote_open_delay_minutes, m.vote_duration_hours, m.vote_notified,
			m.created_at, m.updated_at,
			g.name, g.timezone,
			COUNT(a.id) FILTER (WHERE a.status = 'confirmed') AS confirmed_count
		FROM matches m
		JOIN groups g ON g.id = m.group_id
		LEFT JOIN attendances a ON a.match_id = m.id
		WHERE g.is_public = TRUE
		  AND m.status = 'open'
	`

	args := []interface{}{}
	argNum := 0

	if playerID != nil {
		query += `
		  AND m.group_id NOT IN (SELECT group_id FROM group_members WHERE player_id = $` + strconv.Itoa(argNum+1) + `)
		  AND m.id NOT IN (SELECT match_id FROM match_waitlist WHERE player_id = $` + strconv.Itoa(argNum+2) + `)`
		args = append(args, *playerID, *playerID)
		argNum += 2
	}

	argNum++
	argNum++
	query += `
		GROUP BY m.id, g.name, g.timezone
		HAVING m.max_players IS NULL OR COUNT(a.id) FILTER (WHERE a.status = 'confirmed') < m.max_players
		ORDER BY m.match_date ASC, m.start_time ASC
		LIMIT $` + strconv.Itoa(argNum-1) + ` OFFSET $` + strconv.Itoa(argNum) + `
	`
	args = append(args, limit, offset)

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]DiscoverMatch, 0)
	for rows.Next() {
		var d DiscoverMatch
		if err := rows.Scan(
			&d.ID, &d.GroupID, &d.Number, &d.Hash,
			&d.MatchDate, &d.StartTime, &d.EndTime,
			&d.Location, &d.Address, &d.CourtType,
			&d.PlayersPerTeam, &d.MaxPlayers, &d.Notes,
			&d.Status, &d.VoteOpenDelayMinutes, &d.VoteDurationHours, &d.VoteNotified,
			&d.CreatedAt, &d.UpdatedAt,
			&d.GroupName, &d.GroupTimezone, &d.ConfirmedCount,
		); err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	return result, rows.Err()
}

func GetAttendancesForMatch(ctx context.Context, pool *pgxpool.Pool, matchID uuid.UUID) ([]AttendanceWithPlayer, error) {
	rows, err := pool.Query(ctx, `
		SELECT
			a.id, a.match_id, a.player_id, a.status::TEXT, a.updated_at,
			p.name, p.nickname, p.avatar_url, p.role::TEXT,
			COALESCE(gm.position, 'mei'),
			gm.nickname
		FROM attendances a
		JOIN players p ON p.id = a.player_id
		JOIN matches m ON m.id = a.match_id
		LEFT JOIN group_members gm ON gm.player_id = a.player_id AND gm.group_id = m.group_id
		WHERE a.match_id = $1
		  AND p.role != 'admin'
		ORDER BY p.name`, matchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]AttendanceWithPlayer, 0)
	for rows.Next() {
		var a AttendanceWithPlayer
		if err := rows.Scan(
			&a.ID, &a.MatchID, &a.PlayerID, &a.Status, &a.UpdatedAt,
			&a.PlayerName, &a.PlayerNickname, &a.PlayerAvatarURL, &a.PlayerRole,
			&a.Position, &a.GroupNickname,
		); err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	return result, rows.Err()
}

func SetAttendance(ctx context.Context, pool *pgxpool.Pool, matchID, playerID uuid.UUID, status string) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO attendances (match_id, player_id, status)
		VALUES ($1, $2, $3)
		ON CONFLICT (match_id, player_id) DO UPDATE SET status = EXCLUDED.status`,
		matchID, playerID, status)
	return err
}

func CountAttendances(ctx context.Context, pool *pgxpool.Pool, matchID uuid.UUID, status string) (int, error) {
	var n int
	err := pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM attendances WHERE match_id=$1 AND status=$2`,
		matchID, status).Scan(&n)
	return n, err
}

func BulkSetAttendancePending(ctx context.Context, pool *pgxpool.Pool, matchID uuid.UUID, playerIDs []uuid.UUID) error {
	for _, pid := range playerIDs {
		_, err := pool.Exec(ctx, `
			INSERT INTO attendances (match_id, player_id, status)
			VALUES ($1, $2, 'pending')
			ON CONFLICT (match_id, player_id) DO NOTHING`, matchID, pid)
		if err != nil {
			return err
		}
	}
	return nil
}

func GetGroupMemberPlayerIDs(ctx context.Context, pool *pgxpool.Pool, groupID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := pool.Query(ctx,
		`SELECT player_id FROM group_members WHERE group_id = $1`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func GetNonAdminMemberPlayerIDs(ctx context.Context, pool *pgxpool.Pool, groupID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := pool.Query(ctx,
		`SELECT gm.player_id FROM group_members gm
		 JOIN players p ON p.id = gm.player_id
		 WHERE gm.group_id = $1 AND p.role != 'admin'`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func GetOpenMatchesForGroup(ctx context.Context, pool *pgxpool.Pool, groupID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := pool.Query(ctx,
		`SELECT id FROM matches WHERE group_id=$1 AND status IN ('open','in_progress')`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func GetMatchPlayerStats(ctx context.Context, pool *pgxpool.Pool, matchID uuid.UUID) ([]MatchPlayerStat, error) {
	rows, err := pool.Query(ctx, `
		SELECT s.id, s.match_id, s.player_id, s.goals, s.assists,
		       p.name, p.avatar_url
		FROM match_player_stats s
		JOIN players p ON p.id = s.player_id
		WHERE s.match_id = $1
		ORDER BY s.goals DESC, s.assists DESC`, matchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	stats := make([]MatchPlayerStat, 0)
	for rows.Next() {
		var s MatchPlayerStat
		if err := rows.Scan(&s.ID, &s.MatchID, &s.PlayerID, &s.Goals, &s.Assists, &s.PlayerName, &s.AvatarURL); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

func UpsertMatchPlayerStat(ctx context.Context, pool *pgxpool.Pool, matchID, playerID, recordedBy uuid.UUID, goals, assists int) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO match_player_stats (match_id, player_id, goals, assists, recorded_by)
		VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT (match_id, player_id)
		DO UPDATE SET goals=EXCLUDED.goals, assists=EXCLUDED.assists, recorded_by=EXCLUDED.recorded_by`,
		matchID, playerID, goals, assists, recordedBy)
	return err
}
