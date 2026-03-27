import io
import secrets
import uuid
from datetime import datetime, timedelta, timezone

from fastapi import APIRouter, Query, Request, UploadFile, File
from PIL import Image, UnidentifiedImageError
from sqlalchemy import select, text

from app.core.dependencies import DB, CurrentPlayer, AdminPlayer
from app.core.exceptions import ConflictError, NotFoundError, ForbiddenError, ValidationError
from app.core.security import hash_password
from app.db.repositories.match_repo import MatchRepository
from app.db.repositories.player_repo import PlayerRepository
from app.db.repositories.player_stats_repo import PlayerStatsRepository
from app.models.player import Player, PlayerRole
from app.schemas.match import MatchResponse, PlayerMatchItem
from app.schemas.player import PlayerCreate, PlayerResponse, PlayerUpdate, ResetPasswordResponse
from app.schemas.player_stats import PlayerFullStats
from app.services import storage as storage_service

router = APIRouter(prefix="/players", tags=["players"])


@router.get("/me/matches", response_model=list[PlayerMatchItem])
async def get_my_matches(db: DB, current: CurrentPlayer):
    repo = MatchRepository(db)
    rows = await repo.get_player_matches(current.id)
    return [
        PlayerMatchItem(
            **MatchResponse.model_validate(match).model_dump(),
            group_name=group_name,
            group_timezone=group_timezone,
            my_attendance=my_attendance,
        )
        for match, group_name, group_timezone, my_attendance in rows
    ]


@router.get("/me/stats/full", response_model=PlayerFullStats)
async def get_my_full_stats(db: DB, current: CurrentPlayer):
    repo = PlayerStatsRepository(db)
    return await repo.get_full_stats(current.id)


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


@router.get("/signups/stats", tags=["admin"])
async def get_signup_stats(db: DB, _: AdminPlayer, limit: int = Query(30, le=100)):
    """Retorna estatísticas de cadastros e os registros mais recentes. Exclusivo para admins."""
    repo = PlayerRepository(db)
    now = datetime.now(timezone.utc)
    total, last_7, last_30, recent = (
        await repo.count_total(),
        await repo.count_since(now - timedelta(days=7)),
        await repo.count_since(now - timedelta(days=30)),
        await repo.get_recent(limit=limit),
    )
    return {
        "total": total,
        "last_7_days": last_7,
        "last_30_days": last_30,
        "recent": [
            {
                "id": str(p.id),
                "name": p.name,
                "nickname": p.nickname,
                "whatsapp": p.whatsapp,
                "active": p.active,
                "created_at": p.created_at.isoformat(),
            }
            for p in recent
        ],
    }


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


@router.put("/me/avatar", response_model=PlayerResponse)
async def upload_my_avatar(
    request: Request,
    db: DB,
    current: CurrentPlayer,
    file: UploadFile = File(...),
):
    """Faz upload de avatar para o jogador autenticado. Aceita JPG, PNG ou WebP (máx. 5 MB)."""
    MAX_BYTES = 5 * 1024 * 1024
    content = await file.read()
    if len(content) > MAX_BYTES:
        raise ValidationError("Imagem muito grande. Máximo 5 MB.")

    ALLOWED = {"JPEG", "PNG", "WEBP"}
    try:
        img = Image.open(io.BytesIO(content))
        if img.format not in ALLOWED:
            raise ValidationError("Formato inválido. Use JPG, PNG ou WebP.")
    except UnidentifiedImageError:
        raise ValidationError("Arquivo não reconhecido como imagem.")

    # Crop quadrado centralizado + resize 256×256 + converte para WebP
    img = img.convert("RGB")
    w, h = img.size
    side = min(w, h)
    img = img.crop(((w - side) // 2, (h - side) // 2, (w + side) // 2, (h + side) // 2))
    img = img.resize((256, 256), Image.LANCZOS)
    buf = io.BytesIO()
    img.save(buf, format="WEBP", quality=85)
    webp_data = buf.getvalue()

    try:
        avatar_url = await storage_service.upload_avatar(str(current.id), webp_data)
    except RuntimeError as e:
        raise ValidationError(str(e))

    # Log de auditoria
    client_ip = request.client.host if request.client else "unknown"
    await db.execute(
        text("INSERT INTO avatar_upload_logs (player_id, ip_address) VALUES (:pid, :ip)"),
        {"pid": current.id, "ip": client_ip},
    )

    current.avatar_url = avatar_url
    await db.flush()
    await db.refresh(current)
    return current


@router.delete("/me/avatar", response_model=PlayerResponse)
async def remove_my_avatar(db: DB, current: CurrentPlayer):
    """Remove o avatar do jogador autenticado."""
    await storage_service.delete_avatar(str(current.id))
    current.avatar_url = None
    await db.flush()
    await db.refresh(current)
    return current


@router.delete("/{player_id}", status_code=204)
async def delete_player(player_id: uuid.UUID, db: DB, _: AdminPlayer):
    repo = PlayerRepository(db)
    player = await repo.get(player_id)
    if not player:
        raise NotFoundError("Jogador não encontrado")
    # Soft delete
    player.active = False
    await db.flush()
