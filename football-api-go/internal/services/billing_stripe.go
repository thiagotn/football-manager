package services

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const stripeAPIBase = "https://api.stripe.com/v1"

// StripeService handles Stripe API interactions.
type StripeService struct {
	secretKey         string
	webhookSecret     string
	priceBasicMonthly string
	priceBasicYearly  string
	priceProMonthly   string
	priceProYearly    string
	frontendURL       string
}

func NewStripeService(secretKey, webhookSecret, priceBasicMonthly, priceBasicYearly, priceProMonthly, priceProYearly, frontendURL string) *StripeService {
	return &StripeService{
		secretKey:         secretKey,
		webhookSecret:     webhookSecret,
		priceBasicMonthly: priceBasicMonthly,
		priceBasicYearly:  priceBasicYearly,
		priceProMonthly:   priceProMonthly,
		priceProYearly:    priceProYearly,
		frontendURL:       frontendURL,
	}
}

func (s *StripeService) priceID(plan, billingCycle string) (string, error) {
	switch plan + "/" + billingCycle {
	case "basic/monthly":
		return s.priceBasicMonthly, nil
	case "basic/yearly":
		return s.priceBasicYearly, nil
	case "pro/monthly":
		return s.priceProMonthly, nil
	case "pro/yearly":
		return s.priceProYearly, nil
	}
	return "", fmt.Errorf("unknown plan/billing_cycle: %s/%s", plan, billingCycle)
}

// GetOrCreateCustomer returns an existing Stripe customer ID or creates one.
func (s *StripeService) GetOrCreateCustomer(playerID, name, phone string) (string, error) {
	params := url.Values{}
	params.Set("name", name)
	params.Set("phone", phone)
	params.Set("metadata[player_id]", playerID)

	resp, err := s.post("/customers", params)
	if err != nil {
		return "", err
	}
	customerID, _ := resp["id"].(string)
	if customerID == "" {
		return "", fmt.Errorf("stripe: missing customer id in response")
	}
	return customerID, nil
}

// CreateCheckoutSession creates a Stripe Checkout Session and returns the URL.
func (s *StripeService) CreateCheckoutSession(customerID, playerID, plan, billingCycle string) (string, error) {
	priceID, err := s.priceID(plan, billingCycle)
	if err != nil {
		return "", err
	}

	params := url.Values{}
	params.Set("customer", customerID)
	params.Set("mode", "subscription")
	params.Set("line_items[0][price]", priceID)
	params.Set("line_items[0][quantity]", "1")
	params.Set("success_url", s.frontendURL+"/account/checkout/success?session_id={CHECKOUT_SESSION_ID}")
	params.Set("cancel_url", s.frontendURL+"/account/checkout/failure")
	params.Set("metadata[player_id]", playerID)
	params.Set("metadata[plan]", plan)
	params.Set("metadata[billing_cycle]", billingCycle)
	params.Set("subscription_data[metadata][player_id]", playerID)
	params.Set("subscription_data[metadata][plan]", plan)
	params.Set("subscription_data[metadata][billing_cycle]", billingCycle)

	resp, err := s.post("/checkout/sessions", params)
	if err != nil {
		return "", err
	}
	checkoutURL, _ := resp["url"].(string)
	if checkoutURL == "" {
		return "", fmt.Errorf("stripe: missing url in checkout session response")
	}
	return checkoutURL, nil
}

// VerifyWebhookSignature validates the Stripe-Signature header and returns the parsed event.
func (s *StripeService) VerifyWebhookSignature(payload []byte, sigHeader string) (map[string]any, error) {
	// Parse Stripe-Signature header: t=<ts>,v1=<sig>
	var timestamp string
	var sig string
	for _, part := range strings.Split(sigHeader, ",") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "t":
			timestamp = kv[1]
		case "v1":
			sig = kv[1]
		}
	}
	if timestamp == "" || sig == "" {
		return nil, fmt.Errorf("invalid Stripe-Signature header")
	}

	// Validate timestamp (within 5 minutes)
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid timestamp in Stripe-Signature")
	}
	diff := time.Now().Unix() - ts
	if diff > 300 || diff < -300 {
		return nil, fmt.Errorf("webhook timestamp too old or in the future")
	}

	// Compute HMAC-SHA256
	signed := timestamp + "." + string(payload)
	mac := hmac.New(sha256.New, []byte(s.webhookSecret))
	mac.Write([]byte(signed))
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expected), []byte(sig)) {
		return nil, fmt.Errorf("webhook signature mismatch")
	}

	var event map[string]any
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, fmt.Errorf("invalid webhook payload JSON: %w", err)
	}
	return event, nil
}

// CancelSubscription cancels a Stripe subscription immediately.
func (s *StripeService) CancelSubscription(subID string) error {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodDelete, stripeAPIBase+"/subscriptions/"+subID, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(s.secretKey, "")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("stripe cancel: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("stripe cancel HTTP %d: %s", resp.StatusCode, body)
	}
	return nil
}

func (s *StripeService) post(path string, params url.Values) (map[string]any, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, stripeAPIBase+path, strings.NewReader(params.Encode()))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(s.secretKey, "")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("stripe HTTP error: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	body, _ := io.ReadAll(resp.Body)
	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("stripe: failed to parse response: %w", err)
	}
	if resp.StatusCode >= 400 {
		if errObj, ok := result["error"].(map[string]any); ok {
			return nil, fmt.Errorf("stripe error: %v", errObj["message"])
		}
		return nil, fmt.Errorf("stripe HTTP %d", resp.StatusCode)
	}
	return result, nil
}
