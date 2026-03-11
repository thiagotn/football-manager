import random
from uuid import UUID

TEAM_NAMES = [
    "Corinthians da Várzea", "Flamengo do Asfalto", "Palmeiras de Chinelo",
    "Santos FC do Bairro", "Botafogo da Esquina", "Grêmio do Pé-Sujo",
    "Internacional do Calçadão", "Vasco da Gama do Povo",
    "América do Campinho", "Atlético Mineiro do Mato",
    "Cruzeiro da Galera", "Bahia do Batalhão", "Sport do Racha",
    "Vitória do Fim de Semana", "Fortaleza do Pelada",
    "Ceará do Campinho", "CSA do Baldão", "Náutico do Parque",
    "Ponte Preta do Povão", "Guarani do Bloco",
    "Unidos da Bola", "Estrela do Norte", "Leões do Asfalto",
    "Tubarões da Várzea", "Falcões do Bairro",
    "Dragões do Racha", "Lobos da Pelada",
    "Garotos do Campo", "Guerreiros do Baldão",
    "Alvinegros do Pé-Sujo", "Tricolores da Esquina",
    "Rubro-Negros do Campinho", "Azulões da Galera",
    "Vermelhão do Fim de Semana", "Bicho Solto FC",
    "Real Madruga", "Barcelusa", "Bahia de Munique", "Real da Resenha",
    "Atlético da Pelada", "União do Chopp", "Estrela do Bar", "Sporting da Galera",
    "Inter de Buteco", "Porto da Pelada", "Ajax da Laje", "Chelsea do Churrasco",
    "Borussia da Vila", "River da Resenha", "Boca da Vila", "Milan de Minas",
    "Juventus da Quebrada", "Bayer de Buteco", "Dinamo da Laje", "Racing da Resenha",
    "Estudiantes da Pelada", "Galácticos da Vila", "União da Bola Quadrada",
    "Atlético Pé de Rato", "Real Cansados", "Atlético Sem Fôlego", "Deportivo Ressaca",
    "Sporting Chinelinho", "União dos Últimos", "Real Varzeanos",
    "Atlético GPS Perdido", "Real Amigos do Bar", "Estrela da Pelada",
    "Atlético da Ladeira", "Real da Quebrada", "Sporting da Várzea",
    "Dinamo da Resenha", "União da Laje", "Real do Churrasco", "Atlético da Gelada",
    "Deportivo da Laje", "Barcemengo", "Flamchester", "Corinthester",
    "Santchester", "Palmilan", "Vaslona", "Gremadrid", "Flamadrid",
    "Galácticos da Saideira",
]

TEAM_COLORS = [
    "#e63946", "#2a9d8f", "#e9c46a", "#264653", "#f4a261",
    "#457b9d", "#6a4c93", "#1982c4", "#8ac926", "#ff595e",
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

    # Reservas são os jogadores excedentes (últimos da lista original)
    useful = confirmed[: n_times * team_size]
    reserves = confirmed[n_times * team_size :]

    goleiros = [p for p in useful if p["is_goalkeeper"]]
    nao_goleiros = [p for p in useful if not p["is_goalkeeper"]]

    times: list[list[dict]] = [[] for _ in range(n_times)]

    # 1. Distribui um goleiro por time
    for i, g in enumerate(goleiros[:n_times]):
        times[i].append(g)

    # Goleiros excedentes voltam ao pool de não-goleiros
    pool = sorted(
        nao_goleiros + goleiros[n_times:],
        key=lambda p: p["skill_stars"],
        reverse=True,
    )

    # 2. Snake draft nos restantes
    indices = list(range(n_times)) + list(range(n_times - 1, -1, -1))
    i = 0
    for jogador in pool:
        time_idx = indices[i % len(indices)]
        times[time_idx].append(jogador)
        i += 1

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
