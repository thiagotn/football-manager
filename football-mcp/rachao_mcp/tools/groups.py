from rachao_mcp.client import api

_READ = {"readOnlyHint": True, "idempotentHint": True}


async def list_groups() -> list[dict]:
    """Lista todos os grupos do usuário autenticado."""
    return await api.get("/groups")


async def get_group(group_id: str) -> dict:
    """Detalhes de um grupo: membros, stats e slots de times."""
    return await api.get(f"/groups/{group_id}")


async def get_group_stats(group_id: str) -> dict:
    """Artilheiros, assistências e presença por jogador do grupo."""
    return await api.get(f"/groups/{group_id}/stats")


READ_TOOLS: list[tuple] = [
    (list_groups, _READ),
    (get_group, _READ),
    (get_group_stats, _READ),
]
