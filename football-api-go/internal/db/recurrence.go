package db

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// GetGroupsWithRecurrence returns all groups that have recurrence_enabled = true.
func GetGroupsWithRecurrence(ctx context.Context, pool *pgxpool.Pool) ([]*Group, error) {
	rows, err := pool.Query(ctx, `
		SELECT `+groupSelectCols+`
		FROM groups g
		WHERE g.recurrence_enabled = true`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var groups []*Group
	for rows.Next() {
		g, err := scanGroup(rows.Scan)
		if err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	return groups, rows.Err()
}

// HasOpenMatch returns true if the group has any match with status 'open' or 'in_progress'.
func HasOpenMatch(ctx context.Context, pool *pgxpool.Pool, groupID uuid.UUID) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM matches WHERE group_id=$1 AND status IN ('open','in_progress'))`,
		groupID).Scan(&exists)
	return exists, err
}

// GetLastMatch returns the most recent match for a group, ordered by match_date DESC.
func GetLastMatch(ctx context.Context, pool *pgxpool.Pool, groupID uuid.UUID) (*Match, error) {
	row := pool.QueryRow(ctx,
		`SELECT `+matchCols+`
		 FROM matches m
		 WHERE m.group_id = $1
		 ORDER BY m.match_date DESC, m.start_time DESC
		 LIMIT 1`,
		groupID,
	)
	return scanMatch(row.Scan)
}

// CreateMatchRecurrenceParams holds data for creating a match via the recurrence job.
type CreateMatchRecurrenceParams struct {
	GroupID              uuid.UUID
	Hash                 string
	Number               int
	MatchDate            string  // YYYY-MM-DD
	StartTime            string  // HH:MM:SS
	EndTime              *string // nullable
	Location             string
	Address              *string
	CourtType            *string
	PlayersPerTeam       *int
	MaxPlayers           *int
	Notes                *string
	VoteOpenDelayMinutes int
	VoteDurationHours    int
}

// CreateMatchForRecurrence inserts a new match row with vote delay settings inherited from the group.
func CreateMatchForRecurrence(ctx context.Context, pool *pgxpool.Pool, p CreateMatchRecurrenceParams) (*Match, error) {
	row := pool.QueryRow(ctx, `
		INSERT INTO matches
			(group_id, hash, number, match_date, start_time, end_time,
			 location, address, court_type, players_per_team, max_players, notes,
			 vote_open_delay_minutes, vote_duration_hours)
		VALUES
			($1,$2,$3,$4::DATE,$5::TIME,$6::TIME,
			 $7,$8,$9::court_type,$10,$11,$12,$13,$14)
		RETURNING `+matchReturnCols,
		p.GroupID, p.Hash, p.Number, p.MatchDate, p.StartTime, p.EndTime,
		p.Location, p.Address, p.CourtType, p.PlayersPerTeam, p.MaxPlayers, p.Notes,
		p.VoteOpenDelayMinutes, p.VoteDurationHours,
	)
	return scanMatch(row.Scan)
}

// CreateAttendances inserts pending attendance rows for each player (ignores conflicts).
func CreateAttendances(ctx context.Context, pool *pgxpool.Pool, matchID uuid.UUID, playerIDs []uuid.UUID) error {
	for _, pid := range playerIDs {
		_, err := pool.Exec(ctx, `
			INSERT INTO attendances (match_id, player_id, status)
			VALUES ($1, $2, 'pending')
			ON CONFLICT (match_id, player_id) DO NOTHING`,
			matchID, pid,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

// ClosePastMatches closes matches that should no longer be open/in_progress
// according to Brazil time (UTC-3). Returns total number of rows transitioned.
//
// Mirrors v1's MatchRepository.close_past_matches (4 transitions in one call):
//   1. OPEN/IN_PROGRESS → CLOSED when match_date is before today
//   2. IN_PROGRESS → CLOSED when match_date is today AND end_time has passed
//
// Note: the OPEN → IN_PROGRESS transition for today's matches that already
// started is handled separately by GetInProgressCandidates +
// TransitionToInProgress so the scheduler can also send "bola rolando" push.
func ClosePastMatches(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	// #1: date < today (BRT)
	r1, err := pool.Exec(ctx, `
		UPDATE matches
		SET status = 'closed'
		WHERE match_date < (CURRENT_TIMESTAMP AT TIME ZONE 'America/Sao_Paulo')::DATE
		  AND status IN ('open', 'in_progress')`)
	if err != nil {
		return 0, err
	}

	// #2: date = today AND end_time <= now (BRT) AND in_progress
	r2, err := pool.Exec(ctx, `
		UPDATE matches
		SET status = 'closed'
		WHERE status = 'in_progress'
		  AND match_date = (CURRENT_TIMESTAMP AT TIME ZONE 'America/Sao_Paulo')::DATE
		  AND end_time IS NOT NULL
		  AND end_time <= (CURRENT_TIMESTAMP AT TIME ZONE 'America/Sao_Paulo')::TIME`)
	if err != nil {
		return int(r1.RowsAffected()), err
	}

	return int(r1.RowsAffected() + r2.RowsAffected()), nil
}

// InProgressCandidate holds enough data to transition a match to in_progress and send push.
type InProgressCandidate struct {
	ID        uuid.UUID
	Hash      string
	GroupID   uuid.UUID
	GroupName string
}

// GetInProgressCandidates returns matches that should transition to 'in_progress':
// match_date = today (BRT), start_time <= now (BRT), status = 'open'.
func GetInProgressCandidates(ctx context.Context, pool *pgxpool.Pool) ([]InProgressCandidate, error) {
	brtNow := time.Now().UTC().Add(-3 * time.Hour)
	todayBRT := brtNow.Format("2006-01-02")
	nowTimeBRT := brtNow.Format("15:04:05")

	rows, err := pool.Query(ctx, `
		SELECT m.id, m.hash, g.id, g.name
		FROM matches m
		JOIN groups g ON g.id = m.group_id
		WHERE m.match_date::TEXT = $1
		  AND m.start_time::TEXT <= $2
		  AND m.status = 'open'`,
		todayBRT, nowTimeBRT,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []InProgressCandidate
	for rows.Next() {
		var c InProgressCandidate
		if err := rows.Scan(&c.ID, &c.Hash, &c.GroupID, &c.GroupName); err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

// TransitionToInProgress updates a list of matches to 'in_progress' (only if still 'open').
func TransitionToInProgress(ctx context.Context, pool *pgxpool.Pool, matchIDs []uuid.UUID) error {
	for _, id := range matchIDs {
		_, err := pool.Exec(ctx,
			`UPDATE matches SET status='in_progress' WHERE id=$1 AND status='open'`, id)
		if err != nil {
			return err
		}
	}
	return nil
}

// CheckAndIncrementChatRateLimit atomically checks and increments the player's
// chat request counter within a 1-hour sliding window.
// Returns (true, nil) if the request is allowed, (false, nil) if rate-limited.
func CheckAndIncrementChatRateLimit(ctx context.Context, pool *pgxpool.Pool, playerID uuid.UUID, limit int) (bool, error) {
	var count int32
	var window *time.Time
	err := pool.QueryRow(ctx,
		`SELECT chat_req_count, chat_req_window FROM players WHERE id=$1`,
		playerID).Scan(&count, &window)
	if err != nil {
		return false, err
	}

	now := time.Now().UTC()
	windowExpired := window == nil || now.Sub(*window) > time.Hour

	if windowExpired {
		_, err = pool.Exec(ctx,
			`UPDATE players SET chat_req_count=1, chat_req_window=$2 WHERE id=$1`,
			playerID, now)
		return true, err
	}
	if int(count) >= limit {
		return false, nil
	}
	_, err = pool.Exec(ctx,
		`UPDATE players SET chat_req_count=chat_req_count+1 WHERE id=$1`,
		playerID)
	return true, err
}

// UpdatePlayerChatEnabled sets the chat_enabled flag for a player.
func UpdatePlayerChatEnabled(ctx context.Context, pool *pgxpool.Pool, playerID uuid.UUID, enabled bool) error {
	_, err := pool.Exec(ctx,
		`UPDATE players SET chat_enabled=$2 WHERE id=$1`, playerID, enabled)
	return err
}
