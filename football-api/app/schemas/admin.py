from datetime import date, datetime, time
from uuid import UUID

from pydantic import BaseModel


class AdminStatsResponse(BaseModel):
    total_matches: int
    total_groups: int
    total_players: int
    platform_minutes_played: int
    signups_total: int
    signups_last_7_days: int
    signups_last_30_days: int
    total_reviews: int


class AdminMatchItem(BaseModel):
    id: UUID
    hash: str
    number: int
    group_id: UUID
    group_name: str
    match_date: date
    start_time: time
    end_time: time | None
    location: str
    status: str


class AdminMatchListResponse(BaseModel):
    total: int
    items: list[AdminMatchItem]


class AdminGroupItem(BaseModel):
    id: UUID
    name: str
    description: str | None
    slug: str
    total_members: int
    total_matches: int
    created_at: datetime


class AdminGroupListResponse(BaseModel):
    total: int
    items: list[AdminGroupItem]


# ── Subscriptions ─────────────────────────────────────────────

class AdminSubscriptionBreakdownItem(BaseModel):
    plan: str
    billing_cycle: str
    count: int


class AdminSubscriptionSummary(BaseModel):
    total_players: int
    active: int
    free: int
    past_due: int
    canceled: int
    mrr_cents: int
    breakdown: list[AdminSubscriptionBreakdownItem]


class AdminSubscriptionItem(BaseModel):
    player_id: UUID
    player_name: str
    plan: str
    billing_cycle: str
    status: str
    current_period_end: datetime | None
    grace_period_end: datetime | None
    gateway_customer_id: str | None
    gateway_sub_id: str | None
    created_at: datetime


class AdminSubscriptionListResponse(BaseModel):
    total: int
    page: int
    page_size: int
    items: list[AdminSubscriptionItem]


class AdminSubscriptionUpdateRequest(BaseModel):
    plan: str
    status: str = "active"
    billing_cycle: str = "monthly"
    reason: str = "manual_admin_override"


# ── Players admin ──────────────────────────────────────────────

class AdminPlayerItem(BaseModel):
    id: UUID
    name: str
    nickname: str | None
    whatsapp: str
    role: str
    active: bool
    created_at: datetime
    plan: str
    total_groups: int
    avatar_url: str | None = None


class AdminPlayerListResponse(BaseModel):
    total: int
    page: int
    page_size: int
    items: list[AdminPlayerItem]
