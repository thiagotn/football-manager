from uuid import UUID

from datetime import date

from sqlalchemy import and_, delete, func, or_, select, update
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy.orm import selectinload

from app.db.repositories.base import BaseRepository
from app.models.match import Attendance, AttendanceStatus, Match, MatchStatus


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

    async def close_past_matches(self) -> int:
        """Fecha partidas abertas cuja data já passou ou cujo horário de hoje já encerrou."""
        today = date.today()
        current_time = func.current_time()
        result = await self.session.execute(
            update(Match)
            .where(
                Match.status == MatchStatus.OPEN,
                or_(
                    Match.match_date < today,
                    and_(
                        Match.match_date == today,
                        Match.end_time.isnot(None),
                        Match.end_time <= current_time,
                    ),
                    and_(
                        Match.match_date == today,
                        Match.end_time.is_(None),
                        Match.start_time <= current_time,
                    ),
                ),
            )
            .values(status=MatchStatus.CLOSED)
        )
        await self.session.flush()
        return result.rowcount

    async def get_last_match(self, group_id: UUID) -> Match | None:
        result = await self.session.execute(
            select(Match)
            .where(Match.group_id == group_id)
            .order_by(Match.match_date.desc())
            .limit(1)
        )
        return result.scalar_one_or_none()

    async def has_open_match(self, group_id: UUID) -> bool:
        result = await self.session.execute(
            select(func.count()).where(
                Match.group_id == group_id,
                Match.status == MatchStatus.OPEN,
            )
        )
        return result.scalar_one() > 0

    async def get_attendance_player_ids(self, match_id: UUID) -> list[UUID]:
        result = await self.session.execute(
            select(Attendance.player_id).where(Attendance.match_id == match_id)
        )
        return list(result.scalars().all())

    async def delete_player_attendances_in_open_matches(self, group_id: UUID, player_id: UUID) -> None:
        open_match_ids = select(Match.id).where(
            Match.group_id == group_id,
            Match.status == MatchStatus.OPEN,
        )
        await self.session.execute(
            delete(Attendance).where(
                Attendance.player_id == player_id,
                Attendance.match_id.in_(open_match_ids),
            )
        )
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
