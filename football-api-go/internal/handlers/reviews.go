package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thiagotn/football-manager/football-api-go/internal/apierror"
	"github.com/thiagotn/football-manager/football-api-go/internal/db"
	"github.com/thiagotn/football-manager/football-api-go/internal/middleware"
)

type ReviewStore interface {
	GetMyReview(ctx context.Context, playerID uuid.UUID) (*db.AppReview, error)
	UpsertReview(ctx context.Context, playerID uuid.UUID, rating int, comment *string) (*db.AppReview, error)
	GetReviewSummary(ctx context.Context) (*db.ReviewSummary, error)
	ListReviews(ctx context.Context, ratings []int, orderBy string, page, pageSize int) (*db.ReviewPage, error)
}

type pgReviewStore struct {
	pool *pgxpool.Pool
}

func (s *pgReviewStore) GetMyReview(ctx context.Context, playerID uuid.UUID) (*db.AppReview, error) {
	return db.GetMyReview(ctx, s.pool, playerID)
}

func (s *pgReviewStore) UpsertReview(ctx context.Context, playerID uuid.UUID, rating int, comment *string) (*db.AppReview, error) {
	return db.UpsertReview(ctx, s.pool, playerID, rating, comment)
}

func (s *pgReviewStore) GetReviewSummary(ctx context.Context) (*db.ReviewSummary, error) {
	return db.GetReviewSummary(ctx, s.pool)
}

func (s *pgReviewStore) ListReviews(ctx context.Context, ratings []int, orderBy string, page, pageSize int) (*db.ReviewPage, error) {
	return db.ListReviews(ctx, s.pool, ratings, orderBy, page, pageSize)
}

type ReviewHandler struct {
	Store ReviewStore
}

func NewReviewHandler(pool *pgxpool.Pool) *ReviewHandler {
	return &ReviewHandler{Store: &pgReviewStore{pool: pool}}
}

func (h *ReviewHandler) GetMyReview(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())
	if player == nil {
		renderError(w, apierror.Unauthorized())
		return
	}
	if player.Role == db.PlayerRoleAdmin {
		renderError(w, apierror.Forbidden("admins cannot submit reviews"))
		return
	}

	review, err := h.Store.GetMyReview(r.Context(), player.ID)
	if err == db.ErrNotFound {
		renderJSON(w, http.StatusOK, map[string]any{"review": nil})
		return
	}
	if err != nil {
		renderError(w, apierror.Internal("failed to fetch review"))
		return
	}
	renderJSON(w, http.StatusOK, review)
}

func (h *ReviewHandler) UpsertMyReview(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())
	if player == nil {
		renderError(w, apierror.Unauthorized())
		return
	}
	if player.Role == db.PlayerRoleAdmin {
		renderError(w, apierror.Forbidden("admins cannot submit reviews"))
		return
	}

	var body struct {
		Rating  int     `json:"rating"`
		Comment *string `json:"comment"`
	}
	if err := decodeJSON(r, &body); err != nil {
		renderError(w, err)
		return
	}
	if body.Rating < 1 || body.Rating > 5 {
		renderError(w, apierror.Unprocessable("rating must be between 1 and 5"))
		return
	}
	if body.Comment != nil && len(*body.Comment) > 500 {
		renderError(w, apierror.Unprocessable("comment must be at most 500 characters"))
		return
	}

	review, err := h.Store.UpsertReview(r.Context(), player.ID, body.Rating, body.Comment)
	if err != nil {
		renderError(w, apierror.Internal("failed to save review"))
		return
	}
	renderJSON(w, http.StatusOK, review)
}

func (h *ReviewHandler) GetSummary(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())
	if player == nil || player.Role != db.PlayerRoleAdmin {
		renderError(w, apierror.Forbidden("admin only"))
		return
	}

	summary, err := h.Store.GetReviewSummary(r.Context())
	if err != nil {
		renderError(w, apierror.Internal("failed to fetch summary"))
		return
	}
	renderJSON(w, http.StatusOK, summary)
}

func (h *ReviewHandler) ListReviews(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())
	if player == nil || player.Role != db.PlayerRoleAdmin {
		renderError(w, apierror.Forbidden("admin only"))
		return
	}

	var ratings []int
	if ratingStr := r.URL.Query().Get("rating"); ratingStr != "" {
		for _, s := range strings.Split(ratingStr, ",") {
			n, err := strconv.Atoi(strings.TrimSpace(s))
			if err != nil || n < 1 || n > 5 {
				renderError(w, apierror.Unprocessable("invalid rating filter"))
				return
			}
			ratings = append(ratings, n)
		}
	}

	orderBy := r.URL.Query().Get("order_by")
	if orderBy != "rating" {
		orderBy = "created_at"
	}

	page := 1
	pageSize := 20
	if p := r.URL.Query().Get("page"); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 0 {
			page = n
		}
	}
	if ps := r.URL.Query().Get("page_size"); ps != "" {
		if n, err := strconv.Atoi(ps); err == nil && n > 0 && n <= 100 {
			pageSize = n
		}
	}

	result, err := h.Store.ListReviews(r.Context(), ratings, orderBy, page, pageSize)
	if err != nil {
		renderError(w, apierror.Internal("failed to list reviews"))
		return
	}
	renderJSON(w, http.StatusOK, result)
}
