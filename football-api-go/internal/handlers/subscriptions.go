package handlers

import (
	"net/http"

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

type subscriptionHandler struct {
	pool   *pgxpool.Pool
	stripe *services.StripeService
}

func NewSubscriptionHandler(pool *pgxpool.Pool, stripe *services.StripeService) *subscriptionHandler {
	return &subscriptionHandler{pool: pool, stripe: stripe}
}

func (h *subscriptionHandler) GetMySubscription(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())
	if player == nil {
		renderError(w, apierror.Unauthorized())
		return
	}

	sub, err := db.GetOrCreateSubscription(r.Context(), h.pool, player.ID)
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

	groupsUsed, err := db.CountAdminGroups(r.Context(), h.pool, player.ID)
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

func (h *subscriptionHandler) CreateCheckoutSession(w http.ResponseWriter, r *http.Request) {
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

	if h.stripe == nil {
		renderError(w, apierror.Internal("payment service not configured"))
		return
	}

	sub, err := db.GetOrCreateSubscription(r.Context(), h.pool, player.ID)
	if err != nil {
		renderError(w, apierror.Internal("failed to get subscription"))
		return
	}

	customerID := ""
	if sub.GatewayCustomerID != nil {
		customerID = *sub.GatewayCustomerID
	}

	if customerID == "" {
		customerID, err = h.stripe.GetOrCreateCustomer(player.ID.String(), player.Name, player.WhatsApp)
		if err != nil {
			renderError(w, apierror.Internal("failed to create customer"))
			return
		}
		_, _ = db.UpdateSubscription(r.Context(), h.pool, player.ID, db.UpdateSubscriptionParams{
			Plan:              sub.Plan,
			Status:            sub.Status,
			GatewayCustomerID: &customerID,
		})
	}

	checkoutURL, err := h.stripe.CreateCheckoutSession(customerID, player.ID.String(), body.Plan, body.BillingCycle)
	if err != nil {
		renderError(w, apierror.Internal("failed to create checkout session"))
		return
	}

	renderJSON(w, http.StatusCreated, map[string]string{"checkout_url": checkoutURL})
}

func intPtr(n int) *int { return &n }
