package server

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thiagotn/football-manager/football-api-go/internal/config"
	"github.com/thiagotn/football-manager/football-api-go/internal/handlers"
	"github.com/thiagotn/football-manager/football-api-go/internal/middleware"
	"github.com/thiagotn/football-manager/football-api-go/internal/services"
)

func NewRouter(cfg *config.Config, pool *pgxpool.Pool) http.Handler {
	// Services
	authSvc := services.NewAuthService(pool, cfg)

	var stripeSvc *services.StripeService
	if cfg.StripeSecretKey != "" && cfg.StripeWebhookSecret != "" {
		stripeSvc = services.NewStripeService(
			cfg.StripeSecretKey,
			cfg.StripeWebhookSecret,
			cfg.StripePriceBasicMonthly,
			cfg.StripePriceBasicYearly,
			cfg.StripePriceProMonthly,
			cfg.StripePriceProYearly,
			cfg.FrontendURL,
		)
	}

	var storageSvc *services.StorageService
	if cfg.SupabaseURL != "" && cfg.SupabaseServiceRoleKey != "" {
		storageSvc = services.NewStorageService(cfg.SupabaseURL, cfg.SupabaseServiceRoleKey)
	}

	// Rate limiters
	loginRL := middleware.NewLoginRateLimiter()

	// Handlers
	authH         := handlers.NewAuthHandler(authSvc, loginRL)
	groupH        := handlers.NewGroupHandler(pool)
	matchH        := handlers.NewMatchHandler(pool)
	playerH       := handlers.NewPlayerHandler(pool, storageSvc)
	inviteH       := handlers.NewInviteHandler(pool)
	teamH         := handlers.NewTeamHandler(pool)
	voteH         := handlers.NewVoteHandler(pool)
	financeH      := handlers.NewFinanceHandler(pool)
	subscriptionH := handlers.NewSubscriptionHandler(pool, stripeSvc)
	webhookH      := handlers.NewWebhookHandler(pool, stripeSvc)
	pushH         := handlers.NewPushHandler(pool, cfg.VAPIDPublicKey)
	rankingH      := handlers.NewRankingHandler(pool)
	reviewH       := handlers.NewReviewHandler(pool)
	mcpTokenH     := handlers.NewMCPTokenHandler(pool)
	betaH         := handlers.NewBetaHandler(pool)
	adminH        := handlers.NewAdminHandler(pool, stripeSvc, storageSvc)
	chatH         := handlers.NewChatHandler(pool, cfg.AnthropicAPIKey, cfg.LLMModel, cfg.ChatRateLimit)

	r := chi.NewRouter()

	// Global middlewares
	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.RealIP)
	r.Use(chiMiddleware.Logger)
	r.Use(middleware.Recovery)
	r.Use(middleware.CORS(cfg.CORSOriginsList()))

	// Health check — no auth, exempt from api_v2_enabled gate
	r.Get("/api/v2/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	authMw  := middleware.Auth(cfg.SecretKey, pool)
	apiV2Mw := middleware.ApiV2Access

	r.Route("/api/v2", func(r chi.Router) {
		// ── Auth routes (public + protected combined — avoids chi double-mount) ──
		r.Mount("/auth", authH.Routes(authMw, apiV2Mw))

		// ── Other public routes (no auth required) ────────────────────
		r.Get("/matches/discover", matchH.DiscoverMatches)
		r.Get("/matches/public/{hash}", matchH.GetPublicMatch)
		r.Get("/matches/public/{hash}/player-stats", matchH.GetPublicMatchStats)
		r.Get("/matches/public/{hash}/votes/results", voteH.GetPublicVoteResults)
		r.Get("/matches/public/{hash}/votes/ballots", voteH.GetPublicVoteBallots)
		r.Mount("/invites", inviteH.PublicRoutes())
		r.Get("/matches/{matchID}/teams", teamH.GetTeams)
		r.Get("/ranking", rankingH.GetRanking)
		r.Get("/push/vapid-public-key", pushH.GetVapidKey)

		// Webhooks — raw body needed, no auth
		r.Post("/webhooks/payment", webhookH.HandleStripeWebhook)

		// Beta signup — optional auth
		r.Post("/beta/android-signup", betaH.AndroidSignup)

		// ── Authenticated routes (JWT + api_v2_enabled gate) ──────────
		r.Group(func(r chi.Router) {
			r.Use(authMw)
			r.Use(apiV2Mw)

			// (auth routes already mounted above via authH.Routes)

			// Domain routes
			r.Mount("/groups", groupH.Routes())
			r.Mount("/players", playerH.Routes())
			r.Mount("/invites", inviteH.AuthRoutes())
			r.Mount("/mcp-tokens", mcpTokenH.Routes())

			// Match routes nested under groups
			r.Route("/groups/{groupID}/matches", func(r chi.Router) {
				r.Mount("/", matchH.GroupMatchRoutes())
			})

			// Finance routes
			r.Get("/groups/{groupID}/finance/periods", financeH.ListPeriods)
			r.Get("/groups/{groupID}/finance/periods/{year}/{month}", financeH.GetPeriod)
			r.Patch("/finance/payments/{paymentID}", financeH.UpdatePayment)

			// Match-level routes
			r.Put("/matches/{hash}/player-stats", matchH.UpsertPlayerStats)
			r.Post("/matches/{matchID}/teams", teamH.DrawTeams)

			// Vote routes
			r.Get("/matches/{matchID}/votes/status", voteH.GetVoteStatus)
			r.Post("/matches/{matchID}/votes", voteH.SubmitVote)
			r.Post("/matches/{matchID}/votes/close", voteH.CloseVoting)
			r.Get("/matches/{matchID}/votes/results", voteH.GetVoteResults)
			r.Get("/votes/pending", voteH.GetPendingVotes)

			// Subscription routes
			r.Get("/subscriptions/me", subscriptionH.GetMySubscription)
			r.Post("/subscriptions", subscriptionH.CreateCheckoutSession)

			// Push notification routes
			r.Post("/push/subscribe", pushH.Subscribe)
			r.Delete("/push/subscribe", pushH.Unsubscribe)

			// Review routes
			r.Get("/reviews/me", reviewH.GetMyReview)
			r.Put("/reviews/me", reviewH.UpsertMyReview)
			r.Get("/reviews/summary", reviewH.GetSummary)
			r.Get("/reviews", reviewH.ListReviews)

			// Chat
			r.Post("/chat", chatH.Chat)
		})

		// ── Admin-only routes ──────────────────────────────────────────
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(cfg.SecretKey, pool))
			r.Use(middleware.RequireAdmin)

			r.Mount("/admin", adminH.Routes())
			r.Get("/admin/chat-users", chatH.ListChatUsers)
			r.Patch("/admin/chat-users/{userID}", chatH.UpdateChatAccess)
		})
	})

	return r
}
