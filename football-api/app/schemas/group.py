import uuid
from datetime import datetime
from decimal import Decimal

from pydantic import BaseModel, Field, field_validator, model_validator
import re

from app.models.group import GroupMemberRole
from app.schemas.player import PlayerPublic, PlayerMemberView, normalize_nickname


VALID_BIB_COLORS = {"laranja", "azul", "verde", "vermelho", "amarelo", "preto", "branco"}


class TeamSlot(BaseModel):
    color: str | None = None   # slug da paleta ou None
    name: str | None = Field(None, max_length=40)

    @field_validator("color")
    @classmethod
    def validate_color(cls, v: str | None) -> str | None:
        if v is not None and v not in VALID_BIB_COLORS:
            raise ValueError(f"Cor inválida: {v!r}. Use um dos slugs: {sorted(VALID_BIB_COLORS)}")
        return v

    @model_validator(mode="after")
    def at_least_one(self) -> "TeamSlot":
        if not self.color and not self.name:
            raise ValueError("slot deve ter cor ou nome")
        return self


def _validate_iana_timezone(v: str) -> str:
    import zoneinfo
    try:
        zoneinfo.ZoneInfo(v)
    except (KeyError, Exception):
        raise ValueError(f"Timezone inválido: {v!r}")
    return v


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
    is_public: bool = True
    vote_open_delay_minutes: int = Field(20, ge=0, le=120)
    vote_duration_hours: int = Field(24, ge=2, le=72)
    timezone: str = Field("America/Sao_Paulo", max_length=60)

    @field_validator("slug", mode="before")
    @classmethod
    def validate_slug(cls, v):
        if v:
            return re.sub(r"[^a-z0-9-]", "", v.lower())
        return v

    @field_validator("timezone")
    @classmethod
    def validate_timezone(cls, v: str) -> str:
        return _validate_iana_timezone(v)


class GroupUpdate(BaseModel):
    name: str | None = Field(None, min_length=2, max_length=100)
    description: str | None = None
    per_match_amount: Decimal | None = None
    monthly_amount: Decimal | None = None
    recurrence_enabled: bool | None = None
    is_public: bool | None = None
    vote_open_delay_minutes: int | None = Field(None, ge=0, le=120)
    vote_duration_hours: int | None = Field(None, ge=2, le=72)
    timezone: str | None = Field(None, max_length=60)
    team_slots: list[TeamSlot] | None = Field(None, max_length=5)

    @field_validator("timezone")
    @classmethod
    def validate_timezone(cls, v: str | None) -> str | None:
        if v is None:
            return v
        return _validate_iana_timezone(v)


class GroupMemberResponse(BaseModel):
    model_config = {"from_attributes": True}

    id: uuid.UUID
    player: PlayerMemberView
    role: GroupMemberRole
    skill_stars: int | None = None
    position: str | None = None
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
    is_public: bool
    vote_open_delay_minutes: int
    vote_duration_hours: int
    timezone: str = "America/Sao_Paulo"
    team_slots: list[TeamSlot] | None = None
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


class UpdateMemberRequest(BaseModel):
    role: GroupMemberRole | None = None
    skill_stars: int | None = Field(None, ge=1, le=5)
    position: str | None = Field(None, pattern=r'^(gk|zag|lat|mei|ata)$')
    nickname: str | None = Field(None, max_length=50)

    @field_validator("nickname", mode="before")
    @classmethod
    def trim_nickname(cls, v: str | None) -> str | None:
        return normalize_nickname(v)


class SelfUpdatePositionRequest(BaseModel):
    position: str = Field(..., pattern=r'^(gk|zag|lat|mei|ata)$')


# ── Add member by phone ───────────────────────────────────────────────────────

class LookupPlayerInfo(BaseModel):
    id: uuid.UUID
    name: str
    nickname: str | None = None
    avatar_url: str | None = None


class LookupMemberResponse(BaseModel):
    status: str  # "found" | "not_found" | "already_member"
    player: LookupPlayerInfo | None = None


class AddMemberByPhoneRequest(BaseModel):
    whatsapp: str
    name: str | None = Field(None, min_length=2, max_length=100)
    nickname: str | None = Field(None, max_length=50)
    skill_stars: int = Field(2, ge=1, le=5)
    position: str = Field("mei", pattern=r'^(gk|zag|lat|mei|ata)$')

    @field_validator("nickname", mode="before")
    @classmethod
    def trim_nickname(cls, v: str | None) -> str | None:
        return normalize_nickname(v)


class AddMemberByPhoneResponse(BaseModel):
    member: GroupMemberResponse
    is_new: bool


# ── Waitlist ──────────────────────────────────────────────────────────────────

class WaitlistJoinRequest(BaseModel):
    agreed: bool
    intro: str | None = Field(None, max_length=500)


class WaitlistActionRequest(BaseModel):
    action: str  # "accept" or "reject"


class WaitlistEntryResponse(BaseModel):
    model_config = {"from_attributes": True}

    id: uuid.UUID
    match_id: uuid.UUID
    player_id: uuid.UUID
    player_name: str
    player_nickname: str | None
    intro: str | None
    status: str
    created_at: datetime
