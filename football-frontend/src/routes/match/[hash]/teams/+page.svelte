<script lang="ts">
  import { page } from '$app/stores';
  import { goto } from '$app/navigation';
  import { matches as matchesApi, teams as teamsApi, groups as groupsApi, ApiError } from '$lib/api';
  import type { MatchDetail, TeamsResponse } from '$lib/api';
  import { currentPlayer, isAdmin, isLoggedIn } from '$lib/stores/auth';
  import { toastSuccess, toastError } from '$lib/stores/toast';
  import PageBackground from '$lib/components/PageBackground.svelte';
  import MatchBannerCard from '$lib/components/MatchBannerCard.svelte';
  import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
  import JoinCTABanner from '$lib/components/JoinCTABanner.svelte';
  import { Copy, RefreshCw, Shield } from 'lucide-svelte';
  import { t } from '$lib/i18n';
  import { playerDisplayName } from '$lib/utils';

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

  function shareTeamsWhatsApp() {
    if (!match || !teamsData) return;

    const MONTHS_PT = ['jan','fev','mar','abr','mai','jun','jul','ago','set','out','nov','dez'];
    const dt = new Date(match.match_date + 'T12:00:00');
    const weekday = dt.toLocaleDateString('pt-BR', { weekday: 'long' });
    const dateStr = `${weekday.charAt(0).toUpperCase() + weekday.slice(1)}, ${dt.getDate()} de ${MONTHS_PT[dt.getMonth()]}`;

    const lines: string[] = [
      `*Times do Rachão — ${match.group_name}*`,
      `${dateStr} · ${match.location}`,
      '',
    ];

    if (teamsData.teams.length >= 2) {
      lines.push(`*1º Jogo do Rachão*`);
      lines.push(`${teamsData.teams[0].name} x ${teamsData.teams[1].name}`);
      lines.push('');
    }

    for (const team of teamsData.teams) {
      lines.push(`*${team.name}*`);
      team.players.forEach((p, i) => {
        const name = playerDisplayName(p.name, p.nickname);
        lines.push(`${i + 1}. ${name}${p.position === 'gk' ? ' (GK)' : ''}`);
      });
      lines.push('');
    }

    if (teamsData.reserves.length > 0) {
      lines.push(`*Reservas*`);
      teamsData.reserves.forEach(p => {
        lines.push(`- ${playerDisplayName(p.name, p.nickname)}${p.position === 'gk' ? ' (GK)' : ''}`);
      });
      lines.push('');
    }

    lines.push(window.location.href);

    const text = encodeURIComponent(lines.join('\n'));
    window.open(`https://wa.me/?text=${text}`, '_blank');
  }

</script>

<svelte:head>
  <title>Times — rachao.app</title>
  <meta property="og:title" content="Times do Rachão — rachao.app" />
  <meta property="og:description" content="Veja os times sorteados para o rachão!" />
  <meta property="og:image" content="https://rachao.app/og-teams.jpg" />
  <meta property="og:image:width" content="1200" />
  <meta property="og:image:height" content="630" />
</svelte:head>

<PageBackground>
  <main class="relative z-10 max-w-2xl mx-auto px-3 pt-3 {$isLoggedIn ? 'pb-6' : 'pb-24'}">

    <!-- Match banner card -->
    {#if match}
      <MatchBannerCard {match} {isGroupAdmin} />
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
                  <span class="flex-1 text-xs text-gray-800 dark:text-gray-200 truncate">{playerDisplayName(p.name, p.nickname)}</span>
                  {#if p.position}
                    <span class="text-[9px] px-1 leading-tight rounded font-bold shrink-0
                      {p.position === 'gk' ? 'bg-amber-400/20 text-amber-300' :
                       p.position === 'zag' ? 'bg-blue-400/20 text-blue-300' :
                       p.position === 'lat' ? 'bg-cyan-400/20 text-cyan-300' :
                       p.position === 'mei' ? 'bg-emerald-400/20 text-emerald-300' :
                       'bg-red-400/20 text-red-300'}">{p.position.toUpperCase()}</span>
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
                {playerDisplayName(p.name, p.nickname)}{p.position === 'gk' ? ' (GK)' : ''}
              </span>
            {/each}
          </div>
        </div>
      {/if}

      <!-- Share -->
      <div class="flex gap-2">
        <button onclick={shareTeamsWhatsApp} class="flex-1 btn btn-secondary btn-sm justify-center gap-1.5">
          <svg xmlns="http://www.w3.org/2000/svg" width="13" height="13" viewBox="0 0 24 24" fill="currentColor" class="shrink-0">
            <path d="M17.472 14.382c-.297-.149-1.758-.867-2.03-.967-.273-.099-.471-.148-.67.15-.197.297-.767.966-.94 1.164-.173.199-.347.223-.644.075-.297-.15-1.255-.463-2.39-1.475-.883-.788-1.48-1.761-1.653-2.059-.173-.297-.018-.458.13-.606.134-.133.298-.347.446-.52.149-.174.198-.298.298-.497.099-.198.05-.371-.025-.52-.075-.149-.669-1.612-.916-2.207-.242-.579-.487-.5-.669-.51-.173-.008-.371-.01-.57-.01-.198 0-.52.074-.792.372-.272.297-1.04 1.016-1.04 2.479 0 1.462 1.065 2.875 1.213 3.074.149.198 2.096 3.2 5.077 4.487.709.306 1.262.489 1.694.625.712.227 1.36.195 1.871.118.571-.085 1.758-.719 2.006-1.413.248-.694.248-1.289.173-1.413-.074-.124-.272-.198-.57-.347m-5.421 7.403h-.004a9.87 9.87 0 0 1-5.031-1.378l-.361-.214-3.741.982.998-3.648-.235-.374a9.86 9.86 0 0 1-1.51-5.26c.001-5.45 4.436-9.884 9.888-9.884 2.64 0 5.122 1.03 6.988 2.898a9.825 9.825 0 0 1 2.893 6.994c-.003 5.45-4.437 9.884-9.885 9.884m8.413-18.297A11.815 11.815 0 0 0 12.05 0C5.495 0 .16 5.335.157 11.892c0 2.096.547 4.142 1.588 5.945L.057 24l6.305-1.654a11.882 11.882 0 0 0 5.683 1.448h.005c6.554 0 11.89-5.335 11.893-11.893a11.821 11.821 0 0 0-3.48-8.413Z"/>
          </svg>
          WhatsApp
        </button>
        <button onclick={copyLink} class="flex-1 btn btn-secondary btn-sm justify-center gap-1.5">
          <Copy size={13} /> {$t('teams.copy_link')}
        </button>
      </div>
    {/if}

  </main>
</PageBackground>

<JoinCTABanner />

<ConfirmDialog
  bind:open={confirmOpen}
  message={$t('teams.rebuild_confirm')}
  confirmLabel={$t('teams.rebuild_label')}
  danger={false}
  onConfirm={regenerateTeams}
/>
