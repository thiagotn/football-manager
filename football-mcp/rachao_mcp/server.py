import os

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


mcp = create_server()


def main() -> None:
    mcp.run()


if __name__ == "__main__":
    main()
