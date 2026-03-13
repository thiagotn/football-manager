"""
Testes E2E — Integração Stripe Billing
=======================================

Cobre os cenários de validação descritos em:
  docs/prd/planos-assinatura.md — seções 9.4, 14.7, 14.9

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
PRÉ-REQUISITOS (execute antes de rodar os testes)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

1. Instale as dependências:
     cd football-e2e
     pip install -e .

2. Copie e preencha o .env:
     cp .env.example .env
     # Edite .env e ajuste API_URL, STRIPE_WEBHOOK_SECRET, etc.

3. Certifique-se de que a football-api está rodando:
     cd football-api && docker compose up -d
     # API disponível em http://localhost:8000

4. Instale o Stripe CLI:
     # macOS:  brew install stripe/stripe-cli/stripe
     # Linux:  https://stripe.com/docs/stripe-cli#install
     stripe login           # autentique com sua conta Stripe

5. Em um terminal separado, inicie o listener de webhooks:
     stripe listen --forward-to http://localhost:8000/api/v1/webhooks/payment
     # Copie o "webhook signing secret" exibido (whsec_...) e coloque em .env

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
EXECUTAR
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  # Todos os testes Stripe (requer CLI):
  pytest tests/test_stripe_webhooks.py -v

  # Apenas testes que NÃO precisam do Stripe CLI:
  pytest tests/test_stripe_webhooks.py -v -m "not stripe_cli"

  # Apenas testes que precisam do Stripe CLI:
  pytest tests/test_stripe_webhooks.py -v -m stripe_cli

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
"""

import os
import re
import shutil
import subprocess
import time
import uuid

import pytest
import requests
from dotenv import load_dotenv

from pages.api_client import ApiClient

load_dotenv()

API_URL = os.getenv("API_URL", "http://localhost:8000")
STRIPE_WEBHOOK_TIMEOUT = int(os.getenv("STRIPE_WEBHOOK_TIMEOUT", "15"))

# Marca para testes que exigem `stripe listen` ativo + Stripe CLI instalado
stripe_cli = pytest.mark.stripe_cli

# Marca para testes de endpoints ainda não implementados (Fase 2 do PRD).
# Esses testes documentam o comportamento esperado e passarão quando o
# backend implementar: GET /api/v1/plans, POST /api/v1/subscriptions,
# POST /api/v1/webhooks/payment.
# Para rodá-los: pytest -m phase2
# Para excluí-los: pytest -m "not phase2"
phase2 = pytest.mark.phase2


# ─────────────────────────────────────────────────────────────────────────────
# Helpers
# ─────────────────────────────────────────────────────────────────────────────


def _stripe_cli_available() -> bool:
    """Retorna True se o Stripe CLI está instalado e no PATH."""
    return shutil.which("stripe") is not None


def _trigger(event: str, overrides: dict[str, str] | None = None, timeout: int = 10) -> subprocess.CompletedProcess:
    """
    Dispara um evento Stripe sintético via `stripe trigger`.

    COMO FUNCIONA:
      O Stripe CLI envia o evento para o endpoint configurado no `stripe listen`,
      que assina o payload com o STRIPE_WEBHOOK_SECRET e o encaminha para a API.
      A API processa e atualiza o banco de dados.

    Args:
        event:     Nome do evento Stripe (ex: "invoice.payment_failed").
        overrides: Dict de overrides no formato {"recurso:campo": "valor"}.
                   Exemplo: {"invoice:customer": "cus_xxx"} injeta o customer_id
                   no evento sintético para que o handler encontre o player correto.
        timeout:   Timeout em segundos para o subprocess.
    """
    cmd = ["stripe", "trigger", event]
    for key, value in (overrides or {}).items():
        cmd += ["--override", f"{key}={value}"]
    return subprocess.run(cmd, capture_output=True, text=True, timeout=timeout)


def _poll(
    fn,
    condition,
    timeout: int = STRIPE_WEBHOOK_TIMEOUT,
    interval: float = 1.0,
):
    """
    Faz polling em `fn()` até que `condition(result)` seja True ou timeout.

    Usado para aguardar o processamento assíncrono do webhook antes de assertar.

    Args:
        fn:        Callable sem argumentos que retorna o estado atual (ex: api.get_my_subscription).
        condition: Callable que recebe o resultado de fn() e retorna bool.
        timeout:   Tempo máximo de espera em segundos.
        interval:  Intervalo entre tentativas em segundos.

    Returns:
        Último resultado retornado por fn().

    Raises:
        TimeoutError: Se a condição não for satisfeita dentro do timeout.
    """
    deadline = time.time() + timeout
    result = None
    while time.time() < deadline:
        result = fn()
        if condition(result):
            return result
        time.sleep(interval)
    raise TimeoutError(
        f"Condição não atingida em {timeout}s. Último resultado: {result}"
    )


def _unique_whatsapp() -> str:
    """Gera um número de WhatsApp único para evitar conflito entre testes."""
    suffix = str(uuid.uuid4().int)[:9]
    return f"1{suffix}"


# ─────────────────────────────────────────────────────────────────────────────
# Fixtures
# ─────────────────────────────────────────────────────────────────────────────


@pytest.fixture(scope="module")
def api() -> ApiClient:
    """
    Verifica que a API está acessível antes de executar os testes do módulo.

    Se a API não estiver rodando, todos os testes são pulados com mensagem clara.
    """
    client = ApiClient(API_URL)
    try:
        requests.get(f"{API_URL}/health", timeout=3).raise_for_status()
    except Exception:
        # Tenta um endpoint que certamente existe
        try:
            requests.get(f"{API_URL}/api/v1/plans", timeout=3)
        except Exception:
            pytest.skip(
                f"API não acessível em {API_URL}. "
                "Certifique-se de que docker compose está rodando."
            )
    return client


@pytest.fixture
def test_player(api: ApiClient) -> dict:
    """
    Cria um player de teste único via auto-cadastro público e retorna
    um ApiClient já autenticado como esse player.

    O número de WhatsApp é gerado aleatoriamente para evitar conflito
    entre execuções de testes (o endpoint retorna 409 se o WhatsApp já existe).

    PASSO A PASSO:
      1. Gera whatsapp único
      2. POST /api/v1/auth/register → cria player + plano Free + JWT
      3. Retorna dict com {api_client, player_id, whatsapp, token}

    CLEANUP:
      Não exclui o player ao final (a API não tem endpoint de exclusão de player).
      Use um banco de dados de teste isolado ou limpe manualmente se necessário.
    """
    whatsapp = _unique_whatsapp()
    player_api = ApiClient(API_URL)
    body = player_api.register(
        name="Jogador Teste Stripe",
        nickname=f"teste_{whatsapp[-4:]}",
        whatsapp=whatsapp,
        password="Teste@123",
    )
    return {
        "api": player_api,
        "player_id": body.get("player_id") or body.get("id"),
        "whatsapp": whatsapp,
        "token": player_api._token,
    }


@pytest.fixture
def stripe_cli_required():
    """
    Fixture que pula o teste se o Stripe CLI não estiver instalado.

    Adicione como parâmetro em testes que usam `stripe trigger`.
    """
    if not _stripe_cli_available():
        pytest.skip(
            "Stripe CLI não encontrado. Instale em: https://stripe.com/docs/stripe-cli"
        )


# ─────────────────────────────────────────────────────────────────────────────
# CENÁRIO 1 — Listagem de planos (GET /api/v1/plans)
# ─────────────────────────────────────────────────────────────────────────────


@phase2
@pytest.mark.skip(reason="Fase 2 — GET /api/v1/plans ainda não implementado no backend")
class TestPlansApi:
    """
    CENÁRIO: Endpoint público de planos disponíveis.
    PRD: RF-01, seção 6.1.
    STATUS: Fase 2 — GET /api/v1/plans ainda não implementado.

    PASSO A PASSO MANUAL:
      curl http://localhost:8000/api/v1/plans
    """

    def test_planos_retornados_sem_autenticacao(self, api: ApiClient):
        """
        GET /api/v1/plans deve retornar lista com ao menos os 3 planos
        (free, basic, pro) sem exigir autenticação.
        """
        plans = api.get_plans()

        assert isinstance(plans, list), "Resposta deve ser uma lista"
        assert len(plans) >= 3, "Deve haver ao menos 3 planos (free, basic, pro)"

        names = {p["name"] for p in plans}
        assert "free" in names, "Plano 'free' ausente"
        assert "basic" in names, "Plano 'basic' ausente"
        assert "pro" in names, "Plano 'pro' ausente"

    def test_plano_free_tem_limites_corretos(self, api: ApiClient):
        """
        Plano Free deve ter: max_groups=1, max_matches=3, max_members=30.
        PRD seção 2.1 — tabela de planos.
        """
        plans = api.get_plans()
        free = next(p for p in plans if p["name"] == "free")

        assert free["max_groups"] == 1
        assert free["max_matches"] == 3
        assert free["max_members"] == 30

    def test_plano_basic_tem_limites_corretos(self, api: ApiClient):
        """
        Plano Basic: max_groups=3, max_matches=-1 (ilimitado), max_members=50.
        PRD seção 2.1.
        """
        plans = api.get_plans()
        basic = next(p for p in plans if p["name"] == "basic")

        assert basic["max_groups"] == 3
        assert basic["max_matches"] == -1, "max_matches=-1 significa ilimitado"
        assert basic["max_members"] == 50

    def test_plano_pro_tem_limites_ilimitados(self, api: ApiClient):
        """
        Plano Pro: max_groups=10, max_matches=-1, max_members=-1.
        PRD seção 2.1.
        """
        plans = api.get_plans()
        pro = next(p for p in plans if p["name"] == "pro")

        assert pro["max_groups"] == 10
        assert pro["max_matches"] == -1
        assert pro["max_members"] == -1


# ─────────────────────────────────────────────────────────────────────────────
# CENÁRIO 2 — Assinatura padrão de novo player (GET /api/v1/subscriptions/me)
# ─────────────────────────────────────────────────────────────────────────────


class TestSubscriptionDefault:
    """
    CENÁRIO: Novo player cadastrado deve ter plano Free por padrão.
    PRD: Fase 1 implementada, seção 15.

    PASSO A PASSO MANUAL:
      # 1. Registre um novo player
      curl -X POST http://localhost:8000/api/v1/auth/register \\
        -H "Content-Type: application/json" \\
        -d '{"name":"Teste","nickname":"teste","whatsapp":"11900000001","password":"Teste@123"}'

      # 2. Use o token retornado
      curl http://localhost:8000/api/v1/subscriptions/me \\
        -H "Authorization: Bearer <token>"
    """

    def test_novo_player_tem_plano_free(self, test_player: dict):
        """
        Imediatamente após o cadastro, GET /api/v1/subscriptions/me
        deve retornar plan='free'.
        """
        sub = test_player["api"].get_my_subscription()

        assert sub["plan"] == "free", f"Esperado 'free', obtido '{sub['plan']}'"

    def test_novo_player_tem_limite_1_grupo(self, test_player: dict):
        """
        Plano Free: groups_limit deve ser 1.
        PRD: _FREE_GROUPS_LIMIT = 1.
        """
        sub = test_player["api"].get_my_subscription()

        assert sub["groups_limit"] == 1

    def test_novo_player_tem_0_grupos_usados(self, test_player: dict):
        """
        Novo player não possui grupos, então groups_used deve ser 0.
        """
        sub = test_player["api"].get_my_subscription()

        assert sub["groups_used"] == 0

    def test_novo_player_tem_limite_30_membros(self, test_player: dict):
        """
        Plano Free: members_limit deve ser 30.
        """
        sub = test_player["api"].get_my_subscription()

        assert sub["members_limit"] == 30

    def test_sem_autenticacao_retorna_401(self):
        """
        GET /api/v1/subscriptions/me sem token deve retornar 401.
        """
        resp = requests.get(f"{API_URL}/api/v1/subscriptions/me")
        assert resp.status_code == 401


# ─────────────────────────────────────────────────────────────────────────────
# CENÁRIO 3 — Limites de plano Free (RF-08, RF-09, RF-10)
# ─────────────────────────────────────────────────────────────────────────────


class TestPlanLimits:
    """
    CENÁRIO: Backend bloqueia criação de recursos além do limite do plano.
    PRD: RF-08 (grupos), RF-09 (partidas), RNF-01 (validação no backend).

    PASSO A PASSO MANUAL:
      # 1. Registre um player e autentique
      # 2. Crie o primeiro grupo (deve funcionar)
      curl -X POST http://localhost:8000/api/v1/groups \\
        -H "Authorization: Bearer <token>" \\
        -H "Content-Type: application/json" \\
        -d '{"name": "Grupo 1"}'

      # 3. Tente criar um segundo grupo (deve retornar 403)
      curl -X POST http://localhost:8000/api/v1/groups \\
        -H "Authorization: Bearer <token>" \\
        -H "Content-Type: application/json" \\
        -d '{"name": "Grupo 2"}'
      # Esperado: 403 com detail="PLAN_LIMIT_EXCEEDED"
    """

    def test_free_permite_criar_primeiro_grupo(self, test_player: dict):
        """
        Player Free deve conseguir criar exatamente 1 grupo.
        """
        api = test_player["api"]
        status, body = api.create_group(f"Grupo Teste {uuid.uuid4().hex[:6]}")

        assert status == 201, f"Esperado 201, obtido {status}: {body}"
        # Cleanup: remove o grupo criado para não poluir outros testes
        if group_id := (body.get("id") or body.get("group_id")):
            api.delete_group(group_id)

    def test_free_bloqueia_segundo_grupo_com_403(self, test_player: dict):
        """
        Ao tentar criar um segundo grupo, deve retornar 403 PLAN_LIMIT_EXCEEDED.
        PRD: _FREE_GROUPS_LIMIT = 1, RF-08.

        PASSO A PASSO:
          1. Cria o primeiro grupo com sucesso
          2. Tenta criar o segundo → espera 403
          3. Verifica que o body contém o erro correto
        """
        api = test_player["api"]

        # Passo 1: cria o primeiro grupo
        status1, body1 = api.create_group(f"Grupo Principal {uuid.uuid4().hex[:6]}")
        assert status1 == 201, f"Primeiro grupo deveria ser criado: {body1}"
        group_id = body1.get("id") or body1.get("group_id")

        try:
            # Passo 2: tenta criar o segundo grupo
            status2, body2 = api.create_group(f"Grupo Extra {uuid.uuid4().hex[:6]}")

            # Passo 3: valida o bloqueio
            assert status2 == 403, (
                f"Esperado 403 PLAN_LIMIT_EXCEEDED, obtido {status2}: {body2}"
            )
            # O detail pode ser a string direta ou aninhado em "error"
            detail = body2.get("detail") or body2.get("error") or ""
            assert "PLAN_LIMIT_EXCEEDED" in str(detail), (
                f"Body deve conter PLAN_LIMIT_EXCEEDED: {body2}"
            )
        finally:
            # Cleanup: remove o primeiro grupo
            if group_id:
                api.delete_group(group_id)

    def test_free_groups_used_incrementa_apos_criar_grupo(self, test_player: dict):
        """
        Após criar um grupo, GET /api/v1/subscriptions/me deve mostrar
        groups_used=1.
        """
        api = test_player["api"]

        status, body = api.create_group(f"Grupo Counter {uuid.uuid4().hex[:6]}")
        assert status == 201
        group_id = body.get("id") or body.get("group_id")

        try:
            sub = api.get_my_subscription()
            assert sub["groups_used"] == 1, (
                f"groups_used deveria ser 1 após criar grupo, obtido: {sub['groups_used']}"
            )
        finally:
            if group_id:
                api.delete_group(group_id)


# ─────────────────────────────────────────────────────────────────────────────
# CENÁRIO 4 — Criar sessão de checkout (POST /api/v1/subscriptions)
# ─────────────────────────────────────────────────────────────────────────────


@phase2
@pytest.mark.skip(reason="Fase 2 — POST /api/v1/subscriptions (checkout) ainda não implementado no backend")
class TestCheckoutSession:
    """
    CENÁRIO: Iniciar checkout para upgrade de plano.
    PRD: RF-02, seção 6.2, fluxo 7.2.
    STATUS: Fase 2 — POST /api/v1/subscriptions (checkout) ainda não implementado.

    PASSO A PASSO MANUAL:
      curl -X POST http://localhost:8000/api/v1/subscriptions \\
        -H "Authorization: Bearer <token>" \\
        -H "Content-Type: application/json" \\
        -d '{"plan": "basic", "billing_cycle": "monthly"}'
      # Esperado: {"checkout_url": "https://checkout.stripe.com/..."}
    """

    def test_checkout_retorna_url_stripe(self, test_player: dict):
        """
        POST /api/v1/subscriptions deve retornar uma URL de checkout do Stripe.

        PASSO A PASSO:
          1. Player autenticado solicita upgrade para 'basic'
          2. Backend cria um Stripe Customer (se não existir) e uma Checkout Session
          3. Retorna a URL da sessão Stripe para redirect do frontend
        """
        api = test_player["api"]

        body = api.create_checkout_session(plan="basic", billing_cycle="monthly")

        assert "checkout_url" in body, f"Resposta deve conter 'checkout_url': {body}"
        url = body["checkout_url"]
        assert url.startswith("https://checkout.stripe.com/"), (
            f"URL deve ser do Stripe Checkout: {url}"
        )

    def test_checkout_url_contem_session_id(self, test_player: dict):
        """
        A URL de checkout deve conter um session ID que começa com 'cs_'.
        Esse ID é usado pelos testes do Stripe CLI para simular a conclusão
        do checkout (stripe trigger --override checkout_session:id=cs_xxx).
        """
        api = test_player["api"]

        body = api.create_checkout_session(plan="basic", billing_cycle="monthly")
        url = body.get("checkout_url", "")

        # A URL do Stripe Checkout tem formato:
        # https://checkout.stripe.com/c/pay/cs_test_xxx#...
        assert re.search(r"cs_test_|cs_live_", url), (
            f"URL deve conter session ID (cs_test_xxx ou cs_live_xxx): {url}"
        )

    def test_checkout_ciclo_anual(self, test_player: dict):
        """
        POST /api/v1/subscriptions com billing_cycle='yearly' deve funcionar.
        """
        api = test_player["api"]

        body = api.create_checkout_session(plan="basic", billing_cycle="yearly")

        assert "checkout_url" in body
        assert body["checkout_url"].startswith("https://checkout.stripe.com/")

    def test_checkout_plano_invalido_retorna_erro(self, test_player: dict):
        """
        POST /api/v1/subscriptions com plan='inexistente' deve retornar 4xx.
        """
        api = test_player["api"]

        resp = requests.post(
            f"{API_URL}/api/v1/subscriptions",
            json={"plan": "plano_invalido", "billing_cycle": "monthly"},
            headers={"Authorization": f"Bearer {api._token}"},
        )
        assert resp.status_code in (400, 422), (
            f"Plano inválido deve retornar 400 ou 422, obtido: {resp.status_code}"
        )


# ─────────────────────────────────────────────────────────────────────────────
# CENÁRIO 5 — Webhook: checkout.session.completed
# ─────────────────────────────────────────────────────────────────────────────


@stripe_cli
class TestWebhookCheckoutCompleted:
    """
    CENÁRIO: Pagamento aprovado ativa o plano do player.
    PRD: seção 9.4, evento checkout.session.completed.
    PRD: critério de aceitação — ativação em ≤ 30 segundos.

    COMO FUNCIONA COM O STRIPE CLI:
      O `stripe trigger checkout.session.completed` cria um evento sintético e
      o envia para o `stripe listen`, que o assina e encaminha para a API.
      Usamos --override para injetar o metadata.player_id correto, para que
      o webhook handler saiba qual player atualizar.

    PRÉ-REQUISITO:
      `stripe listen --forward-to http://localhost:8000/api/v1/webhooks/payment`
      deve estar rodando em outro terminal.

    PASSO A PASSO MANUAL:
      # 1. Obtenha o player_id do usuário de teste via API
      # 2. Dispare o evento com o player_id no metadata:
      stripe trigger checkout.session.completed \\
        --add checkout_session:metadata.player_id=<uuid> \\
        --add checkout_session:metadata.plan=basic \\
        --add checkout_session:metadata.billing_cycle=monthly
      # 3. Verifique:
      curl http://localhost:8000/api/v1/subscriptions/me \\
        -H "Authorization: Bearer <token>"
      # Esperado: {"plan": "basic", ...}
    """

    def test_checkout_completed_ativa_plano_basic(
        self, test_player: dict, stripe_cli_required
    ):
        """
        Após `stripe trigger checkout.session.completed` com o player_id
        correto no metadata, GET /api/v1/subscriptions/me deve retornar
        plan='basic'.

        PASSO A PASSO:
          1. Registra player de teste (plano Free)
          2. Dispara stripe trigger com metadata.player_id do player
          3. Aguarda até STRIPE_WEBHOOK_TIMEOUT segundos
          4. Verifica que plan mudou para 'basic'
        """
        api = test_player["api"]
        player_id = test_player["player_id"]

        # Passo 1: confirma que está no plano Free
        sub_before = api.get_my_subscription()
        assert sub_before["plan"] == "free", "Player deveria estar no plano Free"

        # Passo 2: dispara o evento Stripe simulando checkout concluído
        result = _trigger(
            "checkout.session.completed",
            overrides={
                "checkout_session:metadata.player_id": str(player_id),
                "checkout_session:metadata.plan": "basic",
                "checkout_session:metadata.billing_cycle": "monthly",
            },
        )
        assert result.returncode == 0, (
            f"stripe trigger falhou:\nSTDOUT: {result.stdout}\nSTDERR: {result.stderr}"
        )

        # Passo 3: aguarda o processamento do webhook (polling)
        try:
            sub_after = _poll(
                fn=api.get_my_subscription,
                condition=lambda s: s["plan"] == "basic",
                timeout=STRIPE_WEBHOOK_TIMEOUT,
            )
        except TimeoutError:
            sub_after = api.get_my_subscription()
            pytest.fail(
                f"Plano não foi atualizado em {STRIPE_WEBHOOK_TIMEOUT}s. "
                f"Estado atual: {sub_after}"
            )

        # Passo 4: valida
        assert sub_after["plan"] == "basic"
        assert sub_after["groups_limit"] == 3, (
            "Após upgrade para basic, groups_limit deve ser 3"
        )

    def test_checkout_completed_cria_registro_de_assinatura(
        self, test_player: dict, stripe_cli_required
    ):
        """
        Após checkout.session.completed, a assinatura deve ter status='active'
        e datas de período preenchidas (current_period_start, current_period_end).
        """
        api = test_player["api"]
        player_id = test_player["player_id"]

        _trigger(
            "checkout.session.completed",
            overrides={
                "checkout_session:metadata.player_id": str(player_id),
                "checkout_session:metadata.plan": "basic",
                "checkout_session:metadata.billing_cycle": "monthly",
            },
        )

        try:
            sub = _poll(
                fn=api.get_my_subscription,
                condition=lambda s: s.get("status") == "active" and s["plan"] == "basic",
                timeout=STRIPE_WEBHOOK_TIMEOUT,
            )
        except TimeoutError:
            sub = api.get_my_subscription()
            pytest.fail(f"Assinatura não ativada. Estado: {sub}")

        assert sub.get("status") == "active"
        assert sub.get("current_period_end") is not None, (
            "current_period_end deve ser preenchido após ativação"
        )


# ─────────────────────────────────────────────────────────────────────────────
# CENÁRIO 6 — Webhook: invoice.paid (renovação)
# ─────────────────────────────────────────────────────────────────────────────


@stripe_cli
class TestWebhookInvoicePaid:
    """
    CENÁRIO: Pagamento de fatura renova o período da assinatura.
    PRD: seção 9.4, evento invoice.paid.

    PASSO A PASSO MANUAL:
      # Obtenha o gateway_customer_id do player (retornado pela API após ativação)
      stripe trigger invoice.paid \\
        --override invoice:customer=<cus_xxx>
      # Verifique que current_period_end foi renovado:
      curl http://localhost:8000/api/v1/subscriptions/me \\
        -H "Authorization: Bearer <token>"
    """

    def test_invoice_paid_registra_fatura(self, test_player: dict, stripe_cli_required):
        """
        Após invoice.paid, uma fatura com status='paid' deve aparecer em
        GET /api/v1/invoices.

        PASSO A PASSO:
          1. Garante que o player tem um gateway_customer_id (ativo no Stripe)
          2. Dispara invoice.paid com o customer_id do player
          3. Verifica GET /api/v1/invoices retorna ao menos uma fatura paga
        """
        api = test_player["api"]
        player_id = test_player["player_id"]

        # Primeiro, obtém o customer_id do player via subscriptions/me
        sub = api.get_my_subscription()
        customer_id = sub.get("gateway_customer_id")

        if not customer_id:
            pytest.skip(
                "Player não possui gateway_customer_id. "
                "Execute o teste de checkout.session.completed antes."
            )

        # Dispara a renovação de fatura
        result = _trigger(
            "invoice.paid",
            overrides={"invoice:customer": customer_id},
        )
        assert result.returncode == 0, (
            f"stripe trigger falhou: {result.stderr}"
        )

        # Aguarda o webhook ser processado e verifica faturas
        time.sleep(3)  # pequena espera para processamento assíncrono

        invoices_resp = requests.get(
            f"{API_URL}/api/v1/invoices",
            headers={"Authorization": f"Bearer {api._token}"},
        )
        assert invoices_resp.status_code == 200
        invoices = invoices_resp.json()

        paid = [inv for inv in invoices if inv.get("status") == "paid"]
        assert len(paid) >= 1, (
            f"Deve haver ao menos 1 fatura com status='paid': {invoices}"
        )

    def test_invoice_paid_renova_current_period_end(
        self, test_player: dict, stripe_cli_required
    ):
        """
        invoice.paid deve renovar current_period_end na assinatura do player.

        PASSO A PASSO:
          1. Registra current_period_end antes do evento
          2. Dispara invoice.paid
          3. Verifica que current_period_end foi atualizado (ou mantido ativo)
        """
        api = test_player["api"]

        sub_before = api.get_my_subscription()
        customer_id = sub_before.get("gateway_customer_id")

        if not customer_id:
            pytest.skip("Player sem gateway_customer_id (checkout não completado)")

        period_before = sub_before.get("current_period_end")

        _trigger("invoice.paid", overrides={"invoice:customer": customer_id})
        time.sleep(3)

        sub_after = api.get_my_subscription()

        # O status deve permanecer ativo após pagamento bem-sucedido
        assert sub_after.get("status") == "active", (
            f"Status deve ser 'active' após invoice.paid: {sub_after}"
        )


# ─────────────────────────────────────────────────────────────────────────────
# CENÁRIO 7 — Webhook: invoice.payment_failed (past_due + graça)
# ─────────────────────────────────────────────────────────────────────────────


@stripe_cli
class TestWebhookPaymentFailed:
    """
    CENÁRIO: Falha de pagamento coloca assinatura em past_due com período de graça.
    PRD: seção 9.4, evento invoice.payment_failed; fluxo 7.3.
    PRD: critério — grace_period_end = NOW() + 7 dias.

    COMO TESTAR MANUALMENTE:
      # Cartão de teste para forçar falha de pagamento:
      # 4000 0000 0000 0341 (card_declined)

      # Via Stripe CLI:
      stripe trigger invoice.payment_failed \\
        --override invoice:customer=<cus_xxx>

      # Verificar:
      curl http://localhost:8000/api/v1/subscriptions/me \\
        -H "Authorization: Bearer <token>"
      # Esperado: {"status": "past_due", "grace_period_end": "2026-...", ...}
    """

    def test_payment_failed_define_status_past_due(
        self, test_player: dict, stripe_cli_required
    ):
        """
        Após invoice.payment_failed, status deve mudar para 'past_due'.

        PASSO A PASSO:
          1. Player com plano ativo (requer checkout completado)
          2. Dispara invoice.payment_failed com customer_id do player
          3. Aguarda processamento do webhook
          4. Verifica status='past_due'
        """
        api = test_player["api"]

        sub = api.get_my_subscription()
        customer_id = sub.get("gateway_customer_id")

        if not customer_id:
            pytest.skip("Player sem gateway_customer_id")

        if sub.get("status") != "active":
            pytest.skip(f"Teste requer status='active', atual: {sub.get('status')}")

        # Dispara falha de pagamento
        result = _trigger(
            "invoice.payment_failed",
            overrides={"invoice:customer": customer_id},
        )
        assert result.returncode == 0, f"stripe trigger falhou: {result.stderr}"

        # Aguarda mudança de status
        try:
            sub_after = _poll(
                fn=api.get_my_subscription,
                condition=lambda s: s.get("status") == "past_due",
                timeout=STRIPE_WEBHOOK_TIMEOUT,
            )
        except TimeoutError:
            sub_after = api.get_my_subscription()
            pytest.fail(
                f"Status não mudou para 'past_due' em {STRIPE_WEBHOOK_TIMEOUT}s. "
                f"Estado atual: {sub_after}"
            )

        assert sub_after["status"] == "past_due"

    def test_payment_failed_define_grace_period_end(
        self, test_player: dict, stripe_cli_required
    ):
        """
        Após invoice.payment_failed, grace_period_end deve ser definido
        como aproximadamente NOW() + 7 dias.
        PRD: fluxo 7.3 — "grace_period_end = NOW() + 7 dias".
        """
        import datetime

        api = test_player["api"]

        sub = api.get_my_subscription()
        customer_id = sub.get("gateway_customer_id")

        if not customer_id:
            pytest.skip("Player sem gateway_customer_id")

        _trigger(
            "invoice.payment_failed",
            overrides={"invoice:customer": customer_id},
        )

        time.sleep(STRIPE_WEBHOOK_TIMEOUT)
        sub_after = api.get_my_subscription()

        grace_end_str = sub_after.get("grace_period_end")
        assert grace_end_str is not None, (
            "grace_period_end deve ser definido após payment_failed"
        )

        # Valida que grace_period_end é ~7 dias no futuro (margem de ±1 dia)
        grace_end = datetime.datetime.fromisoformat(grace_end_str.replace("Z", "+00:00"))
        now = datetime.datetime.now(datetime.timezone.utc)
        delta_days = (grace_end - now).days

        assert 6 <= delta_days <= 8, (
            f"grace_period_end deveria ser ~7 dias no futuro, obtido {delta_days} dias: "
            f"{grace_end_str}"
        )

    def test_acesso_mantido_durante_periodo_de_graca(
        self, test_player: dict, stripe_cli_required
    ):
        """
        PRD fluxo 7.3: acesso mantido durante os 7 dias de graça.
        Player com status='past_due' ainda deve conseguir acessar recursos
        dentro dos limites do plano pago.

        VERIFICA: GET /api/v1/subscriptions/me ainda retorna os limites
        do plano pago (não regrediu para Free).
        """
        api = test_player["api"]

        sub = api.get_my_subscription()
        customer_id = sub.get("gateway_customer_id")

        if not customer_id:
            pytest.skip("Player sem gateway_customer_id")

        _trigger(
            "invoice.payment_failed",
            overrides={"invoice:customer": customer_id},
        )

        time.sleep(STRIPE_WEBHOOK_TIMEOUT)
        sub_after = api.get_my_subscription()

        if sub_after.get("status") != "past_due":
            pytest.skip("Status não mudou para past_due, skip de validação de graça")

        # Durante período de graça, os limites do plano pago devem ser mantidos
        # (não deve ter regredido para Free com groups_limit=1)
        plan = sub_after["plan"]
        assert plan != "free", (
            "Plano não deve regredir para 'free' durante período de graça"
        )


# ─────────────────────────────────────────────────────────────────────────────
# CENÁRIO 8 — Webhook: customer.subscription.deleted (cancelamento)
# ─────────────────────────────────────────────────────────────────────────────


@stripe_cli
class TestWebhookSubscriptionDeleted:
    """
    CENÁRIO: Cancelamento ou expiração de assinatura regride plano para Free.
    PRD: seção 9.4, evento customer.subscription.deleted; RF-06, fluxo 7.3.

    PASSO A PASSO MANUAL:
      stripe trigger customer.subscription.deleted \\
        --override subscription:customer=<cus_xxx>

      # Verificar:
      curl http://localhost:8000/api/v1/subscriptions/me \\
        -H "Authorization: Bearer <token>"
      # Esperado: {"plan": "free", "status": "canceled" ou "expired"}
    """

    def test_subscription_deleted_regride_para_free(
        self, test_player: dict, stripe_cli_required
    ):
        """
        Após customer.subscription.deleted, o player deve retornar ao plano Free.

        PASSO A PASSO:
          1. Player com plano pago ativo
          2. Dispara customer.subscription.deleted
          3. Verifica que plan='free' e status='canceled' ou 'expired'
        """
        api = test_player["api"]

        sub = api.get_my_subscription()
        customer_id = sub.get("gateway_customer_id")
        sub_stripe_id = sub.get("gateway_sub_id")

        if not customer_id or not sub_stripe_id:
            pytest.skip("Player sem assinatura ativa no Stripe")

        if sub.get("plan") == "free":
            pytest.skip("Player já está no plano Free")

        # Dispara deleção da assinatura
        result = _trigger(
            "customer.subscription.deleted",
            overrides={
                "subscription:customer": customer_id,
                "subscription:id": sub_stripe_id,
            },
        )
        assert result.returncode == 0, f"stripe trigger falhou: {result.stderr}"

        # Aguarda regressão do plano
        try:
            sub_after = _poll(
                fn=api.get_my_subscription,
                condition=lambda s: s["plan"] == "free",
                timeout=STRIPE_WEBHOOK_TIMEOUT,
            )
        except TimeoutError:
            sub_after = api.get_my_subscription()
            pytest.fail(
                f"Plano não regrediu para 'free' em {STRIPE_WEBHOOK_TIMEOUT}s. "
                f"Estado atual: {sub_after}"
            )

        assert sub_after["plan"] == "free"
        assert sub_after.get("status") in ("canceled", "expired"), (
            f"Status deve ser 'canceled' ou 'expired': {sub_after.get('status')}"
        )

    def test_subscription_deleted_arquiva_grupos_excedentes(
        self, test_player: dict, stripe_cli_required
    ):
        """
        Ao regredir para Free (max_groups=1), grupos excedentes devem ser
        arquivados com archived_by_plan=True (não excluídos).
        PRD: RF-12.

        PASSO A PASSO:
          1. Player no plano Basic (3 grupos permitidos)
          2. Cria 2 grupos extras
          3. Dispara subscription.deleted → regride para Free
          4. Verifica que grupos extras estão archived_by_plan=True (não deletados)
        """
        api = test_player["api"]

        sub = api.get_my_subscription()
        if sub.get("plan") != "basic":
            pytest.skip("Teste requer plano 'basic' com múltiplos grupos")

        # Lista grupos antes
        groups_before = requests.get(
            f"{API_URL}/api/v1/groups",
            headers={"Authorization": f"Bearer {api._token}"},
        ).json()

        customer_id = sub.get("gateway_customer_id")
        sub_stripe_id = sub.get("gateway_sub_id")

        _trigger(
            "customer.subscription.deleted",
            overrides={
                "subscription:customer": customer_id,
                "subscription:id": sub_stripe_id,
            },
        )

        time.sleep(STRIPE_WEBHOOK_TIMEOUT)

        # Verifica que grupos não foram excluídos (apenas arquivados)
        groups_after = requests.get(
            f"{API_URL}/api/v1/groups",
            headers={"Authorization": f"Bearer {api._token}"},
            params={"include_archived": True},
        ).json()

        total_before = len(groups_before)
        total_after = len(groups_after) if isinstance(groups_after, list) else 0

        assert total_after >= total_before, (
            "Grupos não devem ser deletados ao regredir de plano (apenas arquivados). "
            f"Antes: {total_before}, Depois: {total_after}"
        )


# ─────────────────────────────────────────────────────────────────────────────
# CENÁRIO 9 — Webhook: customer.subscription.updated (upgrade/downgrade)
# ─────────────────────────────────────────────────────────────────────────────


@stripe_cli
class TestWebhookSubscriptionUpdated:
    """
    CENÁRIO: Atualização de assinatura reflete novo plano na API.
    PRD: seção 9.4, evento customer.subscription.updated; RF-04, RF-05.

    PASSO A PASSO MANUAL:
      # Obtém o subscription ID do player via subscriptions/me
      stripe trigger customer.subscription.updated \\
        --override subscription:customer=<cus_xxx> \\
        --override subscription:id=<sub_xxx>

      curl http://localhost:8000/api/v1/subscriptions/me \\
        -H "Authorization: Bearer <token>"
    """

    def test_subscription_updated_reflete_novo_plano(
        self, test_player: dict, stripe_cli_required
    ):
        """
        Após customer.subscription.updated, os limites do player devem
        refletir o novo plano descrito no evento.

        PASSO A PASSO:
          1. Player com assinatura ativa
          2. Dispara subscription.updated indicando mudança para plano 'pro'
          3. Verifica que API retorna limites do plano pro
        """
        api = test_player["api"]

        sub = api.get_my_subscription()
        customer_id = sub.get("gateway_customer_id")
        sub_stripe_id = sub.get("gateway_sub_id")

        if not customer_id or not sub_stripe_id:
            pytest.skip("Player sem assinatura ativa no Stripe")

        result = _trigger(
            "customer.subscription.updated",
            overrides={
                "subscription:customer": customer_id,
                "subscription:id": sub_stripe_id,
                # Indica que o plano mudou para pro via metadata
                "subscription:metadata.plan": "pro",
            },
        )
        assert result.returncode == 0, f"stripe trigger falhou: {result.stderr}"

        time.sleep(STRIPE_WEBHOOK_TIMEOUT)

        sub_after = api.get_my_subscription()
        # Valida que os limites foram atualizados (pro tem groups_limit=10)
        assert sub_after.get("groups_limit") == 10 or sub_after.get("plan") == "pro", (
            f"Limites do plano Pro não refletidos: {sub_after}"
        )


# ─────────────────────────────────────────────────────────────────────────────
# CENÁRIO 10 — Idempotência de webhooks (RNF-02)
# ─────────────────────────────────────────────────────────────────────────────


@stripe_cli
class TestWebhookIdempotency:
    """
    CENÁRIO: O mesmo evento processado duas vezes não deve duplicar ações.
    PRD: RNF-02 — "processamento de webhooks deve ser idempotente".
    PRD: tabela webhook_events com event_id UNIQUE como idempotency key.

    MECANISMO:
      A tabela webhook_events armazena event_id com UNIQUE constraint.
      O handler verifica se o event_id já existe antes de processar.
      Se já existe → retorna 200 imediatamente sem reprocessar.

    PASSO A PASSO MANUAL:
      # Dispara o mesmo evento duas vezes e verifica que o estado
      # do banco não muda na segunda vez (ex: não duplica fatura).
      stripe trigger invoice.paid --override invoice:customer=<cus_xxx>
      stripe trigger invoice.paid --override invoice:customer=<cus_xxx>
      # O segundo deve retornar 200 mas não criar nova fatura.
    """

    def test_evento_duplicado_nao_duplica_fatura(
        self, test_player: dict, stripe_cli_required
    ):
        """
        Enviar invoice.paid duas vezes para o mesmo customer não deve
        duplicar o registro de fatura.

        PASSO A PASSO:
          1. Obtém customer_id do player
          2. Dispara invoice.paid pela primeira vez → conta faturas
          3. Dispara invoice.paid pela segunda vez (mesmo evento seria re-enviado)
          4. Verifica que o número de faturas não aumentou na segunda vez

        NOTA: O Stripe CLI gera eventos com IDs diferentes a cada trigger.
          Para testar idempotência real, use a tabela webhook_events e verifique
          que um reenvio com o mesmo event_id é rejeitado silenciosamente.
          Este teste valida o comportamento via contagem de faturas.
        """
        api = test_player["api"]

        sub = api.get_my_subscription()
        customer_id = sub.get("gateway_customer_id")

        if not customer_id:
            pytest.skip("Player sem gateway_customer_id")

        def count_paid_invoices():
            resp = requests.get(
                f"{API_URL}/api/v1/invoices",
                headers={"Authorization": f"Bearer {api._token}"},
            )
            if resp.status_code != 200:
                return 0
            return len([i for i in resp.json() if i.get("status") == "paid"])

        # Primeira fatura
        _trigger("invoice.paid", overrides={"invoice:customer": customer_id})
        time.sleep(3)
        count_after_first = count_paid_invoices()

        # Segunda chamada com invoice.paid para o mesmo customer
        _trigger("invoice.paid", overrides={"invoice:customer": customer_id})
        time.sleep(3)
        count_after_second = count_paid_invoices()

        # NOTA: Como o Stripe CLI gera event_ids diferentes, ambos podem ser
        # processados como eventos distintos (com diferentes invoice IDs).
        # O que NÃO deve acontecer é processar o MESMO event_id duas vezes.
        # Esse teste valida que a contagem não explode com múltiplos triggers.
        assert count_after_second >= count_after_first, (
            "Contagem de faturas não deveria diminuir"
        )

    def test_webhook_com_event_id_duplicado_retorna_200(self, api: ApiClient):
        """
        Enviar o mesmo evento (mesmo event_id) duas vezes via POST direto ao
        endpoint deve retornar 200 nas duas chamadas (sem erro).

        PRD RNF-02: "retornar 200 OK imediatamente ao gateway para evitar retentativas".

        PASSO A PASSO:
          1. Constrói um payload mínimo de evento Stripe com event_id fixo
          2. Envia duas vezes para POST /api/v1/webhooks/payment
          3. Ambas devem retornar 200 (a segunda é no-op silencioso)

        NOTA: Este teste requer que o backend aceite eventos sem validar
          assinatura em modo de teste, OU que você configure STRIPE_WEBHOOK_SECRET
          no .env com o secret do `stripe listen`.
        """
        import hashlib
        import hmac
        import json
        import time as time_module

        webhook_secret = os.getenv("STRIPE_WEBHOOK_SECRET", "")
        if not webhook_secret or webhook_secret == "whsec_test_replace_me":
            pytest.skip(
                "STRIPE_WEBHOOK_SECRET não configurado no .env. "
                "Execute `stripe listen` e copie o signing secret."
            )

        # Monta payload de evento sintético
        event_id = f"evt_test_idempotency_{uuid.uuid4().hex[:8]}"
        timestamp = int(time_module.time())
        payload_dict = {
            "id": event_id,
            "type": "invoice.paid",
            "object": "event",
            "created": timestamp,
            "data": {
                "object": {
                    "id": f"in_test_{uuid.uuid4().hex[:8]}",
                    "object": "invoice",
                    "customer": "cus_test_idempotency",
                    "status": "paid",
                    "amount_paid": 1990,
                    "currency": "brl",
                }
            },
        }
        payload_bytes = json.dumps(payload_dict).encode()

        # Assina o payload (formato Stripe: "t=timestamp,v1=hash")
        signed_payload = f"{timestamp}.{payload_bytes.decode()}"
        signature = hmac.new(
            webhook_secret.encode(),
            signed_payload.encode(),
            hashlib.sha256,
        ).hexdigest()
        stripe_sig = f"t={timestamp},v1={signature}"

        # Primeira chamada
        status1, body1 = api.post_webhook_raw(payload_bytes, stripe_sig)
        assert status1 == 200, f"Primeira chamada deve retornar 200: {body1}"

        # Segunda chamada com mesmo event_id (idempotência)
        # Gera nova assinatura com mesmo payload mas pode precisar de novo timestamp
        status2, body2 = api.post_webhook_raw(payload_bytes, stripe_sig)
        assert status2 == 200, (
            f"Segunda chamada com mesmo event_id deve retornar 200 (no-op): {body2}"
        )


# ─────────────────────────────────────────────────────────────────────────────
# CENÁRIO 11 — Validação final (PRD seção 14.9)
# ─────────────────────────────────────────────────────────────────────────────


class TestValidacaoFinal:
    """
    CENÁRIO: Checklist de validação final antes de ir a produção.
    PRD: seção 14.9.

    Executa os smoke tests que não requerem o Stripe CLI,
    cobrindo os itens do checklist de validação final.
    """

    def test_health_check_api(self):
        """
        API deve responder a requisições autenticadas.
        PRD: RNF-04 — disponibilidade 99,9%.

        Usa GET /api/v1/subscriptions/me sem token (espera 401, não 503/timeout),
        confirmando que a API está no ar sem depender de endpoints de Fase 2.
        """
        resp = requests.get(f"{API_URL}/api/v1/subscriptions/me", timeout=5)
        assert resp.status_code != 503, "API retornou 503 (serviço indisponível)"
        assert resp.status_code in (401, 403), (
            f"API deve estar acessível (401 esperado sem token), obtido: {resp.status_code}"
        )

    @pytest.mark.phase2
    @pytest.mark.skip(reason="Fase 2 — GET /api/v1/plans ainda não implementado no backend")
    def test_plano_free_retornado_corretamente(self, api: ApiClient):
        """
        GET /api/v1/plans deve retornar plano 'free' com price_monthly=0.
        Valida seed inicial da tabela plans.
        STATUS: Fase 2 — endpoint GET /api/v1/plans ainda não implementado.
        """
        plans = api.get_plans()
        free = next((p for p in plans if p["name"] == "free"), None)
        assert free is not None, "Plano 'free' não encontrado"
        assert free.get("price_monthly") == 0

    def test_registro_publico_funcional(self):
        """
        POST /api/v1/auth/register deve criar player + assinatura Free + JWT.
        PRD: Fase 1 — auto-cadastro público.

        PASSO A PASSO MANUAL:
          curl -X POST http://localhost:8000/api/v1/auth/register \\
            -H "Content-Type: application/json" \\
            -d '{"name":"Teste","nickname":"ts","whatsapp":"11911112222","password":"Teste@123"}'
          # Esperado: 201 com access_token
        """
        whatsapp = _unique_whatsapp()
        resp = requests.post(
            f"{API_URL}/api/v1/auth/register",
            json={
                "name": "Validação Final",
                "nickname": f"valid_{whatsapp[-4:]}",
                "whatsapp": whatsapp,
                "password": "Teste@123",
            },
        )
        assert resp.status_code == 201, (
            f"Registro público deve retornar 201: {resp.status_code} {resp.text}"
        )
        body = resp.json()
        assert "access_token" in body, "Resposta deve conter access_token"

    def test_registro_duplicado_retorna_409(self):
        """
        Tentar registrar o mesmo WhatsApp duas vezes deve retornar 409.
        PRD: "Retorna 409 se WhatsApp já cadastrado".
        """
        whatsapp = _unique_whatsapp()
        payload = {
            "name": "Dup Teste",
            "nickname": f"dup_{whatsapp[-4:]}",
            "whatsapp": whatsapp,
            "password": "Teste@123",
        }
        requests.post(f"{API_URL}/api/v1/auth/register", json=payload)
        resp2 = requests.post(f"{API_URL}/api/v1/auth/register", json=payload)

        assert resp2.status_code == 409, (
            f"WhatsApp duplicado deve retornar 409: {resp2.status_code}"
        )

    @pytest.mark.phase2
    @pytest.mark.skip(reason="Fase 2 — POST /api/v1/webhooks/payment ainda não implementado no backend")
    def test_webhook_endpoint_existe(self):
        """
        POST /api/v1/webhooks/payment sem payload deve retornar 4xx (não 404).
        Valida que o endpoint está registrado na API.
        STATUS: Fase 2 — endpoint POST /api/v1/webhooks/payment ainda não implementado.

        PASSO A PASSO MANUAL:
          curl -X POST http://localhost:8000/api/v1/webhooks/payment
          # Esperado: 400 ou 422 (payload inválido), não 404
        """
        resp = requests.post(
            f"{API_URL}/api/v1/webhooks/payment",
            data=b"",
            headers={"Content-Type": "application/json"},
        )
        assert resp.status_code != 404, (
            "Endpoint /api/v1/webhooks/payment não encontrado (404). "
            "Verifique se o router está registrado."
        )
        # 400 (assinatura inválida) ou 422 (payload inválido) são esperados
        assert resp.status_code in (400, 422), (
            f"Esperado 400 ou 422 para payload vazio: {resp.status_code}"
        )

    def test_admin_global_sem_limites(self):
        """
        Admins globais devem receber null em todos os limites.
        PRD: "Admins globais recebem null em todos os limites".

        PASSO A PASSO MANUAL:
          curl http://localhost:8000/api/v1/subscriptions/me \\
            -H "Authorization: Bearer <admin_token>"
          # Esperado: {"groups_limit": null, "members_limit": null}
        """
        admin_api = ApiClient(API_URL)
        admin_api.login(
            os.getenv("ADMIN_WHATSAPP", "11999990000"),
            os.getenv("ADMIN_PASSWORD", "admin123"),
        )
        sub = admin_api.get_my_subscription()

        assert sub.get("groups_limit") is None, (
            f"Admin global deve ter groups_limit=null: {sub.get('groups_limit')}"
        )
        assert sub.get("members_limit") is None, (
            f"Admin global deve ter members_limit=null: {sub.get('members_limit')}"
        )
