"""
Testes unitários — POST /api/v1/auth/login e endpoints relacionados.

Regras de negócio cobertas:
- Login com credenciais corretas retorna token
- Login com senha errada retorna 401
- Login com conta inativa retorna 401
- WhatsApp normalizado (remove não-dígitos) antes de consultar
- GET /auth/me retorna o player atual
- change_password com senha atual correta → 204
- change_password com senha atual errada → 401
- change_password sem credenciais → 422
- forgot_password/reset com token inválido → 401
"""
from unittest.mock import AsyncMock, MagicMock, patch
from uuid import uuid4

import pytest

from app.core.security import create_otp_token, hash_password
from app.models.player import PlayerRole


# ── Helpers ───────────────────────────────────────────────────────────────────


def _make_player(active: bool = True, whatsapp: str = "+5511999990001") -> MagicMock:
    p = MagicMock()
    p.id = uuid4()
    p.name = "João Silva"
    p.nickname = None
    p.whatsapp = whatsapp
    p.password_hash = hash_password("senha123")
    p.role = PlayerRole.PLAYER
    p.active = active
    p.must_change_password = False
    return p


# ── Login ─────────────────────────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_login_correct_credentials_returns_token(api_client, mocker):
    player = _make_player()
    mocker.patch(
        "app.api.v1.routers.auth.PlayerRepository.get_by_whatsapp",
        new=AsyncMock(return_value=player),
    )

    response = await api_client.post(
        "/api/v1/auth/login",
        json={"whatsapp": "+5511999990001", "password": "senha123"},
    )

    assert response.status_code == 200
    data = response.json()
    assert "access_token" in data
    assert data["player_id"] == str(player.id)


@pytest.mark.asyncio
async def test_login_wrong_password_returns_401(api_client, mocker):
    player = _make_player()
    mocker.patch(
        "app.api.v1.routers.auth.PlayerRepository.get_by_whatsapp",
        new=AsyncMock(return_value=player),
    )

    response = await api_client.post(
        "/api/v1/auth/login",
        json={"whatsapp": "+5511999990001", "password": "errada"},
    )

    assert response.status_code == 401


@pytest.mark.asyncio
async def test_login_user_not_found_returns_401(api_client, mocker):
    mocker.patch(
        "app.api.v1.routers.auth.PlayerRepository.get_by_whatsapp",
        new=AsyncMock(return_value=None),
    )

    response = await api_client.post(
        "/api/v1/auth/login",
        json={"whatsapp": "+5511000000000", "password": "qualquer"},
    )

    assert response.status_code == 401


@pytest.mark.asyncio
async def test_login_inactive_account_returns_401(api_client, mocker):
    player = _make_player(active=False)
    mocker.patch(
        "app.api.v1.routers.auth.PlayerRepository.get_by_whatsapp",
        new=AsyncMock(return_value=player),
    )

    response = await api_client.post(
        "/api/v1/auth/login",
        json={"whatsapp": "+5511999990001", "password": "senha123"},
    )

    assert response.status_code == 401


@pytest.mark.asyncio
async def test_login_normalizes_whatsapp(api_client, mocker):
    """WhatsApp com formatação (+55 (11) 9 9999-0001) deve ser normalizado para E.164."""
    player = _make_player(whatsapp="+5511999990001")
    mock_get = AsyncMock(return_value=player)
    mocker.patch("app.api.v1.routers.auth.PlayerRepository.get_by_whatsapp", new=mock_get)

    await api_client.post(
        "/api/v1/auth/login",
        json={"whatsapp": "+55 (11) 9 9999-0001", "password": "senha123"},
    )

    called_with = mock_get.call_args[0][0]
    assert called_with == "+5511999990001"


# ── GET /auth/me ──────────────────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_me_returns_current_player(api_client, player_user):
    player_user.name = "João Silva"
    player_user.whatsapp = "+5511999990001"
    player_user.nickname = None
    player_user.role = PlayerRole.PLAYER
    player_user.active = True
    player_user.created_at = "2026-01-01T00:00:00"
    player_user.updated_at = "2026-01-01T00:00:00"
    player_user.must_change_password = False

    response = await api_client.get("/api/v1/auth/me")

    assert response.status_code == 200
    assert response.json()["name"] == "João Silva"


# ── change-password ───────────────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_change_password_correct_current_password(api_client, player_user, mocker):
    player_user.password_hash = hash_password("senha_atual")
    player_user.whatsapp = "+5511999990001"

    db_player = MagicMock()
    mocker.patch(
        "app.api.v1.routers.auth.PlayerRepository.get",
        new=AsyncMock(return_value=db_player),
    )

    response = await api_client.post(
        "/api/v1/auth/change-password",
        json={"current_password": "senha_atual", "new_password": "nova_senha_123"},
    )

    assert response.status_code == 204


@pytest.mark.asyncio
async def test_change_password_wrong_current_password_returns_401(api_client, player_user):
    player_user.password_hash = hash_password("senha_correta")

    response = await api_client.post(
        "/api/v1/auth/change-password",
        json={"current_password": "senha_errada", "new_password": "nova_senha"},
    )

    assert response.status_code == 401


@pytest.mark.asyncio
async def test_change_password_no_credentials_returns_422(api_client):
    response = await api_client.post(
        "/api/v1/auth/change-password",
        json={"new_password": "nova_senha"},
    )

    assert response.status_code == 422


@pytest.mark.asyncio
async def test_change_password_with_valid_otp_token(api_client, player_user, mocker):
    player_user.whatsapp = "+5511999990001"
    otp_token = create_otp_token("+5511999990001")

    db_player = MagicMock()
    mocker.patch(
        "app.api.v1.routers.auth.PlayerRepository.get",
        new=AsyncMock(return_value=db_player),
    )

    response = await api_client.post(
        "/api/v1/auth/change-password",
        json={"otp_token": otp_token, "new_password": "nova_senha_123"},
    )

    assert response.status_code == 204


@pytest.mark.asyncio
async def test_change_password_with_invalid_otp_token_returns_401(api_client):
    response = await api_client.post(
        "/api/v1/auth/change-password",
        json={"otp_token": "token.invalido.aqui", "new_password": "nova_senha"},
    )

    assert response.status_code == 401


# ── forgot-password/reset ─────────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_forgot_password_reset_invalid_token_returns_401(api_client):
    response = await api_client.post(
        "/api/v1/auth/forgot-password/reset",
        json={"whatsapp": "+5511999990001", "otp_token": "invalido", "new_password": "nova123"},
    )

    assert response.status_code == 401


@pytest.mark.asyncio
async def test_forgot_password_reset_valid_token(api_client, mocker):
    whatsapp = "+5511999990001"
    otp_token = create_otp_token(whatsapp)
    player = _make_player(whatsapp=whatsapp)

    mocker.patch(
        "app.api.v1.routers.auth.PlayerRepository.get_by_whatsapp",
        new=AsyncMock(return_value=player),
    )

    response = await api_client.post(
        "/api/v1/auth/forgot-password/reset",
        json={"whatsapp": whatsapp, "otp_token": otp_token, "new_password": "nova_senha_123"},
    )

    assert response.status_code == 204
