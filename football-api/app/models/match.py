import uuid
from datetime import date, time
from enum import Enum as PyEnum

from sqlalchemy import Date, ForeignKey, Integer, SmallInteger, String, Text, Time, UniqueConstraint
from sqlalchemy.dialects.postgresql import ENUM as PgEnum, UUID
from sqlalchemy import text
from sqlalchemy.orm import Mapped, mapped_column, relationship

from app.models.base import TimestampMixin, UUIDMixin
from app.db.session import Base


class MatchStatus(str, PyEnum):
    OPEN = "open"
    CLOSED = "closed"


class CourtType(str, PyEnum):
    CAMPO    = "campo"
    SINTETICO = "sintetico"
    TERRAO   = "terrao"
    QUADRA   = "quadra"


class AttendanceStatus(str, PyEnum):
    PENDING = "pending"
    CONFIRMED = "confirmed"
    DECLINED = "declined"


_match_status_col = PgEnum(
    MatchStatus,
    name="match_status",
    create_type=False,
    values_callable=lambda x: [e.value for e in x],
)

_court_type_col = PgEnum(
    CourtType,
    name="court_type",
    create_type=False,
    values_callable=lambda x: [e.value for e in x],
)

_attendance_status_col = PgEnum(
    AttendanceStatus,
    name="attendance_status",
    create_type=False,
    values_callable=lambda x: [e.value for e in x],
)


class Match(Base, UUIDMixin, TimestampMixin):
    __tablename__ = "matches"

    number: Mapped[int] = mapped_column(
        Integer, nullable=False, server_default=text("nextval('matches_number_seq')")
    )
    group_id: Mapped[uuid.UUID] = mapped_column(
        UUID(as_uuid=True), ForeignKey("groups.id", ondelete="CASCADE"), nullable=False
    )
    match_date: Mapped[date] = mapped_column(Date, nullable=False)
    start_time: Mapped[time] = mapped_column(Time, nullable=False)
    location: Mapped[str] = mapped_column(String(200), nullable=False)
    address: Mapped[str | None] = mapped_column(String(300), nullable=True)
    court_type: Mapped[CourtType | None] = mapped_column(_court_type_col, nullable=True)
    players_per_team: Mapped[int | None] = mapped_column(SmallInteger, nullable=True)
    max_players: Mapped[int | None] = mapped_column(SmallInteger, nullable=True)
    notes: Mapped[str | None] = mapped_column(Text, nullable=True)
    hash: Mapped[str] = mapped_column(String(12), nullable=False, unique=True, index=True)
    status: Mapped[MatchStatus] = mapped_column(
        _match_status_col, nullable=False, default=MatchStatus.OPEN
    )
    created_by_id: Mapped[uuid.UUID | None] = mapped_column(
        UUID(as_uuid=True), ForeignKey("players.id", ondelete="SET NULL"), nullable=True
    )

    # Relationships
    group = relationship("Group", back_populates="matches")
    created_by = relationship("Player", foreign_keys=[created_by_id])
    attendances = relationship("Attendance", back_populates="match", cascade="all, delete-orphan")


class Attendance(Base, UUIDMixin, TimestampMixin):
    __tablename__ = "attendances"

    __table_args__ = (UniqueConstraint("match_id", "player_id", name="uq_match_player"),)

    match_id: Mapped[uuid.UUID] = mapped_column(
        UUID(as_uuid=True), ForeignKey("matches.id", ondelete="CASCADE"), nullable=False
    )
    player_id: Mapped[uuid.UUID] = mapped_column(
        UUID(as_uuid=True), ForeignKey("players.id", ondelete="CASCADE"), nullable=False
    )
    status: Mapped[AttendanceStatus] = mapped_column(
        _attendance_status_col, nullable=False, default=AttendanceStatus.PENDING
    )

    # Relationships
    match = relationship("Match", back_populates="attendances")
    player = relationship("Player", back_populates="attendances")
