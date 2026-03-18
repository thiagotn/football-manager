"""
Testes unitários — routers/subscriptions.py

Regras de negócio cobertas:
- GET /subscriptions/me admin → limites None (isento)
- GET /subscriptions/me player free → groups_limit=1, members_limit=30
- POST /subscriptions plano inválido → 400
- POST /subscriptions billing_cycle inválido → 400
"""
from unittest.mock import AsyncMock, MagicMock

import pytest


# ── Helpers ───────────────────────────────────────────────────────────────────


def _make_subscription(plan: str = "free") -> MagicMock:
    sub = MagicMock()
    sub.plan = plan
    sub.status = "active"
    sub.gateway_customer_id = None
    sub.gateway_sub_id = None
    sub.current_period_end = None
    sub.grace_period_end = None
    return sub


# ── GET /subscriptions/me ─────────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_get_subscription_admin_returns_no_limits(admin_client, mocker):
    """Super admin é isento de limites de plano."""
    sub = _make_subscription("pro")
    mocker.patch(
        "app.api.v1.routers.subscriptions.SubscriptionRepository.get_or_create",
        new=AsyncMock(return_value=sub),
    )

    response = await admin_client.get("/api/v1/subscriptions/me")

    assert response.status_code == 200
    data = response.json()
    assert data["groups_limit"] is None
    assert data["members_limit"] is None
    assert data["groups_used"] == 0


@pytest.mark.asyncio
async def test_get_subscription_free_player_returns_limits(api_client, mocker):
    """Player com plano free tem groups_limit=1 e members_limit=30."""
    sub = _make_subscription("free")
    mocker.patch(
        "app.api.v1.routers.subscriptions.SubscriptionRepository.get_or_create",
        new=AsyncMock(return_value=sub),
    )
    mocker.patch(
        "app.api.v1.routers.subscriptions.SubscriptionRepository.count_admin_groups",
        new=AsyncMock(return_value=0),
    )

    response = await api_client.get("/api/v1/subscriptions/me")

    assert response.status_code == 200
    data = response.json()
    assert data["plan"] == "free"
    assert data["groups_limit"] == 1
    assert data["members_limit"] == 30
    assert data["groups_used"] == 0


# ── POST /subscriptions ───────────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_create_checkout_invalid_plan_returns_400(api_client):
    """Plano inexistente retorna 400."""
    response = await api_client.post(
        "/api/v1/subscriptions",
        json={"plan": "ultra", "billing_cycle": "monthly"},
    )

    assert response.status_code == 400


@pytest.mark.asyncio
async def test_create_checkout_invalid_billing_cycle_returns_400(api_client):
    """billing_cycle fora de monthly/yearly retorna 400."""
    response = await api_client.post(
        "/api/v1/subscriptions",
        json={"plan": "basic", "billing_cycle": "weekly"},
    )

    assert response.status_code == 400
