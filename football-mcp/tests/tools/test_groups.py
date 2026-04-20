import httpx
import pytest
import respx

from rachao_mcp.tools.groups import get_group, get_group_stats, list_groups


@pytest.mark.asyncio
async def test_list_groups(mock_api):
    mock_api.get("/groups").mock(
        return_value=httpx.Response(200, json=[{"id": "g1", "name": "Pelada"}])
    )
    result = await list_groups()
    assert result[0]["name"] == "Pelada"


@pytest.mark.asyncio
async def test_get_group(mock_api):
    mock_api.get("/groups/g1").mock(
        return_value=httpx.Response(200, json={"id": "g1", "name": "Pelada"})
    )
    result = await get_group("g1")
    assert result["id"] == "g1"


@pytest.mark.asyncio
async def test_get_group_stats(mock_api):
    mock_api.get("/groups/g1/stats").mock(
        return_value=httpx.Response(200, json={"top_scorers": []})
    )
    result = await get_group_stats("g1")
    assert "top_scorers" in result


@pytest.mark.asyncio
async def test_get_group_not_found(mock_api):
    mock_api.get("/groups/bad").mock(return_value=httpx.Response(404))
    with pytest.raises(LookupError):
        await get_group("bad")
