# PRD — Migração de UUID v4 para UUID v7 nas PKs

## 1. Contexto

Todas as entidades principais do rachao.app usam **UUID v4** como chave primária (`app/models/base.py`), gerado via `uuid.uuid4()` do Python. UUID v4 é completamente aleatório, o que garante unicidade global sem coordenação, mas introduz um problema estrutural de performance em bancos de dados relacionais à medida que as tabelas crescem.

A exceção atual é `push_subscription`, que usa `Integer` com auto-increment por ser uma tabela técnica.

---

## 2. Problema

### UUID v4 e fragmentação de índice B-tree

O PostgreSQL armazena índices como árvores B-tree ordenadas. Quando um novo registro é inserido, o banco precisa encontrar sua posição correta no índice pelo valor da PK.

Com UUID v4 (aleatório):
- Cada novo registro cai em uma **posição aleatória** no índice
- Isso força o PostgreSQL a fazer **page splits** frequentes — uma página cheia é dividida em duas para acomodar o novo valor
- O resultado é um índice **fragmentado**, com páginas parcialmente vazias e má localidade de cache
- Leituras que percorrem o índice têm mais **cache misses** porque os dados não seguem uma ordem previsível

Com PKs monotonicamente crescentes (inteiros ou UUID v7):
- Novos registros sempre vão para o **final do índice**
- Sem page splits — as páginas são preenchidas sequencialmente
- Melhor localidade de cache e menor overhead de I/O

### Impacto estimado

O problema é mensurável principalmente em:
- Tabelas com **dezenas de milhões de linhas** e alta taxa de escrita concorrente
- Índices em FKs (`group_id`, `player_id`, `match_id`) que também sofrem fragmentação

No volume atual do rachao.app, o impacto é desprezível. Este PRD documenta a melhoria para adoção futura, antes que o crescimento torne a migração mais custosa.

**Referências:**
- https://gist.github.com/rponte/bf362945a1af948aa04b587f8ff332f8
- https://buildkite.com/resources/blog/goodbye-integers-hello-uuids/

---

## 3. Solução Proposta: UUID v7

O **UUID v7** (RFC 9562, ratificado em abril de 2024) resolve o problema mantendo todas as vantagens do UUID v4:

| Característica         | UUID v4       | UUID v7            |
|------------------------|---------------|--------------------|
| Unicidade global       | ✅             | ✅                  |
| Sem coordenação central| ✅             | ✅                  |
| Ordenável por tempo    | ❌             | ✅ (prefixo ms)     |
| Fragmentação de índice | Alta           | Mínima              |
| Tamanho                | 16 bytes      | 16 bytes           |
| Compatibilidade UUID   | ✅             | ✅ (mesmo formato)  |
| PostgreSQL nativo      | `gen_random_uuid()` | `uuidv7()` (PG 17+) |

O UUID v7 usa os primeiros 48 bits para timestamp em milissegundos e o restante para aleatoriedade, garantindo unicidade mesmo com múltiplas gerações no mesmo milissegundo.

---

## 4. Escopo

### Tabelas afetadas (PKs a migrar)

Todas as entidades que herdam de `Base` em `app/models/base.py`:

| Tabela          | Modelo            |
|-----------------|-------------------|
| `players`       | `Player`          |
| `groups`        | `Group`           |
| `matches`       | `Match`           |
| `attendances`   | `Attendance`      |
| `invites`       | `Invite`          |
| `waitlist`      | `WaitlistEntry`   |
| `match_votes`   | `MatchVote`       |
| `teams`         | `Team`            |
| `app_reviews`   | `AppReview`       |
| `subscriptions` | `PlayerSubscription` |
| `invoices`      | `Invoice`         |

### Fora do escopo

- `push_subscription` — mantém `Integer` (tabela técnica, sem necessidade de UUID)
- FKs não precisam ser migradas: os valores existentes continuam válidos como UUID v7 é formato-compatível com v4

---

## 5. Abordagem de Implementação

### Fase 1 — Novos registros (zero downtime, sem migrar dados)

**Backend Python:**

Instalar o pacote `uuid7`:
```
poetry add uuid7
```

Atualizar `app/models/base.py`:
```python
from uuid7 import uuid7

class Base(DeclarativeBase):
    id: Mapped[uuid.UUID] = mapped_column(
        UUID(as_uuid=True), primary_key=True, default=uuid7
    )
```

**PostgreSQL (opcional, para geração no banco):**

PostgreSQL 17+ suporta `uuidv7()` nativamente. Para versões anteriores, a geração no Python é suficiente.

**Resultado da Fase 1:**
- Novos registros passam a usar UUID v7
- Registros existentes (UUID v4) continuam intactos e coexistem sem conflito
- Índices ficam gradualmente mais ordenados conforme novos dados entram

### Fase 2 — Migração dos dados existentes (opcional, futuro)

Reescrever as PKs de UUID v4 para UUID v7 nos registros existentes é uma operação **de alto risco e impacto zero na performance** a curto prazo. Recomendável apenas se:
- O banco tiver crescido significativamente e a fragmentação for mensurável via `pg_stat_user_indexes`
- Houver uma janela de manutenção disponível

A migração exigiria:
1. Criar nova coluna `id_v7`
2. Preencher com novos UUIDs v7 para cada linha
3. Atualizar todas as FKs em cascata
4. Renomear colunas

Esta fase **não está no escopo imediato** deste PRD.

---

## 6. Riscos e Mitigações

| Risco | Mitigação |
|-------|-----------|
| UUID v7 não é suportado pela lib padrão `uuid` do Python | Usar `uuid7` (PyPI) ou implementar geração manual via `os.urandom` + timestamp |
| SQLAlchemy pode não reconhecer UUID v7 como UUID válido | UUID v7 é formato-compatível com UUID padrão — funciona com `UUID(as_uuid=True)` sem alteração |
| Supabase/PostgreSQL rejeitar UUID v7 | UUID v7 é um UUID válido pelo formato `xxxxxxxx-xxxx-7xxx-...` — aceito por qualquer coluna `UUID` |
| Dados existentes (v4) misturados com novos (v7) no mesmo índice | Convivência é inofensiva; a melhoria de performance é progressiva |

---

## 7. Critérios de Aceitação

- [ ] `uuid7` adicionado como dependência em `pyproject.toml`
- [ ] `app/models/base.py` atualizado para usar `uuid7` como `default`
- [ ] Todos os testes unitários continuam passando
- [ ] Novos registros criados via API têm PKs no formato UUID v7 (`xxxxxxxx-xxxx-7xxx-...`)
- [ ] Registros existentes com UUID v4 não são afetados

---

## 8. Quando Priorizar

Este PRD deve ser priorizado quando:
- Qualquer tabela principal ultrapassar **1 milhão de linhas**, ou
- O monitoramento (Grafana/Prometheus) indicar degradação em queries de escrita, ou
- Houver planejamento de crescimento acelerado de usuários

A Fase 1 é de **baixo risco e baixo esforço** (alteração em um único arquivo) e pode ser feita a qualquer momento como melhoria proativa.
