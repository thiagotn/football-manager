"""
E2E — Quebra-Cuca (games.thiagotn.com/quebra-cuca/, app `games/`, PRD 059).

100% client-side: estado em `localStorage` (`games_quebra_cuca`). Cada peça
expõe a posição certa em `data-piece="linha-coluna"` e se já encaixou em
`data-placed`; o tabuleiro é `[data-board]` (cena de 400×300 unidades), então o
teste calcula o centro de cada vaga e arrasta a peça até lá com o mouse.

    cd games && npm run build && npx vite preview --port 4173
    cd football-e2e && pytest tests/test_quebra_cuca.py
"""

import json
import os
import re
import urllib.request

import pytest
from playwright.sync_api import Page, expect

GAMES_BASE_URL = os.getenv("GAMES_BASE_URL", "http://localhost:4173").rstrip("/")
STORAGE_KEY = "games_quebra_cuca"
SCENE_W, SCENE_H = 400, 300
GRID = {4: (2, 2), 6: (3, 2), 9: (3, 3), 12: (4, 3)}


def _reachable() -> bool:
    try:
        urllib.request.urlopen(GAMES_BASE_URL + "/", timeout=3)
        return True
    except Exception:
        return False


pytestmark = pytest.mark.skipif(not _reachable(), reason=f"games app fora do ar em {GAMES_BASE_URL}")


def _saved(page: Page) -> dict:
    return json.loads(page.evaluate(f"() => localStorage.getItem('{STORAGE_KEY}')") or "{}")


def _open(page: Page, state: dict | None = None, locale: str = "pt-BR") -> Page:
    """Abre o jogo limpo (ou com `state`), gravado a partir do hub."""
    page.set_viewport_size({"width": 390, "height": 844})
    page.goto(f"{GAMES_BASE_URL}/")
    page.evaluate(
        "([k, s, l]) => { localStorage.clear(); localStorage.setItem('games_locale', l);"
        " if (s) localStorage.setItem(k, s); }",
        [STORAGE_KEY, json.dumps(state) if state else None, locale],
    )
    page.goto(f"{GAMES_BASE_URL}/quebra-cuca/")
    expect(page.locator("[data-theme]")).to_have_count(4)
    return page


def _start(page: Page, theme: str = "dinos", scene: str = "vale", pieces: int = 4) -> None:
    page.locator(f"[data-theme='{theme}']").click()
    page.locator(f"[data-scene='{scene}']").click()
    page.locator(f"[data-pieces='{pieces}']").click()
    expect(page.locator("[data-piece]")).to_have_count(pieces)
    page.wait_for_timeout(300)  # bandeja medida e assentada


def _drag(page: Page, piece_id: str, x: float, y: float) -> None:
    piece = page.locator(f"[data-piece='{piece_id}']")
    box = piece.bounding_box()
    assert box
    page.mouse.move(box["x"] + box["width"] / 2, box["y"] + box["height"] / 2)
    page.mouse.down()
    page.mouse.move(x, y, steps=8)
    page.mouse.up()
    page.wait_for_timeout(350)  # a peça desliza até a vaga ou de volta à bandeja


def _slot_center(page: Page, piece_id: str, pieces: int) -> tuple[float, float]:
    board = page.locator("[data-board]").bounding_box()
    assert board
    cols, rows = GRID[pieces]
    r, c = (int(v) for v in piece_id.split("-"))
    k = board["width"] / SCENE_W
    return (
        board["x"] + (c + 0.5) * SCENE_W / cols * k,
        board["y"] + (r + 0.5) * SCENE_H / rows * k,
    )


def _solve(page: Page, pieces: int) -> None:
    for pid in page.locator("[data-piece]").evaluate_all("els => els.map(e => e.dataset.piece)"):
        _drag(page, pid, *_slot_center(page, pid, pieces))
        expect(page.locator(f"[data-piece='{pid}']")).to_have_attribute("data-placed", "true")


# ── Hub e abertura ──────────────────────────────────────────────────────────


def test_hub_mostra_o_card_com_selo_novo(page: Page) -> None:
    page.goto(f"{GAMES_BASE_URL}/")
    page.evaluate("() => localStorage.setItem('games_locale', 'pt-BR')")
    page.reload()
    card = page.locator("a[href='/quebra-cuca/']")
    expect(card).to_be_visible()
    expect(card.get_by_text("Quebra-Cuca", exact=True)).to_be_visible()
    expect(card.get_by_text("Para crianças", exact=True)).to_be_visible()
    expect(card.get_by_text("Novo", exact=True)).to_be_visible()


def test_manifesto_tem_escopo_proprio(page: Page) -> None:
    _open(page)
    href = page.locator("link[rel=manifest]").get_attribute("href")
    assert href == "/quebra-cuca/manifest.webmanifest"
    manifest = page.evaluate(f"() => fetch('{href}').then(r => r.json())")
    assert manifest["scope"] == "/quebra-cuca/"
    assert manifest["start_url"] == "/quebra-cuca/"


def test_quatro_temas_com_tres_cenas_cada(page: Page) -> None:
    _open(page)
    for theme in ("dinos", "herois", "fazenda", "circo"):
        page.locator(f"[data-theme='{theme}']").click()
        expect(page.locator("[data-scene]")).to_have_count(3)
        page.locator("header button").first.click()  # voltar
    expect(page.get_by_text("Dinossauros")).to_be_visible()
    expect(page.get_by_text("Circo")).to_be_visible()


def test_escolhe_quantas_pecas(page: Page) -> None:
    _open(page)
    page.locator("[data-theme='circo']").click()
    page.locator("[data-scene='lona']").click()
    expect(page.locator("[data-pieces]")).to_have_count(4)
    page.locator("[data-pieces='9']").click()
    expect(page.locator("[data-piece]")).to_have_count(9)
    expect(page.locator("[data-piece][data-placed='false']")).to_have_count(9)
    expect(page.get_by_text("0 de 9 peças")).to_be_visible()


# ── Montar ──────────────────────────────────────────────────────────────────


def test_peca_longe_do_lugar_volta_para_a_bandeja(page: Page) -> None:
    _open(page)
    _start(page)
    board = page.locator("[data-board]").bounding_box()
    assert board
    # Peça 0-0 solta no canto oposto (vaga 1-1): não encaixa.
    _drag(page, "0-0", *_slot_center(page, "1-1", 4))
    expect(page.locator("[data-piece='0-0']")).to_have_attribute("data-placed", "false")
    box = page.locator("[data-piece='0-0']").bounding_box()
    assert box and box["y"] > board["y"] + board["height"], "voltou para a bandeja, embaixo do tabuleiro"
    expect(page.get_by_text("0 de 4 peças")).to_be_visible()


def test_peca_perto_do_lugar_encaixa(page: Page) -> None:
    _open(page)
    _start(page)
    x, y = _slot_center(page, "1-0", 4)
    _drag(page, "1-0", x + 12, y - 10)  # não precisa ser exato
    expect(page.locator("[data-piece='1-0']")).to_have_attribute("data-placed", "true")
    expect(page.get_by_text("1 de 4 peças")).to_be_visible()


def test_montar_tudo_comemora_e_guarda_as_estrelas(page: Page) -> None:
    _open(page)
    _start(page, "fazenda", "bichos", 6)
    _solve(page, 6)
    done = page.get_by_role("dialog")
    expect(done.get_by_text("Parabéns!")).to_be_visible()
    expect(done.get_by_role("button", name="9 peças")).to_be_visible()
    saved = _saved(page)
    assert saved["done"] == {"bichos": 6}
    assert saved["settings"]["pieces"] == 6

    # Escolher outro → o tema mostra a cena com estrelas e o progresso.
    done.get_by_role("button", name="Escolher outro").click()
    expect(page.locator("[data-scene='bichos']").get_by_label("2 estrelas")).to_be_visible()
    page.locator("header button").first.click()
    expect(page.locator("[data-theme='fazenda']").get_by_text("1/3")).to_be_visible()


def test_proximo_abre_a_proxima_cena_do_tema(page: Page) -> None:
    _open(page)
    _start(page, "herois", "voo", 4)
    _solve(page, 4)
    page.get_by_role("button", name="Próximo").click()
    expect(page.get_by_role("heading", name="Heroína no espaço")).to_be_visible()
    expect(page.locator("[data-piece][data-placed='false']")).to_have_count(4)


def test_figura_de_fundo_pode_ser_desligada(page: Page) -> None:
    _open(page)
    _start(page)
    expect(page.locator("[data-board] use")).to_have_count(1)
    page.locator("header button").first.click()  # sair: volta para as cenas do tema
    expect(page.locator("[data-scene]")).to_have_count(3)
    page.get_by_role("button", name="Ajustes").click()
    page.locator("#qc-ghost").uncheck()
    page.get_by_role("button", name="Fechar").click()
    assert _saved(page)["settings"]["ghost"] is False
    page.locator("[data-scene='vale']").click()
    page.locator("[data-pieces='4']").click()
    expect(page.locator("[data-board] use")).to_have_count(0)


def test_em_ingles(page: Page) -> None:
    _open(page, locale="en")
    expect(page.get_by_role("heading", name="Puzzle Pals")).to_be_visible()
    expect(page.get_by_text("Dinosaurs")).to_be_visible()
    page.locator("[data-theme='dinos']").click()
    page.locator("[data-scene='vale']").click()
    expect(page.get_by_text("How many pieces?")).to_be_visible()
