import re

from fastapi import APIRouter
from pydantic import BaseModel, field_validator
from sqlalchemy import text

from app.core.dependencies import DB, OptionalPlayer

router = APIRouter(prefix="/beta", tags=["beta"])


class AndroidBetaSignupRequest(BaseModel):
    google_email: str

    @field_validator("google_email")
    @classmethod
    def valid_email(cls, v: str) -> str:
        v = v.strip().lower()
        if not re.match(r'^[^\s@]+@[^\s@]+\.[^\s@]+$', v):
            raise ValueError("Email inválido")
        if len(v) > 254:
            raise ValueError("Email muito longo")
        return v


@router.post("/android-signup", status_code=201)
async def submit_android_beta(
    body: AndroidBetaSignupRequest,
    db: DB,
    current: OptionalPlayer,
):
    """Registra interesse de um usuário Android na faixa de testes do Google Play."""
    player_id = current.id if current else None
    await db.execute(
        text("""
            INSERT INTO android_beta_signups (google_email, player_id)
            VALUES (:email, :player_id)
        """),
        {"email": body.google_email, "player_id": player_id},
    )
    return {"status": "ok"}
