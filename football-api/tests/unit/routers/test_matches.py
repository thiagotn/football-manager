"""
Testes unitários — routers/matches.py

Regras de negócio cobertas:
- GET /matches/public/{hash} sem autenticação → 200
- GET /matches/public/{hash} não encontrado → 404
- GET /groups/{id}/matches não-membro → 403
- GET /groups/{id}/matches/{id} não-membro → 403
- POST /groups/{id}/matches não-admin do grupo → 403
- DELETE /groups/{id}/matches/{id} não-admin do grupo → 403
- _build_detail exclui super admin das listas de presença
"""
from datetime import time
from unittest.mock import AsyncMock, MagicMock
from uuid import uuid4

import pytest

from app.models.match import AttendanceStatus, MatchStatus
from app.models.player import PlayerRole


# ── Helpers ───────────────────────────────────────────────────────────────────


def _make_attendance(status: AttendanceStatus, is_admin: bool = False) -> MagicMock:
    a = MagicMock()
    a.id = uuid4()
    a.status = status
    a.player = MagicMock()
    a.player.role = PlayerRole.ADMIN if is_admin else PlayerRole.PLAYER
    a.player.id = uuid4()
    a.player.name = "Jogador"
    a.player.nickname = None
    a.player.avatar_url = None
    return a


def _make_match(hash_: str = "abc123", with_admin: bool = False) -> MagicMock:
    m = MagicMock()
    m.id = uuid4()
    m.number = 1
    m.group_id = uuid4()
    m.hash = hash_
    m.status = MatchStatus.OPEN
    m.match_date = "2026-03-10"
    m.start_time = time(10, 0)
    m.end_time = None
    m.location = "Quadra X"
    m.address = None
    m.court_type = None
    m.players_per_team = None
    m.max_players = None
    m.notes = None
    m.created_at = "2026-01-01T00:00:00"
    m.updated_at = "2026-01-01T00:00:00"
    m.group = MagicMock()
    m.group.name = "Pelada"
    m.group.per_match_amount = None
    m.group.monthly_amount = None
    m.group.timezone = "America/Sao_Paulo"

    attendances = [_make_attendance(AttendanceStatus.CONFIRMED)]
    if with_admin:
        attendances.append(_make_attendance(AttendanceStatus.CONFIRMED, is_admin=True))
    m.attendances = attendances
    return m


# ── GET /matches/public/{hash} ────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_get_public_match_found(api_client, mocker):
    """Endpoint público retorna 200 com dados da partida."""
    match = _make_match("hash123")
    mocker.patch(
        "app.api.v1.routers.matches.MatchRepository.get_by_hash_with_attendances",
        new=AsyncMock(return_value=match),
    )

    response = await api_client.get("/api/v1/matches/public/hash123")

    assert response.status_code == 200
    assert response.json()["hash"] == "hash123"


@pytest.mark.asyncio
async def test_get_public_match_not_found_returns_404(api_client, mocker):
    mocker.patch(
        "app.api.v1.routers.matches.MatchRepository.get_by_hash_with_attendances",
        new=AsyncMock(return_value=None),
    )

    response = await api_client.get("/api/v1/matches/public/naoexiste")

    assert response.status_code == 404


@pytest.mark.asyncio
async def test_get_public_match_excludes_admin_attendance(api_client, mocker):
    """Super admin não deve aparecer nas listas de presença."""
    match = _make_match("hash456", with_admin=True)
    mocker.patch(
        "app.api.v1.routers.matches.MatchRepository.get_by_hash_with_attendances",
        new=AsyncMock(return_value=match),
    )

    response = await api_client.get("/api/v1/matches/public/hash456")

    data = response.json()
    # Partida tem 1 jogador + 1 admin → lista deve ter apenas 1
    assert data["confirmed_count"] == 1
    assert len(data["attendances"]) == 1


# ── GET /groups/{id}/matches ──────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_list_group_matches_non_member_returns_403(api_client, mocker):
    group_id = uuid4()
    group = MagicMock()
    mocker.patch(
        "app.api.v1.routers.matches.GroupRepository.get",
        new=AsyncMock(return_value=group),
    )
    mocker.patch(
        "app.api.v1.routers.matches.GroupRepository.get_member",
        new=AsyncMock(return_value=None),  # não é membro
    )

    response = await api_client.get(f"/api/v1/groups/{group_id}/matches")

    assert response.status_code == 403


@pytest.mark.asyncio
async def test_list_group_matches_group_not_found_returns_404(api_client, mocker):
    group_id = uuid4()
    mocker.patch(
        "app.api.v1.routers.matches.GroupRepository.get",
        new=AsyncMock(return_value=None),
    )

    response = await api_client.get(f"/api/v1/groups/{group_id}/matches")

    assert response.status_code == 404


# ── GET /groups/{id}/matches/{id} ────────────────────────────────────────────


@pytest.mark.asyncio
async def test_get_match_non_member_returns_403(api_client, mocker):
    group_id = uuid4()
    match_id = uuid4()
    mocker.patch(
        "app.api.v1.routers.matches.GroupRepository.get_member",
        new=AsyncMock(return_value=None),
    )

    response = await api_client.get(f"/api/v1/groups/{group_id}/matches/{match_id}")

    assert response.status_code == 403


@pytest.mark.asyncio
async def test_get_match_not_found_returns_404(api_client, mocker):
    group_id = uuid4()
    match_id = uuid4()
    member = MagicMock()
    mocker.patch(
        "app.api.v1.routers.matches.GroupRepository.get_member",
        new=AsyncMock(return_value=member),
    )
    mocker.patch(
        "app.api.v1.routers.matches.MatchRepository.get_with_attendances",
        new=AsyncMock(return_value=None),
    )

    response = await api_client.get(f"/api/v1/groups/{group_id}/matches/{match_id}")

    assert response.status_code == 404


# ── POST /groups/{id}/matches ─────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_create_match_non_group_admin_returns_403(api_client, mocker):
    group_id = uuid4()
    group = MagicMock()
    mocker.patch(
        "app.api.v1.routers.matches.GroupRepository.get",
        new=AsyncMock(return_value=group),
    )
    member = MagicMock()
    member.role = "member"  # não é admin do grupo
    mocker.patch(
        "app.api.v1.routers.matches.GroupRepository.get_member",
        new=AsyncMock(return_value=member),
    )

    response = await api_client.post(
        f"/api/v1/groups/{group_id}/matches",
        json={
            "match_date": "2026-04-01",
            "start_time": "10:00:00",
            "location": "Quadra X",
        },
    )

    assert response.status_code == 403


# ── DELETE /groups/{id}/matches/{id} ─────────────────────────────────────────


@pytest.mark.asyncio
async def test_delete_match_non_group_admin_returns_403(api_client, mocker):
    group_id = uuid4()
    match_id = uuid4()
    member = MagicMock()
    member.role = "member"
    mocker.patch(
        "app.api.v1.routers.matches.GroupRepository.get_member",
        new=AsyncMock(return_value=member),
    )

    response = await api_client.delete(f"/api/v1/groups/{group_id}/matches/{match_id}")

    assert response.status_code == 403


# ── PATCH /groups/{id}/matches/{id} ──────────────────────────────────────────


@pytest.mark.asyncio
async def test_update_match_non_group_admin_returns_403(api_client, mocker):
    group_id = uuid4()
    match_id = uuid4()
    member = MagicMock()
    member.role = "member"
    mocker.patch(
        "app.api.v1.routers.matches.GroupRepository.get_member",
        new=AsyncMock(return_value=member),
    )

    response = await api_client.patch(
        f"/api/v1/groups/{group_id}/matches/{match_id}",
        json={"location": "Nova Quadra"},
    )

    assert response.status_code == 403


@pytest.mark.asyncio
async def test_update_match_not_found_returns_404(api_client, mocker):
    group_id = uuid4()
    match_id = uuid4()
    member = MagicMock()
    member.role = "admin"
    mocker.patch(
        "app.api.v1.routers.matches.GroupRepository.get_member",
        new=AsyncMock(return_value=member),
    )
    mocker.patch(
        "app.api.v1.routers.matches.MatchRepository.get",
        new=AsyncMock(return_value=None),
    )

    response = await api_client.patch(
        f"/api/v1/groups/{group_id}/matches/{match_id}",
        json={"location": "Nova Quadra"},
    )

    assert response.status_code == 404


# ── DELETE /groups/{id}/matches/{id} — not found ─────────────────────────────


@pytest.mark.asyncio
async def test_delete_match_not_found_returns_404(api_client, mocker):
    group_id = uuid4()
    match_id = uuid4()
    member = MagicMock()
    member.role = "admin"
    mocker.patch(
        "app.api.v1.routers.matches.GroupRepository.get_member",
        new=AsyncMock(return_value=member),
    )
    mocker.patch(
        "app.api.v1.routers.matches.MatchRepository.get",
        new=AsyncMock(return_value=None),
    )

    response = await api_client.delete(f"/api/v1/groups/{group_id}/matches/{match_id}")

    assert response.status_code == 404


# ── POST /groups/{id}/matches — group not found ───────────────────────────────


@pytest.mark.asyncio
async def test_create_match_group_not_found_returns_404(api_client, mocker):
    mocker.patch(
        "app.api.v1.routers.matches.GroupRepository.get",
        new=AsyncMock(return_value=None),
    )

    response = await api_client.post(
        f"/api/v1/groups/{uuid4()}/matches",
        json={"match_date": "2026-04-01", "start_time": "10:00:00", "location": "Quadra X"},
    )

    assert response.status_code == 404
