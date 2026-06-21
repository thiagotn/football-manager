package db

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const minEligibleVoters = 10

type RankingTopItem struct {
	Position    int       `json:"position"`
	PlayerID    uuid.UUID `json:"player_id"`
	Name        string    `json:"name"`
	Nickname    *string   `json:"nickname"`
	AvatarURL   *string   `json:"avatar_url"`
	TotalPoints int       `json:"total_points"`
}

type RankingFlopItem struct {
	Position       int       `json:"position"`
	PlayerID       uuid.UUID `json:"player_id"`
	Name           string    `json:"name"`
	Nickname       *string   `json:"nickname"`
	AvatarURL      *string   `json:"avatar_url"`
	TotalFlopVotes int       `json:"total_flop_votes"`
}

func GetTopRanking(ctx context.Context, pool *pgxpool.Pool, year *int, month *int) ([]RankingTopItem, error) {
	periodFilter, args := buildPeriodFilter(year, month)

	query := `
		SELECT t.player_id, p.name, p.nickname, p.avatar_url, SUM(t.points)::int AS total_points
		FROM match_vote_top5 t
		JOIN match_votes v ON v.id = t.vote_id
		JOIN players p ON p.id = t.player_id
		WHERE v.match_id IN (
			SELECT match_id FROM attendances
			WHERE status='confirmed'
			GROUP BY match_id HAVING COUNT(*) >= ` + intToStr(minEligibleVoters) + `
		)
		AND p.role != 'admin'
		-- Issue #3: exclui pontos quando o jogador estava confirmado naquela
		-- partida E não tem voto registrado (penaliza free-rider).
		AND NOT EXISTS (
			SELECT 1 FROM attendances a
			WHERE a.match_id = v.match_id
			  AND a.player_id = t.player_id
			  AND a.status = 'confirmed'
			  AND NOT EXISTS (
			    SELECT 1 FROM match_votes mv2
			    WHERE mv2.match_id = a.match_id
			      AND mv2.voter_id = a.player_id
			  )
		)
		` + periodFilter + `
		GROUP BY t.player_id, p.name, p.nickname, p.avatar_url
		ORDER BY total_points DESC
		LIMIT 10`

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []RankingTopItem
	prevScore, pos, rank := -1, 0, 0
	for rows.Next() {
		var item RankingTopItem
		if err := rows.Scan(&item.PlayerID, &item.Name, &item.Nickname, &item.AvatarURL, &item.TotalPoints); err != nil {
			return nil, err
		}
		rank++
		if item.TotalPoints != prevScore {
			pos = rank
			prevScore = item.TotalPoints
		}
		item.Position = pos
		items = append(items, item)
	}
	return items, rows.Err()
}

func GetFlopRanking(ctx context.Context, pool *pgxpool.Pool, year *int, month *int) ([]RankingFlopItem, error) {
	periodFilter, args := buildPeriodFilter(year, month)

	query := `
		SELECT f.player_id, p.name, p.nickname, p.avatar_url, COUNT(f.id)::int AS total_flop_votes
		FROM match_vote_flop f
		JOIN match_votes v ON v.id = f.vote_id
		JOIN players p ON p.id = f.player_id
		WHERE v.match_id IN (
			SELECT match_id FROM attendances
			WHERE status='confirmed'
			GROUP BY match_id HAVING COUNT(*) >= ` + intToStr(minEligibleVoters) + `
		)
		AND p.role != 'admin'
		-- Issue #3: exclui votos de flop quando o jogador estava confirmado
		-- naquela partida E não votou (paridade com a regra do top).
		AND NOT EXISTS (
			SELECT 1 FROM attendances a
			WHERE a.match_id = v.match_id
			  AND a.player_id = f.player_id
			  AND a.status = 'confirmed'
			  AND NOT EXISTS (
			    SELECT 1 FROM match_votes mv2
			    WHERE mv2.match_id = a.match_id
			      AND mv2.voter_id = a.player_id
			  )
		)
		` + periodFilter + `
		GROUP BY f.player_id, p.name, p.nickname, p.avatar_url
		ORDER BY total_flop_votes DESC
		LIMIT 10`

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []RankingFlopItem
	prevScore, pos, rank := -1, 0, 0
	for rows.Next() {
		var item RankingFlopItem
		if err := rows.Scan(&item.PlayerID, &item.Name, &item.Nickname, &item.AvatarURL, &item.TotalFlopVotes); err != nil {
			return nil, err
		}
		rank++
		if item.TotalFlopVotes != prevScore {
			pos = rank
			prevScore = item.TotalFlopVotes
		}
		item.Position = pos
		items = append(items, item)
	}
	return items, rows.Err()
}

// buildPeriodFilter returns a SQL WHERE clause fragment and args slice for period filtering.
// Filters on match_votes.submitted_at joined through the query.
func buildPeriodFilter(year *int, month *int) (string, []any) {
	if year == nil {
		return "", nil
	}
	var start, end time.Time
	if month == nil {
		start = time.Date(*year, 1, 1, 0, 0, 0, 0, time.UTC)
		end = time.Date(*year+1, 1, 1, 0, 0, 0, 0, time.UTC)
	} else {
		start = time.Date(*year, time.Month(*month), 1, 0, 0, 0, 0, time.UTC)
		if *month == 12 {
			end = time.Date(*year+1, 1, 1, 0, 0, 0, 0, time.UTC)
		} else {
			end = time.Date(*year, time.Month(*month+1), 1, 0, 0, 0, 0, time.UTC)
		}
	}
	return `AND v.submitted_at >= $1 AND v.submitted_at < $2`, []any{start, end}
}

func intToStr(n int) string {
	s := ""
	if n == 0 {
		return "0"
	}
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}
