import re

import pytest
from playwright.sync_api import Page, expect


def test_listagem_de_jogadores_carrega(admin_page: Page):
    admin_page.goto("/players")
    admin_page.wait_for_load_state("networkidle")
    expect(admin_page.locator("table")).to_be_visible()
    assert admin_page.locator("table tbody tr").count() > 0


def test_botao_editar_visivel_na_listagem(admin_page: Page):
    # Editar is now inside the Detalhes modal
    admin_page.goto("/players")
    admin_page.wait_for_load_state("networkidle")
    admin_page.get_by_role("button", name="Detalhes").first.click()
    expect(admin_page.get_by_role("button", name="Editar")).to_be_visible()


def test_botao_senha_visivel_na_listagem(admin_page: Page):
    # Resetar Senha is now inside the Detalhes modal
    admin_page.goto("/players")
    admin_page.wait_for_load_state("networkidle")
    admin_page.get_by_role("button", name="Detalhes").first.click()
    expect(admin_page.get_by_role("button", name="Resetar Senha")).to_be_visible()


def test_modal_editar_jogador_abre(admin_page: Page):
    admin_page.goto("/players")
    admin_page.wait_for_load_state("networkidle")
    admin_page.get_by_role("button", name="Detalhes").first.click()
    admin_page.get_by_role("button", name="Editar").click()
    expect(admin_page.locator("text=Editar —")).to_be_visible()


def test_modal_reset_senha_abre(admin_page: Page):
    admin_page.goto("/players")
    admin_page.wait_for_load_state("networkidle")
    admin_page.get_by_role("button", name="Detalhes").first.click()
    admin_page.get_by_role("button", name="Resetar Senha").click()
    expect(admin_page.locator("text=Resetar Senha —")).to_be_visible()


def test_busca_por_nome_filtra_resultado(admin_page: Page):
    admin_page.goto("/players")
    admin_page.wait_for_load_state("networkidle")  # wait for initial data to load
    rows_before = admin_page.locator("table tbody tr").count()
    admin_page.locator("input[placeholder*='Buscar']").fill("admin")
    admin_page.wait_for_load_state("networkidle")  # wait for search results
    rows_after = admin_page.locator("table tbody tr").count()
    assert rows_after <= rows_before


def test_pagina_players_inacessivel_sem_autenticacao(page: Page):
    import re
    page.goto("/players")
    expect(page).to_have_url(re.compile(r".*/login"))
