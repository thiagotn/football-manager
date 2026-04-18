from __future__ import annotations

from pydantic import BaseModel


class MonthlyStatItem(BaseModel):
    month: str  # "YYYY-MM"
    matches_confirmed: int
    minutes_played: int


class RecentMatchItem(BaseModel):
    match_date: str
    group_name: str
    status: str  # "confirmed" | "declined"


class GroupStatItem(BaseModel):
    group_id: str
    group_name: str
    skill_stars: int
    position: str
    role: str  # "admin" | "member"
    matches_confirmed: int


class PlayerFullStats(BaseModel):
    total_matches_confirmed: int
    total_minutes_played: int
    total_vote_points: int
    total_flop_votes: int
    top1_count: int
    top5_count: int
    total_goals: int
    total_assists: int
    current_streak: int
    best_streak: int
    attendance_rate: int  # 0–100
    monthly_stats: list[MonthlyStatItem]
    recent_matches: list[RecentMatchItem]
    groups: list[GroupStatItem]
