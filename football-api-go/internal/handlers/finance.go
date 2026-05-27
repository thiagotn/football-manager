package handlers

import (
	"context"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thiagotn/football-manager/football-api-go/internal/apierror"
	"github.com/thiagotn/football-manager/football-api-go/internal/db"
	"github.com/thiagotn/football-manager/football-api-go/internal/middleware"
)

type FinanceStore interface {
	ListFinancePeriods(ctx context.Context, groupID uuid.UUID) ([]db.FinancePeriod, error)
	GetFinancePeriod(ctx context.Context, groupID uuid.UUID, year, month int) (*db.FinancePeriod, error)
	GetOrCreateFinancePeriod(ctx context.Context, groupID uuid.UUID, year, month int) (*db.FinancePeriod, error)
	GetPaymentsForPeriod(ctx context.Context, periodID uuid.UUID) ([]db.FinancePayment, error)
	GetFinancePayment(ctx context.Context, paymentID uuid.UUID) (*db.FinancePayment, error)
	GetPeriodGroupID(ctx context.Context, periodID uuid.UUID) (uuid.UUID, error)
	GetGroupByID(ctx context.Context, groupID uuid.UUID) (*db.Group, error)
	GetGroupMember(ctx context.Context, groupID, playerID uuid.UUID) (*db.GroupMember, error)
	MarkPaymentPaid(ctx context.Context, paymentID uuid.UUID, paymentType string, amountCents int) (*db.FinancePayment, error)
	MarkPaymentPending(ctx context.Context, paymentID uuid.UUID) (*db.FinancePayment, error)
}

type pgFinanceStore struct {
	pool *pgxpool.Pool
}

func (s *pgFinanceStore) ListFinancePeriods(ctx context.Context, groupID uuid.UUID) ([]db.FinancePeriod, error) {
	return db.ListFinancePeriods(ctx, s.pool, groupID)
}

func (s *pgFinanceStore) GetFinancePeriod(ctx context.Context, groupID uuid.UUID, year, month int) (*db.FinancePeriod, error) {
	return db.GetFinancePeriod(ctx, s.pool, groupID, year, month)
}

func (s *pgFinanceStore) GetOrCreateFinancePeriod(ctx context.Context, groupID uuid.UUID, year, month int) (*db.FinancePeriod, error) {
	return db.GetOrCreateFinancePeriod(ctx, s.pool, groupID, year, month)
}

func (s *pgFinanceStore) GetPaymentsForPeriod(ctx context.Context, periodID uuid.UUID) ([]db.FinancePayment, error) {
	return db.GetPaymentsForPeriod(ctx, s.pool, periodID)
}

func (s *pgFinanceStore) GetFinancePayment(ctx context.Context, paymentID uuid.UUID) (*db.FinancePayment, error) {
	return db.GetFinancePayment(ctx, s.pool, paymentID)
}

func (s *pgFinanceStore) GetPeriodGroupID(ctx context.Context, periodID uuid.UUID) (uuid.UUID, error) {
	return db.GetPeriodGroupID(ctx, s.pool, periodID)
}

func (s *pgFinanceStore) GetGroupByID(ctx context.Context, groupID uuid.UUID) (*db.Group, error) {
	return db.GetGroupByID(ctx, s.pool, groupID)
}

func (s *pgFinanceStore) GetGroupMember(ctx context.Context, groupID, playerID uuid.UUID) (*db.GroupMember, error) {
	return db.GetGroupMember(ctx, s.pool, groupID, playerID)
}

func (s *pgFinanceStore) MarkPaymentPaid(ctx context.Context, paymentID uuid.UUID, paymentType string, amountCents int) (*db.FinancePayment, error) {
	return db.MarkPaymentPaid(ctx, s.pool, paymentID, paymentType, amountCents)
}

func (s *pgFinanceStore) MarkPaymentPending(ctx context.Context, paymentID uuid.UUID) (*db.FinancePayment, error) {
	return db.MarkPaymentPending(ctx, s.pool, paymentID)
}

type FinanceHandler struct {
	Store FinanceStore
}

func NewFinanceHandler(pool *pgxpool.Pool) *FinanceHandler {
	return &FinanceHandler{Store: &pgFinanceStore{pool: pool}}
}

type financeSummary struct {
	ReceivedCents int `json:"received_cents"`
	PendingCount  int `json:"pending_count"`
	PaidCount     int `json:"paid_count"`
	TotalMembers  int `json:"total_members"`
	CompliancePct int `json:"compliance_pct"`
}

func buildFinanceSummary(payments []db.FinancePayment) financeSummary {
	var active, paid, pending []db.FinancePayment
	for _, p := range payments {
		if p.Status != "excluded" {
			active = append(active, p)
			switch p.Status {
			case "paid":
				paid = append(paid, p)
			case "pending":
				pending = append(pending, p)
			}
		}
	}
	received := 0
	for _, p := range paid {
		if p.AmountDue != nil {
			received += *p.AmountDue
		}
	}
	compliance := 0
	if len(active) > 0 {
		compliance = int(float64(len(paid)) / float64(len(active)) * 100)
	}
	return financeSummary{
		ReceivedCents: received,
		PendingCount:  len(pending),
		PaidCount:     len(paid),
		TotalMembers:  len(active),
		CompliancePct: compliance,
	}
}

func sortPayments(payments []db.FinancePayment) {
	sort.Slice(payments, func(i, j int) bool {
		pi, pj := payments[i], payments[j]
		statusI := 0
		if pi.Status != "pending" {
			statusI = 1
		}
		statusJ := 0
		if pj.Status != "pending" {
			statusJ = 1
		}
		if statusI != statusJ {
			return statusI < statusJ
		}
		return strings.ToLower(pi.PlayerName) < strings.ToLower(pj.PlayerName)
	})
}

func (h *FinanceHandler) ListPeriods(w http.ResponseWriter, r *http.Request) {
	groupIDStr := chi.URLParam(r, "groupID")
	groupID, err := uuid.Parse(groupIDStr)
	if err != nil {
		renderError(w, apierror.NotFound("group not found"))
		return
	}

	player := middleware.PlayerFromCtx(r.Context())
	if player == nil {
		renderError(w, apierror.Unauthorized())
		return
	}

	if !h.isMemberOrAdmin(r, groupID, player) {
		renderError(w, apierror.Forbidden("not a group member"))
		return
	}

	periods, err := h.Store.ListFinancePeriods(r.Context(), groupID)
	if err != nil {
		renderError(w, apierror.Internal("failed to list periods"))
		return
	}

	type periodItem struct {
		ID    uuid.UUID `json:"id"`
		Year  int       `json:"year"`
		Month int       `json:"month"`
	}
	items := make([]periodItem, len(periods))
	for i, p := range periods {
		items[i] = periodItem{ID: p.ID, Year: p.Year, Month: p.Month}
	}
	renderJSON(w, http.StatusOK, items)
}

func (h *FinanceHandler) GetPeriod(w http.ResponseWriter, r *http.Request) {
	groupIDStr := chi.URLParam(r, "groupID")
	groupID, err := uuid.Parse(groupIDStr)
	if err != nil {
		renderError(w, apierror.NotFound("group not found"))
		return
	}

	yearStr := chi.URLParam(r, "year")
	monthStr := chi.URLParam(r, "month")
	year, err1 := strconv.Atoi(yearStr)
	month, err2 := strconv.Atoi(monthStr)
	if err1 != nil || err2 != nil || month < 1 || month > 12 {
		renderError(w, apierror.Unprocessable("invalid year or month"))
		return
	}

	player := middleware.PlayerFromCtx(r.Context())
	if player == nil {
		renderError(w, apierror.Unauthorized())
		return
	}

	if !h.isMemberOrAdmin(r, groupID, player) {
		renderError(w, apierror.Forbidden("not a group member"))
		return
	}

	now := time.Now()
	if year == now.Year() && month == int(now.Month()) {
		if _, err := h.Store.GetOrCreateFinancePeriod(r.Context(), groupID, year, month); err != nil {
			renderError(w, apierror.Internal("failed to get or create period"))
			return
		}
	}

	period, err := h.Store.GetFinancePeriod(r.Context(), groupID, year, month)
	if err != nil {
		renderError(w, apierror.NotFound("period not found"))
		return
	}

	payments, err := h.Store.GetPaymentsForPeriod(r.Context(), period.ID)
	if err != nil {
		renderError(w, apierror.Internal("failed to get payments"))
		return
	}

	sortPayments(payments)
	summary := buildFinanceSummary(payments)
	renderJSON(w, http.StatusOK, map[string]any{
		"period_id": period.ID,
		"year":      period.Year,
		"month":     period.Month,
		"summary":   summary,
		"payments":  payments,
	})
}

func (h *FinanceHandler) UpdatePayment(w http.ResponseWriter, r *http.Request) {
	paymentIDStr := chi.URLParam(r, "paymentID")
	paymentID, err := uuid.Parse(paymentIDStr)
	if err != nil {
		renderError(w, apierror.NotFound("payment not found"))
		return
	}

	player := middleware.PlayerFromCtx(r.Context())
	if player == nil {
		renderError(w, apierror.Unauthorized())
		return
	}

	var body struct {
		Status      string  `json:"status"`
		PaymentType *string `json:"payment_type"`
	}
	if err := decodeJSON(r, &body); err != nil {
		renderError(w, err)
		return
	}

	payment, err := h.Store.GetFinancePayment(r.Context(), paymentID)
	if err != nil {
		renderError(w, apierror.NotFound("payment not found"))
		return
	}

	groupID, err := h.Store.GetPeriodGroupID(r.Context(), payment.PeriodID)
	if err != nil {
		renderError(w, apierror.Internal("failed to get group"))
		return
	}

	if !h.isGroupAdminOrSuperAdmin(r, groupID, player) {
		renderError(w, apierror.Forbidden("group admin required"))
		return
	}

	if body.Status == "paid" {
		if body.PaymentType == nil {
			renderError(w, apierror.Unprocessable("payment_type is required when marking as paid"))
			return
		}
		if *body.PaymentType != "monthly" && *body.PaymentType != "per_match" {
			renderError(w, apierror.Unprocessable("payment_type must be 'monthly' or 'per_match'"))
			return
		}

		group, err := h.Store.GetGroupByID(r.Context(), groupID)
		if err != nil {
			renderError(w, apierror.Internal("failed to get group"))
			return
		}

		amountCents := 0
		if *body.PaymentType == "monthly" && group.MonthlyAmount != nil {
			amountCents = int(*group.MonthlyAmount * 100)
		} else if *body.PaymentType == "per_match" && group.PerMatchAmount != nil {
			amountCents = int(*group.PerMatchAmount * 100)
		}

		updated, err := h.Store.MarkPaymentPaid(r.Context(), paymentID, *body.PaymentType, amountCents)
		if err != nil {
			renderError(w, apierror.Internal("failed to mark payment"))
			return
		}
		renderJSON(w, http.StatusOK, updated)
		return
	}

	updated, err := h.Store.MarkPaymentPending(r.Context(), paymentID)
	if err != nil {
		renderError(w, apierror.Internal("failed to mark payment"))
		return
	}
	renderJSON(w, http.StatusOK, updated)
}

func (h *FinanceHandler) isMemberOrAdmin(r *http.Request, groupID uuid.UUID, player *db.Player) bool {
	if player.Role == db.PlayerRoleAdmin {
		return true
	}
	member, err := h.Store.GetGroupMember(r.Context(), groupID, player.ID)
	return err == nil && member != nil
}

func (h *FinanceHandler) isGroupAdminOrSuperAdmin(r *http.Request, groupID uuid.UUID, player *db.Player) bool {
	if player.Role == db.PlayerRoleAdmin {
		return true
	}
	member, err := h.Store.GetGroupMember(r.Context(), groupID, player.ID)
	if err != nil || member == nil {
		return false
	}
	return member.Role == db.GroupMemberRoleAdmin
}
