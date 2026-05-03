from pydantic import BaseModel, Field, field_validator

from app.schemas.player import normalize_whatsapp, normalize_nickname


class LoginRequest(BaseModel):
    whatsapp: str = Field(..., description="Número WhatsApp em formato E.164")
    password: str

    @field_validator("whatsapp")
    @classmethod
    def validate_whatsapp(cls, v: str) -> str:
        return normalize_whatsapp(v)


class SendOtpRequest(BaseModel):
    whatsapp: str

    @field_validator("whatsapp")
    @classmethod
    def validate_whatsapp(cls, v: str) -> str:
        return normalize_whatsapp(v)


class SendOtpResponse(BaseModel):
    status: str = "pending"
    expires_in_seconds: int = 600


class VerifyOtpRequest(BaseModel):
    whatsapp: str
    otp_code: str = Field(..., min_length=6, max_length=6)

    @field_validator("whatsapp")
    @classmethod
    def validate_whatsapp(cls, v: str) -> str:
        return normalize_whatsapp(v)


class VerifyOtpMeRequest(BaseModel):
    otp_code: str = Field(..., min_length=6, max_length=6)


class VerifyOtpResponse(BaseModel):
    otp_token: str


class RegisterRequest(BaseModel):
    name: str = Field(..., min_length=2)
    whatsapp: str
    password: str = Field(..., min_length=6)
    nickname: str | None = None
    otp_token: str

    @field_validator("whatsapp")
    @classmethod
    def validate_whatsapp(cls, v: str) -> str:
        return normalize_whatsapp(v)

    @field_validator("nickname", mode="before")
    @classmethod
    def trim_nickname(cls, v: str | None) -> str | None:
        return normalize_nickname(v)


class TokenResponse(BaseModel):
    access_token: str
    refresh_token: str | None = None
    token_type: str = "bearer"
    player_id: str
    name: str
    nickname: str | None = None
    role: str
    must_change_password: bool = False
    avatar_url: str | None = None
    chat_enabled: bool = False


class RefreshRequest(BaseModel):
    refresh_token: str


class RefreshResponse(BaseModel):
    access_token: str
    refresh_token: str
    token_type: str = "bearer"


class ChangePasswordRequest(BaseModel):
    current_password: str | None = None
    new_password: str = Field(..., min_length=6)
    otp_token: str | None = None


class ForgotPasswordResetRequest(BaseModel):
    whatsapp: str
    otp_token: str
    new_password: str = Field(..., min_length=6)

    @field_validator("whatsapp")
    @classmethod
    def validate_whatsapp(cls, v: str) -> str:
        return normalize_whatsapp(v)
