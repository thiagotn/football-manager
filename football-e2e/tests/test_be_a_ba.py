"""
E2E — Bê-á-Bá (games.thiagotn.com/be-a-ba/, app `games/`, PRD 058).

100% client-side: estado em `localStorage` (`games_be_a_ba`), fotos das
palavras da família no IndexedDB. Cada desafio expõe a resposta em
`data-answer` (palavra ou número) e o tipo em `data-mode`/`data-kind`, para o
teste jogar rodadas inteiras sem depender do sorteio.

    cd games && npm run build && npx vite preview --port 4173
    cd football-e2e && pytest tests/test_be_a_ba.py
"""

import json
import os
import re
import struct
import unicodedata
import urllib.request
import zlib

import pytest
from playwright.sync_api import Page, expect

GAMES_BASE_URL = os.getenv("GAMES_BASE_URL", "http://localhost:4173").rstrip("/")
STORAGE_KEY = "games_be_a_ba"
GATE_WORDS = {
    "vinte e sete": 27,
    "quarenta e dois": 42,
    "cinquenta e três": 53,
    "sessenta e oito": 68,
    "setenta e quatro": 74,
    "oitenta e nove": 89,
}


def _reachable() -> bool:
    try:
        urllib.request.urlopen(GAMES_BASE_URL + "/", timeout=3)
        return True
    except Exception:
        return False


pytestmark = pytest.mark.skipif(not _reachable(), reason=f"games app fora do ar em {GAMES_BASE_URL}")


def _plain(s: str) -> str:
    return "".join(c for c in unicodedata.normalize("NFD", s) if unicodedata.category(c) != "Mn")


def _saved(page: Page) -> dict:
    return page.evaluate(f"() => JSON.parse(localStorage.getItem('{STORAGE_KEY}'))")


def _state(**over) -> dict:
    base = {
        "ageMode": "kids",
        "childName": "",
        "family": [],
        "letters": {"stage": 0, "locked": True, "goodRounds": 0, "boxes": {}},
        "numbers": {"stage": 0, "locked": True, "goodRounds": 0, "boxes": {}},
        "little": {
            "letters": {"stage": 0, "locked": True, "goodRounds": 0, "boxes": {}},
            "numbers": {"stage": 0, "locked": True, "goodRounds": 0, "boxes": {}},
        },
        "stickers": [],
        "stars": 0,
        "rounds": 0,
        "playtime": {},
        "settings": {"voice": False, "rate": 0.85},
    }
    for k, v in over.items():
        base[k] = {**base[k], **v} if isinstance(v, dict) and isinstance(base.get(k), dict) else v
    return base


def _little(letters: int = 0, numbers: int = 0, **over) -> dict:
    """Estado no modo Pequenos (3–4 anos), com os estágios fixos."""
    trail = lambda stage: {"stage": stage, "locked": True, "goodRounds": 0, "boxes": {}}  # noqa: E731
    return _state(ageMode="little", little={"letters": trail(letters), "numbers": trail(numbers)}, **over)


def _open_fresh(page: Page) -> Page:
    """Primeira visita de verdade: nada salvo (o jogo pergunta a idade)."""
    page.goto(f"{GAMES_BASE_URL}/")
    page.evaluate("() => { localStorage.clear(); localStorage.setItem('games_locale', 'pt-BR'); }")
    page.goto(f"{GAMES_BASE_URL}/be-a-ba/")
    return page


def _open(page: Page, state: dict | None = None, locale: str = "pt-BR") -> Page:
    """Abre o jogo com `state` (padrão: modo 5–7 anos, fase 1, nada ganho).

    O estado é gravado a partir do hub: a tela do jogo salva ao sair
    (`pagehide`) e sobrescreveria o que fosse injetado com ela aberta.
    """
    page.goto(f"{GAMES_BASE_URL}/")
    page.evaluate(
        "([k, s, l]) => { localStorage.clear(); localStorage.setItem('games_locale', l);"
        " if (s) localStorage.setItem(k, s); }",
        [STORAGE_KEY, json.dumps(state or _state()), locale],
    )
    page.goto(f"{GAMES_BASE_URL}/be-a-ba/")
    expect(page.get_by_role("button", name=re.compile("Números|Numbers"))).to_be_visible()
    return page


def _solve_letter(page: Page) -> str:
    """Resolve o desafio de letras na tela; devolve a palavra."""
    ch = page.locator("[data-mode]")
    mode = ch.get_attribute("data-mode")
    word = ch.get_attribute("data-answer")
    if mode == "first":
        ch.get_by_role("button", name=f"Letra {_plain(word[0])}", exact=True).click()
    elif mode == "build":
        for letter in word:
            ch.locator(".tile:not(.used)", has_text=re.compile(f"^{_plain(letter)}$")).first.click()
    else:
        for letter in word:
            page.keyboard.press(_plain(letter).lower())
    return word


def _solve_number(page: Page) -> str:
    ch = page.locator("[data-kind]")
    answer = ch.get_attribute("data-answer")
    if ch.get_attribute("data-kind") == "compare":
        ch.locator(".side").nth(0 if answer == "left" else 1).click()
    else:
        ch.get_by_role("button", name=answer, exact=True).click()
    return answer


def _solve_little(page: Page) -> str:
    """Resolve o desafio do modo Pequenos na tela; devolve o tipo."""
    ch = page.locator("[data-little-kind]")
    kind = ch.get_attribute("data-little-kind")
    answer = ch.get_attribute("data-answer")
    if kind == "hear":
        ch.locator(f"[data-option='{answer}']").click()
    elif kind == "pick_letter":
        ch.get_by_role("button", name=f"Letra {answer}", exact=True).click()
    elif kind == "tap":
        for obj in ch.locator(".tap-object").all():
            obj.click()
    elif kind == "find_number":
        ch.get_by_role("button", name=answer, exact=True).click()
    else:
        ch.locator(".group").nth(int(answer)).click()
    return kind


def _next(page: Page) -> None:
    """Espera o elogio do acerto aparecer e sumir (o próximo desafio ou o prêmio entra)."""
    expect(page.locator(".praise")).to_be_visible()
    expect(page.locator(".praise")).to_have_count(0, timeout=5000)


def _play_round(page: Page, solve, size: int = 5) -> list[str]:
    answers = []
    for _ in range(size):
        expect(page.locator("[data-answer]")).to_be_visible()
        answers.append(solve(page))
        _next(page)
    expect(page.get_by_role("heading", name=re.compile("Rodada completa|Round complete"))).to_be_visible()
    return answers


def _enter_parents(page: Page) -> None:
    page.get_by_role("button", name="Pais").click()
    dialog = page.get_by_role("dialog", name="Área dos pais")
    ask = dialog.locator("p").first.inner_text()
    n = next(v for k, v in GATE_WORDS.items() if k in ask)
    dialog.get_by_role("button", name=str(n), exact=True).click()
    expect(page.get_by_role("heading", name="Área dos pais")).to_be_visible()


def _png() -> bytes:
    """PNG 4×4 vermelho, sem depender de Pillow."""

    def chunk(kind: bytes, data: bytes) -> bytes:
        return struct.pack(">I", len(data)) + kind + data + struct.pack(">I", zlib.crc32(kind + data))

    raw = b"".join(b"\x00" + b"\xff\x00\x00" * 4 for _ in range(4))
    ihdr = struct.pack(">IIBBBBB", 4, 4, 8, 2, 0, 0, 0)
    return b"\x89PNG\r\n\x1a\n" + chunk(b"IHDR", ihdr) + chunk(b"IDAT", zlib.compress(raw)) + chunk(b"IEND", b"")


# ── Hub e abertura ──────────────────────────────────────────────────────────


def test_hub_mostra_o_card(page: Page) -> None:
    page.goto(f"{GAMES_BASE_URL}/")
    page.evaluate("() => localStorage.setItem('games_locale', 'pt-BR')")
    page.reload()
    card = page.get_by_role("link", name=re.compile("Bê-á-Bá"))
    expect(card).to_have_attribute("href", "/be-a-ba/")
    expect(card.get_by_text("Para crianças")).to_be_visible()
    # O selo "Novo" anda de jogo em jogo — agora está aqui.
    expect(card.get_by_text("Novo", exact=True)).to_be_visible()


def test_manifesto_tem_escopo_proprio(page: Page) -> None:
    _open(page)
    href = page.locator("link[rel=manifest]").get_attribute("href")
    assert href == "/be-a-ba/manifest.webmanifest"
    manifest = page.evaluate(f"() => fetch('{href}').then(r => r.json())")
    assert manifest["scope"] == "/be-a-ba/"
    assert manifest["start_url"] == "/be-a-ba/"


def test_inicio_mostra_as_duas_trilhas_e_o_nome(page: Page) -> None:
    _open(page, _state(childName="Arthur"))
    expect(page.get_by_text("Oi, Arthur!")).to_be_visible()
    expect(page.get_by_role("button", name=re.compile("Letras.*Fase 1"))).to_be_enabled()
    expect(page.get_by_role("button", name=re.compile("Números.*Fase 1"))).to_be_enabled()


# ── Trilha das Letras ───────────────────────────────────────────────────────


def test_rodada_de_letras_completa_da_figurinha_e_estrelas(page: Page) -> None:
    _open(page, _state(letters={"stage": 2}))
    page.get_by_role("button", name=re.compile("Letras")).click()
    words = _play_round(page, _solve_letter)

    expect(page.get_by_text("Você ganhou uma figurinha!")).to_be_visible()
    saved = _saved(page)
    assert saved["stars"] == 5
    assert saved["rounds"] == 1
    assert len(saved["stickers"]) == 1
    for w in words:
        assert saved["letters"]["boxes"][_plain(w).lower()]["box"] == 2  # acerto limpo sobe a caixa


def test_primeira_letra_errada_balanca_e_a_certa_brilha(page: Page) -> None:
    _open(page, _state(letters={"stage": 0}))
    page.get_by_role("button", name=re.compile("Letras")).click()
    ch = page.locator("[data-mode=first]")
    first = _plain(ch.get_attribute("data-answer")[0])
    wrong = [b for b in ch.locator(".big-letter").all() if b.inner_text() != first]

    wrong[0].click()
    expect(wrong[0]).to_have_class(re.compile("wrong"))
    expect(page.locator("[data-mode=first]")).to_be_visible()  # não avançou
    wrong[1].click()
    expect(ch.get_by_role("button", name=f"Letra {first}", exact=True)).to_have_class(re.compile("glow"))

    ch.get_by_role("button", name=f"Letra {first}", exact=True).click()
    expect(page.locator(".praise")).to_be_visible()


def test_montar_letra_errada_fica_na_bandeja(page: Page) -> None:
    _open(page, _state(letters={"stage": 1}))
    page.get_by_role("button", name=re.compile("Letras")).click()
    ch = page.locator("[data-mode=build]")
    word = ch.get_attribute("data-answer")
    tiles = ch.locator(".tile")
    wrong = next(t for t in tiles.all() if t.inner_text() != _plain(word[0]))
    wrong.click()
    expect(wrong).not_to_have_class(re.compile("used"))
    expect(ch.locator(".slot.filled")).to_have_count(0)

    ch.locator(".tile:not(.used)", has_text=re.compile(f"^{_plain(word[0])}$")).first.click()
    expect(ch.locator(".slot.filled")).to_have_count(1)
    expect(ch.locator(".slot").first).to_have_text(word[0])


def test_escrever_sem_acento_completa_a_palavra_acentuada(page: Page) -> None:
    family = [{"id": "fam-1", "text": "MAMÃE", "photo": False}]
    _open(page, _state(letters={"stage": 3}, family=family))
    page.get_by_role("button", name=re.compile("Letras")).click()
    for _ in range(5):
        ch = page.locator("[data-mode=type]")
        word = ch.get_attribute("data-answer")
        _solve_letter(page)
        if word == "MAMÃE":
            expect(ch.locator(".slot")).to_have_text(list("MAMÃE"))
            break
        _next(page)
    else:
        pytest.fail("a palavra da família não entrou na rodada")


def test_nome_da_crianca_da_figurinha_especial(page: Page) -> None:
    family = [{"id": "child", "text": "ARTHUR", "photo": False, "isChildName": True}]
    _open(page, _state(childName="Arthur", family=family, letters={"stage": 1}))
    page.get_by_role("button", name=re.compile("Letras")).click()
    words = _play_round(page, _solve_letter)
    assert "ARTHUR" in words

    expect(page.get_by_text("Você escreveu o seu nome!")).to_be_visible()
    assert "meu_nome" in _saved(page)["stickers"]
    page.get_by_role("button", name="Ver o álbum").click()
    expect(page.get_by_text("Especial")).to_be_visible()


def test_sobe_de_fase_depois_de_duas_rodadas_boas(page: Page) -> None:
    _open(page, _state(letters={"stage": 1, "locked": False}))
    page.get_by_role("button", name=re.compile("Letras")).click()
    _play_round(page, _solve_letter)
    assert _saved(page)["letters"]["stage"] == 1
    page.get_by_role("button", name="Jogar de novo").click()
    _play_round(page, _solve_letter)
    expect(page.get_by_text("Você passou de fase!")).to_be_visible()
    assert _saved(page)["letters"]["stage"] == 2


def test_instrucao_e_falada_ao_abrir_o_desafio(page: Page) -> None:
    _open(page, _state(letters={"stage": 1}))
    page.get_by_role("button", name=re.compile("Letras")).click()
    word = page.locator("[data-mode]").get_attribute("data-answer")
    spoken = page.evaluate("() => document.documentElement.dataset.spoken")
    assert spoken == f"Monte a palavra {word.lower()}."


# ── Trilha dos Números ──────────────────────────────────────────────────────


def test_rodada_de_somar_e_tirar(page: Page) -> None:
    _open(page, _state(numbers={"stage": 3}))
    page.get_by_role("button", name=re.compile("Números")).click()
    kinds = set()
    for _ in range(5):
        expect(page.locator("[data-kind]")).to_be_visible()
        kinds.add(page.locator("[data-kind]").get_attribute("data-kind"))
        _solve_number(page)
        _next(page)
    assert kinds == {"add", "sub"}
    expect(page.get_by_role("heading", name="Rodada completa!")).to_be_visible()
    boxes = _saved(page)["numbers"]["boxes"]
    assert set(boxes) == {"add", "sub"}


def test_contar_resposta_errada_nao_avanca(page: Page) -> None:
    _open(page, _state(numbers={"stage": 0}))
    page.get_by_role("button", name=re.compile("Números")).click()
    ch = page.locator("[data-kind=count]")
    answer = ch.get_attribute("data-answer")
    assert ch.locator(".objects img").count() == int(answer)
    wrong = next(b for b in ch.locator(".number").all() if b.inner_text() != answer)
    wrong.click()
    expect(wrong).to_have_class(re.compile("wrong"))
    expect(page.locator(".praise")).to_have_count(0)
    ch.get_by_role("button", name=answer, exact=True).click()
    expect(page.locator(".praise")).to_be_visible()


def test_comparar_qual_lado_tem_mais(page: Page) -> None:
    _open(page, _state(numbers={"stage": 1}))
    page.get_by_role("button", name=re.compile("Números")).click()
    page.locator("[data-kind]").first.wait_for()
    for _ in range(5):
        if page.locator("[data-kind]").get_attribute("data-kind") == "compare":
            break
        _solve_number(page)
        _next(page)
    ch = page.locator("[data-kind=compare]")
    left, right = (s.locator("img").count() for s in ch.locator(".side").all())
    assert (left > right) == (ch.get_attribute("data-answer") == "left")
    _solve_number(page)
    expect(page.locator(".praise")).to_be_visible()


# ── Área dos pais ───────────────────────────────────────────────────────────


def test_portao_dos_pais_numero_errado_nao_entra(page: Page) -> None:
    _open(page)
    page.get_by_role("button", name="Pais").click()
    dialog = page.get_by_role("dialog", name="Área dos pais")
    ask = dialog.locator("p").first.inner_text()
    target = next(v for k, v in GATE_WORDS.items() if k in ask)
    other = next(b for b in dialog.locator(".big-btn").all() if b.inner_text() != str(target))
    other.click()
    expect(page.get_by_text("Esse não é. Tente de novo.")).to_be_visible()
    expect(page.get_by_role("heading", name="Área dos pais")).to_have_count(1)  # só o do diálogo


def test_cadastra_palavra_da_familia_com_foto(page: Page) -> None:
    _open(page)
    _enter_parents(page)
    page.locator("#bab-word").fill("vovó")
    page.locator("#bab-photo").set_input_files({"name": "vovo.png", "mimeType": "image/png", "buffer": _png()})
    page.get_by_role("button", name="Adicionar").click()

    item = page.locator("[data-family='VOVÓ']")
    expect(item.locator(".thumb img")).to_be_visible()
    fam = _saved(page)["family"]
    assert [(f["text"], f["photo"]) for f in fam] == [("VOVÓ", True)]

    # Palavra repetida e inválida são recusadas
    page.locator("#bab-word").fill("Vovo")
    page.get_by_role("button", name="Adicionar").click()
    expect(page.get_by_text("Essa palavra já está na lista.")).to_be_visible()
    page.locator("#bab-word").fill("123")
    page.get_by_role("button", name="Adicionar").click()
    expect(page.get_by_text("Use só letras, até 12.")).to_be_visible()

    # Remover pede confirmação
    page.get_by_role("button", name="Remover VOVÓ").click()
    page.get_by_role("dialog").get_by_role("button", name="Remover VOVÓ").click()
    expect(item).to_have_count(0)
    assert _saved(page)["family"] == []


def test_nome_da_crianca_vira_palavra_da_familia(page: Page) -> None:
    _open(page)
    _enter_parents(page)
    page.locator("#bab-child").fill("Arthur")
    page.get_by_role("button", name="Salvar").click()
    expect(page.locator("[data-family='ARTHUR']")).to_be_visible()
    saved = _saved(page)
    assert saved["childName"] == "Arthur"
    assert saved["family"][0]["isChildName"] is True
    page.get_by_role("button", name="Início").click()
    expect(page.get_by_text("Oi, Arthur!")).to_be_visible()


def test_fixar_fase_pelos_pais(page: Page) -> None:
    _open(page, _state(letters={"locked": False}))
    _enter_parents(page)
    page.locator("#bab-stage-letters").select_option("3")
    saved = _saved(page)["letters"]
    assert saved["locked"] is True and saved["stage"] == 3
    page.locator("#bab-stage-letters").select_option("auto")
    assert _saved(page)["letters"]["locked"] is False


def test_zerar_progresso_mantem_a_familia(page: Page) -> None:
    family = [{"id": "fam-1", "text": "MAMÃE", "photo": False}]
    _open(page, _state(stars=40, stickers=["leao"], rounds=8, family=family))
    _enter_parents(page)
    page.get_by_role("button", name="Zerar progresso").click()
    page.get_by_role("dialog").get_by_role("button", name="Zerar progresso").click()
    saved = _saved(page)
    assert (saved["stars"], saved["stickers"], saved["rounds"]) == (0, [], 0)
    assert saved["family"] == family


# ── Idioma ──────────────────────────────────────────────────────────────────


def test_em_ingles_as_letras_ficam_para_depois_e_os_numeros_funcionam(page: Page) -> None:
    _open(page, _state(numbers={"stage": 0}), locale="en")
    expect(page.get_by_role("button", name=re.compile("Letters"))).to_be_disabled()
    expect(page.get_by_text("Coming soon in English")).to_be_visible()
    page.get_by_role("button", name=re.compile("Numbers")).click()
    expect(page.get_by_text("How many?")).to_be_visible()


# ── Modo Pequenos (3–4 anos) ────────────────────────────────────────────────


def test_primeira_visita_pergunta_a_idade(page: Page) -> None:
    _open_fresh(page)
    expect(page.get_by_text("Quantos anos tem quem vai brincar?")).to_be_visible()
    page.get_by_role("button", name=re.compile("Pequenos")).click()
    expect(page.locator("[data-age=little]")).to_be_visible()
    assert _saved(page)["ageMode"] == "little"


def test_quem_ja_jogava_continua_no_modo_de_5_a_7(page: Page) -> None:
    state = _state()
    del state["ageMode"], state["little"]
    _open(page, state)
    expect(page.locator("[data-age=kids]")).to_be_visible()


def test_pequenos_rodada_de_letras_tem_3_desafios_e_figurinha(page: Page) -> None:
    _open(page, _little(letters=1))
    page.get_by_role("button", name=re.compile("Letras")).click()
    kinds = _play_round(page, _solve_little, size=3)
    assert set(kinds) <= {"hear", "pick_letter"}
    expect(page.get_by_text("Você ganhou uma figurinha!")).to_be_visible()
    saved = _saved(page)
    assert saved["stars"] == 3 and len(saved["stickers"]) == 1
    keys = saved["little"]["letters"]["boxes"].keys()
    assert keys and all(k.startswith(("hear:", "letter:")) for k in keys)
    assert saved["letters"]["boxes"] == {}  # o modo 5–7 anos não é tocado


def test_pequenos_cade_figura_errada_fica_apagada(page: Page) -> None:
    _open(page, _little(letters=0))
    page.get_by_role("button", name=re.compile("Letras")).click()
    ch = page.locator("[data-little-kind=hear]")
    answer = ch.get_attribute("data-answer")
    word = page.evaluate("() => document.documentElement.dataset.spoken")
    assert word.startswith("Cadê ")
    wrong = ch.locator(f".choice:not([data-option='{answer}'])").first
    wrong.click()
    expect(wrong).to_have_class(re.compile("wrong"))
    expect(page.locator(".praise")).to_have_count(0)
    ch.locator(f"[data-option='{answer}']").click()
    expect(page.locator(".praise")).to_be_visible()


def test_pequenos_contar_tocando_cada_objeto(page: Page) -> None:
    _open(page, _little(numbers=0))
    page.get_by_role("button", name=re.compile("Números")).click()
    ch = page.locator("[data-little-kind=tap]")
    n = int(ch.get_attribute("data-answer"))
    objects = ch.locator(".tap-object")
    expect(objects).to_have_count(n)
    objects.first.click()
    objects.first.click()  # tocar de novo não conta duas vezes
    expect(ch.locator(".badge")).to_have_count(1)
    assert page.evaluate("() => document.documentElement.dataset.spoken") == "1"
    for obj in objects.all()[1:]:
        obj.click()
    expect(ch.locator(".badge")).to_have_text([str(i + 1) for i in range(n)])
    expect(page.locator(".praise")).to_be_visible()


def test_pequenos_rodada_de_numeros_ate_5(page: Page) -> None:
    _open(page, _little(numbers=2))
    page.get_by_role("button", name=re.compile("Números")).click()
    kinds = _play_round(page, _solve_little, size=3)
    assert set(kinds) <= {"find_number", "where", "tap"}
    assert set(_saved(page)["little"]["numbers"]["boxes"]) == set(kinds)


def test_trocar_a_idade_na_area_dos_pais(page: Page) -> None:
    _open(page)
    _enter_parents(page)
    page.locator("#bab-age").select_option("little")
    expect(page.locator("#bab-stage-letters option").nth(1)).to_have_text(re.compile("Cadê"))
    assert _saved(page)["ageMode"] == "little"
    page.locator("#bab-stage-numbers").select_option("2")
    assert _saved(page)["little"]["numbers"]["stage"] == 2
    assert _saved(page)["numbers"]["stage"] == 0
