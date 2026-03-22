import re
import uuid
from datetime import datetime

from pydantic import BaseModel, Field, field_validator

from app.models.player import PlayerRole


def normalize_whatsapp(v: str) -> str:
    """Normalize a phone number to E.164 format (+CCNUMBER)."""
    cleaned = re.sub(r"[\s\-\(\)\.]", "", v)
    if cleaned.startswith("+"):
        digits = re.sub(r"\D", "", cleaned[1:])
        cleaned = "+" + digits
    else:
        digits = re.sub(r"\D", "", cleaned)
        cleaned = "+" + digits
    if not re.match(r"^\+[1-9]\d{6,14}$", cleaned):
        raise ValueError("WhatsApp inválido. Use formato internacional: +5511999990000")
    return cleaned


class PlayerCreate(BaseModel):
    name: str = Field(..., min_length=2, max_length=100)
    nickname: str | None = Field(None, max_length=50)
    whatsapp: str = Field(..., description="Número WhatsApp (somente dígitos, com DDD)")
    password: str = Field(..., min_length=6)
    role: PlayerRole = PlayerRole.PLAYER

    @field_validator("whatsapp")
    @classmethod
    def validate_whatsapp(cls, v: str) -> str:
        return normalize_whatsapp(v)


class PlayerUpdate(BaseModel):
    name: str | None = Field(None, min_length=2, max_length=100)
    nickname: str | None = Field(None, max_length=50)
    whatsapp: str | None = None
    password: str | None = Field(None, min_length=6)
    role: PlayerRole | None = None
    active: bool | None = None

    @field_validator("whatsapp")
    @classmethod
    def validate_whatsapp(cls, v: str | None) -> str | None:
        if v is None:
            return v
        return normalize_whatsapp(v)


class PlayerResponse(BaseModel):
    model_config = {"from_attributes": True}

    id: uuid.UUID
    name: str
    nickname: str | None
    whatsapp: str
    role: PlayerRole
    active: bool
    must_change_password: bool
    created_at: datetime
    updated_at: datetime


class ResetPasswordResponse(BaseModel):
    temp_password: str


class PlayerPublic(BaseModel):
    """Dados públicos de um jogador (sem whatsapp)"""
    model_config = {"from_attributes": True}

    id: uuid.UUID
    name: str
    nickname: str | None
    role: PlayerRole


class PlayerMemberView(BaseModel):
    """Dados de jogador exibidos para admins de grupo (inclui whatsapp)"""
    model_config = {"from_attributes": True}

    id: uuid.UUID
    name: str
    nickname: str | None
    role: PlayerRole
    whatsapp: str
