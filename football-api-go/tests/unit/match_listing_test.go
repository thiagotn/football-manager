package unit_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/thiagotn/football-manager/football-api-go/internal/services"
)

// helpers ────────────────────────────────────────────────────────────────────

func mkMatch(status, matchDate string, opts ...func(*services.ListingMatch)) services.ListingMatch {
	m := services.ListingMatch{
		ID:                   uuid.New(),
		Status:               status,
		MatchDate:            matchDate,
		StartTime:            "20:00:00",
		EndTime:              strPtr("22:00:00"),
		VoteOpenDelayMinutes: 20,
		VoteDurationHours:    24,
	}
	for _, o := range opts {
		o(&m)
	}
	return m
}

func endTimeAt(t time.Time) func(*services.ListingMatch) {
	return func(m *services.ListingMatch) {
		m.MatchDate = t.Format("2006-01-02")
		m.StartTime = t.Format("15:04:05")
		s := t.Format("15:04:05")
		m.EndTime = &s
	}
}

// Cenário A: status open → is_current=true, voting_status=not_open
func TestClassify_OpenMatch_IsCurrent(t *testing.T) {
	future := time.Now().AddDate(0, 0, 7).Format("2006-01-02")
	m := mkMatch("open", future)

	res := services.ClassifyMatches([]services.ListingMatch{m})[m.ID]

	assert.True(t, res.IsCurrent)
	assert.Equal(t, services.VotingStatusNotOpen, res.VotingStatus)
}

// Cenário B: closed, end_time há 2h, voting ainda aberta → is_current=true
func TestClassify_ClosedVotingOpen_IsCurrent(t *testing.T) {
	// BRT é UTC-3. Pra simular end_time há 2h em BRT, pega "agora em BRT" e subtrai 2h.
	brtNow := time.Now().UTC().Add(-3 * time.Hour)
	endTwoHoursAgo := brtNow.Add(-2 * time.Hour)
	m := mkMatch("closed", "", endTimeAt(endTwoHoursAgo))

	res := services.ClassifyMatches([]services.ListingMatch{m})[m.ID]

	assert.True(t, res.IsCurrent)
	assert.Equal(t, services.VotingStatusOpen, res.VotingStatus)
}

// Cenário C: closed antigo, voting fechada, OUTRO match aberto no grupo → is_current=false
func TestClassify_ClosedVotingClosed_WithFuture_IsNotCurrent(t *testing.T) {
	oldDate := time.Now().AddDate(0, 0, -10).Format("2006-01-02")
	futureDate := time.Now().AddDate(0, 0, 5).Format("2006-01-02")
	oldMatch := mkMatch("closed", oldDate)
	futureMatch := mkMatch("open", futureDate)

	results := services.ClassifyMatches([]services.ListingMatch{oldMatch, futureMatch})

	assert.False(t, results[oldMatch.ID].IsCurrent)
	assert.Equal(t, services.VotingStatusClosed, results[oldMatch.ID].VotingStatus)
	assert.True(t, results[futureMatch.ID].IsCurrent)
}

// Cenário D: único match do grupo está closed com voting fechada → ainda is_current
func TestClassify_UniqueClosedNoFuture_IsCurrent(t *testing.T) {
	oldDate := time.Now().AddDate(0, 0, -10).Format("2006-01-02")
	m := mkMatch("closed", oldDate)

	res := services.ClassifyMatches([]services.ListingMatch{m})[m.ID]

	assert.True(t, res.IsCurrent)
	assert.Equal(t, services.VotingStatusClosed, res.VotingStatus)
}

// Cenário E: 2 closed sem voting nem próxima → só a mais recente é is_current
func TestClassify_TwoClosed_OnlyMostRecentIsCurrent(t *testing.T) {
	older := mkMatch("closed", time.Now().AddDate(0, 0, -20).Format("2006-01-02"))
	recent := mkMatch("closed", time.Now().AddDate(0, 0, -5).Format("2006-01-02"))

	results := services.ClassifyMatches([]services.ListingMatch{older, recent})

	assert.False(t, results[older.ID].IsCurrent)
	assert.True(t, results[recent.ID].IsCurrent)
}

// Sanidade: status in_progress sempre is_current
func TestClassify_InProgress_IsCurrent(t *testing.T) {
	today := time.Now().Format("2006-01-02")
	m := mkMatch("in_progress", today)

	res := services.ClassifyMatches([]services.ListingMatch{m})[m.ID]

	assert.True(t, res.IsCurrent)
}

// Sanidade: lista vazia devolve mapa vazio
func TestClassify_EmptyList_EmptyMap(t *testing.T) {
	results := services.ClassifyMatches(nil)
	assert.Empty(t, results)
}

// VotingWindow + ComputeVotingStatus: parse error devolve closed (modo seguro)
func TestComputeVotingStatus_InvalidDate_ReturnsClosed(t *testing.T) {
	vs := services.ComputeVotingStatus(services.VotingInput{
		MatchDate: "not-a-date",
	})
	assert.Equal(t, services.VotingStatusClosed, vs)
}

// VotingWindow: end_time nil usa default 23:59
func TestVotingWindow_NoEndTime_DefaultsToEndOfDay(t *testing.T) {
	in := services.VotingInput{
		MatchDate:            "2026-06-01",
		StartTime:            "20:00:00",
		EndTime:              nil,
		VoteOpenDelayMinutes: 20,
		VoteDurationHours:    24,
	}
	opens, closes, ok := services.VotingWindow(in)
	assert.True(t, ok)
	// end_time default 23:59 BRT + 20min delay = 00:19 UTC do dia seguinte
	assert.True(t, closes.After(opens))
	assert.Equal(t, 24*time.Hour, closes.Sub(opens))
}
