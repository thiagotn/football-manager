# PRD — Configuração de Votação por Grupo
## Rachao.app · Flexibilidade nas Regras de Votação Pós-Partida

| | |
|---|---|
| **Versão** | 1.1 |
| **Status** | ✅ Implementado · Março de 2026 |
| **Data** | Março de 2026 |
| **Plataforma** | https://rachao.app |

---

## Estado de Implementação

### ✅ Implementado (Março 2026)

#### Backend
- **`migrations/018_group_voting_config.sql`**: colunas `vote_open_delay_minutes` (0–120, default 20) e `vote_duration_hours` (2–72, default 24) na tabela `groups`, com constraints CHECK.
- **`migrations/019_match_voting_snapshot.sql`**: mesmas colunas em `matches` (snapshot no momento da criação, default 20/24).
- **`app/models/group.py`** e **`app/models/match.py`**: campos adicionados aos modelos SQLAlchemy.
- **`app/schemas/group.py`**: campos em `GroupCreate` (`ge=0, le=120` e `ge=2, le=72`), `GroupUpdate` (opcionais) e `GroupResponse`.
- **`app/schemas/vote.py`**: campo `vote_open_delay_minutes` adicionado a `VoteStatusResponse`.
- **`app/services/voting.py`**: constantes globais `VOTING_OPEN_DELAY` e `VOTING_DURATION` removidas. `voting_window(match)` lê `match.vote_open_delay_minutes` e `match.vote_duration_hours` via `getattr` com fallback para os valores padrão.
- **`app/api/v1/routers/votes.py`**: SQL de `GET /votes/pending` atualizado para usar as colunas dinâmicas da partida (`(m.vote_open_delay_minutes || ' minutes')::interval`). Endpoint de status inclui `vote_open_delay_minutes` na resposta.
- **`app/api/v1/routers/groups.py`**: criação de grupo passa os configs de votação; `get_group` inclui os novos campos na resposta.
- **`app/api/v1/routers/matches.py`**: criação de partida faz snapshot dos configs do grupo (`vote_open_delay_minutes`, `vote_duration_hours`).
- **`app/core/config.py`**: campos de billing Stripe adicionados ao `Settings` para evitar erro de validação do Pydantic com variáveis de ambiente já presentes.

#### Frontend
- **`src/lib/api.ts`**: tipo `Group` e `VoteStatusResponse` atualizados; métodos `groups.create` e `groups.update` aceitam os novos campos.
- **`src/routes/groups/new/+page.svelte`**: seção "Configurações de votação" com selects para atraso (0/10/20/30/60 min) e duração (2/4/6/12/24/48/72 h).
- **`src/routes/groups/[id]/+page.svelte`**:
  - Modal "Editar Grupo": mesmos selects, pré-preenchidos com os valores atuais do grupo.
  - Header do grupo: badge `🗳️ Votação: +Xmin · Yh` visível apenas para admin do grupo e super admin.
- **`src/routes/match/[hash]/+page.svelte`**: mensagem dinâmica baseada em `voteStatus.vote_open_delay_minutes` — exibe "Imediatamente após o término" quando 0, ou "X min após o término" caso contrário.
- **`src/routes/lp/+page.svelte`**: texto atualizado para "A votação abre automaticamente após o término da partida — você define quando." (sem mencionar valor fixo).

---

## 1. Contexto e Motivação

As regras de votação pós-partida eram **globais e fixas** em toda a plataforma:

- A votação **abria 20 minutos** após o encerramento da partida.
- A votação **encerrava 24 horas** após ser aberta.

Esses valores estavam hardcoded em `football-api/app/services/voting.py` e replicados em queries SQL brutas no router de votos, além de serem citados explicitamente na landing page e na página de partida do frontend.

Grupos diferentes têm dinâmicas distintas: alguns preferem que a votação abra imediatamente após a partida, outros querem dar mais tempo para todos votarem. Tornar essas configurações ajustáveis por grupo aumenta a flexibilidade sem adicionar complexidade para quem não precisa customizar.

---

## 2. Objetivo

Permitir que o administrador do grupo configure, no momento da criação do grupo e nas configurações do grupo, os seguintes parâmetros de votação:

1. **Atraso para abertura da votação** — quantos minutos após o término da partida a votação abre.
2. **Duração da votação** — por quantas horas a votação permanece aberta.

---

## 3. Requisitos Funcionais

### RF-01 — Configuração na criação do grupo ✅

O formulário de criação de grupo (`/groups/new`) exibe uma seção **"Configurações de votação"** com dois campos `<select>`:

| Campo | Opções | Padrão | Restrições (backend) |
|---|---|---|---|
| Atraso para abertura | 0 min · 10 min · 20 min · 30 min · 60 min | 20 min | 0 a 120 min |
| Duração da votação | 2 h · 4 h · 6 h · 12 h · 24 h · 48 h · 72 h | 24 h | 2 a 72 h |

### RF-02 — Edição nas configurações do grupo ✅

O modal "Editar Grupo" em `/groups/[id]` exibe os mesmos selects, pré-preenchidos com os valores atuais do grupo. A alteração afeta apenas **partidas criadas após a mudança** — partidas existentes mantêm o snapshot salvo no momento da criação (RF-06).

### RF-03 — Exibição da configuração nos detalhes do grupo ✅

No header da página do grupo, admins do grupo e super admins visualizam um badge resumido com as configurações atuais:

```
🗳️ Votação: +20min · 24h
```

Quando o atraso é zero, exibe `imediata` em lugar de `+0min`. Membros comuns não veem essa informação.

### RF-04 — Mensagem dinâmica na página da partida ✅

A mensagem exibida quando a votação ainda não abriu usa o valor retornado pela API (`vote_open_delay_minutes` em `VoteStatusResponse`):

- Atraso = 0 → `"Imediatamente após o término da partida"`
- Atraso = X min → `"X min após o término da partida"`

### RF-05 — Landing page ✅

A menção fixa `"20 minutos após o término da partida"` foi substituída por:

> "A votação abre automaticamente após o término da partida — você define quando. Todos os confirmados são notificados."

### RF-06 — Snapshot dos valores na partida ✅

No momento da criação de cada partida, os valores vigentes do grupo (`vote_open_delay_minutes`, `vote_duration_hours`) são copiados para a partida. O cálculo de `voting_window()` usa os valores da própria partida, garantindo que alterações futuras nas configurações do grupo não afetem partidas já criadas.

### RF-07 — Backend: cálculo dinâmico ✅

A função `voting_window(match)` em `voting.py` usa `getattr(match, 'vote_open_delay_minutes', 20)` e `getattr(match, 'vote_duration_hours', 24)`. As constantes globais `VOTING_OPEN_DELAY` e `VOTING_DURATION` foram removidas.

A query SQL de `GET /votes/pending` usa as colunas dinâmicas da partida:
```sql
+ (m.vote_open_delay_minutes || ' minutes')::interval
+ (m.vote_duration_hours || ' hours')::interval
```

---

## 4. Modelagem de Dados

### Migration 018 — configurações no grupo

```sql
-- Migration: 018_group_voting_config.sql

ALTER TABLE groups
  ADD COLUMN vote_open_delay_minutes INT NOT NULL DEFAULT 20,
  ADD COLUMN vote_duration_hours     INT NOT NULL DEFAULT 24;

ALTER TABLE groups
  ADD CONSTRAINT chk_vote_open_delay
    CHECK (vote_open_delay_minutes >= 0 AND vote_open_delay_minutes <= 120);

ALTER TABLE groups
  ADD CONSTRAINT chk_vote_duration
    CHECK (vote_duration_hours >= 2 AND vote_duration_hours <= 72);
```

### Migration 019 — snapshot na partida

```sql
-- Migration: 019_match_voting_snapshot.sql

ALTER TABLE matches
  ADD COLUMN vote_open_delay_minutes INT NOT NULL DEFAULT 20,
  ADD COLUMN vote_duration_hours     INT NOT NULL DEFAULT 24;
```

---

## 5. Impacto em Dados Existentes

Grupos existentes recebem os valores padrão (`20` e `24`) via `DEFAULT` na migration — comportamento idêntico ao anterior, sem breaking change.

Partidas existentes recebem os mesmos valores padrão via `DEFAULT`, garantindo continuidade do comportamento histórico.

---

## 6. Critérios de Aceitação

- [x] Formulário de criação de grupo exibe campos de configuração de votação com valores padrão pré-selecionados.
- [x] Modal de edição do grupo exibe os campos pré-preenchidos com os valores atuais.
- [x] Admin do grupo e super admin veem badge de configuração no header do grupo.
- [x] Ao criar um grupo com atraso de 0 min, a votação abre imediatamente após encerrar a partida.
- [x] Ao criar um grupo com duração de 2 h (mínimo), a votação permanece aberta por 2 horas.
- [x] Alterar as configurações do grupo não afeta partidas já criadas (snapshot na partida).
- [x] A página de partida exibe o atraso correto configurado ("Imediatamente" ou "X min após o término").
- [x] A landing page não menciona mais o valor fixo de "20 minutos".
- [x] A API valida e rejeita valores fora dos ranges permitidos com erro `422`.
- [x] Grupos existentes continuam funcionando com o comportamento anterior (default = 20 min / 24 h).

---

## 7. Fora de Escopo (v1)

- Configuração de votação por **partida** individualmente (apenas por grupo).
- Desativar completamente a votação em um grupo.
- Configurar horário fixo de abertura da votação (ex: sempre às 22h do dia da partida).
- Notificações customizadas por grupo para abertura da votação.

---

*Documento elaborado para uso interno da equipe de produto e engenharia do Rachao.app.*
