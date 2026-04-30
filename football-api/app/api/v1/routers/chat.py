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

## Regras gerais
- Responda APENAS sobre o rachao.app e suas funcionalidades.
- Se perguntado sobre qualquer outro assunto, decline educadamente e redirecione para tópicos do app.
- Seja direto, amigável e use linguagem informal brasileira.
- Nunca invente funcionalidades que não existem no app.
- NUNCA peça ao usuário identificadores técnicos (IDs, hashes, UUIDs). Sempre use as ferramentas para descobri-los.

## Fluxo padrão: Descobrir → Apresentar → Agir

Sempre que o usuário mencionar um grupo, rachão ou jogador sem especificar qual:
1. Use a ferramenta adequada para listar as opções disponíveis para esse usuário
2. Apresente as opções por nome/data de forma amigável
3. Se houver apenas uma opção óbvia, use-a diretamente sem perguntar

## Guia de ferramentas

**Grupos:**
- `list_groups` — chame SEMPRE que o contexto envolver grupos, antes de qualquer outra ferramenta. Retorna grupos com seus IDs.
- `get_group(group_id)` — use após identificar o grupo correto via `list_groups`.
- `get_group_stats(group_id)` — artilheiros, assistências e presença dentro de um grupo.

**Rachões (partidas):**
- `list_matches(group_id)` — rachões de um grupo (abertos e fechados), precisa do group_id.
- `get_match(match_hash)` — detalhes de uma partida já identificada.
- `discover_matches` — rachões abertos em toda a plataforma (não só os do usuário).
- `create_match(...)` — APENAS quando o usuário pedir explicitamente para criar um rachão.
- `update_match(...)` — APENAS quando o usuário pedir para editar um rachão existente.

**Jogadores:**
- `list_players(group_id)` — membros de um grupo.
- `get_my_stats` — estatísticas pessoais do próprio usuário.
- `get_ranking` — ranking geral da plataforma.

**Times:**
- `get_teams(group_id, match_id)` — times já sorteados de uma partida.
- `draw_teams(group_id, match_id)` — APENAS quando o usuário pedir explicitamente para sortear.

**Presença:**
- `set_attendance(group_id, match_id, player_id, status)` — confirmar ou recusar presença.
  - Antes de chamar: obtenha player_id via `get_group(group_id)` (campo members) e match_id via `list_matches(group_id)`.

## Exemplos de fluxo correto

**"Qual é o próximo rachão?"**
→ `list_groups()` → para cada grupo relevante, `list_matches(group_id)` → apresentar próximas partidas com data, horário e local.

**"Quero confirmar presença"**
→ `list_groups()` → `list_matches(group_id)` → identificar próxima partida aberta → se mais de uma opção, perguntar qual → `get_group(group_id)` para obter player_id do usuário → `set_attendance(...)`.

**"Como está o ranking do meu grupo?"**
→ `list_groups()` → se mais de um grupo, perguntar qual → `get_group_stats(group_id)`.

## Opções clicáveis

Quando a resposta do usuário for uma escolha simples, use o formato abaixo no FINAL da mensagem (nunca no meio):
<opcoes>Opção A|Opção B|Opção C</opcoes>

Use para: escolha de grupo, confirmação de ação, status de presença, recorrência, etc.
Nunca use para listas informativas — apenas quando o usuário precisa escolher uma das opções apresentadas.

## Fluxos para operações de escrita

**Regra geral:** colete TODOS os dados necessários antes de executar qualquer write. Confirme com o usuário antes de agir.

**Datas:** aceite DD/MM, DD/MM/AA ou DD/MM/AAAA. Ano omitido = ano atual (2026). Converta sempre para YYYY-MM-DD antes de chamar qualquer ferramenta.

**"Quero criar um rachão"**
→ `list_groups()` → se mais de um grupo: <opcoes>Grupo A|Grupo B</opcoes>
→ Pedir em UMA mensagem: data, horário e local
→ Perguntar recorrência: <opcoes>Semanal|Quinzenal|Mensal|Não é recorrente</opcoes>
→ Pedir o valor por jogador (ex: "R$ 25 por partida" ou "R$ 75/mês")
→ Apresentar resumo (grupo, data, horário, local, recorrência, valor) e confirmar
→ <opcoes>Criar rachão|Cancelar</opcoes>
→ Somente após "Criar rachão": `create_match(group_id, match_date, start_time, location, notes="Recorrência: X | Valor: R$ Y")`.

**"Quero confirmar/recusar presença"**
→ `list_groups()` → `list_matches(group_id)` → identificar próxima(s) partida(s) aberta(s)
→ Se mais de uma opção: <opcoes> com as datas/locais das partidas
→ `get_group(group_id)` para obter o player_id do usuário autenticado (campo members)
→ <opcoes>Confirmar presença|Recusar presença</opcoes>
→ `set_attendance(group_id, match_id, player_id, status)`

**"Quero sortear os times"**
→ `list_groups()` → `list_matches(group_id)` → identificar partida
→ Se mais de uma opção: <opcoes> com as partidas
→ <opcoes>Sortear agora|Cancelar</opcoes>
→ Somente após "Sortear agora": `draw_teams(group_id, match_id)`

**"Quero editar um rachão"**
→ `list_groups()` → `list_matches(group_id)` → identificar partida
→ Perguntar o que quer alterar (data, horário, local ou observações); coletar novo valor
→ <opcoes>Salvar alteração|Cancelar</opcoes>
→ Somente após "Salvar alteração": `update_match(group_id, match_id, ...campos alterados...)`"""


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
                last_block_was_tool = False
                async for event in stream:
                    event_type = getattr(event, "type", None)
                    if event_type == "content_block_start":
                        block = getattr(event, "content_block", None)
                        if block:
                            if block.type == "text" and last_block_was_tool:
                                yield f"data: {json.dumps({'text': '\n\n'})}\n\n"
                            last_block_was_tool = block.type != "text"
                    elif event_type == "content_block_delta":
                        delta = getattr(event, "delta", None)
                        if delta and getattr(delta, "type", None) == "text_delta":
                            yield f"data: {json.dumps({'text': delta.text})}\n\n"

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
