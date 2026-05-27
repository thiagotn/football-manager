package unit_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"

	"github.com/thiagotn/football-manager/football-api-go/internal/handlers"
)

// ── Router helpers ────────────────────────────────────────────────────────────

func teamsRouter() http.Handler {
	r := chi.NewRouter()
	h := handlers.NewTeamHandler(nil) // nil pool — only pre-DB validation tested
	r.Get("/matches/{matchID}/teams", h.GetTeams)
	return r
}

// ── Unit tests ────────────────────────────────────────────────────────────────

// TestGetTeams_InvalidMatchID verifies that an invalid UUID in the matchID parameter
// returns a 404 error with "match not found" message.
func TestGetTeams_InvalidMatchID(t *testing.T) {
	router := teamsRouter()

	testCases := []string{
		"not-a-uuid",
		"12345",
		"",
		"00000000-0000-invalid",
	}

	for _, invalid := range testCases {
		path := "/matches/" + invalid + "/teams"
		rec := doRequest(router, http.MethodGet, path, "")

		assert.Equal(t, http.StatusNotFound, rec.Code, "path: %s", path)

		var respBody map[string]string
		err := json.Unmarshal(rec.Body.Bytes(), &respBody)
		assert.NoError(t, err)
		assert.Equal(t, "match not found", respBody["detail"])
	}
}

