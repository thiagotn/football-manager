import asyncio
import structlog

from twilio.base.exceptions import TwilioRestException
from twilio.rest import Client

from app.core.config import get_settings

log = structlog.get_logger()


def _is_bypass_active() -> bool:
    s = get_settings()
    return bool(s.otp_bypass_code) and not s.is_prod


def _send(whatsapp: str) -> None:
    # whatsapp is E.164 (e.g. +5511999990000)
    s = get_settings()
    Client(s.twilio_account_sid, s.twilio_auth_token) \
        .verify.v2.services(s.twilio_verify_sid) \
        .verifications.create(to=whatsapp, channel="sms")


def _check(whatsapp: str, code: str) -> str:
    # whatsapp is E.164 (e.g. +5511999990000)
    s = get_settings()
    check = Client(s.twilio_account_sid, s.twilio_auth_token) \
        .verify.v2.services(s.twilio_verify_sid) \
        .verification_checks.create(to=whatsapp, code=code)
    return check.status


async def send_otp(whatsapp: str) -> None:
    """Send OTP to WhatsApp number via Twilio Verify (WhatsApp channel)."""
    if _is_bypass_active():
        log.info("OTP bypass ativo — envio simulado, Twilio não foi chamado", whatsapp=whatsapp)
        return
    await asyncio.to_thread(_send, whatsapp)


async def check_otp(whatsapp: str, code: str) -> bool:
    """Verify OTP. Returns True if approved, False otherwise."""
    if _is_bypass_active():
        accepted = code == get_settings().otp_bypass_code
        log.info("OTP bypass ativo — verificação local", whatsapp=whatsapp, accepted=accepted)
        return accepted
    try:
        status = await asyncio.to_thread(_check, whatsapp, code)
        return status == "approved"
    except TwilioRestException:
        return False
