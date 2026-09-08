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
    try:
        # Os grupos são carregados client-side após a hidratação — aguardar
        # o primeiro link em vez de contar imediatamente (evita skip por timing)
        links.first.wait_for(state="visible", timeout=10_000)
    except Exception:
        pytest.skip("Nenhum grupo disponível no ambiente de teste")
    links.first.click()
    admin_page.wait_for_load_state("networkidle")
    return admin_page, GroupPage(admin_page)


def test_dashboard_exibe_grupos(admin_page: Page):
    dash = DashboardPage(admin_page)
    dash.goto()
    expect(dash.group_links().first).to_be_visible()


def test_grupo_exibe_abas_principais(first_group_page):
    import re

    page, gp = first_group_page
    # "Últimos" só aparece quando o grupo tem partidas passadas — não é garantida
    expect(page.get_by_role("button", name=re.compile(r"Atuais"))).to_be_visible()
    expect(page.get_by_role("button", name=re.compile(r"Jogadores"))).to_be_visible()
    expect(page.get_by_role("button", name=re.compile(r"Estatísticas"))).to_be_visible()


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


def test_modal_adicionar_membro_abre_com_busca_por_whatsapp(first_group_page):
    page, gp = first_group_page
    gp.tab_members()
    gp.add_member_button().click()
    expect(page.get_by_text("WhatsApp do jogador")).to_be_visible()


def test_aba_jogadores_ordena_por_mais_recentes(first_group_page):
    page, gp = first_group_page
    gp.tab_members()
    if gp.member_rows().count() == 0:
        pytest.skip("Grupo sem membros no ambiente de teste")
    expect(gp.sort_by_name_pill()).to_be_visible()
    rows_before = gp.member_rows().count()
    gp.sort_by_recent_pill().click()
    # Ordenar não altera a quantidade de linhas
    expect(gp.member_rows()).to_have_count(rows_before)
    gp.sort_by_name_pill().click()
    expect(gp.member_rows()).to_have_count(rows_before)


def test_aba_jogadores_inverte_ordenacao_por_nome(first_group_page):
    import re

    page, gp = first_group_page
    gp.tab_members()
    pill = gp.sort_by_name_pill()
    if pill.count() == 0:
        pytest.skip("Grupo sem membros no ambiente de teste")
    rows = gp.member_rows()
    count = rows.count()
    first_before = rows.first.inner_text()

    # Pill ativa: segundo clique inverte para Z–A
    pill.click()
    expect(page.get_by_role("button", name=re.compile(r"Nome Z–A"))).to_be_visible()
    expect(gp.member_rows()).to_have_count(count)
    if count > 1:
        first_after = gp.member_rows().first.inner_text()
        assert first_after != first_before, "primeira linha deve mudar ao inverter para Z–A"

    # Terceiro clique volta para A–Z e restaura a primeira linha
    pill.click()
    expect(page.get_by_role("button", name=re.compile(r"Nome A–Z"))).to_be_visible()
    assert gp.member_rows().first.inner_text() == first_before


def test_aba_proximos_exibe_botao_novo_rachao(first_group_page):
    page, gp = first_group_page
    gp.tab_upcoming()
    expect(gp.new_match_button()).to_be_visible()


def test_aba_ultimos_nao_exibe_botao_novo_rachao(first_group_page):
    import re

    page, gp = first_group_page
    if page.get_by_role("button", name=re.compile(r"Últimos")).count() == 0:
        pytest.skip("Grupo sem partidas passadas — aba Últimos não é exibida")
    gp.tab_past()
    expect(gp.new_match_button()).not_to_be_visible()


def test_modal_detalhes_jogador_escudo(first_group_page):
    """Modal 'Detalhes do Jogador' no formato card-escudo: abre, mostra o escudo
    e as ações, fecha com Esc e devolve o foco ao botão de origem."""
    page, gp = first_group_page
    gp.tab_members()
    btn = gp.member_details_button()
    if btn.count() == 0:
        pytest.skip("Grupo sem membros no ambiente de teste")

    btn.click()
    modal = gp.member_detail_modal()
    expect(modal).to_be_visible()
    expect(modal.get_by_role("heading", name="Detalhes do Jogador")).to_be_visible()
    expect(gp.player_crest()).to_be_visible()
    expect(modal.get_by_role("button", name="Editar habilidade")).to_be_visible()
    # Tab fica preso dentro do modal
    page.keyboard.press("Tab")
    assert page.evaluate("document.activeElement.closest('[role=dialog]') !== null")

    page.keyboard.press("Escape")
    expect(modal).to_have_count(0)
    expect(btn).to_be_focused()
