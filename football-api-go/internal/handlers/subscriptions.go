package handlers

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thiagotn/football-manager/football-api-go/internal/apierror"
	"github.com/thiagotn/football-manager/football-api-go/internal/db"
	"github.com/thiagotn/football-manager/football-api-go/internal/middleware"
	"github.com/thiagotn/football-manager/football-api-go/internal/services"
)

var planLimits = map[string]struct {
	groupsLimit  *int
	membersLimit *int
}{
	"free":  {intPtr(1), intPtr(30)},
	"basic": {intPtr(3), intPtr(50)},
	"pro":   {nil, nil},
}

var paidPlans = map[string]bool{"basic": true, "pro": true}

type SubscriptionStore interface {
	GetOrCreateSubscription(ctx context.Context, playerID uuid.UUID) (*db.PlayerSubscription, error)
	UpdateSubscription(ctx context.Context, playerID uuid.UUID, params db.UpdateSubscriptionParams) (*db.PlayerSubscription, error)
	CountAdminGroups(ctx context.Context, playerID uuid.UUID) (int, error)
}

type pgSubscriptionStore struct {
	pool *pgxpool.Pool
}

func (s *pgSubscriptionStore) GetOrCreateSubscription(ctx context.Context, playerID uuid.UUID) (*db.PlayerSubscription, error) {
	return db.GetOrCreateSubscription(ctx, s.pool, playerID)
}

func (s *pgSubscriptionStore) UpdateSubscription(ctx context.Context, playerID uuid.UUID, params db.UpdateSubscriptionParams) (*db.PlayerSubscription, error) {
	return db.UpdateSubscription(ctx, s.pool, playerID, params)
}

func (s *pgSubscriptionStore) CountAdminGroups(ctx context.Context, playerID uuid.UUID) (int, error) {
	return db.CountAdminGroups(ctx, s.pool, playerID)
}

type SubscriptionHandler struct {
	Store  SubscriptionStore
	Stripe *services.StripeService
}

func NewSubscriptionHandler(pool *pgxpool.Pool, stripe *services.StripeService) *SubscriptionHandler {
	return &SubscriptionHandler{Store: &pgSubscriptionStore{pool: pool}, Stripe: stripe}
}

func (h *SubscriptionHandler) GetMySubscription(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())
	if player == nil {
		renderError(w, apierror.Unauthorized())
		return
	}

	sub, err := h.Store.GetOrCreateSubscription(r.Context(), player.ID)
	if err != nil {
		renderError(w, apierror.Internal("failed to get subscription"))
		return
	}

	if player.Role == db.PlayerRoleAdmin {
		renderJSON(w, http.StatusOK, map[string]any{
			"plan":          sub.Plan,
			"groups_limit":  nil,
			"groups_used":   0,
			"members_limit": nil,
			"status":        sub.Status,
		})
		return
	}

	limits, ok := planLimits[sub.Plan]
	if !ok {
		limits = planLimits["free"]
	}

	groupsUsed, err := h.Store.CountAdminGroups(r.Context(), player.ID)
	if err != nil {
		renderError(w, apierror.Internal("failed to count groups"))
		return
	}

	renderJSON(w, http.StatusOK, map[string]any{
		"plan":                sub.Plan,
		"groups_limit":        limits.groupsLimit,
		"groups_used":         groupsUsed,
		"members_limit":       limits.membersLimit,
		"status":              sub.Status,
		"gateway_customer_id": sub.GatewayCustomerID,
		"gateway_sub_id":      sub.GatewaySubID,
		"current_period_end":  sub.CurrentPeriodEnd,
		"grace_period_end":    sub.GracePeriodEnd,
	})
}

func (h *SubscriptionHandler) CreateCheckoutSession(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())
	if player == nil {
		renderError(w, apierror.Unauthorized())
		return
	}

	var body struct {
		Plan         string `json:"plan"`
		BillingCycle string `json:"billing_cycle"`
	}
	if err := decodeJSON(r, &body); err != nil {
		renderError(w, err)
		return
	}

	if !paidPlans[body.Plan] {
		renderError(w, apierror.Unprocessable("invalid plan for checkout"))
		return
	}
	if body.BillingCycle != "monthly" && body.BillingCycle != "yearly" {
		renderError(w, apierror.Unprocessable("billing_cycle must be 'monthly' or 'yearly'"))
		return
	}

	if h.Stripe == nil {
		renderError(w, apierror.Internal("payment service not configured"))
		return
	}

	if err := h.Stripe.ValidatePriceID(body.Plan, body.BillingCycle); err != nil {
		renderError(w, apierror.Internal(err.Error()))
		return
	}

	sub, err := h.Store.GetOrCreateSubscription(r.Context(), player.ID)
	if err != nil {
		renderError(w, apierror.Internal("failed to get subscription"))
		return
	}

	customerID := ""
	if sub.GatewayCustomerID != nil {
		customerID = *sub.GatewayCustomerID
	}

	if customerID == "" {
		customerID, err = h.Stripe.GetOrCreateCustomer(player.ID.String(), player.Name, player.WhatsApp)
		if err != nil {
			renderError(w, apierror.Internal("failed to create customer"))
			return
		}
		_, _ = h.Store.UpdateSubscription(r.Context(), player.ID, db.UpdateSubscriptionParams{
			Plan:              sub.Plan,
			Status:            sub.Status,
			GatewayCustomerID: &customerID,
		})
	}

	checkoutURL, err := h.Stripe.CreateCheckoutSession(customerID, player.ID.String(), body.Plan, body.BillingCycle)
	if err != nil {
		renderError(w, apierror.Internal("failed to create checkout session"))
		return
	}

	renderJSON(w, http.StatusCreated, map[string]string{"checkout_url": checkoutURL})
}

func intPtr(n int) *int { return &n }
