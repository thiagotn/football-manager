import uuid
from datetime import datetime
from enum import Enum as PyEnum

from sqlalchemy import ForeignKey, Text, UniqueConstraint
from sqlalchemy.dialects.postgresql import ENUM as PgEnum, UUID
from sqlalchemy import TIMESTAMP
from sqlalchemy.orm import Mapped, mapped_column, relationship

from app.models.base import UUIDMixin
from app.db.session import Base


class WaitlistStatus(str, PyEnum):
    PENDING = "pending"
    ACCEPTED = "accepted"
    REJECTED = "rejected"


_waitlist_status_col = PgEnum(
    WaitlistStatus,
    name="waitlist_status",
    create_type=False,
    values_callable=lambda x: [e.value for e in x],
)


class MatchWaitlist(Base, UUIDMixin):
    __tablename__ = "match_waitlist"
    __table_args__ = (UniqueConstraint("match_id", "player_id", name="uq_waitlist_match_player"),)

    match_id: Mapped[uuid.UUID] = mapped_column(
        UUID(as_uuid=True), ForeignKey("matches.id", ondelete="CASCADE"), nullable=False
    )
    player_id: Mapped[uuid.UUID] = mapped_column(
        UUID(as_uuid=True), ForeignKey("players.id", ondelete="CASCADE"), nullable=False
    )
    intro: Mapped[str | None] = mapped_column(Text, nullable=True)
    agreed_at: Mapped[datetime] = mapped_column(TIMESTAMP(timezone=True), nullable=False)
    status: Mapped[WaitlistStatus] = mapped_column(
        _waitlist_status_col, nullable=False, default=WaitlistStatus.PENDING
    )
    reviewed_by: Mapped[uuid.UUID | None] = mapped_column(
        UUID(as_uuid=True), ForeignKey("players.id"), nullable=True
    )
    reviewed_at: Mapped[datetime | None] = mapped_column(TIMESTAMP(timezone=True), nullable=True)
    created_at: Mapped[datetime] = mapped_column(TIMESTAMP(timezone=True), nullable=False)

    # Relationships
    match = relationship("Match", foreign_keys=[match_id])
    player = relationship("Player", foreign_keys=[player_id])
    reviewer = relationship("Player", foreign_keys=[reviewed_by])
