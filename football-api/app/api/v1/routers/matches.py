import asyncio
import secrets
import uuid
from datetime import date, datetime, timezone, timedelta
from typing import Annotated

from fastapi import APIRouter, Query

from app.core.dependencies import DB, CurrentPlayer, OptionalPlayer
from app.core.exceptions import ConflictError, ForbiddenError, NotFoundError
from app.db.repositories.group_repo import GroupRepository
from app.db.repositories.match_repo import MatchRepository
from app.db.repositories.match_stats_repo import MatchStatsRepository
from app.models.match import AttendanceStatus, Match, MatchStatus
from app.models.player import PlayerRole
from app.models.group import GroupMemberRole
from app.services.recurrence import run_recurrence
from app.services.push import send_push

_MONTHS_PT = ["jan","fev","mar","abr","mai","jun","jul","ago","set","out","nov","dez"]

def _fmt_date(d) -> str:
    return f"{d.day} de {_MONTHS_PT[d.month - 1]}"
from app.schemas.match import (
    AttendanceResponse,
    DiscoverMatchResponse,
    MatchCreate,
    MatchDetailResponse,
    MatchPlayerStatsRequest,
    MatchPlayerStatsResponse,
    MatchResponse,
    MatchUpdate,
    PlayerStatResponse,
    SetAttendanceRequest,
)

router = APIRouter(tags=["matches"])


def _generate_hash() -> str:
    return secrets.token_urlsafe(8)[:10]


def _build_detail(match: Match, position_map: dict | None = None) -> MatchDetailResponse:
    # Exclui o super admin (role=admin) das listas de presença
    attendances = [a for a in match.attendances if a.player.role != PlayerRole.ADMIN]
    confirmed = [a for a in attendances if a.status == AttendanceStatus.CONFIRMED]
    declined = [a for a in attendances if a.status == AttendanceStatus.DECLINED]
    pending = [a for a in attendances if a.status == AttendanceStatus.PENDING]

    def _attendance(a) -> AttendanceResponse:
        r = AttendanceResponse.model_validate(a)
        if position_map:
            r = r.model_copy(update={"position": position_map.get(str(a.player.id))})
        return r

    return MatchDetailResponse(
        id=match.id,
        number=match.number,
        group_id=match.group_id,
        match_date=match.match_date,
        start_time=match.start_time,
        end_time=match.end_time,
        location=match.location,
        address=match.address,
        court_type=match.court_type,
        players_per_team=match.players_per_team,
        max_players=match.max_players,
        notes=match.notes,
        hash=match.hash,
        status=match.status,
        created_at=match.created_at,
        updated_at=match.updated_at,
        attendances=[_attendance(a) for a in attendances],
        confirmed_count=len(confirmed),
        declined_count=len(declined),
        pending_count=len(pending),
        group_name=match.group.name if match.group else "",
        group_per_match_amount=match.group.per_match_amount if match.group else None,
        group_monthly_amount=match.group.monthly_amount if match.group else None,
        group_is_public=match.group.is_public if match.group else True,
        group_timezone=match.group.timezone if match.group else "America/Sao_Paulo",
    )


# ── Discover feed ─────────────────────────────────────────────────────────────

@router.get("/matches/discover", response_model=list[DiscoverMatchResponse])
async def discover_matches(
    db: DB,
    current: OptionalPlayer,
    date_from: Annotated[date | None, Query()] = None,
    date_to: Annotated[date | None, Query()] = None,
    court_type: Annotated[list[str] | None, Query()] = None,
    weekday: Annotated[list[int] | None, Query()] = None,
    limit: Annotated[int, Query(ge=1, le=50)] = 20,
    offset: Annotated[int, Query(ge=0)] = 0,
):
    """Partidas abertas de grupos públicos. Não requer autenticação.
    Jogadores autenticados não veem partidas de grupos onde já são membros."""
    if current is not None and current.role == PlayerRole.ADMIN:
        return []
    player_id = current.id if current is not None else None
    m_repo = MatchRepository(db)
    rows = await m_repo.get_discover_matches(
        player_id=player_id,
        date_from=date_from,
        date_to=date_to,
        court_types=court_type,
        weekdays=weekday,
        limit=limit,
        offset=offset,
    )
    result = []
    for row in rows:
        m = row["match"]
        confirmed = row["confirmed_count"]
        result.append(DiscoverMatchResponse(
            id=m.id,
            hash=m.hash,
            number=m.number,
            match_date=m.match_date,
            start_time=m.start_time,
            end_time=m.end_time,
            location=m.location,
            address=m.address,
            court_type=m.court_type,
            players_per_team=m.players_per_team,
            max_players=m.max_players,
            notes=m.notes,
            group_id=m.group_id,
            group_name=row["group_name"],
            confirmed_count=confirmed,
            spots_left=(m.max_players - confirmed) if m.max_players else None,
            group_timezone=row.get("group_timezone", "America/Sao_Paulo"),
        ))
    return result


# ── Public endpoint (no auth) ─────────────────────────────────────────────────

@router.get("/matches/public/{match_hash}", response_model=MatchDetailResponse, tags=["public"])
async def get_match_public(match_hash: str, db: DB):
    """Endpoint público para visualizar uma partida via hash/link único."""
    repo = MatchRepository(db)
    match = await repo.get_by_hash_with_attendances(match_hash)
    if not match:
        raise NotFoundError("Partida não encontrada")
    player_ids = [a.player.id for a in match.attendances if a.player.role != PlayerRole.ADMIN]
    g_repo = GroupRepository(db)
    skills = await g_repo.get_member_skills(match.group_id, player_ids)
    position_map = {str(pid): data["position"] for pid, data in skills.items()}
    return _build_detail(match, position_map)


# ── Group matches ─────────────────────────────────────────────────────────────

@router.get("/groups/{group_id}/matches", response_model=list[MatchResponse])
async def list_group_matches(group_id: uuid.UUID, db: DB, current: CurrentPlayer):
    g_repo = GroupRepository(db)
    group = await g_repo.get(group_id)
    if not group:
        raise NotFoundError("Grupo não encontrado")

    if current.role != PlayerRole.ADMIN:
        member = await g_repo.get_member(group_id, current.id)
        if not member:
            raise ForbiddenError("Você não é membro deste grupo")

    repo = MatchRepository(db)
    in_progress_candidates = await repo.get_in_progress_candidates()
    closed = await repo.close_past_matches()
    if closed:
        await run_recurrence(db)

    if in_progress_candidates:
        for m in in_progress_candidates:
            confirmed_ids = await repo.get_confirmed_player_ids(m.id)
            await asyncio.gather(*[
                send_push(
                    db, pid,
                    title=f"⚽ Bola rolando! — {m.group.name}",
                    body="A partida de hoje já começou! 🎉",
                    url=f"https://rachao.app/match/{m.hash}",
                )
                for pid in confirmed_ids
            ], return_exceptions=True)

    return await repo.get_group_matches(group_id)


@router.post("/groups/{group_id}/matches", response_model=MatchResponse, status_code=201)
async def create_match(group_id: uuid.UUID, body: MatchCreate, db: DB, current: CurrentPlayer):
    g_repo = GroupRepository(db)
    group = await g_repo.get(group_id)
    if not group:
        raise NotFoundError("Grupo não encontrado")

    if current.role != PlayerRole.ADMIN:
        member = await g_repo.get_member(group_id, current.id)
        if not member or member.role != GroupMemberRole.ADMIN:
            raise ForbiddenError("Apenas admins do grupo podem criar partidas")

    m_repo = MatchRepository(db)
    # Ensure unique hash
    for _ in range(5):
        hash_ = _generate_hash()
        if not await m_repo.get_by_hash(hash_):
            break

    match = await m_repo.create(
        group_id=group_id,
        number=await m_repo.next_number_for_group(group_id),
        match_date=body.match_date,
        start_time=body.start_time,
        end_time=body.end_time,
        location=body.location,
        address=body.address,
        court_type=body.court_type,
        players_per_team=body.players_per_team,
        max_players=body.max_players,
        notes=body.notes,
        hash=hash_,
        created_by_id=current.id,
        vote_open_delay_minutes=group.vote_open_delay_minutes,
        vote_duration_hours=group.vote_duration_hours,
    )

    member_ids = await g_repo.get_non_admin_member_ids(group_id)
    await m_repo.create_pending_attendances(match.id, member_ids)

    match_url = f"https://rachao.app/match/{hash_}"
    await asyncio.gather(*[
        send_push(
            db, pid,
            title=f"⚽ Novo rachão — {group.name}",
            body=f"Partida em {_fmt_date(match.match_date)}. Confirme sua presença!",
            url=match_url,
        )
        for pid in member_ids
    ], return_exceptions=True)

    return match


@router.get("/groups/{group_id}/matches/{match_id}", response_model=MatchDetailResponse)
async def get_match(group_id: uuid.UUID, match_id: uuid.UUID, db: DB, current: CurrentPlayer):
    g_repo = GroupRepository(db)
    if current.role != PlayerRole.ADMIN:
        member = await g_repo.get_member(group_id, current.id)
        if not member:
            raise ForbiddenError()

    repo = MatchRepository(db)
    match = await repo.get_with_attendances(match_id)
    if not match or match.group_id != group_id:
        raise NotFoundError("Partida não encontrada")
    player_ids = [a.player.id for a in match.attendances if a.player.role != PlayerRole.ADMIN]
    g_repo = GroupRepository(db)
    skills = await g_repo.get_member_skills(group_id, player_ids)
    position_map = {str(pid): data["position"] for pid, data in skills.items()}
    return _build_detail(match, position_map)


@router.patch("/groups/{group_id}/matches/{match_id}", response_model=MatchResponse)
async def update_match(
    group_id: uuid.UUID, match_id: uuid.UUID, body: MatchUpdate, db: DB, current: CurrentPlayer
):
    g_repo = GroupRepository(db)
    if current.role != PlayerRole.ADMIN:
        member = await g_repo.get_member(group_id, current.id)
        if not member or member.role != GroupMemberRole.ADMIN:
            raise ForbiddenError()

    repo = MatchRepository(db)
    match = await repo.get(match_id)
    if not match or match.group_id != group_id:
        raise NotFoundError("Partida não encontrada")

    closing = body.status == MatchStatus.CLOSED and match.status != MatchStatus.CLOSED
    going_in_progress = (
        body.status == MatchStatus.IN_PROGRESS
        and match.status not in (MatchStatus.IN_PROGRESS, MatchStatus.CLOSED)
    )

    for field, value in body.model_dump(exclude_unset=True).items():
        setattr(match, field, value)
    await db.flush()
    await db.refresh(match)

    if closing:
        await run_recurrence(db)

    if going_in_progress:
        g_repo2 = GroupRepository(db)
        group = await g_repo2.get(group_id)
        group_name = group.name if group else ""
        confirmed_ids = await repo.get_confirmed_player_ids(match_id)
        await asyncio.gather(*[
            send_push(
                db, pid,
                title=f"⚽ Bola rolando! — {group_name}",
                body="A partida de hoje já começou! 🎉",
                url=f"https://rachao.app/match/{match.hash}",
            )
            for pid in confirmed_ids
        ], return_exceptions=True)

    return match


@router.delete("/groups/{group_id}/matches/{match_id}", status_code=204)
async def delete_match(group_id: uuid.UUID, match_id: uuid.UUID, db: DB, current: CurrentPlayer):
    g_repo = GroupRepository(db)
    if current.role != PlayerRole.ADMIN:
        member = await g_repo.get_member(group_id, current.id)
        if not member or member.role != GroupMemberRole.ADMIN:
            raise ForbiddenError()

    repo = MatchRepository(db)
    match = await repo.get(match_id)
    if not match or match.group_id != group_id:
        raise NotFoundError("Partida não encontrada")
    await repo.delete(match)


# ── Player stats (goals & assists) ───────────────────────────────────────────


@router.get(
    "/matches/public/{match_hash}/player-stats",
    response_model=MatchPlayerStatsResponse,
    tags=["public"],
)
async def get_public_match_player_stats(match_hash: str, db: DB):
    """Retorna gols e assistências de todos os jogadores da partida. Público."""
    m_repo = MatchRepository(db)
    match = await m_repo.get_by_hash(match_hash)
    if not match:
        raise NotFoundError("Partida não encontrada")

    stats_repo = MatchStatsRepository(db)
    records = await stats_repo.get_by_match(match.id)

    return MatchPlayerStatsResponse(
        match_hash=match_hash,
        registered=len(records) > 0,
        stats=[
            PlayerStatResponse(
                player_id=r.player_id,
                player_name=r.player.name,
                avatar_url=r.player.avatar_url,
                goals=r.goals,
                assists=r.assists,
            )
            for r in records
        ],
    )


@router.put(
    "/matches/{match_hash}/player-stats",
    response_model=MatchPlayerStatsResponse,
)
async def put_match_player_stats(
    match_hash: str,
    body: MatchPlayerStatsRequest,
    db: DB,
    current: CurrentPlayer,
):
    """Registra/atualiza gols e assistências da partida. Apenas admin do grupo."""
    m_repo = MatchRepository(db)
    match = await m_repo.get_by_hash(match_hash)
    if not match:
        raise NotFoundError("Partida não encontrada")

    # Verifica se o usuário é admin do grupo
    if current.role != PlayerRole.ADMIN:
        g_repo = GroupRepository(db)
        member = await g_repo.get_member(match.group_id, current.id)
        if not member or member.role != GroupMemberRole.ADMIN:
            raise ForbiddenError("Apenas admins do grupo podem registrar estatísticas")

    # Valida que todos os player_ids são confirmados nesta partida
    confirmed_ids = set(await m_repo.get_confirmed_player_ids(match.id))
    for item in body.stats:
        if item.player_id not in confirmed_ids:
            raise ConflictError(
                f"Jogador {item.player_id} não está confirmado nesta partida"
            )

    stats_repo = MatchStatsRepository(db)
    records = await stats_repo.upsert_stats(
        match_id=match.id,
        recorded_by_id=current.id,
        stats=[{"player_id": s.player_id, "goals": s.goals, "assists": s.assists} for s in body.stats],
    )

    return MatchPlayerStatsResponse(
        match_hash=match_hash,
        registered=len(records) > 0,
        stats=[
            PlayerStatResponse(
                player_id=r.player_id,
                player_name=r.player.name,
                avatar_url=r.player.avatar_url,
                goals=r.goals,
                assists=r.assists,
            )
            for r in records
        ],
    )


# ── Attendance ────────────────────────────────────────────────────────────────

@router.post(
    "/groups/{group_id}/matches/{match_id}/attendance",
    response_model=AttendanceResponse,
)
async def set_attendance(
    group_id: uuid.UUID,
    match_id: uuid.UUID,
    body: SetAttendanceRequest,
    db: DB,
    current: CurrentPlayer,
):
    # Super admins não participam de partidas
    if current.role == PlayerRole.ADMIN and current.id == body.player_id:
        raise ForbiddenError("Administradores não participam de partidas")

    # Can set own attendance or admin/group-admin can set anyone's
    if current.id != body.player_id:
        g_repo = GroupRepository(db)
        if current.role != PlayerRole.ADMIN:
            member = await g_repo.get_member(group_id, current.id)
            if not member or member.role != GroupMemberRole.ADMIN:
                raise ForbiddenError("Você só pode confirmar sua própria presença")

    repo = MatchRepository(db)
    match = await repo.get(match_id)
    if not match or match.group_id != group_id:
        raise NotFoundError("Partida não encontrada")

    # Fecha automaticamente se a data já passou (fallback — usa UTC-3 explícito)
    today_brazil = datetime.now(timezone(timedelta(hours=-3))).date()
    if match.match_date < today_brazil and match.status == MatchStatus.OPEN:
        match.status = MatchStatus.CLOSED
        await db.flush()

    if match.status not in (MatchStatus.OPEN, MatchStatus.IN_PROGRESS):
        raise ForbiddenError("Esta partida está encerrada")

    if body.status == AttendanceStatus.CONFIRMED and match.max_players is not None:
        confirmed = await repo.count_confirmed(match_id, exclude_player_id=body.player_id)
        if confirmed >= match.max_players:
            raise ForbiddenError(
                f"Partida lotada. O limite de {match.max_players} jogadores já foi atingido."
            )

    attendance = await repo.upsert_attendance(match_id, body.player_id, body.status)
    await db.refresh(attendance, ["player"])
    g_repo = GroupRepository(db)
    member = await g_repo.get_member(group_id, body.player_id)
    position = member.position if member else None
    r = AttendanceResponse.model_validate(attendance)
    return r.model_copy(update={"position": position})
