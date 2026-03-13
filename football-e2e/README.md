# football-e2e

Testes end-to-end do rachao.app usando [Playwright](https://playwright.dev/python/) + [pytest](https://docs.pytest.org/).

---

## Arquitetura

```
football-e2e/
├── conftest.py              # Fixtures globais: login, contextos autenticados
├── pages/                   # Page Object Model + clientes auxiliares
│   ├── login_page.py
│   ├── dashboard_page.py
│   ├── group_page.py
│   ├── match_page.py
│   └── api_client.py        # Cliente HTTP direto para a API (sem browser)
└── tests/                   # Suites de testes por domínio
    ├── test_auth.py          # Login, logout, redirecionamentos de acesso
    ├── test_groups.py        # Grupos: abas, convite, adicionar membro
    ├── test_matches.py       # Rachões: listagem, status, navegação
    ├── test_players.py       # Jogadores: listagem, edição, busca
    ├── test_attendance.py    # Partidas: acesso público, presença
    └── test_stripe_webhooks.py # Integração Stripe Billing (planos, webhooks)
```

### Page Object Model (POM)

Cada página da aplicação é representada por uma classe em `pages/`. Os testes interagem somente com os métodos dessas classes, nunca com seletores diretamente. Isso centraliza a manutenção: se um componente mudar, apenas o POM é atualizado.

```python
# uso no teste
gp = GroupPage(page)
gp.tab_members()
gp.invite_button().click()
```

### Fixtures de autenticação

O login é feito uma única vez por sessão (`scope="session"`) e o estado resultante (cookies + localStorage com o token JWT) é salvo em disco e reutilizado por todos os testes que usam `admin_page`. Isso elimina overhead de autenticação repetida.

```
conftest.py
└── admin_storage_state (session)  ← faz login uma vez, salva estado
    └── admin_page (function)      ← novo contexto por teste, carrega estado salvo
```

---

## Pré-requisitos

- Python 3.11+
- Stack local rodando (`docker compose up` em `football-api/`)
- Para testes Stripe: Stripe CLI instalado e conta configurada (ver seção abaixo)

---

## Executar localmente

### 1. Instalar dependências

> **Sem pip instalado?** No Ubuntu/Debian, `pip` não vem por padrão no Python do sistema.
> Instale com `sudo apt install python3-pip python3-venv -y` e use um virtualenv:

```bash
cd football-e2e

# Cria e ativa o virtualenv (recomendado)
python3 -m venv .venv
source .venv/bin/activate

# Instala as dependências do projeto
pip install -e .

# Instala o browser Playwright
playwright install chromium
```

A partir daí, sempre que abrir um novo terminal para rodar os testes:

```bash
cd football-e2e
source .venv/bin/activate
```

### 2. Configurar ambiente

```bash
cp .env.example .env
# edite se necessário (padrão já aponta para localhost:3000 / localhost:8000)
```

### 3. Rodar os testes

```bash
# todos os testes (UI + API)
pytest tests/ -v

# suite específica
pytest tests/test_auth.py -v

# com screenshots em falha
pytest tests/ -v --screenshot=only-on-failure --output=test-results/

# modo headed (vê o browser abrindo)
pytest tests/ --headed
```

---

## Variáveis de ambiente

| Variável                  | Descrição                                              | Padrão                    |
|---------------------------|--------------------------------------------------------|---------------------------|
| `BASE_URL`                | URL base do frontend                                   | `http://localhost:3000`   |
| `ADMIN_WHATSAPP`          | WhatsApp do usuário admin                              | `11999990000`             |
| `ADMIN_PASSWORD`          | Senha do usuário admin                                 | `admin123`                |
| `API_URL`                 | URL direta da API (sem frontend)                       | `http://localhost:8000`   |
| `STRIPE_WEBHOOK_SECRET`   | Signing secret do `stripe listen` (`whsec_...`)        | —                         |
| `STRIPE_WEBHOOK_TIMEOUT`  | Segundos para aguardar processamento de webhook        | `15`                      |

---

## Suites de testes

| Arquivo                    | Cenários cobertos                                                     |
|----------------------------|-----------------------------------------------------------------------|
| `test_auth.py`             | Login válido/inválido, logout, redirect sem autenticação              |
| `test_groups.py`           | 3 abas do grupo, modal convite, modal adicionar membro                |
| `test_matches.py`          | Aba Próximos/Últimos, navegação para detalhes, status                 |
| `test_players.py`          | Listagem, busca, modais editar e resetar senha                        |
| `test_attendance.py`       | Acesso público à partida, contagem confirmados, compartilhar          |
| `test_stripe_webhooks.py`  | Planos, limites de plano, checkout, webhooks Stripe, idempotência     |

---

## Testes de integração Stripe

Os testes em `test_stripe_webhooks.py` cobrem os cenários de validação do
[PRD de planos de assinatura](../docs/prd/planos-assinatura.md) (seções 9.4, 14.7 e 14.9).

Há dois grupos de testes:

- **Sem Stripe CLI** (`-m "not stripe_cli"`): testam a API diretamente (planos, limites, checkout session, smoke tests). Rodam sem nenhuma configuração extra além da API estar no ar.
- **Com Stripe CLI** (`-m stripe_cli`): disparam eventos reais via `stripe trigger` e verificam que o webhook handler processa e atualiza o banco corretamente.

### Pré-requisitos para testes com Stripe CLI

**1. Instale o Stripe CLI**

```bash
# Ubuntu/Debian
curl -s https://packages.stripe.dev/api/security/keypair/stripe-cli-gpg/public \
  | gpg --dearmor | sudo tee /usr/share/keyrings/stripe.gpg > /dev/null
echo "deb [signed-by=/usr/share/keyrings/stripe.gpg] \
  https://packages.stripe.dev/stripe-cli-debian-local stable main" \
  | sudo tee /etc/apt/sources.list.d/stripe.list
sudo apt update && sudo apt install stripe -y

# macOS
brew install stripe/stripe-cli/stripe
```

**2. Autentique o CLI**

```bash
stripe login
# Abrirá o browser para autorizar. Siga as instruções.
```

**3. Inicie o listener de webhooks** (terminal separado, deixe rodando)

```bash
stripe listen --forward-to http://localhost:8000/api/v1/webhooks/payment
# Saída esperada:
# > Ready! Your webhook signing secret is whsec_xxxxxxxxxxxxxxxx (^C to quit)
```

**4. Copie o signing secret para o `.env`**

```bash
# .env
STRIPE_WEBHOOK_SECRET=whsec_xxxxxxxxxxxxxxxx   # valor exibido pelo stripe listen
```

### Rodando os testes Stripe

```bash
# Somente smoke tests (sem CLI — útil em CI antes de ter Stripe configurado)
pytest tests/test_stripe_webhooks.py -v -m "not stripe_cli"

# Somente testes que disparam eventos Stripe (requer stripe listen ativo)
pytest tests/test_stripe_webhooks.py -v -m stripe_cli

# Todos os testes Stripe
pytest tests/test_stripe_webhooks.py -v
```

### Disparar eventos manualmente (referência rápida)

```bash
# Simula pagamento aprovado (ativa plano)
stripe trigger checkout.session.completed \
  --add checkout_session:metadata.player_id=<uuid> \
  --add checkout_session:metadata.plan=basic \
  --add checkout_session:metadata.billing_cycle=monthly

# Simula renovação de fatura
stripe trigger invoice.paid \
  --override invoice:customer=<cus_xxx>

# Simula falha de pagamento (→ past_due + período de graça 7 dias)
stripe trigger invoice.payment_failed \
  --override invoice:customer=<cus_xxx>

# Simula cancelamento/expiração (→ regride para Free)
stripe trigger customer.subscription.deleted \
  --override subscription:customer=<cus_xxx> \
  --override subscription:id=<sub_xxx>

# Simula upgrade/downgrade
stripe trigger customer.subscription.updated \
  --override subscription:customer=<cus_xxx>
```

### Cartões de teste (checkout manual no browser)

| Cartão               | Comportamento              |
|----------------------|----------------------------|
| `4242 4242 4242 4242` | Pagamento aprovado         |
| `4000 0000 0000 0341` | Pagamento recusado (declined) |

---

## CI/CD

Os testes rodam automaticamente no GitHub Actions (`.github/workflows/e2e.yml`) em todo push para `main` que altere `football-frontend/`, `football-api/` ou `football-e2e/`. Também podem ser disparados manualmente via **Actions → E2E Tests → Run workflow**.

Em caso de falha, screenshots são salvas como artifact e ficam disponíveis por 7 dias.
