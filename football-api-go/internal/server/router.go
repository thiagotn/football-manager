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
	authH := handlers.NewAuthHandler(authSvc, loginRL)

	r := chi.NewRouter()

	// Global middlewares
	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.RealIP)
	r.Use(chiMiddleware.Logger)
	r.Use(middleware.Recovery)
	r.Use(middleware.CORS(cfg.CORSOriginsList()))

	// Health check — no auth, no api_v2_enabled gate
	r.Get("/api/v2/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	r.Route("/api/v2", func(r chi.Router) {
		// Public auth routes (login, register, OTP, refresh)
		r.Mount("/auth", authH.PublicRoutes())

		// Authenticated routes (require valid JWT + api_v2_enabled gate)
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(cfg.SecretKey, pool))
			r.Use(middleware.ApiV2Access)

			// Protected auth routes (/me, change-password, etc.)
			r.Mount("/auth", authH.ProtectedRoutes())

			// TODO: mount remaining domain handlers in Phase 2+
			// r.Mount("/groups",     groupH.AuthRoutes())
			// r.Mount("/matches",    matchH.AuthRoutes())
			// r.Mount("/players",    playerH.AuthRoutes())
			// r.Mount("/votes",      voteH.Routes())
			// r.Mount("/finance",    financeH.Routes())
			// r.Mount("/chat",       chatH.Routes())
			// r.Mount("/mcp-tokens", mcpTokenH.Routes())
		})

		// Super-admin routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(cfg.SecretKey, pool))
			r.Use(middleware.RequireAdmin)

			// TODO: r.Mount("/admin", adminH.Routes())
		})
	})

	return r
}
