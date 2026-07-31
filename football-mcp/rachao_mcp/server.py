import os

from mcp.server.mcpserver import MCPServer
from mcp.server.transport_security import TransportSecuritySettings
from mcp_types import ToolAnnotations

from rachao_mcp.auth import get_token
from rachao_mcp.tools import groups, matches, players, teams


def _transport_security() -> TransportSecuritySettings | None:
    allowed_hosts = os.getenv("MCP_ALLOWED_HOSTS", "").split(",")
    allowed_hosts = [h.strip() for h in allowed_hosts if h.strip()]
    return TransportSecuritySettings(allowed_hosts=allowed_hosts) if allowed_hosts else None


def _build_mcp_server() -> MCPServer:
    read_only = os.getenv("RACHAO_MCP_READ_ONLY", "false").lower() == "true"
    _allowed_raw = os.getenv("RACHAO_MCP_ALLOWED_TOOLS", "")
    allowed_tools: set[str] | None = set(_allowed_raw.split(",")) if _allowed_raw else None

    server = MCPServer("rachao.app", version="0.1.0")

    def register(tool_list: list[tuple]) -> None:
        for fn, annotations in tool_list:
            if allowed_tools is None or fn.__name__ in allowed_tools:
                server.tool(annotations=ToolAnnotations(**annotations))(fn)

    register(groups.READ_TOOLS)
    register(matches.READ_TOOLS)
    register(players.READ_TOOLS)
    register(teams.READ_TOOLS)

    if not read_only:
        register(matches.WRITE_TOOLS)
        register(teams.WRITE_TOOLS)

    return server


def create_server() -> MCPServer:
    get_token()  # fail fast — RuntimeError se RACHAO_TOKEN não estiver definido
    return _build_mcp_server()


def main() -> None:
    transport = os.getenv("MCP_TRANSPORT", "stdio")

    if transport == "sse":
        raise RuntimeError(
            "O transporte SSE foi removido (deprecated na spec MCP 2026-07-28). "
            "Use MCP_TRANSPORT=http (streamable-http)."
        )

    if transport == "http":
        import uvicorn

        from rachao_mcp.middleware import BearerTokenMiddleware

        # In HTTP mode tokens arrive per-request — no RACHAO_TOKEN env var required.
        host = os.getenv("MCP_HOST", "127.0.0.1")
        port = int(os.getenv("MCP_PORT", "8080"))

        mcp = _build_mcp_server()
        raw_app = mcp.streamable_http_app(
            stateless_http=True,
            json_response=True,
            transport_security=_transport_security(),
            host=host,
        )
        app = BearerTokenMiddleware(raw_app)
        uvicorn.run(app, host=host, port=port)
    else:
        mcp = create_server()
        mcp.run()
