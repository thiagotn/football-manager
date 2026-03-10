from uuid import UUID

from sqlalchemy import text
from sqlalchemy.ext.asyncio import AsyncSession

from app.schemas.group_stats import PlayerStatItem

_STATS_QUERY = text("""
    WITH vote_pts AS (
        SELECT mvt.player_id,
               SUM(mvt.points) AS total_points
        FROM   match_vote_top5 mvt
        JOIN   match_votes mv ON mv.id  = mvt.vote_id
        JOIN   matches     m  ON m.id   = mv.match_id
        WHERE  m.group_id = :group_id
          AND  m.status   = 'closed'
        GROUP BY mvt.player_id
    ),
    flop_cnt AS (
        SELECT mvf.player_id,
               COUNT(*) AS total_flops
        FROM   match_vote_flop mvf
        JOIN   match_votes mv ON mv.id = mvf.vote_id
        JOIN   matches     m  ON m.id  = mv.match_id
        WHERE  m.group_id = :group_id
          AND  m.status   = 'closed'
        GROUP BY mvf.player_id
    ),
    mins AS (
        SELECT a.player_id,
               SUM(EXTRACT(EPOCH FROM (m.end_time - m.start_time)) / 60)::int AS total_minutes
        FROM   attendances a
        JOIN   matches m ON m.id = a.match_id
        WHERE  m.group_id  = :group_id
          AND  m.status    = 'closed'
          AND  a.status    = 'confirmed'
          AND  m.end_time  IS NOT NULL
        GROUP BY a.player_id
    )
    SELECT
        p.id                                    AS player_id,
        COALESCE(p.nickname, p.name)            AS display_name,
        COALESCE(vp.total_points,  0)::int      AS vote_points,
        COALESCE(fc.total_flops,   0)::int      AS flop_votes,
        COALESCE(ms.total_minutes, 0)::int      AS minutes_played
    FROM   group_members gm
    JOIN   players p        ON p.id = gm.player_id
    LEFT JOIN vote_pts  vp  ON vp.player_id = p.id
    LEFT JOIN flop_cnt  fc  ON fc.player_id = p.id
    LEFT JOIN mins      ms  ON ms.player_id = p.id
    WHERE  gm.group_id = :group_id
      AND  p.role != 'admin'
    ORDER BY vote_points DESC, display_name ASC
""")


class GroupStatsRepository:
    def __init__(self, session: AsyncSession):
        self.session = session

    async def get_group_stats(self, group_id: UUID) -> list[PlayerStatItem]:
        result = await self.session.execute(_STATS_QUERY, {"group_id": group_id})
        rows = result.mappings().all()
        return [PlayerStatItem(**row) for row in rows]
