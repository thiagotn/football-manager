import httpx
import pytest

from rachao_mcp.tools.teams import draw_teams, get_teams


@pytest.mark.asyncio
async def test_get_teams(mock_api):
    mock_api.get("/groups/g1/matches/m1/teams").mock(
        return_value=httpx.Response(200, json=[{"name": "Time A"}, {"name": "Time B"}])
    )
    result = await get_teams("g1", "m1")
    assert len(result) == 2
    assert result[0]["name"] == "Time A"


@pytest.mark.asyncio
async def test_get_teams_not_found(mock_api):
    mock_api.get("/groups/g1/matches/m1/teams").mock(
        return_value=httpx.Response(404)
    )
    with pytest.raises(LookupError):
        await get_teams("g1", "m1")


@pytest.mark.asyncio
async def test_draw_teams(mock_api):
    mock_api.post("/groups/g1/matches/m1/teams/draw").mock(
        return_value=httpx.Response(200, json=[{"name": "Time A"}, {"name": "Time B"}])
    )
    result = await draw_teams("g1", "m1")
    assert len(result) == 2


@pytest.mark.asyncio
async def test_draw_teams_posts_to_correct_endpoint(mock_api):
    mock_api.post("/groups/g1/matches/m1/teams/draw").mock(
        return_value=httpx.Response(200, json=[])
    )
    await draw_teams("g1", "m1")
    assert mock_api.calls[0].request.method == "POST"
