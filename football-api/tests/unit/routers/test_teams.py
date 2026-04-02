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


# ── POST /matches/{id}/teams — com reservas ───────────────────────────────────


@pytest.mark.asyncio
async def test_generate_teams_with_reserves_returns_201(admin_client, mocker):
    """Quando há jogadores excedentes, reservas são criadas em time separado."""
    match = _make_match(players_per_team=4)
    confirmed = [
        {
            "player_id": uuid4(),
            "name": f"Jogador {i}",
            "nickname": None,
            "skill_stars": 3,
            "is_goalkeeper": False,
        }
        for i in range(10)
    ]
    reserve_player = {
        "player_id": uuid4(),
        "name": "Reserva",
        "nickname": None,
        "skill_stars": 2,
        "is_goalkeeper": False,
    }
    teams_data = [
        {"name": "Time 1", "color": "#e53e3e", "position": 1, "players": confirmed[:5]},
        {"name": "Time 2", "color": "#3b82f6", "position": 2, "players": confirmed[5:]},
    ]

    mocker.patch(
        "app.api.v1.routers.teams.MatchRepository.get_with_attendances",
        new=AsyncMock(return_value=match),
    )
    mocker.patch(
        "app.api.v1.routers.teams.GroupRepository.get_confirmed_players_with_skills",
        new=AsyncMock(return_value=confirmed + [reserve_player]),
    )
    mocker.patch(
        "app.api.v1.routers.teams.build_teams",
        return_value=(teams_data, [reserve_player]),
    )
    mocker.patch(
        "app.api.v1.routers.teams.TeamRepository.delete_by_match",
        new=AsyncMock(return_value=None),
    )

    created_team = MagicMock()
    created_team.id = uuid4()
    created_team.name = "Time 1"
    created_team.color = "#e53e3e"
    created_team.position = 1
    created_team.players = []

    reserve_team = MagicMock()
    reserve_team.id = uuid4()
    reserve_team.name = "Reservas"
    reserve_team.color = None
    reserve_team.position = 0
    reserve_team.players = []

    mocker.patch(
        "app.api.v1.routers.teams.TeamRepository.create_team",
        new=AsyncMock(side_effect=[created_team, created_team, reserve_team]),
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
    data = response.json()
    assert "reserves" in data
    # A reserva deve aparecer na resposta
    assert len(data["reserves"]) == 1
    assert data["reserves"][0]["name"] == "Reserva"


# ── GET /matches/{id}/teams — com time de reservas no banco ──────────────────


@pytest.mark.asyncio
async def test_get_teams_with_reserve_team_returns_reserves(api_client, mocker):
    """Times com position=0 são retornados como reservas."""
    match = _make_match(players_per_team=4)
    player_id = uuid4()

    # Time regular (position > 0)
    regular_team = MagicMock()
    regular_team.id = uuid4()
    regular_team.name = "Time A"
    regular_team.color = "#e53e3e"
    regular_team.position = 1
    regular_player = MagicMock()
    regular_player.player_id = player_id
    regular_player.is_reserve = False
    regular_player.player = MagicMock()
    regular_player.player.name = "Jogador A"
    regular_player.player.nickname = None
    regular_team.players = [regular_player]

    # Time de reservas (position = 0)
    reserve_team = MagicMock()
    reserve_team.id = uuid4()
    reserve_team.position = 0
    reserve_player = MagicMock()
    reserve_player.player_id = uuid4()
    reserve_player.player = MagicMock()
    reserve_player.player.name = "Reserva"
    reserve_player.player.nickname = None
    reserve_team.players = [reserve_player]

    mocker.patch(
        "app.api.v1.routers.teams.MatchRepository.get_with_attendances",
        new=AsyncMock(return_value=match),
    )
    mocker.patch(
        "app.api.v1.routers.teams.TeamRepository.get_by_match",
        new=AsyncMock(return_value=[regular_team, reserve_team]),
    )
    mocker.patch(
        "app.api.v1.routers.teams.GroupRepository.get_member_skills",
        new=AsyncMock(return_value={
            player_id: {"skill_stars": 4, "is_goalkeeper": False},
            reserve_player.player_id: {"skill_stars": 2, "is_goalkeeper": False},
        }),
    )

    response = await api_client.get(f"/api/v1/matches/{uuid4()}/teams")

    assert response.status_code == 200
    data = response.json()
    assert len(data["reserves"]) == 1
    assert data["reserves"][0]["name"] == "Reserva"
