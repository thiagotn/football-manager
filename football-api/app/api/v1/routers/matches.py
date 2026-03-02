import secrets
import uuid

from fastapi import APIRouter

from app.core.dependencies import DB, CurrentPlayer
from app.core.exceptions import ConflictError, ForbiddenError, NotFoundError
from app.db.repositories.group_repo import GroupRepository
from app.db.repositories.match_repo import MatchRepository
from app.models.match import AttendanceStatus, Match, MatchStatus
from app.models.player import PlayerRole
from app.models.group import GroupMemberRole
from app.schemas.match import (
    AttendanceResponse,
    MatchCreate,
    MatchDetailResponse,
    MatchResponse,
    MatchUpdate,
    SetAttendanceRequest,
)

router = APIRouter(tags=["matches"])


def _generate_hash() -> str:
    return secrets.token_urlsafe(8)[:10]


def _build_detail(match: Match) -> MatchDetailResponse:
    # Exclui o super admin (role=admin) das listas de presença
    attendances = [a for a in match.attendances if a.player.role != PlayerRole.ADMIN]
    confirmed = [a for a in attendances if a.status == AttendanceStatus.CONFIRMED]
    declined = [a for a in attendances if a.status == AttendanceStatus.DECLINED]
    pending = [a for a in attendances if a.status == AttendanceStatus.PENDING]

    return MatchDetailResponse(
        id=match.id,
        number=match.number,
        group_id=match.group_id,
        match_date=match.match_date,
        start_time=match.start_time,
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
        attendances=[AttendanceResponse.model_validate(a) for a in attendances],
        confirmed_count=len(confirmed),
        declined_count=len(declined),
        pending_count=len(pending),
        group_name=match.group.name if match.group else "",
    )


# ── Public endpoint (no auth) ─────────────────────────────────────────────────

@router.get("/matches/public/{match_hash}", response_model=MatchDetailResponse, tags=["public"])
async def get_match_public(match_hash: str, db: DB):
    """Endpoint público para visualizar uma partida via hash/link único."""
    repo = MatchRepository(db)
    match = await repo.get_by_hash_with_attendances(match_hash)
    if not match:
        raise NotFoundError("Partida não encontrada")
    return _build_detail(match)


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
        match_date=body.match_date,
        start_time=body.start_time,
        location=body.location,
        address=body.address,
        court_type=body.court_type,
        players_per_team=body.players_per_team,
        max_players=body.max_players,
        notes=body.notes,
        hash=hash_,
        created_by_id=current.id,
    )

    member_ids = await g_repo.get_non_admin_member_ids(group_id)
    await m_repo.create_pending_attendances(match.id, member_ids)

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
    return _build_detail(match)


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

    for field, value in body.model_dump(exclude_none=True).items():
        setattr(match, field, value)
    await db.flush()
    await db.refresh(match)
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
    if match.status == MatchStatus.CLOSED:
        raise ForbiddenError("Esta partida está encerrada")

    if body.status == AttendanceStatus.CONFIRMED and match.max_players is not None:
        confirmed = await repo.count_confirmed(match_id, exclude_player_id=body.player_id)
        if confirmed >= match.max_players:
            raise ForbiddenError(
                f"Partida lotada. O limite de {match.max_players} jogadores já foi atingido."
            )

    attendance = await repo.upsert_attendance(match_id, body.player_id, body.status)
    await db.refresh(attendance, ["player"])
    return AttendanceResponse.model_validate(attendance)
