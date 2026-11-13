"""
E2E — Sudoku (games.thiagotn.com, app `games/`).

Mesmo molde do `test_cobrinha.py`: roda em `GAMES_BASE_URL` (padrão: o build do
`games/` servido em http://localhost:4173), é 100% client-side e guarda tudo em
`localStorage` (`games_sudoku`).

As regras (gerar, deduzir, ganhar) são cobertas pelo vitest do engine; aqui
testamos a casca: abas, entrada do jogador, partida retomada, fim de partida,
persistência e PWA.

    cd games && npm run build && npx vite preview --port 4173
    cd football-e2e && pytest tests/test_sudoku.py
"""

import json
import os
import urllib.request

import pytest
from playwright.sync_api import Page, expect

GAMES_BASE_URL = os.getenv("GAMES_BASE_URL", "http://localhost:4173").rstrip("/")
STORAGE_KEY = "games_sudoku"


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
    """Abre o Sudoku em pt-BR, com o localStorage limpo (ou com `state`).

    O estado é gravado a partir do hub porque a tela do jogo salva ao entrar. O
    idioma é fixado porque os seletores abaixo são os textos em português.
    """
    page.goto(f"{GAMES_BASE_URL}/")
    page.evaluate(
        "([k, s]) => { localStorage.clear(); localStorage.setItem('games_locale', 'pt-BR');"
        " if (s) localStorage.setItem(k, s); }",
        [STORAGE_KEY, json.dumps(state) if state else None],
    )
    page.goto(f"{GAMES_BASE_URL}/sudoku/")
    return page


def _today(page: Page) -> str:
    return page.evaluate(
        "() => { const d = new Date(); const p = (n) => String(n).padStart(2, '0');"
        " return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`; }"
    )


def _empty_stats() -> dict:
    zero = {"games": 0, "wins": 0, "best": 0, "bestHints": 0, "bestMistakes": 0, "totalTime": 0}
    return {d: dict(zero) for d in ("facil", "medio", "dificil", "especialista")}


@pytest.fixture
def sudoku(page: Page) -> Page:
    """Primeira visita: sem recorde nem visual liberado."""
    return _open(page)


def _start(page: Page, difficulty: str = "Fácil") -> None:
    """Começa uma partida e espera o tabuleiro ficar pronto."""
    page.get_by_role("button", name=difficulty, exact=False).first.click()
    expect(page.locator(".board")).to_be_visible(timeout=15_000)
    expect(page.locator(".cell")).to_have_count(81)


def _first_empty(page: Page):
    return page.locator(".cell:not(.given)").first


def _digit(page: Page, digit: int):
    """Botão do teclado numérico (o rótulo tem o número e quantos faltam)."""
    return page.locator(".grid-cols-9 > button").nth(digit - 1)


# ── Hub e abertura ──────────────────────────────────────────────────────────


def test_hub_mostra_o_card_do_sudoku(page: Page) -> None:
    page.goto(f"{GAMES_BASE_URL}/")
    page.evaluate("() => localStorage.setItem('games_locale', 'pt-BR')")
    page.reload()

    card = page.locator("a[href='/sudoku/']")
    expect(card).to_be_visible()
    expect(card.get_by_text("Sudoku", exact=True)).to_be_visible()
    # O selo "Novo" anda de jogo em jogo (agora está no Comilão) — quem testa é o dono da vez.
    expect(card.get_by_text("Puzzle")).to_be_visible()


def test_hub_leva_ao_jogo(page: Page) -> None:
    page.goto(f"{GAMES_BASE_URL}/")
    page.evaluate("() => localStorage.setItem('games_locale', 'pt-BR')")
    page.reload()
    page.locator("a[href='/sudoku/']").click()

    expect(page).to_have_url(f"{GAMES_BASE_URL}/sudoku/")
    expect(page.get_by_text("Escolha a dificuldade")).to_be_visible()


def test_inicio_lista_as_quatro_dificuldades(sudoku: Page) -> None:
    for difficulty in ("Fácil", "Médio", "Difícil", "Especialista"):
        expect(sudoku.get_by_text(difficulty, exact=True)).to_be_visible()
    # Sem partida salva, não há o que continuar.
    expect(sudoku.get_by_text("Continuar", exact=False)).to_have_count(0)


def test_as_quatro_abas_abrem(sudoku: Page) -> None:
    for tab, marker in (
        ("Diário", "Tabuleiro do dia"),
        ("Visuais", "Papel e tinta"),
        ("Ajustes", "Do seu jeito"),
        ("Início", "Escolha a dificuldade"),
    ):
        sudoku.get_by_role("button", name=tab, exact=True).click()
        expect(sudoku.get_by_text(marker).first).to_be_visible()


# ── Partida ─────────────────────────────────────────────────────────────────


def test_comecar_um_facil_monta_o_tabuleiro(sudoku: Page) -> None:
    _start(sudoku)
    expect(sudoku.locator(".cell.given")).not_to_have_count(0)
    # O relógio só anda depois do primeiro lance.
    expect(sudoku.get_by_text("0:00")).to_be_visible()


def test_linha_sem_numeros_tem_a_mesma_altura(sudoku: Page) -> None:
    """Regressão: a altura da linha seguia o conteúdo e uma linha vazia encolhia."""
    _start(sudoku)
    heights = sudoku.evaluate(
        """() => {
          const cells = [...document.querySelectorAll('.board .cell')];
          for (let i = 9; i < 18; i++) cells[i].replaceChildren();  // linha 2 sem nada
          return [...Array(9)].map((_, r) => cells[r * 9].getBoundingClientRect().height);
        }"""
    )
    assert max(heights) - min(heights) < 1, heights


def test_escrever_um_numero_na_casa_escolhida(sudoku: Page) -> None:
    _start(sudoku)
    cell = _first_empty(sudoku)
    cell.click()
    _digit(sudoku, 5).click()

    expect(cell).to_have_text("5")
    assert _saved(sudoku)["game"]["values"].count(0) < 81


def test_sem_casa_escolhida_o_jogo_avisa(sudoku: Page) -> None:
    _start(sudoku)
    _digit(sudoku, 5).click()
    expect(sudoku.get_by_text("Toque numa casa antes de escolher o número.")).to_be_visible()


def test_o_mesmo_numero_duas_vezes_apaga(sudoku: Page) -> None:
    _start(sudoku)
    cell = _first_empty(sudoku)
    cell.click()
    _digit(sudoku, 4).click()
    expect(cell).to_have_text("4")
    _digit(sudoku, 4).click()
    expect(cell).to_have_text("")


def test_lapis_escreve_anotacao_e_desfazer_volta(sudoku: Page) -> None:
    _start(sudoku)
    cell = _first_empty(sudoku)
    cell.click()
    sudoku.get_by_role("button", name="Lápis").click()
    _digit(sudoku, 2).click()
    _digit(sudoku, 7).click()

    expect(cell.locator(".notes")).to_be_visible()
    expect(cell).to_contain_text("2")
    expect(cell).to_contain_text("7")

    sudoku.get_by_role("button", name="Desfazer").click()
    expect(cell).not_to_contain_text("7")


def test_apagar_limpa_a_casa(sudoku: Page) -> None:
    _start(sudoku)
    cell = _first_empty(sudoku)
    cell.click()
    _digit(sudoku, 6).click()
    expect(cell).to_have_text("6")

    cell.click()
    sudoku.get_by_role("button", name="Apagar").click()
    expect(cell).to_have_text("")


def test_numero_repetido_na_linha_fica_marcado(sudoku: Page) -> None:
    _start(sudoku)
    # Repete numa casa vazia o dígito de um dado da mesma linha.
    repeated = sudoku.evaluate(
        """() => {
          const cells = [...document.querySelectorAll('.cell')];
          for (let i = 0; i < 81; i++) {
            if (!cells[i].classList.contains('given')) continue;
            const row = Math.floor(i / 9);
            for (let c = 0; c < 9; c++) {
              const j = row * 9 + c;
              if (!cells[j].classList.contains('given') && !cells[j].textContent.trim()) {
                return { empty: j, digit: Number(cells[i].textContent.trim()) };
              }
            }
          }
          return null;
        }"""
    )
    assert repeated, "o tabuleiro deveria ter uma linha com dado e casa vazia"

    sudoku.locator(".cell").nth(repeated["empty"]).click()
    _digit(sudoku, repeated["digit"]).click()

    cor = sudoku.locator(".cell").nth(repeated["empty"]).evaluate("el => getComputedStyle(el).color")
    assert cor == "rgb(215, 52, 42)", f"esperava a tinta de conflito, veio {cor}"


def test_a_dica_preenche_uma_casa_e_explica(sudoku: Page) -> None:
    _start(sudoku)
    before = sudoku.locator(".cell:not(.given)").evaluate_all(
        "els => els.filter((el) => el.textContent.trim()).length"
    )
    sudoku.get_by_role("button", name="Dica").click()

    after = sudoku.locator(".cell:not(.given)").evaluate_all(
        "els => els.filter((el) => el.textContent.trim()).length"
    )
    assert after == before + 1
    expect(sudoku.locator(".toast")).to_be_visible()
    # O HUD conta a ajuda — e no singular, não "1 dicas".
    expect(sudoku.get_by_text("1 dica", exact=True)).to_be_visible()


def test_pausar_mostra_o_overlay_e_volta(sudoku: Page) -> None:
    _start(sudoku)
    sudoku.get_by_role("button", name="Pausar").click()
    expect(sudoku.get_by_text("Pausado")).to_be_visible()

    sudoku.get_by_role("button", name="Voltar ao jogo").click()
    expect(sudoku.get_by_text("Pausado")).to_have_count(0)


def test_sair_guarda_a_partida_e_o_inicio_oferece_continuar(sudoku: Page) -> None:
    _start(sudoku)
    _first_empty(sudoku).click()
    _digit(sudoku, 1).click()
    sudoku.get_by_role("button", name="Sair").click()

    expect(sudoku.get_by_role("button", name="Continuar", exact=False)).to_be_visible()
    assert _saved(sudoku)["game"] is not None

    sudoku.reload()
    sudoku.get_by_role("button", name="Continuar", exact=False).click()
    expect(sudoku.locator(".board")).to_be_visible()


def test_ganhar_mostra_a_tabela_e_salva_o_recorde(sudoku: Page) -> None:
    _start(sudoku)
    # Resolver 50 casas na mão é inviável no E2E: preenche pela solução salva.
    sudoku.evaluate(
        """() => {
          const cells = [...document.querySelectorAll('.cell')];
          const saved = JSON.parse(localStorage.getItem('games_sudoku'));
          return saved.game.solution.map((d, i) => [i, d, cells[i].classList.contains('given')]);
        }"""
    )
    plan = sudoku.evaluate(
        """() => {
          const saved = JSON.parse(localStorage.getItem('games_sudoku'));
          const g = saved.game;
          return g.values.map((v, i) => (v ? null : [i, g.solution[i]])).filter(Boolean);
        }"""
    )
    for at, digit in plan:
        sudoku.locator(".cell").nth(at).click()
        _digit(sudoku, digit).click()

    expect(sudoku.get_by_text("Tabuleiro fechado!")).to_be_visible(timeout=10_000)
    expect(sudoku.get_by_text("Partida", exact=True)).to_be_visible()
    expect(sudoku.get_by_text("De sempre", exact=True)).to_be_visible()
    expect(sudoku.get_by_text("Novo recorde")).to_be_visible()

    saved = _saved(sudoku)
    assert saved["stats"]["facil"]["wins"] == 1
    assert saved["game"] is None
    assert len(saved["history"]) == 1


# ── Tabuleiro do dia ────────────────────────────────────────────────────────


def test_diario_mostra_o_calendario_do_mes(sudoku: Page) -> None:
    sudoku.get_by_role("button", name="Diário", exact=True).click()
    expect(sudoku.get_by_text("Um tabuleiro por dia")).to_be_visible()
    # Sete cabeçalhos de dia da semana + os dias do mês.
    expect(sudoku.locator(".grid-cols-7 > *")).not_to_have_count(0)
    expect(sudoku.get_by_role("button", name="Jogar o de hoje")).to_be_visible()


def test_dia_concluido_aparece_marcado_e_conta_a_sequencia(page: Page) -> None:
    today = _today(page)
    state = {
        "game": None,
        "stats": _empty_stats(),
        "history": [],
        "daily": {today: {"done": True, "time": 254, "hints": 1}},
        "settings": {
            "errors": True,
            "highlight": True,
            "autoNotes": True,
            "numberFirst": False,
            "timer": True,
            "theme": "jornal",
        },
    }
    _open(page, state)

    expect(page.get_by_text("Resolvido em 4:14")).to_be_visible()
    page.get_by_role("button", name="Diário", exact=True).click()
    expect(page.locator("button.bg-\\[\\#1b5e3a\\]")).to_have_count(1)


def test_o_tabuleiro_do_dia_abre_pelo_cartao(sudoku: Page) -> None:
    sudoku.get_by_role("button", name="Jogar", exact=True).click()
    expect(sudoku.locator(".board")).to_be_visible(timeout=20_000)
    expect(sudoku.get_by_text("Do dia")).to_be_visible()


# ── Visuais ─────────────────────────────────────────────────────────────────


def test_visual_bloqueado_mostra_quantas_vitorias_faltam(sudoku: Page) -> None:
    sudoku.get_by_role("button", name="Visuais", exact=True).click()
    expect(sudoku.get_by_text("Em uso")).to_be_visible()
    # "5 vitórias" também casa dentro de "15 vitórias": o item do Linho é quem manda.
    expect(sudoku.locator("li", has_text="Linho").get_by_text("5 vitórias", exact=True)).to_be_visible()
    expect(sudoku.get_by_text("Falta pouco para o Linho")).to_be_visible()


def test_visual_liberado_pode_ser_usado_e_fica_salvo(page: Page) -> None:
    stats = _empty_stats()
    stats["facil"] = {"games": 9, "wins": 9, "best": 200, "bestHints": 0, "bestMistakes": 0, "totalTime": 2000}
    state = {
        "game": None,
        "stats": stats,
        "history": [],
        "daily": {},
        "settings": {
            "errors": True,
            "highlight": True,
            "autoNotes": True,
            "numberFirst": False,
            "timer": True,
            "theme": "jornal",
        },
    }
    _open(page, state)
    page.get_by_role("button", name="Visuais", exact=True).click()
    page.get_by_role("button", name="Usar").first.click()

    expect(page.get_by_text("Em uso").first).to_be_visible()
    assert _saved(page)["settings"]["theme"] == "linho"


# ── Ajustes ─────────────────────────────────────────────────────────────────


def test_ajuste_fica_salvo(sudoku: Page) -> None:
    sudoku.get_by_role("button", name="Ajustes", exact=True).click()
    sudoku.get_by_text("Mostrar o tempo").click()

    assert _saved(sudoku)["settings"]["timer"] is False
    sudoku.reload()
    sudoku.get_by_role("button", name="Ajustes", exact=True).click()
    expect(sudoku.locator("input[type=checkbox]").nth(4)).not_to_be_checked()


def test_numero_primeiro_inverte_a_ordem_do_toque(sudoku: Page) -> None:
    sudoku.get_by_role("button", name="Ajustes", exact=True).click()
    sudoku.get_by_text("Número primeiro").click()
    sudoku.get_by_role("button", name="Início", exact=True).click()
    _start(sudoku)

    _digit(sudoku, 3).click()
    cell = _first_empty(sudoku)
    cell.click()
    expect(cell).to_have_text("3")


def test_zerar_estatisticas_pelo_confirm_dialog(page: Page) -> None:
    stats = _empty_stats()
    stats["medio"] = {"games": 4, "wins": 3, "best": 310, "bestHints": 1, "bestMistakes": 0, "totalTime": 1200}
    state = {
        "game": None,
        "stats": stats,
        "history": [{"at": 1, "difficulty": "medio", "time": 310, "hints": 1, "mistakes": 0, "daily": None}],
        "daily": {},
        "settings": {
            "errors": True,
            "highlight": True,
            "autoNotes": True,
            "numberFirst": False,
            "timer": True,
            "theme": "jornal",
        },
    }
    _open(page, state)
    expect(page.get_by_text("3 resolvidos").first).to_be_visible()

    page.get_by_role("button", name="Ajustes", exact=True).click()
    page.get_by_role("button", name="Zerar estatísticas").click()
    page.locator(".card").get_by_role("button", name="Zerar estatísticas").click()

    assert _saved(page)["stats"]["medio"]["wins"] == 0
    expect(page.get_by_text("Nenhum resolvido ainda").first).to_be_visible()


# ── PWA ─────────────────────────────────────────────────────────────────────


def test_manifesto_do_sudoku(sudoku: Page) -> None:
    href = sudoku.locator("link[rel=manifest]").get_attribute("href")
    assert href == "/sudoku/manifest.webmanifest"

    manifest = json.loads(urllib.request.urlopen(GAMES_BASE_URL + href, timeout=5).read())
    assert manifest["id"] == "/sudoku/"
    assert manifest["scope"] == "/sudoku/"
    assert manifest["start_url"] == "/sudoku/"
    assert len(manifest["icons"]) == 3


def test_service_worker_instala_e_o_jogo_abre_offline(page: Page, context) -> None:
    _open(page)
    page.wait_for_function("() => navigator.serviceWorker.controller !== null", timeout=45_000)

    cached = page.evaluate(
        """async () => {
          const names = await caches.keys();
          const cache = await caches.open(names[0]);
          const keys = await cache.keys();
          return keys.map((r) => new URL(r.url).pathname);
        }"""
    )
    assert "/sudoku/" in cached

    context.set_offline(True)
    try:
        page.reload()
        expect(page.get_by_text("Escolha a dificuldade")).to_be_visible(timeout=15_000)
    finally:
        context.set_offline(False)
