import re
import uuid

from fastapi import APIRouter, Depends
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.dependencies import DB, CurrentPlayer, AdminPlayer
from app.core.exceptions import ConflictError, NotFoundError, ForbiddenError
from app.db.repositories.group_repo import GroupRepository
from app.db.repositories.player_repo import PlayerRepository
from app.models.group import GroupMemberRole
from app.models.player import PlayerRole
from app.schemas.group import (
    AddMemberRequest,
    GroupCreate,
    GroupDetailResponse,
    GroupMemberResponse,
    GroupResponse,
    GroupUpdate,
    UpdateMemberRoleRequest,
)

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
async def create_group(body: GroupCreate, db: DB, current: AdminPlayer):
    repo = GroupRepository(db)

    slug = body.slug or _auto_slug(body.name)
    existing = await repo.get_by_slug(slug)
    if existing:
        raise ConflictError(f"Slug '{slug}' já está em uso")

    group = await repo.create(name=body.name, description=body.description, slug=slug)

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

    return GroupDetailResponse(
        id=group.id,
        name=group.name,
        description=group.description,
        slug=group.slug,
        created_at=group.created_at,
        updated_at=group.updated_at,
        members=[GroupMemberResponse.model_validate(m) for m in group.members],
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

    for field, value in body.model_dump(exclude_none=True).items():
        setattr(group, field, value)
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
    if current.role != PlayerRole.ADMIN:
        member = await repo.get_member(group_id, current.id)
        if not member:
            raise ForbiddenError()
    return [GroupMemberResponse.model_validate(m) for m in group.members]


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

    member = await g_repo.add_member(group_id, body.player_id, body.role)
    await db.refresh(member)
    # Eager load player for response
    await db.refresh(member, ["player"])
    return GroupMemberResponse.model_validate(member)


@router.patch("/{group_id}/members/{player_id}", response_model=GroupMemberResponse)
async def update_member_role(
    group_id: uuid.UUID,
    player_id: uuid.UUID,
    body: UpdateMemberRoleRequest,
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

    member.role = body.role
    await db.flush()
    await db.refresh(member, ["player"])
    return GroupMemberResponse.model_validate(member)


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
    await repo.delete(member)
