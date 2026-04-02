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


# ── Evento duplicado ──────────────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_webhook_duplicate_event_returns_already_processed(api_client, mock_db, mocker):
    """Evento já processado retorna 'already_processed' sem reprocessar."""
    event = _make_event("invoice.paid", {}, event_id="evt_dup_001")
    _patch_billing_verify(mocker, event)

    # Simula evento duplicado (já existe no banco)
    dup_result = MagicMock()
    dup_result.scalar_one_or_none.return_value = MagicMock()
    mock_db.execute = AsyncMock(return_value=dup_result)

    response = await api_client.post(
        "/api/v1/webhooks/payment",
        content=b"payload",
        headers={"stripe-signature": "v1=fake"},
    )

    assert response.status_code == 200
    assert response.json()["status"] == "already_processed"


# ── Erros durante processamento ───────────────────────────────────────────────


@pytest.mark.asyncio
async def test_webhook_processing_exception_returns_error_logged(api_client, mock_db, mocker):
    """Exceção durante processamento retorna 'error_logged' (evita retentativas do Stripe)."""
    event = _make_event(
        "invoice.paid",
        {"customer": "cus_error"},
        event_id="evt_error_001",
    )
    _patch_billing_verify(mocker, event)
    _patch_not_duplicate(mock_db)

    # Lança exceção dentro do handler
    mocker.patch(
        "app.api.v1.routers.webhooks.SubscriptionRepository.get_by_gateway_customer",
        new=AsyncMock(side_effect=Exception("DB error")),
    )

    response = await api_client.post(
        "/api/v1/webhooks/payment",
        content=b"payload",
        headers={"stripe-signature": "v1=fake"},
    )

    assert response.status_code == 200
    assert response.json()["status"] == "error_logged"


# ── checkout.session.completed — edge cases ───────────────────────────────────


@pytest.mark.asyncio
async def test_webhook_checkout_completed_missing_metadata_returns_ok(api_client, mock_db, mocker):
    """checkout.session.completed sem metadata ignora graciosamente."""
    event = _make_event(
        "checkout.session.completed",
        {"customer": "cus_nometa", "subscription": None, "metadata": {}},
        event_id="evt_checkout_nometa",
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


@pytest.mark.asyncio
async def test_webhook_checkout_completed_invalid_uuid_returns_ok(api_client, mock_db, mocker):
    """checkout.session.completed com player_id UUID inválido ignora graciosamente."""
    event = _make_event(
        "checkout.session.completed",
        {
            "customer": "cus_baduuid",
            "subscription": None,
            "metadata": {"player_id": "not-a-uuid", "plan": "basic"},
        },
        event_id="evt_checkout_baduuid",
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


# ── invoice.paid — edge cases ─────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_webhook_invoice_paid_missing_customer_returns_ok(api_client, mock_db, mocker):
    """invoice.paid sem customer_id ignora graciosamente."""
    event = _make_event(
        "invoice.paid",
        {},  # sem campo 'customer'
        event_id="evt_paid_nocust",
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


@pytest.mark.asyncio
async def test_webhook_invoice_paid_customer_not_found_returns_ok(api_client, mock_db, mocker):
    """invoice.paid com customer não encontrado na base retorna 200."""
    event = _make_event(
        "invoice.paid",
        {"customer": "cus_ghost"},
        event_id="evt_paid_ghost",
    )
    _patch_billing_verify(mocker, event)
    _patch_not_duplicate(mock_db)

    mocker.patch(
        "app.api.v1.routers.webhooks.SubscriptionRepository.get_by_gateway_customer",
        new=AsyncMock(return_value=None),
    )

    response = await api_client.post(
        "/api/v1/webhooks/payment",
        content=b"payload",
        headers={"stripe-signature": "v1=fake"},
    )

    assert response.status_code == 200
    assert response.json()["status"] == "ok"


# ── invoice.payment_failed — edge cases ──────────────────────────────────────


@pytest.mark.asyncio
async def test_webhook_payment_failed_missing_customer_returns_ok(api_client, mock_db, mocker):
    """invoice.payment_failed sem customer ignora graciosamente."""
    event = _make_event(
        "invoice.payment_failed",
        {},
        event_id="evt_failed_nocust",
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


@pytest.mark.asyncio
async def test_webhook_payment_failed_customer_not_found_returns_ok(api_client, mock_db, mocker):
    """invoice.payment_failed com customer sem subscription na base retorna 200."""
    event = _make_event(
        "invoice.payment_failed",
        {"customer": "cus_ghost_fail"},
        event_id="evt_failed_ghost",
    )
    _patch_billing_verify(mocker, event)
    _patch_not_duplicate(mock_db)

    mocker.patch(
        "app.api.v1.routers.webhooks.SubscriptionRepository.get_by_gateway_customer",
        new=AsyncMock(return_value=None),
    )

    response = await api_client.post(
        "/api/v1/webhooks/payment",
        content=b"payload",
        headers={"stripe-signature": "v1=fake"},
    )

    assert response.status_code == 200
    assert response.json()["status"] == "ok"


# ── customer.subscription.deleted — edge cases ───────────────────────────────


@pytest.mark.asyncio
async def test_webhook_subscription_deleted_missing_customer_returns_ok(api_client, mock_db, mocker):
    """customer.subscription.deleted sem customer ignora graciosamente."""
    event = _make_event(
        "customer.subscription.deleted",
        {},
        event_id="evt_del_nocust",
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


@pytest.mark.asyncio
async def test_webhook_subscription_deleted_not_found_returns_ok(api_client, mock_db, mocker):
    """customer.subscription.deleted com customer não encontrado retorna 200."""
    event = _make_event(
        "customer.subscription.deleted",
        {"customer": "cus_ghost_del"},
        event_id="evt_del_ghost",
    )
    _patch_billing_verify(mocker, event)
    _patch_not_duplicate(mock_db)

    mocker.patch(
        "app.api.v1.routers.webhooks.SubscriptionRepository.get_by_gateway_customer",
        new=AsyncMock(return_value=None),
    )

    response = await api_client.post(
        "/api/v1/webhooks/payment",
        content=b"payload",
        headers={"stripe-signature": "v1=fake"},
    )

    assert response.status_code == 200
    assert response.json()["status"] == "ok"


# ── customer.subscription.updated — edge cases ───────────────────────────────


@pytest.mark.asyncio
async def test_webhook_subscription_updated_no_plan_in_metadata_returns_ok(api_client, mock_db, mocker):
    """customer.subscription.updated sem plan no metadata ignora graciosamente."""
    event = _make_event(
        "customer.subscription.updated",
        {"customer": "cus_upd", "metadata": {}},  # sem 'plan'
        event_id="evt_upd_noplan",
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


@pytest.mark.asyncio
async def test_webhook_subscription_updated_customer_not_found_returns_ok(api_client, mock_db, mocker):
    """customer.subscription.updated com customer não encontrado retorna 200."""
    event = _make_event(
        "customer.subscription.updated",
        {"customer": "cus_ghost_upd", "metadata": {"plan": "pro"}},
        event_id="evt_upd_ghost",
    )
    _patch_billing_verify(mocker, event)
    _patch_not_duplicate(mock_db)

    mocker.patch(
        "app.api.v1.routers.webhooks.SubscriptionRepository.get_by_gateway_customer",
        new=AsyncMock(return_value=None),
    )

    response = await api_client.post(
        "/api/v1/webhooks/payment",
        content=b"payload",
        headers={"stripe-signature": "v1=fake"},
    )

    assert response.status_code == 200
    assert response.json()["status"] == "ok"
