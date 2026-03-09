from pydantic import BaseModel, Field


class LoginRequest(BaseModel):
    whatsapp: str = Field(..., description="Número WhatsApp (somente dígitos)")
    password: str


class RegisterRequest(BaseModel):
    name: str = Field(..., min_length=2)
    whatsapp: str
    password: str = Field(..., min_length=6)
    nickname: str | None = None


class TokenResponse(BaseModel):
    access_token: str
    token_type: str = "bearer"
    player_id: str
    name: str
    role: str
    must_change_password: bool = False


class ChangePasswordRequest(BaseModel):
    current_password: str
    new_password: str = Field(..., min_length=6)
