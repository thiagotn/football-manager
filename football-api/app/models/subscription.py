import uuid
from datetime import datetime

from sqlalchemy import ForeignKey, String
from sqlalchemy.dialects.postgresql import TIMESTAMP, UUID
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
    status: Mapped[str] = mapped_column(String(20), nullable=False, default="active")

    # Stripe IDs — preenchidos após checkout.session.completed
    gateway_customer_id: Mapped[str | None] = mapped_column(String(255), nullable=True)
    gateway_sub_id: Mapped[str | None] = mapped_column(String(255), nullable=True)

    # Período da assinatura — gerenciado pelo webhook handler
    current_period_end: Mapped[datetime | None] = mapped_column(
        TIMESTAMP(timezone=True), nullable=True
    )
    grace_period_end: Mapped[datetime | None] = mapped_column(
        TIMESTAMP(timezone=True), nullable=True
    )
