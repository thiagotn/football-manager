import asyncio

from twilio.base.exceptions import TwilioRestException
from twilio.rest import Client

from app.core.config import get_settings


def _send(whatsapp: str) -> None:
    s = get_settings()
    Client(s.twilio_account_sid, s.twilio_auth_token) \
        .verify.v2.services(s.twilio_verify_sid) \
        .verifications.create(to=f"+55{whatsapp}", channel="sms")


def _check(whatsapp: str, code: str) -> str:
    s = get_settings()
    check = Client(s.twilio_account_sid, s.twilio_auth_token) \
        .verify.v2.services(s.twilio_verify_sid) \
        .verification_checks.create(to=f"+55{whatsapp}", code=code)
    return check.status


async def send_otp(whatsapp: str) -> None:
    """Send OTP to WhatsApp number via Twilio Verify (WhatsApp channel)."""
    await asyncio.to_thread(_send, whatsapp)


async def check_otp(whatsapp: str, code: str) -> bool:
    """Verify OTP. Returns True if approved, False otherwise."""
    try:
        status = await asyncio.to_thread(_check, whatsapp, code)
        return status == "approved"
    except TwilioRestException:
        return False
