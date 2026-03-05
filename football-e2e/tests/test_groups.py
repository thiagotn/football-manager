import pytest
from playwright.sync_api import Page, expect

from pages.dashboard_page import DashboardPage
from pages.group_page import GroupPage


@pytest.fixture
def first_group_page(admin_page: Page):
    """Navega para o primeiro grupo disponível e retorna (admin_page, GroupPage)."""
    dash = DashboardPage(admin_page)
    dash.goto()
    links = dash.group_links()
    if links.count() == 0:
        pytest.skip("Nenhum grupo disponível no ambiente de teste")
    links.first.click()
    admin_page.wait_for_load_state("networkidle")
    return admin_page, GroupPage(admin_page)


def test_dashboard_exibe_grupos(admin_page: Page):
    dash = DashboardPage(admin_page)
    dash.goto()
    expect(dash.group_links().first).to_be_visible()


def test_grupo_exibe_tres_abas(first_group_page):
    page, gp = first_group_page
    expect(page.get_by_role("button", name=lambda n: "Próximos" in n)).to_be_visible()
    expect(page.get_by_role("button", name=lambda n: "Últimos" in n)).to_be_visible()
    expect(page.get_by_role("button", name=lambda n: "Jogadores" in n)).to_be_visible()


def test_aba_jogadores_exibe_convidar_e_adicionar(first_group_page):
    page, gp = first_group_page
    gp.tab_members()
    expect(gp.invite_button()).to_be_visible()
    expect(gp.add_member_button()).to_be_visible()


def test_modal_convite_abre_com_link(first_group_page):
    page, gp = first_group_page
    gp.tab_members()
    gp.invite_button().click()
    invite_input = page.locator("input[readonly]")
    expect(invite_input).to_be_visible()
    assert "invite" in (invite_input.input_value() or "")


def test_modal_adicionar_membro_abre_com_descricao(first_group_page):
    page, gp = first_group_page
    gp.tab_members()
    gp.add_member_button().click()
    expect(page.get_by_text("Selecione um jogador cadastrado")).to_be_visible()


def test_aba_proximos_exibe_botao_novo_rachao(first_group_page):
    page, gp = first_group_page
    gp.tab_upcoming()
    expect(gp.new_match_button()).to_be_visible()


def test_aba_ultimos_nao_exibe_botao_novo_rachao(first_group_page):
    page, gp = first_group_page
    gp.tab_past()
    expect(gp.new_match_button()).not_to_be_visible()
