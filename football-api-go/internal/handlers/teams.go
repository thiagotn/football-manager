package handlers

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thiagotn/football-manager/football-api-go/internal/apierror"
	"github.com/thiagotn/football-manager/football-api-go/internal/db"
	"github.com/thiagotn/football-manager/football-api-go/internal/middleware"
	"github.com/thiagotn/football-manager/football-api-go/internal/services"
)

type teamHandler struct {
	pool *pgxpool.Pool
}

func NewTeamHandler(pool *pgxpool.Pool) *teamHandler {
	return &teamHandler{pool: pool}
}

// Routes are registered directly in the router as exported methods DrawTeams and GetTeams.

// ── Response types ────────────────────────────────────────────────────────────

type teamPlayerResp struct {
	PlayerID   interface{} `json:"player_id"`
	Name       string      `json:"name"`
	Nickname   *string     `json:"nickname"`
	SkillStars int         `json:"skill_stars"`
	Position   string      `json:"position"`
}

type teamResp struct {
	ID         interface{}      `json:"id"`
	Name       string           `json:"name"`
	Color      *string          `json:"color"`
	Position   int              `json:"position"`
	SkillTotal int              `json:"skill_total"`
	Players    []teamPlayerResp `json:"players"`
}

type teamsResp struct {
	Teams    []teamResp       `json:"teams"`
	Reserves []teamPlayerResp `json:"reserves"`
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func toTeamPlayerResp(p db.PlayerForDraw) teamPlayerResp {
	return teamPlayerResp{
		PlayerID:   p.PlayerID,
		Name:       p.Name,
		Nickname:   p.Nickname,
		SkillStars: p.SkillStars,
		Position:   p.Position,
	}
}

// ── Handlers ──────────────────────────────────────────────────────────────────

func (h *teamHandler) DrawTeams(w http.ResponseWriter, r *http.Request) {
	player := middleware.PlayerFromCtx(r.Context())
	matchID, err := matchIDParam(r)
	if err != nil {
		renderError(w, apierror.NotFound("match not found"))
		return
	}

	match, err := db.GetMatchByID(r.Context(), h.pool, matchID)
	if err != nil {
		renderError(w, err)
		return
	}

	if player.Role != db.PlayerRoleAdmin {
		m, err := db.GetGroupMember(r.Context(), h.pool, match.GroupID, player.ID)
		if err != nil || m.Role != db.GroupMemberRoleAdmin {
			renderError(w, apierror.Forbidden("only group admins can draw teams"))
			return
		}
	}

	if match.PlayersPerTeam == nil {
		renderError(w, apierror.Unprocessable("players_per_team must be set on the match before drawing teams"))
		return
	}

	confirmed, err := db.GetConfirmedPlayersForMatch(r.Context(), h.pool, matchID)
	if err != nil {
		renderError(w, err)
		return
	}

	// Get group's team_slots
	group, _ := db.GetGroupByID(r.Context(), h.pool, match.GroupID)
	var slots []db.TeamSlot
	if group != nil {
		slots = group.TeamSlots
	}

	teamResults, reserves, err := services.BuildTeams(confirmed, *match.PlayersPerTeam, slots)
	if err != nil {
		renderError(w, apierror.Unprocessable(err.Error()))
		return
	}

	// Persist teams
	if err := db.DeleteTeamsByMatch(r.Context(), h.pool, matchID); err != nil {
		renderError(w, err)
		return
	}

	resp := teamsResp{
		Teams:    make([]teamResp, 0, len(teamResults)),
		Reserves: make([]teamPlayerResp, 0, len(reserves)),
	}

	for _, tr := range teamResults {
		color := tr.Color
		team, err := db.CreateTeam(r.Context(), h.pool, matchID, tr.Name, &color, tr.Position)
		if err != nil {
			renderError(w, err)
			return
		}

		tResp := teamResp{
			ID:         team.ID,
			Name:       team.Name,
			Color:      team.Color,
			Position:   team.Position,
			SkillTotal: tr.SkillTotal,
			Players:    make([]teamPlayerResp, 0, len(tr.Players)),
		}

		for _, p := range tr.Players {
			_ = db.AddPlayerToTeam(r.Context(), h.pool, team.ID, p.PlayerID, false)
			tResp.Players = append(tResp.Players, toTeamPlayerResp(p))
		}
		resp.Teams = append(resp.Teams, tResp)
	}

	// Persist reserves as a virtual team with position=0
	if len(reserves) > 0 {
		reserveTeam, _ := db.CreateTeam(r.Context(), h.pool, matchID, "Reservas", nil, 0)
		if reserveTeam != nil {
			for _, p := range reserves {
				_ = db.AddPlayerToTeam(r.Context(), h.pool, reserveTeam.ID, p.PlayerID, true)
				resp.Reserves = append(resp.Reserves, toTeamPlayerResp(p))
			}
		}
	}

	renderJSON(w, http.StatusCreated, resp)
}

func (h *teamHandler) GetTeams(w http.ResponseWriter, r *http.Request) {
	matchID, err := matchIDParam(r)
	if err != nil {
		renderError(w, apierror.NotFound("match not found"))
		return
	}

	teams, err := db.GetTeamsForMatch(r.Context(), h.pool, matchID)
	if err != nil {
		renderError(w, err)
		return
	}

	resp := teamsResp{
		Teams:    make([]teamResp, 0),
		Reserves: make([]teamPlayerResp, 0),
	}

	for _, t := range teams {
		if t.Position == 0 {
			// Reserves
			for _, p := range t.Players {
				resp.Reserves = append(resp.Reserves, teamPlayerResp{
					PlayerID:   p.PlayerID,
					Name:       p.Name,
					Nickname:   p.Nickname,
					SkillStars: p.SkillStars,
					Position:   p.Position,
				})
			}
			continue
		}

		skillTotal := 0
		players := make([]teamPlayerResp, 0, len(t.Players))
		for _, p := range t.Players {
			skillTotal += p.SkillStars
			players = append(players, teamPlayerResp{
				PlayerID:   p.PlayerID,
				Name:       p.Name,
				Nickname:   p.Nickname,
				SkillStars: p.SkillStars,
				Position:   p.Position,
			})
		}

		resp.Teams = append(resp.Teams, teamResp{
			ID:         t.ID,
			Name:       t.Name,
			Color:      t.Color,
			Position:   t.Position,
			SkillTotal: skillTotal,
			Players:    players,
		})
	}

	renderJSON(w, http.StatusOK, resp)
}
