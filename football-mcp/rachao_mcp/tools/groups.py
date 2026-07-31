from rachao_mcp.client import api

_READ = {"readOnlyHint": True, "idempotentHint": True}


async def list_groups() -> list[dict]:
    """Lista todos os grupos do usuário autenticado."""
    return await api.get("/groups")


async def get_group(group_id: str) -> dict:
    """Detalhes de um grupo: membros, stats e slots de times."""
    return await api.get(f"/groups/{group_id}")


async def get_group_stats(group_id: str, period: str = "annual", month: str | None = None) -> dict:
    """Estatísticas agregadas do grupo por jogador: pontos de votação, votos de flop e minutos jogados.
    period: annual (padrão) | monthly. month: YYYY-MM (usado com period=monthly).
    Para gols/assistências ou melhor jogador de UMA partida, use get_match_stats / get_vote_results."""
    params = f"?period={period}" + (f"&month={month}" if month else "")
    return await api.get(f"/groups/{group_id}/stats{params}")


READ_TOOLS: list[tuple] = [
    (list_groups, _READ),
    (get_group, _READ),
    (get_group_stats, _READ),
]
