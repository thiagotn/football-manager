import random

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
    Recebe lista de confirmados com player_id, skill_stars, position.
    Retorna (teams, reserves).

    players_per_team = jogadores de LINHA por time (exclui goleiro).
    Tamanho total de cada time = players_per_team + 1 (linha + goleiro).

    Algoritmo:
    1. Goleiros: 1 por time em snake draft por estrelas.
    2. Jogadores de linha por posição (lat, zag, mei, ata): distribui
       floor(N/n_times) por time em snake draft — garante equilíbrio de contagem.
       O índice snake continua globalmente entre grupos para manter equilíbrio de estrelas.
    3. Overflow (excedentes de posição + GKs extras) preenche slots restantes via snake.
    4. Jogadores além da capacidade total viram reservas.
    """
    team_size = players_per_team + 1
    n_times = len(confirmed) // team_size
    if n_times < 2:
        raise ValueError("Confirmados insuficientes para montar times.")

    # Shuffle para aleatorizar jogadores com mesmas estrelas entre sorteios
    pool = confirmed[:]
    random.shuffle(pool)

    # Separa por posição e ordena por estrelas desc dentro de cada grupo
    by_pos: dict[str, list[dict]] = {}
    for p in pool:
        pos = p.get("position") or "mei"
        by_pos.setdefault(pos, []).append(p)
    for pos in by_pos:
        by_pos[pos].sort(key=lambda p: p["skill_stars"], reverse=True)

    gks = by_pos.pop("gk", [])
    times: list[list[dict]] = [[] for _ in range(n_times)]
    overflow: list[dict] = []

    # Ciclo snake: [0,1,...,n-1, n-1,...,1,0] — cada time recebe exatamente 2 picks por ciclo
    snake = list(range(n_times)) + list(range(n_times - 1, -1, -1))
    si = 0  # índice global do snake — continua entre grupos para equilibrar estrelas

    # Passo 1: Goleiros — 1 por time, snake por estrelas
    for gk in gks[:n_times]:
        times[snake[si % len(snake)]].append(gk)
        si += 1
    overflow.extend(gks[n_times:])

    # Passo 2: Jogadores de linha — distribuição equilibrada por posição
    # Cada time recebe floor(N/n_times) jogadores de cada posição.
    # O restante (N % n_times) vai para overflow e é distribuído no passo 3.
    for pos in ("lat", "zag", "mei", "ata"):
        group = by_pos.get(pos, [])
        if not group:
            continue
        per_team = len(group) // n_times
        # Distribui per_team * n_times jogadores em snake draft contínuo
        for player in group[: per_team * n_times]:
            times[snake[si % len(snake)]].append(player)
            si += 1
        # Excedentes de posição vão para overflow
        overflow.extend(group[per_team * n_times :])

    # Passo 3: Overflow preenche slots restantes em snake draft (por estrelas)
    overflow.sort(key=lambda p: p["skill_stars"], reverse=True)
    remaining = [team_size - len(t) for t in times]
    total_needed = sum(remaining)

    for_dist = overflow[:total_needed]
    reserves = overflow[total_needed:]

    for player in for_dist:
        skips = 0
        while remaining[snake[si % len(snake)]] == 0 and skips < n_times * 2:
            si += 1
            skips += 1
        ti = snake[si % len(snake)]
        times[ti].append(player)
        remaining[ti] -= 1
        si += 1

    names = _pick_names(n_times)
    colors = TEAM_COLORS * ((n_times // len(TEAM_COLORS)) + 1)

    result = []
    for pos_idx, (players, name, color) in enumerate(zip(times, names, colors), start=1):
        result.append(
            {
                "name": name,
                "color": color,
                "position": pos_idx,
                "players": players,
                "skill_total": sum(p["skill_stars"] for p in players),
            }
        )

    return result, reserves
