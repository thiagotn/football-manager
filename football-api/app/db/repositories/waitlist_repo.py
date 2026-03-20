from datetime import datetime, timezone
from uuid import UUID

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy.orm import selectinload

from app.models.waitlist import MatchWaitlist, WaitlistStatus


class WaitlistRepository:
    def __init__(self, session: AsyncSession):
        self.session = session

    async def get_entry(self, match_id: UUID, player_id: UUID) -> MatchWaitlist | None:
        result = await self.session.execute(
            select(MatchWaitlist).where(
                MatchWaitlist.match_id == match_id,
                MatchWaitlist.player_id == player_id,
            )
        )
        return result.scalar_one_or_none()

    async def get_by_id(self, entry_id: UUID) -> MatchWaitlist | None:
        from app.models.match import Match
        from app.models.group import Group
        result = await self.session.execute(
            select(MatchWaitlist)
            .options(
                selectinload(MatchWaitlist.player),
                selectinload(MatchWaitlist.match).selectinload(Match.group),
            )
            .where(MatchWaitlist.id == entry_id)
        )
        return result.scalar_one_or_none()

    async def get_pending_for_match(self, match_id: UUID) -> list[MatchWaitlist]:
        result = await self.session.execute(
            select(MatchWaitlist)
            .options(selectinload(MatchWaitlist.player))
            .where(
                MatchWaitlist.match_id == match_id,
                MatchWaitlist.status == WaitlistStatus.PENDING,
            )
            .order_by(MatchWaitlist.created_at)
        )
        return list(result.scalars().all())

    async def get_all_for_match(self, match_id: UUID) -> list[MatchWaitlist]:
        result = await self.session.execute(
            select(MatchWaitlist)
            .options(selectinload(MatchWaitlist.player))
            .where(MatchWaitlist.match_id == match_id)
            .order_by(MatchWaitlist.created_at)
        )
        return list(result.scalars().all())

    async def create(
        self, match_id: UUID, player_id: UUID, intro: str | None
    ) -> MatchWaitlist:
        now = datetime.now(timezone.utc)
        entry = MatchWaitlist(
            match_id=match_id,
            player_id=player_id,
            intro=intro,
            agreed_at=now,
            status=WaitlistStatus.PENDING,
            created_at=now,
        )
        self.session.add(entry)
        await self.session.flush()
        await self.session.refresh(entry, ["player"])
        return entry

    async def accept(self, entry: MatchWaitlist, reviewed_by: UUID) -> MatchWaitlist:
        entry.status = WaitlistStatus.ACCEPTED
        entry.reviewed_by = reviewed_by
        entry.reviewed_at = datetime.now(timezone.utc)
        await self.session.flush()
        return entry

    async def reject(self, entry: MatchWaitlist, reviewed_by: UUID) -> MatchWaitlist:
        entry.status = WaitlistStatus.REJECTED
        entry.reviewed_by = reviewed_by
        entry.reviewed_at = datetime.now(timezone.utc)
        await self.session.flush()
        return entry
