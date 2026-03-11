from uuid import UUID

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy.orm import selectinload

from app.db.repositories.base import BaseRepository
from app.models.group import Group, GroupMember, GroupMemberRole
from app.models.match import Attendance, AttendanceStatus
from app.models.player import Player, PlayerRole


class GroupRepository(BaseRepository[Group]):
    model = Group

    def __init__(self, session: AsyncSession):
        super().__init__(session)

    async def get_by_slug(self, slug: str) -> Group | None:
        result = await self.session.execute(
            select(Group).where(Group.slug == slug)
        )
        return result.scalar_one_or_none()

    async def get_with_members(self, group_id: UUID) -> Group | None:
        result = await self.session.execute(
            select(Group)
            .options(selectinload(Group.members).selectinload(GroupMember.player))
            .where(Group.id == group_id)
        )
        return result.scalar_one_or_none()

    async def get_member(self, group_id: UUID, player_id: UUID) -> GroupMember | None:
        result = await self.session.execute(
            select(GroupMember).where(
                GroupMember.group_id == group_id,
                GroupMember.player_id == player_id,
            )
        )
        return result.scalar_one_or_none()

    async def add_member(self, group_id: UUID, player_id: UUID, role: GroupMemberRole) -> GroupMember:
        member = GroupMember(group_id=group_id, player_id=player_id, role=role)
        self.session.add(member)
        await self.session.flush()
        await self.session.refresh(member)
        return member

    async def get_member_ids(self, group_id: UUID) -> list[UUID]:
        result = await self.session.execute(
            select(GroupMember.player_id).where(GroupMember.group_id == group_id)
        )
        return list(result.scalars().all())

    async def get_non_admin_member_ids(self, group_id: UUID) -> list[UUID]:
        """Retorna IDs dos membros do grupo excluindo jogadores com role admin."""
        result = await self.session.execute(
            select(GroupMember.player_id)
            .join(Player, Player.id == GroupMember.player_id)
            .where(
                GroupMember.group_id == group_id,
                Player.role != PlayerRole.ADMIN,
            )
        )
        return list(result.scalars().all())

    async def get_groups_with_recurrence(self) -> list[Group]:
        result = await self.session.execute(
            select(Group).where(Group.recurrence_enabled == True)  # noqa: E712
        )
        return list(result.scalars().all())

    async def get_confirmed_players_with_skills(self, match_id: UUID, group_id: UUID) -> list[dict]:
        """Retorna confirmados da partida com skill_stars e is_goalkeeper de group_members."""
        result = await self.session.execute(
            select(
                Attendance.player_id,
                Player.name,
                Player.nickname,
                GroupMember.skill_stars,
                GroupMember.is_goalkeeper,
            )
            .join(Player, Player.id == Attendance.player_id)
            .join(
                GroupMember,
                (GroupMember.player_id == Attendance.player_id)
                & (GroupMember.group_id == group_id),
            )
            .where(
                Attendance.match_id == match_id,
                Attendance.status == AttendanceStatus.CONFIRMED,
                Player.role != PlayerRole.ADMIN,
            )
        )
        return [
            {
                "player_id": row.player_id,
                "name": row.name,
                "nickname": row.nickname,
                "skill_stars": row.skill_stars,
                "is_goalkeeper": row.is_goalkeeper,
            }
            for row in result.all()
        ]

    async def get_member_skills(self, group_id: UUID, player_ids: list[UUID]) -> dict[UUID, dict]:
        """Retorna skill_stars e is_goalkeeper de membros específicos do grupo."""
        if not player_ids:
            return {}
        result = await self.session.execute(
            select(GroupMember.player_id, GroupMember.skill_stars, GroupMember.is_goalkeeper)
            .where(
                GroupMember.group_id == group_id,
                GroupMember.player_id.in_(player_ids),
            )
        )
        return {
            row.player_id: {"skill_stars": row.skill_stars, "is_goalkeeper": row.is_goalkeeper}
            for row in result.all()
        }

    async def get_player_groups(self, player_id: UUID) -> list[Group]:
        result = await self.session.execute(
            select(Group)
            .join(GroupMember, GroupMember.group_id == Group.id)
            .where(GroupMember.player_id == player_id)
            .order_by(Group.name)
        )
        return list(result.scalars().all())
