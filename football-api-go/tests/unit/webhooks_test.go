package unit_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/thiagotn/football-manager/football-api-go/internal/apierror"
)

// ── Webhook error responses (via apierror) ────────────────────────────────────

func TestWebhookErrorResponses(t *testing.T) {
	t.Run("bad request for invalid signature", func(t *testing.T) {
		err := apierror.BadRequest("invalid signature")
		assert.Equal(t, "invalid signature", err.Error())
		apiErr := err.(*apierror.APIError)
		assert.Equal(t, 400, apiErr.Code)
	})

	t.Run("unprocessable for validation", func(t *testing.T) {
		err := apierror.Unprocessable("invalid webhook payload")
		assert.Equal(t, "invalid webhook payload", err.Error())
		apiErr := err.(*apierror.APIError)
		assert.Equal(t, 422, apiErr.Code)
	})

	t.Run("JSON serialization of error", func(t *testing.T) {
		err := apierror.BadRequest("test error")
		apiErr := err.(*apierror.APIError)

		data := map[string]any{
			"detail": apiErr.Detail,
		}
		b, _ := json.Marshal(data)
		assert.Contains(t, string(b), "test error")
	})
}
