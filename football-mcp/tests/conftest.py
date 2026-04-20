import pytest
import respx


@pytest.fixture(autouse=True)
def set_token(monkeypatch):
    monkeypatch.setenv("RACHAO_TOKEN", "test-jwt-token")
    monkeypatch.setenv("RACHAO_API_URL", "https://api.rachao.app/api/v1")
    monkeypatch.delenv("RACHAO_MCP_GROUP_ALLOWLIST", raising=False)
    monkeypatch.delenv("RACHAO_MCP_ALLOWED_TOOLS", raising=False)
    monkeypatch.delenv("RACHAO_MCP_READ_ONLY", raising=False)


@pytest.fixture
def mock_api():
    with respx.mock(base_url="https://api.rachao.app/api/v1", assert_all_called=False) as mock:
        yield mock
