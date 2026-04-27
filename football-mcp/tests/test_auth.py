import pytest


def test_get_token_returns_value(monkeypatch):
    monkeypatch.setenv("RACHAO_TOKEN", "my-token")
    from rachao_mcp.auth import get_token
    assert get_token() == "my-token"


def test_get_token_missing_raises(monkeypatch):
    monkeypatch.delenv("RACHAO_TOKEN", raising=False)
    from rachao_mcp.auth import get_token
    with pytest.raises(RuntimeError, match="RACHAO_TOKEN"):
        get_token()


def test_get_token_context_var_takes_precedence(monkeypatch):
    monkeypatch.setenv("RACHAO_TOKEN", "env-token")
    from rachao_mcp.auth import _request_token, get_token
    ctx = _request_token.set("request-token")
    try:
        assert get_token() == "request-token"
    finally:
        _request_token.reset(ctx)


def test_get_token_falls_back_to_env_when_context_empty(monkeypatch):
    monkeypatch.setenv("RACHAO_TOKEN", "env-token")
    from rachao_mcp.auth import _request_token, get_token
    assert _request_token.get() is None
    assert get_token() == "env-token"


def test_get_api_url_default(monkeypatch):
    monkeypatch.delenv("RACHAO_API_URL", raising=False)
    from rachao_mcp.auth import get_api_url
    assert get_api_url() == "https://api.rachao.app/api/v1"


def test_get_api_url_custom(monkeypatch):
    monkeypatch.setenv("RACHAO_API_URL", "https://staging.api.rachao.app/api/v1")
    from rachao_mcp.auth import get_api_url
    assert get_api_url() == "https://staging.api.rachao.app/api/v1"


def test_get_api_url_strips_trailing_slash(monkeypatch):
    monkeypatch.setenv("RACHAO_API_URL", "https://api.rachao.app/api/v1/")
    from rachao_mcp.auth import get_api_url
    assert not get_api_url().endswith("/")
