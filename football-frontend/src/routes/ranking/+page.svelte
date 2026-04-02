<script lang="ts">
  import { ranking as rankingApi } from '$lib/api';
  import type { RankingTopItem, RankingFlopItem } from '$lib/api';
  import { currentPlayer } from '$lib/stores/auth';
  import { playerDisplayName } from '$lib/utils.js';
  import { t } from '$lib/i18n';
  import PageBackground from '$lib/components/PageBackground.svelte';
  import AvatarImage from '$lib/components/AvatarImage.svelte';
  import JoinCTABanner from '$lib/components/JoinCTABanner.svelte';
  import { Award } from 'lucide-svelte';

  type TabType = 'top' | 'flop';

  const now = new Date();
  const CURRENT_YEAR  = now.getFullYear();
  const CURRENT_MONTH = now.getMonth() + 1; // 1–12
  const START_YEAR = 2024;

  const YEARS = Array.from({ length: CURRENT_YEAR - START_YEAR + 1 }, (_, i) => START_YEAR + i);
  const MONTHS = Array.from({ length: 12 }, (_, i) => i + 1);

  let selectedYear  = $state<number | null>(CURRENT_YEAR);
  let selectedMonth = $state<number | null>(CURRENT_MONTH);
  let activeTab     = $state<TabType>('top');

  let topItems  = $state<RankingTopItem[]>([]);
  let flopItems = $state<RankingFlopItem[]>([]);
  let loading   = $state(true);
  let error     = $state<string | null>(null);

  async function loadRanking(year: number | null, month: number | null) {
    loading = true;
    error = null;
    try {
      const [topRes, flopRes] = await Promise.all([
        rankingApi.get(year, month, 'top'),
        rankingApi.get(year, month, 'flop'),
      ]);
      topItems  = topRes.items as RankingTopItem[];
      flopItems = flopRes.items as RankingFlopItem[];
    } catch (e) {
      error = String(e);
    } finally {
      loading = false;
    }
  }

  let _initialized = false;
  $effect(() => {
    if (_initialized) return;
    _initialized = true;
    loadRanking(selectedYear, selectedMonth);
  });

  function setAllTime() {
    selectedYear  = null;
    selectedMonth = null;
    loadRanking(null, null);
  }

  function setYear(y: number) {
    selectedYear = y;
    // Se o mês selecionado é futuro para o ano escolhido, volta para o mês atual
    if (selectedMonth !== null && y === CURRENT_YEAR && selectedMonth > CURRENT_MONTH) {
      selectedMonth = CURRENT_MONTH;
    }
    loadRanking(y, selectedMonth);
  }

  function setMonth(m: number) {
    if (isFutureMonth(m)) return;
    selectedMonth = m;
    loadRanking(selectedYear, m);
  }

  function isFutureMonth(m: number): boolean {
    return selectedYear === CURRENT_YEAR && m > CURRENT_MONTH;
  }

  const MEDALS: Record<number, string> = { 1: '🥇', 2: '🥈', 3: '🥉' };

  function isCurrentPlayer(playerId: string): boolean {
    return $currentPlayer?.id === playerId;
  }
</script>

<svelte:head>
  <title>Ranking Geral — rachao.app</title>
</svelte:head>

<PageBackground>
  <main class="relative z-10 max-w-5xl mx-auto px-4 py-8 pb-24">

    <!-- Header -->
    <div class="mb-4">
      <h1 class="text-2xl font-bold text-white flex items-center gap-2">
        <Award size={24} class="text-primary-400" /> {$t('ranking.title')}
      </h1>
      <p class="text-sm text-white/60 mt-0.5">{$t('ranking.subtitle')}</p>
    </div>

    <!-- Picker de ano + todos os tempos -->
    <div class="flex flex-wrap gap-1 bg-black/20 rounded-lg p-1 mb-2">
      {#each YEARS as y}
        <button
          onclick={() => setYear(y)}
          class="px-3 py-1.5 rounded-md text-sm font-medium transition-colors
            {selectedYear === y
              ? 'bg-primary-500 text-white'
              : 'text-white/70 hover:text-white hover:bg-white/10'}"
        >
          {y}
        </button>
      {/each}
      <button
        onclick={setAllTime}
        class="ml-auto px-3 py-1.5 rounded-md text-sm font-medium transition-colors
          {selectedYear === null
            ? 'bg-primary-500 text-white'
            : 'text-white/70 hover:text-white hover:bg-white/10'}"
      >
        {$t('ranking.all_time')}
      </button>
    </div>

    <!-- Picker de mês (só exibido quando um ano está selecionado) -->
    {#if selectedYear !== null}
      <div class="grid grid-cols-6 lg:grid-cols-12 gap-1 bg-black/10 rounded-lg p-1 mb-4">
        {#each MONTHS as m}
          <button
            onclick={() => setMonth(m)}
            disabled={isFutureMonth(m)}
            class="py-1.5 rounded-md text-xs font-medium transition-colors
              {isFutureMonth(m)
                ? 'text-white/20 cursor-not-allowed'
                : selectedMonth === m
                  ? 'bg-primary-500 text-white'
                  : 'text-white/70 hover:text-white hover:bg-white/10'}"
          >
            {$t(`ranking.month_${m}`)}
          </button>
        {/each}
      </div>
    {/if}

    <!-- Tab switcher: mobile only -->
    <div class="lg:hidden flex mb-4 bg-white/10 rounded-xl p-1">
      <button
        onclick={() => activeTab = 'top'}
        class="flex-1 py-2 rounded-lg text-sm font-semibold transition-colors
          {activeTab === 'top' ? 'bg-white text-gray-900' : 'text-white/80 hover:text-white'}"
      >
        🏆 {$t('ranking.tab_top')}
      </button>
      <button
        onclick={() => activeTab = 'flop'}
        class="flex-1 py-2 rounded-lg text-sm font-semibold transition-colors
          {activeTab === 'flop' ? 'bg-white text-gray-900' : 'text-white/80 hover:text-white'}"
      >
        😬 {$t('ranking.tab_flop')}
      </button>
    </div>

    {#if loading}
      <!-- Skeleton -->
      <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {#each [0, 1] as _}
          <div class="card p-4 animate-pulse space-y-4">
            <div class="h-5 bg-gray-200 dark:bg-gray-700 rounded w-1/3"></div>
            {#each [0, 1, 2, 3] as _}
              <div class="flex items-center gap-3">
                <div class="w-8 h-8 bg-gray-200 dark:bg-gray-700 rounded-full shrink-0"></div>
                <div class="flex-1 h-4 bg-gray-200 dark:bg-gray-700 rounded"></div>
                <div class="h-4 w-12 bg-gray-200 dark:bg-gray-700 rounded"></div>
              </div>
            {/each}
          </div>
        {/each}
      </div>

    {:else if error}
      <div class="card p-8 text-center">
        <p class="text-3xl mb-3">⚠️</p>
        <p class="text-sm text-gray-500 dark:text-gray-400">{error}</p>
        <button onclick={() => loadRanking(selectedYear, selectedMonth)} class="mt-4 btn btn-secondary btn-sm">
          Tentar novamente
        </button>
      </div>

    {:else}
      <div class="grid grid-cols-1 lg:grid-cols-2 gap-6 items-start">

        <!-- Top section -->
        <div class="{activeTab !== 'top' ? 'hidden lg:block' : ''}">
          <div class="card overflow-hidden">
            <div class="px-4 py-3 border-b border-gray-100 dark:border-gray-700 flex items-center gap-2">
              <span class="text-base">🏆</span>
              <p class="text-sm font-semibold text-gray-700 dark:text-gray-200">{$t('ranking.tab_top')}</p>
            </div>

            {#if topItems.length === 0}
              <div class="px-4 py-10 text-center">
                <p class="text-4xl mb-3 text-gray-300 dark:text-gray-600">🏆</p>
                <p class="text-sm font-medium text-gray-600 dark:text-gray-300">{$t('ranking.empty')}</p>
                <p class="text-xs text-gray-400 dark:text-gray-500 mt-1">{$t('ranking.empty_sub')}</p>
              </div>
            {:else}
              <div class="divide-y divide-gray-100 dark:divide-gray-700">
                {#each topItems as item}
                  <div class="flex items-center gap-3 px-4 py-2.5
                    {isCurrentPlayer(item.player_id) ? 'bg-primary-50 dark:bg-primary-900/20' : ''}">
                    <AvatarImage name={item.name} avatarUrl={item.avatar_url} size={32} />
                    <span class="text-sm font-semibold w-8 shrink-0 text-center">
                      {#if MEDALS[item.position]}
                        {MEDALS[item.position]}
                      {:else}
                        <span class="text-gray-500 dark:text-gray-400">{item.position}º</span>
                      {/if}
                    </span>
                    <span class="flex-1 text-sm font-medium text-gray-800 dark:text-gray-100 truncate">
                      {playerDisplayName(item.name, item.nickname)}
                      {#if isCurrentPlayer(item.player_id)}
                        <span class="ml-1 inline-flex items-center bg-primary-100 text-primary-700 dark:bg-primary-800 dark:text-primary-200 text-xs px-1.5 py-0.5 rounded font-medium">{$t('ranking.you_badge')}</span>
                      {/if}
                    </span>
                    <span class="text-xs font-bold text-primary-600 dark:text-primary-400 shrink-0">{item.total_points} {$t('ranking.points_label')}</span>
                  </div>
                {/each}
              </div>
            {/if}
          </div>
        </div>

        <!-- Flop section -->
        <div class="{activeTab !== 'flop' ? 'hidden lg:block' : ''}">
          <div class="card overflow-hidden">
            <div class="px-4 py-3 border-b border-gray-100 dark:border-gray-700 flex items-center gap-2 bg-red-50 dark:bg-red-900/10">
              <span class="text-base">😬</span>
              <p class="text-sm font-semibold text-red-600 dark:text-red-400">{$t('ranking.tab_flop')}</p>
            </div>

            {#if flopItems.length === 0}
              <div class="px-4 py-10 text-center">
                <p class="text-4xl mb-3 text-gray-300 dark:text-gray-600">😬</p>
                <p class="text-sm font-medium text-gray-600 dark:text-gray-300">{$t('ranking.empty')}</p>
                <p class="text-xs text-gray-400 dark:text-gray-500 mt-1">{$t('ranking.empty_sub')}</p>
              </div>
            {:else}
              <div class="divide-y divide-gray-100 dark:divide-gray-700">
                {#each flopItems as item}
                  <div class="flex items-center gap-3 px-4 py-2.5
                    {isCurrentPlayer(item.player_id) ? 'bg-primary-50 dark:bg-primary-900/20' : ''}">
                    <AvatarImage name={item.name} avatarUrl={item.avatar_url} size={32} />
                    <span class="text-sm font-semibold text-gray-500 dark:text-gray-400 w-6 shrink-0 text-right">{item.position}º</span>
                    <span class="flex-1 text-sm font-medium text-gray-800 dark:text-gray-100 truncate">
                      {playerDisplayName(item.name, item.nickname)}
                      {#if isCurrentPlayer(item.player_id)}
                        <span class="ml-1 inline-flex items-center bg-primary-100 text-primary-700 dark:bg-primary-800 dark:text-primary-200 text-xs px-1.5 py-0.5 rounded font-medium">{$t('ranking.you_badge')}</span>
                      {/if}
                    </span>
                    <span class="text-xs font-bold text-red-600 dark:text-red-400 shrink-0">{item.total_flop_votes} {$t('ranking.votes_label')}</span>
                  </div>
                {/each}
              </div>
            {/if}
          </div>
        </div>

      </div>
    {/if}
  </main>
</PageBackground>

<JoinCTABanner />
