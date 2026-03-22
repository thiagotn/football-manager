"""
Fixtures globais — aplicadas a todos os testes.

- patch_startup: impede que run_migrations tente conectar ao banco real
- player_user / admin_user: objetos Player falsos reutilizáveis
- mock_db: sessão SQLAlchemy mockada (AsyncMock)
"""
from unittest.mock import AsyncMock, MagicMock, patch
from uuid import uuid4

import pytest

from app.models.player import Player, PlayerRole


# ── Patch de startup ─────────────────────────────────────────────────────────


@pytest.fixture(autouse=True, scope="session")
def patch_startup():
    """Evita conexão real ao banco durante os testes (lifespan chama run_migrations)."""
    with patch("app.main.run_migrations", new_callable=AsyncMock):
        yield


# ── Players fake ─────────────────────────────────────────────────────────────


@pytest.fixture
def player_user() -> Player:
    p = MagicMock(spec=Player)
    p.id = uuid4()
    p.name = "João Silva"
    p.whatsapp = "+5511999990001"
    p.role = PlayerRole.PLAYER
    p.active = True
    p.must_change_password = False
    return p


@pytest.fixture
def admin_user() -> Player:
    p = MagicMock(spec=Player)
    p.id = uuid4()
    p.name = "Super Admin"
    p.whatsapp = "+5511999990000"
    p.role = PlayerRole.ADMIN
    p.active = True
    p.must_change_password = False
    return p


# ── Sessão de banco mockada ───────────────────────────────────────────────────


@pytest.fixture
def mock_db() -> AsyncMock:
    db = AsyncMock()
    db.flush = AsyncMock()
    db.refresh = AsyncMock()
    db.commit = AsyncMock()
    db.rollback = AsyncMock()
    return db
