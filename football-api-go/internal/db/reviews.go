package db

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AppReview struct {
	ID        uuid.UUID `json:"id"`
	PlayerID  uuid.UUID `json:"player_id"`
	PlayerName string    `json:"player_name"`
	Rating    int       `json:"rating"`
	Comment   *string   `json:"comment"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type DistributionEntry struct {
	Count   int     `json:"count"`
	Percent float64 `json:"percent"`
}

type ReviewSummary struct {
	Average      float64                   `json:"average"`
	Total        int                       `json:"total"`
	Distribution map[string]DistributionEntry `json:"distribution"` // "1"-"5" -> {count, percent}
}

type ReviewPage struct {
	Items      []AppReview `json:"items"`
	Total      int         `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalPages int         `json:"total_pages"`
}

func GetMyReview(ctx context.Context, pool *pgxpool.Pool, playerID uuid.UUID) (*AppReview, error) {
	var r AppReview
	err := pool.QueryRow(ctx,
		`SELECT id, player_id, rating, comment, created_at, updated_at
		 FROM app_reviews WHERE player_id=$1`,
		playerID,
	).Scan(&r.ID, &r.PlayerID, &r.Rating, &r.Comment, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		return nil, ErrNotFound
	}
	return &r, nil
}

func UpsertReview(ctx context.Context, pool *pgxpool.Pool, playerID uuid.UUID, rating int, comment *string) (*AppReview, error) {
	var r AppReview
	err := pool.QueryRow(ctx,
		`INSERT INTO app_reviews (player_id, rating, comment)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (player_id) DO UPDATE
		 SET rating=EXCLUDED.rating, comment=EXCLUDED.comment, updated_at=NOW()
		 RETURNING id, player_id, rating, comment, created_at, updated_at`,
		playerID, rating, comment,
	).Scan(&r.ID, &r.PlayerID, &r.Rating, &r.Comment, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func GetReviewSummary(ctx context.Context, pool *pgxpool.Pool) (*ReviewSummary, error) {
	rows, err := pool.Query(ctx,
		`SELECT rating, COUNT(*) FROM app_reviews GROUP BY rating ORDER BY rating`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dist := map[int]int{1: 0, 2: 0, 3: 0, 4: 0, 5: 0}
	total := 0
	sum := 0
	for rows.Next() {
		var rating, count int
		if err := rows.Scan(&rating, &count); err != nil {
			return nil, err
		}
		dist[rating] = count
		total += count
		sum += rating * count
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	avg := 0.0
	if total > 0 {
		avg = float64(sum) / float64(total)
	}

	// Build distribution with percentages
	distribution := make(map[string]DistributionEntry)
	for i := 1; i <= 5; i++ {
		count := dist[i]
		percent := 0.0
		if total > 0 {
			percent = float64(count) / float64(total) * 100
			percent = math.Round(percent*10) / 10 // Round to 1 decimal place
		}
		distribution[fmt.Sprintf("%d", i)] = DistributionEntry{
			Count:   count,
			Percent: percent,
		}
	}

	return &ReviewSummary{
		Average:      math.Round(avg*10) / 10, // Round to 1 decimal place
		Total:        total,
		Distribution: distribution,
	}, nil
}

func ListReviews(ctx context.Context, pool *pgxpool.Pool, ratings []int, orderBy string, page, pageSize int) (*ReviewPage, error) {
	offset := (page - 1) * pageSize

	// Build rating filter
	whereClause := ""
	var args []any
	if len(ratings) > 0 {
		whereClause = "WHERE ar.rating = ANY($1)"
		args = append(args, ratings)
	}

	// Count total
	var total int
	err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM app_reviews ar `+whereClause, args...).Scan(&total)
	if err != nil {
		return nil, err
	}

	// Validate order
	if orderBy != "rating" {
		orderBy = "created_at"
	}

	// Fetch page
	limitArgs := append(args, pageSize, offset)
	limitPlaceholders := ""
	if len(ratings) > 0 {
		limitPlaceholders = " LIMIT $2 OFFSET $3"
	} else {
		limitPlaceholders = " LIMIT $1 OFFSET $2"
	}

	rows, err := pool.Query(ctx,
		`SELECT ar.id, ar.player_id, ar.rating, ar.comment, ar.created_at, ar.updated_at, p.name
		 FROM app_reviews ar
		 LEFT JOIN players p ON p.id = ar.player_id `+whereClause+`
		 ORDER BY `+orderBy+` DESC`+limitPlaceholders,
		limitArgs...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []AppReview{}
	for rows.Next() {
		var r AppReview
		if err := rows.Scan(&r.ID, &r.PlayerID, &r.Rating, &r.Comment, &r.CreatedAt, &r.UpdatedAt, &r.PlayerName); err != nil {
			return nil, err
		}
		items = append(items, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	totalPages := (total + pageSize - 1) / pageSize
	return &ReviewPage{Items: items, Total: total, Page: page, PageSize: pageSize, TotalPages: totalPages}, nil
}
