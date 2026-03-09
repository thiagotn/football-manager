<script lang="ts">
  import { reviews as reviewsApi } from '$lib/api';
  import type { ReviewSummaryResponse, ReviewListResponse } from '$lib/api';
  import StarRating from '$lib/components/StarRating.svelte';
  import PageBackground from '$lib/components/PageBackground.svelte';

  let summary = $state<ReviewSummaryResponse | null>(null);
  let list = $state<ReviewListResponse | null>(null);
  let loading = $state(true);

  let ratingFilter = $state('');
  let orderBy = $state('created_at');
  let page = $state(1);
  const PAGE_SIZE = 20;

  async function fetchData() {
    loading = true;
    try {
      const [s, l] = await Promise.all([
        reviewsApi.summary(),
        reviewsApi.list({
          rating: ratingFilter || undefined,
          order_by: orderBy,
          page,
          page_size: PAGE_SIZE,
        }),
      ]);
      summary = s;
      list = l;
    } finally {
      loading = false;
    }
  }

  $effect(() => {
    fetchData();
  });

  function applyFilter() {
    page = 1;
    fetchData();
  }

  function formatDate(iso: string) {
    return new Date(iso).toLocaleDateString('pt-BR');
  }
</script>

<svelte:head>
  <title>Avaliações do App — rachao.app</title>
</svelte:head>

<PageBackground>
<main class="relative z-10 max-w-2xl mx-auto px-4 py-8">
  <div class="mb-6">
    <h1 class="text-2xl font-bold text-white">Avaliações do App</h1>
    <p class="text-sm text-gray-300 mt-0.5">Feedback dos usuários da plataforma</p>
  </div>

  {#if loading && !summary}
    <div class="card card-body flex items-center justify-center py-12">
      <span class="text-gray-400 text-sm">Carregando…</span>
    </div>
  {:else if summary}
    <!-- Resumo -->
    <div class="card card-body mb-6">
      {#if summary.total === 0}
        <p class="text-sm text-gray-500 dark:text-gray-400 text-center py-4">Nenhuma avaliação recebida ainda.</p>
      {:else}
        <div class="flex items-center gap-4 mb-5">
          <div class="text-center">
            <p class="text-4xl font-extrabold text-gray-900 dark:text-gray-100">{summary.average.toFixed(1)}</p>
            <div class="flex justify-center mt-1">
              <StarRating rating={Math.round(summary.average)} readonly size={18} />
            </div>
          </div>
          <div class="text-sm text-gray-500 dark:text-gray-400">
            {summary.total} avaliação{summary.total !== 1 ? 'ões' : ''}
          </div>
        </div>

        <div class="space-y-2">
          {#each [5, 4, 3, 2, 1] as star}
            {@const entry = summary.distribution[String(star)]}
            <div class="flex items-center gap-2 text-sm">
              <span class="w-5 text-right text-gray-500 dark:text-gray-400 shrink-0">{star}★</span>
              <div class="flex-1 h-2 bg-gray-100 dark:bg-gray-700 rounded-full overflow-hidden">
                <div
                  class="h-full rounded-full bg-amber-400 transition-all"
                  style="width: {entry?.percent ?? 0}%"
                ></div>
              </div>
              <span class="w-8 text-right text-gray-500 dark:text-gray-400 text-xs shrink-0">{entry?.percent ?? 0}%</span>
              <span class="w-6 text-right text-gray-400 dark:text-gray-500 text-xs shrink-0">({entry?.count ?? 0})</span>
            </div>
          {/each}
        </div>
      {/if}
    </div>

    <!-- Filtros -->
    <div class="card card-body mb-4">
      <div class="flex flex-wrap gap-3 items-end">
        <div class="form-group mb-0 flex-1 min-w-[140px]">
          <label class="label" for="filter-rating">Filtrar por nota</label>
          <select id="filter-rating" class="input" bind:value={ratingFilter}>
            <option value="">Todas</option>
            <option value="5">5 estrelas</option>
            <option value="4">4 estrelas</option>
            <option value="3">3 estrelas</option>
            <option value="2">2 estrelas</option>
            <option value="1">1 estrela</option>
            <option value="1,2">1 e 2 estrelas</option>
          </select>
        </div>
        <div class="form-group mb-0 flex-1 min-w-[140px]">
          <label class="label" for="filter-order">Ordenar por</label>
          <select id="filter-order" class="input" bind:value={orderBy}>
            <option value="created_at">Mais recentes</option>
            <option value="rating">Nota (maior)</option>
          </select>
        </div>
        <button class="btn-primary btn-sm shrink-0" onclick={applyFilter} disabled={loading}>
          Filtrar
        </button>
      </div>
    </div>

    <!-- Lista -->
    {#if list}
      {#if list.items.length === 0}
        <div class="card card-body text-center py-8 text-sm text-gray-500 dark:text-gray-400">
          Nenhuma avaliação encontrada com os filtros selecionados.
        </div>
      {:else}
        <div class="space-y-3">
          {#each list.items as item}
            <div class="card card-body">
              <div class="flex items-start justify-between gap-3">
                <div class="flex-1 min-w-0">
                  <p class="font-medium text-gray-900 dark:text-gray-100 text-sm truncate">{item.player_name}</p>
                  {#if item.comment}
                    <p class="text-sm text-gray-600 dark:text-gray-400 mt-1 leading-relaxed">"{item.comment}"</p>
                  {:else}
                    <p class="text-xs text-gray-400 dark:text-gray-500 mt-1 italic">sem comentário</p>
                  {/if}
                </div>
                <div class="text-right shrink-0">
                  <StarRating rating={item.rating} readonly size={14} />
                  <p class="text-xs text-gray-400 dark:text-gray-500 mt-1">{formatDate(item.created_at)}</p>
                </div>
              </div>
            </div>
          {/each}
        </div>

        <!-- Paginação -->
        {#if list.total_pages > 1}
          <div class="flex items-center justify-center gap-3 mt-6">
            <button
              class="btn-secondary btn-sm"
              disabled={page === 1 || loading}
              onclick={() => { page -= 1; fetchData(); }}
            >← Anterior</button>
            <span class="text-sm text-gray-300">Página {list.page} de {list.total_pages}</span>
            <button
              class="btn-secondary btn-sm"
              disabled={page >= list.total_pages || loading}
              onclick={() => { page += 1; fetchData(); }}
            >Próxima →</button>
          </div>
        {/if}
      {/if}
    {/if}
  {/if}
</main>
</PageBackground>
