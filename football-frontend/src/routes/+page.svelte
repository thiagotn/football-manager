<script lang="ts">
  import { groups, matches, players as playersApi, votes as votesApi } from '$lib/api';
  import type { DiscoverMatch, Group, Match, SignupStats, VotePendingItem } from '$lib/api';
  import WaitlistModal from '$lib/components/WaitlistModal.svelte';
  import { authStore, currentPlayer, isAdmin, isLoggedIn } from '$lib/stores/auth';
  import { goto } from '$app/navigation';
  import { Trophy, Calendar, Clock, MapPin, ChevronRight, Users, UserPlus, Compass } from 'lucide-svelte';
  import PageBackground from '$lib/components/PageBackground.svelte';
  import { relativeDate, formatWhatsapp } from '$lib/utils.js';
  import { t, locale } from '$lib/i18n';

  type MatchWithGroup = Match & { group_name: string; group_slug: string; group_id: string };

  let myGroups: Group[] = $state([]);
  let allMatches: MatchWithGroup[] = $state([]);
  let loading = $state(true);
  let matchTab: 'past' | 'upcoming' = $state('upcoming');
  let playerCount = $state(0);
  let minutesPlayed = $state(0);
  let platformMinutesPlayed = $state(0);
  let platformTotalMatches = $state(0);
  let signupStats: SignupStats | null = $state(null);
  let pendingVotes: VotePendingItem[] = $state([]);
  let discoverMatches: DiscoverMatch[] = $state([]);
  let discoverWaitlistMatch = $state<DiscoverMatch | null>(null);
  let showDiscoverModal = $state(false);
  let discoverSubmitting = $state(false);

  function fmtPlaytime(minutes: number): string {
    if (minutes === 0) return '0min';
    if (minutes < 60) return `${minutes}min`;
    const h = Math.floor(minutes / 60);
    const min = minutes % 60;
    return min > 0 ? `${h}h${String(min).padStart(2, '0')}` : `${h}h`;
  }

  const today = new Date().toISOString().slice(0, 10);

  function matchSortKey(m: { match_date: string; start_time: string }) {
    return `${m.match_date}T${m.start_time}`;
  }

  let upcomingMatches = $derived(
    allMatches
      .filter(m => m.status === 'open' || m.status === 'in_progress')
      .sort((a, b) => matchSortKey(a).localeCompare(matchSortKey(b)))
      .slice(0, 8)
  );

  let pastMatches = $derived(
    allMatches
      .filter(m => m.status === 'closed')
      .sort((a, b) => matchSortKey(b).localeCompare(matchSortKey(a)))
      .slice(0, 8)
  );

  async function fetchDashboard() {
    try {
      const fetchGroups = groups.list();
      const fetchPlayers = $isAdmin ? playersApi.list() : Promise.resolve(null);
      const fetchStats = playersApi.myStats();
      const fetchSignups = $isAdmin ? playersApi.signupStats(30) : Promise.resolve(null);
      const [gs, pl, stats, signups] = await Promise.all([fetchGroups, fetchPlayers, fetchStats, fetchSignups]);
      myGroups = gs;
      if (pl) playerCount = pl.filter(p => p.id !== $currentPlayer?.id).length;
      minutesPlayed = stats.minutes_played;
      platformMinutesPlayed = stats.platform_minutes_played ?? 0;
      platformTotalMatches = stats.platform_total_matches ?? 0;
      if (signups) signupStats = signups;
      const fetched: MatchWithGroup[] = [];
      await Promise.all(gs.map(async g => {
        const ms = await matches.list(g.id);
        fetched.push(...ms.map(m => ({ ...m, group_name: g.name, group_slug: g.slug, group_id: g.id })));
      }));
      allMatches = fetched;
    } catch (e) { console.error('[dashboard] erro:', e); }
    loading = false;
  }

  // Redireciona super admins para o painel dedicado
  $effect(() => {
    if (!$authStore.loading && $isAdmin) {
      goto('/admin', { replaceState: true });
    }
  });

  // Carrega votações pendentes para jogadores não-admin
  $effect(() => {
    if ($authStore.loading || $isAdmin || !$isLoggedIn) return;
    votesApi.getPending()
      .then(r => { pendingVotes = r.items; })
      .catch(() => {});
  });

  // Feed de descoberta — apenas jogadores não-admin
  $effect(() => {
    if ($authStore.loading || $isAdmin || !$isLoggedIn) return;
    matches.discover({ limit: 3 })
      .then(r => { discoverMatches = r; })
      .catch(() => {});
  });

  $effect(() => {
    fetchDashboard();

    function onVisibilityChange() {
      if (document.visibilityState === 'visible') fetchDashboard();
    }
    document.addEventListener('visibilitychange', onVisibilityChange);
    return () => document.removeEventListener('visibilitychange', onVisibilityChange);
  });

  function fmtDate(d: string) {
    return relativeDate(d, { weekday: 'short', day: '2-digit', month: 'short' }, $locale, {
      today: $t('date.today'),
      tomorrow: $t('date.tomorrow'),
      yesterday: $t('date.yesterday')
    });
  }

  function daysAgo(iso: string): string {
    const diff = Math.floor((Date.now() - new Date(iso).getTime()) / 86400000);
    if (diff === 0) return $t('date.today').toLowerCase();
    if (diff === 1) return $t('date.yesterday').toLowerCase();
    return `${diff}d`;
  }
</script>

<svelte:head><title>Dashboard — rachao.app</title></svelte:head>

<PageBackground>
  <main class="relative z-10 max-w-7xl mx-auto px-4 py-8">
  <div class="mb-8">
    <h1 class="text-2xl font-bold text-white">
      {$t('dash.greeting').replace('{name}', $currentPlayer?.name?.split(' ')[0] ?? '')}
    </h1>
    <p class="text-gray-300 text-sm mt-1">{$t('dash.subtitle')}</p>
  </div>

  <!-- Stats row -->
  <div class="grid gap-2 sm:gap-4 mb-8 {$isAdmin ? 'grid-cols-4' : 'grid-cols-3'}">
    <svelte:element this={$isAdmin ? 'div' : 'a'} href={$isAdmin ? undefined : '/matches'}
      class="card p-4 flex flex-col items-center text-center gap-1.5 {$isAdmin ? '' : 'hover:shadow-md transition-shadow cursor-pointer'}"
      title="{$isAdmin ? 'Total de rachões cadastrados na plataforma' : 'Próximos rachões agendados'}">
      <div class="w-8 h-8 rounded-full bg-blue-100 dark:bg-blue-900/30 flex items-center justify-center">
        <Calendar size={16} class="text-blue-600 dark:text-blue-400" />
      </div>
      <p class="text-2xl font-bold text-gray-900 dark:text-gray-100 leading-none">{$isAdmin ? platformTotalMatches : upcomingMatches.length}</p>
      <p class="text-xs text-gray-500 dark:text-gray-400">{$isAdmin ? $t('dash.platform_matches') : $t('dash.upcoming')}</p>
    </svelte:element>
    <a href="/groups" class="card p-4 flex flex-col items-center text-center gap-1.5 hover:shadow-md transition-shadow"
      title="Grupos que você participa">
      <div class="w-8 h-8 rounded-full bg-primary-100 dark:bg-primary-900/30 flex items-center justify-center">
        <Trophy size={16} class="text-primary-600 dark:text-primary-400" />
      </div>
      <p class="text-2xl font-bold text-gray-900 dark:text-gray-100 leading-none">{myGroups.length}</p>
      <p class="text-xs text-gray-500 dark:text-gray-400">{myGroups.length === 1 ? $t('dash.groups_one') : $t('dash.groups_other')}</p>
    </a>
    {#if $isAdmin}
      <div class="card p-4 flex flex-col items-center text-center gap-1.5"
        title="Jogadores ativos cadastrados">
        <div class="w-8 h-8 rounded-full bg-green-100 dark:bg-green-900/30 flex items-center justify-center">
          <Users size={16} class="text-green-600 dark:text-green-400" />
        </div>
        <p class="text-2xl font-bold text-gray-900 dark:text-gray-100 leading-none">{playerCount}</p>
        <p class="text-xs text-gray-500 dark:text-gray-400">{$t('dash.players')}</p>
      </div>
    {/if}
    <svelte:element this={$isAdmin ? 'div' : 'a'} href={$isAdmin ? undefined : '/profile/stats'}
      class="card p-4 flex flex-col items-center text-center gap-1.5 {$isAdmin ? '' : 'hover:shadow-md transition-shadow cursor-pointer'}"
      title="{$isAdmin ? 'Total de horas de partidas encerradas na plataforma' : 'Horas jogadas em partidas encerradas com presença confirmada'}">
      <div class="w-8 h-8 rounded-full bg-orange-100 dark:bg-orange-900/30 flex items-center justify-center">
        <Clock size={16} class="text-orange-600 dark:text-orange-400" />
      </div>
      <p class="text-2xl font-bold text-gray-900 dark:text-gray-100 leading-none">
        {fmtPlaytime($isAdmin ? platformMinutesPlayed : minutesPlayed)}
      </p>
      <p class="text-xs text-gray-500 dark:text-gray-400">{$t('dash.played')}</p>
    </svelte:element>
  </div>

  <!-- Banner de votações pendentes -->
  {#if pendingVotes.length > 0}
    <div class="mb-6 space-y-2">
      {#each pendingVotes as pv}
        <a
          href="/match/{pv.match_hash}"
          class="flex items-center gap-3 px-4 py-3 rounded-xl bg-amber-50 dark:bg-amber-900/20 border border-amber-300 dark:border-amber-700/60 hover:bg-amber-100 dark:hover:bg-amber-900/30 transition-colors">
          <span class="text-xl shrink-0">🏆</span>
          <div class="flex-1 min-w-0">
            <p class="text-sm font-semibold text-amber-800 dark:text-amber-200">{$t('dash.vote_banner').replace('{number}', String(pv.match_number))}</p>
            <p class="text-xs text-amber-600 dark:text-amber-400">{pv.group_name} · {pv.time_label} · {pv.voter_count} de {pv.eligible_count} já votaram</p>
          </div>
          <span class="text-amber-600 dark:text-amber-400 text-sm font-bold shrink-0">→</span>
        </a>
      {/each}
    </div>
  {/if}

  <div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
    <!-- Matches with tabs — primeiro no mobile -->
    <div class="card">
      <div class="card-header pb-0">
        <div class="flex gap-1 border-b border-gray-200 dark:border-gray-700 -mb-px">
          <button
            class="px-4 py-2 text-sm font-medium border-b-2 transition-colors {matchTab === 'upcoming' ? 'border-primary-600 text-primary-600' : 'border-transparent text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300'}"
            onclick={() => matchTab = 'upcoming'}>
            {$t('dash.upcoming_matches')}
          </button>
          <button
            class="px-4 py-2 text-sm font-medium border-b-2 transition-colors {matchTab === 'past' ? 'border-primary-600 text-primary-600' : 'border-transparent text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300'}"
            onclick={() => matchTab = 'past'}>
            {$t('dash.past_matches')}
          </button>
        </div>
      </div>
      <div class="divide-y divide-gray-100 dark:divide-gray-700">
        {#if loading}
          {#each [1,2,3] as _}
            <div class="px-6 py-4 animate-pulse"><div class="h-4 bg-gray-100 dark:bg-gray-700 rounded w-3/4"></div></div>
          {/each}
        {:else}
          {@const list = matchTab === 'past' ? pastMatches : upcomingMatches}
          {@const empty = matchTab === 'past' ? $t('dash.no_past') : $t('dash.no_upcoming')}
          {#if list.length === 0}
            <div class="px-6 py-8 text-center text-gray-400 dark:text-gray-500 text-sm">{empty}</div>
          {:else}
            {#each list as m}
              <a href="/match/{m.hash}" class="flex items-start gap-3 px-4 py-3 hover:bg-gray-50 dark:hover:bg-gray-700">
                <div class="w-9 h-9 rounded-lg {m.status === 'open' ? 'bg-green-100 dark:bg-green-900/30' : m.status === 'in_progress' ? 'bg-red-100 dark:bg-red-900/30' : 'bg-gray-100 dark:bg-gray-700'} flex items-center justify-center shrink-0 mt-0.5">
                  <Calendar size={16} class="{m.status === 'open' ? 'text-green-600 dark:text-green-400' : m.status === 'in_progress' ? 'text-red-500 dark:text-red-400' : 'text-gray-400 dark:text-gray-500'}" />
                </div>
                <div class="flex-1 min-w-0">
                  <div class="flex items-center justify-between gap-2">
                    <p class="text-sm font-medium text-gray-900 dark:text-gray-100 capitalize leading-tight">
                      {fmtDate(m.match_date)}
                      <span class="text-xs text-gray-400 dark:text-gray-500 font-normal ml-1">#{m.number}</span>
                    </p>
                    {#if m.status === 'in_progress'}
                      <span class="shrink-0 inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-semibold bg-red-500/20 text-red-400 border border-red-500/30">
                        <span class="w-1.5 h-1.5 rounded-full bg-red-400 animate-pulse"></span>
                        {$t('dash.live')}
                      </span>
                    {:else}
                      <span class="badge shrink-0 {m.status === 'open' ? 'badge-green' : 'badge-gray'}">
                        {m.status === 'open' ? $t('dash.open') : $t('dash.closed')}
                      </span>
                    {/if}
                  </div>
                  <p class="text-xs text-gray-400 dark:text-gray-500 flex flex-wrap items-center gap-x-2 mt-0.5">
                    <span class="flex items-center gap-1"><Clock size={11} />{m.start_time.slice(0,5)}{m.end_time ? ` – ${m.end_time.slice(0,5)}` : ''}</span>
                    <span class="flex items-center gap-1 min-w-0"><MapPin size={11} /><span class="truncate">{m.location}</span></span>
                  </p>
                  <p class="text-xs text-primary-500 mt-0.5 font-medium">{m.group_name}</p>
                </div>
              </a>
            {/each}
          {/if}
        {/if}
      </div>
    </div>

    <!-- Groups — segundo no mobile -->
    <div class="card">
      <div class="card-header flex items-center justify-between">
        <h2 class="font-semibold flex items-center gap-2"><Trophy size={16} class="text-primary-600" /> {$t('dash.my_groups')}</h2>
        <a href="/groups" class="text-xs text-primary-600 hover:underline font-medium">{$t('dash.see_all')}</a>
      </div>
      <div class="divide-y divide-gray-100 dark:divide-gray-700">
        {#if loading}
          {#each [1,2,3] as _}
            <div class="px-6 py-4 animate-pulse"><div class="h-4 bg-gray-100 dark:bg-gray-700 rounded w-3/4"></div></div>
          {/each}
        {:else if myGroups.length === 0}
          <div class="px-6 py-8 text-center text-gray-400 dark:text-gray-500 text-sm">{$t('dash.no_groups')}</div>
        {:else}
          {#each myGroups.slice(0, 5) as g}
            <a href="/groups/{g.id}" class="flex items-center justify-between px-6 py-4 hover:bg-gray-50 dark:hover:bg-gray-700">
              <div>
                <p class="font-medium text-sm text-gray-900 dark:text-gray-100">{g.name}</p>
                {#if g.description}<p class="text-xs text-gray-400 dark:text-gray-500 mt-0.5 truncate max-w-xs">{g.description}</p>{/if}
              </div>
              <ChevronRight size={16} class="text-gray-400 dark:text-gray-500" />
            </a>
          {/each}
        {/if}
      </div>
    </div>
  </div>
  <!-- ── Discover: Rachões com vaga ──────────────────────── -->
  {#if !$isAdmin}
    <div class="mt-6">
      <div class="flex items-center justify-between mb-3">
        <h2 class="text-base font-semibold text-white flex items-center gap-2">
          <Compass size={16} class="text-primary-400" /> {$t('dash.discover_title')}
        </h2>
        <a href="/discover" class="text-xs text-primary-400 hover:text-primary-300 font-medium">{$t('dash.discover_see_all')}</a>
      </div>
      {#if discoverMatches.length === 0}
        <a href="/discover" class="card px-4 py-6 flex flex-col items-center gap-2 text-center hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors">
          <Compass size={28} class="text-primary-400" />
          <p class="text-sm font-medium text-gray-700 dark:text-gray-300">{$t('dash.discover_cta')}</p>
          <p class="text-xs text-gray-400 dark:text-gray-500">{$t('dash.discover_cta_sub')}</p>
          <span class="text-xs text-primary-600 dark:text-primary-400 font-semibold mt-1">{$t('dash.discover_explore')}</span>
        </a>
      {:else}
        <div class="space-y-2">
          {#each discoverMatches as dm}
            <div class="card px-4 py-3">
              <div class="flex items-start gap-3">
                <div class="w-9 h-9 rounded-lg bg-blue-100 dark:bg-blue-900/30 flex items-center justify-center shrink-0 mt-0.5">
                  <Calendar size={16} class="text-blue-600 dark:text-blue-400" />
                </div>
                <div class="flex-1 min-w-0">
                  <p class="text-sm font-semibold text-gray-900 dark:text-gray-100">{dm.group_name}</p>
                  <p class="text-xs text-gray-500 dark:text-gray-400 flex flex-wrap gap-x-2 mt-0.5">
                    <span class="flex items-center gap-1"><Calendar size={11} />{new Date(dm.match_date + 'T12:00').toLocaleDateString($locale, { weekday: 'short', day: '2-digit', month: 'short' })}</span>
                    <span class="flex items-center gap-1"><Clock size={11} />{dm.start_time.slice(0,5)}</span>
                    <span class="flex items-center gap-1 min-w-0"><MapPin size={11} /><span class="truncate">{dm.location}</span></span>
                  </p>
                  <p class="text-xs mt-1 {dm.spots_left !== null && dm.spots_left <= 3 ? 'text-amber-500 dark:text-amber-400 font-medium' : 'text-gray-400 dark:text-gray-500'}">
                    {dm.spots_left !== null
                      ? (dm.spots_left !== 1 ? $t('dash.spots_available_plural').replace('{n}', String(dm.spots_left)) : $t('dash.spots_available').replace('{n}', String(dm.spots_left)))
                      : $t('dash.spots_open')}
                  </p>
                </div>
              </div>
              <div class="flex items-center justify-between mt-2 pt-2 border-t border-gray-100 dark:border-gray-700">
                <a href="/match/{dm.hash}" class="text-xs text-primary-600 dark:text-primary-400 hover:underline">
                  {$t('discover.see_details')}
                </a>
                <button
                  onclick={() => { discoverWaitlistMatch = dm; showDiscoverModal = true; }}
                  class="btn btn-sm btn-primary">
                  {$t('dash.want_to_play')}
                </button>
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </div>
  {/if}

  {#if discoverWaitlistMatch && showDiscoverModal}
    <WaitlistModal
      bind:open={showDiscoverModal}
      match={{ ...discoverWaitlistMatch, attendances: [], confirmed_count: discoverWaitlistMatch.confirmed_count, declined_count: 0, pending_count: 0, group_name: discoverWaitlistMatch.group_name, group_per_match_amount: null, group_monthly_amount: null, group_is_public: true }}
      submitting={discoverSubmitting}
      onsubmit={async (data) => {
        if (!discoverWaitlistMatch) return;
        discoverSubmitting = true;
        try {
          await groups.joinWaitlist(discoverWaitlistMatch.group_id, data);
          const removedId = discoverWaitlistMatch.id;
          showDiscoverModal = false;
          discoverWaitlistMatch = null;
          discoverMatches = discoverMatches.filter(m => m.id !== removedId);
        } catch { /* errors shown by api */ } finally {
          discoverSubmitting = false;
        }
      }}
      onclose={() => { showDiscoverModal = false; discoverWaitlistMatch = null; }}
    />
  {/if}

  <!-- ── Admin-only: Novos Cadastros ──────────────────────── -->
  {#if $isAdmin && signupStats}
    <div class="mt-6">
      <h2 class="text-base font-semibold text-white flex items-center gap-2 mb-3">
        <UserPlus size={16} class="text-primary-400" /> {$t('dash.admin_signups')}
      </h2>

      <div class="grid grid-cols-3 gap-2 sm:gap-3 mb-4">
        <div class="card p-4 text-center">
          <p class="text-2xl font-bold text-primary-600 dark:text-primary-400">{signupStats.total}</p>
          <p class="text-xs text-gray-500 dark:text-gray-400 mt-0.5">{$t('dash.total')}</p>
        </div>
        <div class="card p-4 text-center">
          <p class="text-2xl font-bold text-green-600 dark:text-green-400">{signupStats.last_7_days}</p>
          <p class="text-xs text-gray-500 dark:text-gray-400 mt-0.5">{$t('dash.last_7_days')}</p>
        </div>
        <div class="card p-4 text-center">
          <p class="text-2xl font-bold text-blue-600 dark:text-blue-400">{signupStats.last_30_days}</p>
          <p class="text-xs text-gray-500 dark:text-gray-400 mt-0.5">{$t('dash.last_30_days')}</p>
        </div>
      </div>

      <div class="card overflow-hidden">
        <div class="px-4 py-3 border-b border-gray-100 dark:border-gray-700 flex items-center justify-between">
          <span class="text-sm font-medium text-gray-700 dark:text-gray-300">{$t('dash.recent_registrations')}</span>
          <a href="/players" class="text-xs text-primary-600 dark:text-primary-400 hover:underline">{$t('dash.see_all_players')}</a>
        </div>
        {#if signupStats.recent.length === 0}
          <div class="px-4 py-8 text-center text-sm text-gray-400">{$t('dash.no_registrations')}</div>
        {:else}
          <div class="divide-y divide-gray-100 dark:divide-gray-700">
            {#each signupStats.recent as p}
              <div class="flex items-center justify-between px-4 py-3 gap-3">
                <div class="min-w-0">
                  <p class="font-medium text-sm text-gray-900 dark:text-gray-100 truncate">
                    {p.nickname ? `${p.nickname} (${p.name})` : p.name}
                  </p>
                  <p class="text-xs text-gray-500 dark:text-gray-400 font-mono">{formatWhatsapp(p.whatsapp)}</p>
                </div>
                <div class="text-right shrink-0">
                  <span class="text-xs {p.active ? 'text-green-600 dark:text-green-400' : 'text-red-500'} font-medium">
                    {p.active ? $t('dash.active') : $t('dash.inactive')}
                  </span>
                  <p class="text-xs text-gray-400 mt-0.5 flex items-center gap-1 justify-end">
                    <Clock size={10} />
                    {daysAgo(p.created_at)}
                  </p>
                </div>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    </div>
  {/if}

  </main>
</PageBackground>
