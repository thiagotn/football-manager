import uuid
from datetime import datetime
from enum import Enum as PyEnum

from sqlalchemy import Boolean, ForeignKey, Integer, String, UniqueConstraint
from sqlalchemy.dialects.postgresql import ENUM as PgEnum, TIMESTAMP, UUID
from sqlalchemy.orm import Mapped, mapped_column, relationship

from app.models.base import TimestampMixin, UUIDMixin
from app.db.session import Base


class PlayerRole(str, PyEnum):
    ADMIN = "admin"
    PLAYER = "player"


_player_role_col = PgEnum(
    PlayerRole,
    name="player_role",
    create_type=False,
    values_callable=lambda x: [e.value for e in x],
)


class Player(Base, UUIDMixin, TimestampMixin):
    __tablename__ = "players"

    name: Mapped[str] = mapped_column(String(100), nullable=False)
    nickname: Mapped[str | None] = mapped_column(String(50), nullable=True)
    whatsapp: Mapped[str] = mapped_column(String(20), nullable=False, unique=True, index=True)
    password_hash: Mapped[str] = mapped_column(String(255), nullable=False)
    role: Mapped[PlayerRole] = mapped_column(
        _player_role_col, nullable=False, default=PlayerRole.PLAYER
    )
    active: Mapped[bool] = mapped_column(Boolean, nullable=False, default=True)
    must_change_password: Mapped[bool] = mapped_column(Boolean, nullable=False, default=False)
    avatar_url: Mapped[str | None] = mapped_column(String(500), nullable=True)
    chat_enabled: Mapped[bool] = mapped_column(Boolean, nullable=False, default=False)
    chat_req_count: Mapped[int] = mapped_column(Integer, nullable=False, default=0)
    chat_req_window: Mapped[datetime | None] = mapped_column(TIMESTAMP(timezone=True), nullable=True)

    # Relationships
    group_members = relationship("GroupMember", back_populates="player", cascade="all, delete-orphan")
    attendances = relationship("Attendance", back_populates="player", cascade="all, delete-orphan")
    created_invites = relationship("InviteToken", foreign_keys="InviteToken.created_by_id", back_populates="created_by")
    used_invites = relationship("InviteToken", foreign_keys="InviteToken.used_by_id", back_populates="used_by")
