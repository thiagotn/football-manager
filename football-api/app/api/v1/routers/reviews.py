from fastapi import APIRouter, HTTPException, Query

from app.core.dependencies import AdminPlayer, CurrentPlayer, DB
from app.core.exceptions import ForbiddenError
from app.db.repositories.review_repo import ReviewRepository
from app.models.player import PlayerRole
from app.schemas.review import (
    ReviewListResponse,
    ReviewResponse,
    ReviewSummaryResponse,
    ReviewUpsertRequest,
)

router = APIRouter(prefix="/reviews", tags=["reviews"])


@router.get("/me", response_model=ReviewResponse)
async def get_my_review(db: DB, current: CurrentPlayer):
    if current.role == PlayerRole.ADMIN:
        raise ForbiddenError("Super-admins não podem avaliar o app")
    repo = ReviewRepository(db)
    review = await repo.get_by_player(current.id)
    if not review:
        raise HTTPException(status_code=404, detail="Nenhuma avaliação encontrada")
    return review


@router.put("/me", response_model=ReviewResponse)
async def upsert_my_review(body: ReviewUpsertRequest, db: DB, current: CurrentPlayer):
    if current.role == PlayerRole.ADMIN:
        raise ForbiddenError("Super-admins não podem avaliar o app")
    repo = ReviewRepository(db)
    review = await repo.upsert(current.id, body.rating, body.comment)
    return review


@router.get("/summary", response_model=ReviewSummaryResponse)
async def get_summary(db: DB, _: AdminPlayer):
    repo = ReviewRepository(db)
    data = await repo.get_summary()
    return data


@router.get("", response_model=ReviewListResponse)
async def list_reviews(
    db: DB,
    _: AdminPlayer,
    rating: str | None = Query(None, description="Filtrar por notas, ex: 1,2"),
    order_by: str = Query("created_at", pattern="^(created_at|rating)$"),
    page: int = Query(1, ge=1),
    page_size: int = Query(20, ge=1, le=100),
):
    ratings = [int(r) for r in rating.split(",") if r.strip().isdigit()] if rating else None
    repo = ReviewRepository(db)
    items, total = await repo.list_all(ratings, order_by, page, page_size)
    total_pages = max(1, -(-total // page_size))  # ceil division
    return ReviewListResponse(
        items=items,
        total=total,
        page=page,
        page_size=page_size,
        total_pages=total_pages,
    )
