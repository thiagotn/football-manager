<script lang="ts">
  import { goto } from '$app/navigation';
  import { page } from '$app/stores';
  import { authStore, isAdmin } from '$lib/stores/auth';
  import { admin as adminApi } from '$lib/api';
  import type { AdminSubscriptionSummary, AdminSubscriptionListResponse, AdminSubscriptionItem } from '$lib/api';
  import { CreditCard, AlertTriangle, TrendingUp, Users, ExternalLink, ChevronLeft, ChevronRight, X, XCircle } from 'lucide-svelte';
  import PageBackground from '$lib/components/PageBackground.svelte';
  import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';

  // ── State ──────────────────────────────────────────────────────
  let summary = $state<AdminSubscriptionSummary | null>(null);
  let list = $state<AdminSubscriptionListResponse | null>(null);
  let loadingSummary = $state(true);
  let loadingList = $state(true);
  let error = $state('');

  let filterStatus = $state($page.url.searchParams.get('status') ?? '');
  let filterPlan = $state('');
  let currentPage = $state(1);

  // Confirmação de cancelamento
  let cancelDialogOpen = $state(false);
  let cancelTarget = $state<AdminSubscriptionItem | null>(null);
  let canceling = $state(false);

  function openCancelDialog(item: AdminSubscriptionItem) {
    cancelTarget = item;
    cancelDialogOpen = true;
  }

  async function confirmCancel() {
    if (!cancelTarget) return;
    canceling = true;
    try {
      await adminApi.cancelSubscription(cancelTarget.player_id);
      cancelDialogOpen = false;
      cancelTarget = null;
      await Promise.all([loadSummary(), loadList()]);
    } catch {
      error = 'Erro ao cancelar assinatura.';
      cancelDialogOpen = false;
    } finally {
      canceling = false;
    }
  }

  // Modal de ativação manual
  let modalOpen = $state(false);
  let modalPlayer = $state<AdminSubscriptionItem | null>(null);
  let modalPlan = $state('basic');
  let modalCycle = $state<'monthly' | 'yearly'>('monthly');
  let modalSaving = $state(false);
  let modalError = $state('');

  // ── Auth guard ─────────────────────────────────────────────────
  let loaded = false;
  $effect(() => {
    if ($authStore.loading) return;
    if (!$isAdmin) { goto('/', { replaceState: true }); return; }
    if (loaded) return;
    loaded = true;
    loadSummary();
    loadList();
  });

  // ── Reload list when filters change ───────────────────────────
  $effect(() => {
    filterStatus; filterPlan; currentPage;
    if (loaded) loadList();
  });

  async function loadSummary() {
    loadingSummary = true;
    try {
      summary = await adminApi.getSubscriptionSummary();
    } catch {
      error = 'Não foi possível carregar o resumo.';
    } finally {
      loadingSummary = false;
    }
  }

  async function loadList() {
    loadingList = true;
    try {
      list = await adminApi.getSubscriptions({
        status: filterStatus || undefined,
        plan: filterPlan || undefined,
        page: currentPage,
        page_size: 20,
      });
    } catch {
      error = 'Não foi possível carregar as assinaturas.';
    } finally {
      loadingList = false;
    }
  }

  // ── Helpers ────────────────────────────────────────────────────
  function fmtDate(iso: string | null): string {
    if (!iso) return '—';
    return new Date(iso).toLocaleDateString('pt-BR', { day: '2-digit', month: '2-digit', year: 'numeric' });
  }

  function fmtPlan(plan: string, cycle: string): string {
    const labels: Record<string, string> = { basic: 'Básico', pro: 'Pro', free: 'Grátis' };
    const cycleLabel = cycle === 'yearly' ? 'Anual' : 'Mensal';
    return `${labels[plan] ?? plan} ${cycleLabel}`;
  }

  function fmtMrr(cents: number): string {
    return (cents / 100).toLocaleString('pt-BR', { style: 'currency', currency: 'BRL', minimumFractionDigits: 2 });
  }

  function statusBadge(status: string): string {
    if (status === 'active')   return 'bg-green-500/20 text-green-300 border border-green-500/30';
    if (status === 'past_due') return 'bg-yellow-500/20 text-yellow-300 border border-yellow-500/30 animate-pulse';
    if (status === 'canceled') return 'bg-gray-500/20 text-gray-400 border border-gray-500/30';
    return 'bg-gray-500/20 text-gray-400';
  }

  function statusLabel(status: string): string {
    if (status === 'active')   return 'Ativo';
    if (status === 'past_due') return 'Pag. Pendente';
    if (status === 'canceled') return 'Cancelado';
    return status;
  }

  function stripeUrl(customerId: string | null): string {
    if (!customerId) return '';
    return `https://dashboard.stripe.com/customers/${customerId}`;
  }

  const totalPages = $derived(list ? Math.ceil(list.total / 20) : 1);

  // ── Modal ──────────────────────────────────────────────────────
  function openModal(item: AdminSubscriptionItem) {
    modalPlayer = item;
    modalPlan = item.plan === 'free' ? 'basic' : item.plan;
    modalCycle = (item.billing_cycle as 'monthly' | 'yearly') ?? 'monthly';
    modalError = '';
    modalOpen = true;
  }

  function closeModal() {
    modalOpen = false;
    modalPlayer = null;
    modalError = '';
  }

  async function saveModal() {
    if (!modalPlayer) return;
    modalSaving = true;
    modalError = '';
    try {
      await adminApi.updateSubscription(modalPlayer.player_id, {
        plan: modalPlan,
        status: 'active',
        billing_cycle: modalCycle,
        reason: 'manual_admin_override',
      });
      closeModal();
      await Promise.all([loadSummary(), loadList()]);
    } catch {
      modalError = 'Erro ao salvar. Tente novamente.';
    } finally {
      modalSaving = false;
    }
  }
</script>

<svelte:head><title>Assinaturas — Admin rachao.app</title></svelte:head>

<PageBackground>
<main class="relative z-10 max-w-7xl mx-auto px-4 py-8 space-y-6">

  <!-- Header -->
  <div class="flex items-center justify-between mb-6">
    <div>
      <h1 class="text-2xl font-bold text-white flex items-center gap-2">
        <CreditCard size={24} class="text-primary-400" /> Assinaturas
      </h1>
      <p class="text-sm text-white/60 mt-0.5">Gestão de planos e assinantes da plataforma</p>
    </div>
  </div>

  {#if error}
    <div class="alert-error">{error}</div>
  {/if}

  <!-- Alerta past_due -->
  {#if summary && summary.past_due > 0}
    <button
      onclick={() => { filterStatus = 'past_due'; currentPage = 1; }}
      class="w-full flex items-center gap-3 px-4 py-3 rounded-xl bg-yellow-500/10 border border-yellow-500/30 text-yellow-300 text-sm hover:bg-yellow-500/20 transition-colors text-left">
      <AlertTriangle size={16} class="shrink-0" />
      <span><strong>{summary.past_due}</strong> assinatura{summary.past_due > 1 ? 's' : ''} com pagamento pendente</span>
      <ChevronRight size={14} class="ml-auto shrink-0" />
    </button>
  {/if}

  <!-- Cards de resumo -->
  {#if loadingSummary}
    <div class="grid grid-cols-2 sm:grid-cols-4 gap-4">
      {#each [1,2,3,4] as _}
        <div class="card p-5 animate-pulse">
          <div class="h-8 bg-gray-700 rounded w-1/2 mx-auto mb-2"></div>
          <div class="h-3 bg-gray-700 rounded w-2/3 mx-auto"></div>
        </div>
      {/each}
    </div>
  {:else if summary}
    <div class="grid grid-cols-2 sm:grid-cols-4 gap-4">
      <div class="card p-5 text-center">
        <div class="flex items-center justify-center mb-1"><TrendingUp size={14} class="text-emerald-400 opacity-70" /></div>
        <p class="text-3xl font-bold text-emerald-400">{summary.active}</p>
        <p class="text-xs text-gray-400 mt-1">Ativos</p>
      </div>
      <div class="card p-5 text-center">
        <div class="flex items-center justify-center mb-1"><AlertTriangle size={14} class="{summary.past_due > 0 ? 'text-yellow-400' : 'text-gray-500'} opacity-70" /></div>
        <p class="text-3xl font-bold {summary.past_due > 0 ? 'text-yellow-400' : 'text-gray-500'}">{summary.past_due}</p>
        <p class="text-xs text-gray-400 mt-1">Pag. Pendente</p>
      </div>
      <div class="card p-5 text-center">
        <div class="flex items-center justify-center mb-1"><Users size={14} class="text-gray-400 opacity-70" /></div>
        <p class="text-3xl font-bold text-gray-400">{summary.free}</p>
        <p class="text-xs text-gray-400 mt-1">Grátis</p>
      </div>
      <div class="card p-5 text-center col-span-2 sm:col-span-1">
        <div class="flex items-center justify-center mb-1"><CreditCard size={14} class="text-primary-400 opacity-70" /></div>
        <p class="text-2xl font-bold text-primary-400">{fmtMrr(summary.mrr_cents)}</p>
        <p class="text-xs text-gray-400 mt-1">MRR estimado</p>
      </div>
    </div>

    <!-- Breakdown por plano -->
    {#if summary.breakdown.length > 0}
      <div class="card p-4 space-y-2">
        <p class="text-xs font-semibold text-gray-400 uppercase tracking-wide mb-3">Distribuição por plano</p>
        {#each summary.breakdown as b}
          {@const pct = summary.active > 0 ? Math.round((b.count / summary.active) * 100) : 0}
          <div class="flex items-center gap-3">
            <span class="text-xs text-gray-300 w-28 shrink-0">{fmtPlan(b.plan, b.billing_cycle)}</span>
            <div class="flex-1 bg-gray-700 rounded-full h-1.5">
              <div class="bg-primary-500 h-1.5 rounded-full" style="width: {pct}%"></div>
            </div>
            <span class="text-xs text-gray-400 w-12 text-right">{b.count} ({pct}%)</span>
          </div>
        {/each}
      </div>
    {/if}
  {/if}

  <!-- Filtros -->
  <div class="flex items-center gap-2 overflow-x-auto pb-1">
    <span class="text-xs text-gray-400 shrink-0">Status:</span>
    {#each [['', 'Todos'], ['active', 'Ativos'], ['past_due', 'Pag. Pendente'], ['canceled', 'Cancelados']] as [val, label]}
      <button
        onclick={() => { filterStatus = val; currentPage = 1; }}
        class="px-3 py-1.5 rounded-full text-xs font-medium whitespace-nowrap transition-colors shrink-0
          {filterStatus === val ? 'bg-primary-600 text-white' : 'bg-white/10 text-gray-300 hover:bg-white/20'}">
        {label}
      </button>
    {/each}
    <span class="text-xs text-gray-400 shrink-0 ml-2">Plano:</span>
    {#each [['', 'Todos'], ['basic', 'Basic'], ['pro', 'Pro']] as [val, label]}
      <button
        onclick={() => { filterPlan = val; currentPage = 1; }}
        class="px-3 py-1.5 rounded-full text-xs font-medium whitespace-nowrap transition-colors shrink-0
          {filterPlan === val ? 'bg-primary-600 text-white' : 'bg-white/10 text-gray-300 hover:bg-white/20'}">
        {label}
      </button>
    {/each}
  </div>

  <!-- Lista mobile / Tabela desktop -->
  {#if loadingList}
    <div class="space-y-3">
      {#each [1,2,3] as _}
        <div class="card p-4 animate-pulse">
          <div class="h-4 bg-gray-700 rounded w-1/3 mb-2"></div>
          <div class="h-3 bg-gray-700 rounded w-1/2"></div>
        </div>
      {/each}
    </div>
  {:else if list && list.items.length === 0}
    <div class="card p-8 text-center text-gray-400 text-sm">Nenhuma assinatura encontrada.</div>
  {:else if list}

    <!-- Mobile: cards -->
    <div class="sm:hidden space-y-3">
      {#each list.items as item}
        <div class="card p-4 space-y-2">
          <div class="flex items-start justify-between gap-2">
            <div>
              <p class="text-sm font-medium text-white">{item.player_name}</p>
              <p class="text-xs text-gray-400">{fmtPlan(item.plan, item.billing_cycle)}</p>
            </div>
            <span class="text-xs px-2 py-0.5 rounded-full shrink-0 {statusBadge(item.status)}">
              {statusLabel(item.status)}
            </span>
          </div>
          <p class="text-xs text-gray-400">Vence: {fmtDate(item.current_period_end)}</p>
          {#if item.status === 'past_due' && item.grace_period_end}
            <p class="text-xs text-yellow-400">Tolerância até: {fmtDate(item.grace_period_end)}</p>
          {/if}
          <div class="flex gap-2 mt-1 flex-wrap">
            {#if item.status !== 'active'}
              <button onclick={() => openModal(item)} class="btn-sm btn-ghost text-xs">
                <CreditCard size={12} /> Ativar plano
              </button>
            {/if}
            {#if item.status !== 'canceled' && item.plan !== 'free'}
              <button onclick={() => openCancelDialog(item)} class="btn-sm btn-ghost text-xs text-red-400 hover:text-red-300">
                <XCircle size={12} /> Cancelar
              </button>
            {/if}
          </div>
        </div>
      {/each}
    </div>

    <!-- Desktop: tabela -->
    <div class="hidden sm:block card overflow-hidden">
      <table class="w-full text-sm">
        <thead>
          <tr class="border-b border-gray-700 text-xs text-gray-400 uppercase tracking-wide">
            <th class="text-left px-4 py-3">Jogador</th>
            <th class="text-left px-4 py-3">Plano</th>
            <th class="text-left px-4 py-3">Status</th>
            <th class="text-left px-4 py-3 hidden md:table-cell">Vencimento</th>
            <th class="text-left px-4 py-3 hidden md:table-cell">Tolerância</th>
            <th class="text-left px-4 py-3 hidden lg:table-cell">Stripe</th>
            <th class="px-4 py-3"></th>
          </tr>
        </thead>
        <tbody>
          {#each list.items as item}
            <tr class="border-b border-gray-700/50 hover:bg-white/5 transition-colors">
              <td class="px-4 py-3 text-white font-medium">{item.player_name}</td>
              <td class="px-4 py-3 text-gray-300">{fmtPlan(item.plan, item.billing_cycle)}</td>
              <td class="px-4 py-3">
                <span class="text-xs px-2 py-0.5 rounded-full {statusBadge(item.status)}">
                  {statusLabel(item.status)}
                </span>
              </td>
              <td class="px-4 py-3 text-gray-400 hidden md:table-cell">{fmtDate(item.current_period_end)}</td>
              <td class="px-4 py-3 hidden md:table-cell">
                {#if item.status === 'past_due' && item.grace_period_end}
                  <span class="text-yellow-400 text-xs">{fmtDate(item.grace_period_end)}</span>
                {:else}
                  <span class="text-gray-600">—</span>
                {/if}
              </td>
              <td class="px-4 py-3 hidden lg:table-cell">
                {#if item.gateway_customer_id}
                  <a href={stripeUrl(item.gateway_customer_id)} target="_blank" rel="noopener"
                    class="text-xs text-primary-400 hover:text-primary-300 flex items-center gap-1 transition-colors">
                    {item.gateway_customer_id.slice(0, 14)}…
                    <ExternalLink size={11} />
                  </a>
                {:else}
                  <span class="text-gray-600">—</span>
                {/if}
              </td>
              <td class="px-4 py-3">
                <div class="flex items-center gap-2">
                  {#if item.status !== 'active'}
                    <button onclick={() => openModal(item)} class="btn-sm btn-ghost text-xs">
                      <CreditCard size={12} /> Ativar
                    </button>
                  {/if}
                  {#if item.status !== 'canceled' && item.plan !== 'free'}
                    <button onclick={() => openCancelDialog(item)} class="btn-sm btn-ghost text-xs text-red-400 hover:text-red-300">
                      <XCircle size={12} /> Cancelar
                    </button>
                  {/if}
                </div>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>

    <!-- Paginação -->
    {#if totalPages > 1}
      <div class="flex items-center justify-between text-sm text-gray-400">
        <span>Página {currentPage} de {totalPages} · {list.total} assinantes</span>
        <div class="flex gap-2">
          <button
            disabled={currentPage <= 1}
            onclick={() => currentPage--}
            class="btn-sm btn-ghost disabled:opacity-30">
            <ChevronLeft size={14} /> Anterior
          </button>
          <button
            disabled={currentPage >= totalPages}
            onclick={() => currentPage++}
            class="btn-sm btn-ghost disabled:opacity-30">
            Próxima <ChevronRight size={14} />
          </button>
        </div>
      </div>
    {/if}
  {/if}

</main>
</PageBackground>

<ConfirmDialog
  bind:open={cancelDialogOpen}
  message="Cancelar a assinatura de {cancelTarget?.player_name}? O plano voltará para Grátis imediatamente e a assinatura será cancelada no Stripe."
  confirmLabel={canceling ? 'Cancelando…' : 'Cancelar assinatura'}
  danger={true}
  onConfirm={confirmCancel}
/>

<!-- Modal de ativação manual -->
{#if modalOpen && modalPlayer}
  <button class="fixed inset-0 z-40 bg-black/60" onclick={closeModal} aria-label="Fechar"></button>

  <!-- Mobile: bottom sheet / Desktop: modal centralizado -->
  <div class="fixed z-50 inset-x-0 bottom-0 sm:inset-0 sm:flex sm:items-center sm:justify-center">
    <div class="bg-gray-900 border border-gray-700 rounded-t-2xl sm:rounded-2xl p-6 w-full sm:max-w-md space-y-4">
      <div class="flex items-center justify-between">
        <h2 class="text-base font-semibold text-white">Ativar plano manualmente</h2>
        <button onclick={closeModal} class="p-1.5 rounded-lg hover:bg-white/10 transition-colors text-gray-400">
          <X size={18} />
        </button>
      </div>

      <p class="text-sm text-gray-400">Jogador: <span class="text-white font-medium">{modalPlayer.player_name}</span></p>

      <div class="space-y-3">
        <div>
          <label class="block text-xs text-gray-400 mb-1">Plano</label>
          <select bind:value={modalPlan} class="input w-full">
            <option value="basic">Basic</option>
            <option value="pro">Pro</option>
          </select>
        </div>
        <div>
          <label class="block text-xs text-gray-400 mb-1">Ciclo</label>
          <select bind:value={modalCycle} class="input w-full">
            <option value="monthly">Mensal</option>
            <option value="yearly">Anual</option>
          </select>
        </div>
      </div>

      {#if modalError}
        <p class="text-xs text-red-400">{modalError}</p>
      {/if}

      <div class="flex gap-3 pt-1">
        <button onclick={closeModal} class="btn-secondary flex-1">Cancelar</button>
        <button onclick={saveModal} disabled={modalSaving} class="btn-primary flex-1 disabled:opacity-60">
          {modalSaving ? 'Salvando…' : 'Confirmar'}
        </button>
      </div>
    </div>
  </div>
{/if}
