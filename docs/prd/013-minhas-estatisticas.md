# PRD — Minhas Estatísticas

**Produto:** rachao.app
**Feature:** Página "Minhas Estatísticas" para jogadores
**Status:** Implementado
**Data:** 2026-03-12

---

## Problema

Os jogadores não tinham visibilidade alguma sobre sua participação e desempenho na plataforma. Sem isso, o engajamento é passivo: o usuário entra só para confirmar presença, sem razão para explorar ou voltar com frequência.

---

## Objetivo

Criar uma página pessoal de estatísticas que mostre ao jogador sua trajetória, desempenho e reputação, gerando engajamento recorrente através de gamificação leve e dados visuais.

---

## Público-alvo

Jogadores e admins de grupo (`role = 'player'`). Apenas super admins (`role = 'admin'`) não têm acesso — o item de menu e o card de atalho são ocultados para eles.

---

## Rota

`/profile/stats`
Acessível via:
- Item **"Estatísticas"** no menu de navegação principal (após "Grupos"), visível em mobile e desktop
- Card de atalho em `/profile`
- Botão voltar (navbar) navega para `/profile`

---

## Blocos da página

### Bloco 1 — Cartão de identidade
Cabeçalho visual com gradiente azul. Exibe:
- Nome e apelido do jogador
- Data de entrada na plataforma ("Membro desde Mês Ano")
- Badge "Goleiro" se for goleiro em ao menos um grupo
- Contagem rápida de partidas confirmadas

### Bloco 2 — Métricas rápidas (grid 2×2)
Quatro chips com ícone + número em destaque:

| Chip | Métrica | Dado |
|---|---|---|
| Partidas | Total de partidas confirmadas | `total_matches_confirmed` |
| Em campo | Total de minutos jogados | `total_minutes_played` |
| Pontos | Pontos acumulados nas votações | `total_vote_points` |
| Decepções | Vezes votado como decepção | `total_flop_votes` |

> Nota de linguagem: o conceito de "flop" (pior da partida) é exibido sempre como **"Decepção"** na UI, mantendo o nome técnico interno apenas no código/banco.

### Bloco 3 — Presença
- **Donut chart** via `conic-gradient` CSS (sem biblioteca) mostrando o percentual de confirmações vs. recusas
- **Sequência atual**: partidas consecutivas confirmadas contadas do jogo mais recente
- **Maior sequência**: recorde histórico de consecutivas
- Barra de progresso linear complementar ao donut

### Bloco 4 — Histórico recente
Linha de pontos coloridos (verde = confirmado, vermelho = faltou) representando as últimas 20 partidas, da mais recente para a mais antiga. Cada ponto tem tooltip com data e nome do grupo.

### Bloco 5 — Reputação
Exibido apenas se o jogador tem ao menos 1 partida confirmada. Mostra:
- Vezes no **1º lugar** da votação (melhor da partida)
- Vezes em **qualquer Top 5**
- **Total de pontos** acumulados
- **Barra de aprovação**: `(top5_count / total_matches_confirmed) × 100%`, com gradiente azul→roxo

### Bloco 6 — Evolução mensal
Gráfico de barras CSS (sem biblioteca) com os últimos 6 meses. Barras proporcionais ao número de partidas confirmadas no mês. Meses sem jogos exibem barra vazia. Exibido apenas se há ao menos um mês com dados.

### Bloco 7 — Meus grupos
Lista de todos os grupos que o jogador participa, ordenados por número de partidas. Para cada grupo:
- Nome do grupo
- Número de partidas confirmadas
- Estrelas de habilidade (`skill_stars`)
- Badges "GK" (goleiro) e "Admin" quando aplicável

---

## Backend

### Novo endpoint

```
GET /players/me/stats/full
Authorization: Bearer <token>
Response: PlayerFullStats
```

Registrado antes de `/players/{player_id}` no router para evitar conflito de rota.

### Schema de resposta (`PlayerFullStats`)

```json
{
  "total_matches_confirmed": 42,
  "total_minutes_played": 3780,
  "total_vote_points": 127,
  "total_flop_votes": 3,
  "top1_count": 5,
  "top5_count": 18,
  "current_streak": 4,
  "best_streak": 9,
  "attendance_rate": 87,
  "monthly_stats": [
    { "month": "2025-10", "matches_confirmed": 3, "minutes_played": 270 },
    ...
  ],
  "recent_matches": [
    { "match_date": "2026-03-08", "group_name": "Rachão da Rua", "status": "confirmed" },
    ...
  ],
  "groups": [
    {
      "group_id": "uuid",
      "group_name": "Rachão da Rua",
      "skill_stars": 4,
      "is_goalkeeper": false,
      "role": "member",
      "matches_confirmed": 31
    },
    ...
  ]
}
```

### Repositório: `PlayerStatsRepository`

Arquivo: `football-api/app/db/repositories/player_stats_repo.py`

Executa 4 queries sequenciais em uma única chamada:
1. **Scalar aggregates** — CTE com totals, vote_pts, flop_cnt, att_rate (retorna 1 linha)
2. **History** — últimas partidas para cálculo de streak e exibição dos pontos
3. **Monthly** — últimos 6 meses com padding em Python para meses sem dados
4. **Groups** — participação por grupo com LEFT JOIN em attendances

### Cálculo de streaks

Feito em Python após consulta ordenada por `match_date DESC`:
- **current_streak**: conta confirmadas consecutivas desde o início da lista até a primeira recusa
- **best_streak**: maior sequência de confirmadas em qualquer ponto do histórico

---

## Regras de negócio

- **Apenas partidas `status = 'closed'`** contam para todas as métricas
- **Minutos jogados**: apenas partidas com `end_time` registrado; calculado como `(end_time - start_time)` em minutos
- **Taxa de presença**: `confirmed / (confirmed + declined)` — status `pending` excluído
- **Pontos de votação**: apenas de votações onde o jogador aparece no `match_vote_top5`
- **Decepções**: contagem de registros em `match_vote_flop` vinculados ao jogador
- **Séries mensais**: sempre 6 meses exibidos, com zeros para meses sem jogos

---

## Arquivos criados/modificados

| Arquivo | Ação |
|---|---|
| `football-api/app/schemas/player_stats.py` | Criado |
| `football-api/app/db/repositories/player_stats_repo.py` | Criado |
| `football-api/app/api/v1/routers/players.py` | Modificado — novo endpoint |
| `football-frontend/src/lib/api.ts` | Modificado — tipos + método `myFullStats()` |
| `football-frontend/src/routes/profile/stats/+page.svelte` | Criado |
| `football-frontend/src/routes/profile/+page.svelte` | Modificado — card de atalho |
| `football-frontend/src/lib/components/Navbar.svelte` | Modificado — back href + link no drawer |

---

## Métricas de engajamento esperadas

- **Aumento de sessões**: jogadores tendem a revisitar a plataforma fora do período pré-jogo para conferir suas estatísticas
- **Efeito streak**: a sequência de presenças consecutivas cria incentivo para não faltar
- **Retenção via reputação**: pontos e top 5 visíveis motivam participar das votações pós-jogo
