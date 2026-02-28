from datetime import datetime, timezone

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.db.repositories.base import BaseRepository
from app.models.invite import InviteToken


class InviteRepository(BaseRepository[InviteToken]):
    model = InviteToken

    def __init__(self, session: AsyncSession):
        super().__init__(session)

    async def get_by_token(self, token: str) -> InviteToken | None:
        result = await self.session.execute(
            select(InviteToken).where(InviteToken.token == token)
        )
        return result.scalar_one_or_none()

    async def get_valid_token(self, token: str) -> InviteToken | None:
        now = datetime.now(timezone.utc)
        result = await self.session.execute(
            select(InviteToken).where(
                InviteToken.token == token,
                InviteToken.used == False,
                InviteToken.expires_at > now,
            )
        )
        return result.scalar_one_or_none()
