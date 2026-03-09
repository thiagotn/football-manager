# PRD — Minha Conta: Visualização do Plano Atual
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

O sistema de planos já está parcialmente implementado (Fase 1 — plano Free, commit `5b2b1d9`). O endpoint `GET /api/v1/subscriptions/me` já retorna `plan`, `groups_limit`, `groups_used` e `members_limit`. Porém, não existe ainda uma página dedicada onde o usuário logado possa visualizar em qual plano está e quais são seus limites.

### 1.2 Problema

O usuário admin de grupo não tem visibilidade clara do seu plano atual nem dos limites associados. A única forma de descobrir que atingiu um limite é ao tentar criar um recurso e receber o `UpsellModal` — experiência reativa. Uma página de conta proativa melhora a experiência e dá ao usuário controle sobre o que está usando.

### 1.3 Objetivo

Criar a seção **"Minha Conta"** (`/account`) com foco inicial em exibir o plano atual do usuário logado, os limites do plano e o uso real, preparando a estrutura para futuras funcionalidades (histórico de faturas, upgrade/downgrade, dados pessoais).

---

## 2. Requisitos Funcionais

**RF-01 — Página /account**
Criar rota `/account` acessível apenas para usuários autenticados. Exibir no mínimo:
- Nome e apelido do usuário
- Plano atual (ex: "Free", "Básico", "Pro")
- Limites do plano com uso atual:
  - Grupos: X de Y utilizados
  - Membros por grupo: limite do plano
  - Partidas abertas por grupo: limite do plano

**RF-02 — Dados via endpoint existente**
Usar `GET /api/v1/subscriptions/me` (já implementado) para obter `plan`, `groups_limit`, `groups_used`, `members_limit`. Nenhuma alteração de backend é necessária nesta fase.

**RF-03 — Acesso via menu de navegação**
Adicionar link "Minha Conta" no menu principal (navbar/sidebar), visível apenas para usuários logados.

---

## 3. Requisitos Não Funcionais

**RNF-01** — A página deve respeitar o design system existente (TailwindCSS, padrão visual do projeto).  
**RNF-02** — Rota protegida: redirecionar para `/login` se não autenticado.  
**RNF-03** — Estrutura da página deve suportar adição futura das seções: dados pessoais, histórico de faturas, gerenciamento de assinatura (previsto no PRD de planos como `/account/subscription` e `/account/invoices`).

---

## 4. Interface do Usuário

### 4.1 Estrutura da página `/account`

```
┌─────────────────────────────────────────────┐
│  Minha Conta                                │
│─────────────────────────────────────────────│
│  👤 João Silva (@joaozinho)                 │
│─────────────────────────────────────────────│
│  SEU PLANO ATUAL                            │
│                                             │
│  ┌─────────────────────────────────────┐    │
│  │  🟢 Plano Free                      │    │
│  │                                     │    │
│  │  Grupos         ██░░░░  1 de 1      │    │
│  │  Membros/grupo  ──────  até 30      │    │
│  │  Partidas/grupo ──────  até 3       │    │
│  └─────────────────────────────────────┘    │
└─────────────────────────────────────────────┘
```

### 4.2 Indicador de uso

Seguir o mesmo padrão de cores já definido no PRD de planos:
- Verde: uso < 80% do limite
- Amarelo: uso entre 80–99%
- Vermelho: 100% (limite atingido)

---

## 5. Arquivos a Criar/Modificar

| Arquivo | Ação | Descrição |
|---|---|---|
| `football-frontend/src/routes/account/+page.svelte` | Criar | Página principal de Minha Conta |
| `football-frontend/src/routes/account/+page.ts` | Criar | Load function com chamada ao `subscriptions/me` |
| `football-frontend/src/lib/components/PlanUsageCard.svelte` | Criar | Card reutilizável de uso do plano |
| `football-frontend/src/lib/components/nav/Navbar.svelte` (ou equivalente) | Modificar | Adicionar link "Minha Conta" |

---

## 6. Fluxo de Dados

```
/account carrega
    ↓
+page.ts chama GET /api/v1/subscriptions/me
    ↓
Retorna: { plan, groups_limit, groups_used, members_limit }
    ↓
+page.svelte renderiza PlanUsageCard com os dados
```

---

## 7. Critérios de Aceitação

- [ ] Página `/account` acessível apenas para usuários logados
- [ ] Exibe nome e apelido do usuário
- [ ] Exibe o plano atual corretamente (Free / Básico / Pro)
- [ ] Exibe uso real de grupos (groups_used de groups_limit)
- [ ] Exibe limites de membros e partidas do plano
- [ ] Indicador de uso muda de cor conforme uso (verde/amarelo/vermelho)
- [ ] Link "Minha Conta" aparece no menu de navegação para usuários logados
- [ ] Estrutura de página preparada para receber seções futuras (faturas, edição de perfil)

---

## 8. Dependências

- `GET /api/v1/subscriptions/me` — já implementado (Fase 1)
- PRD de Planos de Assinatura (seções 8 e 14) — `/account/subscription` e `/account/invoices` serão subpáginas desta mesma seção no futuro

---

## 9. Fora de Escopo (desta versão)

- Edição de dados pessoais (nome, apelido, senha)
- Histórico de faturas (`/account/invoices`)
- Gerenciamento de assinatura — upgrade/downgrade/cancelamento (`/account/subscription`)
- Foto de perfil
- CTA de upsell / comparativo de planos

---

*Documento elaborado para uso interno da equipe de produto e engenharia do Rachao.app.*
