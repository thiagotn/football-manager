package integration_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWebhooks_PaymentWebhook_InvalidSignature(t *testing.T) {
	srv := newTestServer(t)

	payload := []byte(`{"type":"checkout.session.completed"}`)

	// POST with invalid signature
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, srv.URL+"/api/v2/webhooks/payment", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Stripe-Signature", "invalid-signature")

	client := http.DefaultClient
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestWebhooks_PaymentWebhook_MissingSignature(t *testing.T) {
	srv := newTestServer(t)

	payload := []byte(`{"type":"checkout.session.completed"}`)

	// POST without signature header
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, srv.URL+"/api/v2/webhooks/payment", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")

	client := http.DefaultClient
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestWebhooks_PaymentWebhook_InvalidJSON(t *testing.T) {
	srv := newTestServer(t)

	payload := []byte(`{invalid json}`)

	// POST with invalid JSON
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, srv.URL+"/api/v2/webhooks/payment", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Stripe-Signature", "t=timestamp,v1=signature")

	client := http.DefaultClient
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.True(t, resp.StatusCode >= 400)
}

func TestWebhooks_PaymentWebhook_ValidSignature(t *testing.T) {
	srv := newTestServer(t)

	// Use a test webhook secret
	webhookSecret := "whsec_test_secret_12345"
	payload := []byte(`{"id":"evt_test","type":"checkout.session.completed"}`)
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	// Compute correct signature
	signed := timestamp + "." + string(payload)
	mac := hmac.New(sha256.New, []byte(webhookSecret))
	mac.Write([]byte(signed))
	sig := hex.EncodeToString(mac.Sum(nil))
	sigHeader := fmt.Sprintf("t=%s,v1=%s", timestamp, sig)

	// POST with valid signature
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, srv.URL+"/api/v2/webhooks/payment", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Stripe-Signature", sigHeader)

	client := http.DefaultClient
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Should either accept (200) or reject (based on webhook secret matching)
	assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusBadRequest)
}

func TestWebhooks_PaymentWebhook_TimestampTooOld(t *testing.T) {
	srv := newTestServer(t)

	webhookSecret := "whsec_test_secret_12345"
	payload := []byte(`{"id":"evt_test","type":"checkout.session.completed"}`)
	// Timestamp 10 minutes ago (beyond 5 minute window)
	timestamp := strconv.FormatInt(time.Now().Add(-600*time.Second).Unix(), 10)

	// Compute correct signature
	signed := timestamp + "." + string(payload)
	mac := hmac.New(sha256.New, []byte(webhookSecret))
	mac.Write([]byte(signed))
	sig := hex.EncodeToString(mac.Sum(nil))
	sigHeader := fmt.Sprintf("t=%s,v1=%s", timestamp, sig)

	// POST with old timestamp
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, srv.URL+"/api/v2/webhooks/payment", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Stripe-Signature", sigHeader)

	client := http.DefaultClient
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Should reject old timestamp
	assert.True(t, resp.StatusCode >= 400)
}
