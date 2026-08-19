import uuid
from typing import Literal

from pydantic import BaseModel

DrawStrategy = Literal["balanced", "simple"]


class GenerateTeamsRequest(BaseModel):
    strategy: DrawStrategy = "balanced"


class TeamPlayerItem(BaseModel):
    player_id: uuid.UUID
    name: str
    nickname: str | None
    skill_stars: int
    position: str


class TeamItem(BaseModel):
    id: uuid.UUID
    name: str
    color: str | None
    position: int
    skill_total: int
    players: list[TeamPlayerItem]


class TeamsResponse(BaseModel):
    teams: list[TeamItem]
    reserves: list[TeamPlayerItem]
