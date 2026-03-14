<script lang="ts">
  import { goto } from '$app/navigation';
  import { authStore, isAdmin } from '$lib/stores/auth';
  import { admin as adminApi } from '$lib/api';
  import type { AdminStatsResponse, AdminSubscriptionSummary } from '$lib/api';
  import { Users, Calendar, Clock, UserPlus, Star, ChevronRight, CreditCard, AlertTriangle, TrendingUp } from 'lucide-svelte';
  import PageBackground from '$lib/components/PageBackground.svelte';

  let stats = $state<AdminStatsResponse | null>(null);
  let billing = $state<AdminSubscriptionSummary | null>(null);
  let loading = $state(true);
  let error = $state('');

  function fmtHours(minutes: number): string {
    if (minutes === 0) return '—';
    const h = Math.floor(minutes / 60);
    const m = minutes % 60;
    if (m === 0) return `${h}h`;
    return `${h}h ${m}min`;
  }

  let loaded = false;
  $effect(() => {
    if ($authStore.loading) return;
    if (!$isAdmin) { goto('/dashboard', { replaceState: true }); return; }
    if (loaded) return;
    loaded = true;
    Promise.all([
      adminApi.getStats(),
      adminApi.getSubscriptionSummary(),
    ])
      .then(([s, b]) => { stats = s; billing = b; })
      .catch(() => { error = 'Não foi possível carregar os dados.'; })
      .finally(() => { loading = false; });
  });
</script>

<svelte:head><title>Painel Admin — rachao.app</title></svelte:head>

<PageBackground>
<main class="relative z-10 max-w-7xl mx-auto px-4 py-8 space-y-8">

  <!-- Header -->
  <div>
    <h1 class="text-2xl font-bold text-white">Painel Admin</h1>
    <p class="text-gray-300 mt-1 text-sm">Visão geral da plataforma</p>
  </div>

  {#if loading}
    <div class="grid grid-cols-2 sm:grid-cols-5 gap-4">
      {#each [1,2,3,4,5] as _}
        <div class="card p-5 animate-pulse text-center">
          <div class="h-8 bg-gray-200 dark:bg-gray-700 rounded w-1/2 mx-auto mb-2"></div>
          <div class="h-3 bg-gray-100 dark:bg-gray-700 rounded w-2/3 mx-auto"></div>
        </div>
      {/each}
    </div>
  {:else if error}
    <div class="alert-error">{error}</div>
  {:else if stats}

    <!-- Big numbers — plataforma -->
    <div>
      <h2 class="text-sm font-semibold text-gray-300 uppercase tracking-wide mb-3">Plataforma</h2>
      <div class="grid grid-cols-2 sm:grid-cols-5 gap-4">

        <a href="/admin/matches" class="card p-5 text-center group hover:shadow-md transition-shadow cursor-pointer">
          <div class="flex items-center justify-center gap-1 mb-1">
            <Calendar size={14} class="text-primary-400 opacity-70" />
          </div>
          <p class="text-3xl font-bold text-primary-400">{stats.total_matches}</p>
          <p class="text-xs text-gray-400 mt-1 flex items-center justify-center gap-0.5">
            Rachões <ChevronRight size={11} class="group-hover:translate-x-0.5 transition-transform" />
          </p>
        </a>

        <a href="/admin/groups" class="card p-5 text-center group hover:shadow-md transition-shadow cursor-pointer">
          <div class="flex items-center justify-center gap-1 mb-1">
            <Users size={14} class="text-blue-400 opacity-70" />
          </div>
          <p class="text-3xl font-bold text-blue-400">{stats.total_groups}</p>
          <p class="text-xs text-gray-400 mt-1 flex items-center justify-center gap-0.5">
            Grupos <ChevronRight size={11} class="group-hover:translate-x-0.5 transition-transform" />
          </p>
        </a>

        <a href="/players" class="card p-5 text-center group hover:shadow-md transition-shadow cursor-pointer">
          <div class="flex items-center justify-center gap-1 mb-1">
            <Users size={14} class="text-amber-400 opacity-70" />
          </div>
          <p class="text-3xl font-bold text-amber-400">{stats.total_players}</p>
          <p class="text-xs text-gray-400 mt-1 flex items-center justify-center gap-0.5">
            Jogadores <ChevronRight size={11} class="group-hover:translate-x-0.5 transition-transform" />
          </p>
        </a>

        <div class="card p-5 text-center">
          <div class="flex items-center justify-center gap-1 mb-1">
            <Clock size={14} class="text-purple-400 opacity-70" />
          </div>
          <p class="text-3xl font-bold text-purple-400">{fmtHours(stats.platform_minutes_played)}</p>
          <p class="text-xs text-gray-400 mt-1">Horas jogadas</p>
        </div>

        <a href="/admin/reviews" class="card p-5 text-center group hover:shadow-md transition-shadow cursor-pointer col-span-2 sm:col-span-1">
          <div class="flex items-center justify-center gap-1 mb-1">
            <Star size={14} class="text-yellow-400 opacity-70" />
          </div>
          <p class="text-3xl font-bold text-yellow-400">{stats.total_reviews}</p>
          <p class="text-xs text-gray-400 mt-1 flex items-center justify-center gap-0.5">
            Avaliações <ChevronRight size={11} class="group-hover:translate-x-0.5 transition-transform" />
          </p>
        </a>

      </div>
    </div>

    <!-- Novos Cadastros -->
    <div>
      <h2 class="text-sm font-semibold text-gray-300 uppercase tracking-wide mb-3 flex items-center gap-2">
        <UserPlus size={14} class="text-primary-400" /> Novos Cadastros
      </h2>
      <div class="grid grid-cols-3 gap-3">
        <div class="card p-4 text-center">
          <p class="text-2xl font-bold text-primary-600 dark:text-primary-400">{stats.signups_total}</p>
          <p class="text-xs text-gray-400 mt-0.5">Total</p>
        </div>
        <div class="card p-4 text-center">
          <p class="text-2xl font-bold text-green-600 dark:text-green-400">{stats.signups_last_7_days}</p>
          <p class="text-xs text-gray-400 mt-0.5">Últimos 7 dias</p>
        </div>
        <div class="card p-4 text-center">
          <p class="text-2xl font-bold text-blue-600 dark:text-blue-400">{stats.signups_last_30_days}</p>
          <p class="text-xs text-gray-400 mt-0.5">Últimos 30 dias</p>
        </div>
      </div>
    </div>

    <!-- Billing -->
    {#if billing}
      {#if billing.past_due > 0}
        <a href="/admin/subscriptions?status=past_due"
          class="flex items-center gap-3 px-4 py-3 rounded-xl bg-yellow-500/10 border border-yellow-500/30 text-yellow-300 text-sm hover:bg-yellow-500/20 transition-colors">
          <AlertTriangle size={16} class="shrink-0" />
          <span><strong>{billing.past_due}</strong> assinatura{billing.past_due > 1 ? 's' : ''} com pagamento pendente — clique para revisar</span>
          <ChevronRight size={14} class="ml-auto shrink-0" />
        </a>
      {/if}

      <div>
        <h2 class="text-sm font-semibold text-gray-300 uppercase tracking-wide mb-3 flex items-center gap-2">
          <CreditCard size={14} class="text-emerald-400" /> Billing
        </h2>
        <div class="grid grid-cols-2 sm:grid-cols-4 gap-4">
          <a href="/admin/subscriptions" class="card p-5 text-center group hover:shadow-md transition-shadow">
            <div class="flex items-center justify-center mb-1">
              <TrendingUp size={14} class="text-emerald-400 opacity-70" />
            </div>
            <p class="text-3xl font-bold text-emerald-400">{billing.active}</p>
            <p class="text-xs text-gray-400 mt-1 flex items-center justify-center gap-0.5">
              Assinantes <ChevronRight size={11} class="group-hover:translate-x-0.5 transition-transform" />
            </p>
          </a>

          <a href="/admin/subscriptions?status=past_due" class="card p-5 text-center group hover:shadow-md transition-shadow">
            <div class="flex items-center justify-center mb-1">
              <AlertTriangle size={14} class="{billing.past_due > 0 ? 'text-yellow-400' : 'text-gray-500'} opacity-70" />
            </div>
            <p class="text-3xl font-bold {billing.past_due > 0 ? 'text-yellow-400' : 'text-gray-500'}">{billing.past_due}</p>
            <p class="text-xs text-gray-400 mt-1 flex items-center justify-center gap-0.5">
              Pag. Pendente <ChevronRight size={11} class="group-hover:translate-x-0.5 transition-transform" />
            </p>
          </a>

          <div class="card p-5 text-center col-span-2 sm:col-span-1">
            <div class="flex items-center justify-center mb-1">
              <CreditCard size={14} class="text-primary-400 opacity-70" />
            </div>
            <p class="text-3xl font-bold text-primary-400">
              {(billing.mrr_cents / 100).toLocaleString('pt-BR', { style: 'currency', currency: 'BRL', minimumFractionDigits: 2 })}
            </p>
            <p class="text-xs text-gray-400 mt-1">MRR estimado</p>
          </div>

          <a href="/admin/subscriptions" class="card p-5 text-center group hover:shadow-md transition-shadow hidden sm:block">
            <div class="flex items-center justify-center mb-1">
              <Users size={14} class="text-gray-400 opacity-70" />
            </div>
            <p class="text-3xl font-bold text-gray-400">{billing.free}</p>
            <p class="text-xs text-gray-400 mt-1 flex items-center justify-center gap-0.5">
              Free <ChevronRight size={11} class="group-hover:translate-x-0.5 transition-transform" />
            </p>
          </a>
        </div>
      </div>
    {/if}

  {/if}
</main>
</PageBackground>
