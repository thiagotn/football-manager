import uuid
from datetime import datetime

from pydantic import BaseModel


class InviteCreateRequest(BaseModel):
    group_id: uuid.UUID


class InviteResponse(BaseModel):
    model_config = {"from_attributes": True}

    id: uuid.UUID
    group_id: uuid.UUID
    token: str
    expires_at: datetime
    used: bool
    created_at: datetime


class InviteAcceptRequest(BaseModel):
    token: str
    name: str
    nickname: str | None = None
    whatsapp: str
    password: str
