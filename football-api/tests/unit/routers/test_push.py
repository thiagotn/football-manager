"""
Testes unitários — routers/push.py

Regras de negócio cobertas:
- GET /push/vapid-public-key → 200 com chave pública
- POST /push/subscribe autenticado → 201
- POST /push/subscribe sem autenticação → 401
- DELETE /push/subscribe autenticado → 204
- DELETE /push/subscribe sem autenticação → 401
"""
from unittest.mock import AsyncMock, MagicMock

import pytest
from httpx import ASGITransport, AsyncClient

from app.core.dependencies import get_db
from app.main import app


# ── GET /push/vapid-public-key ────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_get_vapid_public_key_returns_200(api_client, mocker):
    """Endpoint retorna a chave pública VAPID configurada."""
    mock_settings = MagicMock()
    mock_settings.vapid_public_key = "BFakeVapidPublicKey=="
    mocker.patch(
        "app.api.v1.routers.push.get_settings",
        return_value=mock_settings,
    )

    response = await api_client.get("/api/v1/push/vapid-public-key")

    assert response.status_code == 200
    assert response.json()["public_key"] == "BFakeVapidPublicKey=="


# ── POST /push/subscribe ──────────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_subscribe_authenticated_returns_201(api_client, mock_db, mocker):
    """Jogador autenticado pode criar ou atualizar uma push subscription."""
    # Simula que não existe subscription existente (insert path)
    mock_result = MagicMock()
    mock_result.scalar_one_or_none.return_value = None
    mock_db.execute = AsyncMock(return_value=mock_result)

    response = await api_client.post(
        "/api/v1/push/subscribe",
        json={
            "endpoint": "https://fcm.googleapis.com/test-endpoint",
            "keys": {"p256dh": "BNc...", "auth": "aKs..."},
            "user_agent": "Mozilla/5.0",
        },
    )

    assert response.status_code == 201
    assert response.json()["status"] == "subscribed"


@pytest.mark.asyncio
async def test_subscribe_upsert_existing_returns_201(api_client, mock_db):
    """Quando endpoint já existe, atualiza as chaves e retorna 201."""
    existing_sub = MagicMock()
    mock_result = MagicMock()
    mock_result.scalar_one_or_none.return_value = existing_sub
    mock_db.execute = AsyncMock(return_value=mock_result)

    response = await api_client.post(
        "/api/v1/push/subscribe",
        json={
            "endpoint": "https://fcm.googleapis.com/existing-endpoint",
            "keys": {"p256dh": "BNew...", "auth": "aNew..."},
        },
    )

    assert response.status_code == 201
    assert response.json()["status"] == "subscribed"


@pytest.mark.asyncio
async def test_subscribe_unauthenticated_returns_401(mock_db):
    """Sem autenticação a rota retorna 401."""
    app.dependency_overrides[get_db] = lambda: mock_db
    try:
        async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
            response = await client.post(
                "/api/v1/push/subscribe",
                json={
                    "endpoint": "https://fcm.googleapis.com/test",
                    "keys": {"p256dh": "B...", "auth": "a..."},
                },
            )
    finally:
        app.dependency_overrides.clear()

    assert response.status_code == 401


# ── DELETE /push/subscribe ────────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_unsubscribe_authenticated_returns_204(api_client, mock_db):
    """Jogador autenticado pode remover todas as suas subscriptions."""
    mock_result = MagicMock()
    mock_db.execute = AsyncMock(return_value=mock_result)

    response = await api_client.delete("/api/v1/push/subscribe")

    assert response.status_code == 204


@pytest.mark.asyncio
async def test_unsubscribe_unauthenticated_returns_401(mock_db):
    """Sem autenticação a rota de remoção retorna 401."""
    app.dependency_overrides[get_db] = lambda: mock_db
    try:
        async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
            response = await client.delete("/api/v1/push/subscribe")
    finally:
        app.dependency_overrides.clear()

    assert response.status_code == 401
