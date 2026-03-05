import pytest
from playwright.sync_api import Page, expect

from pages.dashboard_page import DashboardPage
from pages.group_page import GroupPage
from pages.match_page import MatchPage


@pytest.fixture
def match_url(admin_page: Page) -> str:
    """Retorna a URL de uma partida aberta, ou pula o teste se não houver."""
    dash = DashboardPage(admin_page)
    dash.goto()
    links = dash.group_links()
    if links.count() == 0:
        pytest.skip("Nenhum grupo disponível")
    links.first.click()
    admin_page.wait_for_load_state("networkidle")
    GroupPage(admin_page).tab_upcoming()
    detalhes = admin_page.get_by_role("link", name="Detalhes").first
    if not detalhes.is_visible():
        pytest.skip("Nenhuma partida aberta no ambiente de teste")
    href = detalhes.get_attribute("href")
    return href or ""


def test_pagina_publica_da_partida_acessivel_sem_login(page: Page, match_url: str):
    """A página de detalhes da partida deve ser acessível sem autenticação."""
    page.goto(match_url)
    expect(page.locator("h1, h2").first).to_be_visible()


def test_pagina_da_partida_exibe_contagem_de_confirmados(page: Page, match_url: str):
    page.goto(match_url)
    expect(page.locator("text=/Confirmados/i")).to_be_visible()


def test_pagina_da_partida_exibe_botao_whatsapp(page: Page, match_url: str):
    page.goto(match_url)
    expect(page.get_by_role("link", name="Compartilhar no WhatsApp")).to_be_visible()


def test_pagina_da_partida_exibe_botao_copiar_link(page: Page, match_url: str):
    page.goto(match_url)
    expect(page.get_by_role("button", name="Copiar link")).to_be_visible()
