import hashlib
import secrets
from datetime import datetime, timedelta, timezone
from uuid import UUID

from sqlalchemy import select, update
from sqlalchemy.ext.asyncio import AsyncSession

from app.models.refresh_token import RefreshToken

REFRESH_TOKEN_DAYS = 30


class RefreshTokenRepository:
    def __init__(self, db: AsyncSession):
        self.db = db

    def _hash(self, token: str) -> str:
        return hashlib.sha256(token.encode()).hexdigest()

    async def create(self, player_id: UUID) -> str:
        token = secrets.token_urlsafe(32)
        rt = RefreshToken(
            player_id=player_id,
            token_hash=self._hash(token),
            expires_at=datetime.now(timezone.utc) + timedelta(days=REFRESH_TOKEN_DAYS),
        )
        self.db.add(rt)
        await self.db.flush()
        return token

    async def get_valid(self, token: str) -> RefreshToken | None:
        result = await self.db.execute(
            select(RefreshToken).where(
                RefreshToken.token_hash == self._hash(token),
                RefreshToken.revoked_at.is_(None),
                RefreshToken.expires_at > datetime.now(timezone.utc),
            )
        )
        return result.scalar_one_or_none()

    async def revoke(self, rt: RefreshToken) -> None:
        rt.revoked_at = datetime.now(timezone.utc)
        await self.db.flush()

    async def revoke_all_for_player(self, player_id: UUID) -> None:
        await self.db.execute(
            update(RefreshToken)
            .where(RefreshToken.player_id == player_id, RefreshToken.revoked_at.is_(None))
            .values(revoked_at=datetime.now(timezone.utc))
        )
        await self.db.flush()
