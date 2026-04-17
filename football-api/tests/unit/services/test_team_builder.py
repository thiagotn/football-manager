"""
Testes unitários — app/services/team_builder.py

Cobre: snake draft, distribuição de goleiros, reservas, nome/cor dos times.
Não requer banco de dados nem HTTP.
"""
from uuid import uuid4

import pytest

from app.services.team_builder import TEAM_COLORS, build_teams


def make_player(skill: int = 3, position: str = "mei") -> dict:
    return {
        "player_id": str(uuid4()),
        "name": f"Jogador {uuid4().hex[:4]}",
        "nickname": None,
        "skill_stars": skill,
        "position": position,
    }


def make_players(n: int, skill: int = 3, position: str = "mei") -> list[dict]:
    return [make_player(skill=skill, position=position) for _ in range(n)]


# ── Validação de entrada ──────────────────────────────────────────────────────


def test_raises_with_insufficient_players():
    """Menos jogadores do que o mínimo para 2 times deve levantar ValueError."""
    players = make_players(3)  # precisa de ≥ 2*(players_per_team+1)
    with pytest.raises(ValueError, match="Confirmados insuficientes"):
        build_teams(players, players_per_team=3)


def test_raises_with_zero_players():
    with pytest.raises(ValueError):
        build_teams([], players_per_team=3)


# ── Formação dos times ────────────────────────────────────────────────────────


def test_creates_two_teams_for_exact_fit():
    """8 jogadores com players_per_team=3 → 2 times de 4, sem reservas."""
    players = make_players(8)
    teams, reserves = build_teams(players, players_per_team=3)
    assert len(teams) == 2
    assert len(reserves) == 0


def test_creates_three_teams_when_enough_players():
    """12 jogadores com players_per_team=3 → 3 times de 4."""
    players = make_players(12)
    teams, reserves = build_teams(players, players_per_team=3)
    assert len(teams) == 3


def test_each_team_has_correct_size():
    """8 jogadores = 2 times × 4 exatos → sem time incompleto."""
    players = make_players(8)
    teams, _ = build_teams(players, players_per_team=3)
    for team in teams:
        assert len(team["players"]) == 4


def test_incomplete_roster_creates_extra_team_no_reserves():
    """
    Com ceil, 9 jogadores e team_size=4 gera 3 times sem reservas.
    Todos os jogadores entram em algum time — 1 time fica com 1 jogador a menos.
    """
    players = make_players(9)
    teams, reserves = build_teams(players, players_per_team=3)
    assert len(teams) == 3
    assert len(reserves) == 0
    assert sum(len(t["players"]) for t in teams) == 9


def test_39_players_creates_4_teams_no_reserves():
    """Regressão principal: 39 jogadores com 9+GK devem criar 4 times, sem reservas."""
    players = (
        make_players(3, position="gk")
        + make_players(5, position="lat")
        + make_players(11, position="zag")
        + make_players(16, position="mei")
        + make_players(4, position="ata")
    )
    teams, reserves = build_teams(players, players_per_team=9)
    assert len(teams) == 4
    assert len(reserves) == 0
    sizes = sorted(len(t["players"]) for t in teams)
    # 3 times completos (10) + 1 com 9 = 39
    assert sizes == [9, 10, 10, 10]


def test_total_players_conserved():
    """Jogadores ativos + reservas deve ser igual ao total de confirmados."""
    players = make_players(10)
    teams, reserves = build_teams(players, players_per_team=3)
    active = sum(len(t["players"]) for t in teams)
    assert active + len(reserves) == 10


# ── Goleiros ─────────────────────────────────────────────────────────────────


def test_goalkeeper_distributed_one_per_team():
    """Com 2 goleiros para 2 times, cada time deve ter exatamente 1."""
    players = make_players(6, position="mei")
    players[0]["position"] = "gk"
    players[1]["position"] = "gk"
    teams, _ = build_teams(players, players_per_team=2)
    for team in teams:
        gks = [p for p in team["players"] if p["position"] == "gk"]
        assert len(gks) == 1


def test_excess_goalkeeper_goes_to_outfield_pool():
    """3 goleiros para 2 times → 2 ficam como GK titulares, 1 vai para o pool de linha."""
    players = make_players(8, position="mei")
    players[0]["position"] = "gk"
    players[1]["position"] = "gk"
    players[2]["position"] = "gk"
    teams, reserves = build_teams(players, players_per_team=3)
    # Todos os 8 jogadores devem estar distribuídos (2 times de 4, sem reservas)
    total_active = sum(len(t["players"]) for t in teams)
    assert total_active + len(reserves) == 8
    assert len(teams) == 2


# ── Metadados dos times ───────────────────────────────────────────────────────


def test_teams_have_name_and_color():
    players = make_players(8)
    teams, _ = build_teams(players, players_per_team=3)
    for team in teams:
        assert team["name"]
        assert team["color"].startswith("#")


def test_team_colors_are_from_valid_pool():
    players = make_players(8)
    teams, _ = build_teams(players, players_per_team=3)
    for team in teams:
        assert team["color"] in TEAM_COLORS


def test_teams_have_position():
    players = make_players(8)
    teams, _ = build_teams(players, players_per_team=3)
    positions = [t["position"] for t in teams]
    assert sorted(positions) == list(range(1, len(teams) + 1))


def test_skill_total_is_sum_of_player_skills():
    players = make_players(8, skill=4)
    teams, _ = build_teams(players, players_per_team=3)
    for team in teams:
        expected = sum(p["skill_stars"] for p in team["players"])
        assert team["skill_total"] == expected


# ── Snake draft (equilíbrio) ──────────────────────────────────────────────────


def test_swap_optimization_produces_balanced_teams():
    """Com jogadores de habilidades variadas, a diferença de skill_total deve ser pequena."""
    # 4 de skill 5, 4 de skill 1 → otimização deve equilibrar perfeitamente
    players = make_players(4, skill=5) + make_players(4, skill=1)
    teams, _ = build_teams(players, players_per_team=3)
    totals = [t["skill_total"] for t in teams]
    diff = max(totals) - min(totals)
    assert diff <= 2  # otimização deve equalizar ao máximo


def test_swap_optimization_real_world_scenario():
    """
    Regressão do caso real: 40 jogadores com skill heterogêneo (1-5★).
    A diferença entre o time mais forte e mais fraco não deve ultrapassar 4★.
    """
    # Distribui as estrelas de forma heterogênea por posição (semelhante ao caso real)
    import random as _rnd
    _rnd.seed(42)
    players = (
        make_players(1, skill=2, position="gk")   # GK fraco
        + make_players(3, skill=4, position="gk")  # GKs bons
        + make_players(2, skill=4, position="lat")
        + make_players(2, skill=3, position="lat")
        + make_players(1, skill=5, position="zag")
        + make_players(2, skill=4, position="zag")
        + make_players(3, skill=3, position="zag")
        + make_players(2, skill=2, position="zag")
        + make_players(2, skill=1, position="zag")
        + make_players(2, skill=5, position="mei")
        + make_players(5, skill=4, position="mei")
        + make_players(7, skill=3, position="mei")
        + make_players(1, skill=2, position="mei")
        + make_players(1, skill=5, position="ata")
        + make_players(2, skill=4, position="ata")
        + make_players(1, skill=4, position="ata")
    )
    teams, _ = build_teams(players, players_per_team=9)
    totals = [t["skill_total"] for t in teams]
    diff = max(totals) - min(totals)
    assert diff <= 4, f"Desequilíbrio excessivo: totais={totals}, diff={diff}"


# ── Equilíbrio de posição ────────────────────────────────────────────────────


def test_position_balance_no_team_without_forward():
    """Nenhum time deve ficar sem ATA quando há ATAs suficientes."""
    # 4 times de 9 campo + 1 GK = 10 — 40 jogadores
    players = (
        make_players(4, position="gk")
        + make_players(5, position="lat")
        + make_players(11, position="zag")
        + make_players(16, position="mei")
        + make_players(4, position="ata")
    )
    teams, _ = build_teams(players, players_per_team=9)
    for team in teams:
        atas = [p for p in team["players"] if p["position"] == "ata"]
        assert len(atas) >= 1, f"Time '{team['name']}' ficou sem atacante"


def test_position_balance_no_team_without_fullback():
    """Com 5 laterais para 4 times, nenhum time deve ficar sem LAT."""
    players = (
        make_players(4, position="gk")
        + make_players(5, position="lat")
        + make_players(11, position="zag")
        + make_players(16, position="mei")
        + make_players(4, position="ata")
    )
    teams, _ = build_teams(players, players_per_team=9)
    for team in teams:
        lats = [p for p in team["players"] if p["position"] == "lat"]
        assert len(lats) >= 1, f"Time '{team['name']}' ficou sem lateral"


def test_position_counts_differ_by_at_most_one():
    """A contagem de cada posição entre times deve diferir em no máximo 1."""
    players = (
        make_players(4, position="gk")
        + make_players(5, position="lat")
        + make_players(11, position="zag")
        + make_players(16, position="mei")
        + make_players(4, position="ata")
    )
    teams, _ = build_teams(players, players_per_team=9)
    for pos in ("lat", "zag", "mei", "ata"):
        counts = [len([p for p in t["players"] if p["position"] == pos]) for t in teams]
        assert max(counts) - min(counts) <= 1, (
            f"Posição '{pos}' desequilibrada: {counts}"
        )


# ── Tamanho correto com composição desbalanceada (regressão) ─────────────────


def test_no_team_exceeds_team_size_with_mei_heavy_roster():
    """
    Regressão: 39 jogadores com muitos MEIs causava times de 12 em vez de ≤10.
    Nenhum time deve ter mais que team_size jogadores.
    """
    players = (
        make_players(3, position="gk")
        + make_players(5, position="lat")
        + make_players(11, position="zag")
        + make_players(16, position="mei")
        + make_players(4, position="ata")
    )
    assert len(players) == 39
    teams, reserves = build_teams(players, players_per_team=9)
    assert len(teams) == 4  # ceil(39/10) = 4
    assert len(reserves) == 0
    for team in teams:
        assert len(team["players"]) <= 10, (
            f"Time '{team['name']}' tem {len(team['players'])} jogadores, máximo 10"
        )
    assert sum(len(t["players"]) for t in teams) == 39


# ── _pick_names — overflow além do pool ──────────────────────────────────────


def test_pick_names_overflow_beyond_pool():
    """Com mais de 15 times (tamanho do pool), _pick_names gera nomes com sufixo numérico."""
    # 16 times: players_per_team=1 → team_size=2, precisa 32 jogadores
    players = make_players(32, skill=3)
    teams, reserves = build_teams(players, players_per_team=1)

    assert len(teams) == 16
    # Todos os times devem ter nome não-vazio
    names = [t["name"] for t in teams]
    assert all(n for n in names)
    # Não deve haver duplicatas
    assert len(set(names)) == len(names)
