from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.db.repositories.base import BaseRepository
from app.models.player import Player


class PlayerRepository(BaseRepository[Player]):
    model = Player

    def __init__(self, session: AsyncSession):
        super().__init__(session)

    async def get_by_whatsapp(self, whatsapp: str) -> Player | None:
        result = await self.session.execute(
            select(Player).where(Player.whatsapp == whatsapp)
        )
        return result.scalar_one_or_none()

    async def get_active(self) -> list[Player]:
        result = await self.session.execute(
            select(Player).where(Player.active == True).order_by(Player.name)
        )
        return list(result.scalars().all())
