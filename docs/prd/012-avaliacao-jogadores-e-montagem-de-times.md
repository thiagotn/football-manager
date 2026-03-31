# PRD — Avaliação de Jogadores e Montagem de Times
## Rachao.app · Notas por grupo, posição de goleiro e sorteio equilibrado de times

| | |
|---|---|
| **Versão** | 1.0 |
| **Status** | ✅ Implementado — Março 2026 |
| **Data** | Março de 2026 |
| **Plataforma** | https://rachao.app |

---

## 1. Contexto e Motivação

Atualmente, todos os jogadores entram na plataforma sem nenhuma informação sobre nível técnico ou posição. A montagem de times é feita manualmente pelo administrador do grupo, sem suporte da ferramenta.

Esta feature introduz três capacidades encadeadas:

1. **Nota de habilidade por grupo:** cada jogador possui uma nota de 1 a 5 estrelas dentro de cada grupo, atribuída e editável pelo administrador do grupo.
2. **Flag de goleiro por grupo:** o administrador pode marcar se um jogador é goleiro naquele grupo.
3. **Sorteio equilibrado de times:** com base nas notas e no flag de goleiro dos confirmados, o sistema monta times equilibrados automaticamente e os exibe em uma página pública dedicada.

---

## 2. Modelo de Dados Atual (referência)

| Tabela | Campos relevantes |
|---|---|
| `players` | `id`, `name`, `nickname`, `role` (global: admin/player) |
| `group_members` | `id`, `group_id`, `player_id`, `role` (admin/member no grupo) |
| `matches` | `id`, `group_id`, `hash`, `players_per_team`, `status` |
| `attendances` | `match_id`, `player_id`, `status` (pending/confirmed/declined) |

> **Decisão de design:** nota e flag de goleiro devem ser armazenados em `group_members`, não em `players`. Um jogador pode ter habilidades e posições diferentes em grupos distintos (ex: goleiro no rachão de quinta, meia no de sexta). Essa granularidade por grupo é a correta.

---

## 3. Requisitos Funcionais

### Bloco A — Nota e Flag de Goleiro

#### RF-01 — Nota padrão ao aceitar convite

Quando um jogador aceita um convite e é adicionado a um grupo (via `POST /api/v1/invites/{token}/accept`), o registro em `group_members` deve ser criado com `skill_stars = 2` por padrão.

O mesmo se aplica ao fluxo de auto-cadastro público (`POST /api/v1/auth/register`) quando associado a um grupo via convite.

#### RF-02 — Nota padrão ao adicionar membro diretamente

Quando um administrador adiciona um membro diretamente ao grupo (se essa rota existir ou for criada), o membro também entra com `skill_stars = 2`.

#### RF-03 — Edição de nota pelo admin do grupo

Na visão de membros do grupo (página `/groups/[id]`), o administrador do grupo deve poder editar a nota de cada membro (1 a 5 estrelas) e marcar/desmarcar o flag de goleiro.

- A edição deve ser inline ou via modal, sem redirecionar para outra página.
- O campo de nota deve usar um seletor visual de estrelas (1–5).
- O flag de goleiro deve ser um toggle/checkbox com label "Goleiro".
- Apenas o `GroupMember` com `role = 'admin'` no grupo pode editar esses campos.
- Administradores globais (`Player.role = 'admin'`) também têm permissão.

#### RF-04 — Exibição da nota na listagem de membros

Na listagem de membros do grupo, exibir para o administrador:
- A nota do jogador (estrelas cheias/vazias).
- Indicador visual de goleiro (ex: ícone de luva ou badge "GK").

Para membros comuns, **não exibir** a nota nem o flag de goleiro — essa informação é interna do administrador.

---

### Bloco B — Montagem de Times

> *(especificações originais mantidas — ver abaixo)*

---

### Bloco C — Página Pública de Resultados de Votação

#### RF-10 — Link de acesso aos resultados na página da partida

Na página da partida (`/match/[hash]`), quando o status de votação for `closed` (período encerrado) **e** houver pelo menos um voto registrado, exibir um card/botão de destaque com link para `/match/[hash]/results`.

- Visível para todos (logados e não logados).
- Texto sugerido: **"🏆 Ver os melhores do Rachão"**.
- O link usa o mesmo `hash` da partida — não é necessário nenhum token adicional.

#### RF-11 — Página pública de resultados (`/match/[hash]/results`)

Nova rota pública acessível sem autenticação, com URL compartilhável via WhatsApp ou qualquer canal.

**Conteúdo obrigatório:**

1. **Header da partida**: nome do grupo, número da partida (`#N`), data e local — mesmo estilo visual do header de `/match/[hash]`.
2. **Pódio dos top 3**: apresentação visual em destaque para os três primeiros colocados. Layout clássico de pódio (2º à esquerda, 1º ao centro elevado, 3º à direita). Cada posição exibe: medalha/emoji de posição (🥇🥈🥉), apelido ou nome do jogador, pontuação de votos.
3. **Ranking completo**: lista ordenada por pontuação (maior → menor) exibindo todos os jogadores que receberam votos. Cada linha: posição numérica, nome/apelido, barra de progresso proporcional ao líder, pontuação.
4. **Rodapé de contexto**: quantidade de votantes (`X de Y jogadores votaram`).
5. **Botão "Compartilhar"**: copia URL para clipboard, disponível para todos.
6. **Link "← Voltar para a partida"**: retorna para `/match/[hash]`.

**Estados da página:**

| Condição | Comportamento |
|---|---|
| Votação ainda aberta (`status = 'open'`) | Exibe mensagem *"A votação ainda está em andamento."* com link para a partida |
| Votação encerrada, sem votos | Exibe mensagem *"Nenhum voto foi registrado nesta partida."* |
| Votação encerrada, com votos | Exibe pódio + ranking completo |
| Partida não encontrada | Redireciona para página de erro |

#### RF-12 — Layout responsivo da página de resultados

**Mobile (< 640px):**
- Header da partida em largura total.
- Pódio ocupa largura total: três colunas de tamanho fixo centralizadas, com o 1º lugar visualmente mais alto (via `padding-top` ou `height` maior).
- Cada coluna do pódio: medalha emoji (grande, ~2rem), nome truncado em 1 linha, pontuação em destaque.
- Ranking: lista vertical de cards compactos com barra de progresso.
- Botão "Compartilhar" fixo ou em destaque abaixo do ranking.

**Desktop (≥ 640px):**
- Max-width `max-w-2xl` centralizado.
- Pódio com colunas mais largas, espaçamento generoso, fontes maiores.
- Ranking em tabela ou lista com hover state.
- Layout de duas colunas é fora de escopo para v1 — foco em hierarquia vertical clara.

#### RF-13 — Open Graph / preview no WhatsApp

A página deve ter meta tags `og:title`, `og:description` e `og:image` (ou fallback para a imagem padrão do site) para que ao compartilhar o link no WhatsApp apareça uma preview informativa.

- `og:title`: `"🏆 Resultados do Rachão #N — [Nome do Grupo]"`
- `og:description`: `"[Apelido do 1º colocado] foi o melhor do rachão! Veja o ranking completo."`
- A rota deve usar `+page.server.ts` (com SSR) para renderizar essas meta tags no servidor.

---

#### RF-05 — Botão "Montar times" na lista de confirmados da partida

Na página da partida (`/match/[hash]`), na seção de jogadores confirmados, o administrador do grupo deve ver um botão **"Montar times"**. Jogadores comuns não veem essa opção.

O botão só fica ativo quando:
- A partida tem `status = 'open'` ou `'in_progress'`.
- Há pelo menos `(players_per_team + 1) * 2` jogadores confirmados (mínimo para 2 times completos).
- O campo `players_per_team` está definido na partida.

#### RF-06 — Algoritmo de montagem de times equilibrados

Ao clicar em "Montar times", o backend executa o sorteio com base nos jogadores com `attendance.status = 'confirmed'` naquela partida.

**Inputs do algoritmo:**
- Lista de jogadores confirmados com `skill_stars` e `is_goalkeeper` (lidos de `group_members`).
- `players_per_team` da partida = número de **jogadores de linha** por time (exclui goleiro).
- **Tamanho real de cada time** = `players_per_team + 1` (linha + 1 goleiro ou substituto).
- Número de times = `floor(total_confirmados / (players_per_team + 1))`. Jogadores excedentes ficam como "reservas".

**Lógica de balanceamento:**
1. Separar goleiros dos demais jogadores.
2. Distribuir **um goleiro por time** (se houver goleiros suficientes). Goleiros excedentes entram no pool geral.
3. Para os jogadores restantes (não-goleiros ou goleiros excedentes), aplicar o algoritmo de balanceamento:
   - Ordenar jogadores por `skill_stars` (maior para menor).
   - Distribuir em "serpentina" (snake draft): time 1 → time 2 → ... → time N → time N → ... → time 1, repetindo até distribuir todos.
   - Essa abordagem garante que as somas de estrelas de cada time sejam as mais próximas possíveis.
4. Jogadores sem nota configurada tratados como `skill_stars = 2` (padrão).
5. Jogadores excedentes (quando `total % (players_per_team + 1) != 0`) ficam como lista de **reservas**, sem time atribuído.

#### RF-07 — Persistência dos times gerados

Os times gerados são salvos no banco de dados associados à partida. Uma nova chamada ao endpoint de montagem **sobrescreve** os times anteriores.

Cada time recebe:
- Um **nome gerado automaticamente** a partir de um conjunto pré-definido (ver seção 4).
- Uma **cor** opcional para identificação visual (pode ser um conjunto de cores pré-definidas rotacionadas).

#### RF-08 — Página pública de times (`/match/[hash]/teams`)

Os times gerados são exibidos em uma página pública (sem autenticação obrigatória), acessível via:

```
/match/[hash]/teams
```

A página exibe:
- **Card "Primeiro jogo do rachão"** (acima dos times): destaque com os nomes e cores dos dois primeiros times sorteados (posição 1 × posição 2), indicando qual será a primeira partida. Visível para todos.
- Nome de cada time e sua lista de jogadores (nome/apelido + indicador de goleiro).
- Soma de estrelas de cada time (visível apenas para o admin do grupo).
- Lista de reservas (se houver).
- Botão **"Remontar times"** — visível apenas para o admin do grupo, que ao clicar executa novo sorteio e atualiza a página.
- Botão de compartilhamento (copia URL para clipboard).

Se nenhum time foi gerado ainda, a página exibe mensagem informativa e, para o admin, o botão "Montar times".

#### RF-09 — Link para a página de times na partida

Na página da partida (`/match/[hash]`), após os times serem gerados pela primeira vez, exibir um link/card de acesso para `/match/[hash]/teams` visível para todos os participantes (não apenas o admin).

---

## 4. Nomes de Times (Geração Automática)

Os nomes são sorteados aleatoriamente no momento da geração. O conjunto deve cobrir pelo menos 30 opções para evitar repetição frequente em grupos ativos.

**Conjunto de nomes sugeridos — estilo várzea brasileira com humor:**

```
Real Madruga, Barcelusa, Barsemlona, Meia Boca Juniors,
Baile de Munique, Varmeiras, Atecubanos FC, Inter de Limão,
Manchester Cachaça, Real Matismo, Paysanduba, Horriver Plate,
Patético de Madrid, Shakhtar dos Leks, Espressinho da Mooca
```

> O sorteio deve garantir que dois times na mesma partida **não recebam o mesmo nome**. Se o número de times for maior que o conjunto, repetir com sufixo numérico (ex: "Leões do Asfalto 2").

---

## 5. Modelagem de Dados

### 5.1 Migration — campos em `group_members`

```sql
-- Migration: 018_group_member_skill.sql

ALTER TABLE group_members
  ADD COLUMN skill_stars   SMALLINT NOT NULL DEFAULT 2,
  ADD COLUMN is_goalkeeper BOOLEAN  NOT NULL DEFAULT FALSE;

ALTER TABLE group_members
  ADD CONSTRAINT chk_skill_stars
    CHECK (skill_stars >= 1 AND skill_stars <= 5);

-- Jogadores já existentes recebem nota 2 por padrão (default acima já cobre)
```

### 5.2 Migration — tabelas de times

```sql
-- Migration: 019_match_teams.sql

CREATE TABLE match_teams (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    match_id   UUID NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    name       VARCHAR(100) NOT NULL,
    color      VARCHAR(7),              -- hex opcional, ex: '#e63946'
    position   SMALLINT NOT NULL,       -- ordem do time (1, 2, ...)
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE match_team_players (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id     UUID NOT NULL REFERENCES match_teams(id) ON DELETE CASCADE,
    player_id   UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    is_reserve  BOOLEAN NOT NULL DEFAULT FALSE,
    UNIQUE (team_id, player_id)
);

-- Índice para busca por partida
CREATE INDEX idx_match_teams_match_id ON match_teams(match_id);
```

---

## 6. Endpoints da API

### 6.1 Edição de membro do grupo (novo campo)

**`PATCH /api/v1/groups/{group_id}/members/{player_id}`** *(novo ou extensão do existente)*

Request:
```json
{
  "skill_stars": 4,
  "is_goalkeeper": true
}
```

- Apenas admin do grupo ou admin global pode chamar.
- Retorna o `GroupMember` atualizado.

### 6.2 Geração de times

**`POST /api/v1/matches/{match_id}/teams`**

- Apenas admin do grupo.
- Sem body obrigatório — lê confirmados e configurações do grupo/partida.
- Sobrescreve times existentes (delete + insert).
- Retorna os times gerados com jogadores.

Response:
```json
{
  "teams": [
    {
      "id": "uuid",
      "name": "Leões do Asfalto",
      "color": "#e63946",
      "position": 1,
      "players": [
        { "player_id": "uuid", "name": "João", "nickname": "Joãozinho", "skill_stars": 4, "is_goalkeeper": true },
        { "player_id": "uuid", "name": "Carlos", "nickname": "Carlão", "skill_stars": 3, "is_goalkeeper": false }
      ]
    },
    {
      "id": "uuid",
      "name": "Tubarões da Várzea",
      "color": "#2a9d8f",
      "position": 2,
      "players": [ ... ]
    }
  ],
  "reserves": [
    { "player_id": "uuid", "name": "Pedro", "nickname": "Pedrinho", "skill_stars": 2, "is_goalkeeper": false }
  ]
}
```

### 6.3 Busca dos times gerados

**`GET /api/v1/matches/{match_id}/teams`**

- Público (sem autenticação obrigatória, igual ao endpoint de status da partida).
- Retorna os times atuais ou `{ "teams": [], "reserves": [] }` se nenhum time foi gerado.

### 6.4 Resultados de votação (endpoint existente — sem alteração)

**`GET /api/v1/matches/{match_id}/votes/results`**

- Já implementado. Público (sem autenticação obrigatória).
- Retorna ranking de jogadores por pontuação, contagem de votantes e elegíveis.
- A página `/match/[hash]/results` usa `match_id` obtido a partir do hash via `GET /matches/public/{hash}`.

**`GET /api/v1/matches/{match_id}/votes/status`**

- Já implementado. Usado para determinar o estado da votação (`open`, `closed`, `not_started`).

### 6.5 Listagem de membros do grupo (campo novo na resposta)

**`GET /api/v1/groups/{group_id}/members`**

- Incluir `skill_stars` e `is_goalkeeper` na resposta.
- Para membros comuns: omitir `skill_stars` e `is_goalkeeper` da resposta (ou retornar `null`). Apenas admin recebe esses campos.

---

## 7. Alterações de Frontend

### 7.1 `/groups/[id]` — listagem de membros (visão admin)

- Exibir nota de cada membro como estrelas visuais (1–5), editáveis inline (clique na estrela desejada).
- Exibir toggle "Goleiro" ao lado da nota.
- Chamar `PATCH /groups/{group_id}/members/{player_id}` ao alterar.
- Feedback visual imediato (otimista) com rollback em caso de erro.

### 7.2 `/match/[hash]` — seção de confirmados

- Exibir botão **"Montar times"** apenas para admin do grupo.
- Botão desabilitado com tooltip se não houver `players_per_team` definido ou menos de 4 confirmados.
- Após geração bem-sucedida, exibir card com link para `/match/[hash]/teams`.
- Se times já existem, botão vira **"Remontar times"** com ícone de refresh.

### 7.3 `/match/[hash]/results` — página pública de resultados de votação

Nova rota: `football-frontend/src/routes/match/[hash]/results/+page.svelte` + `+page.server.ts`

**Fluxo de dados:**
1. `+page.server.ts` carrega via SSR: dados da partida (`GET /matches/public/{hash}`), status da votação e resultados.
2. Meta tags Open Graph renderizadas no servidor para preview no WhatsApp.
3. Dados passados como `props` para o componente Svelte.

**Estrutura visual (mobile-first):**

```
┌─────────────────────────────────┐
│  ← Voltar    Rachão #N          │  ← header
│  [Grupo] · [Data] · [Local]     │
├─────────────────────────────────┤
│  🏆 Melhores do Rachão          │  ← título da seção
│                                 │
│    🥈         🥇        🥉      │
│   [2º]      [1º]      [3º]      │  ← pódio
│  Nome      NOME      Nome       │
│   Xpts     Xpts      Xpts       │
│  ▓▓▓▓▓   ▓▓▓▓▓▓▓   ▓▓▓         │  ← altura proporcional
├─────────────────────────────────┤
│  Ranking completo               │
│  1. Nome ████████████ 12 pts    │
│  2. Nome ████████     9 pts     │
│  3. Nome ██████        7 pts    │
│  ...                            │
├─────────────────────────────────┤
│  X de Y jogadores votaram       │
│  [📋 Compartilhar resultado]    │
└─────────────────────────────────┘
```

**Detalhes de implementação:**

- **Pódio**: `display: flex`, colunas com `align-items: flex-end` para efeito de altura variável.
  - 1º lugar: coluna mais alta (~120px), fonte maior, fundo dourado/âmbar.
  - 2º lugar: coluna média (~90px), fundo cinza-prata.
  - 3º lugar: coluna menor (~70px), fundo bronze.
  - Se menos de 3 jogadores votados: exibir apenas as posições disponíveis, sem lacunas.
- **Barra de progresso no ranking**: `width` proporcional ao líder (líder = 100%, demais = `pts/pts_lider * 100%`). Usar `bg-primary-500` com transição CSS.
- **Nomes**: usar `nickname` se disponível, senão `name`. Truncar com `truncate` em mobile.
- **Botão Compartilhar**: `navigator.clipboard.writeText(window.location.href)` com feedback toast.
- **SSR obrigatório** para meta tags — não usar `export const ssr = false`.

**Acessibilidade:**
- Pódio deve ter `aria-label` descrevendo a posição (ex: `aria-label="1º lugar: Joãozinho, 12 pontos"`).

### 7.4 `/match/[hash]` — link de entrada para resultados

Na página existente da partida, adicionar após a seção de confirmados:

- Quando `vote_status === 'closed'` **e** `results.length > 0`: exibir card com fundo âmbar/dourado sutil e botão "🏆 Ver os melhores do Rachão" apontando para `/match/[hash]/results`.
- O card deve ser visível para todos (logados e não logados).

### 7.5 `/match/[hash]/teams` — página pública de times

Nova rota: `football-frontend/src/routes/match/[hash]/teams/+page.svelte`

Layout:
- Cards lado a lado (flex/grid, mobile-first: coluna única; tablet+: lado a lado).
- Cada card: nome do time, lista de jogadores (apelido ou nome, badge "GK" se goleiro).
- Soma de estrelas do time: visível apenas para admin.
- Reservas: seção separada ao final, se houver.
- Botão "Remontar times" no topo: apenas para admin, abre `ConfirmDialog` antes de executar.
- Botão "Compartilhar": copia URL para clipboard, disponível para todos.
- Link "← Voltar para a partida" no topo.

Se não há times gerados:
- Mensagem: *"Os times ainda não foram sorteados."*
- Para admin: botão "Montar times" (redireciona para a ação na página da partida ou dispara diretamente).

---

## 8. Algoritmo — Pseudocódigo de Referência

```python
def montar_times(confirmados, players_per_team):
    """
    confirmados: list[{ player_id, skill_stars, is_goalkeeper }]
    players_per_team: int  -- jogadores de LINHA (exclui goleiro)
    Tamanho real de cada time = players_per_team + 1 (linha + goleiro/substituto)
    """
    team_size = players_per_team + 1
    n_times = len(confirmados) // team_size
    reservas = confirmados[n_times * team_size:]  # excedentes

    goleiros = [p for p in confirmados if p.is_goalkeeper]
    nao_goleiros = [p for p in confirmados if not p.is_goalkeeper]

    times = [[] for _ in range(n_times)]

    # 1. Distribuir um goleiro por time
    for i, goleiro in enumerate(goleiros[:n_times]):
        times[i].append(goleiro)

    # Goleiros excedentes voltam ao pool
    pool = sorted(
        nao_goleiros + goleiros[n_times:],
        key=lambda p: p.skill_stars,
        reverse=True
    )

    # 2. Snake draft nos restantes
    indices = list(range(n_times)) + list(range(n_times - 1, -1, -1))
    i = 0
    for jogador in pool:
        time_idx = indices[i % len(indices)]
        times[time_idx].append(jogador)
        i += 1

    return times, reservas
```

---

## 9. Levantamento de Impacto por Camada

| Camada | Arquivo / Área | Tipo de mudança |
|---|---|---|
| **DB** | `group_members` | +2 colunas (`skill_stars`, `is_goalkeeper`) |
| **DB** | Novas tabelas `match_teams`, `match_team_players` | Criação |
| **Backend model** | `app/models/group.py` — `GroupMember` | +2 campos |
| **Backend schema** | `app/schemas/group.py` | Campos em response (condicional por role) |
| **Backend repo** | `app/db/repositories/group_repo.py` | Update de membro |
| **Backend repo** | Novo `app/db/repositories/team_repo.py` | CRUD de times |
| **Backend service** | Novo `app/services/team_builder.py` | Algoritmo de sorteio |
| **Backend router** | `app/api/v1/routers/groups.py` | `PATCH /members/{player_id}` |
| **Backend router** | Novo `app/api/v1/routers/teams.py` | `POST` e `GET /matches/{id}/teams` |
| **Backend invite** | `app/api/v1/routers/invites.py` | Garantir `skill_stars=2` ao criar `GroupMember` |
| **Frontend** | `src/routes/groups/[id]/+page.svelte` | Edição inline de estrelas e goleiro |
| **Frontend** | `src/routes/match/[hash]/+page.svelte` | Botão "Montar times" + link para times + card de resultados |
| **Frontend** | Nova `src/routes/match/[hash]/teams/+page.svelte` | Página pública de times |
| **Frontend** | Nova `src/routes/match/[hash]/results/+page.svelte` + `+page.server.ts` | Página pública de resultados com pódio e ranking |

---

## 10. Critérios de Aceitação

- [x] Jogador que aceita convite entra no grupo com `skill_stars = 2` e `is_goalkeeper = false`.
- [x] Admin do grupo consegue editar nota (1–5) e flag de goleiro de qualquer membro.
- [x] Membro comum não visualiza notas nem flags de outros jogadores.
- [x] Botão "Montar times" só aparece para admin do grupo na página da partida.
- [x] Times gerados têm soma de estrelas a mais equilibrada possível (desvio máximo de 1 estrela entre times em cenário ideal).
- [x] Cada time recebe no máximo um goleiro (quando há goleiros suficientes).
- [x] Jogadores excedentes aparecem como reservas, não atribuídos a nenhum time.
- [x] Times recebem nomes distintos entre si na mesma partida.
- [x] Página `/match/[hash]/teams` é acessível sem login.
- [x] Admin consegue remontar os times, sobrescrevendo o sorteio anterior (com confirmação via `ConfirmDialog`).
- [x] Link para `/match/[hash]/teams` aparece na página da partida após a geração dos times.
- [x] Card "Primeiro jogo do rachão" é exibido no topo da página de times, mostrando o confronto entre o time de posição 1 e o de posição 2.
- [x] Grupos existentes não são afetados — membros atuais recebem `skill_stars = 2` via valor padrão da migration.
- [x] Após encerramento da votação, card "🏆 Ver os melhores do Rachão" aparece em `/match/[hash]` para todos os visitantes.
- [x] Página `/match/[hash]/results` é acessível sem login.
- [x] Pódio exibe corretamente os 3 primeiros colocados com altura visual proporcional à posição.
- [x] Se menos de 3 jogadores receberam votos, pódio exibe apenas as posições disponíveis.
- [x] Ranking completo lista todos os jogadores votados em ordem decrescente com barra de progresso.
- [x] Rodapé exibe contagem de votantes (`X de Y jogadores votaram`).
- [x] Botão "Compartilhar" copia a URL para clipboard com feedback toast.
- [x] Meta tags Open Graph renderizadas no servidor para preview correto no WhatsApp.
- [x] Se votação ainda aberta, página exibe mensagem informativa sem revelar o ranking parcial.

---

## 11. Fora de Escopo (v1)

- Posições além de goleiro (lateral, zagueiro, atacante etc.).
- Manter histórico de sorteios anteriores da mesma partida.
- Edição manual de times pelo admin após o sorteio (arrastar jogadores entre times).
- Nota baseada em performance automaticamente calculada a partir dos votos pós-partida.
- Times fixos persistentes por grupo (escalação padrão).
- Limite de goleiros por time configurável.

---

*Documento elaborado para uso interno da equipe de produto e engenharia do Rachao.app.*
