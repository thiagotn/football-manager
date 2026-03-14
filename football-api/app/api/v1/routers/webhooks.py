"""
Webhook handler para eventos Stripe.

Endpoint: POST /api/v1/webhooks/payment

Eventos suportados:
  - checkout.session.completed → ativa plano pago
  - invoice.paid               → renova current_period_end
  - invoice.payment_failed     → past_due + grace period (7 dias)
  - customer.subscription.deleted → regride para free
  - customer.subscription.updated → atualiza plano via metadata

RNF-02: idempotência garantida pela tabela webhook_events (event_id UNIQUE).
"""

from datetime import datetime, timedelta, timezone

import stripe.error
import structlog
from fastapi import APIRouter, HTTPException, Request

from app.core.dependencies import DB
from app.db.repositories.subscription_repo import SubscriptionRepository
from app.services import billing

logger = structlog.get_logger()

router = APIRouter(prefix="/webhooks", tags=["webhooks"])

# Limite de groups no plano free (para cleanup ao cancelar)
_FREE_GROUPS_LIMIT = 1

# Dias de graça após falha de pagamento
_GRACE_PERIOD_DAYS = 7


async def _is_duplicate(db, event_id: str) -> bool:
    """Verifica se o event_id já foi processado (idempotência)."""
    from sqlalchemy import text
    result = await db.execute(
        text("SELECT 1 FROM webhook_events WHERE event_id = :eid"),
        {"eid": event_id},
    )
    return result.scalar_one_or_none() is not None


async def _mark_processed(db, event_id: str, event_type: str) -> None:
    """Registra o event_id como processado."""
    from sqlalchemy import text
    await db.execute(
        text(
            "INSERT INTO webhook_events (event_id, event_type) VALUES (:eid, :etype)"
            " ON CONFLICT (event_id) DO NOTHING"
        ),
        {"eid": event_id, "etype": event_type},
    )


@router.post("/payment", status_code=200)
async def handle_stripe_webhook(request: Request, db: DB):
    """
    Recebe e processa eventos do Stripe.

    A assinatura HMAC-SHA256 é verificada antes de processar qualquer evento.
    Eventos já processados (mesmo event_id) são ignorados silenciosamente.
    """
    payload = await request.body()
    signature = request.headers.get("stripe-signature", "")

    # 1. Verifica assinatura
    try:
        event = billing.verify_webhook_signature(payload, signature)
    except Exception as exc:
        logger.warning("webhook_invalid_signature", error=str(exc))
        raise HTTPException(status_code=400, detail="Invalid signature")

    event_id = event.get("id", "")
    event_type = event.get("type", "")
    data_obj = event.get("data", {}).get("object", {})

    log = logger.bind(event_id=event_id, event_type=event_type)

    # 2. Idempotência
    if await _is_duplicate(db, event_id):
        log.info("webhook_duplicate_skipped")
        return {"status": "already_processed"}

    sub_repo = SubscriptionRepository(db)

    # 3. Despacha por tipo de evento
    try:
        if event_type == "checkout.session.completed":
            await _handle_checkout_completed(sub_repo, data_obj, log)

        elif event_type == "invoice.paid":
            await _handle_invoice_paid(sub_repo, data_obj, log)

        elif event_type == "invoice.payment_failed":
            await _handle_payment_failed(sub_repo, data_obj, log)

        elif event_type == "customer.subscription.deleted":
            await _handle_subscription_deleted(sub_repo, data_obj, log)

        elif event_type == "customer.subscription.updated":
            await _handle_subscription_updated(sub_repo, data_obj, log)

        else:
            log.info("webhook_event_ignored")

    except Exception as exc:
        # Não raise — retorna 200 para evitar retentativas infinitas do Stripe.
        # O erro é logado para investigação.
        log.error("webhook_processing_error", error=str(exc))
        return {"status": "error_logged"}

    # 4. Marca como processado
    await _mark_processed(db, event_id, event_type)
    log.info("webhook_processed")
    return {"status": "ok"}


# ─── Handlers de eventos ──────────────────────────────────────────────────────


async def _handle_checkout_completed(sub_repo, session: dict, log) -> None:
    """
    checkout.session.completed → ativa o plano pago.

    O metadata contém player_id, plan e billing_cycle (injetados em
    billing_stripe.create_checkout_session).
    """
    metadata = session.get("metadata") or {}
    player_id = metadata.get("player_id")
    plan = metadata.get("plan")
    subscription_id = session.get("subscription")
    customer_id = session.get("customer")

    if not player_id or not plan:
        log.warning("checkout_completed_missing_metadata", metadata=metadata)
        return

    import asyncio
    import uuid

    import stripe as _stripe

    from app.core.config import get_settings

    try:
        pid = uuid.UUID(player_id)
    except ValueError:
        log.warning("checkout_completed_invalid_player_id", player_id=player_id)
        return

    # Busca current_period_end da Subscription (não está na sessão de checkout)
    billing_cycle = metadata.get("billing_cycle", "monthly")
    current_period_end = None
    if subscription_id:
        try:
            api_key = get_settings().stripe_secret_key

            def _fetch():
                return _stripe.Subscription.retrieve(subscription_id, api_key=api_key)

            stripe_sub = await asyncio.to_thread(_fetch)
            period_end_ts = getattr(stripe_sub, "current_period_end", None)
            if period_end_ts:
                current_period_end = datetime.fromtimestamp(period_end_ts, tz=timezone.utc)
        except Exception as exc:
            log.warning("checkout_completed_sub_fetch_failed", error=str(exc))

    # Fallback: calcula data estimada se o fetch falhou (ex: eventos sintéticos de testes)
    if current_period_end is None:
        days = 365 if billing_cycle == "yearly" else 30
        current_period_end = datetime.now(tz=timezone.utc) + timedelta(days=days)

    await sub_repo.update_plan(
        player_id=pid,
        plan=plan,
        status="active",
        gateway_customer_id=customer_id,
        gateway_sub_id=subscription_id,
        current_period_end=current_period_end,
        billing_cycle=billing_cycle,
    )
    log.info("checkout_completed_plan_activated", player_id=player_id, plan=plan)


async def _handle_invoice_paid(sub_repo, invoice: dict, log) -> None:
    """
    invoice.paid → renova current_period_end, garante status=active.
    """
    customer_id = invoice.get("customer")
    if not customer_id:
        return

    sub = await sub_repo.get_by_gateway_customer(customer_id)
    if not sub:
        log.warning("invoice_paid_customer_not_found", customer_id=customer_id)
        return

    # lines.data[0].period.end contém o fim do período renovado
    period_end: int | None = None
    lines = invoice.get("lines", {}).get("data", [])
    if lines:
        period_end = lines[0].get("period", {}).get("end")

    current_period_end = (
        datetime.fromtimestamp(period_end, tz=timezone.utc) if period_end else None
    )

    await sub_repo.update_plan(
        player_id=sub.player_id,
        plan=sub.plan,
        status="active",
        current_period_end=current_period_end,
    )
    log.info("invoice_paid_renewed", customer_id=customer_id)


async def _handle_payment_failed(sub_repo, invoice: dict, log) -> None:
    """
    invoice.payment_failed → past_due + grace_period_end = NOW() + 7 dias.
    """
    customer_id = invoice.get("customer")
    if not customer_id:
        return

    sub = await sub_repo.get_by_gateway_customer(customer_id)
    if not sub:
        log.warning("payment_failed_customer_not_found", customer_id=customer_id)
        return

    grace_period_end = datetime.now(tz=timezone.utc) + timedelta(days=_GRACE_PERIOD_DAYS)

    await sub_repo.update_plan(
        player_id=sub.player_id,
        plan=sub.plan,
        status="past_due",
        grace_period_end=grace_period_end,
    )
    log.info("payment_failed_grace_period_set", customer_id=customer_id, grace_end=grace_period_end.isoformat())


async def _handle_subscription_deleted(sub_repo, subscription: dict, log) -> None:
    """
    customer.subscription.deleted → regride para free, status=canceled.
    """
    customer_id = subscription.get("customer")
    if not customer_id:
        return

    sub = await sub_repo.get_by_gateway_customer(customer_id)
    if not sub:
        log.warning("subscription_deleted_customer_not_found", customer_id=customer_id)
        return

    await sub_repo.update_plan(
        player_id=sub.player_id,
        plan="free",
        status="canceled",
    )
    log.info("subscription_deleted_reverted_to_free", customer_id=customer_id)


async def _handle_subscription_updated(sub_repo, subscription: dict, log) -> None:
    """
    customer.subscription.updated → atualiza plano via metadata.plan.
    """
    customer_id = subscription.get("customer")
    metadata = subscription.get("metadata") or {}
    plan = metadata.get("plan")

    if not customer_id or not plan:
        log.info("subscription_updated_no_plan_in_metadata", metadata=metadata)
        return

    sub = await sub_repo.get_by_gateway_customer(customer_id)
    if not sub:
        log.warning("subscription_updated_customer_not_found", customer_id=customer_id)
        return

    await sub_repo.update_plan(
        player_id=sub.player_id,
        plan=plan,
        status="active",
    )
    log.info("subscription_updated_plan_changed", customer_id=customer_id, plan=plan)
