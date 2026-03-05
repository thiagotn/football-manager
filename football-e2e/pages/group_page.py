from playwright.sync_api import Page


class GroupPage:
    def __init__(self, page: Page):
        self.page = page

    def tab_upcoming(self):
        self.page.get_by_role("button", name=lambda n: n.startswith("Próximos")).click()

    def tab_past(self):
        self.page.get_by_role("button", name=lambda n: n.startswith("Últimos")).click()

    def tab_members(self):
        self.page.get_by_role("button", name=lambda n: n.startswith("Jogadores")).click()

    def new_match_button(self):
        return self.page.get_by_role("button", name="Novo Rachão")

    def invite_button(self):
        return self.page.get_by_role("button", name="Convidar")

    def add_member_button(self):
        return self.page.get_by_role("button", name="Adicionar")

    def edit_group_button(self):
        return self.page.get_by_role("button", name="Editar").first

    def match_cards(self):
        return self.page.locator(".card").filter(has=self.page.get_by_role("link", name="Detalhes"))

    def member_rows(self):
        return self.page.locator(".divide-y > div")
