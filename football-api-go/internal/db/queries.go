// Package db contains hand-crafted query functions for Phase 1.
// These will be replaced by sqlc-generated code when `make generate` runs.
package db

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PlayerRole mirrors the player_role PostgreSQL enum.
type PlayerRole string

const (
	PlayerRoleAdmin  PlayerRole = "admin"
	PlayerRolePlayer PlayerRole = "player"
)

// Player mirrors the players table.
type Player struct {
	ID                 uuid.UUID  `json:"id"`
	Name               string     `json:"name"`
	Nickname           *string    `json:"nickname"`
	WhatsApp           string     `json:"whatsapp"`
	PasswordHash       string     `json:"-"`
	Role               PlayerRole `json:"role"`
	Active             bool       `json:"active"`
	MustChangePassword bool       `json:"must_change_password"`
	AvatarURL          *string    `json:"avatar_url"`
	ChatEnabled        bool       `json:"chat_enabled"`
	ChatReqCount       int32      `json:"-"`
	ChatReqWindow      *time.Time `json:"-"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// RefreshToken mirrors the refresh_tokens table.
type RefreshToken struct {
	ID        uuid.UUID
	PlayerID  uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

// MCPToken mirrors the mcp_tokens table.
type MCPToken struct {
	ID          uuid.UUID
	PlayerID    uuid.UUID
	Name        string
	TokenHash   string
	TokenPrefix string
	ExpiresAt   *time.Time
	CreatedAt   time.Time
	LastUsedAt  *time.Time
	RevokedAt   *time.Time
}

var ErrNotFound = pgx.ErrNoRows

// PlayerSelectCols is the column list for SELECT queries on the players table.
const PlayerSelectCols = `
	id, name, nickname, whatsapp, password_hash,
	role, active, must_change_password, avatar_url,
	chat_enabled, chat_req_count, chat_req_window,
	created_at, updated_at`

// ScanPlayer scans player fields from any pgx scan function (Row or Rows).
func ScanPlayer(scanFn func(dest ...any) error) (*Player, error) {
	var p Player
	err := scanFn(
		&p.ID, &p.Name, &p.Nickname, &p.WhatsApp, &p.PasswordHash,
		&p.Role, &p.Active, &p.MustChangePassword, &p.AvatarURL,
		&p.ChatEnabled, &p.ChatReqCount, &p.ChatReqWindow,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

// scanPlayer wraps ScanPlayer for the legacy pgx.Row interface used in auth queries.
func scanPlayer(row pgx.Row) (*Player, error) {
	return ScanPlayer(row.Scan)
}

// CreatePlayerParams is the canonical params type for creating players.
// (Alias of CreatePlayerArgs for backward compat.)
type CreatePlayerParams = CreatePlayerArgs

const playerColumns = `
	id, name, nickname, whatsapp, password_hash,
	role, active, must_change_password, avatar_url,
	chat_enabled, chat_req_count, chat_req_window,
	created_at, updated_at`

// GetPlayerByWhatsApp fetches an active player by their WhatsApp number.
func GetPlayerByWhatsApp(ctx context.Context, pool *pgxpool.Pool, whatsapp string) (*Player, error) {
	row := pool.QueryRow(ctx,
		`SELECT`+playerColumns+` FROM players WHERE whatsapp = $1 AND active = true`,
		whatsapp,
	)
	p, err := scanPlayer(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("GetPlayerByWhatsApp: %w", err)
	}
	return p, nil
}

// GetPlayerByID fetches an active player by UUID.
func GetPlayerByID(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*Player, error) {
	row := pool.QueryRow(ctx,
		`SELECT`+playerColumns+` FROM players WHERE id = $1 AND active = true`,
		id,
	)
	p, err := scanPlayer(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("GetPlayerByID: %w", err)
	}
	return p, nil
}

// CreatePlayerArgs holds parameters for CreatePlayer.
type CreatePlayerArgs struct {
	Name         string
	Nickname     *string
	WhatsApp     string
	PasswordHash string
}

// CreatePlayer inserts a new player and returns the created record.
func CreatePlayer(ctx context.Context, pool *pgxpool.Pool, args CreatePlayerArgs) (*Player, error) {
	row := pool.QueryRow(ctx,
		`INSERT INTO players (name, nickname, whatsapp, password_hash, role)
		 VALUES ($1, $2, $3, $4, 'player')
		 RETURNING`+playerColumns,
		args.Name, args.Nickname, args.WhatsApp, args.PasswordHash,
	)
	p, err := scanPlayer(row)
	if err != nil {
		return nil, fmt.Errorf("CreatePlayer: %w", err)
	}
	return p, nil
}

// UpdatePlayerPassword updates the password hash and clears must_change_password.
func UpdatePlayerPassword(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, hash string) error {
	_, err := pool.Exec(ctx,
		`UPDATE players SET password_hash = $1, must_change_password = false WHERE id = $2`,
		hash, id,
	)
	return err
}

// UpdatePlayerMustChangePassword sets the must_change_password flag.
func UpdatePlayerMustChangePassword(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, val bool) error {
	_, err := pool.Exec(ctx,
		`UPDATE players SET must_change_password = $1 WHERE id = $2`,
		val, id,
	)
	return err
}

// GetPlayerByMCPToken validates an MCP token hash and returns the associated player.
// Updates last_used_at as a side-effect.
func GetPlayerByMCPToken(ctx context.Context, pool *pgxpool.Pool, tokenHash string) (*Player, error) {
	row := pool.QueryRow(ctx,
		`SELECT p.`+playerColumns+`
		 FROM players p
		 JOIN mcp_tokens m ON m.player_id = p.id
		 WHERE m.token_hash = $1
		   AND m.revoked_at IS NULL
		   AND (m.expires_at IS NULL OR m.expires_at > now())
		   AND p.active = true`,
		tokenHash,
	)
	p, err := scanPlayer(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("GetPlayerByMCPToken: %w", err)
	}
	// Update last_used_at asynchronously (best-effort)
	go func() { //nolint:gosec
		_, _ = pool.Exec(context.Background(),
			`UPDATE mcp_tokens SET last_used_at = now() WHERE token_hash = $1`, tokenHash)
	}()
	return p, nil
}

// HashToken returns the SHA-256 hex digest of a token string.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// CreateRefreshToken inserts a new refresh token (stored as hash).
func CreateRefreshToken(ctx context.Context, pool *pgxpool.Pool, playerID uuid.UUID, tokenHash string) error {
	_, err := pool.Exec(ctx,
		`INSERT INTO refresh_tokens (player_id, token_hash, expires_at)
		 VALUES ($1, $2, now() + interval '30 days')`,
		playerID, tokenHash,
	)
	return err
}

// GetValidRefreshToken fetches a non-revoked, non-expired refresh token by hash.
func GetValidRefreshToken(ctx context.Context, pool *pgxpool.Pool, tokenHash string) (*RefreshToken, error) {
	var rt RefreshToken
	err := pool.QueryRow(ctx,
		`SELECT id, player_id, token_hash, expires_at, revoked_at, created_at
		 FROM refresh_tokens
		 WHERE token_hash = $1
		   AND revoked_at IS NULL
		   AND expires_at > now()`,
		tokenHash,
	).Scan(&rt.ID, &rt.PlayerID, &rt.TokenHash, &rt.ExpiresAt, &rt.RevokedAt, &rt.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("GetValidRefreshToken: %w", err)
	}
	return &rt, nil
}

// RevokeRefreshToken marks a single refresh token as revoked.
func RevokeRefreshToken(ctx context.Context, pool *pgxpool.Pool, tokenHash string) error {
	_, err := pool.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = now() WHERE token_hash = $1`,
		tokenHash,
	)
	return err
}

// RevokeAllRefreshTokensForPlayer revokes every refresh token for a player.
func RevokeAllRefreshTokensForPlayer(ctx context.Context, pool *pgxpool.Pool, playerID uuid.UUID) error {
	_, err := pool.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = now()
		 WHERE player_id = $1 AND revoked_at IS NULL`,
		playerID,
	)
	return err
}

// PlayerMatch row for player's match history
type PlayerMatch struct {
	ID             uuid.UUID
	GroupID        uuid.UUID
	Number         int
	Hash           string
	MatchDate      string
	StartTime      string
	EndTime        *string
	Location       string
	Address        *string
	CourtType      *string
	PlayersPerTeam *int
	MaxPlayers     *int
	Notes          *string
	Status         string
	CreatedAt      string
	UpdatedAt      string
	GroupName      string
	GroupTimezone  string
	MyAttendance   string
}

// GetPlayerMatches returns matches for a player
func GetPlayerMatches(ctx context.Context, pool *pgxpool.Pool, playerID uuid.UUID) ([]PlayerMatch, error) {
	rows, err := pool.Query(ctx, `
		SELECT
			m.id, m.group_id, m.number, m.hash,
			m.match_date::TEXT, m.start_time::TEXT, m.end_time::TEXT,
			m.location, m.address, m.court_type::TEXT,
			m.players_per_team, m.max_players, m.notes,
			m.status::TEXT, m.created_at::TEXT, m.updated_at::TEXT,
			g.name, g.timezone,
			a.status::TEXT
		FROM matches m
		JOIN groups g ON g.id = m.group_id
		JOIN attendances a ON a.match_id = m.id AND a.player_id = $1
		ORDER BY m.match_date DESC, m.start_time DESC`, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var matches []PlayerMatch
	for rows.Next() {
		var m PlayerMatch
		if err := rows.Scan(&m.ID, &m.GroupID, &m.Number, &m.Hash,
			&m.MatchDate, &m.StartTime, &m.EndTime,
			&m.Location, &m.Address, &m.CourtType,
			&m.PlayersPerTeam, &m.MaxPlayers, &m.Notes,
			&m.Status, &m.CreatedAt, &m.UpdatedAt,
			&m.GroupName, &m.GroupTimezone, &m.MyAttendance); err != nil {
			return nil, err
		}
		matches = append(matches, m)
	}
	return matches, rows.Err()
}

// GetPlayerStatsMinutes returns total minutes played
func GetPlayerStatsMinutes(ctx context.Context, pool *pgxpool.Pool, playerID uuid.UUID) (int, error) {
	var minutes int
	err := pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(
			EXTRACT(EPOCH FROM (
				CASE
					WHEN m.end_time IS NOT NULL
					THEN m.end_time::INTERVAL - m.start_time::INTERVAL
					ELSE INTERVAL '90 minutes'
				END
			)) / 60
		), 0)::INT
		FROM attendances a
		JOIN matches m ON m.id = a.match_id
		WHERE a.player_id = $1 AND a.status = 'confirmed' AND m.status = 'closed'`, playerID).Scan(&minutes)
	return minutes, err
}

// GetPlatformMatchStats returns closed matches, unique players, and total minutes for platform stats
func GetPlatformMatchStats(ctx context.Context, pool *pgxpool.Pool) (closedMatches, uniquePlayers, totalMinutes int, err error) {
	var totalMinutesFloat float64
	err = pool.QueryRow(ctx, `
		SELECT
			COALESCE(COUNT(*) FILTER (WHERE m.status='closed'), 0),
			COALESCE(COUNT(DISTINCT a.player_id) FILTER (WHERE m.status='closed'), 0),
			COALESCE(SUM(EXTRACT(EPOCH FROM (m.end_time - m.start_time)) / 60.0) FILTER (WHERE m.status='closed'), 0)
		FROM attendances a
		JOIN matches m ON m.id = a.match_id
		WHERE a.status = 'confirmed'`).Scan(&closedMatches, &uniquePlayers, &totalMinutesFloat)
	totalMinutes = int(totalMinutesFloat)
	return
}

// GroupStat row for player's stats per group
type GroupStat struct {
	GroupID   uuid.UUID
	GroupName string
	Matches   int
}

// GetPlayerGroupStats returns match counts per group for a player
func GetPlayerGroupStats(ctx context.Context, pool *pgxpool.Pool, playerID uuid.UUID) ([]GroupStat, error) {
	rows, err := pool.Query(ctx, `
		SELECT g.id, g.name, COUNT(a.id)
		FROM attendances a
		JOIN matches m ON m.id = a.match_id
		JOIN groups g ON g.id = m.group_id
		WHERE a.player_id = $1 AND a.status = 'confirmed'
		GROUP BY g.id, g.name
		ORDER BY g.name`, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var stats []GroupStat
	for rows.Next() {
		var s GroupStat
		if err := rows.Scan(&s.GroupID, &s.GroupName, &s.Matches); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

// GetPlayerGoalsAssists returns total goals and assists for a player
func GetPlayerGoalsAssists(ctx context.Context, pool *pgxpool.Pool, playerID uuid.UUID) (goals, assists int, err error) {
	err = pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(goals),0), COALESCE(SUM(assists),0)
		FROM match_player_stats WHERE player_id = $1`, playerID).Scan(&goals, &assists)
	return
}

// ── Full player stats (Rachão Score) ─────────────────────────────────────────

// PlayerFullStatsRow holds the scalar aggregates portion of full stats.
type PlayerFullStatsRow struct {
	TotalMatchesConfirmed int
	TotalMinutesPlayed    int
	TotalVotePoints       int
	Top1Count             int
	Top5Count             int
	TotalFlopVotes        int
	TotalGoals            int
	TotalAssists          int
	AttendanceRate        int
}

// PlayerAttendanceHistoryItem represents one closed-match attendance for a player.
type PlayerAttendanceHistoryItem struct {
	MatchDate string
	GroupName string
	Status    string
}

// PlayerMonthlyStat is one bucket of monthly aggregated stats.
type PlayerMonthlyStat struct {
	Month            string // "YYYY-MM"
	MatchesConfirmed int
	MinutesPlayed    int
}

// PlayerGroupFullStat is a per-group view of a player's history.
type PlayerGroupFullStat struct {
	GroupID          string
	GroupName        string
	SkillStars       int
	Position         string
	Role             string
	MatchesConfirmed int
}

// GetPlayerFullStatsScalar runs the CTE that computes all single-value
// aggregates used by the Rachão Score page in one round-trip.
func GetPlayerFullStatsScalar(ctx context.Context, pool *pgxpool.Pool, playerID uuid.UUID) (*PlayerFullStatsRow, error) {
	var r PlayerFullStatsRow
	err := pool.QueryRow(ctx, `
		WITH
		totals AS (
			SELECT
				COUNT(*)::int AS total_matches_confirmed,
				COALESCE(SUM(
					CASE WHEN m.end_time IS NOT NULL
					     THEN GREATEST(0, EXTRACT(EPOCH FROM (m.end_time - m.start_time)) / 60)
					     ELSE 0 END
				), 0)::int AS total_minutes_played
			FROM attendances a
			JOIN matches m ON m.id = a.match_id
			WHERE a.player_id = $1
			  AND a.status = 'confirmed'
			  AND m.status = 'closed'
		),
		vote_pts AS (
			SELECT
				COALESCE(SUM(mvt.points), 0)::int                AS total_vote_points,
				COUNT(*) FILTER (WHERE mvt.position = 1)::int    AS top1_count,
				COUNT(*)::int                                    AS top5_count
			FROM match_vote_top5 mvt
			JOIN match_votes mv ON mv.id = mvt.vote_id
			WHERE mvt.player_id = $1
		),
		flop_cnt AS (
			SELECT COUNT(*)::int AS total_flop_votes
			FROM match_vote_flop mvf
			JOIN match_votes mv ON mv.id = mvf.vote_id
			WHERE mvf.player_id = $1
		),
		goals_ast AS (
			SELECT
				COALESCE(SUM(mps.goals),   0)::int AS total_goals,
				COALESCE(SUM(mps.assists), 0)::int AS total_assists
			FROM match_player_stats mps
			WHERE mps.player_id = $1
		),
		att_rate AS (
			SELECT
				COUNT(*) FILTER (WHERE a.status = 'confirmed')::int AS confirmed_cnt,
				COUNT(*) FILTER (WHERE a.status = 'declined')::int  AS declined_cnt
			FROM attendances a
			JOIN matches m ON m.id = a.match_id
			WHERE a.player_id = $1
			  AND m.status = 'closed'
		)
		SELECT
			t.total_matches_confirmed,
			t.total_minutes_played,
			v.total_vote_points,
			v.top1_count,
			v.top5_count,
			f.total_flop_votes,
			g.total_goals,
			g.total_assists,
			CASE WHEN (r.confirmed_cnt + r.declined_cnt) = 0 THEN 0
			     ELSE ROUND(r.confirmed_cnt * 100.0 / (r.confirmed_cnt + r.declined_cnt))
			END::int AS attendance_rate
		FROM totals t, vote_pts v, flop_cnt f, goals_ast g, att_rate r
	`, playerID).Scan(
		&r.TotalMatchesConfirmed, &r.TotalMinutesPlayed,
		&r.TotalVotePoints, &r.Top1Count, &r.Top5Count,
		&r.TotalFlopVotes, &r.TotalGoals, &r.TotalAssists,
		&r.AttendanceRate,
	)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// GetPlayerAttendanceHistory returns closed-match attendances (confirmed or
// declined), most recent first. Used by callers to derive streaks and recent
// matches list.
func GetPlayerAttendanceHistory(ctx context.Context, pool *pgxpool.Pool, playerID uuid.UUID) ([]PlayerAttendanceHistoryItem, error) {
	rows, err := pool.Query(ctx, `
		SELECT
			m.match_date::text AS match_date,
			g.name             AS group_name,
			a.status::text
		FROM attendances a
		JOIN matches m ON m.id = a.match_id
		JOIN groups  g ON g.id = m.group_id
		WHERE a.player_id = $1
		  AND m.status = 'closed'
		  AND a.status IN ('confirmed', 'declined')
		ORDER BY m.match_date DESC`, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]PlayerAttendanceHistoryItem, 0)
	for rows.Next() {
		var h PlayerAttendanceHistoryItem
		if err := rows.Scan(&h.MatchDate, &h.GroupName, &h.Status); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// GetPlayerMonthlyStats returns confirmed-match counts and minutes per month
// for the last 6 months. Missing months are NOT returned — callers must pad.
func GetPlayerMonthlyStats(ctx context.Context, pool *pgxpool.Pool, playerID uuid.UUID) ([]PlayerMonthlyStat, error) {
	rows, err := pool.Query(ctx, `
		SELECT
			TO_CHAR(m.match_date, 'YYYY-MM') AS month,
			COUNT(*)::int                    AS matches_confirmed,
			COALESCE(SUM(
				CASE WHEN m.end_time IS NOT NULL
				     THEN GREATEST(0, EXTRACT(EPOCH FROM (m.end_time - m.start_time)) / 60)
				     ELSE 0 END
			), 0)::int                       AS minutes_played
		FROM attendances a
		JOIN matches m ON m.id = a.match_id
		WHERE a.player_id = $1
		  AND a.status = 'confirmed'
		  AND m.status = 'closed'
		  AND m.match_date >= DATE_TRUNC('month', NOW() AT TIME ZONE 'America/Sao_Paulo') - INTERVAL '5 months'
		GROUP BY TO_CHAR(m.match_date, 'YYYY-MM')
		ORDER BY month`, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]PlayerMonthlyStat, 0)
	for rows.Next() {
		var s PlayerMonthlyStat
		if err := rows.Scan(&s.Month, &s.MatchesConfirmed, &s.MinutesPlayed); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// GetPlayerGroupFullStats returns per-group skill/position/role/match-count
// rows for the Rachão Score "Grupos" section.
func GetPlayerGroupFullStats(ctx context.Context, pool *pgxpool.Pool, playerID uuid.UUID) ([]PlayerGroupFullStat, error) {
	rows, err := pool.Query(ctx, `
		SELECT
			gm.group_id::text AS group_id,
			g.name            AS group_name,
			COALESCE(gm.skill_stars, 2)::int   AS skill_stars,
			COALESCE(gm.position, 'mei')        AS position,
			gm.role::text                       AS role,
			COUNT(a.match_id) FILTER (
				WHERE a.status = 'confirmed' AND m.status = 'closed'
			)::int AS matches_confirmed
		FROM group_members gm
		JOIN groups g ON g.id = gm.group_id
		LEFT JOIN matches m     ON m.group_id = gm.group_id
		LEFT JOIN attendances a ON a.match_id = m.id AND a.player_id = gm.player_id
		WHERE gm.player_id = $1
		GROUP BY gm.group_id, g.name, gm.skill_stars, gm.position, gm.role
		ORDER BY matches_confirmed DESC`, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]PlayerGroupFullStat, 0)
	for rows.Next() {
		var g PlayerGroupFullStat
		if err := rows.Scan(
			&g.GroupID, &g.GroupName,
			&g.SkillStars, &g.Position, &g.Role, &g.MatchesConfirmed,
		); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// GetPublicPlayerStats returns match count and goals/assists for a player
func GetPublicPlayerStats(ctx context.Context, pool *pgxpool.Pool, playerID uuid.UUID) (totalConfirmed, totalGoals, totalAssists int, err error) {
	err = pool.QueryRow(ctx, `
		SELECT
			COALESCE((SELECT COUNT(*) FROM attendances a JOIN matches m ON m.id = a.match_id WHERE a.player_id = $1 AND a.status = 'confirmed' AND m.status = 'closed'), 0),
			COALESCE((SELECT SUM(goals) FROM match_player_stats WHERE player_id = $1), 0),
			COALESCE((SELECT SUM(assists) FROM match_player_stats WHERE player_id = $1), 0)
	`, playerID).Scan(&totalConfirmed, &totalGoals, &totalAssists)
	return
}

// UpdatePlayerProfile updates player name, nickname, and password
func UpdatePlayerProfile(ctx context.Context, pool *pgxpool.Pool, playerID uuid.UUID, name, nickname, passwordHash string) error {
	_, err := pool.Exec(ctx, `
		UPDATE players SET name=$1, nickname=$2, password_hash=$3 WHERE id=$4`,
		name, nickname, passwordHash, playerID)
	return err
}

// ListPlayersActive returns active players
func ListPlayersActive(ctx context.Context, pool *pgxpool.Pool, limit, offset int, activeOnly bool) ([]*Player, error) {
	query := `SELECT ` + PlayerSelectCols + ` FROM players WHERE role = 'player'`
	if activeOnly {
		query += ` AND active = TRUE`
	}
	query += ` ORDER BY name LIMIT $1 OFFSET $2`

	rows, err := pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var players []*Player
	for rows.Next() {
		p, err := ScanPlayer(rows.Scan)
		if err != nil {
			return nil, err
		}
		players = append(players, p)
	}
	return players, rows.Err()
}

// GetSignupStats returns player signup statistics
func GetSignupStats(ctx context.Context, pool *pgxpool.Pool) (total, last7, last30 int, err error) {
	err = pool.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE role='player'),
			COUNT(*) FILTER (WHERE role='player' AND created_at >= NOW() - INTERVAL '7 days'),
			COUNT(*) FILTER (WHERE role='player' AND created_at >= NOW() - INTERVAL '30 days')
		FROM players WHERE active=TRUE`).Scan(&total, &last7, &last30)
	return
}

// RecentSignup row for signup stats
type RecentSignup struct {
	ID        uuid.UUID
	Name      string
	Nickname  *string
	WhatsApp  string
	Active    bool
	CreatedAt time.Time
}

// GetRecentSignups returns recent active players
func GetRecentSignups(ctx context.Context, pool *pgxpool.Pool, limit int) ([]RecentSignup, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, name, nickname, whatsapp, active, created_at
		FROM players WHERE role='player' AND active=TRUE
		ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var recent []RecentSignup
	for rows.Next() {
		var s RecentSignup
		if err := rows.Scan(&s.ID, &s.Name, &s.Nickname, &s.WhatsApp, &s.Active, &s.CreatedAt); err != nil {
			return nil, err
		}
		recent = append(recent, s)
	}
	return recent, rows.Err()
}

// UpdatePlayerAvatarURL updates player's avatar URL
func UpdatePlayerAvatarURL(ctx context.Context, pool *pgxpool.Pool, playerID uuid.UUID, avatarURL string) error {
	_, err := pool.Exec(ctx, `UPDATE players SET avatar_url=$2 WHERE id=$1`, playerID, avatarURL)
	return err
}

// DeletePlayerAvatarURL removes player's avatar URL
func DeletePlayerAvatarURL(ctx context.Context, pool *pgxpool.Pool, playerID uuid.UUID) error {
	_, err := pool.Exec(ctx, `UPDATE players SET avatar_url=NULL WHERE id=$1`, playerID)
	return err
}

// ChatUser row for chat users listing
type ChatUser struct {
	ID          uuid.UUID
	Name        string
	WhatsApp    string
	ChatEnabled bool
	CreatedAt   time.Time
}

// ListChatUsers returns all chat users for admin
func ListChatUsers(ctx context.Context, pool *pgxpool.Pool) ([]ChatUser, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, name, whatsapp, chat_enabled, created_at
		FROM players
		WHERE role = 'player'
		ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []ChatUser
	for rows.Next() {
		var u ChatUser
		if err := rows.Scan(&u.ID, &u.Name, &u.WhatsApp,
			&u.ChatEnabled, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// AdminStats row for admin dashboard stats
type AdminStats struct {
	TotalMatches    int
	TotalGroups     int
	TotalPlayers    int
	PlatformMinutes int
	SignupsTotal    int
	Signups7D       int
	Signups30D      int
	TotalReviews    int
}

// GetAdminStats returns platform-wide statistics
func GetAdminStats(ctx context.Context, pool *pgxpool.Pool) (*AdminStats, error) {
	var stats AdminStats
	err := pool.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*)::INT FROM matches)  AS total_matches,
			(SELECT COUNT(*)::INT FROM groups)   AS total_groups,
			(SELECT COUNT(*)::INT FROM players WHERE role != 'admin') AS total_players,
			(SELECT COALESCE(SUM(GREATEST(0,
				EXTRACT(EPOCH FROM (end_time::INTERVAL - start_time::INTERVAL)) / 60
			)),0)::INT
			 FROM matches WHERE status = 'closed' AND end_time IS NOT NULL) AS platform_minutes,
			(SELECT COUNT(*)::INT FROM players WHERE role != 'admin') AS signups_total,
			(SELECT COUNT(*)::INT FROM players WHERE role != 'admin'
			   AND created_at >= NOW() - INTERVAL '7 days')  AS signups_7d,
			(SELECT COUNT(*)::INT FROM players WHERE role != 'admin'
			   AND created_at >= NOW() - INTERVAL '30 days') AS signups_30d,
			(SELECT COUNT(*)::INT FROM app_reviews) AS total_reviews`).Scan(
		&stats.TotalMatches, &stats.TotalGroups, &stats.TotalPlayers,
		&stats.PlatformMinutes,
		&stats.SignupsTotal, &stats.Signups7D, &stats.Signups30D,
		&stats.TotalReviews,
	)
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

// AdminMatch row for matches list
type AdminMatch struct {
	ID        uuid.UUID
	Hash      string
	Number    int
	GroupID   uuid.UUID
	GroupName string
	MatchDate string
	StartTime string
	EndTime   *string
	Location  string
	Status    string
}

// CountMatches returns count of matches with optional status filter
func CountMatches(ctx context.Context, pool *pgxpool.Pool, status *string) (int, error) {
	var count int
	var err error
	if status != nil && *status != "" {
		err = pool.QueryRow(ctx, `SELECT COUNT(*)::INT FROM matches WHERE status=$1`, *status).Scan(&count)
	} else {
		err = pool.QueryRow(ctx, `SELECT COUNT(*)::INT FROM matches`).Scan(&count)
	}
	return count, err
}

// ListMatches returns matches with optional status filter
func ListMatches(ctx context.Context, pool *pgxpool.Pool, status *string, limit, offset int) ([]AdminMatch, error) {
	var rows pgx.Rows
	var err error
	query := `
		SELECT m.id, m.hash, m.number, m.group_id, g.name AS group_name,
		       m.match_date::TEXT, m.start_time::TEXT, m.end_time::TEXT,
		       m.location, m.status::TEXT
		FROM matches m JOIN groups g ON g.id = m.group_id`

	if status != nil && *status != "" {
		query += ` WHERE m.status = $1 ORDER BY m.match_date DESC, m.start_time DESC LIMIT $2 OFFSET $3`
		rows, err = pool.Query(ctx, query, *status, limit, offset)
	} else {
		query += ` ORDER BY m.match_date DESC, m.start_time DESC LIMIT $1 OFFSET $2`
		rows, err = pool.Query(ctx, query, limit, offset)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []AdminMatch
	for rows.Next() {
		var m AdminMatch
		if err := rows.Scan(&m.ID, &m.Hash, &m.Number, &m.GroupID, &m.GroupName,
			&m.MatchDate, &m.StartTime, &m.EndTime, &m.Location, &m.Status); err != nil {
			return nil, err
		}
		matches = append(matches, m)
	}
	return matches, rows.Err()
}

// AdminGroup row for groups list
type AdminGroup struct {
	ID           uuid.UUID
	Name         string
	Description  *string
	Slug         string
	CreatedAt    time.Time
	TotalMembers int
	TotalMatches int
}

// ListGroups returns paginated groups with stats
func ListGroups(ctx context.Context, pool *pgxpool.Pool, limit, offset int) (int, []AdminGroup, error) {
	var total int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*)::INT FROM groups`).Scan(&total); err != nil {
		return 0, nil, err
	}

	rows, err := pool.Query(ctx, `
		SELECT g.id, g.name, g.description, g.slug, g.created_at,
		       COUNT(DISTINCT gm.player_id)::INT AS total_members,
		       COUNT(DISTINCT m.id)::INT          AS total_matches
		FROM groups g
		LEFT JOIN group_members gm ON gm.group_id = g.id
		LEFT JOIN matches m        ON m.group_id  = g.id
		GROUP BY g.id
		ORDER BY g.created_at DESC
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()

	var groups []AdminGroup
	for rows.Next() {
		var g AdminGroup
		if err := rows.Scan(&g.ID, &g.Name, &g.Description, &g.Slug, &g.CreatedAt,
			&g.TotalMembers, &g.TotalMatches); err != nil {
			return 0, nil, err
		}
		groups = append(groups, g)
	}
	return total, groups, rows.Err()
}

// SubscriptionSummary stat item
type SubscriptionSummaryStat struct {
	Status       string
	Plan         string
	BillingCycle string
	Count        int
}

// GetSubscriptionSummary returns subscription summary stats
func GetSubscriptionSummary(ctx context.Context, pool *pgxpool.Pool) (totalPlayers int, stats []SubscriptionSummaryStat, err error) {
	if err = pool.QueryRow(ctx,
		`SELECT COUNT(*)::INT FROM players WHERE role != 'admin'`).Scan(&totalPlayers); err != nil {
		return
	}

	rows, err := pool.Query(ctx, `
		SELECT
			COALESCE(ps.status, 'active')        AS status,
			COALESCE(ps.plan, 'free')            AS plan,
			COALESCE(ps.billing_cycle, 'monthly') AS billing_cycle,
			COUNT(*)::INT AS cnt
		FROM players p
		LEFT JOIN player_subscriptions ps ON ps.player_id = p.id
		WHERE p.role != 'admin'
		GROUP BY ps.status, ps.plan, ps.billing_cycle`)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var s SubscriptionSummaryStat
		if err = rows.Scan(&s.Status, &s.Plan, &s.BillingCycle, &s.Count); err != nil {
			return
		}
		stats = append(stats, s)
	}
	err = rows.Err()
	return
}

// AdminSubscription row for subscription list
type AdminSubscription struct {
	PlayerID          uuid.UUID
	PlayerName        string
	Plan              string
	BillingCycle      *string
	Status            *string
	CurrentPeriodEnd  *time.Time
	GracePeriodEnd    *time.Time
	GatewayCustomerID *string
	GatewaySubID      *string
	CreatedAt         time.Time
}

// ListSubscriptionsParams for filtering
type ListSubscriptionsParams struct {
	Status string // optional
	Plan   string // optional
	Limit  int
	Offset int
}

// CountSubscriptions counts subscriptions with filters
func CountSubscriptions(ctx context.Context, pool *pgxpool.Pool, params ListSubscriptionsParams) (int, error) {
	var total int
	var err error
	if params.Status != "" && params.Plan != "" {
		err = pool.QueryRow(ctx, `
			SELECT COUNT(*)::INT
			FROM player_subscriptions ps JOIN players p ON p.id = ps.player_id
			WHERE p.role != 'admin' AND ps.status=$1 AND ps.plan=$2`, params.Status, params.Plan).Scan(&total)
	} else if params.Status != "" {
		err = pool.QueryRow(ctx, `
			SELECT COUNT(*)::INT
			FROM player_subscriptions ps JOIN players p ON p.id = ps.player_id
			WHERE p.role != 'admin' AND ps.status=$1`, params.Status).Scan(&total)
	} else if params.Plan != "" {
		err = pool.QueryRow(ctx, `
			SELECT COUNT(*)::INT
			FROM player_subscriptions ps JOIN players p ON p.id = ps.player_id
			WHERE p.role != 'admin' AND ps.plan=$1`, params.Plan).Scan(&total)
	} else {
		err = pool.QueryRow(ctx, `
			SELECT COUNT(*)::INT
			FROM player_subscriptions ps JOIN players p ON p.id = ps.player_id
			WHERE p.role != 'admin'`).Scan(&total)
	}
	return total, err
}

// ListSubscriptions returns paginated subscriptions with filters
func ListSubscriptions(ctx context.Context, pool *pgxpool.Pool, params ListSubscriptionsParams) ([]AdminSubscription, error) {
	baseSelect := `
		SELECT p.id AS player_id, p.name AS player_name,
		       ps.plan, ps.billing_cycle, ps.status,
		       ps.current_period_end, ps.grace_period_end,
		       ps.gateway_customer_id, ps.gateway_sub_id, ps.created_at
		FROM player_subscriptions ps
		JOIN players p ON p.id = ps.player_id
		WHERE p.role != 'admin'`

	var rows pgx.Rows
	var err error

	if params.Status != "" && params.Plan != "" {
		rows, err = pool.Query(ctx, baseSelect+
			` AND ps.status=$1 AND ps.plan=$2 ORDER BY ps.current_period_end ASC NULLS LAST LIMIT $3 OFFSET $4`,
			params.Status, params.Plan, params.Limit, params.Offset)
	} else if params.Status != "" {
		rows, err = pool.Query(ctx, baseSelect+
			` AND ps.status=$1 ORDER BY ps.current_period_end ASC NULLS LAST LIMIT $2 OFFSET $3`,
			params.Status, params.Limit, params.Offset)
	} else if params.Plan != "" {
		rows, err = pool.Query(ctx, baseSelect+
			` AND ps.plan=$1 ORDER BY ps.current_period_end ASC NULLS LAST LIMIT $2 OFFSET $3`,
			params.Plan, params.Limit, params.Offset)
	} else {
		rows, err = pool.Query(ctx, baseSelect+
			` ORDER BY ps.current_period_end ASC NULLS LAST LIMIT $1 OFFSET $2`,
			params.Limit, params.Offset)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []AdminSubscription
	for rows.Next() {
		var s AdminSubscription
		if err := rows.Scan(
			&s.PlayerID, &s.PlayerName, &s.Plan, &s.BillingCycle, &s.Status,
			&s.CurrentPeriodEnd, &s.GracePeriodEnd, &s.GatewayCustomerID, &s.GatewaySubID, &s.CreatedAt,
		); err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	return subs, rows.Err()
}

// AdminPlayer row for players list
type AdminPlayer struct {
	ID          uuid.UUID
	Name        string
	Nickname    *string
	WhatsApp    string
	Role        string
	Active      bool
	CreatedAt   time.Time
	AvatarURL   *string
	Plan        string
	TotalGroups int
}

// CountPlayers counts players with optional search
func CountPlayers(ctx context.Context, pool *pgxpool.Pool, search *string) (int, error) {
	var total int
	var err error
	if search != nil && *search != "" {
		pat := "%" + *search + "%"
		err = pool.QueryRow(ctx, `
			SELECT COUNT(*)::INT FROM players p
			WHERE p.role != 'admin'
			  AND (p.name ILIKE $1 OR p.nickname ILIKE $1 OR p.whatsapp LIKE $1)`, pat).Scan(&total)
	} else {
		err = pool.QueryRow(ctx,
			`SELECT COUNT(*)::INT FROM players p WHERE p.role != 'admin'`).Scan(&total)
	}
	return total, err
}

// ListPlayers returns paginated players with optional search
func ListPlayers(ctx context.Context, pool *pgxpool.Pool, search *string, limit, offset int) ([]AdminPlayer, error) {
	baseSelect := `
		SELECT p.id, p.name, p.nickname, p.whatsapp, p.role, p.active, p.created_at, p.avatar_url,
		       COALESCE(ps.plan, 'free') AS plan,
		       COUNT(DISTINCT gm.group_id)::INT AS total_groups
		FROM players p
		LEFT JOIN player_subscriptions ps ON ps.player_id = p.id
		LEFT JOIN group_members gm ON gm.player_id = p.id
		WHERE p.role != 'admin'`

	var rows pgx.Rows
	var err error

	if search != nil && *search != "" {
		pat := "%" + *search + "%"
		rows, err = pool.Query(ctx, baseSelect+
			` AND (p.name ILIKE $1 OR p.nickname ILIKE $1 OR p.whatsapp LIKE $1)
			  GROUP BY p.id, p.name, p.nickname, p.whatsapp, p.role, p.active, p.created_at, p.avatar_url, ps.plan
			  ORDER BY p.created_at DESC LIMIT $2 OFFSET $3`, pat, limit, offset)
	} else {
		rows, err = pool.Query(ctx, baseSelect+
			` GROUP BY p.id, p.name, p.nickname, p.whatsapp, p.role, p.active, p.created_at, p.avatar_url, ps.plan
			  ORDER BY p.created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var players []AdminPlayer
	for rows.Next() {
		var p AdminPlayer
		if err := rows.Scan(&p.ID, &p.Name, &p.Nickname, &p.WhatsApp, &p.Role, &p.Active, &p.CreatedAt, &p.AvatarURL, &p.Plan, &p.TotalGroups); err != nil {
			return nil, err
		}
		players = append(players, p)
	}
	return players, rows.Err()
}

// AndroidBetaSignup row for beta signups list
type AndroidBetaSignup struct {
	ID         uuid.UUID
	Email      string
	PlayerID   *uuid.UUID
	PlayerName *string
	CreatedAt  time.Time
}

// CountBetaSignups returns count of beta signups
func CountBetaSignups(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	var total int
	err := pool.QueryRow(ctx,
		`SELECT COUNT(*)::INT FROM android_beta_signups`).Scan(&total)
	return total, err
}

// ListBetaSignups returns paginated beta signups
func ListBetaSignups(ctx context.Context, pool *pgxpool.Pool, limit, offset int) ([]AndroidBetaSignup, error) {
	rows, err := pool.Query(ctx, `
		SELECT s.id, s.google_email, s.player_id, p.name AS player_name, s.created_at
		FROM android_beta_signups s
		LEFT JOIN players p ON p.id = s.player_id
		ORDER BY s.created_at DESC
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var signups []AndroidBetaSignup
	for rows.Next() {
		var s AndroidBetaSignup
		if err := rows.Scan(&s.ID, &s.Email, &s.PlayerID, &s.PlayerName, &s.CreatedAt); err != nil {
			return nil, err
		}
		signups = append(signups, s)
	}
	return signups, rows.Err()
}
