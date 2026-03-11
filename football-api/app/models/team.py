import uuid

from sqlalchemy import Boolean, ForeignKey, SmallInteger, String
from sqlalchemy.dialects.postgresql import UUID
from sqlalchemy.orm import Mapped, mapped_column, relationship

from app.models.base import TimestampMixin, UUIDMixin
from app.db.session import Base


class MatchTeam(Base, UUIDMixin, TimestampMixin):
    __tablename__ = "match_teams"

    match_id: Mapped[uuid.UUID] = mapped_column(
        UUID(as_uuid=True), ForeignKey("matches.id", ondelete="CASCADE"), nullable=False
    )
    name: Mapped[str] = mapped_column(String(100), nullable=False)
    color: Mapped[str | None] = mapped_column(String(7), nullable=True)
    position: Mapped[int] = mapped_column(SmallInteger, nullable=False)

    players = relationship("MatchTeamPlayer", back_populates="team", cascade="all, delete-orphan")


class MatchTeamPlayer(Base, UUIDMixin):
    __tablename__ = "match_team_players"

    team_id: Mapped[uuid.UUID] = mapped_column(
        UUID(as_uuid=True), ForeignKey("match_teams.id", ondelete="CASCADE"), nullable=False
    )
    player_id: Mapped[uuid.UUID] = mapped_column(
        UUID(as_uuid=True), ForeignKey("players.id", ondelete="CASCADE"), nullable=False
    )
    is_reserve: Mapped[bool] = mapped_column(Boolean, nullable=False, default=False)

    team = relationship("MatchTeam", back_populates="players")
    player = relationship("Player")
