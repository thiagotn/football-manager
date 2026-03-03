import re
from fastapi import APIRouter, Depends
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.dependencies import DB, CurrentPlayer
from app.core.exceptions import UnauthorizedError
from app.core.security import create_access_token, hash_password, verify_password
from app.db.repositories.player_repo import PlayerRepository
from app.schemas.auth import ChangePasswordRequest, LoginRequest, TokenResponse
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
