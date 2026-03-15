from datetime import date
from decimal import Decimal
from uuid import UUID

from fastapi import APIRouter, HTTPException, status

from app.core.dependencies import DB, CurrentPlayer
from app.core.exceptions import ForbiddenError, NotFoundError
from app.db.repositories.finance_repo import FinanceRepository
from app.db.repositories.group_repo import GroupRepository
from app.models.finance import FinancePeriod
from app.models.group import GroupMemberRole
from app.models.player import PlayerRole
from app.schemas.finance import (
    FinancePaymentResponse,
    FinancePeriodResponse,
    FinanceSummary,
    MarkPaymentRequest,
    PeriodListItem,
)

router = APIRouter(tags=["finance"])

_HISTORY_MONTHS: dict[str, int] = {
    "free": 3,
    "basic": 12,
    "pro": 9999,
}


def _build_summary(payments) -> FinanceSummary:
    active = [p for p in payments if p.status != "excluded"]
    paid = [p for p in active if p.status == "paid"]
    pending = [p for p in active if p.status == "pending"]
    received = sum(p.amount_due or 0 for p in paid)
    total = len(active)
    compliance = round(len(paid) / total * 100) if total > 0 else 0
    return FinanceSummary(
        received_cents=received,
        pending_count=len(pending),
        paid_count=len(paid),
        total_members=total,
        compliance_pct=compliance,
    )


async def _get_group_and_check_member(group_id: UUID, current: CurrentPlayer, db: DB):
    repo = GroupRepository(db)
    group = await repo.get(group_id)
    if not group:
        raise NotFoundError("Grupo não encontrado")
    if current.role == PlayerRole.ADMIN:
        return group, None
    member = await repo.get_member(group_id, current.id)
    if not member:
        raise ForbiddenError("Você não é membro deste grupo")
    return group, member


async def _require_group_admin(group_id: UUID, current: CurrentPlayer, db: DB):
    group, member = await _get_group_and_check_member(group_id, current, db)
    if current.role != PlayerRole.ADMIN:
        if not member or member.role != GroupMemberRole.ADMIN:
            raise ForbiddenError("Apenas o admin do grupo pode realizar esta ação")
    return group


@router.get("/groups/{group_id}/finance/periods", response_model=list[PeriodListItem])
async def list_periods(group_id: UUID, db: DB, current: CurrentPlayer):
    await _get_group_and_check_member(group_id, current, db)
    repo = FinanceRepository(db)
    periods = await repo.list_periods(group_id)
    return [PeriodListItem(id=p.id, year=p.year, month=p.month) for p in periods]


@router.get(
    "/groups/{group_id}/finance/periods/{year}/{month}",
    response_model=FinancePeriodResponse,
)
async def get_period(
    group_id: UUID, year: int, month: int, db: DB, current: CurrentPlayer
):
    await _get_group_and_check_member(group_id, current, db)

    now = date.today()
    is_current = year == now.year and month == now.month

    repo = FinanceRepository(db)

    if is_current:
        await repo.get_or_create_period(group_id, year, month)

    period = await repo.get_period_with_payments(group_id, year, month)
    if not period:
        raise NotFoundError("Período não encontrado")

    payments = sorted(
        period.payments,
        key=lambda p: (0 if p.status == "pending" else 1, p.player_name.lower()),
    )
    summary = _build_summary(payments)

    return FinancePeriodResponse(
        period_id=period.id,
        year=period.year,
        month=period.month,
        summary=summary,
        payments=[
            FinancePaymentResponse(
                id=p.id,
                player_id=p.player_id,
                player_name=p.player_name,
                payment_type=p.payment_type,
                amount_due=p.amount_due,
                status=p.status,
                paid_at=p.paid_at,
            )
            for p in payments
        ],
    )


@router.patch("/finance/payments/{payment_id}", response_model=FinancePaymentResponse)
async def update_payment(
    payment_id: UUID, body: MarkPaymentRequest, db: DB, current: CurrentPlayer
):
    repo = FinanceRepository(db)
    payment = await repo.get_payment(payment_id)
    if not payment:
        raise NotFoundError("Registro de pagamento não encontrado")

    period = await db.get(FinancePeriod, payment.period_id)
    group = await _require_group_admin(period.group_id, current, db)

    if body.status == "paid":
        if not body.payment_type:
            raise HTTPException(
                status_code=status.HTTP_422_UNPROCESSABLE_ENTITY,
                detail="payment_type é obrigatório ao marcar como pago",
            )
        if body.payment_type == "monthly":
            amount = int((group.monthly_amount or Decimal("0")) * 100)
        else:
            amount = int((group.per_match_amount or Decimal("0")) * 100)

        payment = await repo.mark_paid(payment, body.payment_type, amount)
    else:
        payment = await repo.mark_pending(payment)

    return FinancePaymentResponse(
        id=payment.id,
        player_id=payment.player_id,
        player_name=payment.player_name,
        payment_type=payment.payment_type,
        amount_due=payment.amount_due,
        status=payment.status,
        paid_at=payment.paid_at,
    )
