import uuid
from datetime import datetime

from pydantic import BaseModel, field_validator

from app.schemas.player import normalize_whatsapp


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


class InviteCheckRequest(BaseModel):
    whatsapp: str

    @field_validator("whatsapp")
    @classmethod
    def validate_whatsapp(cls, v: str) -> str:
        return normalize_whatsapp(v)


class InviteCheckResponse(BaseModel):
    exists: bool
    first_name: str | None = None


class InviteAcceptRequest(BaseModel):
    name: str | None = None      # obrigatório apenas para usuário novo
    nickname: str | None = None
    whatsapp: str
    password: str                # obrigatório para ambos (novo cadastro e login)

    @field_validator("whatsapp")
    @classmethod
    def validate_whatsapp(cls, v: str) -> str:
        return normalize_whatsapp(v)
