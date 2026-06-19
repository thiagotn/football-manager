"""Testes da regra `is_current` / `voting_status` em app/services/match_listing.py.

A regra `is_current` deve cobrir três sinais:
1. status aberto/em progresso
2. votação ainda aberta (mesmo com status closed)
3. ser a partida mais recente do grupo sem próxima criada
"""

from datetime import date, datetime, time, timedelta
from types import SimpleNamespace
from uuid import uuid4
from zoneinfo import ZoneInfo

import pytest

from app.models.match import MatchStatus
from app.services.match_listing import classify_matches

BRT = ZoneInfo("America/Sao_Paulo")


def _make_match(*, match_date, start_time=time(20, 0), end_time=time(22, 0),
                status=MatchStatus.OPEN, vote_delay=20, vote_hours=24):
    """Cria um stub minimal compatível com voting_status() e classify_matches()."""
    return SimpleNamespace(
        id=uuid4(),
        match_date=match_date,
        start_time=start_time,
        end_time=end_time,
        status=status,
        vote_open_delay_minutes=vote_delay,
        vote_duration_hours=vote_hours,
    )


def _result(matches, m_id):
    return classify_matches(matches)[str(m_id)]


@pytest.mark.asyncio
async def test_cenario_a_open_match_is_current():
    """Cenário A: status `open` no futuro → is_current=True, voting_status='not_open'."""
    future = date.today() + timedelta(days=7)
    m = _make_match(match_date=future, status=MatchStatus.OPEN)

    is_current, vstatus = _result([m], m.id)

    assert is_current is True
    assert vstatus == "not_open"


@pytest.mark.asyncio
async def test_cenario_b_closed_match_voting_still_open_is_current():
    """Cenário B: closed, end_time há 2h, voting ainda aberta → is_current=True."""
    now = datetime.now(BRT)
    # end há 2h → voting abre há 1h40min, fecha em ~22h → status 'open'
    end_dt = now - timedelta(hours=2)
    m = _make_match(
        match_date=end_dt.date(),
        start_time=time(end_dt.hour, end_dt.minute),
        end_time=time(end_dt.hour, end_dt.minute),
        status=MatchStatus.CLOSED,
    )

    is_current, vstatus = _result([m], m.id)

    assert is_current is True
    assert vstatus == "open"


@pytest.mark.asyncio
async def test_cenario_c_closed_voting_closed_with_future_match_is_not_current():
    """Cenário C: closed há 5 dias, voting fechada, OUTRO match aberto no grupo → is_current=False."""
    today = date.today()
    old = _make_match(
        match_date=today - timedelta(days=5),
        status=MatchStatus.CLOSED,
    )
    future = _make_match(
        match_date=today + timedelta(days=2),
        status=MatchStatus.OPEN,
    )

    is_current_old, vstatus_old = _result([old, future], old.id)
    is_current_future, _ = _result([old, future], future.id)

    assert is_current_old is False
    assert vstatus_old == "closed"
    assert is_current_future is True


@pytest.mark.asyncio
async def test_cenario_d_unique_closed_match_no_future_remains_current():
    """Cenário D: única match do grupo está closed + voting fechada → ainda is_current=True."""
    old = _make_match(
        match_date=date.today() - timedelta(days=5),
        status=MatchStatus.CLOSED,
    )

    is_current, vstatus = _result([old], old.id)

    assert is_current is True
    assert vstatus == "closed"


@pytest.mark.asyncio
async def test_cenario_e_two_closed_only_most_recent_is_current():
    """Cenário E: 2 closed sem voting nem próxima → só a mais recente é is_current."""
    today = date.today()
    older = _make_match(
        match_date=today - timedelta(days=10),
        status=MatchStatus.CLOSED,
    )
    recent = _make_match(
        match_date=today - timedelta(days=2),
        status=MatchStatus.CLOSED,
    )

    is_current_older, _ = _result([older, recent], older.id)
    is_current_recent, _ = _result([older, recent], recent.id)

    assert is_current_older is False
    assert is_current_recent is True


@pytest.mark.asyncio
async def test_empty_list_returns_empty_dict():
    assert classify_matches([]) == {}


@pytest.mark.asyncio
async def test_in_progress_status_is_current():
    """Sanidade: status in_progress sempre is_current=True, independente de outros sinais."""
    m = _make_match(
        match_date=date.today(),
        status=MatchStatus.IN_PROGRESS,
    )

    is_current, _ = _result([m], m.id)

    assert is_current is True
