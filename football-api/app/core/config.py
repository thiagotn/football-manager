from functools import lru_cache
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        case_sensitive=False,
    )

    app_env: str = "development"
    app_name: str = "Football API"
    app_version: str = "1.0.0"
    debug: bool = True

    # Database
    database_url: str = "postgresql+asyncpg://postgres:postgres@localhost:5432/football"

    # JWT
    secret_key: str = "change-me-in-production-use-openssl-rand-hex-32"
    algorithm: str = "HS256"
    access_token_expire_minutes: int = 60 * 24  # 24h
    invite_token_expire_minutes: int = 30

    # CORS
    cors_origins: str = "http://localhost:5173,http://localhost:3000"

    # VAPID (Web Push)
    vapid_private_key: str = ""
    vapid_public_key: str = ""
    vapid_claims_email: str = "admin@rachao.app"

    # Frontend (usado para montar success_url / cancel_url no checkout)
    frontend_url: str = "http://localhost:3000"

    # Twilio (OTP verification)
    twilio_account_sid: str = ""
    twilio_auth_token: str = ""
    twilio_verify_sid: str = ""

    # Billing (Stripe)
    billing_provider: str = "stripe"
    stripe_secret_key: str = ""
    stripe_publishable_key: str = ""
    stripe_webhook_secret: str = ""

    # Stripe Price IDs — preencher após criar os produtos no Stripe Dashboard.
    # Obter em: dashboard.stripe.com → Product catalog → (plano) → Price ID
    stripe_price_basic_monthly: str = ""
    stripe_price_basic_yearly: str = ""
    stripe_price_pro_monthly: str = ""
    stripe_price_pro_yearly: str = ""

    def get_price_id(self, plan: str, billing_cycle: str) -> str:
        mapping = {
            ("basic", "monthly"): self.stripe_price_basic_monthly,
            ("basic", "yearly"):  self.stripe_price_basic_yearly,
            ("pro",   "monthly"): self.stripe_price_pro_monthly,
            ("pro",   "yearly"):  self.stripe_price_pro_yearly,
        }
        price_id = mapping.get((plan, billing_cycle), "")
        if not price_id:
            raise ValueError(
                f"STRIPE_PRICE_{plan.upper()}_{billing_cycle.upper()} não configurado no .env"
            )
        return price_id

    @property
    def cors_origins_list(self) -> list[str]:
        return [o.strip() for o in self.cors_origins.split(",")]

    @property
    def is_prod(self) -> bool:
        return self.app_env == "production"


@lru_cache
def get_settings() -> Settings:
    return Settings()
