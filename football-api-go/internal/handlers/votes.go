package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thiagotn/football-manager/football-api-go/internal/apierror"
	"github.com/thiagotn/football-manager/football-api-go/internal/db"
	"github.com/thiagotn/football-manager/football-api-go/internal/middleware"
)

var brtLoc *time.Location

func init() {
	var err error
	brtLoc, err = time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		brtLoc = time.UTC
	}
}

type voteHandler struct {
	pool *pgxpool.Pool
}

func NewVoteHandler(pool *pgxpool.Pool) *voteHandler {
	return &voteHandler{pool: pool}
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

func (h *voteHandler) GetVoteStatus(w http.ResponseWriter, r *http.Request) {
	matchIDStr := chi.URLParam(r, "matchID")
	matchID, err := uuid.Parse(matchIDStr)
	if err != nil {
		renderError(w, apierror.NotFound("match not found"))
		return
	}

	player := middleware.PlayerFromCtx(r.Context())
	if player == nil {
		renderError(w, apierror.Unauthorized())
		return
	}

	match, err := db.GetMatchByID(r.Context(), h.pool, matchID)
	if err != nil {
		renderError(w, apierror.NotFound("match not found"))
		return
	}

	attendances, err := db.GetAttendancesForMatch(r.Context(), h.pool, matchID)
	if err != nil {
		renderError(w, apierror.Internal("failed to get attendances"))
		return
	}

	status := votingStatus(match)
	opensAt, closesAt := votingWindow(match)
	confirmedIDs := confirmedPlayerIDs(attendances)

	voterCount, err := db.VoterCount(r.Context(), h.pool, matchID)
	if err != nil {
		renderError(w, apierror.Internal("failed to get voter count"))
		return
	}

	hasVoted, err := db.HasVoted(r.Context(), h.pool, matchID, player.ID)
	if err != nil {
		renderError(w, apierror.Internal("failed to check vote"))
		return
	}

	var votedIDs []uuid.UUID
	if status == "open" || status == "closed" {
		if votedIDs, err = db.VoterIDs(r.Context(), h.pool, matchID); err != nil {
			renderError(w, apierror.Internal("failed to get voter ids"))
			return
		}
	}
	if votedIDs == nil {
		votedIDs = []uuid.UUID{}
	}

	if status == "open" && !match.VoteNotified {
		_ = db.MarkVoteNotified(r.Context(), h.pool, matchID)
	}

	var timeLabel string
	switch status {
	case "not_open":
		timeLabel = "Abre em " + timeUntil(opensAt)
	case "open":
		timeLabel = "Fecha em " + timeUntil(closesAt)
	default:
		timeLabel = "Votação encerrada"
	}

	renderJSON(w, http.StatusOK, map[string]any{
		"status":                  status,
		"opens_at":                opensAt,
		"closes_at":               closesAt,
		"voter_count":             voterCount,
		"eligible_count":          len(confirmedIDs),
		"current_player_voted":    hasVoted,
		"time_label":              timeLabel,
		"voted_player_ids":        votedIDs,
		"vote_open_delay_minutes": match.VoteOpenDelayMinutes,
	})
}

func (h *voteHandler) SubmitVote(w http.ResponseWriter, r *http.Request) {
	matchIDStr := chi.URLParam(r, "matchID")
	matchID, err := uuid.Parse(matchIDStr)
	if err != nil {
		renderError(w, apierror.NotFound("match not found"))
		return
	}

	player := middleware.PlayerFromCtx(r.Context())
	if player == nil {
		renderError(w, apierror.Unauthorized())
		return
	}

	var body struct {
		Top5 []struct {
			PlayerID uuid.UUID `json:"player_id"`
			Position int       `json:"position"`
		} `json:"top5"`
		FlopPlayerID *uuid.UUID `json:"flop_player_id"`
	}
	if err := decodeJSON(r, &body); err != nil {
		renderError(w, err)
		return
	}

	match, err := db.GetMatchByID(r.Context(), h.pool, matchID)
	if err != nil {
		renderError(w, apierror.NotFound("match not found"))
		return
	}

	if votingStatus(match) != "open" {
		renderError(w, apierror.Forbidden("VOTING_CLOSED"))
		return
	}
	if player.Role == db.PlayerRoleAdmin {
		renderError(w, apierror.Forbidden("NOT_ELIGIBLE"))
		return
	}

	attendances, err := db.GetAttendancesForMatch(r.Context(), h.pool, matchID)
	if err != nil {
		renderError(w, apierror.Internal("failed to check attendance"))
		return
	}
	confirmedIDs := confirmedPlayerIDs(attendances)
	isConfirmed := false
	for _, id := range confirmedIDs {
		if id == player.ID {
			isConfirmed = true
			break
		}
	}
	if !isConfirmed {
		renderError(w, apierror.Forbidden("NOT_ELIGIBLE"))
		return
	}

	hasVoted, err := db.HasVoted(r.Context(), h.pool, matchID, player.ID)
	if err != nil {
		renderError(w, apierror.Internal("failed to check vote"))
		return
	}
	if hasVoted {
		renderError(w, apierror.Conflict("ALREADY_VOTED"))
		return
	}

	for _, item := range body.Top5 {
		if item.PlayerID == player.ID {
			renderError(w, apierror.Unprocessable("SELF_VOTE"))
			return
		}
		if item.Position < 1 || item.Position > 5 {
			renderError(w, apierror.Unprocessable("position must be between 1 and 5"))
			return
		}
	}
	if body.FlopPlayerID != nil && *body.FlopPlayerID == player.ID {
		renderError(w, apierror.Unprocessable("SELF_VOTE"))
		return
	}

	top5 := make([]db.VoteTop5Item, len(body.Top5))
	for i, item := range body.Top5 {
		top5[i] = db.VoteTop5Item{PlayerID: item.PlayerID, Position: item.Position}
	}

	if err := db.SubmitVote(r.Context(), h.pool, matchID, player.ID, top5, body.FlopPlayerID); err != nil {
		renderError(w, apierror.Internal("failed to submit vote"))
		return
	}

	renderJSON(w, http.StatusCreated, map[string]string{"message": "Voto registrado com sucesso."})
}

func (h *voteHandler) GetPendingVotes(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())
	if player == nil {
		renderError(w, apierror.Unauthorized())
		return
	}

	if player.Role == db.PlayerRoleAdmin {
		renderJSON(w, http.StatusOK, map[string]any{"items": []any{}})
		return
	}

	items, err := db.GetPendingVotes(r.Context(), h.pool, player.ID)
	if err != nil {
		renderError(w, apierror.Internal("failed to get pending votes"))
		return
	}

	type pendingResp struct {
		MatchID       uuid.UUID `json:"match_id"`
		MatchHash     string    `json:"match_hash"`
		MatchNumber   int       `json:"match_number"`
		GroupName     string    `json:"group_name"`
		TimeLabel     string    `json:"time_label"`
		VoterCount    int       `json:"voter_count"`
		EligibleCount int       `json:"eligible_count"`
	}
	resp := make([]pendingResp, len(items))
	for i, item := range items {
		resp[i] = pendingResp{
			MatchID:       item.MatchID,
			MatchHash:     item.MatchHash,
			MatchNumber:   item.MatchNumber,
			GroupName:     item.GroupName,
			TimeLabel:     "Fecha em " + timeUntil(item.ClosesAt),
			VoterCount:    item.VoterCount,
			EligibleCount: item.EligibleCount,
		}
	}
	renderJSON(w, http.StatusOK, map[string]any{"items": resp})
}

func (h *voteHandler) GetPublicVoteResults(w http.ResponseWriter, r *http.Request) {
	hash := chi.URLParam(r, "hash")
	match, err := db.GetMatchByHash(r.Context(), h.pool, hash)
	if err != nil {
		renderError(w, apierror.NotFound("match not found"))
		return
	}
	if votingStatus(match) != "closed" {
		renderError(w, apierror.NotFound("results not available"))
		return
	}

	attendances, _ := db.GetAttendancesForMatch(r.Context(), h.pool, match.ID)
	confirmedIDs := confirmedPlayerIDs(attendances)

	results, err := db.GetVoteResults(r.Context(), h.pool, match.ID)
	if err != nil {
		renderError(w, apierror.Internal("failed to get results"))
		return
	}
	renderJSON(w, http.StatusOK, map[string]any{
		"top5":            results.Top5,
		"flop":            results.Flop,
		"total_voters":    results.TotalVoters,
		"eligible_voters": len(confirmedIDs),
	})
}

func (h *voteHandler) GetPublicVoteBallots(w http.ResponseWriter, r *http.Request) {
	hash := chi.URLParam(r, "hash")
	match, err := db.GetMatchByHash(r.Context(), h.pool, hash)
	if err != nil {
		renderError(w, apierror.NotFound("match not found"))
		return
	}
	if votingStatus(match) != "closed" {
		renderError(w, apierror.NotFound("ballots not available"))
		return
	}

	ballots, err := db.GetVoteBallots(r.Context(), h.pool, match.ID)
	if err != nil {
		renderError(w, apierror.Internal("failed to get ballots"))
		return
	}
	totalVoters, _ := db.VoterCount(r.Context(), h.pool, match.ID)
	renderJSON(w, http.StatusOK, map[string]any{
		"ballots":      ballots,
		"total_voters": totalVoters,
	})
}

func (h *voteHandler) GetVoteResults(w http.ResponseWriter, r *http.Request) {
	matchIDStr := chi.URLParam(r, "matchID")
	matchID, err := uuid.Parse(matchIDStr)
	if err != nil {
		renderError(w, apierror.NotFound("match not found"))
		return
	}

	player := middleware.PlayerFromCtx(r.Context())
	if player == nil {
		renderError(w, apierror.Unauthorized())
		return
	}

	match, err := db.GetMatchByID(r.Context(), h.pool, matchID)
	if err != nil {
		renderError(w, apierror.NotFound("match not found"))
		return
	}
	if votingStatus(match) != "closed" {
		renderError(w, apierror.Forbidden("RESULTS_NOT_AVAILABLE"))
		return
	}

	attendances, _ := db.GetAttendancesForMatch(r.Context(), h.pool, match.ID)
	confirmedIDs := confirmedPlayerIDs(attendances)

	results, err := db.GetVoteResults(r.Context(), h.pool, matchID)
	if err != nil {
		renderError(w, apierror.Internal("failed to get results"))
		return
	}
	renderJSON(w, http.StatusOK, map[string]any{
		"top5":            results.Top5,
		"flop":            results.Flop,
		"total_voters":    results.TotalVoters,
		"eligible_voters": len(confirmedIDs),
	})
}

func (h *voteHandler) CloseVoting(w http.ResponseWriter, r *http.Request) {
	matchIDStr := chi.URLParam(r, "matchID")
	matchID, err := uuid.Parse(matchIDStr)
	if err != nil {
		renderError(w, apierror.NotFound("match not found"))
		return
	}

	player := middleware.PlayerFromCtx(r.Context())
	if player == nil {
		renderError(w, apierror.Unauthorized())
		return
	}

	match, err := db.GetMatchByID(r.Context(), h.pool, matchID)
	if err != nil {
		renderError(w, apierror.NotFound("match not found"))
		return
	}

	if player.Role != db.PlayerRoleAdmin {
		member, err := db.GetGroupMember(r.Context(), h.pool, match.GroupID, player.ID)
		if err != nil || member == nil || member.Role != db.GroupMemberRoleAdmin {
			renderError(w, apierror.Forbidden("NOT_GROUP_ADMIN"))
			return
		}
	}

	if votingStatus(match) != "open" {
		renderError(w, apierror.Forbidden("VOTING_NOT_OPEN"))
		return
	}

	if err := db.CloseVotingEarly(r.Context(), h.pool, matchID); err != nil {
		renderError(w, apierror.Internal("failed to close voting"))
		return
	}

	renderJSON(w, http.StatusOK, map[string]string{"message": "Votação encerrada com sucesso."})
}
