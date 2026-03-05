import os

import pytest
from dotenv import load_dotenv
from playwright.sync_api import Browser

load_dotenv()

BASE_URL = os.getenv("BASE_URL", "http://localhost:3000")
ADMIN_WHATSAPP = os.getenv("ADMIN_WHATSAPP", "11999990000")
ADMIN_PASSWORD = os.getenv("ADMIN_PASSWORD", "admin123")


def pytest_configure(config):
    """Garante que --base-url aponta para BASE_URL quando não informado via CLI."""
    if not config.getoption("--base-url", default=None):
        config.option.base_url = BASE_URL


@pytest.fixture(scope="session")
def admin_storage_state(browser: Browser, tmp_path_factory):
    """
    Realiza o login como admin uma única vez por sessão de testes e persiste
    o storage state (cookies + localStorage) para reutilização nos demais testes,
    evitando overhead de autenticação repetida.
    """
    state_path = str(tmp_path_factory.mktemp("auth") / "admin_state.json")
    ctx = browser.new_context(base_url=BASE_URL)
    page = ctx.new_page()
    page.goto("/login")
    page.locator("#whatsapp").fill(ADMIN_WHATSAPP)
    page.locator("#password").fill(ADMIN_PASSWORD)
    page.get_by_role("button", name="Entrar").click()
    page.wait_for_url("**/")
    ctx.storage_state(path=state_path)
    ctx.close()
    return state_path


@pytest.fixture
def admin_page(browser: Browser, admin_storage_state):
    """Abre uma nova página autenticada como admin para cada teste."""
    ctx = browser.new_context(base_url=BASE_URL, storage_state=admin_storage_state)
    page = ctx.new_page()
    yield page
    ctx.close()
