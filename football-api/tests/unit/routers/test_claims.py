"""
Testes unitários — routers/claims.py + geração de claim-invite (routers/groups.py)

Regras de negócio cobertas:
- POST /groups/{id}/members/{player_id}/claim-invite: 201 admin, 403 não-admin,
  409 PLAYER_NOT_PENDING, 404 não-membro
- GET /claims/{token}: 200 válido, 404 inexistente/purpose errado, 403 usado/expirado,
  403 ALREADY_CLAIMED
- POST /claims/{token}/send-otp: 409 WHATSAPP_TAKEN (número de outro player),
  200 com número do próprio alvo
- POST /claims/{token}/verify-otp: 422 OTP inválido, 200 devolve otp_token
- POST /claims/{token}/complete: 200 atualiza whatsapp+senha, limpa flags, marca used,
  revoga refresh tokens; 401 otp_token inválido; 409 WHATSAPP_TAKEN
"""
from datetime import datetime, timedelta, timezone
from unittest.mock import AsyncMock, MagicMock
from uuid import uuid4

import pytest

from app.models.player import PlayerRole


# ── Helpers ───────────────────────────────────────────────────────────────────


def _make_claim_invite(
    used: bool = False,
    expired: bool = False,
    purpose: str = "registration_claim",
    target_player_id=None,
) -> MagicMock:
    inv = MagicMock()
    inv.id = uuid4()
    inv.group_id = uuid4()
    inv.token = "claimtoken123"
    inv.used = used
    inv.purpose = purpose
    inv.target_player_id = target_player_id or uuid4()
    inv.used_by_id = None
    inv.expires_at = (
        datetime.now(timezone.utc) - timedelta(hours=1)
        if expired
        else datetime.now(timezone.utc) + timedelta(days=7)
    )
    return inv


def _make_pending_player(whatsapp: str = "+5500000000000") -> MagicMock:
    p = MagicMock()
    p.id = uuid4()
    p.name = "Fulano de Tal"
    p.nickname = None
    p.whatsapp = whatsapp
    p.role = PlayerRole.PLAYER
    p.active = True
    p.must_change_password = True
    p.pending_registration = True
    p.avatar_url = None
    p.chat_enabled = False
    return p


def _make_group(name: str = "Rachão da Firma") -> MagicMock:
    g = MagicMock()
    g.id = uuid4()
    g.name = name
    return g


def _patch_claim_lookup(mocker, invite, player):
    mocker.patch(
        "app.api.v1.routers.claims.InviteRepository.get_by_token",
        new=AsyncMock(return_value=invite),
    )
    mocker.patch(
        "app.api.v1.routers.claims.PlayerRepository.get",
        new=AsyncMock(return_value=player),
    )


# ── POST /groups/{id}/members/{player_id}/claim-invite ────────────────────────


@pytest.mark.asyncio
async def test_create_claim_invite_as_group_admin(api_client, mocker):
    """Admin do grupo gera claim-invite para membro pendente → 201 com token e +7 dias."""
    group = _make_group()
    target = _make_pending_player()
    group_admin = MagicMock()
    group_admin.role = "admin"

    invite = _make_claim_invite(target_player_id=target.id)
    invite.group_id = group.id

    mocker.patch("app.api.v1.routers.groups.GroupRepository.get", new=AsyncMock(return_value=group))
    mocker.patch(
        "app.api.v1.routers.groups.GroupRepository.get_member",
        side_effect=[group_admin, MagicMock()],
    )
    mocker.patch("app.api.v1.routers.groups.PlayerRepository.get", new=AsyncMock(return_value=target))
    mock_create = mocker.patch(
        "app.api.v1.routers.groups.InviteRepository.create",
        new=AsyncMock(return_value=invite),
    )

    response = await api_client.post(f"/api/v1/groups/{group.id}/members/{target.id}/claim-invite")

    assert response.status_code == 201
    data = response.json()
    assert data["token"] == "claimtoken123"
    kwargs = mock_create.call_args.kwargs
    assert kwargs["purpose"] == "registration_claim"
    assert kwargs["target_player_id"] == target.id
    delta = kwargs["expires_at"] - datetime.now(timezone.utc)
    assert timedelta(days=6, hours=23) < delta < timedelta(days=7, hours=1)


@pytest.mark.asyncio
async def test_create_claim_invite_forbidden_non_admin(api_client, mocker):
    """Membro comum do grupo não pode gerar claim-invite → 403."""
    group = _make_group()
    member = MagicMock()
    member.role = "member"

    mocker.patch("app.api.v1.routers.groups.GroupRepository.get", new=AsyncMock(return_value=group))
    mocker.patch("app.api.v1.routers.groups.GroupRepository.get_member", new=AsyncMock(return_value=member))

    response = await api_client.post(f"/api/v1/groups/{group.id}/members/{uuid4()}/claim-invite")

    assert response.status_code == 403


@pytest.mark.asyncio
async def test_create_claim_invite_conflict_not_pending(api_client, mocker):
    """Jogador com cadastro completo não pode ser alvo de claim → 409 PLAYER_NOT_PENDING."""
    group = _make_group()
    target = _make_pending_player()
    target.pending_registration = False
    group_admin = MagicMock()
    group_admin.role = "admin"

    mocker.patch("app.api.v1.routers.groups.GroupRepository.get", new=AsyncMock(return_value=group))
    mocker.patch(
        "app.api.v1.routers.groups.GroupRepository.get_member",
        side_effect=[group_admin, MagicMock()],
    )
    mocker.patch("app.api.v1.routers.groups.PlayerRepository.get", new=AsyncMock(return_value=target))

    response = await api_client.post(f"/api/v1/groups/{group.id}/members/{target.id}/claim-invite")

    assert response.status_code == 409
    assert response.json()["detail"] == "PLAYER_NOT_PENDING"


@pytest.mark.asyncio
async def test_create_claim_invite_target_not_member(api_client, mocker):
    """Jogador que não é membro do grupo → 404."""
    group = _make_group()
    group_admin = MagicMock()
    group_admin.role = "admin"

    mocker.patch("app.api.v1.routers.groups.GroupRepository.get", new=AsyncMock(return_value=group))
    mocker.patch(
        "app.api.v1.routers.groups.GroupRepository.get_member",
        side_effect=[group_admin, None],
    )

    response = await api_client.post(f"/api/v1/groups/{group.id}/members/{uuid4()}/claim-invite")

    assert response.status_code == 404


# ── GET /claims/{token} ───────────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_claim_info_ok(api_client, mocker):
    invite = _make_claim_invite()
    player = _make_pending_player()
    _patch_claim_lookup(mocker, invite, player)
    mocker.patch(
        "app.api.v1.routers.claims.GroupRepository.get",
        new=AsyncMock(return_value=_make_group()),
    )

    response = await api_client.get("/api/v1/claims/claimtoken123")

    assert response.status_code == 200
    data = response.json()
    assert data["valid"] is True
    assert data["player_first_name"] == "Fulano"
    assert data["group_name"] == "Rachão da Firma"


@pytest.mark.asyncio
async def test_claim_info_not_found(api_client, mocker):
    mocker.patch(
        "app.api.v1.routers.claims.InviteRepository.get_by_token",
        new=AsyncMock(return_value=None),
    )

    response = await api_client.get("/api/v1/claims/naoexiste")

    assert response.status_code == 404


@pytest.mark.asyncio
async def test_claim_info_wrong_purpose_returns_404(api_client, mocker):
    """Token de convite de grupo (group_join) não funciona no fluxo de claim."""
    invite = _make_claim_invite(purpose="group_join")
    mocker.patch(
        "app.api.v1.routers.claims.InviteRepository.get_by_token",
        new=AsyncMock(return_value=invite),
    )

    response = await api_client.get("/api/v1/claims/claimtoken123")

    assert response.status_code == 404


@pytest.mark.asyncio
async def test_claim_info_used_returns_403(api_client, mocker):
    invite = _make_claim_invite(used=True)
    mocker.patch(
        "app.api.v1.routers.claims.InviteRepository.get_by_token",
        new=AsyncMock(return_value=invite),
    )

    response = await api_client.get("/api/v1/claims/claimtoken123")

    assert response.status_code == 403
    assert response.json()["detail"] == "INVITE_USED"


@pytest.mark.asyncio
async def test_claim_info_expired_returns_403(api_client, mocker):
    invite = _make_claim_invite(expired=True)
    mocker.patch(
        "app.api.v1.routers.claims.InviteRepository.get_by_token",
        new=AsyncMock(return_value=invite),
    )

    response = await api_client.get("/api/v1/claims/claimtoken123")

    assert response.status_code == 403
    assert response.json()["detail"] == "INVITE_EXPIRED"


@pytest.mark.asyncio
async def test_claim_info_already_claimed_returns_403(api_client, mocker):
    """Alvo já completou o cadastro (pending_registration=False) → 403 ALREADY_CLAIMED."""
    invite = _make_claim_invite()
    player = _make_pending_player()
    player.pending_registration = False
    _patch_claim_lookup(mocker, invite, player)

    response = await api_client.get("/api/v1/claims/claimtoken123")

    assert response.status_code == 403
    assert response.json()["detail"] == "ALREADY_CLAIMED"


# ── POST /claims/{token}/send-otp ─────────────────────────────────────────────


@pytest.mark.asyncio
async def test_claim_send_otp_rejects_whatsapp_of_other_player(api_client, mocker):
    """Número já pertence a OUTRO player → 409 WHATSAPP_TAKEN, sem enviar OTP."""
    invite = _make_claim_invite()
    player = _make_pending_player()
    other = _make_pending_player(whatsapp="+5511988887777")
    _patch_claim_lookup(mocker, invite, player)
    mocker.patch(
        "app.api.v1.routers.claims.PlayerRepository.get_by_whatsapp",
        new=AsyncMock(return_value=other),
    )
    mock_send = mocker.patch(
        "app.api.v1.routers.claims.twilio_verify.send_otp",
        new=AsyncMock(),
    )

    response = await api_client.post(
        "/api/v1/claims/claimtoken123/send-otp",
        json={"whatsapp": "+5511988887777"},
    )

    assert response.status_code == 409
    assert response.json()["detail"] == "WHATSAPP_TAKEN"
    mock_send.assert_not_called()


@pytest.mark.asyncio
async def test_claim_send_otp_allows_own_current_number(api_client, mocker):
    """Número atual do próprio alvo (caso: número era real, só falta senha) → 200."""
    invite = _make_claim_invite()
    player = _make_pending_player(whatsapp="+5511999991234")
    _patch_claim_lookup(mocker, invite, player)
    mocker.patch(
        "app.api.v1.routers.claims.PlayerRepository.get_by_whatsapp",
        new=AsyncMock(return_value=player),
    )
    mock_send = mocker.patch(
        "app.api.v1.routers.claims.twilio_verify.send_otp",
        new=AsyncMock(),
    )

    response = await api_client.post(
        "/api/v1/claims/claimtoken123/send-otp",
        json={"whatsapp": "+5511999991234"},
    )

    assert response.status_code == 200
    assert response.json()["status"] == "pending"
    mock_send.assert_called_once_with("+5511999991234")


@pytest.mark.asyncio
async def test_claim_send_otp_invalid_whatsapp_returns_422(api_client, mocker):
    """Número fora do E.164 estrito é rejeitado pelo schema → 422."""
    invite = _make_claim_invite()
    player = _make_pending_player()
    _patch_claim_lookup(mocker, invite, player)

    response = await api_client.post(
        "/api/v1/claims/claimtoken123/send-otp",
        json={"whatsapp": "+0000"},
    )

    assert response.status_code == 422


# ── POST /claims/{token}/verify-otp ───────────────────────────────────────────


@pytest.mark.asyncio
async def test_claim_verify_otp_invalid_code_returns_422(api_client, mocker):
    invite = _make_claim_invite()
    player = _make_pending_player()
    _patch_claim_lookup(mocker, invite, player)
    mocker.patch(
        "app.api.v1.routers.claims.PlayerRepository.get_by_whatsapp",
        new=AsyncMock(return_value=None),
    )
    mocker.patch(
        "app.api.v1.routers.claims.twilio_verify.check_otp",
        new=AsyncMock(return_value=False),
    )

    response = await api_client.post(
        "/api/v1/claims/claimtoken123/verify-otp",
        json={"whatsapp": "+5511999991234", "otp_code": "000000"},
    )

    assert response.status_code == 422
    assert response.json()["detail"] == "OTP_INVALID"


@pytest.mark.asyncio
async def test_claim_verify_otp_success_returns_otp_token(api_client, mocker):
    invite = _make_claim_invite()
    player = _make_pending_player()
    _patch_claim_lookup(mocker, invite, player)
    mocker.patch(
        "app.api.v1.routers.claims.PlayerRepository.get_by_whatsapp",
        new=AsyncMock(return_value=None),
    )
    mocker.patch(
        "app.api.v1.routers.claims.twilio_verify.check_otp",
        new=AsyncMock(return_value=True),
    )

    response = await api_client.post(
        "/api/v1/claims/claimtoken123/verify-otp",
        json={"whatsapp": "+5511999991234", "otp_code": "123456"},
    )

    assert response.status_code == 200
    assert response.json()["otp_token"]


# ── POST /claims/{token}/complete ─────────────────────────────────────────────


@pytest.mark.asyncio
async def test_claim_complete_updates_whatsapp_password_and_clears_flags(api_client, mocker):
    """Claim completo: atualiza credenciais, limpa flags, marca convite usado e loga."""
    invite = _make_claim_invite()
    player = _make_pending_player()
    _patch_claim_lookup(mocker, invite, player)
    mocker.patch(
        "app.api.v1.routers.claims.decode_otp_token",
        return_value="+5511999991234",
    )
    mocker.patch(
        "app.api.v1.routers.claims.PlayerRepository.get_by_whatsapp",
        new=AsyncMock(return_value=None),
    )
    mock_revoke = mocker.patch(
        "app.api.v1.routers.claims.RefreshTokenRepository.revoke_all_for_player",
        new=AsyncMock(),
    )
    mocker.patch(
        "app.api.v1.routers.claims.RefreshTokenRepository.create",
        new=AsyncMock(return_value="refresh-token-novo"),
    )

    response = await api_client.post(
        "/api/v1/claims/claimtoken123/complete",
        json={
            "whatsapp": "+5511999991234",
            "otp_token": "otp-jwt-valido",
            "password": "senhaNova123",
        },
    )

    assert response.status_code == 200
    data = response.json()
    assert "access_token" in data
    assert data["refresh_token"] == "refresh-token-novo"
    assert data["must_change_password"] is False

    assert player.whatsapp == "+5511999991234"
    assert player.pending_registration is False
    assert player.must_change_password is False
    assert invite.used is True
    assert invite.used_by_id == player.id
    mock_revoke.assert_called_once_with(player.id)


@pytest.mark.asyncio
async def test_claim_complete_invalid_otp_token_returns_401(api_client, mocker):
    invite = _make_claim_invite()
    player = _make_pending_player()
    _patch_claim_lookup(mocker, invite, player)
    mocker.patch(
        "app.api.v1.routers.claims.decode_otp_token",
        return_value=None,
    )

    response = await api_client.post(
        "/api/v1/claims/claimtoken123/complete",
        json={
            "whatsapp": "+5511999991234",
            "otp_token": "invalido",
            "password": "senhaNova123",
        },
    )

    assert response.status_code == 401


@pytest.mark.asyncio
async def test_claim_complete_whatsapp_taken_returns_409(api_client, mocker):
    """Número reivindicado já pertence a outro player → 409 WHATSAPP_TAKEN."""
    invite = _make_claim_invite()
    player = _make_pending_player()
    other = _make_pending_player(whatsapp="+5511999991234")
    _patch_claim_lookup(mocker, invite, player)
    mocker.patch(
        "app.api.v1.routers.claims.decode_otp_token",
        return_value="+5511999991234",
    )
    mocker.patch(
        "app.api.v1.routers.claims.PlayerRepository.get_by_whatsapp",
        new=AsyncMock(return_value=other),
    )

    response = await api_client.post(
        "/api/v1/claims/claimtoken123/complete",
        json={
            "whatsapp": "+5511999991234",
            "otp_token": "otp-jwt-valido",
            "password": "senhaNova123",
        },
    )

    assert response.status_code == 409
    assert response.json()["detail"] == "WHATSAPP_TAKEN"
