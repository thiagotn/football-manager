import re
from fastapi import APIRouter, Depends
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.dependencies import DB, CurrentPlayer
from app.core.exceptions import ConflictError, UnauthorizedError
from app.core.security import create_access_token, hash_password, verify_password
from app.db.repositories.player_repo import PlayerRepository
from app.db.repositories.subscription_repo import SubscriptionRepository
from app.models.player import PlayerRole
from app.schemas.auth import ChangePasswordRequest, LoginRequest, RegisterRequest, TokenResponse
from app.schemas.player import PlayerResponse

router = APIRouter(prefix="/auth", tags=["auth"])


@router.post("/login", response_model=TokenResponse)
async def login(body: LoginRequest, db: DB):
    whatsapp = re.sub(r"\D", "", body.whatsapp)
    repo = PlayerRepository(db)
    player = await repo.get_by_whatsapp(whatsapp)

    if not player or not verify_password(body.password, player.password_hash):
        raise UnauthorizedError("WhatsApp ou senha incorretos")
    if not player.active:
        raise UnauthorizedError("Conta desativada")

    token = create_access_token(str(player.id))
    return TokenResponse(
        access_token=token,
        player_id=str(player.id),
        name=player.name,
        role=player.role,
        must_change_password=player.must_change_password,
    )


@router.post("/register", response_model=TokenResponse, status_code=201)
async def register(body: RegisterRequest, db: DB):
    """Cria uma nova conta de jogador com plano gratuito."""
    whatsapp = re.sub(r"\D", "", body.whatsapp)
    repo = PlayerRepository(db)
    existing = await repo.get_by_whatsapp(whatsapp)
    if existing:
        raise ConflictError("WhatsApp já cadastrado")

    player = await repo.create(
        name=body.name.strip(),
        nickname=body.nickname,
        whatsapp=whatsapp,
        password_hash=hash_password(body.password),
        role=PlayerRole.PLAYER,
    )
    sub_repo = SubscriptionRepository(db)
    await sub_repo.get_or_create(player.id)

    token = create_access_token(str(player.id))
    return TokenResponse(
        access_token=token,
        player_id=str(player.id),
        name=player.name,
        role=player.role,
        must_change_password=False,
    )


@router.get("/me", response_model=PlayerResponse)
async def me(current: CurrentPlayer):
    return current


@router.post("/change-password", status_code=204)
async def change_password(body: ChangePasswordRequest, db: DB, current: CurrentPlayer):
    if not verify_password(body.current_password, current.password_hash):
        raise UnauthorizedError("Senha atual incorreta")

    repo = PlayerRepository(db)
    player = await repo.get(current.id)
    player.password_hash = hash_password(body.new_password)
    player.must_change_password = False
    await db.flush()
