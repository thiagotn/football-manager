"""
Testes unitários — app/services/twilio_verify.py

Regras de negócio cobertas:
- send_otp em bypass não chama Twilio
- check_otp em bypass aceita o código correto
- check_otp em bypass rejeita código errado
- bypass nunca é ativo quando APP_ENV=production
"""
from unittest.mock import AsyncMock, patch

import pytest

from app.services import twilio_verify


def _patch_settings(otp_bypass_code: str, is_prod: bool):
    """Retorna um mock de Settings com os valores desejados."""
    from unittest.mock import MagicMock
    s = MagicMock()
    s.otp_bypass_code = otp_bypass_code
    s.is_prod = is_prod
    s.twilio_account_sid = "ACfake"
    s.twilio_auth_token = "fake"
    s.twilio_verify_sid = "VAfake"
    return s


# ── Bypass ativo (dev + código definido) ──────────────────────────────────────


@pytest.mark.asyncio
async def test_send_otp_bypass_skips_twilio(mocker):
    """Com bypass ativo, send_otp não chama asyncio.to_thread (Twilio)."""
    mocker.patch(
        "app.services.twilio_verify.get_settings",
        return_value=_patch_settings(otp_bypass_code="000000", is_prod=False),
    )
    mock_thread = mocker.patch("app.services.twilio_verify.asyncio.to_thread", new=AsyncMock())

    await twilio_verify.send_otp("+5511999990001")

    mock_thread.assert_not_called()


@pytest.mark.asyncio
async def test_check_otp_bypass_accepts_correct_code(mocker):
    """Com bypass ativo, código correto retorna True."""
    mocker.patch(
        "app.services.twilio_verify.get_settings",
        return_value=_patch_settings(otp_bypass_code="000000", is_prod=False),
    )
    mocker.patch("app.services.twilio_verify.asyncio.to_thread", new=AsyncMock())

    result = await twilio_verify.check_otp("+5511999990001", "000000")

    assert result is True


@pytest.mark.asyncio
async def test_check_otp_bypass_rejects_wrong_code(mocker):
    """Com bypass ativo, código errado retorna False."""
    mocker.patch(
        "app.services.twilio_verify.get_settings",
        return_value=_patch_settings(otp_bypass_code="000000", is_prod=False),
    )
    mocker.patch("app.services.twilio_verify.asyncio.to_thread", new=AsyncMock())

    result = await twilio_verify.check_otp("+5511999990001", "999999")

    assert result is False


# ── Bypass bloqueado em produção ──────────────────────────────────────────────


@pytest.mark.asyncio
async def test_bypass_disabled_in_production(mocker):
    """Com APP_ENV=production, o bypass não é ativado mesmo com OTP_BYPASS_CODE definido."""
    mocker.patch(
        "app.services.twilio_verify.get_settings",
        return_value=_patch_settings(otp_bypass_code="000000", is_prod=True),
    )
    mock_thread = mocker.patch("app.services.twilio_verify.asyncio.to_thread", new=AsyncMock(return_value="approved"))

    result = await twilio_verify.check_otp("+5511999990001", "000000")

    # Em produção, deve chamar o Twilio (asyncio.to_thread), não o bypass
    mock_thread.assert_called_once()
    assert result is True


# ── Bypass desabilitado (código vazio) ────────────────────────────────────────


@pytest.mark.asyncio
async def test_bypass_disabled_when_code_empty(mocker):
    """Com OTP_BYPASS_CODE vazio, o fluxo real do Twilio é chamado."""
    mocker.patch(
        "app.services.twilio_verify.get_settings",
        return_value=_patch_settings(otp_bypass_code="", is_prod=False),
    )
    mock_thread = mocker.patch("app.services.twilio_verify.asyncio.to_thread", new=AsyncMock())

    await twilio_verify.send_otp("+5511999990001")

    mock_thread.assert_called_once()
