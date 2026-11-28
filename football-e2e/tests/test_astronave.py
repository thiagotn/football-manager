"""
E2E — Astronave (games.thiagotn.com, app `games/`).

Mesmo molde do `test_cobrinha.py`: roda em `GAMES_BASE_URL` (padrão: o build do
`games/` servido em http://localhost:4173), é 100% client-side e guarda tudo em
`localStorage` (`games_astronave`).

As regras (tiros, colisões, ondas) são cobertas pelo vitest do engine; aqui
testamos a casca: abas, os dois modos de controle, fim de partida, persistência
e PWA. O que acontece dentro do canvas chega ao teste pelos `data-*` da arena
(`data-ship-x`, `data-fired`, `data-started`).

    cd games && npm run build && npx vite preview --port 4173
    cd football-e2e && pytest tests/test_astronave.py
"""

import json
import os
import urllib.request

import pytest
from playwright.sync_api import Page, expect

GAMES_BASE_URL = os.getenv("GAMES_BASE_URL", "http://localhost:4173").rstrip("/")
STORAGE_KEY = "games_astronave"
PHONE = {"width": 412, "height": 803}

START_TOUCH = "Arraste para mover · segure para atirar"
START_PAD = "Use os botões para mover e atirar"


def _reachable() -> bool:
    try:
        urllib.request.urlopen(GAMES_BASE_URL + "/", timeout=3)
        return True
    except Exception:
        return False


pytestmark = pytest.mark.skipif(not _reachable(), reason=f"games app fora do ar em {GAMES_BASE_URL}")


def _saved(page: Page) -> dict | None:
    return page.evaluate(
        f"() => {{ const s = localStorage.getItem('{STORAGE_KEY}'); return s ? JSON.parse(s) : null; }}"
    )


def _open(page: Page, state: dict | None = None) -> Page:
    """Abre a Astronave em pt-BR, num viewport de celular, com o localStorage limpo (ou com `state`)."""
    page.set_viewport_size(PHONE)
    page.goto(f"{GAMES_BASE_URL}/")
    page.evaluate(
        "([k, s]) => { localStorage.clear(); localStorage.setItem('games_locale', 'pt-BR');"
        " if (s) localStorage.setItem(k, s); }",
        [STORAGE_KEY, json.dumps(state) if state else None],
    )
    page.goto(f"{GAMES_BASE_URL}/astronave/")
    return page


def _state(controls: str = "touch", **over) -> dict:
    base = {
        "stats": {"games": 0, "total": 0, "best": 0, "bestWave": 0, "bestKills": 0},
        "history": [],
        "daily": {},
        "settings": {"controls": controls, "skin": "pioneira"},
    }
    base.update(over)
    return base


def _today(page: Page) -> str:
    return page.evaluate(
        "() => { const d = new Date(); const p = (n) => String(n).padStart(2, '0');"
        " return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`; }"
    )


@pytest.fixture
def astronave(page: Page) -> Page:
    """Primeira visita: sem recorde, só toque."""
    return _open(page)


def _board(page: Page):
    return page.locator("[data-board]")


def _ship_x(page: Page) -> int:
    return int(_board(page).get_attribute("data-ship-x") or "-1")


def _fired(page: Page) -> int:
    return int(_board(page).get_attribute("data-fired") or "0")


def _start_free_game(page: Page) -> None:
    page.get_by_role("button", name="Nova partida").click()
    expect(_board(page)).to_have_attribute("data-ship-x", "50")


# ── Hub e abertura ────────────────────────────────────────────────────────────


def test_hub_mostra_o_card_com_selo_novo(page: Page):
    page.goto(f"{GAMES_BASE_URL}/")
    page.evaluate("() => localStorage.setItem('games_locale', 'pt-BR')")
    page.reload()
    card = page.locator("a[href='/astronave/']")
    expect(card).to_be_visible()
    expect(card.get_by_text("Astronave", exact=True)).to_be_visible()
    expect(card.get_by_text("Arcade", exact=True)).to_be_visible()
    expect(card.get_by_text("Novo", exact=True)).to_be_visible()


def test_hub_leva_ao_jogo(page: Page):
    page.goto(f"{GAMES_BASE_URL}/")
    page.evaluate("() => localStorage.setItem('games_locale', 'pt-BR')")
    page.reload()
    page.locator("a[href='/astronave/']").click()
    expect(page.get_by_role("button", name="Nova partida")).to_be_visible()


def test_inicio_mostra_desafio_do_dia_e_resumo(astronave: Page):
    expect(astronave.get_by_text("Desafio de hoje")).to_be_visible()
    expect(astronave.get_by_text("0 partidas")).to_be_visible()


def test_volta_para_o_hub(astronave: Page):
    astronave.get_by_role("link", name="Todos os jogos").click()
    expect(astronave.get_by_role("heading", name="Jogos")).to_be_visible()


def test_manifesto_tem_escopo_proprio(astronave: Page):
    href = astronave.locator("link[rel='manifest']").get_attribute("href")
    assert href == "/astronave/manifest.webmanifest"
    manifest = json.loads(urllib.request.urlopen(GAMES_BASE_URL + href, timeout=5).read())
    assert manifest["scope"] == "/astronave/"
    assert manifest["start_url"] == "/astronave/"


# ── Partida: só toque ─────────────────────────────────────────────────────────


def test_nada_se_mexe_antes_do_primeiro_comando(astronave: Page):
    """Sem isso o jogador perderia uma vida antes de pegar no controle."""
    _start_free_game(astronave)
    expect(astronave.get_by_text(START_TOUCH)).to_be_visible()
    astronave.wait_for_timeout(1500)
    expect(_board(astronave)).to_have_attribute("data-started", "false")
    expect(astronave.get_by_test_id("lives")).to_have_attribute("data-lives", "3")
    expect(astronave.get_by_test_id("score")).to_have_text("0")


def test_so_toque_nao_tem_botoes_e_a_arena_ocupa_a_tela(astronave: Page):
    _start_free_game(astronave)
    expect(_board(astronave)).to_have_attribute("data-controls", "touch")
    expect(astronave.locator("[data-pad]")).to_have_count(0)
    expect(astronave.get_by_role("button", name="Atirar")).to_have_count(0)

    box = astronave.locator("canvas").bounding_box()
    assert box
    # Num celular de 412 × 803 a arena pega quase a largura toda e vai até o pé da tela.
    assert box["width"] >= PHONE["width"] * 0.92
    assert box["y"] + box["height"] >= PHONE["height"] * 0.95


def test_arrastar_move_a_nave_e_segurar_atira(astronave: Page):
    _start_free_game(astronave)
    box = astronave.locator("canvas").bounding_box()
    assert box
    # O dedo fica no alto da arena, longe da nave: o arraste é relativo.
    x, y = box["x"] + box["width"] / 2, box["y"] + box["height"] * 0.3
    astronave.mouse.move(x, y)
    astronave.mouse.down()
    expect(astronave.get_by_text(START_TOUCH)).to_have_count(0)
    astronave.mouse.move(x + box["width"] * 0.3, y, steps=6)
    astronave.wait_for_timeout(700)
    assert _ship_x(astronave) >= 75
    assert _fired(astronave) >= 2

    # Soltou: para de atirar.
    astronave.mouse.up()
    astronave.wait_for_timeout(300)
    fired = _fired(astronave)
    astronave.wait_for_timeout(500)
    assert _fired(astronave) == fired


def test_teclado_move_e_atira(astronave: Page):
    _start_free_game(astronave)
    astronave.keyboard.down("ArrowLeft")
    astronave.wait_for_timeout(500)
    astronave.keyboard.up("ArrowLeft")
    assert _ship_x(astronave) < 40
    assert _fired(astronave) == 0

    astronave.keyboard.down("Space")
    astronave.wait_for_timeout(600)
    astronave.keyboard.up("Space")
    assert _fired(astronave) >= 2


# ── Partida: botões ───────────────────────────────────────────────────────────


def test_escolher_botoes_nos_ajustes_fica_salvo(astronave: Page):
    astronave.get_by_role("button", name="Ajustes", exact=True).click()
    expect(astronave.get_by_role("button", name="Só toque")).to_have_attribute("aria-pressed", "true")
    astronave.get_by_role("button", name="Botões", exact=True).click()
    expect(astronave.get_by_role("button", name="Botões", exact=True)).to_have_attribute("aria-pressed", "true")
    saved = _saved(astronave)
    assert saved["settings"]["controls"] == "pad"


def test_botoes_movem_e_atiram(page: Page):
    _open(page, _state("pad"))
    _start_free_game(page)
    expect(page.get_by_text(START_PAD)).to_be_visible()
    expect(_board(page)).to_have_attribute("data-controls", "pad")

    right = page.get_by_role("button", name="Direita", exact=True)
    fire = page.get_by_role("button", name="Atirar", exact=True)
    expect(right).to_be_visible()
    expect(page.get_by_role("button", name="Esquerda", exact=True)).to_be_visible()

    def hold(button, ms: int) -> None:
        box = button.bounding_box()
        assert box
        page.mouse.move(box["x"] + box["width"] / 2, box["y"] + box["height"] / 2)
        page.mouse.down()
        page.wait_for_timeout(ms)
        page.mouse.up()

    hold(right, 500)
    assert _ship_x(page) > 60
    assert _fired(page) == 0

    hold(fire, 600)
    assert _fired(page) >= 2


def test_com_botoes_a_arena_encolhe_para_caber_tudo(page: Page):
    _open(page, _state("pad"))
    _start_free_game(page)
    canvas = page.locator("canvas").bounding_box()
    pad = page.locator("[data-pad]").bounding_box()
    assert canvas and pad
    # A arena termina antes dos botões, e os botões cabem inteiros na tela.
    assert canvas["y"] + canvas["height"] <= pad["y"] + 1
    assert pad["y"] + pad["height"] <= PHONE["height"]


# ── Pausa ─────────────────────────────────────────────────────────────────────


def test_pausa_congela_e_deixa_trocar_os_controles(astronave: Page):
    _start_free_game(astronave)
    astronave.keyboard.press("ArrowLeft")
    astronave.get_by_role("button", name="Pausar").click()
    expect(astronave.get_by_text("Pausado")).to_be_visible()

    # Pausado, nada anda: as três vidas continuam lá bem depois do primeiro meteoro.
    astronave.wait_for_timeout(2500)
    expect(astronave.get_by_test_id("lives")).to_have_attribute("data-lives", "3")

    astronave.get_by_role("button", name="Botões", exact=True).click()
    astronave.get_by_role("button", name="Continuar").click()
    expect(astronave.get_by_text("Pausado")).to_have_count(0)
    expect(astronave.get_by_role("button", name="Atirar", exact=True)).to_be_visible()
    assert _saved(astronave)["settings"]["controls"] == "pad"


# ── Fim de partida ────────────────────────────────────────────────────────────


def test_fim_de_partida_salva_e_deixa_jogar_de_novo(astronave: Page):
    """Parada e sem atirar, a nave é abatida em menos de um minuto (o vitest garante)."""
    _start_free_game(astronave)
    astronave.keyboard.press("ArrowLeft")
    expect(astronave.get_by_text("Fim de jogo")).to_be_visible(timeout=90_000)
    expect(astronave.get_by_text("Jogo atual")).to_be_visible()
    expect(astronave.get_by_text("De sempre")).to_be_visible()
    expect(astronave.get_by_test_id("lives")).to_have_attribute("data-lives", "0")

    saved = _saved(astronave)
    assert saved["stats"]["games"] == 1
    assert len(saved["history"]) == 1
    assert saved["history"][0]["daily"] is None
    assert saved["history"][0]["wave"] >= 1

    astronave.get_by_role("button", name="Jogar de novo").click()
    expect(astronave.get_by_text("Fim de jogo")).to_have_count(0)
    expect(astronave.get_by_text(START_TOUCH)).to_be_visible()
    expect(astronave.get_by_test_id("lives")).to_have_attribute("data-lives", "3")

    astronave.get_by_role("button", name="Sair").click()
    expect(astronave.get_by_role("button", name="Nova partida")).to_be_visible()
    expect(astronave.get_by_text("1 partidas")).to_be_visible()


# ── Desafio do dia ────────────────────────────────────────────────────────────


def test_desafio_do_dia_mostra_a_meta(astronave: Page):
    astronave.get_by_role("button", name="Jogar", exact=True).click()
    expect(astronave.get_by_text("Desafio do dia")).to_be_visible()
    # A barra da meta: "0/400" a "0/800".
    expect(astronave.get_by_text("0/", exact=False)).to_be_visible()


def test_dia_concluido_aparece_no_inicio_e_no_calendario(page: Page):
    _open(page)
    today = _today(page)
    _open(page, _state(daily={today: {"best": 950, "done": True}}))
    expect(page.get_by_text("Concluído com 950 pontos")).to_be_visible()
    expect(page.get_by_text("1 dia seguido")).to_be_visible()

    page.get_by_role("button", name="Diário", exact=True).click()
    expect(page.get_by_text("Melhor do dia")).to_be_visible()
    expect(page.get_by_text("950", exact=True)).to_be_visible()


# ── Naves ─────────────────────────────────────────────────────────────────────


def test_naves_bloqueadas_no_comeco(astronave: Page):
    astronave.get_by_role("button", name="Naves", exact=True).click()
    expect(astronave.get_by_text("Próxima: Cometa")).to_be_visible()
    expect(astronave.get_by_text("Em uso")).to_be_visible()
    expect(astronave.get_by_role("button", name="Cometa")).to_be_disabled()


def test_nave_liberada_pode_ser_escolhida(page: Page):
    _open(
        page,
        _state(stats={"games": 5, "total": 2000, "best": 700, "bestWave": 4, "bestKills": 12}),
    )
    page.get_by_role("button", name="Naves", exact=True).click()
    page.get_by_role("button", name="Brasa").click()
    assert _saved(page)["settings"]["skin"] == "brasa"
    expect(page.get_by_role("button", name="Gelo")).to_be_disabled()


# ── Ajustes ───────────────────────────────────────────────────────────────────


def test_zerar_estatisticas(page: Page):
    _open(
        page,
        _state(
            "pad",
            stats={"games": 9, "total": 9000, "best": 820, "bestWave": 6, "bestKills": 30},
            history=[{"at": 1, "score": 820, "wave": 6, "kills": 30, "daily": None}],
            settings={"controls": "pad", "skin": "neon"},
        ),
    )
    expect(page.get_by_text("820 recorde")).to_be_visible()

    page.get_by_role("button", name="Ajustes", exact=True).click()
    page.get_by_role("button", name="Zerar estatísticas").click()
    page.locator(".overlay").get_by_role("button", name="Zerar estatísticas").click()

    saved = _saved(page)
    assert saved["stats"]["games"] == 0
    assert saved["history"] == []
    # Os controles são preferência, não estatística; a nave volta à padrão.
    assert saved["settings"] == {"controls": "pad", "skin": "pioneira"}
