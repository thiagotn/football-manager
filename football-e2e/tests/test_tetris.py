"""
E2E — Tetris 3D (games.thiagotn.com, app `games/`).

O jogo virou rota Svelte com o three.js importado como módulo (PRD 054). Como
o canvas WebGL roda em software no CI, nenhuma asserção olha o que foi
desenhado: tudo se apoia no HUD e no DOM.

    cd games && npm run build && npx vite preview --port 4173
    cd football-e2e && pytest tests/test_tetris.py
"""

import os
import urllib.request

import pytest
from playwright.sync_api import Page, expect

GAMES_BASE_URL = os.getenv("GAMES_BASE_URL", "http://localhost:4173").rstrip("/")


def _reachable() -> bool:
    try:
        urllib.request.urlopen(GAMES_BASE_URL + "/", timeout=3)
        return True
    except Exception:
        return False


pytestmark = pytest.mark.skipif(not _reachable(), reason=f"games app fora do ar em {GAMES_BASE_URL}")


def _hud(page: Page, label: str) -> int:
    """Valor de uma caixa do HUD. `label` é o texto do DOM, que o CSS exibe em
    caixa alta — por isso 'Pontos', não 'PONTOS'."""
    return int(page.get_by_text(label, exact=True).locator("xpath=following-sibling::div").inner_text())


@pytest.fixture
def tetris(page: Page) -> Page:
    page.goto(f"{GAMES_BASE_URL}/")
    page.evaluate("() => { localStorage.clear(); localStorage.setItem('games_locale', 'pt-BR'); }")
    page.goto(f"{GAMES_BASE_URL}/tetris/")
    expect(page.get_by_role("button", name="Jogar", exact=True)).to_be_visible()
    return page


def test_hub_linka_o_tetris(page: Page):
    page.goto(f"{GAMES_BASE_URL}/")
    card = page.get_by_role("link", name="Tetris 3D")
    expect(card).to_have_attribute("href", "/tetris/")


def test_manifesto_do_tetris_tem_escopo_proprio(page: Page):
    page.goto(f"{GAMES_BASE_URL}/tetris/")
    href = page.get_attribute("link[rel=manifest]", "href")
    assert href == "/tetris/manifest.webmanifest"

    manifest = page.evaluate("async (url) => (await fetch(url)).json()", GAMES_BASE_URL + href)
    assert manifest["id"] == "/tetris/"
    assert manifest["scope"] == "/tetris/"
    assert manifest["start_url"] == "/tetris/"


def test_abre_no_overlay_e_comeca_zerado(tetris: Page):
    tetris.get_by_role("button", name="Jogar", exact=True).click()
    expect(tetris.get_by_role("button", name="Jogar", exact=True)).to_have_count(0)
    assert _hud(tetris, "Pontos") == 0
    assert _hud(tetris, "Nível") == 1
    assert _hud(tetris, "Linhas") == 0


def test_soltar_pontua(tetris: Page):
    tetris.get_by_role("button", name="Jogar", exact=True).click()
    tetris.get_by_role("button", name="Soltar").click()
    tetris.wait_for_timeout(300)
    # Hard drop paga 2 por linha percorrida — a peça nasce no topo do poço.
    assert _hud(tetris, "Pontos") > 0


def test_teclado_tambem_solta(tetris: Page):
    tetris.get_by_role("button", name="Jogar", exact=True).click()
    tetris.keyboard.press("Space")
    tetris.wait_for_timeout(300)
    assert _hud(tetris, "Pontos") > 0


def test_botoes_tateis_nao_quebram_o_jogo(tetris: Page):
    tetris.get_by_role("button", name="Jogar", exact=True).click()
    for name in ["Mover para a esquerda", "Mover para a direita", "Rotacionar", "Descer"]:
        tetris.get_by_role("button", name=name).click()
        tetris.wait_for_timeout(120)
    assert _hud(tetris, "Nível") == 1


def test_pausa_e_retoma(tetris: Page):
    tetris.get_by_role("button", name="Jogar", exact=True).click()
    tetris.get_by_role("button", name="Pausar").click()
    expect(tetris.get_by_text("Pausado")).to_be_visible()
    tetris.get_by_role("button", name="Continuar").click()
    expect(tetris.get_by_text("Pausado")).to_have_count(0)


def test_volta_para_o_hub(tetris: Page):
    # O link do overlay — o do HUD fica atrás dele enquanto o jogo não começou.
    tetris.get_by_role("link", name="Todos os jogos").last.click()
    expect(tetris.get_by_role("heading", name="Jogos")).to_be_visible()


def test_sem_erro_de_console(tetris: Page):
    erros: list[str] = []
    tetris.on("pageerror", lambda e: erros.append(str(e)))
    tetris.on("console", lambda m: erros.append(m.text) if m.type == "error" else None)

    tetris.get_by_role("button", name="Jogar", exact=True).click()
    tetris.keyboard.press("ArrowLeft")
    tetris.keyboard.press("ArrowUp")
    tetris.keyboard.press("Space")
    tetris.wait_for_timeout(1500)
    assert erros == []


def test_offline_depois_do_service_worker(tetris: Page, context):
    """O chunk do three é grande; o precache tem de cobri-lo mesmo assim."""
    tetris.wait_for_function("() => navigator.serviceWorker.controller !== null", timeout=45_000)
    cached = tetris.evaluate(
        """async () => {
            const key = (await caches.keys())[0];
            const reqs = await (await caches.open(key)).keys();
            return reqs.map((r) => new URL(r.url).pathname);
        }"""
    )
    assert "/tetris/" in cached

    context.set_offline(True)
    try:
        tetris.reload()
        expect(tetris.get_by_role("button", name="Jogar", exact=True)).to_be_visible()
    finally:
        context.set_offline(False)
