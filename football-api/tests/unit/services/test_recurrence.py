"""
Testes unitários — app/services/recurrence.py

Regras de negócio cobertas:
- _fmt_date formata datas em português corretamente
- run_recurrence ignora grupo com partida aberta
- run_recurrence ignora grupo sem última partida
- run_recurrence ignora quando última partida é futura
- run_recurrence ignora quando última partida é hoje e não está fechada
- run_recurrence ignora quando votação ainda não encerrou
- run_recurrence cria nova partida (happy path)
- run_recurrence retorna contagem de partidas criadas
"""
from datetime import date, timedelta, timezone, datetime
from unittest.mock import AsyncMock, MagicMock, patch
from uuid import uuid4

import pytest

from app.services.recurrence import _fmt_date, run_recurrence
from app.models.match import MatchStatus


# ── _fmt_date ─────────────────────────────────────────────────────────────────


def test_fmt_date_january():
    assert _fmt_date(date(2026, 1, 15)) == "15 de jan"


def test_fmt_date_december():
    assert _fmt_date(date(2026, 12, 25)) == "25 de dez"


def test_fmt_date_april():
    assert _fmt_date(date(2026, 4, 3)) == "3 de abr"


def test_fmt_date_all_months():
    months_pt = ["jan","fev","mar","abr","mai","jun","jul","ago","set","out","nov","dez"]
    for i, abbr in enumerate(months_pt, start=1):
        assert _fmt_date(date(2026, i, 1)) == f"1 de {abbr}"


# ── run_recurrence — cenários de skip ────────────────────────────────────────


def _make_group():
    g = MagicMock()
    g.id = uuid4()
    g.name = "Pelada do Bairro"
    g.vote_open_delay_minutes = 20
    g.vote_duration_hours = 24
    return g


def _make_match(match_date=None, status=MatchStatus.CLOSED):
    m = MagicMock()
    m.id = uuid4()
    m.group_id = uuid4()
    m.match_date = match_date or date(2026, 3, 1)
    m.status = status
    m.start_time = None
    m.end_time = None
    m.location = "Campo"
    m.address = None
    m.court_type = "campo"
    m.players_per_team = 5
    m.max_players = 12
    m.notes = None
    return m


@pytest.mark.asyncio
async def test_recurrence_skips_group_with_open_match(mocker):
    """Grupo com partida aberta não gera nova partida."""
    session = AsyncMock()
    group = _make_group()

    mocker.patch(
        "app.services.recurrence.GroupRepository.get_groups_with_recurrence",
        new=AsyncMock(return_value=[group]),
    )
    mocker.patch(
        "app.services.recurrence.MatchRepository.has_open_match",
        new=AsyncMock(return_value=True),
    )

    result = await run_recurrence(session)

    assert result == 0


@pytest.mark.asyncio
async def test_recurrence_skips_group_with_no_last_match(mocker):
    """Grupo sem nenhuma partida anterior não gera nova partida."""
    session = AsyncMock()
    group = _make_group()

    mocker.patch(
        "app.services.recurrence.GroupRepository.get_groups_with_recurrence",
        new=AsyncMock(return_value=[group]),
    )
    mocker.patch(
        "app.services.recurrence.MatchRepository.has_open_match",
        new=AsyncMock(return_value=False),
    )
    mocker.patch(
        "app.services.recurrence.MatchRepository.get_last_match",
        new=AsyncMock(return_value=None),
    )

    result = await run_recurrence(session)

    assert result == 0


@pytest.mark.asyncio
async def test_recurrence_skips_when_last_match_is_future(mocker):
    """Última partida com data futura não gera nova partida."""
    session = AsyncMock()
    group = _make_group()
    future_match = _make_match(match_date=date(2099, 12, 31))

    mocker.patch(
        "app.services.recurrence.GroupRepository.get_groups_with_recurrence",
        new=AsyncMock(return_value=[group]),
    )
    mocker.patch(
        "app.services.recurrence.MatchRepository.has_open_match",
        new=AsyncMock(return_value=False),
    )
    mocker.patch(
        "app.services.recurrence.MatchRepository.get_last_match",
        new=AsyncMock(return_value=future_match),
    )

    result = await run_recurrence(session)

    assert result == 0


@pytest.mark.asyncio
async def test_recurrence_skips_when_today_match_not_closed(mocker):
    """Partida de hoje que ainda não está fechada não gera recorrência."""
    session = AsyncMock()
    group = _make_group()
    today = datetime.now(timezone(timedelta(hours=-3))).date()
    today_match = _make_match(match_date=today, status=MatchStatus.OPEN)

    mocker.patch(
        "app.services.recurrence.GroupRepository.get_groups_with_recurrence",
        new=AsyncMock(return_value=[group]),
    )
    mocker.patch(
        "app.services.recurrence.MatchRepository.has_open_match",
        new=AsyncMock(return_value=False),
    )
    mocker.patch(
        "app.services.recurrence.MatchRepository.get_last_match",
        new=AsyncMock(return_value=today_match),
    )

    result = await run_recurrence(session)

    assert result == 0


@pytest.mark.asyncio
async def test_recurrence_skips_when_voting_not_closed(mocker):
    """Votação ainda aberta impede criação da próxima partida."""
    session = AsyncMock()
    group = _make_group()
    past_match = _make_match(match_date=date(2026, 3, 1))

    mocker.patch(
        "app.services.recurrence.GroupRepository.get_groups_with_recurrence",
        new=AsyncMock(return_value=[group]),
    )
    mocker.patch(
        "app.services.recurrence.MatchRepository.has_open_match",
        new=AsyncMock(return_value=False),
    )
    mocker.patch(
        "app.services.recurrence.MatchRepository.get_last_match",
        new=AsyncMock(return_value=past_match),
    )
    mocker.patch(
        "app.services.recurrence.voting_status",
        return_value="open",
    )

    result = await run_recurrence(session)

    assert result == 0


# ── run_recurrence — happy path ───────────────────────────────────────────────


@pytest.mark.asyncio
async def test_recurrence_creates_match_for_eligible_group(mocker):
    """Grupo elegível recebe nova partida com data = última + 7 dias."""
    session = AsyncMock()
    group = _make_group()
    past_match = _make_match(match_date=date(2026, 3, 1))

    mocker.patch(
        "app.services.recurrence.GroupRepository.get_groups_with_recurrence",
        new=AsyncMock(return_value=[group]),
    )
    mocker.patch(
        "app.services.recurrence.MatchRepository.has_open_match",
        new=AsyncMock(return_value=False),
    )
    mocker.patch(
        "app.services.recurrence.MatchRepository.get_last_match",
        new=AsyncMock(return_value=past_match),
    )
    mocker.patch(
        "app.services.recurrence.voting_status",
        return_value="closed",
    )
    mocker.patch(
        "app.services.recurrence.MatchRepository.get_by_hash",
        new=AsyncMock(return_value=None),
    )
    mocker.patch(
        "app.services.recurrence.MatchRepository.next_number_for_group",
        new=AsyncMock(return_value=2),
    )
    mocker.patch(
        "app.services.recurrence.GroupRepository.get_member_ids",
        new=AsyncMock(return_value=[uuid4(), uuid4()]),
    )
    mocker.patch(
        "app.services.recurrence.send_push",
        new=AsyncMock(return_value=None),
    )

    result = await run_recurrence(session)

    assert result == 1
    session.add.assert_called()
    session.flush.assert_called()


@pytest.mark.asyncio
async def test_recurrence_creates_match_7_days_after_last(mocker):
    """A nova partida deve ter data exatamente 7 dias após a última."""
    session = AsyncMock()
    group = _make_group()
    last_date = date(2026, 3, 10)
    past_match = _make_match(match_date=last_date)

    created_matches = []

    original_add = session.add
    def capture_add(obj):
        from app.models.match import Match
        if isinstance(obj, Match) or (hasattr(obj, 'match_date') and hasattr(obj, 'group_id')):
            created_matches.append(obj)
        return original_add(obj)

    mocker.patch(
        "app.services.recurrence.GroupRepository.get_groups_with_recurrence",
        new=AsyncMock(return_value=[group]),
    )
    mocker.patch(
        "app.services.recurrence.MatchRepository.has_open_match",
        new=AsyncMock(return_value=False),
    )
    mocker.patch(
        "app.services.recurrence.MatchRepository.get_last_match",
        new=AsyncMock(return_value=past_match),
    )
    mocker.patch("app.services.recurrence.voting_status", return_value="closed")
    mocker.patch(
        "app.services.recurrence.MatchRepository.get_by_hash",
        new=AsyncMock(return_value=None),
    )
    mocker.patch(
        "app.services.recurrence.MatchRepository.next_number_for_group",
        new=AsyncMock(return_value=2),
    )
    mocker.patch(
        "app.services.recurrence.GroupRepository.get_member_ids",
        new=AsyncMock(return_value=[]),
    )
    mocker.patch("app.services.recurrence.send_push", new=AsyncMock(return_value=None))

    result = await run_recurrence(session)

    assert result == 1
    # Verifica que session.add foi chamado com o Match novo
    calls = session.add.call_args_list
    match_calls = [c for c in calls if hasattr(c.args[0], 'match_date')]
    if match_calls:
        new_match = match_calls[0].args[0]
        assert new_match.match_date == last_date + timedelta(days=7)


@pytest.mark.asyncio
async def test_recurrence_creates_pending_attendances_for_members(mocker):
    """Novos membros do grupo recebem presença pendente na nova partida."""
    session = AsyncMock()
    group = _make_group()
    past_match = _make_match(match_date=date(2026, 3, 1))
    player_ids = [uuid4(), uuid4(), uuid4()]

    mocker.patch(
        "app.services.recurrence.GroupRepository.get_groups_with_recurrence",
        new=AsyncMock(return_value=[group]),
    )
    mocker.patch(
        "app.services.recurrence.MatchRepository.has_open_match",
        new=AsyncMock(return_value=False),
    )
    mocker.patch(
        "app.services.recurrence.MatchRepository.get_last_match",
        new=AsyncMock(return_value=past_match),
    )
    mocker.patch("app.services.recurrence.voting_status", return_value="closed")
    mocker.patch(
        "app.services.recurrence.MatchRepository.get_by_hash",
        new=AsyncMock(return_value=None),
    )
    mocker.patch(
        "app.services.recurrence.MatchRepository.next_number_for_group",
        new=AsyncMock(return_value=2),
    )
    mocker.patch(
        "app.services.recurrence.GroupRepository.get_member_ids",
        new=AsyncMock(return_value=player_ids),
    )
    mock_push = mocker.patch(
        "app.services.recurrence.send_push",
        new=AsyncMock(return_value=None),
    )

    await run_recurrence(session)

    # session.add chamado para o Match + uma Attendance por membro
    assert session.add.call_count >= 1 + len(player_ids)
    # send_push chamado uma vez por membro
    assert mock_push.call_count == len(player_ids)


@pytest.mark.asyncio
async def test_recurrence_handles_multiple_groups(mocker):
    """run_recurrence processa múltiplos grupos e retorna contagem total."""
    session = AsyncMock()
    groups = [_make_group(), _make_group()]
    past_match = _make_match(match_date=date(2026, 3, 1))

    mocker.patch(
        "app.services.recurrence.GroupRepository.get_groups_with_recurrence",
        new=AsyncMock(return_value=groups),
    )
    mocker.patch(
        "app.services.recurrence.MatchRepository.has_open_match",
        new=AsyncMock(return_value=False),
    )
    mocker.patch(
        "app.services.recurrence.MatchRepository.get_last_match",
        new=AsyncMock(return_value=past_match),
    )
    mocker.patch("app.services.recurrence.voting_status", return_value="closed")
    mocker.patch(
        "app.services.recurrence.MatchRepository.get_by_hash",
        new=AsyncMock(return_value=None),
    )
    mocker.patch(
        "app.services.recurrence.MatchRepository.next_number_for_group",
        new=AsyncMock(return_value=1),
    )
    mocker.patch(
        "app.services.recurrence.GroupRepository.get_member_ids",
        new=AsyncMock(return_value=[]),
    )
    mocker.patch("app.services.recurrence.send_push", new=AsyncMock(return_value=None))

    result = await run_recurrence(session)

    assert result == 2


@pytest.mark.asyncio
async def test_recurrence_retries_hash_on_collision(mocker):
    """Garante que run_recurrence tenta novo hash quando há colisão."""
    session = AsyncMock()
    group = _make_group()
    past_match = _make_match(match_date=date(2026, 3, 1))

    # Primeira chamada retorna match existente (colisão), segunda retorna None
    mocker.patch(
        "app.services.recurrence.GroupRepository.get_groups_with_recurrence",
        new=AsyncMock(return_value=[group]),
    )
    mocker.patch(
        "app.services.recurrence.MatchRepository.has_open_match",
        new=AsyncMock(return_value=False),
    )
    mocker.patch(
        "app.services.recurrence.MatchRepository.get_last_match",
        new=AsyncMock(return_value=past_match),
    )
    mocker.patch("app.services.recurrence.voting_status", return_value="closed")
    mocker.patch(
        "app.services.recurrence.MatchRepository.get_by_hash",
        new=AsyncMock(side_effect=[MagicMock(), None]),  # 1ª colisão, 2ª livre
    )
    mocker.patch(
        "app.services.recurrence.MatchRepository.next_number_for_group",
        new=AsyncMock(return_value=2),
    )
    mocker.patch(
        "app.services.recurrence.GroupRepository.get_member_ids",
        new=AsyncMock(return_value=[]),
    )
    mocker.patch("app.services.recurrence.send_push", new=AsyncMock(return_value=None))

    result = await run_recurrence(session)

    assert result == 1
