"""
Testes unitários — app/services/telegram.py

Regras cobertas:
- Sem credenciais configuradas → retorna sem chamar httpx
- Com credenciais → chama sendMessage com payload correto
- Falha no httpx → não propaga exceção (apenas loga)
"""
from unittest.mock import AsyncMock, MagicMock, patch

import pytest

from app.services import telegram


def _patch_settings(token: str = "", chat_id: str = ""):
    s = MagicMock()
    s.telegram_bot_token = token
    s.telegram_chat_id = chat_id
    return s


@pytest.mark.asyncio
async def test_notify_no_op_when_token_missing(mocker):
    """Sem TELEGRAM_BOT_TOKEN configurado, não faz nenhuma chamada HTTP."""
    mocker.patch("app.services.telegram.get_settings", return_value=_patch_settings(token="", chat_id="123"))
    mock_client = mocker.patch("app.services.telegram.httpx.AsyncClient")

    await telegram.notify_new_player("Thiago", "+5511999990000", "cadastro")

    mock_client.assert_not_called()


@pytest.mark.asyncio
async def test_notify_no_op_when_chat_id_missing(mocker):
    """Sem TELEGRAM_CHAT_ID configurado, não faz nenhuma chamada HTTP."""
    mocker.patch("app.services.telegram.get_settings", return_value=_patch_settings(token="abc:123", chat_id=""))
    mock_client = mocker.patch("app.services.telegram.httpx.AsyncClient")

    await telegram.notify_new_player("Thiago", "+5511999990000", "cadastro")

    mock_client.assert_not_called()


@pytest.mark.asyncio
async def test_notify_sends_correct_payload(mocker):
    """Com credenciais configuradas, envia POST para sendMessage com payload correto."""
    mocker.patch(
        "app.services.telegram.get_settings",
        return_value=_patch_settings(token="BOT_TOKEN", chat_id="CHAT_ID"),
    )
    mock_resp = MagicMock()
    mock_resp.raise_for_status = MagicMock()
    mock_post = AsyncMock(return_value=mock_resp)

    mock_client_instance = AsyncMock()
    mock_client_instance.__aenter__ = AsyncMock(return_value=mock_client_instance)
    mock_client_instance.__aexit__ = AsyncMock(return_value=False)
    mock_client_instance.post = mock_post

    mocker.patch("app.services.telegram.httpx.AsyncClient", return_value=mock_client_instance)

    await telegram.notify_new_player("Thiago", "+5511999990000", "admin")

    mock_post.assert_called_once()
    call_kwargs = mock_post.call_args
    assert "api.telegram.org/botBOT_TOKEN/sendMessage" in call_kwargs[0][0]
    payload = call_kwargs[1]["json"]
    assert payload["chat_id"] == "CHAT_ID"
    assert "Thiago" in payload["text"]
    assert "+5511999990000" in payload["text"]
    assert "admin" in payload["text"]
    assert payload["parse_mode"] == "Markdown"


@pytest.mark.asyncio
async def test_notify_does_not_raise_on_http_error(mocker):
    """Falha na chamada HTTP é logada mas não propaga exceção."""
    mocker.patch(
        "app.services.telegram.get_settings",
        return_value=_patch_settings(token="BOT_TOKEN", chat_id="CHAT_ID"),
    )
    mock_client_instance = AsyncMock()
    mock_client_instance.__aenter__ = AsyncMock(return_value=mock_client_instance)
    mock_client_instance.__aexit__ = AsyncMock(return_value=False)
    mock_client_instance.post = AsyncMock(side_effect=Exception("connection refused"))

    mocker.patch("app.services.telegram.httpx.AsyncClient", return_value=mock_client_instance)
    mock_log = mocker.patch("app.services.telegram.logger")

    await telegram.notify_new_player("Thiago", "+5511999990000", "convite")

    mock_log.warning.assert_called_once_with("telegram_notify_failed", error="connection refused")
