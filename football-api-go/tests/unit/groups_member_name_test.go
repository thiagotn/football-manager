package unit_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/thiagotn/football-manager/football-api-go/internal/db"
	"github.com/thiagotn/football-manager/football-api-go/internal/handlers"
	"github.com/thiagotn/football-manager/football-api-go/internal/middleware"
)

// ── PATCH /groups/{id}/members/{playerID} — nome de cadastro pendente ────────

// memberNameRouter mounts the groups routes with the given store.
func memberNameRouter(store handlers.GroupStore) http.Handler {
	r := chi.NewRouter()
	h := &handlers.GroupHandler{Store: store}
	r.Mount("/groups", h.Routes())
	return r
}

func TestUpdateMember_NamePendingRegistration(t *testing.T) {
	groupID := uuid.New()
	targetID := uuid.New()
	var saved string

	store := &mockGroupStoreForBusiness{
		getGroupMemberFn: func(ctx context.Context, gID, pID uuid.UUID) (*db.GroupMember, error) {
			return &db.GroupMember{Role: db.GroupMemberRoleAdmin}, nil
		},
		getPlayerByIDFn: func(ctx context.Context, pID uuid.UUID) (*db.Player, error) {
			return &db.Player{ID: pID, Name: "Jogadr", PendingRegistration: true}, nil
		},
		updatePlayerNameFn: func(ctx context.Context, pID uuid.UUID, name string) error {
			saved = name
			return nil
		},
	}

	r := memberNameRouter(store)
	ctx := middleware.InjectPlayerForTest(context.Background(), fakePlayer())
	w := sendRequestWithContext(r, http.MethodPatch,
		fmt.Sprintf("/groups/%s/members/%s", groupID, targetID),
		`{"name":"  Jogador Certo  "}`, ctx)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "Jogador Certo", saved)
}

func TestUpdateMember_NameAlreadyClaimed(t *testing.T) {
	groupID := uuid.New()
	targetID := uuid.New()
	called := false

	store := &mockGroupStoreForBusiness{
		getGroupMemberFn: func(ctx context.Context, gID, pID uuid.UUID) (*db.GroupMember, error) {
			return &db.GroupMember{Role: db.GroupMemberRoleAdmin}, nil
		},
		getPlayerByIDFn: func(ctx context.Context, pID uuid.UUID) (*db.Player, error) {
			return &db.Player{ID: pID, Name: "Jogador", PendingRegistration: false}, nil
		},
		updatePlayerNameFn: func(ctx context.Context, pID uuid.UUID, name string) error {
			called = true
			return nil
		},
	}

	r := memberNameRouter(store)
	ctx := middleware.InjectPlayerForTest(context.Background(), fakePlayer())
	w := sendRequestWithContext(r, http.MethodPatch,
		fmt.Sprintf("/groups/%s/members/%s", groupID, targetID),
		`{"name":"Outro Nome"}`, ctx)

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "PLAYER_NOT_PENDING")
	assert.False(t, called)
}

func TestUpdateMember_NameTooShort(t *testing.T) {
	groupID := uuid.New()
	targetID := uuid.New()

	store := &mockGroupStoreForBusiness{
		getGroupMemberFn: func(ctx context.Context, gID, pID uuid.UUID) (*db.GroupMember, error) {
			return &db.GroupMember{Role: db.GroupMemberRoleAdmin}, nil
		},
	}

	r := memberNameRouter(store)
	ctx := middleware.InjectPlayerForTest(context.Background(), fakePlayer())
	w := sendRequestWithContext(r, http.MethodPatch,
		fmt.Sprintf("/groups/%s/members/%s", groupID, targetID),
		`{"name":"x"}`, ctx)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestUpdateMember_NameIgnoredForNonAdmin(t *testing.T) {
	groupID := uuid.New()
	self := fakePlayer()
	called := false

	store := &mockGroupStoreForBusiness{
		getGroupMemberFn: func(ctx context.Context, gID, pID uuid.UUID) (*db.GroupMember, error) {
			return &db.GroupMember{Role: db.GroupMemberRoleMember}, nil
		},
		updatePlayerNameFn: func(ctx context.Context, pID uuid.UUID, name string) error {
			called = true
			return nil
		},
	}

	r := memberNameRouter(store)
	ctx := middleware.InjectPlayerForTest(context.Background(), self)
	w := sendRequestWithContext(r, http.MethodPatch,
		fmt.Sprintf("/groups/%s/members/%s", groupID, self.ID),
		`{"name":"Nome Novo"}`, ctx)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.False(t, called)
}
