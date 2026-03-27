<script lang="ts">
  import { page } from '$app/stores';
  import { players as playersApi, ApiError } from '$lib/api';
  import type { PlayerPublicStats } from '$lib/api';
  import { isLoggedIn } from '$lib/stores/auth';
  import { playerDisplayName } from '$lib/utils.js';
  import { t } from '$lib/i18n';
  import { untrack } from 'svelte';
  import PageBackground from '$lib/components/PageBackground.svelte';
  import AvatarImage from '$lib/components/AvatarImage.svelte';
  import JoinCTABanner from '$lib/components/JoinCTABanner.svelte';
  import { toastSuccess, toastError } from '$lib/stores/toast';
  import { ChevronLeft, Share2 } from 'lucide-svelte';

  let stats = $state<PlayerPublicStats | null>(null);
  let loading = $state(true);
  let error = $state('');

  let playerId = $derived($page.params.id);

  let _loaded = false;
  $effect(() => {
    const id = playerId;
    if (!id || _loaded) return;
    _loaded = true;
    untrack(async () => {
      try {
        const data = await playersApi.getPublicStats(id);
        stats = data;
      } catch (e) {
        error = e instanceof ApiError ? e.message : $t('player_public.not_found');
      } finally {
        loading = false;
      }
    });
  });

  async function shareScore() {
    const url = `https://rachao.app/players/${playerId}`;
    if (navigator.share) {
      try {
        await navigator.share({
          title: stats ? playerDisplayName(stats.name, stats.nickname) + ' — Rachão Score' : 'Rachão Score',
          url,
        });
      } catch {
        /* user cancelled */
      }
    } else {
      try {
        await navigator.clipboard.writeText(url);
        toastSuccess($t('player_public.share_copied'));
      } catch {
        toastError($t('player_public.share_error'));
      }
    }
  }

  function displayName(s: PlayerPublicStats) {
    return playerDisplayName(s.name, s.nickname);
  }
</script>

<svelte:head>
  {#if stats}
    <title>{displayName(stats)} — Rachão Score | rachao.app</title>
    <meta name="description" content="{displayName(stats)} tem {stats.total_matches_confirmed} rachões, {stats.attendance_rate}% de presença e {stats.total_vote_points} pontos no Rachão Score." />
  {:else}
    <title>{$t('player_public.title')} — rachao.app</title>
  {/if}
</svelte:head>

<PageBackground>
  <main class="relative z-10 max-w-lg mx-auto px-4 py-8 {!$isLoggedIn ? 'pb-24' : ''}">

    <!-- Cabeçalho -->
    <div class="flex items-center justify-between mb-6">
      <a href="/" class="text-sm text-white/70 hover:text-white transition-colors flex items-center gap-1">
        <ChevronLeft size={16} />
        {$t('player_public.back')}
      </a>
      {#if stats}
        <button
          onclick={shareScore}
          class="btn btn-sm btn-ghost text-white border border-white/20 hover:bg-white/10 flex items-center gap-1.5">
          <Share2 size={14} /> {$t('player_public.share')}
        </button>
      {/if}
    </div>

    {#if loading}
      <div class="card p-8 text-center animate-pulse">
        <div class="w-20 h-20 rounded-full bg-gray-200 dark:bg-gray-700 mx-auto mb-4"></div>
        <div class="h-5 bg-gray-200 dark:bg-gray-700 rounded w-1/2 mx-auto mb-2"></div>
        <div class="h-3 bg-gray-100 dark:bg-gray-800 rounded w-1/3 mx-auto"></div>
      </div>

    {:else if error}
      <div class="card p-8 text-center">
        <p class="text-4xl mb-3">⚽</p>
        <p class="text-base font-semibold text-gray-700 dark:text-gray-300">{error}</p>
      </div>

    {:else if stats}
      <!-- Card de identidade -->
      <div class="card overflow-hidden mb-4">
        <div class="relative px-4 py-6 text-white text-center overflow-hidden"
          style="background: linear-gradient(135deg, #166534 0%, #15803d 60%, #16a34a 100%);">
          <div class="absolute -right-6 -top-6 w-28 h-28 rounded-full opacity-10 bg-white"></div>
          <div class="absolute -left-4 bottom-0 w-16 h-16 rounded-full opacity-10 bg-white"></div>
          <div class="relative flex flex-col items-center gap-3">
            <AvatarImage
              name={stats.name}
              avatarUrl={stats.avatar_url}
              size={80}
              class="ring-4 ring-white/30"
            />
            <div>
              <p class="text-xl font-bold leading-tight">{displayName(stats)}</p>
              {#if stats.nickname}
                <p class="text-sm text-green-200">{stats.name}</p>
              {/if}
            </div>
            <!-- Estrelas de habilidade -->
            <div class="flex items-center gap-1">
              {#each Array.from({ length: 5 }) as _, i}
                <span class="text-lg {i < stats.skill_stars ? 'text-amber-300' : 'text-white/20'}">★</span>
              {/each}
            </div>
          </div>
        </div>
      </div>

      <!-- Grid de métricas -->
      <div class="grid grid-cols-2 gap-3 mb-4">
        <div class="card card-body flex flex-col items-center py-4">
          <p class="text-2xl font-bold text-gray-900 dark:text-gray-100">{stats.total_matches_confirmed}</p>
          <p class="text-xs text-gray-500 dark:text-gray-400 mt-0.5 text-center">{$t('player_public.matches')}</p>
        </div>
        <div class="card card-body flex flex-col items-center py-4">
          <p class="text-2xl font-bold text-green-600 dark:text-green-400">{stats.attendance_rate}%</p>
          <p class="text-xs text-gray-500 dark:text-gray-400 mt-0.5 text-center">{$t('player_public.attendance')}</p>
        </div>
        <div class="card card-body flex flex-col items-center py-4">
          <p class="text-2xl font-bold text-orange-500 dark:text-orange-400">
            🔥 {stats.current_streak}
          </p>
          <p class="text-xs text-gray-500 dark:text-gray-400 mt-0.5 text-center">{$t('player_public.current_streak')}</p>
        </div>
        <div class="card card-body flex flex-col items-center py-4">
          <p class="text-2xl font-bold text-primary-600 dark:text-primary-400">{stats.best_streak}</p>
          <p class="text-xs text-gray-500 dark:text-gray-400 mt-0.5 text-center">{$t('player_public.best_streak')}</p>
        </div>
      </div>

      <!-- Reputação -->
      <div class="card card-body space-y-2 mb-4">
        <p class="text-sm text-gray-700 dark:text-gray-200 font-medium">
          {$t('player_public.top5').replace('{n}', String(stats.top5_count))}
        </p>
        <p class="text-sm text-purple-600 dark:text-purple-400 font-medium">
          {$t('player_public.points').replace('{n}', String(stats.total_vote_points))}
        </p>
        {#if stats.total_flop_votes > 0}
          <p class="text-sm text-gray-500 dark:text-gray-400">
            {$t('player_public.flop').replace('{n}', String(stats.total_flop_votes))}
          </p>
        {/if}
      </div>

    {/if}
  </main>
</PageBackground>

<JoinCTABanner />
