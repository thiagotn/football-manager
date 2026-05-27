package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thiagotn/football-manager/football-api-go/internal/apierror"
	"github.com/thiagotn/football-manager/football-api-go/internal/db"
)

type RankingStore interface {
	GetTopRanking(ctx context.Context, year *int, month *int) ([]db.RankingTopItem, error)
	GetFlopRanking(ctx context.Context, year *int, month *int) ([]db.RankingFlopItem, error)
}

type pgRankingStore struct {
	pool *pgxpool.Pool
}

func (s *pgRankingStore) GetTopRanking(ctx context.Context, year *int, month *int) ([]db.RankingTopItem, error) {
	return db.GetTopRanking(ctx, s.pool, year, month)
}

func (s *pgRankingStore) GetFlopRanking(ctx context.Context, year *int, month *int) ([]db.RankingFlopItem, error) {
	return db.GetFlopRanking(ctx, s.pool, year, month)
}

type RankingHandler struct {
	Store RankingStore
}

func NewRankingHandler(pool *pgxpool.Pool) *RankingHandler {
	return &RankingHandler{Store: &pgRankingStore{pool: pool}}
}

func (h *RankingHandler) GetRanking(w http.ResponseWriter, r *http.Request) {
	rankType := r.URL.Query().Get("type")
	if rankType == "" {
		rankType = "top"
	}
	if rankType != "top" && rankType != "flop" {
		renderError(w, apierror.Unprocessable("type must be 'top' or 'flop'"))
		return
	}

	var year *int
	var month *int

	if yearStr := r.URL.Query().Get("year"); yearStr != "" {
		y, err := strconv.Atoi(yearStr)
		if err != nil || y < 2024 || y > 2100 {
			renderError(w, apierror.Unprocessable("year must be between 2024 and 2100"))
			return
		}
		year = &y
	}

	if monthStr := r.URL.Query().Get("month"); monthStr != "" {
		if year == nil {
			renderError(w, apierror.Unprocessable("month requires year to be provided"))
			return
		}
		m, err := strconv.Atoi(monthStr)
		if err != nil || m < 1 || m > 12 {
			renderError(w, apierror.Unprocessable("month must be between 1 and 12"))
			return
		}
		month = &m
	}

	if rankType == "top" {
		items, err := h.Store.GetTopRanking(r.Context(), year, month)
		if err != nil {
			renderError(w, apierror.Internal("failed to fetch ranking"))
			return
		}
		if items == nil {
			items = []db.RankingTopItem{}
		}
		renderJSON(w, http.StatusOK, map[string]any{
			"year":  year,
			"month": month,
			"type":  rankType,
			"items": items,
		})
		return
	}

	items, err := h.Store.GetFlopRanking(r.Context(), year, month)
	if err != nil {
		renderError(w, apierror.Internal("failed to fetch ranking"))
		return
	}
	if items == nil {
		items = []db.RankingFlopItem{}
	}
	renderJSON(w, http.StatusOK, map[string]any{
		"year":  year,
		"month": month,
		"type":  rankType,
		"items": items,
	})
}
