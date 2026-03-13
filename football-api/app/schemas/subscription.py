from datetime import datetime

from pydantic import BaseModel


class SubscriptionMeResponse(BaseModel):
    plan: str
    groups_limit: int | None    # None = ilimitado
    groups_used: int
    members_limit: int | None   # None = ilimitado
    status: str | None = None
    gateway_customer_id: str | None = None
    gateway_sub_id: str | None = None
    current_period_end: datetime | None = None
    grace_period_end: datetime | None = None


class CheckoutSessionRequest(BaseModel):
    plan: str           # "basic" | "pro"
    billing_cycle: str  # "monthly" | "yearly"


class CheckoutSessionResponse(BaseModel):
    checkout_url: str
