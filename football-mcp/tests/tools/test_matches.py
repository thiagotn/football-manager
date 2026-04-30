import json

import httpx
import pytest
import respx

from rachao_mcp.tools.matches import (
    create_match,
    discover_matches,
    get_match,
    list_matches,
    list_my_matches,
    set_attendance,
    update_match,
)


@pytest.mark.asyncio
async def test_list_my_matches_aggregates_all_groups(mock_api):
    mock_api.get("/groups").mock(
        return_value=httpx.Response(200, json=[
            {"id": "g1", "name": "Futebol GQC"},
            {"id": "g2", "name": "Alliance FC"},
        ])
    )
    mock_api.get("/groups/g1/matches").mock(
        return_value=httpx.Response(200, json=[{"id": "m1", "match_date": "2026-05-01"}])
    )
    mock_api.get("/groups/g2/matches").mock(
        return_value=httpx.Response(200, json=[{"id": "m2", "match_date": "2026-05-07"}])
    )
    result = await list_my_matches()
    assert len(result) == 2
    group_names = {m["group_name"] for m in result}
    assert group_names == {"Futebol GQC", "Alliance FC"}
    assert all("group_id" in m for m in result)


@pytest.mark.asyncio
async def test_list_my_matches_empty_when_no_groups(mock_api):
    mock_api.get("/groups").mock(return_value=httpx.Response(200, json=[]))
    result = await list_my_matches()
    assert result == []


@pytest.mark.asyncio
async def test_list_matches(mock_api):
    mock_api.get("/groups/g1/matches").mock(
        return_value=httpx.Response(200, json=[{"id": "m1"}])
    )
    result = await list_matches("g1")
    assert result[0]["id"] == "m1"


@pytest.mark.asyncio
async def test_get_match(mock_api):
    mock_api.get("/matches/public/abc123").mock(
        return_value=httpx.Response(200, json={"hash": "abc123"})
    )
    result = await get_match("abc123")
    assert result["hash"] == "abc123"


@pytest.mark.asyncio
async def test_discover_matches(mock_api):
    mock_api.get("/matches/discover").mock(
        return_value=httpx.Response(200, json=[{"id": "m1", "group_name": "Pelada"}])
    )
    result = await discover_matches()
    assert result[0]["group_name"] == "Pelada"


@pytest.mark.asyncio
async def test_create_match_posts_correct_body(mock_api):
    mock_api.post("/groups/g1/matches").mock(
        return_value=httpx.Response(201, json={"id": "m1"})
    )
    result = await create_match("g1", "2026-05-10", "20:00", "Campo do Zé")
    body = json.loads(mock_api.calls[0].request.content)
    assert body["match_date"] == "2026-05-10"
    assert body["start_time"] == "20:00"
    assert body["location"] == "Campo do Zé"
    assert result["id"] == "m1"


@pytest.mark.asyncio
async def test_create_match_with_notes(mock_api):
    mock_api.post("/groups/g1/matches").mock(
        return_value=httpx.Response(201, json={"id": "m1"})
    )
    await create_match("g1", "2026-05-10", "20:00", "Campo", notes="Levar colete")
    body = json.loads(mock_api.calls[0].request.content)
    assert body["notes"] == "Levar colete"


@pytest.mark.asyncio
async def test_update_match_sends_only_provided_fields(mock_api):
    mock_api.patch("/groups/g1/matches/m1").mock(
        return_value=httpx.Response(200, json={"id": "m1"})
    )
    await update_match("g1", "m1", location="Novo Campo")
    body = json.loads(mock_api.calls[0].request.content)
    assert body == {"location": "Novo Campo"}
    assert "match_date" not in body
    assert "status" not in body


@pytest.mark.asyncio
async def test_set_attendance_posts_correct_body(mock_api):
    mock_api.post("/groups/g1/matches/m1/attendance").mock(
        return_value=httpx.Response(200, json={"status": "confirmed"})
    )
    result = await set_attendance("g1", "m1", "player-uuid", "confirmed")
    body = json.loads(mock_api.calls[0].request.content)
    assert body["player_id"] == "player-uuid"
    assert body["status"] == "confirmed"
    assert result["status"] == "confirmed"
