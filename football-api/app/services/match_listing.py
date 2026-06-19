"""Regras de classificação 'partida atual' compartilhadas entre routers.

Uma partida é considerada `is_current` quando ainda merece destaque no card
de "Atuais" do dashboard / aba do grupo. A regra combina três sinais:

1. Status `open` ou `in_progress` (jogo ainda por vir / acontecendo).
2. Janela de votação não fechou (`voting_status != 'closed'`).
3. É a partida mais recente do grupo E não há partida futura criada
   ainda — evita que a aba "Atuais" fique vazia entre rachões.

Documentado em docs/prd/044 §17 (paridade v2→prod oficial) — a v2 (Go) deve
expor os mesmos campos quando atender o frontend de produção.
"""

from collections.abc import Iterable

from app.models.match import MatchStatus
from app.services.voting import voting_status as compute_voting_status


def classify_matches(matches: Iterable) -> dict[str, tuple[bool, str]]:
    """Para cada match na coleção, retorna (is_current, voting_status).

    `matches` deve ser a lista completa de partidas DO MESMO grupo — o cálculo
    do "é a mais recente sem próxima criada" precisa do contexto do grupo. Se
    matches vier de múltiplos grupos, agrupe antes (vide players.get_my_matches).
    """
    matches_list = list(matches)
    if not matches_list:
        return {}

    has_future = any(
        m.status in (MatchStatus.OPEN, MatchStatus.IN_PROGRESS) for m in matches_list
    )
    closed_matches = [m for m in matches_list if m.status == MatchStatus.CLOSED]
    most_recent_closed_id = None
    if closed_matches:
        # Ordena por (match_date, start_time) descendente — pega a última do grupo.
        most_recent_closed_id = max(
            closed_matches, key=lambda m: (m.match_date, m.start_time)
        ).id

    out: dict[str, tuple[bool, str]] = {}
    for m in matches_list:
        vstatus = compute_voting_status(m)
        if m.status in (MatchStatus.OPEN, MatchStatus.IN_PROGRESS):
            is_current = True
        elif vstatus in ("not_open", "open"):
            is_current = True
        elif m.id == most_recent_closed_id and not has_future:
            is_current = True
        else:
            is_current = False
        out[str(m.id)] = (is_current, vstatus)
    return out
