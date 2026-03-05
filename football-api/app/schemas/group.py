import uuid
from datetime import datetime
from decimal import Decimal

from pydantic import BaseModel, Field, field_validator
import re

from app.models.group import GroupMemberRole
from app.schemas.player import PlayerPublic, PlayerMemberView


def _make_slug(name: str) -> str:
    slug = name.lower().strip()
    slug = re.sub(r"[^\w\s-]", "", slug)
    slug = re.sub(r"[\s_]+", "-", slug)
    slug = re.sub(r"-+", "-", slug).strip("-")
    return slug[:60]


class GroupCreate(BaseModel):
    name: str = Field(..., min_length=2, max_length=100)
    description: str | None = Field(None, max_length=500)
    slug: str | None = Field(None, max_length=60, description="Deixe vazio para gerar automaticamente")
    per_match_amount: Decimal | None = None
    monthly_amount: Decimal | None = None

    @field_validator("slug", mode="before")
    @classmethod
    def validate_slug(cls, v):
        if v:
            return re.sub(r"[^a-z0-9-]", "", v.lower())
        return v


class GroupUpdate(BaseModel):
    name: str | None = Field(None, min_length=2, max_length=100)
    description: str | None = None
    per_match_amount: Decimal | None = None
    monthly_amount: Decimal | None = None
    recurrence_enabled: bool | None = None


class GroupMemberResponse(BaseModel):
    model_config = {"from_attributes": True}

    id: uuid.UUID
    player: PlayerMemberView
    role: GroupMemberRole
    created_at: datetime


class GroupResponse(BaseModel):
    model_config = {"from_attributes": True}

    id: uuid.UUID
    name: str
    description: str | None
    slug: str
    per_match_amount: Decimal | None
    monthly_amount: Decimal | None
    recurrence_enabled: bool
    created_at: datetime
    updated_at: datetime


class GroupDetailResponse(GroupResponse):
    members: list[GroupMemberResponse] = []
    total_members: int = 0


class AddMemberRequest(BaseModel):
    player_id: uuid.UUID
    role: GroupMemberRole = GroupMemberRole.MEMBER


class UpdateMemberRoleRequest(BaseModel):
    role: GroupMemberRole
