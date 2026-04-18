# PRD — Gols e Assistências por Partida
## Rachao.app · Estatísticas de Desempenho Individual

| | |
|---|---|
| **Versão** | 1.0 |
| **Status** | 📋 Proposto — aguardando priorização |
| **Data** | Abril de 2026 |
| **Plataforma** | https://rachao.app |

---

## 1. Contexto e Motivação

Atualmente o único dado pós-partida registrado no sistema é a **votação** (Top 5 + Decepção), que reflete percepção subjetiva dos participantes. Não há registro de dados objetivos de desempenho: quem marcou gol, quem deu assistência.

Grupos mais competitivos já controlam essas informações em planilhas ou no WhatsApp. A proposta é trazer esse controle para dentro da plataforma, enriquecendo o histórico individual de cada jogador e gerando mais dados para rankings e estatísticas.

**Decisão confirmada:** Apenas o **admin do grupo** (presidente) pode registrar e editar gols e assistências de uma partida. Jogadores não registram os próprios dados.

---

## 2. Decisões de Produto

| Decisão | Escolha | Justificativa |
|---------|---------|---------------|
| Quem registra | Admin do grupo exclusivamente | Evita auto-inflação de estatísticas; mantém responsabilidade centralizada |
| Quando pode registrar | Qualquer momento após a partida ser criada (sem restrição de status) | Admin pode querer registrar durante o jogo ou logo após |
| Granularidade | Por jogador por partida | Unidade natural de registro |
| Edição | Permitida a qualquer momento pelo admin | Correções de digitação são esperadas |
| Jogadores elegíveis | Apenas confirmados (`attendance.status = confirmed`) | Apenas quem jogou pode ter gols/assistências atribuídos |
| Visibilidade | Pública — qualquer pessoa com o link da partida vê os dados | Alinhado com a visibilidade atual de resultados de votação |
| Exibição em stats | Acumula no perfil do jogador (total de gols e assistências da carreira) | Incentiva o registro retroativo de partidas anteriores |
| Partidas sem registro | Exibir como "sem dados" (não zero) | Distinguir "não registrado" de "ninguém marcou" |

---

## 3. Requisitos Funcionais

**RF-01 — Registro de gols e assistências**
O admin do grupo pode registrar gols e assistências para qualquer jogador confirmado na partida. O registro é feito em lote — o admin envia os dados de todos os jogadores de uma vez (ou atualiza individualmente).

Exemplo de fluxo:
- Admin acessa a partida encerrada
- Vê lista de jogadores confirmados com campos numéricos (gols e assistências)
- Preenche e salva → dados são persistidos com `upsert` (cria ou atualiza)

**RF-02 — Edição posterior**
Um admin pode corrigir ou atualizar os dados de uma partida já registrada a qualquer momento. Não há versionamento — apenas o valor atual é armazenado.

**RF-03 — Exibição na partida**
Na página pública `/match/{hash}` e `/match/{hash}/results`, exibir os gols e assistências de cada jogador confirmado. Jogadores sem dado registrado não aparecem na lista (só aparecem se ao menos um registro existir para a partida).

**RF-04 — Exibição no perfil público**
Na página `/players/{id}`, exibir o total de gols e assistências da carreira do jogador, acumulado de todas as partidas registradas em todos os grupos que ele participa.

**RF-05 — Exibição nas estatísticas pessoais**
Na página `/profile/stats`, exibir total de gols e assistências com filtros de período (alinhado ao padrão já existente de filtro por período nas stats).

**RF-06 — Validações**
- Gols e assistências: inteiro ≥ 0, máximo 20 por jogador por partida (evita erros de digitação absurdos)
- Não é possível registrar dados para um jogador que não esteja na lista de confirmados da partida
- Se o admin tentar registrar para um `player_id` não confirmado nessa partida → `422 PLAYER_NOT_CONFIRMED`

---

## 4. Requisitos Não-Funcionais

- **Autorização:** Endpoint de escrita exige `CurrentPlayer` com `GroupMember.role = "admin"` para o grupo dono da partida. Violação → `403 ForbiddenError`.
- **Performance:** A query de leitura dos stats da partida deve fazer um único `SELECT` com JOIN em `match_player_stats` + `players`. Sem N+1.
- **Idempotência:** O endpoint de escrita usa `INSERT ... ON CONFLICT (match_id, player_id) DO UPDATE` para ser idempotente.
- **Super admin:** `PlayerRole.ADMIN` (super admin da plataforma) não tem acesso implícito ao endpoint de escrita — precisa ser admin do grupo como qualquer outro.

---

## 5. Modelagem de Dados

### 5.1 Nova tabela: `match_player_stats`

**Migration:** `038_match_player_stats.sql`

```sql
CREATE TABLE IF NOT EXISTS match_player_stats (
  id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  match_id    UUID        NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
  player_id   UUID        NOT NULL REFERENCES players(id) ON DELETE CASCADE,
  goals       INTEGER     NOT NULL DEFAULT 0 CHECK (goals >= 0 AND goals <= 20),
  assists     INTEGER     NOT NULL DEFAULT 0 CHECK (assists >= 0 AND assists <= 20),
  recorded_by UUID        NOT NULL REFERENCES players(id),
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (match_id, player_id)
);

CREATE INDEX IF NOT EXISTS idx_match_player_stats_match_id  ON match_player_stats(match_id);
CREATE INDEX IF NOT EXISTS idx_match_player_stats_player_id ON match_player_stats(player_id);
```

### 5.2 Novo modelo ORM: `MatchPlayerStats`

Arquivo: `football-api/app/models/match.py` (adicionar junto aos modelos existentes de partida)

```python
class MatchPlayerStats(Base):
    __tablename__ = "match_player_stats"

    id          = Column(UUID(as_uuid=True), primary_key=True, default=uuid4)
    match_id    = Column(UUID(as_uuid=True), ForeignKey("matches.id", ondelete="CASCADE"), nullable=False)
    player_id   = Column(UUID(as_uuid=True), ForeignKey("players.id", ondelete="CASCADE"), nullable=False)
    goals       = Column(Integer, nullable=False, default=0)
    assists     = Column(Integer, nullable=False, default=0)
    recorded_by = Column(UUID(as_uuid=True), ForeignKey("players.id"), nullable=False)
    created_at  = Column(DateTime(timezone=True), server_default=func.now(), nullable=False)
    updated_at  = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), nullable=False)

    match  = relationship("Match",  back_populates="player_stats", foreign_keys=[match_id])
    player = relationship("Player", foreign_keys=[player_id])
```

Adicionar em `Match`:
```python
player_stats = relationship("MatchPlayerStats", back_populates="match",
                             foreign_keys="[MatchPlayerStats.match_id]",
                             cascade="all, delete-orphan")
```

### 5.3 Impacto em `player_stats_repo.py`

Adicionar dois novos campos nas queries de stats agregadas:
- `total_goals` — `SUM(mps.goals)` filtrando por `player_id` e, quando aplicável, por `group_id`
- `total_assists` — `SUM(mps.assists)` com o mesmo escopo

---

## 6. Endpoints da API

| Método | Path | Auth | Descrição |
|--------|------|------|-----------|
| `GET`  | `/api/v1/matches/{hash}/player-stats` | Público | Retorna gols e assistências de todos os jogadores da partida |
| `PUT`  | `/api/v1/matches/{hash}/player-stats` | Admin do grupo | Upsert em lote dos dados de gols e assistências |

### 6.1 `GET /api/v1/matches/{hash}/player-stats`

**Response 200:**
```json
{
  "match_hash": "abc123",
  "registered": true,
  "stats": [
    {
      "player_id": "uuid",
      "player_name": "João Silva",
      "avatar_url": null,
      "goals": 2,
      "assists": 1
    }
  ]
}
```

Se nenhum registro existir para a partida, retornar `"registered": false` e `"stats": []`.

### 6.2 `PUT /api/v1/matches/{hash}/player-stats`

**Request body:**
```json
{
  "stats": [
    { "player_id": "uuid-a", "goals": 2, "assists": 1 },
    { "player_id": "uuid-b", "goals": 0, "assists": 2 },
    { "player_id": "uuid-c", "goals": 1, "assists": 0 }
  ]
}
```

- Apenas `player_id`s com `attendance.status = "confirmed"` são aceitos.
- Registros de jogadores confirmados **não incluídos** no payload são deletados (o PUT é substitutivo — representa o estado completo).
- **Response 200:** mesmo formato do GET.

---

## 7. Schemas Pydantic

Arquivo: `football-api/app/schemas/match.py`

```python
class PlayerStatInput(BaseModel):
    player_id: UUID
    goals:   int = Field(ge=0, le=20)
    assists: int = Field(ge=0, le=20)

class MatchPlayerStatsRequest(BaseModel):
    stats: list[PlayerStatInput]

class PlayerStatResponse(BaseModel):
    player_id:   UUID
    player_name: str
    avatar_url:  str | None
    goals:       int
    assists:     int

class MatchPlayerStatsResponse(BaseModel):
    match_hash:  str
    registered:  bool
    stats:       list[PlayerStatResponse]
```

---

## 8. Alterações de Frontend

### 8.1 `/match/[hash]` — Seção de registro (admin only)

Exibir abaixo da lista de presentes, visível apenas para o admin do grupo quando a partida existe (qualquer status):

- Card **"Gols e Assistências"** com lista dos jogadores confirmados
- Cada linha: avatar + nome + input numérico de gols + input numérico de assistências
- Botão **"Salvar"** — chama `PUT /api/v1/matches/{hash}/player-stats`
- Se já há dados registrados, os inputs carregam os valores existentes (GET ao montar)
- Inputs com `type="number"`, `min="0"`, `max="20"`, largura mínima para mobile

### 8.2 `/match/[hash]/results` — Exibição pública

Adicionar seção **"Artilheiros & Assistências"** quando `registered = true`:

- Tabela: Jogador | Gols | Assistências
- Ordenação: maior número de gols primeiro; empate → maior número de assistências
- Exibir apenas jogadores com pelo menos 1 gol ou 1 assistência registrados

### 8.3 `/players/[id]` — Perfil público

Adicionar dois badges no card de stats do jogador:
- ⚽ **X gols** (carreira)
- 🅰️ **X assistências** (carreira)

Exibir apenas se houver dados (não mostrar "0 gols" se nenhuma partida foi registrada).

### 8.4 `/profile/stats` — Estatísticas pessoais

Adicionar gols e assistências no bloco de big numbers já existente, com filtro de período.

### 8.5 Componente novo sugerido

`MatchPlayerStatsEditor.svelte` — componente reutilizável do editor (admin), isolado para facilitar testes e reuso.

---

## 9. Levantamento de Impacto por Camada

| Camada | Arquivo(s) | Tipo de mudança |
|--------|-----------|-----------------|
| Migration | `migrations/038_match_player_stats.sql` | Nova tabela |
| Model | `app/models/match.py` | Nova classe `MatchPlayerStats` + relacionamento em `Match` |
| Schema | `app/schemas/match.py` | 4 novos schemas Pydantic |
| Repository | `app/db/repositories/match_stats_repo.py` (novo) | GET e UPSERT de stats |
| Repository | `app/db/repositories/player_stats_repo.py` | Adicionar `total_goals` e `total_assists` às queries agregadas |
| Router | `app/api/v1/routers/matches.py` | 2 novos endpoints |
| Frontend | `src/routes/match/[hash]/+page.svelte` | Seção de edição (admin) |
| Frontend | `src/routes/match/[hash]/results/+page.svelte` | Seção de artilheiros |
| Frontend | `src/routes/players/[id]/+page.svelte` | Badges de gols/assistências |
| Frontend | `src/routes/profile/stats/+page.svelte` | Big numbers de gols/assistências |
| Frontend | `src/lib/api.ts` | Novos métodos `getMatchPlayerStats`, `putMatchPlayerStats` |
| i18n | `messages/pt-BR.json`, `en.json`, `es.json` | Chaves `stats.goals`, `stats.assists`, `stats.scorers`, etc. |

---

## 10. Testes Unitários

Arquivo: `tests/unit/routers/test_match_player_stats.py`

| Caso | Entrada | Esperado |
|------|---------|----------|
| GET sem dados | Partida sem nenhum registro | `registered: false`, `stats: []` |
| GET com dados | Partida com registros | Lista correta de jogadores com gols/assists |
| PUT — happy path | Admin + jogadores confirmados + valores válidos | 200, registros persistidos |
| PUT — não admin | Jogador sem role admin | 403 |
| PUT — player não confirmado | `player_id` com attendance pending | 422 `PLAYER_NOT_CONFIRMED` |
| PUT — valor inválido | `goals: 25` (acima do máximo) | 422 validation error |
| PUT — idempotência | Mesmo payload duas vezes | Resultado idêntico, sem duplicação |
| PUT — substitutivo | Primeiro PUT com A,B,C; segundo PUT com apenas A | B e C são removidos |

---

## 11. Critérios de Aceitação

- [ ] Admin do grupo consegue registrar gols e assistências para jogadores confirmados
- [ ] Admin consegue editar um registro já existente
- [ ] Jogador não-admin não consegue acessar o endpoint de escrita (403)
- [ ] Tentativa de registrar para jogador não confirmado retorna 422
- [ ] Página de resultados exibe artilheiros quando há dados registrados
- [ ] Página de resultados não exibe a seção quando `registered = false`
- [ ] Perfil público do jogador exibe total de gols e assistências da carreira
- [ ] Stats pessoais exibem gols e assistências com filtro de período
- [ ] Telas funcionam corretamente em mobile (inputs numéricos usáveis no teclado mobile)
- [ ] Todos os textos visíveis usam chaves i18n (pt-BR, en, es)

---

## 12. Fora de Escopo (v1)

- **Gols contra** — não há distinção entre gol normal e gol contra
- **Assistências duplas / passes decisivos** — apenas uma assistência por gol
- **Histórico de edições** — não há auditoria de quem editou o quê e quando
- **Notificação push** ao registrar gols — fora de escopo inicial
- **Ranking de artilheiros** na página de ranking global — pode ser adicionado em v2
- **Importação retroativa em lote** de partidas antigas — manual, pelo fluxo normal
- **Gols por time** — placar final da partida não é escopo desse PRD (seria uma feature separada de "Resultado da partida")
