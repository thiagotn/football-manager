# PRD — Painel Admin: Gestão de Assinaturas

**Versão:** 1.1
**Data:** 2026-03-14
**Status:** ✅ Implementado — Março 2026

---

## 1. Contexto

Com o lançamento do sistema de billing (Stripe), o super admin precisa de visibilidade sobre as assinaturas da plataforma. Hoje o painel admin (`/admin`) exibe apenas métricas de uso (rachões, grupos, jogadores). Não há como saber quem está pagando, quais assinaturas estão com problema, ou qual é a receita corrente sem acessar o Stripe Dashboard diretamente.

Esta feature cria a página `/admin/subscriptions` com uma visão administrativa centralizada das assinaturas, complementando o painel existente sem substituí-lo.

---

## 2. Objetivo

Dar ao super admin visibilidade operacional sobre:
- Quantos usuários pagam e quais planos têm
- Assinaturas com problema (past_due, grace period ativo)
- Receita recorrente estimada (MRR)
- Histórico recente de conversões e cancelamentos

---

## 3. Personas

**Super Admin** (`PlayerRole.ADMIN`) — único perfil com acesso. Acessa periodicamente para checar saúde financeira da plataforma e intervir em casos de problema.

---

## 4. Requisitos Funcionais

### RF-01 — Cards de resumo (topo da página)

Exibir 5 métricas em destaque:

| Métrica | Descrição |
|---|---|
| **Assinantes ativos** | Contagem de `status=active` com `plan != free` |
| **Free** | Contagem de `plan=free` |
| **Past Due** | Contagem de `status=past_due` — requer atenção |
| **Cancelados** | Contagem de `status=canceled` |
| **MRR estimado** | Soma dos planos ativos: basic_monthly=R$19,90 / basic_yearly=R$199/12 / pro_monthly=R$39,90 / pro_yearly=R$399/12 |

> O MRR é calculado no backend com os valores de preço configurados no sistema. Não é consultado via Stripe API.

### RF-02 — Breakdown por plano

Gráfico simples (ou tabela) mostrando a distribuição:
- Free / Basic Mensal / Basic Anual / Pro Mensal / Pro Anual
- Com contagem e percentual do total de jogadores

### RF-03 — Tabela de assinantes pagos

Lista paginada (20/página) de todos os jogadores com `plan != free`, contendo:

| Coluna | Fonte |
|---|---|
| Jogador (nome) | `players.name` |
| Plano | `player_subscriptions.plan` |
| Ciclo | Inferido via `gateway_sub_id` ou campo novo `billing_cycle` |
| Status | `status` com badge colorido |
| Vencimento | `current_period_end` formatado |
| Grace até | `grace_period_end` (visível apenas quando `status=past_due`) |
| Customer ID | `gateway_customer_id` com link direto para o Stripe Dashboard |

**Filtros disponíveis:**
- Por status: Todos / Ativo / Past Due / Cancelado
- Por plano: Todos / Basic / Pro

**Ordenação padrão:** `status=past_due` primeiro, depois por `current_period_end` ASC (quem vence primeiro).

### RF-04 — Ação de reativação manual

Para casos onde o webhook falhou (como o incidente documentado no projeto), o admin pode forçar a ativação de um plano diretamente:

- Botão **"Ativar plano"** em cada linha da tabela
- Abre um modal com campos: Plano (select), Ciclo (mensal/anual), Observação
- Chama endpoint admin `PATCH /api/v1/admin/subscriptions/{player_id}`
- Registra no log com `reason=manual_admin_override`

> Caso de uso principal: webhook falhou em produção e o pagamento foi confirmado pelo Stripe mas o plano não foi ativado.

### RF-05 — Alertas de past_due no painel principal

No painel `/admin` (home), adicionar um banner de alerta quando houver assinaturas `past_due` com `grace_period_end` próximo (≤ 3 dias), com link para `/admin/subscriptions?filter=past_due`.

### RF-06 — Novos cards de billing no painel principal (`/admin`)

Adicionar uma seção "Billing" ao painel existente com 3 cards clicáveis:
- **Assinantes pagos** → link para `/admin/subscriptions`
- **Past due** → link para `/admin/subscriptions?filter=past_due`
- **MRR estimado** → somente leitura

---

## 5. Requisitos Não Funcionais

- **Acesso restrito:** endpoint retorna `403` para qualquer player sem `PlayerRole.ADMIN`
- **Performance:** consultas com JOIN entre `players` e `player_subscriptions`, paginadas — sem N+1
- **Sem dados de cartão:** nenhuma informação sensível de pagamento é exibida. Apenas IDs do Stripe com links externos
- **Mobile-first:** o app é primariamente mobile — toda a página deve funcionar bem em telas pequenas antes de considerar o desktop

---

## 5.1 Layout Responsivo — Especificação por Breakpoint

### Cards de resumo (RF-01)

| Breakpoint | Grid |
|---|---|
| Mobile (`< sm`) | 2 colunas — MRR ocupa largura total (`col-span-2`) na última linha |
| Desktop (`sm+`) | 5 colunas em linha única |

```
Mobile:                     Desktop:
┌──────────┬──────────┐     ┌──────┬──────┬──────┬──────┬──────┐
│  Ativos  │   Free   │     │Ativo │ Free │P.Due │Canc. │ MRR  │
├──────────┼──────────┤     └──────┴──────┴──────┴──────┴──────┘
│ Past Due │Cancelado │
├──────────┴──────────┤
│    MRR estimado     │
└─────────────────────┘
```

### Breakdown por plano (RF-02)

- **Mobile:** lista vertical de linhas (nome do plano + barra de progresso + contagem)
- **Desktop:** mantém lista vertical, mas em card de largura limitada ao lado dos filtros da tabela (layout de duas colunas com `lg:grid-cols-3`, breakdown ocupa 1 coluna e tabela ocupa 2)

### Tabela de assinantes (RF-03)

A tabela não cabe em mobile — substituir por **lista de cards** em telas pequenas:

**Mobile — card por assinante:**
```
┌─────────────────────────────────────┐
│ João Silva              [past_due ●]│
│ Basic · vence 13/04/2026            │
│ Grace até 21/03/2026                │  ← visível só se past_due
│                      [Ativar plano] │  ← ação manual
└─────────────────────────────────────┘
```

**Desktop — tabela completa:**

| Coluna | Mobile | Desktop |
|---|---|---|
| Jogador | ✅ (no card) | ✅ |
| Plano | ✅ (no card) | ✅ |
| Status | ✅ (badge no card) | ✅ |
| Ciclo | — | ✅ (`hidden sm:table-cell`) |
| Vencimento | ✅ (no card) | ✅ |
| Grace até | ✅ condicional | ✅ condicional (`hidden sm:table-cell`) |
| Customer ID / Link Stripe | — | ✅ (`hidden lg:table-cell`) |
| Ação | ✅ (no card) | ✅ |

### Filtros

- **Mobile:** filtros em linha horizontal com scroll (`flex overflow-x-auto gap-2`), chips de seleção simples
- **Desktop:** mesma linha, mas sem scroll (cabem todos)

### Modal de ativação manual (RF-04)

- **Mobile:** bottom sheet (desliza de baixo — padrão já usado no `ConfirmDialog`)
- **Desktop:** modal centralizado

### Alerta past_due no painel home (RF-05)

- **Mobile e Desktop:** banner full-width abaixo do header, com ícone `AlertTriangle` e link para `/admin/subscriptions?filter=past_due`

---

## 6. API — Novos Endpoints

### `GET /api/v1/admin/subscriptions`

Retorna lista paginada de assinaturas com dados do jogador.

**Query params:**
- `status` — filtro por status (`active`, `past_due`, `canceled`)
- `plan` — filtro por plano (`basic`, `pro`)
- `page` — padrão 1
- `page_size` — padrão 20, máximo 100

**Response:**
```json
{
  "total": 42,
  "page": 1,
  "page_size": 20,
  "items": [
    {
      "player_id": "uuid",
      "player_name": "João Silva",
      "plan": "basic",
      "status": "past_due",
      "current_period_end": "2026-04-13T00:00:00Z",
      "grace_period_end": "2026-03-21T00:00:00Z",
      "gateway_customer_id": "cus_xxx",
      "gateway_sub_id": "sub_xxx",
      "created_at": "2026-03-14T00:00:00Z"
    }
  ]
}
```

### `GET /api/v1/admin/subscriptions/summary`

Retorna os totais para os cards de resumo.

**Response:**
```json
{
  "total_players": 150,
  "active": 38,
  "free": 108,
  "past_due": 3,
  "canceled": 1,
  "mrr_cents": 75620,
  "breakdown": [
    { "plan": "basic", "cycle": "monthly", "count": 25 },
    { "plan": "basic", "cycle": "yearly",  "count": 8 },
    { "plan": "pro",   "cycle": "monthly", "count": 5 },
    { "plan": "pro",   "cycle": "yearly",  "count": 0 }
  ]
}
```

### `PATCH /api/v1/admin/subscriptions/{player_id}`

Força atualização manual do plano.

**Body:**
```json
{
  "plan": "basic",
  "status": "active",
  "reason": "webhook_failed_manual_fix"
}
```

---

## 7. Frontend — Rota e Layout

**Rota:** `/admin/subscriptions`

```
/admin/subscriptions
  ├── [Banner alerta past_due — condicional]
  ├── Cards de resumo (5 métricas)
  ├── Desktop: grid 3 colunas
  │     ├── Coluna 1: Breakdown por plano
  │     └── Colunas 2-3: Filtros + Tabela + Paginação
  └── Mobile: stack vertical
        ├── Breakdown por plano (lista)
        ├── Filtros (chips scroll horizontal)
        └── Lista de cards de assinantes
```

**Container:** `max-w-7xl mx-auto px-4 py-8` (padrão de páginas de listagem)

**Ícone de navegação:** `<CreditCard size={24} class="text-primary-400" />`

**Badge de status:**
- `active` → verde
- `past_due` → amarelo com pulsação (`animate-pulse`) para chamar atenção
- `canceled` → cinza
- `free` — não aparece na lista de pagantes

**Link externo para o Stripe:** `https://dashboard.stripe.com/customers/{gateway_customer_id}` — abre em nova aba com `<ExternalLink size={12} />` — visível apenas em desktop (`hidden lg:inline-flex`).

---

## 8. Modelo de dados — Campo `billing_cycle`

Atualmente `PlayerSubscription` não armazena o ciclo de cobrança (`monthly`/`yearly`) — essa informação está apenas no Stripe. Para exibir o ciclo na tabela sem consultar a API do Stripe a cada request:

- Adicionar coluna `billing_cycle VARCHAR(10)` na tabela `player_subscriptions`
- Preencher em `_handle_checkout_completed` via `metadata.billing_cycle` (já disponível)
- Migration: `018_add_billing_cycle_to_subscriptions.sql`

---

## 9. Fora do Escopo (v1)

- Gráficos de evolução de MRR ao longo do tempo
- Exportação CSV da lista de assinantes
- Emissão manual de cupons/descontos via Stripe
- Notificação automática ao admin quando um pagamento falha (coberto por email do Stripe)
- Cancelamento de assinatura pelo admin (deve ser feito pelo próprio usuário ou via Stripe Dashboard)
