package unit_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/thiagotn/football-manager/football-api-go/internal/services"
)

// ── store mock ──────────────────────────────────────────────────────────────

type fakeReminderStore struct {
	candidates []services.VoteReminderCandidate
	pending    map[uuid.UUID][]uuid.UUID
	marked     map[uuid.UUID]time.Time
	mu         sync.Mutex
}

func (s *fakeReminderStore) GetReminderCandidates(ctx context.Context) ([]services.VoteReminderCandidate, error) {
	return s.candidates, nil
}
func (s *fakeReminderStore) GetConfirmedPendingVoters(ctx context.Context, matchID uuid.UUID) ([]uuid.UUID, error) {
	return s.pending[matchID], nil
}
func (s *fakeReminderStore) MarkReminderSent(ctx context.Context, matchID uuid.UUID, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.marked == nil {
		s.marked = map[uuid.UUID]time.Time{}
	}
	s.marked[matchID] = at
	return nil
}

// ── helpers ─────────────────────────────────────────────────────────────────

// makeCandidate builds a candidate whose voting window closes in `closesIn`
// from "now" (UTC). Uses BRT internally to match VotingWindow's semantics.
func makeCandidate(closesIn time.Duration) services.VoteReminderCandidate {
	now := time.Now().UTC()
	endUTC := now.Add(closesIn).Add(-20 * time.Minute).Add(-24 * time.Hour)
	endBRT := endUTC.Add(-3 * time.Hour)
	endTimeStr := endBRT.Format("15:04:05")
	return services.VoteReminderCandidate{
		ID:                   uuid.New(),
		Hash:                 fmt.Sprintf("hash%010d", time.Now().UnixNano()%1e10),
		Number:               42,
		GroupName:            "Pelada",
		MatchDate:            endBRT.Format("2006-01-02"),
		StartTime:            endTimeStr,
		EndTime:              &endTimeStr,
		VoteOpenDelayMinutes: 20,
		VoteDurationHours:    24,
	}
}

// ── tests ───────────────────────────────────────────────────────────────────

func TestVoteReminder_NoCandidates_ReturnsZero(t *testing.T) {
	store := &fakeReminderStore{}
	push := &recordingPush{}
	n, err := services.RunVoteReminderWithStore(context.Background(), store, push)
	assert.NoError(t, err)
	assert.Equal(t, 0, n)
	assert.Empty(t, push.recipients)
}

func TestVoteReminder_WindowFarFromClosing_Skipped(t *testing.T) {
	c := makeCandidate(2 * time.Hour)
	store := &fakeReminderStore{
		candidates: []services.VoteReminderCandidate{c},
		pending:    map[uuid.UUID][]uuid.UUID{c.ID: {uuid.New()}},
	}
	push := &recordingPush{}

	n, err := services.RunVoteReminderWithStore(context.Background(), store, push)

	assert.NoError(t, err)
	assert.Equal(t, 0, n)
	assert.Empty(t, push.recipients)
	_, marked := store.marked[c.ID]
	assert.False(t, marked, "não deve marcar quando ainda está longe de fechar")
}

func TestVoteReminder_WindowAlreadyClosed_Skipped(t *testing.T) {
	c := makeCandidate(-5 * time.Minute)
	store := &fakeReminderStore{
		candidates: []services.VoteReminderCandidate{c},
		pending:    map[uuid.UUID][]uuid.UUID{c.ID: {uuid.New()}},
	}
	push := &recordingPush{}

	n, err := services.RunVoteReminderWithStore(context.Background(), store, push)

	assert.NoError(t, err)
	assert.Equal(t, 0, n)
	assert.Empty(t, push.recipients)
}

func TestVoteReminder_NotOpenYet_Skipped(t *testing.T) {
	// closes_at em 24h+25min → opens_at ainda no futuro
	c := makeCandidate(24*time.Hour + 25*time.Minute)
	store := &fakeReminderStore{
		candidates: []services.VoteReminderCandidate{c},
	}
	push := &recordingPush{}

	n, err := services.RunVoteReminderWithStore(context.Background(), store, push)

	assert.NoError(t, err)
	assert.Equal(t, 0, n)
}

func TestVoteReminder_WithinWindow_AndOnePending_SendsPush(t *testing.T) {
	c := makeCandidate(20 * time.Minute)
	pending := uuid.New()
	store := &fakeReminderStore{
		candidates: []services.VoteReminderCandidate{c},
		pending:    map[uuid.UUID][]uuid.UUID{c.ID: {pending}},
	}
	push := &recordingPush{}

	n, err := services.RunVoteReminderWithStore(context.Background(), store, push)

	assert.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.Equal(t, []uuid.UUID{pending}, push.recipients)
	_, marked := store.marked[c.ID]
	assert.True(t, marked)
}

func TestVoteReminder_NoPending_MarksWithoutPush(t *testing.T) {
	c := makeCandidate(20 * time.Minute)
	store := &fakeReminderStore{
		candidates: []services.VoteReminderCandidate{c},
		pending:    map[uuid.UUID][]uuid.UUID{c.ID: {}},
	}
	push := &recordingPush{}

	n, err := services.RunVoteReminderWithStore(context.Background(), store, push)

	assert.NoError(t, err)
	assert.Equal(t, 0, n)
	assert.Empty(t, push.recipients)
	_, marked := store.marked[c.ID]
	assert.True(t, marked, "deve marcar pra não reavaliar mesmo sem pendentes")
}
