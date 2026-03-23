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
