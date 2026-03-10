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
