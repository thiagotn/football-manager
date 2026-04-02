"""
Testes unitários — app/services/billing_stripe.py

Regras de negócio cobertas:
- get_or_create_customer delega para asyncio.to_thread e retorna customer_id
- create_checkout_session delega para asyncio.to_thread e retorna URL
- cancel_subscription delega para asyncio.to_thread
- verify_webhook_signature retorna evento quando assinatura válida
- verify_webhook_signature propaga exceção quando assinatura inválida
"""
from unittest.mock import AsyncMock, MagicMock, patch
from uuid import uuid4

import pytest
import stripe.error

from app.services import billing_stripe


# ── get_or_create_customer ────────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_get_or_create_customer_returns_customer_id(mocker):
    """Cria customer no Stripe e retorna o ID."""
    fake_customer = MagicMock()
    fake_customer.id = "cus_test_abc123"

    mock_thread = mocker.patch(
        "app.services.billing_stripe.asyncio.to_thread",
        new=AsyncMock(return_value=fake_customer),
    )
    mocker.patch(
        "app.services.billing_stripe.get_settings",
        return_value=MagicMock(stripe_secret_key="sk_test_fake"),
    )

    result = await billing_stripe.get_or_create_customer(
        player_id=str(uuid4()),
        name="João Silva",
        phone="+5511999990001",
    )

    assert result == "cus_test_abc123"
    mock_thread.assert_called_once()


# ── create_checkout_session ───────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_create_checkout_session_returns_url(mocker):
    """Cria checkout session e retorna a URL de redirect."""
    fake_session = MagicMock()
    fake_session.url = "https://checkout.stripe.com/pay/cs_test_abc"

    mock_settings = MagicMock()
    mock_settings.stripe_secret_key = "sk_test_fake"
    mock_settings.get_price_id.return_value = "price_test_basic_monthly"

    mocker.patch(
        "app.services.billing_stripe.get_settings",
        return_value=mock_settings,
    )
    mocker.patch(
        "app.services.billing_stripe.asyncio.to_thread",
        new=AsyncMock(return_value=fake_session),
    )

    result = await billing_stripe.create_checkout_session(
        customer_id="cus_test_001",
        player_id=str(uuid4()),
        plan="basic",
        billing_cycle="monthly",
        success_url="https://rachao.app/account/checkout?success=1",
        cancel_url="https://rachao.app/account/checkout?cancel=1",
    )

    assert result == "https://checkout.stripe.com/pay/cs_test_abc"


# ── cancel_subscription ───────────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_cancel_subscription_calls_stripe(mocker):
    """Cancela assinatura delegando para asyncio.to_thread."""
    mock_thread = mocker.patch(
        "app.services.billing_stripe.asyncio.to_thread",
        new=AsyncMock(return_value=None),
    )
    mocker.patch(
        "app.services.billing_stripe.get_settings",
        return_value=MagicMock(stripe_secret_key="sk_test_fake"),
    )

    await billing_stripe.cancel_subscription("sub_test_001")

    mock_thread.assert_called_once()


# ── verify_webhook_signature ──────────────────────────────────────────────────


def test_verify_webhook_signature_returns_event(mocker):
    """Assinatura válida retorna o evento como dict."""
    fake_event = {
        "id": "evt_test_001",
        "type": "invoice.paid",
        "data": {"object": {"customer": "cus_test"}},
    }
    mocker.patch(
        "app.services.billing_stripe.stripe.Webhook.construct_event",
        return_value=fake_event,
    )
    mocker.patch(
        "app.services.billing_stripe.get_settings",
        return_value=MagicMock(stripe_webhook_secret="whsec_fake"),
    )

    result = billing_stripe.verify_webhook_signature(
        payload=b'{"id": "evt_test_001"}',
        signature="v1=fakesignature",
    )

    assert result["id"] == "evt_test_001"
    assert result["type"] == "invoice.paid"


def test_verify_webhook_signature_raises_on_invalid(mocker):
    """Assinatura inválida propaga SignatureVerificationError."""
    mocker.patch(
        "app.services.billing_stripe.stripe.Webhook.construct_event",
        side_effect=stripe.error.SignatureVerificationError("Invalid sig", "bad_sig"),
    )
    mocker.patch(
        "app.services.billing_stripe.get_settings",
        return_value=MagicMock(stripe_webhook_secret="whsec_fake"),
    )

    with pytest.raises(stripe.error.SignatureVerificationError):
        billing_stripe.verify_webhook_signature(
            payload=b"payload",
            signature="v1=badsig",
        )
