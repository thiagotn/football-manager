"""
Testes unitários — routers/groups.py

Regras de negócio cobertas:
- Limite de grupos por plano (free=1, basic=3, pro=10)
- Super admin é isento de qualquer limite
- Super admin não pode ser adicionado como membro de grupo
- Slug duplicado retorna 409
- GET /groups/{id} não encontrado → 404
- GET /groups/{id} não-membro → 403
- PATCH /groups/{id} não-admin do grupo → 403
- PATCH /groups/{id} não encontrado → 404
- DELETE /groups/{id} não encontrado → 404
- GET /groups/{id}/members não-membro → 403
"""
from unittest.mock import AsyncMock, MagicMock, patch
from uuid import uuid4

import pytest

from app.models.player import PlayerRole


# ── Helpers ───────────────────────────────────────────────────────────────────


def _make_subscription(plan: str = "free") -> MagicMock:
    sub = MagicMock()
    sub.plan = plan
    return sub


def _make_group(name: str = "Grupo Teste") -> MagicMock:
    g = MagicMock()
    g.id = uuid4()
    g.name = name
    g.slug = name.lower().replace(" ", "-")
    g.description = None
    g.per_match_amount = None
    g.monthly_amount = None
    g.recurrence_enabled = False
    g.is_public = True
    g.vote_open_delay_minutes = 20
    g.vote_duration_hours = 24
    g.timezone = "America/Sao_Paulo"
    g.created_at = "2026-01-01T00:00:00"
    g.updated_at = "2026-01-01T00:00:00"
    return g


GROUP_PAYLOAD = {"name": "Pelada do Bairro", "description": "Toda sexta"}


# ── Limite de plano — criar grupo ─────────────────────────────────────────────


@pytest.mark.asyncio
async def test_create_group_free_plan_limit_exceeded(api_client, mocker):
    """Usuário free que já tem 1 grupo não pode criar outro → 403 PLAN_LIMIT_EXCEEDED."""
    sub = _make_subscription("free")

    mocker.patch(
        "app.api.v1.routers.groups.SubscriptionRepository.get_or_create",
        new=AsyncMock(return_value=sub),
    )
    mocker.patch(
        "app.api.v1.routers.groups.SubscriptionRepository.count_admin_groups",
        new=AsyncMock(return_value=1),  # já atingiu o limite free (1)
    )

    response = await api_client.post("/api/v1/groups", json=GROUP_PAYLOAD)

    assert response.status_code == 403
    assert response.json()["detail"] == "PLAN_LIMIT_EXCEEDED"


@pytest.mark.asyncio
async def test_create_group_basic_plan_allows_up_to_three(api_client, mocker):
    """Usuário basic com 2 grupos pode criar o terceiro."""
    sub = _make_subscription("basic")
    group = _make_group("Pelada do Bairro")

    mocker.patch(
        "app.api.v1.routers.groups.SubscriptionRepository.get_or_create",
        new=AsyncMock(return_value=sub),
    )
    mocker.patch(
        "app.api.v1.routers.groups.SubscriptionRepository.count_admin_groups",
        new=AsyncMock(return_value=2),  # abaixo do limite basic (3)
    )
    mocker.patch(
        "app.api.v1.routers.groups.GroupRepository.get_by_slug",
        new=AsyncMock(return_value=None),
    )
    mocker.patch(
        "app.api.v1.routers.groups.GroupRepository.create",
        new=AsyncMock(return_value=group),
    )
    mocker.patch(
        "app.api.v1.routers.groups.GroupRepository.add_member",
        new=AsyncMock(return_value=None),
    )

    response = await api_client.post("/api/v1/groups", json=GROUP_PAYLOAD)

    assert response.status_code == 201


@pytest.mark.asyncio
async def test_create_group_admin_exempt_from_plan_limit(admin_client, mocker):
    """Super admin cria grupo mesmo sem verificar plano."""
    group = _make_group("Grupo Admin")

    mocker.patch(
        "app.api.v1.routers.groups.GroupRepository.get_by_slug",
        new=AsyncMock(return_value=None),
    )
    mocker.patch(
        "app.api.v1.routers.groups.GroupRepository.create",
        new=AsyncMock(return_value=group),
    )
    mocker.patch(
        "app.api.v1.routers.groups.GroupRepository.add_member",
        new=AsyncMock(return_value=None),
    )
    # SubscriptionRepository NÃO deve ser chamado para admin
    sub_mock = mocker.patch("app.api.v1.routers.groups.SubscriptionRepository")

    response = await admin_client.post("/api/v1/groups", json=GROUP_PAYLOAD)

    assert response.status_code == 201
    sub_mock.assert_not_called()


# ── Slug duplicado ────────────────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_create_group_duplicate_slug_returns_conflict(api_client, mocker):
    """Tentar criar grupo com slug já existente deve retornar 409."""
    sub = _make_subscription("free")
    existing_group = _make_group("Grupo Existente")

    mocker.patch(
        "app.api.v1.routers.groups.SubscriptionRepository.get_or_create",
        new=AsyncMock(return_value=sub),
    )
    mocker.patch(
        "app.api.v1.routers.groups.SubscriptionRepository.count_admin_groups",
        new=AsyncMock(return_value=0),
    )
    mocker.patch(
        "app.api.v1.routers.groups.GroupRepository.get_by_slug",
        new=AsyncMock(return_value=existing_group),  # slug já em uso
    )

    response = await api_client.post(
        "/api/v1/groups", json={**GROUP_PAYLOAD, "slug": "grupo-existente"}
    )

    assert response.status_code == 409


# ── Adicionar membro — super admin bloqueado ──────────────────────────────────


@pytest.mark.asyncio
async def test_add_member_rejects_super_admin(api_client, admin_user, mocker):
    """Tentar adicionar o super admin como membro de grupo deve retornar 403."""
    group_id = uuid4()

    group = _make_group("Pelada")
    group.id = group_id

    member_requester = MagicMock()
    member_requester.role = "admin"  # requester is group admin

    super_admin_player = MagicMock()
    super_admin_player.id = admin_user.id
    super_admin_player.role = PlayerRole.ADMIN

    mocker.patch(
        "app.api.v1.routers.groups.GroupRepository.get",
        new=AsyncMock(return_value=group),
    )
    mocker.patch(
        "app.api.v1.routers.groups.GroupRepository.get_member",
        new=AsyncMock(return_value=member_requester),
    )
    mocker.patch(
        "app.api.v1.routers.groups.PlayerRepository.get",
        new=AsyncMock(return_value=super_admin_player),
    )

    response = await api_client.post(
        f"/api/v1/groups/{group_id}/members",
        json={"player_id": str(admin_user.id)},
    )

    assert response.status_code == 403


# ── GET /groups/{id} ──────────────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_get_group_not_found_returns_404(api_client, mocker):
    mocker.patch(
        "app.api.v1.routers.groups.GroupRepository.get_with_members",
        new=AsyncMock(return_value=None),
    )

    response = await api_client.get(f"/api/v1/groups/{uuid4()}")

    assert response.status_code == 404


@pytest.mark.asyncio
async def test_get_group_non_member_returns_403(api_client, mocker):
    group = _make_group()
    mocker.patch(
        "app.api.v1.routers.groups.GroupRepository.get_with_members",
        new=AsyncMock(return_value=group),
    )
    mocker.patch(
        "app.api.v1.routers.groups.GroupRepository.get_member",
        new=AsyncMock(return_value=None),
    )

    response = await api_client.get(f"/api/v1/groups/{group.id}")

    assert response.status_code == 403


# ── PATCH /groups/{id} ────────────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_update_group_not_found_returns_404(api_client, mocker):
    mocker.patch(
        "app.api.v1.routers.groups.GroupRepository.get",
        new=AsyncMock(return_value=None),
    )

    response = await api_client.patch(
        f"/api/v1/groups/{uuid4()}",
        json={"name": "Novo Nome"},
    )

    assert response.status_code == 404


@pytest.mark.asyncio
async def test_update_group_non_group_admin_returns_403(api_client, mocker):
    group = _make_group()
    mocker.patch(
        "app.api.v1.routers.groups.GroupRepository.get",
        new=AsyncMock(return_value=group),
    )
    member = MagicMock()
    member.role = "member"  # não é admin do grupo
    mocker.patch(
        "app.api.v1.routers.groups.GroupRepository.get_member",
        new=AsyncMock(return_value=member),
    )

    response = await api_client.patch(
        f"/api/v1/groups/{group.id}",
        json={"name": "Novo Nome"},
    )

    assert response.status_code == 403


# ── DELETE /groups/{id} ───────────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_delete_group_not_found_returns_404(admin_client, mocker):
    mocker.patch(
        "app.api.v1.routers.groups.GroupRepository.get",
        new=AsyncMock(return_value=None),
    )

    response = await admin_client.delete(f"/api/v1/groups/{uuid4()}")

    assert response.status_code == 404


# ── GET /groups/{id}/members ──────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_list_members_non_member_returns_403(api_client, mocker):
    group = _make_group()
    mocker.patch(
        "app.api.v1.routers.groups.GroupRepository.get_with_members",
        new=AsyncMock(return_value=group),
    )
    mocker.patch(
        "app.api.v1.routers.groups.GroupRepository.get_member",
        new=AsyncMock(return_value=None),
    )

    response = await api_client.get(f"/api/v1/groups/{group.id}/members")

    assert response.status_code == 403


# ── GET /groups — happy path ──────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_list_groups_returns_200(api_client, mocker):
    """Jogador autenticado recebe lista dos seus grupos."""
    groups = [_make_group("Pelada A"), _make_group("Pelada B")]
    mocker.patch(
        "app.api.v1.routers.groups.GroupRepository.get_player_groups",
        new=AsyncMock(return_value=groups),
    )

    response = await api_client.get("/api/v1/groups")

    assert response.status_code == 200
    assert len(response.json()) == 2


# ── POST /groups — happy path ─────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_create_group_success_returns_201(api_client, mocker):
    """Criação bem-sucedida de grupo retorna 201."""
    sub = _make_subscription("basic")
    group = _make_group("Pelada do Bairro")

    mocker.patch(
        "app.api.v1.routers.groups.SubscriptionRepository.get_or_create",
        new=AsyncMock(return_value=sub),
    )
    mocker.patch(
        "app.api.v1.routers.groups.SubscriptionRepository.count_admin_groups",
        new=AsyncMock(return_value=0),
    )
    mocker.patch(
        "app.api.v1.routers.groups.GroupRepository.get_by_slug",
        new=AsyncMock(return_value=None),
    )
    mocker.patch(
        "app.api.v1.routers.groups.GroupRepository.create",
        new=AsyncMock(return_value=group),
    )
    mocker.patch(
        "app.api.v1.routers.groups.GroupRepository.add_member",
        new=AsyncMock(return_value=None),
    )

    response = await api_client.post("/api/v1/groups", json=GROUP_PAYLOAD)

    assert response.status_code == 201
    assert response.json()["name"] == group.name


# ── GET /groups/{id} — happy path ─────────────────────────────────────────────


@pytest.mark.asyncio
async def test_get_group_member_returns_200(api_client, player_user, mocker):
    """Membro do grupo pode ver os detalhes do grupo."""
    group = _make_group("Pelada")
    group.members = []
    group.description = "Toda sexta"
    group.slug = "pelada"
    group.per_match_amount = None
    group.monthly_amount = None
    group.recurrence_enabled = False
    group.is_public = True

    mocker.patch(
        "app.api.v1.routers.groups.GroupRepository.get_with_members",
        new=AsyncMock(return_value=group),
    )
    member = MagicMock()
    member.role = "member"
    mocker.patch(
        "app.api.v1.routers.groups.GroupRepository.get_member",
        new=AsyncMock(return_value=member),
    )

    response = await api_client.get(f"/api/v1/groups/{group.id}")

    assert response.status_code == 200
    assert response.json()["id"] == str(group.id)


# ── PATCH /groups/{id} — happy path ───────────────────────────────────────────


@pytest.mark.asyncio
async def test_update_group_success_returns_200(api_client, mocker):
    """Admin do grupo pode atualizar dados do grupo."""
    group = _make_group("Pelada")

    mocker.patch(
        "app.api.v1.routers.groups.GroupRepository.get",
        new=AsyncMock(return_value=group),
    )
    member = MagicMock()
    member.role = "admin"
    mocker.patch(
        "app.api.v1.routers.groups.GroupRepository.get_member",
        new=AsyncMock(return_value=member),
    )

    response = await api_client.patch(
        f"/api/v1/groups/{group.id}",
        json={"name": "Novo Nome"},
    )

    assert response.status_code == 200


# ── DELETE /groups/{id} — happy path ──────────────────────────────────────────


@pytest.mark.asyncio
async def test_delete_group_success_returns_204(admin_client, mocker):
    """Super admin pode deletar um grupo existente."""
    group = _make_group("Pelada")
    mocker.patch(
        "app.api.v1.routers.groups.GroupRepository.get",
        new=AsyncMock(return_value=group),
    )
    mocker.patch(
        "app.api.v1.routers.groups.GroupRepository.delete",
        new=AsyncMock(return_value=None),
    )

    response = await admin_client.delete(f"/api/v1/groups/{group.id}")

    assert response.status_code == 204


# ── GET /groups/{id}/members — happy path ─────────────────────────────────────


@pytest.mark.asyncio
async def test_list_members_success_returns_200(api_client, mocker):
    """Membro do grupo pode listar os membros."""
    group = _make_group("Pelada")
    group.members = []
    mocker.patch(
        "app.api.v1.routers.groups.GroupRepository.get_with_members",
        new=AsyncMock(return_value=group),
    )
    member = MagicMock()
    member.role = "member"
    mocker.patch(
        "app.api.v1.routers.groups.GroupRepository.get_member",
        new=AsyncMock(return_value=member),
    )

    response = await api_client.get(f"/api/v1/groups/{group.id}/members")

    assert response.status_code == 200
    assert response.json() == []


# ── POST /groups/{id}/members — happy path ────────────────────────────────────


@pytest.mark.asyncio
async def test_add_member_success_returns_201(api_client, mock_db, mocker):
    """Admin do grupo pode adicionar um membro."""
    from datetime import datetime
    from app.models.player import PlayerRole
    from app.models.group import GroupMemberRole

    group = _make_group("Pelada")
    group_id = group.id

    caller_member = MagicMock()
    caller_member.role = "admin"

    player_id = uuid4()
    player = MagicMock()
    player.id = player_id
    player.role = PlayerRole.PLAYER
    player.name = "Novo Membro"
    player.nickname = None
    player.whatsapp = "+5511999990099"
    player.avatar_url = None

    new_member = MagicMock()
    new_member.id = uuid4()
    new_member.group_id = group_id
    new_member.player_id = player_id
    new_member.role = GroupMemberRole.MEMBER
    new_member.skill_stars = 3
    new_member.position = "mei"
    new_member.created_at = datetime(2026, 1, 1)
    new_member.player = player

    mocker.patch(
        "app.api.v1.routers.groups.GroupRepository.get",
        new=AsyncMock(return_value=group),
    )
    mocker.patch(
        "app.api.v1.routers.groups.GroupRepository.get_member",
        new=AsyncMock(side_effect=[caller_member, None]),
    )
    mocker.patch(
        "app.api.v1.routers.groups.PlayerRepository.get",
        new=AsyncMock(return_value=player),
    )
    mocker.patch(
        "app.api.v1.routers.groups.SubscriptionRepository.get_or_create",
        new=AsyncMock(return_value=_make_subscription("pro")),
    )
    mocker.patch(
        "app.api.v1.routers.groups.GroupRepository.get_non_admin_member_ids",
        new=AsyncMock(return_value=[]),
    )
    mocker.patch(
        "app.api.v1.routers.groups.GroupRepository.add_member",
        new=AsyncMock(return_value=new_member),
    )
    mocker.patch(
        "app.api.v1.routers.groups.MatchRepository.get_active_matches",
        new=AsyncMock(return_value=[]),
    )
    mocker.patch(
        "app.api.v1.routers.groups.FinanceRepository.ensure_member_in_current_period",
        new=AsyncMock(return_value=None),
    )

    response = await api_client.post(
        f"/api/v1/groups/{group_id}/members",
        json={"player_id": str(player_id)},
    )

    assert response.status_code == 201


# ── PATCH /groups/{id}/members/{player_id} — happy path ──────────────────────


@pytest.mark.asyncio
async def test_update_member_skill_stars_returns_200(api_client, mocker):
    """Admin do grupo pode atualizar skill_stars de um membro."""
    from datetime import datetime
    from app.models.player import PlayerRole
    from app.models.group import GroupMemberRole

    group_id = uuid4()
    player_id = uuid4()

    player = MagicMock()
    player.id = player_id
    player.name = "Jogador"
    player.nickname = None
    player.avatar_url = None
    player.role = PlayerRole.PLAYER
    player.whatsapp = "+5511999990099"

    member = MagicMock()
    member.id = uuid4()
    member.group_id = group_id
    member.player_id = player_id
    member.role = GroupMemberRole.MEMBER
    member.skill_stars = 4
    member.position = "mei"
    member.created_at = datetime(2026, 1, 1)
    member.player = player

    caller = MagicMock()
    caller.role = GroupMemberRole.ADMIN

    mocker.patch(
        "app.api.v1.routers.groups.GroupRepository.get_member",
        new=AsyncMock(side_effect=[caller, member]),
    )

    response = await api_client.patch(
        f"/api/v1/groups/{group_id}/members/{player_id}",
        json={"skill_stars": 4},
    )

    assert response.status_code == 200


# ── PATCH /groups/{id}/members/me — self-service position ────────────────────


@pytest.mark.asyncio
async def test_update_my_position_returns_200(api_client, player_user, mocker):
    """Membro pode alterar sua própria posição no grupo."""
    from datetime import datetime
    from app.models.group import GroupMemberRole

    group_id = uuid4()

    member = MagicMock()
    member.id = uuid4()
    member.group_id = group_id
    member.player_id = player_user.id
    member.role = GroupMemberRole.MEMBER
    member.skill_stars = 3
    member.position = "gk"
    member.created_at = datetime(2026, 1, 1)
    member.player = player_user

    mocker.patch(
        "app.api.v1.routers.groups.GroupRepository.get_member",
        new=AsyncMock(return_value=member),
    )

    response = await api_client.patch(
        f"/api/v1/groups/{group_id}/members/me",
        json={"position": "gk"},
    )

    assert response.status_code == 200
    assert response.json()["position"] == "gk"


@pytest.mark.asyncio
async def test_update_my_position_not_member_returns_404(api_client, player_user, mocker):
    """Jogador que não é membro do grupo recebe 404."""
    group_id = uuid4()

    mocker.patch(
        "app.api.v1.routers.groups.GroupRepository.get_member",
        new=AsyncMock(return_value=None),
    )

    response = await api_client.patch(
        f"/api/v1/groups/{group_id}/members/me",
        json={"position": "ata"},
    )

    assert response.status_code == 404


# ── DELETE /groups/{id}/members/{player_id} — happy path ─────────────────────


@pytest.mark.asyncio
async def test_remove_member_success_returns_204(api_client, mocker):
    """Admin do grupo pode remover um membro."""
    group_id = uuid4()
    player_id = uuid4()

    caller = MagicMock()
    caller.role = "admin"
    member = MagicMock()

    mocker.patch(
        "app.api.v1.routers.groups.GroupRepository.get_member",
        new=AsyncMock(side_effect=[caller, member]),
    )
    mocker.patch(
        "app.api.v1.routers.groups.MatchRepository.delete_player_attendances_in_open_matches",
        new=AsyncMock(return_value=None),
    )
    mocker.patch(
        "app.api.v1.routers.groups.GroupRepository.delete",
        new=AsyncMock(return_value=None),
    )

    response = await api_client.delete(f"/api/v1/groups/{group_id}/members/{player_id}")

    assert response.status_code == 204


# ── POST /groups/{id}/waitlist — happy path ───────────────────────────────────


@pytest.mark.asyncio
async def test_join_waitlist_success_returns_201(api_client, player_user, mock_db, mocker):
    """Jogador não-membro pode entrar na fila de espera de um grupo público."""
    from datetime import date

    group_id = uuid4()
    group = _make_group("Pelada Pública")
    group.id = group_id
    group.is_public = True

    open_match = MagicMock()
    open_match.id = uuid4()
    open_match.max_players = None
    open_match.match_date = date(2026, 4, 1)

    entry = MagicMock()
    entry.id = uuid4()
    entry.match_id = open_match.id
    entry.player_id = player_user.id
    entry.intro = "Quero jogar!"
    entry.status = "pending"
    entry.created_at = "2026-04-01T10:00:00"
    entry.player = MagicMock()
    entry.player.name = player_user.name
    entry.player.nickname = None

    mocker.patch(
        "app.api.v1.routers.groups.GroupRepository.get",
        new=AsyncMock(return_value=group),
    )
    mocker.patch(
        "app.api.v1.routers.groups.GroupRepository.get_member",
        new=AsyncMock(return_value=None),  # não é membro
    )
    mocker.patch(
        "app.api.v1.routers.groups.MatchRepository.get_active_matches",
        new=AsyncMock(return_value=[open_match]),
    )
    mocker.patch(
        "app.api.v1.routers.groups.WaitlistRepository.get_entry",
        new=AsyncMock(return_value=None),
    )
    mocker.patch(
        "app.api.v1.routers.groups.WaitlistRepository.create",
        new=AsyncMock(return_value=entry),
    )
    # Sem admins no grupo para notificar
    result_mock = MagicMock()
    result_mock.scalars.return_value.all.return_value = []
    mock_db.execute = AsyncMock(return_value=result_mock)
    mocker.patch("app.api.v1.routers.groups.send_push", new=AsyncMock(return_value=None))

    response = await api_client.post(
        f"/api/v1/groups/{group_id}/waitlist",
        json={"agreed": True, "intro": "Quero jogar!"},
    )

    assert response.status_code == 201


# ── GET /groups/{id}/waitlist — admin only ────────────────────────────────────


@pytest.mark.asyncio
async def test_list_waitlist_non_group_admin_returns_403(api_client, mocker):
    """Apenas admin do grupo pode ver a lista de espera."""
    group = _make_group("Pelada")

    mocker.patch(
        "app.api.v1.routers.groups.GroupRepository.get",
        new=AsyncMock(return_value=group),
    )
    member = MagicMock()
    member.role = "member"  # não é admin do grupo
    mocker.patch(
        "app.api.v1.routers.groups.GroupRepository.get_member",
        new=AsyncMock(return_value=member),
    )

    response = await api_client.get(f"/api/v1/groups/{group.id}/waitlist")

    assert response.status_code == 403


@pytest.mark.asyncio
async def test_list_waitlist_admin_returns_200(api_client, mocker):
    """Admin do grupo pode listar candidatos da fila de espera."""
    group = _make_group("Pelada")

    mocker.patch(
        "app.api.v1.routers.groups.GroupRepository.get",
        new=AsyncMock(return_value=group),
    )
    member = MagicMock()
    member.role = "admin"
    mocker.patch(
        "app.api.v1.routers.groups.GroupRepository.get_member",
        new=AsyncMock(return_value=member),
    )
    mocker.patch(
        "app.api.v1.routers.groups.MatchRepository.get_active_matches",
        new=AsyncMock(return_value=[]),  # sem partida ativa → lista vazia
    )

    response = await api_client.get(f"/api/v1/groups/{group.id}/waitlist")

    assert response.status_code == 200
    assert response.json() == []


# ── GET /groups/{id}/waitlist/me ──────────────────────────────────────────────


@pytest.mark.asyncio
async def test_get_my_waitlist_entry_no_active_match_returns_none(api_client, mocker):
    """Sem partida ativa, endpoint retorna null (200)."""
    group = _make_group("Pelada")

    mocker.patch(
        "app.api.v1.routers.groups.GroupRepository.get",
        new=AsyncMock(return_value=group),
    )
    mocker.patch(
        "app.api.v1.routers.groups.MatchRepository.get_active_matches",
        new=AsyncMock(return_value=[]),
    )

    response = await api_client.get(f"/api/v1/groups/{group.id}/waitlist/me")

    assert response.status_code == 200
    assert response.json() is None


# ── PATCH /groups/{id}/waitlist/{entry_id} — accept/reject ───────────────────


@pytest.mark.asyncio
async def test_review_waitlist_non_group_admin_returns_403(api_client, mocker):
    """Apenas admin do grupo pode aceitar/rejeitar candidatos."""
    group = _make_group("Pelada")

    mocker.patch(
        "app.api.v1.routers.groups.GroupRepository.get",
        new=AsyncMock(return_value=group),
    )
    member = MagicMock()
    member.role = "member"
    mocker.patch(
        "app.api.v1.routers.groups.GroupRepository.get_member",
        new=AsyncMock(return_value=member),
    )

    response = await api_client.patch(
        f"/api/v1/groups/{group.id}/waitlist/{uuid4()}",
        json={"action": "accept"},
    )

    assert response.status_code == 403


# ── GET /groups/{id}/members/lookup ──────────────────────────────────────────


def _make_player(name: str = "Carlos Silva", whatsapp: str = "+5511999990002") -> MagicMock:
    p = MagicMock()
    p.id = uuid4()
    p.name = name
    p.nickname = "Carlão"
    p.avatar_url = None
    p.whatsapp = whatsapp
    p.role = PlayerRole.PLAYER
    return p


def _make_group_admin_member() -> MagicMock:
    m = MagicMock()
    m.role = "admin"
    return m


@pytest.mark.asyncio
async def test_lookup_found_not_member(api_client, mocker):
    """Número encontrado, jogador não é membro → status: found."""
    group = _make_group()
    player = _make_player()
    group_admin = _make_group_admin_member()

    mocker.patch("app.api.v1.routers.groups.GroupRepository.get", new=AsyncMock(return_value=group))
    mocker.patch("app.api.v1.routers.groups.GroupRepository.get_member", side_effect=[group_admin, None])
    mocker.patch("app.api.v1.routers.groups.PlayerRepository.get_by_whatsapp", new=AsyncMock(return_value=player))

    response = await api_client.get(
        f"/api/v1/groups/{group.id}/members/lookup?whatsapp=%2B5511999990002"
    )

    assert response.status_code == 200
    data = response.json()
    assert data["status"] == "found"
    assert data["player"]["name"] == player.name


@pytest.mark.asyncio
async def test_lookup_already_member(api_client, mocker):
    """Número encontrado, jogador já é membro → status: already_member."""
    group = _make_group()
    player = _make_player()
    group_admin = _make_group_admin_member()
    existing_member = MagicMock()

    mocker.patch("app.api.v1.routers.groups.GroupRepository.get", new=AsyncMock(return_value=group))
    mocker.patch("app.api.v1.routers.groups.GroupRepository.get_member", side_effect=[group_admin, existing_member])
    mocker.patch("app.api.v1.routers.groups.PlayerRepository.get_by_whatsapp", new=AsyncMock(return_value=player))

    response = await api_client.get(
        f"/api/v1/groups/{group.id}/members/lookup?whatsapp=%2B5511999990002"
    )

    assert response.status_code == 200
    assert response.json()["status"] == "already_member"


@pytest.mark.asyncio
async def test_lookup_not_found(api_client, mocker):
    """Número não encontrado → status: not_found, sem campo player."""
    group = _make_group()
    group_admin = _make_group_admin_member()

    mocker.patch("app.api.v1.routers.groups.GroupRepository.get", new=AsyncMock(return_value=group))
    mocker.patch("app.api.v1.routers.groups.GroupRepository.get_member", new=AsyncMock(return_value=group_admin))
    mocker.patch("app.api.v1.routers.groups.PlayerRepository.get_by_whatsapp", new=AsyncMock(return_value=None))

    response = await api_client.get(
        f"/api/v1/groups/{group.id}/members/lookup?whatsapp=%2B5511999990099"
    )

    assert response.status_code == 200
    data = response.json()
    assert data["status"] == "not_found"
    assert data.get("player") is None


@pytest.mark.asyncio
async def test_lookup_non_admin_returns_403(api_client, mocker):
    """Caller não é admin do grupo → 403."""
    group = _make_group()
    non_admin_member = MagicMock()
    non_admin_member.role = "member"

    mocker.patch("app.api.v1.routers.groups.GroupRepository.get", new=AsyncMock(return_value=group))
    mocker.patch("app.api.v1.routers.groups.GroupRepository.get_member", new=AsyncMock(return_value=non_admin_member))

    response = await api_client.get(
        f"/api/v1/groups/{group.id}/members/lookup?whatsapp=%2B5511999990002"
    )

    assert response.status_code == 403


# ── POST /groups/{id}/members/by-phone ───────────────────────────────────────


def _make_member_mock(player: MagicMock, skill_stars: int = 2, position: str = "mei") -> MagicMock:
    m = MagicMock()
    m.id = uuid4()
    m.player = player
    m.player_id = player.id
    m.role = "member"
    m.skill_stars = skill_stars
    m.position = position
    m.created_at = "2026-01-01T00:00:00"
    return m


@pytest.mark.asyncio
async def test_add_by_phone_existing_player_not_member(api_client, mocker):
    """Jogador existente, não é membro → 201, is_new=false."""
    group = _make_group()
    player = _make_player()
    group_admin = _make_group_admin_member()
    sub = _make_subscription("pro")
    new_member = _make_member_mock(player)

    mocker.patch("app.api.v1.routers.groups.GroupRepository.get", new=AsyncMock(return_value=group))
    mocker.patch("app.api.v1.routers.groups.GroupRepository.get_member", side_effect=[group_admin, None])
    mocker.patch("app.api.v1.routers.groups.PlayerRepository.get_by_whatsapp", new=AsyncMock(return_value=player))
    mocker.patch("app.api.v1.routers.groups.SubscriptionRepository.get_or_create", new=AsyncMock(return_value=sub))
    mocker.patch("app.api.v1.routers.groups.GroupRepository.get_non_admin_member_ids", new=AsyncMock(return_value=[]))
    mocker.patch("app.api.v1.routers.groups.GroupRepository.add_member", new=AsyncMock(return_value=new_member))
    mocker.patch("app.api.v1.routers.groups.MatchRepository.get_active_matches", new=AsyncMock(return_value=[]))
    mocker.patch("app.api.v1.routers.groups.FinanceRepository.ensure_member_in_current_period", new=AsyncMock())

    response = await api_client.post(
        f"/api/v1/groups/{group.id}/members/by-phone",
        json={"whatsapp": "+5511999990002", "skill_stars": 3, "position": "mei"},
    )

    assert response.status_code == 201
    data = response.json()
    assert data["is_new"] is False


@pytest.mark.asyncio
async def test_add_by_phone_existing_player_already_member(api_client, mocker):
    """Jogador existente, já é membro → 409."""
    group = _make_group()
    player = _make_player()
    group_admin = _make_group_admin_member()
    existing_member = MagicMock()

    mocker.patch("app.api.v1.routers.groups.GroupRepository.get", new=AsyncMock(return_value=group))
    mocker.patch("app.api.v1.routers.groups.GroupRepository.get_member", side_effect=[group_admin, existing_member])
    mocker.patch("app.api.v1.routers.groups.PlayerRepository.get_by_whatsapp", new=AsyncMock(return_value=player))

    response = await api_client.post(
        f"/api/v1/groups/{group.id}/members/by-phone",
        json={"whatsapp": "+5511999990002"},
    )

    assert response.status_code == 409


@pytest.mark.asyncio
async def test_add_by_phone_new_player_with_name(api_client, mocker):
    """Jogador novo com nome preenchido → 201, is_new=true."""
    group = _make_group()
    new_player = _make_player("Carlos Silva")
    new_player.must_change_password = True
    group_admin = _make_group_admin_member()
    sub = _make_subscription("pro")
    new_member = _make_member_mock(new_player)

    mocker.patch("app.api.v1.routers.groups.GroupRepository.get", new=AsyncMock(return_value=group))
    mocker.patch("app.api.v1.routers.groups.GroupRepository.get_member", new=AsyncMock(return_value=group_admin))
    mocker.patch("app.api.v1.routers.groups.PlayerRepository.get_by_whatsapp", new=AsyncMock(return_value=None))
    mocker.patch("app.api.v1.routers.groups.SubscriptionRepository.get_or_create", new=AsyncMock(return_value=sub))
    mocker.patch("app.api.v1.routers.groups.GroupRepository.get_non_admin_member_ids", new=AsyncMock(return_value=[]))
    mocker.patch("app.api.v1.routers.groups.PlayerRepository.create", new=AsyncMock(return_value=new_player))
    mocker.patch("app.api.v1.routers.groups.GroupRepository.add_member", new=AsyncMock(return_value=new_member))
    mocker.patch("app.api.v1.routers.groups.MatchRepository.get_active_matches", new=AsyncMock(return_value=[]))
    mocker.patch("app.api.v1.routers.groups.FinanceRepository.ensure_member_in_current_period", new=AsyncMock())

    response = await api_client.post(
        f"/api/v1/groups/{group.id}/members/by-phone",
        json={"whatsapp": "+5511999990099", "name": "Carlos Silva", "skill_stars": 2},
    )

    assert response.status_code == 201
    data = response.json()
    assert data["is_new"] is True


@pytest.mark.asyncio
async def test_add_by_phone_new_player_without_name_returns_422(api_client, mocker):
    """Jogador novo sem nome → 422."""
    group = _make_group()
    group_admin = _make_group_admin_member()

    mocker.patch("app.api.v1.routers.groups.GroupRepository.get", new=AsyncMock(return_value=group))
    mocker.patch("app.api.v1.routers.groups.GroupRepository.get_member", new=AsyncMock(return_value=group_admin))
    mocker.patch("app.api.v1.routers.groups.PlayerRepository.get_by_whatsapp", new=AsyncMock(return_value=None))

    response = await api_client.post(
        f"/api/v1/groups/{group.id}/members/by-phone",
        json={"whatsapp": "+5511999990099"},
    )

    assert response.status_code == 422


@pytest.mark.asyncio
async def test_add_by_phone_plan_limit_exceeded(api_client, mocker):
    """Limite de plano atingido → 403 PLAN_LIMIT_EXCEEDED."""
    group = _make_group()
    player = _make_player()
    group_admin = _make_group_admin_member()
    sub = _make_subscription("free")

    mocker.patch("app.api.v1.routers.groups.GroupRepository.get", new=AsyncMock(return_value=group))
    mocker.patch("app.api.v1.routers.groups.GroupRepository.get_member", side_effect=[group_admin, None])
    mocker.patch("app.api.v1.routers.groups.PlayerRepository.get_by_whatsapp", new=AsyncMock(return_value=player))
    mocker.patch("app.api.v1.routers.groups.SubscriptionRepository.get_or_create", new=AsyncMock(return_value=sub))
    mocker.patch(
        "app.api.v1.routers.groups.GroupRepository.get_non_admin_member_ids",
        new=AsyncMock(return_value=list(range(30))),  # free limit = 30
    )

    response = await api_client.post(
        f"/api/v1/groups/{group.id}/members/by-phone",
        json={"whatsapp": "+5511999990002"},
    )

    assert response.status_code == 403
    assert response.json()["detail"] == "PLAN_LIMIT_EXCEEDED"


@pytest.mark.asyncio
async def test_add_by_phone_non_admin_returns_403(api_client, mocker):
    """Caller não é admin do grupo → 403."""
    group = _make_group()
    non_admin_member = MagicMock()
    non_admin_member.role = "member"

    mocker.patch("app.api.v1.routers.groups.GroupRepository.get", new=AsyncMock(return_value=group))
    mocker.patch("app.api.v1.routers.groups.GroupRepository.get_member", new=AsyncMock(return_value=non_admin_member))

    response = await api_client.post(
        f"/api/v1/groups/{group.id}/members/by-phone",
        json={"whatsapp": "+5511999990002"},
    )

    assert response.status_code == 403
