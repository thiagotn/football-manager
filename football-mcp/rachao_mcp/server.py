import os
from typing import Any

from mcp.server.fastmcp import FastMCP

from rachao_mcp.auth import get_token
from rachao_mcp.tools import groups, matches, players, teams


def create_server() -> FastMCP:
    get_token()  # fail fast — RuntimeError se RACHAO_TOKEN não estiver definido

    read_only = os.getenv("RACHAO_MCP_READ_ONLY", "false").lower() == "true"
    _allowed_raw = os.getenv("RACHAO_MCP_ALLOWED_TOOLS", "")
    allowed_tools: set[str] | None = set(_allowed_raw.split(",")) if _allowed_raw else None

    server = FastMCP("rachao.app")

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


class _BearerAuthMiddleware:
    """ASGI middleware: rejeita requisições sem o bearer token correto."""

    def __init__(self, app: Any, secret_key: str) -> None:
        self.app = app
        self.secret_key = secret_key

    async def __call__(self, scope: Any, receive: Any, send: Any) -> None:
        if scope["type"] in ("http", "websocket"):
            headers = {k.lower(): v for k, v in scope.get("headers", [])}
            auth = headers.get(b"authorization", b"").decode()
            if not auth.startswith("Bearer "):
                token = ""
            else:
                token = auth[len("Bearer "):].strip()
            if token != self.secret_key:
                await send({
                    "type": "http.response.start",
                    "status": 401,
                    "headers": [
                        (b"content-type", b"text/plain"),
                        (b"www-authenticate", b"Bearer"),
                    ],
                })
                await send({"type": "http.response.body", "body": b"Unauthorized"})
                return
        await self.app(scope, receive, send)


def main() -> None:
    transport = os.getenv("MCP_TRANSPORT", "stdio")

    if transport in ("sse", "http"):
        import uvicorn

        mcp = create_server()
        app = mcp.streamable_http_app() if transport == "http" else mcp.sse_app()

        secret_key = os.getenv("MCP_SECRET_KEY")
        if secret_key:
            app = _BearerAuthMiddleware(app, secret_key)  # type: ignore[assignment]

        host = os.getenv("MCP_HOST", "127.0.0.1")
        port = int(os.getenv("MCP_PORT", "8080"))
        uvicorn.run(app, host=host, port=port)
    else:
        mcp = create_server()
        mcp.run()
