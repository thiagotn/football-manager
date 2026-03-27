from typing import Annotated
from uuid import UUID

from fastapi import Depends
from fastapi.security import HTTPAuthorizationCredentials, HTTPBearer
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.exceptions import UnauthorizedError, ForbiddenError
from app.core.security import decode_access_token
from app.db.repositories.player_repo import PlayerRepository
from app.db.repositories.group_repo import GroupRepository
from app.db.session import get_db
from app.models.player import Player, PlayerRole
from app.models.group import GroupMemberRole

bearer_scheme = HTTPBearer(auto_error=False)


async def get_current_player(
    credentials: Annotated[HTTPAuthorizationCredentials | None, Depends(bearer_scheme)],
    db: Annotated[AsyncSession, Depends(get_db)],
) -> Player:
    if not credentials:
        raise UnauthorizedError("Token de acesso requerido")

    player_id = decode_access_token(credentials.credentials)
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
    player_id = decode_access_token(credentials.credentials)
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
