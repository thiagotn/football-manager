import re

from fastapi import APIRouter, HTTPException, status
from twilio.base.exceptions import TwilioRestException

from app.core.dependencies import DB, CurrentPlayer
from app.core.exceptions import ConflictError, UnauthorizedError, ValidationError
from app.core.security import (
    create_access_token,
    create_otp_token,
    decode_otp_token,
    hash_password,
    verify_password,
)
from app.db.repositories.player_repo import PlayerRepository
from app.db.repositories.subscription_repo import SubscriptionRepository
from app.models.player import PlayerRole
from app.schemas.auth import (
    ChangePasswordRequest,
    LoginRequest,
    RegisterRequest,
    SendOtpRequest,
    SendOtpResponse,
    TokenResponse,
    VerifyOtpRequest,
    VerifyOtpResponse,
)
from app.schemas.player import PlayerResponse
from app.services import twilio_verify

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


@router.post("/send-otp", response_model=SendOtpResponse)
async def send_otp(body: SendOtpRequest, db: DB):
    """Send OTP verification code to WhatsApp number."""
    whatsapp = re.sub(r"\D", "", body.whatsapp)

    if await PlayerRepository(db).get_by_whatsapp(whatsapp):
        raise ConflictError("WhatsApp já cadastrado")

    try:
        await twilio_verify.send_otp(whatsapp)
    except TwilioRestException as e:
        if e.code in (60200, 60203):
            raise HTTPException(
                status_code=status.HTTP_429_TOO_MANY_REQUESTS,
                detail="Muitas tentativas. Aguarde antes de solicitar um novo código.",
            )
        raise HTTPException(
            status_code=status.HTTP_502_BAD_GATEWAY,
            detail="Falha ao enviar código. Tente novamente.",
        )

    return SendOtpResponse()


@router.post("/verify-otp", response_model=VerifyOtpResponse)
async def verify_otp(body: VerifyOtpRequest, db: DB):
    """Verify OTP code and return a signed token confirming phone ownership."""
    whatsapp = re.sub(r"\D", "", body.whatsapp)

    if await PlayerRepository(db).get_by_whatsapp(whatsapp):
        raise ConflictError("WhatsApp já cadastrado")

    approved = await twilio_verify.check_otp(whatsapp, body.otp_code)
    if not approved:
        raise ValidationError("OTP_INVALID")

    return VerifyOtpResponse(otp_token=create_otp_token(whatsapp))


@router.post("/register", response_model=TokenResponse, status_code=201)
async def register(body: RegisterRequest, db: DB):
    """Create new player account with free plan. Requires a valid otp_token."""
    verified_whatsapp = decode_otp_token(body.otp_token)
    if not verified_whatsapp:
        raise ValidationError("OTP_TOKEN_INVALID")

    whatsapp = re.sub(r"\D", "", body.whatsapp)
    if whatsapp != verified_whatsapp:
        raise ValidationError("OTP_TOKEN_INVALID")

    repo = PlayerRepository(db)
    if await repo.get_by_whatsapp(whatsapp):
        raise ConflictError("WhatsApp já cadastrado")

    player = await repo.create(
        name=body.name.strip(),
        nickname=body.nickname,
        whatsapp=whatsapp,
        password_hash=hash_password(body.password),
        role=PlayerRole.PLAYER,
    )
    await SubscriptionRepository(db).get_or_create(player.id)

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
