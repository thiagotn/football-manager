from calendar import monthrange
from datetime import datetime, timezone
from uuid import UUID

from sqlalchemy import text
from sqlalchemy.ext.asyncio import AsyncSession

from app.schemas.group_stats import PlayerStatItem

_MONTHS_PT = [
    "Janeiro", "Fevereiro", "Março", "Abril", "Maio", "Junho",
    "Julho", "Agosto", "Setembro", "Outubro", "Novembro", "Dezembro",
]

# Partidas elegíveis: encerradas E com votação também encerrada (closes_at < NOW())
# closes_at = match_date + end_time (ou 23:59) em BRT + vote_open_delay_minutes + vote_duration_hours
_QUERY_TEMPLATE = """
WITH eligible_matches AS (
    SELECT id, match_date
    FROM   matches
    WHERE  group_id = :group_id
      AND  status   = 'closed'
      AND  (
        (match_date + COALESCE(end_time, '23:59:00'::time))::timestamp
        AT TIME ZONE 'America/Sao_Paulo'
        + (vote_open_delay_minutes || ' minutes')::interval
        + (vote_duration_hours    || ' hours')::interval
      ) < NOW()
    {date_filter}
),
vote_pts AS (
    SELECT mvt.player_id,
           SUM(mvt.points) AS total_points
    FROM   match_vote_top5 mvt
    JOIN   match_votes mv ON mv.id  = mvt.vote_id
    JOIN   eligible_matches em ON em.id = mv.match_id
    GROUP BY mvt.player_id
),
flop_cnt AS (
    SELECT mvf.player_id,
           COUNT(*) AS total_flops
    FROM   match_vote_flop mvf
    JOIN   match_votes mv ON mv.id = mvf.vote_id
    JOIN   eligible_matches em ON em.id = mv.match_id
    GROUP BY mvf.player_id
),
mins AS (
    SELECT a.player_id,
           SUM(GREATEST(0, EXTRACT(EPOCH FROM (m.end_time - m.start_time)) / 60))::int AS total_minutes
    FROM   attendances a
    JOIN   matches m ON m.id = a.match_id
    JOIN   eligible_matches em ON em.id = m.id
    WHERE  a.status    = 'confirmed'
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
"""


class GroupStatsRepository:
    def __init__(self, session: AsyncSession):
        self.session = session

    async def get_group_stats(
        self,
        group_id: UUID,
        period: str = "annual",
        month: str | None = None,
    ) -> tuple[list[PlayerStatItem], str]:
        params: dict = {"group_id": group_id}

        if period == "monthly" and month:
            year, m = int(month[:4]), int(month[5:7])
            _, last_day = monthrange(year, m)
            date_filter = "AND match_date BETWEEN :month_start AND :month_end"
            params["month_start"] = f"{year:04d}-{m:02d}-01"
            params["month_end"] = f"{year:04d}-{m:02d}-{last_day:02d}"
            period_label = f"{_MONTHS_PT[m - 1]} {year}"
        else:
            current_year = datetime.now(timezone.utc).year
            date_filter = "AND EXTRACT(YEAR FROM match_date) = :current_year"
            params["current_year"] = current_year
            period_label = f"Ano {current_year}"

        query = _QUERY_TEMPLATE.format(date_filter=date_filter)
        result = await self.session.execute(text(query), params)
        rows = result.mappings().all()
        return [PlayerStatItem(**row) for row in rows], period_label
