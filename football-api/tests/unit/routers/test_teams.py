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
