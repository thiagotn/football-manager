import uuid
from datetime import datetime

from pydantic import BaseModel, field_validator


class FinanceSummary(BaseModel):
    received_cents: int
    pending_count: int
    paid_count: int
    total_members: int
    compliance_pct: int


class FinancePaymentResponse(BaseModel):
    model_config = {"from_attributes": True}

    id: uuid.UUID
    player_id: uuid.UUID
    player_name: str
    payment_type: str | None
    amount_due: int | None
    status: str
    paid_at: datetime | None


class FinancePeriodResponse(BaseModel):
    period_id: uuid.UUID
    year: int
    month: int
    summary: FinanceSummary
    payments: list[FinancePaymentResponse]


class PeriodListItem(BaseModel):
    model_config = {"from_attributes": True}

    id: uuid.UUID
    year: int
    month: int


class MarkPaymentRequest(BaseModel):
    status: str  # "paid" | "pending"
    payment_type: str | None = None  # required when status="paid"

    @field_validator("status")
    @classmethod
    def validate_status(cls, v: str) -> str:
        if v not in ("paid", "pending"):
            raise ValueError("status deve ser 'paid' ou 'pending'")
        return v

    @field_validator("payment_type")
    @classmethod
    def validate_payment_type(cls, v: str | None) -> str | None:
        if v is not None and v not in ("monthly", "per_match"):
            raise ValueError("payment_type deve ser 'monthly' ou 'per_match'")
        return v
