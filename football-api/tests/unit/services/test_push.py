"""
Testes unitários — app/services/push.py (helper send_push_to_group_admins)

Regras cobertas:
- Faz fanout para todos os admins do grupo
- Pula o admin em `exclude` (quem executou a ação)
- No-op quando o grupo não tem admins
"""
from unittest.mock import AsyncMock, MagicMock, patch
from uuid import uuid4

import pytest

from app.services.push import send_push_to_group_admins


@pytest.mark.asyncio
async def test_send_push_to_group_admins_fans_out_to_all():
    group_id = uuid4()
    admin_a, admin_b, admin_c = uuid4(), uuid4(), uuid4()

    db = MagicMock()
    result = MagicMock()
    result.scalars.return_value.all.return_value = [admin_a, admin_b, admin_c]
    db.execute = AsyncMock(return_value=result)

    with patch("app.services.push.send_push", new=AsyncMock(return_value=None)) as mock_send:
        await send_push_to_group_admins(
            db, group_id, title="t", body="b", url="/u",
        )

    assert mock_send.await_count == 3
    notified = {call.args[1] for call in mock_send.await_args_list}
    assert notified == {admin_a, admin_b, admin_c}


@pytest.mark.asyncio
async def test_send_push_to_group_admins_excludes_actor():
    group_id = uuid4()
    actor = uuid4()
    other_a, other_b = uuid4(), uuid4()

    db = MagicMock()
    result = MagicMock()
    result.scalars.return_value.all.return_value = [actor, other_a, other_b]
    db.execute = AsyncMock(return_value=result)

    with patch("app.services.push.send_push", new=AsyncMock(return_value=None)) as mock_send:
        await send_push_to_group_admins(
            db, group_id, title="t", body="b", exclude=actor,
        )

    assert mock_send.await_count == 2
    notified = {call.args[1] for call in mock_send.await_args_list}
    assert actor not in notified
    assert notified == {other_a, other_b}


@pytest.mark.asyncio
async def test_send_push_to_group_admins_noop_when_no_admins():
    group_id = uuid4()

    db = MagicMock()
    result = MagicMock()
    result.scalars.return_value.all.return_value = []
    db.execute = AsyncMock(return_value=result)

    with patch("app.services.push.send_push", new=AsyncMock(return_value=None)) as mock_send:
        await send_push_to_group_admins(
            db, group_id, title="t", body="b",
        )

    mock_send.assert_not_awaited()


@pytest.mark.asyncio
async def test_send_push_to_group_admins_noop_when_only_actor_is_admin():
    """Se o único admin do grupo é o próprio actor, ninguém recebe push."""
    group_id = uuid4()
    actor = uuid4()

    db = MagicMock()
    result = MagicMock()
    result.scalars.return_value.all.return_value = [actor]
    db.execute = AsyncMock(return_value=result)

    with patch("app.services.push.send_push", new=AsyncMock(return_value=None)) as mock_send:
        await send_push_to_group_admins(
            db, group_id, title="t", body="b", exclude=actor,
        )

    mock_send.assert_not_awaited()
