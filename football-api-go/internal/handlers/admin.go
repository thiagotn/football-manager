package handlers

import (
	"context"
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

type AdminStore interface {
	GetAdminStats(ctx context.Context) (*db.AdminStats, error)
	CountMatches(ctx context.Context, status *string) (int, error)
	ListMatches(ctx context.Context, status *string, limit, offset int) ([]db.AdminMatch, error)
	ListGroups(ctx context.Context, limit, offset int) (int, []db.AdminGroup, error)
	GetSubscriptionSummary(ctx context.Context) (int, []db.SubscriptionSummaryStat, error)
	CountSubscriptions(ctx context.Context, params db.ListSubscriptionsParams) (int, error)
	ListSubscriptions(ctx context.Context, params db.ListSubscriptionsParams) ([]db.AdminSubscription, error)
	GetSubscriptionByPlayer(ctx context.Context, playerID uuid.UUID) (*db.PlayerSubscription, error)
	UpdateSubscription(ctx context.Context, playerID uuid.UUID, params db.UpdateSubscriptionParams) (*db.PlayerSubscription, error)
	CountPlayers(ctx context.Context, search *string) (int, error)
	ListPlayers(ctx context.Context, search *string, limit, offset int) ([]db.AdminPlayer, error)
	GetPlayerByID(ctx context.Context, playerID uuid.UUID) (*db.Player, error)
	DeletePlayerAvatarURL(ctx context.Context, playerID uuid.UUID) error
	CountBetaSignups(ctx context.Context) (int, error)
	ListBetaSignups(ctx context.Context, limit, offset int) ([]db.AndroidBetaSignup, error)
}

type pgAdminStore struct {
	pool *pgxpool.Pool
}

func (s *pgAdminStore) GetAdminStats(ctx context.Context) (*db.AdminStats, error) {
	return db.GetAdminStats(ctx, s.pool)
}

func (s *pgAdminStore) CountMatches(ctx context.Context, status *string) (int, error) {
	return db.CountMatches(ctx, s.pool, status)
}

func (s *pgAdminStore) ListMatches(ctx context.Context, status *string, limit, offset int) ([]db.AdminMatch, error) {
	return db.ListMatches(ctx, s.pool, status, limit, offset)
}

func (s *pgAdminStore) ListGroups(ctx context.Context, limit, offset int) (int, []db.AdminGroup, error) {
	return db.ListGroups(ctx, s.pool, limit, offset)
}

func (s *pgAdminStore) GetSubscriptionSummary(ctx context.Context) (int, []db.SubscriptionSummaryStat, error) {
	return db.GetSubscriptionSummary(ctx, s.pool)
}

func (s *pgAdminStore) CountSubscriptions(ctx context.Context, params db.ListSubscriptionsParams) (int, error) {
	return db.CountSubscriptions(ctx, s.pool, params)
}

func (s *pgAdminStore) ListSubscriptions(ctx context.Context, params db.ListSubscriptionsParams) ([]db.AdminSubscription, error) {
	return db.ListSubscriptions(ctx, s.pool, params)
}

func (s *pgAdminStore) GetSubscriptionByPlayer(ctx context.Context, playerID uuid.UUID) (*db.PlayerSubscription, error) {
	return db.GetSubscriptionByPlayer(ctx, s.pool, playerID)
}

func (s *pgAdminStore) UpdateSubscription(ctx context.Context, playerID uuid.UUID, params db.UpdateSubscriptionParams) (*db.PlayerSubscription, error) {
	return db.UpdateSubscription(ctx, s.pool, playerID, params)
}

func (s *pgAdminStore) CountPlayers(ctx context.Context, search *string) (int, error) {
	return db.CountPlayers(ctx, s.pool, search)
}

func (s *pgAdminStore) ListPlayers(ctx context.Context, search *string, limit, offset int) ([]db.AdminPlayer, error) {
	return db.ListPlayers(ctx, s.pool, search, limit, offset)
}

func (s *pgAdminStore) GetPlayerByID(ctx context.Context, playerID uuid.UUID) (*db.Player, error) {
	return db.GetPlayerByID(ctx, s.pool, playerID)
}

func (s *pgAdminStore) DeletePlayerAvatarURL(ctx context.Context, playerID uuid.UUID) error {
	return db.DeletePlayerAvatarURL(ctx, s.pool, playerID)
}

func (s *pgAdminStore) CountBetaSignups(ctx context.Context) (int, error) {
	return db.CountBetaSignups(ctx, s.pool)
}

func (s *pgAdminStore) ListBetaSignups(ctx context.Context, limit, offset int) ([]db.AndroidBetaSignup, error) {
	return db.ListBetaSignups(ctx, s.pool, limit, offset)
}

type AdminHandler struct {
	Store   AdminStore
	stripe  *services.StripeService
	storage *services.StorageService
}

func NewAdminHandler(pool *pgxpool.Pool, stripe *services.StripeService, storage *services.StorageService) *AdminHandler {
	return &AdminHandler{
		Store:   &pgAdminStore{pool: pool},
		stripe:  stripe,
		storage: storage,
	}
}

func (h *AdminHandler) Routes() chi.Router {
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
	return r
}

// ── Request types ─────────────────────────────────────────────────────────────

type updateSubscriptionReq struct {
	Plan         string  `json:"plan"`
	Status       string  `json:"status"`
	BillingCycle *string `json:"billing_cycle"`
	Reason       *string `json:"reason"`
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

func (h *AdminHandler) getStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	stats, err := h.Store.GetAdminStats(ctx)
	if err != nil {
		renderError(w, err)
		return
	}

	renderJSON(w, http.StatusOK, map[string]any{
		"total_matches":           stats.TotalMatches,
		"total_groups":            stats.TotalGroups,
		"total_players":           stats.TotalPlayers,
		"platform_minutes_played": stats.PlatformMinutes,
		"signups_total":           stats.SignupsTotal,
		"signups_last_7_days":     stats.Signups7D,
		"signups_last_30_days":    stats.Signups30D,
		"total_reviews":           stats.TotalReviews,
	})
}

func (h *AdminHandler) listMatches(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	statusParam := r.URL.Query().Get("status")
	var status *string
	if statusParam != "" {
		status = &statusParam
	}
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

	total, err := h.Store.CountMatches(ctx, status)
	if err != nil {
		renderError(w, err)
		return
	}

	matches, err := h.Store.ListMatches(ctx, status, limit, offset)
	if err != nil {
		renderError(w, err)
		return
	}

	items := make([]map[string]any, len(matches))
	for i, m := range matches {
		items[i] = map[string]any{
			"id": m.ID, "hash": m.Hash, "number": m.Number,
			"group_id": m.GroupID, "group_name": m.GroupName,
			"match_date": m.MatchDate, "start_time": m.StartTime, "end_time": m.EndTime,
			"location": m.Location, "status": m.Status,
		}
	}
	renderJSON(w, http.StatusOK, map[string]any{"total": total, "items": items})
}

func (h *AdminHandler) listGroups(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_, _, limit, offset := pageParams(r, 50)

	total, groups, err := h.Store.ListGroups(ctx, limit, offset)
	if err != nil {
		renderError(w, err)
		return
	}

	items := make([]map[string]any, len(groups))
	for i, g := range groups {
		items[i] = map[string]any{
			"id": g.ID, "name": g.Name, "description": g.Description, "slug": g.Slug,
			"created_at": g.CreatedAt, "total_members": g.TotalMembers, "total_matches": g.TotalMatches,
		}
	}
	renderJSON(w, http.StatusOK, map[string]any{"total": total, "items": items})
}

func (h *AdminHandler) subscriptionSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	totalPlayers, stats, err := h.Store.GetSubscriptionSummary(ctx)
	if err != nil {
		renderError(w, err)
		return
	}

	active, free, pastDue, canceled, mrrCentsTotal := 0, 0, 0, 0, 0
	type breakdownKey struct{ plan, cycle string }
	breakdownMap := map[breakdownKey]int{}

	for _, s := range stats {
		switch {
		case s.Plan == "free":
			free += s.Count
		case s.Status == "canceled":
			canceled += s.Count
		case s.Status == "past_due":
			pastDue += s.Count
		case s.Status == "active":
			active += s.Count
			if price, ok := mrrCents[[2]string{s.Plan, s.BillingCycle}]; ok {
				mrrCentsTotal += price * s.Count
			}
		default:
			free += s.Count
		}
		if s.Plan != "free" && s.Status != "canceled" {
			k := breakdownKey{s.Plan, s.BillingCycle}
			breakdownMap[k] += s.Count
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

func (h *AdminHandler) listSubscriptions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	page, pageSize, limit, offset := pageParams(r, 20)
	filterStatus := r.URL.Query().Get("status")
	filterPlan := r.URL.Query().Get("plan")

	params := db.ListSubscriptionsParams{
		Status: filterStatus,
		Plan:   filterPlan,
		Limit:  limit,
		Offset: offset,
	}

	total, err := h.Store.CountSubscriptions(ctx, params)
	if err != nil {
		renderError(w, err)
		return
	}

	subs, err := h.Store.ListSubscriptions(ctx, params)
	if err != nil {
		renderError(w, err)
		return
	}

	items := make([]map[string]any, len(subs))
	for i, s := range subs {
		items[i] = map[string]any{
			"player_id": s.PlayerID, "player_name": s.PlayerName,
			"plan": s.Plan, "billing_cycle": s.BillingCycle, "status": s.Status,
			"current_period_end": s.CurrentPeriodEnd, "grace_period_end": s.GracePeriodEnd,
			"gateway_customer_id": s.GatewayCustomerID, "gateway_sub_id": s.GatewaySubID,
			"created_at": s.CreatedAt,
		}
	}
	renderJSON(w, http.StatusOK, map[string]any{
		"total": total, "page": page, "page_size": pageSize, "items": items,
	})
}

func (h *AdminHandler) updateSubscription(w http.ResponseWriter, r *http.Request) {
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

	sub, err := h.Store.GetSubscriptionByPlayer(ctx, playerID)
	if err != nil {
		renderError(w, apierror.NotFound("subscription not found for this player"))
		return
	}

	params := db.UpdateSubscriptionParams{
		Plan:         req.Plan,
		Status:       req.Status,
		BillingCycle: req.BillingCycle,
	}
	if _, err := h.Store.UpdateSubscription(ctx, sub.PlayerID, params); err != nil {
		renderError(w, err)
		return
	}

	log.Printf("admin_subscription_manual_update player_id=%s plan=%s status=%s", playerID, req.Plan, req.Status) //nolint:gosec
	renderJSON(w, http.StatusOK, map[string]string{"status": "ok", "plan": req.Plan})
}

func (h *AdminHandler) cancelSubscription(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	playerID, err := adminTargetPlayerID(r)
	if err != nil {
		renderError(w, apierror.NotFound("player not found"))
		return
	}

	sub, err := h.Store.GetSubscriptionByPlayer(ctx, playerID)
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
			log.Printf("stripe cancel warning player_id=%s err=%v", playerID, err) //nolint:gosec
			// Continue — update DB even if Stripe cancel failed (already canceled there)
		}
	}

	if _, err := h.Store.UpdateSubscription(ctx, playerID, db.UpdateSubscriptionParams{
		Plan:   "free",
		Status: "canceled",
	}); err != nil {
		renderError(w, err)
		return
	}

	log.Printf("admin_subscription_canceled player_id=%s", playerID) //nolint:gosec
	renderJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *AdminHandler) listPlayers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	page, pageSize, limit, offset := pageParams(r, 20)
	search := r.URL.Query().Get("search")

	var searchPtr *string
	if search != "" {
		searchPtr = &search
	}

	total, err := h.Store.CountPlayers(ctx, searchPtr)
	if err != nil {
		renderError(w, err)
		return
	}

	players, err := h.Store.ListPlayers(ctx, searchPtr, limit, offset)
	if err != nil {
		renderError(w, err)
		return
	}

	items := make([]map[string]any, len(players))
	for i, p := range players {
		items[i] = map[string]any{
			"id": p.ID, "name": p.Name, "nickname": p.Nickname, "whatsapp": p.WhatsApp,
			"role": p.Role, "active": p.Active, "created_at": p.CreatedAt,
			"avatar_url": p.AvatarURL, "plan": p.Plan, "total_groups": p.TotalGroups,
		}
	}
	renderJSON(w, http.StatusOK, map[string]any{
		"total": total, "page": page, "page_size": pageSize, "items": items,
	})
}

func (h *AdminHandler) deletePlayerAvatar(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	playerID, err := adminTargetPlayerID(r)
	if err != nil {
		renderError(w, apierror.NotFound("player not found"))
		return
	}

	player, err := h.Store.GetPlayerByID(ctx, playerID)
	if err != nil {
		renderError(w, apierror.NotFound("player not found"))
		return
	}

	if player.AvatarURL != nil && h.storage != nil {
		_ = h.storage.DeleteAvatarByURL(ctx, *player.AvatarURL)
	}

	if err := h.Store.DeletePlayerAvatarURL(ctx, playerID); err != nil {
		renderError(w, err)
		return
	}
	log.Printf("admin_avatar_removed player_id=%s", playerID) //nolint:gosec
	noContent(w)
}

func (h *AdminHandler) listBetaSignups(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	page, pageSize, limit, offset := pageParams(r, 20)

	total, err := h.Store.CountBetaSignups(ctx)
	if err != nil {
		renderError(w, err)
		return
	}

	signups, err := h.Store.ListBetaSignups(ctx, limit, offset)
	if err != nil {
		renderError(w, err)
		return
	}

	items := make([]map[string]any, len(signups))
	for i, s := range signups {
		items[i] = map[string]any{
			"id": s.ID, "google_email": s.Email, "player_id": s.PlayerID,
			"player_name": s.PlayerName, "created_at": s.CreatedAt,
		}
	}
	renderJSON(w, http.StatusOK, map[string]any{
		"total": total, "page": page, "page_size": pageSize, "items": items,
	})
}
