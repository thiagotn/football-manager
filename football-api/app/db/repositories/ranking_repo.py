"""
Repositório de ranking geral da plataforma.

Regras:
- Apenas partidas com ao menos 10 presenças confirmadas contam (D2).
  `eligible_voters` não é coluna persistida em `matches`, por isso usamos
  subquery de contagem de `attendances` com status='confirmed'.
- Período: filtra por `match_votes.submitted_at` via date_trunc.
- Exclui super admin (PlayerRole.ADMIN) dos resultados.
- Top: soma de pontos de `match_vote_top5` por jogador, limitado a 10.
- Flop: contagem de votos em `match_vote_flop` por jogador, limitado a 10.
- Empates recebem a mesma posição.
"""
from typing import Literal

from sqlalchemy import func, select
from sqlalchemy.ext.asyncio import AsyncSession

from app.models.match import Attendance, AttendanceStatus
from app.models.match_vote import MatchVote, MatchVoteFlop, MatchVoteTop5
from app.models.player import Player, PlayerRole

# Número mínimo de presenças confirmadas para que a partida conte no ranking
MIN_ELIGIBLE_VOTERS = 10


class RankingRepository:
    def __init__(self, session: AsyncSession):
        self.session = session

    def _eligible_match_ids_subquery(self):
        """
        Retorna subquery de IDs de partidas com ao menos MIN_ELIGIBLE_VOTERS
        presenças confirmadas.
        """
        return (
            select(Attendance.match_id)
            .where(Attendance.status == AttendanceStatus.CONFIRMED)
            .group_by(Attendance.match_id)
            .having(func.count(Attendance.id) >= MIN_ELIGIBLE_VOTERS)
            .scalar_subquery()
        )

    def _period_filter(self, period: Literal["month", "year", "all"]):
        if period == "month":
            return MatchVote.submitted_at >= func.date_trunc("month", func.now())
        if period == "year":
            return MatchVote.submitted_at >= func.date_trunc("year", func.now())
        return None  # all-time: sem filtro

    def _assign_positions(self, rows: list, score_attr: str) -> list[dict]:
        """Atribui posições com suporte a empate (mesmo score → mesma posição)."""
        result = []
        prev_score = None
        pos = 0
        rank = 0
        for row in rows:
            rank += 1
            score = getattr(row, score_attr)
            if score != prev_score:
                pos = rank
                prev_score = score
            result.append({
                "position": pos,
                "player_id": row.player_id,
                "name": row.name,
                "nickname": row.nickname,
                "avatar_url": row.avatar_url,
                score_attr: score,
            })
        return result

    async def get_top(self, period: Literal["month", "year", "all"]) -> list[dict]:
        """
        Retorna os top 10 jogadores por soma de pontos recebidos em votações,
        considerando apenas partidas elegíveis.
        """
        eligible_match_ids = self._eligible_match_ids_subquery()
        period_filter = self._period_filter(period)

        query = (
            select(
                MatchVoteTop5.player_id,
                Player.name,
                Player.nickname,
                Player.avatar_url,
                func.sum(MatchVoteTop5.points).label("total_points"),
            )
            .join(MatchVote, MatchVote.id == MatchVoteTop5.vote_id)
            .join(Player, Player.id == MatchVoteTop5.player_id)
            .where(
                MatchVote.match_id.in_(eligible_match_ids),
                Player.role != PlayerRole.ADMIN,
            )
            .group_by(
                MatchVoteTop5.player_id,
                Player.name,
                Player.nickname,
                Player.avatar_url,
            )
            .order_by(func.sum(MatchVoteTop5.points).desc())
            .limit(10)
        )

        if period_filter is not None:
            query = query.where(period_filter)

        result = await self.session.execute(query)
        rows = result.all()
        return self._assign_positions(rows, "total_points")

    async def get_flop(self, period: Literal["month", "year", "all"]) -> list[dict]:
        """
        Retorna os top 10 jogadores por número de votos de decepção recebidos,
        considerando apenas partidas elegíveis.
        """
        eligible_match_ids = self._eligible_match_ids_subquery()
        period_filter = self._period_filter(period)

        query = (
            select(
                MatchVoteFlop.player_id,
                Player.name,
                Player.nickname,
                Player.avatar_url,
                func.count(MatchVoteFlop.id).label("total_flop_votes"),
            )
            .join(MatchVote, MatchVote.id == MatchVoteFlop.vote_id)
            .join(Player, Player.id == MatchVoteFlop.player_id)
            .where(
                MatchVote.match_id.in_(eligible_match_ids),
                Player.role != PlayerRole.ADMIN,
            )
            .group_by(
                MatchVoteFlop.player_id,
                Player.name,
                Player.nickname,
                Player.avatar_url,
            )
            .order_by(func.count(MatchVoteFlop.id).desc())
            .limit(10)
        )

        if period_filter is not None:
            query = query.where(period_filter)

        result = await self.session.execute(query)
        rows = result.all()
        return self._assign_positions(rows, "total_flop_votes")
