from rachao_mcp.client import api

_READ = {"readOnlyHint": True, "idempotentHint": True}
_WRITE = {"readOnlyHint": False, "destructiveHint": False, "idempotentHint": False}


async def get_teams(match_id: str) -> list[dict]:
    """Times já sorteados para uma partida."""
    return await api.get(f"/matches/{match_id}/teams")


async def draw_teams(match_id: str, strategy: str = "balanced") -> list[dict]:
    """Sorteia times para uma partida. Substitui sorteio anterior se existir.

    strategy: 'balanced' (default — equilibra posições e estrelas) ou
    'simple' (equilibra apenas estrelas, sem cota por posição; goleiros
    continuam 1 por time).
    """
    return await api.post(f"/matches/{match_id}/teams", json={"strategy": strategy})


READ_TOOLS: list[tuple] = [
    (get_teams, _READ),
]

WRITE_TOOLS: list[tuple] = [
    (draw_teams, _WRITE),
]
