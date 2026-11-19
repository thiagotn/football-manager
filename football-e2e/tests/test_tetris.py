"""
E2E — Tetris 3D (games.thiagotn.com, app `games/`).

O jogo virou rota Svelte com o three.js importado como módulo (PRD 054). Como
o canvas WebGL roda em software no CI, nenhuma asserção olha o que foi
desenhado: tudo se apoia no HUD e no DOM. O modo da arena aparece no DOM em
`canvas[data-depth]` justamente por isso — a câmera em si não é observável
daqui. A peça em jogo (`data-piece-*`) e o tamanho da casa (`data-cell`)
também, para os testes dos gestos: no celular não há botões, um dedo na arena
joga e, no 3D, dois dedos giram a câmera.

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


def _peca(page: Page) -> dict:
    """A peça em jogo, lida de `canvas[data-piece-*]` (e o tamanho da casa)."""
    cv = page.locator("canvas[data-depth]")
    return {
        "type": cv.get_attribute("data-piece-type"),
        "col": int(cv.get_attribute("data-piece-col") or -99),
        "row": int(cv.get_attribute("data-piece-row") or -99),
        "rot": int(cv.get_attribute("data-piece-rot") or 0),
        "cell": int(cv.get_attribute("data-cell")),
    }


def _arrastar(page: Page, x0: float, y0: float, x1: float, y1: float, steps: int, pausa_ms: int = 0) -> None:
    """Um dedo (o mouse gera os mesmos pointer events) arrastando na arena."""
    page.mouse.move(x0, y0)
    page.mouse.down()
    for i in range(1, steps + 1):
        page.mouse.move(x0 + (x1 - x0) * i / steps, y0 + (y1 - y0) * i / steps)
        if pausa_ms:
            page.wait_for_timeout(pausa_ms)
    page.mouse.up()


def _jogando_com_peca_que_gira(page: Page) -> dict:
    """Começa a partida e garante uma peça que não seja o quadrado (O não gira)."""
    page.get_by_role("button", name="Jogar", exact=True).click()
    for _ in range(6):
        peca = _peca(page)
        if peca["type"] and peca["type"] != "O":
            return peca
        page.keyboard.press("Space")
        page.wait_for_timeout(150)
    pytest.fail("só vieram quadrados")


def _ajustes(page: Page) -> dict:
    """O que está salvo em `games_tetris`."""
    return page.evaluate("() => JSON.parse(localStorage.getItem('games_tetris') ?? 'null')")


@pytest.fixture
def tetris_novo(page: Page) -> Page:
    """Aparelho sem nada salvo: a pergunta do 3D ainda vai aparecer."""
    page.goto(f"{GAMES_BASE_URL}/")
    page.evaluate("() => { localStorage.clear(); localStorage.setItem('games_locale', 'pt-BR'); }")
    page.goto(f"{GAMES_BASE_URL}/tetris/")
    expect(page.get_by_role("button", name="Jogar", exact=True)).to_be_visible()
    return page


@pytest.fixture
def tetris(tetris_novo: Page) -> Page:
    """Pergunta do 3D já respondida (arena plana) — é o estado da maioria dos testes."""
    tetris_novo.evaluate(
        "() => localStorage.setItem('games_tetris', JSON.stringify({ settings: { depth: false }, asked: true }))"
    )
    tetris_novo.reload()
    expect(tetris_novo.get_by_role("button", name="Jogar", exact=True)).to_be_visible()
    return tetris_novo


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


def test_teclado_tambem_solta(tetris: Page):
    tetris.get_by_role("button", name="Jogar", exact=True).click()
    tetris.keyboard.press("Space")
    tetris.wait_for_timeout(300)
    assert _hud(tetris, "Pontos") > 0


def test_sem_setas_na_tela(tetris: Page):
    tetris.get_by_role("button", name="Jogar", exact=True).click()
    for nome in ["Mover para a esquerda", "Mover para a direita", "Rotacionar", "Descer", "Soltar"]:
        expect(tetris.get_by_role("button", name=nome)).to_have_count(0)
    # A pausa continua, no topo, ao lado do voltar.
    expect(tetris.get_by_role("button", name="Pausar")).to_be_visible()


def test_toque_na_arena_gira_a_peca(tetris: Page):
    antes = _jogando_com_peca_que_gira(tetris)
    tetris.mouse.click(195, 450)
    assert _peca(tetris)["rot"] == antes["rot"] + 1


def test_arrastar_para_o_lado_move_a_peca(tetris: Page):
    antes = _jogando_com_peca_que_gira(tetris)
    cell = antes["cell"]
    _arrastar(tetris, 120, 450, 120 + 2 * cell + cell // 2, 452, steps=10)
    depois = _peca(tetris)
    assert depois["col"] == antes["col"] + 2
    assert depois["rot"] == antes["rot"]  # arrastar não gira

    _arrastar(tetris, 260, 450, 260 - 2 * cell - cell // 2, 449, steps=10)
    assert _peca(tetris)["col"] == antes["col"]


def test_arrastar_para_baixo_devagar_desce_sem_soltar(tetris: Page):
    antes = _jogando_com_peca_que_gira(tetris)
    cell = antes["cell"]
    _arrastar(tetris, 195, 250, 196, 250 + 3 * cell + 5, steps=12, pausa_ms=40)
    depois = _peca(tetris)
    assert depois["type"] == antes["type"]
    assert depois["row"] >= antes["row"] + 3
    assert _hud(tetris, "Pontos") == 0  # descer devagar não pontua nem trava


def test_deslizar_rapido_para_baixo_solta_a_peca(tetris: Page):
    antes = _jogando_com_peca_que_gira(tetris)
    _arrastar(tetris, 195, 200, 195, 200 + 8 * antes["cell"], steps=4)
    tetris.wait_for_timeout(200)
    # Soltou: a peça travou no fundo e a próxima nasceu no topo.
    assert _peca(tetris)["row"] < antes["row"] + 3
    assert _hud(tetris, "Pontos") > 0


def test_dois_dedos_giram_a_camera_no_3d_sem_mexer_na_peca(browser):
    ctx = browser.new_context(viewport={"width": 390, "height": 844}, has_touch=True, is_mobile=True)
    page = ctx.new_page()
    page.goto(f"{GAMES_BASE_URL}/")
    page.evaluate(
        "() => { localStorage.setItem('games_locale', 'pt-BR');"
        " localStorage.setItem('games_tetris', JSON.stringify({ settings: { depth: true }, asked: true })); }"
    )
    page.goto(f"{GAMES_BASE_URL}/tetris/")
    page.get_by_role("button", name="Jogar", exact=True).click()
    page.get_by_role("button", name="Pausar").click()  # a peça fica parada durante o gesto
    page.get_by_role("button", name="Continuar").click()
    antes = _peca(page)
    tela_antes = page.screenshot()

    cdp = ctx.new_cdp_session(page)

    def toque(tipo: str, pontos: list[tuple[float, float]]) -> None:
        cdp.send("Input.dispatchTouchEvent", {"type": tipo, "touchPoints": [{"x": x, "y": y, "id": i} for i, (x, y) in enumerate(pontos)]})

    toque("touchStart", [(120, 450), (270, 450)])
    for i in range(1, 9):
        toque("touchMove", [(120 + i * 15, 450), (270 + i * 15, 450)])
    toque("touchEnd", [])
    page.wait_for_timeout(150)

    depois = _peca(page)
    assert (depois["col"], depois["rot"]) == (antes["col"], antes["rot"])
    assert page.screenshot() != tela_antes  # a câmera girou
    ctx.close()


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


def test_3d_desligado_por_padrao(tetris_novo: Page):
    assert _ajustes(tetris_novo) is None
    expect(tetris_novo.locator("canvas[data-depth]")).to_have_attribute("data-depth", "false")
    expect(tetris_novo.get_by_text("Arraste para mover · toque para girar", exact=False)).to_be_visible()


def test_pergunta_o_3d_na_primeira_partida(tetris_novo: Page):
    tetris_novo.get_by_role("button", name="Jogar", exact=True).click()
    expect(tetris_novo.get_by_role("heading", name="Jogar em 3D?")).to_be_visible()
    # A pergunta vem antes da partida: o HUD segue zerado.
    assert _hud(tetris_novo, "Pontos") == 0


def test_responder_plano_mantem_a_arena_fixa(tetris_novo: Page):
    tetris_novo.get_by_role("button", name="Jogar", exact=True).click()
    tetris_novo.get_by_role("button", name="Jogar plano").click()

    expect(tetris_novo.get_by_role("heading", name="Jogar em 3D?")).to_have_count(0)
    expect(tetris_novo.get_by_role("button", name="Jogar", exact=True)).to_have_count(0)
    expect(tetris_novo.locator("canvas[data-depth]")).to_have_attribute("data-depth", "false")
    assert _ajustes(tetris_novo) == {"settings": {"depth": False}, "asked": True}


def test_responder_3d_liga_o_ajuste(tetris_novo: Page):
    tetris_novo.get_by_role("button", name="Jogar", exact=True).click()
    tetris_novo.get_by_role("button", name="Jogar em 3D").click()

    expect(tetris_novo.locator("canvas[data-depth]")).to_have_attribute("data-depth", "true")
    expect(tetris_novo.get_by_text("dois dedos giram a câmera", exact=False)).to_be_visible()
    assert _ajustes(tetris_novo) == {"settings": {"depth": True}, "asked": True}


def test_pergunta_nao_volta_na_proxima_partida(tetris_novo: Page):
    tetris_novo.get_by_role("button", name="Jogar", exact=True).click()
    tetris_novo.get_by_role("button", name="Jogar plano").click()
    tetris_novo.reload()

    tetris_novo.get_by_role("button", name="Jogar", exact=True).click()
    expect(tetris_novo.get_by_role("heading", name="Jogar em 3D?")).to_have_count(0)
    expect(tetris_novo.get_by_role("button", name="Jogar", exact=True)).to_have_count(0)


def test_chave_do_3d_no_inicio_liga_e_desliga(tetris: Page):
    chave = tetris.get_by_label("Arena em 3D")
    expect(chave).not_to_be_checked()

    chave.check()
    expect(tetris.locator("canvas[data-depth]")).to_have_attribute("data-depth", "true")
    assert _ajustes(tetris)["settings"]["depth"] is True

    chave.uncheck()
    expect(tetris.locator("canvas[data-depth]")).to_have_attribute("data-depth", "false")
    assert _ajustes(tetris)["settings"]["depth"] is False


def test_chave_do_3d_na_pausa_vale_no_meio_da_partida(tetris: Page):
    tetris.get_by_role("button", name="Jogar", exact=True).click()
    tetris.get_by_role("button", name="Pausar").click()

    tetris.get_by_label("Arena em 3D").check()
    expect(tetris.locator("canvas[data-depth]")).to_have_attribute("data-depth", "true")

    tetris.get_by_role("button", name="Continuar").click()
    expect(tetris.get_by_text("Pausado")).to_have_count(0)
    assert _ajustes(tetris)["settings"]["depth"] is True


def test_3d_salvo_volta_ligado_no_reload(tetris: Page):
    tetris.get_by_label("Arena em 3D").check()
    tetris.reload()
    expect(tetris.locator("canvas[data-depth]")).to_have_attribute("data-depth", "true")
    expect(tetris.get_by_label("Arena em 3D")).to_be_checked()


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
