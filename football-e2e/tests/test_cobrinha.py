"""
E2E — Cobrinha (games.thiagotn.com, app `games/`).

Mesmo molde do `test_paciencia.py`: roda em `GAMES_BASE_URL` (padrão: o build do
`games/` servido em http://localhost:4173), é 100% client-side e guarda tudo em
`localStorage` (`games_cobrinha`).

As regras (crescer, morrer, pontuar) são cobertas pelo vitest do engine; aqui
testamos a casca: abas, entrada do jogador, fim de partida, persistência e PWA.

    cd games && npm run build && npx vite preview --port 4173
    cd football-e2e && pytest tests/test_cobrinha.py
"""

import json
import os
import urllib.request

import pytest
from playwright.sync_api import Page, expect

GAMES_BASE_URL = os.getenv("GAMES_BASE_URL", "http://localhost:4173").rstrip("/")
STORAGE_KEY = "games_cobrinha"


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
    """Abre a Cobrinha em pt-BR, com o localStorage limpo (ou com `state`).

    O estado é gravado a partir do hub porque a tela do jogo salva ao sair. O
    idioma é fixado porque os seletores abaixo são os textos em português.
    """
    page.goto(f"{GAMES_BASE_URL}/")
    page.evaluate(
        "([k, s]) => { localStorage.clear(); localStorage.setItem('games_locale', 'pt-BR');"
        " if (s) localStorage.setItem(k, s); }",
        [STORAGE_KEY, json.dumps(state) if state else None],
    )
    page.goto(f"{GAMES_BASE_URL}/cobrinha/")
    return page


def _today(page: Page) -> str:
    return page.evaluate(
        "() => { const d = new Date(); const p = (n) => String(n).padStart(2, '0');"
        " return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`; }"
    )


@pytest.fixture
def cobrinha(page: Page) -> Page:
    """Primeira visita: sem recorde nem visual liberado."""
    return _open(page)


def _start_free_game(page: Page) -> None:
    page.get_by_role("button", name="Nova partida").click()
    expect(page.get_by_text("Arraste ou use o direcional")).to_be_visible()


def _hold(page: Page, label: str, times: int, pause_ms: int = 260) -> None:
    """Aperta um botão do direcional algumas vezes, dando tempo de a cobra andar."""
    for _ in range(times):
        page.get_by_role("button", name=label, exact=True).click()
        page.wait_for_timeout(pause_ms)


# ── Hub e abertura ────────────────────────────────────────────────────────────


def test_hub_mostra_o_card_da_cobrinha(page: Page):
    page.goto(f"{GAMES_BASE_URL}/")
    page.evaluate("() => localStorage.setItem('games_locale', 'pt-BR')")
    page.reload()
    card = page.locator("a[href='/cobrinha/']")
    expect(card).to_be_visible()
    expect(card.get_by_text("Cobrinha", exact=True)).to_be_visible()
    # O selo "Novo" anda de jogo em jogo (agora está no Sudoku) — quem testa é o dono da vez.
    expect(card.get_by_text("Clássico")).to_be_visible()


def test_hub_leva_ao_jogo(page: Page):
    page.goto(f"{GAMES_BASE_URL}/")
    page.evaluate("() => localStorage.setItem('games_locale', 'pt-BR')")
    page.reload()
    page.locator("a[href='/cobrinha/']").click()
    expect(page.get_by_role("button", name="Nova partida")).to_be_visible()


def test_inicio_mostra_desafio_do_dia_e_resumo(cobrinha: Page):
    expect(cobrinha.get_by_text("Desafio de hoje")).to_be_visible()
    expect(cobrinha.get_by_text("0 partidas")).to_be_visible()


def test_volta_para_o_hub(cobrinha: Page):
    cobrinha.get_by_role("link", name="Todos os jogos").click()
    expect(cobrinha.get_by_role("heading", name="Jogos")).to_be_visible()


# ── Partida ───────────────────────────────────────────────────────────────────


def test_a_cobra_so_anda_depois_do_primeiro_comando(cobrinha: Page):
    """Sem isso o jogador perderia antes de pegar no controle."""
    _start_free_game(cobrinha)
    cobrinha.wait_for_timeout(1200)
    # Nada aconteceu: o aviso continua e o placar está zerado.
    expect(cobrinha.get_by_text("Arraste ou use o direcional")).to_be_visible()
    expect(cobrinha.get_by_text("Fim de jogo")).to_have_count(0)

    cobrinha.get_by_role("button", name="Para baixo", exact=True).click()
    expect(cobrinha.get_by_text("Arraste ou use o direcional")).to_have_count(0)


def test_bater_na_parede_encerra_a_partida_e_salva(cobrinha: Page):
    _start_free_game(cobrinha)
    # Reto para baixo: são 9 linhas até a borda de baixo.
    cobrinha.get_by_role("button", name="Para baixo", exact=True).click()
    expect(cobrinha.get_by_text("Fim de jogo")).to_be_visible(timeout=10_000)
    expect(cobrinha.get_by_text("Jogo atual")).to_be_visible()
    expect(cobrinha.get_by_text("De sempre")).to_be_visible()

    saved = _saved(cobrinha)
    assert saved["stats"]["games"] == 1
    assert len(saved["history"]) == 1
    assert saved["history"][0]["daily"] is None


def test_jogar_de_novo_recomeca_a_partida(cobrinha: Page):
    _start_free_game(cobrinha)
    cobrinha.get_by_role("button", name="Para baixo", exact=True).click()
    expect(cobrinha.get_by_text("Fim de jogo")).to_be_visible(timeout=10_000)

    cobrinha.get_by_role("button", name="Jogar de novo").click()
    expect(cobrinha.get_by_text("Fim de jogo")).to_have_count(0)
    expect(cobrinha.get_by_text("Arraste ou use o direcional")).to_be_visible()


def test_sair_volta_para_o_inicio(cobrinha: Page):
    _start_free_game(cobrinha)
    cobrinha.get_by_role("button", name="Para baixo", exact=True).click()
    expect(cobrinha.get_by_text("Fim de jogo")).to_be_visible(timeout=10_000)
    cobrinha.locator(".card").get_by_role("button", name="Sair").click()
    expect(cobrinha.get_by_role("button", name="Nova partida")).to_be_visible()


def test_pausa_e_continua(cobrinha: Page):
    _start_free_game(cobrinha)
    cobrinha.get_by_role("button", name="Para cima", exact=True).click()
    cobrinha.get_by_role("button", name="Pausar").click()
    expect(cobrinha.get_by_text("Pausado")).to_be_visible()

    # Pausado, a cobra não anda: ela continua viva depois de bem mais tempo
    # do que levaria para cruzar o tabuleiro.
    cobrinha.wait_for_timeout(2500)
    expect(cobrinha.get_by_text("Fim de jogo")).to_have_count(0)

    cobrinha.get_by_role("button", name="Continuar").click()
    expect(cobrinha.get_by_text("Pausado")).to_have_count(0)


# ── Desafio do dia ────────────────────────────────────────────────────────────


def test_desafio_do_dia_mostra_a_meta(cobrinha: Page):
    cobrinha.get_by_role("button", name="Jogar", exact=True).click()
    expect(cobrinha.get_by_text("Desafio do dia")).to_be_visible()
    # A meta aparece como "0/<meta>" na barra de progresso.
    expect(cobrinha.get_by_text("0/", exact=False)).to_be_visible()


def test_o_dia_concluido_aparece_no_inicio_e_no_diario(page: Page):
    _open(page)
    today = _today(page)
    _open(
        page,
        {
            "stats": {"games": 3, "total": 600, "best": 260, "bestFruits": 9, "bestLevel": 2},
            "history": [],
            "daily": {today: {"best": 260, "done": True}},
            "settings": {"wrap": False, "pad": True, "speed": "normal", "skin": "folha"},
        },
    )
    expect(page.get_by_text("Concluído com 260 pontos")).to_be_visible()

    page.get_by_role("button", name="Diário", exact=True).click()
    # Um dia concluído vira um de sequência.
    expect(page.get_by_text("Concluídos")).to_be_visible()
    expect(page.get_by_text("Sequência")).to_be_visible()
    assert page.locator("button.bg-\\[\\#1b5e3a\\]").count() == 1


def test_desafio_do_dia_pelo_calendario(cobrinha: Page):
    cobrinha.get_by_role("button", name="Diário", exact=True).click()
    cobrinha.get_by_role("button", name="Jogar o desafio de hoje").click()
    expect(cobrinha.get_by_text("Desafio do dia")).to_be_visible()


# ── Visuais ───────────────────────────────────────────────────────────────────


def test_visual_bloqueado_nao_pode_ser_escolhido(cobrinha: Page):
    cobrinha.get_by_role("button", name="Visuais", exact=True).click()
    expect(cobrinha.get_by_text("Próximo: Brasa")).to_be_visible()
    brasa = cobrinha.get_by_role("button", name="Brasa", exact=False)
    expect(brasa).to_be_disabled()
    assert _saved(cobrinha) is None or _saved(cobrinha)["settings"]["skin"] == "folha"


def test_visual_liberado_pela_pontuacao_pode_ser_usado(page: Page):
    _open(
        page,
        {
            "stats": {"games": 20, "total": 4000, "best": 500, "bestFruits": 20, "bestLevel": 5},
            "history": [],
            "daily": {},
            "settings": {"wrap": False, "pad": True, "speed": "normal", "skin": "folha"},
        },
    )
    page.get_by_role("button", name="Visuais", exact=True).click()
    page.get_by_role("button", name="Gelo", exact=False).click()
    expect(page.get_by_text("Em uso")).to_be_visible()
    assert _saved(page)["settings"]["skin"] == "gelo"


# ── Ajustes ───────────────────────────────────────────────────────────────────


def test_ajustes_ficam_salvos(cobrinha: Page):
    cobrinha.get_by_role("button", name="Ajustes", exact=True).click()
    cobrinha.get_by_text("Atravessar paredes").click()
    cobrinha.get_by_role("button", name="Veloz").click()

    saved = _saved(cobrinha)
    assert saved["settings"]["wrap"] is True
    assert saved["settings"]["speed"] == "veloz"


def test_esconder_o_direcional(cobrinha: Page):
    cobrinha.get_by_role("button", name="Ajustes", exact=True).click()
    cobrinha.get_by_text("Mostrar o direcional").click()
    cobrinha.get_by_role("button", name="Início", exact=True).click()
    _start_free_game(cobrinha)
    expect(cobrinha.get_by_role("button", name="Para baixo", exact=True)).to_have_count(0)


def test_zerar_estatisticas(page: Page):
    _open(
        page,
        {
            "stats": {"games": 9, "total": 9000, "best": 820, "bestFruits": 30, "bestLevel": 7},
            "history": [{"at": 1, "score": 820, "fruits": 30, "level": 7, "daily": None}],
            "daily": {},
            "settings": {"wrap": False, "pad": True, "speed": "normal", "skin": "neon"},
        },
    )
    expect(page.get_by_text("820 recorde")).to_be_visible()

    page.get_by_role("button", name="Ajustes", exact=True).click()
    page.get_by_role("button", name="Zerar estatísticas").click()
    page.locator(".card").get_by_role("button", name="Zerar estatísticas").click()

    saved = _saved(page)
    assert saved["stats"]["games"] == 0
    assert saved["stats"]["total"] == 0
    # O visual volta ao padrão, senão ficaria um bloqueado em uso.
    assert saved["settings"]["skin"] == "folha"


# ── PWA ───────────────────────────────────────────────────────────────────────


def test_manifesto_do_jogo(page: Page):
    page.goto(f"{GAMES_BASE_URL}/cobrinha/")
    href = page.locator("link[rel=manifest]").get_attribute("href")
    assert href == "/cobrinha/manifest.webmanifest"

    manifest = json.loads(urllib.request.urlopen(GAMES_BASE_URL + href, timeout=5).read())
    assert manifest["id"] == "/cobrinha/"
    assert manifest["scope"] == "/cobrinha/"
    assert manifest["start_url"] == "/cobrinha/"
    assert len(manifest["icons"]) == 3


def test_offline_depois_do_service_worker(page: Page, context):
    _open(page)
    page.wait_for_function("() => navigator.serviceWorker.controller !== null", timeout=45_000)
    cached = page.evaluate(
        """async () => {
            const key = (await caches.keys())[0];
            const reqs = await (await caches.open(key)).keys();
            return reqs.map((r) => new URL(r.url).pathname);
        }"""
    )
    assert "/cobrinha/" in cached

    context.set_offline(True)
    try:
        page.reload()
        expect(page.get_by_role("button", name="Nova partida")).to_be_visible()
    finally:
        context.set_offline(False)
