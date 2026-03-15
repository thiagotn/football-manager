from datetime import datetime, timezone
from uuid import UUID

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy.orm import selectinload

from app.models.finance import FinancePeriod, FinancePayment


class FinanceRepository:
    def __init__(self, session: AsyncSession):
        self.session = session

    async def get_period(self, group_id: UUID, year: int, month: int) -> FinancePeriod | None:
        result = await self.session.execute(
            select(FinancePeriod).where(
                FinancePeriod.group_id == group_id,
                FinancePeriod.year == year,
                FinancePeriod.month == month,
            )
        )
        return result.scalar_one_or_none()

    async def get_or_create_period(
        self, group_id: UUID, year: int, month: int
    ) -> FinancePeriod:
        """Get or create period and populate with current group members."""
        period = await self.get_period(group_id, year, month)
        if period is not None:
            return period

        period = FinancePeriod(group_id=group_id, year=year, month=month)
        self.session.add(period)
        await self.session.flush()
        await self.session.refresh(period)
        await self._populate_members(period)
        return period

    async def _populate_members(self, period: FinancePeriod) -> None:
        from app.models.group import GroupMember
        from app.models.player import Player

        result = await self.session.execute(
            select(GroupMember, Player)
            .join(Player, GroupMember.player_id == Player.id)
            .where(GroupMember.group_id == period.group_id)
        )
        for member, player in result.all():
            payment = FinancePayment(
                period_id=period.id,
                player_id=player.id,
                player_name=player.nickname or player.name,
                status="pending",
            )
            self.session.add(payment)
        await self.session.flush()

    async def get_period_with_payments(
        self, group_id: UUID, year: int, month: int
    ) -> FinancePeriod | None:
        result = await self.session.execute(
            select(FinancePeriod)
            .options(selectinload(FinancePeriod.payments))
            .where(
                FinancePeriod.group_id == group_id,
                FinancePeriod.year == year,
                FinancePeriod.month == month,
            )
        )
        return result.scalar_one_or_none()

    async def list_periods(self, group_id: UUID) -> list[FinancePeriod]:
        result = await self.session.execute(
            select(FinancePeriod)
            .where(FinancePeriod.group_id == group_id)
            .order_by(FinancePeriod.year.desc(), FinancePeriod.month.desc())
        )
        return list(result.scalars().all())

    async def get_payment(self, payment_id: UUID) -> FinancePayment | None:
        return await self.session.get(FinancePayment, payment_id)

    async def mark_paid(
        self, payment: FinancePayment, payment_type: str, amount_due: int
    ) -> FinancePayment:
        payment.status = "paid"
        payment.payment_type = payment_type
        payment.amount_due = amount_due
        payment.paid_at = datetime.now(timezone.utc)
        await self.session.flush()
        return payment

    async def mark_pending(self, payment: FinancePayment) -> FinancePayment:
        payment.status = "pending"
        payment.payment_type = None
        payment.amount_due = None
        payment.paid_at = None
        await self.session.flush()
        return payment
