import secrets
from datetime import date, timedelta

import structlog
from sqlalchemy.ext.asyncio import AsyncSession

from app.db.repositories.group_repo import GroupRepository
from app.db.repositories.match_repo import MatchRepository
from app.db.session import get_session_factory
from app.models.match import Attendance, AttendanceStatus, Match, MatchStatus

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
    today = date.today()

    for group in groups:
        if await m_repo.has_open_match(group.id):
            logger.info("recurrence_skipped_open_match", group_id=str(group.id))
            continue

        last_match = await m_repo.get_last_match(group.id)
        if not last_match:
            logger.info("recurrence_skipped_no_match", group_id=str(group.id))
            continue

        if last_match.match_date >= today:
            logger.info("recurrence_skipped_future_match", group_id=str(group.id))
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
        created += 1
        logger.info(
            "recurrence_match_created",
            group_id=str(group.id),
            match_date=str(next_date),
            inherited_players=len(player_ids),
        )

    return created


async def run_recurrence_job() -> None:
    """Entry point chamado pelo scheduler — gerencia sua própria sessão."""
    session_factory = get_session_factory()
    async with session_factory() as session:
        try:
            m_repo = MatchRepository(session)
            closed = await m_repo.close_past_matches()
            if closed:
                logger.info("recurrence_auto_closed", matches_closed=closed)

            count = await run_recurrence(session)
            await session.commit()
            logger.info("recurrence_job_done", matches_created=count)
        except Exception:
            await session.rollback()
            logger.exception("recurrence_job_failed")
