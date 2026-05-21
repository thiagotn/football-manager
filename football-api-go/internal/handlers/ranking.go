package handlers

import (
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thiagotn/football-manager/football-api-go/internal/apierror"
	"github.com/thiagotn/football-manager/football-api-go/internal/db"
)

type rankingHandler struct {
	pool *pgxpool.Pool
}

func NewRankingHandler(pool *pgxpool.Pool) *rankingHandler {
	return &rankingHandler{pool: pool}
}

func (h *rankingHandler) GetRanking(w http.ResponseWriter, r *http.Request) {
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
		items, err := db.GetTopRanking(r.Context(), h.pool, year, month)
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

	items, err := db.GetFlopRanking(r.Context(), h.pool, year, month)
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
