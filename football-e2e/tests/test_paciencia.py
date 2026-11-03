"""
E2E — Paciência (games.thiagotn.com, app `games/`).

Portado de `test_games.py` quando o jogo morava no rachao.app (PRD 054). Roda
em `GAMES_BASE_URL` (padrão: `vite preview` do `games/` em http://localhost:4173),
é 100% client-side e guarda tudo em `localStorage` (`games_paciencia`).

    cd games && npm run build && npx vite preview --port 4173
    cd football-e2e && pytest tests/test_paciencia.py
"""

import json
import os
import urllib.request

import pytest
from playwright.sync_api import Page, expect

GAMES_BASE_URL = os.getenv("GAMES_BASE_URL", "http://localhost:4173").rstrip("/")
STORAGE_KEY = "games_paciencia"


def _reachable() -> bool:
    try:
        urllib.request.urlopen(GAMES_BASE_URL + "/", timeout=3)
        return True
    except Exception:
        return False


pytestmark = pytest.mark.skipif(not _reachable(), reason=f"games app fora do ar em {GAMES_BASE_URL}")


def _saved_game(page: Page) -> dict | None:
    return page.evaluate(
        f"() => {{ const s = localStorage.getItem('{STORAGE_KEY}');"
        f" return s ? JSON.parse(s).game : null; }}"
    )


def _open(page: Page, state: dict | None = None) -> Page:
    """Abre a Paciência em pt-BR, com o localStorage limpo (ou com `state`).

    O estado é gravado a partir do hub: a tela do jogo salva ao sair, e
    sobrescreveria o que fosse injetado com ela já aberta. O idioma é fixado
    porque os seletores abaixo são os textos em português.
    """
    page.goto(f"{GAMES_BASE_URL}/")
    page.evaluate(
        "([k, s]) => { localStorage.clear(); localStorage.setItem('games_locale', 'pt-BR');"
        " if (s) localStorage.setItem(k, s); }",
        [STORAGE_KEY, json.dumps(state) if state else None],
    )
    page.goto(f"{GAMES_BASE_URL}/paciencia/")
    return page


@pytest.fixture
def paciencia(page: Page) -> Page:
    """Primeira visita: sem partida salva."""
    _open(page)
    expect(page.get_by_role("button", name="Novo jogo")).to_be_visible()
    return page


# ── Hub e PWA ─────────────────────────────────────────────────────────────────


def test_hub_linka_a_paciencia(page: Page):
    _open(page)  # fixa games_locale e volta ao hub: o runner do CI roda em inglês
    page.goto(f"{GAMES_BASE_URL}/")
    card = page.get_by_role("link", name="Paciência")
    expect(card).to_have_attribute("href", "/paciencia/")
    card.click()
    expect(page.get_by_role("button", name="Novo jogo")).to_be_visible()


def test_manifesto_da_paciencia_tem_escopo_proprio(page: Page):
    page.goto(f"{GAMES_BASE_URL}/paciencia/")
    href = page.get_attribute("link[rel=manifest]", "href")
    assert href == "/paciencia/manifest.webmanifest"

    manifest = page.evaluate("async (url) => (await fetch(url)).json()", GAMES_BASE_URL + href)
    assert manifest["id"] == "/paciencia/"
    assert manifest["scope"] == "/paciencia/"
    assert manifest["start_url"] == "/paciencia/"
    assert {icon["sizes"] for icon in manifest["icons"]} == {"192x192", "512x512"}


def test_volta_para_o_hub(paciencia: Page):
    paciencia.get_by_role("link", name="Todos os jogos").click()
    expect(paciencia.get_by_role("heading", name="Jogos")).to_be_visible()


# ── Partida ───────────────────────────────────────────────────────────────────


def test_inicio_mostra_o_desafio_do_dia(paciencia: Page):
    expect(paciencia.get_by_text("Desafio do dia")).to_be_visible()
    expect(paciencia.get_by_text("Mesma distribuição para todos")).to_be_visible()
    # Sem partida em andamento não existe o botão "Continuar"
    expect(paciencia.get_by_role("button", name="Continuar")).to_have_count(0)


def test_novo_jogo_distribui_as_cartas(paciencia: Page):
    paciencia.get_by_role("button", name="Novo jogo").click()

    game = _saved_game(paciencia)
    assert game is not None
    assert [len(col) for col in game["tableau"]] == [1, 2, 3, 4, 5, 6, 7]
    assert len(game["stock"]) == 24
    assert all(len(f) == 0 for f in game["found"])
    # Só a última carta de cada coluna começa virada para cima
    assert all(col[-1]["up"] and not any(c["up"] for c in col[:-1]) for col in game["tableau"])


def test_compra_do_monte_conta_jogada_e_liga_o_cronometro(paciencia: Page):
    paciencia.get_by_role("button", name="Novo jogo").click()
    paciencia.get_by_role("button", name="Comprar carta do monte").click()

    assert _saved_game(paciencia)["moves"] == 1
    assert len(_saved_game(paciencia)["waste"]) == 1
    # O tempo só começa a correr a partir da primeira jogada
    paciencia.wait_for_timeout(2500)
    assert _saved_game(paciencia)["elapsed"] >= 1


def test_desfazer_volta_o_estado_anterior(paciencia: Page):
    paciencia.get_by_role("button", name="Novo jogo").click()
    paciencia.get_by_role("button", name="Comprar carta do monte").click()
    assert len(_saved_game(paciencia)["waste"]) == 1

    paciencia.get_by_role("button", name="Desfazer").click()
    game = _saved_game(paciencia)
    assert game["waste"] == []
    assert len(game["stock"]) == 24
    assert game["moves"] == 2  # desfazer também conta como jogada


def test_partida_em_andamento_sobrevive_ao_reload(paciencia: Page):
    paciencia.get_by_role("button", name="Novo jogo").click()
    paciencia.get_by_role("button", name="Comprar carta do monte").click()
    before = _saved_game(paciencia)["tableau"]

    paciencia.reload()
    expect(paciencia.get_by_role("button", name="Continuar")).to_be_visible()
    assert _saved_game(paciencia)["tableau"] == before


def test_desafio_diario_e_deterministico(paciencia: Page):
    paciencia.get_by_role("button", name="Jogar", exact=True).click()
    first = [c["id"] for col in _saved_game(paciencia)["tableau"] for c in col]
    daily_date = _saved_game(paciencia)["daily"]
    assert daily_date is not None

    # Rejogar o mesmo dia distribui exatamente as mesmas cartas
    paciencia.get_by_role("button", name="Novo", exact=True).click()
    second = [c["id"] for col in _saved_game(paciencia)["tableau"] for c in col]
    assert second == first
    assert _saved_game(paciencia)["daily"] == daily_date


def test_navegacao_entre_as_abas(paciencia: Page):
    paciencia.get_by_role("button", name="Diário").click()
    expect(paciencia.get_by_text("Uma distribuição por dia", exact=False)).to_be_visible()

    paciencia.get_by_role("button", name="Partidas").click()
    expect(paciencia.get_by_text("Nenhuma partida ainda", exact=False)).to_be_visible()

    paciencia.get_by_role("button", name="Ajustes").click()
    expect(paciencia.get_by_text("Uniforme das cartas")).to_be_visible()

    paciencia.get_by_role("button", name="Início").click()
    expect(paciencia.get_by_role("button", name="Novo jogo")).to_be_visible()


def test_ajuste_de_compra_vale_do_proximo_jogo(paciencia: Page):
    paciencia.get_by_role("button", name="Ajustes").click()
    paciencia.get_by_role("button", name="3 cartas").click()
    paciencia.get_by_role("button", name="Início").click()
    paciencia.get_by_role("button", name="Novo jogo").click()

    assert _saved_game(paciencia)["draw"] == 3
    paciencia.get_by_role("button", name="Comprar carta do monte").click()
    assert len(_saved_game(paciencia)["waste"]) == 3


def test_troca_de_idioma_nos_ajustes(paciencia: Page):
    paciencia.get_by_role("button", name="Ajustes").click()
    paciencia.get_by_role("button", name="EN", exact=True).click()
    expect(paciencia.get_by_text("Card kit")).to_be_visible()


def test_partida_abandonada_entra_no_historico(paciencia: Page):
    paciencia.get_by_role("button", name="Novo jogo").click()
    paciencia.get_by_role("button", name="Comprar carta do monte").click()
    paciencia.get_by_role("button", name="Novo", exact=True).click()

    # As abas só aparecem fora da mesa — volta pelo cabeçalho do jogo
    paciencia.get_by_role("button", name="Voltar ao início do jogo").click()
    paciencia.get_by_role("button", name="Partidas").click()
    expect(paciencia.get_by_text("Abandonou")).to_be_visible()


# ── Arrastar e soltar ─────────────────────────────────────────────────────────
#
# O arraste convive com o toque: só vira arraste depois de ~6px de movimento, e
# o clique que o navegador dispara ao fim do gesto é descartado. Os cenários
# abaixo semeiam mesas determinísticas para não depender da sorte do baralho.

K_SPADES = {"id": 13, "s": 0, "r": 13, "up": True}
Q_HEARTS = {"id": 25, "s": 1, "r": 12, "up": True}
A_SPADES = {"id": 1, "s": 0, "r": 1, "up": True}
NINE_SPADES = {"id": 9, "s": 0, "r": 9, "up": True}
SIX_HEARTS = {"id": 19, "s": 1, "r": 6, "up": True}
FIVE_CLUBS = {"id": 44, "s": 3, "r": 5, "up": True}
SEVEN_SPADES = {"id": 7, "s": 0, "r": 7, "up": True}


def _state(tableau: list[list[dict]], waste: list[dict] | None = None, found=None) -> dict:
    return {
        "game": {
            "stock": [],
            "waste": waste or [],
            "found": found or [[], [], [], []],
            "tableau": tableau + [[]] * (7 - len(tableau)),
            "moves": 0,
            "score": 0,
            "elapsed": 0,
            "won": False,
            "draw": 1,
            "daily": None,
        },
        "history": [],
        "daily": {},
        "settings": {"draw": 1, "timer": True, "autoFinish": False, "back": "classico"},
    }


def _seed(page: Page, tableau: list[list[dict]], waste: list[dict] | None = None) -> None:
    """Semeia uma mesa e entra nela, esperando o layout assentar."""
    _open(page, _state(tableau, waste))
    page.get_by_role("button", name="Continuar").click()
    # As posições das cartas dependem da medição da mesa; sem isso o arraste
    # usaria coordenadas de um layout que ainda vai mudar.
    page.wait_for_timeout(400)


def _drag(page: Page, card: str, onto: str | None = None, dx: int = 0, dy: int = 0) -> None:
    """Arrasta `card` até a carta `onto` (ou por um deslocamento livre)."""
    box = page.get_by_role("button", name=card).bounding_box()
    page.mouse.move(box["x"] + box["width"] / 2, box["y"] + 10)
    page.mouse.down()
    page.mouse.move(box["x"] + box["width"] / 2 + 15, box["y"] + 40, steps=5)
    if onto:
        target = page.get_by_role("button", name=onto).bounding_box()
        page.mouse.move(target["x"] + target["width"] / 2, target["y"] + 40, steps=8)
    else:
        page.mouse.move(box["x"] + dx, box["y"] + dy, steps=8)
    page.mouse.up()
    page.wait_for_timeout(350)


def test_arrasta_carta_do_descarte_para_o_tableau(page: Page):
    _seed(page, [[K_SPADES]], waste=[Q_HEARTS])
    _drag(page, "Q de copas", onto="K de espadas")

    game = _saved_game(page)
    assert [c["r"] for c in game["tableau"][0]] == [13, 12]
    assert game["waste"] == []
    assert game["moves"] == 1
    assert game["score"] == 5  # descarte -> tableau


def test_arrasta_para_a_fundacao(page: Page):
    _seed(page, [[K_SPADES]], waste=[A_SPADES])
    box = page.get_by_role("button", name="A de espadas").bounding_box()
    page.mouse.move(box["x"] + box["width"] / 2, box["y"] + 10)
    page.mouse.down()
    page.mouse.move(box["x"] + 40, box["y"] + 30, steps=5)
    page.mouse.move(box["x"] + 160, box["y"], steps=8)  # coluna 3 = fundação ♠
    page.mouse.up()
    page.wait_for_timeout(350)

    game = _saved_game(page)
    assert len(game["found"][0]) == 1
    assert game["score"] == 10


def test_arrasta_sequencia_a_partir_do_meio_da_coluna(page: Page):
    _seed(page, [[NINE_SPADES, SIX_HEARTS, FIVE_CLUBS], [SEVEN_SPADES]])
    _drag(page, "6 de copas", onto="7 de espadas")

    game = _saved_game(page)
    assert [c["r"] for c in game["tableau"][0]] == [9]
    assert [c["r"] for c in game["tableau"][1]] == [7, 6, 5]
    assert game["moves"] == 1


def test_soltar_em_lugar_invalido_nao_move_a_carta(page: Page):
    _seed(page, [[NINE_SPADES, SIX_HEARTS, FIVE_CLUBS]])
    # Coluna vazia só aceita rei — a sequência 6♥-5♣ volta para o lugar
    _drag(page, "6 de copas", dx=120, dy=40)

    game = _saved_game(page)
    assert [c["r"] for c in game["tableau"][0]] == [9, 6, 5]
    assert game["tableau"][1] == []
    assert game["moves"] == 0


def test_toque_continua_movendo_depois_do_arraste(page: Page):
    _seed(page, [[NINE_SPADES, SIX_HEARTS, FIVE_CLUBS]])
    _drag(page, "6 de copas", dx=120, dy=40)  # arraste inválido
    assert _saved_game(page)["moves"] == 0

    _seed(page, [[K_SPADES]], waste=[Q_HEARTS])
    page.get_by_role("button", name="Q de copas").click()
    page.wait_for_timeout(250)

    game = _saved_game(page)
    assert [c["r"] for c in game["tableau"][0]] == [13, 12]
    assert game["moves"] == 1


# ── Vitória ───────────────────────────────────────────────────────────────────


def test_vitoria_mostra_as_estrelas(page: Page):
    """Mesa a uma jogada do fim: a última carta fecha o jogo."""
    # Copas, ouros e paus completos; espadas só até a dama, com o rei na mesa.
    full = [[], [], [], []]
    for suit in range(4):
        last = 12 if suit == 0 else 13
        for rank in range(1, last + 1):
            full[suit].append({"id": suit * 13 + rank, "s": suit, "r": rank, "up": True})
    king = {"id": 13, "s": 0, "r": 13, "up": True}

    _seed_state = _state([[king]], found=full)
    _open(page, _seed_state)
    page.get_by_role("button", name="Continuar").click()
    page.wait_for_timeout(400)
    page.get_by_role("button", name="K de espadas").click()
    page.wait_for_timeout(400)

    expect(page.get_by_text("Fechou!")).to_be_visible()
    assert _saved_game(page)["won"] is True
