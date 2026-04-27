"""
Testes unitários — app/core/dependencies.py

Regras cobertas:
- MCP token válido autentica o player corretamente
- MCP token revogado retorna None
- MCP token expirado retorna None
- MCP token inexistente retorna None
- get_current_player com JWT válido → retorna player
- get_current_player com JWT inválido → 401
- get_current_player sem token → 401
- get_current_player com MCP token válido → retorna player
- get_current_player com MCP token revogado → 401
- get_optional_player com MCP token válido → retorna player
- get_optional_player com MCP token inválido → None (sem erro)
- last_used_at é atualizado ao usar MCP token válido
"""
from datetime import datetime, timedelta, timezone
from unittest.mock import AsyncMock, MagicMock, patch
from uuid import uuid4

import pytest

from app.core.dependencies import _authenticate_mcp_token, get_current_player, get_optional_player
from app.models.mcp_token import MCPToken


# ── Helpers ───────────────────────────────────────────────────────────────────


def _make_mcp_record(player_id=None, expires_at=None, revoked_at=None) -> MagicMock:
    m = MagicMock()
    m.id = uuid4()
    m.player_id = player_id or uuid4()
    m.expires_at = expires_at
    m.revoked_at = revoked_at
    return m


def _make_player(active: bool = True) -> MagicMock:
    p = MagicMock()
    p.id = uuid4()
    p.active = active
    return p


def _make_db(mcp_record=None, player=None):
    """Retorna um mock de AsyncSession que responde às 3 queries de _authenticate_mcp_token.

    Execute order: (1) SELECT MCPToken, (2) UPDATE last_used_at, (3) SELECT Player.
    """
    db = AsyncMock()

    mcp_result = MagicMock()
    mcp_result.scalar_one_or_none.return_value = mcp_record

    update_result = MagicMock()

    player_result = MagicMock()
    player_result.scalar_one_or_none.return_value = player

    db.execute = AsyncMock(side_effect=[mcp_result, update_result, player_result])
    return db


# ── _authenticate_mcp_token ───────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_valid_mcp_token_returns_player():
    player = _make_player()
    mcp = _make_mcp_record(player_id=player.id)
    db = _make_db(mcp_record=mcp, player=player)

    result = await _authenticate_mcp_token("rachao_qualquertoken", db)

    assert result is player


@pytest.mark.asyncio
async def test_mcp_token_not_found_returns_none():
    db = _make_db(mcp_record=None)

    result = await _authenticate_mcp_token("rachao_inexistente", db)

    assert result is None


@pytest.mark.asyncio
async def test_mcp_token_revoked_returns_none():
    mcp = _make_mcp_record(revoked_at=datetime.now(timezone.utc))
    db = AsyncMock()
    result_mock = MagicMock()
    result_mock.scalar_one_or_none.return_value = None  # WHERE revoked_at IS NULL filtra
    db.execute = AsyncMock(return_value=result_mock)

    result = await _authenticate_mcp_token("rachao_revogado", db)

    assert result is None


@pytest.mark.asyncio
async def test_mcp_token_expired_returns_none():
    past = datetime.now(timezone.utc) - timedelta(hours=1)
    mcp = _make_mcp_record(expires_at=past)
    db = _make_db(mcp_record=mcp)

    result = await _authenticate_mcp_token("rachao_expirado", db)

    assert result is None


@pytest.mark.asyncio
async def test_mcp_token_updates_last_used_at():
    player = _make_player()
    mcp = _make_mcp_record(player_id=player.id)
    db = _make_db(mcp_record=mcp, player=player)

    await _authenticate_mcp_token("rachao_qualquer", db)

    # SELECT MCPToken + UPDATE last_used_at + SELECT Player
    assert db.execute.call_count == 3


# ── get_current_player ────────────────────────────────────────────────────────


def _make_credentials(token: str):
    creds = MagicMock()
    creds.credentials = token
    return creds


@pytest.mark.asyncio
async def test_get_current_player_no_credentials_raises_401():
    from app.core.exceptions import UnauthorizedError
    from fastapi import Request

    db = AsyncMock()
    with pytest.raises(UnauthorizedError):
        await get_current_player(None, db)


@pytest.mark.asyncio
async def test_get_current_player_invalid_jwt_raises_401():
    from app.core.exceptions import UnauthorizedError

    db = AsyncMock()
    with pytest.raises(UnauthorizedError):
        await get_current_player(_make_credentials("not.a.jwt.token"), db)


@pytest.mark.asyncio
async def test_get_current_player_valid_mcp_token_returns_player():
    player = _make_player()
    mcp = _make_mcp_record(player_id=player.id)
    db = _make_db(mcp_record=mcp, player=player)

    result = await get_current_player(_make_credentials("rachao_valid"), db)

    assert result is player


@pytest.mark.asyncio
async def test_get_current_player_revoked_mcp_token_raises_401():
    from app.core.exceptions import UnauthorizedError

    db = AsyncMock()
    result_mock = MagicMock()
    result_mock.scalar_one_or_none.return_value = None
    db.execute = AsyncMock(return_value=result_mock)

    with pytest.raises(UnauthorizedError):
        await get_current_player(_make_credentials("rachao_revogado"), db)


@pytest.mark.asyncio
async def test_get_current_player_inactive_player_via_mcp_raises_401():
    from app.core.exceptions import UnauthorizedError

    inactive = _make_player(active=False)
    mcp = _make_mcp_record(player_id=inactive.id)
    db = _make_db(mcp_record=mcp, player=inactive)

    with pytest.raises(UnauthorizedError):
        await get_current_player(_make_credentials("rachao_inactive"), db)


# ── get_optional_player ───────────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_get_optional_player_valid_mcp_token_returns_player():
    player = _make_player()
    mcp = _make_mcp_record(player_id=player.id)
    db = _make_db(mcp_record=mcp, player=player)

    result = await get_optional_player(_make_credentials("rachao_ok"), db)

    assert result is player


@pytest.mark.asyncio
async def test_get_optional_player_invalid_mcp_token_returns_none():
    db = AsyncMock()
    result_mock = MagicMock()
    result_mock.scalar_one_or_none.return_value = None
    db.execute = AsyncMock(return_value=result_mock)

    result = await get_optional_player(_make_credentials("rachao_invalido"), db)

    assert result is None


@pytest.mark.asyncio
async def test_get_optional_player_no_credentials_returns_none():
    db = AsyncMock()

    result = await get_optional_player(None, db)

    assert result is None
