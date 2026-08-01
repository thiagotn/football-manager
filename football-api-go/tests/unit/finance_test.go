package unit_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/thiagotn/football-manager/football-api-go/internal/db"
	"github.com/thiagotn/football-manager/football-api-go/internal/middleware"
)

func memberOfGroup() func(ctx context.Context, groupID, playerID uuid.UUID) (*db.GroupMember, error) {
	return func(ctx context.Context, groupID, playerID uuid.UUID) (*db.GroupMember, error) {
		return &db.GroupMember{GroupID: groupID, PlayerID: playerID, Role: db.GroupMemberRoleMember}, nil
	}
}

// GET period without payments must return "payments": [] — never null.
// Regression test: nil slice serialized as null broke the finance tab in the frontend.
func TestFinance_GetPeriod_EmptyPayments_ReturnsEmptyArrayNotNull(t *testing.T) {
	groupID := uuid.New()
	periodID := uuid.New()
	store := &mockFinanceStore{
		getGroupMemberFn: memberOfGroup(),
		getFinancePeriodFn: func(ctx context.Context, gID uuid.UUID, year, month int) (*db.FinancePeriod, error) {
			return &db.FinancePeriod{ID: periodID, GroupID: gID, Year: year, Month: month}, nil
		},
		getPaymentsForPeriodFn: func(ctx context.Context, pID uuid.UUID) ([]db.FinancePayment, error) {
			return nil, nil
		},
	}
	r := financeRouter(store)
	w := sendRequestWithContext(r, "GET",
		fmt.Sprintf("/groups/%s/finance/2023/5", groupID),
		"",
		middleware.InjectPlayerForTest(context.Background(), fakePlayer()),
	)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, strings.Contains(w.Body.String(), `"payments":[]`),
		"body must contain payments as empty array, got: %s", w.Body.String())

	var resp struct {
		PeriodID string            `json:"period_id"`
		Year     int               `json:"year"`
		Month    int               `json:"month"`
		Payments []json.RawMessage `json:"payments"`
		Summary  struct {
			ReceivedCents int `json:"received_cents"`
			PendingCount  int `json:"pending_count"`
			PaidCount     int `json:"paid_count"`
		} `json:"summary"`
	}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, periodID.String(), resp.PeriodID)
	assert.Equal(t, 2023, resp.Year)
	assert.Equal(t, 5, resp.Month)
	assert.NotNil(t, resp.Payments)
	assert.Len(t, resp.Payments, 0)
	assert.Equal(t, 0, resp.Summary.ReceivedCents)
}

func TestFinance_GetPeriod_WithPayments_ReturnsSortedList(t *testing.T) {
	groupID := uuid.New()
	periodID := uuid.New()
	amount := 2500
	store := &mockFinanceStore{
		getGroupMemberFn: memberOfGroup(),
		getFinancePeriodFn: func(ctx context.Context, gID uuid.UUID, year, month int) (*db.FinancePeriod, error) {
			return &db.FinancePeriod{ID: periodID, GroupID: gID, Year: year, Month: month}, nil
		},
		getPaymentsForPeriodFn: func(ctx context.Context, pID uuid.UUID) ([]db.FinancePayment, error) {
			return []db.FinancePayment{
				{ID: uuid.New(), PeriodID: pID, PlayerID: uuid.New(), PlayerName: "Zico", Status: "paid", AmountDue: &amount},
				{ID: uuid.New(), PeriodID: pID, PlayerID: uuid.New(), PlayerName: "Adriano", Status: "pending"},
			}, nil
		},
	}
	r := financeRouter(store)
	w := sendRequestWithContext(r, "GET",
		fmt.Sprintf("/groups/%s/finance/2023/5", groupID),
		"",
		middleware.InjectPlayerForTest(context.Background(), fakePlayer()),
	)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Payments []struct {
			PlayerName string `json:"player_name"`
			Status     string `json:"status"`
		} `json:"payments"`
		Summary struct {
			ReceivedCents int `json:"received_cents"`
			PaidCount     int `json:"paid_count"`
			PendingCount  int `json:"pending_count"`
		} `json:"summary"`
	}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Len(t, resp.Payments, 2)
	assert.Equal(t, "Adriano", resp.Payments[0].PlayerName, "pending payments come first")
	assert.Equal(t, 2500, resp.Summary.ReceivedCents)
	assert.Equal(t, 1, resp.Summary.PaidCount)
	assert.Equal(t, 1, resp.Summary.PendingCount)
}

func TestFinance_GetPeriod_NonMember_Returns403(t *testing.T) {
	groupID := uuid.New()
	r := financeRouter(&mockFinanceStore{})
	w := sendRequestWithContext(r, "GET",
		fmt.Sprintf("/groups/%s/finance/2023/5", groupID),
		"",
		middleware.InjectPlayerForTest(context.Background(), fakePlayer()),
	)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestFinance_GetPeriod_InvalidMonth_Returns422(t *testing.T) {
	groupID := uuid.New()
	r := financeRouter(&mockFinanceStore{})
	w := sendRequestWithContext(r, "GET",
		fmt.Sprintf("/groups/%s/finance/2023/13", groupID),
		"",
		middleware.InjectPlayerForTest(context.Background(), fakePlayer()),
	)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}
