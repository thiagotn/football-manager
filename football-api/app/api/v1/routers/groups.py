import re
import uuid

from fastapi import APIRouter, Depends, Query
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.dependencies import DB, CurrentPlayer, AdminPlayer
from app.core.exceptions import ConflictError, NotFoundError, ForbiddenError, PlanLimitError
from app.db.repositories.subscription_repo import SubscriptionRepository

# Limites por plano — fonte de verdade no backend (sincronizar com subscriptions.py)
_PLAN_GROUPS_LIMIT: dict[str, int | None] = {
    "free":  1,
    "basic": 3,
    "pro":   10,
}
_PLAN_MEMBERS_LIMIT: dict[str, int | None] = {
    "free":  30,
    "basic": 50,
    "pro":   None,  # ilimitado
}
from app.db.repositories.group_repo import GroupRepository
from app.db.repositories.group_stats_repo import GroupStatsRepository
from app.db.repositories.match_repo import MatchRepository
from app.db.repositories.player_repo import PlayerRepository
from app.models.group import GroupMemberRole
from app.models.match import AttendanceStatus
from app.models.player import PlayerRole
from app.schemas.group import (
    AddMemberRequest,
    GroupCreate,
    GroupDetailResponse,
    GroupMemberResponse,
    GroupResponse,
    GroupUpdate,
    UpdateMemberRoleRequest,
    UpdateMemberRequest,
)
from app.schemas.group_stats import GroupStatsResponse

router = APIRouter(prefix="/groups", tags=["groups"])


def _auto_slug(name: str, existing_slugs: set[str] = set()) -> str:
    base = re.sub(r"[^\w\s-]", "", name.lower())
    base = re.sub(r"[\s_]+", "-", base).strip("-")[:50]
    slug = base
    i = 2
    while slug in existing_slugs:
        slug = f"{base}-{i}"
        i += 1
    return slug


@router.get("", response_model=list[GroupResponse])
async def list_groups(db: DB, current: CurrentPlayer):
    repo = GroupRepository(db)
    if current.role == PlayerRole.ADMIN:
        return await repo.get_all()
    return await repo.get_player_groups(current.id)


@router.post("", response_model=GroupResponse, status_code=201)
async def create_group(body: GroupCreate, db: DB, current: CurrentPlayer):
    repo = GroupRepository(db)

    # Verifica limite de grupos do plano (admins globais são isentos)
    if current.role != PlayerRole.ADMIN:
        sub_repo = SubscriptionRepository(db)
        sub = await sub_repo.get_or_create(current.id)
        groups_limit = _PLAN_GROUPS_LIMIT.get(sub.plan, 1)
        groups_used = await sub_repo.count_admin_groups(current.id)
        if groups_limit is not None and groups_used >= groups_limit:
            raise PlanLimitError()

    slug = body.slug or _auto_slug(body.name)
    existing = await repo.get_by_slug(slug)
    if existing:
        raise ConflictError(f"Slug '{slug}' já está em uso")

    group = await repo.create(
        name=body.name,
        description=body.description,
        slug=slug,
        vote_open_delay_minutes=body.vote_open_delay_minutes,
        vote_duration_hours=body.vote_duration_hours,
    )

    # Creator becomes group admin
    await repo.add_member(group.id, current.id, GroupMemberRole.ADMIN)

    return group


@router.get("/{group_id}", response_model=GroupDetailResponse)
async def get_group(group_id: uuid.UUID, db: DB, current: CurrentPlayer):
    repo = GroupRepository(db)
    group = await repo.get_with_members(group_id)
    if not group:
        raise NotFoundError("Grupo não encontrado")

    # Check membership unless admin
    if current.role != PlayerRole.ADMIN:
        member = await repo.get_member(group_id, current.id)
        if not member:
            raise ForbiddenError("Você não é membro deste grupo")

    caller_is_admin = current.role == PlayerRole.ADMIN
    if not caller_is_admin:
        caller_member = await repo.get_member(group_id, current.id)
        caller_is_admin = caller_member is not None and caller_member.role == GroupMemberRole.ADMIN

    def _member_response(m, include_skill: bool) -> GroupMemberResponse:
        r = GroupMemberResponse.model_validate(m)
        if include_skill:
            r.skill_stars = m.skill_stars
            r.is_goalkeeper = m.is_goalkeeper
        return r

    return GroupDetailResponse(
        id=group.id,
        name=group.name,
        description=group.description,
        slug=group.slug,
        per_match_amount=group.per_match_amount,
        monthly_amount=group.monthly_amount,
        recurrence_enabled=group.recurrence_enabled,
        vote_open_delay_minutes=group.vote_open_delay_minutes,
        vote_duration_hours=group.vote_duration_hours,
        created_at=group.created_at,
        updated_at=group.updated_at,
        members=[_member_response(m, caller_is_admin) for m in group.members],
        total_members=len(group.members),
    )


@router.patch("/{group_id}", response_model=GroupResponse)
async def update_group(group_id: uuid.UUID, body: GroupUpdate, db: DB, current: CurrentPlayer):
    repo = GroupRepository(db)
    group = await repo.get(group_id)
    if not group:
        raise NotFoundError("Grupo não encontrado")

    # Must be global admin or group admin
    if current.role != PlayerRole.ADMIN:
        member = await repo.get_member(group_id, current.id)
        if not member or member.role != GroupMemberRole.ADMIN:
            raise ForbiddenError("Apenas admins do grupo podem editar")

    for field, value in body.model_dump(exclude_none=True, exclude={'per_match_amount', 'monthly_amount'}).items():
        setattr(group, field, value)
    # Campos de cobrança são explicitamente anuláveis: atualiza se enviados,
    # mesmo que o valor seja null (para zerar o campo).
    if 'per_match_amount' in body.model_fields_set:
        group.per_match_amount = body.per_match_amount
    if 'monthly_amount' in body.model_fields_set:
        group.monthly_amount = body.monthly_amount
    await db.flush()
    await db.refresh(group)
    return group


@router.delete("/{group_id}", status_code=204)
async def delete_group(group_id: uuid.UUID, db: DB, _: AdminPlayer):
    repo = GroupRepository(db)
    group = await repo.get(group_id)
    if not group:
        raise NotFoundError("Grupo não encontrado")
    await repo.delete(group)


# ── Members ───────────────────────────────────────────────────────────────────

@router.get("/{group_id}/members", response_model=list[GroupMemberResponse])
async def list_members(group_id: uuid.UUID, db: DB, current: CurrentPlayer):
    repo = GroupRepository(db)
    group = await repo.get_with_members(group_id)
    if not group:
        raise NotFoundError("Grupo não encontrado")

    caller_is_admin = current.role == PlayerRole.ADMIN
    if not caller_is_admin:
        member = await repo.get_member(group_id, current.id)
        if not member:
            raise ForbiddenError()
        caller_is_admin = member.role == GroupMemberRole.ADMIN

    def _member_response(m, include_skill: bool) -> GroupMemberResponse:
        r = GroupMemberResponse.model_validate(m)
        if include_skill:
            r.skill_stars = m.skill_stars
            r.is_goalkeeper = m.is_goalkeeper
        return r

    return [_member_response(m, caller_is_admin) for m in group.members]


@router.post("/{group_id}/members", response_model=GroupMemberResponse, status_code=201)
async def add_member(group_id: uuid.UUID, body: AddMemberRequest, db: DB, current: CurrentPlayer):
    g_repo = GroupRepository(db)
    p_repo = PlayerRepository(db)

    group = await g_repo.get(group_id)
    if not group:
        raise NotFoundError("Grupo não encontrado")

    if current.role != PlayerRole.ADMIN:
        caller_member = await g_repo.get_member(group_id, current.id)
        if not caller_member or caller_member.role != GroupMemberRole.ADMIN:
            raise ForbiddenError()

    player = await p_repo.get(body.player_id)
    if not player:
        raise NotFoundError("Jogador não encontrado")

    existing = await g_repo.get_member(group_id, body.player_id)
    if existing:
        raise ConflictError("Jogador já é membro deste grupo")

    # Verifica limite de membros do plano (admins globais são isentos)
    if current.role != PlayerRole.ADMIN:
        sub_repo = SubscriptionRepository(db)
        sub = await sub_repo.get_or_create(current.id)
        members_limit = _PLAN_MEMBERS_LIMIT.get(sub.plan, 30)
        if members_limit is not None:
            member_count = len(await g_repo.get_non_admin_member_ids(group_id))
            if member_count >= members_limit:
                raise PlanLimitError()

    member = await g_repo.add_member(group_id, body.player_id, body.role)

    # Adiciona o novo membro como pendente nas partidas abertas/em andamento
    m_repo = MatchRepository(db)
    active_matches = await m_repo.get_active_matches(group_id)
    for match in active_matches:
        await m_repo.upsert_attendance(match.id, body.player_id, AttendanceStatus.PENDING)

    await db.refresh(member)
    # Eager load player for response
    await db.refresh(member, ["player"])
    return GroupMemberResponse.model_validate(member)


@router.patch("/{group_id}/members/{player_id}", response_model=GroupMemberResponse)
async def update_member(
    group_id: uuid.UUID,
    player_id: uuid.UUID,
    body: UpdateMemberRequest,
    db: DB,
    current: CurrentPlayer,
):
    repo = GroupRepository(db)
    if current.role != PlayerRole.ADMIN:
        caller = await repo.get_member(group_id, current.id)
        if not caller or caller.role != GroupMemberRole.ADMIN:
            raise ForbiddenError()

    member = await repo.get_member(group_id, player_id)
    if not member:
        raise NotFoundError("Membro não encontrado")

    if body.role is not None:
        member.role = body.role
    if body.skill_stars is not None:
        member.skill_stars = body.skill_stars
    if body.is_goalkeeper is not None:
        member.is_goalkeeper = body.is_goalkeeper

    await db.flush()
    await db.refresh(member, ["player"])
    r = GroupMemberResponse.model_validate(member)
    r.skill_stars = member.skill_stars
    r.is_goalkeeper = member.is_goalkeeper
    return r


@router.delete("/{group_id}/members/{player_id}", status_code=204)
async def remove_member(
    group_id: uuid.UUID, player_id: uuid.UUID, db: DB, current: CurrentPlayer
):
    repo = GroupRepository(db)
    if current.role != PlayerRole.ADMIN:
        caller = await repo.get_member(group_id, current.id)
        if not caller or caller.role != GroupMemberRole.ADMIN:
            raise ForbiddenError()

    member = await repo.get_member(group_id, player_id)
    if not member:
        raise NotFoundError("Membro não encontrado")

    m_repo = MatchRepository(db)
    await m_repo.delete_player_attendances_in_open_matches(group_id, player_id)
    await repo.delete(member)


# ── Stats ──────────────────────────────────────────────────────────────────────

@router.get("/{group_id}/stats", response_model=GroupStatsResponse)
async def get_group_stats(
    group_id: uuid.UUID,
    db: DB,
    current: CurrentPlayer,
    period: str = Query("annual"),
    month: str | None = Query(None),
):
    repo = GroupRepository(db)
    group = await repo.get(group_id)
    if not group:
        raise NotFoundError("Grupo não encontrado")
    if current.role != PlayerRole.ADMIN:
        member = await repo.get_member(group_id, current.id)
        if not member:
            raise ForbiddenError("Você não é membro deste grupo")

    # Valida formato do mês (YYYY-MM)
    if month and not re.match(r"^\d{4}-\d{2}$", month):
        month = None

    stats_repo = GroupStatsRepository(db)
    players, period_label = await stats_repo.get_group_stats(group_id, period=period, month=month)
    return GroupStatsResponse(players=players, period_label=period_label)
