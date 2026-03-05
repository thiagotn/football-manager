# PRD — Planos de Assinatura
## Rachao.app · Gerenciamento de Grupos e Partidas

| | |
|---|---|
| **Versão** | 1.0 |
| **Status** | Draft |
| **Data** | Março de 2026 |
| **Plataforma** | https://rachao.app |

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
| Jogadores por grupo | 20 | 50 | Ilimitados |
| Links de convite | ✅ | ✅ | ✅ |
| Confirmação de presença | ✅ | ✅ | ✅ |
| URL pública de partida | ✅ | ✅ | ✅ |
| Histórico de partidas | 30 dias | 6 meses | Ilimitado |
| Suporte | Comunidade | E-mail | Prioritário |
| **Preço mensal** | **Grátis** | **R$ 19,90** | **R$ 49,90** |
| **Preço anual** | **Grátis** | **R$ 191,04** (-20%) | **R$ 478,80** (-20%) |

> **Nota:** Os valores acima são sugestões iniciais e devem ser validados com pesquisa de precificação antes do lançamento.

### 2.2 Definição de Limites

**Limite de Grupos:** número máximo de grupos ativos que o organizador pode possuir simultaneamente. O organizador é o `Player` com `role = 'admin'` no `GroupMember`. Grupos arquivados ou excluídos não contam para o limite.

**Limite de Partidas:** número máximo de partidas com `status = 'open'` por grupo. Partidas com `status = 'closed'` não contam para o limite. No plano Free, o limite de 3 partidas é por grupo.

**Limite de Jogadores por grupo:** baseado nos registros ativos em `group_members` (excluindo players com `role = 'admin'` global, que não participam de partidas).

**Comportamento ao atingir o limite:** ao tentar criar um recurso além do limite, o sistema exibe um modal de upsell informando o plano atual, o recurso bloqueado e as opções de upgrade disponíveis.

---

## 3. Requisitos Funcionais

### 3.1 Gestão de Planos e Assinaturas

**RF-01 — Exibição de planos**
O sistema deve exibir uma página de planos (`/planos`) com comparativo visual dos três planos, preços mensais e anuais, e botões de ação (começar grátis / assinar).

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
O usuário deve poder visualizar o histórico de faturas (data, valor, status, link para nota fiscal) em `/conta/faturas`.

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
INSERT INTO plans (name, display_name, price_monthly, price_yearly, max_groups, max_matches, max_members, history_days) VALUES
  ('free',  'Free',   0,      0,      1,  3,  20, 30),
  ('basic', 'Básico', 19.90,  191.04, 3, -1,  50, 180),
  ('pro',   'Pro',    49.90,  478.80, 10, -1, -1, -1);
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
    "upgrade_url": "/planos"
  }
}
```

---

## 7. Fluxos Principais

### 7.1 Fluxo de Upgrade de Plano

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

### 7.2 Fluxo de Falha de Pagamento

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

### 7.3 Fluxo de Downgrade

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

**Página de Planos (`/planos`)**
- Cards comparativos com os três planos lado a lado.
- Toggle mensal/anual com destaque da economia anual.
- CTA primário em cada card.
- FAQ sobre cobrança, cancelamento e reembolso.

**Painel de Conta (`/conta/assinatura`)**
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

### 9.1 Recomendação

Recomenda-se o uso do **Stripe** (com suporte via [Stripe Brazil](https://stripe.com/br)) ou **Pagar.me** para processamento local em BRL com suporte nativo a PIX e boleto.

| Critério | Stripe | Pagar.me |
|---|:---:|:---:|
| PIX nativo | ✅ | ✅ |
| Boleto | ✅ | ✅ |
| Cartão recorrente | ✅ | ✅ |
| SDK bem documentado | ✅ | ✅ |
| Webhook confiável | ✅ | ✅ |
| Suporte em PT-BR | ⚠️ | ✅ |
| Taxa por transação | ~4,99% | ~2,49% + fixo |

### 9.2 Estratégia de Webhooks

- Validar assinatura do webhook (HMAC) antes de processar.
- Registrar todos os eventos recebidos em tabela `webhook_events` com idempotency key (`event_id UNIQUE`).
- Processar eventos de forma assíncrona via FastAPI Background Tasks ou Celery.
- Retornar `200 OK` imediatamente ao gateway para evitar retentativas desnecessárias.

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
| Fraude em pagamentos | Segurança | Médio | Delegar anti-fraude ao gateway (Stripe Radar / Pagar.me) |
| Indisponibilidade do gateway | Confiabilidade | Alto | Implementar retry com backoff exponencial para webhooks |

---

## 14. Plano de Lançamento

### Fase 1 — Backend (Semanas 1–3)
- Migrations `013` e `014` (tabelas `plans`, `subscriptions`, `invoices`, `webhook_events`, colunas `archived_by_plan`).
- Seed dos planos iniciais.
- Endpoints de planos e assinaturas em `football-api/app/api/v1/routers/`.
- Repositórios em `football-api/app/db/repositories/` para `plans` e `subscriptions`.
- Lógica de verificação de limites nos routers de `groups` e `matches`.
- Processamento de webhooks.

### Fase 2 — Frontend (Semanas 3–5)
- Página de planos (`/planos`) em `football-frontend/src/routes/planos/`.
- Painel de conta (`/conta/assinatura`) em `football-frontend/src/routes/conta/`.
- Modal de upsell (seguindo padrão do `ConfirmDialog`).
- Indicadores de limite no dashboard (`/`).
- Fluxo de checkout e páginas de retorno (`/conta/checkout/sucesso`, `/conta/checkout/falha`).

### Fase 3 — Testes e Validação (Semana 6)
- Testes E2E em `football-e2e/tests/` cobrindo fluxo de upgrade, falha e regressão.
- Testes dos fluxos de falha, graça e regressão de plano.
- Teste de carga nos endpoints de verificação de limite.
- UAT (User Acceptance Testing) com usuários beta.

### Fase 4 — Lançamento (Semana 7)
- Migration de players existentes para plano Free.
- Comunicação prévia por e-mail à base de usuários.
- Ativação em produção com feature flag.
- Monitoramento intensivo nas primeiras 48 horas.

---

*Documento elaborado para uso interno da equipe de produto e engenharia do Rachao.app.*
