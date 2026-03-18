from playwright.sync_api import Page


class DashboardPage:
    def __init__(self, page: Page):
        self.page = page

    def goto(self):
        # Admin users are redirected from "/" to "/admin"; go to "/groups" which works for all roles
        self.page.goto("/groups")

    def select_upcoming_tab(self):
        self.page.get_by_role("button", name="Próximos Rachões").click()

    def select_past_tab(self):
        self.page.get_by_role("button", name="Últimos Rachões").click()

    def group_links(self):
        return self.page.locator("a[href^='/groups/']")

    def match_items(self):
        return self.page.locator("a[href^='/match/']")
