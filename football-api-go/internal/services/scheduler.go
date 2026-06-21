package services

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/robfig/cron/v3"
)

// Scheduler wraps a cron.Cron with the rachao.app background jobs.
// Mirrors the APScheduler setup in v1 (football-api/app/services/recurrence.py).
//
// Schedule:
//   - Status sync: every hour at :30 (close past matches, mark today's as in_progress)
//   - Recurrence:  daily at 07:00 (create next match for groups with recurrence_enabled)
//
// Both cron expressions are evaluated in the local timezone of the process.
// In production we run with TZ=America/Sao_Paulo to match v1 behaviour.
type Scheduler struct {
	cron *cron.Cron
	pool *pgxpool.Pool
	push PushService
}

func NewScheduler(pool *pgxpool.Pool) *Scheduler {
	return &Scheduler{
		cron: cron.New(),
		pool: pool,
		push: NewPushService(pool),
	}
}

// Start registers the jobs and begins the cron loop in a background goroutine.
// Safe to call exactly once per Scheduler.
func (s *Scheduler) Start() error {
	if _, err := s.cron.AddFunc("30 * * * *", s.runStatusSync); err != nil {
		return err
	}
	if _, err := s.cron.AddFunc("0 7 * * *", s.runRecurrence); err != nil {
		return err
	}
	if _, err := s.cron.AddFunc("*/5 * * * *", s.runVoteReminder); err != nil {
		return err
	}
	s.cron.Start()
	slog.Info("scheduler started", "jobs", []string{"status_sync@:30", "recurrence@07:00", "vote_reminder@*/5"})
	return nil
}

// Stop signals the cron loop to halt and waits for any in-flight job to finish.
func (s *Scheduler) Stop() {
	ctx := s.cron.Stop()
	<-ctx.Done()
	slog.Info("scheduler stopped")
}

func (s *Scheduler) runStatusSync() {
	ctx := context.Background()
	closed, err := RunStatusSyncJob(ctx, s.pool)
	if err != nil {
		slog.Error("scheduler: status sync failed", "error", err)
		return
	}
	slog.Info("scheduler: status sync completed", "closed", closed)
}

func (s *Scheduler) runRecurrence() {
	ctx := context.Background()
	created, err := RunRecurrence(ctx, s.pool)
	if err != nil {
		slog.Error("scheduler: recurrence failed", "error", err)
		return
	}
	slog.Info("scheduler: recurrence completed", "matches_created", created)
}

func (s *Scheduler) runVoteReminder() {
	ctx := context.Background()
	notified, err := RunVoteReminderJob(ctx, s.pool, s.push)
	if err != nil {
		slog.Error("scheduler: vote reminder failed", "error", err)
		return
	}
	slog.Info("scheduler: vote reminder completed", "matches_notified", notified)
}
