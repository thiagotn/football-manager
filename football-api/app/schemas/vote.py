from datetime import datetime
from typing import Optional
from uuid import UUID

from pydantic import BaseModel, field_validator, model_validator


class Top5Item(BaseModel):
    player_id: UUID
    position: int

    @field_validator("position")
    @classmethod
    def valid_position(cls, v: int) -> int:
        if v not in (1, 2, 3, 4, 5):
            raise ValueError("Posição deve ser entre 1 e 5")
        return v


class VoteSubmitRequest(BaseModel):
    top5: list[Top5Item]
    flop_player_id: Optional[UUID] = None

    @field_validator("top5")
    @classmethod
    def validate_top5(cls, v: list[Top5Item]) -> list[Top5Item]:
        if not v:
            raise ValueError("Selecione ao menos o 1º lugar")
        if len(v) > 5:
            raise ValueError("Máximo de 5 escolhas")
        positions = [item.position for item in v]
        if len(positions) != len(set(positions)):
            raise ValueError("Posições duplicadas")
        player_ids = [item.player_id for item in v]
        if len(player_ids) != len(set(player_ids)):
            raise ValueError("Jogador repetido no top 5")
        # Posição 1 é obrigatória
        if 1 not in positions:
            raise ValueError("O 1º lugar é obrigatório")
        return v

    @model_validator(mode="after")
    def flop_not_in_top5(self) -> "VoteSubmitRequest":
        if self.flop_player_id:
            top5_ids = {item.player_id for item in self.top5}
            if self.flop_player_id in top5_ids:
                raise ValueError("O mesmo jogador não pode estar no Top 5 e na Decepção")
        return self


class VoteStatusResponse(BaseModel):
    status: str           # 'not_open' | 'open' | 'closed'
    opens_at: datetime
    closes_at: datetime
    voter_count: int
    eligible_count: int
    current_player_voted: bool
    time_label: str       # ex: "Abre em 0h 45min" ou "Fecha em 18h 20min"
    voted_player_ids: list[UUID] = []  # vazio quando status = 'not_open'


class Top5ResultItem(BaseModel):
    position: int
    player_id: UUID
    name: str
    points: int


class FlopResultItem(BaseModel):
    player_id: UUID
    name: str
    votes: int


class VoteResultsResponse(BaseModel):
    top5: list[Top5ResultItem]
    flop: list[FlopResultItem]
    total_voters: int
    eligible_voters: int


class VotePendingItem(BaseModel):
    match_id: UUID
    match_hash: str
    match_number: int
    group_name: str
    time_label: str
    voter_count: int
    eligible_count: int


class VotePendingResponse(BaseModel):
    items: list[VotePendingItem]
