package middleware

import (
	"net/http"

	"github.com/thiagotn/football-manager/football-api-go/internal/apierror"
	"github.com/thiagotn/football-manager/football-api-go/internal/db"
)

// ApiV2Access enforces per-user access control for the /api/v2 router.
// - Unauthenticated requests pass through (public endpoints).
// - Admins always pass through.
// - Regular players are blocked unless api_v2_enabled = true.
// Backwards-compatible: use ApiV2AccessFor to opt into env-aware bypass.
func ApiV2Access(next http.Handler) http.Handler {
	return apiV2AccessImpl(next, false)
}

// ApiV2AccessFor returns the access middleware with env-aware bypass.
// When devBypass=true, the api_v2_enabled flag is ignored — every authenticated
// player passes through. Intended for APP_ENV=development.
func ApiV2AccessFor(devBypass bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return apiV2AccessImpl(next, devBypass)
	}
}

func apiV2AccessImpl(next http.Handler, devBypass bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		player := PlayerFromCtx(r.Context())
		if player == nil {
			next.ServeHTTP(w, r)
			return
		}
		if player.Role == db.PlayerRoleAdmin {
			next.ServeHTTP(w, r)
			return
		}
		if devBypass {
			next.ServeHTTP(w, r)
			return
		}
		if !player.ApiV2Enabled {
			writeJSON(w, http.StatusForbidden, apierror.APIV2NotEnabled())
			return
		}
		next.ServeHTTP(w, r)
	})
}
