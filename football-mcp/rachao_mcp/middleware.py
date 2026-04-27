from typing import Callable

from rachao_mcp.auth import _request_token


class BearerTokenMiddleware:
    """Extracts the Bearer token from each HTTP/SSE request and sets it in a ContextVar.

    This enables multi-tenant HTTP deployments where each client authenticates
    via Authorization: Bearer <token> instead of a shared RACHAO_TOKEN env var.
    """

    def __init__(self, app: Callable) -> None:
        self.app = app

    async def __call__(self, scope: dict, receive: Callable, send: Callable) -> None:
        if scope["type"] in ("http", "websocket"):
            headers = {k: v for k, v in scope.get("headers", [])}
            auth = headers.get(b"authorization", b"").decode()
            if auth.startswith("Bearer "):
                ctx_token = _request_token.set(auth[7:])
                try:
                    await self.app(scope, receive, send)
                finally:
                    _request_token.reset(ctx_token)
                return
        await self.app(scope, receive, send)
