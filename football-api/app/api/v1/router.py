from fastapi import APIRouter

from app.api.v1.routers import admin, auth, beta, finance, groups, invites, matches, players, push, ranking, reviews, subscriptions, teams, votes, webhooks

api_router = APIRouter(prefix="/api/v1")

api_router.include_router(auth.router)
api_router.include_router(players.router)
api_router.include_router(groups.router)
api_router.include_router(matches.router)
api_router.include_router(invites.router)
api_router.include_router(push.router)
api_router.include_router(subscriptions.router)
api_router.include_router(reviews.router)
api_router.include_router(votes.router)
api_router.include_router(teams.router)
api_router.include_router(admin.router)
api_router.include_router(finance.router)
api_router.include_router(webhooks.router)
api_router.include_router(ranking.router)
api_router.include_router(beta.router)
