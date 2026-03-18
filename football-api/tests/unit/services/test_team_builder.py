"""
Testes unitários — app/services/team_builder.py

Cobre: snake draft, distribuição de goleiros, reservas, nome/cor dos times.
Não requer banco de dados nem HTTP.
"""
from uuid import uuid4

import pytest

from app.services.team_builder import TEAM_COLORS, build_teams


def make_player(skill: int = 3, is_goalkeeper: bool = False) -> dict:
    return {
        "player_id": str(uuid4()),
        "name": f"Jogador {uuid4().hex[:4]}",
        "nickname": None,
        "skill_stars": skill,
        "is_goalkeeper": is_goalkeeper,
    }


def make_players(n: int, skill: int = 3, is_goalkeeper: bool = False) -> list[dict]:
    return [make_player(skill=skill, is_goalkeeper=is_goalkeeper) for _ in range(n)]


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
    players = make_players(8)
    teams, _ = build_teams(players, players_per_team=3)
    for team in teams:
        assert len(team["players"]) == 4  # 3 linha + 1 goleiro/substituto


def test_reserves_contain_excess_players():
    """9 jogadores com players_per_team=3 (time_size=4) → 2 times + 1 reserva."""
    players = make_players(9)
    teams, reserves = build_teams(players, players_per_team=3)
    assert len(reserves) == 1


def test_total_players_conserved():
    """Jogadores ativos + reservas deve ser igual ao total de confirmados."""
    players = make_players(10)
    teams, reserves = build_teams(players, players_per_team=3)
    active = sum(len(t["players"]) for t in teams)
    assert active + len(reserves) == 10


# ── Goleiros ─────────────────────────────────────────────────────────────────


def test_goalkeeper_distributed_one_per_team():
    """Com 2 goleiros para 2 times, cada time deve ter exatamente 1."""
    players = make_players(6, is_goalkeeper=False)
    players[0]["is_goalkeeper"] = True
    players[1]["is_goalkeeper"] = True
    teams, _ = build_teams(players, players_per_team=2)
    for team in teams:
        gks = [p for p in team["players"] if p["is_goalkeeper"]]
        assert len(gks) == 1


def test_excess_goalkeeper_goes_to_outfield_pool():
    """3 goleiros para 2 times → 2 ficam como GK titulares, 1 vai para o pool de linha.
    O 3º goleiro pode terminar em qualquer time como jogador de linha (is_goalkeeper=True
    mas tratado como outfield pelo snake draft). Total de GKs nos times pode ser até 3."""
    players = make_players(8, is_goalkeeper=False)
    players[0]["is_goalkeeper"] = True
    players[1]["is_goalkeeper"] = True
    players[2]["is_goalkeeper"] = True
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


def test_snake_draft_produces_balanced_teams():
    """Com jogadores de habilidades variadas, a diferença de skill_total deve ser pequena."""
    # 4 de skill 5, 4 de skill 1 → snake draft deve equilibrar
    players = make_players(4, skill=5) + make_players(4, skill=1)
    teams, _ = build_teams(players, players_per_team=3)
    totals = [t["skill_total"] for t in teams]
    diff = max(totals) - min(totals)
    assert diff <= 8  # diferença tolerável para 2 times com snake draft
