package unit_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/thiagotn/football-manager/football-api-go/internal/db"
	"github.com/thiagotn/football-manager/football-api-go/internal/handlers"
	"github.com/thiagotn/football-manager/football-api-go/internal/middleware"
)

func newGroupStatsRouter(store handlers.GroupStore) http.Handler {
	r := chi.NewRouter()
	h := handlers.NewGroupHandlerWithDeps(store, nil, &recordingPush{})
	r.Mount("/groups", h.Routes())
	return r
}

func TestGroupStats_AnnualHappyPath(t *testing.T) {
	groupID := uuid.New()
	playerID := uuid.New()
	store := &mockGroupStoreForBusiness{
		getGroupStatsFn: func(ctx context.Context, gID uuid.UUID, monthStart, monthEnd *time.Time, year int) ([]db.GroupPlayerStat, error) {
			assert.Equal(t, groupID, gID)
			assert.Nil(t, monthStart)
			assert.Nil(t, monthEnd)
			assert.Equal(t, time.Now().UTC().Year(), year)
			return []db.GroupPlayerStat{
				{PlayerID: playerID, DisplayName: "Zico", VotePoints: 12, FlopVotes: 1, MinutesPlayed: 180},
			}, nil
		},
	}
	r := newGroupStatsRouter(store)
	w := sendRequestWithContext(r, "GET",
		fmt.Sprintf("/groups/%s/stats", groupID), "",
		middleware.InjectPlayerForTest(context.Background(), fakePlayer(asAdmin())),
	)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Players     []db.GroupPlayerStat `json:"players"`
		PeriodLabel string               `json:"period_label"`
	}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Len(t, resp.Players, 1)
	assert.Equal(t, "Zico", resp.Players[0].DisplayName)
	assert.Equal(t, 12, resp.Players[0].VotePoints)
	assert.Equal(t, fmt.Sprintf("Ano %d", time.Now().UTC().Year()), resp.PeriodLabel)
}

func TestGroupStats_MonthlyPeriodLabelAndRange(t *testing.T) {
	groupID := uuid.New()
	store := &mockGroupStoreForBusiness{
		getGroupStatsFn: func(ctx context.Context, gID uuid.UUID, monthStart, monthEnd *time.Time, year int) ([]db.GroupPlayerStat, error) {
			if assert.NotNil(t, monthStart) && assert.NotNil(t, monthEnd) {
				assert.Equal(t, "2026-07-01", monthStart.Format("2006-01-02"))
				assert.Equal(t, "2026-07-31", monthEnd.Format("2006-01-02"))
			}
			return []db.GroupPlayerStat{}, nil
		},
	}
	r := newGroupStatsRouter(store)
	w := sendRequestWithContext(r, "GET",
		fmt.Sprintf("/groups/%s/stats?period=monthly&month=2026-07", groupID), "",
		middleware.InjectPlayerForTest(context.Background(), fakePlayer(asAdmin())),
	)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "Julho 2026", resp["period_label"])
	assert.Equal(t, []any{}, resp["players"])
}

func TestGroupStats_InvalidMonthFallsBackToAnnual(t *testing.T) {
	groupID := uuid.New()
	store := &mockGroupStoreForBusiness{
		getGroupStatsFn: func(ctx context.Context, gID uuid.UUID, monthStart, monthEnd *time.Time, year int) ([]db.GroupPlayerStat, error) {
			assert.Nil(t, monthStart)
			assert.Nil(t, monthEnd)
			return []db.GroupPlayerStat{}, nil
		},
	}
	r := newGroupStatsRouter(store)
	w := sendRequestWithContext(r, "GET",
		fmt.Sprintf("/groups/%s/stats?period=monthly&month=banana", groupID), "",
		middleware.InjectPlayerForTest(context.Background(), fakePlayer(asAdmin())),
	)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGroupStats_NonMemberForbidden(t *testing.T) {
	groupID := uuid.New()
	store := &mockGroupStoreForBusiness{} // GetGroupMember → ErrNotFound
	r := newGroupStatsRouter(store)
	w := sendRequestWithContext(r, "GET",
		fmt.Sprintf("/groups/%s/stats", groupID), "",
		middleware.InjectPlayerForTest(context.Background(), fakePlayer()),
	)
	assert.Equal(t, http.StatusForbidden, w.Code)
}
