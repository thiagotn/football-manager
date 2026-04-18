"""
Testes unitários — endpoints de gols e assistências por partida.

Casos cobertos:
- GET /matches/public/{hash}/player-stats sem dados → registered=false, stats=[]
- GET /matches/public/{hash}/player-stats com dados → lista correta
- GET partida não encontrada → 404
- PUT happy path (admin do grupo) → 200, registros persistidos
- PUT não autenticado / não-admin do grupo → 403
- PUT player não confirmado → 409
- PUT partida não encontrada → 404
"""
from unittest.mock import AsyncMock, MagicMock
from uuid import uuid4

import pytest

from app.models.group import GroupMemberRole
from app.models.match import AttendanceStatus, MatchStatus
from app.models.player import PlayerRole


# ── Helpers ───────────────────────────────────────────────────────────────────


def _make_match(hash_: str = "testhash") -> MagicMock:
    m = MagicMock()
    m.id = uuid4()
    m.hash = hash_
    m.group_id = uuid4()
    m.status = MatchStatus.CLOSED
    return m


def _make_stat(goals: int = 1, assists: int = 0) -> MagicMock:
    s = MagicMock()
    s.player_id = uuid4()
    s.player = MagicMock()
    s.player.name = "Fulano"
    s.player.avatar_url = None
    s.goals = goals
    s.assists = assists
    return s


def _make_member(role: GroupMemberRole = GroupMemberRole.ADMIN) -> MagicMock:
    m = MagicMock()
    m.role = role
    return m


# ── GET /matches/public/{hash}/player-stats ───────────────────────────────────


@pytest.mark.asyncio
async def test_get_player_stats_no_data_returns_registered_false(api_client, mocker):
    """Partida sem registros → registered=false e stats vazio."""
    match = _make_match("nostats")
    mocker.patch(
        "app.api.v1.routers.matches.MatchRepository.get_by_hash",
        new=AsyncMock(return_value=match),
    )
    mocker.patch(
        "app.api.v1.routers.matches.MatchStatsRepository.get_by_match",
        new=AsyncMock(return_value=[]),
    )

    response = await api_client.get("/api/v1/matches/public/nostats/player-stats")

    assert response.status_code == 200
    data = response.json()
    assert data["registered"] is False
    assert data["stats"] == []


@pytest.mark.asyncio
async def test_get_player_stats_with_data(api_client, mocker):
    """Partida com registros → lista correta de jogadores."""
    match = _make_match("withstats")
    stat = _make_stat(goals=2, assists=1)
    mocker.patch(
        "app.api.v1.routers.matches.MatchRepository.get_by_hash",
        new=AsyncMock(return_value=match),
    )
    mocker.patch(
        "app.api.v1.routers.matches.MatchStatsRepository.get_by_match",
        new=AsyncMock(return_value=[stat]),
    )

    response = await api_client.get("/api/v1/matches/public/withstats/player-stats")

    assert response.status_code == 200
    data = response.json()
    assert data["registered"] is True
    assert len(data["stats"]) == 1
    assert data["stats"][0]["goals"] == 2
    assert data["stats"][0]["assists"] == 1


@pytest.mark.asyncio
async def test_get_player_stats_match_not_found(api_client, mocker):
    """Partida inexistente → 404."""
    mocker.patch(
        "app.api.v1.routers.matches.MatchRepository.get_by_hash",
        new=AsyncMock(return_value=None),
    )

    response = await api_client.get("/api/v1/matches/public/notfound/player-stats")

    assert response.status_code == 404


# ── PUT /matches/{hash}/player-stats ─────────────────────────────────────────


@pytest.mark.asyncio
async def test_put_player_stats_happy_path(api_client, player_user, mocker):
    """Admin do grupo pode registrar gols e assistências."""
    match = _make_match("put_ok")
    confirmed_id = uuid4()
    stat = _make_stat(goals=1, assists=1)
    stat.player_id = confirmed_id

    mocker.patch(
        "app.api.v1.routers.matches.MatchRepository.get_by_hash",
        new=AsyncMock(return_value=match),
    )
    mocker.patch(
        "app.api.v1.routers.matches.GroupRepository.get_member",
        new=AsyncMock(return_value=_make_member(GroupMemberRole.ADMIN)),
    )
    mocker.patch(
        "app.api.v1.routers.matches.MatchRepository.get_confirmed_player_ids",
        new=AsyncMock(return_value=[confirmed_id]),
    )
    mocker.patch(
        "app.api.v1.routers.matches.MatchStatsRepository.upsert_stats",
        new=AsyncMock(return_value=[stat]),
    )

    response = await api_client.put(
        "/api/v1/matches/put_ok/player-stats",
        json={"stats": [{"player_id": str(confirmed_id), "goals": 1, "assists": 1}]},
    )

    assert response.status_code == 200
    data = response.json()
    assert data["registered"] is True
    assert data["stats"][0]["goals"] == 1


@pytest.mark.asyncio
async def test_put_player_stats_non_group_admin_returns_403(api_client, mocker):
    """Jogador sem role admin do grupo recebe 403."""
    match = _make_match("put_403")
    mocker.patch(
        "app.api.v1.routers.matches.MatchRepository.get_by_hash",
        new=AsyncMock(return_value=match),
    )
    mocker.patch(
        "app.api.v1.routers.matches.GroupRepository.get_member",
        new=AsyncMock(return_value=_make_member(GroupMemberRole.MEMBER)),
    )

    response = await api_client.put(
        "/api/v1/matches/put_403/player-stats",
        json={"stats": []},
    )

    assert response.status_code == 403


@pytest.mark.asyncio
async def test_put_player_stats_non_member_returns_403(api_client, mocker):
    """Jogador que não é membro do grupo recebe 403."""
    match = _make_match("put_nomember")
    mocker.patch(
        "app.api.v1.routers.matches.MatchRepository.get_by_hash",
        new=AsyncMock(return_value=match),
    )
    mocker.patch(
        "app.api.v1.routers.matches.GroupRepository.get_member",
        new=AsyncMock(return_value=None),
    )

    response = await api_client.put(
        "/api/v1/matches/put_nomember/player-stats",
        json={"stats": []},
    )

    assert response.status_code == 403


@pytest.mark.asyncio
async def test_put_player_stats_not_confirmed_returns_409(api_client, mocker):
    """Jogador não confirmado na partida retorna 409."""
    match = _make_match("put_409")
    not_confirmed_id = uuid4()

    mocker.patch(
        "app.api.v1.routers.matches.MatchRepository.get_by_hash",
        new=AsyncMock(return_value=match),
    )
    mocker.patch(
        "app.api.v1.routers.matches.GroupRepository.get_member",
        new=AsyncMock(return_value=_make_member(GroupMemberRole.ADMIN)),
    )
    mocker.patch(
        "app.api.v1.routers.matches.MatchRepository.get_confirmed_player_ids",
        new=AsyncMock(return_value=[]),  # ninguém confirmado
    )

    response = await api_client.put(
        "/api/v1/matches/put_409/player-stats",
        json={"stats": [{"player_id": str(not_confirmed_id), "goals": 1, "assists": 0}]},
    )

    assert response.status_code == 409


@pytest.mark.asyncio
async def test_put_player_stats_match_not_found(api_client, mocker):
    """Partida inexistente → 404."""
    mocker.patch(
        "app.api.v1.routers.matches.MatchRepository.get_by_hash",
        new=AsyncMock(return_value=None),
    )

    response = await api_client.put(
        "/api/v1/matches/notexists/player-stats",
        json={"stats": []},
    )

    assert response.status_code == 404


@pytest.mark.asyncio
async def test_put_player_stats_empty_payload_deletes_all(api_client, mocker):
    """Payload vazio → todos os registros da partida são removidos."""
    match = _make_match("put_empty")
    mocker.patch(
        "app.api.v1.routers.matches.MatchRepository.get_by_hash",
        new=AsyncMock(return_value=match),
    )
    mocker.patch(
        "app.api.v1.routers.matches.GroupRepository.get_member",
        new=AsyncMock(return_value=_make_member(GroupMemberRole.ADMIN)),
    )
    mocker.patch(
        "app.api.v1.routers.matches.MatchRepository.get_confirmed_player_ids",
        new=AsyncMock(return_value=[]),
    )
    upsert_mock = mocker.patch(
        "app.api.v1.routers.matches.MatchStatsRepository.upsert_stats",
        new=AsyncMock(return_value=[]),
    )

    response = await api_client.put(
        "/api/v1/matches/put_empty/player-stats",
        json={"stats": []},
    )

    assert response.status_code == 200
    assert response.json()["registered"] is False
    upsert_mock.assert_awaited_once()
