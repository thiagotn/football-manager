package services

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PushNotification represents the payload for a push notification.
type PushNotification struct {
	Title string
	Body  string
	URL   string
}

// PushService interface for dispatching push notifications.
type PushService interface {
	SendToPlayers(ctx context.Context, playerIDs []uuid.UUID, notification PushNotification) error
}

// pushService implements PushService.
type pushService struct {
	pool *pgxpool.Pool
}

// NewPushService creates a new push service instance.
func NewPushService(pool *pgxpool.Pool) PushService {
	return &pushService{pool: pool}
}

// SendToPlayers sends a push notification to multiple players.
// This is a stub implementation — actual VAPID push sending requires a
// dedicated library and is deferred to a future phase.
func (s *pushService) SendToPlayers(ctx context.Context, playerIDs []uuid.UUID, notification PushNotification) error {
	// TODO: implement VAPID push via golang.org/x/crypto/acme or webpush-go
	// For now, this is a no-op that prevents errors
	return nil
}

// SendPushToSubscription sends a push notification to a single subscription.
// This is a stub implementation — actual VAPID push sending requires a
// dedicated library and is deferred to a future phase.
func SendPushToSubscription(endpoint, p256dh, auth, vapidPrivateKey, vapidEmail string, n PushNotification) error {
	// TODO: implement VAPID push via golang.org/x/crypto/acme or webpush-go
	return nil
}
