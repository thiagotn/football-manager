# PRD — Configuração de Votação por Grupo
## Rachao.app · Flexibilidade nas Regras de Votação Pós-Partida

| | |
|---|---|
| **Versão** | 1.0 |
| **Status** | Aguardando implementação |
| **Data** | Março de 2026 |
| **Plataforma** | https://rachao.app |

---

## 1. Contexto e Motivação

Atualmente, as regras de votação pós-partida são **globais e fixas** em toda a plataforma:

- A votação **abre 20 minutos** após o encerramento da partida.
- A votação **encerra 24 horas** após ser aberta.

Esses valores estão hardcoded em `football-api/app/services/voting.py` e replicados em queries SQL brutas no router de votos, além de serem citados explicitamente na landing page e na página de partida do frontend.

Grupos diferentes têm dinâmicas distintas: alguns preferem que a votação abra imediatamente após a partida, outros querem dar mais tempo para todos votarem. Tornar essas configurações ajustáveis por grupo aumenta a flexibilidade sem adicionar complexidade para quem não precisa customizar.

---

## 2. Objetivo

Permitir que o administrador do grupo configure, no momento da criação do grupo (e posteriormente nas configurações do grupo), os seguintes parâmetros de votação:

1. **Atraso para abertura da votação** — quantos minutos após o término da partida a votação abre.
2. **Duração da votação** — por quantas horas a votação permanece aberta.

---

## 3. Levantamento de Impacto (Estado Atual)

### 3.1 Backend — onde os valores estão hardcoded

| Arquivo | Linha | Trecho |
|---|---|---|
| `football-api/app/services/voting.py` | 9 | `VOTING_OPEN_DELAY = timedelta(minutes=20)` |
| `football-api/app/services/voting.py` | 10 | `VOTING_DURATION = timedelta(hours=24)` |
| `football-api/app/services/voting.py` | 13–19 | Função `voting_window(match)` — calcula `opens_at` e `closes_at` |
| `football-api/app/api/v1/routers/votes.py` | 152 | SQL: `interval '24 hours 20 minutes'` |
| `football-api/app/api/v1/routers/votes.py` | 166 | SQL: `interval '20 minutes'` |
| `football-api/app/api/v1/routers/votes.py` | 170 | SQL: `interval '24 hours 20 minutes'` |

### 3.2 Frontend — onde os valores são exibidos ou comentados

| Arquivo | Linha | Trecho |
|---|---|---|
| `football-frontend/src/routes/lp/+page.svelte` | 196 | `"20 minutos após o término da partida, a votação abre sozinha..."` |
| `football-frontend/src/routes/match/[hash]/+page.svelte` | 607 | `"⏳ 20 min após o término da partida"` |

### 3.3 Banco de dados — tabelas a alterar

| Tabela | Situação |
|---|---|
| `groups` | Recebe duas novas colunas de configuração |
| `matches` | Sem alteração — os valores são lidos do grupo no momento do cálculo |

---

## 4. Requisitos Funcionais

### RF-01 — Configuração na criação do grupo

O formulário de criação de grupo (`/groups/new`) deve exibir uma seção **"Configurações de votação"** com dois campos:

| Campo | Tipo | Padrão | Restrições |
|---|---|---|---|
| Atraso para abertura da votação | Seleção ou numérico (minutos) | 20 min | 0 a 120 min |
| Duração da votação | Seleção ou numérico (horas) | 24 h | 1 a 72 h |

O campo de atraso pode exibir opções pré-definidas: **0 min (imediato)**, **10 min**, **20 min (padrão)**, **30 min**, **60 min** e uma opção "Personalizado" com input numérico livre dentro das restrições.

O campo de duração pode exibir opções: **12 h**, **24 h (padrão)**, **48 h**, **72 h** e "Personalizado".

### RF-02 — Edição nas configurações do grupo

A página de configurações/edição do grupo deve permitir alterar esses parâmetros a qualquer momento. A alteração afeta apenas as **partidas criadas após a mudança** — partidas já existentes mantêm o cálculo baseado nas configurações vigentes no momento da criação.

> **Nota de implementação:** como o cálculo de votação é feito dinamicamente a partir do objeto `match` e das configurações do grupo, e não é persistido na partida, a alteração afetará partidas futuras e também as partidas existentes ainda não encerradas. Avaliar se esse comportamento é desejado ou se deve-se armazenar os valores no momento da criação da partida. Recomendação: armazenar na partida (ver RF-06).

### RF-03 — Exibição na página da partida

A mensagem exibida quando a votação ainda não abriu (`"⏳ 20 min após o término da partida"`) deve ser dinâmica, refletindo o valor configurado no grupo ao qual a partida pertence.

Exemplo: se o grupo configurou atraso de 0 minutos, exibir `"⏳ A votação abre imediatamente após o término da partida"`. Se 30 minutos, exibir `"⏳ 30 min após o término da partida"`.

### RF-04 — Landing page

A menção fixa `"20 minutos após o término da partida"` na `/lp` deve ser ajustada para refletir que esse valor é configurável. Sugestão de novo texto: `"Logo após o término da partida, a votação abre sozinha para todos que confirmaram presença."` ou `"A votação abre automaticamente — você define quando."`.

### RF-05 — Backend: cálculo dinâmico por grupo

A função `voting_window(match)` em `voting.py` deve receber ou acessar as configurações do grupo associado à partida, substituindo as constantes globais pelos valores do grupo:

```python
# Antes (hardcoded):
VOTING_OPEN_DELAY = timedelta(minutes=20)
VOTING_DURATION   = timedelta(hours=24)

# Depois (por grupo):
open_delay = timedelta(minutes=match.group.vote_open_delay_minutes)
duration   = timedelta(hours=match.group.vote_duration_hours)
```

As queries SQL brutas em `votes.py` que usam `interval '20 minutes'` e `interval '24 hours 20 minutes'` devem ser refatoradas para usar os valores do grupo dinamicamente, ou a lógica deve ser movida para Python (recomendado para manutenibilidade).

### RF-06 — Snapshot dos valores na partida (recomendado)

Para garantir que mudanças nas configurações do grupo não afetem retroativamente partidas já criadas, recomenda-se adicionar à tabela `matches` as colunas `vote_open_delay_minutes` e `vote_duration_hours`, preenchidas no momento da criação da partida com os valores vigentes do grupo. O cálculo de `voting_window()` usaria então os valores da própria partida.

> Esta abordagem é mais robusta e consistente. É o comportamento esperado pelo usuário: a votação de uma partida passada não deve mudar se o administrador alterar as configurações do grupo depois.

---

## 5. Modelagem de Dados

### 5.1 Migration — configurações no grupo

```sql
-- Migration: 0XX_group_voting_config.sql

ALTER TABLE groups
  ADD COLUMN vote_open_delay_minutes INT NOT NULL DEFAULT 20,
  ADD COLUMN vote_duration_hours     INT NOT NULL DEFAULT 24;

-- Constraint: delay entre 0 e 120 minutos
ALTER TABLE groups
  ADD CONSTRAINT chk_vote_open_delay
    CHECK (vote_open_delay_minutes >= 0 AND vote_open_delay_minutes <= 120);

-- Constraint: duração entre 1 e 72 horas
ALTER TABLE groups
  ADD CONSTRAINT chk_vote_duration
    CHECK (vote_duration_hours >= 1 AND vote_duration_hours <= 72);
```

### 5.2 Migration — snapshot na partida (RF-06, recomendado)

```sql
-- Migration: 0XX_match_voting_snapshot.sql

ALTER TABLE matches
  ADD COLUMN vote_open_delay_minutes INT,
  ADD COLUMN vote_duration_hours     INT;

-- Preencher retroativamente com os valores padrão
UPDATE matches SET
  vote_open_delay_minutes = 20,
  vote_duration_hours     = 24
WHERE vote_open_delay_minutes IS NULL;

-- Tornar NOT NULL após o backfill
ALTER TABLE matches
  ALTER COLUMN vote_open_delay_minutes SET NOT NULL,
  ALTER COLUMN vote_duration_hours     SET NOT NULL;
```

> Nas próximas migrations, seguir a numeração sequencial do projeto (atualmente em `017_*`).

---

## 6. Endpoints da API

### 6.1 Criação de grupo — campos novos

`POST /api/v1/groups`

```json
{
  "name": "Rachão da Sexta",
  "description": "...",
  "vote_open_delay_minutes": 20,
  "vote_duration_hours": 24
}
```

Valores omitidos assumem o padrão (`20` e `24`). Validação de range no schema Pydantic.

### 6.2 Edição de grupo — campos novos

`PATCH /api/v1/groups/{group_id}`

Mesmos campos, opcionais. Retorna o grupo atualizado.

### 6.3 Resposta do grupo

`GET /api/v1/groups/{group_id}` deve incluir os novos campos na resposta para que o frontend possa exibi-los nas configurações.

---

## 7. Alterações de Frontend

### 7.1 `/groups/new` — formulário de criação

Adicionar seção "Configurações de votação" ao final do formulário, com os dois campos (atraso e duração). Valores padrão pré-selecionados.

Sugestão de UX mobile-first:
- Usar `<select>` com opções pré-definidas + opção "Personalizado" que revela um `<input type="number">`.
- Exibir tooltip/helper text explicando o funcionamento: *"A votação abre automaticamente X minutos após o término da partida e fica disponível por Y horas."*

### 7.2 Página de configurações do grupo

Adicionar os mesmos campos na tela de edição do grupo. Exibir os valores atuais pré-preenchidos.

### 7.3 `match/[hash]/+page.svelte`

A mensagem de votação pendente (`"⏳ 20 min após o término da partida"`) deve usar o valor retornado pela API de status da votação (`GET /matches/{match_id}/votes/status`), que já retorna `opens_at`. Calcular a diferença entre `opens_at` e o horário de encerramento da partida para exibir o atraso correto, ou incluir `vote_open_delay_minutes` diretamente na resposta de status.

### 7.4 `lp/+page.svelte`

Substituir a menção hardcoded `"20 minutos após o término da partida"` por texto genérico que destaque a automaticidade do processo sem fixar o valor.

**Texto atual:**
> "20 minutos após o término da partida, a votação abre sozinha para todos que confirmaram presença."

**Texto proposto:**
> "A votação abre automaticamente após o término da partida — você define quando. Todos os jogadores confirmados são notificados."

---

## 8. Alterações de Backend

### 8.1 `app/services/voting.py`

- Remover constantes globais `VOTING_OPEN_DELAY` e `VOTING_DURATION`.
- Atualizar `voting_window(match)` para ler `match.vote_open_delay_minutes` e `match.vote_duration_hours`.
- Atualizar `voting_status(match)` — sem alteração na lógica, apenas depende de `voting_window()`.

### 8.2 `app/api/v1/routers/votes.py`

- Refatorar a query SQL de `GET /votes/pending` que usa `interval '20 minutes'` e `interval '24 hours 20 minutes'` para parametrizar os valores por grupo/partida. Opção mais simples: mover a lógica de filtragem de janela de votação para Python após buscar as partidas elegíveis, usando `voting_window()`.

### 8.3 `app/db/repositories/group_repo.py`

- Incluir os novos campos no `INSERT` de criação e no `UPDATE` de edição.

### 8.4 `app/schemas/group.py`

- Adicionar campos `vote_open_delay_minutes: int = 20` e `vote_duration_hours: int = 24` nos schemas de criação (`GroupCreate`), edição (`GroupUpdate`) e resposta (`GroupResponse`).
- Incluir validação de range (`ge=0, le=120` e `ge=1, le=72`).

### 8.5 `app/api/v1/routers/matches.py` (criação de partida)

- Ao criar uma partida, copiar `vote_open_delay_minutes` e `vote_duration_hours` do grupo para a partida (snapshot — RF-06).

---

## 9. Impacto em Dados Existentes

Grupos existentes receberão os valores padrão (`vote_open_delay_minutes = 20`, `vote_duration_hours = 24`) via `DEFAULT` na migration — comportamento idêntico ao atual, sem breaking change.

Partidas existentes (snapshot retroativo) serão preenchidas com `20` e `24` via `UPDATE` na migration.

---

## 10. Critérios de Aceitação

- [ ] Formulário de criação de grupo exibe campos de configuração de votação com valores padrão pré-selecionados.
- [ ] Ao criar um grupo com atraso de 0 min, a votação abre imediatamente após encerrar a partida.
- [ ] Ao criar um grupo com duração de 48 h, a votação permanece aberta por 48 horas.
- [ ] Alterar as configurações do grupo não afeta partidas já encerradas (snapshot na partida).
- [ ] A página de partida exibe o atraso correto configurado no grupo ("⏳ X min após o término").
- [ ] A landing page não menciona mais o valor fixo de "20 minutos".
- [ ] A API valida e rejeita valores fora dos ranges permitidos com erro `422`.
- [ ] Grupos existentes continuam funcionando com o comportamento anterior (default = 20 min / 24 h).

---

## 11. Fora de Escopo (v1)

- Configuração de votação por **partida** individualmente (apenas por grupo).
- Desativar completamente a votação em um grupo.
- Configurar horário fixo de abertura da votação (ex: sempre às 22h do dia da partida).
- Notificações customizadas por grupo para abertura da votação.

---

*Documento elaborado para uso interno da equipe de produto e engenharia do Rachao.app.*
