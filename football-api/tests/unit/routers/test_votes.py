"""
Testes unitários — routers/votes.py

Regras de negócio cobertas:
- POST /matches/{id}/votes votação fechada → 403 VOTING_CLOSED
- POST /matches/{id}/votes super admin não pode votar → 403 NOT_ELIGIBLE
- POST /matches/{id}/votes player não confirmado → 403 NOT_ELIGIBLE
- POST /matches/{id}/votes player já votou → 409 ALREADY_VOTED
- POST /matches/{id}/votes auto-voto → 422 SELF_VOTE
- GET /votes/pending admin retorna lista vazia imediatamente
- GET /matches/public/{hash}/votes/results não encontrado → 404
- GET /matches/public/{hash}/votes/results votação não encerrada → 404
"""
from unittest.mock import AsyncMock, MagicMock
from uuid import uuid4

import pytest

from app.models.match import AttendanceStatus, MatchStatus


# ── Helpers ───────────────────────────────────────────────────────────────────


def _make_match(player_ids: list | None = None) -> MagicMock:
    m = MagicMock()
    m.id = uuid4()
    m.hash = "abc123"
    m.status = MatchStatus.CLOSED
    m.match_date = "2026-03-01"
    m.end_time = "22:00"
    m.vote_open_delay_minutes = 20
    m.vote_duration_hours = 24
    m.vote_notified = True

    attendances = []
    for pid in (player_ids or []):
        a = MagicMock()
        a.player_id = pid
        a.status = AttendanceStatus.CONFIRMED
        attendances.append(a)
    m.attendances = attendances
    return m


_VOTE_BODY = {
    "top5": [{"player_id": str(uuid4()), "position": 1}],
}


# ── POST /matches/{id}/votes ──────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_submit_vote_voting_closed_returns_403(api_client, mocker):
    """Tentar votar fora da janela de votação retorna 403 VOTING_CLOSED."""
    match = _make_match()
    mocker.patch(
        "app.api.v1.routers.votes.MatchRepository.get_with_attendances",
        new=AsyncMock(return_value=match),
    )
    mocker.patch("app.api.v1.routers.votes.voting_status", return_value="not_open")

    response = await api_client.post(f"/api/v1/matches/{uuid4()}/votes", json=_VOTE_BODY)

    assert response.status_code == 403
    assert response.json()["detail"] == "VOTING_CLOSED"


@pytest.mark.asyncio
async def test_submit_vote_admin_not_eligible_returns_403(admin_client, mocker):
    """Super admin não pode votar."""
    match = _make_match()
    mocker.patch(
        "app.api.v1.routers.votes.MatchRepository.get_with_attendances",
        new=AsyncMock(return_value=match),
    )
    mocker.patch("app.api.v1.routers.votes.voting_status", return_value="open")

    response = await admin_client.post(f"/api/v1/matches/{uuid4()}/votes", json=_VOTE_BODY)

    assert response.status_code == 403
    assert response.json()["detail"] == "NOT_ELIGIBLE"


@pytest.mark.asyncio
async def test_submit_vote_player_not_confirmed_returns_403(api_client, mocker):
    """Jogador sem presença confirmada não pode votar."""
    match = _make_match(player_ids=[])  # nenhum confirmado
    mocker.patch(
        "app.api.v1.routers.votes.MatchRepository.get_with_attendances",
        new=AsyncMock(return_value=match),
    )
    mocker.patch("app.api.v1.routers.votes.voting_status", return_value="open")

    response = await api_client.post(f"/api/v1/matches/{uuid4()}/votes", json=_VOTE_BODY)

    assert response.status_code == 403
    assert response.json()["detail"] == "NOT_ELIGIBLE"


@pytest.mark.asyncio
async def test_submit_vote_already_voted_returns_409(api_client, player_user, mocker):
    """Votar duas vezes na mesma partida retorna 409 ALREADY_VOTED."""
    match = _make_match(player_ids=[player_user.id])
    mocker.patch(
        "app.api.v1.routers.votes.MatchRepository.get_with_attendances",
        new=AsyncMock(return_value=match),
    )
    mocker.patch("app.api.v1.routers.votes.voting_status", return_value="open")
    mocker.patch(
        "app.api.v1.routers.votes.VoteRepository.has_voted",
        new=AsyncMock(return_value=True),
    )

    response = await api_client.post(f"/api/v1/matches/{uuid4()}/votes", json=_VOTE_BODY)

    assert response.status_code == 409
    assert response.json()["detail"] == "ALREADY_VOTED"


@pytest.mark.asyncio
async def test_submit_vote_self_vote_returns_422(api_client, player_user, mocker):
    """Votar em si mesmo retorna 422 SELF_VOTE."""
    match = _make_match(player_ids=[player_user.id])
    mocker.patch(
        "app.api.v1.routers.votes.MatchRepository.get_with_attendances",
        new=AsyncMock(return_value=match),
    )
    mocker.patch("app.api.v1.routers.votes.voting_status", return_value="open")
    mocker.patch(
        "app.api.v1.routers.votes.VoteRepository.has_voted",
        new=AsyncMock(return_value=False),
    )

    body = {"top5": [{"player_id": str(player_user.id), "position": 1}]}
    response = await api_client.post(f"/api/v1/matches/{uuid4()}/votes", json=body)

    assert response.status_code == 422
    assert response.json()["detail"] == "SELF_VOTE"


# ── GET /votes/pending ────────────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_get_pending_votes_admin_returns_empty_list(admin_client):
    """Admin não tem votos pendentes — retorno imediato com lista vazia."""
    response = await admin_client.get("/api/v1/votes/pending")

    assert response.status_code == 200
    assert response.json()["items"] == []


# ── GET /matches/public/{hash}/votes/results ──────────────────────────────────


@pytest.mark.asyncio
async def test_get_public_vote_results_not_found_returns_404(api_client, mocker):
    mocker.patch(
        "app.api.v1.routers.votes.MatchRepository.get_by_hash_with_attendances",
        new=AsyncMock(return_value=None),
    )

    response = await api_client.get("/api/v1/matches/public/naoexiste/votes/results")

    assert response.status_code == 404


@pytest.mark.asyncio
async def test_get_public_vote_results_not_closed_returns_404(api_client, mocker):
    """Resultados só disponíveis após encerramento da votação."""
    match = _make_match()
    mocker.patch(
        "app.api.v1.routers.votes.MatchRepository.get_by_hash_with_attendances",
        new=AsyncMock(return_value=match),
    )
    mocker.patch("app.api.v1.routers.votes.voting_status", return_value="open")

    response = await api_client.get("/api/v1/matches/public/abc123/votes/results")

    assert response.status_code == 404
