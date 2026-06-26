package db

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Points per voting position (matches Python POINTS dict).
var VotePoints = map[int]int{1: 10, 2: 8, 3: 6, 4: 4, 5: 2}

type VoteTop5Item struct {
	PlayerID uuid.UUID `json:"player_id"`
	Position int       `json:"position"`
}

type VoteTop5Result struct {
	Position  int       `json:"position"`
	PlayerID  uuid.UUID `json:"player_id"`
	Name      string    `json:"name"`
	Nickname  *string   `json:"nickname"`
	AvatarURL *string   `json:"avatar_url"`
	Points    int       `json:"points"`
}

type VoteFlopResult struct {
	PlayerID  uuid.UUID `json:"player_id"`
	Name      string    `json:"name"`
	Nickname  *string   `json:"nickname"`
	AvatarURL *string   `json:"avatar_url"`
	Votes     int       `json:"votes"`
}

type VoteResults struct {
	Top5        []VoteTop5Result `json:"top5"`
	Flop        []VoteFlopResult `json:"flop"`
	TotalVoters int              `json:"total_voters"`
}

type BallotTop5Item struct {
	Position int       `json:"position"`
	PlayerID uuid.UUID `json:"player_id"`
	Name     string    `json:"name"`
	Nickname *string   `json:"nickname"`
}

type BallotFlopItem struct {
	PlayerID uuid.UUID `json:"player_id"`
	Name     string    `json:"name"`
	Nickname *string   `json:"nickname"`
}

type Ballot struct {
	VoterID        uuid.UUID        `json:"voter_id"`
	VoterName      string           `json:"voter_name"`
	VoterNickname  *string          `json:"voter_nickname"`
	VoterAvatarURL *string          `json:"voter_avatar_url"`
	Top5           []BallotTop5Item `json:"top5"`
	Flop           *BallotFlopItem  `json:"flop"`
}

type PendingVoteItem struct {
	MatchID       uuid.UUID `json:"match_id"`
	MatchHash     string    `json:"match_hash"`
	MatchNumber   int       `json:"match_number"`
	GroupName     string    `json:"group_name"`
	TimeLabel     string    `json:"time_label"`
	VoterCount    int       `json:"voter_count"`
	EligibleCount int       `json:"eligible_count"`
	ClosesAt      time.Time `json:"-"`
}

func HasVoted(ctx context.Context, pool *pgxpool.Pool, matchID, voterID uuid.UUID) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM match_votes WHERE match_id=$1 AND voter_id=$2)`,
		matchID, voterID,
	).Scan(&exists)
	return exists, err
}

func VoterCount(ctx context.Context, pool *pgxpool.Pool, matchID uuid.UUID) (int, error) {
	var n int
	err := pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM match_votes WHERE match_id=$1`, matchID,
	).Scan(&n)
	return n, err
}

func VoterIDs(ctx context.Context, pool *pgxpool.Pool, matchID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := pool.Query(ctx,
		`SELECT voter_id FROM match_votes WHERE match_id=$1`, matchID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func SubmitVote(ctx context.Context, pool *pgxpool.Pool, matchID, voterID uuid.UUID, top5 []VoteTop5Item, flopID *uuid.UUID) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var voteID uuid.UUID
	err = tx.QueryRow(ctx,
		`INSERT INTO match_votes (match_id, voter_id) VALUES ($1, $2) RETURNING id`,
		matchID, voterID,
	).Scan(&voteID)
	if err != nil {
		return err
	}

	for _, item := range top5 {
		pts := VotePoints[item.Position]
		_, err = tx.Exec(ctx,
			`INSERT INTO match_vote_top5 (vote_id, player_id, position, points) VALUES ($1,$2,$3,$4)`,
			voteID, item.PlayerID, item.Position, pts,
		)
		if err != nil {
			return err
		}
	}

	if flopID != nil {
		_, err = tx.Exec(ctx,
			`INSERT INTO match_vote_flop (vote_id, player_id) VALUES ($1,$2)`,
			voteID, *flopID,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func MarkVoteNotified(ctx context.Context, pool *pgxpool.Pool, matchID uuid.UUID) error {
	_, err := pool.Exec(ctx,
		`UPDATE matches SET vote_notified=true WHERE id=$1`, matchID,
	)
	return err
}

func CloseVotingEarly(ctx context.Context, pool *pgxpool.Pool, matchID uuid.UUID) error {
	_, err := pool.Exec(ctx,
		`UPDATE matches SET vote_duration_hours=0 WHERE id=$1`, matchID,
	)
	return err
}

func GetPendingVotes(ctx context.Context, pool *pgxpool.Pool, playerID uuid.UUID) ([]PendingVoteItem, error) {
	rows, err := pool.Query(ctx, `
		SELECT
			m.id, m.hash, m.number, g.name,
			(SELECT COUNT(*)::int FROM match_votes mv2 WHERE mv2.match_id = m.id) AS voter_count,
			(SELECT COUNT(*)::int FROM attendances a2
			 WHERE a2.match_id = m.id AND a2.status = 'confirmed') AS eligible_count,
			(
			  (m.match_date + COALESCE(m.end_time, '23:59:00'::time))::timestamp
			  AT TIME ZONE 'America/Sao_Paulo'
			  + (m.vote_open_delay_minutes || ' minutes')::interval
			  + (m.vote_duration_hours || ' hours')::interval
			) AS closes_at
		FROM matches m
		JOIN groups g ON g.id = m.group_id
		JOIN attendances a ON a.match_id = m.id
			AND a.player_id = $1
			AND a.status = 'confirmed'
		WHERE m.status = 'closed'
		  AND g.voting_enabled = true
		  AND NOT EXISTS (
			SELECT 1 FROM match_votes mv
			WHERE mv.match_id = m.id AND mv.voter_id = $1
		  )
		  AND (
			(m.match_date + COALESCE(m.end_time, '23:59:00'::time))::timestamp
			AT TIME ZONE 'America/Sao_Paulo'
			+ (m.vote_open_delay_minutes || ' minutes')::interval
		  ) <= NOW()
		  AND (
			(m.match_date + COALESCE(m.end_time, '23:59:00'::time))::timestamp
			AT TIME ZONE 'America/Sao_Paulo'
			+ (m.vote_open_delay_minutes || ' minutes')::interval
			+ (m.vote_duration_hours || ' hours')::interval
		  ) >= NOW()
		ORDER BY m.match_date DESC, m.start_time DESC`,
		playerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []PendingVoteItem
	for rows.Next() {
		var item PendingVoteItem
		if err := rows.Scan(
			&item.MatchID, &item.MatchHash, &item.MatchNumber, &item.GroupName,
			&item.VoterCount, &item.EligibleCount, &item.ClosesAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func GetVoteResults(ctx context.Context, pool *pgxpool.Pool, matchID uuid.UUID) (*VoteResults, error) {
	// Top5 aggregated. Tie-break: among players tied on this match's points, favor the
	// one with FEWER points in the group ranking up to this match's date (asc); name asc
	// is the final deterministic tie-break.
	top5rows, err := pool.Query(ctx, `
		SELECT t.player_id, p.name, p.nickname, p.avatar_url, SUM(t.points) AS total_points
		FROM match_vote_top5 t
		JOIN match_votes v ON v.id = t.vote_id
		JOIN players p ON p.id = t.player_id
		WHERE v.match_id = $1
		GROUP BY t.player_id, p.name, p.nickname, p.avatar_url
		ORDER BY total_points DESC,
			(
				SELECT COALESCE(SUM(t2.points), 0)
				FROM match_vote_top5 t2
				JOIN match_votes v2 ON v2.id = t2.vote_id
				JOIN matches m2 ON m2.id = v2.match_id
				WHERE t2.player_id = t.player_id
				  AND m2.group_id = (SELECT group_id FROM matches WHERE id = $1)
				  AND m2.match_date <= (SELECT match_date FROM matches WHERE id = $1)
			) ASC,
			p.name ASC`,
		matchID,
	)
	if err != nil {
		return nil, err
	}
	defer top5rows.Close()

	var top5 []VoteTop5Result
	prevPts := -1
	pos, rank := 0, 0
	for top5rows.Next() {
		var r VoteTop5Result
		if err := top5rows.Scan(&r.PlayerID, &r.Name, &r.Nickname, &r.AvatarURL, &r.Points); err != nil {
			return nil, err
		}
		rank++
		if r.Points != prevPts {
			pos = rank
			prevPts = r.Points
		}
		r.Position = pos
		top5 = append(top5, r)
	}
	if err := top5rows.Err(); err != nil {
		return nil, err
	}

	// Flop aggregated
	floprows, err := pool.Query(ctx, `
		SELECT f.player_id, p.name, p.nickname, p.avatar_url, COUNT(f.id) AS vote_count
		FROM match_vote_flop f
		JOIN match_votes v ON v.id = f.vote_id
		JOIN players p ON p.id = f.player_id
		WHERE v.match_id = $1
		GROUP BY f.player_id, p.name, p.nickname, p.avatar_url
		ORDER BY vote_count DESC`,
		matchID,
	)
	if err != nil {
		return nil, err
	}
	defer floprows.Close()

	var flop []VoteFlopResult
	maxFlop := -1
	for floprows.Next() {
		var r VoteFlopResult
		if err := floprows.Scan(&r.PlayerID, &r.Name, &r.Nickname, &r.AvatarURL, &r.Votes); err != nil {
			return nil, err
		}
		if maxFlop < 0 {
			maxFlop = r.Votes
		}
		if r.Votes == maxFlop {
			flop = append(flop, r)
		}
	}
	if err := floprows.Err(); err != nil {
		return nil, err
	}

	total, err := VoterCount(ctx, pool, matchID)
	if err != nil {
		return nil, err
	}

	return &VoteResults{Top5: top5, Flop: flop, TotalVoters: total}, nil
}

func GetVoteBallots(ctx context.Context, pool *pgxpool.Pool, matchID uuid.UUID) ([]Ballot, error) {
	// Votes with voter info
	voteRows, err := pool.Query(ctx, `
		SELECT v.id, v.voter_id, p.name, p.nickname, p.avatar_url
		FROM match_votes v
		JOIN players p ON p.id = v.voter_id
		WHERE v.match_id = $1
		ORDER BY v.submitted_at`,
		matchID,
	)
	if err != nil {
		return nil, err
	}
	defer voteRows.Close()

	type voteRow struct {
		id      uuid.UUID
		voterID uuid.UUID
		name    string
		nick    *string
		avatar  *string
	}
	var votes []voteRow
	for voteRows.Next() {
		var r voteRow
		if err := voteRows.Scan(&r.id, &r.voterID, &r.name, &r.nick, &r.avatar); err != nil {
			return nil, err
		}
		votes = append(votes, r)
	}
	if err := voteRows.Err(); err != nil {
		return nil, err
	}
	if len(votes) == 0 {
		return []Ballot{}, nil
	}

	voteIDs := make([]uuid.UUID, len(votes))
	for i, v := range votes {
		voteIDs[i] = v.id
	}

	// Top5 entries per vote
	top5Map := map[uuid.UUID][]BallotTop5Item{}
	t5rows, err := pool.Query(ctx, `
		SELECT t.vote_id, t.position, t.player_id, p.name, p.nickname
		FROM match_vote_top5 t
		JOIN players p ON p.id = t.player_id
		WHERE t.vote_id = ANY($1)
		ORDER BY t.vote_id, t.position`,
		voteIDs,
	)
	if err != nil {
		return nil, err
	}
	defer t5rows.Close()
	for t5rows.Next() {
		var vid uuid.UUID
		var item BallotTop5Item
		if err := t5rows.Scan(&vid, &item.Position, &item.PlayerID, &item.Name, &item.Nickname); err != nil {
			return nil, err
		}
		top5Map[vid] = append(top5Map[vid], item)
	}

	// Flop entries per vote
	flopMap := map[uuid.UUID]*BallotFlopItem{}
	frows, err := pool.Query(ctx, `
		SELECT f.vote_id, f.player_id, p.name, p.nickname
		FROM match_vote_flop f
		JOIN players p ON p.id = f.player_id
		WHERE f.vote_id = ANY($1)`,
		voteIDs,
	)
	if err != nil {
		return nil, err
	}
	defer frows.Close()
	for frows.Next() {
		var vid uuid.UUID
		item := &BallotFlopItem{}
		if err := frows.Scan(&vid, &item.PlayerID, &item.Name, &item.Nickname); err != nil {
			return nil, err
		}
		flopMap[vid] = item
	}

	ballots := make([]Ballot, len(votes))
	for i, v := range votes {
		top5 := top5Map[v.id]
		if top5 == nil {
			top5 = []BallotTop5Item{}
		}
		ballots[i] = Ballot{
			VoterID:        v.voterID,
			VoterName:      v.name,
			VoterNickname:  v.nick,
			VoterAvatarURL: v.avatar,
			Top5:           top5,
			Flop:           flopMap[v.id],
		}
	}
	return ballots, nil
}
