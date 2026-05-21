package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thiagotn/football-manager/football-api-go/internal/db"
)

var monthsPT = [...]string{"jan", "fev", "mar", "abr", "mai", "jun", "jul", "ago", "set", "out", "nov", "dez"}

func fmtDatePT(d time.Time) string {
	return fmt.Sprintf("%d de %s", d.Day(), monthsPT[d.Month()-1])
}

func generateMatchHash() (string, error) {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b)[:10], nil
}

// matchVotingClosed reports whether the voting window for m has already passed.
func matchVotingClosed(m *db.Match) bool {
	matchDate, err := time.Parse("2006-01-02", m.MatchDate)
	if err != nil {
		return false
	}
	baseTimeStr := m.StartTime
	if m.EndTime != nil {
		baseTimeStr = *m.EndTime
	}
	baseTime, err := time.Parse("15:04:05", baseTimeStr)
	if err != nil {
		return false
	}
	// Reconstruct as BRT (UTC-3) wall-clock moment in UTC
	matchRef := time.Date(
		matchDate.Year(), matchDate.Month(), matchDate.Day(),
		baseTime.Hour(), baseTime.Minute(), baseTime.Second(), 0,
		time.UTC,
	).Add(3 * time.Hour) // shift BRT → UTC

	votingOpens := matchRef.Add(time.Duration(m.VoteOpenDelayMinutes) * time.Minute)
	votingCloses := votingOpens.Add(time.Duration(m.VoteDurationHours) * time.Hour)
	return time.Now().UTC().After(votingCloses)
}

// sendPushToGroup sends a push notification to all players in a group (best-effort stub).
func sendPushToGroup(ctx context.Context, pool *pgxpool.Pool, groupID uuid.UUID, matchURL, title, body string) {
	playerIDs, err := db.GetGroupMemberPlayerIDs(ctx, pool, groupID)
	if err != nil {
		return
	}
	for _, pid := range playerIDs {
		subs, err := db.GetPushSubscriptionsForPlayer(ctx, pool, pid)
		if err != nil {
			continue
		}
		for i := range subs {
			_ = SendPushToSubscription(
				subs[i].Endpoint, subs[i].P256dh, subs[i].Auth,
				"", "",
				PushNotification{Title: title, Body: body, URL: matchURL},
			)
		}
	}
}

// RunRecurrence creates the next recurring match for each eligible group.
// A group is eligible when: recurrence_enabled=true, no open match exists,
// the last match date has passed, and its voting window is closed.
// Returns the number of matches created.
func RunRecurrence(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	groups, err := db.GetGroupsWithRecurrence(ctx, pool)
	if err != nil {
		return 0, fmt.Errorf("recurrence: list groups: %w", err)
	}

	brtNow := time.Now().UTC().Add(-3 * time.Hour)
	today := time.Date(brtNow.Year(), brtNow.Month(), brtNow.Day(), 0, 0, 0, 0, time.UTC)

	created := 0
	for _, group := range groups {
		hasOpen, err := db.HasOpenMatch(ctx, pool, group.ID)
		if err != nil {
			slog.Error("recurrence: check open match", "group_id", group.ID, "error", err)
			continue
		}
		if hasOpen {
			continue
		}

		lastMatch, err := db.GetLastMatch(ctx, pool, group.ID)
		if err != nil || lastMatch == nil {
			continue
		}

		lastDate, err := time.Parse("2006-01-02", lastMatch.MatchDate)
		if err != nil {
			continue
		}

		if lastDate.After(today) {
			continue
		}
		if lastDate.Equal(today) && lastMatch.Status != "closed" {
			continue
		}
		if !matchVotingClosed(lastMatch) {
			continue
		}

		nextDate := lastDate.Add(7 * 24 * time.Hour)

		var hash string
		for range 5 {
			h, err := generateMatchHash()
			if err != nil {
				break
			}
			existing, _ := db.GetMatchByHash(ctx, pool, h)
			if existing == nil {
				hash = h
				break
			}
		}
		if hash == "" {
			slog.Error("recurrence: unique hash not found", "group_id", group.ID)
			continue
		}

		num, err := db.NextMatchNumber(ctx, pool, group.ID)
		if err != nil {
			slog.Error("recurrence: next match number", "group_id", group.ID, "error", err)
			continue
		}

		newMatch, err := db.CreateMatchForRecurrence(ctx, pool, db.CreateMatchRecurrenceParams{
			GroupID:              group.ID,
			Hash:                 hash,
			Number:               num,
			MatchDate:            nextDate.Format("2006-01-02"),
			StartTime:            lastMatch.StartTime,
			EndTime:              lastMatch.EndTime,
			Location:             lastMatch.Location,
			Address:              lastMatch.Address,
			CourtType:            lastMatch.CourtType,
			PlayersPerTeam:       lastMatch.PlayersPerTeam,
			MaxPlayers:           lastMatch.MaxPlayers,
			Notes:                lastMatch.Notes,
			VoteOpenDelayMinutes: group.VoteOpenDelayMinutes,
			VoteDurationHours:    group.VoteDurationHours,
		})
		if err != nil {
			slog.Error("recurrence: create match", "group_id", group.ID, "error", err)
			continue
		}

		playerIDs, err := db.GetGroupMemberPlayerIDs(ctx, pool, group.ID)
		if err != nil {
			slog.Error("recurrence: get member ids", "group_id", group.ID, "error", err)
			continue
		}
		if err := db.CreateAttendances(ctx, pool, newMatch.ID, playerIDs); err != nil {
			slog.Error("recurrence: create attendances", "group_id", group.ID, "error", err)
			continue
		}

		matchURL := "https://rachao.app/match/" + hash
		sendPushToGroup(ctx, pool, group.ID, matchURL,
			"⚽ Novo rachão — "+group.Name,
			"Partida em "+fmtDatePT(nextDate)+". Confirme sua presença!",
		)

		created++
		slog.Info("recurrence: match created",
			"group_id", group.ID,
			"match_date", nextDate.Format("2006-01-02"),
			"players", len(playerIDs),
		)
	}
	return created, nil
}

// RunStatusSyncJob closes past matches and transitions today's matches to in_progress.
func RunStatusSyncJob(ctx context.Context, pool *pgxpool.Pool) error {
	candidates, err := db.GetInProgressCandidates(ctx, pool)
	if err != nil {
		return fmt.Errorf("status sync: get candidates: %w", err)
	}

	closed, err := db.ClosePastMatches(ctx, pool)
	if err != nil {
		return fmt.Errorf("status sync: close past: %w", err)
	}
	if closed > 0 {
		slog.Info("status_sync: closed past matches", "count", closed)
	}

	for _, c := range candidates {
		ids := []uuid.UUID{c.ID}
		if err := db.TransitionToInProgress(ctx, pool, ids); err != nil {
			slog.Error("status_sync: transition", "match_id", c.ID, "error", err)
			continue
		}
		matchURL := "https://rachao.app/match/" + c.Hash
		sendPushToGroup(ctx, pool, c.GroupID, matchURL,
			"⚽ Bola rolando! — "+c.GroupName,
			"A partida de hoje já começou! 🎉",
		)
	}
	return nil
}
