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


# ── GET /matches/{id}/votes/status — happy path ───────────────────────────────


@pytest.mark.asyncio
async def test_get_vote_status_returns_200(api_client, player_user, mocker):
    """Endpoint retorna status da votação para partida existente."""
    from datetime import datetime, timezone

    match = _make_match(player_ids=[player_user.id])
    match.vote_open_delay_minutes = 20
    match.vote_notified = True

    mocker.patch(
        "app.api.v1.routers.votes.MatchRepository.get_with_attendances",
        new=AsyncMock(return_value=match),
    )
    mocker.patch("app.api.v1.routers.votes.voting_status", return_value="not_open")

    now = datetime.now(timezone.utc)
    mocker.patch(
        "app.api.v1.routers.votes.voting_window",
        return_value=(now, now),
    )
    mocker.patch(
        "app.api.v1.routers.votes.VoteRepository.voter_count",
        new=AsyncMock(return_value=0),
    )
    mocker.patch(
        "app.api.v1.routers.votes.VoteRepository.has_voted",
        new=AsyncMock(return_value=False),
    )
    mocker.patch(
        "app.api.v1.routers.votes.time_until",
        return_value="2h 30m",
    )

    response = await api_client.get(f"/api/v1/matches/{uuid4()}/votes/status")

    assert response.status_code == 200
    data = response.json()
    assert "status" in data
    assert "voter_count" in data


# ── POST /matches/{id}/votes — happy path ────────────────────────────────────


@pytest.mark.asyncio
async def test_submit_vote_success_returns_201(api_client, player_user, mocker):
    """Jogador elegível pode submeter voto com sucesso."""
    other_player_id = uuid4()
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
    mocker.patch(
        "app.api.v1.routers.votes.VoteRepository.submit",
        new=AsyncMock(return_value=None),
    )

    body = {"top5": [{"player_id": str(other_player_id), "position": 1}]}
    response = await api_client.post(f"/api/v1/matches/{uuid4()}/votes", json=body)

    assert response.status_code == 201
    assert response.json()["message"] == "Voto registrado com sucesso."


# ── GET /votes/pending — happy path ──────────────────────────────────────────


@pytest.mark.asyncio
async def test_get_pending_votes_player_returns_200(api_client, player_user, mock_db, mocker):
    """Jogador comum recebe lista de votos pendentes."""
    mock_result = MagicMock()
    mock_result.mappings.return_value.all.return_value = []
    mock_db.execute = AsyncMock(return_value=mock_result)

    response = await api_client.get("/api/v1/votes/pending")

    assert response.status_code == 200
    assert response.json()["items"] == []


# ── GET /matches/public/{hash}/votes/results — happy path ────────────────────


@pytest.mark.asyncio
async def test_get_public_vote_results_closed_returns_200(api_client, mocker):
    """Resultados públicos disponíveis após encerramento da votação."""
    match = _make_match()

    mocker.patch(
        "app.api.v1.routers.votes.MatchRepository.get_by_hash_with_attendances",
        new=AsyncMock(return_value=match),
    )
    mocker.patch("app.api.v1.routers.votes.voting_status", return_value="closed")
    mocker.patch(
        "app.api.v1.routers.votes.VoteRepository.get_results",
        new=AsyncMock(return_value={"top5": [], "flop": [], "total_voters": 0}),
    )

    response = await api_client.get("/api/v1/matches/public/abc123/votes/results")

    assert response.status_code == 200
    data = response.json()
    assert "top5" in data
    assert "total_voters" in data


# ── POST /matches/{id}/votes/close — happy path ──────────────────────────────


@pytest.mark.asyncio
async def test_close_voting_early_group_admin_returns_200(api_client, mocker):
    """Admin do grupo pode encerrar a votação antes do prazo."""
    match = _make_match()

    g_member = MagicMock()
    g_member.role = "admin"

    mocker.patch(
        "app.api.v1.routers.votes.MatchRepository.get_with_attendances",
        new=AsyncMock(return_value=match),
    )
    mocker.patch(
        "app.api.v1.routers.votes.GroupRepository.get_member",
        new=AsyncMock(return_value=g_member),
    )
    mocker.patch("app.api.v1.routers.votes.voting_status", return_value="open")

    response = await api_client.post(f"/api/v1/matches/{uuid4()}/votes/close")

    assert response.status_code == 200
    assert response.json()["message"] == "Votação encerrada com sucesso."


@pytest.mark.asyncio
async def test_close_voting_early_not_open_returns_403(api_client, mocker):
    """Não é possível encerrar votação que não está aberta."""
    match = _make_match()

    g_member = MagicMock()
    g_member.role = "admin"

    mocker.patch(
        "app.api.v1.routers.votes.MatchRepository.get_with_attendances",
        new=AsyncMock(return_value=match),
    )
    mocker.patch(
        "app.api.v1.routers.votes.GroupRepository.get_member",
        new=AsyncMock(return_value=g_member),
    )
    mocker.patch("app.api.v1.routers.votes.voting_status", return_value="not_open")

    response = await api_client.post(f"/api/v1/matches/{uuid4()}/votes/close")

    assert response.status_code == 403
    assert response.json()["detail"] == "VOTING_NOT_OPEN"
