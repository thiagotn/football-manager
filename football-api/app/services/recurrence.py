import asyncio
import secrets
from datetime import date, datetime, timezone, timedelta

import structlog
from sqlalchemy.ext.asyncio import AsyncSession

from app.db.repositories.group_repo import GroupRepository
from app.db.repositories.match_repo import MatchRepository
from app.db.session import get_session_factory
from app.models.match import Attendance, AttendanceStatus, Match, MatchStatus
from app.services.push import send_push

_MONTHS_PT = ["jan","fev","mar","abr","mai","jun","jul","ago","set","out","nov","dez"]

def _fmt_date(d) -> str:
    return f"{d.day} de {_MONTHS_PT[d.month - 1]}"

logger = structlog.get_logger()


def _generate_hash() -> str:
    return secrets.token_urlsafe(8)[:10]


async def run_recurrence(session: AsyncSession) -> int:
    """
    Para cada grupo com recorrência ativa:
    - Se já há uma partida aberta, não faz nada.
    - Se a última partida já passou (match_date < hoje), cria a próxima
      com data = última + 7 dias, herdando configurações e convidados (status pending).

    Retorna o número de partidas criadas.
    """
    g_repo = GroupRepository(session)
    m_repo = MatchRepository(session)

    groups = await g_repo.get_groups_with_recurrence()
    created = 0
    today = datetime.now(timezone(timedelta(hours=-3))).date()

    for group in groups:
        if await m_repo.has_open_match(group.id):
            logger.info("recurrence_skipped_open_match", group_id=str(group.id))
            continue

        last_match = await m_repo.get_last_match(group.id)
        if not last_match:
            logger.info("recurrence_skipped_no_match", group_id=str(group.id))
            continue

        if last_match.match_date > today:
            logger.info("recurrence_skipped_future_match", group_id=str(group.id))
            continue

        if last_match.match_date == today and last_match.status != MatchStatus.CLOSED:
            logger.info("recurrence_skipped_today_not_closed", group_id=str(group.id))
            continue

        next_date = last_match.match_date + timedelta(days=7)

        # Garantia de hash único
        for _ in range(5):
            hash_ = _generate_hash()
            if not await m_repo.get_by_hash(hash_):
                break

        new_match = Match(
            group_id=group.id,
            match_date=next_date,
            start_time=last_match.start_time,
            end_time=last_match.end_time,
            location=last_match.location,
            address=last_match.address,
            court_type=last_match.court_type,
            players_per_team=last_match.players_per_team,
            max_players=last_match.max_players,
            notes=last_match.notes,
            hash=hash_,
            status=MatchStatus.OPEN,
            created_by_id=None,
        )
        session.add(new_match)
        await session.flush()

        player_ids = await m_repo.get_attendance_player_ids(last_match.id)
        for player_id in player_ids:
            session.add(Attendance(
                match_id=new_match.id,
                player_id=player_id,
                status=AttendanceStatus.PENDING,
            ))

        await session.flush()

        match_url = f"https://rachao.app/match/{hash_}"
        await asyncio.gather(*[
            send_push(
                session, pid,
                title=f"⚽ Novo rachão — {group.name}",
                body=f"Partida em {_fmt_date(next_date)}. Confirme sua presença!",
                url=match_url,
            )
            for pid in player_ids
        ], return_exceptions=True)

        created += 1
        logger.info(
            "recurrence_match_created",
            group_id=str(group.id),
            match_date=str(next_date),
            inherited_players=len(player_ids),
        )

    return created


async def _send_in_progress_pushes(session, m_repo: MatchRepository, candidates: list) -> None:
    """Envia push 'Bola rolando' para os jogadores confirmados dos candidatos a IN_PROGRESS."""
    for m in candidates:
        confirmed_ids = await m_repo.get_confirmed_player_ids(m.id)
        await asyncio.gather(*[
            send_push(
                session, pid,
                title=f"⚽ Bola rolando! — {m.group.name}",
                body="A partida de hoje já começou! 🎉",
                url=f"https://rachao.app/match/{m.hash}",
            )
            for pid in confirmed_ids
        ], return_exceptions=True)


async def run_status_sync_job() -> None:
    """
    Job horário — fecha partidas com data passada e transiciona para IN_PROGRESS.
    Não cria novos rachões (responsabilidade exclusiva do job das 07h).
    """
    session_factory = get_session_factory()
    async with session_factory() as session:
        try:
            m_repo = MatchRepository(session)
            in_progress_candidates = await m_repo.get_in_progress_candidates()
            closed = await m_repo.close_past_matches()
            await session.commit()
            if closed:
                logger.info("status_sync_auto_closed", matches_closed=closed)
            await _send_in_progress_pushes(session, m_repo, in_progress_candidates)
        except Exception:
            await session.rollback()
            logger.exception("status_sync_job_failed")


async def run_recurrence_job() -> None:
    """Job das 07h — fecha partidas passadas, cria próximos rachões e envia notificações."""
    session_factory = get_session_factory()
    async with session_factory() as session:
        try:
            m_repo = MatchRepository(session)
            in_progress_candidates = await m_repo.get_in_progress_candidates()
            closed = await m_repo.close_past_matches()
            if closed:
                logger.info("recurrence_auto_closed", matches_closed=closed)
            count = await run_recurrence(session)
            await session.commit()
            logger.info("recurrence_job_done", matches_created=count)
            await _send_in_progress_pushes(session, m_repo, in_progress_candidates)
        except Exception:
            await session.rollback()
            logger.exception("recurrence_job_failed")
