# PRD — Votação Pós-Partida
## Rachao.app · Gerenciamento de Grupos e Partidas

| | |
|---|---|
| **Versão** | 1.3 |
| **Status** | Implementado (v1.2) · Aba de Estatísticas pendente |
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

A decepção é **obrigatória** quando há jogadores elegíveis suficientes para preenchê-la, ou seja, quando `elegíveis > 5` (após preencher as 5 posições do Top 5, sobra ao menos 1 jogador). Se `elegíveis ≤ 5`, o campo de decepção não é exibido.

### 2.4 Exibição dos resultados

- Durante a janela de votação: resultados **ocultos** (evita influência nos votos restantes).
- Após o encerramento da janela: resultados **públicos** para todos os membros do grupo.
- A quantidade de participantes que já votaram é exibida durante a votação (ex: "7 de 12 jogadores votaram"), sem revelar os votos individuais.

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

**RF-08 — Aba de Estatísticas do grupo**
Na página de detalhes do grupo (`/groups/[id]`), adicionar uma aba **"Estatísticas"** (ao lado das abas Próximos / Últimos / Jogadores) que exibe o ranking acumulado de todos os membros do grupo. A tabela contém:

| Coluna | Fonte | Descrição |
|---|---|---|
| Posição | calculada | Ranking por pontos de votação (medalha para top 3) |
| Jogador | `players.nickname \|\| name` | Nome do membro |
| Pts Votação | `SUM(match_vote_top5.points)` | Total de pontos recebidos nas votações de todas as partidas encerradas do grupo |
| Decepções | `COUNT(match_vote_flop)` | Total de vezes eleito decepção do jogo |
| Horas jogadas | `SUM(end_time - start_time)` | Total de horas em partidas confirmadas com `end_time` definido |

Regras:
- Apenas partidas com `status = 'closed'` do grupo são computadas.
- Apenas membros com `group_members.role != 'admin'` global (`players.role != 'admin'`) são listados.
- Partidas sem `end_time` não contribuem para as horas jogadas (tratadas como 0).
- Ordenação padrão: pontos de votação decrescente; em caso de empate, alfabética por nome.
- A aba é exibida sempre que o grupo tiver ao menos 1 partida encerrada; caso contrário, mostrar mensagem vazia.
- Visível para todos os membros logados (inclusive admins de grupo).

**RF-07 — Notificação de abertura**
Ao acessar o status da votação pela primeira vez com `status = 'open'` (abordagem lazy), enviar push notification para todos os participantes confirmados da partida. Para evitar notificações duplicadas, usar uma flag `vote_notified` na tabela `matches` ou controle por cache/flag na requisição.

> **Nota de implementação:** `send_push` em `app/services/push.py` recebe `(db, player_id, title, body, url)` e opera por player. O disparo das notificações deve iterar sobre os `attendances` confirmados da partida e chamar `send_push` para cada um — conforme o padrão já utilizado em `recurrence.py`.

---

## 4. Requisitos Não Funcionais

**RNF-01** — A elegibilidade para votar deve ser validada no backend, nunca apenas no frontend.
**RNF-02** — A janela de votação deve ser calculada com base no fuso horário de Brasília (BRT, UTC-3), usando `zoneinfo.ZoneInfo("America/Sao_Paulo")` (stdlib Python 3.9+, sem dependência de `pytz`).
**RNF-03** — Os resultados parciais não devem ser expostos por nenhum endpoint enquanto a votação estiver aberta.
**RNF-04** — A submissão de voto deve ser idempotente por `(match_id, voter_id)` — tentativas duplicadas retornam `409` sem efeito colateral.

---

## 5. Modelagem de Dados

```sql
-- Migration: 017_match_votes.sql

-- Registro de cada voto submetido
CREATE TABLE match_votes (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    match_id     UUID NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    voter_id     UUID NOT NULL REFERENCES players(id),
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (match_id, voter_id)    -- um voto por jogador por partida
);

-- Escolhas do top 5 (1 a 5 registros por voto)
CREATE TABLE match_vote_top5 (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vote_id    UUID NOT NULL REFERENCES match_votes(id) ON DELETE CASCADE,
    player_id  UUID NOT NULL REFERENCES players(id),
    position   SMALLINT NOT NULL CHECK (position IN (1, 2, 3, 4, 5)),
    points     SMALLINT NOT NULL CHECK (points IN (2, 4, 6, 8, 10)),
    UNIQUE (vote_id, position),    -- uma posição por voto
    UNIQUE (vote_id, player_id)    -- um jogador por posição por voto
);

-- Escolha da decepção (0 ou 1 registro por voto)
CREATE TABLE match_vote_flop (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vote_id    UUID NOT NULL REFERENCES match_votes(id) ON DELETE CASCADE,
    player_id  UUID NOT NULL REFERENCES players(id),
    UNIQUE (vote_id)               -- uma decepção por voto
);

-- Coluna para controle de notificação de abertura
ALTER TABLE matches ADD COLUMN vote_notified BOOLEAN NOT NULL DEFAULT FALSE;

-- Índices
CREATE INDEX idx_match_votes_match   ON match_votes (match_id);
CREATE INDEX idx_match_vote_top5_vote ON match_vote_top5 (vote_id);
CREATE INDEX idx_match_vote_flop_vote ON match_vote_flop (vote_id);
```

> `UNIQUE` em `(match_id, voter_id)` garante no banco que existe no máximo um voto por player por partida.
> A coluna `vote_notified` na tabela `matches` garante que a push notification de abertura seja enviada apenas uma vez.

### 5.1 Pontuação por posição

| Posição | Pontos | Constraint SQL |
|:---:|:---:|:---:|
| 1º | 10 | `points = 10` |
| 2º | 8  | `points = 8`  |
| 3º | 6  | `points = 6`  |
| 4º | 4  | `points = 4`  |
| 5º | 2  | `points = 2`  |

O campo `points` é calculado no backend pelo mapeamento `{1: 10, 2: 8, 3: 6, 4: 4, 5: 2}` — nunca enviado pelo cliente.

### 5.2 Consulta de Estatísticas por Grupo

As três métricas (pontos, decepções, horas) são calculadas de forma independente por subquery/CTE e unidas por `player_id`. Pseudocódigo da consulta principal:

```sql
-- Pontos de votação acumulados por jogador no grupo
WITH vote_points AS (
    SELECT mvt.player_id, SUM(mvt.points) AS total_points
    FROM match_vote_top5 mvt
    JOIN match_votes mv  ON mv.id       = mvt.vote_id
    JOIN matches m       ON m.id        = mv.match_id
    WHERE m.group_id = :group_id AND m.status = 'closed'
    GROUP BY mvt.player_id
),
-- Decepções acumuladas por jogador no grupo
flop_counts AS (
    SELECT mvf.player_id, COUNT(*) AS total_flops
    FROM match_vote_flop mvf
    JOIN match_votes mv ON mv.id    = mvf.vote_id
    JOIN matches m      ON m.id     = mv.match_id
    WHERE m.group_id = :group_id AND m.status = 'closed'
    GROUP BY mvf.player_id
),
-- Minutos jogados por jogador no grupo (apenas partidas com end_time)
minutes_played AS (
    SELECT a.player_id,
           SUM(
               EXTRACT(EPOCH FROM (m.end_time::time - m.start_time::time)) / 60
           ) AS total_minutes
    FROM attendances a
    JOIN matches m ON m.id = a.match_id
    WHERE m.group_id = :group_id
      AND m.status   = 'closed'
      AND a.status   = 'confirmed'
      AND m.end_time IS NOT NULL
    GROUP BY a.player_id
)
-- Resultado final: todos os membros não-admin do grupo
SELECT
    p.id                                    AS player_id,
    COALESCE(p.nickname, p.name)            AS display_name,
    COALESCE(vp.total_points, 0)            AS vote_points,
    COALESCE(fc.total_flops, 0)             AS flop_votes,
    COALESCE(mp.total_minutes, 0)           AS minutes_played
FROM group_members gm
JOIN players p       ON p.id = gm.player_id
LEFT JOIN vote_points  vp ON vp.player_id = p.id
LEFT JOIN flop_counts  fc ON fc.player_id = p.id
LEFT JOIN minutes_played mp ON mp.player_id = p.id
WHERE gm.group_id = :group_id
  AND p.role      != 'admin'     -- exclui super-admins globais
ORDER BY vote_points DESC, display_name ASC;
```

> **Nota sobre horas jogadas:** o cálculo usa `end_time - start_time` da própria partida. Partidas sem `end_time` são excluídas desse cômputo. A coluna exibe o total como `Xh Ymin` (ex: "4h 30min"); se zero, exibir "—".

### 5.3 Lógica da janela de votação

A janela de votação é calculada dinamicamente a partir da partida, sem coluna extra de status:

```python
from datetime import datetime, timedelta, time as dt_time
from zoneinfo import ZoneInfo

BRT = ZoneInfo("America/Sao_Paulo")

def voting_window(match) -> tuple[datetime, datetime]:
    """Retorna (opens_at, closes_at) em BRT."""
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

> Usa `zoneinfo` (stdlib Python 3.9+) em vez de `pytz` — sem dependência adicional.

---

## 6. Endpoints da API

| Método | Endpoint | Descrição |
|---|---|---|
| `GET` | `/api/v1/matches/{match_id}/votes/status` | Status da votação e contagem de votos |
| `POST` | `/api/v1/matches/{match_id}/votes` | Submete o voto do jogador autenticado |
| `GET` | `/api/v1/matches/{match_id}/votes/results` | Resultados (apenas após encerramento) |
| `GET` | `/api/v1/groups/{group_id}/stats` | Estatísticas acumuladas dos jogadores do grupo |

> Os endpoints de votação usam `match_id` (UUID). O frontend já tem o `id` da partida via `GET /matches/public/{hash}`.

### 6.4 `GET /groups/{group_id}/stats`

Requer autenticação. Disponível para qualquer membro do grupo (inclusive admins de grupo).

**Response 200:**
```json
{
  "players": [
    {
      "player_id": "uuid-joao",
      "display_name": "Joãozinho",
      "vote_points": 84,
      "flop_votes": 2,
      "minutes_played": 270
    },
    {
      "player_id": "uuid-pedro",
      "display_name": "Pedro",
      "vote_points": 60,
      "flop_votes": 0,
      "minutes_played": 330
    }
  ]
}
```

> `minutes_played` é um inteiro (total de minutos). O frontend converte para `Xh Ymin`.
> Apenas membros com `players.role != 'admin'` (não super-admin) são incluídos.
> A resposta inclui todos os membros do grupo, mesmo os com todos os valores zerados.

### 6.1 `GET /matches/{match_id}/votes/status`

Disponível durante e após a votação. Retorna o status da janela, se o usuário já votou e a contagem anônima. **Este endpoint também dispara a notificação push se `status = 'open'` e `vote_notified = false`.**

**Response 200:**
```json
{
  "status": "open",
  "opens_at": "2026-03-08T22:20:00-03:00",
  "closes_at": "2026-03-09T22:20:00-03:00",
  "voter_count": 7,
  "eligible_count": 12,
  "current_player_voted": false
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

> `top5` deve ter entre 1 e 5 itens. `flop_player_id` é opcional.
> O campo `points` é calculado no backend conforme a posição — nunca enviado pelo cliente.

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

Disponível apenas quando `voting_status == 'closed'`. Retorna erro `403 RESULTS_NOT_AVAILABLE` se a votação ainda estiver aberta ou não tiver iniciado.

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

> `flop` é um array para suportar empates. `top5` também pode ter mais de 5 itens em caso de empate na 5ª posição.

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
    Mensagem: "🏆 Votação aberta! Escolha os melhores da pelada."
    ↓
Participante acessa /match/[hash]
Frontend chama GET /votes/status → exibe formulário de voto
    ↓
Participante submete voto → POST /votes
Backend valida elegibilidade, janela, autovotos e duplicatas
Persiste match_votes + match_vote_top5 + match_vote_flop (transação)
    ↓
24 horas após abertura
    ↓
voting_status = 'closed'
GET /votes/results passa a responder
Página da partida exibe pódio e decepção
```

---

## 8. Interface do Usuário

### 8.1 Na página da partida (`/match/[hash]`)

**Estado: votação não aberta**
```
┌──────────────────────────────────────┐
│  🗳️  Votação                         │
│  Abre em 0h 45min                    │
└──────────────────────────────────────┘
```

**Estado: votação aberta (jogador ainda não votou)**
```
┌──────────────────────────────────────┐
│  🗳️  Votação aberta · 7 de 12 votos  │
│  Fecha em 18h 20min                  │
│──────────────────────────────────────│
│  TOP 5 — Melhores da partida         │
│                                      │
│  🥇 1º  (10 pts) [ Selecionar ▾]    │
│  🥈 2º  ( 8 pts) [ Selecionar ▾]    │
│  🥉 3º  ( 6 pts) [ Selecionar ▾]    │
│     4º  ( 4 pts) [ Selecionar ▾]    │
│     5º  ( 2 pts) [ Selecionar ▾]    │
│                                      │
│  DECEPÇÃO DO JOGO  (opcional)        │
│  [ Selecionar jogador ▾]             │
│                                      │
│  [ Enviar voto ]                     │
└──────────────────────────────────────┘
```

**Estado: votação aberta (jogador já votou)**
```
┌──────────────────────────────────────┐
│  ✅ Você já votou · 9 de 12 votos    │
│  Resultados disponíveis em 14h       │
└──────────────────────────────────────┘
```

**Estado: votação encerrada — resultados**
```
┌──────────────────────────────────────┐
│  🏆 Melhores da Partida              │
│──────────────────────────────────────│
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

### 8.2 Regras de UX do formulário

- As listas de seleção do Top 5 e Decepção excluem automaticamente o próprio jogador logado.
- Um jogador selecionado em qualquer posição do Top 5 não pode aparecer nas demais posições nem na Decepção.
- Um jogador selecionado como Decepção é removido das opções do Top 5.
- Somente as posições onde há jogadores disponíveis são exibidas (`min(elegíveis, 5)`); todas são obrigatórias.
- O campo Decepção é exibido e obrigatório apenas quando `elegíveis > 5`; do contrário, é ocultado.
- Botão "Enviar voto" fica desabilitado até que todas as posições visíveis e a Decepção (se exibida) sejam preenchidas.

### 8.3 Aba de Estatísticas na página do grupo (`/groups/[id]`)

**Layout da tabela (mobile: colunas secundárias ocultas; desktop: todas visíveis):**

```
┌─────────────────────────────────────────────────────────────┐
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
- Coluna "Decepções": destaque em vermelho quando > 0.
- Coluna "Horas": exibe `Xh Ymin`; se zero, exibe "—".
- Em mobile: ocultar colunas "Decepções" e "Horas" (`hidden sm:table-cell`), manter Jogador e Pts Votação.
- Se não houver partidas encerradas: exibir estado vazio com mensagem "Nenhuma estatística disponível ainda. As estatísticas aparecem após o encerramento das partidas."

### 8.4 Banner de votação na página do grupo

Na página de detalhes do grupo (`/groups/[id]`), exibir um banner entre os detalhes gerais e as abas (Próximos / Últimos / Jogadores) para cada partida encerrada com votação aberta e não votada pelo jogador atual. O banner é um link para a página da partida e exibe o número da partida, o `time_label` e a contagem de votantes. Visível apenas para jogadores logados (não-admin).

---

## 9. Decisões de Arquitetura

- **Lazy vs. job agendado:** o status da votação é calculado lazily no momento da requisição via `voting_status(match)`. Não é necessário job agendado para controlar abertura/encerramento.
- **Push notification:** disparada no primeiro `GET /votes/status` com `status='open'`, controlada pela flag `vote_notified` na tabela `matches`. Padrão idêntico ao de `recurrence.py` — cria sessão DB e chama `send_push(db, player_id, ...)` para cada participante confirmado.
- **Router registration:** o router de votes é registrado em `football-api/app/api/v1/router.py`, não em `main.py`. O `main.py` permanece sem alterações (apenas registra o `api_router`).
- **`zoneinfo` em vez de `pytz`:** stdlib Python 3.9+, sem dependência adicional. Uso: `ZoneInfo("America/Sao_Paulo")` com `datetime.combine(...).replace(tzinfo=BRT)`.
- **Tabela renomeada:** `match_vote_top3` → `match_vote_top5` para refletir a expansão para 5 posições.

---

## 10. Arquivos a Criar/Modificar

| Arquivo | Ação | Descrição |
|---|---|---|
| `football-api/migrations/017_match_votes.sql` | Criar | Tabelas `match_votes`, `match_vote_top5`, `match_vote_flop`; coluna `vote_notified` em `matches` |
| `football-api/app/models/match_vote.py` | Criar | Models SQLAlchemy das 3 tabelas de votação |
| `football-api/app/models/match.py` | Modificar | Adicionar campo `vote_notified: bool` ao model `Match` |
| `football-api/app/schemas/vote.py` | Criar | Schemas Pydantic: `VoteSubmitRequest`, `VoteStatusResponse`, `VoteResultsResponse` |
| `football-api/app/db/repositories/vote_repo.py` | Criar | Repositório: submissão atômica, agregação de resultados, contagem de votos |
| `football-api/app/services/voting.py` | Criar | Funções `voting_window`, `voting_status`, mapa de pontuação `POINTS = {1:10, 2:8, 3:6, 4:4, 5:2}` |
| `football-api/app/api/v1/routers/votes.py` | Criar | Endpoints de status, submissão e resultados |
| `football-api/app/api/v1/router.py` | Modificar | Importar e registrar `votes.router` |
| `football-frontend/src/lib/api.ts` | Modificar | Adicionar tipos e chamadas `votes.getStatus`, `votes.submit`, `votes.getResults` |
| `football-frontend/src/routes/match/[hash]/+page.svelte` | Modificar | Adicionar seção de votação conforme estado |
| `football-frontend/src/lib/components/VoteForm.svelte` | Criar | Formulário de votação (Top 5 + Decepção com lógica de exclusão mútua) |
| `football-frontend/src/lib/components/VoteResults.svelte` | Criar | Exibição do pódio (Top 5) e decepção |
| `football-api/app/db/repositories/group_stats_repo.py` | Criar | Query de estatísticas do grupo (pontos, decepções, horas) via CTE |
| `football-api/app/schemas/group_stats.py` | Criar | Schema `GroupStatsResponse` com lista de `PlayerStatItem` |
| `football-api/app/api/v1/routers/groups.py` | Modificar | Adicionar endpoint `GET /groups/{group_id}/stats` |
| `football-frontend/src/lib/api.ts` | Modificar | Adicionar tipo `GroupStatsResponse` e `groups.getStats(groupId)` |
| `football-frontend/src/routes/groups/[id]/+page.svelte` | Modificar | Adicionar aba "Estatísticas" com tabela de ranking |

---

## 11. Critérios de Aceitação

- [ ] Votação não aparece antes de 20 minutos após o fim da partida
- [ ] Votação encerra automaticamente após 24 horas da abertura
- [ ] Apenas participantes com presença confirmada conseguem votar
- [ ] Jogador não consegue votar em si mesmo (frontend e backend)
- [ ] Voto submetido não pode ser alterado — segunda submissão retorna `409`
- [ ] Resultados parciais não são expostos enquanto a votação está aberta
- [ ] Pontuação do Top 5 é calculada corretamente: 1º=10, 2º=8, 3º=6, 4º=4, 5º=2
- [ ] Empates no Top 5 e na Decepção são exibidos corretamente
- [ ] Todas as posições visíveis são obrigatórias; Decepção obrigatória quando elegíveis > 5
- [ ] Jogador escolhido no Top 5 não aparece como opção de Decepção (e vice-versa)
- [ ] Contador "X de Y jogadores votaram" atualiza a cada acesso à página
- [ ] Push notification de abertura é enviada apenas uma vez por partida
- [ ] Partida sem `end_time` usa `match_date 23:59` como referência
- [ ] Aba "Estatísticas" exibe ranking correto de pontos acumulados de votação
- [ ] Coluna "Decepções" reflete o total de vezes eleito em todas as partidas do grupo
- [ ] Coluna "Horas" soma apenas partidas com `end_time` definido; exibe "—" se zero
- [ ] Super-admins globais não aparecem na tabela de estatísticas
- [ ] Em mobile, colunas "Decepções" e "Horas" são ocultadas
- [ ] Estado vazio exibido quando não há partidas encerradas

---

## 12. Dependências

- `football-api/app/services/push.py` — `send_push(db, player_id, title, body, url)` para notificação de abertura
- `football-api/app/api/v1/router.py` — para registrar o novo router
- Tabela `attendances` — para validar elegibilidade (`status = 'confirmed'`) e iterar para push
- Tabela `matches` — para calcular a janela de votação via `match_date`, `end_time` e `vote_notified`

---

## 13. Fora de Escopo (desta versão)

- Reação/comentários nos resultados
- Votação aberta para não-participantes (ex: torcida do grupo)
- Edição de voto após submissão

---

*Documento elaborado para uso interno da equipe de produto e engenharia do Rachao.app.*
