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

	// Rate limiters
	loginRL := middleware.NewLoginRateLimiter()

	// Handlers
	authH   := handlers.NewAuthHandler(authSvc, loginRL)
	groupH  := handlers.NewGroupHandler(pool)
	matchH  := handlers.NewMatchHandler(pool)
	playerH := handlers.NewPlayerHandler(pool)
	inviteH := handlers.NewInviteHandler(pool)
	teamH   := handlers.NewTeamHandler(pool)

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

	r.Route("/api/v2", func(r chi.Router) {
		// ── Public routes (no auth required) ──────────────────────────
		r.Mount("/auth", authH.PublicRoutes())
		r.Get("/matches/discover", matchH.DiscoverMatches)
		r.Get("/matches/public/{hash}", matchH.GetPublicMatch)
		r.Get("/matches/public/{hash}/player-stats", matchH.GetPublicMatchStats)
		r.Mount("/invites", inviteH.PublicRoutes())
		r.Get("/matches/{matchID}/teams", teamH.GetTeams)

		// ── Authenticated routes (JWT + api_v2_enabled gate) ──────────
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(cfg.SecretKey, pool))
			r.Use(middleware.ApiV2Access)

			// Auth protected routes
			r.Mount("/auth", authH.ProtectedRoutes())

			// Domain routes
			r.Mount("/groups", groupH.Routes())
			r.Mount("/players", playerH.Routes())
			r.Mount("/invites", inviteH.AuthRoutes())

			// Match routes nested under groups
			r.Route("/groups/{groupID}/matches", func(r chi.Router) {
				r.Mount("/", matchH.GroupMatchRoutes())
			})

			// Match routes at top level (player-stats write, team draw)
			r.Put("/matches/{hash}/player-stats", matchH.UpsertPlayerStats)
			r.Post("/matches/{matchID}/teams", teamH.DrawTeams)

			// Rankings, push, reviews, mcp-tokens — Phase 3
		})

		// ── Admin-only routes ──────────────────────────────────────────
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(cfg.SecretKey, pool))
			r.Use(middleware.RequireAdmin)

			// TODO Phase 4: r.Mount("/admin", adminH.Routes())
		})
	})

	return r
}
