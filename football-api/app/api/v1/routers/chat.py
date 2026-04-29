import json
import logging
from datetime import datetime, timedelta, timezone
from uuid import UUID

import anthropic
from fastapi import APIRouter, Request
from fastapi.responses import StreamingResponse
from sqlalchemy import select, update
from sqlalchemy.ext.asyncio import AsyncSession

from app.core.config import get_settings
from app.core.dependencies import AdminPlayer, CurrentPlayer, DB
from app.core.exceptions import ForbiddenError, NotFoundError, RateLimitError
from app.models.player import Player, PlayerRole
from app.schemas.chat import ChatAccessUpdate, ChatRequest, ChatUserItem, ChatUsersResponse

logger = logging.getLogger(__name__)

router = APIRouter(tags=["chat"])

SYSTEM_PROMPT = """Você é o assistente oficial do rachao.app, uma plataforma para organização de peladas e rachões no Brasil.

Seu papel é ajudar usuários com dúvidas sobre funcionalidades, fluxos, pagamentos, convites, confirmações de presença e configurações do app.

Regras:
- Responda APENAS sobre o rachao.app e suas funcionalidades.
- Se perguntado sobre qualquer outro assunto, decline educadamente e redirecione para tópicos do app.
- Seja direto, amigável e use linguagem informal brasileira.
- Use as ferramentas disponíveis para buscar informações reais do produto quando necessário.
- Nunca invente funcionalidades que não existem no app."""


async def _check_and_increment_rate_limit(player: Player, db: AsyncSession) -> bool:
    """Returns True if request is allowed, False if rate-limited."""
    settings = get_settings()
    now = datetime.now(timezone.utc)

    window_expired = (
        player.chat_req_window is None
        or (now - player.chat_req_window) > timedelta(hours=1)
    )

    if window_expired:
        await db.execute(
            update(Player)
            .where(Player.id == player.id)
            .values(chat_req_count=1, chat_req_window=now)
        )
        return True

    if player.chat_req_count >= settings.chat_rate_limit:
        return False

    await db.execute(
        update(Player)
        .where(Player.id == player.id)
        .values(chat_req_count=Player.chat_req_count + 1)
    )
    return True


@router.post("/chat")
async def chat(body: ChatRequest, request: Request, current_player: CurrentPlayer, db: DB):
    if not current_player.chat_enabled:
        raise ForbiddenError("Acesso ao assistente não habilitado para este usuário")

    allowed = await _check_and_increment_rate_limit(current_player, db)
    if not allowed:
        raise RateLimitError()

    settings = get_settings()
    token = request.headers.get("Authorization", "").removeprefix("Bearer ").strip()
    messages = [{"role": m.role, "content": m.content} for m in body.messages]

    async def event_stream():
        try:
            if not settings.anthropic_api_key:
                yield f"data: {json.dumps({'error': 'Assistente não configurado. Contate o administrador.'})}\n\n"
                return
            client = anthropic.AsyncAnthropic(api_key=settings.anthropic_api_key)

            async with client.beta.messages.stream(
                model=settings.llm_model,
                max_tokens=1024,
                system=SYSTEM_PROMPT,
                messages=messages,
                betas=["mcp-client-2025-04-04"],
                mcp_servers=[
                    {
                        "type": "url",
                        "url": "https://mcp.rachao.app/mcp",
                        "name": "rachao",
                        "authorization_token": token,
                    }
                ],
            ) as stream:
                async for text_chunk in stream.text_stream:
                    yield f"data: {json.dumps({'text': text_chunk})}\n\n"

            yield "data: [DONE]\n\n"
        except anthropic.APIError as e:
            logger.error("anthropic_api_error error=%s", str(e))
            yield f"data: {json.dumps({'error': 'Erro ao conectar com o assistente. Tente novamente.'})}\n\n"
        except Exception as e:
            logger.error("chat_stream_error type=%s error=%s", type(e).__name__, str(e))
            yield f"data: {json.dumps({'error': 'Erro interno. Tente novamente.'})}\n\n"

    logger.info("chat_request player_id=%s model=%s", str(current_player.id), settings.llm_model)

    return StreamingResponse(
        event_stream(),
        media_type="text/event-stream",
        headers={
            "Cache-Control": "no-cache",
            "X-Accel-Buffering": "no",
        },
    )


@router.get("/admin/chat-users", response_model=ChatUsersResponse)
async def list_chat_users(db: DB, _admin: AdminPlayer):
    result = await db.execute(
        select(Player)
        .where(Player.role == PlayerRole.PLAYER)
        .order_by(Player.created_at.desc())
    )
    players = result.scalars().all()

    users = [ChatUserItem.model_validate(p) for p in players]
    total_enabled = sum(1 for u in users if u.chat_enabled)

    return ChatUsersResponse(users=users, total_enabled=total_enabled)


@router.patch("/admin/chat-users/{user_id}", response_model=ChatUserItem)
async def update_chat_access(user_id: UUID, body: ChatAccessUpdate, db: DB, _admin: AdminPlayer):
    result = await db.execute(select(Player).where(Player.id == user_id))
    player = result.scalar_one_or_none()

    if not player:
        raise NotFoundError("Usuário não encontrado")

    await db.execute(
        update(Player).where(Player.id == user_id).values(chat_enabled=body.chat_enabled)
    )

    result = await db.execute(select(Player).where(Player.id == user_id))
    updated = result.scalar_one()
    return ChatUserItem.model_validate(updated)
