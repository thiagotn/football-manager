<script lang="ts">
  import { page } from '$app/stores';
  import { goto } from '$app/navigation';
  import { matches as matchesApi, matchStats, groups as groupsApi, ApiError } from '$lib/api';
  import type { MatchDetail, MatchPlayerStatsResponse } from '$lib/api';
  import { currentPlayer, isAdmin, isLoggedIn } from '$lib/stores/auth';
  import { toastSuccess, toastError } from '$lib/stores/toast';
  import PageBackground from '$lib/components/PageBackground.svelte';
  import MatchBannerCard from '$lib/components/MatchBannerCard.svelte';
  import { Save } from 'lucide-svelte';
  import { t } from '$lib/i18n';
  import { playerDisplayName } from '$lib/utils';

  const matchHash = $page.params.hash;

  let match = $state<MatchDetail | null>(null);
  let loading = $state(true);
  let isGroupAdmin = $state(false);
  let adminChecked = $state(false);
  let statsMap = $state<Record<string, { goals: number; assists: number }>>({});
  let saving = $state(false);

  $effect(() => {
    let cancelled = false;
    (async () => {
      try {
        const m = await matchesApi.getByHash(matchHash);
        if (cancelled) return;
        match = m;
        const ps = await matchStats.getPublic(matchHash).catch(() => null);
        if (cancelled) return;
        const initial: Record<string, { goals: number; assists: number }> = {};
        if (ps?.stats) {
          for (const s of ps.stats) initial[s.player_id] = { goals: s.goals, assists: s.assists };
        }
        statsMap = initial;
      } catch {
        if (!cancelled) match = null;
      }
      if (!cancelled) loading = false;
    })();
    return () => { cancelled = true; };
  });

  $effect(() => {
    const player = $currentPlayer;
    const m = match;
    if (!player || !m) return;
    if (player.role === 'admin') { isGroupAdmin = true; adminChecked = true; return; }
    groupsApi.get(m.group_id)
      .then(g => {
        isGroupAdmin = g.members.some(mb => mb.player.id === player.id && mb.role === 'admin');
        adminChecked = true;
      })
      .catch(() => { isGroupAdmin = false; adminChecked = true; });
  });

  // Only redirect once both data and admin check are complete
  $effect(() => {
    if (!loading && adminChecked && match) {
      const allowed = match.status === 'in_progress' || match.status === 'closed';
      if (!$isLoggedIn || (!isGroupAdmin && !$isAdmin) || !allowed) {
        goto(`/match/${matchHash}`);
      }
    }
  });

  function setGoals(playerId: string, v: number) {
    statsMap = { ...statsMap, [playerId]: { goals: v, assists: statsMap[playerId]?.assists ?? 0 } };
  }

  function setAssists(playerId: string, v: number) {
    statsMap = { ...statsMap, [playerId]: { goals: statsMap[playerId]?.goals ?? 0, assists: v } };
  }

  async function save() {
    if (!match) return;
    saving = true;
    try {
      const confirmed = match.attendances.filter(a => a.status === 'confirmed');
      const payload = confirmed.map(a => ({
        player_id: a.player.id,
        goals: statsMap[a.player.id]?.goals ?? 0,
        assists: statsMap[a.player.id]?.assists ?? 0,
      }));
      await matchStats.put(match.hash, payload);
      toastSuccess($t('match.stats_saved'));
      goto(`/match/${matchHash}`);
    } catch {
      toastError($t('match.stats_save_error'));
    } finally {
      saving = false;
    }
  }

  let confirmed = $derived(match?.attendances.filter(a => a.status === 'confirmed') ?? []);
</script>

<svelte:head><title>{$t('match.stats_section')} — rachao.app</title></svelte:head>

<PageBackground>
  {#if loading}
    <div class="flex items-center justify-center min-h-screen">
      <div class="w-8 h-8 border-4 border-primary-500 border-t-transparent rounded-full animate-spin"></div>
    </div>
  {:else if match}
    <main class="relative z-10 max-w-lg mx-auto px-4 py-6">
      <MatchBannerCard {match} />

      <!-- Editor card -->
      <div class="card overflow-hidden mt-4">
        <div class="px-4 py-3 border-b border-gray-100 dark:border-gray-700 flex items-center justify-between">
          <h2 class="text-sm font-semibold text-gray-700 dark:text-gray-200">📊 {$t('match.stats_section')}</h2>
          <div class="flex items-center gap-1 text-xs text-gray-400 dark:text-gray-500">
            <span>⚽ {$t('match.stats_col_goals')}</span>
            <span class="mx-1">·</span>
            <span>🅰 {$t('match.stats_col_assists')}</span>
          </div>
        </div>

        {#if confirmed.length === 0}
          <p class="px-4 py-6 text-sm text-center text-gray-400 dark:text-gray-500">{$t('match.stats_no_confirmed')}</p>
        {:else}
          <div class="divide-y divide-gray-100 dark:divide-gray-700">
            {#each confirmed as a}
              <div class="flex items-center gap-2 px-4 py-2.5">
                <span class="flex-1 text-sm text-gray-800 dark:text-gray-100 truncate min-w-0">
                  {playerDisplayName(a.player.name, a.player.nickname)}
                </span>
                <label class="flex items-center gap-1 shrink-0">
                  <span class="text-base leading-none">⚽</span>
                  <input
                    type="number" min="0" max="20"
                    class="w-12 text-center text-sm rounded-lg border border-gray-200 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-800 dark:text-gray-100 py-1.5 px-1 focus:outline-none focus:ring-2 focus:ring-primary-500"
                    value={statsMap[a.player.id]?.goals ?? 0}
                    oninput={(e) => { const v = Math.max(0, Math.min(20, parseInt((e.target as HTMLInputElement).value) || 0)); setGoals(a.player.id, v); }}
                  />
                </label>
                <label class="flex items-center gap-1 shrink-0">
                  <span class="text-base leading-none">🅰</span>
                  <input
                    type="number" min="0" max="20"
                    class="w-12 text-center text-sm rounded-lg border border-gray-200 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-800 dark:text-gray-100 py-1.5 px-1 focus:outline-none focus:ring-2 focus:ring-primary-500"
                    value={statsMap[a.player.id]?.assists ?? 0}
                    oninput={(e) => { const v = Math.max(0, Math.min(20, parseInt((e.target as HTMLInputElement).value) || 0)); setAssists(a.player.id, v); }}
                  />
                </label>
              </div>
            {/each}
          </div>

          <div class="px-4 py-3 border-t border-gray-100 dark:border-gray-700">
            <button
              onclick={save}
              disabled={saving}
              class="w-full btn btn-primary justify-center gap-2">
              <Save size={15} />
              {saving ? $t('match.stats_saving') : $t('match.stats_save')}
            </button>
          </div>
        {/if}
      </div>
    </main>
  {/if}
</PageBackground>
