from datetime import datetime, timezone
from typing import Annotated
from uuid import UUID

from fastapi import Depends
from fastapi.security import HTTPAuthorizationCredentials, HTTPBearer
from sqlalchemy import select, update
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.exceptions import UnauthorizedError, ForbiddenError
from app.core.security import decode_access_token
from app.db.repositories.player_repo import PlayerRepository
from app.db.repositories.group_repo import GroupRepository
from app.db.session import get_db
from app.models.mcp_token import MCPToken
from app.models.player import Player, PlayerRole
from app.models.group import GroupMemberRole

bearer_scheme = HTTPBearer(auto_error=False)


async def _authenticate_mcp_token(token: str, db: AsyncSession) -> Player | None:
    token_hash = MCPToken.hash_token(token)
    result = await db.execute(
        select(MCPToken).where(MCPToken.token_hash == token_hash, MCPToken.revoked_at.is_(None))
    )
    mcp = result.scalar_one_or_none()
    if not mcp:
        return None
    now = datetime.now(timezone.utc)
    if mcp.expires_at and mcp.expires_at < now:
        return None
    await db.execute(
        update(MCPToken).where(MCPToken.id == mcp.id).values(last_used_at=now)
    )
    result = await db.execute(select(Player).where(Player.id == mcp.player_id))
    return result.scalar_one_or_none()


async def get_current_player(
    credentials: Annotated[HTTPAuthorizationCredentials | None, Depends(bearer_scheme)],
    db: Annotated[AsyncSession, Depends(get_db)],
) -> Player:
    if not credentials:
        raise UnauthorizedError("Token de acesso requerido")

    token = credentials.credentials

    if token.startswith("rachao_"):
        player = await _authenticate_mcp_token(token, db)
        if not player or not player.active:
            raise UnauthorizedError("Token inválido ou expirado")
        return player

    player_id = decode_access_token(token)
    if not player_id:
        raise UnauthorizedError("Token inválido ou expirado")

    repo = PlayerRepository(db)
    player = await repo.get(UUID(player_id))
    if not player or not player.active:
        raise UnauthorizedError("Usuário não encontrado ou inativo")

    return player


async def require_admin(
    current_player: Annotated[Player, Depends(get_current_player)],
) -> Player:
    if current_player.role != PlayerRole.ADMIN:
        raise ForbiddenError("Apenas administradores podem realizar esta ação")
    return current_player


async def require_group_admin(
    group_id: UUID,
    current_player: Annotated[Player, Depends(get_current_player)],
    db: Annotated[AsyncSession, Depends(get_db)],
) -> Player:
    """Verifica se o player é admin global ou admin do grupo específico."""
    if current_player.role == PlayerRole.ADMIN:
        return current_player
    repo = GroupRepository(db)
    member = await repo.get_member(group_id, current_player.id)
    if not member or member.role != GroupMemberRole.ADMIN:
        raise ForbiddenError("Você precisa ser admin deste grupo")
    return current_player


async def get_optional_player(
    credentials: Annotated[HTTPAuthorizationCredentials | None, Depends(bearer_scheme)],
    db: Annotated[AsyncSession, Depends(get_db)],
) -> Player | None:
    if not credentials:
        return None
    token = credentials.credentials
    if token.startswith("rachao_"):
        player = await _authenticate_mcp_token(token, db)
        return player if player and player.active else None
    player_id = decode_access_token(token)
    if not player_id:
        return None
    repo = PlayerRepository(db)
    player = await repo.get(UUID(player_id))
    if not player or not player.active:
        return None
    return player


# Type aliases for cleaner signatures
CurrentPlayer = Annotated[Player, Depends(get_current_player)]
AdminPlayer = Annotated[Player, Depends(require_admin)]
OptionalPlayer = Annotated[Player | None, Depends(get_optional_player)]
DB = Annotated[AsyncSession, Depends(get_db)]
