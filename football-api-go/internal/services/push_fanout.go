package services

import (
	"context"

	"github.com/google/uuid"
)

// GroupAdminLister abstracts the lookup of admin player IDs for a group so
// that the fanout helper stays independent of the SQL pool — handlers wire
// their Store as the lister; tests inject a fake.
type GroupAdminLister interface {
	GetGroupAdminIDs(ctx context.Context, groupID uuid.UUID) ([]uuid.UUID, error)
}

// NotifyGroupAdmins fans out a push notification to every admin of a group,
// optionally skipping one player (typically the admin who triggered the action).
// Mirrors send_push_to_group_admins in the Python API for parity (PRD 044 §17).
//
// Returns the number of admins actually notified.
func NotifyGroupAdmins(
	ctx context.Context,
	lister GroupAdminLister,
	push PushService,
	groupID uuid.UUID,
	exclude *uuid.UUID,
	notification PushNotification,
) (int, error) {
	if lister == nil {
		return 0, nil
	}
	admins, err := lister.GetGroupAdminIDs(ctx, groupID)
	if err != nil {
		return 0, err
	}
	filtered := make([]uuid.UUID, 0, len(admins))
	for _, id := range admins {
		if exclude != nil && id == *exclude {
			continue
		}
		filtered = append(filtered, id)
	}
	if len(filtered) == 0 {
		return 0, nil
	}
	if err := push.SendToPlayers(ctx, filtered, notification); err != nil {
		return 0, err
	}
	return len(filtered), nil
}
