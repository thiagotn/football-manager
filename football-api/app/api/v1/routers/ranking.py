from typing import Literal

from fastapi import APIRouter, Query

from app.core.dependencies import DB
from app.db.repositories.ranking_repo import RankingRepository
from app.schemas.ranking import FlopRankingItem, RankingResponse, TopRankingItem

router = APIRouter(prefix="/ranking", tags=["ranking"])


@router.get("", response_model=RankingResponse)
async def get_ranking(
    db: DB,
    period: Literal["month", "year", "all"] = Query("month"),
    type: Literal["top", "flop"] = Query("top"),
):
    """
    Retorna o ranking geral da plataforma (público, sem autenticação).

    - period: month | year | all
    - type: top (melhores por pontos) | flop (decepções por votos)

    Apenas partidas com ao menos 10 presenças confirmadas são consideradas.
    Super admins são excluídos do ranking.
    """
    repo = RankingRepository(db)
    if type == "top":
        items = await repo.get_top(period)
        return RankingResponse(
            period=period,
            type=type,
            items=[TopRankingItem(**i) for i in items],
        )
    else:
        items = await repo.get_flop(period)
        return RankingResponse(
            period=period,
            type=type,
            items=[FlopRankingItem(**i) for i in items],
        )
