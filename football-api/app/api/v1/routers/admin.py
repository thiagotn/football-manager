from datetime import datetime, timedelta, timezone

from fastapi import APIRouter, Query
from sqlalchemy import text

from app.core.dependencies import DB, AdminPlayer
from app.schemas.admin import (
    AdminGroupListResponse,
    AdminMatchListResponse,
    AdminStatsResponse,
)

router = APIRouter(prefix="/admin", tags=["admin"])


@router.get("/stats", response_model=AdminStatsResponse)
async def get_admin_stats(db: DB, _: AdminPlayer):
    """Estatísticas globais da plataforma. Exclusivo para super admins."""
    now = datetime.now(timezone.utc)

    result = await db.execute(
        text("""
            SELECT
                (SELECT COUNT(*)::int FROM matches)  AS total_matches,
                (SELECT COUNT(*)::int FROM groups)   AS total_groups,
                (SELECT COUNT(*)::int FROM players)  AS total_players,
                (SELECT COALESCE(
                    SUM(GREATEST(0, EXTRACT(EPOCH FROM (end_time - start_time)) / 60)), 0
                )::int FROM matches WHERE status = 'closed' AND end_time IS NOT NULL
                ) AS platform_minutes_played,
                (SELECT COUNT(*)::int FROM players)  AS signups_total,
                (SELECT COUNT(*)::int FROM players WHERE created_at >= :since_7)  AS signups_last_7_days,
                (SELECT COUNT(*)::int FROM players WHERE created_at >= :since_30) AS signups_last_30_days,
                (SELECT COUNT(*)::int FROM app_reviews)                          AS total_reviews
        """),
        {
            "since_7": now - timedelta(days=7),
            "since_30": now - timedelta(days=30),
        },
    )
    row = result.one()
    return AdminStatsResponse(
        total_matches=row.total_matches,
        total_groups=row.total_groups,
        total_players=row.total_players,
        platform_minutes_played=row.platform_minutes_played,
        signups_total=row.signups_total,
        signups_last_7_days=row.signups_last_7_days,
        signups_last_30_days=row.signups_last_30_days,
        total_reviews=row.total_reviews,
    )


@router.get("/matches", response_model=AdminMatchListResponse)
async def list_admin_matches(
    db: DB,
    _: AdminPlayer,
    status: str | None = Query(None),
    limit: int = Query(50, le=200),
    offset: int = Query(0),
):
    """Lista global de todas as partidas. Exclusivo para super admins."""
    where = "WHERE m.status = :status" if status else ""
    params: dict = {"limit": limit, "offset": offset}
    if status:
        params["status"] = status

    count_result = await db.execute(
        text(f"SELECT COUNT(*)::int FROM matches m {where}"),
        params,
    )
    total = count_result.scalar_one()

    rows = await db.execute(
        text(f"""
            SELECT m.id, m.hash, m.number, m.group_id, g.name AS group_name,
                   m.match_date, m.start_time, m.end_time, m.location, m.status
            FROM matches m
            JOIN groups g ON g.id = m.group_id
            {where}
            ORDER BY m.match_date DESC, m.start_time DESC
            LIMIT :limit OFFSET :offset
        """),
        params,
    )
    items = [dict(row._mapping) for row in rows]
    return {"total": total, "items": items}


@router.get("/groups", response_model=AdminGroupListResponse)
async def list_admin_groups(
    db: DB,
    _: AdminPlayer,
    limit: int = Query(50, le=200),
    offset: int = Query(0),
):
    """Lista global de todos os grupos. Exclusivo para super admins."""
    count_result = await db.execute(text("SELECT COUNT(*)::int FROM groups"))
    total = count_result.scalar_one()

    rows = await db.execute(
        text("""
            SELECT g.id, g.name, g.description, g.slug, g.created_at,
                   COUNT(DISTINCT gm.player_id)::int AS total_members,
                   COUNT(DISTINCT m.id)::int          AS total_matches
            FROM groups g
            LEFT JOIN group_members gm ON gm.group_id = g.id
            LEFT JOIN matches m        ON m.group_id  = g.id
            GROUP BY g.id
            ORDER BY g.created_at DESC
            LIMIT :limit OFFSET :offset
        """),
        {"limit": limit, "offset": offset},
    )
    items = [dict(row._mapping) for row in rows]
    return {"total": total, "items": items}
