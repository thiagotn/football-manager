"""
E2E — Enche Aí! (games.thiagotn.com, app `games/`).

App separado do rachao.app: roda em `GAMES_BASE_URL` (padrão: `vite preview`
do `games/` em http://localhost:4173). 100% client-side — estado em
`localStorage` (`games_enche_ai`), fases geradas por seed e validadas por solver.

    cd games && npm run build && npx vite preview --port 4173
    cd football-e2e && pytest tests/test_enche_ai.py
"""

import json
import os
import urllib.request

import pytest
from playwright.sync_api import Page, expect

GAMES_BASE_URL = os.getenv("GAMES_BASE_URL", "http://localhost:4173").rstrip("/")
STORAGE_KEY = "games_enche_ai"


def _reachable() -> bool:
    try:
        urllib.request.urlopen(GAMES_BASE_URL + "/", timeout=3)
        return True
    except Exception:
        return False


pytestmark = pytest.mark.skipif(not _reachable(), reason=f"games app fora do ar em {GAMES_BASE_URL}")


def _saved(page: Page) -> dict:
    return page.evaluate(f"() => JSON.parse(localStorage.getItem('{STORAGE_KEY}'))")


def _tubes(page: Page):
    return page.get_by_role("region", name="Recipientes no balcão").get_by_role("button")


def _open(page: Page, state: dict | None = None) -> Page:
    """Abre o jogo em pt-BR com o localStorage limpo (ou com `state`).

    O estado é gravado a partir do hub: a tela do jogo salva a partida ao sair
    (`pagehide`) e sobrescreveria o que fosse injetado com ela aberta.
    """
    page.goto(f"{GAMES_BASE_URL}/")
    page.evaluate(
        "([k, s]) => { localStorage.clear(); localStorage.setItem('games_locale', 'pt-BR');"
        " if (s) localStorage.setItem(k, s); }",
        [STORAGE_KEY, json.dumps(state) if state else None],
    )
    page.goto(f"{GAMES_BASE_URL}/enche-ai/")
    return page


def _near_win(level: int = 1) -> dict:
    """Pedido 1 (café, leite, chocolate) a uma jogada da vitória."""
    return {
        "current": {
            "level": level,
            "daily": None,
            "board": [["cafe", "cafe", "cafe"], ["cafe"], ["leite"] * 4, ["chocolate"] * 4, [], []],
            "history": [],
            "moves": 7,
            "elapsed": 30,
            "left": {"undo": 3, "extra": 1, "hint": 3},
            "used": {"undo": False, "extra": False, "hint": False},
        },
        "level": level,
        "stars": {},
        "daily": {},
        "settings": {"patterns": True, "vibrate": False},
    }


# ── Hub ───────────────────────────────────────────────────────────────────────


def test_hub_lista_o_enche_ai(page: Page):
    page.goto(f"{GAMES_BASE_URL}/")
    page.evaluate("() => localStorage.setItem('games_locale', 'pt-BR')")
    page.reload()
    expect(page.get_by_role("heading", name="Jogos")).to_be_visible()
    page.get_by_role("link", name="Enche Aí!").click()
    expect(page).to_have_url(f"{GAMES_BASE_URL}/enche-ai/")


def test_manifesto_do_jogo_tem_escopo_proprio(page: Page):
    _open(page)
    href = page.locator("link[rel=manifest]").get_attribute("href")
    assert href == "/enche-ai/manifest.webmanifest"
    manifest = page.evaluate(f"() => fetch('{href}').then(r => r.json())")
    assert manifest["scope"] == "/enche-ai/"
    assert manifest["start_url"] == "/enche-ai/"


# ── Jogo ──────────────────────────────────────────────────────────────────────


def test_primeira_visita_abre_o_inicio(page: Page):
    _open(page)
    expect(page.get_by_role("dialog", name="Enche Aí!")).to_be_visible()
    page.get_by_role("button", name="Começar", exact=True).click()
    expect(page.get_by_role("heading", name="Pedido 1")).to_be_visible()
    # Pedido 1: 3 sabores + 2 recipientes vazios
    expect(_tubes(page)).to_have_count(5)


def test_despejar_conta_jogada_e_desfazer_volta(page: Page):
    state = _near_win()
    state["current"]["board"] = [["leite", "cafe"], ["cafe", "cafe", "cafe"], ["leite"] * 3, ["chocolate"] * 4, [], []]
    _open(page, state)
    tubes = _tubes(page)

    tubes.nth(0).click()
    tubes.nth(1).click()
    expect(page.get_by_text("Café pronto!")).to_be_visible()
    assert _saved(page)["current"]["board"][1] == ["cafe"] * 4
    assert _saved(page)["current"]["moves"] == 8

    page.get_by_role("button", name="Desfazer jogada").click()
    saved = _saved(page)["current"]
    assert saved["board"][0] == ["leite", "cafe"]
    assert saved["left"]["undo"] == 2
    assert saved["used"]["undo"] is True


def test_despejo_invalido_nao_muda_nada(page: Page):
    state = _near_win()
    state["current"]["board"] = [["leite", "cafe"], ["cafe", "leite"], ["cafe", "cafe", "leite", "leite"], ["chocolate"] * 4, [], []]
    _open(page, state)
    tubes = _tubes(page)
    tubes.nth(0).click()
    tubes.nth(1).click()  # café sobre leite: inválido → só troca a seleção
    assert _saved(page)["current"]["moves"] == 7
    expect(tubes.nth(1)).to_have_attribute("aria-pressed", "true")


def test_vitoria_da_estrelas_e_libera_o_proximo(page: Page):
    _open(page, _near_win())
    tubes = _tubes(page)
    tubes.nth(1).click()
    tubes.nth(0).click()

    dialog = page.get_by_role("dialog", name="Pedido entregue!")
    expect(dialog).to_be_visible()
    saved = _saved(page)
    assert saved["stars"]["1"] == 3
    assert saved["level"] == 2
    assert saved["current"] is None

    dialog.get_by_role("button", name="Próximo pedido").click()
    expect(page.get_by_role("heading", name="Pedido 2")).to_be_visible()


def test_copo_extra_e_dica_custam_estrela(page: Page):
    state = _near_win()
    state["current"]["board"] = [["leite", "cafe"], ["cafe", "cafe", "cafe"], ["leite"] * 3, ["chocolate"] * 4, [], []]
    _open(page, state)
    page.get_by_role("button", name="Copo extra").click()
    expect(_tubes(page)).to_have_count(7)
    page.get_by_role("button", name="Dica").click()
    expect(page.get_by_text("Despeje do primeiro copo que brilha")).to_be_visible()
    saved = _saved(page)["current"]
    assert saved["left"] == {"undo": 3, "extra": 0, "hint": 2}
    expect(page.get_by_label("1 de 3 estrelas possíveis")).to_be_visible()


def test_partida_sobrevive_ao_reload(page: Page):
    _open(page, _near_win())
    page.reload()
    expect(page.get_by_role("heading", name="Pedido 1")).to_be_visible()
    assert _saved(page)["current"]["moves"] == 7


def test_pedido_do_dia_e_deterministico(page: Page):
    _open(page, _near_win())
    page.get_by_role("button", name="Pedido do dia").click()
    first = _saved(page)["current"]
    assert first["daily"] is not None
    page.get_by_role("button", name="Recomeçar pedido").click()
    assert _saved(page)["current"]["board"] == first["board"]


def test_cardapio_bloqueado_ate_liberar(page: Page):
    _open(page, _near_win())
    page.get_by_role("button", name="Início").click()
    expect(page.get_by_role("button", name="Lanchonete")).to_be_disabled()
    expect(page.get_by_role("button", name="Padaria")).to_be_enabled()


def test_service_worker_instala_e_o_jogo_abre_offline(page: Page, context):
    """PWA: o service worker precisa ativar e servir o jogo sem rede.

    Já quebrou em produção: `_headers` (configuração do Cloudflare Pages) ia
    para a lista de precache, o Pages devolvia 404 e o `cache.addAll` original
    derrubava a instalação inteira — ficava sem service worker e sem offline.
    """
    page.goto(f"{GAMES_BASE_URL}/enche-ai/")
    page.wait_for_function("() => navigator.serviceWorker.controller !== null", timeout=45_000)

    cached = page.evaluate(
        """async () => {
            const key = (await caches.keys())[0];
            const reqs = await (await caches.open(key)).keys();
            return reqs.map((r) => new URL(r.url).pathname);
        }"""
    )
    assert len(cached) > 20, f"precache quase vazio: {cached}"
    assert "/enche-ai/" in cached
    # Config do Pages nunca é servida: se entrar no precache, a instalação falha
    assert not [p for p in cached if p in ("/_headers", "/_redirects", "/_routes.json")]

    context.set_offline(True)
    try:
        page.reload()
        expect(page.get_by_role("region", name="Recipientes no balcão")).to_be_visible()
    finally:
        context.set_offline(False)


def test_troca_de_idioma(page: Page):
    _open(page, _near_win())
    page.get_by_role("button", name="Início").click()
    page.get_by_role("button", name="EN", exact=True).click()
    expect(page.get_by_role("heading", name="Fill It Up!")).to_be_visible()
    page.get_by_role("button", name="Close").click()
    expect(page.get_by_role("heading", name="Order 1")).to_be_visible()
