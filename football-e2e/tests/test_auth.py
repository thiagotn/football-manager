import re

import pytest
from playwright.sync_api import Page, expect

from pages.login_page import LoginPage


def test_sessao_expirada_redireciona_e_exibe_banner(admin_page: Page):
    """
    Simula expiração de JWT interceptando GET /groups e retornando 401.
    Verifica que o usuário é redirecionado para /login e o banner
    contextual de sessão expirada é exibido.
    """
    admin_page.route(
        "**/api/v1/groups",
        lambda route: route.fulfill(
            status=401,
            content_type="application/json",
            body='{"detail": "Not authenticated"}',
        ),
    )

    admin_page.goto("/groups")

    # Deve redirecionar para /login
    expect(admin_page).to_have_url(re.compile(r".*/login"), timeout=8000)

    # Banner contextual na página de login deve estar visível
    expect(admin_page.locator("[data-testid='session-expired-banner']")).to_be_visible()


def test_login_com_credenciais_validas(page: Page):
    login = LoginPage(page)
    login.goto()
    login.login("11999990000", "admin123")
    # Admin may be redirected to /profile (must_change_password) or / depending on seed
    expect(page).not_to_have_url(re.compile(r".*/login"))


def test_login_com_senha_incorreta(page: Page):
    login = LoginPage(page)
    login.goto()
    login.login("11999990000", "senhaerrada")
    expect(page.locator(".alert-error")).to_be_visible()


def test_login_com_whatsapp_inexistente(page: Page):
    login = LoginPage(page)
    login.goto()
    login.login("00000000000", "qualquercoisa")
    expect(page.locator(".alert-error")).to_be_visible()


def test_redireciona_para_login_sem_autenticacao(page: Page):
    page.goto("/groups")
    expect(page).to_have_url(re.compile(r".*/login"))


def test_logout_redireciona_para_login(admin_page: Page):
    admin_page.goto("/groups")
    # "Sair" is a <button> in the Navbar, not a link
    admin_page.get_by_role("button", name="Sair").first.click()
    expect(admin_page).to_have_url(re.compile(r".*/login"))
