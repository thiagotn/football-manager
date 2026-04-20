import httpx
import pytest

from rachao_mcp.tools.players import get_my_stats, get_ranking, list_players


@pytest.mark.asyncio
async def test_list_players(mock_api):
    mock_api.get("/groups/g1/members").mock(
        return_value=httpx.Response(200, json=[{"id": "p1", "name": "João"}])
    )
    result = await list_players("g1")
    assert result[0]["name"] == "João"


@pytest.mark.asyncio
async def test_get_my_stats(mock_api):
    mock_api.get("/players/me/stats/full").mock(
        return_value=httpx.Response(200, json={"goals": 5, "assists": 3})
    )
    result = await get_my_stats()
    assert result["goals"] == 5


@pytest.mark.asyncio
async def test_get_ranking(mock_api):
    mock_api.get("/ranking").mock(
        return_value=httpx.Response(200, json=[{"player_name": "João", "goals": 10}])
    )
    result = await get_ranking()
    assert result[0]["player_name"] == "João"
