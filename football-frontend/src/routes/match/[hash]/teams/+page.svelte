<script lang="ts">
  import { page } from '$app/stores';
  import { goto } from '$app/navigation';
  import { matches as matchesApi, teams as teamsApi, groups as groupsApi, ApiError } from '$lib/api';
  import type { MatchDetail, TeamsResponse } from '$lib/api';
  import { currentPlayer, isAdmin, isLoggedIn } from '$lib/stores/auth';
  import { toastSuccess, toastError } from '$lib/stores/toast';
  import PageBackground from '$lib/components/PageBackground.svelte';
  import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
  import { ChevronLeft, Copy, RefreshCw, Shield } from 'lucide-svelte';
  import { t } from '$lib/i18n';

  const matchHash = $page.params.hash;

  let match = $state<MatchDetail | null>(null);
  let teamsData = $state<TeamsResponse | null>(null);
  let loading = $state(true);
  let isGroupAdmin = $state(false);
  let confirmOpen = $state(false);
  let regenerating = $state(false);

  $effect(() => {
    let cancelled = false;
    (async () => {
      try {
        const m = await matchesApi.getByHash(matchHash);
        if (cancelled) return;
        match = m;
        const t = await teamsApi.get(m.id);
        if (!cancelled) teamsData = t.teams.length > 0 ? t : null;
      } catch {
        if (!cancelled) { match = null; teamsData = null; }
      }
      if (!cancelled) loading = false;
    })();
    return () => { cancelled = true; };
  });

  $effect(() => {
    const player = $currentPlayer;
    const m = match;
    if (!player || !m) { isGroupAdmin = false; return; }
    if (player.role === 'admin') { isGroupAdmin = true; return; }
    groupsApi.get(m.group_id)
      .then(g => {
        isGroupAdmin = g.members.some(mb => mb.player.id === player.id && mb.role === 'admin');
      })
      .catch(() => { isGroupAdmin = false; });
  });

  async function regenerateTeams() {
    if (!match) return;
    regenerating = true;
    try {
      const result = await teamsApi.generate(match.id);
      teamsData = result;
      toastSuccess($t('teams.rebuilt_success'));
    } catch (e) {
      toastError(e instanceof ApiError ? e.message : $t('teams.rebuild_error'));
    } finally {
      regenerating = false;
    }
  }

  function copyLink() {
    navigator.clipboard.writeText(window.location.href);
    toastSuccess($t('teams.link_copied'));
  }

  const MONTHS_PT = ['jan','fev','mar','abr','mai','jun','jul','ago','set','out','nov','dez'];
  function fmtDate(d: string) {
    const dt = new Date(d + 'T12:00:00');
    return `${dt.getDate()} de ${MONTHS_PT[dt.getMonth()]}`;
  }
</script>

<svelte:head>
  <title>Times — rachao.app</title>
</svelte:head>

<PageBackground>
  <main class="relative z-10 max-w-2xl mx-auto px-3 pb-6 pt-3">

    <!-- Header -->
    <div class="flex items-center gap-2 mb-3">
      <button onclick={() => goto(`/match/${matchHash}`, { replaceState: true })} class="p-1.5 rounded-lg hover:bg-white/10 text-white/80 transition-colors">
        <ChevronLeft size={22} />
      </button>
      <div class="flex-1 min-w-0">
        {#if match}
          <p class="text-white font-semibold truncate">{match.group_name}</p>
          <p class="text-white/60 text-xs">{fmtDate(match.match_date)} · {match.location}</p>
        {:else if loading}
          <p class="text-white/60 text-sm">{$t('teams.loading')}</p>
        {:else}
          <p class="text-white/60 text-sm">{$t('teams.not_found')}</p>
        {/if}
      </div>
      <button onclick={copyLink} class="p-1.5 rounded-lg hover:bg-white/10 text-white/80 transition-colors" title={$t('teams.link_copied')}>
        <Copy size={18} />
      </button>
    </div>

    <!-- Title + admin action -->
    <div class="flex items-center justify-between mb-3">
      <h1 class="text-white text-base font-bold">{$t('teams.title')}</h1>
      {#if !loading && teamsData && isGroupAdmin}
        <button
          onclick={() => confirmOpen = true}
          disabled={regenerating}
          class="btn-secondary btn-sm gap-1 text-xs py-1">
          <RefreshCw size={12} class={regenerating ? 'animate-spin' : ''} />
          {regenerating ? $t('teams.rebuilding') : $t('teams.rebuild')}
        </button>
      {/if}
    </div>

    {#if loading}
      <div class="text-center py-16 text-white/50">{$t('teams.loading')}</div>

    {:else if !match}
      <div class="card card-body text-center text-gray-500 dark:text-gray-400">
        <p>{$t('teams.not_found')}</p>
      </div>

    {:else if !teamsData || teamsData.teams.length === 0}
      <div class="card card-body text-center py-10">
        <p class="text-4xl mb-3">🎲</p>
        <p class="font-semibold text-gray-700 dark:text-gray-200 mb-1">{$t('teams.not_sorted_title')}</p>
        <p class="text-sm text-gray-400 dark:text-gray-500 mb-4">{$t('teams.not_sorted_desc')}</p>
        <a href="/match/{matchHash}" class="btn-primary btn-sm mx-auto">{$t('teams.go_to_match')}</a>
      </div>

    {:else}
      <!-- Primeiro confronto -->
      {#if teamsData.teams.length >= 2}
        {@const t1 = teamsData.teams[0]}
        {@const t2 = teamsData.teams[1]}
        <div class="card overflow-hidden mb-3">
          <div class="px-3 py-2 text-center">
            <p class="text-[10px] font-bold text-gray-400 dark:text-gray-500 uppercase tracking-wide mb-1">{$t('teams.first_match')}</p>
            <div class="flex items-center justify-center gap-2 min-w-0">
              <div class="flex items-center gap-1 min-w-0 justify-end flex-1">
                <span class="text-xs font-bold text-gray-900 dark:text-gray-100 truncate">{t1.name}</span>
                <Shield size={12} class="shrink-0" style="color: {t1.color ?? '#6b7280'};" fill={t1.color ?? '#6b7280'} />
              </div>
              <span class="text-xs font-black text-gray-400 shrink-0">×</span>
              <div class="flex items-center gap-1 min-w-0 flex-1">
                <Shield size={12} class="shrink-0" style="color: {t2.color ?? '#6b7280'};" fill={t2.color ?? '#6b7280'} />
                <span class="text-xs font-bold text-gray-900 dark:text-gray-100 truncate">{t2.name}</span>
              </div>
            </div>
          </div>
        </div>
      {/if}

      <!-- Teams grid — 2 colunas sempre -->
      <div class="grid grid-cols-2 gap-2 mb-3">
        {#each teamsData.teams as team}
          <div class="card overflow-hidden" style="border-left: 3px solid {team.color ?? '#6b7280'}; border-top: 2px solid {team.color ?? '#6b7280'}40;">
            <!-- Team header -->
            <div class="px-2 py-1.5 flex items-center gap-1.5"
              style="background-color: {team.color ?? '#374151'}1a; border-bottom: 2px solid {team.color ?? '#6b7280'};">
              <Shield size={13} class="shrink-0" style="color: {team.color ?? '#6b7280'};" fill={team.color ?? '#6b7280'} />
              <h2 class="font-bold text-xs text-gray-900 dark:text-gray-100 truncate flex-1">{team.name}</h2>
              {#if isGroupAdmin}
                <span class="text-[10px] text-gray-400 shrink-0 leading-none">{team.skill_total}★</span>
              {/if}
            </div>
            <!-- Players -->
            <ul class="divide-y divide-gray-100 dark:divide-gray-700">
              {#each team.players as p}
                <li class="px-2 py-1 flex items-center gap-1">
                  <span class="flex-1 text-xs text-gray-800 dark:text-gray-200 truncate">{p.nickname || p.name}</span>
                  {#if p.is_goalkeeper}
                    <span class="text-[9px] px-1 leading-tight rounded font-bold bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400 shrink-0">GK</span>
                  {/if}
                </li>
              {/each}
            </ul>
            {#if isGroupAdmin}
              <div class="px-2 py-1 border-t border-gray-100 dark:border-gray-700">
                <p class="text-[10px] text-gray-400 dark:text-gray-500">{$t('teams.players_count').replace('{n}', String(team.players.length))}</p>
              </div>
            {/if}
          </div>
        {/each}
      </div>

      <!-- Reserves -->
      {#if teamsData.reserves.length > 0}
        <div class="card overflow-hidden mb-3">
          <div class="px-3 py-1.5 border-b border-gray-100 dark:border-gray-700">
            <h3 class="text-xs font-semibold text-gray-500 dark:text-gray-400">{$t('teams.reserves').replace('{n}', String(teamsData.reserves.length))}</h3>
          </div>
          <div class="px-3 py-1.5 flex flex-wrap gap-x-3 gap-y-0.5">
            {#each teamsData.reserves as p}
              <span class="text-xs text-gray-600 dark:text-gray-400">
                {p.nickname || p.name}{p.is_goalkeeper ? ' (GK)' : ''}
              </span>
            {/each}
          </div>
        </div>
      {/if}

      <!-- Share -->
      <button onclick={copyLink} class="btn btn-secondary w-full justify-center gap-2">
        <Copy size={15} /> {$t('teams.copy_link')}
      </button>
    {/if}

  </main>
</PageBackground>

<ConfirmDialog
  bind:open={confirmOpen}
  message={$t('teams.rebuild_confirm')}
  confirmLabel={$t('teams.rebuild_label')}
  danger={false}
  onConfirm={regenerateTeams}
/>
