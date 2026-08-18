from datetime import datetime, timezone

import structlog
from fastapi import APIRouter, HTTPException, status
from sqlalchemy.exc import IntegrityError
from twilio.base.exceptions import TwilioRestException

from app.core.dependencies import DB
from app.core.exceptions import ConflictError, ForbiddenError, NotFoundError, UnauthorizedError, ValidationError
from app.core.security import create_access_token, create_otp_token, decode_otp_token, hash_password
from app.db.repositories.group_repo import GroupRepository
from app.db.repositories.invite_repo import InviteRepository
from app.db.repositories.player_repo import PlayerRepository
from app.db.repositories.refresh_token_repo import RefreshTokenRepository
from app.models.invite import InviteToken
from app.models.player import Player
from app.schemas.auth import TokenResponse, VerifyOtpResponse
from app.schemas.invite import (
    ClaimCompleteRequest,
    ClaimInfoResponse,
    ClaimSendOtpRequest,
    ClaimVerifyOtpRequest,
)
from app.services import twilio_verify

router = APIRouter(prefix="/claims", tags=["claims"])
logger = structlog.get_logger()


async def _get_claim_invite_or_raise(db, token: str) -> tuple[InviteToken, Player]:
    """Valida token de claim (existência, uso, expiração, alvo ainda pendente)."""
    invite = await InviteRepository(db).get_by_token(token)
    if not invite or invite.purpose != "registration_claim" or not invite.target_player_id:
        raise NotFoundError("INVITE_NOT_FOUND")
    if invite.used:
        raise ForbiddenError("INVITE_USED")
    if invite.expires_at < datetime.now(timezone.utc):
        raise ForbiddenError("INVITE_EXPIRED")

    player = await PlayerRepository(db).get(invite.target_player_id)
    if not player:
        raise NotFoundError("INVITE_NOT_FOUND")
    if not player.pending_registration:
        raise ForbiddenError("ALREADY_CLAIMED")
    return invite, player


def _raise_twilio_http_error(e: TwilioRestException) -> None:
    if e.code in (60200, 60203):
        raise HTTPException(
            status_code=status.HTTP_429_TOO_MANY_REQUESTS,
            detail="Muitas tentativas. Aguarde antes de solicitar um novo código.",
        )
    raise HTTPException(
        status_code=status.HTTP_502_BAD_GATEWAY,
        detail="Falha ao enviar código. Tente novamente.",
    )


@router.get("/{token}", response_model=ClaimInfoResponse)
async def get_claim_info(token: str, db: DB):
    """Informações públicas do convite de claim (primeiro nome do jogador + grupo)."""
    invite, player = await _get_claim_invite_or_raise(db, token)
    group = await GroupRepository(db).get(invite.group_id)
    return ClaimInfoResponse(
        valid=True,
        player_first_name=player.name.split()[0],
        group_name=group.name if group else "—",
        expires_at=invite.expires_at,
    )


@router.post("/{token}/send-otp")
async def claim_send_otp(token: str, body: ClaimSendOtpRequest, db: DB):
    """Envia OTP para o número REAL informado pelo jogador que está assumindo o cadastro."""
    _, player = await _get_claim_invite_or_raise(db, token)

    existing = await PlayerRepository(db).get_by_whatsapp(body.whatsapp)
    if existing and existing.id != player.id:
        raise ConflictError("WHATSAPP_TAKEN")

    try:
        await twilio_verify.send_otp(body.whatsapp)
    except TwilioRestException as e:
        _raise_twilio_http_error(e)

    return {"status": "pending"}


@router.post("/{token}/verify-otp", response_model=VerifyOtpResponse)
async def claim_verify_otp(token: str, body: ClaimVerifyOtpRequest, db: DB):
    """Confere o OTP e devolve um otp_token assinado comprovando posse do número."""
    _, player = await _get_claim_invite_or_raise(db, token)

    existing = await PlayerRepository(db).get_by_whatsapp(body.whatsapp)
    if existing and existing.id != player.id:
        raise ConflictError("WHATSAPP_TAKEN")

    approved = await twilio_verify.check_otp(body.whatsapp, body.otp_code)
    if not approved:
        raise ValidationError("OTP_INVALID")

    return VerifyOtpResponse(otp_token=create_otp_token(body.whatsapp))


@router.post("/{token}/complete", response_model=TokenResponse)
async def claim_complete(token: str, body: ClaimCompleteRequest, db: DB):
    """
    Conclui o claim: atualiza whatsapp + senha do jogador pendente, limpa as flags,
    marca o convite como usado e devolve tokens de login.
    """
    invite, player = await _get_claim_invite_or_raise(db, token)

    verified_whatsapp = decode_otp_token(body.otp_token)
    if not verified_whatsapp or verified_whatsapp != body.whatsapp:
        raise UnauthorizedError("OTP_TOKEN_INVALID")

    p_repo = PlayerRepository(db)
    existing = await p_repo.get_by_whatsapp(body.whatsapp)
    if existing and existing.id != player.id:
        raise ConflictError("WHATSAPP_TAKEN")

    rt_repo = RefreshTokenRepository(db)
    player.whatsapp = body.whatsapp
    player.password_hash = hash_password(body.password)
    player.must_change_password = False
    player.pending_registration = False
    await rt_repo.revoke_all_for_player(player.id)

    invite.used = True
    invite.used_by_id = player.id

    try:
        await db.flush()
    except IntegrityError:
        # Corrida: outro cadastro assumiu o número entre o verify-otp e o complete
        raise ConflictError("WHATSAPP_TAKEN")

    logger.info("claim_completed", player_id=str(player.id), group_id=str(invite.group_id))

    access_token = create_access_token(str(player.id))
    refresh_token = await rt_repo.create(player.id)
    return TokenResponse(
        access_token=access_token,
        refresh_token=refresh_token,
        player_id=str(player.id),
        name=player.name,
        nickname=player.nickname,
        role=player.role,
        must_change_password=False,
        avatar_url=player.avatar_url,
        chat_enabled=player.chat_enabled,
    )
