from rachao_mcp.client import api

_READ = {"readOnlyHint": True, "idempotentHint": True}
_WRITE = {"readOnlyHint": False, "destructiveHint": False, "idempotentHint": False}
_WRITE_IDEM = {"readOnlyHint": False, "destructiveHint": False, "idempotentHint": True}


async def list_matches(group_id: str) -> list[dict]:
    """Lista as partidas de um grupo (abertas e encerradas)."""
    return await api.get(f"/groups/{group_id}/matches")


async def get_match(match_hash: str) -> dict:
    """Detalhe de uma partida: presença, stats e times. Aceita hash público."""
    return await api.get(f"/matches/public/{match_hash}")


async def discover_matches() -> list[dict]:
    """Partidas públicas abertas em todos os grupos da plataforma."""
    return await api.get("/matches/discover")


async def create_match(
    group_id: str,
    match_date: str,
    start_time: str,
    location: str,
    notes: str | None = None,
) -> dict:
    """Cria uma nova partida no grupo. match_date: YYYY-MM-DD, start_time: HH:MM."""
    return await api.post(f"/groups/{group_id}/matches", json={
        "match_date": match_date,
        "start_time": start_time,
        "location": location,
        "notes": notes,
    })


async def update_match(
    group_id: str,
    match_id: str,
    match_date: str | None = None,
    start_time: str | None = None,
    location: str | None = None,
    notes: str | None = None,
    status: str | None = None,
) -> dict:
    """Atualiza dados de uma partida. Envie apenas os campos a alterar."""
    body = {k: v for k, v in {
        "match_date": match_date,
        "start_time": start_time,
        "location": location,
        "notes": notes,
        "status": status,
    }.items() if v is not None}
    return await api.patch(f"/groups/{group_id}/matches/{match_id}", json=body)


async def set_attendance(
    group_id: str,
    match_id: str,
    player_id: str,
    status: str,
) -> dict:
    """Confirma ou recusa presença de um jogador. status: confirmed | declined | pending."""
    return await api.post(
        f"/groups/{group_id}/matches/{match_id}/attendance",
        json={"player_id": player_id, "status": status},
    )


READ_TOOLS: list[tuple] = [
    (list_matches, _READ),
    (get_match, _READ),
    (discover_matches, _READ),
]

WRITE_TOOLS: list[tuple] = [
    (create_match, _WRITE),
    (update_match, _WRITE),
    (set_attendance, _WRITE_IDEM),
]
