<script lang="ts">
  import { page } from '$app/stores';
  import { goto } from '$app/navigation';
  import { matches as matchesApi, teams as teamsApi, groups as groupsApi, ApiError } from '$lib/api';
  import type { MatchDetail, TeamsResponse } from '$lib/api';
  import { currentPlayer, isAdmin, isLoggedIn } from '$lib/stores/auth';
  import { toastSuccess, toastError } from '$lib/stores/toast';
  import PageBackground from '$lib/components/PageBackground.svelte';
  import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
  import { ChevronLeft, Copy, RefreshCw, Shield, Clock, MapPin } from 'lucide-svelte';
  import { t } from '$lib/i18n';
  import { formatMatchTimeRange } from '$lib/timezoneUtils';

  const courtLabels: Record<string, string> = {
    society: 'Society', futsal: 'Futsal', campo: 'Campo',
    quadra: 'Quadra', beach: 'Beach Soccer', sintetico: 'Sintético',
  };

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

    <!-- Back button -->
    <button
      onclick={() => goto(`/match/${matchHash}`, { replaceState: true })}
      class="mb-3 flex items-center gap-1 text-sm font-medium text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 transition-colors">
      {$t('match.back')}
    </button>

    <!-- Match banner card -->
    {#if match}
      <div class="card mb-4 overflow-hidden">
        <div class="relative overflow-hidden px-4 py-4 text-white" style="min-height:100px;">
          <picture>
            <source srcset="/banners/banner-{match.court_type ?? 'default'}.webp" type="image/webp" />
            <img
              src="/banners/banner-{match.court_type ?? 'default'}.jpg"
              alt=""
              aria-hidden="true"
              width="1920"
              height="600"
              class="absolute inset-0 w-full h-full object-cover object-center"
            />
          </picture>
          <div class="absolute inset-0 bg-primary-900/80"></div>
          <div class="relative flex items-stretch gap-4">
            <!-- Left: match details -->
            <div class="flex-1 min-w-0">
              <div class="flex items-center flex-wrap gap-x-2 gap-y-1 mb-1">
                <p class="text-sm font-bold text-white">#{match.number} {match.group_name}</p>
                {#if match.status === 'in_progress'}
                  <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-semibold bg-red-500/30 text-red-200 border border-red-400/40">
                    <span class="w-1.5 h-1.5 rounded-full bg-red-400 animate-pulse"></span>
                    {$t('match.live')}
                  </span>
                {:else}
                  <span class="badge {match.status === 'open' ? 'bg-green-400 text-green-900' : 'bg-gray-400 text-gray-900'}">
                    {match.status === 'open' ? $t('match.open') : $t('match.closed')}
                  </span>
                {/if}
              </div>
              <h1 class="text-xl font-bold capitalize">{fmtDate(match.match_date)}</h1>
              <div class="flex flex-wrap gap-3 mt-2 text-primary-100 text-sm">
                <span class="flex items-center gap-1.5"><Clock size={14} />{formatMatchTimeRange(match.start_time, match.end_time, match.group_timezone)}</span>
                {#if match.address}
                  <a
                    href="https://maps.google.com/?q={encodeURIComponent(match.address)}"
                    target="_blank"
                    rel="noopener noreferrer"
                    class="flex items-center gap-1.5 underline underline-offset-2 hover:text-white transition-colors">
                    <MapPin size={14} />{match.location}
                  </a>
                {:else}
                  <span class="flex items-center gap-1.5"><MapPin size={14} />{match.location}</span>
                {/if}
              </div>
              {#if match.court_type || match.players_per_team || match.max_players}
                <div class="flex flex-wrap gap-3 mt-2 text-primary-200 text-xs">
                  {#if match.court_type}
                    <span class="bg-primary-800/40 rounded px-2 py-0.5">{courtLabels[match.court_type] ?? match.court_type}</span>
                  {/if}
                  {#if match.players_per_team}
                    <span class="bg-primary-800/40 rounded px-2 py-0.5">{$t('match.line_plus_goalkeeper').replace('{n}', String(match.players_per_team))}</span>
                  {/if}
                  {#if match.max_players}
                    <span class="bg-primary-800/40 rounded px-2 py-0.5 {match.confirmed_count >= match.max_players ? 'text-red-300 font-semibold' : ''}">
                      {$t('match.spots').replace('{n}', String(match.confirmed_count)).replace('{max}', String(match.max_players))}
                    </span>
                  {/if}
                </div>
              {/if}
            </div>
            <!-- Right: logo -->
            <div class="flex items-center shrink-0 -mt-4 -mb-4 -mr-4">
              <img
                src="/logo.png"
                alt="rachao.app"
                width="320"
                height="174"
                aria-hidden="true"
                class="w-52 drop-shadow-lg pointer-events-none select-none"
              />
            </div>
          </div><!-- /flex row -->
        </div><!-- /banner header -->
      </div>
    {/if}

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
