package unit_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/thiagotn/football-manager/football-api-go/internal/db"
	"github.com/thiagotn/football-manager/football-api-go/internal/handlers"
	"github.com/thiagotn/football-manager/football-api-go/internal/middleware"
)

// ── beta ──────────────────────────────────────────────────────────────────────

type mockBetaStore struct {
	insertAndroidBetaSignupFn func(ctx context.Context, email string, playerID *uuid.UUID) error
}

func (m *mockBetaStore) InsertAndroidBetaSignup(ctx context.Context, email string, playerID *uuid.UUID) error {
	if m.insertAndroidBetaSignupFn != nil {
		return m.insertAndroidBetaSignupFn(ctx, email, playerID)
	}
	return nil
}

func betaRouter() http.Handler {
	r := chi.NewRouter()
	h := &handlers.BetaHandler{Store: &mockBetaStore{}}
	r.Post("/beta/android-signup", h.AndroidSignup)
	return r
}

func TestBeta_AndroidSignup_InvalidEmail(t *testing.T) {
	r := betaRouter()
	w := postJSON(r, "/beta/android-signup", `{"google_email":"not-an-email"}`)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestBeta_AndroidSignup_MissingEmail(t *testing.T) {
	r := betaRouter()
	w := postJSON(r, "/beta/android-signup", `{}`)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestBeta_AndroidSignup_MalformedJSON(t *testing.T) {
	r := betaRouter()
	w := postJSON(r, "/beta/android-signup", `{bad}`)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

// ── mcp_tokens ────────────────────────────────────────────────────────────────

type mockMCPTokenStore struct {
	generateMCPTokenFn func() (raw, hash, prefix string, err error)
	createMCPTokenFn   func(ctx context.Context, params db.CreateMCPTokenParams) (*db.MCPToken, error)
	listMCPTokensFn    func(ctx context.Context, playerID uuid.UUID) ([]db.MCPToken, error)
	getMCPTokenFn      func(ctx context.Context, tokenID uuid.UUID) (*db.MCPToken, error)
	revokeMCPTokenFn   func(ctx context.Context, tokenID uuid.UUID) error
}

func (m *mockMCPTokenStore) GenerateMCPToken() (raw, hash, prefix string, err error) {
	if m.generateMCPTokenFn != nil {
		return m.generateMCPTokenFn()
	}
	return "raw", "hash", "prefix", nil
}

func (m *mockMCPTokenStore) CreateMCPToken(ctx context.Context, params db.CreateMCPTokenParams) (*db.MCPToken, error) {
	if m.createMCPTokenFn != nil {
		return m.createMCPTokenFn(ctx, params)
	}
	return nil, nil
}

func (m *mockMCPTokenStore) ListMCPTokens(ctx context.Context, playerID uuid.UUID) ([]db.MCPToken, error) {
	if m.listMCPTokensFn != nil {
		return m.listMCPTokensFn(ctx, playerID)
	}
	return []db.MCPToken{}, nil
}

func (m *mockMCPTokenStore) GetMCPToken(ctx context.Context, tokenID uuid.UUID) (*db.MCPToken, error) {
	if m.getMCPTokenFn != nil {
		return m.getMCPTokenFn(ctx, tokenID)
	}
	return nil, nil
}

func (m *mockMCPTokenStore) RevokeMCPToken(ctx context.Context, tokenID uuid.UUID) error {
	if m.revokeMCPTokenFn != nil {
		return m.revokeMCPTokenFn(ctx, tokenID)
	}
	return nil
}

func mcpTokenRouter(player *db.Player) http.Handler {
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := middleware.InjectPlayerForTest(req.Context(), player)
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	h := &handlers.MCPTokenHandler{Store: &mockMCPTokenStore{}}
	r.Mount("/mcp-tokens", h.Routes())
	return r
}

func TestMCPToken_Create_MissingName(t *testing.T) {
	r := mcpTokenRouter(fakePlayer())
	w := postJSON(r, "/mcp-tokens", `{"expires_in":null}`)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestMCPToken_Create_InvalidExpiresIn(t *testing.T) {
	r := mcpTokenRouter(fakePlayer())
	w := postJSON(r, "/mcp-tokens", `{"name":"my-token","expires_in":"1year"}`)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestMCPToken_Revoke_InvalidUUID(t *testing.T) {
	r := mcpTokenRouter(fakePlayer())
	w := doRequest(r, http.MethodDelete, "/mcp-tokens/not-a-uuid", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ── reviews ───────────────────────────────────────────────────────────────────

type mockReviewStore struct {
	getMyReviewFn      func(ctx context.Context, playerID uuid.UUID) (*db.AppReview, error)
	upsertReviewFn     func(ctx context.Context, playerID uuid.UUID, rating int, comment *string) (*db.AppReview, error)
	getReviewSummaryFn func(ctx context.Context) (*db.ReviewSummary, error)
	listReviewsFn      func(ctx context.Context, ratings []int, orderBy string, page, pageSize int) (*db.ReviewPage, error)
}

func (m *mockReviewStore) GetMyReview(ctx context.Context, playerID uuid.UUID) (*db.AppReview, error) {
	if m.getMyReviewFn != nil {
		return m.getMyReviewFn(ctx, playerID)
	}
	return nil, nil
}

func (m *mockReviewStore) UpsertReview(ctx context.Context, playerID uuid.UUID, rating int, comment *string) (*db.AppReview, error) {
	if m.upsertReviewFn != nil {
		return m.upsertReviewFn(ctx, playerID, rating, comment)
	}
	return nil, nil
}

func (m *mockReviewStore) GetReviewSummary(ctx context.Context) (*db.ReviewSummary, error) {
	if m.getReviewSummaryFn != nil {
		return m.getReviewSummaryFn(ctx)
	}
	return nil, nil
}

func (m *mockReviewStore) ListReviews(ctx context.Context, ratings []int, orderBy string, page, pageSize int) (*db.ReviewPage, error) {
	if m.listReviewsFn != nil {
		return m.listReviewsFn(ctx, ratings, orderBy, page, pageSize)
	}
	return nil, nil
}

func reviewRouter(player *db.Player) http.Handler {
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := middleware.InjectPlayerForTest(req.Context(), player)
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	h := &handlers.ReviewHandler{Store: &mockReviewStore{}}
	r.Get("/reviews/me", h.GetMyReview)
	r.Put("/reviews/me", h.UpsertMyReview)
	r.Get("/reviews/summary", h.GetSummary)
	r.Get("/reviews", h.ListReviews)
	return r
}

func TestReviews_Admin_Forbidden(t *testing.T) {
	r := reviewRouter(fakePlayer(asAdmin()))
	w := doRequest(r, http.MethodGet, "/reviews/me", "")
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestReviews_Upsert_InvalidRating(t *testing.T) {
	r := reviewRouter(fakePlayer())
	// HTTP PUT, not POST
	w := doRequest(r, http.MethodPut, "/reviews/me", `{"rating":6}`)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestReviews_Summary_NonAdmin_Forbidden(t *testing.T) {
	r := reviewRouter(fakePlayer())
	w := doRequest(r, http.MethodGet, "/reviews/summary", "")
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestReviews_List_NonAdmin_Forbidden(t *testing.T) {
	r := reviewRouter(fakePlayer())
	w := doRequest(r, http.MethodGet, "/reviews", "")
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// ── push ──────────────────────────────────────────────────────────────────────

func pushRouter(player *db.Player) http.Handler {
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := middleware.InjectPlayerForTest(req.Context(), player)
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	h := handlers.NewPushHandler(nil, "test-vapid-key")
	r.Get("/push/vapid-public-key", h.GetVapidKey)
	r.Post("/push/subscribe", h.Subscribe)
	r.Delete("/push/subscribe", h.Unsubscribe)
	return r
}

func TestPush_VapidKey_PublicRoute(t *testing.T) {
	r := chi.NewRouter()
	h := handlers.NewPushHandler(nil, "test-vapid-key")
	r.Get("/push/vapid-public-key", h.GetVapidKey)
	w := doRequest(r, http.MethodGet, "/push/vapid-public-key", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPush_Subscribe_MissingEndpoint(t *testing.T) {
	r := pushRouter(fakePlayer())
	w := postJSON(r, "/push/subscribe", `{"keys":{"p256dh":"abc","auth":"def"}}`)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

// ── ranking ───────────────────────────────────────────────────────────────────

func rankingRouter() http.Handler {
	r := chi.NewRouter()
	r.Get("/ranking", handlers.NewRankingHandler(nil).GetRanking)
	return r
}

func TestRanking_InvalidType(t *testing.T) {
	r := rankingRouter()
	w := doRequest(r, http.MethodGet, "/ranking?type=invalid", "")
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestRanking_MonthWithoutYear(t *testing.T) {
	r := rankingRouter()
	w := doRequest(r, http.MethodGet, "/ranking?month=5", "")
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestRanking_InvalidYear(t *testing.T) {
	r := rankingRouter()
	w := doRequest(r, http.MethodGet, "/ranking?year=2020", "")
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

// ── votes (validation before DB) ─────────────────────────────────────────────

func votesRouter(player *db.Player) http.Handler {
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := middleware.InjectPlayerForTest(req.Context(), player)
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	h := handlers.NewVoteHandler(nil)
	r.Get("/matches/{matchID}/votes/status", h.GetVoteStatus)
	r.Post("/matches/{matchID}/votes", h.SubmitVote)
	r.Post("/matches/{matchID}/votes/close", h.CloseVoting)
	r.Get("/matches/{matchID}/votes/results", h.GetVoteResults)
	r.Get("/votes/pending", h.GetPendingVotes)
	return r
}

func TestVotes_Status_InvalidMatchID(t *testing.T) {
	r := votesRouter(fakePlayer())
	w := doRequest(r, http.MethodGet, "/matches/not-a-uuid/votes/status", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestVotes_Submit_InvalidMatchID(t *testing.T) {
	r := votesRouter(fakePlayer())
	w := postJSON(r, "/matches/not-a-uuid/votes", `{"top5":[],"flop_player_id":null}`)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestVotes_PendingVotes_AdminGetsEmpty(t *testing.T) {
	r := votesRouter(fakePlayer(asAdmin()))
	w := doRequest(r, http.MethodGet, "/votes/pending", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

// ── finance ───────────────────────────────────────────────────────────────────

func financeRouter(player *db.Player) http.Handler {
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := middleware.InjectPlayerForTest(req.Context(), player)
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	h := handlers.NewFinanceHandler(nil)
	r.Get("/groups/{groupID}/finance/periods", h.ListPeriods)
	r.Get("/groups/{groupID}/finance/periods/{year}/{month}", h.GetPeriod)
	r.Patch("/finance/payments/{paymentID}", h.UpdatePayment)
	return r
}

func TestFinance_ListPeriods_InvalidGroupID(t *testing.T) {
	r := financeRouter(fakePlayer())
	w := doRequest(r, http.MethodGet, "/groups/not-a-uuid/finance/periods", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestFinance_GetPeriod_InvalidGroupID(t *testing.T) {
	r := financeRouter(fakePlayer())
	w := doRequest(r, http.MethodGet, "/groups/not-a-uuid/finance/periods/2025/1", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestFinance_UpdatePayment_InvalidUUID(t *testing.T) {
	r := financeRouter(fakePlayer())
	w := doRequest(r, http.MethodPatch, "/finance/payments/not-a-uuid", `{"status":"paid"}`)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ── subscriptions ─────────────────────────────────────────────────────────────

func subscriptionRouter(player *db.Player) http.Handler {
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := middleware.InjectPlayerForTest(req.Context(), player)
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	h := handlers.NewSubscriptionHandler(nil, nil)
	r.Get("/subscriptions/me", h.GetMySubscription)
	r.Post("/subscriptions", h.CreateCheckoutSession)
	return r
}

func TestSubscription_CreateCheckout_InvalidPlan(t *testing.T) {
	r := subscriptionRouter(fakePlayer())
	w := postJSON(r, "/subscriptions", `{"plan":"free","billing_cycle":"monthly"}`)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestSubscription_CreateCheckout_InvalidBillingCycle(t *testing.T) {
	r := subscriptionRouter(fakePlayer())
	w := postJSON(r, "/subscriptions", `{"plan":"basic","billing_cycle":"quarterly"}`)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}
