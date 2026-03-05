from playwright.sync_api import Page


class LoginPage:
    def __init__(self, page: Page):
        self.page = page

    def goto(self):
        self.page.goto("/login")

    def login(self, whatsapp: str, password: str):
        self.page.locator("#whatsapp").fill(whatsapp)
        self.page.locator("#password").fill(password)
        self.page.get_by_role("button", name="Entrar").click()

    def error_message(self) -> str | None:
        el = self.page.locator(".alert-error")
        return el.text_content() if el.is_visible() else None
