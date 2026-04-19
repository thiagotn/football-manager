# PRD — Cores de Coletes e Nomes de Times por Grupo

## Rachao.app · Gestão de Grupos / Sorteio de Times

| | |
|---|---|
| **Número** | 037 |
| **Versão** | 1.0 |
| **Status** | 📋 Proposto — aguardando priorização |
| **Data** | Abril de 2026 |
| **Plataforma** | https://rachao.app |

---

## Problema

Após o sorteio de times, os nomes gerados são aleatórios (ex: "Real Madruga", "Barsemlona") e não têm relação com a realidade do grupo. Na prática, os jogadores se identificam pelo colete que vestem (laranja, azul, verde…) ou por um apelido que o grupo já usa. Nomear os times pelas cores dos coletes ou pelos nomes habituais torna a comunicação imediata: "você está no Time Laranja" é inequívoco em campo.

---

## Solução

Cada grupo pode configurar até 5 **slots de time** — cada slot combina uma cor de colete (opcional) e um nome customizado (opcional). Quando pelo menos um slot estiver preenchido, o sorteio usa essas configurações em vez dos nomes aleatórios. O campo é totalmente opcional: grupos que não configurarem nada continuam com o comportamento atual.

---

## Paleta de cores disponíveis

7 cores pré-definidas, representando os coletes mais comuns no futebol de várzea brasileiro:

| Slug | Label | Hex |
|------|-------|-----|
| `laranja` | Laranja | `#f97316` |
| `azul` | Azul | `#3b82f6` |
| `verde` | Verde | `#22c55e` |
| `vermelho` | Vermelho | `#ef4444` |
| `amarelo` | Amarelo | `#eab308` |
| `preto` | Preto | `#1f2937` |
| `branco` | Branco | `#f1f5f9` |

---

## Estrutura de um slot

Cada slot tem dois campos, ambos opcionais (mas ao menos um deve estar preenchido ao salvar):

| Campo | Tipo | Descrição |
|-------|------|-----------|
| `color` | string (slug) ou null | Cor do colete da paleta acima |
| `name` | string (max 40 chars) ou null | Nome customizado do time (ex: "Leões do Rei") |

---

## Lógica de nomes ao sortear

Para cada i-ésimo time sorteado, o nome é determinado em ordem de prioridade:

| Condição | Nome usado | Cor visual |
|----------|-----------|------------|
| Slot i tem **nome** preenchido | Nome customizado (ex: "Leões do Rei") | Hex do slot (se tiver cor) ou `TEAM_COLORS[i % 8]` |
| Slot i tem **cor** mas sem nome | "Time {Cor}" (ex: "Time Laranja") | Hex da cor do slot |
| Slot i não existe (mais times que slots) | `TEAM_NAMES` aleatório (comportamento atual) | `TEAM_COLORS[i % 8]` |
| Nenhum slot cadastrado | `TEAM_NAMES` aleatório para todos | `TEAM_COLORS[i % 8]` |

---

## Escopo técnico

### Backend

**Migration `039_group_team_slots.sql`**
```sql
ALTER TABLE groups
  ADD COLUMN IF NOT EXISTS team_slots JSONB DEFAULT NULL;
```
Armazena array de objetos, ex:
```json
[
  {"color": "laranja", "name": "Leões do Rei"},
  {"color": "azul",    "name": null},
  {"color": null,      "name": "Os Brabos"}
]
```

**`app/schemas/group.py`**
```python
class TeamSlot(BaseModel):
    color: str | None = None   # slug da paleta ou None
    name:  str | None = None   # max 40 chars

    @model_validator(mode='after')
    def at_least_one(self):
        if not self.color and not self.name:
            raise ValueError('slot must have color or name')
        return self

class GroupUpdate(BaseModel):
    ...
    team_slots: list[TeamSlot] | None = None  # max 5 itens
```
- `GroupResponse`: expor `team_slots: list[TeamSlot] | None`

**`app/models/group.py`**
- Adicionar: `team_slots: Mapped[list | None]` usando `JSON` (JSONB via dialect)

**`app/api/v1/routers/teams.py`**
- Na geração de times, após sortear, verificar `group.team_slots`
- Para cada time i: aplicar a lógica de prioridade acima
- Constante `BIB_COLOR_HEX` no router (ou em módulo compartilhado) mapeia slug → hex

### Frontend

**`football-frontend/src/lib/team-names.ts`**
- Exportar `BIB_COLOR_PALETTE: {slug, label, hex}[]` com os 7 itens

**`football-frontend/src/routes/groups/[id]/+page.svelte`** (aba Configurações)
- Nova seção "Times do grupo"
- Até 5 slots, cada slot tem:
  - Seletor visual de cor (botões circulares com os 7 coletes)
  - Input de texto para nome do time
  - Botão de remover o slot
- Botão "Adicionar time" (desabilitado após 5 slots)
- Botão "Salvar" → `PATCH /api/v1/groups/{id}` com `{team_slots: [...]}`

**`football-frontend/src/routes/groups/new/+page.svelte`**
- Mesma seção de slots, opcional no formulário de criação

**i18n** (`messages/pt-BR.json`, `en.json`, `es.json`)

| Chave | PT-BR |
|-------|-------|
| `group.team_slots_label` | Times do grupo |
| `group.team_slots_hint` | Configure até 5 times. Informe a cor do colete e/ou o nome. Os times sorteados usarão essas configurações na ordem definida. |
| `group.team_slot_color` | Cor do colete |
| `group.team_slot_name` | Nome do time |
| `group.team_slot_name_placeholder` | Ex: Leões do Rei |
| `group.team_slot_add` | Adicionar time |
| `group.team_slots_save` | Salvar times |
| `group.bib_color_laranja` | Laranja |
| `group.bib_color_azul` | Azul |
| `group.bib_color_verde` | Verde |
| `group.bib_color_vermelho` | Vermelho |
| `group.bib_color_amarelo` | Amarelo |
| `group.bib_color_preto` | Preto |
| `group.bib_color_branco` | Branco |

---

## Arquivos a criar/modificar

| Arquivo | Ação |
|---------|------|
| `football-api/migrations/039_group_team_slots.sql` | Criar |
| `football-api/app/models/group.py` | Modificar |
| `football-api/app/schemas/group.py` | Modificar |
| `football-api/app/api/v1/routers/teams.py` | Modificar |
| `football-frontend/src/lib/team-names.ts` | Modificar |
| `football-frontend/src/routes/groups/[id]/+page.svelte` | Modificar |
| `football-frontend/src/routes/groups/new/+page.svelte` | Modificar |
| `football-frontend/messages/pt-BR.json` | Modificar |
| `football-frontend/messages/en.json` | Modificar |
| `football-frontend/messages/es.json` | Modificar |
| `football-api/CLAUDE.md` | Atualizar próxima migration → 040 |

---

## Fora de escopo (v1)

- Cores customizadas (hex livre) — apenas paleta pré-definida
- Reordenar slots via drag-and-drop — usar apenas ordem de inserção
- Internacionalização do nome "Time {Cor}" — sempre PT-BR (é configuração do grupo)
- Exibir cor do colete como badge visual na listagem de times (`/match/{hash}/teams`) — pode ser adicionado em v2

---

## Verificação

1. Grupo sem slots → sortear → nomes aleatórios normais ✓
2. 3 slots com cor apenas → "Time Laranja", "Time Azul", "Time Verde" ✓
3. 3 slots com nome apenas → usa o nome customizado ✓
4. Slot com cor + nome → usa o nome customizado (nome tem prioridade) ✓
5. Sortear 4 times com 3 slots → 4º time recebe nome aleatório ✓
6. Tentar adicionar 6º slot → botão desabilitado ✓
7. Salvar slots vazio / limpar → volta ao comportamento padrão ✓
8. Página pública `/match/{hash}/teams` exibe corretamente os nomes configurados ✓
9. Slot sem cor e sem nome → schema rejeita com 422 ✓
