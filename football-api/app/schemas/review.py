from datetime import datetime
from typing import Optional
from uuid import UUID

from pydantic import BaseModel, field_validator


class ReviewUpsertRequest(BaseModel):
    rating: int
    comment: Optional[str] = None

    @field_validator("rating")
    @classmethod
    def rating_range(cls, v: int) -> int:
        if not 1 <= v <= 5:
            raise ValueError("A nota deve ser entre 1 e 5")
        return v

    @field_validator("comment")
    @classmethod
    def no_html(cls, v: Optional[str]) -> Optional[str]:
        if v and "<" in v:
            raise ValueError("O comentário não pode conter HTML")
        if v:
            v = v.strip()
            if len(v) > 500:
                raise ValueError("O comentário deve ter no máximo 500 caracteres")
            return v or None
        return None


class ReviewResponse(BaseModel):
    id: UUID
    rating: int
    comment: Optional[str]
    created_at: datetime
    updated_at: datetime

    model_config = {"from_attributes": True}


class ReviewAdminItem(BaseModel):
    id: UUID
    player_id: UUID
    player_name: str
    rating: int
    comment: Optional[str]
    created_at: datetime
    updated_at: datetime


class DistributionEntry(BaseModel):
    count: int
    percent: float


class ReviewSummaryResponse(BaseModel):
    average: float
    total: int
    distribution: dict[str, DistributionEntry]


class ReviewListResponse(BaseModel):
    items: list[ReviewAdminItem]
    total: int
    page: int
    page_size: int
    total_pages: int
