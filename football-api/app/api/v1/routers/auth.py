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
    ForgotPasswordResetRequest,
    LoginRequest,
    RegisterRequest,
    SendOtpRequest,
    SendOtpResponse,
    TokenResponse,
    VerifyOtpMeRequest,
    VerifyOtpRequest,
    VerifyOtpResponse,
)
from app.schemas.player import PlayerResponse
from app.services import twilio_verify

router = APIRouter(prefix="/auth", tags=["auth"])


@router.post("/login", response_model=TokenResponse)
async def login(body: LoginRequest, db: DB):
    repo = PlayerRepository(db)
    player = await repo.get_by_whatsapp(body.whatsapp)

    if not player or not verify_password(body.password, player.password_hash):
        raise UnauthorizedError("WhatsApp ou senha incorretos")
    if not player.active:
        raise UnauthorizedError("Conta desativada")

    token = create_access_token(str(player.id))
    return TokenResponse(
        access_token=token,
        player_id=str(player.id),
        name=player.name,
        nickname=player.nickname,
        role=player.role,
        must_change_password=player.must_change_password,
    )


@router.post("/send-otp", response_model=SendOtpResponse)
async def send_otp(body: SendOtpRequest, db: DB):
    """Send OTP verification code to WhatsApp number."""
    if await PlayerRepository(db).get_by_whatsapp(body.whatsapp):
        raise ConflictError("WhatsApp já cadastrado")

    try:
        await twilio_verify.send_otp(body.whatsapp)
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
    if await PlayerRepository(db).get_by_whatsapp(body.whatsapp):
        raise ConflictError("WhatsApp já cadastrado")

    approved = await twilio_verify.check_otp(body.whatsapp, body.otp_code)
    if not approved:
        raise ValidationError("OTP_INVALID")

    return VerifyOtpResponse(otp_token=create_otp_token(body.whatsapp))


@router.post("/register", response_model=TokenResponse, status_code=201)
async def register(body: RegisterRequest, db: DB):
    """Create new player account with free plan. Requires a valid otp_token."""
    verified_whatsapp = decode_otp_token(body.otp_token)
    if not verified_whatsapp:
        raise ValidationError("OTP_TOKEN_INVALID")

    if body.whatsapp != verified_whatsapp:
        raise ValidationError("OTP_TOKEN_INVALID")

    repo = PlayerRepository(db)
    if await repo.get_by_whatsapp(body.whatsapp):
        raise ConflictError("WhatsApp já cadastrado")

    player = await repo.create(
        name=body.name.strip(),
        nickname=body.nickname,
        whatsapp=body.whatsapp,
        password_hash=hash_password(body.password),
        role=PlayerRole.PLAYER,
    )
    await SubscriptionRepository(db).get_or_create(player.id)

    token = create_access_token(str(player.id))
    return TokenResponse(
        access_token=token,
        player_id=str(player.id),
        name=player.name,
        nickname=player.nickname,
        role=player.role,
        must_change_password=False,
    )


@router.get("/me", response_model=PlayerResponse)
async def me(current: CurrentPlayer):
    return current


@router.post("/forgot-password/send-otp", response_model=SendOtpResponse)
async def forgot_password_send_otp(body: SendOtpRequest, db: DB):
    """Send OTP to a registered WhatsApp number for password reset."""
    if not await PlayerRepository(db).get_by_whatsapp(body.whatsapp):
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="Número não encontrado. Verifique e tente novamente.",
        )

    try:
        await twilio_verify.send_otp(body.whatsapp)
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


@router.post("/forgot-password/verify-otp", response_model=VerifyOtpResponse)
async def forgot_password_verify_otp(body: VerifyOtpRequest, db: DB):
    """Verify OTP for a registered user and return a signed otp_token."""
    if not await PlayerRepository(db).get_by_whatsapp(body.whatsapp):
        raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail="Número não encontrado.")

    approved = await twilio_verify.check_otp(body.whatsapp, body.otp_code)
    if not approved:
        raise ValidationError("OTP_INVALID")

    return VerifyOtpResponse(otp_token=create_otp_token(body.whatsapp))


@router.post("/forgot-password/reset", status_code=204)
async def forgot_password_reset(body: ForgotPasswordResetRequest, db: DB):
    """Reset password using a verified OTP token."""
    verified_whatsapp = decode_otp_token(body.otp_token)
    if not verified_whatsapp or verified_whatsapp != body.whatsapp:
        raise UnauthorizedError("Token de verificação inválido ou expirado")

    repo = PlayerRepository(db)
    player = await repo.get_by_whatsapp(body.whatsapp)
    if not player:
        raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail="Usuário não encontrado")

    if verify_password(body.new_password, player.password_hash):
        raise HTTPException(
            status_code=status.HTTP_422_UNPROCESSABLE_ENTITY,
            detail="SAME_PASSWORD",
        )

    player.password_hash = hash_password(body.new_password)
    player.must_change_password = False
    await db.flush()


@router.post("/send-otp/me", response_model=SendOtpResponse)
async def send_otp_me(current: CurrentPlayer):
    """Send OTP to the authenticated user's own WhatsApp number."""
    try:
        await twilio_verify.send_otp(current.whatsapp)
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


@router.post("/verify-otp/me", response_model=VerifyOtpResponse)
async def verify_otp_me(body: VerifyOtpMeRequest, current: CurrentPlayer):
    """Verify OTP for the authenticated user and return a signed otp_token."""
    approved = await twilio_verify.check_otp(current.whatsapp, body.otp_code)
    if not approved:
        raise ValidationError("OTP_INVALID")
    return VerifyOtpResponse(otp_token=create_otp_token(current.whatsapp))


@router.post("/change-password", status_code=204)
async def change_password(body: ChangePasswordRequest, db: DB, current: CurrentPlayer):
    if body.otp_token:
        verified_whatsapp = decode_otp_token(body.otp_token)
        if not verified_whatsapp or verified_whatsapp != current.whatsapp:
            raise UnauthorizedError("Token de verificação inválido ou expirado")
    elif body.current_password:
        if not verify_password(body.current_password, current.password_hash):
            raise UnauthorizedError("Senha atual incorreta")
    else:
        raise HTTPException(
            status_code=status.HTTP_422_UNPROCESSABLE_ENTITY,
            detail="Informe a senha atual ou verifique via SMS.",
        )

    if verify_password(body.new_password, current.password_hash):
        raise HTTPException(
            status_code=status.HTTP_422_UNPROCESSABLE_ENTITY,
            detail="SAME_PASSWORD",
        )

    repo = PlayerRepository(db)
    player = await repo.get(current.id)
    player.password_hash = hash_password(body.new_password)
    player.must_change_password = False
    await db.flush()
