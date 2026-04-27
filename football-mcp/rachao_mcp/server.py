import os

from mcp.server.fastmcp import FastMCP
from mcp.server.transport_security import TransportSecuritySettings

from rachao_mcp.auth import get_token
from rachao_mcp.tools import groups, matches, players, teams


def _build_mcp_server() -> FastMCP:
    read_only = os.getenv("RACHAO_MCP_READ_ONLY", "false").lower() == "true"
    _allowed_raw = os.getenv("RACHAO_MCP_ALLOWED_TOOLS", "")
    allowed_tools: set[str] | None = set(_allowed_raw.split(",")) if _allowed_raw else None

    allowed_hosts = os.getenv("MCP_ALLOWED_HOSTS", "").split(",")
    allowed_hosts = [h.strip() for h in allowed_hosts if h.strip()]
    transport_security = TransportSecuritySettings(allowed_hosts=allowed_hosts) if allowed_hosts else None

    server = FastMCP("rachao.app", transport_security=transport_security)

    def register(tool_list: list[tuple]) -> None:
        for fn, annotations in tool_list:
            if allowed_tools is None or fn.__name__ in allowed_tools:
                server.tool()(fn)

    register(groups.READ_TOOLS)
    register(matches.READ_TOOLS)
    register(players.READ_TOOLS)
    register(teams.READ_TOOLS)

    if not read_only:
        register(matches.WRITE_TOOLS)
        register(teams.WRITE_TOOLS)

    return server


def create_server() -> FastMCP:
    get_token()  # fail fast — RuntimeError se RACHAO_TOKEN não estiver definido
    return _build_mcp_server()


def main() -> None:
    transport = os.getenv("MCP_TRANSPORT", "stdio")

    if transport in ("sse", "http"):
        import uvicorn

        from rachao_mcp.middleware import BearerTokenMiddleware

        # In HTTP/SSE mode tokens arrive per-request — no RACHAO_TOKEN env var required.
        mcp = _build_mcp_server()
        raw_app = mcp.streamable_http_app() if transport == "http" else mcp.sse_app()
        app = BearerTokenMiddleware(raw_app)

        host = os.getenv("MCP_HOST", "127.0.0.1")
        port = int(os.getenv("MCP_PORT", "8080"))
        uvicorn.run(app, host=host, port=port)
    else:
        mcp = create_server()
        mcp.run()
