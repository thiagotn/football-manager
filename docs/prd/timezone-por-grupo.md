# PRD — Timezone por Grupo

| | |
|---|---|
| **Status** | ✅ Implementado — Março 2026 |
| **Data** | Março de 2026 |

---

## 1. Contexto

O rachao.app armazena `match_date` como `DATE` e `start_time`/`end_time` como `TIME` no PostgreSQL — valores sem informação de fuso horário. Isso funciona corretamente enquanto todos os participantes de um grupo estão no mesmo fuso, pois o admin digita "20:00" e todos entendem como "20:00 horário local".

Com a internacionalização (en, es, pt-BR) e a possibilidade de grupos em outros países, surgiu o cenário de grupos fora do Brasil. Um grupo na Alemanha cria rachões às 19:00 horário de Berlim; um jogador que acessa o app de outro fuso precisa saber em qual referência horária o evento ocorre.

A solução mais simples e sem dependências externas é permitir que o admin defina o **timezone do grupo** manualmente ao criar ou editar o grupo. Todos os horários de partidas desse grupo são exibidos no fuso configurado, com uma indicação clara para o usuário.

---

## 2. Problema

- `start_time`/`end_time` são `TIME` sem timezone — não há como saber em qual fuso o horário foi digitado
- Não existe campo `timezone` na tabela `groups`
- O frontend exibe `start_time` diretamente como string (`"20:00"`), sem qualquer conversão
- Um jogador em fuso diferente do grupo não tem como saber se "20:00" é horário de Brasília, de Berlim ou de Lisboa
- Geocodificação automática via endereço dependeria de APIs pagas (Google) ou com restrições (Nominatim) e exigiria que o campo `location` fosse um endereço estruturado — hoje é texto livre

---

## 3. Objetivos

- Permitir que cada grupo tenha um fuso horário configurado
- Exibir todos os horários de partidas desse grupo no fuso configurado
- Indicar visualmente o fuso ao usuário quando ele difere do fuso local do dispositivo
- Não introduzir dependências externas (sem APIs de geocodificação)
- Não quebrar dados existentes (grupos sem timezone devem continuar funcionando)

## 4. Não-objetivos

- Geocodificação automática de endereço para timezone
- Conversão de horário para o fuso do dispositivo do usuário (o horário exibido é sempre o do evento)
- Migração da coluna `start_time` para `TIMESTAMPTZ`
- Suporte a timezones por partida individual (granularidade de grupo é suficiente)

---

## 5. Solução Proposta

### 5.1 Backend

**Nova coluna na tabela `groups`:**

```sql
ALTER TABLE groups
  ADD COLUMN IF NOT EXISTS timezone TEXT NOT NULL DEFAULT 'America/Sao_Paulo';
```

- Tipo: `TEXT` com valor padrão `America/Sao_Paulo`
- Validação no schema Pydantic: deve ser um timezone IANA válido (ex: `Europe/Berlin`, `America/New_York`)
- Grupos existentes herdam `America/Sao_Paulo` automaticamente via `DEFAULT`

**Modelo SQLAlchemy (`Group`):**

```python
timezone: Mapped[str] = mapped_column(
    String(60), nullable=False, default="America/Sao_Paulo", server_default="America/Sao_Paulo"
)
```

**Schema (`GroupCreate` / `GroupUpdate`):**

```python
timezone: str = Field("America/Sao_Paulo", description="IANA timezone identifier")

@field_validator("timezone")
@classmethod
def validate_timezone(cls, v: str) -> str:
    import zoneinfo
    try:
        zoneinfo.ZoneInfo(v)
    except Exception:
        raise ValueError(f"Timezone inválido: {v}")
    return v
```

**`GroupResponse`:** adicionar `timezone: str` ao schema de resposta.

**Nota:** `zoneinfo` é stdlib Python 3.9+ — sem dependência externa.

### 5.2 Frontend — Seletor de Timezone

**Componente de seleção:**

Um `<select>` com os timezones mais comuns pré-listados, agrupados por região. Não é necessário listar todos os ~600 timezones IANA — uma lista curada de ~50 fusos cobre os casos de uso relevantes (Américas, Europa, África, Ásia, Oceania).

Exemplo de estrutura:

```
── América ──
  America/Sao_Paulo      (Brasília, UTC-3)
  America/Manaus         (Manaus, UTC-4)
  America/Belem          (Belém, UTC-3)
  America/Fortaleza      (Fortaleza, UTC-3)
  America/New_York       (Nova York, UTC-5/-4)
  America/Chicago        (Chicago, UTC-6/-5)
  America/Denver         (Denver, UTC-7/-6)
  America/Los_Angeles    (Los Angeles, UTC-8/-7)
  America/Argentina/Buenos_Aires (Buenos Aires, UTC-3)
  America/Santiago       (Santiago, UTC-4/-3)
  America/Bogota         (Bogotá, UTC-5)
  America/Lima           (Lima, UTC-5)
── Europa ──
  Europe/Lisbon          (Lisboa, UTC+0/+1)
  Europe/London          (Londres, UTC+0/+1)
  Europe/Paris           (Paris, UTC+1/+2)
  Europe/Berlin          (Berlim, UTC+1/+2)
  Europe/Madrid          (Madrid, UTC+1/+2)
  Europe/Rome            (Roma, UTC+1/+2)
  Europe/Amsterdam       (Amsterdã, UTC+1/+2)
  ...
── África ──
  Africa/Luanda          (Luanda, UTC+1)
  Africa/Maputo          (Maputo, UTC+2)
  ...
── Ásia / Oceania ──
  Asia/Tokyo             (Tóquio, UTC+9)
  Asia/Dubai             (Dubai, UTC+4)
  Australia/Sydney       (Sydney, UTC+10/+11)
  ...
```

**Localização do seletor:** seção de configurações do grupo (`/groups/new` e `/groups/[id]` aba de edição), logo abaixo do campo de descrição.

**Comportamento na criação:** o seletor abre com `America/Sao_Paulo` pré-selecionado. O admin pode alterá-lo antes de salvar. O campo é obrigatório — não é possível criar um grupo sem timezone definido.

**Comportamento na edição:** o seletor exibe o timezone atual do grupo. O admin pode alterá-lo a qualquer momento; a mudança afeta apenas a exibição dos horários, não os valores armazenados de `start_time`/`end_time`.

### 5.3 Frontend — Exibição do Horário

**Lógica de exibição:**

```typescript
function formatMatchTime(timeStr: string, groupTimezone: string): string {
  // Constrói uma data com o horário no fuso do grupo
  // Formata usando Intl.DateTimeFormat com timeZone do grupo
  // Ex: "20:00 (Europe/Berlin)"
}
```

**Indicação de fuso diferente:**

Se `groupTimezone !== Intl.DateTimeFormat().resolvedOptions().timeZone`, exibir o identificador do fuso ao lado do horário:

- Mesmo fuso: `20:00`
- Fuso diferente: `20:00 · Berlim` (nome curto localizado via `Intl.DateTimeFormat`)

Essa indicação aparece em:
- Card de partida na listagem do grupo
- Página de detalhe da partida (`/match/[hash]`)
- Cards na tela Descobrir (`/discover`)
- Dashboard (próximas partidas)

**Sem conversão de fuso:** o horário exibido é **sempre o horário local do evento** (fuso do grupo), não convertido para o fuso do dispositivo. A indicação apenas informa ao usuário qual é a referência.

---

## 6. Impacto em Outras Funcionalidades

| Funcionalidade | Impacto |
|---|---|
| Notificações push | Sem impacto imediato — backend já envia horário como string. Refinamento futuro: formatar com timezone do grupo |
| Recorrência de partidas | Sem impacto — gera `match_date` a partir de lógica de calendário, não de timestamps |
| Exportação / iCal | Sem impacto hoje (não implementado) — futuramente, usar o timezone do grupo no DTSTART |
| Filtro por dia da semana em Descobrir | Sem impacto — filtro é feito no servidor por `match_date` sem conversão de fuso |
| Votação pós-partida | Sem impacto — `vote_open_delay_minutes` é relativo ao `start_time`, sem conversão |

---

## 7. Migração de Dados

```sql
-- 021_add_group_timezone.sql
ALTER TABLE groups
  ADD COLUMN IF NOT EXISTS timezone TEXT NOT NULL DEFAULT 'America/Sao_Paulo';
```

- Idempotente (`IF NOT EXISTS`)
- Grupos existentes recebem `America/Sao_Paulo` via `DEFAULT` da migration — sem intervenção manual e sem quebrar dados
- Grupos novos criados após a migration também partem de `America/Sao_Paulo` pré-selecionado, mas o admin pode escolher outro antes de salvar
- Nenhum dado de partida existente (`match_date`, `start_time`, `end_time`) é alterado

---

## 8. Critérios de Aceite

- [x] Admin pode definir o timezone ao criar um grupo
- [x] Admin pode alterar o timezone nas configurações do grupo
- [x] O timezone é exibido na tela de edição com o valor atual
- [x] Horários de partidas são exibidos no fuso do grupo
- [x] Quando o fuso do grupo difere do dispositivo, o nome do fuso aparece ao lado do horário
- [x] Grupos existentes mantêm `America/Sao_Paulo` e continuam funcionando sem alteração
- [x] Timezone inválido é rejeitado pelo backend com erro 422
- [x] O seletor de timezone é responsivo e usável em mobile

---

## 9. Fora do Escopo (Futuro)

- Conversão automática de timezone via geocodificação de endereço
- Exibição do horário convertido para o fuso do dispositivo do usuário
- Suporte a timezone por partida individual
- Detecção automática de timezone do grupo com base na localização do admin
