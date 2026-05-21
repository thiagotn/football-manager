package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thiagotn/football-manager/football-api-go/internal/apierror"
	"github.com/thiagotn/football-manager/football-api-go/internal/db"
	"github.com/thiagotn/football-manager/football-api-go/internal/middleware"
)

type financeHandler struct {
	pool *pgxpool.Pool
}

func NewFinanceHandler(pool *pgxpool.Pool) *financeHandler {
	return &financeHandler{pool: pool}
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

func (h *financeHandler) ListPeriods(w http.ResponseWriter, r *http.Request) {
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

	periods, err := db.ListFinancePeriods(r.Context(), h.pool, groupID)
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

func (h *financeHandler) GetPeriod(w http.ResponseWriter, r *http.Request) {
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
		if _, err := db.GetOrCreateFinancePeriod(r.Context(), h.pool, groupID, year, month); err != nil {
			renderError(w, apierror.Internal("failed to get or create period"))
			return
		}
	}

	period, err := db.GetFinancePeriod(r.Context(), h.pool, groupID, year, month)
	if err != nil {
		renderError(w, apierror.NotFound("period not found"))
		return
	}

	payments, err := db.GetPaymentsForPeriod(r.Context(), h.pool, period.ID)
	if err != nil {
		renderError(w, apierror.Internal("failed to get payments"))
		return
	}

	summary := buildFinanceSummary(payments)
	renderJSON(w, http.StatusOK, map[string]any{
		"period_id": period.ID,
		"year":      period.Year,
		"month":     period.Month,
		"summary":   summary,
		"payments":  payments,
	})
}

func (h *financeHandler) UpdatePayment(w http.ResponseWriter, r *http.Request) {
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

	payment, err := db.GetFinancePayment(r.Context(), h.pool, paymentID)
	if err != nil {
		renderError(w, apierror.NotFound("payment not found"))
		return
	}

	groupID, err := db.GetPeriodGroupID(r.Context(), h.pool, payment.PeriodID)
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

		group, err := db.GetGroupByID(r.Context(), h.pool, groupID)
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

		updated, err := db.MarkPaymentPaid(r.Context(), h.pool, paymentID, *body.PaymentType, amountCents)
		if err != nil {
			renderError(w, apierror.Internal("failed to mark payment"))
			return
		}
		renderJSON(w, http.StatusOK, updated)
		return
	}

	updated, err := db.MarkPaymentPending(r.Context(), h.pool, paymentID)
	if err != nil {
		renderError(w, apierror.Internal("failed to mark payment"))
		return
	}
	renderJSON(w, http.StatusOK, updated)
}

func (h *financeHandler) isMemberOrAdmin(r *http.Request, groupID uuid.UUID, player *db.Player) bool {
	if player.Role == db.PlayerRoleAdmin {
		return true
	}
	member, err := db.GetGroupMember(r.Context(), h.pool, groupID, player.ID)
	return err == nil && member != nil
}

func (h *financeHandler) isGroupAdminOrSuperAdmin(r *http.Request, groupID uuid.UUID, player *db.Player) bool {
	if player.Role == db.PlayerRoleAdmin {
		return true
	}
	member, err := db.GetGroupMember(r.Context(), h.pool, groupID, player.ID)
	if err != nil || member == nil {
		return false
	}
	return member.Role == db.GroupMemberRoleAdmin
}
