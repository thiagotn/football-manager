"""
E2E — exibição de posições na página de times (/match/[hash]/teams).

No sorteio **simplificado** (`strategy = "simple"`) o algoritmo equilibra só as
estrelas e ignora as posições de linha de propósito. Exibi-las nesse modo faz o
resultado parecer desequilibrado e gera questionamento, então a página esconde o
badge de posição — exceto o do goleiro, a única posição que o modo simplificado
continua garantindo (um por time).

Sortear de verdade exigiria grupo + partida + jogadores confirmados com posições
controladas, então estes testes mockam `/matches/public/{hash}` e
`/matches/*/teams` (mesmo padrão de `test_videos.py`) e validam exatamente a
regra de renderização.
"""

import json

import pytest
from playwright.sync_api import Page, expect

FAKE_HASH = "e2eteamshash"

FAKE_MATCH = {
    "id": "33333333-3333-3333-3333-333333333333",
    "group_id": "44444444-4444-4444-4444-444444444444",
    "number": 7,
    "hash": FAKE_HASH,
    "match_date": "2026-08-20",
    "start_time": "19:00:00",
    "end_time": "20:00:00",
    "location": "Quadra E2E",
    "address": None,
    "court_type": None,
    "players_per_team": 5,
    "max_players": None,
    "notes": None,
    "status": "closed",
    "vote_open_delay_minutes": 0,
    "vote_duration_hours": 24,
    "created_at": "2026-08-20T10:00:00Z",
    "updated_at": "2026-08-20T10:00:00Z",
    "attendances": [],
    "confirmed_count": 0,
    "declined_count": 0,
    "pending_count": 0,
    "group_name": "Grupo E2E Times",
    "group_timezone": "America/Sao_Paulo",
    "group_per_match_amount": None,
    "group_videos_enabled": False,
}


def _player(player_id, name, position, stars=3):
    return {
        "player_id": player_id,
        "name": name,
        "nickname": None,
        "skill_stars": stars,
        "position": position,
    }


# Um goleiro por time e posições de linha variadas — é o badge de linha que deve
# sumir no modo simplificado, e o do goleiro que deve sobreviver.
TEAM_A = [
    _player("a1", "Marcos Goleiro", "gk"),
    _player("a2", "Pedro Zagueiro", "zag"),
    _player("a3", "Bruno Meia", "mei"),
    _player("a4", "Tiago Atacante", "ata"),
]
TEAM_B = [
    _player("b1", "Alex Goleiro", "gk"),
    _player("b2", "Rafael Lateral", "lat"),
    _player("b3", "Caio Meia", "mei"),
    _player("b4", "Diego Atacante", "ata"),
]


def _fulfill_json(route, payload):
    route.fulfill(
        status=200,
        content_type="application/json",
        body=json.dumps(payload),
    )


def _mock_teams(page: Page, strategy: str):
    page.route(
        f"**/matches/public/{FAKE_HASH}",
        lambda route: _fulfill_json(route, FAKE_MATCH),
    )
    page.route(
        f"**/matches/public/{FAKE_HASH}/player-stats",
        lambda route: _fulfill_json(route, {"stats": []}),
    )
    payload = {
        "teams": [
            {
                "id": "t1",
                "name": "Time Vermelho",
                "color": "#e63946",
                "position": 1,
                "skill_total": sum(p["skill_stars"] for p in TEAM_A),
                "players": TEAM_A,
            },
            {
                "id": "t2",
                "name": "Time Azul",
                "color": "#457b9d",
                "position": 2,
                "skill_total": sum(p["skill_stars"] for p in TEAM_B),
                "players": TEAM_B,
            },
        ],
        "reserves": [],
        "strategy": strategy,
    }
    page.route("**/matches/*/teams", lambda route: _fulfill_json(route, payload))


@pytest.fixture
def anon_page(browser):
    ctx = browser.new_context()
    page = ctx.new_page()
    yield page
    ctx.close()


def test_sorteio_equilibrado_mostra_todas_as_posicoes(anon_page: Page, base_url):
    _mock_teams(anon_page, "balanced")
    anon_page.goto(f"{base_url}/match/{FAKE_HASH}/teams")

    expect(anon_page.get_by_test_id("team-card")).to_have_count(2)

    badges = anon_page.get_by_test_id("position-badge")
    expect(badges).to_have_count(len(TEAM_A) + len(TEAM_B))
    assert set(badges.all_inner_texts()) == {"GK", "ZAG", "MEI", "ATA", "LAT"}


def test_sorteio_simplificado_esconde_posicoes_de_linha(anon_page: Page, base_url):
    _mock_teams(anon_page, "simple")
    anon_page.goto(f"{base_url}/match/{FAKE_HASH}/teams")

    expect(anon_page.get_by_test_id("team-card")).to_have_count(2)

    # Só sobram os dois goleiros — nenhum badge de linha.
    badges = anon_page.get_by_test_id("position-badge")
    expect(badges).to_have_count(2)
    assert set(badges.all_inner_texts()) == {"GK"}


def test_sorteio_simplificado_mantem_os_jogadores_visiveis(anon_page: Page, base_url):
    """Esconder a posição não pode esconder o jogador."""
    _mock_teams(anon_page, "simple")
    anon_page.goto(f"{base_url}/match/{FAKE_HASH}/teams")

    expect(anon_page.get_by_test_id("team-player")).to_have_count(
        len(TEAM_A) + len(TEAM_B)
    )
    # A página exibe o nome de exibição (primeiro nome, sem apelido aqui).
    for p in TEAM_A + TEAM_B:
        first_name = p["name"].split()[0]
        expect(anon_page.get_by_text(first_name, exact=True)).to_be_visible()


def test_sorteio_simplificado_ordena_goleiro_primeiro(anon_page: Page, base_url):
    """Sem o badge, ordenar por posição ainda agruparia os jogadores por posição.

    No modo simplificado a ordem é goleiro primeiro e o resto alfabético.
    """
    _mock_teams(anon_page, "simple")
    anon_page.goto(f"{base_url}/match/{FAKE_HASH}/teams")

    first_team = anon_page.get_by_test_id("team-card").first
    rows = first_team.get_by_test_id("team-player")
    expect(rows).to_have_count(len(TEAM_A))  # ancora a espera: all_inner_texts não espera

    names = [row.split()[0] for row in rows.all_inner_texts()]
    assert names == ["Marcos", "Bruno", "Pedro", "Tiago"]
