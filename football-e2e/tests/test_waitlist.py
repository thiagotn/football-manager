"""
E2E tests for the public groups & waitlist feature.

These tests verify:
- Public group badge visibility
- Waitlist button not shown for group members
- CTA shown on match page for non-logged users on public groups
- Waitlist panel (admin only) visibility
- Group visibility toggle in edit modal
"""
import re

import pytest
from playwright.sync_api import Browser, Page, expect

from pages.dashboard_page import DashboardPage
from pages.group_page import GroupPage


# ── Fixtures ──────────────────────────────────────────────────────────────────

@pytest.fixture
def first_group_url(admin_page: Page) -> str:
    """Return URL of the first available group."""
    admin_page.goto("/groups")
    admin_page.wait_for_load_state("networkidle")
    links = admin_page.locator("a[href^='/groups/']")
    if links.count() == 0:
        pytest.skip("Nenhum grupo disponível")
    href = links.first.get_attribute("href") or ""
    return href


@pytest.fixture
def open_match_url(admin_page: Page) -> str:
    """Return URL of the first open match found across groups."""
    admin_page.goto("/groups")
    admin_page.wait_for_load_state("networkidle")
    links = admin_page.locator("a[href^='/groups/']")
    if links.count() == 0:
        pytest.skip("Nenhum grupo disponível")
    links.first.click()
    admin_page.wait_for_load_state("networkidle")
    GroupPage(admin_page).tab_upcoming()
    detalhes = admin_page.get_by_role("link", name="Detalhes").first
    if not detalhes.is_visible():
        pytest.skip("Nenhuma partida aberta no ambiente de teste")
    href = detalhes.get_attribute("href") or ""
    return href


# ── Group visibility badge ────────────────────────────────────────────────────

def test_grupo_exibe_badge_de_visibilidade(admin_page: Page, first_group_url: str):
    """Group detail page should show a visibility badge (Público or Fechado)."""
    admin_page.goto(first_group_url)
    admin_page.wait_for_load_state("networkidle")
    # Either "Público" or "Fechado" badge should be visible
    badge = admin_page.locator("span", has_text=re.compile(r"Público|Fechado"))
    expect(badge.first).to_be_visible()


# ── Edit group modal includes is_public toggle ────────────────────────────────

def test_modal_editar_grupo_exibe_toggle_publico(admin_page: Page, first_group_url: str):
    """Edit group modal should include the 'Grupo público' toggle."""
    admin_page.goto(first_group_url)
    admin_page.wait_for_load_state("networkidle")
    admin_page.get_by_role("button", name="Editar").first.click()
    admin_page.wait_for_selector("text=Grupo público", timeout=5000)
    expect(admin_page.locator("text=Grupo público").first).to_be_visible()


# ── Admin sees waitlist panel ─────────────────────────────────────────────────

def test_admin_ve_painel_lista_de_espera_em_grupo_publico(admin_page: Page, first_group_url: str):
    """
    Admin on a public group with an open match should see the waitlist panel
    (even if empty) in the Próximos tab.
    """
    admin_page.goto(first_group_url)
    admin_page.wait_for_load_state("networkidle")

    # Make sure it's a public group with an open match
    is_public = admin_page.locator("span", has_text="Público").count() > 0
    if not is_public:
        pytest.skip("Grupo não é público")

    GroupPage(admin_page).tab_upcoming()
    has_match = admin_page.get_by_role("link", name="Detalhes").first.is_visible()
    if not has_match:
        pytest.skip("Nenhuma partida aberta")

    # Waitlist panel should be visible (may be empty)
    panel = admin_page.locator("text=/Lista de espera/i")
    expect(panel.first).to_be_visible()


# ── Member does NOT see "join waitlist" button ────────────────────────────────

def test_membro_nao_ve_botao_entrar_na_fila(admin_page: Page, first_group_url: str):
    """
    The admin user is already a member, so they should NOT see
    the "Quero jogar!" / join waitlist button on the group page.
    """
    admin_page.goto(first_group_url)
    admin_page.wait_for_load_state("networkidle")
    GroupPage(admin_page).tab_upcoming()
    # "Quero jogar!" should not be present for a member/admin
    expect(admin_page.locator("button", has_text="Quero jogar!")).not_to_be_visible()


# ── Non-logged user sees CTA on public match page ─────────────────────────────

def test_pagina_da_partida_exibe_cta_para_usuario_nao_logado(page: Page, open_match_url: str):
    """
    An unauthenticated user on a public group's match page with open spots
    should see the CTA to register/login and join the waitlist.
    """
    page.goto(open_match_url)
    page.wait_for_load_state("networkidle")

    # Page should be accessible without auth
    expect(page.locator("h1, h2").first).to_be_visible()

    # Check if CTA is present (only for public groups with open spots)
    cta = page.locator("text=Quer jogar?")
    if cta.count() == 0:
        pytest.skip("Grupo não é público ou partida está lotada")

    expect(cta.first).to_be_visible()
    expect(page.get_by_role("link", name="Criar conta e participar")).to_be_visible()
    expect(page.get_by_role("link", name="Já tenho conta")).to_be_visible()


def test_cta_criar_conta_redireciona_com_parametros(page: Page, open_match_url: str):
    """
    The 'Criar conta e participar' CTA link should include ?next= and ?join_waitlist=1 params.
    """
    page.goto(open_match_url)
    page.wait_for_load_state("networkidle")

    cta = page.locator("text=Quer jogar?")
    if cta.count() == 0:
        pytest.skip("Grupo não é público ou partida está lotada")

    link = page.get_by_role("link", name="Criar conta e participar")
    href = link.get_attribute("href") or ""
    assert "join_waitlist=1" in href, f"Expected join_waitlist=1 in href, got: {href}"
    assert "next=" in href, f"Expected next= param in href, got: {href}"
