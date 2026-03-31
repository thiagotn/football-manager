# PRD — Votação Pós-Partida
## Rachao.app · Gerenciamento de Grupos e Partidas

| | |
|---|---|
| **Versão** | 1.5 |
| **Status** | ✅ Implementado — Março 2026 |
| **Data** | Março de 2026 |
| **Plataforma** | https://rachao.app |

---

## 1. Visão Geral

### 1.1 Contexto

Após cada partida encerrada, os participantes poderão votar nos melhores jogadores e na decepção do jogo. A votação é aberta automaticamente 20 minutos após o fim da partida e fica disponível por 24 horas.

### 1.2 Objetivo

Aumentar o engajamento dos jogadores no período pós-partida, criando um ritual de reconhecimento dentro do grupo. O resultado da votação é exibido na página da partida ao final do período, gerando histórico e competição saudável entre os membros.

---

## 2. Regras de Negócio

### 2.1 Janela de votação

- **Abertura:** 20 minutos após o horário de término da partida (`end_time`). Se a partida não tiver `end_time` definido, usar `match_date` às 23h59 como referência de término.
- **Encerramento:** 24 horas após a abertura.
- Fora dessa janela, o endpoint de submissão retorna erro `403 VOTING_CLOSED`.

### 2.2 Elegibilidade para votar

- Apenas jogadores com `attendance.status = 'confirmed'` na partida podem votar.
- Cada jogador pode submeter **um único voto** por partida (imutável após envio).
- O jogador **não pode votar em si mesmo** em nenhuma categoria.

### 2.3 Categorias de voto

**Top 5 — Melhores da partida**

O votante deve escolher jogadores em ordem de preferência. O número de posições exibidas e obrigatórias é determinado pela quantidade de elegíveis: se há N elegíveis (excluindo o próprio votante), exibir e exigir `min(N, 5)` posições.

| Posição | Pontos |
|:---:|:---:|
| 1º lugar | 10 |
| 2º lugar | 8 |
| 3º lugar | 6 |
| 4º lugar | 4 |
| 5º lugar | 2 |

O resultado final é a soma dos pontos recebidos por cada jogador. O pódio exibe os 5 com maior pontuação total.

**Decepção do jogo**

O votante escolhe 1 jogador como decepção da partida. O resultado é uma contagem simples de votos — o jogador com mais votos é declarado a decepção da partida. Em caso de empate, todos os empatados são exibidos.

A decepção é **obrigatória** quando há jogadores elegíveis suficientes para preenchê-la, ou seja, quando `elegíveis > 5`. Se `elegíveis ≤ 5`, o campo de decepção não é exibido.

### 2.4 Exibição dos resultados

- Durante a janela de votação: resultados **ocultos** (evita influência nos votos restantes).
- Após o encerramento da janela: resultados **públicos** para todos os membros do grupo.
- A quantidade de participantes que já votaram é exibida durante a votação (ex: "7 de 12 jogadores votaram"), sem revelar os votos individuais.

### 2.5 Indicador de voto na listagem de confirmados

Durante e após a janela de votação, a listagem de jogadores confirmados na partida exibe, ao lado de cada jogador, se ele já votou ou não. Visível para admins de grupo e para todos os jogadores confirmados. Os votos individuais permanecem ocultos — apenas o status "votou / não votou" é exposto.

### 2.6 Aviso de votação em aberto na home

O dashboard (home logada) exibe um aviso para o usuário sempre que ele tiver votações pendentes em partidas das quais participou. O aviso é exibido imediatamente abaixo dos cards de big numbers (próximos, grupos, horas jogadas) e é visível para todos os usuários que não são super-admin.

### 2.7 Período nas estatísticas de grupo

A aba de Estatísticas exibe, por padrão, o ranking **anual** (ano corrente). O usuário pode alternar para a visão **mensal**, selecionando o mês desejado dentro do ano corrente. A troca de período é feita via seletor na própria aba, sem recarregar a página.

---

## 3. Requisitos Funcionais

**RF-01 — Abertura automática da votação**
O sistema deve abrir a votação automaticamente 20 minutos após o `end_time` da partida. Implementação **lazy** (calculado no momento da requisição via `voting_status(match)`), sem job agendado para o cálculo do status.

**RF-02 — Encerramento automático da votação**
A votação encerra automaticamente 24 horas após a abertura. Após esse prazo, nenhum novo voto é aceito.

**RF-03 — Submissão de voto**
O jogador elegível pode submeter um voto contendo:
- `min(elegíveis, 5)` jogadores para o Top 5, todos obrigatórios
- 1 jogador para decepção, obrigatório quando `elegíveis > 5`; não exibido quando `elegíveis ≤ 5`

A submissão é atômica — parcial ou com erro não deve persistir nada.

**RF-04 — Imutabilidade do voto**
Uma vez submetido, o voto não pode ser alterado. O endpoint deve retornar `409 ALREADY_VOTED` se o jogador tentar votar novamente.

**RF-05 — Exibição do status da votação**
Na página da partida (`/match/[hash]`), exibir:
- Se a votação ainda não abriu: "Votação abre em X horas"
- Se a votação está aberta: formulário de voto + contador "X de Y jogadores votaram"
- Se o jogador já votou: confirmação do voto enviado + contador
- Se a votação encerrou: resultados completos (Top 5 + Decepção)

**RF-06 — Resultados**
Após o encerramento, exibir na página da partida:
- Pódio do Top 5 com nome e pontuação total
- Decepção do jogo com nome e quantidade de votos recebidos
- Em caso de empate no Top 5: todos os empatados são exibidos na mesma posição
- Em caso de empate na Decepção: exibir todos os empatados

**RF-07 — Notificação de abertura**
Ao acessar o status da votação pela primeira vez com `status = 'open'` (abordagem lazy), enviar push notification para todos os participantes confirmados da partida. Para evitar notificações duplicadas, usar a flag `vote_notified` na tabela `matches`.

> **Nota de implementação:** `send_push` em `app/services/push.py` recebe `(db, player_id, title, body, url)` e opera por player. O disparo deve iterar sobre os `attendances` confirmados e chamar `send_push` para cada um — conforme o padrão já utilizado em `recurrence.py`.

**RF-08 — Indicador de voto na listagem de confirmados**
Na listagem de jogadores confirmados da partida (página `/match/[hash]`), cada linha deve indicar visualmente se o jogador já votou ou não, quando `voting_status = 'open'` ou `'closed'`. Visível para todos os jogadores confirmados e admins de grupo. Os votos individuais nunca são expostos.

O endpoint `GET /matches/{match_id}/votes/status` retorna `voted_player_ids` (lista de `player_id`s que já votaram) para uso pelo frontend na montagem do indicador.

**RF-09 — Aba de Estatísticas do grupo**
Na página de detalhes do grupo (`/groups/[id]`), adicionar uma aba **"Estatísticas"** com ranking acumulado dos membros do grupo. A tabela contém:

| Coluna | Fonte | Descrição |
|---|---|---|
| Posição | calculada | Ranking por pontos de votação (medalha para top 3) |
| Jogador | `players.nickname \|\| name` | Nome do membro |
| Pts Votação | `SUM(match_vote_top5.points)` | Total de pontos recebidos nas votações **encerradas** do período selecionado |
| Decepções | `COUNT(match_vote_flop)` | Total de vezes eleito decepção em votações **encerradas** do período selecionado |
| Horas jogadas | `SUM(end_time - start_time)` | Total de horas em partidas confirmadas com `end_time` definido no período selecionado |

Regras:
- Pontos e decepções só são computados de partidas com `status = 'closed'` **e** `voting_status = 'closed'`.
- Horas jogadas consideram partidas com `status = 'closed'` e `end_time` definido, independentemente do status da votação.
- Apenas membros com `players.role != 'admin'` são listados.
- Ordenação padrão: pontos de votação decrescente; em caso de empate, alfabética por nome.
- A aba é exibida sempre que o grupo tiver ao menos 1 partida elegível no período; caso contrário, mostrar mensagem vazia.
- Visível para todos os membros logados (inclusive admins de grupo).

**RF-10 — Seletor de período nas Estatísticas**
A aba Estatísticas exibe por padrão o **ano corrente**. O usuário pode alternar para a visão **mensal** via seletor, escolhendo qualquer mês do ano corrente. A seleção dispara nova chamada ao endpoint `GET /groups/{group_id}/stats` com os parâmetros de período correspondentes.

**RF-11 — Aviso de votação em aberto na home**
No dashboard (`/`), exibir um banner de aviso imediatamente abaixo dos cards de big numbers para usuários que possuem votações pendentes (partidas confirmadas com `voting_status = 'open'` e `current_player_voted = false`). O banner é visível apenas para usuários que não são super-admin.

Cada votação pendente deve aparecer como um item clicável no banner, com nome/data da partida e link direto para `/match/[hash]`. Se houver mais de uma votação pendente, listar todas. O banner desaparece automaticamente quando não há mais votações pendentes.

O endpoint existente `GET /matches/{match_id}/votes/status` já provê as informações necessárias; um novo endpoint de listagem pode ser necessário para consolidar as votações pendentes do player logado.

---

## 4. Requisitos Não Funcionais

**RNF-01** — A elegibilidade para votar deve ser validada no backend, nunca apenas no frontend.
**RNF-02** — A janela de votação deve ser calculada com base no fuso horário de Brasília (BRT, UTC-3), usando `zoneinfo.ZoneInfo("America/Sao_Paulo")` (stdlib Python 3.9+, sem dependência de `pytz`).
**RNF-03** — Os resultados parciais (em quem cada um votou) não devem ser expostos por nenhum endpoint enquanto a votação estiver aberta.
**RNF-04** — A submissão de voto deve ser idempotente por `(match_id, voter_id)` — tentativas duplicadas retornam `409` sem efeito colateral.
**RNF-05** — O indicador de "quem votou" expõe apenas o `player_id` dos votantes, nunca os votos individuais.
**RNF-06** — O endpoint de votações pendentes deve ser eficiente: retornar apenas partidas dentro da janela de votação ativa do player logado, sem varrer todo o histórico.

---

## 5. Modelagem de Dados

```sql
-- Migration: 017_match_votes.sql

CREATE TABLE match_votes (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    match_id     UUID NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    voter_id     UUID NOT NULL REFERENCES players(id),
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (match_id, voter_id)
);

CREATE TABLE match_vote_top5 (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vote_id    UUID NOT NULL REFERENCES match_votes(id) ON DELETE CASCADE,
    player_id  UUID NOT NULL REFERENCES players(id),
    position   SMALLINT NOT NULL CHECK (position IN (1, 2, 3, 4, 5)),
    points     SMALLINT NOT NULL CHECK (points IN (2, 4, 6, 8, 10)),
    UNIQUE (vote_id, position),
    UNIQUE (vote_id, player_id)
);

CREATE TABLE match_vote_flop (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vote_id    UUID NOT NULL REFERENCES match_votes(id) ON DELETE CASCADE,
    player_id  UUID NOT NULL REFERENCES players(id),
    UNIQUE (vote_id)
);

ALTER TABLE matches ADD COLUMN vote_notified BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX idx_match_votes_match    ON match_votes (match_id);
CREATE INDEX idx_match_vote_top5_vote ON match_vote_top5 (vote_id);
CREATE INDEX idx_match_vote_flop_vote ON match_vote_flop (vote_id);
```

### 5.1 Pontuação por posição

| Posição | Pontos |
|:---:|:---:|
| 1º | 10 |
| 2º | 8  |
| 3º | 6  |
| 4º | 4  |
| 5º | 2  |

O campo `points` é calculado no backend pelo mapeamento `{1: 10, 2: 8, 3: 6, 4: 4, 5: 2}` — nunca enviado pelo cliente.

### 5.2 Consulta de Estatísticas por Grupo

A query recebe dois parâmetros adicionais de período: `period_start` e `period_end` (datas em BRT), derivados do seletor de período no frontend (ano ou mês corrente).

```sql
WITH eligible_matches AS (
    SELECT id
    FROM matches
    WHERE group_id = :group_id
      AND status   = 'closed'
      AND match_date BETWEEN :period_start AND :period_end
      AND (
          (end_time IS NOT NULL
           AND (match_date + end_time + INTERVAL '24 hours 20 minutes') < NOW() AT TIME ZONE 'America/Sao_Paulo')
          OR
          (end_time IS NULL
           AND (match_date + TIME '23:59' + INTERVAL '24 hours 20 minutes') < NOW() AT TIME ZONE 'America/Sao_Paulo')
      )
),
vote_points AS (
    SELECT mvt.player_id, SUM(mvt.points) AS total_points
    FROM match_vote_top5 mvt
    JOIN match_votes mv ON mv.id = mvt.vote_id
    WHERE mv.match_id IN (SELECT id FROM eligible_matches)
    GROUP BY mvt.player_id
),
flop_counts AS (
    SELECT mvf.player_id, COUNT(*) AS total_flops
    FROM match_vote_flop mvf
    JOIN match_votes mv ON mv.id = mvf.vote_id
    WHERE mv.match_id IN (SELECT id FROM eligible_matches)
    GROUP BY mvf.player_id
),
minutes_played AS (
    SELECT a.player_id,
           SUM(EXTRACT(EPOCH FROM (m.end_time::time - m.start_time::time)) / 60) AS total_minutes
    FROM attendances a
    JOIN matches m ON m.id = a.match_id
    WHERE m.group_id  = :group_id
      AND m.status    = 'closed'
      AND a.status    = 'confirmed'
      AND m.end_time  IS NOT NULL
      AND m.match_date BETWEEN :period_start AND :period_end
    GROUP BY a.player_id
)
SELECT
    p.id                             AS player_id,
    COALESCE(p.nickname, p.name)     AS display_name,
    COALESCE(vp.total_points, 0)     AS vote_points,
    COALESCE(fc.total_flops, 0)      AS flop_votes,
    COALESCE(mp.total_minutes, 0)    AS minutes_played
FROM group_members gm
JOIN players p            ON p.id = gm.player_id
LEFT JOIN vote_points  vp ON vp.player_id = p.id
LEFT JOIN flop_counts  fc ON fc.player_id = p.id
LEFT JOIN minutes_played mp ON mp.player_id = p.id
WHERE gm.group_id = :group_id
  AND p.role      != 'admin'
ORDER BY vote_points DESC, display_name ASC;
```

### 5.3 Lógica da janela de votação

```python
from datetime import datetime, timedelta, time as dt_time
from zoneinfo import ZoneInfo

BRT = ZoneInfo("America/Sao_Paulo")

def voting_window(match) -> tuple[datetime, datetime]:
    end_t = match.end_time if match.end_time else dt_time(23, 59)
    end_dt = datetime.combine(match.match_date, end_t).replace(tzinfo=BRT)
    opens_at  = end_dt + timedelta(minutes=20)
    closes_at = opens_at + timedelta(hours=24)
    return opens_at, closes_at

def voting_status(match) -> str:
    """'not_open' | 'open' | 'closed'"""
    now = datetime.now(BRT)
    opens_at, closes_at = voting_window(match)
    if now < opens_at:
        return "not_open"
    if now <= closes_at:
        return "open"
    return "closed"
```

---

## 6. Endpoints da API

| Método | Endpoint | Descrição |
|---|---|---|
| `GET` | `/api/v1/matches/{match_id}/votes/status` | Status da votação, contagem e lista de quem votou |
| `POST` | `/api/v1/matches/{match_id}/votes` | Submete o voto do jogador autenticado |
| `GET` | `/api/v1/matches/{match_id}/votes/results` | Resultados (apenas após encerramento) |
| `GET` | `/api/v1/votes/pending` | Votações pendentes do player logado (para o banner da home) |
| `GET` | `/api/v1/groups/{group_id}/stats` | Estatísticas acumuladas dos jogadores do grupo por período |

### 6.1 `GET /matches/{match_id}/votes/status`

Disponível quando a votação está aberta ou encerrada. Retorna o status da janela, se o usuário já votou, a contagem anônima e a lista de `player_id`s que já votaram. **Também dispara a notificação push se `status = 'open'` e `vote_notified = false`.**

**Response 200:**
```json
{
  "status": "open",
  "opens_at": "2026-03-08T22:20:00-03:00",
  "closes_at": "2026-03-09T22:20:00-03:00",
  "voter_count": 7,
  "eligible_count": 12,
  "current_player_voted": false,
  "voted_player_ids": ["uuid-joao", "uuid-pedro", "uuid-carlos"]
}
```

### 6.2 `POST /matches/{match_id}/votes`

**Request:**
```json
{
  "top5": [
    { "player_id": "uuid-joao",   "position": 1 },
    { "player_id": "uuid-pedro",  "position": 2 },
    { "player_id": "uuid-carlos", "position": 3 },
    { "player_id": "uuid-marcos", "position": 4 },
    { "player_id": "uuid-rafael", "position": 5 }
  ],
  "flop_player_id": "uuid-lucas"
}
```

**Response 201:**
```json
{ "message": "Voto registrado com sucesso." }
```

**Erros:**
- `403 VOTING_CLOSED` — fora da janela de votação
- `403 NOT_ELIGIBLE` — jogador não confirmou presença na partida
- `409 ALREADY_VOTED` — voto já registrado
- `422 SELF_VOTE` — tentativa de votar em si mesmo
- `422` — posições duplicadas ou jogador repetido no top5

### 6.3 `GET /matches/{match_id}/votes/results`

Disponível apenas quando `voting_status == 'closed'`. Retorna `403 RESULTS_NOT_AVAILABLE` caso contrário.

**Response 200:**
```json
{
  "top5": [
    { "position": 1, "player_id": "uuid-joao",   "name": "João",   "points": 42 },
    { "position": 2, "player_id": "uuid-pedro",  "name": "Pedro",  "points": 35 },
    { "position": 3, "player_id": "uuid-carlos", "name": "Carlos", "points": 28 },
    { "position": 4, "player_id": "uuid-marcos", "name": "Marcos", "points": 18 },
    { "position": 5, "player_id": "uuid-rafael", "name": "Rafael", "points": 10 }
  ],
  "flop": [
    { "player_id": "uuid-lucas", "name": "Lucas", "votes": 5 }
  ],
  "total_voters": 9,
  "eligible_voters": 12
}
```

### 6.4 `GET /votes/pending`

Retorna as votações pendentes do player logado — partidas confirmadas com `voting_status = 'open'` e sem voto registrado. Usado exclusivamente pelo banner da home.

**Response 200:**
```json
{
  "pending": [
    {
      "match_id": "uuid-match",
      "match_hash": "abc123",
      "group_name": "Pelada da Quinta",
      "match_date": "2026-03-08",
      "closes_at": "2026-03-09T22:20:00-03:00",
      "voter_count": 7,
      "eligible_count": 12
    }
  ]
}
```

> Retorna lista vazia (`"pending": []`) quando não há votações pendentes. O frontend omite o banner nesse caso.

### 6.5 `GET /groups/{group_id}/stats`

Requer autenticação. Disponível para qualquer membro do grupo. Aceita query params `period=annual` (padrão) ou `period=monthly&month=YYYY-MM`.

**Exemplos:**
- `GET /groups/{id}/stats` → ano corrente
- `GET /groups/{id}/stats?period=annual` → ano corrente
- `GET /groups/{id}/stats?period=monthly&month=2026-03` → março de 2026

**Response 200:**
```json
{
  "period": "annual",
  "period_label": "2026",
  "players": [
    {
      "player_id": "uuid-joao",
      "display_name": "Joãozinho",
      "vote_points": 84,
      "flop_votes": 2,
      "minutes_played": 270
    }
  ]
}
```

> `minutes_played` é inteiro (total de minutos). O frontend converte para `Xh Ymin`.
> `period_label` é exibido no cabeçalho da aba (ex: "2026" ou "Março 2026").

---

## 7. Fluxo Principal

```
Partida encerrada (status = 'closed')
    ↓
20 minutos após end_time
    ↓
voting_status(match) = 'open'  ← calculado lazily na requisição
    ↓
Primeiro GET /votes/status com status='open' e vote_notified=false:
    Backend envia send_push para cada participante confirmado
    Atualiza matches.vote_notified = true
    ↓
Player acessa a home (/)
Frontend chama GET /votes/pending
    → se há pendências: exibe banner abaixo dos big numbers
    → cada item é link para /match/[hash]
    ↓
Player acessa /match/[hash]
Frontend chama GET /votes/status
    → exibe formulário de voto
    → exibe indicador "votou / não votou" na listagem de confirmados
    ↓
Player submete voto → POST /votes
Backend valida elegibilidade, janela, autovotos e duplicatas
Persiste match_votes + match_vote_top5 + match_vote_flop (transação)
    ↓
Banner da home desaparece (próximo acesso ao dashboard)
    ↓
24 horas após abertura
    ↓
voting_status = 'closed'
GET /votes/results passa a responder
Página da partida exibe pódio e decepção
Aba Estatísticas do grupo passa a computar pontos desta partida
```

---

## 8. Interface do Usuário

### 8.1 Banner de votações pendentes na home (`/`)

Exibido abaixo dos cards de big numbers, apenas para não super-admins, quando `GET /votes/pending` retorna ao menos um item:

```
┌──────────────────────────────────────────────────┐
│  🗳️  Você tem votação em aberto!                 │
│──────────────────────────────────────────────────│
│  Pelada da Quinta · 08/03              →  votar  │
│  Rachinha do Sábado · 07/03            →  votar  │
└──────────────────────────────────────────────────┘
```

- Cada linha exibe nome do grupo + data da partida e é um link para `/match/[hash]`.
- Se houver apenas uma pendência, pode ser um banner simples em vez de lista.
- O banner não é exibido se `pending` estiver vazio.
- Não é exibido para super-admins.

### 8.2 Seção de votação na página da partida (`/match/[hash]`)

**Votação não aberta**
```
┌──────────────────────────────────────┐
│  🗳️  Votação                         │
│  Abre em 0h 45min                    │
└──────────────────────────────────────┘
```

**Votação aberta — jogador ainda não votou**
```
┌──────────────────────────────────────┐
│  🗳️  Votação aberta · 7 de 12 votos  │
│  Fecha em 18h 20min                  │
│──────────────────────────────────────│
│  TOP 5 — Melhores da partida         │
│  🥇 1º  (10 pts) [ Selecionar ▾]    │
│  🥈 2º  ( 8 pts) [ Selecionar ▾]    │
│  🥉 3º  ( 6 pts) [ Selecionar ▾]    │
│     4º  ( 4 pts) [ Selecionar ▾]    │
│     5º  ( 2 pts) [ Selecionar ▾]    │
│                                      │
│  DECEPÇÃO DO JOGO                    │
│  [ Selecionar jogador ▾]             │
│                                      │
│  [ Enviar voto ]                     │
└──────────────────────────────────────┘
```

**Votação aberta — jogador já votou**
```
┌──────────────────────────────────────┐
│  ✅ Você já votou · 9 de 12 votos    │
│  Resultados disponíveis em 14h       │
└──────────────────────────────────────┘
```

**Votação encerrada — resultados**
```
┌──────────────────────────────────────┐
│  🏆 Melhores da Partida              │
│  🥇 João Silva          42 pts       │
│  🥈 Pedro Alves         35 pts       │
│  🥉 Carlos Mendes       28 pts       │
│     Marcos Lima         18 pts       │
│     Rafael Costa        10 pts       │
│                                      │
│  😬 Decepção do Jogo                 │
│  Lucas Ferreira         5 votos      │
│──────────────────────────────────────│
│  9 de 12 jogadores votaram           │
└──────────────────────────────────────┘
```

### 8.3 Indicador de voto na listagem de confirmados

Exibido apenas quando `voting_status = 'open'` ou `'closed'`:

```
Confirmados (12)
┌─────────────────────────────────────┐
│  João Silva        ✅ votou          │
│  Pedro Alves       ✅ votou          │
│  Carlos Mendes     ⏳ não votou      │
│  Marcos Lima       ✅ votou          │
│  Lucas Ferreira    ⏳ não votou      │
└─────────────────────────────────────┘
```

- Não aparece antes da votação abrir.
- Permanece visível após o encerramento.
- Votos individuais nunca são expostos.

### 8.4 Regras de UX do formulário de voto

- Excluir automaticamente o próprio jogador logado de todas as listas.
- Jogador selecionado em qualquer posição do Top 5 é removido das demais posições e da Decepção.
- Jogador selecionado como Decepção é removido das opções do Top 5.
- Todas as posições visíveis (`min(elegíveis, 5)`) são obrigatórias.
- Decepção exibida e obrigatória apenas quando `elegíveis > 5`.
- Botão "Enviar voto" desabilitado até todas as seleções obrigatórias estarem preenchidas.

### 8.5 Aba de Estatísticas na página do grupo (`/groups/[id]`)

**Seletor de período** no topo da aba:

```
[ Anual ▾ ]  ou  [ Mensal: Março 2026 ▾ ]
```

- Padrão: **Anual** (ano corrente).
- Opção mensal: dropdown com os meses do ano corrente (apenas meses passados ou corrente).
- Troca de período dispara nova chamada a `GET /groups/{id}/stats?period=...` sem recarregar a página.

**Tabela de ranking:**

```
┌─────────────────────────────────────────────────────────────┐
│  Estatísticas · 2026                    [ Anual ▾ ]         │
│─────────────────────────────────────────────────────────────│
│  #   Jogador          Pts Votação   Decepções   Horas        │
│─────────────────────────────────────────────────────────────│
│  🥇  Joãozinho              84          2        4h 30min    │
│  🥈  Pedro                  60          0        5h 30min    │
│  🥉  Carlos                 48          1        3h 00min    │
│   4  Marcos                 22          3        2h 00min    │
│   5  Rafael                  0          0           —        │
└─────────────────────────────────────────────────────────────┘
```

- Posições 1–3 exibem medalha (🥇🥈🥉); demais exibem número.
- "Decepções" em destaque vermelho quando > 0.
- "Horas": exibe `Xh Ymin`; se zero, exibe "—".
- Em mobile: ocultar colunas "Decepções" e "Horas" (`hidden sm:table-cell`).
- Estado vazio: "Nenhuma estatística disponível para este período. As estatísticas aparecem após o encerramento das votações."

### 8.6 Banner de votação na página do grupo

Na página de detalhes do grupo (`/groups/[id]`), exibir banner entre os detalhes gerais e as abas para cada partida encerrada com votação aberta e não votada pelo jogador logado. Link para a página da partida. Visível apenas para jogadores confirmados.

---

## 9. Decisões de Arquitetura

- **Lazy vs. job agendado:** status da votação calculado lazily via `voting_status(match)`.
- **Estatísticas apenas com votação encerrada:** pontos e decepções só entram no ranking após `voting_status = 'closed'`, evitando que o ranking flutue durante uma votação em andamento. Horas jogadas são independentes.
- **Período nas estatísticas:** o frontend envia `period_start` / `period_end` derivados do seletor; o backend aplica o filtro de `match_date` diretamente na CTE.
- **Votações pendentes na home:** endpoint dedicado `GET /votes/pending` para evitar lógica complexa no frontend ao carregar o dashboard.
- **Push notification:** disparada no primeiro `GET /votes/status` com `status='open'`, controlada pela flag `vote_notified`.
- **Router registration:** em `football-api/app/api/v1/router.py`, não em `main.py`.
- **`zoneinfo` em vez de `pytz`:** stdlib Python 3.9+, sem dependência adicional.

---

## 10. Arquivos a Criar/Modificar

| Arquivo | Ação | Descrição |
|---|---|---|
| `football-api/migrations/017_match_votes.sql` | Criar | Tabelas `match_votes`, `match_vote_top5`, `match_vote_flop`; coluna `vote_notified` em `matches` |
| `football-api/app/models/match_vote.py` | Criar | Models SQLAlchemy das 3 tabelas |
| `football-api/app/models/match.py` | Modificar | Adicionar campo `vote_notified: bool` |
| `football-api/app/schemas/vote.py` | Criar | `VoteSubmitRequest`, `VoteStatusResponse` (inclui `voted_player_ids`), `VoteResultsResponse`, `PendingVoteItem`, `PendingVotesResponse` |
| `football-api/app/db/repositories/vote_repo.py` | Criar | Submissão atômica, resultados, lista de votantes, listagem de pendências |
| `football-api/app/services/voting.py` | Criar | `voting_window`, `voting_status`, `POINTS = {1:10, 2:8, 3:6, 4:4, 5:2}` |
| `football-api/app/api/v1/routers/votes.py` | Criar | Endpoints de status, submissão, resultados e pendências |
| `football-api/app/api/v1/router.py` | Modificar | Registrar `votes.router` |
| `football-api/app/db/repositories/group_stats_repo.py` | Criar | Query de estatísticas com filtro `voting_status = 'closed'` e suporte a `period_start` / `period_end` |
| `football-api/app/schemas/group_stats.py` | Criar | `GroupStatsResponse` / `PlayerStatItem` (inclui `period_label`) |
| `football-api/app/api/v1/routers/groups.py` | Modificar | Adicionar `GET /groups/{group_id}/stats` com query params de período |
| `football-frontend/src/lib/api.ts` | Modificar | Tipos e chamadas `votes.*` e `groups.getStats(groupId, period?)` |
| `football-frontend/src/routes/+page.svelte` (home) | Modificar | Banner de votações pendentes abaixo dos big numbers |
| `football-frontend/src/routes/match/[hash]/+page.svelte` | Modificar | Seção de votação + indicador de voto na listagem de confirmados |
| `football-frontend/src/lib/components/VoteForm.svelte` | Criar | Formulário de votação |
| `football-frontend/src/lib/components/VoteResults.svelte` | Criar | Pódio Top 5 e Decepção |
| `football-frontend/src/lib/components/PendingVotesBanner.svelte` | Criar | Banner de votações pendentes (home) |
| `football-frontend/src/routes/groups/[id]/+page.svelte` | Modificar | Aba "Estatísticas" com seletor de período e tabela de ranking |

---

## 11. Critérios de Aceitação

- [x] Votação não aparece antes de 20 minutos após o fim da partida
- [x] Votação encerra automaticamente após 24 horas da abertura
- [x] Apenas participantes com presença confirmada conseguem votar
- [x] Jogador não consegue votar em si mesmo (frontend e backend)
- [x] Voto submetido não pode ser alterado — segunda submissão retorna `409`
- [x] Resultados parciais (em quem cada um votou) não são expostos durante a votação
- [x] Pontuação do Top 5 calculada corretamente: 1º=10, 2º=8, 3º=6, 4º=4, 5º=2
- [x] Empates no Top 5 e na Decepção são exibidos corretamente
- [x] Todas as posições visíveis obrigatórias; Decepção obrigatória quando elegíveis > 5
- [x] Jogador no Top 5 não aparece como opção de Decepção (e vice-versa)
- [x] Contador "X de Y jogadores votaram" atualiza a cada acesso
- [x] Push notification de abertura enviada apenas uma vez por partida
- [x] Partida sem `end_time` usa `match_date 23:59` como referência
- [x] Indicador "votou / não votou" aparece na listagem de confirmados quando `voting_status = 'open'` ou `'closed'`
- [x] Indicador não aparece antes da votação abrir
- [x] Votos individuais nunca são expostos
- [x] Banner de votações pendentes aparece na home abaixo dos big numbers quando há pendências
- [x] Banner não é exibido para super-admins
- [x] Banner desaparece quando não há mais votações pendentes
- [x] Cada item do banner linka corretamente para `/match/[hash]`
- [x] Aba "Estatísticas" exibe ranking anual por padrão
- [x] Seletor de período permite alternar entre anual e mensal (meses do ano corrente)
- [x] Troca de período atualiza a tabela sem recarregar a página
- [x] `period_label` exibido corretamente no cabeçalho da aba ("2026" ou "Março 2026")
- [x] Aba computa pontos e decepções apenas de partidas com `voting_status = 'closed'`
- [x] Horas jogadas computadas independentemente do status da votação
- [x] Ranking não muda durante votação em andamento
- [x] Mensagem de estado vazio exibida quando não há partidas elegíveis no período
- [x] Super-admins globais não aparecem na tabela de estatísticas
- [x] Em mobile, colunas "Decepções" e "Horas" são ocultadas

---

## 12. Dependências

- `football-api/app/services/push.py` — `send_push(db, player_id, title, body, url)`
- `football-api/app/api/v1/router.py` — registro do novo router
- Tabela `attendances` — elegibilidade e iteração para push
- Tabela `matches` — janela de votação via `match_date`, `end_time` e `vote_notified`

---

## 13. Fora de Escopo (desta versão)

- Reação/comentários nos resultados
- Votação aberta para não-participantes
- Edição de voto após submissão
- Estatísticas de períodos anteriores ao ano corrente

---

*Documento elaborado para uso interno da equipe de produto e engenharia do Rachao.app.*
