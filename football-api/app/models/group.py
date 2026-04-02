import uuid
from decimal import Decimal
from enum import Enum as PyEnum

from sqlalchemy import Boolean, ForeignKey, Integer, Numeric, SmallInteger, String, Text, UniqueConstraint
from sqlalchemy.dialects.postgresql import ENUM as PgEnum, UUID
from sqlalchemy.orm import Mapped, mapped_column, relationship

from app.models.base import TimestampMixin, UUIDMixin
from app.db.session import Base


class GroupMemberRole(str, PyEnum):
    ADMIN = "admin"
    MEMBER = "member"


_group_member_role_col = PgEnum(
    GroupMemberRole,
    name="group_member_role",
    create_type=False,
    values_callable=lambda x: [e.value for e in x],
)


class Group(Base, UUIDMixin, TimestampMixin):
    __tablename__ = "groups"

    name: Mapped[str] = mapped_column(String(100), nullable=False)
    description: Mapped[str | None] = mapped_column(Text, nullable=True)
    slug: Mapped[str] = mapped_column(String(60), nullable=False, unique=True, index=True)
    per_match_amount: Mapped[Decimal | None] = mapped_column(Numeric(10, 2), nullable=True)
    monthly_amount: Mapped[Decimal | None] = mapped_column(Numeric(10, 2), nullable=True)
    recurrence_enabled: Mapped[bool] = mapped_column(Boolean, nullable=False, default=False)
    is_public: Mapped[bool] = mapped_column(Boolean, nullable=False, default=True, server_default="true")
    vote_open_delay_minutes: Mapped[int] = mapped_column(Integer, nullable=False, default=20)
    vote_duration_hours: Mapped[int] = mapped_column(Integer, nullable=False, default=24)
    timezone: Mapped[str] = mapped_column(String(60), nullable=False, default="America/Sao_Paulo", server_default="America/Sao_Paulo")

    # Relationships
    members = relationship("GroupMember", back_populates="group", cascade="all, delete-orphan")
    matches = relationship("Match", back_populates="group", cascade="all, delete-orphan")
    invite_tokens = relationship("InviteToken", back_populates="group", cascade="all, delete-orphan")


class GroupMember(Base, UUIDMixin, TimestampMixin):
    __tablename__ = "group_members"
    __table_args__ = (UniqueConstraint("group_id", "player_id", name="uq_group_player"),)

    group_id: Mapped[uuid.UUID] = mapped_column(
        UUID(as_uuid=True), ForeignKey("groups.id", ondelete="CASCADE"), nullable=False
    )
    player_id: Mapped[uuid.UUID] = mapped_column(
        UUID(as_uuid=True), ForeignKey("players.id", ondelete="CASCADE"), nullable=False
    )
    role: Mapped[GroupMemberRole] = mapped_column(
        _group_member_role_col, nullable=False, default=GroupMemberRole.MEMBER
    )
    skill_stars: Mapped[int] = mapped_column(SmallInteger, nullable=False, default=2)
    position: Mapped[str] = mapped_column(String(3), nullable=False, default="mei")

    # Relationships
    group = relationship("Group", back_populates="members")
    player = relationship("Player", back_populates="group_members")
