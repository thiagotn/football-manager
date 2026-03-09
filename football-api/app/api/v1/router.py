from fastapi import APIRouter

from app.api.v1.routers import auth, groups, invites, matches, players, push, reviews, subscriptions

api_router = APIRouter(prefix="/api/v1")

api_router.include_router(auth.router)
api_router.include_router(players.router)
api_router.include_router(groups.router)
api_router.include_router(matches.router)
api_router.include_router(invites.router)
api_router.include_router(push.router)
api_router.include_router(subscriptions.router)
api_router.include_router(reviews.router)
