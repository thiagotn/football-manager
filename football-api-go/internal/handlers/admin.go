package handlers

import (
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thiagotn/football-manager/football-api-go/internal/apierror"
	"github.com/thiagotn/football-manager/football-api-go/internal/db"
	"github.com/thiagotn/football-manager/football-api-go/internal/services"
)

// mrrCents maps (plan, billing_cycle) → monthly revenue in centavos.
var mrrCents = map[[2]string]int{
	{"basic", "monthly"}: 1990,
	{"basic", "yearly"}:  199_00 / 12,
	{"pro", "monthly"}:   3990,
	{"pro", "yearly"}:    399_00 / 12,
}

type adminHandler struct {
	pool    *pgxpool.Pool
	stripe  *services.StripeService
	storage *services.StorageService
}

func NewAdminHandler(pool *pgxpool.Pool, stripe *services.StripeService, storage *services.StorageService) *adminHandler {
	return &adminHandler{pool: pool, stripe: stripe, storage: storage}
}

func (h *adminHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/stats", h.getStats)
	r.Get("/matches", h.listMatches)
	r.Get("/groups", h.listGroups)
	r.Get("/subscriptions/summary", h.subscriptionSummary)
	r.Get("/subscriptions", h.listSubscriptions)
	r.Patch("/subscriptions/{playerID}", h.updateSubscription)
	r.Post("/subscriptions/{playerID}/cancel", h.cancelSubscription)
	r.Get("/players", h.listPlayers)
	r.Delete("/players/{playerID}/avatar", h.deletePlayerAvatar)
	r.Get("/beta-signups", h.listBetaSignups)
	r.Get("/api-v2-users", h.listApiV2Users)
	r.Patch("/api-v2-users/{playerID}", h.toggleApiV2User)
	return r
}

// ── Request types ─────────────────────────────────────────────────────────────

type updateSubscriptionReq struct {
	Plan         string  `json:"plan"`
	Status       string  `json:"status"`
	BillingCycle *string `json:"billing_cycle"`
	Reason       *string `json:"reason"`
}

type toggleApiV2Req struct {
	ApiV2Enabled bool `json:"api_v2_enabled"`
}

type toggleChatReq struct {
	ChatEnabled bool `json:"chat_enabled"`
}

// ── Pagination helpers ────────────────────────────────────────────────────────

func pageParams(r *http.Request, defaultSize int) (page, pageSize, limit, offset int) {
	page = 1
	pageSize = defaultSize
	if v := r.URL.Query().Get("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 {
			page = n
		}
	}
	if v := r.URL.Query().Get("page_size"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 && n <= 100 {
			pageSize = n
		}
	}
	limit = pageSize
	offset = (page - 1) * pageSize
	return
}

func adminTargetPlayerID(r *http.Request) (uuid.UUID, error) {
	return uuid.Parse(chi.URLParam(r, "playerID"))
}

// ── Handlers ──────────────────────────────────────────────────────────────────

func (h *adminHandler) getStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var (
		totalMatches, totalGroups, totalPlayers int
		platformMinutes                         int
		signupsTotal, signups7d, signups30d     int
		totalReviews                            int
	)

	row := h.pool.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*)::INT FROM matches)  AS total_matches,
			(SELECT COUNT(*)::INT FROM groups)   AS total_groups,
			(SELECT COUNT(*)::INT FROM players WHERE role != 'admin') AS total_players,
			(SELECT COALESCE(SUM(GREATEST(0,
				EXTRACT(EPOCH FROM (end_time::INTERVAL - start_time::INTERVAL)) / 60
			)),0)::INT
			 FROM matches WHERE status = 'closed' AND end_time IS NOT NULL) AS platform_minutes,
			(SELECT COUNT(*)::INT FROM players WHERE role != 'admin') AS signups_total,
			(SELECT COUNT(*)::INT FROM players WHERE role != 'admin'
			   AND created_at >= NOW() - INTERVAL '7 days')  AS signups_7d,
			(SELECT COUNT(*)::INT FROM players WHERE role != 'admin'
			   AND created_at >= NOW() - INTERVAL '30 days') AS signups_30d,
			(SELECT COUNT(*)::INT FROM app_reviews) AS total_reviews`)
	if err := row.Scan(
		&totalMatches, &totalGroups, &totalPlayers,
		&platformMinutes,
		&signupsTotal, &signups7d, &signups30d,
		&totalReviews,
	); err != nil {
		renderError(w, err)
		return
	}

	renderJSON(w, http.StatusOK, map[string]any{
		"total_matches":          totalMatches,
		"total_groups":           totalGroups,
		"total_players":          totalPlayers,
		"platform_minutes_played": platformMinutes,
		"signups_total":          signupsTotal,
		"signups_last_7_days":    signups7d,
		"signups_last_30_days":   signups30d,
		"total_reviews":          totalReviews,
	})
}

func (h *adminHandler) listMatches(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	status := r.URL.Query().Get("status")
	limit := 50
	offset := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 && n <= 200 {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	var total int
	var countErr error
	if status != "" {
		countErr = h.pool.QueryRow(ctx,
			`SELECT COUNT(*)::INT FROM matches WHERE status=$1`, status).Scan(&total)
	} else {
		countErr = h.pool.QueryRow(ctx, `SELECT COUNT(*)::INT FROM matches`).Scan(&total)
	}
	if countErr != nil {
		renderError(w, countErr)
		return
	}

	var rows interface{ Close() }
	var queryErr error
	if status != "" {
		rows2, err := h.pool.Query(ctx, `
			SELECT m.id, m.hash, m.number, m.group_id, g.name AS group_name,
			       m.match_date::TEXT, m.start_time::TEXT, m.end_time::TEXT,
			       m.location, m.status::TEXT
			FROM matches m JOIN groups g ON g.id = m.group_id
			WHERE m.status = $1
			ORDER BY m.match_date DESC, m.start_time DESC
			LIMIT $2 OFFSET $3`, status, limit, offset)
		rows, queryErr = rows2, err
		if queryErr == nil {
			defer rows2.Close()
			items := make([]map[string]any, 0)
			for rows2.Next() {
				var id, groupID uuid.UUID
				var hash, groupName, matchDate, startTime, location, st string
				var number int
				var endTime *string
				if err := rows2.Scan(&id, &hash, &number, &groupID, &groupName,
					&matchDate, &startTime, &endTime, &location, &st); err != nil {
					renderError(w, err)
					return
				}
				items = append(items, map[string]any{
					"id": id, "hash": hash, "number": number,
					"group_id": groupID, "group_name": groupName,
					"match_date": matchDate, "start_time": startTime, "end_time": endTime,
					"location": location, "status": st,
				})
			}
			renderJSON(w, http.StatusOK, map[string]any{"total": total, "items": items})
			return
		}
	} else {
		rows2, err := h.pool.Query(ctx, `
			SELECT m.id, m.hash, m.number, m.group_id, g.name AS group_name,
			       m.match_date::TEXT, m.start_time::TEXT, m.end_time::TEXT,
			       m.location, m.status::TEXT
			FROM matches m JOIN groups g ON g.id = m.group_id
			ORDER BY m.match_date DESC, m.start_time DESC
			LIMIT $1 OFFSET $2`, limit, offset)
		rows, queryErr = rows2, err
		if queryErr == nil {
			defer rows2.Close()
			items := make([]map[string]any, 0)
			for rows2.Next() {
				var id, groupID uuid.UUID
				var hash, groupName, matchDate, startTime, location, st string
				var number int
				var endTime *string
				if err := rows2.Scan(&id, &hash, &number, &groupID, &groupName,
					&matchDate, &startTime, &endTime, &location, &st); err != nil {
					renderError(w, err)
					return
				}
				items = append(items, map[string]any{
					"id": id, "hash": hash, "number": number,
					"group_id": groupID, "group_name": groupName,
					"match_date": matchDate, "start_time": startTime, "end_time": endTime,
					"location": location, "status": st,
				})
			}
			renderJSON(w, http.StatusOK, map[string]any{"total": total, "items": items})
			return
		}
	}
	if queryErr != nil {
		renderError(w, queryErr)
	}
	_ = rows
}

func (h *adminHandler) listGroups(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_, _, limit, offset := pageParams(r, 50)

	var total int
	if err := h.pool.QueryRow(ctx, `SELECT COUNT(*)::INT FROM groups`).Scan(&total); err != nil {
		renderError(w, err)
		return
	}

	rows, err := h.pool.Query(ctx, `
		SELECT g.id, g.name, g.description, g.slug, g.created_at,
		       COUNT(DISTINCT gm.player_id)::INT AS total_members,
		       COUNT(DISTINCT m.id)::INT          AS total_matches
		FROM groups g
		LEFT JOIN group_members gm ON gm.group_id = g.id
		LEFT JOIN matches m        ON m.group_id  = g.id
		GROUP BY g.id
		ORDER BY g.created_at DESC
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		renderError(w, err)
		return
	}
	defer rows.Close()

	items := make([]map[string]any, 0)
	for rows.Next() {
		var id uuid.UUID
		var name, slug string
		var description *string
		var createdAt interface{}
		var members, matches int
		if err := rows.Scan(&id, &name, &description, &slug, &createdAt, &members, &matches); err != nil {
			renderError(w, err)
			return
		}
		items = append(items, map[string]any{
			"id": id, "name": name, "description": description, "slug": slug,
			"created_at": createdAt, "total_members": members, "total_matches": matches,
		})
	}
	renderJSON(w, http.StatusOK, map[string]any{"total": total, "items": items})
}

func (h *adminHandler) subscriptionSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var totalPlayers int
	if err := h.pool.QueryRow(ctx,
		`SELECT COUNT(*)::INT FROM players WHERE role != 'admin'`).Scan(&totalPlayers); err != nil {
		renderError(w, err)
		return
	}

	rows, err := h.pool.Query(ctx, `
		SELECT
			COALESCE(ps.status, 'active')        AS status,
			COALESCE(ps.plan, 'free')            AS plan,
			COALESCE(ps.billing_cycle, 'monthly') AS billing_cycle,
			COUNT(*)::INT AS cnt
		FROM players p
		LEFT JOIN player_subscriptions ps ON ps.player_id = p.id
		WHERE p.role != 'admin'
		GROUP BY ps.status, ps.plan, ps.billing_cycle`)
	if err != nil {
		renderError(w, err)
		return
	}
	defer rows.Close()

	active, free, pastDue, canceled, mrrCentsTotal := 0, 0, 0, 0, 0
	type breakdownKey struct{ plan, cycle string }
	breakdownMap := map[breakdownKey]int{}

	for rows.Next() {
		var status, plan, cycle string
		var cnt int
		if err := rows.Scan(&status, &plan, &cycle, &cnt); err != nil {
			renderError(w, err)
			return
		}
		switch {
		case plan == "free":
			free += cnt
		case status == "canceled":
			canceled += cnt
		case status == "past_due":
			pastDue += cnt
		case status == "active":
			active += cnt
			if price, ok := mrrCents[[2]string{plan, cycle}]; ok {
				mrrCentsTotal += price * cnt
			}
		default:
			free += cnt
		}
		if plan != "free" && status != "canceled" {
			k := breakdownKey{plan, cycle}
			breakdownMap[k] += cnt
		}
	}

	breakdown := make([]map[string]any, 0, len(breakdownMap))
	for k, cnt := range breakdownMap {
		breakdown = append(breakdown, map[string]any{
			"plan": k.plan, "billing_cycle": k.cycle, "count": cnt,
		})
	}

	renderJSON(w, http.StatusOK, map[string]any{
		"total_players": totalPlayers,
		"active":        active,
		"free":          free,
		"past_due":      pastDue,
		"canceled":      canceled,
		"mrr_cents":     mrrCentsTotal,
		"breakdown":     breakdown,
	})
}

func (h *adminHandler) listSubscriptions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	page, pageSize, limit, offset := pageParams(r, 20)
	filterStatus := r.URL.Query().Get("status")
	filterPlan := r.URL.Query().Get("plan")

	var total int
	var err error
	if filterStatus != "" && filterPlan != "" {
		err = h.pool.QueryRow(ctx, `
			SELECT COUNT(*)::INT
			FROM player_subscriptions ps JOIN players p ON p.id = ps.player_id
			WHERE p.role != 'admin' AND ps.status=$1 AND ps.plan=$2`, filterStatus, filterPlan).Scan(&total)
	} else if filterStatus != "" {
		err = h.pool.QueryRow(ctx, `
			SELECT COUNT(*)::INT
			FROM player_subscriptions ps JOIN players p ON p.id = ps.player_id
			WHERE p.role != 'admin' AND ps.status=$1`, filterStatus).Scan(&total)
	} else if filterPlan != "" {
		err = h.pool.QueryRow(ctx, `
			SELECT COUNT(*)::INT
			FROM player_subscriptions ps JOIN players p ON p.id = ps.player_id
			WHERE p.role != 'admin' AND ps.plan=$1`, filterPlan).Scan(&total)
	} else {
		err = h.pool.QueryRow(ctx, `
			SELECT COUNT(*)::INT
			FROM player_subscriptions ps JOIN players p ON p.id = ps.player_id
			WHERE p.role != 'admin'`).Scan(&total)
	}
	if err != nil {
		renderError(w, err)
		return
	}

	baseSelect := `
		SELECT p.id AS player_id, p.name AS player_name,
		       ps.plan, ps.billing_cycle, ps.status,
		       ps.current_period_end, ps.grace_period_end,
		       ps.gateway_customer_id, ps.gateway_sub_id, ps.created_at
		FROM player_subscriptions ps
		JOIN players p ON p.id = ps.player_id
		WHERE p.role != 'admin'`

	var queryRows interface {
		Close()
		Next() bool
		Scan(...any) error
	}

	switch {
	case filterStatus != "" && filterPlan != "":
		queryRows, err = h.pool.Query(ctx, baseSelect+
			` AND ps.status=$1 AND ps.plan=$2 ORDER BY ps.current_period_end ASC NULLS LAST LIMIT $3 OFFSET $4`,
			filterStatus, filterPlan, limit, offset)
	case filterStatus != "":
		queryRows, err = h.pool.Query(ctx, baseSelect+
			` AND ps.status=$1 ORDER BY ps.current_period_end ASC NULLS LAST LIMIT $2 OFFSET $3`,
			filterStatus, limit, offset)
	case filterPlan != "":
		queryRows, err = h.pool.Query(ctx, baseSelect+
			` AND ps.plan=$1 ORDER BY ps.current_period_end ASC NULLS LAST LIMIT $2 OFFSET $3`,
			filterPlan, limit, offset)
	default:
		queryRows, err = h.pool.Query(ctx, baseSelect+
			` ORDER BY ps.current_period_end ASC NULLS LAST LIMIT $1 OFFSET $2`,
			limit, offset)
	}
	if err != nil {
		renderError(w, err)
		return
	}
	defer queryRows.Close()

	items := make([]map[string]any, 0)
	for queryRows.Next() {
		var playerID uuid.UUID
		var playerName, plan string
		var billingCycle, status, gatewayCustomerID, gatewaySubID *string
		var currentPeriodEnd, gracePeriodEnd, createdAt interface{}
		if err := queryRows.Scan(
			&playerID, &playerName, &plan, &billingCycle, &status,
			&currentPeriodEnd, &gracePeriodEnd, &gatewayCustomerID, &gatewaySubID, &createdAt,
		); err != nil {
			renderError(w, err)
			return
		}
		items = append(items, map[string]any{
			"player_id": playerID, "player_name": playerName,
			"plan": plan, "billing_cycle": billingCycle, "status": status,
			"current_period_end": currentPeriodEnd, "grace_period_end": gracePeriodEnd,
			"gateway_customer_id": gatewayCustomerID, "gateway_sub_id": gatewaySubID,
			"created_at": createdAt,
		})
	}
	renderJSON(w, http.StatusOK, map[string]any{
		"total": total, "page": page, "page_size": pageSize, "items": items,
	})
}

func (h *adminHandler) updateSubscription(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	playerID, err := adminTargetPlayerID(r)
	if err != nil {
		renderError(w, apierror.NotFound("player not found"))
		return
	}

	var req updateSubscriptionReq
	if err := decodeJSON(r, &req); err != nil {
		renderError(w, err)
		return
	}
	if req.Plan == "" || req.Status == "" {
		renderError(w, apierror.Unprocessable("plan and status are required"))
		return
	}

	sub, err := db.GetSubscriptionByPlayer(ctx, h.pool, playerID)
	if err != nil {
		renderError(w, apierror.NotFound("subscription not found for this player"))
		return
	}

	params := db.UpdateSubscriptionParams{
		Plan:         req.Plan,
		Status:       req.Status,
		BillingCycle: req.BillingCycle,
	}
	if _, err := db.UpdateSubscription(ctx, h.pool, sub.PlayerID, params); err != nil {
		renderError(w, err)
		return
	}

	log.Printf("admin_subscription_manual_update player_id=%s plan=%s status=%s", playerID, req.Plan, req.Status)
	renderJSON(w, http.StatusOK, map[string]string{"status": "ok", "plan": req.Plan})
}

func (h *adminHandler) cancelSubscription(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	playerID, err := adminTargetPlayerID(r)
	if err != nil {
		renderError(w, apierror.NotFound("player not found"))
		return
	}

	sub, err := db.GetSubscriptionByPlayer(ctx, h.pool, playerID)
	if err != nil {
		renderError(w, apierror.NotFound("subscription not found for this player"))
		return
	}
	if sub.Plan == "free" || sub.Status == "canceled" {
		renderError(w, apierror.Unprocessable("subscription is already free or canceled"))
		return
	}

	if sub.GatewaySubID != nil && *sub.GatewaySubID != "" && h.stripe != nil {
		if err := h.stripe.CancelSubscription(*sub.GatewaySubID); err != nil {
			log.Printf("stripe cancel warning player_id=%s err=%v", playerID, err)
			// Continue — update DB even if Stripe cancel failed (already canceled there)
		}
	}

	if _, err := db.UpdateSubscription(ctx, h.pool, playerID, db.UpdateSubscriptionParams{
		Plan:   "free",
		Status: "canceled",
	}); err != nil {
		renderError(w, err)
		return
	}

	log.Printf("admin_subscription_canceled player_id=%s", playerID)
	renderJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *adminHandler) listPlayers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	page, pageSize, limit, offset := pageParams(r, 20)
	search := r.URL.Query().Get("search")

	var total int
	var err error
	if search != "" {
		pat := "%" + search + "%"
		err = h.pool.QueryRow(ctx, `
			SELECT COUNT(*)::INT FROM players p
			WHERE p.role != 'admin'
			  AND (p.name ILIKE $1 OR p.nickname ILIKE $1 OR p.whatsapp LIKE $1)`, pat).Scan(&total)
	} else {
		err = h.pool.QueryRow(ctx,
			`SELECT COUNT(*)::INT FROM players p WHERE p.role != 'admin'`).Scan(&total)
	}
	if err != nil {
		renderError(w, err)
		return
	}

	baseSelect := `
		SELECT p.id, p.name, p.nickname, p.whatsapp, p.role, p.active, p.created_at, p.avatar_url,
		       COALESCE(ps.plan, 'free') AS plan,
		       COUNT(DISTINCT gm.group_id)::INT AS total_groups
		FROM players p
		LEFT JOIN player_subscriptions ps ON ps.player_id = p.id
		LEFT JOIN group_members gm ON gm.player_id = p.id
		WHERE p.role != 'admin'`

	var rows interface {
		Close()
		Next() bool
		Scan(...any) error
	}
	if search != "" {
		pat := "%" + search + "%"
		rows, err = h.pool.Query(ctx, baseSelect+
			` AND (p.name ILIKE $1 OR p.nickname ILIKE $1 OR p.whatsapp LIKE $1)
			  GROUP BY p.id, p.name, p.nickname, p.whatsapp, p.role, p.active, p.created_at, p.avatar_url, ps.plan
			  ORDER BY p.created_at DESC LIMIT $2 OFFSET $3`, pat, limit, offset)
	} else {
		rows, err = h.pool.Query(ctx, baseSelect+
			` GROUP BY p.id, p.name, p.nickname, p.whatsapp, p.role, p.active, p.created_at, p.avatar_url, ps.plan
			  ORDER BY p.created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	}
	if err != nil {
		renderError(w, err)
		return
	}
	defer rows.Close()

	items := make([]map[string]any, 0)
	for rows.Next() {
		var id uuid.UUID
		var name, whatsapp, role, plan string
		var nickname, avatarURL *string
		var active bool
		var createdAt interface{}
		var totalGroups int
		if err := rows.Scan(&id, &name, &nickname, &whatsapp, &role, &active, &createdAt, &avatarURL, &plan, &totalGroups); err != nil {
			renderError(w, err)
			return
		}
		items = append(items, map[string]any{
			"id": id, "name": name, "nickname": nickname, "whatsapp": whatsapp,
			"role": role, "active": active, "created_at": createdAt,
			"avatar_url": avatarURL, "plan": plan, "total_groups": totalGroups,
		})
	}
	renderJSON(w, http.StatusOK, map[string]any{
		"total": total, "page": page, "page_size": pageSize, "items": items,
	})
}

func (h *adminHandler) deletePlayerAvatar(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	playerID, err := adminTargetPlayerID(r)
	if err != nil {
		renderError(w, apierror.NotFound("player not found"))
		return
	}

	player, err := db.GetPlayerByID(ctx, h.pool, playerID)
	if err != nil {
		renderError(w, apierror.NotFound("player not found"))
		return
	}

	if player.AvatarURL != nil && h.storage != nil {
		_ = h.storage.DeleteAvatarByURL(ctx, *player.AvatarURL)
	}

	if _, err := h.pool.Exec(ctx,
		`UPDATE players SET avatar_url=NULL WHERE id=$1`, playerID); err != nil {
		renderError(w, err)
		return
	}
	log.Printf("admin_avatar_removed player_id=%s", playerID)
	noContent(w)
}

func (h *adminHandler) listBetaSignups(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	page, pageSize, limit, offset := pageParams(r, 20)

	var total int
	if err := h.pool.QueryRow(ctx,
		`SELECT COUNT(*)::INT FROM android_beta_signups`).Scan(&total); err != nil {
		renderError(w, err)
		return
	}

	rows, err := h.pool.Query(ctx, `
		SELECT s.id, s.google_email, s.player_id, p.name AS player_name, s.created_at
		FROM android_beta_signups s
		LEFT JOIN players p ON p.id = s.player_id
		ORDER BY s.created_at DESC
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		renderError(w, err)
		return
	}
	defer rows.Close()

	items := make([]map[string]any, 0)
	for rows.Next() {
		var id int
		var email string
		var playerID *uuid.UUID
		var playerName *string
		var createdAt interface{}
		if err := rows.Scan(&id, &email, &playerID, &playerName, &createdAt); err != nil {
			renderError(w, err)
			return
		}
		items = append(items, map[string]any{
			"id": id, "google_email": email, "player_id": playerID,
			"player_name": playerName, "created_at": createdAt,
		})
	}
	renderJSON(w, http.StatusOK, map[string]any{
		"total": total, "page": page, "page_size": pageSize, "items": items,
	})
}

func (h *adminHandler) listApiV2Users(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_, _, limit, offset := pageParams(r, 50)

	rows, err := h.pool.Query(ctx, `
		SELECT id, name, whatsapp, api_v2_enabled
		FROM players
		WHERE role != 'admin' AND active = true
		ORDER BY name ASC
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		renderError(w, err)
		return
	}
	defer rows.Close()

	items := make([]map[string]any, 0)
	for rows.Next() {
		var id uuid.UUID
		var name, whatsapp string
		var apiV2Enabled bool
		if err := rows.Scan(&id, &name, &whatsapp, &apiV2Enabled); err != nil {
			renderError(w, err)
			return
		}
		items = append(items, map[string]any{
			"id": id, "name": name, "whatsapp": whatsapp, "api_v2_enabled": apiV2Enabled,
		})
	}
	renderJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *adminHandler) toggleApiV2User(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	playerID, err := adminTargetPlayerID(r)
	if err != nil {
		renderError(w, apierror.NotFound("player not found"))
		return
	}

	var req toggleApiV2Req
	if err := decodeJSON(r, &req); err != nil {
		renderError(w, err)
		return
	}

	if err := db.UpdatePlayerApiV2Enabled(ctx, h.pool, playerID, req.ApiV2Enabled); err != nil {
		renderError(w, err)
		return
	}
	renderJSON(w, http.StatusOK, map[string]any{"api_v2_enabled": req.ApiV2Enabled})
}
