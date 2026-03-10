import uuid

from pydantic import BaseModel


class PlayerStatItem(BaseModel):
    player_id: uuid.UUID
    display_name: str
    vote_points: int
    flop_votes: int
    minutes_played: int


class GroupStatsResponse(BaseModel):
    players: list[PlayerStatItem]
    period_label: str
