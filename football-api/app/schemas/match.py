import uuid
from datetime import date, datetime, time

from pydantic import BaseModel, Field

from app.models.match import AttendanceStatus, CourtType, MatchStatus
from app.schemas.player import PlayerPublic


class MatchCreate(BaseModel):
    match_date: date
    start_time: time
    location: str = Field(..., min_length=2, max_length=200)
    address: str | None = Field(None, max_length=300)
    court_type: CourtType | None = None
    players_per_team: int | None = Field(None, ge=2, le=15)
    max_players: int | None = Field(None, ge=2)
    notes: str | None = Field(None, max_length=500)


class MatchUpdate(BaseModel):
    match_date: date | None = None
    start_time: time | None = None
    location: str | None = Field(None, min_length=2, max_length=200)
    address: str | None = Field(None, max_length=300)
    court_type: CourtType | None = None
    players_per_team: int | None = Field(None, ge=2, le=15)
    max_players: int | None = Field(None, ge=2)
    notes: str | None = None
    status: MatchStatus | None = None


class AttendanceResponse(BaseModel):
    model_config = {"from_attributes": True}

    id: uuid.UUID
    player: PlayerPublic
    status: AttendanceStatus
    updated_at: datetime


class MatchResponse(BaseModel):
    model_config = {"from_attributes": True}

    id: uuid.UUID
    number: int
    group_id: uuid.UUID
    match_date: date
    start_time: time
    location: str
    address: str | None
    court_type: CourtType | None
    players_per_team: int | None
    max_players: int | None
    notes: str | None
    hash: str
    status: MatchStatus
    created_at: datetime
    updated_at: datetime


class MatchDetailResponse(MatchResponse):
    attendances: list[AttendanceResponse] = []
    confirmed_count: int = 0
    declined_count: int = 0
    pending_count: int = 0
    group_name: str = ""


class SetAttendanceRequest(BaseModel):
    player_id: uuid.UUID
    status: AttendanceStatus
