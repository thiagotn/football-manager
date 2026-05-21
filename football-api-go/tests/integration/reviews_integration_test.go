package integration_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReviews_GetMyReview_None(t *testing.T) {
	srv := newTestServer(t)
	p := registerAndLogin(t, srv, "Test Player")
	enableApiV2(t, p.ID)

	res := apiCall(t, srv, http.MethodGet, "/api/v2/reviews/me", p.Token, nil)
	assert.Equal(t, http.StatusOK, res.Code)

	body := res.Body
	// No review yet
	assert.True(t, body["id"] == nil || body["id"] == "")
}

func TestReviews_UpsertMyReview_Create(t *testing.T) {
	srv := newTestServer(t)
	p := registerAndLogin(t, srv, "Test Player")
	enableApiV2(t, p.ID)

	payload := map[string]any{
		"rating": 5,
		"text":   "Great app!",
	}

	res := apiCall(t, srv, http.MethodPut, "/api/v2/reviews/me", p.Token, payload)
	assert.True(t, res.Code == http.StatusOK || res.Code == http.StatusCreated)

	body := res.Body
	assert.Equal(t, float64(5), body["rating"])
	assert.Equal(t, "Great app!", body["text"])
}

func TestReviews_UpsertMyReview_Update(t *testing.T) {
	srv := newTestServer(t)
	p := registerAndLogin(t, srv, "Test Player")
	enableApiV2(t, p.ID)

	// Create
	apiCall(t, srv, http.MethodPut, "/api/v2/reviews/me", p.Token, map[string]any{
		"rating": 4,
		"text":   "Good",
	})

	// Update
	res := apiCall(t, srv, http.MethodPut, "/api/v2/reviews/me", p.Token, map[string]any{
		"rating": 5,
		"text":   "Excellent!",
	})

	assert.Equal(t, http.StatusOK, res.Code)

	body := res.Body
	assert.Equal(t, float64(5), body["rating"])
	assert.Equal(t, "Excellent!", body["text"])
}

func TestReviews_UpsertMyReview_InvalidRating(t *testing.T) {
	srv := newTestServer(t)
	p := registerAndLogin(t, srv, "Test Player")
	enableApiV2(t, p.ID)

	payload := map[string]any{
		"rating": 6, // Out of range
		"text":   "Invalid",
	}

	res := apiCall(t, srv, http.MethodPut, "/api/v2/reviews/me", p.Token, payload)
	assert.Equal(t, http.StatusUnprocessableEntity, res.Code)
}

func TestReviews_GetSummary(t *testing.T) {
	srv := newTestServer(t)
	p := registerAndLogin(t, srv, "Test Player")
	enableApiV2(t, p.ID)

	// Create a review
	apiCall(t, srv, http.MethodPut, "/api/v2/reviews/me", p.Token, map[string]any{
		"rating": 5,
		"text":   "Great!",
	})

	// Get summary
	res := apiCall(t, srv, http.MethodGet, "/api/v2/reviews/summary", p.Token, nil)
	assert.Equal(t, http.StatusOK, res.Code)

	body := res.Body
	assert.Contains(t, body, "average_rating")
	assert.Contains(t, body, "total_reviews")
}

func TestReviews_ListReviews(t *testing.T) {
	srv := newTestServer(t)
	p := registerAndLogin(t, srv, "Test Player")
	enableApiV2(t, p.ID)

	// Create a review
	apiCall(t, srv, http.MethodPut, "/api/v2/reviews/me", p.Token, map[string]any{
		"rating": 4,
		"text":   "Good app",
	})

	// List all reviews
	res := apiCall(t, srv, http.MethodGet, "/api/v2/reviews", p.Token, nil)
	assert.Equal(t, http.StatusOK, res.Code)

	body := res.Body
	assert.Contains(t, body, "reviews")
}

func TestReviews_RequiresAuth(t *testing.T) {
	srv := newTestServer(t)

	res := apiCall(t, srv, http.MethodGet, "/api/v2/reviews/me", "", nil)
	assert.Equal(t, http.StatusUnauthorized, res.Code)
}
