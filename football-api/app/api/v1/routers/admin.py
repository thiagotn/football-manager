from datetime import datetime, timedelta, timezone
from uuid import UUID

import structlog
from fastapi import APIRouter, HTTPException, Query
from sqlalchemy import text

import stripe.error

from app.core.dependencies import DB, AdminPlayer
from app.services import billing as billing_service
from app.db.repositories.subscription_repo import SubscriptionRepository
from app.schemas.admin import (
    AdminGroupListResponse,
    AdminMatchListResponse,
    AdminStatsResponse,
    AdminSubscriptionListResponse,
    AdminSubscriptionSummary,
    AdminSubscriptionUpdateRequest,
)

logger = structlog.get_logger()

router = APIRouter(prefix="/admin", tags=["admin"])

# Preços em centavos para cálculo de MRR estimado
_MRR_CENTS = {
    ("basic", "monthly"): 1990,
    ("basic", "yearly"):  199_00 // 12,   # ~1658
    ("pro",   "monthly"): 3990,
    ("pro",   "yearly"):  399_00 // 12,   # ~3325
}


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


# ── Subscriptions admin ────────────────────────────────────────────────────────

@router.get("/subscriptions/summary", response_model=AdminSubscriptionSummary)
async def get_subscription_summary(db: DB, _: AdminPlayer):
    """Resumo de assinaturas para os cards do painel. Exclusivo para super admins."""
    total_result = await db.execute(text("SELECT COUNT(*)::int FROM players"))
    total_players = total_result.scalar_one()

    rows = await db.execute(text("""
        SELECT
            COALESCE(ps.status, 'active') AS status,
            COALESCE(ps.plan, 'free')     AS plan,
            COALESCE(ps.billing_cycle, 'monthly') AS billing_cycle,
            COUNT(*)::int AS cnt
        FROM players p
        LEFT JOIN player_subscriptions ps ON ps.player_id = p.id
        GROUP BY ps.status, ps.plan, ps.billing_cycle
    """))

    active = free = past_due = canceled = 0
    mrr_cents = 0
    breakdown_map: dict[tuple, int] = {}

    for row in rows:
        plan = row.plan or "free"
        status = row.status or "active"
        cycle = row.billing_cycle or "monthly"
        cnt = row.cnt

        if plan == "free" or status == "canceled":
            free += cnt if plan == "free" else 0
            canceled += cnt if status == "canceled" else 0
        elif status == "past_due":
            past_due += cnt
        elif status == "active" and plan != "free":
            active += cnt
            mrr_cents += _MRR_CENTS.get((plan, cycle), 0) * cnt
        else:
            free += cnt

        if plan != "free" and status != "canceled":
            key = (plan, cycle)
            breakdown_map[key] = breakdown_map.get(key, 0) + cnt

    breakdown = [
        {"plan": plan, "billing_cycle": cycle, "count": cnt}
        for (plan, cycle), cnt in sorted(breakdown_map.items())
    ]

    return AdminSubscriptionSummary(
        total_players=total_players,
        active=active,
        free=free,
        past_due=past_due,
        canceled=canceled,
        mrr_cents=mrr_cents,
        breakdown=breakdown,
    )


@router.get("/subscriptions", response_model=AdminSubscriptionListResponse)
async def list_subscriptions(
    db: DB,
    _: AdminPlayer,
    status: str | None = Query(None),
    plan: str | None = Query(None),
    page: int = Query(1, ge=1),
    page_size: int = Query(20, ge=1, le=100),
):
    """Lista paginada de assinantes pagos. Exclusivo para super admins."""
    conditions = ["ps.plan != 'free'"]
    params: dict = {"limit": page_size, "offset": (page - 1) * page_size}

    if status:
        conditions.append("ps.status = :status")
        params["status"] = status
    if plan:
        conditions.append("ps.plan = :plan")
        params["plan"] = plan

    where = "WHERE " + " AND ".join(conditions)

    count_result = await db.execute(
        text(f"""
            SELECT COUNT(*)::int
            FROM player_subscriptions ps
            JOIN players p ON p.id = ps.player_id
            {where}
        """),
        params,
    )
    total = count_result.scalar_one()

    rows = await db.execute(
        text(f"""
            SELECT
                p.id        AS player_id,
                p.name      AS player_name,
                ps.plan,
                ps.billing_cycle,
                ps.status,
                ps.current_period_end,
                ps.grace_period_end,
                ps.gateway_customer_id,
                ps.gateway_sub_id,
                ps.created_at
            FROM player_subscriptions ps
            JOIN players p ON p.id = ps.player_id
            {where}
            ORDER BY
                CASE ps.status WHEN 'past_due' THEN 0 ELSE 1 END,
                ps.current_period_end ASC NULLS LAST
            LIMIT :limit OFFSET :offset
        """),
        params,
    )
    items = [dict(row._mapping) for row in rows]
    return AdminSubscriptionListResponse(
        total=total,
        page=page,
        page_size=page_size,
        items=items,
    )


@router.patch("/subscriptions/{player_id}", status_code=200)
async def update_subscription(
    player_id: UUID,
    body: AdminSubscriptionUpdateRequest,
    db: DB,
    _: AdminPlayer,
):
    """Força atualização manual do plano. Usado quando webhook falhou. Exclusivo para super admins."""
    sub_repo = SubscriptionRepository(db)
    sub = await sub_repo.get_by_player(player_id)
    if not sub:
        raise HTTPException(status_code=404, detail="Assinatura não encontrada para este player.")

    await sub_repo.update_plan(
        player_id=player_id,
        plan=body.plan,
        status=body.status,
        billing_cycle=body.billing_cycle,
    )

    logger.info(
        "admin_subscription_manual_update",
        player_id=str(player_id),
        plan=body.plan,
        status=body.status,
        reason=body.reason,
    )
    return {"status": "ok", "plan": body.plan}


@router.post("/subscriptions/{player_id}/cancel", status_code=200)
async def cancel_subscription(
    player_id: UUID,
    db: DB,
    _: AdminPlayer,
):
    """Cancela a assinatura no Stripe e regride o player para free. Exclusivo para super admins."""
    sub_repo = SubscriptionRepository(db)
    sub = await sub_repo.get_by_player(player_id)
    if not sub:
        raise HTTPException(status_code=404, detail="Assinatura não encontrada para este player.")

    if sub.plan == "free" or sub.status == "canceled":
        raise HTTPException(status_code=400, detail="Assinatura já está cancelada ou é free.")

    # Cancela no Stripe se houver subscription_id
    if sub.gateway_sub_id:
        try:
            await billing_service.cancel_subscription(sub.gateway_sub_id)
        except stripe.error.InvalidRequestError as e:
            # Sub já cancelada no Stripe — prosseguir para atualizar o DB
            logger.warning("stripe_cancel_already_canceled", error=str(e), player_id=str(player_id))

    await sub_repo.update_plan(
        player_id=player_id,
        plan="free",
        status="canceled",
    )

    logger.info("admin_subscription_canceled", player_id=str(player_id))
    return {"status": "ok"}
