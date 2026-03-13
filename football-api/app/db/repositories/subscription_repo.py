from datetime import datetime
from uuid import UUID

from sqlalchemy import func, select
from sqlalchemy.ext.asyncio import AsyncSession

from app.db.repositories.base import BaseRepository
from app.models.group import GroupMember, GroupMemberRole
from app.models.subscription import PlayerSubscription


class SubscriptionRepository(BaseRepository[PlayerSubscription]):
    model = PlayerSubscription

    def __init__(self, session: AsyncSession):
        super().__init__(session)

    async def get_by_player(self, player_id: UUID) -> PlayerSubscription | None:
        result = await self.session.execute(
            select(PlayerSubscription).where(PlayerSubscription.player_id == player_id)
        )
        return result.scalar_one_or_none()

    async def get_or_create(self, player_id: UUID) -> PlayerSubscription:
        sub = await self.get_by_player(player_id)
        if sub:
            return sub
        sub = PlayerSubscription(player_id=player_id, plan="free")
        self.session.add(sub)
        await self.session.flush()
        await self.session.refresh(sub)
        return sub

    async def get_by_gateway_customer(self, gateway_customer_id: str) -> PlayerSubscription | None:
        result = await self.session.execute(
            select(PlayerSubscription).where(
                PlayerSubscription.gateway_customer_id == gateway_customer_id
            )
        )
        return result.scalar_one_or_none()

    async def update_plan(
        self,
        player_id: UUID,
        plan: str,
        status: str = "active",
        gateway_customer_id: str | None = None,
        gateway_sub_id: str | None = None,
        current_period_end: datetime | None = None,
        grace_period_end: datetime | None = None,
    ) -> PlayerSubscription:
        sub = await self.get_or_create(player_id)
        sub.plan = plan
        sub.status = status
        if gateway_customer_id is not None:
            sub.gateway_customer_id = gateway_customer_id
        if gateway_sub_id is not None:
            sub.gateway_sub_id = gateway_sub_id
        if current_period_end is not None:
            sub.current_period_end = current_period_end
        if grace_period_end is not None:
            sub.grace_period_end = grace_period_end
        await self.session.flush()
        await self.session.refresh(sub)
        return sub

    async def count_admin_groups(self, player_id: UUID) -> int:
        """Conta grupos onde este player é admin do grupo (GroupMemberRole.ADMIN)."""
        result = await self.session.execute(
            select(func.count())
            .select_from(GroupMember)
            .where(
                GroupMember.player_id == player_id,
                GroupMember.role == GroupMemberRole.ADMIN,
            )
        )
        return result.scalar_one()
