# PRD — Posição do Jogador no Grupo
## Rachao.app · Substituir flag "goleiro" por seletor de posição (GK / ZAG / LAT / MEI / ATA)

| | |
|---|---|
| **Versão** | 1.0 |
| **Status** | 📋 Proposto — aguardando revisão |
| **Data** | Abril de 2026 |
| **Plataforma** | https://rachao.app |

---

## 1. Contexto e Motivação

Atualmente o único atributo de posição de um jogador num grupo é a flag `is_goalkeeper` (booleano). Essa abordagem é limitante:

- O sorteio de times no backend (`team_builder.py`) só distingue goleiros de "não-goleiros", ignorando ZAG, LAT, MEI e ATA.
- O simulador público `/draw` já implementa as 5 posições (GK/ZAG/LAT/MEI/ATA) com selector visual e as usa no sorteio equilibrado — mas essa informação não existe nos grupos reais.
- Admins de grupo precisam de uma forma mais rica de classificar jogadores para sorteios mais justos.

Esta feature substitui o campo `is_goalkeeper` pelo campo `position` (enum de 5 valores) em `group_members`, alinhando grupos reais com o simulador `/draw`.

---

## 2. Decisões de Produto

| Decisão | Escolha |
|---------|---------|
| Posições disponíveis | GK · ZAG · LAT · MEI · ATA (mesmas do `/draw`) |
| Campo `is_goalkeeper` | **Removido** — derivado de `position = 'gk'` na migration |
| Posição padrão ao adicionar membro | `"mei"` |
| Quem pode editar | Admin global e admin do grupo |
| Exibição para membros comuns | Badge de posição visível para todos os membros |
| Integração com sorteio | Backend passa `position` ao `team_builder`; GK continua com tratamento especial |

---

## 3. Requisitos Funcionais

### RF-01 — Seletor de posição na edição de membro

Na aba **Jogadores** do grupo (`/groups/[id]`), ao clicar em um membro, o modal de detalhe exibe o seletor de posição no lugar do toggle "Goleiro". O layout segue o mesmo padrão visual do `/draw`:

```
[GK]  [ZAG]  [LAT]  [MEI]  [ATA]
```

Cada badge é clicável e destaca a posição selecionada. Um estado "não definida" (nenhuma selecionada) é permitido.

---

### RF-02 — Seletor de posição no AddMemberModal

O componente `AddMemberModal.svelte` (etapas 2-A e 2-B) substitui o toggle "Goleiro" pelo mesmo seletor de posição. O valor padrão é `"mei"`.

---

### RF-03 — Exibição da posição na lista de membros

Na lista de membros da aba Jogadores, onde hoje é exibida a badge "Goleiro" para membros com `is_goalkeeper = true`, passa a ser exibida a badge de posição (com a mesma cor do `/draw`) para membros cuja posição foi definida.

A badge é visível para todos os membros do grupo (não apenas admin).

---

### RF-04 — Endpoint de atualização de membro

`PATCH /api/v1/groups/{group_id}/members/{player_id}`

O campo `is_goalkeeper` do request é **removido**. Um novo campo `position` é adicionado:

```json
{
  "skill_stars": 3,
  "position": "gk"
}
```

Valores aceitos para `position`: `"gk"`, `"zag"`, `"lat"`, `"mei"`, `"ata"`, `null`. Padrão ao omitir: `"mei"`.

---

### RF-05 — Endpoint de adição de membro por telefone

`POST /api/v1/groups/{group_id}/members/by-phone`

O campo `is_goalkeeper` é removido do request; `position` é adicionado (opcional, padrão `null`).

---

### RF-06 — Integração com sorteio de times

O `team_builder.py` e o router `teams.py` passam a usar `position` em vez de `is_goalkeeper`. A lógica de separação de goleiros continua idêntica, usando `position == 'gk'` como critério.

O `group_repo.py` (método `get_confirmed_with_skill` e `get_members_skill`) retorna `position` em vez de `is_goalkeeper`.

---

## 4. Requisitos Não-Funcionais

- **Retrocompatibilidade**: migration `035_` converte `is_goalkeeper = true → position = 'gk'`, `false → null`, depois dropa a coluna `is_goalkeeper`.
- **Idempotência da migration**: usar `IF NOT EXISTS` / `ON CONFLICT DO NOTHING` e verificar se coluna já existe antes de adicionar.
- **Sorteio sem posição definida**: jogadores com `position = null` são tratados como "linha" (não-goleiro) no sorteio, mantendo o comportamento atual.

---

## 5. Modelagem de Dados

### Migration `035_group_member_position.sql`

```sql
-- 1. Adicionar coluna position com padrão 'mei'
ALTER TABLE group_members
  ADD COLUMN IF NOT EXISTS position VARCHAR(3) NOT NULL DEFAULT 'mei';

-- 2. Migrar is_goalkeeper = true → position = 'gk'
UPDATE group_members
  SET position = 'gk'
  WHERE is_goalkeeper = true;

-- 3. Remover coluna is_goalkeeper
ALTER TABLE group_members
  DROP COLUMN IF EXISTS is_goalkeeper;
```

Valores válidos: `'gk'`, `'zag'`, `'lat'`, `'mei'`, `'ata'`, `NULL`.

> Não é usado enum nativo do PostgreSQL para evitar o custo de criar/alterar um pg enum; a validação fica no schema Pydantic.

### Modelo ORM (`app/models/group.py`)

```python
# Antes:
is_goalkeeper: Mapped[bool] = mapped_column(Boolean, nullable=False, default=False)

# Depois:
position: Mapped[str] = mapped_column(String(3), nullable=False, default="mei")
```

---

## 6. Endpoints da API — Alterações

| Endpoint | Campo removido | Campo adicionado |
|----------|----------------|-----------------|
| `PATCH /groups/{id}/members/{player_id}` | `is_goalkeeper` | `position` (opcional, nullable) |
| `POST /groups/{id}/members/by-phone` | `is_goalkeeper` | `position` (opcional, default null) |
| `GET /groups/{id}` (response) | `is_goalkeeper` em `GroupMemberResponse` | `position` |
| `GET /groups/{id}/members` (response) | `is_goalkeeper` | `position` |

---

## 7. Schemas Pydantic — Alterações

### `app/schemas/group.py`

```python
# GroupMemberResponse
position: str | None = None          # substitui is_goalkeeper

# UpdateMemberRequest
position: str | None = Field(None, pattern=r'^(gk|zag|lat|mei|ata)$')  # substitui is_goalkeeper

# AddMemberByPhoneRequest
position: str = Field("mei", pattern=r'^(gk|zag|lat|mei|ata)$')  # padrão "mei"
```

### `app/schemas/team.py`

```python
# TeamPlayer
position: str | None = None          # substitui is_goalkeeper: bool
```

### `app/schemas/player_stats.py`

```python
# GroupStatItem
position: str | None = None          # substitui is_goalkeeper: bool
```

---

## 8. Alterações de Frontend

### 8.1 Componente `PositionSelector.svelte` (novo)

Componente reutilizável que renderiza os 5 badges clicáveis de posição, análogo ao `<select>` de posição no `/draw`, mas no estilo do resto do app (badges com cores).

```
Props:
  bind:value: string | null
  readonly?: boolean (default false)
```

Reutiliza as cores e abreviações já definidas em `team-builder.ts` (`POS_ABBR`, `POS_COLOR_CLASSES`).

### 8.2 `/groups/[id]` — Modal de detalhe do membro

Substituir toggle "Goleiro" por `<PositionSelector bind:value={roleEditMember.position} />`.

### 8.3 `AddMemberModal.svelte`

Substituir toggle "Goleiro" por `<PositionSelector bind:value={position} />` nas etapas 2-A e 2-B.

### 8.4 Lista de membros — badge de posição

```svelte
<!-- Antes -->
{#if isGroupAdmin() && m.is_goalkeeper}
  <span class="badge-blue">Goleiro</span>
{/if}

<!-- Depois -->
{#if m.position}
  <span class="{POS_COLOR_CLASSES[m.position]} badge">
    {POS_ABBR[m.position]}
  </span>
{/if}
```

A badge é visível para todos os membros (não apenas admin), pois a posição não é informação sensível.

### 8.5 `src/lib/api.ts`

```typescript
// GroupMember: substituir is_goalkeeper por position
export type GroupMember = {
  ...
  position: 'gk' | 'zag' | 'lat' | 'mei' | 'ata' | null;
  // is_goalkeeper removido
};

// updateMemberSkill: substituir is_goalkeeper por position
updateMemberSkill: (groupId, playerId, data: { skill_stars?: number; position?: string | null })
```

### 8.6 i18n

Adicionar chaves para os nomes completos das posições (usados em tooltips/acessibilidade):

| Chave | pt-BR | en | es |
|-------|-------|----|----|
| `position.gk` | Goleiro | Goalkeeper | Portero |
| `position.zag` | Zagueiro | Defender | Defensa central |
| `position.lat` | Lateral | Fullback | Lateral |
| `position.mei` | Meia | Midfielder | Mediocampista |
| `position.ata` | Atacante | Forward | Delantero |
| `position.unset` | Não definida | Not set | Sin definir |

---

## 9. Levantamento de Impacto por Camada

| Camada | Arquivo | Tipo de mudança |
|--------|---------|-----------------|
| **Migration** | `migrations/035_group_member_position.sql` | Nova coluna `position`, remove `is_goalkeeper` |
| **Model** | `app/models/group.py` | `is_goalkeeper → position` |
| **Schema** | `app/schemas/group.py` | `is_goalkeeper → position` em 3 schemas |
| **Schema** | `app/schemas/team.py` | `is_goalkeeper → position` |
| **Schema** | `app/schemas/player_stats.py` | `is_goalkeeper → position` |
| **Repo** | `app/db/repositories/group_repo.py` | `get_confirmed_with_skill`, `get_members_skill` |
| **Service** | `app/services/team_builder.py` | Lógica de GK usa `position == 'gk'` |
| **Router** | `app/api/v1/routers/teams.py` | Passa `position` ao builder |
| **Router** | `app/api/v1/routers/groups.py` | `update_member`, `add_member_by_phone` |
| **Frontend** | Novo `PositionSelector.svelte` | Componente reutilizável |
| **Frontend** | `src/routes/groups/[id]/+page.svelte` | Badge e modal de detalhe |
| **Frontend** | `AddMemberModal.svelte` | Substituir toggle por PositionSelector |
| **Frontend** | `src/lib/api.ts` | Tipos e método `updateMemberSkill` |
| **i18n** | `messages/pt-BR.json`, `en.json`, `es.json` | Chaves `position.*` |

---

## 10. Testes Unitários (Backend)

| Caso | Resultado esperado |
|------|--------------------|
| `PATCH /members/{id}` com `position: "gk"` | 200, `position = "gk"` |
| `PATCH /members/{id}` com `position: null` | 200, `position = null` |
| `PATCH /members/{id}` com valor inválido | 422 |
| `POST /members/by-phone` com `position: "mei"` | 201, member com `position = "mei"` |
| `POST /members/by-phone` sem `position` | 201, member com `position = "mei"` (padrão) |
| Sorteio com membro `position = "gk"` | GK separado corretamente |
| Sorteio com membro `position = "mei"` | Tratado como linha |

---

## 11. Critérios de Aceitação

- [ ] Admin do grupo vê seletor de posição (GK/ZAG/LAT/MEI/ATA) no modal de edição do membro.
- [ ] Posição é salva e exibida como badge colorido na lista de membros.
- [ ] Todo membro tem badge de posição visível (padrão MEI para quem foi cadastrado sem posição).
- [ ] O toggle "Goleiro" não existe mais em nenhuma parte do app.
- [ ] `AddMemberModal` usa seletor de posição nas etapas 2-A e 2-B.
- [ ] Sorteio de times (`/groups/[id]` → aba do rachão) distribui GKs corretamente com base na posição.
- [ ] Membros migrados sem posição anterior assumem `"mei"` automaticamente.
- [ ] Migration converte dados existentes sem perda: `is_goalkeeper=true → position='gk'`.
- [ ] Admin global pode editar posição de qualquer membro de qualquer grupo.
- [ ] Todos os textos usam chaves i18n.

---

## 12. Fora de Escopo (v1)

- Posição por partida (jogador pode jogar MEI num rachão e ATA em outro) — a posição é do membro no grupo, não por partida.
- Restrição de sorteio por posição além de GK (ex: garantir N zagueiros por time) — sorteio continua sendo por estrelas + GK.
- Exibição da posição no perfil público do jogador (`/players/[id]`).
- Filtro/busca de membros por posição.

---

*Documento elaborado para uso interno da equipe de produto e engenharia do Rachao.app.*
