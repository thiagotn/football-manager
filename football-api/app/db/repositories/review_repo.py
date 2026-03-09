from uuid import UUID

from sqlalchemy import func, select, text
from sqlalchemy.ext.asyncio import AsyncSession

from app.models.app_review import AppReview
from app.models.player import Player


class ReviewRepository:
    def __init__(self, session: AsyncSession):
        self.session = session

    async def get_by_player(self, player_id: UUID) -> AppReview | None:
        result = await self.session.execute(
            select(AppReview).where(AppReview.player_id == player_id)
        )
        return result.scalar_one_or_none()

    async def upsert(self, player_id: UUID, rating: int, comment: str | None) -> AppReview:
        review = await self.get_by_player(player_id)
        if review:
            review.rating = rating
            review.comment = comment
            review.updated_at = func.now()  # type: ignore[assignment]
        else:
            review = AppReview(player_id=player_id, rating=rating, comment=comment)
            self.session.add(review)
        await self.session.flush()
        await self.session.refresh(review)
        return review

    async def get_summary(self) -> dict:
        total_result = await self.session.execute(
            select(func.count()).select_from(AppReview)
        )
        total = total_result.scalar_one()

        if total == 0:
            distribution = {
                str(i): {"count": 0, "percent": 0.0} for i in range(1, 6)
            }
            return {"average": 0.0, "total": 0, "distribution": distribution}

        avg_result = await self.session.execute(
            select(func.avg(AppReview.rating))
        )
        average = float(avg_result.scalar_one() or 0)

        dist_result = await self.session.execute(
            select(AppReview.rating, func.count().label("cnt"))
            .group_by(AppReview.rating)
        )
        counts: dict[int, int] = {row.rating: row.cnt for row in dist_result}

        distribution = {}
        for i in range(1, 6):
            cnt = counts.get(i, 0)
            distribution[str(i)] = {
                "count": cnt,
                "percent": round(cnt / total * 100, 1) if total > 0 else 0.0,
            }

        return {"average": round(average, 1), "total": total, "distribution": distribution}

    async def list_all(
        self,
        ratings: list[int] | None,
        order_by: str,
        page: int,
        page_size: int,
    ) -> tuple[list[dict], int]:
        # Base query joining with players for name
        filters = []
        if ratings:
            filters.append(AppReview.rating.in_(ratings))

        count_q = select(func.count()).select_from(AppReview)
        if filters:
            count_q = count_q.where(*filters)
        total_result = await self.session.execute(count_q)
        total = total_result.scalar_one()

        order_col = AppReview.rating if order_by == "rating" else AppReview.created_at.desc()  # type: ignore[attr-defined]
        if order_by == "rating":
            order_col = AppReview.rating.desc()  # type: ignore[attr-defined]

        q = (
            select(AppReview, Player.name.label("player_name"))
            .join(Player, Player.id == AppReview.player_id)
            .order_by(order_col)
            .offset((page - 1) * page_size)
            .limit(page_size)
        )
        if filters:
            q = q.where(*filters)

        result = await self.session.execute(q)
        rows = result.all()

        items = [
            {
                "id": row.AppReview.id,
                "player_id": row.AppReview.player_id,
                "player_name": row.player_name,
                "rating": row.AppReview.rating,
                "comment": row.AppReview.comment,
                "created_at": row.AppReview.created_at,
                "updated_at": row.AppReview.updated_at,
            }
            for row in rows
        ]
        return items, total
