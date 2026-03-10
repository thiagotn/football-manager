import asyncio
import uuid

from fastapi import APIRouter

from app.core.dependencies import CurrentPlayer, DB
from app.core.exceptions import ConflictError, ForbiddenError, NotFoundError, ValidationError
from app.db.repositories.match_repo import MatchRepository
from app.db.repositories.vote_repo import VoteRepository
from app.models.match import AttendanceStatus, MatchStatus
from app.models.player import PlayerRole
from app.schemas.vote import VoteResultsResponse, VoteStatusResponse, VoteSubmitRequest
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
