# PRD — Ranking Geral da Plataforma

## 1. Contexto

O rachao.app já possui votação pós-partida (Top 5 + Decepção) com pontuação definida:

| Posição | Pontos |
|---------|--------|
| 1º lugar | 10 pts |
| 2º lugar | 8 pts |
| 3º lugar | 6 pts |
| 4º lugar | 4 pts |
| 5º lugar | 2 pts |

Esses dados ficam nas tabelas `match_vote_top5` (pontos) e `match_vote_flop` (votos de decepção), vinculados a `match_votes` e `matches`. Hoje essa informação é exibida apenas por partida. Não existe visão consolidada histórica.

---

## 2. Objetivo

Criar uma página pública de **Ranking Geral** que consolide o desempenho histórico de todos os jogadores da plataforma, com filtros por período (mês / ano / all-time), servindo também como vitrine de engajamento para usuários não cadastrados.

---

## 3. Decisões definidas

| # | Decisão | Definição |
|---|---------|-----------|
| D1 | Grupos privados aparecem? | **Sim** — todos os jogadores da plataforma, sem distinção de grupo |
| D2 | Critério mínimo de qualidade | **Apenas partidas com ao menos 10 votantes** contam para o ranking |
| D3 | Tamanho do ranking | **Top 10 fixo**, sem paginação |
| D4 | Exibir grupo do jogador? | **Não** — ranking é da pessoa |
| D5 | Filtro por grupo? | **Não** — fora de escopo (fase 2) |
| D6 | Destacar usuário logado? | **Sim** — linha destacada se o usuário aparece no ranking |

---

## 4. Escopo da página

### Seção 1 — Melhores da Plataforma
- Ranking por somatória de pontos recebidos em votações
- Filtros de período: **Mês atual · Ano atual · Todos os tempos**
- Exibe: posição, avatar, nome (formato "PrimeiroNome (Apelido)"), pontos totais

### Seção 2 — Decepções da Plataforma
- Ranking por total de votos de decepção recebidos
- Mesmos filtros de período
- Exibe: posição, avatar, nome, total de votos de decepção

### Acesso
- Rota pública: `/ranking`
- Sem login obrigatório
- Logados: Navbar normal
- Não logados: `JoinCTABanner` fixo no rodapé (já existe)

---

## 5. Design da página

### Mobile (< 940px) — coluna única

```
┌──────────────────────────────────────────────┐
│  🏆 Ranking Geral                             │
│  Os melhores de toda a plataforma             │
│                                               │
│  ┌──────────────────────────────────────────┐ │
│  │  [Melhores da partida] [Decepções]       │ │
│  └──────────────────────────────────────────┘ │
│                                               │
│  ┌──────────────────────────────────────────┐ │
│  │  [Mês]  [Ano]  [Todos os tempos]         │ │
│  └──────────────────────────────────────────┘ │
│                                               │
│  ── Pódio ──────────────────────────────────  │
│                                               │
│    [Avatar]      [Avatar]     [Avatar]        │
│     Lucas        Thiago        Dudu           │
│      🥈           🥇            🥉             │
│    280 pts      312 pts      241 pts          │
│   ┌──────┐   ┌──────────┐  ┌────┐            │
│   │  2º  │   │    1º    │  │ 3º │            │
│   └──────┘   └──────────┘  └────┘            │
│                                               │
│  ── 4º ao 10º ──────────────────────────────  │
│                                               │
│  4º  [av] Animal          198 pts             │
│  5º  [av] Bruxo           175 pts             │
│  6º  [av] São Marcos      160 pts  ← logado  │
│  ...                                          │
│                                               │
│  ┌──────────────────────────────────────────┐ │
│  │  🏆 Gerencie seu rachão [Entrar][Cadastrar│ │
│  └──────────────────────────────────────────┘ │
└──────────────────────────────────────────────┘
```

### Desktop (≥ 940px) — duas colunas lado a lado

```
┌─────────────────────────────────────────────────────────────────┐
│  🏆 Ranking Geral                                                │
│  Os melhores de toda a plataforma        [Mês] [Ano] [All-time] │
│                                                                  │
│  ┌──────────────────────────┐  ┌──────────────────────────────┐ │
│  │  🏆 Melhores da Partida  │  │  😬 Decepções                │ │
│  │                          │  │                              │ │
│  │  [Pódio com avatares]    │  │  [Lista com avatares]        │ │
│  │                          │  │                              │ │
│  │  4º  [av] Animal  198pts │  │  1º  [av] R9        14 votos │ │
│  │  5º  [av] Bruxo   175pts │  │  2º  [av] Menino N  11 votos │ │
│  │  ...                     │  │  ...                         │ │
│  └──────────────────────────┘  └──────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

### Detalhes de componentes

**Avatar no pódio (top 3)**
- `AvatarImage` size=48, com anel colorido por posição (ouro/prata/bronze)
- Centralizado acima do bloco do pódio

**Avatar na lista (4º–10º e flop)**
- `AvatarImage` size=32, alinhado à esquerda junto ao nome

**Linha do usuário logado (D6)**
- Fundo levemente destacado: `bg-primary-50 dark:bg-primary-900/20`
- Badge "Você" discreta ao lado do nome

**Estado vazio**
- "Nenhuma votação registrada para este período."
- Ícone de troféu em cinza

**Skeleton/shimmer** durante carregamento — mesmo padrão das outras páginas.

---

## 6. Backend

### Novo endpoint (rota pública)

```
GET /api/v1/ranking?period=month|year|all&type=top|flop
```

Sem autenticação obrigatória.

### Filtro D2 — qualidade mínima

Apenas votos de partidas onde `eligible_voters >= 10`:

```sql
JOIN matches m ON m.id = mv.match_id
WHERE m.eligible_voters >= 10
```

> `eligible_voters` já é gravado em `matches` no momento do encerramento.
> **Verificar se a coluna existe antes de implementar** — pode ser necessário usar `COUNT(DISTINCT attendance)` como subquery se a coluna não estiver persistida.

### Filtro de período

```sql
-- month
AND mv.submitted_at >= date_trunc('month', now())

-- year
AND mv.submitted_at >= date_trunc('year', now())

-- all: sem filtro
```

### Response — Top

```json
{
  "period": "month",
  "type": "top",
  "items": [
    {
      "position": 1,
      "player_id": "uuid",
      "name": "Thiago Nunes",
      "nickname": "Thiagol",
      "avatar_url": "https://...",
      "total_points": 312
    }
  ]
}
```

### Response — Flop

```json
{
  "period": "month",
  "type": "flop",
  "items": [
    {
      "position": 1,
      "player_id": "uuid",
      "name": "Roberto Carlos",
      "nickname": null,
      "avatar_url": null,
      "total_flop_votes": 14
    }
  ]
}
```

### Empates
Mesmo critério já implementado no `vote_repo`: jogadores com mesma pontuação/votos recebem a mesma posição.

### Localização dos arquivos novos

| Camada | Arquivo |
|--------|---------|
| Router | `app/api/v1/routers/ranking.py` |
| Repository | `app/db/repositories/ranking_repo.py` |
| Schema | `app/schemas/ranking.py` |

---

## 7. Frontend

| Item | Valor |
|------|-------|
| Rota | `src/routes/ranking/+page.svelte` |
| Rota pública | Adicionar `/ranking` em `PUBLIC_ROUTES` no `+layout.svelte` |
| Navbar | Exibida para logados — já coberto pela remoção de `/match/` da exclusão de `isAppPage` (não é `/match/`, então funciona normalmente) |
| JoinCTABanner | Exibida para não logados |
| Back nav | Sem botão voltar — é rota de nível 1 no menu |

### Item de menu na Navbar

```typescript
{ href: '/ranking', icon: Award, labelKey: 'nav.ranking' }
```

- Visível para todos (logados e não logados — mas Navbar só aparece para logados)
- Posição na lista: após "Grupos", antes de "Descobrir"
- Ícone sugerido: `Award` (Lucide) — troféu estilizado

### Layout da página

- Mobile: `max-w-lg mx-auto` — abas empilhadas (Melhores / Decepções), filtro de período acima
- Desktop: `max-w-5xl mx-auto` — grid `lg:grid-cols-2 gap-6`, ambas as seções visíveis lado a lado, filtro de período no cabeçalho compartilhado

---

## 8. i18n — chaves necessárias

```json
"ranking.title": "Ranking Geral",
"ranking.subtitle": "Os melhores de toda a plataforma",
"ranking.tab_top": "Melhores",
"ranking.tab_flop": "Decepções",
"ranking.period_month": "Mês",
"ranking.period_year": "Ano",
"ranking.period_all": "Todos os tempos",
"ranking.points_suffix": "pts",
"ranking.votes_suffix": "votos",
"ranking.you_badge": "Você",
"ranking.empty": "Nenhuma votação registrada para este período.",
"nav.ranking": "Ranking"
```

---

## 9. Impacto técnico

| Camada | Trabalho |
|--------|---------|
| Migration | **Nenhuma** — dados já existem |
| Backend | 1 router, 1 repo, 1 schema |
| Frontend | 1 rota, atualização da Navbar e `PUBLIC_ROUTES` |
| Testes | Unitários do endpoint (top/flop × 3 períodos + empate + filtro D2) |

---

## 10. Fora de escopo (fase 2)

- Filtro por grupo específico
- Compartilhamento do ranking no WhatsApp
- Ranking por habilidade/skill stars
- Histórico de evolução de posição (gráfico de linha)
- Notificação "você subiu para o Top 10"
