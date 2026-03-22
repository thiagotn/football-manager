<script lang="ts">
  import { matches as matchesApi, groups, ApiError } from '$lib/api';
  import type { DiscoverMatch } from '$lib/api';
  import { isLoggedIn, isAdmin } from '$lib/stores/auth';
  import { goto } from '$app/navigation';
  import { Compass, Calendar, Clock, MapPin, Filter, X } from 'lucide-svelte';
  import PageBackground from '$lib/components/PageBackground.svelte';
  import WaitlistModal from '$lib/components/WaitlistModal.svelte';
  import { toastError, toastSuccess } from '$lib/stores/toast';
  import { t, locale } from '$lib/i18n';

  // Filters
  let selectedCourts = $state<string[]>([]);
  let selectedWeekdays = $state<number[]>([]);
  let dateRange = $state<'week' | 'month' | 'all'>('week');
  let showFilters = $state(false);

  // Results
  let items = $state<DiscoverMatch[]>([]);
  let loading = $state(true);
  let offset = $state(0);
  let hasMore = $state(true);
  const PAGE_SIZE = 20;

  // Waitlist
  let waitlistMatch = $state<DiscoverMatch | null>(null);
  let showModal = $state(false);
  let submitting = $state(false);

  $effect(() => {
    if (!$isLoggedIn) { goto('/login?next=/discover'); return; }
    if ($isAdmin) { goto('/'); return; }
  });

  function dateParams(): { date_from?: string; date_to?: string } {
    const today = new Date();
    const fmt = (d: Date) => d.toISOString().slice(0, 10);
    if (dateRange === 'week') {
      const end = new Date(today); end.setDate(end.getDate() + 7);
      return { date_from: fmt(today), date_to: fmt(end) };
    }
    if (dateRange === 'month') {
      const end = new Date(today); end.setDate(end.getDate() + 30);
      return { date_from: fmt(today), date_to: fmt(end) };
    }
    return {};
  }

  async function load(reset = false) {
    loading = true;
    if (reset) { offset = 0; items = []; hasMore = true; }
    try {
      const params = {
        ...dateParams(),
        court_type: selectedCourts.length ? selectedCourts : undefined,
        weekday: selectedWeekdays.length ? selectedWeekdays : undefined,
        limit: PAGE_SIZE,
        offset,
      };
      const result = await matchesApi.discover(params);
      items = reset ? result : [...items, ...result];
      hasMore = result.length === PAGE_SIZE;
      offset += result.length;
    } catch { /* silent */ } finally {
      loading = false;
    }
  }

  $effect(() => {
    if ($isLoggedIn && !$isAdmin) load(true);
  });

  function toggleCourt(c: string) {
    selectedCourts = selectedCourts.includes(c)
      ? selectedCourts.filter(x => x !== c)
      : [...selectedCourts, c];
  }

  function toggleWeekday(d: number) {
    selectedWeekdays = selectedWeekdays.includes(d)
      ? selectedWeekdays.filter(x => x !== d)
      : [...selectedWeekdays, d];
  }

  function applyFilters() {
    showFilters = false;
    load(true);
  }

  function clearFilters() {
    selectedCourts = [];
    selectedWeekdays = [];
    dateRange = 'week';
    showFilters = false;
    load(true);
  }

  async function submitWaitlist(data: { agreed: boolean; intro: string }) {
    if (!waitlistMatch) return;
    submitting = true;
    try {
      await groups.joinWaitlist(waitlistMatch.group_id, data);
      const removedId = waitlistMatch.id;
      showModal = false;
      waitlistMatch = null;
      items = items.filter(m => m.id !== removedId);
      toastSuccess($t('discover.waitlist_success'));
    } catch (e) {
      toastError(e instanceof ApiError ? e.message : 'Erro ao enviar candidatura');
    } finally {
      submitting = false;
    }
  }

  const activeFilterCount = $derived(
    selectedCourts.length + selectedWeekdays.length + (dateRange !== 'week' ? 1 : 0)
  );

  let courtLabels = $derived<Record<string, string>>({
    campo: $t('discover.court_type') === $t('discover.court_type') ? 'Campo' : 'Campo',
    sintetico: 'Sintético',
    terrao: 'Terrão',
    quadra: 'Quadra'
  });

  const WEEKDAY_LABELS = ['Dom', 'Seg', 'Ter', 'Qua', 'Qui', 'Sex', 'Sáb'];

  let periodOptions = $derived<[string, string][]>([
    ['week', $t('discover.this_week')],
    ['month', $t('discover.next_30_days')],
    ['all', $t('discover.any_date')]
  ]);
</script>

<svelte:head><title>Descobrir Rachões — rachao.app</title></svelte:head>

<PageBackground>
  <main class="relative z-10 max-w-7xl mx-auto px-4 py-8">
    <div class="flex items-start justify-between mb-6 gap-3">
      <div class="min-w-0">
        <h1 class="text-2xl font-bold text-white flex items-center gap-2">
          <Compass size={24} class="text-primary-400 shrink-0" /> {$t('discover.title')}
        </h1>
        <p class="text-sm text-white/60 mt-0.5">{$t('discover.subtitle')}</p>
      </div>
      <button
        onclick={() => showFilters = !showFilters}
        class="btn btn-sm btn-ghost text-white border border-white/20 hover:bg-white/10 relative shrink-0 mt-1">
        <Filter size={14} /> {$t('discover.filters')}
        {#if activeFilterCount > 0}
          <span class="absolute -top-1.5 -right-1.5 w-4 h-4 rounded-full bg-primary-500 text-white text-[10px] flex items-center justify-center font-bold">{activeFilterCount}</span>
        {/if}
      </button>
    </div>

    <!-- Filtros -->
    {#if showFilters}
      <div class="card p-4 mb-6 space-y-4">
        <!-- Período -->
        <div>
          <p class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">{$t('discover.period')}</p>
          <div class="flex gap-2 flex-wrap">
            {#each periodOptions as [val, label]}
              <button
                onclick={() => dateRange = val as typeof dateRange}
                class="px-3 py-1.5 rounded-full text-xs font-medium border transition-colors
                  {dateRange === val ? 'bg-primary-600 text-white border-primary-600' : 'border-gray-300 dark:border-gray-600 text-gray-600 dark:text-gray-400 hover:border-primary-400'}">
                {label}
              </button>
            {/each}
          </div>
        </div>

        <!-- Tipo de quadra -->
        <div>
          <p class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">{$t('discover.court_type')}</p>
          <div class="flex gap-2 flex-wrap">
            {#each Object.entries(courtLabels) as [val, label]}
              <button
                onclick={() => toggleCourt(val)}
                class="px-3 py-1.5 rounded-full text-xs font-medium border transition-colors
                  {selectedCourts.includes(val) ? 'bg-primary-600 text-white border-primary-600' : 'border-gray-300 dark:border-gray-600 text-gray-600 dark:text-gray-400 hover:border-primary-400'}">
                {label}
              </button>
            {/each}
          </div>
        </div>

        <!-- Dia da semana -->
        <div>
          <p class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">{$t('discover.weekday')}</p>
          <div class="flex gap-2 flex-wrap">
            {#each WEEKDAY_LABELS as label, i}
              <button
                onclick={() => toggleWeekday(i)}
                class="px-3 py-1.5 rounded-full text-xs font-medium border transition-colors
                  {selectedWeekdays.includes(i) ? 'bg-primary-600 text-white border-primary-600' : 'border-gray-300 dark:border-gray-600 text-gray-600 dark:text-gray-400 hover:border-primary-400'}">
                {label}
              </button>
            {/each}
          </div>
        </div>

        <div class="flex gap-2 pt-1">
          <button onclick={applyFilters} class="btn btn-sm btn-primary">{$t('discover.apply')}</button>
          <button onclick={clearFilters} class="btn btn-sm btn-ghost text-gray-500">
            <X size={13} /> {$t('discover.clear')}
          </button>
        </div>
      </div>
    {/if}

    <!-- Resultados -->
    {#if loading && items.length === 0}
      <div class="space-y-3">
        {#each [1,2,3] as _}
          <div class="card px-4 py-4 animate-pulse">
            <div class="h-4 bg-gray-200 dark:bg-gray-700 rounded w-1/3 mb-2"></div>
            <div class="h-3 bg-gray-100 dark:bg-gray-800 rounded w-2/3"></div>
          </div>
        {/each}
      </div>
    {:else if items.length === 0}
      <div class="card px-6 py-12 text-center">
        <p class="text-4xl mb-3">⚽</p>
        <p class="text-base font-semibold text-gray-700 dark:text-gray-300">{$t('discover.no_results')}</p>
        <p class="text-sm text-gray-400 dark:text-gray-500 mt-1">{$t('discover.no_results_desc')}</p>
        {#if activeFilterCount > 0}
          <button onclick={clearFilters} class="btn btn-sm btn-ghost mt-4 text-primary-600 dark:text-primary-400">
            <X size={13} /> {$t('discover.clear_filters')}
          </button>
        {/if}
      </div>
    {:else}
      <div class="space-y-3">
        {#each items as dm}
          <div class="card px-4 py-4">
            <div class="flex items-start gap-3">
              <div class="w-10 h-10 rounded-lg bg-blue-100 dark:bg-blue-900/30 flex items-center justify-center shrink-0 mt-0.5">
                <Calendar size={18} class="text-blue-600 dark:text-blue-400" />
              </div>
              <div class="flex-1 min-w-0">
                <div class="flex items-start justify-between gap-2">
                  <div class="min-w-0">
                    <p class="text-sm font-bold text-gray-900 dark:text-gray-100">{dm.group_name}</p>
                    <p class="text-xs text-primary-600 dark:text-primary-400 font-medium mt-0.5">
                      {new Date(dm.match_date + 'T12:00').toLocaleDateString($locale, { weekday: 'long', day: '2-digit', month: 'long' })}
                    </p>
                  </div>
                  <span class="shrink-0 text-xs font-semibold px-2 py-0.5 rounded-full
                    {dm.spots_left !== null && dm.spots_left <= 3 ? 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400' : 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400'}">
                    {dm.spots_left !== null
                      ? (dm.spots_left !== 1 ? $t('discover.spots_plural').replace('{n}', String(dm.spots_left)) : $t('discover.spots').replace('{n}', String(dm.spots_left)))
                      : $t('discover.spots_open')}
                  </span>
                </div>
                <div class="flex flex-wrap items-center gap-x-3 gap-y-0.5 mt-1.5">
                  <span class="text-xs text-gray-500 dark:text-gray-400 flex items-center gap-1">
                    <Clock size={11} />{dm.start_time.slice(0,5)}{dm.end_time ? ` – ${dm.end_time.slice(0,5)}` : ''}
                  </span>
                  <span class="text-xs text-gray-500 dark:text-gray-400 flex items-center gap-1 min-w-0">
                    <MapPin size={11} /><span class="truncate">{dm.location}</span>
                  </span>
                  {#if dm.court_type}
                    <span class="text-xs text-gray-400 dark:text-gray-500">{courtLabels[dm.court_type] ?? dm.court_type}</span>
                  {/if}
                  {#if dm.players_per_team}
                    <span class="text-xs text-gray-400 dark:text-gray-500">{dm.players_per_team}×{dm.players_per_team}</span>
                  {/if}
                </div>
                {#if dm.notes}
                  <p class="text-xs text-gray-400 dark:text-gray-500 mt-1.5 line-clamp-2">{dm.notes}</p>
                {/if}
              </div>
            </div>
            <div class="flex items-center justify-between mt-3 pt-3 border-t border-gray-100 dark:border-gray-700">
              <a href="/match/{dm.hash}" class="text-xs text-primary-600 dark:text-primary-400 hover:underline">
                {$t('discover.see_details')}
              </a>
              <button
                onclick={() => { waitlistMatch = dm; showModal = true; }}
                class="btn btn-sm btn-primary">
                {$t('discover.want_to_play')}
              </button>
            </div>
          </div>
        {/each}

        {#if hasMore}
          <div class="text-center pt-2">
            <button
              onclick={() => load()}
              disabled={loading}
              class="btn btn-sm btn-ghost text-gray-500 dark:text-gray-400 disabled:opacity-40">
              {loading ? $t('discover.loading') : $t('discover.load_more')}
            </button>
          </div>
        {:else if items.length > 0}
          <p class="text-center text-xs text-gray-400 dark:text-gray-500 py-4">{$t('discover.all_loaded')}</p>
        {/if}
      </div>
    {/if}
  </main>
</PageBackground>

{#if waitlistMatch && showModal}
  <WaitlistModal
    bind:open={showModal}
    match={{ ...waitlistMatch, attendances: [], confirmed_count: waitlistMatch.confirmed_count, declined_count: 0, pending_count: 0, group_name: waitlistMatch.group_name, group_per_match_amount: null, group_monthly_amount: null, group_is_public: true }}
    {submitting}
    onsubmit={submitWaitlist}
    onclose={() => { showModal = false; waitlistMatch = null; }}
  />
{/if}
