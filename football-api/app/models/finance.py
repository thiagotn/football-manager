import uuid
from datetime import datetime

from sqlalchemy import DateTime, ForeignKey, Integer, SmallInteger, String, func
from sqlalchemy.dialects.postgresql import UUID
from sqlalchemy.orm import Mapped, mapped_column, relationship
from sqlalchemy import TIMESTAMP

from app.db.session import Base
from app.models.base import TimestampMixin, UUIDMixin


class FinancePeriod(Base, UUIDMixin):
    __tablename__ = "finance_periods"

    group_id: Mapped[uuid.UUID] = mapped_column(
        UUID(as_uuid=True), ForeignKey("groups.id", ondelete="CASCADE"), nullable=False
    )
    year: Mapped[int] = mapped_column(SmallInteger, nullable=False)
    month: Mapped[int] = mapped_column(SmallInteger, nullable=False)
    created_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), server_default=func.now(), nullable=False
    )

    payments: Mapped[list["FinancePayment"]] = relationship(
        "FinancePayment", back_populates="period", cascade="all, delete-orphan"
    )


class FinancePayment(Base, UUIDMixin, TimestampMixin):
    __tablename__ = "finance_payments"

    period_id: Mapped[uuid.UUID] = mapped_column(
        UUID(as_uuid=True), ForeignKey("finance_periods.id", ondelete="CASCADE"), nullable=False
    )
    player_id: Mapped[uuid.UUID] = mapped_column(
        UUID(as_uuid=True), ForeignKey("players.id"), nullable=False
    )
    player_name: Mapped[str] = mapped_column(String(100), nullable=False)
    payment_type: Mapped[str | None] = mapped_column(String(20), nullable=True)
    amount_due: Mapped[int | None] = mapped_column(Integer, nullable=True)
    status: Mapped[str] = mapped_column(String(20), nullable=False, default="pending")
    paid_at: Mapped[datetime | None] = mapped_column(TIMESTAMP(timezone=True), nullable=True)

    period: Mapped["FinancePeriod"] = relationship("FinancePeriod", back_populates="payments")
