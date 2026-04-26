import uuid
from datetime import datetime, timedelta, timezone
from uuid import uuid4

from fastapi import APIRouter
from sqlalchemy import select, update

from app.core.dependencies import DB, CurrentPlayer
from app.core.exceptions import ForbiddenError, NotFoundError
from app.models.mcp_token import MCPToken
from app.schemas.mcp_token import ExpiresIn, MCPTokenCreate, MCPTokenCreated, MCPTokenResponse

router = APIRouter(prefix="/mcp-tokens", tags=["mcp-tokens"])


def _compute_expires_at(expires_in: ExpiresIn | None) -> datetime | None:
    if expires_in == ExpiresIn.h24:
        return datetime.now(tz=timezone.utc) + timedelta(hours=24)
    if expires_in == ExpiresIn.d7:
        return datetime.now(tz=timezone.utc) + timedelta(days=7)
    return None


@router.post("", response_model=MCPTokenCreated, status_code=201)
async def create_token(body: MCPTokenCreate, db: DB, current_player: CurrentPlayer):
    raw, hashed, prefix = MCPToken.generate()
    expires_at = _compute_expires_at(body.expires_in)
    token_id = uuid4()
    now = datetime.now(tz=timezone.utc)

    token = MCPToken(
        id=token_id,
        player_id=current_player.id,
        name=body.name,
        token_hash=hashed,
        token_prefix=prefix,
        expires_at=expires_at,
        created_at=now,
    )
    db.add(token)
    await db.flush()

    return MCPTokenCreated(
        id=token_id,
        name=body.name,
        token=raw,
        token_prefix=prefix,
        expires_at=expires_at,
        created_at=now,
    )


@router.get("", response_model=list[MCPTokenResponse])
async def list_tokens(db: DB, current_player: CurrentPlayer):
    result = await db.execute(
        select(MCPToken)
        .where(MCPToken.player_id == current_player.id, MCPToken.revoked_at.is_(None))
        .order_by(MCPToken.created_at.desc())
    )
    tokens = result.scalars().all()
    now = datetime.now(tz=timezone.utc)

    return [
        MCPTokenResponse(
            id=t.id,
            name=t.name,
            token_prefix=t.token_prefix,
            expires_at=t.expires_at,
            created_at=t.created_at,
            last_used_at=t.last_used_at,
            is_expired=t.expires_at is not None and t.expires_at < now,
        )
        for t in tokens
    ]


@router.delete("/{token_id}", status_code=204)
async def revoke_token(token_id: uuid.UUID, db: DB, current_player: CurrentPlayer):
    result = await db.execute(
        select(MCPToken).where(MCPToken.id == token_id, MCPToken.revoked_at.is_(None))
    )
    token = result.scalar_one_or_none()

    if not token:
        raise NotFoundError("Token not found")
    if token.player_id != current_player.id:
        raise ForbiddenError("Cannot revoke another player's token")

    await db.execute(
        update(MCPToken)
        .where(MCPToken.id == token_id)
        .values(revoked_at=datetime.now(tz=timezone.utc))
    )
