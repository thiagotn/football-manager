# PRD 046 — Admin: Alteração Manual de Plano de Assinante

| Campo | Valor |
|---|---|
| **Versão** | 1.0 |
| **Status** | 📋 Proposto — aguardando decisão ou priorização |
| **Autor** | thiagotn |
| **Data** | 2026-05-25 |

---

## 1. Visão Geral

### 1.1 Contexto

A página `/admin/subscriptions` exibe todos os assinantes com dois botões de ação:
- **"Ativar"** — visível apenas quando `status !== 'active'` (ativa um plano inativo)
- **"Cancelar"** — visível quando `status !== 'canceled' && plan !== 'free'`

Isso deixa um gap: assinantes com `status === 'active'` não têm como ter o plano alterado pela UI (ex: upgrade de `basic → pro` ou downgrade `pro → basic`). Casos comuns: migração manual após falha de webhook, cortesia comercial, suporte ao cliente.

O backend já possui o endpoint `PATCH /api/v1/admin/subscriptions/{player_id}` para esta finalidade, e o frontend já tem o modal de ativação — ambos precisam de apenas ajustes pontuais de lógica condicional e UX.

### 1.2 Objetivo

Expor na UI admin a capacidade de alterar o plano de **qualquer** assinante, independente do status atual, sem passar pelo checkout do Stripe.

### 1.3 Proposta de valor

Super admins conseguem corrigir/ajustar planos de assinantes em segundos pela interface, sem precisar chamar a API diretamente via cURL ou Swagger. Reduz tempo de resolução de suporte.

---

## 2. Escopo

### 2.1 Incluído em v1.0

- Botão **"Alterar plano"** na coluna Ações para assinantes com `status === 'active'`
- Generalização do modal existente ("Ativar plano" / "Alterar plano" conforme contexto)
- Inclusão da opção `free` no select de planos do modal (para downgrades manuais)
- Campo `reason` editável no modal (atualmente hardcoded como `manual_admin_override`)

### 2.2 Fora de escopo

- Integração com Stripe para o fluxo manual
- Audit trail persistente no banco de dados (a API já loga via structlog)
- Alteração em massa de planos via bulk action
- Notificação automática para o assinante ao ter o plano alterado

---

## 3. Requisitos Funcionais

**RF-01 — Botão "Alterar plano" para assinantes ativos**

Na tabela desktop (`hidden sm:block`) e nos cards mobile (`sm:hidden`) de `/admin/subscriptions`, exibir um botão `<Pencil size={14} /> Alterar` para itens com `status === 'active'`. O botão deve coexistir com o botão "Cancelar" quando ambos se aplicam, sem overflow na coluna Ações.

**RF-02 — Modal unificado de alteração/ativação**

O modal existente (linhas 421–465 de `+page.svelte`) deve ser reutilizado com título dinâmico:
- `status !== 'active'` → título: "Ativar plano manualmente"
- `status === 'active'` → título: "Alterar plano manualmente"

**RF-03 — Opção de downgrade para plano gratuito**

Adicionar `<option value="free">Grátis (sem cobrança)</option>` no select de plano do modal, permitindo downgrade manual sem cancelar a assinatura no Stripe. Exemplos de uso:
- Assinante pagante que pediu downgrade durante período de contato pós-churn
- Teste de plano que precisa ser revertido para free

**RF-04 — Campo `reason` personalizável**

Exibir no modal um input de texto `<input type="text" placeholder="Motivo (opcional)">` que preenche o campo `reason` do payload da API. Valor padrão (se vazio): `manual_admin_override`. Campo facilita auditoria via logs estruturados do backend.

---

## 4. Requisitos Não-Funcionais

**RNF-01 — Sem mudanças no backend**

O endpoint `PATCH /api/v1/admin/subscriptions/{player_id}` e o repositório `SubscriptionRepository.update_plan` já suportam todos os campos necessários. Nenhuma migration, nenhuma alteração de schema, nenhuma mudança de lógica de negócio no backend.

**RNF-02 — Responsividade**

A coluna de ações já usa `flex items-center gap-2` (linha 368). O novo botão deve usar classes `btn-sm btn-ghost text-xs`, seguindo o padrão de botões admin existentes. Verificar que não há overflow em mobile com dois botões lado a lado.

**RNF-03 — Sem regressão**

Nenhum teste unitário existente pode quebrar. Nenhuma funcionalidade de assinante com `status !== 'active'` pode ser afetada (RF-02 garante coexistência).

---

## 5. Detalhes de Implementação

### Arquivo: `football-frontend/src/routes/admin/subscriptions/+page.svelte`

**Mudança 1 — Importar ícone Pencil (linha 7)**

```svelte
import { CreditCard, AlertTriangle, TrendingUp, Users, ExternalLink, ChevronLeft, ChevronRight, X, XCircle, Pencil } from 'lucide-svelte';
```

**Mudança 2 — Adicionar estado `modalMode` (linha 54)**

```typescript
let modalMode = $state<'activate' | 'change'>('activate');
let modalReason = $state('');
```

**Mudança 3 — Atualizar função `openModal` (linha 138)**

```typescript
function openModal(item: AdminSubscriptionItem, mode: 'activate' | 'change') {
  modalPlayer = item;
  modalMode = mode;
  modalPlan = item.plan === 'free' ? 'basic' : item.plan;
  modalCycle = (item.billing_cycle as 'monthly' | 'yearly') ?? 'monthly';
  modalReason = '';
  modalError = '';
  modalOpen = true;
}
```

**Mudança 4 — Coluna Ações (linha 369)**

Antes:
```svelte
{#if item.status !== 'active'}
  <button onclick={() => openModal(item)} class="btn-sm btn-ghost text-xs">
    <CreditCard size={12} /> Ativar
  </button>
{/if}
```

Depois:
```svelte
{#if item.status !== 'active'}
  <button onclick={() => openModal(item, 'activate')} class="btn-sm btn-ghost text-xs">
    <CreditCard size={12} /> Ativar
  </button>
{/if}
{#if item.status === 'active'}
  <button onclick={() => openModal(item, 'change')} class="btn-sm btn-ghost text-xs">
    <Pencil size={12} /> Alterar
  </button>
{/if}
```

**Mudança 5 — Modal: título dinâmico (linha 428)**

Antes:
```svelte
<h2 class="text-base font-semibold text-white">Ativar plano manualmente</h2>
```

Depois:
```svelte
<h2 class="text-base font-semibold text-white">
  {modalMode === 'activate' ? 'Ativar plano manualmente' : 'Alterar plano manualmente'}
</h2>
```

**Mudança 6 — Select de plano: adicionar opção `free` (linha 439)**

Antes:
```svelte
<select bind:value={modalPlan} class="input w-full">
  <option value="basic">Basic</option>
  <option value="pro">Pro</option>
</select>
```

Depois:
```svelte
<select bind:value={modalPlan} class="input w-full">
  <option value="free">Grátis</option>
  <option value="basic">Basic</option>
  <option value="pro">Pro</option>
</select>
```

**Mudança 7 — Adicionar campo `reason` (após ciclo, linha 451)**

```svelte
<div>
  <label class="block text-xs text-gray-400 mb-1">Motivo (opcional)</label>
  <input
    type="text"
    bind:value={modalReason}
    placeholder="ex: cortesia comercial, webhook falhado..."
    class="input w-full text-xs"
  />
</div>
```

**Mudança 8 — Função `saveModal` — usar `reason` dinâmico (linha 157)**

Antes:
```typescript
await adminApi.updateSubscription(modalPlayer.player_id, {
  plan: modalPlan,
  status: 'active',
  billing_cycle: modalCycle,
  reason: 'manual_admin_override',
});
```

Depois:
```typescript
await adminApi.updateSubscription(modalPlayer.player_id, {
  plan: modalPlan,
  status: 'active',
  billing_cycle: modalCycle,
  reason: modalReason || 'manual_admin_override',
});
```

---

## 6. Critérios de Aceite

- [ ] Assinante com `status === 'active'` exibe botão "Alterar" (ícone Pencil + texto)
- [ ] Assinante com `status !== 'active'` continua exibindo botão "Ativar" (ícone CreditCard + texto)
- [ ] Assinantes com ambas as condições (ex: `status === 'past_due'` e precisa alterar) veem apenas "Ativar"
- [ ] Modal com título correto: "Ativar plano manualmente" vs "Alterar plano manualmente"
- [ ] Select de plano exibe as três opções: Grátis / Basic / Pro
- [ ] Campo Motivo é preenchível; se vazio, payload usa `manual_admin_override`
- [ ] Após salvar, a tabela recarrega refletindo o novo plano, status e ciclo
- [ ] Layout mobile (cards) exibe o botão "Alterar" com o mesmo padrão
- [ ] Dois botões lado a lado (Ativar/Alterar + Cancelar) não causam overflow
- [ ] Nenhum teste unitário existente quebrado
- [ ] Funcionalidade de cancelar continua intacta

---

## 7. Plano de Rollout

### Fase 1 — Implementação e Teste Local

1. Implementar as 8 mudanças listadas na seção 5
2. Testar localmente com `docker compose up --build`
3. Verificar a lista de critérios de aceite manualmente no navegador
4. Rodar testes unitários (se houver): `docker compose run --rm api poetry run pytest tests/unit/ -q`

### Fase 2 — Verificação em Staging

1. Deploy para staging (`develop` branch)
2. Testar contra dados reais (ou dados de staging)
3. Verificar latência e nenhuma regressão em outras seções admin

### Fase 3 — Deploy em Produção

1. Merge para `main` após aprovação
2. Deploy automático via GitHub Actions
3. Monitorar logs de erro em produção (não haverá novos endpoints, então risco baixo)

---

## 8. Decisões em Aberto

| Decisão | Opções | Impacto |
|---------|--------|--------|
| Adicionar campo `reason` como field autoedit no modal? | Sim (RF-04) · Não (deixar hardcoded) | Auditoria: com field é mais flexível; sem field é mais rápido |
| Permitir downgrade para `free`? | Sim (RF-03) · Não (ocultar opção) | UX: com opção reduz casos de cancelamento; sem opção é mais conservador |
| Validação de transição de plano (ex: pro→free só com aprovação)? | Sim · Não | Controle: sim requer lógica adicional; não é mais ágil para suporte |

**Recomendação**: Implementar conforme RF-01 a RF-04 (inclua campo `reason` e opção `free`). Sem validações de transição em v1.0 — podem ser adicionadas em iterações futuras se necessário.

---

## 9. Anexos

### A. Endpoints backend (sem mudança)

```http
PATCH /api/v1/admin/subscriptions/{player_id}
Content-Type: application/json
Authorization: Bearer <admin_token>

{
  "plan": "pro",
  "status": "active",
  "billing_cycle": "monthly",
  "reason": "cortesia comercial"
}
```

Resposta esperada:
```json
{
  "status": "ok",
  "plan": "pro"
}
```

### B. Referência de componentes Svelte

- `Modal.svelte` — existente mas não usado aqui (modal é inline)
- `ConfirmDialog.svelte` — usado para cancelamento
- `PageBackground.svelte` — wrapper obrigatório da página

### C. Teste manual checklist

```
[ ] Login como admin (+5511999990000 / admin123)
[ ] Ir para /admin/subscriptions
[ ] Encontrar um assinante com status="active"
[ ] Clicar em "Alterar"
[ ] Selecionar plano "Basic" + ciclo "Mensal" + motivo "Teste de downgrade"
[ ] Confirmar
[ ] Verificar que a linha atualiza com o novo plano
[ ] Clicar em um assinante inativo
[ ] Verificar que o botão é "Ativar" (não "Alterar")
[ ] Verificar que em mobile os botões não causam overflow
```
