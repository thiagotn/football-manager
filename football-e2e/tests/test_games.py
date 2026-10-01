"""
E2E — Passatempo (`/games`) e Paciência (`/paciencia`).

Ambas as rotas são públicas e 100% client-side: a Paciência não fala com a API,
guarda tudo em `localStorage` (chave `rachao_paciencia`) e o desafio diário usa
um deal determinístico por data. Os testes cobrem a navegação entre as telas, o
ciclo de uma partida e a persistência dos ajustes.
"""

import re

import pytest
from playwright.sync_api import Page, expect

STORAGE_KEY = "rachao_paciencia"


@pytest.fixture
def paciencia(page: Page) -> Page:
    """Abre a Paciência com o localStorage limpo (estado de primeira visita)."""
    page.goto("/paciencia")
    page.evaluate(f"() => localStorage.removeItem('{STORAGE_KEY}')")
    page.reload()
    expect(page.get_by_role("button", name="Novo jogo")).to_be_visible()
    return page


def _saved_game(page: Page) -> dict | None:
    return page.evaluate(
        f"() => {{ const s = localStorage.getItem('{STORAGE_KEY}');"
        f" return s ? JSON.parse(s).game : null; }}"
    )


# ── /games ────────────────────────────────────────────────────────────────────


def test_games_lista_os_dois_jogos(page: Page):
    page.goto("/games")
    expect(page.get_by_role("heading", name="Passatempo")).to_be_visible()
    expect(page.get_by_role("link", name="Paciência")).to_have_attribute("href", "/paciencia")
    expect(page.get_by_role("link", name="Tetris 3D")).to_have_attribute("href", "/tetris")


def test_games_abre_a_paciencia(page: Page):
    page.goto("/games")
    page.get_by_role("link", name="Paciência").click()
    expect(page).to_have_url(re.compile(r"/paciencia$"))
    expect(page.get_by_role("button", name="Novo jogo")).to_be_visible()


# ── /paciencia ────────────────────────────────────────────────────────────────


def test_inicio_mostra_o_desafio_do_dia(paciencia: Page):
    expect(paciencia.get_by_text("Desafio do dia")).to_be_visible()
    expect(paciencia.get_by_text("Mesma distribuição para todos")).to_be_visible()
    # Sem partida em andamento não existe o botão "Continuar"
    expect(paciencia.get_by_role("button", name="Continuar")).to_have_count(0)


def test_voltar_para_games(paciencia: Page):
    paciencia.get_by_role("link", name="Passatempo").click()
    expect(paciencia).to_have_url(re.compile(r"/games$"))


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

    paciencia.get_by_role("button", name="Campanha").click()
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


def test_partida_abandonada_entra_no_historico(paciencia: Page):
    paciencia.get_by_role("button", name="Novo jogo").click()
    paciencia.get_by_role("button", name="Comprar carta do monte").click()
    paciencia.get_by_role("button", name="Novo", exact=True).click()

    # As abas só aparecem fora da mesa — volta pelo cabeçalho do jogo
    paciencia.get_by_role("button", name="Voltar ao início do jogo").click()
    paciencia.get_by_role("button", name="Campanha").click()
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


def _seed(page: Page, tableau: list[list[dict]], waste: list[dict] | None = None) -> None:
    """Semeia uma mesa e entra nela, esperando o layout assentar."""
    game = {
        "stock": [],
        "waste": waste or [],
        "found": [[], [], [], []],
        "tableau": tableau + [[]] * (7 - len(tableau)),
        "moves": 0,
        "score": 0,
        "elapsed": 0,
        "won": False,
        "draw": 1,
        "daily": None,
    }
    page.evaluate(
        "(g) => localStorage.setItem('rachao_paciencia', JSON.stringify({"
        " game: g, history: [], daily: {},"
        " settings: { draw: 1, timer: true, autoFinish: false, back: 'escudo' } }))",
        game,
    )
    page.reload()
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


def test_arrasta_carta_do_descarte_para_o_tableau(paciencia: Page):
    _seed(paciencia, [[K_SPADES]], waste=[Q_HEARTS])
    _drag(paciencia, "Q de copas", onto="K de espadas")

    game = _saved_game(paciencia)
    assert [c["r"] for c in game["tableau"][0]] == [13, 12]
    assert game["waste"] == []
    assert game["moves"] == 1
    assert game["score"] == 5  # descarte -> tableau


def test_arrasta_para_a_fundacao(paciencia: Page):
    _seed(paciencia, [[K_SPADES]], waste=[A_SPADES])
    box = paciencia.get_by_role("button", name="A de espadas").bounding_box()
    paciencia.mouse.move(box["x"] + box["width"] / 2, box["y"] + 10)
    paciencia.mouse.down()
    paciencia.mouse.move(box["x"] + 40, box["y"] + 30, steps=5)
    paciencia.mouse.move(box["x"] + 160, box["y"], steps=8)  # coluna 3 = fundação ♠
    paciencia.mouse.up()
    paciencia.wait_for_timeout(350)

    game = _saved_game(paciencia)
    assert len(game["found"][0]) == 1
    assert game["score"] == 10


def test_arrasta_sequencia_a_partir_do_meio_da_coluna(paciencia: Page):
    _seed(paciencia, [[NINE_SPADES, SIX_HEARTS, FIVE_CLUBS], [SEVEN_SPADES]])
    _drag(paciencia, "6 de copas", onto="7 de espadas")

    game = _saved_game(paciencia)
    assert [c["r"] for c in game["tableau"][0]] == [9]
    assert [c["r"] for c in game["tableau"][1]] == [7, 6, 5]
    assert game["moves"] == 1


def test_soltar_em_lugar_invalido_nao_move_a_carta(paciencia: Page):
    _seed(paciencia, [[NINE_SPADES, SIX_HEARTS, FIVE_CLUBS]])
    # Coluna vazia só aceita rei — a sequência 6♥-5♣ volta para o lugar
    _drag(paciencia, "6 de copas", dx=120, dy=40)

    game = _saved_game(paciencia)
    assert [c["r"] for c in game["tableau"][0]] == [9, 6, 5]
    assert game["tableau"][1] == []
    assert game["moves"] == 0


def test_toque_continua_movendo_depois_do_arraste(paciencia: Page):
    _seed(paciencia, [[NINE_SPADES, SIX_HEARTS, FIVE_CLUBS]])
    _drag(paciencia, "6 de copas", dx=120, dy=40)  # arraste inválido
    assert _saved_game(paciencia)["moves"] == 0

    _seed(paciencia, [[K_SPADES]], waste=[Q_HEARTS])
    paciencia.get_by_role("button", name="Q de copas").click()
    paciencia.wait_for_timeout(250)

    game = _saved_game(paciencia)
    assert [c["r"] for c in game["tableau"][0]] == [13, 12]
    assert game["moves"] == 1
