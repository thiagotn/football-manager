"""Tests for HTTP/SSE transport and bearer token middleware."""
import pytest
from starlette.testclient import TestClient
from starlette.applications import Starlette
from starlette.responses import PlainTextResponse
from starlette.routing import Route

from rachao_mcp.server import _BearerAuthMiddleware


def _make_app(secret_key: str):
    async def ok(request):
        return PlainTextResponse("ok")

    app = Starlette(routes=[Route("/sse", ok)])
    return _BearerAuthMiddleware(app, secret_key)


def test_missing_auth_header_returns_401():
    app = _make_app("mysecret")
    client = TestClient(app, raise_server_exceptions=False)
    r = client.get("/sse")
    assert r.status_code == 401
    assert r.headers["www-authenticate"] == "Bearer"


def test_wrong_token_returns_401():
    app = _make_app("mysecret")
    client = TestClient(app, raise_server_exceptions=False)
    r = client.get("/sse", headers={"Authorization": "Bearer wrongtoken"})
    assert r.status_code == 401


def test_correct_token_passes_through():
    app = _make_app("mysecret")
    client = TestClient(app, raise_server_exceptions=False)
    r = client.get("/sse", headers={"Authorization": "Bearer mysecret"})
    assert r.status_code == 200
    assert r.text == "ok"


def test_bearer_prefix_is_required():
    app = _make_app("mysecret")
    client = TestClient(app, raise_server_exceptions=False)
    r = client.get("/sse", headers={"Authorization": "mysecret"})
    assert r.status_code == 401
