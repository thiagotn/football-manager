from fastapi import APIRouter

from app.core.dependencies import CurrentPlayer, DB
from app.db.repositories.subscription_repo import SubscriptionRepository
from app.models.player import PlayerRole
from app.schemas.subscription import SubscriptionMeResponse

router = APIRouter(prefix="/subscriptions", tags=["subscriptions"])

# Limites por plano — Fase 1: apenas plano gratuito
PLAN_LIMITS: dict[str, dict] = {
    "free": {"groups_limit": 1, "members_limit": 30},
}


@router.get("/me", response_model=SubscriptionMeResponse)
async def get_my_subscription(db: DB, current: CurrentPlayer):
    """Retorna o plano atual e o uso de recursos do player autenticado."""
    sub_repo = SubscriptionRepository(db)
    sub = await sub_repo.get_or_create(current.id)

    # Admins globais não têm limites
    if current.role == PlayerRole.ADMIN:
        return SubscriptionMeResponse(
            plan=sub.plan,
            groups_limit=None,
            groups_used=0,
            members_limit=None,
        )

    limits = PLAN_LIMITS.get(sub.plan, PLAN_LIMITS["free"])
    groups_used = await sub_repo.count_admin_groups(current.id)

    return SubscriptionMeResponse(
        plan=sub.plan,
        groups_limit=limits["groups_limit"],
        groups_used=groups_used,
        members_limit=limits["members_limit"],
    )
