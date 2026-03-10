<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { authStore, currentPlayer, isAdmin } from '$lib/stores/auth';
  import { groups, players as playersApi } from '$lib/api';
  import type { SignupStats } from '$lib/api';
  import { formatWhatsapp } from '$lib/utils.js';
  import { Users, Plus, UserPlus, Clock } from 'lucide-svelte';
  import PageBackground from '$lib/components/PageBackground.svelte';

  let myGroups: any[] = $state([]);
  let signupStats: SignupStats | null = $state(null);
  let loading = $state(true);
  let statsLoading = $state(false);
  let statsError = $state('');

  onMount(async () => {
    try {
      myGroups = await groups.list();
    } finally {
      loading = false;
    }
  });

  // Redireciona super admins para o painel dedicado
  $effect(() => {
    if (!$authStore.loading && $isAdmin) {
      goto('/admin', { replaceState: true });
    }
  });

  // Aguarda o auth terminar de carregar ($authStore.loading) antes de verificar
  // se é admin — evita race condition com authStore.init() no layout
  let statsLoaded = false;
  $effect(() => {
    if ($authStore.loading) return;
    if ($isAdmin && !statsLoaded) {
      statsLoaded = true;
      statsLoading = true;
      statsError = '';
      playersApi.signupStats(30)
        .then(data => { signupStats = data; })
        .catch(e => {
          console.error('[dashboard] signupStats error:', e);
          statsError = 'Não foi possível carregar os dados.';
        })
        .finally(() => { statsLoading = false; });
    }
  });

  function formatDateTime(iso: string) {
    const d = new Date(iso);
    return d.toLocaleDateString('pt-BR', { day: '2-digit', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit' });
  }

  function daysAgo(iso: string): string {
    const diff = Math.floor((Date.now() - new Date(iso).getTime()) / 86400000);
    if (diff === 0) return 'hoje';
    if (diff === 1) return 'ontem';
    return `${diff}d atrás`;
  }
</script>

<svelte:head><title>Dashboard — rachao.app</title></svelte:head>

<PageBackground>
<main class="relative z-10 max-w-7xl mx-auto px-4 py-8 space-y-8">

  <!-- Header -->
  <div>
    <h1 class="text-2xl font-bold text-white">
      Olá, {$currentPlayer?.name?.split(' ')[0]} 👋
    </h1>
    <p class="text-gray-300 mt-1 text-sm">Bem-vindo ao rachao.app</p>
  </div>

  {#if loading}
    <div class="grid sm:grid-cols-2 lg:grid-cols-3 gap-4">
      {#each [1,2,3] as _}
        <div class="card p-5 animate-pulse">
          <div class="h-4 bg-gray-200 dark:bg-gray-700 rounded w-1/2 mb-3"></div>
          <div class="h-8 bg-gray-100 dark:bg-gray-700 rounded w-1/3"></div>
        </div>
      {/each}
    </div>
  {:else}

    <!-- Groups summary -->
    <div>
      <div class="flex items-center justify-between mb-3">
        <h2 class="text-base font-semibold text-white flex items-center gap-2">
          <Users size={16} class="text-primary-400" /> Meus Grupos
        </h2>
        <a href="/groups" class="text-xs text-primary-300 hover:text-primary-200">Ver todos →</a>
      </div>

      {#if myGroups.length === 0}
        <div class="card p-8 text-center">
          <Users size={32} class="mx-auto mb-2 text-gray-400 opacity-40" />
          <p class="text-sm text-gray-500 dark:text-gray-400 mb-3">Nenhum grupo ainda</p>
          <a href="/groups" class="btn-primary text-sm"><Plus size={14} /> Criar grupo</a>
        </div>
      {:else}
        <div class="grid sm:grid-cols-2 lg:grid-cols-3 gap-3">
          {#each myGroups.slice(0, 6) as g}
            <a href="/groups/{g.id}" class="card card-body hover:shadow-md transition-shadow block">
              <p class="font-semibold text-gray-900 dark:text-gray-100 truncate">{g.name}</p>
              {#if g.description}
                <p class="text-xs text-gray-500 dark:text-gray-400 mt-0.5 line-clamp-1">{g.description}</p>
              {/if}
              <span class="inline-block mt-2 text-xs font-mono bg-gray-100 dark:bg-gray-700 text-gray-500 dark:text-gray-400 px-2 py-0.5 rounded">{g.slug}</span>
            </a>
          {/each}
        </div>
      {/if}
    </div>

    <!-- ── Admin-only: Signup stats ──────────────────────────── -->
    {#if $isAdmin}
      <div>
        <div class="flex items-center gap-2 mb-3">
          <h2 class="text-base font-semibold text-white flex items-center gap-2">
            <UserPlus size={16} class="text-primary-400" /> Novos Cadastros
          </h2>
        </div>

        {#if statsLoading}
          <!-- Skeleton counters -->
          <div class="grid grid-cols-3 gap-3 mb-4">
            {#each [1,2,3] as _}
              <div class="card p-4 animate-pulse text-center">
                <div class="h-7 bg-gray-200 dark:bg-gray-700 rounded w-1/2 mx-auto mb-2"></div>
                <div class="h-3 bg-gray-100 dark:bg-gray-700 rounded w-2/3 mx-auto"></div>
              </div>
            {/each}
          </div>
        {:else if statsError}
          <div class="alert-error mb-4">{statsError}</div>
        {:else if signupStats}
          <!-- Counters -->
          <div class="grid grid-cols-3 gap-3 mb-4">
            <div class="card p-4 text-center">
              <p class="text-2xl font-bold text-primary-600 dark:text-primary-400">{signupStats.total}</p>
              <p class="text-xs text-gray-500 dark:text-gray-400 mt-0.5">Total</p>
            </div>
            <div class="card p-4 text-center">
              <p class="text-2xl font-bold text-green-600 dark:text-green-400">{signupStats.last_7_days}</p>
              <p class="text-xs text-gray-500 dark:text-gray-400 mt-0.5">Últimos 7 dias</p>
            </div>
            <div class="card p-4 text-center">
              <p class="text-2xl font-bold text-blue-600 dark:text-blue-400">{signupStats.last_30_days}</p>
              <p class="text-xs text-gray-500 dark:text-gray-400 mt-0.5">Últimos 30 dias</p>
            </div>
          </div>

          <!-- Recent list -->
          <div class="card overflow-hidden">
            <div class="px-4 py-3 border-b border-gray-100 dark:border-gray-700 flex items-center justify-between">
              <span class="text-sm font-medium text-gray-700 dark:text-gray-300">Registros recentes</span>
              <a href="/players" class="text-xs text-primary-600 dark:text-primary-400 hover:underline">Ver todos →</a>
            </div>
            {#if signupStats.recent.length === 0}
              <div class="px-4 py-8 text-center text-sm text-gray-400">Nenhum cadastro ainda.</div>
            {:else}
              <div class="divide-y divide-gray-100 dark:divide-gray-700">
                {#each signupStats.recent as p}
                  <div class="flex items-center justify-between px-4 py-3 gap-3">
                    <div class="min-w-0">
                      <p class="font-medium text-sm text-gray-900 dark:text-gray-100 truncate">
                        {p.nickname ? `${p.nickname} (${p.name})` : p.name}
                      </p>
                      <p class="text-xs text-gray-500 dark:text-gray-400 font-mono">{formatWhatsapp(p.whatsapp)}</p>
                    </div>
                    <div class="text-right shrink-0">
                      <span class="text-xs {p.active ? 'text-green-600 dark:text-green-400' : 'text-red-500'} font-medium">
                        {p.active ? 'Ativo' : 'Inativo'}
                      </span>
                      <p class="text-xs text-gray-400 mt-0.5 flex items-center gap-1 justify-end">
                        <Clock size={10} />
                        {daysAgo(p.created_at)}
                      </p>
                    </div>
                  </div>
                {/each}
              </div>
            {/if}
          </div>
        {/if}
      </div>
    {/if}

  {/if}
</main>
</PageBackground>
