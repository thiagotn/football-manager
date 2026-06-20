package unit_test

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/thiagotn/football-manager/football-api-go/internal/db"
	"github.com/thiagotn/football-manager/football-api-go/internal/handlers"
	"github.com/thiagotn/football-manager/football-api-go/internal/middleware"
	"github.com/thiagotn/football-manager/football-api-go/internal/services"
)

// ── PushService mock counting recipient IDs ─────────────────────────────────

type recordingPush struct {
	mu        sync.Mutex
	recipients []uuid.UUID
}

func (p *recordingPush) SendToPlayers(ctx context.Context, ids []uuid.UUID, n services.PushNotification) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.recipients = append(p.recipients, ids...)
	return nil
}

// ── GroupStore mock — focused on review_waitlist scenarios ──────────────────

type mockWaitlistStore struct {
	*mockGroupStoreForBusiness
	getGroupByIDFn               func(ctx context.Context, groupID uuid.UUID) (*db.Group, error)
	getWaitlistEntryByIDFn       func(ctx context.Context, entryID uuid.UUID) (*db.WaitlistEntry, error)
	getMatchByIDFn               func(ctx context.Context, matchID uuid.UUID) (*db.Match, error)
	countAttendancesFn           func(ctx context.Context, matchID uuid.UUID, status string) (int, error)
	getGroupMemberFnOverride     func(ctx context.Context, groupID, playerID uuid.UUID) (*db.GroupMember, error)
	updateWaitlistEntryStatusFn  func(ctx context.Context, entryID uuid.UUID, status string, reviewerID uuid.UUID) error
	getPendingWaitlistForMatchFn func(ctx context.Context, matchID uuid.UUID) ([]db.WaitlistEntry, error)
	setAttendanceFn              func(ctx context.Context, matchID, playerID uuid.UUID, status string) error
}

func (m *mockWaitlistStore) GetGroupByID(ctx context.Context, groupID uuid.UUID) (*db.Group, error) {
	if m.getGroupByIDFn != nil {
		return m.getGroupByIDFn(ctx, groupID)
	}
	return &db.Group{ID: groupID, Name: "Pelada", IsPublic: true}, nil
}
func (m *mockWaitlistStore) GetWaitlistEntryByID(ctx context.Context, entryID uuid.UUID) (*db.WaitlistEntry, error) {
	if m.getWaitlistEntryByIDFn != nil {
		return m.getWaitlistEntryByIDFn(ctx, entryID)
	}
	return nil, nil
}
func (m *mockWaitlistStore) GetMatchByID(ctx context.Context, matchID uuid.UUID) (*db.Match, error) {
	if m.getMatchByIDFn != nil {
		return m.getMatchByIDFn(ctx, matchID)
	}
	return nil, nil
}
func (m *mockWaitlistStore) CountAttendances(ctx context.Context, matchID uuid.UUID, status string) (int, error) {
	if m.countAttendancesFn != nil {
		return m.countAttendancesFn(ctx, matchID, status)
	}
	return 0, nil
}
func (m *mockWaitlistStore) GetGroupMember(ctx context.Context, groupID, playerID uuid.UUID) (*db.GroupMember, error) {
	if m.getGroupMemberFnOverride != nil {
		return m.getGroupMemberFnOverride(ctx, groupID, playerID)
	}
	return m.mockGroupStoreForBusiness.GetGroupMember(ctx, groupID, playerID)
}
func (m *mockWaitlistStore) UpdateWaitlistEntryStatus(ctx context.Context, entryID uuid.UUID, status string, reviewerID uuid.UUID) error {
	if m.updateWaitlistEntryStatusFn != nil {
		return m.updateWaitlistEntryStatusFn(ctx, entryID, status, reviewerID)
	}
	return nil
}
func (m *mockWaitlistStore) GetPendingWaitlistForMatch(ctx context.Context, matchID uuid.UUID) ([]db.WaitlistEntry, error) {
	if m.getPendingWaitlistForMatchFn != nil {
		return m.getPendingWaitlistForMatchFn(ctx, matchID)
	}
	return nil, nil
}
func (m *mockWaitlistStore) SetAttendance(ctx context.Context, matchID, playerID uuid.UUID, status string) error {
	if m.setAttendanceFn != nil {
		return m.setAttendanceFn(ctx, matchID, playerID, status)
	}
	return nil
}

// helper to wire a router around a mock store
func newWaitlistRouter(store handlers.GroupStore, push services.PushService) http.Handler {
	r := chi.NewRouter()
	h := handlers.NewGroupHandlerWithDeps(store, nil, push)
	r.Mount("/groups", h.Routes())
	return r
}

func TestReviewWaitlist_NonAdmin_Returns403(t *testing.T) {
	groupID := uuid.New()
	entryID := uuid.New()
	store := &mockWaitlistStore{
		mockGroupStoreForBusiness: &mockGroupStoreForBusiness{},
		getGroupMemberFnOverride: func(ctx context.Context, gID, pID uuid.UUID) (*db.GroupMember, error) {
			return &db.GroupMember{Role: db.GroupMemberRoleMember}, nil
		},
	}
	r := newWaitlistRouter(store, &recordingPush{})
	body := `{"action":"accept"}`
	caller := fakePlayer()
	w := sendRequestWithContext(r, "PATCH",
		fmt.Sprintf("/groups/%s/waitlist/%s", groupID, entryID),
		body,
		middleware.InjectPlayerForTest(context.Background(), caller),
	)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestReviewWaitlist_InvalidAction_Returns422(t *testing.T) {
	groupID := uuid.New()
	entryID := uuid.New()
	store := &mockWaitlistStore{
		mockGroupStoreForBusiness: &mockGroupStoreForBusiness{},
	}
	r := newWaitlistRouter(store, &recordingPush{})
	body := `{"action":"banana"}`
	caller := fakePlayer(asAdmin())
	w := sendRequestWithContext(r, "PATCH",
		fmt.Sprintf("/groups/%s/waitlist/%s", groupID, entryID),
		body,
		middleware.InjectPlayerForTest(context.Background(), caller),
	)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestReviewWaitlist_AcceptHappyPath_UpdatesStatusAndNotifies(t *testing.T) {
	groupID := uuid.New()
	matchID := uuid.New()
	entryID := uuid.New()
	candidateID := uuid.New()

	var statusUpdated string
	store := &mockWaitlistStore{
		mockGroupStoreForBusiness: &mockGroupStoreForBusiness{},
		getWaitlistEntryByIDFn: func(ctx context.Context, id uuid.UUID) (*db.WaitlistEntry, error) {
			return &db.WaitlistEntry{
				ID:         id,
				MatchID:    matchID,
				PlayerID:   candidateID,
				Status:     "pending",
				CreatedAt:  time.Now(),
				PlayerName: "Candidato",
			}, nil
		},
		getMatchByIDFn: func(ctx context.Context, mID uuid.UUID) (*db.Match, error) {
			return &db.Match{ID: mID, GroupID: groupID, Hash: "h"}, nil
		},
		getGroupMemberFnOverride: func(ctx context.Context, gID, pID uuid.UUID) (*db.GroupMember, error) {
			return nil, db.ErrNotFound // candidate is not yet a member
		},
		updateWaitlistEntryStatusFn: func(ctx context.Context, eID uuid.UUID, status string, reviewerID uuid.UUID) error {
			statusUpdated = status
			return nil
		},
	}
	push := &recordingPush{}
	r := newWaitlistRouter(store, push)

	caller := fakePlayer(asAdmin())
	body := `{"action":"accept"}`
	w := sendRequestWithContext(r, "PATCH",
		fmt.Sprintf("/groups/%s/waitlist/%s", groupID, entryID),
		body,
		middleware.InjectPlayerForTest(context.Background(), caller),
	)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "accepted", statusUpdated)
	// Candidate was notified
	assert.Contains(t, push.recipients, candidateID)
}

func TestReviewWaitlist_RejectHappyPath_UpdatesStatusAndNotifies(t *testing.T) {
	groupID := uuid.New()
	matchID := uuid.New()
	entryID := uuid.New()
	candidateID := uuid.New()

	var statusUpdated string
	store := &mockWaitlistStore{
		mockGroupStoreForBusiness: &mockGroupStoreForBusiness{},
		getWaitlistEntryByIDFn: func(ctx context.Context, id uuid.UUID) (*db.WaitlistEntry, error) {
			return &db.WaitlistEntry{
				ID:         id,
				MatchID:    matchID,
				PlayerID:   candidateID,
				Status:     "pending",
				CreatedAt:  time.Now(),
				PlayerName: "Candidato",
			}, nil
		},
		getMatchByIDFn: func(ctx context.Context, mID uuid.UUID) (*db.Match, error) {
			return &db.Match{ID: mID, GroupID: groupID, Hash: "h"}, nil
		},
		updateWaitlistEntryStatusFn: func(ctx context.Context, eID uuid.UUID, status string, reviewerID uuid.UUID) error {
			statusUpdated = status
			return nil
		},
	}
	push := &recordingPush{}
	r := newWaitlistRouter(store, push)

	caller := fakePlayer(asAdmin())
	body := `{"action":"reject"}`
	w := sendRequestWithContext(r, "PATCH",
		fmt.Sprintf("/groups/%s/waitlist/%s", groupID, entryID),
		body,
		middleware.InjectPlayerForTest(context.Background(), caller),
	)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "rejected", statusUpdated)
	assert.Contains(t, push.recipients, candidateID)
}

// ── NotifyGroupAdmins (pure helper) ─────────────────────────────────────────

type fakeAdminLister struct {
	ids []uuid.UUID
	err error
}

func (f *fakeAdminLister) GetGroupAdminIDs(ctx context.Context, groupID uuid.UUID) ([]uuid.UUID, error) {
	return f.ids, f.err
}

func TestNotifyGroupAdmins_FansOutToAll(t *testing.T) {
	a, b, c := uuid.New(), uuid.New(), uuid.New()
	push := &recordingPush{}
	n, err := services.NotifyGroupAdmins(context.Background(),
		&fakeAdminLister{ids: []uuid.UUID{a, b, c}},
		push, uuid.New(), nil,
		services.PushNotification{Title: "t", Body: "b"},
	)
	assert.NoError(t, err)
	assert.Equal(t, 3, n)
	assert.ElementsMatch(t, []uuid.UUID{a, b, c}, push.recipients)
}

func TestNotifyGroupAdmins_ExcludesActor(t *testing.T) {
	actor := uuid.New()
	other := uuid.New()
	push := &recordingPush{}
	n, err := services.NotifyGroupAdmins(context.Background(),
		&fakeAdminLister{ids: []uuid.UUID{actor, other}},
		push, uuid.New(), &actor,
		services.PushNotification{Title: "t", Body: "b"},
	)
	assert.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.Equal(t, []uuid.UUID{other}, push.recipients)
}

func TestNotifyGroupAdmins_NoopWhenOnlyActorIsAdmin(t *testing.T) {
	actor := uuid.New()
	push := &recordingPush{}
	n, err := services.NotifyGroupAdmins(context.Background(),
		&fakeAdminLister{ids: []uuid.UUID{actor}},
		push, uuid.New(), &actor,
		services.PushNotification{Title: "t", Body: "b"},
	)
	assert.NoError(t, err)
	assert.Equal(t, 0, n)
	assert.Empty(t, push.recipients)
}

func TestReviewWaitlist_AlreadyReviewed_Returns409(t *testing.T) {
	groupID := uuid.New()
	matchID := uuid.New()
	entryID := uuid.New()
	store := &mockWaitlistStore{
		mockGroupStoreForBusiness: &mockGroupStoreForBusiness{},
		getWaitlistEntryByIDFn: func(ctx context.Context, id uuid.UUID) (*db.WaitlistEntry, error) {
			return &db.WaitlistEntry{
				ID:        id,
				MatchID:   matchID,
				Status:    "accepted",
				CreatedAt: time.Now(),
			}, nil
		},
	}
	r := newWaitlistRouter(store, &recordingPush{})

	caller := fakePlayer(asAdmin())
	body := `{"action":"accept"}`
	w := sendRequestWithContext(r, "PATCH",
		fmt.Sprintf("/groups/%s/waitlist/%s", groupID, entryID),
		body,
		middleware.InjectPlayerForTest(context.Background(), caller),
	)
	assert.Equal(t, http.StatusConflict, w.Code)
}
