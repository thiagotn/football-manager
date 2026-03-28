"""
Testes unitários — routers/teams.py

Regras de negócio cobertas:
- POST /matches/{id}/teams partida não encontrada → 404
- POST /matches/{id}/teams não-admin do grupo → 403
- POST /matches/{id}/teams players_per_team não definido → 422
- POST /matches/{id}/teams confirmados insuficientes → 422
- GET /matches/{id}/teams partida não encontrada → 404
"""
from unittest.mock import AsyncMock, MagicMock
from uuid import uuid4

import pytest


# ── Helpers ───────────────────────────────────────────────────────────────────


def _make_match(players_per_team: int | None = 5) -> MagicMock:
    m = MagicMock()
    m.id = uuid4()
    m.group_id = uuid4()
    m.players_per_team = players_per_team
    return m


# ── POST /matches/{id}/teams ──────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_generate_teams_match_not_found_returns_404(api_client, mocker):
    mocker.patch(
        "app.api.v1.routers.teams.MatchRepository.get_with_attendances",
        new=AsyncMock(return_value=None),
    )

    response = await api_client.post(f"/api/v1/matches/{uuid4()}/teams")

    assert response.status_code == 404


@pytest.mark.asyncio
async def test_generate_teams_non_group_admin_returns_403(api_client, mocker):
    match = _make_match()
    mocker.patch(
        "app.api.v1.routers.teams.MatchRepository.get_with_attendances",
        new=AsyncMock(return_value=match),
    )
    member = MagicMock()
    member.role = "member"  # não é admin do grupo
    mocker.patch(
        "app.api.v1.routers.teams.GroupRepository.get_member",
        new=AsyncMock(return_value=member),
    )

    response = await api_client.post(f"/api/v1/matches/{uuid4()}/teams")

    assert response.status_code == 403


@pytest.mark.asyncio
async def test_generate_teams_no_players_per_team_returns_422(admin_client, mocker):
    """players_per_team não definido impede o sorteio."""
    match = _make_match(players_per_team=None)
    mocker.patch(
        "app.api.v1.routers.teams.MatchRepository.get_with_attendances",
        new=AsyncMock(return_value=match),
    )

    response = await admin_client.post(f"/api/v1/matches/{uuid4()}/teams")

    assert response.status_code == 422


@pytest.mark.asyncio
async def test_generate_teams_insufficient_players_returns_422(admin_client, mocker):
    """Com poucos confirmados (< (ppt+1)*2) o sorteio é recusado."""
    match = _make_match(players_per_team=5)  # min_needed = (5+1)*2 = 12
    mocker.patch(
        "app.api.v1.routers.teams.MatchRepository.get_with_attendances",
        new=AsyncMock(return_value=match),
    )
    mocker.patch(
        "app.api.v1.routers.teams.GroupRepository.get_confirmed_players_with_skills",
        new=AsyncMock(return_value=[]),  # 0 confirmados
    )

    response = await admin_client.post(f"/api/v1/matches/{uuid4()}/teams")

    assert response.status_code == 422


# ── GET /matches/{id}/teams ───────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_get_teams_match_not_found_returns_404(api_client, mocker):
    mocker.patch(
        "app.api.v1.routers.teams.MatchRepository.get_with_attendances",
        new=AsyncMock(return_value=None),
    )

    response = await api_client.get(f"/api/v1/matches/{uuid4()}/teams")

    assert response.status_code == 404


# ── POST /matches/{id}/teams — happy path ────────────────────────────────────


@pytest.mark.asyncio
async def test_generate_teams_success_returns_201(admin_client, mocker):
    """Super admin pode gerar times para uma partida com confirmados suficientes."""
    from uuid import uuid4

    match = _make_match(players_per_team=4)  # min_needed = (4+1)*2 = 10

    # Cria 10 jogadores confirmados com skills
    confirmed = [
        {
            "player_id": uuid4(),
            "name": f"Jogador {i}",
            "nickname": None,
            "skill_stars": 3,
            "is_goalkeeper": (i == 0 or i == 5),
        }
        for i in range(10)
    ]

    mocker.patch(
        "app.api.v1.routers.teams.MatchRepository.get_with_attendances",
        new=AsyncMock(return_value=match),
    )
    mocker.patch(
        "app.api.v1.routers.teams.GroupRepository.get_confirmed_players_with_skills",
        new=AsyncMock(return_value=confirmed),
    )

    # Mock do team builder para retornar times simples
    team1_players = confirmed[:5]
    team2_players = confirmed[5:]
    teams_data = [
        {"name": "Time 1", "color": "red", "position": 1, "players": team1_players},
        {"name": "Time 2", "color": "blue", "position": 2, "players": team2_players},
    ]
    mocker.patch(
        "app.api.v1.routers.teams.build_teams",
        return_value=(teams_data, []),
    )
    mocker.patch(
        "app.api.v1.routers.teams.TeamRepository.delete_by_match",
        new=AsyncMock(return_value=None),
    )

    created_team = MagicMock()
    created_team.id = uuid4()
    created_team.name = "Time 1"
    created_team.color = "red"
    created_team.position = 1
    created_team.players = []

    mocker.patch(
        "app.api.v1.routers.teams.TeamRepository.create_team",
        new=AsyncMock(return_value=created_team),
    )
    mocker.patch(
        "app.api.v1.routers.teams.TeamRepository.add_player",
        new=AsyncMock(return_value=None),
    )
    mocker.patch(
        "app.api.v1.routers.teams.TeamRepository.get_by_match",
        new=AsyncMock(return_value=[created_team]),
    )

    response = await admin_client.post(f"/api/v1/matches/{uuid4()}/teams")

    assert response.status_code == 201


# ── GET /matches/{id}/teams — happy path ─────────────────────────────────────


@pytest.mark.asyncio
async def test_get_teams_success_returns_200(api_client, mocker):
    """Endpoint público retorna times gerados para a partida."""
    match = _make_match(players_per_team=4)

    mocker.patch(
        "app.api.v1.routers.teams.MatchRepository.get_with_attendances",
        new=AsyncMock(return_value=match),
    )
    mocker.patch(
        "app.api.v1.routers.teams.TeamRepository.get_by_match",
        new=AsyncMock(return_value=[]),
    )
    mocker.patch(
        "app.api.v1.routers.teams.GroupRepository.get_member_skills",
        new=AsyncMock(return_value={}),
    )

    response = await api_client.get(f"/api/v1/matches/{uuid4()}/teams")

    assert response.status_code == 200
    assert "teams" in response.json()
    assert "reserves" in response.json()
