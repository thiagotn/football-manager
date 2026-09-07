import base64
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


# ---------------------------------------------------------------------------
# Lightbox da foto do jogador (modal "Detalhes do Cadastro" em /admin/players)
#
# O stack E2E não tem storage (R2/MinIO), então não dá pra fazer upload real.
# Injetamos `avatar_url` na resposta da listagem via page.route e servimos um
# PNG 1×1 também interceptado.
# ---------------------------------------------------------------------------

_PNG_1X1 = base64.b64decode(
    "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg=="
)
_FAKE_AVATAR_PATH = "/e2e-fake-cdn/avatar.png"
_ADMIN_PLAYERS_LIST = re.compile(r".*/api/v\d+/admin/players(\?.*)?$")


def _route_admin_players_with_avatar(page: Page, avatar_url):
    """Reescreve `avatar_url` de todos os itens da listagem de /admin/players."""

    def inject(route):
        resp = route.fetch()
        data = resp.json()
        for item in data.get("items", []):
            item["avatar_url"] = avatar_url
        route.fulfill(response=resp, json=data)

    page.route(_ADMIN_PLAYERS_LIST, inject)


def test_avatar_sem_foto_nao_abre_lightbox(admin_page: Page):
    _route_admin_players_with_avatar(admin_page, None)
    admin_page.goto("/admin/players")
    admin_page.wait_for_load_state("networkidle")
    admin_page.get_by_role("button", name="Detalhes").first.click()
    expect(admin_page.locator("text=Detalhes do Cadastro")).to_be_visible()
    # avatar com iniciais não vira botão
    expect(admin_page.get_by_role("button", name="Ver foto ampliada")).to_have_count(0)
    expect(admin_page.get_by_test_id("avatar-lightbox")).to_have_count(0)


def test_avatar_com_foto_abre_e_fecha_lightbox(admin_page: Page):
    admin_page.route(
        f"**{_FAKE_AVATAR_PATH}",
        lambda route: route.fulfill(status=200, content_type="image/png", body=_PNG_1X1),
    )
    _route_admin_players_with_avatar(admin_page, _FAKE_AVATAR_PATH)
    admin_page.goto("/admin/players")
    admin_page.wait_for_load_state("networkidle")
    admin_page.get_by_role("button", name="Detalhes").first.click()
    expect(admin_page.locator("text=Detalhes do Cadastro")).to_be_visible()

    thumb = admin_page.get_by_role("button", name="Ver foto ampliada")
    expect(thumb).to_be_visible()
    lightbox = admin_page.get_by_test_id("avatar-lightbox")

    # abre e fecha com Escape
    thumb.click()
    expect(lightbox).to_be_visible()
    expect(lightbox.locator("img")).to_be_visible()
    admin_page.keyboard.press("Escape")
    expect(lightbox).to_have_count(0)

    # abre de novo e fecha clicando no fundo escuro (backdrop é o 1º botão "Fechar")
    thumb.click()
    expect(lightbox).to_be_visible()
    lightbox.get_by_role("button", name="Fechar").first.click(position={"x": 5, "y": 5})
    expect(lightbox).to_have_count(0)

    # o modal de detalhes continua aberto por baixo
    expect(admin_page.locator("text=Detalhes do Cadastro")).to_be_visible()
