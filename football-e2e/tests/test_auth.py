import re

import pytest
from playwright.sync_api import Page, expect

from pages.login_page import LoginPage


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
