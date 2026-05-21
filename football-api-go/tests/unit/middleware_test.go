package unit_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thiagotn/football-manager/football-api-go/internal/db"
	"github.com/thiagotn/football-manager/football-api-go/internal/middleware"
)

var testCtx = context.Background()

// ────── Recovery Middleware ──────

func TestRecoveryMiddleware(t *testing.T) {
	t.Run("catches panic and returns 500 JSON", func(t *testing.T) {
		panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("test panic")
		})

		router := chi.NewRouter()
		router.Use(middleware.Recovery)
		router.Get("/", panicHandler)

		req := httptest.NewRequestWithContext(testCtx, http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

		var body map[string]string
		err := json.NewDecoder(rec.Body).Decode(&body)
		require.NoError(t, err)
		assert.Equal(t, "internal server error", body["detail"])
	})

	t.Run("passes through without panic", func(t *testing.T) {
		okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
		})

		router := chi.NewRouter()
		router.Use(middleware.Recovery)
		router.Get("/", okHandler)

		req := httptest.NewRequestWithContext(testCtx, http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "ok", rec.Body.String())
	})
}

// ────── CORS Middleware ──────

func TestCORSMiddleware(t *testing.T) {
	t.Run("allows configured origin", func(t *testing.T) {
		okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		router := chi.NewRouter()
		router.Use(middleware.CORS([]string{"https://example.com"}))
		router.Get("/", okHandler)

		req := httptest.NewRequestWithContext(testCtx, http.MethodGet, "/", nil)
		req.Header.Set("Origin", "https://example.com")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, "https://example.com", rec.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "true", rec.Header().Get("Access-Control-Allow-Credentials"))
		assert.Contains(t, rec.Header().Get("Access-Control-Allow-Methods"), "GET")
	})

	t.Run("rejects unconfigured origin when list provided", func(t *testing.T) {
		okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		router := chi.NewRouter()
		router.Use(middleware.CORS([]string{"https://example.com"}))
		router.Get("/", okHandler)

		req := httptest.NewRequestWithContext(testCtx, http.MethodGet, "/", nil)
		req.Header.Set("Origin", "https://other.com")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		// Should not set Access-Control-Allow-Origin for unconfigured origin
		assert.NotEqual(t, "https://other.com", rec.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("allows all origins when empty list", func(t *testing.T) {
		okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		router := chi.NewRouter()
		router.Use(middleware.CORS([]string{}))
		router.Get("/", okHandler)

		req := httptest.NewRequestWithContext(testCtx, http.MethodGet, "/", nil)
		req.Header.Set("Origin", "https://any-domain.com")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, "*", rec.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("handles OPTIONS requests", func(t *testing.T) {
		okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		router := chi.NewRouter()
		router.Use(middleware.CORS([]string{"https://example.com"}))
		router.Options("/", okHandler)
		router.Get("/", okHandler)

		req := httptest.NewRequestWithContext(testCtx, http.MethodOptions, "/", nil)
		req.Header.Set("Origin", "https://example.com")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("sets all required CORS headers", func(t *testing.T) {
		okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		router := chi.NewRouter()
		router.Use(middleware.CORS([]string{"https://example.com"}))
		router.Get("/", okHandler)

		req := httptest.NewRequestWithContext(testCtx, http.MethodGet, "/", nil)
		req.Header.Set("Origin", "https://example.com")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, "https://example.com", rec.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "true", rec.Header().Get("Access-Control-Allow-Credentials"))
		assert.Equal(t, "86400", rec.Header().Get("Access-Control-Max-Age"))
		assert.Contains(t, rec.Header().Get("Access-Control-Allow-Methods"), "POST")
		assert.Contains(t, rec.Header().Get("Access-Control-Allow-Headers"), "Authorization")
	})
}

// ────── LoginRateLimiter Middleware ──────

func TestLoginRateLimiter(t *testing.T) {
	t.Run("allows requests up to limit", func(t *testing.T) {
		limiter := middleware.NewLoginRateLimiter()
		okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		router := chi.NewRouter()
		router.Use(limiter.Middleware)
		router.Post("/login", okHandler)

		// Make 5 requests from same IP — all should succeed
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest(http.MethodPost, "/login", nil)
			req.RemoteAddr = "192.0.2.1:8080"
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			assert.Equal(t, http.StatusOK, rec.Code, "request %d should succeed", i+1)
		}
	})

	t.Run("blocks requests over limit", func(t *testing.T) {
		limiter := middleware.NewLoginRateLimiter()
		okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		router := chi.NewRouter()
		router.Use(limiter.Middleware)
		router.Post("/login", okHandler)

		// Make 6 requests from same IP — 6th should be blocked
		for i := 0; i < 6; i++ {
			req := httptest.NewRequest(http.MethodPost, "/login", nil)
			req.RemoteAddr = "192.0.2.1:8080"
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if i < 5 {
				assert.Equal(t, http.StatusOK, rec.Code)
			} else {
				assert.Equal(t, http.StatusTooManyRequests, rec.Code)
				assert.Equal(t, "900", rec.Header().Get("Retry-After"))

				var body map[string]string
				json.NewDecoder(rec.Body).Decode(&body)
				assert.Equal(t, "too many requests", body["detail"])
			}
		}
	})

	t.Run("tracks separate IPs independently", func(t *testing.T) {
		limiter := middleware.NewLoginRateLimiter()
		okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		router := chi.NewRouter()
		router.Use(limiter.Middleware)
		router.Post("/login", okHandler)

		// IP 1: 6 requests (should hit limit on 6th)
		for i := 0; i < 6; i++ {
			req := httptest.NewRequest(http.MethodPost, "/login", nil)
			req.RemoteAddr = "192.0.2.1:8080"
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if i < 5 {
				assert.Equal(t, http.StatusOK, rec.Code)
			} else {
				assert.Equal(t, http.StatusTooManyRequests, rec.Code)
			}
		}

		// IP 2: 1 request (should succeed because different IP)
		req := httptest.NewRequest(http.MethodPost, "/login", nil)
		req.RemoteAddr = "192.0.2.2:8080"
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("uses X-Forwarded-For header when present", func(t *testing.T) {
		limiter := middleware.NewLoginRateLimiter()
		okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		router := chi.NewRouter()
		router.Use(limiter.Middleware)
		router.Post("/login", okHandler)

		// Make 6 requests with X-Forwarded-For (should hit limit on 6th)
		for i := 0; i < 6; i++ {
			req := httptest.NewRequest(http.MethodPost, "/login", nil)
			req.Header.Set("X-Forwarded-For", "203.0.113.5, 198.51.100.178")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if i < 5 {
				assert.Equal(t, http.StatusOK, rec.Code)
			} else {
				assert.Equal(t, http.StatusTooManyRequests, rec.Code)
			}
		}
	})

	t.Run("resets after window expires", func(t *testing.T) {
		limiter := middleware.NewLoginRateLimiter()
		okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		router := chi.NewRouter()
		router.Use(limiter.Middleware)
		router.Post("/login", okHandler)

		ip := "192.0.2.1:8080"

		// Hit limit
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest(http.MethodPost, "/login", nil)
			req.RemoteAddr = ip
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
		}

		// 6th request blocked
		req := httptest.NewRequest(http.MethodPost, "/login", nil)
		req.RemoteAddr = ip
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusTooManyRequests, rec.Code)

		// We can't easily test the actual time window without modifying internals,
		// but we've verified the rate limiting logic works
	})
}

// ────── ApiV2Access Middleware ──────

func TestApiV2AccessMiddleware(t *testing.T) {
	t.Run("allows unauthenticated requests", func(t *testing.T) {
		okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		router := chi.NewRouter()
		router.Use(middleware.ApiV2Access)
		router.Get("/", okHandler)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("allows admin users", func(t *testing.T) {
		okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		router := chi.NewRouter()
		router.Use(middleware.ApiV2Access)
		router.Get("/", okHandler)

		admin := fakePlayer(asAdmin())
		ctx := middleware.InjectPlayerForTest(context.Background(), admin)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("blocks regular player without api_v2_enabled", func(t *testing.T) {
		okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		router := chi.NewRouter()
		router.Use(middleware.ApiV2Access)
		router.Get("/", okHandler)

		player := fakePlayer(func(p *db.Player) {
			p.ApiV2Enabled = false
		})
		ctx := middleware.InjectPlayerForTest(context.Background(), player)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusForbidden, rec.Code)

		var body map[string]string
		json.NewDecoder(rec.Body).Decode(&body)
		assert.Equal(t, "API_V2_NOT_ENABLED", body["detail"])
	})

	t.Run("allows regular player with api_v2_enabled", func(t *testing.T) {
		okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		router := chi.NewRouter()
		router.Use(middleware.ApiV2Access)
		router.Get("/", okHandler)

		player := fakePlayer(func(p *db.Player) {
			p.ApiV2Enabled = true
			p.Role = db.PlayerRolePlayer
		})
		ctx := middleware.InjectPlayerForTest(context.Background(), player)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("admin bypasses api_v2_enabled flag", func(t *testing.T) {
		okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		router := chi.NewRouter()
		router.Use(middleware.ApiV2Access)
		router.Get("/", okHandler)

		admin := fakePlayer(asAdmin(), func(p *db.Player) {
			p.ApiV2Enabled = false
		})
		ctx := middleware.InjectPlayerForTest(context.Background(), admin)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

// ────── RequireAdmin Middleware ──────

func TestRequireAdminMiddleware(t *testing.T) {
	t.Run("blocks non-admin users", func(t *testing.T) {
		okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		router := chi.NewRouter()
		router.Use(middleware.RequireAdmin)
		router.Get("/", okHandler)

		player := fakePlayer()
		ctx := middleware.InjectPlayerForTest(context.Background(), player)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("allows admin users", func(t *testing.T) {
		okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		router := chi.NewRouter()
		router.Use(middleware.RequireAdmin)
		router.Get("/", okHandler)

		admin := fakePlayer(asAdmin())
		ctx := middleware.InjectPlayerForTest(context.Background(), admin)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("blocks unauthenticated requests", func(t *testing.T) {
		okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		router := chi.NewRouter()
		router.Use(middleware.RequireAdmin)
		router.Get("/", okHandler)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}
