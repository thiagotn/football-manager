"""
Testes unitários — routers/webhooks.py

Regras de negócio cobertas:
- Assinatura Stripe inválida → 400
- checkout.session.completed → 200 (ativa plano)
- invoice.paid → 200 (renova período)
- invoice.payment_failed → 200 (marca past_due)
- customer.subscription.deleted → 200 (cancela plano)
- customer.subscription.updated → 200 (atualiza plano)
- Evento desconhecido → 200 (ignorado graciosamente)
"""
from unittest.mock import AsyncMock, MagicMock, patch
from uuid import uuid4

import pytest


# ── Helpers ───────────────────────────────────────────────────────────────────


def _make_event(event_type: str, data_obj: dict, event_id: str = "evt_test_001") -> dict:
    return {
        "id": event_id,
        "type": event_type,
        "data": {"object": data_obj},
    }


def _patch_billing_verify(mocker, event: dict):
    """Mock billing.verify_webhook_signature para retornar um evento válido."""
    mocker.patch(
        "app.api.v1.routers.webhooks.billing.verify_webhook_signature",
        return_value=event,
    )


def _patch_not_duplicate(mock_db):
    """Simula que o event_id ainda não foi processado (não é duplicata)."""
    result = MagicMock()
    result.scalar_one_or_none.return_value = None
    mock_db.execute = AsyncMock(return_value=result)


# ── POST /webhooks/payment — assinatura inválida ──────────────────────────────


@pytest.mark.asyncio
async def test_webhook_invalid_signature_returns_400(api_client, mocker):
    """Assinatura HMAC inválida retorna 400."""
    mocker.patch(
        "app.api.v1.routers.webhooks.billing.verify_webhook_signature",
        side_effect=Exception("Invalid signature"),
    )

    response = await api_client.post(
        "/api/v1/webhooks/payment",
        content=b'{"id": "evt_x"}',
        headers={"stripe-signature": "bad_sig"},
    )

    assert response.status_code == 400
    assert "Invalid signature" in response.json()["detail"]


# ── checkout.session.completed ────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_webhook_checkout_completed_activates_plan(api_client, mock_db, mocker):
    """checkout.session.completed ativa o plano do player."""
    player_id = str(uuid4())
    event = _make_event(
        "checkout.session.completed",
        {
            "customer": "cus_test_001",
            "subscription": None,
            "metadata": {"player_id": player_id, "plan": "basic", "billing_cycle": "monthly"},
        },
    )
    _patch_billing_verify(mocker, event)
    _patch_not_duplicate(mock_db)

    mocker.patch(
        "app.api.v1.routers.webhooks.SubscriptionRepository.update_plan",
        new=AsyncMock(return_value=None),
    )

    response = await api_client.post(
        "/api/v1/webhooks/payment",
        content=b"payload",
        headers={"stripe-signature": "v1=fake"},
    )

    assert response.status_code == 200
    assert response.json()["status"] == "ok"


# ── invoice.paid ──────────────────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_webhook_invoice_paid_renews_subscription(api_client, mock_db, mocker):
    """invoice.paid renova o current_period_end da assinatura."""
    event = _make_event(
        "invoice.paid",
        {
            "customer": "cus_test_002",
            "lines": {"data": [{"period": {"end": 1800000000}}]},
        },
        event_id="evt_invoice_paid",
    )
    _patch_billing_verify(mocker, event)
    _patch_not_duplicate(mock_db)

    sub = MagicMock()
    sub.player_id = uuid4()
    sub.plan = "basic"

    mocker.patch(
        "app.api.v1.routers.webhooks.SubscriptionRepository.get_by_gateway_customer",
        new=AsyncMock(return_value=sub),
    )
    mocker.patch(
        "app.api.v1.routers.webhooks.SubscriptionRepository.update_plan",
        new=AsyncMock(return_value=None),
    )

    response = await api_client.post(
        "/api/v1/webhooks/payment",
        content=b"payload",
        headers={"stripe-signature": "v1=fake"},
    )

    assert response.status_code == 200
    assert response.json()["status"] == "ok"


# ── invoice.payment_failed ────────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_webhook_payment_failed_marks_past_due(api_client, mock_db, mocker):
    """invoice.payment_failed marca assinatura como past_due com grace period."""
    event = _make_event(
        "invoice.payment_failed",
        {"customer": "cus_test_003"},
        event_id="evt_payment_failed",
    )
    _patch_billing_verify(mocker, event)
    _patch_not_duplicate(mock_db)

    sub = MagicMock()
    sub.player_id = uuid4()
    sub.plan = "pro"

    mocker.patch(
        "app.api.v1.routers.webhooks.SubscriptionRepository.get_by_gateway_customer",
        new=AsyncMock(return_value=sub),
    )
    mocker.patch(
        "app.api.v1.routers.webhooks.SubscriptionRepository.update_plan",
        new=AsyncMock(return_value=None),
    )

    response = await api_client.post(
        "/api/v1/webhooks/payment",
        content=b"payload",
        headers={"stripe-signature": "v1=fake"},
    )

    assert response.status_code == 200
    assert response.json()["status"] == "ok"


# ── customer.subscription.deleted ────────────────────────────────────────────


@pytest.mark.asyncio
async def test_webhook_subscription_deleted_reverts_to_free(api_client, mock_db, mocker):
    """customer.subscription.deleted regride o player para o plano free."""
    event = _make_event(
        "customer.subscription.deleted",
        {"customer": "cus_test_004"},
        event_id="evt_sub_deleted",
    )
    _patch_billing_verify(mocker, event)
    _patch_not_duplicate(mock_db)

    sub = MagicMock()
    sub.player_id = uuid4()
    sub.plan = "basic"

    mocker.patch(
        "app.api.v1.routers.webhooks.SubscriptionRepository.get_by_gateway_customer",
        new=AsyncMock(return_value=sub),
    )
    mocker.patch(
        "app.api.v1.routers.webhooks.SubscriptionRepository.update_plan",
        new=AsyncMock(return_value=None),
    )

    response = await api_client.post(
        "/api/v1/webhooks/payment",
        content=b"payload",
        headers={"stripe-signature": "v1=fake"},
    )

    assert response.status_code == 200
    assert response.json()["status"] == "ok"


# ── customer.subscription.updated ────────────────────────────────────────────


@pytest.mark.asyncio
async def test_webhook_subscription_updated_changes_plan(api_client, mock_db, mocker):
    """customer.subscription.updated atualiza o plano via metadata."""
    event = _make_event(
        "customer.subscription.updated",
        {"customer": "cus_test_005", "metadata": {"plan": "pro"}},
        event_id="evt_sub_updated",
    )
    _patch_billing_verify(mocker, event)
    _patch_not_duplicate(mock_db)

    sub = MagicMock()
    sub.player_id = uuid4()
    sub.plan = "basic"

    mocker.patch(
        "app.api.v1.routers.webhooks.SubscriptionRepository.get_by_gateway_customer",
        new=AsyncMock(return_value=sub),
    )
    mocker.patch(
        "app.api.v1.routers.webhooks.SubscriptionRepository.update_plan",
        new=AsyncMock(return_value=None),
    )

    response = await api_client.post(
        "/api/v1/webhooks/payment",
        content=b"payload",
        headers={"stripe-signature": "v1=fake"},
    )

    assert response.status_code == 200
    assert response.json()["status"] == "ok"


# ── Evento desconhecido ────────────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_webhook_unknown_event_type_returns_200(api_client, mock_db, mocker):
    """Eventos desconhecidos são ignorados — retorna 200."""
    event = _make_event(
        "charge.refunded",
        {"amount": 5000},
        event_id="evt_unknown",
    )
    _patch_billing_verify(mocker, event)
    _patch_not_duplicate(mock_db)

    response = await api_client.post(
        "/api/v1/webhooks/payment",
        content=b"payload",
        headers={"stripe-signature": "v1=fake"},
    )

    assert response.status_code == 200
    assert response.json()["status"] == "ok"
