package unit_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/thiagotn/football-manager/football-api-go/internal/db"
	"github.com/thiagotn/football-manager/football-api-go/internal/worker"
)

// ── ffprobe parse ────────────────────────────────────────────────────────────

func TestParseProbeOutput_ValidVideo(t *testing.T) {
	out := `{
		"streams": [
			{"codec_type": "video"},
			{"codec_type": "audio"}
		],
		"format": {"duration": "42.517000"}
	}`
	res, err := worker.ParseProbeOutput([]byte(out))
	assert.NoError(t, err)
	assert.True(t, res.HasVideoStream)
	assert.InDelta(t, 42.517, res.DurationSeconds, 0.001)
}

func TestParseProbeOutput_NoVideoStream(t *testing.T) {
	out := `{"streams": [{"codec_type": "audio"}], "format": {"duration": "10"}}`
	res, err := worker.ParseProbeOutput([]byte(out))
	assert.NoError(t, err)
	assert.False(t, res.HasVideoStream)
}

func TestParseProbeOutput_InvalidJSON(t *testing.T) {
	_, err := worker.ParseProbeOutput([]byte("not json"))
	assert.Error(t, err)
}

func TestParseProbeOutput_BadDuration(t *testing.T) {
	out := `{"streams": [{"codec_type": "video"}], "format": {"duration": "N/A-ish"}}`
	_, err := worker.ParseProbeOutput([]byte(out))
	assert.Error(t, err)
}

// ── Fakes ────────────────────────────────────────────────────────────────────

type fakeWorkerStore struct {
	jobs      []*db.MatchVideo
	ready     map[uuid.UUID]string // id → videoURL
	failed    map[uuid.UUID]string // id → errMsg
	requeued  map[uuid.UUID]int
	readyErr  error
	stale     []db.MatchVideo
	staleRuns int
}

func newFakeWorkerStore() *fakeWorkerStore {
	return &fakeWorkerStore{
		ready:    map[uuid.UUID]string{},
		failed:   map[uuid.UUID]string{},
		requeued: map[uuid.UUID]int{},
	}
}

func (s *fakeWorkerStore) ClaimNextVideoJob(ctx context.Context) (*db.MatchVideo, error) {
	if len(s.jobs) == 0 {
		return nil, db.ErrNotFound
	}
	j := s.jobs[0]
	s.jobs = s.jobs[1:]
	j.Attempts++
	j.Status = db.VideoStatusProcessing
	return j, nil
}
func (s *fakeWorkerStore) MarkVideoReady(ctx context.Context, id uuid.UUID, videoURL, posterURL string, durationSeconds float64, sizeBytes int64) error {
	if s.readyErr != nil {
		return s.readyErr
	}
	s.ready[id] = videoURL
	return nil
}
func (s *fakeWorkerStore) MarkVideoFailed(ctx context.Context, id uuid.UUID, errMsg string) error {
	s.failed[id] = errMsg
	return nil
}
func (s *fakeWorkerStore) RequeueVideoJob(ctx context.Context, id uuid.UUID) error {
	s.requeued[id]++
	return nil
}
func (s *fakeWorkerStore) DeleteStalePendingVideos(ctx context.Context, olderThan time.Duration) ([]db.MatchVideo, error) {
	s.staleRuns++
	return s.stale, nil
}

type fakeWorkerStorage struct {
	downloaded []string
	uploaded   map[string]string // key → contentType
	deleted    []string
	downErr    error
}

func newFakeWorkerStorage() *fakeWorkerStorage {
	return &fakeWorkerStorage{uploaded: map[string]string{}}
}

func (s *fakeWorkerStorage) DownloadFile(ctx context.Context, key, localPath string) error {
	if s.downErr != nil {
		return s.downErr
	}
	s.downloaded = append(s.downloaded, key)
	return os.WriteFile(localPath, []byte("fake-input"), 0o644)
}
func (s *fakeWorkerStorage) UploadFile(ctx context.Context, key, contentType, localPath string) (string, error) {
	s.uploaded[key] = contentType
	return "https://cdn.rachao.app/" + key, nil
}
func (s *fakeWorkerStorage) DeleteObject(ctx context.Context, key string) error {
	s.deleted = append(s.deleted, key)
	return nil
}

type fakeTranscoder struct {
	probe        worker.ProbeResult
	probeErr     error
	transcodeErr error
	posterErr    error
}

func (f *fakeTranscoder) Probe(ctx context.Context, path string) (worker.ProbeResult, error) {
	return f.probe, f.probeErr
}
func (f *fakeTranscoder) Transcode(ctx context.Context, inPath, outPath string) error {
	if f.transcodeErr != nil {
		return f.transcodeErr
	}
	return os.WriteFile(outPath, []byte("fake-h264"), 0o644)
}
func (f *fakeTranscoder) Poster(ctx context.Context, videoPath, posterPath string) error {
	if f.posterErr != nil {
		return f.posterErr
	}
	return os.WriteFile(posterPath, []byte("fake-webp"), 0o644)
}

func testJob(attempts int) *db.MatchVideo {
	return &db.MatchVideo{
		ID:          uuid.New(),
		MatchID:     uuid.New(),
		UploadedBy:  uuid.New(),
		Status:      db.VideoStatusUploaded,
		OriginalKey: "videos/original/m/v.mp4",
		Attempts:    attempts,
	}
}

func newTestWorker(t *testing.T, store worker.Store, storage worker.Storage, tc worker.Transcoder) *worker.Worker {
	t.Helper()
	return &worker.Worker{
		Store:        store,
		Storage:      storage,
		Transcoder:   tc,
		WorkDir:      filepath.Join(t.TempDir(), "work"),
		PollInterval: time.Millisecond,
	}
}

// ── State machine ────────────────────────────────────────────────────────────

func TestWorker_HappyPathReady(t *testing.T) {
	store := newFakeWorkerStore()
	job := testJob(0)
	store.jobs = []*db.MatchVideo{job}
	storage := newFakeWorkerStorage()
	tc := &fakeTranscoder{probe: worker.ProbeResult{HasVideoStream: true, DurationSeconds: 42}}
	w := newTestWorker(t, store, storage, tc)

	assert.True(t, w.ProcessNext(context.Background()))

	assert.Contains(t, store.ready, job.ID)
	assert.Empty(t, store.failed)
	keyPrefix := "videos/" + job.MatchID.String() + "/" + job.ID.String()
	assert.Equal(t, "video/mp4", storage.uploaded[keyPrefix+".mp4"])
	assert.Equal(t, "image/webp", storage.uploaded[keyPrefix+".webp"])
	// original é apagado após sucesso
	assert.Contains(t, storage.deleted, job.OriginalKey)
}

func TestWorker_EmptyQueueReturnsFalse(t *testing.T) {
	w := newTestWorker(t, newFakeWorkerStore(), newFakeWorkerStorage(), &fakeTranscoder{})
	assert.False(t, w.ProcessNext(context.Background()))
}

func TestWorker_TooLongFailsPermanently(t *testing.T) {
	store := newFakeWorkerStore()
	job := testJob(0) // primeira tentativa — mas erro permanente não deve reencaminhar
	store.jobs = []*db.MatchVideo{job}
	storage := newFakeWorkerStorage()
	tc := &fakeTranscoder{probe: worker.ProbeResult{HasVideoStream: true, DurationSeconds: 90}}
	w := newTestWorker(t, store, storage, tc)

	w.ProcessNext(context.Background())

	assert.Contains(t, store.failed, job.ID)
	assert.Contains(t, store.failed[job.ID], "too long")
	assert.Empty(t, store.requeued)
	assert.Empty(t, storage.uploaded)
	// original apagado (não serve mais)
	assert.Contains(t, storage.deleted, job.OriginalKey)
}

func TestWorker_NoVideoStreamFailsPermanently(t *testing.T) {
	store := newFakeWorkerStore()
	job := testJob(0)
	store.jobs = []*db.MatchVideo{job}
	tc := &fakeTranscoder{probe: worker.ProbeResult{HasVideoStream: false, DurationSeconds: 10}}
	w := newTestWorker(t, store, newFakeWorkerStorage(), tc)

	w.ProcessNext(context.Background())
	assert.Contains(t, store.failed[job.ID], "no video stream")
}

func TestWorker_TransientErrorRequeues(t *testing.T) {
	store := newFakeWorkerStore()
	job := testJob(0) // após o claim, attempts = 1 < 3 → requeue
	store.jobs = []*db.MatchVideo{job}
	tc := &fakeTranscoder{probe: worker.ProbeResult{HasVideoStream: true, DurationSeconds: 30}, transcodeErr: errors.New("boom")}
	w := newTestWorker(t, store, newFakeWorkerStorage(), tc)

	w.ProcessNext(context.Background())

	assert.Equal(t, 1, store.requeued[job.ID])
	assert.Empty(t, store.failed)
}

func TestWorker_TransientErrorFailsAfterMaxAttempts(t *testing.T) {
	store := newFakeWorkerStore()
	job := testJob(2) // claim → attempts = 3 == máximo → failed
	store.jobs = []*db.MatchVideo{job}
	tc := &fakeTranscoder{probe: worker.ProbeResult{HasVideoStream: true, DurationSeconds: 30}, transcodeErr: errors.New("boom")}
	w := newTestWorker(t, store, newFakeWorkerStorage(), tc)

	w.ProcessNext(context.Background())

	assert.Contains(t, store.failed, job.ID)
	assert.Empty(t, store.requeued)
}

func TestWorker_DeletedMidProcessingCleansArtifacts(t *testing.T) {
	store := newFakeWorkerStore()
	job := testJob(0)
	store.jobs = []*db.MatchVideo{job}
	store.readyErr = db.ErrNotFound // row sumiu durante o processamento
	storage := newFakeWorkerStorage()
	tc := &fakeTranscoder{probe: worker.ProbeResult{HasVideoStream: true, DurationSeconds: 30}}
	w := newTestWorker(t, store, storage, tc)

	w.ProcessNext(context.Background())

	keyPrefix := "videos/" + job.MatchID.String() + "/" + job.ID.String()
	assert.Contains(t, storage.deleted, keyPrefix+".mp4")
	assert.Contains(t, storage.deleted, keyPrefix+".webp")
	assert.Contains(t, storage.deleted, job.OriginalKey)
	assert.Empty(t, store.failed)
	assert.Empty(t, store.requeued)
}

func TestWorker_CleanupStalePendingDeletesOriginals(t *testing.T) {
	store := newFakeWorkerStore()
	stale := db.MatchVideo{ID: uuid.New(), OriginalKey: "videos/original/m/stale.mp4"}
	store.stale = []db.MatchVideo{stale}
	storage := newFakeWorkerStorage()
	w := newTestWorker(t, store, storage, &fakeTranscoder{})

	w.CleanupStalePending(context.Background())

	assert.Equal(t, 1, store.staleRuns)
	assert.Contains(t, storage.deleted, stale.OriginalKey)
}
