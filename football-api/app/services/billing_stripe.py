"""
Implementação Stripe do serviço de billing.

Não importar diretamente nos routers — usar app.services.billing.
As chamadas ao SDK do Stripe são síncronas e executadas via asyncio.to_thread
para não bloquear o event loop.
"""

import asyncio

import stripe
import stripe.error

from app.core.config import get_settings


def _api_key() -> str:
    return get_settings().stripe_secret_key


async def get_or_create_customer(player_id: str, name: str, phone: str) -> str:
    """Cria um Stripe Customer e retorna o customer_id (cus_xxx)."""

    def _create() -> stripe.Customer:
        return stripe.Customer.create(
            api_key=_api_key(),
            name=name,
            phone=phone,
            metadata={"player_id": str(player_id)},
        )

    customer = await asyncio.to_thread(_create)
    return customer.id


async def create_checkout_session(
    customer_id: str,
    player_id: str,
    plan: str,
    billing_cycle: str,
    success_url: str,
    cancel_url: str,
) -> str:
    """Cria uma Stripe Checkout Session e retorna a URL de redirect."""
    settings = get_settings()
    price_id = settings.get_price_id(plan, billing_cycle)

    def _create() -> stripe.checkout.Session:
        return stripe.checkout.Session.create(
            api_key=_api_key(),
            customer=customer_id,
            mode="subscription",
            line_items=[{"price": price_id, "quantity": 1}],
            success_url=success_url,
            cancel_url=cancel_url,
            # metadata é repassado no evento checkout.session.completed
            # e usado pelo webhook handler para identificar o player
            metadata={
                "player_id": str(player_id),
                "plan": plan,
                "billing_cycle": billing_cycle,
            },
        )

    session = await asyncio.to_thread(_create)
    return session.url


async def cancel_subscription(gateway_sub_id: str) -> None:
    """Cancela imediatamente uma assinatura no Stripe."""

    def _cancel() -> None:
        stripe.Subscription.cancel(gateway_sub_id, api_key=_api_key())

    await asyncio.to_thread(_cancel)


def verify_webhook_signature(payload: bytes, signature: str) -> dict:
    """
    Verifica a assinatura HMAC-SHA256 do webhook Stripe e retorna o evento.

    Lança stripe.error.SignatureVerificationError se inválido.
    Operação síncrona (sem I/O de rede) — não precisa de to_thread.
    """
    settings = get_settings()
    event = stripe.Webhook.construct_event(
        payload,
        signature,
        settings.stripe_webhook_secret,
    )
    return dict(event)
