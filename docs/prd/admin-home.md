# PRD — Home Super Admin (Refatoração)
## Rachao.app · Painel Administrativo

| | |
|---|---|
| **Versão** | 1.0 |
| **Status** | Planejado |
| **Data** | Março de 2026 |
| **Plataforma** | https://rachao.app |

---

## 1. Visão Geral

### 1.1 Contexto

Atualmente, o super admin acessa a mesma rota `/dashboard` que jogadores comuns. Essa página exibe informações irrelevantes para o contexto administrativo — "Meus Grupos" e lista de "Próximos/Últimos Rachões" — misturadas com dados de plataforma (novos cadastros). A experiência é confusa e pouco útil para a gestão operacional.

### 1.2 Objetivo

Criar uma página dedicada para o super admin (`/admin`) com foco exclusivo em métricas de plataforma, apresentadas como big numbers navegáveis. A home do admin deve ser enxuta, rápida de ler e orientada a ação.

### 1.3 Estratégia de Rota

Criar uma **nova rota** `src/routes/admin/+page.svelte` em vez de adaptar o `/dashboard` existente. O `/dashboard` continuará existindo para usuários comuns. O layout ou a rota `/dashboard` deve redirecionar automaticamente o super admin para `/admin`.

---

## 2. Regras de Negócio

- Acesso exclusivo para jogadores com `players.role = 'admin'` (super admin global).
- Qualquer acesso a `/admin` por um usuário não-admin redireciona para `/dashboard`.
- Os big numbers refletem o estado atual da plataforma inteira (não por grupo).
- Os cards clicáveis navegam para páginas de listagem dedicadas.

---

## 3. Requisitos Funcionais

**RF-01 — Redirect automático**
No `/dashboard`, quando o usuário autenticado for super admin, redirecionar imediatamente para `/admin`.

**RF-02 — Proteção da rota `/admin`**
Usuários não-admin que acessarem `/admin` diretamente devem ser redirecionados para `/dashboard`.

**RF-03 — Big numbers de plataforma**
A página exibe 4 cards de métricas, cada um com um número em destaque e label:

| Card | Métrica | Destino ao clicar |
|---|---|---|
| Rachões | Total de partidas cadastradas (`COUNT(matches)`) | `/admin/matches` |
| Grupos | Total de grupos ativos (`COUNT(groups)`) | `/admin/groups` |
| Jogadores | Total de jogadores cadastrados (`COUNT(players)`) | `/players` |
| Horas | Total de horas jogadas na plataforma (partidas encerradas com `end_time`) | — (não clicável) |

**RF-04 — Big numbers de novos cadastros**
Abaixo dos cards de plataforma, exibir os contadores de cadastros (igual ao atual, sem a lista de registros recentes):

| Card | Métrica |
|---|---|
| Total | `signupStats.total` |
| Últimos 7 dias | `signupStats.last_7_days` |
| Últimos 30 dias | `signupStats.last_30_days` |

**RF-05 — Remover da home do admin**
Os seguintes elementos presentes no `/dashboard` atual **não devem aparecer** em `/admin`:
- Card "Meus Grupos" (lista de grupos do usuário)
- Lista "Registros recentes" (tabela com jogadores recentes)

**RF-06 — Listagem global de rachões (`/admin/matches`)**
Nova rota que exibe todos os rachões de toda a plataforma, com as colunas:
- Grupo | Data | Horário | Local | Status (`open` / `in_progress` / `closed`)
- Ordenação padrão: mais recentes primeiro
- Filtro por status (chips: Todos / Abertos / Em andamento / Encerrados)
- Cada linha é clicável e navega para `/match/[hash]`

**RF-07 — Listagem global de grupos (`/admin/groups`)**
Nova rota que exibe todos os grupos da plataforma, com as colunas:
- Nome | Descrição | Membros | Rachões | Data de criação
- Ordenação padrão: mais recentes primeiro
- Cada linha é clicável e navega para `/groups/[id]`

> **Nota:** `/players` (RF-03) já existe e atende o caso de uso — não requer nova rota.

**RF-08 — Formato de horas**
O card "Horas" exibe o total como `Xh Ymin` (ex: "312h 45min"). Se zero, exibe "—". O cálculo usa `SUM(end_time - start_time)` apenas em partidas com `status = 'closed'` e `end_time NOT NULL`.

---

## 4. Endpoint da API

### 4.1 `GET /api/v1/admin/stats`

Novo endpoint. Requer `players.role = 'admin'`. Retorna todas as métricas necessárias para a home do admin em uma única requisição.

**Response 200:**
```json
{
  "total_matches": 148,
  "total_groups": 12,
  "total_players": 203,
  "platform_minutes_played": 18750,
  "signups_total": 203,
  "signups_last_7_days": 8,
  "signups_last_30_days": 31
}
```

> **Nota:** O endpoint `/players/me/stats` atual já retorna `platform_minutes_played` e `platform_total_matches` para admins, e `/players/signups/stats` retorna os dados de cadastro. O novo endpoint `/admin/stats` unifica tudo — inclusive `total_groups` e `total_players` — em uma única chamada, eliminando múltiplas requisições paralelas do frontend.

### 4.2 `GET /api/v1/admin/matches`

Novo endpoint de listagem global de partidas. Requer `players.role = 'admin'`.

**Query params:**
- `status` (opcional): `open` | `in_progress` | `closed`
- `limit` (default: 50, max: 200)
- `offset` (default: 0)

**Response 200:**
```json
{
  "total": 148,
  "items": [
    {
      "id": "uuid",
      "hash": "abc123",
      "number": 9,
      "group_id": "uuid",
      "group_name": "Amigos GQC",
      "match_date": "2026-03-08",
      "start_time": "20:30:00",
      "end_time": "22:00:00",
      "location": "Arena GQC — Quadra 3",
      "status": "closed"
    }
  ]
}
```

### 4.3 `GET /api/v1/admin/groups`

Novo endpoint de listagem global de grupos. Requer `players.role = 'admin'`.

**Query params:**
- `limit` (default: 50, max: 200)
- `offset` (default: 0)

**Response 200:**
```json
{
  "total": 12,
  "items": [
    {
      "id": "uuid",
      "name": "Amigos GQC",
      "description": "Amigos de infância da Mooca",
      "slug": "amigos-gqc",
      "total_members": 8,
      "total_matches": 9,
      "created_at": "2026-01-15T10:00:00Z"
    }
  ]
}
```

---

## 5. Interface do Usuário

### 5.1 Layout da home `/admin`

```
┌─────────────────────────────────────────────────┐
│  Painel Admin                                    │
│  rachao.app                                      │
├─────────────────────────────────────────────────┤
│                                                  │
│  ┌──────────┐ ┌──────────┐ ┌──────┐ ┌───────┐  │
│  │ 148      │ │ 12       │ │ 203  │ │ 312h  │  │
│  │ Rachões→ │ │ Grupos → │ │ Jog→ │ │ Horas │  │
│  └──────────┘ └──────────┘ └──────┘ └───────┘  │
│                                                  │
│  Novos Cadastros                                 │
│  ┌──────────┐ ┌──────────┐ ┌──────────────────┐ │
│  │ 203      │ │ 8        │ │ 31               │ │
│  │ Total    │ │ 7 dias   │ │ 30 dias          │ │
│  └──────────┘ └──────────┘ └──────────────────┘ │
└─────────────────────────────────────────────────┘
```

- Grid 2×2 em mobile, 4 colunas em `sm:` para os cards de plataforma.
- Cards clicáveis têm seta `→` no label ou ícone `ChevronRight`.
- Card "Horas" não é clicável — sem cursor pointer / sem link.
- Seção "Novos Cadastros" em grid 3 colunas (igual ao atual).

### 5.2 Layout da listagem `/admin/matches`

```
┌─────────────────────────────────────────────────┐
│  ← Rachões (148)                                │
│                                                  │
│  [Todos] [Abertos] [Em andamento] [Encerrados]  │
│                                                  │
│  Grupo            Data       Status             │
│  ─────────────────────────────────────────────  │
│  Amigos GQC #9   08/03 20h  🟢 Encerrado  →    │
│  Pelada Mooca #3 07/03 19h  ⚪ Aberto     →    │
│  ...                                            │
└─────────────────────────────────────────────────┘
```

- Filtro de status como chips/tabs horizontais (padrão: "Todos").
- Em mobile: ocultar coluna "Horário" (`hidden sm:table-cell`).
- Cada linha clica em `/match/[hash]`.
- Paginação simples: botão "Carregar mais" (load more) se `total > items.length`.

### 5.3 Layout da listagem `/admin/groups`

```
┌─────────────────────────────────────────────────┐
│  ← Grupos (12)                                  │
│                                                  │
│  Nome              Membros  Rachões  Criado      │
│  ─────────────────────────────────────────────  │
│  Amigos GQC            8        9  Jan 2026  →  │
│  Pelada da Mooca       5        3  Fev 2026  →  │
│  ...                                            │
└─────────────────────────────────────────────────┘
```

- Em mobile: ocultar colunas "Rachões" e "Criado" (`hidden sm:table-cell`).
- Cada linha clica em `/groups/[id]`.

---

## 6. Arquivos a Criar/Modificar

| Arquivo | Ação | Descrição |
|---|---|---|
| `football-api/app/api/v1/routers/admin.py` | Criar | Endpoints `GET /admin/stats`, `GET /admin/matches`, `GET /admin/groups` |
| `football-api/app/api/v1/router.py` | Modificar | Registrar `admin.router` com prefix `/admin` |
| `football-api/app/schemas/admin.py` | Criar | Schemas `AdminStatsResponse`, `AdminMatchItem`, `AdminMatchListResponse`, `AdminGroupItem`, `AdminGroupListResponse` |
| `football-frontend/src/lib/api.ts` | Modificar | Adicionar `admin.getStats()`, `admin.getMatches(params)`, `admin.getGroups(params)` e tipos correspondentes |
| `football-frontend/src/routes/admin/+page.svelte` | Criar | Nova home do super admin com big numbers |
| `football-frontend/src/routes/admin/matches/+page.svelte` | Criar | Listagem global de rachões |
| `football-frontend/src/routes/admin/groups/+page.svelte` | Criar | Listagem global de grupos |
| `football-frontend/src/routes/dashboard/+page.svelte` | Modificar | Adicionar redirect para `/admin` quando `$isAdmin` |

---

## 7. Critérios de Aceitação

- [ ] Acesso a `/admin` por usuário não-admin redireciona para `/dashboard`
- [ ] Acesso a `/dashboard` por super admin redireciona para `/admin`
- [ ] Os 4 big numbers de plataforma são carregados em uma única requisição (`/admin/stats`)
- [ ] Cards de Rachões, Grupos e Jogadores são clicáveis e navegam para as listagens corretas
- [ ] Card de Horas não é clicável
- [ ] Horas exibidas no formato `Xh Ymin`; zero exibe "—"
- [ ] "Meus Grupos" não aparece em `/admin`
- [ ] "Registros recentes" não aparece em `/admin`
- [ ] Novos Cadastros exibe 3 contadores (Total, 7 dias, 30 dias) sem lista de jogadores
- [ ] `/admin/matches` lista todos os rachões da plataforma com filtro por status
- [ ] `/admin/groups` lista todos os grupos da plataforma
- [ ] Colunas secundárias ocultas em mobile nas listagens
- [ ] Loading skeleton nos cards enquanto os dados carregam

---

## 8. Fora de Escopo (desta versão)

- Gráficos de crescimento ao longo do tempo
- Filtros de data nas listagens
- Ações em massa (deletar grupos/partidas em lote)
- Edição de grupos ou partidas a partir das listagens admin

---

*Documento elaborado para uso interno da equipe de produto e engenharia do Rachao.app.*
