from datetime import datetime, timezone
from uuid import UUID

from sqlalchemy import text
from sqlalchemy.ext.asyncio import AsyncSession

from app.schemas.player_stats import (
    GroupStatItem,
    MonthlyStatItem,
    PlayerFullStats,
    RecentMatchItem,
)


class PlayerStatsRepository:
    def __init__(self, session: AsyncSession):
        self.session = session

    async def get_full_stats(self, player_id: UUID) -> PlayerFullStats:
        params: dict = {"player_id": player_id}

        # 1. Scalar aggregates (single-row CTEs, cross-joined)
        scalar_result = await self.session.execute(
            text("""
                WITH
                totals AS (
                    SELECT
                        COUNT(*)::int AS total_matches_confirmed,
                        COALESCE(SUM(
                            CASE WHEN m.end_time IS NOT NULL
                                 THEN GREATEST(0, EXTRACT(EPOCH FROM (m.end_time - m.start_time)) / 60)
                                 ELSE 0 END
                        ), 0)::int AS total_minutes_played
                    FROM attendances a
                    JOIN matches m ON m.id = a.match_id
                    WHERE a.player_id = :player_id
                      AND a.status = 'confirmed'
                      AND m.status = 'closed'
                ),
                vote_pts AS (
                    SELECT
                        COALESCE(SUM(mvt.points), 0)::int                        AS total_vote_points,
                        COUNT(*) FILTER (WHERE mvt.position = 1)::int            AS top1_count,
                        COUNT(*)::int                                             AS top5_count
                    FROM match_vote_top5 mvt
                    JOIN match_votes mv ON mv.id = mvt.vote_id
                    WHERE mvt.player_id = :player_id
                ),
                flop_cnt AS (
                    SELECT COUNT(*)::int AS total_flop_votes
                    FROM match_vote_flop mvf
                    JOIN match_votes mv ON mv.id = mvf.vote_id
                    WHERE mvf.player_id = :player_id
                ),
                goals_ast AS (
                    SELECT
                        COALESCE(SUM(mps.goals),   0)::int AS total_goals,
                        COALESCE(SUM(mps.assists), 0)::int AS total_assists
                    FROM match_player_stats mps
                    WHERE mps.player_id = :player_id
                ),
                att_rate AS (
                    SELECT
                        COUNT(*) FILTER (WHERE a.status = 'confirmed')::int AS confirmed_cnt,
                        COUNT(*) FILTER (WHERE a.status = 'declined')::int  AS declined_cnt
                    FROM attendances a
                    JOIN matches m ON m.id = a.match_id
                    WHERE a.player_id = :player_id
                      AND m.status = 'closed'
                )
                SELECT
                    t.total_matches_confirmed,
                    t.total_minutes_played,
                    v.total_vote_points,
                    v.top1_count,
                    v.top5_count,
                    f.total_flop_votes,
                    g.total_goals,
                    g.total_assists,
                    CASE WHEN (r.confirmed_cnt + r.declined_cnt) = 0 THEN 0
                         ELSE ROUND(r.confirmed_cnt * 100.0 / (r.confirmed_cnt + r.declined_cnt))
                    END::int AS attendance_rate
                FROM totals t, vote_pts v, flop_cnt f, goals_ast g, att_rate r
            """),
            params,
        )
        row = scalar_result.mappings().one()

        # 2. Attendance history ordered by date DESC — used for streaks + recent display
        history_result = await self.session.execute(
            text("""
                SELECT
                    m.match_date::text AS match_date,
                    g.name             AS group_name,
                    a.status
                FROM attendances a
                JOIN matches m ON m.id = a.match_id
                JOIN groups  g ON g.id = m.group_id
                WHERE a.player_id = :player_id
                  AND m.status = 'closed'
                  AND a.status IN ('confirmed', 'declined')
                ORDER BY m.match_date DESC
            """),
            params,
        )
        history = history_result.mappings().all()

        # current_streak: consecutive confirmed matches from most recent
        current_streak = 0
        for m in history:
            if m["status"] == "confirmed":
                current_streak += 1
            else:
                break

        # best_streak: longest consecutive run of confirmed
        best_streak, temp = 0, 0
        for m in history:
            if m["status"] == "confirmed":
                temp += 1
                if temp > best_streak:
                    best_streak = temp
            else:
                temp = 0

        recent_matches = [
            RecentMatchItem(
                match_date=m["match_date"],
                group_name=m["group_name"],
                status=m["status"],
            )
            for m in history[:20]
        ]

        # 3. Monthly stats — last 6 months, padded with zeros for missing months
        monthly_result = await self.session.execute(
            text("""
                SELECT
                    TO_CHAR(m.match_date, 'YYYY-MM') AS month,
                    COUNT(*)::int                    AS matches_confirmed,
                    COALESCE(SUM(
                        CASE WHEN m.end_time IS NOT NULL
                             THEN GREATEST(0, EXTRACT(EPOCH FROM (m.end_time - m.start_time)) / 60)
                             ELSE 0 END
                    ), 0)::int                       AS minutes_played
                FROM attendances a
                JOIN matches m ON m.id = a.match_id
                WHERE a.player_id = :player_id
                  AND a.status = 'confirmed'
                  AND m.status = 'closed'
                  AND m.match_date >= DATE_TRUNC('month', NOW() AT TIME ZONE 'America/Sao_Paulo') - INTERVAL '5 months'
                GROUP BY TO_CHAR(m.match_date, 'YYYY-MM')
                ORDER BY month
            """),
            params,
        )
        monthly_map = {r["month"]: r for r in monthly_result.mappings().all()}

        now = datetime.now(timezone.utc)
        monthly_stats: list[MonthlyStatItem] = []
        for i in range(5, -1, -1):
            month = now.month - i
            year = now.year
            while month <= 0:
                month += 12
                year -= 1
            key = f"{year:04d}-{month:02d}"
            if key in monthly_map:
                r = monthly_map[key]
                monthly_stats.append(
                    MonthlyStatItem(
                        month=key,
                        matches_confirmed=r["matches_confirmed"],
                        minutes_played=r["minutes_played"],
                    )
                )
            else:
                monthly_stats.append(
                    MonthlyStatItem(month=key, matches_confirmed=0, minutes_played=0)
                )

        # 4. Groups the player belongs to, with per-group confirmed match count
        groups_result = await self.session.execute(
            text("""
                SELECT
                    gm.group_id::text AS group_id,
                    g.name            AS group_name,
                    COALESCE(gm.skill_stars, 2)::int   AS skill_stars,
                    COALESCE(gm.position, 'mei')        AS position,
                    gm.role::text                       AS role,
                    COUNT(a.match_id) FILTER (
                        WHERE a.status = 'confirmed' AND m.status = 'closed'
                    )::int AS matches_confirmed
                FROM group_members gm
                JOIN groups g ON g.id = gm.group_id
                LEFT JOIN matches m     ON m.group_id = gm.group_id
                LEFT JOIN attendances a ON a.match_id = m.id AND a.player_id = gm.player_id
                WHERE gm.player_id = :player_id
                GROUP BY gm.group_id, g.name, gm.skill_stars, gm.position, gm.role
                ORDER BY matches_confirmed DESC
            """),
            params,
        )
        groups = [
            GroupStatItem(
                group_id=r["group_id"],
                group_name=r["group_name"],
                skill_stars=r["skill_stars"],
                position=r["position"],
                role=r["role"],
                matches_confirmed=r["matches_confirmed"],
            )
            for r in groups_result.mappings().all()
        ]

        return PlayerFullStats(
            total_matches_confirmed=row["total_matches_confirmed"],
            total_minutes_played=row["total_minutes_played"],
            total_vote_points=row["total_vote_points"],
            top1_count=row["top1_count"],
            top5_count=row["top5_count"],
            total_flop_votes=row["total_flop_votes"],
            total_goals=row["total_goals"],
            total_assists=row["total_assists"],
            current_streak=current_streak,
            best_streak=best_streak,
            attendance_rate=row["attendance_rate"],
            monthly_stats=monthly_stats,
            recent_matches=recent_matches,
            groups=groups,
        )
