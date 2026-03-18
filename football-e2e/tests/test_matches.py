import re

import pytest
from playwright.sync_api import Page, expect

from pages.dashboard_page import DashboardPage
from pages.group_page import GroupPage
from pages.match_page import MatchPage


@pytest.fixture
def open_match_page(admin_page: Page):
    """Navega até a primeira partida aberta encontrada nos grupos."""
    dash = DashboardPage(admin_page)
    dash.goto()
    links = dash.group_links()
    if links.count() == 0:
        pytest.skip("Nenhum grupo disponível")
    links.first.click()
    admin_page.wait_for_load_state("networkidle")
    gp = GroupPage(admin_page)
    gp.tab_upcoming()
    detalhes = admin_page.get_by_role("link", name="Detalhes").first
    if not detalhes.is_visible():
        pytest.skip("Nenhuma partida aberta no ambiente de teste")
    detalhes.click()
    admin_page.wait_for_load_state("networkidle")
    return admin_page, MatchPage(admin_page)


def test_dashboard_aba_proximos_exibe_apenas_partidas_abertas(admin_page: Page):
    # Navigate to a group detail page which has the match tabs (admin is redirected away from "/")
    admin_page.goto("/groups")
    admin_page.wait_for_load_state("networkidle")
    links = admin_page.locator("a[href^='/groups/']")
    if links.count() == 0:
        pytest.skip("Nenhum grupo disponível")
    links.first.click()
    admin_page.wait_for_load_state("networkidle")
    admin_page.get_by_role("button", name=re.compile("Próximos")).click()
    badges = admin_page.locator(".badge-red, .badge-gray").all()
    assert len(badges) == 0, "Aba Próximos não deve conter partidas encerradas"


def test_dashboard_aba_ultimos_exibe_partidas_encerradas(admin_page: Page):
    # Navigate to a group detail page which has the match tabs (admin is redirected away from "/")
    admin_page.goto("/groups")
    admin_page.wait_for_load_state("networkidle")
    links = admin_page.locator("a[href^='/groups/']")
    if links.count() == 0:
        pytest.skip("Nenhum grupo disponível")
    links.first.click()
    admin_page.wait_for_load_state("networkidle")
    # "Últimos" tab is only rendered when pastMatches.length > 0
    btn = admin_page.get_by_role("button", name=re.compile("Últimos"))
    if not btn.is_visible():
        pytest.skip("Nenhuma partida encerrada no grupo de teste")
    btn.click()
    # If there are matches, they must be closed
    open_badges = admin_page.locator(".badge-green").all()
    for badge in open_badges:
        assert "Aberta" not in (badge.text_content() or "")


def test_pagina_da_partida_carrega(open_match_page):
    page, mp = open_match_page
    expect(page.locator("h1, h2").first).to_be_visible()
    assert "/match/" in page.url


def test_partida_aberta_exibe_status_correto(open_match_page):
    _, mp = open_match_page
    assert mp.is_open()


def test_partida_exibe_botao_compartilhar_whatsapp(open_match_page):
    page, mp = open_match_page
    expect(mp.share_whatsapp_button()).to_be_visible()


def test_partida_exibe_botao_copiar_link(open_match_page):
    page, mp = open_match_page
    expect(mp.copy_link_button()).to_be_visible()


def test_botao_detalhes_navega_para_pagina_da_partida(admin_page: Page):
    dash = DashboardPage(admin_page)
    dash.goto()
    links = dash.group_links()
    if links.count() == 0:
        pytest.skip("Nenhum grupo disponível")
    links.first.click()
    GroupPage(admin_page).tab_upcoming()
    detalhes = admin_page.get_by_role("link", name="Detalhes").first
    if not detalhes.is_visible():
        pytest.skip("Nenhuma partida aberta")
    detalhes.click()
    expect(admin_page).to_have_url("**/match/**")
