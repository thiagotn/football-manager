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
