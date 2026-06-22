package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// reminderLeadTime mirrors the Python REMINDER_LEAD_TIME (30min): only matches
// whose voting window closes within this window will receive the reminder.
const reminderLeadTime = 30 * time.Minute

// VoteReminderStore abstracts the queries used by RunVoteReminderJob, making
// the job testable without a real Postgres.
type VoteReminderStore interface {
	GetReminderCandidates(ctx context.Context) ([]VoteReminderCandidate, error)
	GetConfirmedPendingVoters(ctx context.Context, matchID uuid.UUID) ([]uuid.UUID, error)
	MarkReminderSent(ctx context.Context, matchID uuid.UUID, at time.Time) error
}

// VoteReminderCandidate is the projection needed to compute the voting window
// and decide whether to send the push.
type VoteReminderCandidate struct {
	ID                   uuid.UUID
	Hash                 string
	Number               int
	GroupName            string
	MatchDate            string
	EndTime              *string
	StartTime            string
	VoteOpenDelayMinutes int
	VoteDurationHours    int
}

// pgVoteReminderStore is the production implementation.
type pgVoteReminderStore struct{ pool *pgxpool.Pool }

func (s *pgVoteReminderStore) GetReminderCandidates(ctx context.Context) ([]VoteReminderCandidate, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT m.id, m.hash, m.number,
		       COALESCE(g.name, '') AS group_name,
		       m.match_date::TEXT, m.end_time::TEXT, m.start_time::TEXT,
		       m.vote_open_delay_minutes, m.vote_duration_hours
		FROM matches m
		JOIN groups g ON g.id = m.group_id
		WHERE m.status = 'closed'
		  AND g.voting_enabled = true
		  AND m.vote_reminder_sent_at IS NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []VoteReminderCandidate
	for rows.Next() {
		var c VoteReminderCandidate
		if err := rows.Scan(&c.ID, &c.Hash, &c.Number, &c.GroupName,
			&c.MatchDate, &c.EndTime, &c.StartTime,
			&c.VoteOpenDelayMinutes, &c.VoteDurationHours); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *pgVoteReminderStore) GetConfirmedPendingVoters(ctx context.Context, matchID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT a.player_id FROM attendances a
		WHERE a.match_id = $1
		  AND a.status = 'confirmed'
		  AND NOT EXISTS (
		    SELECT 1 FROM match_votes mv
		    WHERE mv.match_id = a.match_id AND mv.voter_id = a.player_id
		  )`, matchID)
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

func (s *pgVoteReminderStore) MarkReminderSent(ctx context.Context, matchID uuid.UUID, at time.Time) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE matches SET vote_reminder_sent_at = $2 WHERE id = $1`,
		matchID, at)
	return err
}

// RunVoteReminderJob is the cron entrypoint. Returns the number of matches
// where at least one reminder was dispatched.
func RunVoteReminderJob(ctx context.Context, pool *pgxpool.Pool, push PushService) (int, error) {
	return RunVoteReminderWithStore(ctx, &pgVoteReminderStore{pool: pool}, push)
}

// RunVoteReminderWithStore is the testable variant.
func RunVoteReminderWithStore(ctx context.Context, store VoteReminderStore, push PushService) (int, error) {
	candidates, err := store.GetReminderCandidates(ctx)
	if err != nil {
		return 0, err
	}
	now := time.Now().UTC()
	notified := 0

	for _, c := range candidates {
		opensAt, closesAt, ok := VotingWindow(VotingInput{
			MatchDate:            c.MatchDate,
			StartTime:            c.StartTime,
			EndTime:              c.EndTime,
			VoteOpenDelayMinutes: c.VoteOpenDelayMinutes,
			VoteDurationHours:    c.VoteDurationHours,
		})
		if !ok {
			continue
		}
		if now.Before(opensAt) {
			continue // not_open yet
		}
		if !now.Before(closesAt) {
			continue // already closed
		}
		if closesAt.Sub(now) > reminderLeadTime {
			continue // still > 30min away
		}

		pending, err := store.GetConfirmedPendingVoters(ctx, c.ID)
		if err != nil {
			slog.Warn("vote_reminder: pending voters lookup failed", "match_id", c.ID, "error", err)
			continue
		}
		if len(pending) == 0 {
			// Mark anyway so we don't reevaluate forever.
			_ = store.MarkReminderSent(ctx, c.ID, now)
			continue
		}

		title := fmt.Sprintf("🗳️ 30 min para fechar a votação — %s", c.GroupName)
		body := fmt.Sprintf("Vote agora no Rachão #%d!", c.Number)
		url := "https://rachao.app/match/" + c.Hash
		if err := push.SendToPlayers(ctx, pending, PushNotification{Title: title, Body: body, URL: url}); err != nil {
			slog.Warn("vote_reminder: push fanout failed", "match_id", c.ID, "error", err)
		}

		if err := store.MarkReminderSent(ctx, c.ID, now); err != nil {
			slog.Warn("vote_reminder: mark sent failed", "match_id", c.ID, "error", err)
		}
		notified++
	}
	return notified, nil
}
