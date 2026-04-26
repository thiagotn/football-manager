import uuid
from datetime import datetime
from enum import Enum

from pydantic import BaseModel


class ExpiresIn(str, Enum):
    h24 = "24h"
    d7 = "7d"


class MCPTokenCreate(BaseModel):
    name: str
    expires_in: ExpiresIn | None = None


class MCPTokenCreated(BaseModel):
    model_config = {"from_attributes": True}

    id: uuid.UUID
    name: str
    token: str
    token_prefix: str
    expires_at: datetime | None
    created_at: datetime


class MCPTokenResponse(BaseModel):
    model_config = {"from_attributes": True}

    id: uuid.UUID
    name: str
    token_prefix: str
    expires_at: datetime | None
    created_at: datetime
    last_used_at: datetime | None
    is_expired: bool
