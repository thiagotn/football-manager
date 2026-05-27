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
	"github.com/thiagotn/football-manager/football-api-go/internal/services"
	"context"
)

var brtLoc *time.Location

func init() {
	var err error
	brtLoc, err = time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		brtLoc = time.UTC
	}
}

type VoteStore interface {
	GetMatchByID(ctx context.Context, matchID uuid.UUID) (*db.Match, error)
	GetMatchByHash(ctx context.Context, hash string) (*db.Match, error)
	GetAttendancesForMatch(ctx context.Context, matchID uuid.UUID) ([]db.AttendanceWithPlayer, error)
	HasVoted(ctx context.Context, matchID, playerID uuid.UUID) (bool, error)
	VoterCount(ctx context.Context, matchID uuid.UUID) (int, error)
	VoterIDs(ctx context.Context, matchID uuid.UUID) ([]uuid.UUID, error)
	MarkVoteNotified(ctx context.Context, matchID uuid.UUID) error
	SubmitVote(ctx context.Context, matchID, playerID uuid.UUID, top5 []db.VoteTop5Item, flopPlayerID *uuid.UUID) error
	GetPendingVotes(ctx context.Context, playerID uuid.UUID) ([]db.PendingVoteItem, error)
	GetVoteResults(ctx context.Context, matchID uuid.UUID) (*db.VoteResults, error)
	GetVoteBallots(ctx context.Context, matchID uuid.UUID) ([]db.Ballot, error)
	CloseVotingEarly(ctx context.Context, matchID uuid.UUID) error
	GetGroupMember(ctx context.Context, groupID, playerID uuid.UUID) (*db.GroupMember, error)
}

type pgVoteStore struct {
	pool *pgxpool.Pool
}

func (s *pgVoteStore) GetMatchByID(ctx context.Context, matchID uuid.UUID) (*db.Match, error) {
	return db.GetMatchByID(ctx, s.pool, matchID)
}
func (s *pgVoteStore) GetMatchByHash(ctx context.Context, hash string) (*db.Match, error) {
	return db.GetMatchByHash(ctx, s.pool, hash)
}
func (s *pgVoteStore) GetAttendancesForMatch(ctx context.Context, matchID uuid.UUID) ([]db.AttendanceWithPlayer, error) {
	return db.GetAttendancesForMatch(ctx, s.pool, matchID)
}
func (s *pgVoteStore) HasVoted(ctx context.Context, matchID, playerID uuid.UUID) (bool, error) {
	return db.HasVoted(ctx, s.pool, matchID, playerID)
}
func (s *pgVoteStore) VoterCount(ctx context.Context, matchID uuid.UUID) (int, error) {
	return db.VoterCount(ctx, s.pool, matchID)
}
func (s *pgVoteStore) VoterIDs(ctx context.Context, matchID uuid.UUID) ([]uuid.UUID, error) {
	return db.VoterIDs(ctx, s.pool, matchID)
}
func (s *pgVoteStore) MarkVoteNotified(ctx context.Context, matchID uuid.UUID) error {
	return db.MarkVoteNotified(ctx, s.pool, matchID)
}
func (s *pgVoteStore) SubmitVote(ctx context.Context, matchID, playerID uuid.UUID, top5 []db.VoteTop5Item, flopPlayerID *uuid.UUID) error {
	return db.SubmitVote(ctx, s.pool, matchID, playerID, top5, flopPlayerID)
}
func (s *pgVoteStore) GetPendingVotes(ctx context.Context, playerID uuid.UUID) ([]db.PendingVoteItem, error) {
	return db.GetPendingVotes(ctx, s.pool, playerID)
}
func (s *pgVoteStore) GetVoteResults(ctx context.Context, matchID uuid.UUID) (*db.VoteResults, error) {
	return db.GetVoteResults(ctx, s.pool, matchID)
}
func (s *pgVoteStore) GetVoteBallots(ctx context.Context, matchID uuid.UUID) ([]db.Ballot, error) {
	return db.GetVoteBallots(ctx, s.pool, matchID)
}
func (s *pgVoteStore) CloseVotingEarly(ctx context.Context, matchID uuid.UUID) error {
	return db.CloseVotingEarly(ctx, s.pool, matchID)
}
func (s *pgVoteStore) GetGroupMember(ctx context.Context, groupID, playerID uuid.UUID) (*db.GroupMember, error) {
	return db.GetGroupMember(ctx, s.pool, groupID, playerID)
}

type VoteHandler struct {
	Store       VoteStore
	PushService services.PushService
}

func NewVoteHandler(pool *pgxpool.Pool, pushService services.PushService) *VoteHandler {
	return &VoteHandler{Store: &pgVoteStore{pool: pool}, PushService: pushService}
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

func (h *VoteHandler) GetVoteStatus(w http.ResponseWriter, r *http.Request) {
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

	match, err := h.Store.GetMatchByID(r.Context(), matchID)
	if err != nil {
		renderError(w, apierror.NotFound("match not found"))
		return
	}

	attendances, err := h.Store.GetAttendancesForMatch(r.Context(), matchID)
	if err != nil {
		renderError(w, apierror.Internal("failed to get attendances"))
		return
	}

	status := votingStatus(match)
	opensAt, closesAt := votingWindow(match)
	confirmedIDs := confirmedPlayerIDs(attendances)

	voterCount, err := h.Store.VoterCount(r.Context(), matchID)
	if err != nil {
		renderError(w, apierror.Internal("failed to get voter count"))
		return
	}

	hasVoted, err := h.Store.HasVoted(r.Context(), matchID, player.ID)
	if err != nil {
		renderError(w, apierror.Internal("failed to check vote"))
		return
	}

	var votedIDs []uuid.UUID
	if status == "open" || status == "closed" {
		if votedIDs, err = h.Store.VoterIDs(r.Context(), matchID); err != nil {
			renderError(w, apierror.Internal("failed to get voter ids"))
			return
		}
	}
	if votedIDs == nil {
		votedIDs = []uuid.UUID{}
	}

	if status == "open" && !match.VoteNotified {
		_ = h.Store.MarkVoteNotified(r.Context(), matchID)
		if h.PushService != nil {
			_ = h.PushService.SendToPlayers(r.Context(), confirmedIDs, services.PushNotification{
				Title: "🏆 Votação aberta!",
				Body:  "Escolha os melhores da pelada de hoje.",
				URL:   "https://rachao.app/match/" + match.Hash,
			})
		}
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

func (h *VoteHandler) SubmitVote(w http.ResponseWriter, r *http.Request) {
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

	match, err := h.Store.GetMatchByID(r.Context(), matchID)
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

	attendances, err := h.Store.GetAttendancesForMatch(r.Context(), matchID)
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

	hasVoted, err := h.Store.HasVoted(r.Context(), matchID, player.ID)
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

	if err := h.Store.SubmitVote(r.Context(), matchID, player.ID, top5, body.FlopPlayerID); err != nil {
		renderError(w, apierror.Internal("failed to submit vote"))
		return
	}

	renderJSON(w, http.StatusCreated, map[string]string{"message": "Voto registrado com sucesso."})
}

func (h *VoteHandler) GetPendingVotes(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())
	if player == nil {
		renderError(w, apierror.Unauthorized())
		return
	}

	if player.Role == db.PlayerRoleAdmin {
		renderJSON(w, http.StatusOK, map[string]any{"items": []any{}})
		return
	}

	items, err := h.Store.GetPendingVotes(r.Context(), player.ID)
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

func (h *VoteHandler) GetPublicVoteResults(w http.ResponseWriter, r *http.Request) {
	hash := chi.URLParam(r, "hash")
	match, err := h.Store.GetMatchByHash(r.Context(), hash)
	if err != nil {
		renderError(w, apierror.NotFound("match not found"))
		return
	}
	if votingStatus(match) != "closed" {
		renderError(w, apierror.NotFound("results not available"))
		return
	}

	attendances, _ := h.Store.GetAttendancesForMatch(r.Context(), match.ID)
	confirmedIDs := confirmedPlayerIDs(attendances)

	results, err := h.Store.GetVoteResults(r.Context(), match.ID)
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

func (h *VoteHandler) GetPublicVoteBallots(w http.ResponseWriter, r *http.Request) {
	hash := chi.URLParam(r, "hash")
	match, err := h.Store.GetMatchByHash(r.Context(), hash)
	if err != nil {
		renderError(w, apierror.NotFound("match not found"))
		return
	}
	if votingStatus(match) != "closed" {
		renderError(w, apierror.NotFound("ballots not available"))
		return
	}

	ballots, err := h.Store.GetVoteBallots(r.Context(), match.ID)
	if err != nil {
		renderError(w, apierror.Internal("failed to get ballots"))
		return
	}
	totalVoters, _ := h.Store.VoterCount(r.Context(), match.ID)
	renderJSON(w, http.StatusOK, map[string]any{
		"ballots":      ballots,
		"total_voters": totalVoters,
	})
}

func (h *VoteHandler) GetVoteResults(w http.ResponseWriter, r *http.Request) {
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

	match, err := h.Store.GetMatchByID(r.Context(), matchID)
	if err != nil {
		renderError(w, apierror.NotFound("match not found"))
		return
	}
	if votingStatus(match) != "closed" {
		renderError(w, apierror.Forbidden("RESULTS_NOT_AVAILABLE"))
		return
	}

	attendances, _ := h.Store.GetAttendancesForMatch(r.Context(), match.ID)
	confirmedIDs := confirmedPlayerIDs(attendances)

	results, err := h.Store.GetVoteResults(r.Context(), matchID)
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

func (h *VoteHandler) CloseVoting(w http.ResponseWriter, r *http.Request) {
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

	match, err := h.Store.GetMatchByID(r.Context(), matchID)
	if err != nil {
		renderError(w, apierror.NotFound("match not found"))
		return
	}

	if player.Role != db.PlayerRoleAdmin {
		member, err := h.Store.GetGroupMember(r.Context(), match.GroupID, player.ID)
		if err != nil || member == nil || member.Role != db.GroupMemberRoleAdmin {
			renderError(w, apierror.Forbidden("NOT_GROUP_ADMIN"))
			return
		}
	}

	if votingStatus(match) != "open" {
		renderError(w, apierror.Forbidden("VOTING_NOT_OPEN"))
		return
	}

	if err := h.Store.CloseVotingEarly(r.Context(), matchID); err != nil {
		renderError(w, apierror.Internal("failed to close voting"))
		return
	}

	renderJSON(w, http.StatusOK, map[string]string{"message": "Votação encerrada com sucesso."})
}
