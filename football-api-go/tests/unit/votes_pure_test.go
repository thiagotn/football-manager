package unit_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/thiagotn/football-manager/football-api-go/internal/db"
)

// Pure functions from handlers/votes.go — testing them as helpers
// votingWindow, votingStatus, timeUntil, confirmedPlayerIDs

var brtLoc *time.Location

func init() {
	var err error
	brtLoc, err = time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		brtLoc = time.UTC
	}
}

func votingWindow(m *db.Match) (time.Time, time.Time) {
	const defaultEnd = "23:59:00"
	endStr := defaultEnd
	if m.EndTime != nil {
		endStr = *m.EndTime
	}

	endDT, err := time.ParseInLocation("2006-01-02T15:04:05", m.MatchDate+"T"+endStr, brtLoc)
	if err != nil {
		endDT, _ = time.ParseInLocation("2006-01-02T15:04:05", m.MatchDate+"T"+defaultEnd, brtLoc)
	}

	opensAt := endDT.Add(time.Duration(m.VoteOpenDelayMinutes) * time.Minute)
	closesAt := opensAt.Add(time.Duration(m.VoteDurationHours) * time.Hour)
	return opensAt, closesAt
}

func votingStatus(m *db.Match) string {
	now := time.Now().In(brtLoc)
	opensAt, closesAt := votingWindow(m)
	if now.Before(opensAt) {
		return "not_open"
	}
	if !now.After(closesAt) {
		return "open"
	}
	return "closed"
}

func timeUntil(target time.Time) string {
	diff := time.Until(target)
	if diff < 0 {
		diff = 0
	}
	h := int(diff.Hours())
	m := int(diff.Minutes()) % 60
	if h > 0 {
		if m > 0 {
			return fmt.Sprintf("%dh %dmin", h, m)
		}
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dmin", m)
}

func confirmedPlayerIDs(attendances []db.AttendanceWithPlayer) []uuid.UUID {
	var ids []uuid.UUID
	for _, a := range attendances {
		if a.Status == "confirmed" {
			ids = append(ids, a.PlayerID)
		}
	}
	return ids
}

// ── Tests for votingWindow ────────────────────────────────────────────────

func TestVotingWindow_DefaultEndTime(t *testing.T) {
	match := &db.Match{
		MatchDate:            "2025-12-25",
		StartTime:            "15:00:00",
		EndTime:              nil,
		VoteOpenDelayMinutes: 30,
		VoteDurationHours:    24,
	}

	opensAt, closesAt := votingWindow(match)

	// Parse expected times: 2025-12-25 23:59:00 BRT + 30 min
	expected, _ := time.ParseInLocation("2006-01-02T15:04:05", "2025-12-25T23:59:00", brtLoc)
	expectedOpens := expected.Add(30 * time.Minute)
	expectedCloses := expectedOpens.Add(24 * time.Hour)

	assert.Equal(t, expectedOpens, opensAt)
	assert.Equal(t, expectedCloses, closesAt)
}

func TestVotingWindow_CustomEndTime(t *testing.T) {
	endTime := "18:30:00"
	match := &db.Match{
		MatchDate:            "2025-12-25",
		StartTime:            "15:00:00",
		EndTime:              &endTime,
		VoteOpenDelayMinutes: 0,
		VoteDurationHours:    12,
	}

	opensAt, closesAt := votingWindow(match)

	// Parse expected: 2025-12-25 18:30:00 BRT + 0 min
	expected, _ := time.ParseInLocation("2006-01-02T15:04:05", "2025-12-25T18:30:00", brtLoc)
	expectedCloses := expected.Add(12 * time.Hour)

	assert.Equal(t, expected, opensAt)
	assert.Equal(t, expectedCloses, closesAt)
}

func TestVotingWindow_LargeDelayAndDuration(t *testing.T) {
	match := &db.Match{
		MatchDate:            "2025-01-15",
		StartTime:            "20:00:00",
		EndTime:              nil,
		VoteOpenDelayMinutes: 1440, // 24 hours
		VoteDurationHours:    48,
	}

	opensAt, closesAt := votingWindow(match)

	expected, _ := time.ParseInLocation("2006-01-02T15:04:05", "2025-01-15T23:59:00", brtLoc)
	expectedOpens := expected.Add(1440 * time.Minute)
	expectedCloses := expectedOpens.Add(48 * time.Hour)

	assert.Equal(t, expectedOpens, opensAt)
	assert.Equal(t, expectedCloses, closesAt)
}

// ── Tests for votingStatus ────────────────────────────────────────────────

func TestVotingStatus_NotOpen(t *testing.T) {
	// Match in the future (voting hasn't opened yet)
	tomorrow := time.Now().In(brtLoc).Add(24 * time.Hour)
	matchDate := fmt.Sprintf("%04d-%02d-%02d", tomorrow.Year(), tomorrow.Month(), tomorrow.Day())

	match := &db.Match{
		MatchDate:            matchDate,
		StartTime:            "15:00:00",
		EndTime:              nil,
		VoteOpenDelayMinutes: 120, // Opens 2 hours after match end
		VoteDurationHours:    24,
	}

	status := votingStatus(match)
	assert.Equal(t, "not_open", status)
}

func TestVotingStatus_Open(t *testing.T) {
	// Match from yesterday (voting should be open if duration hasn't passed)
	yesterday := time.Now().In(brtLoc).Add(-24 * time.Hour)
	matchDate := fmt.Sprintf("%04d-%02d-%02d", yesterday.Year(), yesterday.Month(), yesterday.Day())

	match := &db.Match{
		MatchDate:            matchDate,
		StartTime:            "15:00:00",
		EndTime:              nil,
		VoteOpenDelayMinutes: 0, // Opens immediately after match end
		VoteDurationHours:    24,
	}

	status := votingStatus(match)
	assert.Equal(t, "open", status)
}

func TestVotingStatus_Closed(t *testing.T) {
	// Match from 2 days ago (voting has closed)
	twoDaysAgo := time.Now().In(brtLoc).Add(-48 * time.Hour)
	matchDate := fmt.Sprintf("%04d-%02d-%02d", twoDaysAgo.Year(), twoDaysAgo.Month(), twoDaysAgo.Day())

	match := &db.Match{
		MatchDate:            matchDate,
		StartTime:            "15:00:00",
		EndTime:              nil,
		VoteOpenDelayMinutes: 0,
		VoteDurationHours:    24,
	}

	status := votingStatus(match)
	assert.Equal(t, "closed", status)
}

// ── Tests for timeUntil ───────────────────────────────────────────────────

func TestTimeUntil_FutureTime(t *testing.T) {
	target := time.Now().Add(90*time.Minute + 30*time.Second)
	result := timeUntil(target)
	// Allow for timing variation
	assert.Contains(t, result, "1h")
}

func TestTimeUntil_LessThanOneHour(t *testing.T) {
	target := time.Now().Add(45*time.Minute + 30*time.Second)
	result := timeUntil(target)
	// Should be in 40-50 minute range
	assert.Contains(t, result, "min")
	assert.NotContains(t, result, "h")
}

func TestTimeUntil_ExactHour(t *testing.T) {
	target := time.Now().Add(120*time.Minute + 30*time.Second)
	result := timeUntil(target)
	// Should show hours
	assert.Contains(t, result, "h")
}

func TestTimeUntil_ZeroMinutes(t *testing.T) {
	target := time.Now().Add(1*time.Minute + 30*time.Second)
	result := timeUntil(target)
	// Should be 1 or 2 minutes
	assert.Contains(t, result, "min")
	assert.NotContains(t, result, "h")
}

func TestTimeUntil_PastTime(t *testing.T) {
	target := time.Now().Add(-1 * time.Hour)
	result := timeUntil(target)
	assert.Equal(t, "0min", result)
}

func TestTimeUntil_LargeHourDuration(t *testing.T) {
	target := time.Now().Add(74*time.Hour + 30*time.Second)
	result := timeUntil(target)
	// Should show hours, with or without minutes
	assert.Contains(t, result, "h")
}

// ── Tests for confirmedPlayerIDs ──────────────────────────────────────────

func TestConfirmedPlayerIDs_AllConfirmed(t *testing.T) {
	p1, p2, p3 := uuid.New(), uuid.New(), uuid.New()
	attendances := []db.AttendanceWithPlayer{
		{PlayerID: p1, Status: "confirmed"},
		{PlayerID: p2, Status: "confirmed"},
		{PlayerID: p3, Status: "confirmed"},
	}

	ids := confirmedPlayerIDs(attendances)
	assert.Len(t, ids, 3)
	assert.Contains(t, ids, p1)
	assert.Contains(t, ids, p2)
	assert.Contains(t, ids, p3)
}

func TestConfirmedPlayerIDs_MixedStatuses(t *testing.T) {
	p1, p2, p3 := uuid.New(), uuid.New(), uuid.New()
	attendances := []db.AttendanceWithPlayer{
		{PlayerID: p1, Status: "confirmed"},
		{PlayerID: p2, Status: "declined"},
		{PlayerID: p3, Status: "confirmed"},
	}

	ids := confirmedPlayerIDs(attendances)
	assert.Len(t, ids, 2)
	assert.Contains(t, ids, p1)
	assert.Contains(t, ids, p3)
	assert.NotContains(t, ids, p2)
}

func TestConfirmedPlayerIDs_NoneConfirmed(t *testing.T) {
	attendances := []db.AttendanceWithPlayer{
		{PlayerID: uuid.New(), Status: "declined"},
		{PlayerID: uuid.New(), Status: "pending"},
	}

	ids := confirmedPlayerIDs(attendances)
	assert.Len(t, ids, 0)
}

func TestConfirmedPlayerIDs_Empty(t *testing.T) {
	attendances := []db.AttendanceWithPlayer{}
	ids := confirmedPlayerIDs(attendances)
	assert.Len(t, ids, 0)
}
