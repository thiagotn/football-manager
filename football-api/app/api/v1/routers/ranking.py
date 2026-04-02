from typing import Literal

from fastapi import APIRouter, Query

from app.core.dependencies import DB
from app.core.exceptions import ValidationError
from app.db.repositories.ranking_repo import RankingRepository
from app.schemas.ranking import FlopRankingItem, RankingResponse, TopRankingItem

router = APIRouter(prefix="/ranking", tags=["ranking"])


@router.get("", response_model=RankingResponse)
async def get_ranking(
    db: DB,
    type: Literal["top", "flop"] = Query("top"),
    year: int | None = Query(None, ge=2024, le=2100),
    month: int | None = Query(None, ge=1, le=12),
):
    """
    Retorna o ranking geral da plataforma (público, sem autenticação).

    - year + month → ranking do mês específico
    - year only    → ranking do ano completo
    - sem year     → todos os tempos
    - type: top (melhores por pontos) | flop (decepções por votos)

    Apenas partidas com ao menos 10 presenças confirmadas são consideradas.
    Super admins são excluídos do ranking.
    """
    if month is not None and year is None:
        raise ValidationError("month requires year to be provided")

    repo = RankingRepository(db)
    if type == "top":
        items = await repo.get_top(year, month)
        return RankingResponse(
            year=year,
            month=month,
            type=type,
            items=[TopRankingItem(**i) for i in items],
        )
    else:
        items = await repo.get_flop(year, month)
        return RankingResponse(
            year=year,
            month=month,
            type=type,
            items=[FlopRankingItem(**i) for i in items],
        )
