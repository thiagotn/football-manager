import httpx
import pytest
import respx

from rachao_mcp.client import RachaoClient


@pytest.fixture
def client():
    return RachaoClient()


@pytest.mark.asyncio
async def test_bearer_header_sent(client):
    with respx.mock(base_url="https://api.rachao.app/api/v1") as mock:
        mock.get("/groups").mock(return_value=httpx.Response(200, json=[]))
        await client.get("/groups")
        assert mock.calls[0].request.headers["authorization"] == "Bearer test-jwt-token"


@pytest.mark.asyncio
async def test_get_returns_parsed_json(client):
    with respx.mock(base_url="https://api.rachao.app/api/v1") as mock:
        mock.get("/groups").mock(return_value=httpx.Response(200, json=[{"id": "abc"}]))
        result = await client.get("/groups")
        assert result == [{"id": "abc"}]


@pytest.mark.asyncio
async def test_401_raises_permission_error(client):
    with respx.mock(base_url="https://api.rachao.app/api/v1") as mock:
        mock.get("/groups").mock(return_value=httpx.Response(401))
        with pytest.raises(PermissionError, match="RACHAO_TOKEN"):
            await client.get("/groups")


@pytest.mark.asyncio
async def test_404_raises_lookup_error(client):
    with respx.mock(base_url="https://api.rachao.app/api/v1") as mock:
        mock.get("/groups/bad-id").mock(return_value=httpx.Response(404))
        with pytest.raises(LookupError, match="não encontrado"):
            await client.get("/groups/bad-id")


@pytest.mark.asyncio
async def test_503_raises_runtime_error(client):
    with respx.mock(base_url="https://api.rachao.app/api/v1") as mock:
        mock.get("/groups").mock(return_value=httpx.Response(503))
        with pytest.raises(RuntimeError, match="indisponível"):
            await client.get("/groups")


@pytest.mark.asyncio
async def test_connect_error_raises_runtime_error(client):
    with respx.mock(base_url="https://api.rachao.app/api/v1") as mock:
        mock.get("/groups").mock(side_effect=httpx.ConnectError("timeout"))
        with pytest.raises(RuntimeError, match="indisponível"):
            await client.get("/groups")


@pytest.mark.asyncio
async def test_post_sends_json_body(client):
    with respx.mock(base_url="https://api.rachao.app/api/v1") as mock:
        mock.post("/groups/g1/matches").mock(
            return_value=httpx.Response(201, json={"id": "m1"})
        )
        result = await client.post("/groups/g1/matches", json={"location": "Campo"})
        import json
        body = json.loads(mock.calls[0].request.content)
        assert body["location"] == "Campo"
        assert result == {"id": "m1"}


@pytest.mark.asyncio
async def test_patch_sends_json_body(client):
    with respx.mock(base_url="https://api.rachao.app/api/v1") as mock:
        mock.patch("/groups/g1/matches/m1").mock(
            return_value=httpx.Response(200, json={"id": "m1"})
        )
        await client.patch("/groups/g1/matches/m1", json={"status": "closed"})
        import json
        body = json.loads(mock.calls[0].request.content)
        assert body["status"] == "closed"
