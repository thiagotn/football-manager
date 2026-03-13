from fastapi import APIRouter, HTTPException

from app.core.config import get_settings
from app.core.dependencies import CurrentPlayer, DB
from app.db.repositories.subscription_repo import SubscriptionRepository
from app.models.player import PlayerRole
from app.schemas.subscription import (
    CheckoutSessionRequest,
    CheckoutSessionResponse,
    SubscriptionMeResponse,
)
from app.services import billing

router = APIRouter(prefix="/subscriptions", tags=["subscriptions"])

# Limites por plano — fonte de verdade no backend.
# Sincronizar com src/lib/plans.ts no frontend.
PLAN_LIMITS: dict[str, dict] = {
    "free":  {"groups_limit": 1,  "members_limit": 30},
    "basic": {"groups_limit": 3,  "members_limit": 50},
    "pro":   {"groups_limit": 10, "members_limit": None},
}

PAID_PLANS = {"basic", "pro"}


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
            status=sub.status,
        )

    limits = PLAN_LIMITS.get(sub.plan, PLAN_LIMITS["free"])
    groups_used = await sub_repo.count_admin_groups(current.id)

    return SubscriptionMeResponse(
        plan=sub.plan,
        groups_limit=limits["groups_limit"],
        groups_used=groups_used,
        members_limit=limits["members_limit"],
        status=sub.status,
        gateway_customer_id=sub.gateway_customer_id,
        gateway_sub_id=sub.gateway_sub_id,
        current_period_end=sub.current_period_end,
        grace_period_end=sub.grace_period_end,
    )


@router.post("", response_model=CheckoutSessionResponse, status_code=201)
async def create_checkout_session(
    body: CheckoutSessionRequest,
    db: DB,
    current: CurrentPlayer,
):
    """
    Inicia checkout Stripe para upgrade de plano.

    Cria (ou reutiliza) um Stripe Customer vinculado ao player e gera
    uma Checkout Session. Retorna a URL para redirect do frontend.

    PRD: RF-02, fluxo 7.2.
    """
    if body.plan not in PAID_PLANS:
        raise HTTPException(status_code=400, detail=f"Plano '{body.plan}' inválido para checkout.")

    if body.billing_cycle not in ("monthly", "yearly"):
        raise HTTPException(status_code=400, detail="billing_cycle deve ser 'monthly' ou 'yearly'.")

    settings = get_settings()

    # Valida que o Price ID está configurado antes de criar o Customer
    try:
        settings.get_price_id(body.plan, body.billing_cycle)
    except ValueError as exc:
        raise HTTPException(status_code=503, detail=str(exc)) from exc

    sub_repo = SubscriptionRepository(db)
    sub = await sub_repo.get_or_create(current.id)

    # Reutiliza o Customer Stripe existente ou cria um novo
    customer_id = sub.gateway_customer_id
    if not customer_id:
        customer_id = await billing.get_or_create_customer(
            player_id=str(current.id),
            name=current.name,
            phone=current.whatsapp,
        )
        # Persiste o customer_id imediatamente para evitar duplicatas
        await sub_repo.update_plan(
            player_id=current.id,
            plan=sub.plan,
            gateway_customer_id=customer_id,
        )

    frontend_url = settings.frontend_url
    checkout_url = await billing.create_checkout_session(
        customer_id=customer_id,
        player_id=str(current.id),
        plan=body.plan,
        billing_cycle=body.billing_cycle,
        success_url=f"{frontend_url}/account/checkout/success?session_id={{CHECKOUT_SESSION_ID}}",
        cancel_url=f"{frontend_url}/account/checkout/failure",
    )

    return CheckoutSessionResponse(checkout_url=checkout_url)
