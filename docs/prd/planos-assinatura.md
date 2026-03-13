# PRD — Planos de Assinatura
## Rachao.app · Gerenciamento de Grupos e Partidas

| | |
|---|---|
| **Versão** | 1.5 |
| **Status** | Fase 1 Implementada · Fase 2 Backend Implementada · Fase 2 Frontend Pendente |
| **Data** | Março de 2026 |
| **Plataforma** | https://rachao.app |

---

## Estado de Implementação

### ✅ Fase 1 — Plano Free (implementada · Março 2026, commit `5b2b1d9`)

#### Backend
- **Migration `015_player_subscriptions.sql`**: tabela `player_subscriptions` com `player_id UNIQUE`, `plan VARCHAR(20)`. Seed automático para todos os players existentes.
- **`PlayerSubscription` model** + **`SubscriptionRepository`**: métodos `get_or_create`, `count_admin_groups`.
- **`PlanLimitError`**: 403 com `detail="PLAN_LIMIT_EXCEEDED"`.
- **`GET /api/v1/subscriptions/me`**: retorna `plan`, `groups_limit`, `groups_used`, `members_limit`. Admins globais recebem `null` em todos os limites.
- **`POST /api/v1/auth/register`**: auto-cadastro público. Cria player + subscription gratuita + retorna JWT. Retorna 409 se WhatsApp já cadastrado.
- **`GET /api/v1/players/signups/stats`** (admin-only): `total`, `last_7_days`, `last_30_days`, `recent` (últimos 30 registros).
- **Limites enforced no backend:**
  - `POST /api/v1/groups`: bloqueia com base no plano real da assinatura (free=1, basic=3, pro=10)
  - `POST /api/v1/groups/{id}/members`: bloqueia com base no plano real (free=30, basic=50, pro=ilimitado)
  - `POST /api/v1/invites/{token}/accept`: mesma checagem antes de criar player (evita player órfão)

> **Nota:** o limite de membros implementado é **30** (não 20 como consta na tabela original). Tabela de planos atualizada abaixo.

#### Frontend
- **`/register`**: formulário de auto-cadastro público (nome, apelido, WhatsApp, senha + confirmação).
- **`/lp`**: CTA primário alterado para "Cadastrar grátis" → `/register`.
- **`/login`**: link "Cadastre-se grátis" → `/register`.
- **`UpsellModal`** (`src/lib/components/UpsellModal.svelte`): bottom sheet mobile / modal centrado desktop.
- **Página de grupos (`/groups`)**: indicador de uso "X de Y grupos"; botão com cadeado ao atingir limite; abre `UpsellModal`.
- **Dashboard (`/`)**: seção "Novos Cadastros" (apenas super-admins) com contadores de total, últimos 7 e 30 dias, e lista dos 30 registros mais recentes.
- **`src/lib/plans.ts`**: arquivo de configuração centralizado de planos no frontend (nomes, preços, limites, highlights). Fonte de verdade enquanto não houver endpoint `GET /api/v1/plans`. Commit `Março 2026`.
- **`/lp`**: seção "Planos" com cards dos três planos (Free disponível, Básico/Pro com badge "em breve"). Usa `src/lib/plans.ts`.
- **`/register`**: banner do plano selecionado no topo do formulário, exibindo nome, preço e highlights. Suporta query param `?plan=` para pré-selecionar plano (ex: `/register?plan=free`). Plano padrão: `free`.

---

### ✅ Fase 2 — Backend Stripe Billing (implementada · Março 2026, commits `3001feb`–`abed8e8`)

#### Backend
- **Migration `022_stripe_checkout_fields.sql`**: adiciona `gateway_customer_id`, `gateway_sub_id`, `status`, `current_period_end`, `grace_period_end` na tabela `player_subscriptions`.
- **Migration `023_webhook_events.sql`**: tabela `webhook_events` com `event_id UNIQUE` para garantir idempotência (RNF-02).
- **`app/services/billing.py`**: abstração do gateway de pagamento (seção 9.2). Delega para a implementação ativa via `BILLING_PROVIDER`.
- **`app/services/billing_stripe.py`**: implementação concreta com Stripe SDK — `get_or_create_customer`, `create_checkout_session`, `verify_webhook_signature`.
- **`POST /api/v1/subscriptions`**: cria Stripe Customer (ou reutiliza existente) + Checkout Session. Retorna `checkout_url` para redirect do frontend.
- **`POST /api/v1/webhooks/payment`**: handler completo com verificação de assinatura HMAC-SHA256. Eventos tratados:
  - `checkout.session.completed` → ativa plano, busca `current_period_end` da Subscription Stripe (fallback calculado por `billing_cycle` em eventos sintéticos)
  - `invoice.paid` → renova `current_period_end`, mantém `status=active`
  - `invoice.payment_failed` → `status=past_due`, define `grace_period_end = NOW() + 7d`
  - `customer.subscription.deleted` → `plan=free`, `status=canceled`
  - `customer.subscription.updated` → atualiza plano via `metadata.plan`
- **Limites de grupos e membros**: atualizados para usar o plano real da assinatura (não mais hardcoded para free).

#### Testes E2E (`football-e2e/tests/test_stripe_webhooks.py`)
- **`TestSubscriptionDefault`** (5 testes) ✅ — novo player tem plano free, limites corretos, 401 sem auth.
- **`TestPlanLimits`** (3 testes) ✅ — criação permitida, bloqueio 403 no segundo grupo, `groups_used` incrementado.
- **`TestWebhookCheckoutCompleted`** (2 testes) ✅ — ativa plano basic via `stripe trigger`, verifica `current_period_end`.
- **`TestWebhookIdempotency.test_webhook_com_event_id_duplicado_retorna_200`** ✅ — idempotência com HMAC correto.
- **`TestValidacaoFinal`** (4 testes) ✅ — health check, registro público, 409 duplicado, admin sem limites.
- Demais testes de webhook (`invoice.paid`, `payment_failed`, `subscription.deleted/updated`) requerem `stripe listen` ativo — skipped automaticamente sem o CLI.

#### Pendente (Fases 2 Frontend – 4)
- Limite de partidas abertas por grupo (RF-09)
- Arquivamento por regressão de plano (RF-12)
- Páginas `/plans`, `/account/subscription`, `/account/invoices`
- Upgrade/downgrade/cancelamento/reativação via Customer Portal ou UI própria
- Endpoint `GET /api/v1/plans` (atualmente fonte de verdade está em `src/lib/plans.ts`)

---

## 1. Visão Geral

### 1.1 Contexto

O Rachao.app é uma plataforma de gerenciamento de grupos e partidas de futebol que permite organizar jogadores, agendar partidas, rastrear presença e gerenciar grupos de forma centralizada. Atualmente, a plataforma não possui restrições de uso — qualquer usuário autenticado pode criar grupos e partidas sem limites.

> **Nota técnica:** O modelo central de usuário da plataforma é `Player` (tabela `players`), com roles `player` e `admin`. Não existe tabela `users` separada. Toda referência a "usuário" neste documento corresponde a um registro em `players`.

### 1.2 Problema

Com o crescimento da base de usuários, a ausência de monetização impede a sustentabilidade da plataforma e a evolução das funcionalidades. É necessário introduzir um modelo de planos de assinatura que:

- Permita que novos usuários experimentem a plataforma sem custo (plano gratuito).
- Ofereça planos pagos escaláveis para organizers que gerenciam múltiplos grupos.
- Gere receita recorrente para sustentar a operação e o desenvolvimento do produto.

### 1.3 Objetivo

Implementar um sistema de planos de assinatura com controle de limites de recursos (grupos e partidas), integração com gateway de pagamento, gestão do ciclo de vida das assinaturas e comunicação clara de valor ao usuário.

---

## 2. Planos e Limites

### 2.1 Estrutura de Planos

| Recurso | **Free** | **Básico** | **Pro** |
|---|:---:|:---:|:---:|
| Grupos | 1 | 3 | 10 |
| Partidas por grupo | 3 | Ilimitadas | Ilimitadas |
| Jogadores por grupo | 30 | 50 | Ilimitados |
| Links de convite | ✅ | ✅ | ✅ |
| Confirmação de presença | ✅ | ✅ | ✅ |
| URL pública de partida | ✅ | ✅ | ✅ |
| Histórico de partidas | 30 dias | 6 meses | Ilimitado |
| Suporte | Comunidade | E-mail | Prioritário |
| **Preço mensal** | **Grátis** | A definir | A definir |
| **Preço anual** | **Grátis** | A definir | A definir |

> **Nota:** Os preços dos planos pagos (Básico e Pro) ainda não foram definidos. Um estudo de precificação deve ser realizado antes do lançamento. Os valores em `src/lib/plans.ts` foram zerados (`price_monthly: null`) até que os preços sejam confirmados.

### 2.2 Definição de Limites

**Limite de Grupos:** número máximo de grupos ativos que o organizador pode possuir simultaneamente. O organizador é o `Player` com `role = 'admin'` no `GroupMember`. Grupos arquivados ou excluídos não contam para o limite.

**Limite de Partidas:** número máximo de partidas com `status = 'open'` por grupo. Partidas com `status = 'closed'` não contam para o limite. No plano Free, o limite de 3 partidas é por grupo.

**Limite de Jogadores por grupo:** baseado nos registros ativos em `group_members` (excluindo players com `role = 'admin'` global, que não participam de partidas).

**Comportamento ao atingir o limite:** ao tentar criar um recurso além do limite, o sistema exibe um modal de upsell informando o plano atual, o recurso bloqueado e as opções de upgrade disponíveis.

---

## 3. Requisitos Funcionais

### 3.1 Gestão de Planos e Assinaturas

**RF-01 — Exibição de planos**
O sistema deve exibir uma página de planos (`/plans`) com comparativo visual dos três planos, preços mensais e anuais, e botões de ação (começar grátis / assinar).

**RF-02 — Seleção e contratação de plano**
O usuário deve poder selecionar um plano pago e ser redirecionado para o checkout. O checkout deve suportar cartão de crédito, PIX e boleto bancário.

**RF-03 — Ciclo de vida da assinatura**
O sistema deve gerenciar os seguintes estados de assinatura:
- `active` — assinatura ativa e em dia.
- `past_due` — pagamento em atraso, acesso mantido por período de graça (7 dias).
- `canceled` — assinatura cancelada, acesso mantido até o fim do período pago.
- `expired` — período expirado, plano regredido para Free.

**RF-04 — Upgrade de plano**
O usuário deve poder fazer upgrade imediato. O valor deve ser calculado pro-rata em relação ao período restante do plano atual.

**RF-05 — Downgrade de plano**
O downgrade deve ser agendado para o fim do ciclo de cobrança atual. O sistema deve exibir alerta quando o downgrade resultaria em recursos que excedem os novos limites.

**RF-06 — Cancelamento**
O usuário deve poder cancelar a assinatura a qualquer momento. O acesso ao plano pago deve ser mantido até o fim do período já pago. Após o vencimento, o plano regride automaticamente para Free.

**RF-07 — Reativação**
Usuários com assinatura cancelada ou expirada devem poder reativar o plano de forma simples, sem perda de dados históricos.

**RF-16 — Exibição de planos na Landing Page**
A `/lp` deve exibir uma seção "Planos" com cards dos planos disponíveis (Free, Básico, Pro), seus preços e highlights. Planos ainda não disponíveis devem exibir badge "Em breve", botão desabilitado e **não exibir preço** (mostrar "Preço a definir" até que os valores sejam confirmados). O card do plano Free deve ter destaque visual e CTA "Cadastrar grátis" → `/register`.

**RF-17 — Banner de plano no cadastro**
A página `/register` deve exibir um banner com o plano selecionado (nome, preço, highlights). O plano é determinado pelo query param `?plan=` (ex: `/register?plan=free`). Se omitido, usa `free`. Quando planos pagos estiverem disponíveis, o CTA de planos pagos na `/lp` redirecionará para `/register?plan=basic` ou `/register?plan=pro`.

**RF-18 — Configuração centralizada de planos (`src/lib/plans.ts`)**
O frontend deve manter um único arquivo de configuração de planos com nomes, preços, limites, highlights e flags de disponibilidade. Este arquivo é a fonte de verdade para todos os componentes que exibem informações de plano enquanto não houver endpoint `GET /api/v1/plans` implementado.

### 3.2 Controle de Limites de Recursos

**RF-08 — Validação na criação de grupo**
Ao criar um grupo (endpoint `POST /api/v1/groups`), o backend deve verificar se o player autenticado atingiu o limite de grupos do seu plano atual. Se sim, retornar erro `403 PLAN_LIMIT_EXCEEDED` com detalhes do limite.

**RF-09 — Validação na criação de partida**
Ao criar uma partida (endpoint `POST /api/v1/groups/{group_id}/matches`), o backend deve verificar se o grupo atingiu o limite de partidas abertas (`status = 'open'`) do plano atual do organizador. Se sim, retornar erro `403 PLAN_LIMIT_EXCEEDED`.

**RF-10 — Feedback visual de limites**
O frontend deve exibir indicadores de uso dos recursos (ex: "2 de 3 grupos utilizados"). Ao atingir 100% do limite, o botão de criação deve exibir ícone de cadeado e acionar o modal de upsell ao ser clicado.

**RF-11 — Modal de upsell**
O modal de upsell deve exibir: plano atual do usuário, limite atingido, benefícios do próximo plano e botão direto para upgrade.

**RF-12 — Regressão de plano sem perda de dados**
Ao regredir para um plano com menos recursos, os dados existentes não devem ser deletados automaticamente. Grupos e partidas que excedem os novos limites devem ser marcados como `archived_by_plan = true` e ficar inacessíveis até que o usuário faça upgrade ou realize a limpeza manual.

### 3.3 Gestão de Pagamentos

**RF-13 — Histórico de pagamentos**
O usuário deve poder visualizar o histórico de faturas (data, valor, status, link para nota fiscal) em `/account/invoices`.

**RF-14 — Atualização de método de pagamento**
O usuário deve poder atualizar o cartão de crédito a qualquer momento sem cancelar a assinatura.

**RF-15 — Notificações de cobrança**
O sistema deve enviar notificações por e-mail (e, opcionalmente, WhatsApp) nos seguintes eventos:
- 3 dias antes da renovação.
- Confirmação de pagamento aprovado.
- Falha no pagamento.
- Período de graça iniciado.
- Assinatura cancelada ou expirada.

---

## 4. Requisitos Não Funcionais

**RNF-01 — Consistência dos limites:** a verificação de limites deve ocorrer no backend (FastAPI), nunca apenas no frontend, para evitar bypass.

**RNF-02 — Idempotência de webhooks:** o processamento de webhooks do gateway de pagamento deve ser idempotente para evitar cobranças ou ativações duplicadas.

**RNF-03 — Latência do checkout:** o redirecionamento para o checkout não deve exceder 2 segundos.

**RNF-04 — Disponibilidade:** a funcionalidade de verificação de plano deve ter disponibilidade de 99,9%, pois impacta diretamente todas as operações de criação de recursos.

**RNF-05 — Segurança:** dados de cartão nunca devem trafegar pelos servidores da aplicação. Todo processamento de pagamento deve ser delegado ao gateway (PCI compliance).

---

## 5. Modelagem de Dados

### 5.1 Novas tabelas

> **Nota:** seguir a convenção de migrations numeradas do projeto (`013_*.sql`, `014_*.sql`, etc.), localizadas em `football-api/migrations/`.

```sql
-- Migration: 013_plans_and_subscriptions.sql

-- Definição dos planos disponíveis
CREATE TABLE plans (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          VARCHAR(50) NOT NULL,           -- 'free', 'basic', 'pro'
    display_name  VARCHAR(100) NOT NULL,
    price_monthly NUMERIC(10,2) NOT NULL DEFAULT 0,
    price_yearly  NUMERIC(10,2) NOT NULL DEFAULT 0,
    max_groups    INT NOT NULL DEFAULT 1,          -- -1 = ilimitado
    max_matches   INT NOT NULL DEFAULT 3,          -- -1 = ilimitado, partidas abertas por grupo
    max_members   INT NOT NULL DEFAULT 20,         -- -1 = ilimitado, membros por grupo
    history_days  INT NOT NULL DEFAULT 30,         -- -1 = ilimitado
    is_active     BOOLEAN NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ DEFAULT NOW()
);

-- Assinaturas dos players (organizadores)
-- Referencia a tabela 'players' (não 'users')
CREATE TABLE subscriptions (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    player_id            UUID NOT NULL REFERENCES players(id),
    plan_id              UUID NOT NULL REFERENCES plans(id),
    status               VARCHAR(20) NOT NULL DEFAULT 'active',
    -- 'active' | 'past_due' | 'canceled' | 'expired'
    billing_cycle        VARCHAR(10) NOT NULL DEFAULT 'monthly',
    -- 'monthly' | 'yearly'
    current_period_start TIMESTAMPTZ NOT NULL,
    current_period_end   TIMESTAMPTZ NOT NULL,
    grace_period_end     TIMESTAMPTZ,
    canceled_at          TIMESTAMPTZ,
    gateway_customer_id  VARCHAR(255),             -- ID no gateway de pagamento
    gateway_sub_id       VARCHAR(255),             -- ID da assinatura no gateway
    created_at           TIMESTAMPTZ DEFAULT NOW(),
    updated_at           TIMESTAMPTZ DEFAULT NOW()
);

-- Histórico de faturas
CREATE TABLE invoices (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subscription_id    UUID NOT NULL REFERENCES subscriptions(id),
    player_id          UUID NOT NULL REFERENCES players(id),
    amount             NUMERIC(10,2) NOT NULL,
    status             VARCHAR(20) NOT NULL,
    -- 'pending' | 'paid' | 'failed' | 'refunded'
    gateway_invoice_id VARCHAR(255),
    paid_at            TIMESTAMPTZ,
    due_at             TIMESTAMPTZ NOT NULL,
    invoice_url        TEXT,
    created_at         TIMESTAMPTZ DEFAULT NOW()
);

-- Registro de eventos de webhook para idempotência
CREATE TABLE webhook_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    gateway         VARCHAR(50) NOT NULL,          -- 'stripe' | 'pagarme'
    event_id        VARCHAR(255) NOT NULL UNIQUE,  -- idempotency key
    event_type      VARCHAR(100) NOT NULL,
    payload         JSONB NOT NULL,
    processed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);
```

```sql
-- Migration: 014_plan_limits_on_groups_matches.sql

-- Suporte a arquivamento por limite de plano
ALTER TABLE groups  ADD COLUMN archived_by_plan BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE matches ADD COLUMN archived_by_plan BOOLEAN NOT NULL DEFAULT FALSE;
```

### 5.2 Seed de planos iniciais

```sql
-- Nota: preços de Básico e Pro a definir após estudo de precificação
INSERT INTO plans (name, display_name, price_monthly, price_yearly, max_groups, max_matches, max_members, history_days) VALUES
  ('free',  'Free',   0,    0,    1,  3,  30, 30),
  ('basic', 'Básico', 0,    0,    3, -1,  50, 180),  -- preço a definir
  ('pro',   'Pro',    0,    0,   10, -1,  -1, -1);   -- preço a definir
```

---

## 6. Endpoints da API

### 6.1 Planos

| Método | Endpoint | Descrição |
|---|---|---|
| `GET` | `/api/v1/plans` | Lista todos os planos disponíveis |
| `GET` | `/api/v1/plans/{plan_id}` | Detalha um plano específico |

### 6.2 Assinaturas

| Método | Endpoint | Descrição |
|---|---|---|
| `GET` | `/api/v1/subscriptions/me` | Retorna assinatura e limites do player logado |
| `POST` | `/api/v1/subscriptions` | Inicia checkout para novo plano |
| `PATCH` | `/api/v1/subscriptions/me/upgrade` | Faz upgrade imediato de plano |
| `PATCH` | `/api/v1/subscriptions/me/downgrade` | Agenda downgrade para fim do ciclo |
| `DELETE` | `/api/v1/subscriptions/me` | Cancela assinatura |
| `POST` | `/api/v1/subscriptions/me/reactivate` | Reativa assinatura cancelada |

### 6.3 Faturas

| Método | Endpoint | Descrição |
|---|---|---|
| `GET` | `/api/v1/invoices` | Lista faturas do player logado |
| `GET` | `/api/v1/invoices/{invoice_id}` | Detalha uma fatura |

### 6.4 Webhooks

| Método | Endpoint | Descrição |
|---|---|---|
| `POST` | `/api/v1/webhooks/payment` | Recebe eventos do gateway de pagamento |

#### Eventos de webhook tratados:
- `subscription.activated`
- `subscription.renewed`
- `subscription.payment_failed`
- `subscription.canceled`
- `subscription.expired`
- `invoice.paid`
- `invoice.overdue`

### 6.5 Resposta de erro de limite

```json
{
  "error": "PLAN_LIMIT_EXCEEDED",
  "message": "Você atingiu o limite de grupos do seu plano atual.",
  "details": {
    "resource": "groups",
    "current_plan": "free",
    "limit": 1,
    "current_usage": 1,
    "upgrade_url": "/plans"
  }
}
```

---

## 7. Fluxos Principais

### 7.1 Fluxo de Seleção de Plano no Cadastro (novo usuário)

```
Usuário acessa /lp
    ↓
Vê seção "Planos" com Free, Básico, Pro
    ↓
Clica em "Cadastrar grátis" → /register?plan=free
(futuramente: clica em plano pago → /register?plan=basic)
    ↓
/register exibe banner com detalhes do plano selecionado
    ↓
Usuário preenche formulário e cria conta
    ↓
Conta criada com plano Free
(futuramente: plano pago → redireciona para checkout antes de ativar)
```

### 7.2 Fluxo de Upgrade de Plano (usuário existente)

```
Usuário tenta criar recurso além do limite
    ↓
Frontend exibe Modal de Upsell
    ↓
Usuário clica em "Fazer Upgrade"
    ↓
Frontend chama POST /api/v1/subscriptions (checkout)
    ↓
Backend cria sessão de checkout no gateway
    ↓
Usuário é redirecionado para o checkout do gateway
    ↓
Pagamento aprovado → Gateway dispara webhook
    ↓
Backend processa webhook:
  - Atualiza status da subscription → 'active'
  - Atualiza plan_id do player
  - Desarquiva recursos bloqueados (se houver)
    ↓
Usuário retorna ao app com novo plano ativo
```

### 7.3 Fluxo de Falha de Pagamento

```
Cobrança recorrente falha no gateway
    ↓
Gateway dispara webhook subscription.payment_failed
    ↓
Backend atualiza status → 'past_due'
Backend define grace_period_end = NOW() + 7 dias
    ↓
Sistema envia notificação ao usuário (e-mail / WhatsApp)
    ↓
Se pagamento regularizado dentro do período de graça:
    → status volta para 'active'
    → Nenhum recurso é afetado
Se período de graça expirar sem pagamento:
    → status → 'expired'
    → Plano regride para Free
    → Recursos excedentes são arquivados (archived_by_plan = true)
```

### 7.4 Fluxo de Downgrade

```
Usuário solicita downgrade (ex: Pro → Básico)
    ↓
Backend verifica se recursos atuais excedem novo limite
    ↓
Se exceder: Frontend exibe aviso com lista de recursos afetados
Usuário confirma o downgrade
    ↓
Backend agenda downgrade para current_period_end
    ↓
Na data de vencimento:
    - Novo plano é ativado
    - Recursos excedentes são arquivados (não excluídos)
    - Usuário é notificado
```

---

## 8. Interface do Usuário

### 8.1 Páginas e Componentes

**Página de Planos (`/plans`)**
- Cards comparativos com os três planos lado a lado.
- Toggle mensal/anual com destaque da economia anual.
- CTA primário em cada card.
- FAQ sobre cobrança, cancelamento e reembolso.

**Painel de Conta (`/account/subscription`)**
- Plano atual com data de renovação.
- Barra de uso de recursos (grupos e partidas).
- Botões de upgrade, downgrade e cancelamento.
- Link para histórico de faturas.

**Modal de Upsell**
- Exibido ao tentar criar recurso além do limite.
- Exibe plano atual, o que está bloqueado e os benefícios do próximo plano.
- Botão de upgrade direto com fechamento opcional.
- Seguir padrão visual do `ConfirmDialog` existente (bottom sheet mobile, modal centralizado desktop).

**Indicador de limites (Dashboard)**
- Exibido no topo do dashboard: "X de Y grupos utilizados".
- Cor verde (< 80%), amarela (80–99%), vermelha (100%).

### 8.2 Wireframe — Modal de Upsell

```
┌──────────────────────────────────────────────────────┐
│  Limite atingido — Plano Free                        │
│──────────────────────────────────────────────────────│
│  Você já possui 1 grupo ativo, o máximo do seu       │
│  plano atual.                                        │
│                                                      │
│  Com o Plano Básico você terá:                       │
│  - Até 3 grupos                                      │
│  - Partidas ilimitadas                               │
│  - Histórico de 6 meses                              │
│                                                      │
│  Por apenas R$ 19,90/mês                             │
│                                                      │
│  [ Fazer upgrade ]    [ Agora não ]                  │
└──────────────────────────────────────────────────────┘
```

---

## 9. Gateway de Pagamento

### 9.1 Decisão: Stripe (v1)

**Gateway escolhido para a primeira versão: Stripe** ([Stripe Brazil](https://stripe.com/br)).

O Stripe Billing resolve nativamente os pontos mais complexos do PRD sem implementação adicional:
- Pro-rata automático em upgrades
- Dunning management (retentativas configuráveis)
- Customer Portal hosted (`/account/subscription` pode ser redirect para o portal do Stripe, sem UI própria)
- Gestão de `past_due` / graça / expiração
- SDK Python oficial, bem mantido

**Custos Stripe (referência 2025):**

| Método | Taxa |
|---|---|
| Cartão nacional | 3,4% + R$0,50 |
| Cartão internacional | 4,9% + R$0,50 |
| PIX | 1,5% |
| Boleto | 1,5% + R$1,50 fixo |
| Stripe Billing | +0,7% sobre processado (gratuito até ~R$11k/mês) |

**Tempo de integração estimado com Stripe:** ~1 semana (checkout hosted + webhook lifecycle).

---

### 9.2 Arquitetura de Abstração do Gateway (obrigatório)

O código de negócio **nunca deve chamar a SDK do Stripe diretamente** nos routers ou repositórios. Toda interação com o gateway deve passar por uma interface de serviço em:

```
football-api/app/services/billing.py
```

Essa camada expõe métodos agnósticos de gateway:

```python
# Exemplos de contrato (independente do gateway)
async def create_checkout_session(player_id, plan_name, billing_cycle) -> str  # URL
async def create_customer(player_id, email, name) -> str                        # customer_id
async def cancel_subscription(gateway_sub_id) -> None
async def get_subscription_status(gateway_sub_id) -> dict
async def create_billing_portal_session(gateway_customer_id) -> str            # URL
```

A implementação concreta fica em `app/services/billing_stripe.py` (ou `billing_pagarme.py` futuramente). O `billing.py` importa e delega para a implementação ativa, controlada por variável de ambiente:

```python
# app/services/billing.py
import os
if os.getenv("BILLING_PROVIDER", "stripe") == "stripe":
    from app.services.billing_stripe import *
```

**Benefício:** trocar de Stripe para Pagar.me (ou outro gateway) exige apenas criar `billing_pagarme.py` e mudar a variável de ambiente, sem tocar em routers, repositórios ou lógica de negócio.

---

### 9.3 Análise Comparativa de Gateways (referência para migração futura)

> Esta seção mantém o comparativo completo para subsidiar uma eventual migração de gateway quando o volume processado justificar a troca.

**Requisitos determinantes do PRD para comparação:**
- Assinaturas recorrentes com ciclos mensal e anual
- Upgrade imediato com cálculo pro-rata
- Gestão de ciclo de vida (`active → past_due → expired`) com período de graça de 7 dias
- PIX + Boleto + Cartão
- Webhooks confiáveis para ativação automática em ≤ 30 segundos

#### Stripe vs. Pagar.me vs. Asaas

| Critério | **Stripe** | **Pagar.me** | **Asaas** |
|---|:---:|:---:|:---:|
| Assinaturas nativas completas | ✅ | ⚠️ | ⚠️ |
| Pro-rata automático | ✅ | ❌ manual | ❌ manual |
| Período de graça gerenciado | ✅ | ❌ manual | ❌ manual |
| Customer portal hosted | ✅ | ❌ | ❌ |
| SDK Python oficial | ✅ | ⚠️ parcial | ❌ |
| Dunning (retentativas) | ✅ configurável | ⚠️ básico | ⚠️ básico |
| PIX | ✅ | ✅ | ✅ |
| Boleto | ✅ | ✅ | ✅ |
| Custo cartão | ~3,4% + R$0,50 | ~2,49% + R$0,39 | ~2,99% |
| Custo PIX | 1,5% | ~0,99% | ~0,99% |
| Custo boleto | 1,5% + R$1,50 | ~R$2,50 fixo | ~R$1,99 fixo |
| Suporte PT-BR | ⚠️ | ✅ | ✅ |
| Tempo dev estimado | **~1 semana** | **~2–3 semanas** | **~2–3 semanas** |
| Maturidade para SaaS | ★★★★★ | ★★★☆☆ | ★★☆☆☆ |

**Notas por gateway:**

**Pagar.me (Stone Co.):** Pro-rata, período de graça e customer portal exigem implementação manual no backend. Sem customer portal hosted, toda a UI de gerenciamento de assinatura (`/account/subscription`) precisaria ser construída do zero. A diferença de taxa (~1% no cartão, ~0,5% no PIX) começa a compensar o custo de migração a partir de ~R$30k/mês processados.

**Asaas:** Bom para cobranças simples e PMEs brasileiras. Sem SDK Python oficial. Não recomendado para o ciclo de vida complexo deste PRD.

**Iugu:** Historicamente focado em recorrência no Brasil, mas com instabilidades relatadas e documentação defasada. Não recomendado para novo projeto.

**Quando migrar de Stripe para Pagar.me:** volume mensal acima de ~R$30k processados E necessidade de suporte técnico com SLA em PT-BR. A abstração via `billing.py` (seção 9.2) garante que essa migração não exija reescrita de lógica de negócio.

---

### 9.4 Estratégia de Webhooks

- Validar assinatura do webhook (HMAC-SHA256 via `stripe.Webhook.construct_event`) antes de processar.
- Registrar todos os eventos recebidos em tabela `webhook_events` com idempotency key (`event_id UNIQUE`).
- Processar eventos de forma assíncrona via FastAPI Background Tasks.
- Retornar `200 OK` imediatamente ao gateway para evitar retentativas desnecessárias.

**Eventos Stripe a tratar:**

| Evento Stripe | Ação |
|---|---|
| `checkout.session.completed` | Ativa assinatura, atualiza `player_subscriptions` |
| `invoice.paid` | Registra fatura, renova `current_period_end` |
| `invoice.payment_failed` | Status → `past_due`, define `grace_period_end = NOW() + 7d` |
| `customer.subscription.deleted` | Status → `canceled` ou `expired`, arquiva recursos excedentes |
| `customer.subscription.updated` | Atualiza plano após upgrade/downgrade |

---

## 10. Métricas de Sucesso

| Métrica | Meta (6 meses após lançamento) |
|---|---|
| Conversão Free → Pago | ≥ 5% da base ativa |
| Churn mensal (planos pagos) | ≤ 5% |
| MRR (Receita Recorrente Mensal) | R$ 5.000 |
| NPS pós-upgrade | ≥ 7 |
| Taxa de falha no checkout | ≤ 2% |
| Tempo médio de ativação pós-pagamento | ≤ 30 segundos |

---

## 11. Critérios de Aceitação

- [ ] Usuário Free não consegue criar mais de 1 grupo (erro e upsell exibidos).
- [ ] Usuário Free não consegue criar mais de 3 partidas abertas por grupo (erro e upsell exibidos).
- [ ] Usuário Básico consegue criar até 3 grupos e partidas ilimitadas.
- [ ] Usuário Pro consegue criar até 10 grupos e partidas ilimitadas.
- [ ] Regressão de plano não exclui dados, apenas arquiva recursos excedentes.
- [ ] Webhook de pagamento aprovado ativa o plano em até 30 segundos.
- [ ] Webhook de falha inicia período de graça de 7 dias corretamente.
- [ ] Histórico de faturas exibe todas as cobranças com status correto.
- [ ] Checkout funciona para cartão de crédito, PIX e boleto.
- [ ] Indicadores de uso de recursos são exibidos corretamente no dashboard.

---

## 12. Fora de Escopo (v1)

Os itens abaixo **não fazem parte desta versão** e devem ser considerados para versões futuras:

- Plano para times/organizações (multi-admin).
- White-label para parceiros.
- Integração com notas fiscais eletrônicas (NF-e).
- Programa de afiliados.
- Período de trial pago (ex: 14 dias grátis do plano Pro).
- Cupons e descontos.
- App mobile nativo (iOS/Android).

---

## 13. Dependências e Riscos

| Item | Tipo | Impacto | Mitigação |
|---|---|---|---|
| Integração com gateway de pagamento | Dependência externa | Alto | Iniciar integração cedo; usar sandbox para testes |
| Migração de players existentes para plano Free | Técnico | Médio | Script de migração com rollback; comunicar usuários antecipadamente |
| Regressão de recursos ao fazer downgrade | UX | Alto | Avisos claros, dados nunca excluídos automaticamente |
| Fraude em pagamentos | Segurança | Médio | Delegar anti-fraude ao Stripe Radar |
| Indisponibilidade do gateway | Confiabilidade | Alto | Retry com backoff exponencial para webhooks; `webhook_events` garante idempotência |
| Migração de gateway futura | Técnico | Médio | Abstração via `app/services/billing.py` isola o código de negócio do gateway concreto |

---

## 14. Configuração Manual da Conta Stripe

> Checklist de ações que devem ser realizadas manualmente no dashboard do Stripe **antes** de iniciar o desenvolvimento da integração. Marque cada item conforme concluído.

### 14.1 Criação e Verificação da Conta

- [ ] Acessar [dashboard.stripe.com](https://dashboard.stripe.com) e criar conta com e-mail do negócio
- [ ] Confirmar e-mail
- [ ] Em **Settings → Business details**: preencher nome do negócio ("Rachao.app" ou razão social)
- [ ] Selecionar tipo de entidade: **Pessoa Física (CPF)** ou **Empresa (CNPJ)** conforme o caso
- [ ] Informar endereço e telefone brasileiros
- [ ] Em **Settings → Bank accounts and scheduling**: adicionar conta bancária brasileira para recebimento de saques
- [ ] Completar verificação de identidade (envio de documento + selfie — processo guiado pelo próprio Stripe)
- [ ] Aguardar aprovação da conta para pagamentos reais (geralmente automático em minutos para PF)

> **Atenção:** sem a verificação completa, os pagamentos ficam em modo restrito e os saques ficam bloqueados.

---

### 14.2 Ativar Métodos de Pagamento

- [ ] Em **Settings → Payment methods**:
  - [ ] Confirmar que **Cartão de crédito/débito** está ativo (padrão)
  - [ ] Ativar **PIX** (pode exigir verificação adicional da conta)
  - [ ] Ativar **Boleto bancário** (pode exigir verificação adicional da conta)
- [ ] Definir prazo de vencimento do boleto (recomendado: **3 dias**)

---

### 14.3 Criar Produtos e Preços

> Um **Product** representa cada plano; cada Product tem um ou mais **Prices** (mensal, anual).

- [ ] Em **Product catalog → Add product**: criar produto **"Plano Básico"**
  - [ ] Adicionar Price recorrente **mensal** em BRL (valor a definir)
  - [ ] Adicionar Price recorrente **anual** em BRL (valor a definir)
  - [ ] Copiar os **Price IDs** (`price_xxx`) e guardar — serão usados no código
- [ ] Em **Product catalog → Add product**: criar produto **"Plano Pro"**
  - [ ] Adicionar Price recorrente **mensal** em BRL (valor a definir)
  - [ ] Adicionar Price recorrente **anual** em BRL (valor a definir)
  - [ ] Copiar os **Price IDs** e guardar
- [ ] Repetir os itens acima no modo **Test** antes de fazer em produção (os IDs são diferentes entre ambientes)

---

### 14.4 Configurar o Customer Portal

> O Customer Portal é a UI hosted do Stripe que substitui a necessidade de construir `/account/subscription` do zero.

- [ ] Em **Settings → Customer portal**:
  - [ ] Habilitar **Cancel subscriptions**
  - [ ] Habilitar **Update subscriptions** (upgrade/downgrade entre planos)
  - [ ] Habilitar **Update payment methods**
  - [ ] Habilitar **View invoice history**
  - [ ] Em "Business information": adicionar logo e cores do Rachao.app
  - [ ] Em "Cancellation reasons": habilitar coleta de motivo de cancelamento
  - [ ] Salvar configuração
  - [ ] Copiar a **URL do portal** gerada (usada pelo `billing.py` para redirect)

---

### 14.5 Configurar Webhook

- [ ] Em **Developers → Webhooks → Add endpoint**:
  - [ ] URL: `https://rachao.app/api/v1/webhooks/payment` (produção)
  - [ ] URL alternativa para testes: usar **Stripe CLI** localmente (`stripe listen --forward-to localhost:8000/api/v1/webhooks/payment`)
  - [ ] Selecionar os eventos (mínimo necessário):
    - [ ] `checkout.session.completed`
    - [ ] `invoice.paid`
    - [ ] `invoice.payment_failed`
    - [ ] `customer.subscription.updated`
    - [ ] `customer.subscription.deleted`
  - [ ] Salvar e copiar o **Webhook Signing Secret** (`whsec_xxx`) — usado para validar autenticidade dos eventos

---

### 14.6 Obter Chaves de API

- [ ] Em **Developers → API keys**:
  - [ ] Copiar **Publishable key** de teste (`pk_test_xxx`)
  - [ ] Copiar **Secret key** de teste (`sk_test_xxx`)
  - [ ] Copiar **Publishable key** de produção (`pk_live_xxx`)
  - [ ] Copiar **Secret key** de produção (`sk_live_xxx`)
- [ ] Adicionar ao `.env` do projeto:
  ```
  STRIPE_SECRET_KEY=sk_test_xxx        # trocar para sk_live_xxx em produção
  STRIPE_PUBLISHABLE_KEY=pk_test_xxx
  STRIPE_WEBHOOK_SECRET=whsec_xxx
  BILLING_PROVIDER=stripe
  ```
- [ ] Confirmar que `.env` está no `.gitignore` e **nunca** versionar as chaves

---

### 14.7 Instalar Stripe CLI (desenvolvimento local)

- [ ] Instalar Stripe CLI: `brew install stripe/stripe-cli/stripe` (macOS) ou [instruções Linux](https://stripe.com/docs/stripe-cli)
- [ ] Autenticar: `stripe login`
- [ ] Para testar webhooks localmente: `stripe listen --forward-to localhost:8000/api/v1/webhooks/payment`
- [ ] Para disparar eventos manualmente durante o desenvolvimento: `stripe trigger invoice.paid`

---

### 14.8 Configurar Dunning (retentativas automáticas)

- [ ] Em **Settings → Subscriptions and emails → Manage failed payments**:
  - [ ] Habilitar retentativas automáticas (recomendado: 3, 5 e 7 dias após falha)
  - [ ] Habilitar envio de e-mail automático do Stripe ao cliente em caso de falha
  - [ ] Definir ação após esgotar retentativas: **"Cancel the subscription"** (o backend trata o webhook `customer.subscription.deleted`)

---

### 14.9 Validação Final (antes de ir a produção)

- [ ] Realizar um checkout completo com cartão de teste `4242 4242 4242 4242`
- [ ] Verificar que o webhook `checkout.session.completed` chegou e foi processado
- [ ] Verificar que a assinatura foi criada e o plano atualizado no banco
- [ ] Testar falha de pagamento com cartão `4000 0000 0000 0341`
- [ ] Verificar que `past_due` e `grace_period_end` foram definidos corretamente
- [ ] Testar cancelamento via Customer Portal e verificar webhook
- [ ] Testar PIX em modo sandbox (Stripe disponibiliza simulador)
- [ ] Revisar todos os Price IDs no código — confirmar que apontam para os IDs de **produção** antes do deploy

---

## 15. Plano de Lançamento

### ✅ Fase 1 — Backend (concluída · Março 2026)
- ✅ Migration `015_player_subscriptions.sql` (tabela `player_subscriptions`, seed para players existentes).
- ✅ Migration `022_stripe_checkout_fields.sql` (campos Stripe na `player_subscriptions`).
- ✅ Migration `023_webhook_events.sql` (tabela `webhook_events` para idempotência).
- ✅ `app/services/billing.py` — interface de abstração do gateway (seção 9.2).
- ✅ `app/services/billing_stripe.py` — implementação concreta via Stripe SDK.
- ✅ `GET /api/v1/subscriptions/me`, `POST /api/v1/subscriptions`, `POST /api/v1/webhooks/payment`.
- ✅ `SubscriptionRepository` com todos os métodos necessários.
- ✅ Lógica de verificação de limites nos routers de `groups` (por plano real, não hardcoded).
- ⏳ Limite de partidas abertas por grupo (RF-09) — pendente.
- ⏳ Migrations de `plans`, `invoices`, `archived_by_plan` — pendente (não priorizadas para MVP).

### Fase 2 — Frontend (em andamento · Março 2026)

#### ✅ Implementado
- `src/lib/plans.ts`: configuração centralizada de planos (fonte de verdade do frontend).
- Seção "Planos" na `/lp` com cards comparativos (Free ativo, Básico/Pro em breve).
- Banner de plano selecionado no `/register` com suporte a `?plan=` query param.

#### ⏳ Pendente
- Página de planos (`/plans`) em `football-frontend/src/routes/plans/` — cards detalhados com toggle mensal/anual.
- Painel de conta (`/account/subscription`) em `football-frontend/src/routes/account/`.
- Fluxo de checkout e páginas de retorno em `football-frontend/src/routes/account/checkout/` (`/account/checkout/success`, `/account/checkout/failure`).
- CTA de upgrade nos cards Básico/Pro da `/lp` ao ativar planos pagos: redirecionar para `/register?plan=basic` (novo usuário) ou `/account/subscription` (usuário logado).

### ✅ Fase 3 — Testes E2E (parcialmente concluída · Março 2026)
- ✅ `football-e2e/tests/test_stripe_webhooks.py` — 24 testes (14 pass, 9 skip aguardando `stripe listen`, 1 pendente de fase 2 frontend).
- ✅ Testes de plano free, limites, checkout, idempotência, health check, registro público.
- ⏳ Testes dos fluxos de falha de pagamento, graça e regressão — requerem `stripe listen` ativo (marcados `@stripe_cli`).
- ⏳ Teste de carga nos endpoints de verificação de limite.
- ⏳ UAT (User Acceptance Testing) com usuários beta.

### Fase 4 — Lançamento
- Migration de players existentes para plano Free (já feita via seed na migration 015).
- Comunicação prévia por e-mail à base de usuários.
- Ativação em produção com feature flag.
- Monitoramento intensivo nas primeiras 48 horas.

---

*Documento elaborado para uso interno da equipe de produto e engenharia do Rachao.app.*
