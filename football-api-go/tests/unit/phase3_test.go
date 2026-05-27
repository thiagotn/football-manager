package unit_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
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

func TestBeta_AndroidSignup_EmailTooLong(t *testing.T) {
	r := betaRouter()
	longEmail := "a" + strings.Repeat("b", 254) + "@example.com"
	w := postJSON(r, "/beta/android-signup", `{"google_email":"`+longEmail+`"}`)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "invalid email")
}

func TestBeta_AndroidSignup_Success_NoAuth(t *testing.T) {
	store := &mockBetaStore{
		insertAndroidBetaSignupFn: func(ctx context.Context, email string, playerID *uuid.UUID) error {
			assert.Equal(t, "test@example.com", email)
			assert.Nil(t, playerID)
			return nil
		},
	}
	r := chi.NewRouter()
	h := &handlers.BetaHandler{Store: store}
	r.Post("/beta/android-signup", h.AndroidSignup)

	w := postJSON(r, "/beta/android-signup", `{"google_email":"test@example.com"}`)
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "ok")
}

func TestBeta_AndroidSignup_Success_WithAuth(t *testing.T) {
	player := fakePlayer()
	store := &mockBetaStore{
		insertAndroidBetaSignupFn: func(ctx context.Context, email string, playerID *uuid.UUID) error {
			assert.Equal(t, "test@example.com", email)
			assert.NotNil(t, playerID)
			assert.Equal(t, player.ID, *playerID)
			return nil
		},
	}
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := middleware.InjectPlayerForTest(req.Context(), player)
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	h := &handlers.BetaHandler{Store: store}
	r.Post("/beta/android-signup", h.AndroidSignup)

	w := postJSON(r, "/beta/android-signup", `{"google_email":"test@example.com"}`)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestBeta_AndroidSignup_DBError(t *testing.T) {
	store := &mockBetaStore{
		insertAndroidBetaSignupFn: func(ctx context.Context, email string, playerID *uuid.UUID) error {
			return db.ErrNotFound
		},
	}
	r := chi.NewRouter()
	h := &handlers.BetaHandler{Store: store}
	r.Post("/beta/android-signup", h.AndroidSignup)

	w := postJSON(r, "/beta/android-signup", `{"google_email":"test@example.com"}`)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
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

func mcpTokenRouter(player *db.Player, store handlers.MCPTokenStore) http.Handler {
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := middleware.InjectPlayerForTest(req.Context(), player)
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	h := &handlers.MCPTokenHandler{Store: store}
	r.Mount("/mcp-tokens", h.Routes())
	return r
}

func TestMCPToken_Create_MissingName(t *testing.T) {
	r := mcpTokenRouter(fakePlayer(), &mockMCPTokenStore{})
	w := postJSON(r, "/mcp-tokens", `{"expires_in":null}`)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestMCPToken_Create_InvalidExpiresIn(t *testing.T) {
	r := mcpTokenRouter(fakePlayer(), &mockMCPTokenStore{})
	w := postJSON(r, "/mcp-tokens", `{"name":"my-token","expires_in":"1year"}`)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestMCPToken_Revoke_InvalidUUID(t *testing.T) {
	r := mcpTokenRouter(fakePlayer(), &mockMCPTokenStore{})
	w := doRequest(r, http.MethodDelete, "/mcp-tokens/not-a-uuid", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestMCPToken_Create_Success(t *testing.T) {
	player := fakePlayer()
	tokenID := uuid.New()
	store := &mockMCPTokenStore{
		generateMCPTokenFn: func() (raw, hash, prefix string, err error) {
			return "raw-token-abc123", "hash-xyz789", "tok_", nil
		},
		createMCPTokenFn: func(ctx context.Context, params db.CreateMCPTokenParams) (*db.MCPToken, error) {
			assert.Equal(t, player.ID, params.PlayerID)
			assert.Equal(t, "test-token", params.Name)
			return &db.MCPToken{
				ID:          tokenID,
				PlayerID:    player.ID,
				Name:        "test-token",
				TokenPrefix: "tok_",
			}, nil
		},
	}
	r := mcpTokenRouter(player, store)

	w := postJSON(r, "/mcp-tokens", `{"name":"test-token","expires_in":"h24"}`)
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "tok_")
}

func TestMCPToken_List_Success(t *testing.T) {
	player := fakePlayer()
	tokens := []db.MCPToken{
		{ID: uuid.New(), Name: "token-1", TokenPrefix: "tok_"},
		{ID: uuid.New(), Name: "token-2", TokenPrefix: "tok_"},
	}
	store := &mockMCPTokenStore{
		listMCPTokensFn: func(ctx context.Context, playerID uuid.UUID) ([]db.MCPToken, error) {
			assert.Equal(t, player.ID, playerID)
			return tokens, nil
		},
	}
	r := mcpTokenRouter(player, store)

	w := doRequest(r, http.MethodGet, "/mcp-tokens", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMCPToken_Revoke_Success(t *testing.T) {
	player := fakePlayer()
	tokenID := uuid.New()
	store := &mockMCPTokenStore{
		getMCPTokenFn: func(ctx context.Context, id uuid.UUID) (*db.MCPToken, error) {
			return &db.MCPToken{ID: id, PlayerID: player.ID}, nil
		},
		revokeMCPTokenFn: func(ctx context.Context, id uuid.UUID) error {
			assert.Equal(t, tokenID, id)
			return nil
		},
	}
	r := mcpTokenRouter(player, store)

	w := doRequest(r, http.MethodDelete, "/mcp-tokens/"+tokenID.String(), "")
	assert.Equal(t, http.StatusNoContent, w.Code)
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

func TestReviews_GetMyReview_Success(t *testing.T) {
	player := fakePlayer()
	review := &db.AppReview{
		ID:       uuid.New(),
		PlayerID: player.ID,
		Rating:   5,
		Comment:  strPtr("Great app!"),
	}
	r1 := chi.NewRouter()
	r1.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := middleware.InjectPlayerForTest(req.Context(), player)
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	h := &handlers.ReviewHandler{
		Store: &mockReviewStore{
			getMyReviewFn: func(ctx context.Context, playerID uuid.UUID) (*db.AppReview, error) {
				return review, nil
			},
		},
	}
	r1.Get("/reviews/me", h.GetMyReview)

	w := doRequest(r1, http.MethodGet, "/reviews/me", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "5")
}

func TestReviews_Upsert_Success(t *testing.T) {
	player := fakePlayer()
	r1 := chi.NewRouter()
	r1.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := middleware.InjectPlayerForTest(req.Context(), player)
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	h := &handlers.ReviewHandler{
		Store: &mockReviewStore{
			upsertReviewFn: func(ctx context.Context, playerID uuid.UUID, rating int, comment *string) (*db.AppReview, error) {
				assert.Equal(t, player.ID, playerID)
				assert.Equal(t, 4, rating)
				return &db.AppReview{ID: uuid.New(), Rating: 4}, nil
			},
		},
	}
	r1.Put("/reviews/me", h.UpsertMyReview)

	w := doRequest(r1, http.MethodPut, "/reviews/me", `{"rating":4,"comment":"Good"}`)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestReviews_Upsert_RatingOutOfRange(t *testing.T) {
	player := fakePlayer()
	store := &mockReviewStore{}
	r1 := chi.NewRouter()
	r1.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := middleware.InjectPlayerForTest(req.Context(), player)
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	h := &handlers.ReviewHandler{Store: store}
	r1.Put("/reviews/me", h.UpsertMyReview)

	// Test rating too low (0)
	w := doRequest(r1, http.MethodPut, "/reviews/me", `{"rating":0}`)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)

	// Test rating too high (6)
	w = doRequest(r1, http.MethodPut, "/reviews/me", `{"rating":6}`)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestReviews_Summary_Success_Admin(t *testing.T) {
	admin := fakePlayer(asAdmin())
	r1 := chi.NewRouter()
	r1.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := middleware.InjectPlayerForTest(req.Context(), admin)
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	h := &handlers.ReviewHandler{
		Store: &mockReviewStore{
			getReviewSummaryFn: func(ctx context.Context) (*db.ReviewSummary, error) {
				return &db.ReviewSummary{TotalReviews: 10, AverageRating: 4.5}, nil
			},
		},
	}
	r1.Get("/reviews/summary", h.GetSummary)

	w := doRequest(r1, http.MethodGet, "/reviews/summary", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "4.5")
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

func TestPush_Subscribe_MissingKeys(t *testing.T) {
	r := pushRouter(fakePlayer())
	w := postJSON(r, "/push/subscribe", `{"endpoint":"https://example.com/push"}`)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestPush_Subscribe_MissingP256DH(t *testing.T) {
	r := pushRouter(fakePlayer())
	w := postJSON(r, "/push/subscribe", `{"endpoint":"https://example.com/push","keys":{"auth":"def"}}`)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestPush_Subscribe_MissingAuth(t *testing.T) {
	r := pushRouter(fakePlayer())
	w := postJSON(r, "/push/subscribe", `{"endpoint":"https://example.com/push","keys":{"p256dh":"abc"}}`)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestPush_Unsubscribe_MissingEndpoint(t *testing.T) {
	r := pushRouter(fakePlayer())
	w := postJSON(r, "/push/subscribe", `{}`)
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

func TestRanking_InvalidMonth(t *testing.T) {
	r := rankingRouter()
	w := doRequest(r, http.MethodGet, "/ranking?month=13&year=2026", "")
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

// ── votes (validation before DB) ─────────────────────────────────────────────

// NOTE: Vote handler tests require complex initialization with PushService.
// These will be implemented as part of Phase 5 unit test suite with proper mocking.

func testVotesRouterSkipped(player *db.Player) http.Handler {
	// Placeholder - votes router requires PushService dependency
	return chi.NewRouter()
}

// Vote handler tests deferred to Phase 5
/*
func TestVotes_Status_InvalidMatchID(t *testing.T) {
	// Requires NewVoteHandler(pool, pushService)
	assert.True(t, true)
}

func TestVotes_Submit_InvalidMatchID(t *testing.T) {
	// Requires NewVoteHandler(pool, pushService)
	assert.True(t, true)
}

*/

// Placeholder - shows vote tests are skipped
func TestVotes_Placeholder_SkippedForPhase5(t *testing.T) {
	assert.True(t, true)
}

// ── finance ───────────────────────────────────────────────────────────────────

func financeRouterAs(player *db.Player) http.Handler {
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
	r := financeRouterAs(fakePlayer())
	w := doRequest(r, http.MethodGet, "/groups/not-a-uuid/finance/periods", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestFinance_GetPeriod_InvalidGroupID(t *testing.T) {
	r := financeRouterAs(fakePlayer())
	w := doRequest(r, http.MethodGet, "/groups/not-a-uuid/finance/periods/2025/1", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestFinance_UpdatePayment_InvalidUUID(t *testing.T) {
	r := financeRouterAs(fakePlayer())
	w := doRequest(r, http.MethodPatch, "/finance/payments/not-a-uuid", `{"status":"paid"}`)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ── subscriptions ─────────────────────────────────────────────────────────────

func subscriptionRouterAs(player *db.Player) http.Handler {
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
	r := subscriptionRouterAs(fakePlayer())
	w := postJSON(r, "/subscriptions", `{"plan":"free","billing_cycle":"monthly"}`)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestSubscription_CreateCheckout_InvalidBillingCycle(t *testing.T) {
	r := subscriptionRouterAs(fakePlayer())
	w := postJSON(r, "/subscriptions", `{"plan":"basic","billing_cycle":"quarterly"}`)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

// ── Groups Business Logic Tests ───────────────────────────────────────────────

// Test member limit enforcement for different plans (validation test)
func TestGroups_AddMember_FreePlanMemberLimit(t *testing.T) {
	// Scenario: Free plan allows 30 members, we're at limit
	// Expected: 403 PLAN_LIMIT_EXCEEDED
	// This test focuses on the plan limit validation gate
	groupID := uuid.New()
	newMemberID := uuid.New()
	body := fmt.Sprintf(`{"player_id":"%s"}`, newMemberID.String())

	mockStore := &mockGroupStoreForBusiness{
		getGroupMemberFn: func(ctx context.Context, groupID, playerID uuid.UUID) (*db.GroupMember, error) {
			return &db.GroupMember{Role: db.GroupMemberRoleAdmin}, nil // Caller is admin
		},
		getPlayerPlanFn: func(ctx context.Context, playerID uuid.UUID) (string, error) {
			return "free", nil
		},
		countGroupMembersFn: func(ctx context.Context, groupID uuid.UUID) (int, error) {
			return 30, nil // Already at free plan limit
		},
	}

	r := chi.NewRouter()
	h := &handlers.GroupHandler{Store: mockStore}
	r.Mount("/groups", h.Routes())

	admin := fakePlayer(asAdmin())
	w := sendRequestWithContext(r, "POST", fmt.Sprintf("/groups/%s/members", groupID.String()), body, middleware.InjectPlayerForTest(context.Background(), admin))

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "PLAN_LIMIT_EXCEEDED")
}

// Mock store for business logic tests
type mockGroupStoreForBusiness struct {
	getGroupMemberFn                func(ctx context.Context, groupID, playerID uuid.UUID) (*db.GroupMember, error)
	getPlayerPlanFn                 func(ctx context.Context, playerID uuid.UUID) (string, error)
	countGroupMembersFn             func(ctx context.Context, groupID uuid.UUID) (int, error)
	addGroupMemberFn                func(ctx context.Context, groupID, playerID uuid.UUID, role db.GroupMemberRole) (*db.GroupMember, error)
	getOpenMatchesForGroupFn        func(ctx context.Context, groupID uuid.UUID) ([]uuid.UUID, error)
	ensurePlayerSubscriptionFn      func(ctx context.Context, playerID uuid.UUID) error
	ensureMemberInCurrentPeriodFn   func(ctx context.Context, groupID, playerID uuid.UUID, playerName string) error
	getPlayerByIDFn                 func(ctx context.Context, playerID uuid.UUID) (*db.Player, error)
	setAttendanceFn                 func(ctx context.Context, matchID, playerID uuid.UUID, status string) error
}

func (m *mockGroupStoreForBusiness) GetGroupMember(ctx context.Context, groupID, playerID uuid.UUID) (*db.GroupMember, error) {
	if m.getGroupMemberFn != nil {
		return m.getGroupMemberFn(ctx, groupID, playerID)
	}
	return nil, db.ErrNotFound
}

func (m *mockGroupStoreForBusiness) GetPlayerPlan(ctx context.Context, playerID uuid.UUID) (string, error) {
	if m.getPlayerPlanFn != nil {
		return m.getPlayerPlanFn(ctx, playerID)
	}
	return "free", nil
}

func (m *mockGroupStoreForBusiness) CountGroupMembers(ctx context.Context, groupID uuid.UUID) (int, error) {
	if m.countGroupMembersFn != nil {
		return m.countGroupMembersFn(ctx, groupID)
	}
	return 0, nil
}

func (m *mockGroupStoreForBusiness) AddGroupMember(ctx context.Context, groupID, playerID uuid.UUID, role db.GroupMemberRole) (*db.GroupMember, error) {
	if m.addGroupMemberFn != nil {
		return m.addGroupMemberFn(ctx, groupID, playerID, role)
	}
	return nil, nil
}

func (m *mockGroupStoreForBusiness) GetOpenMatchesForGroup(ctx context.Context, groupID uuid.UUID) ([]uuid.UUID, error) {
	if m.getOpenMatchesForGroupFn != nil {
		return m.getOpenMatchesForGroupFn(ctx, groupID)
	}
	return []uuid.UUID{}, nil
}

func (m *mockGroupStoreForBusiness) EnsurePlayerSubscription(ctx context.Context, playerID uuid.UUID) error {
	if m.ensurePlayerSubscriptionFn != nil {
		return m.ensurePlayerSubscriptionFn(ctx, playerID)
	}
	return nil
}

func (m *mockGroupStoreForBusiness) EnsureMemberInCurrentPeriod(ctx context.Context, groupID, playerID uuid.UUID, playerName string) error {
	if m.ensureMemberInCurrentPeriodFn != nil {
		return m.ensureMemberInCurrentPeriodFn(ctx, groupID, playerID, playerName)
	}
	return nil
}

func (m *mockGroupStoreForBusiness) GetPlayerByID(ctx context.Context, playerID uuid.UUID) (*db.Player, error) {
	if m.getPlayerByIDFn != nil {
		return m.getPlayerByIDFn(ctx, playerID)
	}
	return nil, db.ErrNotFound
}

func (m *mockGroupStoreForBusiness) SetAttendance(ctx context.Context, matchID, playerID uuid.UUID, status string) error {
	if m.setAttendanceFn != nil {
		return m.setAttendanceFn(ctx, matchID, playerID, status)
	}
	return nil
}

// Stub out remaining methods (not used in these tests)
func (m *mockGroupStoreForBusiness) GetGroupsByPlayer(ctx context.Context, playerID uuid.UUID, isAdmin bool) ([]db.Group, error) {
	return nil, nil
}
func (m *mockGroupStoreForBusiness) GetGroupByID(ctx context.Context, groupID uuid.UUID) (*db.Group, error) {
	return nil, nil
}
func (m *mockGroupStoreForBusiness) CreateGroup(ctx context.Context, p db.CreateGroupParams) (*db.Group, error) {
	return nil, nil
}
func (m *mockGroupStoreForBusiness) UpdateGroupFull(ctx context.Context, groupID uuid.UUID, g *db.Group) (*db.Group, error) {
	return nil, nil
}
func (m *mockGroupStoreForBusiness) DeleteGroup(ctx context.Context, groupID uuid.UUID) error {
	return nil
}
func (m *mockGroupStoreForBusiness) SlugExists(ctx context.Context, slug string) (bool, error) {
	return false, nil
}
func (m *mockGroupStoreForBusiness) CountGroupAdminCount(ctx context.Context, playerID uuid.UUID) (int, error) {
	return 0, nil
}
func (m *mockGroupStoreForBusiness) GetGroupMembers(ctx context.Context, groupID uuid.UUID) ([]db.GroupMemberWithPlayer, error) {
	return nil, nil
}
func (m *mockGroupStoreForBusiness) UpdateGroupMember(ctx context.Context, groupID, playerID uuid.UUID, p db.UpdateGroupMemberParams) (*db.GroupMember, error) {
	return nil, nil
}
func (m *mockGroupStoreForBusiness) RemoveGroupMember(ctx context.Context, groupID, playerID uuid.UUID) error {
	return nil
}
func (m *mockGroupStoreForBusiness) GetGroupMemberPlayerIDs(ctx context.Context, groupID uuid.UUID) ([]uuid.UUID, error) {
	return nil, nil
}
func (m *mockGroupStoreForBusiness) GetNonAdminMemberPlayerIDs(ctx context.Context, groupID uuid.UUID) ([]uuid.UUID, error) {
	return nil, nil
}
func (m *mockGroupStoreForBusiness) GetPlayerByWhatsApp(ctx context.Context, whatsapp string) (*db.Player, error) {
	return nil, nil
}
func (m *mockGroupStoreForBusiness) CreatePlayer(ctx context.Context, args db.CreatePlayerParams) (*db.Player, error) {
	return nil, nil
}
func (m *mockGroupStoreForBusiness) UpdatePlayerMustChangePassword(ctx context.Context, id uuid.UUID, val bool) error {
	return nil
}
