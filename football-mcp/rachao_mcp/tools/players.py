from rachao_mcp.client import api

_READ = {"readOnlyHint": True, "idempotentHint": True}


async def list_players(group_id: str) -> list[dict]:
    """Lista os membros (jogadores) de um grupo."""
    return await api.get(f"/groups/{group_id}/members")


async def get_my_stats() -> dict:
    """Estatísticas completas do jogador autenticado: gols, assistências e presença."""
    return await api.get("/players/me/stats/full")


async def get_ranking() -> list[dict]:
    """Ranking geral da plataforma."""
    return await api.get("/ranking")


READ_TOOLS: list[tuple] = [
    (list_players, _READ),
    (get_my_stats, _READ),
    (get_ranking, _READ),
]
