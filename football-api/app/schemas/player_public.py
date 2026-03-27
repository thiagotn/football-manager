from __future__ import annotations
import uuid
from pydantic import BaseModel


class PlayerPublicStats(BaseModel):
    player_id: uuid.UUID
    name: str
    nickname: str | None
    avatar_url: str | None
    # Best skill_stars across all groups (max)
    skill_stars: int
    total_matches_confirmed: int
    attendance_rate: int  # 0-100
    current_streak: int
    best_streak: int
    top1_count: int
    top5_count: int
    total_vote_points: int
    total_flop_votes: int
