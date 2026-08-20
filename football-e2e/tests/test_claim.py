"""
E2E — fluxo de claim de cadastro pendente.

Cenário completo:
  1. Admin cria grupo e adiciona membro by-phone com número placeholder
     → jogador nasce com pending_registration=True (badge "Cadastro temporário").
  2. Admin gera o link de claim via API.
  3. Jogador abre /claim/{token}, informa o número REAL, confirma o OTP
     (bypass local: OTP_BYPASS_CODE) e cria a própria senha.
  4. Login com as novas credenciais funciona e as flags são limpas.

Requer OTP_BYPASS_CODE configurado na API (CI: .env.docker; local: .env.docker).
"""

import json
import os
import uuid

import pytest
import requests
from playwright.sync_api import Page, expect

API_URL = os.getenv("API_V2_URL", "http://localhost:8001")
OTP_BYPASS_CODE = os.getenv("OTP_BYPASS_CODE", "000000")


def _admin_token_from_state(state_path: str) -> str:
    """Extrai o JWT do storage state do admin — evita novo login (rate limit 5/min/IP)."""
    state = json.load(open(state_path))
    for origin in state.get("origins", []):
        for item in origin.get("localStorage", []):
            if item["name"] == "token":
                return item["value"]
    raise RuntimeError("token não encontrado no storage state do admin")


@pytest.fixture
def claim_setup(admin_storage_state):
    """Cria grupo + membro pendente via API e gera o token de claim."""
    token = _admin_token_from_state(admin_storage_state)
    headers = {"Authorization": f"Bearer {token}"}

    group = requests.post(
        f"{API_URL}/api/v2/groups",
        json={"name": f"Grupo Claim E2E {uuid.uuid4().hex[:6]}"},
        headers=headers,
        timeout=10,
    )
    group.raise_for_status()
    group_id = group.json()["id"]

    placeholder = "+550000" + str(uuid.uuid4().int)[:7]
    member = requests.post(
        f"{API_URL}/api/v2/groups/{group_id}/members/by-phone",
        json={"whatsapp": placeholder, "name": "Jogador Pendente E2E"},
        headers=headers,
        timeout=10,
    )
    member.raise_for_status()
    body = member.json()
    assert body["is_new"] is True
    player = body["member"]["player"]
    assert player["pending_registration"] is True
    player_id = player["id"]

    claim = requests.post(
        f"{API_URL}/api/v2/groups/{group_id}/members/{player_id}/claim-invite",
        headers=headers,
        timeout=10,
    )
    assert claim.status_code == 201, (
        f"claim-invite falhou: {claim.status_code} {claim.text} "
        f"(group={group_id} player={player_id})"
    )
    claim_token = claim.json()["token"]

    yield {
        "group_id": group_id,
        "player_id": player_id,
        "claim_token": claim_token,
        "headers": headers,
    }

    # Cleanup: remove grupo (cascade nos convites) e o player criado
    requests.delete(f"{API_URL}/api/v2/groups/{group_id}", headers=headers, timeout=10)
    requests.delete(f"{API_URL}/api/v2/players/{player_id}", headers=headers, timeout=10)


def test_fluxo_claim_completo(page: Page, claim_setup):
    """Jogador assume o cadastro pelo link: OTP + senha → login automático."""
    claim_token = claim_setup["claim_token"]
    real_number = "1198" + str(uuid.uuid4().int)[:7]

    page.goto(f"/claim/{claim_token}")

    # Saudação com o primeiro nome do jogador pendente
    expect(page.locator("text=Olá, Jogador")).to_be_visible(timeout=8000)

    # Etapa 1: número real
    page.locator("#wa").fill(real_number)
    page.get_by_role("button", name="Enviar código").click()

    # Etapa 2: OTP (bypass)
    expect(page.locator("#otp")).to_be_visible(timeout=8000)
    page.locator("#otp").fill(OTP_BYPASS_CODE)
    page.get_by_role("button", name="Verificar código").click()

    # Etapa 3: senha
    expect(page.locator("#pw")).to_be_visible(timeout=8000)
    page.locator("#pw").fill("senhaNova123")
    page.locator("#pw-confirm").fill("senhaNova123")
    page.get_by_role("button", name="Concluir cadastro").click()

    # Sucesso + auto-login
    expect(page.locator("text=Cadastro concluído")).to_be_visible(timeout=8000)

    # Flags limpas na API
    digits = "".join(c for c in real_number if c.isdigit())
    resp = requests.post(
        f"{API_URL}/api/v2/auth/login",
        json={"whatsapp": f"+55{digits}", "password": "senhaNova123"},
        timeout=10,
    )
    assert resp.status_code == 200, resp.text
    assert resp.json()["must_change_password"] is False


def test_link_de_claim_e_uso_unico(page: Page, claim_setup):
    """Depois de usado, o link mostra erro de já utilizado/concluído."""
    claim_token = claim_setup["claim_token"]

    # Completa o claim direto pela API (mais rápido que repetir a UI)
    real_number = "+551197" + str(uuid.uuid4().int)[:8]
    r = requests.post(
        f"{API_URL}/api/v2/claims/{claim_token}/send-otp",
        json={"whatsapp": real_number},
        timeout=10,
    )
    assert r.status_code == 200, r.text
    r = requests.post(
        f"{API_URL}/api/v2/claims/{claim_token}/verify-otp",
        json={"whatsapp": real_number, "otp_code": OTP_BYPASS_CODE},
        timeout=10,
    )
    assert r.status_code == 200, r.text
    otp_token = r.json()["otp_token"]
    r = requests.post(
        f"{API_URL}/api/v2/claims/{claim_token}/complete",
        json={"whatsapp": real_number, "otp_token": otp_token, "password": "senhaNova123"},
        timeout=10,
    )
    assert r.status_code == 200, r.text

    # Reabrir o link → uso único
    page.goto(f"/claim/{claim_token}")
    expect(page.locator("text=Link já utilizado")).to_be_visible(timeout=8000)


def test_badge_cadastro_temporario_na_listagem(admin_page: Page, claim_setup):
    """A aba Jogadores do grupo mostra o badge 'Cadastro temporário'."""
    group_id = claim_setup["group_id"]
    admin_page.goto(f"/groups/{group_id}")
    admin_page.wait_for_load_state("networkidle")
    admin_page.get_by_role("button", name="Jogadores").click()
    expect(admin_page.locator("text=Cadastro temporário").first).to_be_visible(timeout=8000)
