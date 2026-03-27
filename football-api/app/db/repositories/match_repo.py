from uuid import UUID

from datetime import datetime, timezone, timedelta

from datetime import date as date_type

from sqlalchemy import and_, delete, func, or_, select, update
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy.orm import selectinload

from app.db.repositories.base import BaseRepository
from app.models.group import GroupMember
from app.models.match import Attendance, AttendanceStatus, Match, MatchStatus


class MatchRepository(BaseRepository[Match]):
    model = Match

    def __init__(self, session: AsyncSession):
        super().__init__(session)

    async def next_number_for_group(self, group_id: UUID) -> int:
        result = await self.session.execute(
            select(func.coalesce(func.max(Match.number), 0)).where(Match.group_id == group_id)
        )
        return result.scalar_one() + 1

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

    async def get_player_matches(self, player_id: UUID) -> list[tuple[Match, str, str, AttendanceStatus | None]]:
        """All matches from groups the player belongs to, with group name, group timezone and their attendance status."""
        from app.models.group import Group
        stmt = (
            select(Match, Group.name.label("group_name"), Group.timezone.label("group_timezone"), Attendance.status.label("my_attendance"))
            .join(Group, Match.group_id == Group.id)
            .join(GroupMember, and_(GroupMember.group_id == Match.group_id, GroupMember.player_id == player_id))
            .outerjoin(Attendance, and_(Attendance.match_id == Match.id, Attendance.player_id == player_id))
            .order_by(Match.match_date.desc(), Match.start_time.desc())
        )
        result = await self.session.execute(stmt)
        return [(row.Match, row.group_name, row.group_timezone, row.my_attendance) for row in result.all()]

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
        """
        Atualiza o status das partidas usando horário de Brasília (UTC-3) explícito,
        sem depender da configuração de timezone do container/DB.

        - OPEN/IN_PROGRESS → CLOSED: data anterior a hoje (Brasil)
        - OPEN → IN_PROGRESS: partida de hoje (Brasil) cujo start_time já passou
        """
        BRAZIL = timezone(timedelta(hours=-3))
        now = datetime.now(BRAZIL)
        today = now.date()
        now_time = now.time().replace(tzinfo=None)

        # Fecha partidas de dias anteriores
        r1 = await self.session.execute(
            update(Match)
            .where(Match.status == MatchStatus.OPEN, Match.match_date < today)
            .values(status=MatchStatus.CLOSED)
        )
        r2 = await self.session.execute(
            update(Match)
            .where(Match.status == MatchStatus.IN_PROGRESS, Match.match_date < today)
            .values(status=MatchStatus.CLOSED)
        )

        # Marca como em andamento: partidas abertas de hoje cujo start_time já passou
        r3 = await self.session.execute(
            update(Match)
            .where(
                Match.status == MatchStatus.OPEN,
                Match.match_date == today,
                Match.start_time <= now_time,
            )
            .values(status=MatchStatus.IN_PROGRESS)
        )

        # Fecha partidas em andamento de hoje cujo end_time já passou
        r4 = await self.session.execute(
            update(Match)
            .where(
                Match.status == MatchStatus.IN_PROGRESS,
                Match.match_date == today,
                Match.end_time.is_not(None),
                Match.end_time <= now_time,
            )
            .values(status=MatchStatus.CLOSED)
        )

        await self.session.flush()
        return r1.rowcount + r2.rowcount + r3.rowcount + r4.rowcount

    async def get_last_match(self, group_id: UUID) -> Match | None:
        result = await self.session.execute(
            select(Match)
            .where(Match.group_id == group_id)
            .order_by(Match.match_date.desc())
            .limit(1)
        )
        return result.scalar_one_or_none()

    async def get_open_matches(self, group_id: UUID) -> list[Match]:
        result = await self.session.execute(
            select(Match).where(
                Match.group_id == group_id,
                Match.status == MatchStatus.OPEN,
            )
        )
        return list(result.scalars().all())

    async def get_active_matches(self, group_id: UUID) -> list[Match]:
        """Retorna partidas abertas ou em andamento de um grupo."""
        result = await self.session.execute(
            select(Match).where(
                Match.group_id == group_id,
                Match.status.in_([MatchStatus.OPEN, MatchStatus.IN_PROGRESS]),
            )
        )
        return list(result.scalars().all())

    async def has_open_match(self, group_id: UUID) -> bool:
        result = await self.session.execute(
            select(func.count()).where(
                Match.group_id == group_id,
                Match.status.in_([MatchStatus.OPEN, MatchStatus.IN_PROGRESS]),
            )
        )
        return result.scalar_one() > 0

    async def get_confirmed_player_ids(self, match_id: UUID) -> list[UUID]:
        result = await self.session.execute(
            select(Attendance.player_id).where(
                Attendance.match_id == match_id,
                Attendance.status == AttendanceStatus.CONFIRMED,
            )
        )
        return list(result.scalars().all())

    async def get_in_progress_candidates(self) -> list[Match]:
        """Matches that will transition OPEN → IN_PROGRESS on the next close_past_matches call."""
        BRAZIL = timezone(timedelta(hours=-3))
        now = datetime.now(BRAZIL)
        today = now.date()
        now_time = now.time().replace(tzinfo=None)
        result = await self.session.execute(
            select(Match)
            .options(selectinload(Match.group))
            .where(
                Match.status == MatchStatus.OPEN,
                Match.match_date == today,
                Match.start_time <= now_time,
            )
        )
        return list(result.scalars().all())

    async def get_attendance_player_ids(self, match_id: UUID) -> list[UUID]:
        result = await self.session.execute(
            select(Attendance.player_id).where(Attendance.match_id == match_id)
        )
        return list(result.scalars().all())

    async def get_discover_matches(
        self,
        player_id: UUID | None,
        date_from: date_type | None = None,
        date_to: date_type | None = None,
        court_types: list[str] | None = None,
        weekdays: list[int] | None = None,
        limit: int = 20,
        offset: int = 0,
    ) -> list[dict]:
        """Partidas abertas de grupos públicos. Se player_id fornecido, exclui grupos
        onde o jogador já é membro e partidas em que já está na lista de espera."""
        from app.models.group import Group
        from app.models.waitlist import MatchWaitlist

        BRAZIL = timezone(timedelta(hours=-3))
        today = datetime.now(BRAZIL).date()

        confirmed_sub = (
            select(Attendance.match_id, func.count().label("cnt"))
            .where(Attendance.status == AttendanceStatus.CONFIRMED)
            .group_by(Attendance.match_id)
            .subquery()
        )

        base_conditions = [
            Group.is_public.is_(True),
            Match.status == MatchStatus.OPEN,
            Match.match_date >= today,
            or_(
                Match.max_players.is_(None),
                func.coalesce(confirmed_sub.c.cnt, 0) < Match.max_players,
            ),
        ]

        if player_id is not None:
            member_group_ids = select(GroupMember.group_id).where(GroupMember.player_id == player_id)
            waitlisted_match_ids = select(MatchWaitlist.match_id).where(MatchWaitlist.player_id == player_id)
            base_conditions.append(Match.group_id.not_in(member_group_ids))
            base_conditions.append(Match.id.not_in(waitlisted_match_ids))

        stmt = (
            select(
                Match,
                Group.name.label("group_name"),
                Group.timezone.label("group_timezone"),
                func.coalesce(confirmed_sub.c.cnt, 0).label("confirmed_count"),
            )
            .join(Group, Group.id == Match.group_id)
            .outerjoin(confirmed_sub, confirmed_sub.c.match_id == Match.id)
            .where(*base_conditions)
        )

        if date_from:
            stmt = stmt.where(Match.match_date >= date_from)
        if date_to:
            stmt = stmt.where(Match.match_date <= date_to)
        if court_types:
            stmt = stmt.where(Match.court_type.in_(court_types))
        if weekdays:
            stmt = stmt.where(func.extract("dow", Match.match_date).in_(weekdays))

        stmt = stmt.order_by(Match.match_date.asc(), Match.start_time.asc()).limit(limit).offset(offset)

        result = await self.session.execute(stmt)
        return [
            {"match": row.Match, "group_name": row.group_name, "group_timezone": row.group_timezone, "confirmed_count": row.confirmed_count}
            for row in result.all()
        ]

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
