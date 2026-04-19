import uuid

from fastapi import APIRouter

from app.core.dependencies import DB, CurrentPlayer
from app.core.exceptions import ForbiddenError, NotFoundError, ValidationError
from app.db.repositories.group_repo import GroupRepository
from app.db.repositories.match_repo import MatchRepository
from app.db.repositories.team_repo import TeamRepository
from app.models.group import GroupMemberRole
from app.models.player import PlayerRole
from app.schemas.team import TeamItem, TeamPlayerItem, TeamsResponse
from app.services.team_builder import build_teams

router = APIRouter(tags=["teams"])


def _serialize_teams(teams_db, group_member_skills: dict[uuid.UUID, dict]) -> TeamsResponse:
    """Converte MatchTeam[] do banco para TeamsResponse, usando skills de group_members."""
    teams_out = []
    reserves_out = []

    for team in teams_db:
        players_out = []
        for tp in team.players:
            skill_data = group_member_skills.get(tp.player_id, {})
            item = TeamPlayerItem(
                player_id=tp.player_id,
                name=tp.player.name,
                nickname=tp.player.nickname,
                skill_stars=skill_data.get("skill_stars", 2),
                position=skill_data.get("position", "mei"),
            )
            if tp.is_reserve:
                reserves_out.append(item)
            else:
                players_out.append(item)

        teams_out.append(TeamItem(
            id=team.id,
            name=team.name,
            color=team.color,
            position=team.position,
            skill_total=sum(p.skill_stars for p in players_out),
            players=players_out,
        ))

    return TeamsResponse(teams=teams_out, reserves=reserves_out)


@router.post("/matches/{match_id}/teams", response_model=TeamsResponse, status_code=201)
async def generate_teams(match_id: uuid.UUID, db: DB, current: CurrentPlayer):
    """Gera (ou regera) os times sorteados para a partida. Apenas admin do grupo."""
    m_repo = MatchRepository(db)
    match = await m_repo.get_with_attendances(match_id)
    if not match:
        raise NotFoundError("Partida não encontrada")

    g_repo = GroupRepository(db)
    if current.role != PlayerRole.ADMIN:
        member = await g_repo.get_member(match.group_id, current.id)
        if not member or member.role != GroupMemberRole.ADMIN:
            raise ForbiddenError("Apenas admins do grupo podem montar os times")

    if not match.players_per_team:
        raise ValidationError("Defina o número de jogadores por time antes de montar.")

    confirmed = await g_repo.get_confirmed_players_with_skills(match_id, match.group_id)

    # Tamanho real do time = linha + 1 goleiro. Mínimo: 2 times completos.
    min_needed = (match.players_per_team + 1) * 2
    if len(confirmed) < min_needed:
        raise ValidationError(
            f"São necessários pelo menos {min_needed} confirmados para montar os times "
            f"({match.players_per_team} de linha + 1 goleiro por time)."
        )

    group = await g_repo.get(match.group_id)
    team_slots = group.team_slots if group else None
    teams_data, reserves_data = build_teams(confirmed, match.players_per_team, team_slots=team_slots)

    # Sobrescreve times anteriores
    t_repo = TeamRepository(db)
    await t_repo.delete_by_match(match_id)

    # Cria um time "reservas" virtual (position=0) para facilitar o armazenamento
    # Opção mais simples: armazenar reservas como is_reserve=True no time 1
    # Mas vamos criar times reais e um time de reservas com position=0
    skill_map: dict[uuid.UUID, dict] = {
        p["player_id"]: {"skill_stars": p["skill_stars"], "position": p["position"]}
        for p in confirmed
    }

    for t in teams_data:
        team = await t_repo.create_team(
            match_id=match_id,
            name=t["name"],
            color=t["color"],
            position=t["position"],
        )
        for p in t["players"]:
            await t_repo.add_player(team.id, p["player_id"], is_reserve=False)

    # Reservas ficam num time especial position=0
    if reserves_data:
        reserve_team = await t_repo.create_team(
            match_id=match_id,
            name="Reservas",
            color=None,
            position=0,
        )
        for p in reserves_data:
            await t_repo.add_player(reserve_team.id, p["player_id"], is_reserve=True)

    # Recarrega para retornar resposta completa
    teams_db = await t_repo.get_by_match(match_id)
    active_teams = [t for t in teams_db if t.position > 0]
    result = _serialize_teams(active_teams, skill_map)
    # Adiciona reservas
    for p in reserves_data:
        result.reserves.append(TeamPlayerItem(
            player_id=p["player_id"],
            name=p["name"],
            nickname=p["nickname"],
            skill_stars=p["skill_stars"],
            position=p["position"],
        ))
    return result


@router.get("/matches/{match_id}/teams", response_model=TeamsResponse)
async def get_teams(match_id: uuid.UUID, db: DB):
    """Retorna os times gerados para a partida (público, sem autenticação obrigatória)."""
    m_repo = MatchRepository(db)
    match = await m_repo.get_with_attendances(match_id)
    if not match:
        raise NotFoundError("Partida não encontrada")

    t_repo = TeamRepository(db)
    teams_db = await t_repo.get_by_match(match_id)
    active_teams = [t for t in teams_db if t.position > 0]
    reserve_teams = [t for t in teams_db if t.position == 0]

    # Coleta todos os player_ids presentes nos times para buscar skills
    all_player_ids = [
        tp.player_id
        for team in teams_db
        for tp in team.players
    ]

    g_repo = GroupRepository(db)
    skill_map = await g_repo.get_member_skills(match.group_id, all_player_ids)

    result = _serialize_teams(active_teams, skill_map)
    # Adiciona reservas do time especial position=0
    for reserve_team in reserve_teams:
        for tp in reserve_team.players:
            skill_data = skill_map.get(tp.player_id, {})
            result.reserves.append(TeamPlayerItem(
                player_id=tp.player_id,
                name=tp.player.name,
                nickname=tp.player.nickname,
                skill_stars=skill_data.get("skill_stars", 2),
                position=skill_data.get("position", "mei"),
            ))
    return result
