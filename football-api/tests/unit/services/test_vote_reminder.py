"""Testes unitários — app/services/vote_reminder.py (issue #6).

Cenários cobertos:
- Sem partidas elegíveis → 0 push.
- Partida com janela > 30min para fechar → não notifica.
- Partida com janela já fechada (delta <= 0) → não notifica.
- vote_reminder_sent_at preenchido → não entra no candidato.
- 1 partida com 3 confirmados (2 já votaram, 1 pendente) → 1 push.
- Ninguém pendente → marca como enviado pra não reavaliar.
"""
from datetime import date, datetime, time, timedelta, timezone
from unittest.mock import AsyncMock, MagicMock, patch
from uuid import uuid4

import pytest

from app.models.match import MatchStatus
from app.services.vote_reminder import _run

BRT_OFFSET = timezone(timedelta(hours=-3))


def _make_match(*, status=MatchStatus.CLOSED, end_time_minutes_ago=0,
                reminder_sent=None, group_name="Pelada"):
    """Constrói um Match mock com end_time em BRT recente.

    end_time_minutes_ago: quantos minutos antes de "agora BRT" o `end_time` da
    partida cai. Combinado com `vote_open_delay_minutes=20` e
    `vote_duration_hours=24`, controla onde a janela de votação está.
    """
    now_brt = datetime.now(BRT_OFFSET)
    end_dt = now_brt - timedelta(minutes=end_time_minutes_ago)

    m = MagicMock()
    m.id = uuid4()
    m.number = 1
    m.hash = "abc1234567"
    m.status = status
    m.match_date = end_dt.date()
    m.start_time = time(end_dt.hour, end_dt.minute)
    m.end_time = time(end_dt.hour, end_dt.minute)
    m.vote_open_delay_minutes = 20
    m.vote_duration_hours = 24
    m.vote_reminder_sent_at = reminder_sent
    m.group = MagicMock()
    m.group.name = group_name
    return m


def _mock_session(matches: list):
    session = MagicMock()
    scalars = MagicMock()
    scalars.all.return_value = matches
    result = MagicMock()
    result.scalars.return_value = scalars
    session.execute = AsyncMock(return_value=result)
    return session


@pytest.mark.asyncio
async def test_no_eligible_matches_returns_zero():
    session = _mock_session([])
    sent = await _run(session)
    assert sent == 0


@pytest.mark.asyncio
async def test_match_with_window_far_from_close_is_skipped():
    """end_time foi há 1h → fecha em ~23h. Lead time 30min não atingido."""
    match = _make_match(end_time_minutes_ago=60)
    session = _mock_session([match])
    with patch("app.services.vote_reminder.send_push", new=AsyncMock()) as mock_push:
        sent = await _run(session)
    assert sent == 0
    mock_push.assert_not_awaited()


@pytest.mark.asyncio
async def test_match_with_window_already_closed_is_skipped():
    """end_time foi há 25h (janela fechou há 30min) → ignora."""
    match = _make_match(end_time_minutes_ago=25 * 60)
    session = _mock_session([match])
    with patch("app.services.vote_reminder.send_push", new=AsyncMock()) as mock_push:
        sent = await _run(session)
    assert sent == 0
    mock_push.assert_not_awaited()


@pytest.mark.asyncio
async def test_match_in_reminder_window_with_one_pending_sends_push():
    """end_time foi há 23h50min:
    opens_at = end + 20min → 23h30min atrás
    closes_at = opens_at + 24h → daqui a 30min10s
    Cai dentro de REMINDER_LEAD_TIME (30min)."""
    match = _make_match(end_time_minutes_ago=(23 * 60 + 50))
    pending_player = uuid4()

    session = MagicMock()
    # Primeiro execute: select Match
    matches_scalars = MagicMock()
    matches_scalars.all.return_value = [match]
    matches_result = MagicMock()
    matches_result.scalars.return_value = matches_scalars
    # Segundo execute: select pending voter IDs (rows)
    pending_result = MagicMock()
    pending_result.all.return_value = [(pending_player,)]
    session.execute = AsyncMock(side_effect=[matches_result, pending_result])

    with patch("app.services.vote_reminder.send_push", new=AsyncMock()) as mock_push:
        sent = await _run(session)

    assert sent == 1
    mock_push.assert_awaited_once()
    assert mock_push.await_args.args[1] == pending_player
    assert match.vote_reminder_sent_at is not None


@pytest.mark.asyncio
async def test_no_pending_voters_marks_as_sent_without_push():
    """Ninguém pendente — marca para evitar reavaliação contínua."""
    match = _make_match(end_time_minutes_ago=(23 * 60 + 50))

    session = MagicMock()
    matches_scalars = MagicMock()
    matches_scalars.all.return_value = [match]
    matches_result = MagicMock()
    matches_result.scalars.return_value = matches_scalars
    pending_result = MagicMock()
    pending_result.all.return_value = []  # ninguém pendente
    session.execute = AsyncMock(side_effect=[matches_result, pending_result])

    with patch("app.services.vote_reminder.send_push", new=AsyncMock()) as mock_push:
        sent = await _run(session)

    assert sent == 0
    mock_push.assert_not_awaited()
    assert match.vote_reminder_sent_at is not None  # marca pra não reentrar


@pytest.mark.asyncio
async def test_match_with_voting_not_open_yet_is_skipped():
    """end_time agora → voting_status='not_open'. Pula."""
    match = _make_match(end_time_minutes_ago=0)
    session = _mock_session([match])
    with patch("app.services.vote_reminder.send_push", new=AsyncMock()) as mock_push:
        sent = await _run(session)
    assert sent == 0
    mock_push.assert_not_awaited()


# ── heartbeat do job (app/core/job_metrics.py) ────────────────────────────────

import os
import time as _time

from prometheus_client import REGISTRY

from app.core.job_metrics import JOB_VOTE_REMINDER
from app.services.vote_reminder import run_vote_reminder_job

_PID = str(os.getpid())


def _gauge():
    return REGISTRY.get_sample_value(
        "scheduler_job_last_success_timestamp_seconds",
        {"job_name": JOB_VOTE_REMINDER, "pid": _PID},
    )


def _counter():
    return REGISTRY.get_sample_value(
        "scheduler_job_failures_total", {"job_name": JOB_VOTE_REMINDER, "pid": _PID}
    ) or 0.0


def _patch_job_session(mocker):
    session = MagicMock()
    session.commit = AsyncMock()
    session.rollback = AsyncMock()
    factory = MagicMock()
    factory.return_value.__aenter__ = AsyncMock(return_value=session)
    factory.return_value.__aexit__ = AsyncMock(return_value=False)
    mocker.patch("app.services.vote_reminder.get_session_factory", return_value=factory)
    return session


@pytest.mark.asyncio
async def test_job_records_heartbeat_even_with_zero_sent(mocker):
    """Heartbeat de liveness: avança mesmo quando nenhum push foi enviado."""
    _patch_job_session(mocker)
    mocker.patch("app.services.vote_reminder._run", new=AsyncMock(return_value=0))

    t0 = _time.time()
    await run_vote_reminder_job()

    assert _gauge() is not None
    assert _gauge() >= t0


@pytest.mark.asyncio
async def test_job_records_failure_without_raising(mocker):
    session = _patch_job_session(mocker)
    mocker.patch(
        "app.services.vote_reminder._run",
        new=AsyncMock(side_effect=RuntimeError("pooler saturado")),
    )

    gauge_before = _gauge()
    failures_before = _counter()

    await run_vote_reminder_job()  # não deve propagar a exceção

    assert _counter() == failures_before + 1
    assert _gauge() == gauge_before  # heartbeat NÃO avança em falha
    session.rollback.assert_awaited_once()
