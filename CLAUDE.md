# Padrões do Projeto — rachao.app

## Stack
- **Backend**: FastAPI + SQLAlchemy 2 async + PostgreSQL (asyncpg)
- **Frontend**: SvelteKit 2 + Svelte 5 + Tailwind CSS
- **Testes E2E**: Playwright + pytest (`football-e2e/`)

---

## Frontend — Padrões de Página

### Cabeçalho padrão de página

Todas as páginas devem usar este padrão de cabeçalho, sem exceção:

```svelte
<main class="relative z-10 max-w-7xl mx-auto px-4 py-8">
  <div class="flex items-center justify-between mb-6">
    <div>
      <h1 class="text-2xl font-bold text-white flex items-center gap-2">
        <IcôneRelevante size={24} class="text-primary-400" /> Título da Página
      </h1>
      <p class="text-sm text-white/60 mt-0.5">Subtítulo descritivo opcional</p>
    </div>
    <!-- Botão de ação principal (ex: "Novo Grupo") vai aqui, se houver -->
  </div>
  ...
</main>
```

**Regras:**
- `h1` sempre `text-2xl font-bold text-white flex items-center gap-2`
- Ícone sempre `size={24} class="text-primary-400"` — escolher ícone representativo da seção (Lucide)
- Subtítulo sempre `text-sm text-white/60 mt-0.5`
- Wrapper sempre `flex items-center justify-between mb-6` para acomodar botão de ação
- Container sempre `max-w-7xl mx-auto px-4 py-8` (exceto páginas de formulário/detalhe)
- Envolto sempre em `<PageBackground>`

**Exemplos existentes:**
| Página | Ícone |
|--------|-------|
| `/groups` | `<Trophy size={24} />` |
| `/players` | `<Users size={24} />` |
| `/matches` | `<Calendar size={24} />` |
| `/profile` | `<User size={24} />` |

---

### Layout Desktop (duas colunas)

Páginas de configuração/detalhe com múltiplas seções devem usar grid two-col no desktop:

```svelte
<!-- Container menor para páginas de conta/detalhe -->
<main class="relative z-10 max-w-4xl mx-auto px-4 py-8">
  <!-- cabeçalho padrão aqui -->

  <div class="grid grid-cols-1 lg:grid-cols-2 gap-6 items-start">
    <!-- Coluna esquerda: informações principais -->
    <div class="space-y-6">
      <!-- cards -->
    </div>
    <!-- Coluna direita: ações/configurações -->
    <div class="space-y-6">
      <!-- cards -->
    </div>
  </div>
</main>
```

**Quando usar `max-w-4xl` em vez de `max-w-7xl`:**
- Páginas de conta/perfil (`/profile`, `/account/*`)
- Páginas de formulário ou detalhe que não precisam de largura total

---

## Frontend — i18n

### Fluxo obrigatório ao adicionar qualquer texto visível

Sempre que adicionar ou alterar texto visível ao usuário:

1. Adicionar a chave em **todos os 3 arquivos**:
   - `football-frontend/messages/pt-BR.json`
   - `football-frontend/messages/en.json`
   - `football-frontend/messages/es.json`
2. Usar `$t('chave')` no template — nunca string literal

```svelte
<!-- ❌ Errado -->
<p>Confirmar presença</p>

<!-- ✅ Correto -->
<p>{$t('match.confirm_attendance')}</p>
```

### Estrutura das chaves

Prefixo pelo contexto da página ou domínio:

| Prefixo | Uso |
|---------|-----|
| `login.*` | Tela de login e recuperação de senha |
| `register.*` | Fluxo de cadastro |
| `match.*` | Página de partida |
| `groups.*` | Listagem e gestão de grupos |
| `account.*` | Conta / assinatura |
| `plans.*` | Planos e preços |
| `plan.*` | Nomes e bullets de planos (`plans.ts`) |
| `auth.*` | Erros de autenticação compartilhados |

### Planos (`src/lib/plans.ts`)

Os campos `name` e `highlights` de `PlanConfig` armazenam **chaves i18n**, não strings literais. Sempre usar `$t(plan.name)` e `$t(item)` nos templates — nunca `{plan.name}` diretamente.

---

## Frontend — Svelte 5

### `$effect` vs `onMount`

- **`onMount`**: correto apenas quando a lógica deve rodar ao montar um **componente filho dedicado**
- **`$effect`**: correto quando a lógica depende de uma **variável de estado** (ex: step de um fluxo multi-etapa)

```typescript
// ✅ Correto para lógica condicional por step
$effect(() => {
  if (currentStep !== 'otp') return;
  // lógica que só deve rodar quando o step for 'otp'
});

// ❌ onMount não re-executa quando o step muda
onMount(() => {
  // roda uma única vez — antes do step OTP aparecer
});
```

---

## Backend — Estrutura de um novo endpoint

### Localização dos arquivos

| Camada | Localização |
|--------|-------------|
| Router | `football-api/app/api/v1/routers/<dominio>.py` |
| Repository | `football-api/app/db/repositories/<dominio>_repo.py` |
| Schema (request/response) | `football-api/app/schemas/<dominio>.py` |
| Model (ORM) | `football-api/app/models/<dominio>.py` |
| Migration | `football-api/migrations/NNN_descricao.sql` |

### Padrão de imports num router

```python
from app.core.dependencies import DB, CurrentPlayer, AdminPlayer
from app.core.exceptions import ConflictError, NotFoundError, ForbiddenError, PlanLimitError
from app.models.player import PlayerRole
```

### Dependências FastAPI

| Dependência | Uso |
|-------------|-----|
| `DB` | Sessão async do banco |
| `CurrentPlayer` | Jogador autenticado (qualquer role) |
| `AdminPlayer` | Restringe rota a super-admins |

### Erros padrão

| Exceção | HTTP | Quando usar |
|---------|------|-------------|
| `NotFoundError("msg")` | 404 | Recurso não encontrado |
| `ForbiddenError("msg")` | 403 | Sem permissão |
| `ConflictError("msg")` | 409 | Conflito de unicidade |
| `PlanLimitError()` | 403 | Limite do plano atingido (`detail="PLAN_LIMIT_EXCEEDED"`) |
| `ValidationError("msg")` | 422 | Validação de negócio |
| `UnauthorizedError()` | 401 | Não autenticado |

### Limite de plano por recurso
- Verificar sempre o plano real da `PlayerSubscription` (nunca hardcodar limites)
- `PlayerRole.ADMIN` (super admin) é **sempre isento** de limites de plano

---

## Backend — Migrations

- Numeradas sequencialmente: `NNN_descricao.sql` em `football-api/migrations/`
- Sempre usar `IF NOT EXISTS` / `ON CONFLICT DO NOTHING` para idempotência
- Aplicadas automaticamente no startup via `app/db/migrate.py`
- Verificar o número da última migration antes de criar uma nova: `ls football-api/migrations/`

---

## Backend — Testes unitários

### Estrutura

```
football-api/tests/unit/
  test_security.py
  routers/          ← testes de endpoints (usa httpx ASGITransport)
  services/         ← testes de serviços (ex: twilio_verify)
```

### Rodar localmente

```bash
cd football-api
docker compose run --rm api poetry run pytest tests/unit/ -q
```

### Regras

- **Todo novo endpoint** deve ter ao menos: 1 teste caminho feliz + testes dos erros esperados
- **Sempre rodar antes de commitar** qualquer alteração no backend
- Usar `pytest-mock` para mockar dependências externas (Twilio, Stripe, etc.)
- Não mockar o banco em testes de router — usar ASGITransport com DB real ou fixtures

---

## Backend — Rotas da API

- Sempre em inglês, lowercase, hyphen-separated
- Nunca em português
- Prefixo: `/api/v1/`

---

## Regras Gerais

- **Commits**: nunca commitar/pushar automaticamente. Implementar, informar e aguardar validação do usuário.
- **Mobile-first**: este app é primariamente mobile. Toda implementação deve ser responsiva.
- **Botões admin**: sempre ícone + texto (`btn-sm btn-ghost`). Ex: `<Trash2 size={14} /> Excluir`
- **Confirmações destrutivas**: sempre usar `ConfirmDialog`, nunca `window.confirm()`
- **Rotas**: só alfanumérico + underscores nos params do SvelteKit
