from playwright.sync_api import Page


class MatchPage:
    def __init__(self, page: Page):
        self.page = page

    def status_badge(self) -> str:
        return self.page.locator(".badge-green, .badge-gray").first.text_content() or ""

    def is_open(self) -> bool:
        return "Aberta" in self.status_badge()

    def confirmed_count(self) -> int:
        text = self.page.locator("text=/\\d+\\/\\d+/").first.text_content() or "0/0"
        return int(text.split("/")[0])

    def share_whatsapp_button(self):
        return self.page.get_by_role("link", name="Compartilhar no WhatsApp")

    def copy_link_button(self):
        return self.page.get_by_role("button", name="Copiar link")
