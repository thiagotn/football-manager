package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/thiagotn/football-manager/football-api-go/internal/db"
)

const (
	maxAttempts         = 3
	maxDurationSeconds  = 65
	stalePendingMaxAge  = 24 * time.Hour
	defaultPollInterval = 5 * time.Second
	cleanupInterval     = time.Hour
	// maxJobDuration bounds a single pipeline run. Jobs run on a context
	// detached from Run's ctx so an in-flight ffmpeg survives shutdown
	// (SIGTERM stops claiming new jobs; the current one finishes).
	maxJobDuration = 10 * time.Minute
)

var (
	videosProcessed = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "rachao_worker_videos_processed_total",
		Help: "Vídeos processados pelo worker, por resultado final.",
	}, []string{"result"})
	transcodeSeconds = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "rachao_worker_transcode_seconds",
		Help:    "Duração do pipeline completo por vídeo (download+transcode+upload).",
		Buckets: []float64{5, 15, 30, 60, 120, 300, 600},
	})
)

// Store is the queue interface the worker drives (mockable in tests).
type Store interface {
	ClaimNextVideoJob(ctx context.Context) (*db.MatchVideo, error)
	MarkVideoReady(ctx context.Context, id uuid.UUID, videoURL, posterURL string, durationSeconds float64, sizeBytes int64) error
	MarkVideoFailed(ctx context.Context, id uuid.UUID, errMsg string) error
	RequeueVideoJob(ctx context.Context, id uuid.UUID) error
	DeleteStalePendingVideos(ctx context.Context, olderThan time.Duration) ([]db.MatchVideo, error)
}

// Storage is the R2 surface the worker needs (implemented by services.StorageService).
type Storage interface {
	DownloadFile(ctx context.Context, key, localPath string) error
	UploadFile(ctx context.Context, key, contentType, localPath string) (string, error)
	DeleteObject(ctx context.Context, key string) error
}

type pgStore struct{ pool *pgxpool.Pool }

func (s *pgStore) ClaimNextVideoJob(ctx context.Context) (*db.MatchVideo, error) {
	return db.ClaimNextVideoJob(ctx, s.pool)
}
func (s *pgStore) MarkVideoReady(ctx context.Context, id uuid.UUID, videoURL, posterURL string, durationSeconds float64, sizeBytes int64) error {
	return db.MarkVideoReady(ctx, s.pool, id, videoURL, posterURL, durationSeconds, sizeBytes)
}
func (s *pgStore) MarkVideoFailed(ctx context.Context, id uuid.UUID, errMsg string) error {
	return db.MarkVideoFailed(ctx, s.pool, id, errMsg)
}
func (s *pgStore) RequeueVideoJob(ctx context.Context, id uuid.UUID) error {
	return db.RequeueVideoJob(ctx, s.pool, id)
}
func (s *pgStore) DeleteStalePendingVideos(ctx context.Context, olderThan time.Duration) ([]db.MatchVideo, error) {
	return db.DeleteStalePendingVideos(ctx, s.pool, olderThan)
}

// permanentError marks pipeline failures that must not be retried (e.g. video
// too long): the job goes straight to failed regardless of attempts.
type permanentError struct{ msg string }

func (e *permanentError) Error() string { return e.msg }

// Worker polls the match_videos queue and runs the transcode pipeline.
type Worker struct {
	Store        Store
	Storage      Storage
	Transcoder   Transcoder
	WorkDir      string
	PollInterval time.Duration
}

// New builds a production worker on top of a pgx pool and the R2 storage service.
func New(pool *pgxpool.Pool, storage Storage, workDir string) *Worker {
	return &Worker{
		Store:        &pgStore{pool: pool},
		Storage:      storage,
		Transcoder:   &FFmpegTranscoder{},
		WorkDir:      workDir,
		PollInterval: defaultPollInterval,
	}
}

// Run blocks until ctx is cancelled, processing jobs as they appear.
// The in-flight job is allowed to finish (pair with terminationGracePeriodSeconds).
func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.PollInterval)
	defer ticker.Stop()
	cleanup := time.NewTicker(cleanupInterval)
	defer cleanup.Stop()

	slog.Info("worker started", "poll_interval", w.PollInterval.String())
	runJob := func(fn func(context.Context)) {
		jobCtx, cancel := context.WithTimeout(context.Background(), maxJobDuration)
		defer cancel()
		fn(jobCtx)
	}
	runJob(w.CleanupStalePending)

	for {
		select {
		case <-ctx.Done():
			slog.Info("worker stopping")
			return
		case <-cleanup.C:
			runJob(w.CleanupStalePending)
		case <-ticker.C:
			// Drain the queue before going back to sleep.
			for ctx.Err() == nil {
				more := false
				runJob(func(jobCtx context.Context) { more = w.ProcessNext(jobCtx) })
				if !more {
					break
				}
			}
		}
	}
}

// ProcessNext claims and processes one job. Returns false when the queue is empty.
func (w *Worker) ProcessNext(ctx context.Context) bool {
	video, err := w.Store.ClaimNextVideoJob(ctx)
	if err != nil {
		if !errors.Is(err, db.ErrNotFound) {
			slog.Error("claim job failed", "error", err)
		}
		return false
	}

	log := slog.With("video_id", video.ID, "match_id", video.MatchID, "attempt", video.Attempts)
	log.Info("processing video")
	start := time.Now()

	err = w.process(ctx, video)
	switch {
	case err == nil:
		videosProcessed.WithLabelValues("ready").Inc()
		log.Info("video ready", "took", time.Since(start).String())
	default:
		var perm *permanentError
		if errors.As(err, &perm) || video.Attempts >= maxAttempts {
			videosProcessed.WithLabelValues("failed").Inc()
			log.Error("video failed permanently", "error", err)
			if markErr := w.Store.MarkVideoFailed(ctx, video.ID, err.Error()); markErr != nil {
				log.Error("mark failed errored", "error", markErr)
			}
			// O original não serve mais — best-effort cleanup.
			_ = w.Storage.DeleteObject(ctx, video.OriginalKey)
		} else {
			videosProcessed.WithLabelValues("requeued").Inc()
			log.Warn("video requeued for retry", "error", err)
			if reqErr := w.Store.RequeueVideoJob(ctx, video.ID); reqErr != nil {
				log.Error("requeue errored", "error", reqErr)
			}
		}
	}
	transcodeSeconds.Observe(time.Since(start).Seconds())
	return true
}

// process runs the full pipeline for one claimed video.
func (w *Worker) process(ctx context.Context, video *db.MatchVideo) error {
	dir := filepath.Join(w.WorkDir, video.ID.String())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("workdir: %w", err)
	}
	defer os.RemoveAll(dir)

	inPath := filepath.Join(dir, "in.mp4")
	outPath := filepath.Join(dir, "out.mp4")
	posterPath := filepath.Join(dir, "poster.webp")

	if err := w.Storage.DownloadFile(ctx, video.OriginalKey, inPath); err != nil {
		return fmt.Errorf("download original: %w", err)
	}

	probe, err := w.Transcoder.Probe(ctx, inPath)
	if err != nil {
		return &permanentError{msg: fmt.Sprintf("invalid video file: %v", err)}
	}
	if !probe.HasVideoStream {
		return &permanentError{msg: "file has no video stream"}
	}
	if probe.DurationSeconds > maxDurationSeconds {
		return &permanentError{msg: "video too long (max 60s)"}
	}

	if err := w.Transcoder.Transcode(ctx, inPath, outPath); err != nil {
		return err
	}
	if err := w.Transcoder.Poster(ctx, outPath, posterPath); err != nil {
		return err
	}

	outInfo, err := os.Stat(outPath)
	if err != nil {
		return fmt.Errorf("stat output: %w", err)
	}

	keyPrefix := "videos/" + video.MatchID.String() + "/" + video.ID.String()
	videoURL, err := w.Storage.UploadFile(ctx, keyPrefix+".mp4", "video/mp4", outPath)
	if err != nil {
		return fmt.Errorf("upload video: %w", err)
	}
	posterURL, err := w.Storage.UploadFile(ctx, keyPrefix+".webp", "image/webp", posterPath)
	if err != nil {
		return fmt.Errorf("upload poster: %w", err)
	}

	if err := w.Store.MarkVideoReady(ctx, video.ID, videoURL, posterURL, probe.DurationSeconds, outInfo.Size()); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			// Vídeo excluído durante o processamento: remove os artefatos órfãos.
			_ = w.Storage.DeleteObject(ctx, keyPrefix+".mp4")
			_ = w.Storage.DeleteObject(ctx, keyPrefix+".webp")
			_ = w.Storage.DeleteObject(ctx, video.OriginalKey)
			return nil
		}
		return fmt.Errorf("mark ready: %w", err)
	}

	// Original cumpriu o papel — remove do bucket (best-effort).
	_ = w.Storage.DeleteObject(ctx, video.OriginalKey)
	return nil
}

// CleanupStalePending removes abandoned pending rows (>24h without confirm)
// and their possible R2 originals, freeing per-match slots.
func (w *Worker) CleanupStalePending(ctx context.Context) {
	stale, err := w.Store.DeleteStalePendingVideos(ctx, stalePendingMaxAge)
	if err != nil {
		slog.Error("stale pending cleanup failed", "error", err)
		return
	}
	for _, v := range stale {
		_ = w.Storage.DeleteObject(ctx, v.OriginalKey)
	}
	if len(stale) > 0 {
		slog.Info("stale pending uploads cleaned", "count", len(stale))
	}
}
