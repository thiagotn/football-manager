import secrets
import uuid

from fastapi import APIRouter, Query
from sqlalchemy import select, text

from app.core.dependencies import DB, CurrentPlayer, AdminPlayer
from app.core.exceptions import ConflictError, NotFoundError, ForbiddenError
from app.core.security import hash_password
from app.db.repositories.player_repo import PlayerRepository
from app.models.player import Player, PlayerRole
from app.schemas.player import PlayerCreate, PlayerResponse, PlayerUpdate, ResetPasswordResponse

router = APIRouter(prefix="/players", tags=["players"])


@router.get("/me/stats")
async def get_my_stats(db: DB, current: CurrentPlayer):
    personal = await db.execute(
        text("""
            SELECT COALESCE(
                SUM(GREATEST(0, EXTRACT(EPOCH FROM (m.end_time - m.start_time)) / 60)),
                0
            )::int
            FROM attendances a
            JOIN matches m ON m.id = a.match_id
            WHERE a.player_id = :player_id
              AND a.status = 'confirmed'
              AND m.status = 'closed'
              AND m.end_time IS NOT NULL
        """),
        {"player_id": current.id},
    )
    response: dict = {"minutes_played": personal.scalar()}

    if current.role == PlayerRole.ADMIN:
        platform = await db.execute(
            text("""
                SELECT
                    COALESCE(SUM(GREATEST(0, EXTRACT(EPOCH FROM (end_time - start_time)) / 60)), 0)::int
                        AS minutes_played,
                    COUNT(*)::int AS total_matches
                FROM matches
                WHERE status = 'closed'
                  AND end_time IS NOT NULL
            """)
        )
        row = platform.one()
        response["platform_minutes_played"] = row.minutes_played

        total = await db.execute(text("SELECT COUNT(*)::int FROM matches"))
        response["platform_total_matches"] = total.scalar()

    return response


@router.get("", response_model=list[PlayerResponse])
async def list_players(
    db: DB,
    _: AdminPlayer,
    active_only: bool = Query(True),
    limit: int = Query(100, le=500),
    offset: int = Query(0),
):
    repo = PlayerRepository(db)
    if active_only:
        return await repo.get_active()
    return await repo.get_all(limit=limit, offset=offset)


@router.post("", response_model=PlayerResponse, status_code=201)
async def create_player(body: PlayerCreate, db: DB, _: AdminPlayer):
    repo = PlayerRepository(db)
    existing = await repo.get_by_whatsapp(body.whatsapp)
    if existing:
        raise ConflictError(f"WhatsApp {body.whatsapp} já está cadastrado")

    player = await repo.create(
        name=body.name,
        nickname=body.nickname,
        whatsapp=body.whatsapp,
        password_hash=hash_password(body.password),
        role=body.role,
    )
    return player


@router.get("/{player_id}", response_model=PlayerResponse)
async def get_player(player_id: uuid.UUID, db: DB, current: CurrentPlayer):
    # Players can see their own data; admins can see all
    if current.id != player_id and current.role != PlayerRole.ADMIN:
        raise ForbiddenError()

    repo = PlayerRepository(db)
    player = await repo.get(player_id)
    if not player:
        raise NotFoundError("Jogador não encontrado")
    return player


@router.patch("/{player_id}", response_model=PlayerResponse)
async def update_player(player_id: uuid.UUID, body: PlayerUpdate, db: DB, current: CurrentPlayer):
    if current.id != player_id and current.role != PlayerRole.ADMIN:
        raise ForbiddenError()

    repo = PlayerRepository(db)
    player = await repo.get(player_id)
    if not player:
        raise NotFoundError("Jogador não encontrado")

    # Only admins can change roles
    if body.role is not None and current.role != PlayerRole.ADMIN:
        raise ForbiddenError("Apenas administradores podem alterar perfis")

    if body.whatsapp:
        existing = await repo.get_by_whatsapp(body.whatsapp)
        if existing and existing.id != player_id:
            raise ConflictError(f"WhatsApp {body.whatsapp} já está em uso")

    update_data = body.model_dump(exclude_none=True)
    if "password" in update_data:
        update_data["password_hash"] = hash_password(update_data.pop("password"))

    for field, value in update_data.items():
        setattr(player, field, value)

    await db.flush()
    await db.refresh(player)
    return player


@router.post("/{player_id}/reset-password", response_model=ResetPasswordResponse)
async def reset_player_password(player_id: uuid.UUID, db: DB, _: AdminPlayer):
    repo = PlayerRepository(db)
    player = await repo.get(player_id)
    if not player:
        raise NotFoundError("Jogador não encontrado")

    temp_password = secrets.token_urlsafe(8)
    player.password_hash = hash_password(temp_password)
    player.must_change_password = True
    await db.flush()

    return ResetPasswordResponse(temp_password=temp_password)


@router.delete("/{player_id}", status_code=204)
async def delete_player(player_id: uuid.UUID, db: DB, _: AdminPlayer):
    repo = PlayerRepository(db)
    player = await repo.get(player_id)
    if not player:
        raise NotFoundError("Jogador não encontrado")
    # Soft delete
    player.active = False
    await db.flush()
