import re

import pytest
from playwright.sync_api import Page, expect

from pages.login_page import LoginPage


def test_sessao_expirada_exibe_toast_e_redireciona(admin_page: Page):
    """
    Simula expiração de JWT interceptando GET /groups e retornando 401.
    Verifica que o toast de sessão expirada aparece, o usuário é
    redirecionado para /login e o banner contextual é exibido.
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

    # Toast "Sua sessão expirou" deve aparecer
    expect(admin_page.get_by_text("Sua sessão expirou. Faça login novamente.")).to_be_visible(
        timeout=5000
    )

    # Deve redirecionar para /login
    expect(admin_page).to_have_url(re.compile(r".*/login"), timeout=5000)

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
