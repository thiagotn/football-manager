package integration_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReviews_GetMyReview_None(t *testing.T) {
	srv := newTestServer(t)
	p := registerAndLogin(t, srv, "Player")
	enableApiV2(t, p.ID)

	// GET /api/v2/reviews/me before creating a review
	res := apiCall(t, srv, http.MethodGet, "/api/v2/reviews/me", p.Token, nil)
	assert.Equal(t, http.StatusOK, res.Code)
	// Should return empty/null review
	assert.NotNil(t, res.Body)
}

func TestReviews_CreateReview(t *testing.T) {
	srv := newTestServer(t)
	p := registerAndLogin(t, srv, "Player")
	enableApiV2(t, p.ID)

	// PUT /api/v2/reviews/me to create review
	res := apiCall(t, srv, http.MethodPut, "/api/v2/reviews/me", p.Token, map[string]any{
		"rating":  5,
		"comment": "Great experience!",
	})
	assert.True(t, res.Code == http.StatusOK || res.Code == http.StatusCreated)
	assert.Contains(t, res.Body, "rating")
}

func TestReviews_UpdateReview(t *testing.T) {
	srv := newTestServer(t)
	p := registerAndLogin(t, srv, "Player")
	enableApiV2(t, p.ID)

	// Create initial review
	createRes := apiCall(t, srv, http.MethodPut, "/api/v2/reviews/me", p.Token, map[string]any{
		"rating":  4,
		"comment": "Good",
	})
	assert.True(t, createRes.Code == http.StatusOK || createRes.Code == http.StatusCreated)

	// Update review
	updateRes := apiCall(t, srv, http.MethodPut, "/api/v2/reviews/me", p.Token, map[string]any{
		"rating":  5,
		"comment": "Excellent!",
	})
	assert.Equal(t, http.StatusOK, updateRes.Code)
	assert.Equal(t, float64(5), updateRes.Body["rating"])
}

func TestReviews_CreateReview_InvalidRating(t *testing.T) {
	srv := newTestServer(t)
	p := registerAndLogin(t, srv, "Player")
	enableApiV2(t, p.ID)

	// PUT with invalid rating (out of bounds)
	res := apiCall(t, srv, http.MethodPut, "/api/v2/reviews/me", p.Token, map[string]any{
		"rating":  10,
		"comment": "Invalid rating",
	})
	assert.Equal(t, http.StatusUnprocessableEntity, res.Code)
}

func TestReviews_CreateReview_MissingRating(t *testing.T) {
	srv := newTestServer(t)
	p := registerAndLogin(t, srv, "Player")
	enableApiV2(t, p.ID)

	// PUT without rating field
	res := apiCall(t, srv, http.MethodPut, "/api/v2/reviews/me", p.Token, map[string]any{
		"comment": "No rating",
	})
	assert.Equal(t, http.StatusUnprocessableEntity, res.Code)
}

func TestReviews_GetSummary(t *testing.T) {
	srv := newTestServer(t)
	p := registerAndLogin(t, srv, "Player")
	enableApiV2(t, p.ID)

	// GET /api/v2/reviews/summary
	res := apiCall(t, srv, http.MethodGet, "/api/v2/reviews/summary", p.Token, nil)
	assert.Equal(t, http.StatusOK, res.Code)
	assert.NotNil(t, res.Body)
}

func TestReviews_GetAllReviews(t *testing.T) {
	srv := newTestServer(t)
	p := registerAndLogin(t, srv, "Player")
	enableApiV2(t, p.ID)

	// GET /api/v2/reviews to list all reviews
	res := apiCall(t, srv, http.MethodGet, "/api/v2/reviews", p.Token, nil)
	assert.Equal(t, http.StatusOK, res.Code)
	assert.NotNil(t, res.List)
}
