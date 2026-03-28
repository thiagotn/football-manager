"""
Testes unitários — routers/finance.py

Regras de negócio cobertas:
- GET /groups/{id}/finance/periods grupo não encontrado → 404
- GET /groups/{id}/finance/periods não-membro → 403
- PATCH /finance/payments/{id} pagamento não encontrado → 404
- PATCH /finance/payments/{id} status=paid sem payment_type → 422
"""
from unittest.mock import AsyncMock, MagicMock
from uuid import uuid4

import pytest


# ── GET /groups/{id}/finance/periods ─────────────────────────────────────────


@pytest.mark.asyncio
async def test_list_periods_group_not_found_returns_404(api_client, mocker):
    mocker.patch(
        "app.api.v1.routers.finance.GroupRepository.get",
        new=AsyncMock(return_value=None),
    )

    response = await api_client.get(f"/api/v1/groups/{uuid4()}/finance/periods")

    assert response.status_code == 404


@pytest.mark.asyncio
async def test_list_periods_non_member_returns_403(api_client, mocker):
    mocker.patch(
        "app.api.v1.routers.finance.GroupRepository.get",
        new=AsyncMock(return_value=MagicMock()),
    )
    mocker.patch(
        "app.api.v1.routers.finance.GroupRepository.get_member",
        new=AsyncMock(return_value=None),  # não é membro
    )

    response = await api_client.get(f"/api/v1/groups/{uuid4()}/finance/periods")

    assert response.status_code == 403


# ── PATCH /finance/payments/{id} ─────────────────────────────────────────────


@pytest.mark.asyncio
async def test_update_payment_not_found_returns_404(api_client, mocker):
    mocker.patch(
        "app.api.v1.routers.finance.FinanceRepository.get_payment",
        new=AsyncMock(return_value=None),
    )

    response = await api_client.patch(
        f"/api/v1/finance/payments/{uuid4()}",
        json={"status": "paid", "payment_type": "monthly"},
    )

    assert response.status_code == 404


@pytest.mark.asyncio
async def test_update_payment_paid_without_payment_type_returns_422(admin_client, mock_db, mocker):
    """Marcar como pago sem informar payment_type retorna 422."""
    payment = MagicMock()
    payment.period_id = uuid4()

    period = MagicMock()
    period.group_id = uuid4()

    mock_db.get = AsyncMock(return_value=period)

    mocker.patch(
        "app.api.v1.routers.finance.FinanceRepository.get_payment",
        new=AsyncMock(return_value=payment),
    )
    mocker.patch(
        "app.api.v1.routers.finance.GroupRepository.get",
        new=AsyncMock(return_value=MagicMock()),
    )

    response = await admin_client.patch(
        f"/api/v1/finance/payments/{uuid4()}",
        json={"status": "paid"},  # payment_type ausente
    )

    assert response.status_code == 422


# ── GET /groups/{id}/finance/periods — happy path ────────────────────────────


@pytest.mark.asyncio
async def test_list_periods_member_returns_200(api_client, mocker):
    """Membro do grupo pode listar os períodos financeiros."""
    group_id = uuid4()

    mocker.patch(
        "app.api.v1.routers.finance.GroupRepository.get",
        new=AsyncMock(return_value=MagicMock()),
    )
    member = MagicMock()
    mocker.patch(
        "app.api.v1.routers.finance.GroupRepository.get_member",
        new=AsyncMock(return_value=member),
    )

    period1 = MagicMock()
    period1.id = uuid4()
    period1.year = 2026
    period1.month = 3

    mocker.patch(
        "app.api.v1.routers.finance.FinanceRepository.list_periods",
        new=AsyncMock(return_value=[period1]),
    )

    response = await api_client.get(f"/api/v1/groups/{group_id}/finance/periods")

    assert response.status_code == 200
    data = response.json()
    assert len(data) == 1
    assert data[0]["year"] == 2026
    assert data[0]["month"] == 3


# ── GET /groups/{id}/finance/periods/{year}/{month} — happy path ─────────────


@pytest.mark.asyncio
async def test_get_period_member_returns_200(api_client, mocker):
    """Membro do grupo pode ver os dados financeiros de um período."""
    group_id = uuid4()

    mocker.patch(
        "app.api.v1.routers.finance.GroupRepository.get",
        new=AsyncMock(return_value=MagicMock()),
    )
    member = MagicMock()
    mocker.patch(
        "app.api.v1.routers.finance.GroupRepository.get_member",
        new=AsyncMock(return_value=member),
    )

    period = MagicMock()
    period.id = uuid4()
    period.year = 2026
    period.month = 1  # past month, not current
    period.payments = []

    mocker.patch(
        "app.api.v1.routers.finance.FinanceRepository.get_or_create_period",
        new=AsyncMock(return_value=period),
    )
    mocker.patch(
        "app.api.v1.routers.finance.FinanceRepository.get_period_with_payments",
        new=AsyncMock(return_value=period),
    )

    response = await api_client.get(f"/api/v1/groups/{group_id}/finance/periods/2026/1")

    assert response.status_code == 200
    data = response.json()
    assert data["year"] == 2026
    assert data["month"] == 1


# ── PATCH /finance/payments/{id} — happy path ────────────────────────────────


@pytest.mark.asyncio
async def test_update_payment_mark_paid_returns_200(admin_client, mock_db, mocker):
    """Admin do grupo pode marcar pagamento como pago."""
    from decimal import Decimal

    payment_id = uuid4()
    payment = MagicMock()
    payment.id = payment_id
    payment.period_id = uuid4()
    payment.player_id = uuid4()
    payment.player_name = "Jogador"
    payment.payment_type = "monthly"
    payment.amount_due = 5000
    payment.status = "paid"
    payment.paid_at = "2026-03-01T00:00:00"

    period = MagicMock()
    period.group_id = uuid4()

    group = MagicMock()
    group.monthly_amount = Decimal("50.00")
    group.per_match_amount = None

    mock_db.get = AsyncMock(return_value=period)

    mocker.patch(
        "app.api.v1.routers.finance.FinanceRepository.get_payment",
        new=AsyncMock(return_value=payment),
    )
    mocker.patch(
        "app.api.v1.routers.finance.GroupRepository.get",
        new=AsyncMock(return_value=group),
    )
    mocker.patch(
        "app.api.v1.routers.finance.FinanceRepository.mark_paid",
        new=AsyncMock(return_value=payment),
    )

    response = await admin_client.patch(
        f"/api/v1/finance/payments/{payment_id}",
        json={"status": "paid", "payment_type": "monthly"},
    )

    assert response.status_code == 200
    assert response.json()["status"] == "paid"
