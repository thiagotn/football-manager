import pytest
from playwright.sync_api import Page, expect

from pages.login_page import LoginPage


def test_login_com_credenciais_validas(page: Page):
    login = LoginPage(page)
    login.goto()
    login.login("11999990000", "admin123")
    expect(page).to_have_url("**/")


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
    expect(page).to_have_url("**/login**")


def test_logout_redireciona_para_login(admin_page: Page):
    admin_page.goto("/")
    admin_page.get_by_role("link", name="Sair").click()
    expect(admin_page).to_have_url("**/login**")
