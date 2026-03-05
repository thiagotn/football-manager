# PRD — Recorrência com Herança de Convidados

**Status:** Implementado
**Data:** 2026-03-04
**Commits:** `e13cf17`, `af9e87e`

---

## 1. Visão Geral

Partidas criadas automaticamente pela regra de recorrência **herdam a lista de convidados da última partida do grupo**.

Os convidados herdados devem **confirmar presença novamente**, respeitando as regras de confirmação já existentes no sistema.

Essa abordagem garante:

- consistência entre partidas
- menos trabalho manual para o presidente
- manutenção do mesmo grupo de jogadores recorrentes

---

## 2. Comportamento Implementado

Quando o job diário executa:

1. Fecha automaticamente todas as partidas abertas cuja `match_date < hoje`.
2. Para cada grupo com recorrência ativa, verifica se existe alguma partida aberta.
3. Se não houver partida aberta e a última partida já passou, cria a próxima com `match_date = última + 7 dias`.
4. Copia os convidados (`attendances`) da última partida, todos com status `pending`.

Regras:

- Ninguém nasce confirmado automaticamente — todos precisam confirmar novamente.
- Se já há uma partida aberta no grupo, o job não cria outra.
- Se o grupo não tem nenhuma partida anterior, o job não age.

---

## 3. Regras de Negócio

### RN05 — Herança de Convidados

Partidas geradas automaticamente copiam a lista de `attendances` da última partida do grupo.

Não são copiados:

- status de presença (sempre inicia como `pending`)
- histórico de confirmações anteriores

### RN06 — Status Inicial dos Convidados

Todos os registros de `Attendance` criados pela herança iniciam com `status = pending`.

| Valor       | Descrição                  |
|-------------|----------------------------|
| `pending`   | Aguardando confirmação     |
| `confirmed` | Presença confirmada        |
| `declined`  | Jogador não vai comparecer |

### RN07 — Limite de Jogadores

As regras de `max_players` continuam válidas. A confirmação segue o fluxo normal.

### RN08 — Convidados Removidos

A base para herança é a lista atual de `attendances` da última partida. Jogadores removidos manualmente não são herdados.

### RN09 — Encerramento Automático de Partidas

Partidas com `match_date < hoje` e `status = open` são fechadas automaticamente (`status = closed`):

- **Pelo job diário:** executado às 07:00 BRT (10:00 UTC), antes da criação de novas partidas.
- **Em tempo real:** ao tentar confirmar presença em uma partida vencida, o endpoint de attendance fecha a partida imediatamente e rejeita a requisição com erro "Esta partida está encerrada".

Isso garante consistência mesmo que o job ainda não tenha rodado no dia.

---

## 4. Modelo de Dados

### Tabela `groups`

| Campo                | Tipo    | Descrição                                      |
|----------------------|---------|------------------------------------------------|
| `recurrence_enabled` | Boolean | Ativa a criação automática de partidas semanais |

### Tabela `matches`

| Campo             | Tipo            | Descrição                           |
|-------------------|-----------------|-------------------------------------|
| `id`              | UUID            | Identificador                       |
| `group_id`        | UUID (FK)       | Grupo ao qual pertence              |
| `match_date`      | Date            | Data da partida                     |
| `start_time`      | Time            | Horário de início                   |
| `end_time`        | Time (nullable) | Horário de término                  |
| `location`        | String(200)     | Local                               |
| `address`         | String(300)     | Endereço completo (opcional)        |
| `court_type`      | Enum (nullable) | campo / sintetico / terrao / quadra |
| `players_per_team`| SmallInt        | Jogadores por time (opcional)       |
| `max_players`     | SmallInt        | Limite de jogadores (opcional)      |
| `notes`           | Text            | Observações (opcional)              |
| `hash`            | String(12)      | Hash público para compartilhamento  |
| `status`          | Enum            | `open` / `closed`                   |
| `created_by_id`   | UUID (FK)       | Jogador que criou (`null` se automático) |

### Tabela `attendances`

| Campo        | Tipo      | Descrição                            |
|--------------|-----------|--------------------------------------|
| `id`         | UUID      | Identificador                        |
| `match_id`   | UUID (FK) | Partida                              |
| `player_id`  | UUID (FK) | Jogador                              |
| `status`     | Enum      | `pending` / `confirmed` / `declined` |
| `updated_at` | Timestamp | Última atualização                   |

Constraint: `UNIQUE(match_id, player_id)`.

---

## 5. Fluxo do Job (execução diária — 07:00 BRT)

```
1. Fechar partidas vencidas:
   UPDATE matches SET status = 'closed'
   WHERE match_date < hoje AND status = 'open'

2. Para cada grupo com recurrence_enabled = true:

   se has_open_match(grupo.id):
       ignorar  # já tem partida aberta

   ultima_partida = get_last_match(grupo.id)

   se não existe ultima_partida:
       ignorar

   se ultima_partida.match_date >= hoje:
       ignorar  # partida ainda não aconteceu

   proxima_data = ultima_partida.match_date + 7 dias

   nova_partida = criar_partida(
       group_id         = grupo.id,
       match_date       = proxima_data,
       start_time       = ultima_partida.start_time,
       end_time         = ultima_partida.end_time,
       location         = ultima_partida.location,
       address          = ultima_partida.address,
       court_type       = ultima_partida.court_type,
       players_per_team = ultima_partida.players_per_team,
       max_players      = ultima_partida.max_players,
       notes            = ultima_partida.notes,
       status           = 'open',
       created_by_id    = null
   )

   para cada player_id em get_attendance_player_ids(ultima_partida.id):
       inserir attendance(
           match_id  = nova_partida.id,
           player_id = player_id,
           status    = 'pending'
       )
```

O job é **idempotente** — pode rodar múltiplas vezes sem criar duplicatas, pois a condição `has_open_match` impede criação se já existe partida aberta.

---

## 6. Exemplo de Funcionamento

### Partida da semana atual (após a data)

| Jogador | Status      |
|---------|-------------|
| João    | `confirmed` |
| Pedro   | `confirmed` |
| Lucas   | `declined`  |
| Carlos  | `confirmed` |

### Nova partida criada automaticamente (semana seguinte)

| Jogador | Status    |
|---------|-----------|
| João    | `pending` |
| Pedro   | `pending` |
| Lucas   | `pending` |
| Carlos  | `pending` |

---

## 7. Arquivos Implementados

| Arquivo | Mudança |
|---|---|
| `migrations/011_group_recurrence.sql` | Coluna `recurrence_enabled` na tabela `groups` |
| `app/models/group.py` | Campo `recurrence_enabled` no ORM |
| `app/schemas/group.py` | Campo exposto em `GroupResponse` e aceito em `GroupUpdate` |
| `app/api/v1/routers/groups.py` | `recurrence_enabled` incluído na resposta de `get_group` |
| `app/db/repositories/group_repo.py` | Método `get_groups_with_recurrence()` |
| `app/db/repositories/match_repo.py` | Métodos `get_last_match()`, `has_open_match()`, `get_attendance_player_ids()`, `close_past_matches()` |
| `app/services/recurrence.py` | Serviço com lógica completa (arquivo novo) |
| `app/main.py` | `AsyncIOScheduler` iniciado no lifespan — executa às 07:00 BRT |
| `pyproject.toml` | Dependência `apscheduler ^3.10.0` adicionada |
| `src/lib/api.ts` | `recurrence_enabled` no tipo `Group` e em `groups.update()` |
| `src/routes/groups/[id]/+page.svelte` | Toggle no modal "Editar Grupo" + indicador no header |

---

## 8. Benefícios

- Elimina recriação manual da lista de convidados
- Mantém o mesmo grupo recorrente sem esforço do presidente
- Preserva a dinâmica de confirmação a cada partida
- Partidas vencidas são fechadas automaticamente, impedindo RSVPs inconsistentes

---

## 9. Melhorias Futuras

- Permitir ao presidente configurar o comportamento da herança (herdar ou não)
- Lista de **convidados fixos do grupo**, que sempre aparecem em novas partidas automaticamente
- Notificação via WhatsApp quando uma nova partida for criada pela recorrência
