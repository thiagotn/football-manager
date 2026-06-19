package handlers

import (
	"github.com/thiagotn/football-manager/football-api-go/internal/db"
	"github.com/thiagotn/football-manager/football-api-go/internal/services"
)

// matchResponse mirrors v1's MatchResponse (with the listing helpers
// is_current and voting_status). Used by list/create/update match endpoints.
type matchResponse struct {
	*db.Match
	IsCurrent    bool                  `json:"is_current"`
	VotingStatus services.VotingStatus `json:"voting_status"`
}

// matchToListing converts a db.Match into the input shape ClassifyMatches expects.
func matchToListing(m db.Match) services.ListingMatch {
	return services.ListingMatch{
		ID:                   m.ID,
		Status:               m.Status,
		MatchDate:            m.MatchDate,
		StartTime:            m.StartTime,
		EndTime:              m.EndTime,
		VoteOpenDelayMinutes: m.VoteOpenDelayMinutes,
		VoteDurationHours:    m.VoteDurationHours,
	}
}

// enrichGroupMatches classifies a full group's match list and returns the
// wrapped responses ready for renderJSON.
func enrichGroupMatches(matches []db.Match) []matchResponse {
	listing := make([]services.ListingMatch, len(matches))
	for i, m := range matches {
		listing[i] = matchToListing(m)
	}
	classification := services.ClassifyMatches(listing)

	out := make([]matchResponse, len(matches))
	for i := range matches {
		res := classification[matches[i].ID]
		out[i] = matchResponse{
			Match:        &matches[i],
			IsCurrent:    res.IsCurrent,
			VotingStatus: res.VotingStatus,
		}
	}
	return out
}

// enrichOneMatch returns a matchResponse for a single match — used by
// create/update where we have no group context. Single-match context naturally
// resolves the "is the most recent without future" branch.
func enrichOneMatch(m *db.Match) matchResponse {
	classification := services.ClassifyMatches([]services.ListingMatch{matchToListing(*m)})
	res := classification[m.ID]
	return matchResponse{
		Match:        m,
		IsCurrent:    res.IsCurrent,
		VotingStatus: res.VotingStatus,
	}
}

