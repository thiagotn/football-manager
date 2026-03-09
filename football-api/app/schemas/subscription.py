from pydantic import BaseModel


class SubscriptionMeResponse(BaseModel):
    plan: str
    groups_limit: int | None   # None = ilimitado
    groups_used: int
    members_limit: int | None  # None = ilimitado
