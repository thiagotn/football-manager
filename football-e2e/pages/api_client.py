"""
Client HTTP direto para a football-api (sem passar pelo frontend).

Usado pelos testes de integração Stripe para chamar os endpoints
da API com autenticação JWT, sem necessidade do browser.
"""

import requests


class ApiClient:
    def __init__(self, base_url: str):
        self.base_url = base_url.rstrip("/")
        self._token: str | None = None

    # ------------------------------------------------------------------
    # Auth
    # ------------------------------------------------------------------

    def login(self, whatsapp: str, password: str) -> dict:
        """POST /api/v1/auth/login — autentica e armazena o JWT."""
        resp = requests.post(
            f"{self.base_url}/api/v1/auth/login",
            json={"whatsapp": whatsapp, "password": password},
        )
        resp.raise_for_status()
        body = resp.json()
        self._token = body["access_token"]
        return body

    def register(self, name: str, nickname: str, whatsapp: str, password: str) -> dict:
        """POST /api/v1/auth/register — cadastro público de novo player."""
        resp = requests.post(
            f"{self.base_url}/api/v1/auth/register",
            json={
                "name": name,
                "nickname": nickname,
                "whatsapp": whatsapp,
                "password": password,
            },
        )
        resp.raise_for_status()
        body = resp.json()
        self._token = body.get("access_token")
        return body

    @property
    def _headers(self) -> dict:
        if not self._token:
            raise RuntimeError("ApiClient: não autenticado. Chame login() ou register() primeiro.")
        return {"Authorization": f"Bearer {self._token}"}

    # ------------------------------------------------------------------
    # Plans
    # ------------------------------------------------------------------

    def get_plans(self) -> list[dict]:
        """GET /api/v1/plans — lista planos (público)."""
        resp = requests.get(f"{self.base_url}/api/v1/plans")
        resp.raise_for_status()
        return resp.json()

    # ------------------------------------------------------------------
    # Subscriptions
    # ------------------------------------------------------------------

    def get_my_subscription(self) -> dict:
        """GET /api/v1/subscriptions/me — assinatura + limites do player logado."""
        resp = requests.get(
            f"{self.base_url}/api/v1/subscriptions/me",
            headers=self._headers,
        )
        resp.raise_for_status()
        return resp.json()

    def create_checkout_session(self, plan: str, billing_cycle: str = "monthly") -> dict:
        """POST /api/v1/subscriptions — inicia checkout para novo plano.

        Retorna dict com ao menos 'checkout_url' contendo a sessão Stripe.
        """
        resp = requests.post(
            f"{self.base_url}/api/v1/subscriptions",
            json={"plan": plan, "billing_cycle": billing_cycle},
            headers=self._headers,
        )
        resp.raise_for_status()
        return resp.json()

    def cancel_subscription(self) -> dict:
        """DELETE /api/v1/subscriptions/me — cancela assinatura."""
        resp = requests.delete(
            f"{self.base_url}/api/v1/subscriptions/me",
            headers=self._headers,
        )
        resp.raise_for_status()
        return resp.json()

    def reactivate_subscription(self) -> dict:
        """POST /api/v1/subscriptions/me/reactivate."""
        resp = requests.post(
            f"{self.base_url}/api/v1/subscriptions/me/reactivate",
            headers=self._headers,
        )
        resp.raise_for_status()
        return resp.json()

    # ------------------------------------------------------------------
    # Groups (para testar limites de plano)
    # ------------------------------------------------------------------

    def create_group(self, name: str) -> tuple[int, dict]:
        """POST /api/v1/groups — retorna (status_code, body).

        Não usa raise_for_status() para permitir testar respostas 403.
        """
        resp = requests.post(
            f"{self.base_url}/api/v1/groups",
            json={"name": name},
            headers=self._headers,
        )
        return resp.status_code, resp.json()

    def delete_group(self, group_id: str) -> int:
        """DELETE /api/v1/groups/{id} — remove grupo. Retorna status_code."""
        resp = requests.delete(
            f"{self.base_url}/api/v1/groups/{group_id}",
            headers=self._headers,
        )
        return resp.status_code

    # ------------------------------------------------------------------
    # Webhooks (chamada direta, sem CLI, para testes de assinatura HMAC)
    # ------------------------------------------------------------------

    def post_webhook_raw(self, payload: bytes, stripe_signature: str) -> tuple[int, dict]:
        """POST /api/v1/webhooks/payment com payload bruto e header de assinatura Stripe.

        Útil para testar a validação de assinatura sem usar o stripe CLI.
        Retorna (status_code, body).
        """
        resp = requests.post(
            f"{self.base_url}/api/v1/webhooks/payment",
            data=payload,
            headers={
                "Content-Type": "application/json",
                "Stripe-Signature": stripe_signature,
            },
        )
        return resp.status_code, resp.json() if resp.content else {}
