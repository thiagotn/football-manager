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

## Backend — Padrões

### Limite de plano por recurso
- Verificar sempre o plano real da `PlayerSubscription` (nunca hardcodar limites)
- Retornar `403` com `detail="PLAN_LIMIT_EXCEEDED"` via `PlanLimitError`
- Admins globais (`PlayerRole.ADMIN`) são sempre isentos de limites

### Migrations
- Numeradas sequencialmente: `NNN_descricao.sql` em `football-api/migrations/`
- Sempre usar `IF NOT EXISTS` / `ON CONFLICT DO NOTHING` para idempotência
- Aplicadas automaticamente no startup via `app/db/migrate.py`

### Rotas da API
- Sempre em inglês, lowercase, hyphen-separated
- Nunca em português

---

## Regras Gerais

- **Commits**: nunca commitar/pushar automaticamente. Implementar, informar e aguardar validação do usuário.
- **Mobile-first**: este app é primariamente mobile. Toda implementação deve ser responsiva.
- **Botões admin**: sempre ícone + texto (`btn-sm btn-ghost`). Ex: `<Trash2 size={14} /> Excluir`
- **Confirmações destrutivas**: sempre usar `ConfirmDialog`, nunca `window.confirm()`
- **Rotas**: só alfanumérico + underscores nos params do SvelteKit
