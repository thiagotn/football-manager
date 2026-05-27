package db

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5"
)

type FinancePeriod struct {
	ID        uuid.UUID `json:"id"`
	GroupID   uuid.UUID `json:"group_id"`
	Year      int       `json:"year"`
	Month     int       `json:"month"`
	CreatedAt time.Time `json:"created_at"`
}

type FinancePayment struct {
	ID          uuid.UUID  `json:"id"`
	PeriodID    uuid.UUID  `json:"period_id"`
	PlayerID    uuid.UUID  `json:"player_id"`
	PlayerName  string     `json:"player_name"`
	PaymentType *string    `json:"payment_type"`
	AmountDue   *int       `json:"amount_due"`
	Status      string     `json:"status"`
	PaidAt      *time.Time `json:"paid_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func ListFinancePeriods(ctx context.Context, pool *pgxpool.Pool, groupID uuid.UUID) ([]FinancePeriod, error) {
	rows, err := pool.Query(ctx,
		`SELECT id, group_id, year, month, created_at
		 FROM finance_periods
		 WHERE group_id = $1
		 ORDER BY year DESC, month DESC`,
		groupID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var periods []FinancePeriod
	for rows.Next() {
		var p FinancePeriod
		if err := rows.Scan(&p.ID, &p.GroupID, &p.Year, &p.Month, &p.CreatedAt); err != nil {
			return nil, err
		}
		periods = append(periods, p)
	}
	return periods, rows.Err()
}

func GetOrCreateFinancePeriod(ctx context.Context, pool *pgxpool.Pool, groupID uuid.UUID, year, month int) (*FinancePeriod, error) {
	var p FinancePeriod
	err := pool.QueryRow(ctx,
		`INSERT INTO finance_periods (group_id, year, month)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (group_id, year, month) DO UPDATE SET group_id = EXCLUDED.group_id
		 RETURNING id, group_id, year, month, created_at`,
		groupID, year, month,
	).Scan(&p.ID, &p.GroupID, &p.Year, &p.Month, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func GetFinancePeriod(ctx context.Context, pool *pgxpool.Pool, groupID uuid.UUID, year, month int) (*FinancePeriod, error) {
	var p FinancePeriod
	err := pool.QueryRow(ctx,
		`SELECT id, group_id, year, month, created_at
		 FROM finance_periods
		 WHERE group_id=$1 AND year=$2 AND month=$3`,
		groupID, year, month,
	).Scan(&p.ID, &p.GroupID, &p.Year, &p.Month, &p.CreatedAt)
	if err != nil {
		return nil, ErrNotFound
	}
	return &p, nil
}

func GetPaymentsForPeriod(ctx context.Context, pool *pgxpool.Pool, periodID uuid.UUID) ([]FinancePayment, error) {
	rows, err := pool.Query(ctx,
		`SELECT id, period_id, player_id, player_name, payment_type, amount_due,
		        status, paid_at, created_at, updated_at
		 FROM finance_payments
		 WHERE period_id = $1
		 ORDER BY
		   CASE status WHEN 'pending' THEN 0 ELSE 1 END,
		   lower(player_name)`,
		periodID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var payments []FinancePayment
	for rows.Next() {
		var p FinancePayment
		if err := rows.Scan(
			&p.ID, &p.PeriodID, &p.PlayerID, &p.PlayerName,
			&p.PaymentType, &p.AmountDue, &p.Status, &p.PaidAt,
			&p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, err
		}
		payments = append(payments, p)
	}
	return payments, rows.Err()
}

func GetFinancePayment(ctx context.Context, pool *pgxpool.Pool, paymentID uuid.UUID) (*FinancePayment, error) {
	var p FinancePayment
	err := pool.QueryRow(ctx,
		`SELECT id, period_id, player_id, player_name, payment_type, amount_due,
		        status, paid_at, created_at, updated_at
		 FROM finance_payments WHERE id=$1`,
		paymentID,
	).Scan(
		&p.ID, &p.PeriodID, &p.PlayerID, &p.PlayerName,
		&p.PaymentType, &p.AmountDue, &p.Status, &p.PaidAt,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, ErrNotFound
	}
	return &p, nil
}

func GetPeriodGroupID(ctx context.Context, pool *pgxpool.Pool, periodID uuid.UUID) (uuid.UUID, error) {
	var gid uuid.UUID
	err := pool.QueryRow(ctx,
		`SELECT group_id FROM finance_periods WHERE id=$1`, periodID,
	).Scan(&gid)
	if err != nil {
		return uuid.Nil, ErrNotFound
	}
	return gid, nil
}

func MarkPaymentPaid(ctx context.Context, pool *pgxpool.Pool, paymentID uuid.UUID, paymentType string, amountCents int) (*FinancePayment, error) {
	var p FinancePayment
	err := pool.QueryRow(ctx,
		`UPDATE finance_payments
		 SET status='paid', payment_type=$2, amount_due=$3, paid_at=NOW(), updated_at=NOW()
		 WHERE id=$1
		 RETURNING id, period_id, player_id, player_name, payment_type, amount_due,
		           status, paid_at, created_at, updated_at`,
		paymentID, paymentType, amountCents,
	).Scan(
		&p.ID, &p.PeriodID, &p.PlayerID, &p.PlayerName,
		&p.PaymentType, &p.AmountDue, &p.Status, &p.PaidAt,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func MarkPaymentPending(ctx context.Context, pool *pgxpool.Pool, paymentID uuid.UUID) (*FinancePayment, error) {
	var p FinancePayment
	err := pool.QueryRow(ctx,
		`UPDATE finance_payments
		 SET status='pending', payment_type=NULL, amount_due=NULL, paid_at=NULL, updated_at=NOW()
		 WHERE id=$1
		 RETURNING id, period_id, player_id, player_name, payment_type, amount_due,
		           status, paid_at, created_at, updated_at`,
		paymentID,
	).Scan(
		&p.ID, &p.PeriodID, &p.PlayerID, &p.PlayerName,
		&p.PaymentType, &p.AmountDue, &p.Status, &p.PaidAt,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func EnsureMemberInCurrentPeriod(ctx context.Context, pool *pgxpool.Pool, groupID, playerID uuid.UUID, playerName string) error {
	now := time.Now()
	year, month := now.Year(), int(now.Month())

	var periodID uuid.UUID
	err := pool.QueryRow(ctx,
		`SELECT id FROM finance_periods WHERE group_id=$1 AND year=$2 AND month=$3`,
		groupID, year, month,
	).Scan(&periodID)
	if err == pgx.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}

	var existing uuid.UUID
	err = pool.QueryRow(ctx,
		`SELECT id FROM finance_payments WHERE period_id=$1 AND player_id=$2`,
		periodID, playerID,
	).Scan(&existing)
	if err == nil {
		return nil
	}
	if err != pgx.ErrNoRows {
		return err
	}

	_, err = pool.Exec(ctx,
		`INSERT INTO finance_payments (id, period_id, player_id, player_name, status, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, 'pending', NOW(), NOW())`,
		uuid.New(), periodID, playerID, playerName,
	)
	return err
}
