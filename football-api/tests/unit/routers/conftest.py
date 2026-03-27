"""
Fixtures HTTP para testes de routers.

Cria AsyncClient com dependências de banco e autenticação mockadas,
sem precisar de conexão real ao PostgreSQL.
"""
import pytest
from httpx import ASGITransport, AsyncClient

from app.core.dependencies import get_current_player, get_db, get_optional_player
from app.main import app


@pytest.fixture
async def api_client(mock_db, player_user):
    """Cliente autenticado como jogador comum (role=player)."""
    app.dependency_overrides[get_db] = lambda: mock_db
    app.dependency_overrides[get_current_player] = lambda: player_user
    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
        yield client
    app.dependency_overrides.clear()


@pytest.fixture
async def admin_client(mock_db, admin_user):
    """Cliente autenticado como super admin (role=admin)."""
    app.dependency_overrides[get_db] = lambda: mock_db
    app.dependency_overrides[get_current_player] = lambda: admin_user
    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
        yield client
    app.dependency_overrides.clear()


@pytest.fixture
async def anon_client(mock_db):
    """Cliente sem autenticação (visitante anônimo)."""
    app.dependency_overrides[get_db] = lambda: mock_db
    app.dependency_overrides[get_optional_player] = lambda: None
    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
        yield client
    app.dependency_overrides.clear()
