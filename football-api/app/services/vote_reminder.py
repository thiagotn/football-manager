"""Issue #6 — lembrete push 30 min antes de fechar a janela de votação.

Para cada partida cuja votação está aberta e fechará nos próximos 30 minutos,
busca os jogadores confirmados que ainda não votaram e envia um push.
A coluna `matches.vote_reminder_sent_at` (migration 046) garante idempotência:
o mesmo match não recebe dois lembretes.

O job é registrado no scheduler do `app/main.py` para rodar a cada 5 minutos —
granularidade suficiente para o aviso de 30min sem spam.
"""
from datetime import datetime, timedelta, timezone

import structlog
from sqlalchemy import and_, exists, select
from sqlalchemy.ext.asyncio import AsyncSession

from app.db.session import get_session_factory
from app.models.match import Attendance, AttendanceStatus, Match, MatchStatus
from app.models.match_vote import MatchVote
from app.services.push import send_push
from app.services.voting import voting_status, voting_window

logger = structlog.get_logger()

REMINDER_LEAD_TIME = timedelta(minutes=30)


async def run_vote_reminder_job() -> None:
    session_factory = get_session_factory()
    async with session_factory() as session:
        try:
            sent = await _run(session)
            await session.commit()
            if sent:
                logger.info("vote_reminder_job_done", matches_notified=sent)
        except Exception:
            await session.rollback()
            logger.exception("vote_reminder_job_failed")


async def _run(session: AsyncSession) -> int:
    """Implementação testável (recebe a sessão)."""
    # Apenas partidas closed sem lembrete enviado ainda.
    candidates = (
        await session.execute(
            select(Match).where(
                Match.status == MatchStatus.CLOSED,
                Match.vote_reminder_sent_at.is_(None),
            )
        )
    ).scalars().all()

    now = datetime.now(timezone.utc)
    notified = 0
    for match in candidates:
        # `voting_status` / `voting_window` calculam em horário de São Paulo.
        # Comparações com `now` (UTC) abaixo são consistentes porque ambos são
        # aware datetimes.
        status = voting_status(match)
        if status != "open":
            continue
        _opens_at, closes_at = voting_window(match)
        delta = closes_at - now
        if delta > REMINDER_LEAD_TIME or delta <= timedelta(0):
            # Ou ainda falta mais de 30min, ou já fechou — não envia.
            continue

        pending_ids = await _confirmed_pending_voter_ids(session, match.id)
        if not pending_ids:
            # Ninguém faltando — marca como enviado pra não reavaliar.
            match.vote_reminder_sent_at = now
            continue

        group_name = match.group.name if match.group else ""
        title = f"🗳️ 30 min para fechar a votação — {group_name}".strip(" —")
        body = f"Vote agora no Rachão #{match.number}!"
        url = f"https://rachao.app/match/{match.hash}"
        for pid in pending_ids:
            try:
                await send_push(session, pid, title=title, body=body, url=url)
            except Exception:
                logger.exception("vote_reminder_push_failed", player_id=str(pid), match_id=str(match.id))

        match.vote_reminder_sent_at = now
        notified += 1

    return notified


async def _confirmed_pending_voter_ids(session: AsyncSession, match_id) -> list:
    """Confirmados na partida que NÃO têm voto registrado para ela."""
    result = await session.execute(
        select(Attendance.player_id)
        .where(
            Attendance.match_id == match_id,
            Attendance.status == AttendanceStatus.CONFIRMED,
            ~exists().where(
                and_(
                    MatchVote.match_id == match_id,
                    MatchVote.voter_id == Attendance.player_id,
                )
            ),
        )
    )
    return [row[0] for row in result.all()]
