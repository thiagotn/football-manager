import random
from uuid import UUID

TEAM_NAMES = [
    "Real Madruga", "Barcelusa", "Barsemlona", "Meia Boca Juniors",
    "Baile de Munique", "Varmeiras", "Atecubanos FC", "Inter de Limão",
    "Manchester Cachaça", "Real Matismo", "Paysanduba", "Horriver Plate",
    "Patético de Madrid", "Shakhtar dos Leks", "Espressinho da Mooca",
]

TEAM_COLORS = [
    "#e53e3e",  # vermelho
    "#3b82f6",  # azul
    "#f59e0b",  # amarelo
    "#22c55e",  # verde
    "#f97316",  # laranja
    "#a855f7",  # roxo
    "#ec4899",  # rosa
    "#06b6d4",  # ciano
    "#84cc16",  # limão
    "#14b8a6",  # verde-azulado
]


def _pick_names(n: int) -> list[str]:
    pool = TEAM_NAMES[:]
    random.shuffle(pool)
    names: list[str] = []
    used: set[str] = set()
    for name in pool:
        if name not in used:
            names.append(name)
            used.add(name)
        if len(names) == n:
            break
    # Se precisar de mais nomes que o pool, adiciona sufixo numérico
    idx = 2
    while len(names) < n:
        candidate = f"{pool[0]} {idx}"
        if candidate not in used:
            names.append(candidate)
            used.add(candidate)
            idx += 1
    return names


def build_teams(
    confirmed: list[dict],
    players_per_team: int,
) -> tuple[list[dict], list[dict]]:
    """
    Recebe lista de confirmados com player_id, skill_stars, is_goalkeeper.
    Retorna (teams, reserves).

    players_per_team = jogadores de LINHA por time (exclui goleiro).
    Tamanho total de cada time = players_per_team + 1 (linha + goleiro).

    Cada team: { name, color, position, players: [player_dict] }
    Cada player_dict: { player_id, name, nickname, skill_stars, is_goalkeeper }
    """
    team_size = players_per_team + 1  # linha + 1 goleiro (ou substituto)
    n_times = len(confirmed) // team_size
    if n_times < 2:
        raise ValueError("Confirmados insuficientes para montar times.")

    total_in_teams = n_times * team_size

    # Separa goleiros e não-goleiros de TODOS os confirmados
    goleiros = [p for p in confirmed if p["is_goalkeeper"]]
    nao_goleiros = [p for p in confirmed if not p["is_goalkeeper"]]

    times: list[list[dict]] = [[] for _ in range(n_times)]

    # 1. Distribui um goleiro por time (máximo 1 por time)
    gk_para_times = goleiros[:n_times]
    gk_excedentes = goleiros[n_times:]
    for i, g in enumerate(gk_para_times):
        times[i].append(g)

    # 2. Pool para snake draft: não-goleiros + goleiros excedentes, ordenados por skill
    pool_todos = sorted(
        nao_goleiros + gk_excedentes,
        key=lambda p: p["skill_stars"],
        reverse=True,
    )

    # 3. Reservas = jogadores que não caberão nos times (os mais fracos do pool)
    spots_restantes = total_in_teams - len(gk_para_times)
    pool = pool_todos[:spots_restantes]
    reserves = pool_todos[spots_restantes:]

    # 2. Snake draft respeitando a capacidade restante de cada time
    # Cada time precisa de exatamente (team_size - jogadores já atribuídos) picks
    needs = [team_size - len(t) for t in times]
    pick_order: list[int] = []
    round_num = 0
    while any(n > 0 for n in needs):
        team_range = range(n_times) if round_num % 2 == 0 else range(n_times - 1, -1, -1)
        for t in team_range:
            if needs[t] > 0:
                pick_order.append(t)
                needs[t] -= 1
        round_num += 1

    for t_idx, jogador in zip(pick_order, pool):
        times[t_idx].append(jogador)

    names = _pick_names(n_times)
    colors = TEAM_COLORS * ((n_times // len(TEAM_COLORS)) + 1)

    result = []
    for pos, (players, name, color) in enumerate(zip(times, names, colors), start=1):
        result.append(
            {
                "name": name,
                "color": color,
                "position": pos,
                "players": players,
                "skill_total": sum(p["skill_stars"] for p in players),
            }
        )

    return result, reserves
