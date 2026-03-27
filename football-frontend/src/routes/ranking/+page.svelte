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

  type Period = 'month' | 'year' | 'all';
  type TabType = 'top' | 'flop';

  const periods: { value: Period; labelKey: string }[] = [
    { value: 'month', labelKey: 'ranking.period_month' },
    { value: 'year',  labelKey: 'ranking.period_year' },
    { value: 'all',   labelKey: 'ranking.period_all' },
  ];

  let selectedPeriod = $state<Period>('month');
  let activeTab = $state<TabType>('top');

  let topItems = $state<RankingTopItem[]>([]);
  let flopItems = $state<RankingFlopItem[]>([]);
  let loading = $state(true);
  let error = $state<string | null>(null);

  async function loadRanking(period: Period) {
    loading = true;
    error = null;
    try {
      const [topRes, flopRes] = await Promise.all([
        rankingApi.get(period, 'top'),
        rankingApi.get(period, 'flop'),
      ]);
      topItems = topRes.items as RankingTopItem[];
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
    loadRanking(selectedPeriod);
  });

  function setPeriod(p: Period) {
    if (p === selectedPeriod) return;
    selectedPeriod = p;
    loadRanking(p);
  }

  // Podium helpers
  const MEDALS: Record<number, string> = { 1: '🥇', 2: '🥈', 3: '🥉' };
  const PODIUM_ORDER = [2, 1, 3];

  function podiumRingColor(pos: number): string {
    if (pos === 1) return 'ring-amber-400';
    if (pos === 2) return 'ring-gray-400';
    return 'ring-amber-700';
  }

  function podiumTextColor(pos: number): string {
    if (pos === 1) return 'text-amber-400';
    if (pos === 2) return 'text-gray-400';
    return 'text-amber-700';
  }

  function podiumHeight(pos: number): string {
    if (pos === 1) return 'h-20';
    if (pos === 2) return 'h-14';
    return 'h-10';
  }

  function podiumBg(pos: number): string {
    if (pos === 1) return 'bg-amber-400 dark:bg-amber-500';
    if (pos === 2) return 'bg-gray-300 dark:bg-gray-500';
    return 'bg-amber-700 dark:bg-amber-800';
  }

  let top3 = $derived(
    PODIUM_ORDER
      .map(p => topItems.find(x => x.position === p))
      .filter(Boolean) as RankingTopItem[]
  );
  let topRest = $derived(topItems.filter(x => x.position > 3));

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
    <div class="flex items-center justify-between mb-6">
      <div>
        <h1 class="text-2xl font-bold text-white flex items-center gap-2">
          <Award size={24} class="text-primary-400" /> {$t('ranking.title')}
        </h1>
        <p class="text-sm text-white/60 mt-0.5">{$t('ranking.subtitle')}</p>
      </div>

      <!-- Period filter: desktop (in header) -->
      <div class="hidden lg:flex gap-1 bg-black/20 rounded-lg p-1">
        {#each periods as p}
          <button
            onclick={() => setPeriod(p.value)}
            class="px-3 py-1.5 rounded-md text-sm font-medium transition-colors
              {selectedPeriod === p.value
                ? 'bg-primary-500 text-white'
                : 'text-white/70 hover:text-white hover:bg-white/10'}"
          >
            {$t(p.labelKey)}
          </button>
        {/each}
      </div>
    </div>

    <!-- Period filter: mobile (below header) -->
    <div class="lg:hidden flex gap-1 bg-black/20 rounded-lg p-1 mb-4">
      {#each periods as p}
        <button
          onclick={() => setPeriod(p.value)}
          class="flex-1 px-2 py-1.5 rounded-md text-sm font-medium transition-colors
            {selectedPeriod === p.value
              ? 'bg-primary-500 text-white'
              : 'text-white/70 hover:text-white hover:bg-white/10'}"
        >
          {$t(p.labelKey)}
        </button>
      {/each}
    </div>

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
            <div class="flex justify-center gap-4">
              {#each [0, 1, 2] as _}
                <div class="flex flex-col items-center gap-2">
                  <div class="w-12 h-12 bg-gray-200 dark:bg-gray-700 rounded-full"></div>
                  <div class="h-3 w-16 bg-gray-200 dark:bg-gray-700 rounded"></div>
                  <div class="h-3 w-10 bg-gray-200 dark:bg-gray-700 rounded"></div>
                </div>
              {/each}
            </div>
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
        <button onclick={() => loadRanking(selectedPeriod)} class="mt-4 btn btn-secondary btn-sm">
          Tentar novamente
        </button>
      </div>

    {:else}
      <div class="grid grid-cols-1 lg:grid-cols-2 gap-6 items-start">

        <!-- Top section -->
        <div class="{activeTab !== 'top' ? 'hidden lg:block' : ''}">
          <div class="card overflow-hidden">
            <!-- Section header -->
            <div class="px-4 py-3 border-b border-gray-100 dark:border-gray-700 flex items-center gap-2">
              <span class="text-base">🏆</span>
              <p class="text-sm font-semibold text-gray-700 dark:text-gray-200">{$t('ranking.tab_top')}</p>
            </div>

            {#if topItems.length === 0}
              <!-- Empty state -->
              <div class="px-4 py-10 text-center">
                <p class="text-4xl mb-3 text-gray-300 dark:text-gray-600">🏆</p>
                <p class="text-sm font-medium text-gray-600 dark:text-gray-300">{$t('ranking.empty')}</p>
                <p class="text-xs text-gray-400 dark:text-gray-500 mt-1">{$t('ranking.empty_sub')}</p>
              </div>
            {:else}
              <!-- Podium (top 3) -->
              {#if top3.length > 0}
                <div class="px-4 pt-6 pb-4">
                  <div class="flex items-end justify-center gap-3">
                    {#each top3 as item}
                      {@const pos = item.position}
                      <div class="flex flex-col items-center gap-1 flex-1">
                        <!-- Avatar with ring -->
                        <div class="ring-2 {podiumRingColor(pos)} rounded-full">
                          <AvatarImage
                            name={item.name}
                            avatarUrl={item.avatar_url}
                            size={48}
                          />
                        </div>
                        <!-- Name -->
                        <p class="text-center text-xs font-semibold text-gray-800 dark:text-gray-100 leading-tight line-clamp-2 px-1 mt-1">
                          {playerDisplayName(item.name, item.nickname)}
                        </p>
                        <!-- Medal + points -->
                        <span class="text-lg">{MEDALS[pos]}</span>
                        <span class="text-xs font-bold {podiumTextColor(pos)}">{item.total_points} {$t('ranking.points_label')}</span>
                        <!-- Podium block -->
                        <div class="w-full {podiumHeight(pos)} {podiumBg(pos)} rounded-t-lg flex items-center justify-center mt-1">
                          <span class="text-white font-bold text-base">{pos}º</span>
                        </div>
                      </div>
                    {/each}
                  </div>
                </div>
              {/if}

              <!-- 4th–10th list -->
              {#if topRest.length > 0}
                <div class="divide-y divide-gray-100 dark:divide-gray-700 border-t border-gray-100 dark:border-gray-700">
                  {#each topRest as item}
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
                      <span class="text-xs font-bold text-primary-600 dark:text-primary-400 shrink-0">{item.total_points} {$t('ranking.points_label')}</span>
                    </div>
                  {/each}
                </div>
              {/if}

              <!-- Current player highlight in top3 -->
              {#if top3.some(i => isCurrentPlayer(i.player_id))}
                {@const me = top3.find(i => isCurrentPlayer(i.player_id))!}
                <div class="mx-4 mb-3 mt-1 flex items-center gap-2 bg-primary-50 dark:bg-primary-900/20 rounded-lg px-3 py-2">
                  <AvatarImage name={me.name} avatarUrl={me.avatar_url} size={24} />
                  <span class="text-sm font-medium text-gray-800 dark:text-gray-100 flex-1 truncate">
                    {playerDisplayName(me.name, me.nickname)}
                  </span>
                  <span class="inline-flex items-center bg-primary-100 text-primary-700 dark:bg-primary-800 dark:text-primary-200 text-xs px-1.5 py-0.5 rounded font-medium">{$t('ranking.you_badge')}</span>
                  <span class="text-xs font-bold text-primary-600 dark:text-primary-400">{me.total_points} {$t('ranking.points_label')}</span>
                </div>
              {/if}
            {/if}
          </div>
        </div>

        <!-- Flop section -->
        <div class="{activeTab !== 'flop' ? 'hidden lg:block' : ''}">
          <div class="card overflow-hidden">
            <!-- Section header -->
            <div class="px-4 py-3 border-b border-gray-100 dark:border-gray-700 flex items-center gap-2 bg-red-50 dark:bg-red-900/10">
              <span class="text-base">😬</span>
              <p class="text-sm font-semibold text-red-600 dark:text-red-400">{$t('ranking.tab_flop')}</p>
            </div>

            {#if flopItems.length === 0}
              <!-- Empty state -->
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
