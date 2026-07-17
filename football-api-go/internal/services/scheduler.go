package services

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/robfig/cron/v3"
)

// jobTimeout limita cada execução de job (PRD 049 §2, ~120s). Sem ele, uma
// query pendurada num banco travado prende a goroutine para sempre; como o
// cron dispara execuções novas a cada tick, goroutines presas acumulariam
// segurando conexões do pool até esgotá-lo — derrubando também o HTTP.
const jobTimeout = 2 * time.Minute

// Scheduler wraps a cron.Cron with the rachao.app background jobs.
// Mirrors the APScheduler setup in v1 (football-api/app/services/recurrence.py).
//
// Schedule:
//   - Status sync: every hour at :30 (close past matches, mark today's as in_progress)
//   - Recurrence:  daily at 07:00 (create next match for groups with recurrence_enabled)
//   - Vote reminder: every 5 minutes
//
// Cron expressions are evaluated in America/Sao_Paulo, resolvido via tzdata
// embutido no binário (import _ "time/tzdata" em cmd/server) — a imagem de
// produção é scratch, sem /usr/share/zoneinfo, e a env TZ não pode ser a
// única garantia.
type Scheduler struct {
	cron *cron.Cron
	pool *pgxpool.Pool
	push PushService
}

func NewScheduler(pool *pgxpool.Pool) *Scheduler {
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		// Só acontece se o binário for compilado sem o embed time/tzdata E o
		// runtime não tiver zoneinfo. UTC deslocaria o job das 07h — logar alto.
		slog.Error("scheduler: America/Sao_Paulo indisponível, usando UTC", "error", err)
		loc = time.UTC
	}
	cronLogger := cron.PrintfLogger(slog.NewLogLogger(slog.Default().Handler(), slog.LevelWarn))
	return &Scheduler{
		cron: cron.New(
			cron.WithLocation(loc),
			// SkipIfStillRunning: se um tick anterior ainda roda (mesmo com o
			// timeout, durante o drain), o novo é pulado COM log — o oposto do
			// max_instances=1 silencioso do APScheduler (issue #11).
			// Recover: panic num job não derruba o processo.
			cron.WithChain(cron.SkipIfStillRunning(cronLogger), cron.Recover(cronLogger)),
		),
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
	slog.Info("scheduler started",
		"jobs", []string{"status_sync@:30", "recurrence@07:00", "vote_reminder@*/5"},
		"timezone", s.cron.Location().String(),
	)
	return nil
}

// Stop signals the cron loop to halt and waits for any in-flight job to finish.
func (s *Scheduler) Stop() {
	ctx := s.cron.Stop()
	<-ctx.Done()
	slog.Info("scheduler stopped")
}

// ExecuteJob roda um job com timeout e registra o heartbeat Prometheus:
// sucesso atualiza scheduler_job_last_success_timestamp_seconds (mesmo com
// result=0 — liveness, não negócio); erro/timeout incrementa
// scheduler_job_failures_total. Exportada para teste unitário direto.
func ExecuteJob(name string, timeout time.Duration, fn func(context.Context) (int, error)) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	n, err := fn(ctx)
	if err != nil {
		RecordJobFailure(name)
		slog.Error("scheduler: job failed", "job", name, "error", err)
		return n, err
	}
	RecordJobSuccess(name)
	slog.Info("scheduler: job completed", "job", name, "result", n)
	return n, nil
}

func (s *Scheduler) runStatusSync() {
	_, _ = ExecuteJob(JobStatusSync, jobTimeout, func(ctx context.Context) (int, error) {
		return RunStatusSyncJob(ctx, s.pool)
	})
}

func (s *Scheduler) runRecurrence() {
	_, _ = ExecuteJob(JobRecurrence, jobTimeout, func(ctx context.Context) (int, error) {
		return RunRecurrence(ctx, s.pool)
	})
}

func (s *Scheduler) runVoteReminder() {
	_, _ = ExecuteJob(JobVoteReminder, jobTimeout, func(ctx context.Context) (int, error) {
		return RunVoteReminderJob(ctx, s.pool, s.push)
	})
}
