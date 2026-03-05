# PRD — Recorrência com Herança de Convidados

**Status:** Pendente de implementação
**Data:** 2026-03-04

---

## 1. Visão Geral

Partidas criadas automaticamente pela regra de recorrência devem **herdar a lista de convidados da última partida válida do grupo**.

Os convidados herdados deverão **confirmar presença novamente**, respeitando as regras de confirmação já existentes no sistema.

Essa abordagem garante:

- consistência entre partidas
- menos trabalho manual para o presidente
- manutenção do mesmo grupo de jogadores recorrentes

---

## 2. Comportamento Esperado

Quando o sistema criar uma nova partida automaticamente:

1. A partida é criada baseada na última partida existente do grupo.
2. O sistema copia os **convidados (attendances) da última partida**.
3. Todos os convidados iniciam com status `pending`.

Ou seja:

- ninguém nasce confirmado automaticamente
- todos precisam confirmar novamente

---

## 3. Regras de Negócio

### RN05 — Herança de Convidados

Partidas geradas automaticamente devem copiar a lista de convidados (`attendances`) da última partida do grupo.

Não copiar:

- status de presença (sempre inicia como `pending`)
- histórico de confirmações anteriores

### RN06 — Status Inicial

Todos os registros de `Attendance` criados pela herança iniciam com:

```
status = pending
```

Valores possíveis de `AttendanceStatus` no sistema atual:

| Valor       | Descrição                     |
|-------------|-------------------------------|
| `pending`   | Aguardando confirmação        |
| `confirmed` | Presença confirmada           |
| `declined`  | Jogador não vai comparecer    |

> Não existe status `waitlist` no sistema atual. Caso seja implementado no futuro, deve ser tratado como `pending` na herança.

### RN07 — Limite de Jogadores

As regras já existentes de limite de jogadores (`max_players` na tabela `matches`) continuam válidas. A confirmação segue o fluxo normal após a herança.

### RN08 — Convidados Removidos

A base para herança é **a lista atual de `attendances` da última partida válida**. Se um jogador foi removido manualmente da última partida pelo presidente, ele não será herdado.

---

## 4. Modelo de Dados Relevante

### Tabela `matches`

| Campo            | Tipo            | Descrição                          |
|------------------|-----------------|------------------------------------|
| `id`             | UUID            | Identificador                      |
| `group_id`       | UUID (FK)       | Grupo ao qual pertence             |
| `match_date`     | Date            | Data da partida                    |
| `start_time`     | Time            | Horário de início                  |
| `end_time`       | Time (nullable) | Horário de término                 |
| `location`       | String(200)     | Local                              |
| `address`        | String(300)     | Endereço completo (opcional)       |
| `court_type`     | Enum (nullable) | campo / sintetico / terrao / quadra|
| `players_per_team` | SmallInt      | Jogadores por time (opcional)      |
| `max_players`    | SmallInt        | Limite de jogadores (opcional)     |
| `notes`          | Text            | Observações (opcional)             |
| `hash`           | String(12)      | Hash público para compartilhamento |
| `status`         | Enum            | `open` / `closed`                  |
| `created_by_id`  | UUID (FK)       | Jogador que criou (nullable)       |

### Tabela `attendances`

| Campo        | Tipo      | Descrição                              |
|--------------|-----------|----------------------------------------|
| `id`         | UUID      | Identificador                          |
| `match_id`   | UUID (FK) | Partida                                |
| `player_id`  | UUID (FK) | Jogador                                |
| `status`     | Enum      | `pending` / `confirmed` / `declined`   |
| `updated_at` | Timestamp | Última atualização                     |

Constraint: `UNIQUE(match_id, player_id)` — um jogador por partida.

---

## 5. Fluxo de Criação Automática (Cron)

```
para cada grupo com recorrência ativa:

    ultima_partida = buscar_ultima_partida(grupo.id)

    proxima_data = ultima_partida.match_date + 7 dias

    se partida_ja_existe(grupo.id, proxima_data):
        ignorar

    nova_partida = criar_partida(
        group_id       = grupo.id,
        match_date     = proxima_data,
        start_time     = ultima_partida.start_time,
        end_time       = ultima_partida.end_time,
        location       = ultima_partida.location,
        address        = ultima_partida.address,
        court_type     = ultima_partida.court_type,
        players_per_team = ultima_partida.players_per_team,
        max_players    = ultima_partida.max_players,
        notes          = ultima_partida.notes,
        status         = "open",
        created_by_id  = None  # gerada automaticamente
    )

    convidados = buscar_attendances(ultima_partida.id)

    para cada convidado em convidados:
        inserir_attendance(
            match_id  = nova_partida.id,
            player_id = convidado.player_id,
            status    = "pending"
        )
```

---

## 6. Exemplo de Funcionamento

### Partida da semana atual

Quarta 20h

| Jogador | Status      |
|---------|-------------|
| João    | `confirmed` |
| Pedro   | `confirmed` |
| Lucas   | `declined`  |
| Carlos  | `confirmed` |

### Nova partida criada automaticamente (semana seguinte)

Quarta 20h

| Jogador | Status    |
|---------|-----------|
| João    | `pending` |
| Pedro   | `pending` |
| Lucas   | `pending` |
| Carlos  | `pending` |

---

## 7. Pré-requisitos

Esta funcionalidade depende da implementação de **recorrência de partidas**, que ainda não existe no sistema. Os itens necessários antes desta feature:

- [ ] Campo `recurrence` (ou similar) no modelo `Group` ou `Match`
- [ ] Job/cron de criação automática de partidas

---

## 8. Benefícios

- Elimina recriação manual da lista de convidados
- Mantém o mesmo grupo recorrente sem esforço do presidente
- Preserva a dinâmica de confirmação a cada partida

---

## 9. Melhorias Futuras

Permitir ao presidente configurar o comportamento da herança:

```
Herdar convidados automaticamente?
( ) Sim — copia lista da última partida (padrão)
( ) Não — nova partida começa sem convidados
```

Ou ainda, a criação de uma lista de **convidados fixos do grupo**, que sempre aparecem em todas as partidas automaticamente.
