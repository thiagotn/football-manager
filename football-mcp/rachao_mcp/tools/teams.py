from rachao_mcp.client import api

_READ = {"readOnlyHint": True, "idempotentHint": True}
_WRITE = {"readOnlyHint": False, "destructiveHint": False, "idempotentHint": False}


async def get_teams(group_id: str, match_id: str) -> list[dict]:
    """Times já sorteados para uma partida."""
    return await api.get(f"/groups/{group_id}/matches/{match_id}/teams")


async def draw_teams(group_id: str, match_id: str) -> list[dict]:
    """Sorteia times equilibrados para uma partida. Substitui sorteio anterior se existir."""
    return await api.post(f"/groups/{group_id}/matches/{match_id}/teams/draw", json={})


READ_TOOLS: list[tuple] = [
    (get_teams, _READ),
]

WRITE_TOOLS: list[tuple] = [
    (draw_teams, _WRITE),
]
