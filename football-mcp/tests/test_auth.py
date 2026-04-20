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
