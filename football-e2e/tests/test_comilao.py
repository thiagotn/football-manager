"""
E2E — Comilão (games.thiagotn.com, app `games/`).

Mesmo molde do `test_cobrinha.py`: roda em `GAMES_BASE_URL` (padrão: o build do
`games/` servido em http://localhost:4173), é 100% client-side e guarda tudo em
`localStorage` (`games_comilao`).

As regras (comer, susto, combo, morrer, virar de fase) são cobertas pelo vitest
do engine; aqui testamos a casca: abas, entrada do jogador, partida salva, fim
de partida, persistência e PWA. Teste E2E sobre canvas mede pixel, não regra.

    cd games && npm run build && (cd build && python3 -m http.server 4173)
    cd football-e2e && pytest tests/test_comilao.py
"""

import json
import os
import urllib.request

import pytest
from playwright.sync_api import Page, expect

GAMES_BASE_URL = os.getenv("GAMES_BASE_URL", "http://localhost:4173").rstrip("/")
STORAGE_KEY = "games_comilao"

DEFAULT_SETTINGS = {"pad": True, "glow": True, "pace": "normal", "skin": "synth"}


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


def _state(stats: dict | None = None, **extra) -> dict:
    """Estado salvo completo, com os ajustes padrão."""
    base = {
        "game": None,
        "stats": {"games": 0, "total": 0, "best": 0, "bestLevel": 0, "pellets": 0, "ghosts": 0},
        "history": [],
        "daily": {},
        "settings": dict(DEFAULT_SETTINGS),
    }
    if stats:
        base["stats"].update(stats)
    base.update(extra)
    return base


def _open(page: Page, state: dict | None = None) -> Page:
    """Abre o Comilão em pt-BR, com o localStorage limpo (ou com `state`).

    O estado é gravado a partir do hub porque a tela do jogo salva ao sair. O
    idioma é fixado porque os seletores abaixo são os textos em português.
    """
    page.goto(f"{GAMES_BASE_URL}/")
    page.evaluate(
        "([k, s]) => { localStorage.clear(); localStorage.setItem('games_locale', 'pt-BR');"
        " if (s) localStorage.setItem(k, s); }",
        [STORAGE_KEY, json.dumps(state) if state else None],
    )
    page.goto(f"{GAMES_BASE_URL}/comilao/")
    return page


def _today(page: Page) -> str:
    return page.evaluate(
        "() => { const d = new Date(); const p = (n) => String(n).padStart(2, '0');"
        " return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`; }"
    )


@pytest.fixture
def comilao(page: Page) -> Page:
    """Primeira visita: sem recorde nem visual liberado."""
    return _open(page)


def _score(page: Page) -> int:
    return int(page.locator("span.text-\\[34px\\]").inner_text().replace(" ", "").replace(" ", ""))


def _start(page: Page) -> None:
    page.get_by_role("button", name="Nova partida").click()
    expect(page.get_by_text("Prepare-se…")).to_be_visible()
    # A partida só começa depois da pausa do "Prepare-se".
    expect(page.get_by_text("Prepare-se…")).to_have_count(0, timeout=5_000)


def _patch_game(page: Page, patch: str) -> None:
    """Começa uma partida, sai (o que a salva), mexe no estado salvo e retoma.

    É como os testes chegam depressa a situações que levariam minutos de jogo —
    um fantasma colado, a última vida, o susto valendo.
    """
    page.get_by_role("button", name="Nova partida").click()
    page.wait_for_timeout(200)
    page.get_by_label("Sair").click()
    expect(page.get_by_role("button", name="Continuar")).to_be_visible()
    page.evaluate(
        "(patch) => {"
        f" const s = JSON.parse(localStorage.getItem('{STORAGE_KEY}'));"
        " new Function('g', patch)(s.game);"
        f" localStorage.setItem('{STORAGE_KEY}', JSON.stringify(s)); }}",
        patch,
    )
    page.reload()
    page.get_by_role("button", name="Continuar").click()


# ── Hub e abertura ────────────────────────────────────────────────────────────


def test_hub_mostra_o_card_do_comilao(page: Page):
    page.goto(f"{GAMES_BASE_URL}/")
    page.evaluate("() => localStorage.setItem('games_locale', 'pt-BR')")
    page.reload()
    card = page.locator("a[href='/comilao/']")
    expect(card).to_be_visible()
    expect(card.get_by_text("Comilão", exact=True)).to_be_visible()
    # O selo "Novo" anda de jogo em jogo — agora está aqui.
    expect(card.get_by_text("Novo", exact=True)).to_be_visible()
    expect(card.get_by_text("Arcade")).to_be_visible()


def test_hub_leva_ao_jogo(page: Page):
    page.goto(f"{GAMES_BASE_URL}/")
    page.evaluate("() => localStorage.setItem('games_locale', 'pt-BR')")
    page.reload()
    page.locator("a[href='/comilao/']").click()
    expect(page.get_by_role("button", name="Nova partida")).to_be_visible()


def test_inicio_mostra_desafio_do_dia_e_resumo(comilao: Page):
    expect(comilao.get_by_text("Desafio de hoje")).to_be_visible()
    expect(comilao.get_by_text("0 partidas")).to_be_visible()


def test_as_quatro_abas_abrem(comilao: Page):
    for aba, marca in [
        ("Diário", "Concluídos"),
        ("Visuais", "Quem é quem"),
        ("Ajustes", "Ritmo dos fantasmas"),
        ("Início", "Desafio de hoje"),
    ]:
        comilao.get_by_role("button", name=aba, exact=True).click()
        expect(comilao.get_by_text(marca)).to_be_visible()


def test_volta_para_o_hub(comilao: Page):
    comilao.get_by_role("link", name="Todos os jogos").click()
    expect(comilao.get_by_role("heading", name="Jogos")).to_be_visible()


# ── Partida ───────────────────────────────────────────────────────────────────


def test_a_partida_comeca_com_o_prepare_se(comilao: Page):
    comilao.get_by_role("button", name="Nova partida").click()
    expect(comilao.get_by_text("Prepare-se…")).to_be_visible()
    # Durante a pausa ninguém anda: o placar continua zerado.
    assert _score(comilao) == 0
    expect(comilao.get_by_text("Fase")).to_be_visible()
    expect(comilao.get_by_text("Melhor")).to_be_visible()


def test_andar_come_pastilhas_e_pontua(comilao: Page):
    _start(comilao)
    comilao.get_by_role("button", name="Para baixo", exact=True).click()
    comilao.wait_for_timeout(900)
    assert _score(comilao) > 0


def test_o_direcional_vira_o_comilao(comilao: Page):
    _start(comilao)
    comilao.get_by_role("button", name="Para baixo", exact=True).click()
    comilao.wait_for_timeout(500)
    comilao.get_by_role("button", name="Para a direita", exact=True).click()
    comilao.wait_for_timeout(700)
    assert _score(comilao) >= 20


def test_pausa_segura_a_partida(comilao: Page):
    _start(comilao)
    comilao.get_by_role("button", name="Para baixo", exact=True).click()
    comilao.wait_for_timeout(600)
    comilao.get_by_role("button", name="Pausar").click()
    expect(comilao.get_by_text("Pausado")).to_be_visible()

    parado = _score(comilao)
    comilao.wait_for_timeout(1500)
    assert _score(comilao) == parado

    comilao.get_by_role("button", name="Continuar").click()
    expect(comilao.get_by_text("Pausado")).to_have_count(0)


def test_sair_guarda_a_partida_e_retomar_cai_no_prepare_se(comilao: Page):
    _start(comilao)
    comilao.get_by_role("button", name="Para baixo", exact=True).click()
    comilao.wait_for_timeout(600)
    comilao.get_by_label("Sair").click()

    expect(comilao.get_by_text("Partida em andamento")).to_be_visible()
    assert _saved(comilao)["game"] is not None

    comilao.get_by_role("button", name="Continuar").click()
    # Ninguém volta no colo de um fantasma: retomar sempre pausa antes.
    expect(comilao.get_by_text("Prepare-se…")).to_be_visible()


def test_fim_de_partida_mostra_partida_contra_de_sempre(comilao: Page):
    # Última vida e um fantasma colado: a partida acaba em segundos.
    _patch_game(comilao, "g.lives = 1; g.score = 1500; g.pellets = 150;"
                         " g.ghosts[0] = {...g.ghosts[0], state: 'solto', x: g.hero.x, y: g.hero.y, wait: 0};")
    expect(comilao.get_by_text("Fim de jogo")).to_be_visible(timeout=15_000)
    expect(comilao.get_by_text("Pegaram você!")).to_be_visible()
    expect(comilao.get_by_text("Esta partida")).to_be_visible()
    expect(comilao.get_by_text("De sempre")).to_be_visible()
    expect(comilao.get_by_text("Novo recorde!")).to_be_visible()

    saved = _saved(comilao)
    assert saved["stats"]["games"] == 1
    assert saved["stats"]["best"] >= 1500
    assert len(saved["history"]) == 1
    # A partida encerrada não fica guardada para retomar.
    assert saved["game"] is None


def test_comer_fantasma_assustado_paga_200(comilao: Page):
    _patch_game(comilao, "g.frightLeft = 7000; g.combo = 0;"
                         " g.ghosts[0] = {...g.ghosts[0], state: 'assustado', x: g.hero.x, y: g.hero.y, wait: 0};")
    expect(comilao.get_by_text("Prepare-se…")).to_have_count(0, timeout=5_000)
    comilao.wait_for_timeout(300)
    assert _score(comilao) >= 200


def test_jogar_de_novo_recomeca(comilao: Page):
    _patch_game(comilao, "g.lives = 1;"
                         " g.ghosts[0] = {...g.ghosts[0], state: 'solto', x: g.hero.x, y: g.hero.y, wait: 0};")
    expect(comilao.get_by_text("Fim de jogo")).to_be_visible(timeout=15_000)
    comilao.get_by_role("button", name="Jogar de novo").click()
    expect(comilao.get_by_text("Fim de jogo")).to_have_count(0)
    expect(comilao.get_by_text("Prepare-se…")).to_be_visible()


def test_sair_do_fim_volta_para_o_inicio(comilao: Page):
    _patch_game(comilao, "g.lives = 1;"
                         " g.ghosts[0] = {...g.ghosts[0], state: 'solto', x: g.hero.x, y: g.hero.y, wait: 0};")
    expect(comilao.get_by_text("Fim de jogo")).to_be_visible(timeout=15_000)
    comilao.locator(".card").get_by_role("button", name="Sair").click()
    expect(comilao.get_by_role("button", name="Nova partida")).to_be_visible()


# ── Desafio do dia ────────────────────────────────────────────────────────────


def test_desafio_do_dia_mostra_a_meta(comilao: Page):
    comilao.get_by_role("button", name="Jogar", exact=True).click()
    expect(comilao.get_by_text("Desafio do dia")).to_be_visible()
    # A meta aparece como "0/<meta>" na barra de progresso.
    expect(comilao.get_by_text("0/", exact=False)).to_be_visible()


def test_desafio_do_dia_pelo_calendario(comilao: Page):
    comilao.get_by_role("button", name="Diário", exact=True).click()
    comilao.get_by_role("button", name="Jogar o desafio de hoje").click()
    expect(comilao.get_by_text("Desafio do dia")).to_be_visible()


def test_o_dia_concluido_aparece_no_inicio_e_no_diario(page: Page):
    _open(page)
    today = _today(page)
    _open(page, _state({"games": 3, "total": 4200, "best": 1800}, daily={today: {"best": 1800, "done": True}}))
    expect(page.get_by_text("Concluído com 1 800 pontos")).to_be_visible()

    page.get_by_role("button", name="Diário", exact=True).click()
    expect(page.get_by_text("Concluídos")).to_be_visible()
    expect(page.get_by_text("Sequência")).to_be_visible()
    # Um dia concluído: um único quadrado verde no calendário.
    assert page.locator("button.bg-\\[\\#1d5c46\\]").count() == 1


# ── Visuais ───────────────────────────────────────────────────────────────────


def test_visual_bloqueado_nao_pode_ser_escolhido(comilao: Page):
    comilao.get_by_role("button", name="Visuais", exact=True).click()
    expect(comilao.get_by_text("Próximo: Vapor")).to_be_visible()
    expect(comilao.get_by_role("button", name="Vapor", exact=False)).to_be_disabled()
    assert _saved(comilao) is None or _saved(comilao)["settings"]["skin"] == "synth"


def test_visual_liberado_pela_pontuacao_pode_ser_usado(page: Page):
    _open(page, _state({"games": 20, "total": 15000, "best": 3000, "bestLevel": 4}))
    page.get_by_role("button", name="Visuais", exact=True).click()
    page.get_by_role("button", name="Gelo", exact=False).click()
    expect(page.get_by_text("Em uso")).to_be_visible()
    assert _saved(page)["settings"]["skin"] == "gelo"


def test_a_aba_visuais_explica_quem_e_quem(comilao: Page):
    comilao.get_by_role("button", name="Visuais", exact=True).click()
    for nome in ["Caçador", "Emboscador", "Flanqueador", "Ermitão"]:
        expect(comilao.get_by_text(nome, exact=True)).to_be_visible()
    expect(comilao.get_by_text("Vai direto em você e não alivia.")).to_be_visible()


# ── Ajustes ───────────────────────────────────────────────────────────────────


def test_ajustes_ficam_salvos(comilao: Page):
    comilao.get_by_role("button", name="Ajustes", exact=True).click()
    comilao.get_by_text("Brilho neon").click()
    comilao.get_by_role("button", name="Veloz", exact=True).click()

    saved = _saved(comilao)
    assert saved["settings"]["glow"] is False
    assert saved["settings"]["pace"] == "veloz"


def test_esconder_o_direcional(comilao: Page):
    comilao.get_by_role("button", name="Ajustes", exact=True).click()
    comilao.get_by_text("Mostrar o direcional").click()
    comilao.get_by_role("button", name="Início", exact=True).click()
    comilao.get_by_role("button", name="Nova partida").click()
    expect(comilao.get_by_role("button", name="Para baixo", exact=True)).to_have_count(0)


def test_zerar_estatisticas(page: Page):
    _open(
        page,
        _state(
            {"games": 9, "total": 20000, "best": 4200, "bestLevel": 6, "pellets": 900, "ghosts": 20},
            history=[{"at": 1, "score": 4200, "level": 6, "pellets": 300, "ghosts": 8, "daily": None}],
            settings={**DEFAULT_SETTINGS, "skin": "gelo"},
        ),
    )
    expect(page.get_by_text("4 200 recorde")).to_be_visible()

    page.get_by_role("button", name="Ajustes", exact=True).click()
    page.get_by_role("button", name="Zerar estatísticas").click()
    page.locator(".card").get_by_role("button", name="Zerar estatísticas").click()

    saved = _saved(page)
    assert saved["stats"]["games"] == 0
    assert saved["stats"]["total"] == 0
    # O visual volta ao padrão, senão ficaria um bloqueado em uso.
    assert saved["settings"]["skin"] == "synth"


# ── PWA ───────────────────────────────────────────────────────────────────────


def test_manifesto_do_jogo(page: Page):
    page.goto(f"{GAMES_BASE_URL}/comilao/")
    href = page.locator("link[rel=manifest]").get_attribute("href")
    assert href == "/comilao/manifest.webmanifest"

    manifest = json.loads(urllib.request.urlopen(GAMES_BASE_URL + href, timeout=5).read())
    assert manifest["id"] == "/comilao/"
    assert manifest["scope"] == "/comilao/"
    assert manifest["start_url"] == "/comilao/"
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
    assert "/comilao/" in cached

    context.set_offline(True)
    try:
        page.reload()
        expect(page.get_by_role("button", name="Nova partida")).to_be_visible()
    finally:
        context.set_offline(False)
