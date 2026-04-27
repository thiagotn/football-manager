import os
from contextvars import ContextVar

# Stores the per-request Bearer token injected by BearerTokenMiddleware in HTTP mode.
# Falls back to RACHAO_TOKEN env var (stdio / single-tenant deployments).
_request_token: ContextVar[str | None] = ContextVar("request_token", default=None)


def get_token() -> str:
    token = _request_token.get()
    if token:
        return token
    token = os.getenv("RACHAO_TOKEN")
    if not token:
        raise RuntimeError(
            "RACHAO_TOKEN não definido — configure a variável de ambiente antes de iniciar o MCP"
        )
    return token


def get_api_url() -> str:
    return os.getenv("RACHAO_API_URL", "https://api.rachao.app/api/v1").rstrip("/")
