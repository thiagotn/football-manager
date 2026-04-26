"""
Testes unitários — routers/mcp_tokens.py

Regras cobertas:
- POST /mcp-tokens com expiração 24h → 201 + token plaintext
- POST /mcp-tokens com expiração 7d → 201 + expires_at não-nulo
- POST /mcp-tokens sem expiração → 201 + expires_at null
- GET /mcp-tokens → lista sem campo token
- DELETE /mcp-tokens/{id} token inexistente → 404
- DELETE /mcp-tokens/{id} token de outro jogador → 403
- DELETE /mcp-tokens/{id} token próprio → 204
- Endpoints sem autenticação → 401
"""
from datetime import datetime, timezone
from unittest.mock import AsyncMock, MagicMock
from uuid import uuid4

import pytest


def _make_db_token(player_id=None, expires_at=None, revoked_at=None) -> MagicMock:
    t = MagicMock()
    t.id = uuid4()
    t.player_id = player_id or uuid4()
    t.name = "Claude Desktop"
    t.token_hash = "a" * 64
    t.token_prefix = "rachao_a1b2c3"
    t.expires_at = expires_at
    t.created_at = datetime.now(timezone.utc)
    t.last_used_at = None
    t.revoked_at = revoked_at
    return t


# ── POST /mcp-tokens ──────────────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_create_token_24h_returns_201_with_plaintext(api_client):
    response = await api_client.post(
        "/api/v1/mcp-tokens",
        json={"name": "Claude Desktop", "expires_in": "24h"},
    )

    assert response.status_code == 201
    data = response.json()
    assert "token" in data
    assert data["token"].startswith("rachao_")
    assert data["expires_at"] is not None
    assert data["name"] == "Claude Desktop"


@pytest.mark.asyncio
async def test_create_token_7d_returns_201(api_client):
    response = await api_client.post(
        "/api/v1/mcp-tokens",
        json={"name": "VS Code", "expires_in": "7d"},
    )

    assert response.status_code == 201
    data = response.json()
    assert "token" in data
    assert data["expires_at"] is not None


@pytest.mark.asyncio
async def test_create_token_no_expiration_returns_null_expires(api_client):
    response = await api_client.post(
        "/api/v1/mcp-tokens",
        json={"name": "Permanente", "expires_in": None},
    )

    assert response.status_code == 201
    data = response.json()
    assert "token" in data
    assert data["expires_at"] is None


@pytest.mark.asyncio
async def test_create_token_unauthenticated_returns_401(anon_client):
    response = await anon_client.post(
        "/api/v1/mcp-tokens",
        json={"name": "Test", "expires_in": None},
    )

    assert response.status_code == 401


# ── GET /mcp-tokens ───────────────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_list_tokens_does_not_expose_plaintext(api_client, mock_db, player_user):
    db_token = _make_db_token(player_id=player_user.id)

    scalars_mock = MagicMock()
    scalars_mock.all.return_value = [db_token]
    result_mock = MagicMock()
    result_mock.scalars.return_value = scalars_mock
    mock_db.execute = AsyncMock(return_value=result_mock)

    response = await api_client.get("/api/v1/mcp-tokens")

    assert response.status_code == 200
    data = response.json()
    assert isinstance(data, list)
    assert len(data) == 1
    assert "token" not in data[0]
    assert "token_prefix" in data[0]
    assert "is_expired" in data[0]


@pytest.mark.asyncio
async def test_list_tokens_unauthenticated_returns_401(anon_client):
    response = await anon_client.get("/api/v1/mcp-tokens")

    assert response.status_code == 401


# ── DELETE /mcp-tokens/{id} ───────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_revoke_nonexistent_token_returns_404(api_client, mock_db):
    result_mock = MagicMock()
    result_mock.scalar_one_or_none.return_value = None
    mock_db.execute = AsyncMock(return_value=result_mock)

    response = await api_client.delete(f"/api/v1/mcp-tokens/{uuid4()}")

    assert response.status_code == 404


@pytest.mark.asyncio
async def test_revoke_other_player_token_returns_403(api_client, mock_db):
    other_id = uuid4()
    db_token = _make_db_token(player_id=other_id)

    result_mock = MagicMock()
    result_mock.scalar_one_or_none.return_value = db_token
    mock_db.execute = AsyncMock(return_value=result_mock)

    response = await api_client.delete(f"/api/v1/mcp-tokens/{db_token.id}")

    assert response.status_code == 403


@pytest.mark.asyncio
async def test_revoke_own_token_returns_204(api_client, mock_db, player_user):
    db_token = _make_db_token(player_id=player_user.id)

    result_mock = MagicMock()
    result_mock.scalar_one_or_none.return_value = db_token
    mock_db.execute = AsyncMock(return_value=result_mock)

    response = await api_client.delete(f"/api/v1/mcp-tokens/{db_token.id}")

    assert response.status_code == 204


@pytest.mark.asyncio
async def test_revoke_unauthenticated_returns_401(anon_client):
    response = await anon_client.delete(f"/api/v1/mcp-tokens/{uuid4()}")

    assert response.status_code == 401
