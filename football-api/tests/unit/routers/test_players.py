"""
Testes unitários — routers/players.py

Regras de negócio cobertas:
- GET /players/{id} player vê próprios dados → 200
- GET /players/{id} player tenta ver outro → 403
- GET /players/{id} não encontrado → 404
- POST /players não-admin → 403
- POST /players WhatsApp duplicado → 409
- POST /players sucesso → 201
- PATCH /players/{id} player tenta alterar role → 403
- DELETE /players/{id} não encontrado → 404
- POST /players/{id}/reset-password não encontrado → 404
- GET /players/{id}/public-stats jogador válido → 200
- GET /players/{id}/public-stats jogador inativo → 404
- GET /players/{id}/public-stats não encontrado → 404
"""
from datetime import datetime
from unittest.mock import AsyncMock, MagicMock
from uuid import uuid4

import pytest

from app.models.player import PlayerRole


# ── Helpers ───────────────────────────────────────────────────────────────────


def _make_player_db(player_id=None) -> MagicMock:
    p = MagicMock()
    p.id = player_id or uuid4()
    p.name = "Jogador Teste"
    p.nickname = None
    p.whatsapp = "+5511999990001"
    p.role = PlayerRole.PLAYER
    p.active = True
    p.must_change_password = False
    p.avatar_url = None
    p.created_at = datetime(2026, 1, 1)
    p.updated_at = datetime(2026, 1, 1)
    return p


# ── GET /players/{id} ─────────────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_get_player_own_data_returns_200(api_client, player_user, mocker):
    """Player pode acessar seus próprios dados."""
    player = _make_player_db(player_user.id)
    mocker.patch(
        "app.api.v1.routers.players.PlayerRepository.get",
        new=AsyncMock(return_value=player),
    )

    response = await api_client.get(f"/api/v1/players/{player_user.id}")

    assert response.status_code == 200
    assert response.json()["id"] == str(player_user.id)


@pytest.mark.asyncio
async def test_get_player_another_player_returns_403(api_client):
    """Player não pode ver dados de outro jogador."""
    response = await api_client.get(f"/api/v1/players/{uuid4()}")

    assert response.status_code == 403


@pytest.mark.asyncio
async def test_get_player_not_found_returns_404(admin_client, mocker):
    """Admin tentando buscar jogador inexistente → 404."""
    mocker.patch(
        "app.api.v1.routers.players.PlayerRepository.get",
        new=AsyncMock(return_value=None),
    )

    response = await admin_client.get(f"/api/v1/players/{uuid4()}")

    assert response.status_code == 404


# ── POST /players ─────────────────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_create_player_non_admin_returns_403(api_client):
    """Apenas admins podem criar jogadores diretamente."""
    response = await api_client.post(
        "/api/v1/players",
        json={"name": "Novo", "whatsapp": "+5511999990002", "password": "senha123"},
    )

    assert response.status_code == 403


@pytest.mark.asyncio
async def test_create_player_duplicate_whatsapp_returns_409(admin_client, mocker):
    """WhatsApp já cadastrado retorna 409."""
    mocker.patch(
        "app.api.v1.routers.players.PlayerRepository.get_by_whatsapp",
        new=AsyncMock(return_value=MagicMock()),
    )

    response = await admin_client.post(
        "/api/v1/players",
        json={"name": "Novo", "whatsapp": "+5511999990002", "password": "senha123"},
    )

    assert response.status_code == 409


@pytest.mark.asyncio
async def test_create_player_success_returns_201(admin_client, mocker):
    """Criação bem-sucedida de jogador retorna 201."""
    player = _make_player_db()
    mocker.patch(
        "app.api.v1.routers.players.PlayerRepository.get_by_whatsapp",
        new=AsyncMock(return_value=None),
    )
    mocker.patch(
        "app.api.v1.routers.players.PlayerRepository.create",
        new=AsyncMock(return_value=player),
    )

    response = await admin_client.post(
        "/api/v1/players",
        json={"name": "Novo Jogador", "whatsapp": "+5511999990002", "password": "senha123"},
    )

    assert response.status_code == 201


# ── PATCH /players/{id} ───────────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_update_player_change_role_non_admin_returns_403(api_client, player_user, mocker):
    """Jogador não pode alterar seu próprio role."""
    player = _make_player_db(player_user.id)
    mocker.patch(
        "app.api.v1.routers.players.PlayerRepository.get",
        new=AsyncMock(return_value=player),
    )

    response = await api_client.patch(
        f"/api/v1/players/{player_user.id}",
        json={"role": "admin"},
    )

    assert response.status_code == 403


# ── DELETE /players/{id} ──────────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_delete_player_not_found_returns_404(admin_client, mocker):
    mocker.patch(
        "app.api.v1.routers.players.PlayerRepository.get",
        new=AsyncMock(return_value=None),
    )

    response = await admin_client.delete(f"/api/v1/players/{uuid4()}")

    assert response.status_code == 404


# ── POST /players/{id}/reset-password ────────────────────────────────────────


@pytest.mark.asyncio
async def test_reset_password_player_not_found_returns_404(admin_client, mocker):
    mocker.patch(
        "app.api.v1.routers.players.PlayerRepository.get",
        new=AsyncMock(return_value=None),
    )

    response = await admin_client.post(f"/api/v1/players/{uuid4()}/reset-password")

    assert response.status_code == 404


# ── GET /players/{id}/public-stats ───────────────────────────────────────────


def _make_full_stats_mock():
    """Cria mock de PlayerFullStats com dados mínimos."""
    from app.schemas.player_stats import PlayerFullStats, GroupStatItem

    return PlayerFullStats(
        total_matches_confirmed=42,
        total_minutes_played=2520,
        total_vote_points=115,
        top1_count=3,
        top5_count=18,
        total_flop_votes=2,
        current_streak=4,
        best_streak=9,
        attendance_rate=87,
        monthly_stats=[],
        recent_matches=[],
        groups=[
            GroupStatItem(
                group_id=str(uuid4()),
                group_name="Pelada dos Amigos",
                skill_stars=4,
                is_goalkeeper=False,
                role="member",
                matches_confirmed=42,
            )
        ],
    )


@pytest.mark.asyncio
async def test_get_player_public_stats_returns_200(api_client, mocker):
    """Endpoint público retorna 200 com campos corretos."""
    player_id = uuid4()
    player = _make_player_db(player_id)
    mocker.patch(
        "app.api.v1.routers.players.PlayerRepository.get",
        new=AsyncMock(return_value=player),
    )
    mocker.patch(
        "app.api.v1.routers.players.PlayerStatsRepository.get_full_stats",
        new=AsyncMock(return_value=_make_full_stats_mock()),
    )

    response = await api_client.get(f"/api/v1/players/{player_id}/public-stats")

    assert response.status_code == 200
    data = response.json()
    assert data["player_id"] == str(player_id)
    assert data["name"] == player.name
    assert data["total_matches_confirmed"] == 42
    assert data["attendance_rate"] == 87
    assert data["skill_stars"] == 4
    assert data["top5_count"] == 18
    assert data["total_flop_votes"] == 2
    # Campos sensíveis não devem aparecer
    assert "whatsapp" not in data
    assert "role" not in data


@pytest.mark.asyncio
async def test_get_player_public_stats_not_found_returns_404(api_client, mocker):
    """Jogador não encontrado retorna 404."""
    mocker.patch(
        "app.api.v1.routers.players.PlayerRepository.get",
        new=AsyncMock(return_value=None),
    )

    response = await api_client.get(f"/api/v1/players/{uuid4()}/public-stats")

    assert response.status_code == 404


@pytest.mark.asyncio
async def test_get_player_public_stats_inactive_returns_404(api_client, mocker):
    """Jogador inativo retorna 404."""
    player = _make_player_db()
    player.active = False
    mocker.patch(
        "app.api.v1.routers.players.PlayerRepository.get",
        new=AsyncMock(return_value=player),
    )

    response = await api_client.get(f"/api/v1/players/{uuid4()}/public-stats")

    assert response.status_code == 404
