import math
import random

BIB_COLOR_HEX: dict[str, str] = {
    "laranja": "#f97316",
    "azul": "#3b82f6",
    "verde": "#22c55e",
    "vermelho": "#ef4444",
    "amarelo": "#eab308",
    "preto": "#1f2937",
    "branco": "#f1f5f9",
}

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


def _optimize_teams(times: list[list[dict]]) -> None:
    """
    Fase de otimização pós-distribuição: greedy swap para equalizar totais.

    Itera encontrando o melhor par de jogadores (um de cada time) cuja troca
    mais reduz a diferença entre o time com maior e menor skill_total.
    Para quando nenhuma troca melhora ou o spread fica ≤ 1.
    """
    for _ in range(500):
        totals = [sum(p["skill_stars"] for p in t) for t in times]
        current_spread = max(totals) - min(totals)
        if current_spread <= 1:
            break

        best_improvement = 0
        best_swap: tuple[int, int, int, int] | None = None

        n = len(times)
        gk_counts = [sum(1 for p in t if p.get("position") == "gk") for t in times]

        for a in range(n):
            for b in range(a + 1, n):
                for i, pa in enumerate(times[a]):
                    for j, pb in enumerate(times[b]):
                        if pa["skill_stars"] == pb["skill_stars"]:
                            continue

                        # GK constraint: não remover o único goleiro de um time
                        pa_is_gk = pa.get("position") == "gk"
                        pb_is_gk = pb.get("position") == "gk"
                        if pa_is_gk != pb_is_gk:
                            if pa_is_gk and gk_counts[a] <= 1:
                                continue  # time A perderia seu único GK
                            if pb_is_gk and gk_counts[b] <= 1:
                                continue  # time B perderia seu único GK

                        new_totals = totals[:]
                        new_totals[a] = totals[a] - pa["skill_stars"] + pb["skill_stars"]
                        new_totals[b] = totals[b] - pb["skill_stars"] + pa["skill_stars"]
                        new_spread = max(new_totals) - min(new_totals)
                        improvement = current_spread - new_spread
                        if improvement > best_improvement:
                            best_improvement = improvement
                            best_swap = (a, i, b, j)

        if best_swap is None:
            break

        a, i, b, j = best_swap
        times[a][i], times[b][j] = times[b][j], times[a][i]


def build_teams(
    confirmed: list[dict],
    players_per_team: int,
    team_slots: list[dict] | None = None,
) -> tuple[list[dict], list[dict]]:
    """
    Recebe lista de confirmados com player_id, skill_stars, position.
    Retorna (teams, reserves).

    players_per_team = jogadores de LINHA por time (exclui goleiro).
    Tamanho total de cada time = players_per_team + 1 (linha + goleiro).

    Algoritmo — distribuição por posição com sorteio em faixas de estrelas
    seguida de otimização greedy:
    1. Separa jogadores por posição.
    2. Goleiros: sorteia aleatoriamente qual time recebe cada um (1 por time).
    3. Para cada posição de linha (lat, zag, mei, ata):
       - Ordena por estrelas desc.
       - Distribui em rodadas de n_times jogadores por vez ("faixas").
       - Dentro de cada faixa embaralha aleatoriamente antes de atribuir
         um jogador a cada time — jogadores de nível similar vão para times
         diferentes, mas qual time recebe qual é sorteado.
    4. A soma de faixas por posição é limitada ao campo disponível no time
       (para evitar times maiores que team_size).
    5. Excedentes preenchem os slots restantes, distribuídos da mesma forma.
    6. Jogadores além da capacidade total viram reservas.
    7. Fase de otimização: greedy swap entre times para minimizar a diferença
       de skill_total entre o time mais forte e o mais fraco.
    """
    team_size = players_per_team + 1
    # ceil garante que todos os confirmados entram em algum time.
    # Com 39 jogadores e team_size=10 → 4 times (3 completos + 1 com 9).
    n_times = math.ceil(len(confirmed) / team_size)
    if n_times < 2:
        raise ValueError("Confirmados insuficientes para montar times.")

    # Embaralha para que jogadores com mesmas estrelas tenham ordem aleatória
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

    def assign_tiers(group: list[dict], per_team: int) -> None:
        """
        Distribui `per_team` rodadas de `group` (ordenado por estrelas desc)
        entre os times. A cada rodada, pega os próximos n_times jogadores
        (mesma faixa de estrelas), embaralha e atribui um a cada time.
        Isso garante que jogadores de nível similar vão para times diferentes,
        com o sorteio decidindo qual time recebe qual.
        Excedentes vão para overflow.
        """
        to_dist = group[: per_team * n_times]
        overflow.extend(group[per_team * n_times :])

        for round_num in range(per_team):
            tier = to_dist[round_num * n_times : (round_num + 1) * n_times]
            shuffled = tier[:]
            random.shuffle(shuffled)
            for team_idx, player in enumerate(shuffled):
                times[team_idx].append(player)

    # Passo 1: Goleiros — 1 por time, sorteado aleatoriamente
    gks_for_teams = gks[:n_times]
    random.shuffle(gks_for_teams)
    for team_idx, gk in enumerate(gks_for_teams):
        times[team_idx].append(gk)
    overflow.extend(gks[n_times:])

    # Passo 2: Calcula per_team por posição, limitado ao campo disponível
    #
    # A soma de todos os per_team não pode ultrapassar field_slots, senão os
    # times ficam maiores que team_size. Reduz iterativamente a posição mais
    # abundante até caber.
    field_slots = team_size - 1
    positions = ["lat", "zag", "mei", "ata"]

    pos_per_team: dict[str, int] = {
        pos: len(by_pos.get(pos, [])) // n_times
        for pos in positions
    }

    while sum(pos_per_team.values()) > field_slots:
        max_pos = max(
            (p for p in positions if pos_per_team[p] > 0),
            key=lambda p: pos_per_team[p],
        )
        pos_per_team[max_pos] -= 1

    # Passo 3: Distribui cada posição por faixas embaralhadas
    for pos in positions:
        group = by_pos.get(pos, [])
        assign_tiers(group, pos_per_team[pos])

    # Passo 4: Overflow preenche slots restantes
    overflow.sort(key=lambda p: p["skill_stars"], reverse=True)
    remaining = [team_size - len(t) for t in times]
    final_reserves: list[dict] = []

    # 4a: Goleiros excedentes só vão para times que ainda não têm GK;
    #     caso contrário viram reservas (evita 2 goleiros no mesmo time)
    overflow_gks = [p for p in overflow if p.get("position") == "gk"]
    overflow_field = [p for p in overflow if p.get("position") != "gk"]

    random.shuffle(overflow_gks)
    for gk in overflow_gks:
        ti = next(
            (
                i for i, t in enumerate(times)
                if remaining[i] > 0 and not any(p.get("position") == "gk" for p in t)
            ),
            None,
        )
        if ti is not None:
            times[ti].append(gk)
            remaining[ti] -= 1
        else:
            final_reserves.append(gk)

    # 4b: Jogadores de linha preenchem os slots restantes por faixas embaralhadas.
    # Embaralha antes de ordenar para que jogadores com mesmas estrelas tenham
    # ordem aleatória entre grupos de posição — evita que os mesmos jogadores
    # fracos sejam sempre reservas por chegarem sempre no fim do array.
    overflow_field_sorted = overflow_field[:]
    random.shuffle(overflow_field_sorted)
    overflow_field_sorted.sort(key=lambda p: p["skill_stars"], reverse=True)

    idx = 0
    while idx < len(overflow_field_sorted):
        open_teams = [i for i in range(n_times) if remaining[i] > 0]
        if not open_teams:
            break
        batch = overflow_field_sorted[idx : idx + len(open_teams)]
        shuffled = batch[:]
        random.shuffle(shuffled)
        for ti, player in zip(open_teams, shuffled):
            times[ti].append(player)
            remaining[ti] -= 1
        idx += len(batch)

    final_reserves.extend(overflow_field_sorted[idx:])
    reserves = final_reserves

    # Fase de otimização: equaliza skill_total entre os times via greedy swap
    _optimize_teams(times)

    random_names = _pick_names(n_times)
    default_colors = TEAM_COLORS * ((n_times // len(TEAM_COLORS)) + 1)

    # Determine if any slot is configured (at least one non-empty slot)
    has_slots = bool(team_slots)

    result = []
    for pos_idx, players in enumerate(times, start=1):
        i = pos_idx - 1
        slot = team_slots[i] if has_slots and i < len(team_slots) else None

        if slot is not None:
            slot_name = slot.get("name") if isinstance(slot, dict) else getattr(slot, "name", None)
            slot_color_slug = slot.get("color") if isinstance(slot, dict) else getattr(slot, "color", None)
        else:
            slot_name = None
            slot_color_slug = None

        # Name priority: custom name > "Time {Cor}" > random
        if slot_name:
            name = slot_name
        elif slot_color_slug:
            name = f"Time {slot_color_slug.capitalize()}"
        else:
            name = random_names[i]

        # Color priority: slot hex > default
        if slot_color_slug and slot_color_slug in BIB_COLOR_HEX:
            color = BIB_COLOR_HEX[slot_color_slug]
        else:
            color = default_colors[i]

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
