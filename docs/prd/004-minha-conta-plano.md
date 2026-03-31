# PRD — Minha Conta: Visualização do Plano Atual
## Rachao.app · Gerenciamento de Grupos e Partidas

| | |
|---|---|
| **Versão** | 1.1 |
| **Status** | Implementado — Março 2026 |
| **Data** | Março de 2026 |
| **Plataforma** | https://rachao.app |

---

## 1. Visão Geral

### 1.1 Contexto

O sistema de planos já está parcialmente implementado (Fase 1 — plano Free, commit `5b2b1d9`). O endpoint `GET /api/v1/subscriptions/me` já retorna `plan`, `groups_limit`, `groups_used` e `members_limit`. A página `/profile` já existe com o título "Minha Conta" e é acessível via link "Conta" no Navbar.

### 1.2 Problema

O usuário admin de grupo não tem visibilidade clara do seu plano atual nem dos limites associados. A única forma de descobrir que atingiu um limite é ao tentar criar um recurso e receber o `UpsellModal` — experiência reativa. Uma seção de plano proativa na página de conta melhora a experiência e dá ao usuário controle sobre o que está usando.

### 1.3 Objetivo

Adicionar uma seção **"Seu Plano"** à página `/profile` existente, exibindo o plano atual do usuário logado, os limites do plano e o uso real de grupos, preparando a estrutura para futuras funcionalidades (histórico de faturas, upgrade/downgrade).

### 1.4 Decisões de arquitetura

- **Rota:** seção adicionada à `/profile` existente — não criar `/account` (seria duplicata de "Minha Conta").
- **Fetch de dados:** via `$effect` no componente, seguindo o padrão de todas as outras páginas do projeto. `+page.ts` não é compatível com a arquitetura de auth client-side (token em `localStorage`).
- **Sem componente separado `PlanUsageCard`:** implementado inline na `/profile` — sem necessidade de abstração para uso único.

---

## 2. Requisitos Funcionais

**RF-01 — Seção "Seu Plano" na página `/profile`**
Exibir abaixo das informações de perfil existentes:
- Badge com o nome do plano atual (ex: `Free`)
- Uso de grupos: barra de progresso + texto "X de Y grupos utilizados"
- Limite de membros por grupo: texto "até N membros" (sem barra — `members_used` não está disponível no endpoint)

**RF-02 — Dados via endpoint existente**
Usar `GET /api/v1/subscriptions/me` (já implementado). Nenhuma alteração de backend necessária.

**RF-03 — Indicador de uso com cores**
A barra de progresso de grupos segue:
- Verde: uso < 80% do limite
- Amarelo: 80–99%
- Vermelho: 100% (limite atingido)

**RF-04 — Visibilidade**
A seção é exibida apenas para usuários não-admin (admins globais têm `groups_limit = null` — sem limites).

---

## 3. Requisitos Não Funcionais

**RNF-01** — Respeitar o design system existente (TailwindCSS, padrão visual do projeto).
**RNF-02** — Estrutura da seção preparada para receber CTA de upgrade quando planos pagos forem implementados.

---

## 4. Interface do Usuário

### 4.1 Seção adicionada à `/profile`

```
┌─────────────────────────────────────────┐
│  Seu Plano                              │
│─────────────────────────────────────────│
│  🟢 Free                                │
│                                         │
│  Grupos                                 │
│  [████░░░░░░] 1 de 1  ← vermelho        │
│                                         │
│  Membros por grupo                      │
│  até 30 membros                         │
└─────────────────────────────────────────┘
```

### 4.2 Cores da barra de progresso

| Uso | Cor |
|---|---|
| < 80% | Verde (`primary`) |
| 80–99% | Amarelo (`amber`) |
| 100% | Vermelho (`red`) |

---

## 5. Arquivos Modificados

| Arquivo | Ação |
|---|---|
| `football-frontend/src/routes/profile/+page.svelte` | Adicionar seção "Seu Plano" com fetch de `subscriptions/me` |

---

## 6. Fora de Escopo (desta versão)

- Limite de partidas abertas por grupo — não implementado no backend nem exposto em `subscriptions/me`
- `members_used` por grupo — endpoint não retorna; exibe apenas o teto
- CTA de upgrade / comparativo de planos
- Histórico de faturas (`/account/invoices`)
- Gerenciamento de assinatura — upgrade/downgrade/cancelamento

---

## 7. Critérios de Aceitação

- [x] Seção "Seu Plano" aparece em `/profile` para usuários não-admin
- [x] Exibe corretamente o nome do plano
- [x] Barra de progresso reflete `groups_used` / `groups_limit` com cores corretas
- [x] Exibe limite de membros por grupo
- [x] Admins globais não veem a seção (sem limites aplicáveis)
- [x] Sem alterações de backend necessárias

---

*Documento elaborado para uso interno da equipe de produto e engenharia do Rachao.app.*
