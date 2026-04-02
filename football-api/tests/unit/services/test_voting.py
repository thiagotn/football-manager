"""
Testes unitários — app/services/voting.py

Cobre: cálculo da janela de votação, status (not_open/open/closed) e pontuação.
Não requer banco de dados nem HTTP.
"""
from datetime import date, datetime, time
from unittest.mock import MagicMock
from zoneinfo import ZoneInfo

import pytest

import app.services.voting as voting_module
from app.services.voting import POINTS, voting_window, voting_status

BRT = ZoneInfo("America/Sao_Paulo")


def make_match(
    match_date=None,
    end_time=None,
    delay_minutes: int = 20,
    duration_hours: int = 24,
) -> MagicMock:
    m = MagicMock()
    m.match_date = match_date or date(2026, 3, 10)
    m.end_time = end_time
    m.vote_open_delay_minutes = delay_minutes
    m.vote_duration_hours = duration_hours
    return m


# ── voting_window ─────────────────────────────────────────────────────────────


def test_voting_window_opens_after_end_time():
    match = make_match(match_date=date(2026, 3, 10), end_time=time(22, 0), delay_minutes=20)
    opens_at, _ = voting_window(match)
    expected = datetime(2026, 3, 10, 22, 20, tzinfo=BRT)
    assert opens_at == expected


def test_voting_window_closes_after_duration():
    match = make_match(match_date=date(2026, 3, 10), end_time=time(22, 0), delay_minutes=20, duration_hours=24)
    opens_at, closes_at = voting_window(match)
    from datetime import timedelta
    assert closes_at == opens_at + timedelta(hours=24)


def test_voting_window_uses_2359_when_no_end_time():
    """Sem end_time definido, deve usar 23:59 como horário de encerramento."""
    match = make_match(match_date=date(2026, 3, 10), end_time=None, delay_minutes=20)
    opens_at, _ = voting_window(match)
    # 23:59 + 20min = 00:19 do dia seguinte
    expected = datetime(2026, 3, 11, 0, 19, tzinfo=BRT)
    assert opens_at == expected


def test_voting_window_respects_custom_delay():
    match = make_match(match_date=date(2026, 3, 10), end_time=time(20, 0), delay_minutes=60)
    opens_at, _ = voting_window(match)
    expected = datetime(2026, 3, 10, 21, 0, tzinfo=BRT)
    assert opens_at == expected


# ── voting_status ─────────────────────────────────────────────────────────────


class _FakeDatetime:
    """Substitui datetime.now() com um valor fixo para testes de voting_status."""

    def __init__(self, fake_now: datetime):
        self._now = fake_now

    def now(self, tz=None):
        return self._now

    def combine(self, *args, **kwargs):
        return datetime.combine(*args, **kwargs)


def test_voting_status_not_open(monkeypatch):
    match = make_match(match_date=date(2026, 3, 10), end_time=time(22, 0), delay_minutes=20)
    # Opens at 22:20 — now is 20:00, before window
    fake_now = datetime(2026, 3, 10, 20, 0, tzinfo=BRT)
    monkeypatch.setattr(voting_module, "datetime", _FakeDatetime(fake_now))
    assert voting_status(match) == "not_open"


def test_voting_status_open(monkeypatch):
    match = make_match(match_date=date(2026, 3, 10), end_time=time(22, 0), delay_minutes=20, duration_hours=24)
    # Opens at 22:20, closes at next day 22:20 — now is 23:00, inside window
    fake_now = datetime(2026, 3, 10, 23, 0, tzinfo=BRT)
    monkeypatch.setattr(voting_module, "datetime", _FakeDatetime(fake_now))
    assert voting_status(match) == "open"


def test_voting_status_closed(monkeypatch):
    match = make_match(match_date=date(2026, 3, 10), end_time=time(22, 0), delay_minutes=20, duration_hours=24)
    # Closes at 2026-03-11 22:20 — now is 2026-03-12
    fake_now = datetime(2026, 3, 12, 10, 0, tzinfo=BRT)
    monkeypatch.setattr(voting_module, "datetime", _FakeDatetime(fake_now))
    assert voting_status(match) == "closed"


# ── POINTS ────────────────────────────────────────────────────────────────────


def test_points_first_place_is_highest():
    assert POINTS[1] == 10


def test_points_fifth_place_is_lowest():
    assert POINTS[5] == 2


def test_points_descending_order():
    for pos in range(1, 5):
        assert POINTS[pos] > POINTS[pos + 1]


def test_points_covers_all_positions():
    assert set(POINTS.keys()) == {1, 2, 3, 4, 5}


# ── time_until ────────────────────────────────────────────────────────────────


def test_time_until_hours_and_minutes(monkeypatch):
    """Retorna 'Xh Ymin' quando faltam horas e minutos."""
    from datetime import timedelta
    from zoneinfo import ZoneInfo
    import app.services.voting as voting_module

    fake_now = datetime(2026, 3, 10, 20, 0, tzinfo=ZoneInfo("America/Sao_Paulo"))
    monkeypatch.setattr(voting_module, "datetime", _FakeDatetime(fake_now))

    target = datetime(2026, 3, 10, 22, 30, tzinfo=ZoneInfo("America/Sao_Paulo"))
    result = voting_module.time_until(target)

    assert result == "2h 30min"


def test_time_until_exact_hours(monkeypatch):
    """Retorna 'Xh' sem minutos quando diff é exatamente N horas."""
    from zoneinfo import ZoneInfo
    import app.services.voting as voting_module

    fake_now = datetime(2026, 3, 10, 20, 0, tzinfo=ZoneInfo("America/Sao_Paulo"))
    monkeypatch.setattr(voting_module, "datetime", _FakeDatetime(fake_now))

    target = datetime(2026, 3, 10, 23, 0, tzinfo=ZoneInfo("America/Sao_Paulo"))
    result = voting_module.time_until(target)

    assert result == "3h"


def test_time_until_only_minutes(monkeypatch):
    """Retorna 'Xmin' quando faltam menos de 60 minutos."""
    from zoneinfo import ZoneInfo
    import app.services.voting as voting_module

    fake_now = datetime(2026, 3, 10, 22, 0, tzinfo=ZoneInfo("America/Sao_Paulo"))
    monkeypatch.setattr(voting_module, "datetime", _FakeDatetime(fake_now))

    target = datetime(2026, 3, 10, 22, 45, tzinfo=ZoneInfo("America/Sao_Paulo"))
    result = voting_module.time_until(target)

    assert result == "45min"


def test_time_until_past_target_returns_zero(monkeypatch):
    """Target no passado retorna '0min'."""
    from zoneinfo import ZoneInfo
    import app.services.voting as voting_module

    fake_now = datetime(2026, 3, 10, 23, 0, tzinfo=ZoneInfo("America/Sao_Paulo"))
    monkeypatch.setattr(voting_module, "datetime", _FakeDatetime(fake_now))

    target = datetime(2026, 3, 10, 22, 0, tzinfo=ZoneInfo("America/Sao_Paulo"))
    result = voting_module.time_until(target)

    assert result == "0min"
