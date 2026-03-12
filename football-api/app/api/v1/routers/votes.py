import asyncio
import uuid

from fastapi import APIRouter
from sqlalchemy import text
from zoneinfo import ZoneInfo

from app.core.dependencies import CurrentPlayer, DB

_BRT = ZoneInfo("America/Sao_Paulo")
from app.core.exceptions import ConflictError, ForbiddenError, NotFoundError, ValidationError
from app.db.repositories.match_repo import MatchRepository
from app.db.repositories.vote_repo import VoteRepository
from app.models.match import AttendanceStatus, MatchStatus
from app.models.player import PlayerRole
from app.schemas.vote import (
    VotePendingItem,
    VotePendingResponse,
    VoteResultsResponse,
    VoteStatusResponse,
    VoteSubmitRequest,
)
from app.services.push import send_push
from app.services.voting import time_until, voting_status, voting_window

router = APIRouter(tags=["votes"])


async def _get_match_or_404(match_id: uuid.UUID, db: DB):
    repo = MatchRepository(db)
    match = await repo.get_with_attendances(match_id)
    if not match:
        raise NotFoundError("Partida não encontrada")
    return match


def _confirmed_ids(match) -> list[uuid.UUID]:
    return [
        a.player_id
        for a in match.attendances
        if a.status == AttendanceStatus.CONFIRMED
    ]


@router.get("/matches/{match_id}/votes/status", response_model=VoteStatusResponse)
async def get_vote_status(match_id: uuid.UUID, db: DB, current: CurrentPlayer):
    match = await _get_match_or_404(match_id, db)

    status = voting_status(match)
    opens_at, closes_at = voting_window(match)

    confirmed_ids = _confirmed_ids(match)
    eligible_count = len(confirmed_ids)

    vote_repo = VoteRepository(db)
    voter_count = await vote_repo.voter_count(match_id)
    current_voted = await vote_repo.has_voted(match_id, current.id)

    # voted_player_ids — só retorna quando a votação está aberta ou encerrada
    voted_ids: list[uuid.UUID] = []
    if status in ("open", "closed"):
        voted_ids = await vote_repo.voter_ids(match_id)

    # Dispara push notification na primeira chamada com status 'open'
    if status == "open" and not match.vote_notified:
        match.vote_notified = True
        await db.flush()
        await asyncio.gather(*[
            send_push(
                db, pid,
                title="🏆 Votação aberta!",
                body="Escolha os melhores da pelada de hoje.",
                url=f"https://rachao.app/match/{match.hash}",
            )
            for pid in confirmed_ids
        ], return_exceptions=True)

    if status == "not_open":
        label = f"Abre em {time_until(opens_at)}"
    elif status == "open":
        label = f"Fecha em {time_until(closes_at)}"
    else:
        label = "Votação encerrada"

    return VoteStatusResponse(
        status=status,
        opens_at=opens_at,
        closes_at=closes_at,
        voter_count=voter_count,
        eligible_count=eligible_count,
        current_player_voted=current_voted,
        time_label=label,
        voted_player_ids=voted_ids,
        vote_open_delay_minutes=match.vote_open_delay_minutes,
    )


@router.post("/matches/{match_id}/votes", status_code=201)
async def submit_vote(match_id: uuid.UUID, body: VoteSubmitRequest, db: DB, current: CurrentPlayer):
    match = await _get_match_or_404(match_id, db)

    # Valida janela
    if voting_status(match) != "open":
        raise ForbiddenError("VOTING_CLOSED")

    # Super-admin não vota
    if current.role == PlayerRole.ADMIN:
        raise ForbiddenError("NOT_ELIGIBLE")

    # Valida elegibilidade (presença confirmada)
    confirmed_ids = _confirmed_ids(match)
    if current.id not in confirmed_ids:
        raise ForbiddenError("NOT_ELIGIBLE")

    # Já votou?
    vote_repo = VoteRepository(db)
    if await vote_repo.has_voted(match_id, current.id):
        raise ConflictError("ALREADY_VOTED")

    # Autovoto?
    top5_ids = {item.player_id for item in body.top5}
    if current.id in top5_ids or current.id == body.flop_player_id:
        raise ValidationError("SELF_VOTE")

    await vote_repo.submit(
        match_id=match_id,
        voter_id=current.id,
        top5=[{"player_id": item.player_id, "position": item.position} for item in body.top5],
        flop_player_id=body.flop_player_id,
    )

    return {"message": "Voto registrado com sucesso."}


@router.get("/votes/pending", response_model=VotePendingResponse)
async def get_pending_votes(db: DB, current: CurrentPlayer):
    """Votações abertas e pendentes para o jogador logado. Não retorna nada para admins."""
    if current.role == PlayerRole.ADMIN:
        return VotePendingResponse(items=[])

    result = await db.execute(
        text("""
            SELECT
                m.id         AS match_id,
                m.hash       AS match_hash,
                m.number     AS match_number,
                g.name       AS group_name,
                (SELECT COUNT(*)::int FROM match_votes mv2 WHERE mv2.match_id = m.id) AS voter_count,
                (SELECT COUNT(*)::int FROM attendances a2
                 WHERE a2.match_id = m.id AND a2.status = 'confirmed') AS eligible_count,
                (
                  (m.match_date + COALESCE(m.end_time, '23:59:00'::time))::timestamp
                  AT TIME ZONE 'America/Sao_Paulo'
                  + (m.vote_open_delay_minutes || ' minutes')::interval
                  + (m.vote_duration_hours || ' hours')::interval
                ) AS closes_at
            FROM matches m
            JOIN groups g ON g.id = m.group_id
            JOIN attendances a ON a.match_id = m.id
                AND a.player_id = :player_id
                AND a.status = 'confirmed'
            WHERE m.status = 'closed'
              AND NOT EXISTS (
                SELECT 1 FROM match_votes mv
                WHERE mv.match_id = m.id AND mv.voter_id = :player_id
              )
              AND (
                (m.match_date + COALESCE(m.end_time, '23:59:00'::time))::timestamp
                AT TIME ZONE 'America/Sao_Paulo'
                + (m.vote_open_delay_minutes || ' minutes')::interval
              ) <= NOW()
              AND (
                (m.match_date + COALESCE(m.end_time, '23:59:00'::time))::timestamp
                AT TIME ZONE 'America/Sao_Paulo'
                + (m.vote_open_delay_minutes || ' minutes')::interval
                + (m.vote_duration_hours || ' hours')::interval
              ) >= NOW()
            ORDER BY m.match_date DESC, m.start_time DESC
        """),
        {"player_id": current.id},
    )
    rows = result.mappings().all()
    items = []
    for row in rows:
        closes_at = row["closes_at"]
        if closes_at.tzinfo is None:
            closes_at = closes_at.replace(tzinfo=_BRT)
        items.append(VotePendingItem(
            match_id=row["match_id"],
            match_hash=row["match_hash"],
            match_number=row["match_number"],
            group_name=row["group_name"],
            time_label=f"Fecha em {time_until(closes_at)}",
            voter_count=row["voter_count"],
            eligible_count=row["eligible_count"],
        ))
    return VotePendingResponse(items=items)


@router.get("/matches/public/{match_hash}/votes/results", response_model=VoteResultsResponse, tags=["public"])
async def get_public_vote_results(match_hash: str, db: DB):
    """Resultados de votação públicos — disponíveis somente após o encerramento da votação."""
    m_repo = MatchRepository(db)
    match = await m_repo.get_by_hash_with_attendances(match_hash)
    if not match:
        raise NotFoundError("Partida não encontrada")
    if voting_status(match) != "closed":
        raise NotFoundError("Resultados não disponíveis")
    vote_repo = VoteRepository(db)
    data = await vote_repo.get_results(match.id)
    confirmed_ids = _confirmed_ids(match)
    return VoteResultsResponse(
        top5=data["top5"],
        flop=data["flop"],
        total_voters=data["total_voters"],
        eligible_voters=len(confirmed_ids),
    )


@router.get("/matches/{match_id}/votes/results", response_model=VoteResultsResponse)
async def get_vote_results(match_id: uuid.UUID, db: DB, current: CurrentPlayer):
    match = await _get_match_or_404(match_id, db)

    if voting_status(match) != "closed":
        raise ForbiddenError("RESULTS_NOT_AVAILABLE")

    vote_repo = VoteRepository(db)
    data = await vote_repo.get_results(match_id)

    confirmed_ids = _confirmed_ids(match)

    return VoteResultsResponse(
        top5=data["top5"],
        flop=data["flop"],
        total_voters=data["total_voters"],
        eligible_voters=len(confirmed_ids),
    )
