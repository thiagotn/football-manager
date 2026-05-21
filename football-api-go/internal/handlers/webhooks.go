package handlers

import (
	"io"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thiagotn/football-manager/football-api-go/internal/apierror"
	"github.com/thiagotn/football-manager/football-api-go/internal/db"
	"github.com/thiagotn/football-manager/football-api-go/internal/services"
)

const gracePeriodDays = 7

type webhookHandler struct {
	pool   *pgxpool.Pool
	stripe *services.StripeService
}

func NewWebhookHandler(pool *pgxpool.Pool, stripe *services.StripeService) *webhookHandler {
	return &webhookHandler{pool: pool, stripe: stripe}
}

func (h *webhookHandler) HandleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	if h.stripe == nil {
		renderJSON(w, http.StatusOK, map[string]string{"status": "not_configured"})
		return
	}

	payload, err := io.ReadAll(r.Body)
	if err != nil {
		renderError(w, apierror.Unprocessable("failed to read request body"))
		return
	}

	sigHeader := r.Header.Get("Stripe-Signature")
	event, err := h.stripe.VerifyWebhookSignature(payload, sigHeader)
	if err != nil {
		log.Printf("webhook invalid signature: %v", err)
		renderError(w, apierror.Unprocessable("invalid signature"))
		return
	}

	eventID, _ := event["id"].(string)
	eventType, _ := event["type"].(string)
	dataMap, _ := event["data"].(map[string]any)
	obj, _ := dataMap["object"].(map[string]any)

	isDup, err := db.IsWebhookEventProcessed(r.Context(), h.pool, eventID)
	if err != nil {
		log.Printf("webhook idempotency check error: %v", err)
		renderJSON(w, http.StatusOK, map[string]string{"status": "error_logged"})
		return
	}
	if isDup {
		renderJSON(w, http.StatusOK, map[string]string{"status": "already_processed"})
		return
	}

	if err := h.dispatch(r, eventType, obj); err != nil {
		log.Printf("webhook processing error event=%s type=%s err=%v", eventID, eventType, err) //nolint:gosec
		renderJSON(w, http.StatusOK, map[string]string{"status": "error_logged"})
		return
	}

	_ = db.MarkWebhookEventProcessed(r.Context(), h.pool, eventID, eventType)
	renderJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *webhookHandler) dispatch(r *http.Request, eventType string, obj map[string]any) error {
	switch eventType {
	case "checkout.session.completed":
		return h.handleCheckoutCompleted(r, obj)
	case "invoice.paid":
		return h.handleInvoicePaid(r, obj)
	case "invoice.payment_failed":
		return h.handlePaymentFailed(r, obj)
	case "customer.subscription.deleted":
		return h.handleSubscriptionDeleted(r, obj)
	case "customer.subscription.updated":
		return h.handleSubscriptionUpdated(r, obj)
	}
	return nil
}

func (h *webhookHandler) handleCheckoutCompleted(r *http.Request, session map[string]any) error {
	meta, _ := session["metadata"].(map[string]any)
	playerIDStr, _ := meta["player_id"].(string)
	plan, _ := meta["plan"].(string)
	billingCycle, _ := meta["billing_cycle"].(string)
	subscriptionID, _ := session["subscription"].(string)
	customerID, _ := session["customer"].(string)

	if playerIDStr == "" || plan == "" {
		log.Printf("checkout_completed: missing metadata player_id=%s plan=%s", playerIDStr, plan)
		return nil
	}

	playerID, err := uuid.Parse(playerIDStr)
	if err != nil {
		log.Printf("checkout_completed: invalid player_id=%s", playerIDStr)
		return nil
	}

	if billingCycle == "" {
		billingCycle = "monthly"
	}
	days := 30
	if billingCycle == "yearly" {
		days = 365
	}
	periodEnd := time.Now().UTC().Add(time.Duration(days) * 24 * time.Hour)

	_, err = db.UpdateSubscription(r.Context(), h.pool, playerID, db.UpdateSubscriptionParams{
		Plan:              plan,
		Status:            "active",
		GatewayCustomerID: nullableStr(customerID),
		GatewaySubID:      nullableStr(subscriptionID),
		BillingCycle:      &billingCycle,
		CurrentPeriodEnd:  &periodEnd,
	})
	return err
}

func (h *webhookHandler) handleInvoicePaid(r *http.Request, invoice map[string]any) error {
	customerID, _ := invoice["customer"].(string)
	if customerID == "" {
		return nil
	}

	sub, err := db.GetSubscriptionByGatewayCustomer(r.Context(), h.pool, customerID)
	if err != nil {
		log.Printf("invoice.paid: customer not found %s", customerID)
		return nil
	}

	var periodEnd *time.Time
	if lines, ok := invoice["lines"].(map[string]any); ok {
		if data, ok := lines["data"].([]any); ok && len(data) > 0 {
			if line, ok := data[0].(map[string]any); ok {
				if period, ok := line["period"].(map[string]any); ok {
					if endFloat, ok := period["end"].(float64); ok {
						t := time.Unix(int64(endFloat), 0).UTC()
						periodEnd = &t
					}
				}
			}
		}
	}

	_, err = db.UpdateSubscription(r.Context(), h.pool, sub.PlayerID, db.UpdateSubscriptionParams{
		Plan:             sub.Plan,
		Status:           "active",
		CurrentPeriodEnd: periodEnd,
	})
	return err
}

func (h *webhookHandler) handlePaymentFailed(r *http.Request, invoice map[string]any) error {
	customerID, _ := invoice["customer"].(string)
	if customerID == "" {
		return nil
	}

	sub, err := db.GetSubscriptionByGatewayCustomer(r.Context(), h.pool, customerID)
	if err != nil {
		log.Printf("invoice.payment_failed: customer not found %s", customerID)
		return nil
	}

	grace := time.Now().UTC().Add(gracePeriodDays * 24 * time.Hour)
	_, err = db.UpdateSubscription(r.Context(), h.pool, sub.PlayerID, db.UpdateSubscriptionParams{
		Plan:           sub.Plan,
		Status:         "past_due",
		GracePeriodEnd: &grace,
	})
	return err
}

func (h *webhookHandler) handleSubscriptionDeleted(r *http.Request, subscription map[string]any) error {
	customerID, _ := subscription["customer"].(string)
	if customerID == "" {
		return nil
	}

	sub, err := db.GetSubscriptionByGatewayCustomer(r.Context(), h.pool, customerID)
	if err != nil {
		log.Printf("subscription.deleted: customer not found %s", customerID)
		return nil
	}

	_, err = db.UpdateSubscription(r.Context(), h.pool, sub.PlayerID, db.UpdateSubscriptionParams{
		Plan:   "free",
		Status: "canceled",
	})
	return err
}

func (h *webhookHandler) handleSubscriptionUpdated(r *http.Request, subscription map[string]any) error {
	customerID, _ := subscription["customer"].(string)
	meta, _ := subscription["metadata"].(map[string]any)
	plan, _ := meta["plan"].(string)

	if customerID == "" || plan == "" {
		return nil
	}

	sub, err := db.GetSubscriptionByGatewayCustomer(r.Context(), h.pool, customerID)
	if err != nil {
		log.Printf("subscription.updated: customer not found %s", customerID)
		return nil
	}

	_, err = db.UpdateSubscription(r.Context(), h.pool, sub.PlayerID, db.UpdateSubscriptionParams{
		Plan:   plan,
		Status: "active",
	})
	return err
}

func nullableStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
