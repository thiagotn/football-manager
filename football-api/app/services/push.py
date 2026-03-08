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
