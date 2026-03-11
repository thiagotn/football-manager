from uuid import UUID

from sqlalchemy import delete, select
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy.orm import selectinload

from app.db.repositories.base import BaseRepository
from app.models.team import MatchTeam, MatchTeamPlayer


class TeamRepository(BaseRepository[MatchTeam]):
    model = MatchTeam

    def __init__(self, session: AsyncSession):
        super().__init__(session)

    async def get_by_match(self, match_id: UUID) -> list[MatchTeam]:
        result = await self.session.execute(
            select(MatchTeam)
            .options(selectinload(MatchTeam.players).selectinload(MatchTeamPlayer.player))
            .where(MatchTeam.match_id == match_id)
            .order_by(MatchTeam.position)
        )
        return list(result.scalars().all())

    async def delete_by_match(self, match_id: UUID) -> None:
        await self.session.execute(
            delete(MatchTeam).where(MatchTeam.match_id == match_id)
        )
        await self.session.flush()

    async def create_team(
        self, match_id: UUID, name: str, color: str | None, position: int
    ) -> MatchTeam:
        team = MatchTeam(match_id=match_id, name=name, color=color, position=position)
        self.session.add(team)
        await self.session.flush()
        return team

    async def add_player(
        self, team_id: UUID, player_id: UUID, is_reserve: bool = False
    ) -> MatchTeamPlayer:
        tp = MatchTeamPlayer(team_id=team_id, player_id=player_id, is_reserve=is_reserve)
        self.session.add(tp)
        await self.session.flush()
        return tp
