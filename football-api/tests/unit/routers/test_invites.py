"""
Testes unitários — routers/invites.py

Regras de negócio cobertas:
- GET /{token} convite já usado → 403
- GET /{token} convite expirado → 403
- GET /{token} convite válido → 200
- POST /{token}/accept convite inválido → 404
- POST /{token}/accept novo usuário sem nome → 422
- POST /{token}/accept usuário existente com senha errada → 403
- POST /{token}/accept com limite de membros atingido → 403
- POST /{token}/accept novo usuário bem-sucedido → 200 + token
"""
from datetime import datetime, timedelta, timezone
from unittest.mock import AsyncMock, MagicMock
from uuid import uuid4

import pytest

from app.core.security import hash_password
from app.models.player import PlayerRole


# ── Helpers ───────────────────────────────────────────────────────────────────


def _make_invite(used: bool = False, expired: bool = False) -> MagicMock:
    inv = MagicMock()
    inv.group_id = uuid4()
    inv.used = used
    inv.expires_at = (
        datetime.now(timezone.utc) - timedelta(hours=1)
        if expired
        else datetime.now(timezone.utc) + timedelta(hours=1)
    )
    return inv


def _make_group() -> MagicMock:
    g = MagicMock()
    g.name = "Pelada do Bairro"
    return g


def _make_player(whatsapp: str = "11999990001") -> MagicMock:
    p = MagicMock()
    p.id = uuid4()
    p.name = "João"
    p.whatsapp = whatsapp
    p.password_hash = hash_password("senha123")
    p.role = PlayerRole.PLAYER
    p.must_change_password = False
    return p


# ── GET /{token} ──────────────────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_get_invite_used_returns_403(api_client, mocker):
    invite = _make_invite(used=True)
    mocker.patch(
        "app.api.v1.routers.invites.InviteRepository.get_by_token",
        new=AsyncMock(return_value=invite),
    )

    response = await api_client.get("/api/v1/invites/qualquertoken")

    assert response.status_code == 403


@pytest.mark.asyncio
async def test_get_invite_expired_returns_403(api_client, mocker):
    invite = _make_invite(expired=True)
    mocker.patch(
        "app.api.v1.routers.invites.InviteRepository.get_by_token",
        new=AsyncMock(return_value=invite),
    )

    response = await api_client.get("/api/v1/invites/qualquertoken")

    assert response.status_code == 403


@pytest.mark.asyncio
async def test_get_invite_not_found_returns_404(api_client, mocker):
    mocker.patch(
        "app.api.v1.routers.invites.InviteRepository.get_by_token",
        new=AsyncMock(return_value=None),
    )

    response = await api_client.get("/api/v1/invites/naoexiste")

    assert response.status_code == 404


@pytest.mark.asyncio
async def test_get_invite_valid_returns_200(api_client, mocker):
    invite = _make_invite()
    group = _make_group()
    mocker.patch(
        "app.api.v1.routers.invites.InviteRepository.get_by_token",
        new=AsyncMock(return_value=invite),
    )
    mocker.patch(
        "app.api.v1.routers.invites.GroupRepository.get",
        new=AsyncMock(return_value=group),
    )

    response = await api_client.get("/api/v1/invites/tokenvalido")

    assert response.status_code == 200
    data = response.json()
    assert data["valid"] is True
    assert data["group_name"] == "Pelada do Bairro"


# ── POST /{token}/accept ──────────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_accept_invite_invalid_token_returns_404(api_client, mocker):
    mocker.patch(
        "app.api.v1.routers.invites.InviteRepository.get_valid_token",
        new=AsyncMock(return_value=None),
    )

    response = await api_client.post(
        "/api/v1/invites/tokeninvalido/accept",
        json={"whatsapp": "11999990001", "password": "senha"},
    )

    assert response.status_code == 404


@pytest.mark.asyncio
async def test_accept_invite_new_user_without_name_returns_422(api_client, mocker):
    invite = _make_invite()
    mocker.patch(
        "app.api.v1.routers.invites.InviteRepository.get_valid_token",
        new=AsyncMock(return_value=invite),
    )
    mocker.patch(
        "app.api.v1.routers.invites.PlayerRepository.get_by_whatsapp",
        new=AsyncMock(return_value=None),  # novo usuário
    )
    mocker.patch(
        "app.api.v1.routers.invites.GroupRepository.get_non_admin_member_ids",
        new=AsyncMock(return_value=[]),
    )

    response = await api_client.post(
        "/api/v1/invites/tokenvalido/accept",
        json={"whatsapp": "11999990001", "password": "senha123"},
        # name ausente
    )

    assert response.status_code == 422


@pytest.mark.asyncio
async def test_accept_invite_existing_user_wrong_password_returns_403(api_client, mocker):
    invite = _make_invite()
    player = _make_player()
    mocker.patch(
        "app.api.v1.routers.invites.InviteRepository.get_valid_token",
        new=AsyncMock(return_value=invite),
    )
    mocker.patch(
        "app.api.v1.routers.invites.PlayerRepository.get_by_whatsapp",
        new=AsyncMock(return_value=player),
    )

    response = await api_client.post(
        "/api/v1/invites/tokenvalido/accept",
        json={"whatsapp": "11999990001", "password": "senha_errada"},
    )

    assert response.status_code == 403


@pytest.mark.asyncio
async def test_accept_invite_member_limit_reached_returns_403(api_client, mocker):
    invite = _make_invite()
    mocker.patch(
        "app.api.v1.routers.invites.InviteRepository.get_valid_token",
        new=AsyncMock(return_value=invite),
    )
    mocker.patch(
        "app.api.v1.routers.invites.PlayerRepository.get_by_whatsapp",
        new=AsyncMock(return_value=None),  # novo usuário
    )
    # 30 membros — limite free atingido
    mocker.patch(
        "app.api.v1.routers.invites.GroupRepository.get_non_admin_member_ids",
        new=AsyncMock(return_value=list(range(30))),
    )

    response = await api_client.post(
        "/api/v1/invites/tokenvalido/accept",
        json={"whatsapp": "11999990001", "password": "senha123", "name": "Novo Jogador"},
    )

    assert response.status_code == 403
    assert response.json()["detail"] == "PLAN_LIMIT_EXCEEDED"


@pytest.mark.asyncio
async def test_accept_invite_new_user_success(api_client, mocker):
    invite = _make_invite()
    player = _make_player()
    mocker.patch(
        "app.api.v1.routers.invites.InviteRepository.get_valid_token",
        new=AsyncMock(return_value=invite),
    )
    mocker.patch(
        "app.api.v1.routers.invites.PlayerRepository.get_by_whatsapp",
        new=AsyncMock(return_value=None),
    )
    mocker.patch(
        "app.api.v1.routers.invites.GroupRepository.get_non_admin_member_ids",
        new=AsyncMock(return_value=[]),
    )
    mocker.patch(
        "app.api.v1.routers.invites.PlayerRepository.create",
        new=AsyncMock(return_value=player),
    )
    mocker.patch(
        "app.api.v1.routers.invites.SubscriptionRepository.get_or_create",
        new=AsyncMock(return_value=MagicMock()),
    )
    mocker.patch(
        "app.api.v1.routers.invites.GroupRepository.add_member",
        new=AsyncMock(return_value=None),
    )
    mocker.patch(
        "app.api.v1.routers.invites.MatchRepository.get_active_matches",
        new=AsyncMock(return_value=[]),
    )
    mocker.patch(
        "app.api.v1.routers.invites.FinanceRepository.ensure_member_in_current_period",
        new=AsyncMock(return_value=None),
    )

    response = await api_client.post(
        "/api/v1/invites/tokenvalido/accept",
        json={"whatsapp": "11999990002", "password": "senha123", "name": "Novo Jogador"},
    )

    assert response.status_code == 200
    assert "access_token" in response.json()
