package services

// PushNotification represents the payload for a push notification.
type PushNotification struct {
	Title string
	Body  string
	URL   string
}

// SendPushToSubscription sends a push notification to a single subscription.
// This is a stub implementation — actual VAPID push sending requires a
// dedicated library and is deferred to a future phase.
func SendPushToSubscription(endpoint, p256dh, auth, vapidPrivateKey, vapidEmail string, n PushNotification) error {
	// TODO: implement VAPID push via golang.org/x/crypto/acme or webpush-go
	return nil
}
