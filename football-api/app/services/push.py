"""Push notification service — fire-and-forget helper.

Call send_push() wherever you want to trigger a notification.
If VAPID keys are not configured, the call is a no-op.
"""
import asyncio
import json
import uuid

import structlog
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.config import get_settings
from app.models.group import GroupMember, GroupMemberRole
from app.models.push_subscription import PushSubscription

logger = structlog.get_logger()


def _send_webpush(endpoint: str, p256dh: str, auth: str, payload: str) -> int | None:
    """Synchronous webpush call — runs in a thread executor."""
    from pywebpush import WebPushException, webpush

    settings = get_settings()
    try:
        resp = webpush(
            subscription_info={"endpoint": endpoint, "keys": {"p256dh": p256dh, "auth": auth}},
            data=payload,
            vapid_private_key=settings.vapid_private_key,
            vapid_claims={"sub": f"mailto:{settings.vapid_claims_email}"},
        )
        return resp.status_code if resp else None
    except WebPushException as exc:
        status = exc.response.status_code if exc.response else None
        logger.warning("push_send_failed", status=status, error=str(exc)[:200])
        return status


async def send_push(
    db: AsyncSession,
    player_id: uuid.UUID,
    title: str,
    body: str,
    url: str = "/",
) -> None:
    """Send a push notification to all subscriptions of a player.

    Silently removes expired subscriptions (404/410).
    Does nothing if VAPID keys are not set.
    """
    settings = get_settings()
    if not settings.vapid_private_key or not settings.vapid_public_key:
        logger.warning("push_skipped_no_vapid", player_id=str(player_id), title=title)
        return

    result = await db.execute(
        select(PushSubscription).where(PushSubscription.player_id == player_id)
    )
    subscriptions = result.scalars().all()
    if not subscriptions:
        return

    payload = json.dumps({"title": title, "body": body, "url": url})

    for sub in subscriptions:
        status = await asyncio.to_thread(
            _send_webpush, sub.endpoint, sub.p256dh, sub.auth, payload
        )
        if status in (404, 410):
            await db.delete(sub)
            logger.info(
                "push_subscription_removed",
                player_id=str(player_id),
                endpoint=sub.endpoint[:60],
            )
        elif status and status < 300:
            logger.info(
                "push_sent",
                player_id=str(player_id),
                title=title,
                url=url,
                status=status,
            )


async def send_push_to_group_admins(
    db: AsyncSession,
    group_id: uuid.UUID,
    *,
    title: str,
    body: str,
    url: str = "/",
    exclude: uuid.UUID | None = None,
) -> None:
    """Fan-out push to every admin of a group, optionally excluding one player.

    Used for cross-admin coordination (e.g. notifying other admins when one of
    them accepts a waitlist candidate). Mirrors NotifyGroupAdmins in the Go
    API for parity (PRD 044 §17).
    """
    result = await db.execute(
        select(GroupMember.player_id).where(
            GroupMember.group_id == group_id,
            GroupMember.role == GroupMemberRole.ADMIN,
        )
    )
    admin_ids = [aid for aid in result.scalars().all() if aid != exclude]
    if not admin_ids:
        return
    await asyncio.gather(*[
        send_push(db, aid, title=title, body=body, url=url)
        for aid in admin_ids
    ], return_exceptions=True)
