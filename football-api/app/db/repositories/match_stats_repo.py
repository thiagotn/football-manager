from uuid import UUID

from sqlalchemy import delete, and_
from sqlalchemy.dialects.postgresql import insert as pg_insert
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy.future import select
from sqlalchemy.orm import joinedload

from app.models.match import MatchPlayerStats
from app.models.player import Player


class MatchStatsRepository:
    def __init__(self, session: AsyncSession):
        self.session = session

    async def get_by_match(self, match_id: UUID) -> list[MatchPlayerStats]:
        result = await self.session.execute(
            select(MatchPlayerStats)
            .options(joinedload(MatchPlayerStats.player))
            .where(MatchPlayerStats.match_id == match_id)
            .order_by(MatchPlayerStats.goals.desc(), MatchPlayerStats.assists.desc())
        )
        return list(result.scalars().all())

    async def upsert_stats(
        self,
        match_id: UUID,
        recorded_by_id: UUID,
        stats: list[dict],  # list of {"player_id": UUID, "goals": int, "assists": int}
    ) -> list[MatchPlayerStats]:
        """Substitui todos os registros da partida pelo payload recebido (operação PUT).

        Jogadores não incluídos no payload têm seus registros removidos.
        Jogadores incluídos são inseridos ou atualizados via ON CONFLICT DO UPDATE.
        """
        if not stats:
            await self.session.execute(
                delete(MatchPlayerStats).where(MatchPlayerStats.match_id == match_id)
            )
            await self.session.flush()
            return []

        player_ids_in_payload = [s["player_id"] for s in stats]

        # Remove registros de jogadores que não estão no payload
        await self.session.execute(
            delete(MatchPlayerStats).where(
                and_(
                    MatchPlayerStats.match_id == match_id,
                    MatchPlayerStats.player_id.not_in(player_ids_in_payload),
                )
            )
        )

        # Bulk upsert
        values = [
            {
                "match_id": match_id,
                "player_id": s["player_id"],
                "goals": s["goals"],
                "assists": s["assists"],
                "recorded_by": recorded_by_id,
            }
            for s in stats
        ]
        stmt = pg_insert(MatchPlayerStats).values(values)
        stmt = stmt.on_conflict_do_update(
            index_elements=["match_id", "player_id"],
            set_={
                "goals": stmt.excluded.goals,
                "assists": stmt.excluded.assists,
                "recorded_by": stmt.excluded.recorded_by,
                "updated_at": stmt.excluded.updated_at,
            },
        )
        await self.session.execute(stmt)
        await self.session.flush()

        return await self.get_by_match(match_id)
