<script lang="ts">
  import { goto } from '$app/navigation';
  import { authStore, isAdmin } from '$lib/stores/auth';
  import { admin as adminApi } from '$lib/api';
  import type { AdminGroupItem } from '$lib/api';
  import { ChevronLeft, ChevronRight } from 'lucide-svelte';
  import PageBackground from '$lib/components/PageBackground.svelte';

  const PAGE_SIZE = 50;

  let items = $state<AdminGroupItem[]>([]);
  let total = $state(0);
  let loading = $state(true);
  let loadingMore = $state(false);
  let error = $state('');

  function fmtDate(iso: string): string {
    return new Date(iso).toLocaleDateString('pt-BR', { month: 'short', year: 'numeric' });
  }

  async function load(reset = false) {
    if (reset) { items = []; loading = true; error = ''; }
    else loadingMore = true;
    try {
      const res = await adminApi.getGroups({
        limit: PAGE_SIZE,
        offset: reset ? 0 : items.length,
      });
      total = res.total;
      items = reset ? res.items : [...items, ...res.items];
    } catch {
      error = 'Não foi possível carregar os grupos.';
    } finally {
      loading = false;
      loadingMore = false;
    }
  }

  let authReady = false;
  $effect(() => {
    if ($authStore.loading) return;
    if (!$isAdmin) { goto('/dashboard', { replaceState: true }); return; }
    if (!authReady) { authReady = true; load(true); }
  });
</script>

<svelte:head><title>Grupos — Admin — rachao.app</title></svelte:head>

<PageBackground>
<main class="relative z-10 max-w-7xl mx-auto px-4 py-8 space-y-5">

  <!-- Header -->
  <div class="flex items-center gap-3">
    <a href="/admin" class="text-gray-300 hover:text-white transition-colors">
      <ChevronLeft size={20} />
    </a>
    <div>
      <h1 class="text-xl font-bold text-white">
        Grupos {#if !loading}({total}){/if}
      </h1>
      <p class="text-xs text-gray-400">Todos os grupos da plataforma</p>
    </div>
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
      <p class="text-gray-400 text-sm">Nenhum grupo encontrado.</p>
    </div>
  {:else}
    <div class="card overflow-hidden">
      <table class="w-full text-sm">
        <thead>
          <tr class="border-b border-gray-100 dark:border-gray-700 text-xs text-gray-500 dark:text-gray-400">
            <th class="px-4 py-2 text-left">Grupo</th>
            <th class="px-4 py-2 text-right">Jogadores</th>
            <th class="px-4 py-2 text-right hidden sm:table-cell">Rachões</th>
            <th class="px-4 py-2 text-right hidden sm:table-cell">Criado</th>
            <th class="px-4 py-2 w-6"></th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-100 dark:divide-gray-700">
          {#each items as g}
            <tr
              class="hover:bg-gray-50 dark:hover:bg-gray-700/40 transition-colors cursor-pointer"
              onclick={() => goto(`/groups/${g.id}`)}>
              <td class="px-4 py-2.5">
                <p class="font-medium text-gray-900 dark:text-gray-100 truncate max-w-[180px]">{g.name}</p>
                {#if g.description}
                  <p class="text-xs text-gray-500 dark:text-gray-400 truncate max-w-[180px]">{g.description}</p>
                {/if}
              </td>
              <td class="px-4 py-2.5 text-right text-gray-700 dark:text-gray-300">{g.total_members}</td>
              <td class="px-4 py-2.5 text-right text-gray-500 dark:text-gray-400 hidden sm:table-cell">{g.total_matches}</td>
              <td class="px-4 py-2.5 text-right text-gray-500 dark:text-gray-400 hidden sm:table-cell whitespace-nowrap">{fmtDate(g.created_at)}</td>
              <td class="px-3 py-2.5 text-gray-400">
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
