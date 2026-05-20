package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thiagotn/football-manager/football-api-go/internal/apierror"
	"github.com/thiagotn/football-manager/football-api-go/internal/db"
)

type contextKey string

const playerKey contextKey = "player"

// Auth validates a JWT or MCP token and stores the authenticated player in the request context.
func Auth(secretKey string, pool *pgxpool.Pool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearerToken(r)
			if token == "" {
				writeUnauthorized(w)
				return
			}

			var player *db.Player
			var err error

			if strings.HasPrefix(token, "rachao_") {
				hash := db.HashToken(token)
				player, err = db.GetPlayerByMCPToken(r.Context(), pool, hash)
			} else {
				player, err = validateJWT(r.Context(), pool, token, secretKey)
			}

			if err != nil {
				writeUnauthorized(w)
				return
			}

			ctx := context.WithValue(r.Context(), playerKey, player)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalAuth tries to authenticate but allows unauthenticated requests through.
func OptionalAuth(secretKey string, pool *pgxpool.Pool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearerToken(r)
			if token != "" {
				var player *db.Player
				var err error
				if strings.HasPrefix(token, "rachao_") {
					hash := db.HashToken(token)
					player, err = db.GetPlayerByMCPToken(r.Context(), pool, hash)
				} else {
					player, err = validateJWT(r.Context(), pool, token, secretKey)
				}
				if err == nil && player != nil {
					ctx := context.WithValue(r.Context(), playerKey, player)
					r = r.WithContext(ctx)
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireAdmin blocks the request unless the authenticated player has role=admin.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		player := PlayerFromCtx(r.Context())
		if player == nil || player.Role != db.PlayerRoleAdmin {
			writeJSON(w, http.StatusForbidden, apierror.Forbidden("admin access required"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// PlayerFromCtx retrieves the authenticated player from the request context (may be nil).
func PlayerFromCtx(ctx context.Context) *db.Player {
	p, _ := ctx.Value(playerKey).(*db.Player)
	return p
}

// InjectPlayerForTest injects a player directly into the context (test use only).
func InjectPlayerForTest(ctx context.Context, player *db.Player) context.Context {
	return context.WithValue(ctx, playerKey, player)
}

// --- private helpers --------------------------------------------------------

type jwtClaims struct {
	jwt.RegisteredClaims
}

func validateJWT(ctx context.Context, pool *pgxpool.Pool, tokenStr, secretKey string) (*db.Player, error) {
	tok, err := jwt.ParseWithClaims(tokenStr, &jwtClaims{},
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, apierror.Unauthorized()
			}
			return []byte(secretKey), nil
		},
	)
	if err != nil || !tok.Valid {
		return nil, apierror.Unauthorized()
	}

	claims, ok := tok.Claims.(*jwtClaims)
	if !ok {
		return nil, apierror.Unauthorized()
	}

	sub, err := claims.GetSubject()
	if err != nil || sub == "" || strings.HasPrefix(sub, "otp:") {
		return nil, apierror.Unauthorized()
	}

	playerID, err := uuid.Parse(sub)
	if err != nil {
		return nil, apierror.Unauthorized()
	}

	return db.GetPlayerByID(ctx, pool, playerID)
}

func extractBearerToken(r *http.Request) string {
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	writeJSON(w, http.StatusUnauthorized, map[string]string{"detail": "not authenticated"})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
