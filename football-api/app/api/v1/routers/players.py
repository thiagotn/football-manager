import io
import secrets
import uuid
from datetime import datetime, timedelta, timezone

from fastapi import APIRouter, HTTPException, Query, Request, UploadFile, File, status
from PIL import Image, UnidentifiedImageError
from PIL.Image import DecompressionBombError
from sqlalchemy import select, text

# Limita o tamanho máximo de imagem aceito pelo Pillow globalmente para evitar
# decompression bombs (arquivo pequeno que expande para GBs em RAM).
# 25M pixels ≈ 5000×5000 — cobre câmeras de até ~25MP; blocos maiores são
# quase impossíveis com arquivos de ≤5 MB e indicam imagem sintética maliciosa.
Image.MAX_IMAGE_PIXELS = 25_000_000

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
from app.schemas.player_public import PlayerPublicStats
from app.services import storage as storage_service

router = APIRouter(prefix="/players", tags=["players"])

AVATAR_RATE_LIMIT = 5        # uploads por janela
AVATAR_RATE_WINDOW = "1 hour"  # janela de tempo


def _real_ip(request: Request) -> str:
    """Retorna o IP real do cliente.

    Lê X-Forwarded-For (primeiro hop) quando disponível — útil atrás de proxy/nginx.
    O X-Forwarded-For ainda pode ser forjado se o proxy não sanitizar o header;
    a mitigação definitiva é configurar o proxy para sobrescrever o header.
    """
    forwarded = request.headers.get("x-forwarded-for")
    if forwarded:
        return forwarded.split(",")[0].strip()
    return request.client.host if request.client else "unknown"


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


@router.get("/{player_id}/public-stats", response_model=PlayerPublicStats)
async def get_player_public_stats(player_id: uuid.UUID, db: DB):
    """Public endpoint — no auth required. Returns player public stats."""
    repo = PlayerRepository(db)
    player = await repo.get(player_id)
    if not player or not player.active:
        raise NotFoundError("Jogador não encontrado")

    stats_repo = PlayerStatsRepository(db)
    full_stats = await stats_repo.get_full_stats(player_id)

    # Best skill_stars across all groups (max), default to 3 if no groups
    skill_stars = max((g.skill_stars for g in full_stats.groups), default=3)

    return PlayerPublicStats(
        player_id=player.id,
        name=player.name,
        nickname=player.nickname,
        avatar_url=player.avatar_url,
        skill_stars=skill_stars,
        total_matches_confirmed=full_stats.total_matches_confirmed,
        attendance_rate=full_stats.attendance_rate,
        current_streak=full_stats.current_streak,
        best_streak=full_stats.best_streak,
        top1_count=full_stats.top1_count,
        top5_count=full_stats.top5_count,
        total_vote_points=full_stats.total_vote_points,
        total_flop_votes=full_stats.total_flop_votes,
    )


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

    # Rejeita antes de ler qualquer byte se o Content-Length já excede o limite
    raw_cl = request.headers.get("content-length")
    if raw_cl:
        try:
            if int(raw_cl) > MAX_BYTES:
                raise ValidationError("Imagem muito grande. Máximo 5 MB.")
        except ValueError:
            pass

    # Rate limiting: máximo AVATAR_RATE_LIMIT uploads por AVATAR_RATE_WINDOW
    recent = await db.execute(
        text(f"""
            SELECT COUNT(*) FROM avatar_upload_logs
            WHERE player_id = :pid
              AND created_at > NOW() - INTERVAL '{AVATAR_RATE_WINDOW}'
        """),
        {"pid": current.id},
    )
    if (recent.scalar() or 0) >= AVATAR_RATE_LIMIT:
        raise HTTPException(
            status_code=status.HTTP_429_TOO_MANY_REQUESTS,
            detail="Muitas tentativas. Aguarde antes de enviar outro avatar.",
        )

    # Lê em chunks para não carregar arquivo enorme em memória de uma vez
    CHUNK = 64 * 1024
    chunks: list[bytes] = []
    total = 0
    while True:
        chunk = await file.read(CHUNK)
        if not chunk:
            break
        total += len(chunk)
        if total > MAX_BYTES:
            raise ValidationError("Imagem muito grande. Máximo 5 MB.")
        chunks.append(chunk)
    content = b"".join(chunks)

    # Valida formato e protege contra decompression bomb
    ALLOWED = {"JPEG", "PNG", "WEBP"}
    try:
        img = Image.open(io.BytesIO(content))
        if img.format not in ALLOWED:
            raise ValidationError("Formato inválido. Use JPG, PNG ou WebP.")
    except DecompressionBombError:
        raise ValidationError("Imagem muito grande para processar.")
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

    # Remove avatar anterior antes de fazer upload do novo (evita arquivos órfãos)
    if current.avatar_url:
        await storage_service.delete_avatar_by_url(current.avatar_url)

    # Token aleatório no nome do arquivo para evitar enumeração por player_id
    token = secrets.token_urlsafe(16)
    try:
        avatar_url = await storage_service.upload_avatar(str(current.id), webp_data, token)
    except RuntimeError as e:
        raise ValidationError(str(e))

    # Log de auditoria com IP real
    await db.execute(
        text("INSERT INTO avatar_upload_logs (player_id, ip_address) VALUES (:pid, :ip)"),
        {"pid": current.id, "ip": _real_ip(request)},
    )

    current.avatar_url = avatar_url
    await db.flush()
    await db.refresh(current)
    return current


@router.delete("/me/avatar", response_model=PlayerResponse)
async def remove_my_avatar(db: DB, current: CurrentPlayer):
    """Remove o avatar do jogador autenticado."""
    if not current.avatar_url:
        return current
    await storage_service.delete_avatar_by_url(current.avatar_url)
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
