from uuid import UUID

from sqlalchemy import func, select
from sqlalchemy.ext.asyncio import AsyncSession

from app.models.match_vote import MatchVote, MatchVoteFlop, MatchVoteTop5
from app.models.player import Player
from app.services.voting import POINTS


class VoteRepository:
    def __init__(self, session: AsyncSession):
        self.session = session

    async def has_voted(self, match_id: UUID, voter_id: UUID) -> bool:
        result = await self.session.execute(
            select(MatchVote).where(
                MatchVote.match_id == match_id,
                MatchVote.voter_id == voter_id,
            )
        )
        return result.scalar_one_or_none() is not None

    async def voter_count(self, match_id: UUID) -> int:
        result = await self.session.execute(
            select(func.count()).select_from(MatchVote).where(MatchVote.match_id == match_id)
        )
        return result.scalar_one()

    async def voter_ids(self, match_id: UUID) -> list[UUID]:
        result = await self.session.execute(
            select(MatchVote.voter_id).where(MatchVote.match_id == match_id)
        )
        return list(result.scalars().all())

    async def submit(
        self,
        match_id: UUID,
        voter_id: UUID,
        top5: list[dict],        # [{"player_id": UUID, "position": int}]
        flop_player_id: UUID | None,
    ) -> None:
        vote = MatchVote(match_id=match_id, voter_id=voter_id)
        self.session.add(vote)
        await self.session.flush()

        for item in top5:
            self.session.add(MatchVoteTop5(
                vote_id=vote.id,
                player_id=item["player_id"],
                position=item["position"],
                points=POINTS[item["position"]],
            ))

        if flop_player_id:
            self.session.add(MatchVoteFlop(vote_id=vote.id, player_id=flop_player_id))

        await self.session.flush()

    async def get_results(self, match_id: UUID) -> dict:
        # Top 5 — soma de pontos por jogador
        top5_q = await self.session.execute(
            select(
                MatchVoteTop5.player_id,
                Player.name,
                Player.nickname,
                func.sum(MatchVoteTop5.points).label("total_points"),
            )
            .join(MatchVote, MatchVote.id == MatchVoteTop5.vote_id)
            .join(Player, Player.id == MatchVoteTop5.player_id)
            .where(MatchVote.match_id == match_id)
            .group_by(MatchVoteTop5.player_id, Player.name, Player.nickname)
            .order_by(func.sum(MatchVoteTop5.points).desc())
        )
        top5_rows = top5_q.all()

        # Atribuir posições com suporte a empate
        top5_results = []
        prev_pts = None
        pos = 0
        rank = 0
        for row in top5_rows:
            rank += 1
            if row.total_points != prev_pts:
                pos = rank
                prev_pts = row.total_points
            top5_results.append({
                "position": pos,
                "player_id": row.player_id,
                "name": row.name,
                "nickname": row.nickname,
                "points": row.total_points,
            })

        # Flop — contagem de votos
        flop_q = await self.session.execute(
            select(
                MatchVoteFlop.player_id,
                Player.name,
                Player.nickname,
                func.count().label("vote_count"),
            )
            .join(MatchVote, MatchVote.id == MatchVoteFlop.vote_id)
            .join(Player, Player.id == MatchVoteFlop.player_id)
            .where(MatchVote.match_id == match_id)
            .group_by(MatchVoteFlop.player_id, Player.name, Player.nickname)
            .order_by(func.count().desc())
        )
        flop_rows = flop_q.all()

        max_flop = flop_rows[0].vote_count if flop_rows else 0
        flop_results = [
            {"player_id": r.player_id, "name": r.name, "nickname": r.nickname, "votes": r.vote_count}
            for r in flop_rows if r.vote_count == max_flop
        ]

        total_voters = await self.voter_count(match_id)
        return {"top5": top5_results, "flop": flop_results, "total_voters": total_voters}

    async def get_ballots(self, match_id: UUID) -> list[dict]:
        """Retorna as cédulas individuais de votação para uma partida."""
        # 1. Votos com dados do votante
        votes_q = await self.session.execute(
            select(
                MatchVote.id.label("vote_id"),
                MatchVote.voter_id,
                Player.name.label("voter_name"),
                Player.nickname.label("voter_nickname"),
                Player.avatar_url.label("voter_avatar_url"),
            )
            .join(Player, Player.id == MatchVote.voter_id)
            .where(MatchVote.match_id == match_id)
            .order_by(MatchVote.submitted_at)
        )
        votes_rows = votes_q.mappings().all()
        if not votes_rows:
            return []

        vote_ids = [r["vote_id"] for r in votes_rows]

        # 2. Top 5 de cada voto
        top5_q = await self.session.execute(
            select(
                MatchVoteTop5.vote_id,
                MatchVoteTop5.position,
                MatchVoteTop5.player_id,
                Player.name,
                Player.nickname,
            )
            .join(Player, Player.id == MatchVoteTop5.player_id)
            .where(MatchVoteTop5.vote_id.in_(vote_ids))
            .order_by(MatchVoteTop5.vote_id, MatchVoteTop5.position)
        )
        top5_by_vote: dict = {}
        for r in top5_q.mappings().all():
            top5_by_vote.setdefault(r["vote_id"], []).append({
                "position": r["position"],
                "player_id": r["player_id"],
                "name": r["name"],
                "nickname": r["nickname"],
            })

        # 3. Flop de cada voto
        flop_q = await self.session.execute(
            select(
                MatchVoteFlop.vote_id,
                MatchVoteFlop.player_id,
                Player.name,
                Player.nickname,
            )
            .join(Player, Player.id == MatchVoteFlop.player_id)
            .where(MatchVoteFlop.vote_id.in_(vote_ids))
        )
        flop_by_vote: dict = {
            r["vote_id"]: {
                "player_id": r["player_id"],
                "name": r["name"],
                "nickname": r["nickname"],
            }
            for r in flop_q.mappings().all()
        }

        return [
            {
                "voter_id": r["voter_id"],
                "voter_name": r["voter_name"],
                "voter_nickname": r["voter_nickname"],
                "voter_avatar_url": r["voter_avatar_url"],
                "top5": top5_by_vote.get(r["vote_id"], []),
                "flop": flop_by_vote.get(r["vote_id"]),
            }
            for r in votes_rows
        ]
