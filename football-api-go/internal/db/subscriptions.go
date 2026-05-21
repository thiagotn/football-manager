package db

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PlayerSubscription struct {
	ID                uuid.UUID  `json:"id"`
	PlayerID          uuid.UUID  `json:"player_id"`
	Plan              string     `json:"plan"`
	Status            string     `json:"status"`
	GatewayCustomerID *string    `json:"gateway_customer_id,omitempty"`
	GatewaySubID      *string    `json:"gateway_sub_id,omitempty"`
	BillingCycle      string     `json:"billing_cycle"`
	CurrentPeriodEnd  *time.Time `json:"current_period_end,omitempty"`
	GracePeriodEnd    *time.Time `json:"grace_period_end,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

const subCols = `id, player_id, plan, status, gateway_customer_id, gateway_sub_id,
	billing_cycle, current_period_end, grace_period_end, created_at, updated_at`

func scanSubscription(scanFn func(dest ...any) error) (*PlayerSubscription, error) {
	var s PlayerSubscription
	err := scanFn(
		&s.ID, &s.PlayerID, &s.Plan, &s.Status,
		&s.GatewayCustomerID, &s.GatewaySubID,
		&s.BillingCycle, &s.CurrentPeriodEnd, &s.GracePeriodEnd,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func GetOrCreateSubscription(ctx context.Context, pool *pgxpool.Pool, playerID uuid.UUID) (*PlayerSubscription, error) {
	sub, err := GetSubscriptionByPlayer(ctx, pool, playerID)
	if err == nil {
		return sub, nil
	}
	row := pool.QueryRow(ctx,
		`INSERT INTO player_subscriptions (player_id, plan, status)
		 VALUES ($1, 'free', 'active')
		 ON CONFLICT (player_id) DO UPDATE SET player_id = EXCLUDED.player_id
		 RETURNING `+subCols,
		playerID,
	)
	return scanSubscription(row.Scan)
}

func GetSubscriptionByPlayer(ctx context.Context, pool *pgxpool.Pool, playerID uuid.UUID) (*PlayerSubscription, error) {
	row := pool.QueryRow(ctx,
		`SELECT `+subCols+` FROM player_subscriptions WHERE player_id=$1`,
		playerID,
	)
	s, err := scanSubscription(row.Scan)
	if err != nil {
		return nil, ErrNotFound
	}
	return s, nil
}

func GetSubscriptionByGatewayCustomer(ctx context.Context, pool *pgxpool.Pool, customerID string) (*PlayerSubscription, error) {
	row := pool.QueryRow(ctx,
		`SELECT `+subCols+` FROM player_subscriptions WHERE gateway_customer_id=$1`,
		customerID,
	)
	s, err := scanSubscription(row.Scan)
	if err != nil {
		return nil, ErrNotFound
	}
	return s, nil
}

type UpdateSubscriptionParams struct {
	Plan              string
	Status            string
	GatewayCustomerID *string
	GatewaySubID      *string
	BillingCycle      *string
	CurrentPeriodEnd  *time.Time
	GracePeriodEnd    *time.Time
}

func UpdateSubscription(ctx context.Context, pool *pgxpool.Pool, playerID uuid.UUID, p UpdateSubscriptionParams) (*PlayerSubscription, error) {
	sub, err := GetOrCreateSubscription(ctx, pool, playerID)
	if err != nil {
		return nil, err
	}

	if p.GatewayCustomerID != nil {
		sub.GatewayCustomerID = p.GatewayCustomerID
	}
	if p.GatewaySubID != nil {
		sub.GatewaySubID = p.GatewaySubID
	}
	if p.BillingCycle != nil {
		sub.BillingCycle = *p.BillingCycle
	}
	if p.CurrentPeriodEnd != nil {
		sub.CurrentPeriodEnd = p.CurrentPeriodEnd
	}
	if p.GracePeriodEnd != nil {
		sub.GracePeriodEnd = p.GracePeriodEnd
	}

	row := pool.QueryRow(ctx,
		`UPDATE player_subscriptions
		 SET plan=$1, status=$2, gateway_customer_id=$3, gateway_sub_id=$4,
		     billing_cycle=$5, current_period_end=$6, grace_period_end=$7, updated_at=NOW()
		 WHERE player_id=$8
		 RETURNING `+subCols,
		p.Plan, p.Status, sub.GatewayCustomerID, sub.GatewaySubID,
		sub.BillingCycle, sub.CurrentPeriodEnd, sub.GracePeriodEnd, playerID,
	)
	return scanSubscription(row.Scan)
}

func CountAdminGroups(ctx context.Context, pool *pgxpool.Pool, playerID uuid.UUID) (int, error) {
	var n int
	err := pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM group_members WHERE player_id=$1 AND role='admin'`,
		playerID,
	).Scan(&n)
	return n, err
}
