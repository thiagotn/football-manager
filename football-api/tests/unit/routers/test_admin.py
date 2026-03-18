"""
Testes unitários — routers/admin.py

Regras de negócio cobertas:
- Todos os endpoints /admin/** exigem super admin → 403 para players comuns
- PATCH /admin/subscriptions/{id} assinatura não encontrada → 404
- POST /admin/subscriptions/{id}/cancel não encontrada → 404
- POST /admin/subscriptions/{id}/cancel já cancelada/free → 400
"""
from unittest.mock import AsyncMock, MagicMock
from uuid import uuid4

import pytest


# ── Helpers ───────────────────────────────────────────────────────────────────


def _make_sub(plan: str = "basic", status: str = "active") -> MagicMock:
    sub = MagicMock()
    sub.plan = plan
    sub.status = status
    sub.gateway_sub_id = None
    return sub


# ── Acesso negado (player comum) ──────────────────────────────────────────────


@pytest.mark.asyncio
async def test_admin_stats_non_admin_returns_403(api_client):
    response = await api_client.get("/api/v1/admin/stats")
    assert response.status_code == 403


@pytest.mark.asyncio
async def test_admin_matches_non_admin_returns_403(api_client):
    response = await api_client.get("/api/v1/admin/matches")
    assert response.status_code == 403


@pytest.mark.asyncio
async def test_admin_groups_non_admin_returns_403(api_client):
    response = await api_client.get("/api/v1/admin/groups")
    assert response.status_code == 403


@pytest.mark.asyncio
async def test_admin_subscription_summary_non_admin_returns_403(api_client):
    response = await api_client.get("/api/v1/admin/subscriptions/summary")
    assert response.status_code == 403


@pytest.mark.asyncio
async def test_admin_subscriptions_non_admin_returns_403(api_client):
    response = await api_client.get("/api/v1/admin/subscriptions")
    assert response.status_code == 403


@pytest.mark.asyncio
async def test_admin_players_non_admin_returns_403(api_client):
    response = await api_client.get("/api/v1/admin/players")
    assert response.status_code == 403


@pytest.mark.asyncio
async def test_admin_update_subscription_non_admin_returns_403(api_client):
    response = await api_client.patch(
        f"/api/v1/admin/subscriptions/{uuid4()}",
        json={"plan": "basic"},
    )
    assert response.status_code == 403


@pytest.mark.asyncio
async def test_admin_cancel_subscription_non_admin_returns_403(api_client):
    response = await api_client.post(f"/api/v1/admin/subscriptions/{uuid4()}/cancel")
    assert response.status_code == 403


# ── Lógica de negócio (super admin) ──────────────────────────────────────────


@pytest.mark.asyncio
async def test_admin_update_subscription_not_found_returns_404(admin_client, mocker):
    mocker.patch(
        "app.api.v1.routers.admin.SubscriptionRepository.get_by_player",
        new=AsyncMock(return_value=None),
    )

    response = await admin_client.patch(
        f"/api/v1/admin/subscriptions/{uuid4()}",
        json={"plan": "basic"},
    )

    assert response.status_code == 404


@pytest.mark.asyncio
async def test_admin_cancel_subscription_not_found_returns_404(admin_client, mocker):
    mocker.patch(
        "app.api.v1.routers.admin.SubscriptionRepository.get_by_player",
        new=AsyncMock(return_value=None),
    )

    response = await admin_client.post(f"/api/v1/admin/subscriptions/{uuid4()}/cancel")

    assert response.status_code == 404


@pytest.mark.asyncio
async def test_admin_cancel_subscription_already_canceled_returns_400(admin_client, mocker):
    sub = _make_sub(plan="basic", status="canceled")
    mocker.patch(
        "app.api.v1.routers.admin.SubscriptionRepository.get_by_player",
        new=AsyncMock(return_value=sub),
    )

    response = await admin_client.post(f"/api/v1/admin/subscriptions/{uuid4()}/cancel")

    assert response.status_code == 400


@pytest.mark.asyncio
async def test_admin_cancel_subscription_free_plan_returns_400(admin_client, mocker):
    sub = _make_sub(plan="free", status="active")
    mocker.patch(
        "app.api.v1.routers.admin.SubscriptionRepository.get_by_player",
        new=AsyncMock(return_value=sub),
    )

    response = await admin_client.post(f"/api/v1/admin/subscriptions/{uuid4()}/cancel")

    assert response.status_code == 400
