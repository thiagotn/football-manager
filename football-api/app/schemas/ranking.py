from __future__ import annotations
import uuid
from pydantic import BaseModel
from typing import Literal


class RankingItem(BaseModel):
    position: int
    player_id: uuid.UUID
    name: str
    nickname: str | None
    avatar_url: str | None


class TopRankingItem(RankingItem):
    total_points: int


class FlopRankingItem(RankingItem):
    total_flop_votes: int


class RankingResponse(BaseModel):
    year: int | None
    month: int | None
    type: Literal["top", "flop"]
    items: list[TopRankingItem] | list[FlopRankingItem]
