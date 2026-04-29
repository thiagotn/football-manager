"""
Unit tests — routers/chat.py

Covered rules:
- POST /chat: 403 when chat_enabled is False
- POST /chat: 429 when rate limit exhausted
- POST /chat: 200 SSE stream when access is granted
- GET /admin/chat-users: 200 with list for admin
- GET /admin/chat-users: 403 for regular player
- PATCH /admin/chat-users/{id}: 200 with updated user
- PATCH /admin/chat-users/{id}: 404 when user not found
"""
from datetime import datetime, timezone
from unittest.mock import AsyncMock, MagicMock, patch
from uuid import uuid4

import pytest

from app.models.player import PlayerRole


def _make_db_player(chat_enabled=False):
    p = MagicMock()
    p.id = uuid4()
    p.name = "Test Player"
    p.whatsapp = "+5511999990002"
    p.role = PlayerRole.PLAYER
    p.chat_enabled = chat_enabled
    p.chat_req_count = 0
    p.chat_req_window = None
    p.created_at = datetime.now(timezone.utc)
    return p


# ── POST /chat ────────────────────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_chat_disabled_player_returns_403(api_client, player_user):
    player_user.chat_enabled = False

    response = await api_client.post(
        "/api/v1/chat",
        json={"messages": [{"role": "user", "content": "Olá"}]},
    )

    assert response.status_code == 403


@pytest.mark.asyncio
async def test_chat_rate_limited_returns_429(api_client, mock_db, player_user):
    player_user.chat_enabled = True
    player_user.chat_req_window = datetime.now(timezone.utc)
    player_user.chat_req_count = 20  # matches default chat_rate_limit

    response = await api_client.post(
        "/api/v1/chat",
        json={"messages": [{"role": "user", "content": "Olá"}]},
    )

    assert response.status_code == 429


@pytest.mark.asyncio
async def test_chat_streams_sse_for_enabled_player(api_client, mock_db, player_user):
    player_user.chat_enabled = True
    player_user.chat_req_window = None
    player_user.chat_req_count = 0

    async def fake_text_stream():
        yield "Hello"
        yield " world"

    mock_stream = MagicMock()
    mock_stream.__aenter__ = AsyncMock(return_value=mock_stream)
    mock_stream.__aexit__ = AsyncMock(return_value=None)
    mock_stream.text_stream = fake_text_stream()

    mock_client = MagicMock()
    mock_client.beta.messages.stream.return_value = mock_stream

    with patch("app.api.v1.routers.chat.anthropic.AsyncAnthropic", return_value=mock_client):
        response = await api_client.post(
            "/api/v1/chat",
            json={"messages": [{"role": "user", "content": "Hi"}]},
        )

    assert response.status_code == 200
    assert "text/event-stream" in response.headers["content-type"]
    body = response.text
    assert "Hello" in body
    assert "[DONE]" in body


@pytest.mark.asyncio
async def test_chat_unauthenticated_returns_401(anon_client):
    response = await anon_client.post(
        "/api/v1/chat",
        json={"messages": [{"role": "user", "content": "Oi"}]},
    )

    assert response.status_code == 401


# ── GET /admin/chat-users ─────────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_admin_list_chat_users_returns_200(admin_client, mock_db):
    db_player = _make_db_player(chat_enabled=True)

    scalars_mock = MagicMock()
    scalars_mock.all.return_value = [db_player]
    result_mock = MagicMock()
    result_mock.scalars.return_value = scalars_mock
    mock_db.execute = AsyncMock(return_value=result_mock)

    response = await admin_client.get("/api/v1/admin/chat-users")

    assert response.status_code == 200
    data = response.json()
    assert "users" in data
    assert "total_enabled" in data
    assert data["total_enabled"] == 1
    assert len(data["users"]) == 1
    assert data["users"][0]["chat_enabled"] is True


@pytest.mark.asyncio
async def test_admin_list_chat_users_forbidden_for_player(api_client):
    response = await api_client.get("/api/v1/admin/chat-users")

    assert response.status_code == 403


# ── PATCH /admin/chat-users/{id} ─────────────────────────────────────────────


@pytest.mark.asyncio
async def test_admin_update_chat_access_returns_updated_user(admin_client, mock_db):
    player_id = uuid4()
    db_player_before = _make_db_player(chat_enabled=False)
    db_player_before.id = player_id

    db_player_after = _make_db_player(chat_enabled=True)
    db_player_after.id = player_id

    call_count = 0

    async def execute_side_effect(stmt):
        nonlocal call_count
        call_count += 1
        result = MagicMock()
        if call_count == 1:
            # SELECT to find player
            result.scalar_one_or_none.return_value = db_player_before
        elif call_count == 2:
            # UPDATE — no special return needed
            result.scalar_one_or_none.return_value = None
        else:
            # re-SELECT after update
            result.scalar_one.return_value = db_player_after
        return result

    mock_db.execute = AsyncMock(side_effect=execute_side_effect)

    response = await admin_client.patch(
        f"/api/v1/admin/chat-users/{player_id}",
        json={"chat_enabled": True},
    )

    assert response.status_code == 200
    data = response.json()
    assert data["chat_enabled"] is True
    assert str(data["id"]) == str(player_id)


@pytest.mark.asyncio
async def test_admin_update_chat_access_not_found_returns_404(admin_client, mock_db):
    result_mock = MagicMock()
    result_mock.scalar_one_or_none.return_value = None
    mock_db.execute = AsyncMock(return_value=result_mock)

    response = await admin_client.patch(
        f"/api/v1/admin/chat-users/{uuid4()}",
        json={"chat_enabled": True},
    )

    assert response.status_code == 404


@pytest.mark.asyncio
async def test_admin_update_chat_access_forbidden_for_player(api_client):
    response = await api_client.patch(
        f"/api/v1/admin/chat-users/{uuid4()}",
        json={"chat_enabled": True},
    )

    assert response.status_code == 403
