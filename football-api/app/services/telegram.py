import httpx
import structlog

from app.core.config import get_settings

logger = structlog.get_logger()


async def notify_new_player(name: str, whatsapp: str, source: str) -> None:
    """Envia mensagem no Telegram quando um novo jogador é cadastrado.

    Fire-and-forget: erros são logados mas nunca propagados.
    No-op se TELEGRAM_BOT_TOKEN ou TELEGRAM_CHAT_ID não estiverem configurados.
    """
    settings = get_settings()
    if not settings.telegram_bot_token or not settings.telegram_chat_id:
        return

    text = (
        f"\U0001f195 *Novo jogador cadastrado!*\n"
        f"\U0001f464 Nome: {name}\n"
        f"\U0001f4f1 WhatsApp: {whatsapp}\n"
        f"\U0001f4cb Via: {source}"
    )
    url = f"https://api.telegram.org/bot{settings.telegram_bot_token}/sendMessage"
    payload = {
        "chat_id": settings.telegram_chat_id,
        "text": text,
        "parse_mode": "Markdown",
    }
    try:
        async with httpx.AsyncClient(timeout=10) as client:
            resp = await client.post(url, json=payload)
            resp.raise_for_status()
    except Exception as exc:
        logger.warning("telegram_notify_failed", error=str(exc))
