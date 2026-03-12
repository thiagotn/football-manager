import re
import secrets
from datetime import datetime, timedelta, timezone

from fastapi import APIRouter

from app.core.config import get_settings
from app.core.dependencies import DB, CurrentPlayer
from app.core.exceptions import ConflictError, ForbiddenError, NotFoundError, PlanLimitError, ValidationError
from app.db.repositories.subscription_repo import SubscriptionRepository

_FREE_MEMBERS_LIMIT = 30
from app.core.security import create_access_token, hash_password, verify_password
from app.db.repositories.group_repo import GroupRepository
from app.db.repositories.invite_repo import InviteRepository
from app.db.repositories.match_repo import MatchRepository
from app.db.repositories.player_repo import PlayerRepository
from app.models.group import GroupMemberRole
from app.models.player import PlayerRole
from app.schemas.auth import TokenResponse
from app.schemas.invite import InviteAcceptRequest, InviteCheckResponse, InviteCreateRequest, InviteResponse

router = APIRouter(prefix="/invites", tags=["invites"])


@router.post("", response_model=InviteResponse, status_code=201)
async def create_invite(body: InviteCreateRequest, db: DB, current: CurrentPlayer):
    """Gera um link de convite para entrar em um grupo (válido por 30 minutos, uso único)."""
    g_repo = GroupRepository(db)
    group = await g_repo.get(body.group_id)
    if not group:
        raise NotFoundError("Grupo não encontrado")

    # Must be group admin or global admin
    if current.role != PlayerRole.ADMIN:
        member = await g_repo.get_member(body.group_id, current.id)
        if not member or member.role != GroupMemberRole.ADMIN:
            raise ForbiddenError("Apenas admins do grupo podem criar convites")

    settings = get_settings()
    expires_at = datetime.now(timezone.utc) + timedelta(
        minutes=settings.invite_token_expire_minutes
    )
    token = secrets.token_urlsafe(32)

    repo = InviteRepository(db)
    invite = await repo.create(
        group_id=body.group_id,
        token=token,
        expires_at=expires_at,
        created_by_id=current.id,
    )
    return invite


@router.get("/{token}", response_model=dict)
async def get_invite_info(token: str, db: DB):
    """Retorna informações do convite (grupo destino) sem autenticação."""
    repo = InviteRepository(db)
    invite = await repo.get_by_token(token)
    if not invite:
        raise NotFoundError("Convite não encontrado")
    if invite.used:
        raise ForbiddenError("Convite já utilizado")
    if invite.expires_at < datetime.now(timezone.utc):
        raise ForbiddenError("Convite expirado")

    g_repo = GroupRepository(db)
    group = await g_repo.get(invite.group_id)

    return {
        "valid": True,
        "group_id": str(invite.group_id),
        "group_name": group.name if group else "—",
        "expires_at": invite.expires_at.isoformat(),
    }


@router.get("/{token}/check", response_model=InviteCheckResponse)
async def check_whatsapp(token: str, whatsapp: str, db: DB):
    """Verifica se um WhatsApp já tem conta. Requer token válido para evitar enumeração."""
    repo = InviteRepository(db)
    invite = await repo.get_valid_token(token)
    if not invite:
        raise NotFoundError("Convite inválido ou expirado")

    normalized = re.sub(r"\D", "", whatsapp)
    p_repo = PlayerRepository(db)
    player = await p_repo.get_by_whatsapp(normalized)

    if player:
        first_name = player.name.split()[0]
        return InviteCheckResponse(exists=True, first_name=first_name)
    return InviteCheckResponse(exists=False)


@router.post("/{token}/accept", response_model=TokenResponse)
async def accept_invite(token: str, body: InviteAcceptRequest, db: DB):
    """
    Aceita um convite: cria o jogador (se não existe) e entra no grupo.
    Retorna um token JWT para login imediato.
    """
    repo = InviteRepository(db)
    invite = await repo.get_valid_token(token)
    if not invite:
        raise NotFoundError("Convite inválido ou expirado")

    whatsapp = re.sub(r"\D", "", body.whatsapp)
    p_repo = PlayerRepository(db)
    g_repo = GroupRepository(db)

    player = await p_repo.get_by_whatsapp(whatsapp)

    just_joined = False
    if player:
        # Usuário existente — valida senha antes de adicionar ao grupo
        if not verify_password(body.password, player.password_hash):
            raise ForbiddenError("Senha incorreta")
        existing_membership = await g_repo.get_member(invite.group_id, player.id)
        if not existing_membership:
            member_count = len(await g_repo.get_non_admin_member_ids(invite.group_id))
            if member_count >= _FREE_MEMBERS_LIMIT:
                raise PlanLimitError()
            await g_repo.add_member(invite.group_id, player.id, GroupMemberRole.MEMBER)
            just_joined = True
    else:
        # Novo usuário — name é obrigatório
        if not body.name or not body.name.strip():
            raise ValidationError("Nome é obrigatório para novo cadastro")
        # Verifica limite antes de criar o player (evita player órfão em caso de rollback)
        member_count = len(await g_repo.get_non_admin_member_ids(invite.group_id))
        if member_count >= _FREE_MEMBERS_LIMIT:
            raise PlanLimitError()
        player = await p_repo.create(
            name=body.name,
            nickname=body.nickname,
            whatsapp=whatsapp,
            password_hash=hash_password(body.password),
            role=PlayerRole.PLAYER,
        )
        # Auto-cria subscription gratuita para o novo player
        sub_repo = SubscriptionRepository(db)
        await sub_repo.get_or_create(player.id)
        await g_repo.add_member(invite.group_id, player.id, GroupMemberRole.MEMBER)
        just_joined = True

    # Adiciona o jogador como pendente nos rachões abertos do grupo
    if just_joined:
        m_repo = MatchRepository(db)
        open_matches = await m_repo.get_open_matches(invite.group_id)
        for match in open_matches:
            existing_att = await m_repo.get_attendance(match.id, player.id)
            if not existing_att:
                await m_repo.create_pending_attendances(match.id, [player.id])

    # Mark invite as used
    invite.used = True
    invite.used_by_id = player.id
    await db.flush()

    access_token = create_access_token(str(player.id))
    return TokenResponse(
        access_token=access_token,
        player_id=str(player.id),
        name=player.name,
        role=player.role,
    )
