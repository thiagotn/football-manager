from uuid import UUID

from sqlalchemy import func, select
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy.orm import selectinload

from app.db.repositories.base import BaseRepository
from app.models.match import Match, Attendance, AttendanceStatus


class MatchRepository(BaseRepository[Match]):
    model = Match

    def __init__(self, session: AsyncSession):
        super().__init__(session)

    async def get_by_hash(self, hash: str) -> Match | None:
        result = await self.session.execute(
            select(Match).where(Match.hash == hash)
        )
        return result.scalar_one_or_none()

    async def get_with_attendances(self, match_id: UUID) -> Match | None:
        result = await self.session.execute(
            select(Match)
            .options(
                selectinload(Match.attendances).selectinload(Attendance.player),
                selectinload(Match.group),
            )
            .where(Match.id == match_id)
        )
        return result.scalar_one_or_none()

    async def get_by_hash_with_attendances(self, hash: str) -> Match | None:
        result = await self.session.execute(
            select(Match)
            .options(
                selectinload(Match.attendances).selectinload(Attendance.player),
                selectinload(Match.group),
            )
            .where(Match.hash == hash)
        )
        return result.scalar_one_or_none()

    async def get_group_matches(self, group_id: UUID) -> list[Match]:
        result = await self.session.execute(
            select(Match)
            .where(Match.group_id == group_id)
            .order_by(Match.match_date.desc())
        )
        return list(result.scalars().all())

    async def get_attendance(self, match_id: UUID, player_id: UUID) -> Attendance | None:
        result = await self.session.execute(
            select(Attendance).where(
                Attendance.match_id == match_id,
                Attendance.player_id == player_id,
            )
        )
        return result.scalar_one_or_none()

    async def count_confirmed(self, match_id: UUID, exclude_player_id: UUID | None = None) -> int:
        q = select(func.count()).where(
            Attendance.match_id == match_id,
            Attendance.status == AttendanceStatus.CONFIRMED,
        )
        if exclude_player_id:
            q = q.where(Attendance.player_id != exclude_player_id)
        result = await self.session.execute(q)
        return result.scalar_one()

    async def create_pending_attendances(self, match_id: UUID, player_ids: list[UUID]) -> None:
        for player_id in player_ids:
            self.session.add(Attendance(match_id=match_id, player_id=player_id, status=AttendanceStatus.PENDING))
        await self.session.flush()

    async def upsert_attendance(
        self, match_id: UUID, player_id: UUID, status: AttendanceStatus
    ) -> Attendance:
        attendance = await self.get_attendance(match_id, player_id)
        if attendance:
            attendance.status = status
            await self.session.flush()
            await self.session.refresh(attendance)
        else:
            attendance = Attendance(match_id=match_id, player_id=player_id, status=status)
            self.session.add(attendance)
            await self.session.flush()
            await self.session.refresh(attendance)
        return attendance
