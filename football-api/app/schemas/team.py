import uuid

from pydantic import BaseModel


class TeamPlayerItem(BaseModel):
    player_id: uuid.UUID
    name: str
    nickname: str | None
    skill_stars: int
    is_goalkeeper: bool


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
