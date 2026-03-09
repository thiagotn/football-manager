import uuid

from sqlalchemy import ForeignKey, String
from sqlalchemy.dialects.postgresql import UUID
from sqlalchemy.orm import Mapped, mapped_column

from app.db.session import Base
from app.models.base import TimestampMixin, UUIDMixin


class PlayerSubscription(Base, UUIDMixin, TimestampMixin):
    __tablename__ = "player_subscriptions"

    player_id: Mapped[uuid.UUID] = mapped_column(
        UUID(as_uuid=True),
        ForeignKey("players.id", ondelete="CASCADE"),
        nullable=False,
        unique=True,
    )
    plan: Mapped[str] = mapped_column(String(20), nullable=False, default="free")
