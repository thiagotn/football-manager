<script lang="ts">
  import { androidBeta } from '$lib/api';
  import type { AndroidBetaSignupListResponse } from '$lib/api';
  import { Smartphone, ChevronLeft, ChevronRight } from 'lucide-svelte';
  import PageBackground from '$lib/components/PageBackground.svelte';

  let list = $state<AndroidBetaSignupListResponse | null>(null);
  let loading = $state(true);
  let page = $state(1);
  const PAGE_SIZE = 20;

  async function fetchData() {
    loading = true;
    try {
      list = await androidBeta.list({ page, page_size: PAGE_SIZE });
    } finally {
      loading = false;
    }
  }

  $effect(() => {
    fetchData();
  });

  function formatDate(iso: string) {
    return new Date(iso).toLocaleString('pt-BR', {
      day: '2-digit', month: '2-digit', year: 'numeric',
      hour: '2-digit', minute: '2-digit',
    });
  }

  let withPlayer = $derived(list ? list.items.filter(i => i.player_id).length : 0);
  let anonymous  = $derived(list ? list.items.filter(i => !i.player_id).length : 0);
</script>

<svelte:head>
  <title>Beta Android — rachao.app</title>
</svelte:head>

<PageBackground>
  <main class="relative z-10 max-w-7xl mx-auto px-4 py-8">
    <div class="flex items-center justify-between mb-6">
      <div>
        <h1 class="text-2xl font-bold text-white flex items-center gap-2">
          <Smartphone size={24} class="text-primary-400" /> Beta Android
        </h1>
        <p class="text-sm text-white/60 mt-0.5">Emails inscritos na faixa de testes do Google Play</p>
      </div>
    </div>

    <!-- Summary -->
    {#if list}
      <div class="grid grid-cols-3 gap-3 mb-6">
        <div class="card p-4 text-center">
          <p class="text-3xl font-bold text-primary-400">{list.total}</p>
          <p class="text-xs text-gray-400 mt-1">Total inscritos</p>
        </div>
        <div class="card p-4 text-center">
          <p class="text-3xl font-bold text-green-400">{withPlayer}</p>
          <p class="text-xs text-gray-400 mt-1">Com conta linkada</p>
        </div>
        <div class="card p-4 text-center">
          <p class="text-3xl font-bold text-gray-400">{anonymous}</p>
          <p class="text-xs text-gray-400 mt-1">Anônimos</p>
        </div>
      </div>
    {/if}

    <!-- List -->
    {#if loading}
      <div class="text-center py-16 text-white/40">Carregando...</div>
    {:else if list && list.items.length === 0}
      <div class="text-center py-16 text-white/40">Nenhuma inscrição ainda.</div>
    {:else if list}
      <div class="space-y-2">
        {#each list.items as item}
          <div class="card card-body">
            <div class="flex items-center justify-between gap-3">
              <div class="flex-1 min-w-0">
                <p class="font-medium text-white text-sm truncate">{item.google_email}</p>
                {#if item.player_name}
                  <p class="text-xs text-primary-400 mt-0.5">{item.player_name}</p>
                {:else}
                  <p class="text-xs text-white/30 mt-0.5">— sem conta</p>
                {/if}
              </div>
              <p class="text-xs text-white/40 shrink-0">{formatDate(item.created_at)}</p>
            </div>
          </div>
        {/each}
      </div>

      <!-- Pagination -->
      {#if list.total > PAGE_SIZE}
        <div class="flex items-center justify-between mt-6">
          <button
            onclick={() => { page--; fetchData(); }}
            disabled={page === 1}
            class="btn btn-ghost btn-sm flex items-center gap-1 disabled:opacity-40"
          >
            <ChevronLeft size={14} /> Anterior
          </button>
          <span class="text-sm text-white/50">
            Página {page} de {Math.ceil(list.total / PAGE_SIZE)}
          </span>
          <button
            onclick={() => { page++; fetchData(); }}
            disabled={page >= Math.ceil(list.total / PAGE_SIZE)}
            class="btn btn-ghost btn-sm flex items-center gap-1 disabled:opacity-40"
          >
            Próxima <ChevronRight size={14} />
          </button>
        </div>
      {/if}
    {/if}

  </main>
</PageBackground>
