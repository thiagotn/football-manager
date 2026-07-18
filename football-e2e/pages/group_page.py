import re

from playwright.sync_api import Page


class GroupPage:
    def __init__(self, page: Page):
        self.page = page

    def tab_upcoming(self):
        self.page.get_by_role("button", name=re.compile(r"Atuais")).click()

    def tab_past(self):
        self.page.get_by_role("button", name=re.compile(r"Últimos")).click()

    def tab_members(self):
        self.page.get_by_role("button", name=re.compile(r"Jogadores")).click()

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

    def sort_by_name_pill(self):
        # Casa "Nome A–Z" e "Nome Z–A" (o rótulo alterna com a direção)
        return self.page.get_by_role("button", name=re.compile(r"Nome [AZ]"))

    def sort_by_recent_pill(self):
        return self.page.get_by_role("button", name=re.compile(r"Mais recentes"))
