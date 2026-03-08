from fastapi import APIRouter, Request
from pydantic import BaseModel
from sqlalchemy import delete, select

from app.core.config import get_settings
from app.core.dependencies import CurrentPlayer, DB
from app.models.push_subscription import PushSubscription

router = APIRouter(prefix="/push", tags=["push"])


class PushSubscribeBody(BaseModel):
    endpoint: str
    keys: dict  # {"p256dh": str, "auth": str}
    user_agent: str | None = None


@router.get("/vapid-public-key")
async def get_vapid_public_key():
    return {"public_key": get_settings().vapid_public_key}


@router.post("/subscribe", status_code=201)
async def subscribe(body: PushSubscribeBody, db: DB, current: CurrentPlayer, request: Request):
    """Save or update a push subscription for the authenticated player."""
    p256dh = body.keys.get("p256dh", "")
    auth = body.keys.get("auth", "")
    user_agent = body.user_agent or request.headers.get("user-agent")

    # Upsert: update if endpoint already exists for this player, else insert
    result = await db.execute(
        select(PushSubscription).where(
            PushSubscription.player_id == current.id,
            PushSubscription.endpoint == body.endpoint,
        )
    )
    sub = result.scalar_one_or_none()

    if sub:
        sub.p256dh = p256dh
        sub.auth = auth
        sub.user_agent = user_agent
    else:
        sub = PushSubscription(
            player_id=current.id,
            endpoint=body.endpoint,
            p256dh=p256dh,
            auth=auth,
            user_agent=user_agent,
        )
        db.add(sub)

    await db.flush()
    return {"status": "subscribed"}


@router.delete("/subscribe", status_code=204)
async def unsubscribe(db: DB, current: CurrentPlayer):
    """Remove all push subscriptions for the authenticated player."""
    await db.execute(
        delete(PushSubscription).where(PushSubscription.player_id == current.id)
    )
    await db.flush()
