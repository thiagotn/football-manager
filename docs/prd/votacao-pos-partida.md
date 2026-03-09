# PRD — Votação Pós-Partida
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

Após cada partida encerrada, os participantes poderão votar nos melhores jogadores e na decepção do jogo. A votação é aberta automaticamente 1 hora após o fim da partida e fica disponível por 24 horas.

### 1.2 Objetivo

Aumentar o engajamento dos jogadores no período pós-partida, criando um ritual de reconhecimento dentro do grupo. O resultado da votação é exibido na página da partida ao final do período, gerando histórico e competição saudável entre os membros.

---

## 2. Regras de Negócio

### 2.1 Janela de votação

- **Abertura:** 1 hora após o horário de término da partida (`end_time`). Se a partida não tiver `end_time` definido, usar `match_date` às 23h59 como referência de término.
- **Encerramento:** 24 horas após a abertura.
- Fora dessa janela, o endpoint de submissão retorna erro `403 VOTING_CLOSED`.

### 2.2 Elegibilidade para votar

- Apenas jogadores com `attendance.status = 'confirmed'` na partida podem votar.
- Cada jogador pode submeter **um único voto** por partida (imutável após envio).
- O jogador **não pode votar em si mesmo** em nenhuma categoria.

### 2.3 Categorias de voto

**Top 3 — Melhores da partida**

O votante escolhe até 3 jogadores em ordem de preferência:

| Posição | Pontos |
|:---:|:---:|
| 1º lugar | 10 |
| 2º lugar | 8 |
| 3º lugar | 5 |

O resultado final é a soma dos pontos recebidos por cada jogador. O pódio exibe os 3 com maior pontuação total.

**Decepção do jogo**

O votante escolhe 1 jogador como decepção da partida. O resultado é uma contagem simples de votos — o jogador com mais votos é declarado a decepção da partida. Em caso de empate, todos os empatados são exibidos.

### 2.4 Exibição dos resultados

- Durante a janela de votação: resultados **ocultos** (evita influência nos votos restantes).
- Após o encerramento da janela: resultados **públicos** para todos os membros do grupo.
- A quantidade de participantes que já votaram é exibida durante a votação (ex: "7 de 12 jogadores votaram"), sem revelar os votos individuais.

---

## 3. Requisitos Funcionais

**RF-01 — Abertura automática da votação**
O sistema deve abrir a votação automaticamente 1 hora após o `end_time` da partida. Isso pode ser implementado de forma lazy (calculado no momento da requisição) ou via job agendado que atualiza o status da votação.

**RF-02 — Encerramento automático da votação**
A votação encerra automaticamente 24 horas após a abertura. Após esse prazo, nenhum novo voto é aceito.

**RF-03 — Submissão de voto**
O jogador elegível pode submeter um voto contendo:
- 1 a 3 jogadores para o Top 3 (em ordem de preferência)
- 0 ou 1 jogador para decepção (opcional)

A submissão é atômica — parcial ou com erro não deve persistir nada.

**RF-04 — Imutabilidade do voto**
Uma vez submetido, o voto não pode ser alterado. O endpoint deve retornar `409 ALREADY_VOTED` se o jogador tentar votar novamente.

**RF-05 — Exibição do status da votação**
Na página da partida (`/match/[hash]`), exibir:
- Se a votação ainda não abriu: "Votação abre em X horas"
- Se a votação está aberta: formulário de voto + contador "X de Y jogadores votaram"
- Se o jogador já votou: confirmação do voto enviado + contador
- Se a votação encerrou: resultados completos (Top 3 + Decepção)

**RF-06 — Resultados**
Após o encerramento, exibir na página da partida:
- Pódio do Top 3 com nome, foto (se houver) e pontuação total
- Decepção do jogo com nome e quantidade de votos recebidos
- Em caso de empate no Top 3: ordenar por número de votos recebidos como desempate; se ainda empatado, exibir todos
- Em caso de empate na Decepção: exibir todos os empatados

**RF-07 — Notificação de abertura**
Ao abrir a votação, enviar push notification para todos os participantes confirmados da partida (integrar com o serviço `send_push` existente).

---

## 4. Requisitos Não Funcionais

**RNF-01** — A elegibilidade para votar deve ser validada no backend, nunca apenas no frontend.  
**RNF-02** — A janela de votação deve ser calculada com base no fuso horário de Brasília (BRT, UTC-3).  
**RNF-03** — Os resultados parciais não devem ser expostos por nenhum endpoint enquanto a votação estiver aberta.  
**RNF-04** — A submissão de voto deve ser idempotente por `(match_id, voter_id)` — tentativas duplicadas retornam `409` sem efeito colateral.

---

## 5. Modelagem de Dados

```sql
-- Migration: 016_match_votes.sql

-- Registro de cada voto submetido
CREATE TABLE match_votes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    match_id    UUID NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    voter_id    UUID NOT NULL REFERENCES players(id),
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (match_id, voter_id)    -- um voto por jogador por partida
);

-- Escolhas do top 3 (1 a 3 registros por voto)
CREATE TABLE match_vote_top3 (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vote_id     UUID NOT NULL REFERENCES match_votes(id) ON DELETE CASCADE,
    player_id   UUID NOT NULL REFERENCES players(id),
    position    SMALLINT NOT NULL CHECK (position IN (1, 2, 3)),
    points      SMALLINT NOT NULL CHECK (points IN (5, 8, 10)),
    UNIQUE (vote_id, position),    -- uma posição por voto
    UNIQUE (vote_id, player_id)    -- um jogador por posição por voto
);

-- Escolha da decepção (0 ou 1 registro por voto)
CREATE TABLE match_vote_flop (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vote_id     UUID NOT NULL REFERENCES match_votes(id) ON DELETE CASCADE,
    player_id   UUID NOT NULL REFERENCES players(id),
    UNIQUE (vote_id)               -- uma decepção por voto
);

-- Índices
CREATE INDEX idx_match_votes_match ON match_votes (match_id);
CREATE INDEX idx_match_vote_top3_match ON match_vote_top3 (vote_id);
CREATE INDEX idx_match_vote_flop_match ON match_vote_flop (vote_id);
```

### 5.1 Lógica da janela de votação

A janela de votação é calculada dinamicamente a partir da partida, sem coluna extra na tabela `matches`:

```python
from datetime import datetime, timedelta
import pytz

BRT = pytz.timezone("America/Sao_Paulo")

def voting_window(match) -> tuple[datetime, datetime] | None:
    """Retorna (opens_at, closes_at) ou None se end_time não definido."""
    if match.end_time is None:
        end_dt = datetime.combine(match.match_date, time(23, 59), tzinfo=BRT)
    else:
        end_dt = datetime.combine(match.match_date, match.end_time, tzinfo=BRT)
    opens_at  = end_dt + timedelta(hours=1)
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
| `GET` | `/api/v1/matches/{match_id}/votes/status` | Status da votação e contagem de votos |
| `POST` | `/api/v1/matches/{match_id}/votes` | Submete o voto do jogador autenticado |
| `GET` | `/api/v1/matches/{match_id}/votes/results` | Resultados (apenas após encerramento) |

### 6.1 `GET /matches/{match_id}/votes/status`

Disponível durante e após a votação. Retorna o status da janela, se o usuário já votou e a contagem anônima.

**Response 200:**
```json
{
  "status": "open",
  "opens_at": "2026-03-08T23:00:00-03:00",
  "closes_at": "2026-03-09T23:00:00-03:00",
  "voter_count": 7,
  "eligible_count": 12,
  "current_player_voted": false
}
```

### 6.2 `POST /matches/{match_id}/votes`

**Request:**
```json
{
  "top3": [
    { "player_id": "uuid-joao",   "position": 1 },
    { "player_id": "uuid-pedro",  "position": 2 },
    { "player_id": "uuid-carlos", "position": 3 }
  ],
  "flop_player_id": "uuid-lucas"
}
```

> `top3` deve ter entre 1 e 3 itens. `flop_player_id` é opcional.  
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
- `422` — posições duplicadas ou jogador repetido no top3

### 6.3 `GET /matches/{match_id}/votes/results`

Disponível apenas quando `voting_status == 'closed'`. Retorna erro `403 RESULTS_NOT_AVAILABLE` se a votação ainda estiver aberta ou não tiver iniciado.

**Response 200:**
```json
{
  "top3": [
    { "position": 1, "player_id": "uuid-joao",   "name": "João",   "points": 42 },
    { "position": 2, "player_id": "uuid-pedro",  "name": "Pedro",  "points": 35 },
    { "position": 3, "player_id": "uuid-carlos", "name": "Carlos", "points": 28 }
  ],
  "flop": [
    { "player_id": "uuid-lucas", "name": "Lucas", "votes": 5 }
  ],
  "total_voters": 9,
  "eligible_voters": 12
}
```

> `flop` é um array para suportar empates. `top3` também pode ter mais de 3 itens em caso de empate na 3ª posição.

---

## 7. Fluxo Principal

```
Partida encerrada (status = 'closed')
    ↓
1 hora após end_time
    ↓
Sistema calcula voting_status = 'open'
Job (ou evento lazy) dispara send_push para participantes confirmados:
    "🏆 Votação aberta! Escolha os melhores da pelada de ontem."
    ↓
Participante acessa /match/[hash]
Frontend chama GET /votes/status → exibe formulário de voto
    ↓
Participante submete voto → POST /votes
Backend valida elegibilidade, janela, autovotos e duplicatas
Persiste match_votes + match_vote_top3 + match_vote_flop (transação)
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
│  TOP 3 — Melhores da partida         │
│                                      │
│  🥇 1º lugar  [ Selecionar jogador ▾]│
│  🥈 2º lugar  [ Selecionar jogador ▾]│
│  🥉 3º lugar  [ Selecionar jogador ▾]│
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
│                                      │
│  😬 Decepção do Jogo                 │
│  Lucas Ferreira         5 votos      │
│──────────────────────────────────────│
│  9 de 12 jogadores votaram           │
└──────────────────────────────────────┘
```

### 8.2 Regras de UX do formulário

- As listas de seleção do Top 3 e Decepção excluem automaticamente o próprio jogador logado.
- Um jogador selecionado no 1º lugar não pode aparecer nas opções do 2º e 3º (e vice-versa).
- O mesmo jogador do Top 3 pode ser votado como decepção? **Não** — o jogador escolhido para decepção deve ser removido das opções do Top 3 e vice-versa.
- Botão "Enviar voto" fica desabilitado até que ao menos o 1º lugar seja selecionado.

---

## 9. Arquivos a Criar/Modificar

| Arquivo | Ação | Descrição |
|---|---|---|
| `football-api/migrations/016_match_votes.sql` | Criar | Tabelas `match_votes`, `match_vote_top3`, `match_vote_flop` |
| `football-api/app/models/match_vote.py` | Criar | Models SQLAlchemy das 3 tabelas |
| `football-api/app/db/repositories/vote_repo.py` | Criar | Repositório com queries de submissão e agregação |
| `football-api/app/services/voting.py` | Criar | Lógica de `voting_window`, `voting_status`, validações |
| `football-api/app/api/v1/routers/votes.py` | Criar | Endpoints de status, submissão e resultados |
| `football-api/app/main.py` | Modificar | Registrar router de votes; adicionar job de notificação de abertura |
| `football-frontend/src/lib/api.ts` | Modificar | Adicionar chamadas `votes.getStatus`, `votes.submit`, `votes.getResults` |
| `football-frontend/src/routes/match/[hash]/+page.svelte` | Modificar | Adicionar seção de votação conforme estado |
| `football-frontend/src/lib/components/VoteForm.svelte` | Criar | Formulário de votação |
| `football-frontend/src/lib/components/VoteResults.svelte` | Criar | Exibição do pódio e decepção |

---

## 10. Critérios de Aceitação

- [ ] Votação não aparece antes de 1 hora após o fim da partida
- [ ] Votação encerra automaticamente após 24 horas da abertura
- [ ] Apenas participantes com presença confirmada conseguem votar
- [ ] Jogador não consegue votar em si mesmo (frontend e backend)
- [ ] Voto submetido não pode ser alterado — segunda submissão retorna `409`
- [ ] Resultados parciais não são expostos enquanto a votação está aberta
- [ ] Pontuação do Top 3 é calculada corretamente: 1º=10, 2º=8, 3º=5
- [ ] Empates no Top 3 e na Decepção são exibidos corretamente
- [ ] Contador "X de Y jogadores votaram" atualiza em tempo real (ou a cada reload)
- [ ] Push notification é enviada ao abrir a votação
- [ ] Partida sem `end_time` usa `match_date 23:59` como referência

---

## 11. Dependências

- `football-api/app/services/push.py` — `send_push` para notificação de abertura
- `football-api/app/main.py` — `AsyncIOScheduler` (APScheduler já instalado) para job de abertura de votação
- Tabela `attendances` — para validar elegibilidade (`status = 'confirmed'`)
- Tabela `matches` — para calcular a janela de votação via `match_date` e `end_time`

---

## 12. Fora de Escopo (desta versão)

- Ranking histórico de melhores jogadores do grupo (agregação entre partidas)
- Ranking histórico de decepções
- Reação/comentários nos resultados
- Votação aberta para não-participantes (ex: torcida do grupo)
- Edição de voto após submissão

---

*Documento elaborado para uso interno da equipe de produto e engenharia do Rachao.app.*
