package unit_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/thiagotn/football-manager/football-api-go/internal/db"
	"github.com/thiagotn/football-manager/football-api-go/internal/handlers"
	"github.com/thiagotn/football-manager/football-api-go/internal/middleware"
)

// ── Finance Handler ───────────────────────────────────────────────────────────

type mockFinanceStore struct {
	listFinancePeriodsFn       func(ctx context.Context, groupID uuid.UUID) ([]db.FinancePeriod, error)
	getFinancePeriodFn         func(ctx context.Context, groupID uuid.UUID, year, month int) (*db.FinancePeriod, error)
	getOrCreateFinancePeriodFn func(ctx context.Context, groupID uuid.UUID, year, month int) (*db.FinancePeriod, error)
	getPaymentsForPeriodFn     func(ctx context.Context, periodID uuid.UUID) ([]db.FinancePayment, error)
	getFinancePaymentFn        func(ctx context.Context, paymentID uuid.UUID) (*db.FinancePayment, error)
	getPeriodGroupIDFn         func(ctx context.Context, periodID uuid.UUID) (uuid.UUID, error)
	getGroupByIDFn             func(ctx context.Context, groupID uuid.UUID) (*db.Group, error)
	getGroupMemberFn           func(ctx context.Context, groupID, playerID uuid.UUID) (*db.GroupMember, error)
	markPaymentPaidFn          func(ctx context.Context, paymentID uuid.UUID, paymentType string, amountCents int) (*db.FinancePayment, error)
	markPaymentPendingFn       func(ctx context.Context, paymentID uuid.UUID) (*db.FinancePayment, error)
}

func (m *mockFinanceStore) ListFinancePeriods(ctx context.Context, groupID uuid.UUID) ([]db.FinancePeriod, error) {
	if m.listFinancePeriodsFn != nil {
		return m.listFinancePeriodsFn(ctx, groupID)
	}
	return []db.FinancePeriod{}, nil
}

func (m *mockFinanceStore) GetFinancePeriod(ctx context.Context, groupID uuid.UUID, year, month int) (*db.FinancePeriod, error) {
	if m.getFinancePeriodFn != nil {
		return m.getFinancePeriodFn(ctx, groupID, year, month)
	}
	return nil, db.ErrNotFound
}

func (m *mockFinanceStore) GetOrCreateFinancePeriod(ctx context.Context, groupID uuid.UUID, year, month int) (*db.FinancePeriod, error) {
	if m.getOrCreateFinancePeriodFn != nil {
		return m.getOrCreateFinancePeriodFn(ctx, groupID, year, month)
	}
	return nil, nil
}

func (m *mockFinanceStore) GetPaymentsForPeriod(ctx context.Context, periodID uuid.UUID) ([]db.FinancePayment, error) {
	if m.getPaymentsForPeriodFn != nil {
		return m.getPaymentsForPeriodFn(ctx, periodID)
	}
	return []db.FinancePayment{}, nil
}

func (m *mockFinanceStore) GetFinancePayment(ctx context.Context, paymentID uuid.UUID) (*db.FinancePayment, error) {
	if m.getFinancePaymentFn != nil {
		return m.getFinancePaymentFn(ctx, paymentID)
	}
	return nil, db.ErrNotFound
}

func (m *mockFinanceStore) GetPeriodGroupID(ctx context.Context, periodID uuid.UUID) (uuid.UUID, error) {
	if m.getPeriodGroupIDFn != nil {
		return m.getPeriodGroupIDFn(ctx, periodID)
	}
	return uuid.Nil, nil
}

func (m *mockFinanceStore) GetGroupByID(ctx context.Context, groupID uuid.UUID) (*db.Group, error) {
	if m.getGroupByIDFn != nil {
		return m.getGroupByIDFn(ctx, groupID)
	}
	return nil, db.ErrNotFound
}

func (m *mockFinanceStore) GetGroupMember(ctx context.Context, groupID, playerID uuid.UUID) (*db.GroupMember, error) {
	if m.getGroupMemberFn != nil {
		return m.getGroupMemberFn(ctx, groupID, playerID)
	}
	return nil, db.ErrNotFound
}

func (m *mockFinanceStore) MarkPaymentPaid(ctx context.Context, paymentID uuid.UUID, paymentType string, amountCents int) (*db.FinancePayment, error) {
	if m.markPaymentPaidFn != nil {
		return m.markPaymentPaidFn(ctx, paymentID, paymentType, amountCents)
	}
	return nil, nil
}

func (m *mockFinanceStore) MarkPaymentPending(ctx context.Context, paymentID uuid.UUID) (*db.FinancePayment, error) {
	if m.markPaymentPendingFn != nil {
		return m.markPaymentPendingFn(ctx, paymentID)
	}
	return nil, nil
}

func financeRouter(store handlers.FinanceStore) http.Handler {
	r := chi.NewRouter()
	h := &handlers.FinanceHandler{Store: store}
	r.Get("/groups/{groupID}/finance/periods", h.ListPeriods)
	r.Get("/groups/{groupID}/finance/{year}/{month}", h.GetPeriod)
	r.Patch("/finance/payments/{paymentID}", h.UpdatePayment)
	return r
}

func TestFinance_ListPeriods_NoAuth(t *testing.T) {
	r := financeRouter(&mockFinanceStore{})
	groupID := uuid.New()
	w := sendRequest(r, "GET", "/groups/"+groupID.String()+"/finance/periods", "", "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestFinance_UpdatePayment_InvalidPaymentID(t *testing.T) {
	r := financeRouter(&mockFinanceStore{})
	player := &db.Player{ID: uuid.New(), Role: db.PlayerRoleAdmin}
	ctx := middleware.InjectPlayerForTest(context.Background(), player)
	w := sendRequestWithContext(r, "PATCH", "/finance/payments/invalid-uuid", `{}`, ctx)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestFinance_UpdatePayment_PaymentNotFound(t *testing.T) {
	r := financeRouter(&mockFinanceStore{})
	player := &db.Player{ID: uuid.New(), Role: db.PlayerRoleAdmin}
	ctx := middleware.InjectPlayerForTest(context.Background(), player)
	paymentID := uuid.New()
	w := sendRequestWithContext(r, "PATCH", "/finance/payments/"+paymentID.String(), `{}`, ctx)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestFinance_UpdatePayment_MissingPaymentType(t *testing.T) {
	r := financeRouter(&mockFinanceStore{
		getFinancePaymentFn: func(ctx context.Context, paymentID uuid.UUID) (*db.FinancePayment, error) {
			return &db.FinancePayment{ID: paymentID}, nil
		},
		getPeriodGroupIDFn: func(ctx context.Context, periodID uuid.UUID) (uuid.UUID, error) {
			return uuid.New(), nil
		},
	})
	player := &db.Player{ID: uuid.New(), Role: db.PlayerRoleAdmin}
	ctx := middleware.InjectPlayerForTest(context.Background(), player)
	paymentID := uuid.New()
	w := sendRequestWithContext(r, "PATCH", "/finance/payments/"+paymentID.String(), `{"status":"paid"}`, ctx)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

// ── Invites Handler ───────────────────────────────────────────────────────────

type mockInviteStore struct {
	getGroupMemberFn           func(ctx context.Context, groupID, playerID uuid.UUID) (*db.GroupMember, error)
	createInviteFn             func(ctx context.Context, groupID, callerID uuid.UUID, token string, expiresAt time.Time) (*db.Invite, error)
	getInviteByTokenFn         func(ctx context.Context, token string) (*db.InviteWithGroup, error)
	getPlayerByWhatsAppFn      func(ctx context.Context, whatsapp string) (*db.Player, error)
	planMembersLimitFn         func(plan string) int
	countGroupMembersFn        func(ctx context.Context, groupID uuid.UUID) (int, error)
	createPlayerFn             func(ctx context.Context, params db.CreatePlayerParams) (*db.Player, error)
	ensurePlayerSubscriptionFn func(ctx context.Context, playerID uuid.UUID) error
	addGroupMemberFn           func(ctx context.Context, groupID, playerID uuid.UUID, role db.GroupMemberRole) (*db.GroupMember, error)
	getOpenMatchesForGroupFn   func(ctx context.Context, groupID uuid.UUID) ([]uuid.UUID, error)
	setAttendanceFn            func(ctx context.Context, matchID, playerID uuid.UUID, status string) error
	useInviteFn                func(ctx context.Context, token string, playerID uuid.UUID) error
}

func (m *mockInviteStore) GetGroupMember(ctx context.Context, groupID, playerID uuid.UUID) (*db.GroupMember, error) {
	if m.getGroupMemberFn != nil {
		return m.getGroupMemberFn(ctx, groupID, playerID)
	}
	return nil, db.ErrNotFound
}

func (m *mockInviteStore) CreateInvite(ctx context.Context, groupID, callerID uuid.UUID, token string, expiresAt time.Time) (*db.Invite, error) {
	if m.createInviteFn != nil {
		return m.createInviteFn(ctx, groupID, callerID, token, expiresAt)
	}
	return nil, nil
}

func (m *mockInviteStore) GetInviteByToken(ctx context.Context, token string) (*db.InviteWithGroup, error) {
	if m.getInviteByTokenFn != nil {
		return m.getInviteByTokenFn(ctx, token)
	}
	return nil, db.ErrNotFound
}

func (m *mockInviteStore) GetPlayerByWhatsApp(ctx context.Context, whatsapp string) (*db.Player, error) {
	if m.getPlayerByWhatsAppFn != nil {
		return m.getPlayerByWhatsAppFn(ctx, whatsapp)
	}
	return nil, db.ErrNotFound
}

func (m *mockInviteStore) PlanMembersLimit(plan string) int {
	if m.planMembersLimitFn != nil {
		return m.planMembersLimitFn(plan)
	}
	return 50 // default
}

func (m *mockInviteStore) CountGroupMembers(ctx context.Context, groupID uuid.UUID) (int, error) {
	if m.countGroupMembersFn != nil {
		return m.countGroupMembersFn(ctx, groupID)
	}
	return 0, nil
}

func (m *mockInviteStore) CreatePlayer(ctx context.Context, params db.CreatePlayerParams) (*db.Player, error) {
	if m.createPlayerFn != nil {
		return m.createPlayerFn(ctx, params)
	}
	return nil, nil
}

func (m *mockInviteStore) EnsurePlayerSubscription(ctx context.Context, playerID uuid.UUID) error {
	if m.ensurePlayerSubscriptionFn != nil {
		return m.ensurePlayerSubscriptionFn(ctx, playerID)
	}
	return nil
}

func (m *mockInviteStore) AddGroupMember(ctx context.Context, groupID, playerID uuid.UUID, role db.GroupMemberRole) (*db.GroupMember, error) {
	if m.addGroupMemberFn != nil {
		return m.addGroupMemberFn(ctx, groupID, playerID, role)
	}
	return nil, nil
}

func (m *mockInviteStore) GetOpenMatchesForGroup(ctx context.Context, groupID uuid.UUID) ([]uuid.UUID, error) {
	if m.getOpenMatchesForGroupFn != nil {
		return m.getOpenMatchesForGroupFn(ctx, groupID)
	}
	return []uuid.UUID{}, nil
}

func (m *mockInviteStore) SetAttendance(ctx context.Context, matchID, playerID uuid.UUID, status string) error {
	if m.setAttendanceFn != nil {
		return m.setAttendanceFn(ctx, matchID, playerID, status)
	}
	return nil
}

func (m *mockInviteStore) UseInvite(ctx context.Context, token string, playerID uuid.UUID) error {
	if m.useInviteFn != nil {
		return m.useInviteFn(ctx, token, playerID)
	}
	return nil
}

// NOTE: Invite handler tests are integration-level and require full initialization.
// Unit tests for invite logic will be covered with mock Store in future phases.
// Skipping direct handler construction due to complex dependencies (AuthService, Config).

// ── Subscriptions Handler ─────────────────────────────────────────────────────

type mockSubscriptionStore struct {
	getOrCreateSubscriptionFn func(ctx context.Context, playerID uuid.UUID) (*db.PlayerSubscription, error)
	updateSubscriptionFn      func(ctx context.Context, playerID uuid.UUID, params db.UpdateSubscriptionParams) (*db.PlayerSubscription, error)
	countAdminGroupsFn        func(ctx context.Context, playerID uuid.UUID) (int, error)
}

func (m *mockSubscriptionStore) GetOrCreateSubscription(ctx context.Context, playerID uuid.UUID) (*db.PlayerSubscription, error) {
	if m.getOrCreateSubscriptionFn != nil {
		return m.getOrCreateSubscriptionFn(ctx, playerID)
	}
	return &db.PlayerSubscription{PlayerID: playerID, Plan: "free"}, nil
}

func (m *mockSubscriptionStore) UpdateSubscription(ctx context.Context, playerID uuid.UUID, params db.UpdateSubscriptionParams) (*db.PlayerSubscription, error) {
	if m.updateSubscriptionFn != nil {
		return m.updateSubscriptionFn(ctx, playerID, params)
	}
	return nil, nil
}

func (m *mockSubscriptionStore) CountAdminGroups(ctx context.Context, playerID uuid.UUID) (int, error) {
	if m.countAdminGroupsFn != nil {
		return m.countAdminGroupsFn(ctx, playerID)
	}
	return 0, nil
}

func subscriptionRouter(store handlers.SubscriptionStore) http.Handler {
	r := chi.NewRouter()
	h := &handlers.SubscriptionHandler{Store: store, Stripe: nil}
	r.Get("/me", h.GetMySubscription)
	r.Post("/checkout", h.CreateCheckoutSession)
	return r
}

func TestSubscription_GetMySubscription_NoAuth(t *testing.T) {
	r := subscriptionRouter(&mockSubscriptionStore{})
	w := sendRequest(r, "GET", "/me", "", "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSubscription_CreateCheckoutSession_NoAuth(t *testing.T) {
	r := subscriptionRouter(&mockSubscriptionStore{})
	w := postJSON(r, "/checkout", `{"plan":"basic","billing_cycle":"monthly"}`)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSubscription_CreateCheckoutSession_InvalidPlan(t *testing.T) {
	r := subscriptionRouter(&mockSubscriptionStore{})
	player := &db.Player{ID: uuid.New(), Role: db.PlayerRolePlayer}
	ctx := middleware.InjectPlayerForTest(context.Background(), player)
	w := sendRequestWithContext(r, "POST", "/checkout", `{"plan":"free","billing_cycle":"monthly"}`, ctx)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestSubscription_CreateCheckoutSession_InvalidBillingCycle(t *testing.T) {
	r := subscriptionRouter(&mockSubscriptionStore{})
	player := &db.Player{ID: uuid.New(), Role: db.PlayerRolePlayer}
	ctx := middleware.InjectPlayerForTest(context.Background(), player)
	w := sendRequestWithContext(r, "POST", "/checkout", `{"plan":"basic","billing_cycle":"weekly"}`, ctx)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestSubscription_CreateCheckoutSession_StripeNotConfigured(t *testing.T) {
	r := subscriptionRouter(&mockSubscriptionStore{})
	player := &db.Player{ID: uuid.New(), Role: db.PlayerRolePlayer}
	ctx := middleware.InjectPlayerForTest(context.Background(), player)
	w := sendRequestWithContext(r, "POST", "/checkout", `{"plan":"basic","billing_cycle":"monthly"}`, ctx)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ── Votes Handler (basic tests) ───────────────────────────────────────────────

type mockVoteStore struct {
	getMatchByIDFn           func(ctx context.Context, matchID uuid.UUID) (*db.Match, error)
	getMatchByHashFn         func(ctx context.Context, matchHash string) (*db.Match, error)
	getAttendancesForMatchFn func(ctx context.Context, matchID uuid.UUID) ([]db.AttendanceWithPlayer, error)
	voterCountFn             func(ctx context.Context, matchID uuid.UUID) (int, error)
	hasVotedFn               func(ctx context.Context, matchID, playerID uuid.UUID) (bool, error)
	voterIDsFn               func(ctx context.Context, matchID uuid.UUID) ([]uuid.UUID, error)
	markVoteNotifiedFn       func(ctx context.Context, matchID uuid.UUID) error
	submitVoteFn             func(ctx context.Context, matchID, playerID uuid.UUID, top5 []db.VoteTop5Item, flopID *uuid.UUID) error
	getPendingVotesFn        func(ctx context.Context, playerID uuid.UUID) ([]db.PendingVoteItem, error)
	getVoteResultsFn         func(ctx context.Context, matchID uuid.UUID) (*db.VoteResults, error)
	getVoteBallotsFn         func(ctx context.Context, matchID uuid.UUID) ([]db.Ballot, error)
	closeVotingEarlyFn       func(ctx context.Context, matchID uuid.UUID) error
	getGroupMemberFn         func(ctx context.Context, groupID, playerID uuid.UUID) (*db.GroupMember, error)
	groupVotingEnabledFn     func(ctx context.Context, groupID uuid.UUID) (bool, error)
}

func (m *mockVoteStore) GetMatchByID(ctx context.Context, matchID uuid.UUID) (*db.Match, error) {
	if m.getMatchByIDFn != nil {
		return m.getMatchByIDFn(ctx, matchID)
	}
	return nil, db.ErrNotFound
}

func (m *mockVoteStore) GetMatchByHash(ctx context.Context, matchHash string) (*db.Match, error) {
	if m.getMatchByHashFn != nil {
		return m.getMatchByHashFn(ctx, matchHash)
	}
	return nil, db.ErrNotFound
}

func (m *mockVoteStore) GetAttendancesForMatch(ctx context.Context, matchID uuid.UUID) ([]db.AttendanceWithPlayer, error) {
	if m.getAttendancesForMatchFn != nil {
		return m.getAttendancesForMatchFn(ctx, matchID)
	}
	return []db.AttendanceWithPlayer{}, nil
}

func (m *mockVoteStore) VoterCount(ctx context.Context, matchID uuid.UUID) (int, error) {
	if m.voterCountFn != nil {
		return m.voterCountFn(ctx, matchID)
	}
	return 0, nil
}

func (m *mockVoteStore) HasVoted(ctx context.Context, matchID, playerID uuid.UUID) (bool, error) {
	if m.hasVotedFn != nil {
		return m.hasVotedFn(ctx, matchID, playerID)
	}
	return false, nil
}

func (m *mockVoteStore) VoterIDs(ctx context.Context, matchID uuid.UUID) ([]uuid.UUID, error) {
	if m.voterIDsFn != nil {
		return m.voterIDsFn(ctx, matchID)
	}
	return []uuid.UUID{}, nil
}

func (m *mockVoteStore) MarkVoteNotified(ctx context.Context, matchID uuid.UUID) error {
	if m.markVoteNotifiedFn != nil {
		return m.markVoteNotifiedFn(ctx, matchID)
	}
	return nil
}

func (m *mockVoteStore) SubmitVote(ctx context.Context, matchID, playerID uuid.UUID, top5 []db.VoteTop5Item, flopID *uuid.UUID) error {
	if m.submitVoteFn != nil {
		return m.submitVoteFn(ctx, matchID, playerID, top5, flopID)
	}
	return nil
}

func (m *mockVoteStore) GetPendingVotes(ctx context.Context, playerID uuid.UUID) ([]db.PendingVoteItem, error) {
	if m.getPendingVotesFn != nil {
		return m.getPendingVotesFn(ctx, playerID)
	}
	return []db.PendingVoteItem{}, nil
}

func (m *mockVoteStore) GetVoteResults(ctx context.Context, matchID uuid.UUID) (*db.VoteResults, error) {
	if m.getVoteResultsFn != nil {
		return m.getVoteResultsFn(ctx, matchID)
	}
	return nil, db.ErrNotFound
}

func (m *mockVoteStore) GetVoteBallots(ctx context.Context, matchID uuid.UUID) ([]db.Ballot, error) {
	if m.getVoteBallotsFn != nil {
		return m.getVoteBallotsFn(ctx, matchID)
	}
	return []db.Ballot{}, nil
}

func (m *mockVoteStore) CloseVotingEarly(ctx context.Context, matchID uuid.UUID) error {
	if m.closeVotingEarlyFn != nil {
		return m.closeVotingEarlyFn(ctx, matchID)
	}
	return nil
}

func (m *mockVoteStore) GetGroupMember(ctx context.Context, groupID, playerID uuid.UUID) (*db.GroupMember, error) {
	if m.getGroupMemberFn != nil {
		return m.getGroupMemberFn(ctx, groupID, playerID)
	}
	return nil, db.ErrNotFound
}

func (m *mockVoteStore) GroupVotingEnabled(ctx context.Context, groupID uuid.UUID) (bool, error) {
	if m.groupVotingEnabledFn != nil {
		return m.groupVotingEnabledFn(ctx, groupID)
	}
	// Default true to preserve existing test behaviour — tests that exercise the
	// off-path set their own stub.
	return true, nil
}

func voteRouter(store handlers.VoteStore) http.Handler {
	r := chi.NewRouter()
	h := &handlers.VoteHandler{Store: store}
	r.Get("/matches/{matchID}/vote-status", h.GetVoteStatus)
	r.Post("/matches/{matchID}/vote", h.SubmitVote)
	r.Get("/votes/pending", h.GetPendingVotes)
	return r
}

func TestVote_GetVoteStatus_InvalidMatchID(t *testing.T) {
	r := voteRouter(&mockVoteStore{})
	w := sendRequest(r, "GET", "/matches/invalid-uuid/vote-status", "", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestVote_SubmitVote_NoAuth(t *testing.T) {
	r := voteRouter(&mockVoteStore{})
	matchID := uuid.New()
	w := postJSON(r, "/matches/"+matchID.String()+"/vote", `{}`)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestVote_GetVoteStatus_MatchNotFound(t *testing.T) {
	r := voteRouter(&mockVoteStore{})
	matchID := uuid.New()
	player := &db.Player{ID: uuid.New(), Role: db.PlayerRolePlayer}
	ctx := middleware.InjectPlayerForTest(context.Background(), player)
	w := sendRequestWithContext(r, "GET", "/matches/"+matchID.String()+"/vote-status", "", ctx)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestVote_SubmitVote_MatchNotFound(t *testing.T) {
	r := voteRouter(&mockVoteStore{})
	matchID := uuid.New()
	player := &db.Player{ID: uuid.New(), Role: db.PlayerRolePlayer}
	ctx := middleware.InjectPlayerForTest(context.Background(), player)
	// Valid body but match doesn't exist (store returns not found)
	w := sendRequestWithContext(r, "POST", "/matches/"+matchID.String()+"/vote", `{"top5":[],"flop_player_id":null}`, ctx)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestVote_GetPendingVotes_NoAuth(t *testing.T) {
	r := voteRouter(&mockVoteStore{})
	w := sendRequest(r, "GET", "/votes/pending", "", "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ── Webhooks Handler ──────────────────────────────────────────────────────────

type mockWebhookStore struct {
	isWebhookEventProcessedFn func(ctx context.Context, eventID string) (bool, error)
	markWebhookEventProcessedFn func(ctx context.Context, eventID, eventType string) error
	getSubscriptionByGatewayCustomerFn func(ctx context.Context, customerID string) (*db.PlayerSubscription, error)
	updateSubscriptionFn func(ctx context.Context, playerID uuid.UUID, params db.UpdateSubscriptionParams) (*db.PlayerSubscription, error)
}

func (m *mockWebhookStore) IsWebhookEventProcessed(ctx context.Context, eventID string) (bool, error) {
	if m.isWebhookEventProcessedFn != nil {
		return m.isWebhookEventProcessedFn(ctx, eventID)
	}
	return false, nil
}

func (m *mockWebhookStore) MarkWebhookEventProcessed(ctx context.Context, eventID, eventType string) error {
	if m.markWebhookEventProcessedFn != nil {
		return m.markWebhookEventProcessedFn(ctx, eventID, eventType)
	}
	return nil
}

func (m *mockWebhookStore) GetSubscriptionByGatewayCustomer(ctx context.Context, customerID string) (*db.PlayerSubscription, error) {
	if m.getSubscriptionByGatewayCustomerFn != nil {
		return m.getSubscriptionByGatewayCustomerFn(ctx, customerID)
	}
	return nil, db.ErrNotFound
}

func (m *mockWebhookStore) UpdateSubscription(ctx context.Context, playerID uuid.UUID, params db.UpdateSubscriptionParams) (*db.PlayerSubscription, error) {
	if m.updateSubscriptionFn != nil {
		return m.updateSubscriptionFn(ctx, playerID, params)
	}
	return nil, nil
}

func webhookRouter(store handlers.WebhookStore) http.Handler {
	r := chi.NewRouter()
	h := &handlers.WebhookHandler{Store: store, Stripe: nil}
	r.Post("/webhooks/stripe", h.HandleStripeWebhook)
	return r
}

func TestWebhook_HandleStripeWebhook_NotConfigured(t *testing.T) {
	r := webhookRouter(&mockWebhookStore{})
	w := sendRequest(r, "POST", "/webhooks/stripe", `{}`, "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "not_configured")
}

// ── Invites Handler (basic tests - full integration pending) ──────────────────

type mockInviteStoreBasic struct {
	getGroupMemberFn func(ctx context.Context, groupID, playerID uuid.UUID) (*db.GroupMember, error)
}

func (m *mockInviteStoreBasic) GetGroupMember(ctx context.Context, groupID, playerID uuid.UUID) (*db.GroupMember, error) {
	if m.getGroupMemberFn != nil {
		return m.getGroupMemberFn(ctx, groupID, playerID)
	}
	return nil, db.ErrNotFound
}

func (m *mockInviteStoreBasic) CreateInvite(ctx context.Context, groupID, callerID uuid.UUID, token string, expiresAt time.Time) (*db.Invite, error) {
	return nil, nil
}

func (m *mockInviteStoreBasic) GetInviteByToken(ctx context.Context, token string) (*db.InviteWithGroup, error) {
	return nil, db.ErrNotFound
}

func (m *mockInviteStoreBasic) GetPlayerByWhatsApp(ctx context.Context, whatsapp string) (*db.Player, error) {
	return nil, db.ErrNotFound
}

func (m *mockInviteStoreBasic) CountGroupMembers(ctx context.Context, groupID uuid.UUID) (int, error) {
	return 0, nil
}

func (m *mockInviteStoreBasic) CreatePlayer(ctx context.Context, params db.CreatePlayerParams) (*db.Player, error) {
	return nil, nil
}

func (m *mockInviteStoreBasic) EnsurePlayerSubscription(ctx context.Context, playerID uuid.UUID) error {
	return nil
}

func (m *mockInviteStoreBasic) AddGroupMember(ctx context.Context, groupID, playerID uuid.UUID, role db.GroupMemberRole) (*db.GroupMember, error) {
	return nil, nil
}

func (m *mockInviteStoreBasic) GetOpenMatchesForGroup(ctx context.Context, groupID uuid.UUID) ([]uuid.UUID, error) {
	return []uuid.UUID{}, nil
}

func (m *mockInviteStoreBasic) SetAttendance(ctx context.Context, matchID, playerID uuid.UUID, status string) error {
	return nil
}

func (m *mockInviteStoreBasic) UseInvite(ctx context.Context, token string, playerID uuid.UUID) error {
	return nil
}

func (m *mockInviteStoreBasic) EnsureMemberInCurrentPeriod(ctx context.Context, groupID, playerID uuid.UUID, playerName string) error {
	return nil
}

// NOTE: Invite handler tests require auth service and config dependencies.
// Basic validation tests deferred to integration test phase.
