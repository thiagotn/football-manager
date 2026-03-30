<script lang="ts">
  import { goto } from '$app/navigation';
  import { authStore, isAdmin } from '$lib/stores/auth';
  import { admin as adminApi } from '$lib/api';
  import type { AdminMatchItem } from '$lib/api';
  import { ChevronLeft, ChevronRight } from 'lucide-svelte';
  import PageBackground from '$lib/components/PageBackground.svelte';

  const PAGE_SIZE = 50;

  type StatusFilter = '' | 'open' | 'in_progress' | 'closed';

  let statusFilter = $state<StatusFilter>('');
  let items = $state<AdminMatchItem[]>([]);
  let total = $state(0);
  let loading = $state(true);
  let loadingMore = $state(false);
  let error = $state('');

  const STATUS_LABELS: Record<string, string> = {
    open: 'Aberta',
    in_progress: 'Bola rolando',
    closed: 'Encerrada',
  };

  const FILTERS: { value: StatusFilter; label: string }[] = [
    { value: '',            label: 'Todos' },
    { value: 'open',        label: 'Abertos' },
    { value: 'in_progress', label: 'Em andamento' },
    { value: 'closed',      label: 'Encerrados' },
  ];

  function fmtDate(d: string): string {
    const [y, m, day] = d.split('-');
    return `${day}/${m}/${y}`;
  }

  async function load(reset = false) {
    if (reset) { items = []; loading = true; error = ''; }
    else loadingMore = true;
    try {
      const res = await adminApi.getMatches({
        status: statusFilter || undefined,
        limit: PAGE_SIZE,
        offset: reset ? 0 : items.length,
      });
      total = res.total;
      items = reset ? res.items : [...items, ...res.items];
    } catch {
      error = 'Não foi possível carregar os rachões.';
    } finally {
      loading = false;
      loadingMore = false;
    }
  }

  let authReady = $state(false);

  // Auth guard
  $effect(() => {
    if ($authStore.loading) return;
    if (!$isAdmin) { goto('/dashboard', { replaceState: true }); return; }
    authReady = true;
  });

  // Carrega dados — re-dispara quando authReady ou statusFilter mudam
  $effect(() => {
    if (!authReady) return;
    void statusFilter; // rastreia dependência
    load(true);
  });
</script>

<svelte:head><title>Rachões — Admin — rachao.app</title></svelte:head>

<PageBackground>
<main class="relative z-10 max-w-7xl mx-auto px-4 py-8 space-y-5">

  <!-- Header -->
  <div class="flex items-center gap-3">
    <a href="/admin" class="text-gray-300 hover:text-white transition-colors">
      <ChevronLeft size={20} />
    </a>
    <div>
      <h1 class="text-xl font-bold text-white">
        Rachões {#if !loading}({total}){/if}
      </h1>
      <p class="text-xs text-gray-400">Todos os rachões da plataforma</p>
    </div>
  </div>

  <!-- Filtros de status -->
  <div class="flex gap-2 overflow-x-auto [scrollbar-width:none] [&::-webkit-scrollbar]:hidden pb-1">
    {#each FILTERS as f}
      <button
        class="px-3 py-1.5 rounded-full text-xs font-medium whitespace-nowrap transition-colors
          {statusFilter === f.value
            ? 'bg-primary-500 text-white'
            : 'bg-white/10 text-gray-300 hover:bg-white/20'}"
        onclick={() => { statusFilter = f.value; }}>
        {f.label}
      </button>
    {/each}
  </div>

  {#if loading}
    <div class="animate-pulse space-y-2">
      {#each [1,2,3,4,5,6] as _}
        <div class="h-12 bg-white/10 rounded-lg"></div>
      {/each}
    </div>
  {:else if error}
    <div class="alert-error">{error}</div>
  {:else if items.length === 0}
    <div class="card p-12 text-center">
      <p class="text-gray-400 text-sm">Nenhum rachão encontrado.</p>
    </div>
  {:else}
    <div class="card overflow-x-hidden">
      <table class="w-full text-sm">
        <thead>
          <tr class="border-b border-gray-100 dark:border-gray-700 text-xs text-gray-500 dark:text-gray-400">
            <th class="px-4 py-2 text-left">Grupo / Rachão</th>
            <th class="px-4 py-2 text-left hidden sm:table-cell">Data</th>
            <th class="px-4 py-2 text-left hidden sm:table-cell">Horário</th>
            <th class="px-4 py-2 text-left">Status</th>
            <th class="px-4 py-2 w-6"></th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-100 dark:divide-gray-700">
          {#each items as m}
            <tr
              class="hover:bg-gray-50 dark:hover:bg-gray-700/40 transition-colors cursor-pointer"
              onclick={() => goto(`/match/${m.hash}`)}>
              <td class="px-4 py-2.5">
                <p class="font-medium text-gray-900 dark:text-gray-100 truncate max-w-[160px]">{m.group_name}</p>
                <p class="text-xs text-gray-500 dark:text-gray-400 truncate max-w-[200px]">#{m.number} · {m.location.split('—')[0].trim()}</p>
                <p class="text-xs text-gray-500 dark:text-gray-400 sm:hidden">{fmtDate(m.match_date)}</p>
              </td>
              <td class="px-4 py-2.5 text-gray-700 dark:text-gray-300 hidden sm:table-cell whitespace-nowrap">
                {fmtDate(m.match_date)}
              </td>
              <td class="px-4 py-2.5 text-gray-500 dark:text-gray-400 hidden sm:table-cell whitespace-nowrap">
                {m.start_time.slice(0, 5)}
                {#if m.end_time} – {m.end_time.slice(0, 5)}{/if}
              </td>
              <td class="px-4 py-2.5">
                {#if m.status === 'in_progress'}
                  <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-semibold bg-red-500/20 text-red-400 border border-red-500/30 whitespace-nowrap">
                    <span class="w-1.5 h-1.5 rounded-full bg-red-400 animate-pulse"></span>
                    Bola rolando
                  </span>
                {:else}
                  <span class="badge {m.status === 'open' ? 'badge-green' : 'badge-gray'} whitespace-nowrap">
                    {STATUS_LABELS[m.status] ?? m.status}
                  </span>
                {/if}
              </td>
              <td class="pl-2 pr-4 py-2.5 text-gray-400">
                <ChevronRight size={14} />
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>

    {#if items.length < total}
      <div class="text-center">
        <button
          class="btn-secondary btn-sm"
          onclick={() => load(false)}
          disabled={loadingMore}>
          {loadingMore ? 'Carregando…' : `Carregar mais (${total - items.length} restantes)`}
        </button>
      </div>
    {/if}
  {/if}

</main>
</PageBackground>
