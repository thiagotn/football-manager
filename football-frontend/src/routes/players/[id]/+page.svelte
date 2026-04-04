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
  import { Share2, Shield } from 'lucide-svelte';
  import { POS_ABBR, POS_COLOR_CLASSES, type Position } from '$lib/team-builder';

  const API_TO_POS: Record<string, Position> = {
    gk: 'goalkeeper', zag: 'defender', lat: 'fullback', mei: 'midfielder', ata: 'forward'
  };

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
        return;
      } catch (e: any) {
        if (e?.name === 'AbortError') return; // usuário cancelou
        // qualquer outro erro → fallback para clipboard
      }
    }
    try {
      await navigator.clipboard.writeText(url);
      toastSuccess($t('player_public.share_copied'));
    } catch {
      toastError($t('player_public.share_error'));
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
    {#if stats}
      <div class="flex items-center justify-end mb-6">
        <button
          onclick={shareScore}
          class="btn btn-sm btn-ghost text-white border border-white/20 hover:bg-white/10 flex items-center gap-1.5">
          <Share2 size={14} /> {$t('player_public.share')}
        </button>
      </div>
    {:else}
      <div class="mb-6"></div>
    {/if}

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
        <div class="relative px-4 py-5 text-white overflow-hidden"
          style="background: linear-gradient(135deg, #166534 0%, #15803d 60%, #16a34a 100%);">
          <div class="relative flex items-center justify-between gap-3">
            <!-- Esquerda: avatar + nome -->
            <div class="flex items-center gap-3 min-w-0">
              <AvatarImage
                name={stats.name}
                avatarUrl={stats.avatar_url}
                size={56}
                class="shrink-0 ring-2 ring-white/30"
              />
              <div class="min-w-0">
                <p class="text-xl font-bold leading-tight truncate">{displayName(stats)}</p>
                {#if stats.nickname}
                  <p class="text-sm text-green-200 truncate">{stats.name}</p>
                {/if}
                <!-- Estrelas de habilidade -->
                <div class="flex items-center gap-0.5 mt-1">
                  {#each Array.from({ length: 5 }) as _, i}
                    <span class="text-base {i < stats.skill_stars ? 'text-amber-300' : 'text-white/20'}">★</span>
                  {/each}
                </div>
              </div>
            </div>
            <!-- Direita: logo -->
            <img src="/logo.png" alt="rachao.app" class="h-14 w-auto shrink-0 opacity-90" />
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

      <!-- Grupos -->
      {#if stats.groups.length > 0}
        <div class="card overflow-hidden mb-4">
          <div class="px-4 py-2.5 border-b border-gray-100 dark:border-gray-700">
            <h3 class="text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wide">{$t('player_public.groups_section')}</h3>
          </div>
          <ul class="divide-y divide-gray-100 dark:divide-gray-700">
            {#each stats.groups as g}
              <li class="px-4 py-2.5 flex items-center gap-2.5">
                <Shield size={13} class="text-primary-500 shrink-0" fill="currentColor" />
                <span class="flex-1 text-sm text-gray-800 dark:text-gray-200 truncate">{g.group_name}</span>
                {#if g.position && API_TO_POS[g.position]}
                  {@const pos = API_TO_POS[g.position]}
                  <span class="text-[9px] px-1.5 py-0.5 rounded font-bold shrink-0 {POS_COLOR_CLASSES[pos]}">{POS_ABBR[pos]}</span>
                {/if}
                <span class="text-xs text-amber-500 tracking-tighter shrink-0">{'★'.repeat(g.skill_stars)}</span>
              </li>
            {/each}
          </ul>
        </div>
      {/if}

    {/if}
  </main>
</PageBackground>

<JoinCTABanner />
