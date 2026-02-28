from pydantic import BaseModel, Field


class LoginRequest(BaseModel):
    whatsapp: str = Field(..., description="Número WhatsApp (somente dígitos)")
    password: str


class TokenResponse(BaseModel):
    access_token: str
    token_type: str = "bearer"
    player_id: str
    name: str
    role: str


class ChangePasswordRequest(BaseModel):
    current_password: str
    new_password: str = Field(..., min_length=6)
