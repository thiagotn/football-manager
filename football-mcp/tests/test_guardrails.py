import httpx
import pytest
import respx

from rachao_mcp.client import RachaoClient


@pytest.fixture
def client():
    return RachaoClient()


# ── Group allowlist ────────────────────────────────────────────────────────────

@pytest.mark.asyncio
async def test_group_allowlist_blocks_unauthorized(monkeypatch, client):
    allowed = "aaaaaaaa-0000-0000-0000-000000000000"
    blocked = "bbbbbbbb-0000-0000-0000-000000000000"
    monkeypatch.setenv("RACHAO_MCP_GROUP_ALLOWLIST", allowed)
    with pytest.raises(PermissionError, match="allowlist"):
        await client.get(f"/groups/{blocked}/matches")


@pytest.mark.asyncio
async def test_group_allowlist_allows_authorized(monkeypatch, client):
    allowed = "aaaaaaaa-0000-0000-0000-000000000000"
    monkeypatch.setenv("RACHAO_MCP_GROUP_ALLOWLIST", allowed)
    with respx.mock(base_url="https://api.rachao.app/api/v1") as mock:
        mock.get(f"/groups/{allowed}/matches").mock(
            return_value=httpx.Response(200, json=[])
        )
        result = await client.get(f"/groups/{allowed}/matches")
        assert result == []


@pytest.mark.asyncio
async def test_no_allowlist_allows_any_group(client):
    gid = "cccccccc-0000-0000-0000-000000000000"
    with respx.mock(base_url="https://api.rachao.app/api/v1") as mock:
        mock.get(f"/groups/{gid}/matches").mock(
            return_value=httpx.Response(200, json=[])
        )
        result = await client.get(f"/groups/{gid}/matches")
        assert result == []


@pytest.mark.asyncio
async def test_allowlist_does_not_affect_non_group_paths(monkeypatch, client):
    monkeypatch.setenv("RACHAO_MCP_GROUP_ALLOWLIST", "some-id")
    with respx.mock(base_url="https://api.rachao.app/api/v1") as mock:
        mock.get("/matches/discover").mock(return_value=httpx.Response(200, json=[]))
        result = await client.get("/matches/discover")
        assert result == []


# ── Read-only mode ─────────────────────────────────────────────────────────────

def _registered_tool_names(server) -> set[str]:
    return set(server._tool_manager._tools.keys())


def test_read_only_excludes_write_tools(monkeypatch):
    monkeypatch.setenv("RACHAO_MCP_READ_ONLY", "true")
    from rachao_mcp.server import create_server
    server = create_server()
    names = _registered_tool_names(server)
    assert "create_match" not in names
    assert "update_match" not in names
    assert "set_attendance" not in names
    assert "draw_teams" not in names


def test_read_only_keeps_read_tools(monkeypatch):
    monkeypatch.setenv("RACHAO_MCP_READ_ONLY", "true")
    from rachao_mcp.server import create_server
    server = create_server()
    names = _registered_tool_names(server)
    assert "list_groups" in names
    assert "list_matches" in names
    assert "get_match" in names
    assert "discover_matches" in names
    assert "get_ranking" in names


def test_write_mode_includes_all_tools(monkeypatch):
    monkeypatch.setenv("RACHAO_MCP_READ_ONLY", "false")
    from rachao_mcp.server import create_server
    server = create_server()
    names = _registered_tool_names(server)
    assert "create_match" in names
    assert "draw_teams" in names


# ── Allowed tools allowlist ────────────────────────────────────────────────────

def test_allowed_tools_filters_registered_tools(monkeypatch):
    monkeypatch.setenv("RACHAO_MCP_ALLOWED_TOOLS", "list_groups,list_matches")
    from rachao_mcp.server import create_server
    server = create_server()
    names = _registered_tool_names(server)
    assert names == {"list_groups", "list_matches"}


def test_no_allowed_tools_registers_all_read(monkeypatch):
    from rachao_mcp.server import create_server
    server = create_server()
    names = _registered_tool_names(server)
    expected_read = {
        "list_groups", "get_group", "get_group_stats",
        "list_matches", "get_match", "discover_matches",
        "list_players", "get_my_stats", "get_ranking",
        "get_teams",
        "create_match", "update_match", "set_attendance",
        "draw_teams",
    }
    assert expected_read == names
