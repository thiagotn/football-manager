import asyncio
import re
import uuid

from fastapi import APIRouter, Depends, Query
from sqlalchemy import func, select
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.dependencies import DB, CurrentPlayer, AdminPlayer
from app.core.exceptions import ConflictError, NotFoundError, ForbiddenError, PlanLimitError
from app.db.repositories.subscription_repo import SubscriptionRepository
from app.db.repositories.waitlist_repo import WaitlistRepository

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
from app.db.repositories.finance_repo import FinanceRepository
from app.db.repositories.group_repo import GroupRepository
from app.db.repositories.group_stats_repo import GroupStatsRepository
from app.db.repositories.match_repo import MatchRepository
from app.db.repositories.player_repo import PlayerRepository
from app.models.group import GroupMember, GroupMemberRole
from app.models.match import Attendance, AttendanceStatus, MatchStatus
from app.models.player import PlayerRole
from app.models.waitlist import WaitlistStatus
from app.services.push import send_push
from app.schemas.group import (
    AddMemberRequest,
    GroupCreate,
    GroupDetailResponse,
    GroupMemberResponse,
    GroupResponse,
    GroupUpdate,
    UpdateMemberRoleRequest,
    UpdateMemberRequest,
    WaitlistJoinRequest,
    WaitlistActionRequest,
    WaitlistEntryResponse,
)
from app.schemas.group_stats import GroupStatsResponse

_MONTHS_PT = ["jan","fev","mar","abr","mai","jun","jul","ago","set","out","nov","dez"]

def _fmt_date(d) -> str:
    return f"{d.day} de {_MONTHS_PT[d.month - 1]}"

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
        is_public=body.is_public,
        vote_open_delay_minutes=body.vote_open_delay_minutes,
        vote_duration_hours=body.vote_duration_hours,
        timezone=body.timezone,
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
        is_public=group.is_public,
        vote_open_delay_minutes=group.vote_open_delay_minutes,
        vote_duration_hours=group.vote_duration_hours,
        timezone=group.timezone,
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

    if player.role == PlayerRole.ADMIN:
        raise ForbiddenError("Super admin não pode ser adicionado como membro de grupo")

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

    # Garante que o novo membro aparece no período financeiro do mês corrente
    f_repo = FinanceRepository(db)
    await f_repo.ensure_member_in_current_period(
        group_id, body.player_id, player.nickname or player.name
    )

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


# ── Waitlist ────────────────────────────────────────────────────────────────────

@router.post("/{group_id}/waitlist", response_model=WaitlistEntryResponse, status_code=201)
async def join_waitlist(group_id: uuid.UUID, body: WaitlistJoinRequest, db: DB, current: CurrentPlayer):
    """Enter the waitlist for the next open match of a public group."""
    if not body.agreed:
        raise ForbiddenError("É necessário aceitar os termos para entrar na fila")

    g_repo = GroupRepository(db)
    group = await g_repo.get(group_id)
    if not group:
        raise NotFoundError("Grupo não encontrado")
    if not group.is_public:
        raise ForbiddenError("Este grupo não aceita candidatos externos")

    # Player must not be a member
    member = await g_repo.get_member(group_id, current.id)
    if member:
        raise ConflictError("Você já é membro deste grupo")

    # Find the open match
    m_repo = MatchRepository(db)
    active_matches = await m_repo.get_active_matches(group_id)
    if not active_matches:
        raise NotFoundError("Nenhum rachão aberto neste grupo")
    open_match = active_matches[0]

    # Check vacancy (if max_players defined)
    if open_match.max_players is not None:
        result = await db.execute(
            select(func.count()).where(
                Attendance.match_id == open_match.id,
                Attendance.status == AttendanceStatus.CONFIRMED,
            )
        )
        confirmed_count = result.scalar_one()
        if confirmed_count >= open_match.max_players:
            raise ForbiddenError("Rachão lotado — não há vagas disponíveis")

    w_repo = WaitlistRepository(db)
    existing = await w_repo.get_entry(open_match.id, current.id)
    if existing:
        raise ConflictError("Você já está na lista de espera deste rachão")

    entry = await w_repo.create(open_match.id, current.id, body.intro)

    # Notify all group admins
    result = await db.execute(
        select(GroupMember.player_id).where(
            GroupMember.group_id == group_id,
            GroupMember.role == GroupMemberRole.ADMIN,
        )
    )
    admin_ids = list(result.scalars().all())
    await asyncio.gather(*[
        send_push(
            db, aid,
            title=f"⚽ Novo candidato — {group.name}",
            body=f"{current.name} quer participar do rachão em {_fmt_date(open_match.match_date)}. Acesse o grupo para revisar.",
            url=f"https://rachao.app/groups/{group_id}",
        )
        for aid in admin_ids
    ], return_exceptions=True)

    return WaitlistEntryResponse(
        id=entry.id,
        match_id=entry.match_id,
        player_id=entry.player_id,
        player_name=entry.player.name,
        player_nickname=entry.player.nickname,
        intro=entry.intro,
        status=entry.status,
        created_at=entry.created_at,
    )


@router.get("/{group_id}/waitlist", response_model=list[WaitlistEntryResponse])
async def list_waitlist(group_id: uuid.UUID, db: DB, current: CurrentPlayer):
    """List waitlist candidates for the group's active match (admin only)."""
    g_repo = GroupRepository(db)
    group = await g_repo.get(group_id)
    if not group:
        raise NotFoundError("Grupo não encontrado")

    if current.role != PlayerRole.ADMIN:
        member = await g_repo.get_member(group_id, current.id)
        if not member or member.role != GroupMemberRole.ADMIN:
            raise ForbiddenError("Apenas admins do grupo podem ver a lista de espera")

    m_repo = MatchRepository(db)
    active_matches = await m_repo.get_active_matches(group_id)
    if not active_matches:
        return []

    open_match = active_matches[0]
    w_repo = WaitlistRepository(db)
    entries = await w_repo.get_pending_for_match(open_match.id)
    return [
        WaitlistEntryResponse(
            id=e.id,
            match_id=e.match_id,
            player_id=e.player_id,
            player_name=e.player.name,
            player_nickname=e.player.nickname,
            intro=e.intro,
            status=e.status,
            created_at=e.created_at,
        )
        for e in entries
    ]


@router.get("/{group_id}/waitlist/me", response_model=WaitlistEntryResponse | None)
async def get_my_waitlist_entry(group_id: uuid.UUID, db: DB, current: CurrentPlayer):
    """Get the current player's waitlist entry for the group's active match, if any."""
    g_repo = GroupRepository(db)
    group = await g_repo.get(group_id)
    if not group:
        raise NotFoundError("Grupo não encontrado")

    m_repo = MatchRepository(db)
    active_matches = await m_repo.get_active_matches(group_id)
    if not active_matches:
        return None

    w_repo = WaitlistRepository(db)
    entry = await w_repo.get_entry(active_matches[0].id, current.id)
    if not entry:
        return None

    # Load player relationship
    await db.refresh(entry, ["player"])
    return WaitlistEntryResponse(
        id=entry.id,
        match_id=entry.match_id,
        player_id=entry.player_id,
        player_name=entry.player.name,
        player_nickname=entry.player.nickname,
        intro=entry.intro,
        status=entry.status,
        created_at=entry.created_at,
    )


@router.patch("/{group_id}/waitlist/{entry_id}", response_model=WaitlistEntryResponse)
async def review_waitlist_entry(
    group_id: uuid.UUID,
    entry_id: uuid.UUID,
    body: WaitlistActionRequest,
    db: DB,
    current: CurrentPlayer,
):
    """Accept or reject a waitlist candidate (admin only)."""
    if body.action not in ("accept", "reject"):
        raise ForbiddenError("Ação inválida. Use 'accept' ou 'reject'")

    g_repo = GroupRepository(db)
    group = await g_repo.get(group_id)
    if not group:
        raise NotFoundError("Grupo não encontrado")

    if current.role != PlayerRole.ADMIN:
        member = await g_repo.get_member(group_id, current.id)
        if not member or member.role != GroupMemberRole.ADMIN:
            raise ForbiddenError("Apenas admins do grupo podem revisar candidatos")

    w_repo = WaitlistRepository(db)
    entry = await w_repo.get_by_id(entry_id)
    if not entry or entry.match.group_id != group_id:
        raise NotFoundError("Candidatura não encontrada")
    if entry.status != WaitlistStatus.PENDING:
        raise ConflictError("Esta candidatura já foi revisada")

    match = entry.match
    candidate_player = entry.player

    if body.action == "accept":
        # Check vacancy
        if match.max_players is not None:
            result = await db.execute(
                select(func.count()).where(
                    Attendance.match_id == match.id,
                    Attendance.status == AttendanceStatus.CONFIRMED,
                )
            )
            confirmed_count = result.scalar_one()
            if confirmed_count >= match.max_players:
                raise ForbiddenError("Rachão lotado — não é possível aceitar mais candidatos")

        # Add as group member
        existing_member = await g_repo.get_member(group_id, entry.player_id)
        if not existing_member:
            await g_repo.add_member(group_id, entry.player_id, GroupMemberRole.MEMBER)
            # Garante que o novo membro aparece no período financeiro do mês corrente
            f_repo = FinanceRepository(db)
            await f_repo.ensure_member_in_current_period(
                group_id, entry.player_id, candidate_player.nickname or candidate_player.name
            )

        # Confirm attendance for the waitlist match and add as pending to all other active matches
        m_repo = MatchRepository(db)
        await m_repo.upsert_attendance(match.id, entry.player_id, AttendanceStatus.CONFIRMED)
        for active in await m_repo.get_active_matches(group_id):
            if active.id != match.id:
                existing_att = await m_repo.get_attendance(active.id, entry.player_id)
                if not existing_att:
                    await m_repo.create_pending_attendances(active.id, [entry.player_id])

        await w_repo.accept(entry, current.id)

        await send_push(
            db, entry.player_id,
            title="✅ Você foi aceito!",
            body=f"Bem-vindo ao grupo {group.name}! Sua presença no rachão de {_fmt_date(match.match_date)} foi confirmada.",
            url=f"https://rachao.app/match/{match.hash}",
        )
    else:
        await w_repo.reject(entry, current.id)

        await send_push(
            db, entry.player_id,
            title="❌ Candidatura não aprovada",
            body=f"Sua candidatura para o grupo {group.name} não foi aprovada desta vez.",
            url=f"https://rachao.app/groups/{group_id}",
        )

    return WaitlistEntryResponse(
        id=entry.id,
        match_id=entry.match_id,
        player_id=entry.player_id,
        player_name=candidate_player.name,
        player_nickname=candidate_player.nickname,
        intro=entry.intro,
        status=entry.status,
        created_at=entry.created_at,
    )
