import pytest

from rachao_mcp.auth import _request_token
from rachao_mcp.middleware import BearerTokenMiddleware


async def _capture_token_app(scope, receive, send):
    """ASGI app that captures the ContextVar value during the request."""
    scope["captured_token"] = _request_token.get()
    await send({"type": "http.response.start", "status": 200, "headers": []})
    await send({"type": "http.response.body", "body": b""})


def _make_scope(auth_header: str | None = None) -> dict:
    headers = []
    if auth_header:
        headers.append((b"authorization", auth_header.encode()))
    return {"type": "http", "headers": headers}


async def _noop_receive():
    return {}


async def _noop_send(message):
    pass


@pytest.mark.asyncio
async def test_bearer_token_set_in_context():
    middleware = BearerTokenMiddleware(_capture_token_app)
    scope = _make_scope("Bearer my-test-token")
    await middleware(scope, _noop_receive, _noop_send)
    assert scope["captured_token"] == "my-test-token"


@pytest.mark.asyncio
async def test_no_auth_header_leaves_context_empty():
    middleware = BearerTokenMiddleware(_capture_token_app)
    scope = _make_scope()
    await middleware(scope, _noop_receive, _noop_send)
    assert scope.get("captured_token") is None


@pytest.mark.asyncio
async def test_non_bearer_scheme_leaves_context_empty():
    middleware = BearerTokenMiddleware(_capture_token_app)
    scope = _make_scope("Basic dXNlcjpwYXNz")
    await middleware(scope, _noop_receive, _noop_send)
    assert scope.get("captured_token") is None


@pytest.mark.asyncio
async def test_context_var_reset_after_request():
    middleware = BearerTokenMiddleware(_capture_token_app)
    scope = _make_scope("Bearer ephemeral-token")
    await middleware(scope, _noop_receive, _noop_send)
    # After the request completes the ContextVar must be cleared
    assert _request_token.get() is None


@pytest.mark.asyncio
async def test_non_http_scope_passes_through():
    called = []

    async def marker_app(scope, receive, send):
        called.append(True)

    middleware = BearerTokenMiddleware(marker_app)
    scope = {"type": "lifespan"}
    await middleware(scope, _noop_receive, _noop_send)
    assert called
