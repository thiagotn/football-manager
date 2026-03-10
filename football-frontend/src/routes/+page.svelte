<script lang="ts">
  import { groups, matches, players as playersApi } from '$lib/api';
  import type { Group, Match, SignupStats } from '$lib/api';
  import { authStore, currentPlayer, isAdmin } from '$lib/stores/auth';
  import { goto } from '$app/navigation';
  import { Trophy, Calendar, Clock, MapPin, ChevronRight, Users, UserPlus } from 'lucide-svelte';
  import PageBackground from '$lib/components/PageBackground.svelte';
  import { relativeDate, formatWhatsapp } from '$lib/utils.js';

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

  function fmtPlaytime(minutes: number): string {
    if (minutes < 60) return `${minutes}min`;
    const hours = minutes / 60;
    const rounded = Math.round(hours * 10) / 10;
    return rounded.toLocaleString('pt-BR');
  }

  function fmtPlaytimeMobile(minutes: number): string {
    if (minutes < 60) return `${minutes}min`;
    const hours = minutes / 60;
    const rounded = Math.round(hours * 10) / 10;
    return `${rounded.toLocaleString('pt-BR')}h`;
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

  $effect(() => {
    fetchDashboard();

    function onVisibilityChange() {
      if (document.visibilityState === 'visible') fetchDashboard();
    }
    document.addEventListener('visibilitychange', onVisibilityChange);
    return () => document.removeEventListener('visibilitychange', onVisibilityChange);
  });

  function fmtDate(d: string) {
    return relativeDate(d, { weekday: 'short', day: '2-digit', month: 'short' });
  }

  function daysAgo(iso: string): string {
    const diff = Math.floor((Date.now() - new Date(iso).getTime()) / 86400000);
    if (diff === 0) return 'hoje';
    if (diff === 1) return 'ontem';
    return `${diff}d atrás`;
  }
</script>

<svelte:head><title>Dashboard — rachao.app</title></svelte:head>

<PageBackground>
  <main class="relative z-10 max-w-7xl mx-auto px-4 py-8">
  <div class="mb-8">
    <h1 class="text-2xl font-bold text-white">
      Olá, {$currentPlayer?.name?.split(' ')[0]} 👋
    </h1>
    <p class="text-gray-300 text-sm mt-1">Veja os próximos rachões e seus grupos.</p>
  </div>

  <!-- Stats row -->
  <div class="grid gap-4 mb-8 {$isAdmin ? 'grid-cols-4' : 'grid-cols-3'}">
    <div class="card p-4 flex flex-col items-center text-center gap-1.5"
      title="{$isAdmin ? 'Total de rachões cadastrados na plataforma' : 'Próximos rachões agendados'}">
      <div class="w-8 h-8 rounded-full bg-blue-100 dark:bg-blue-900/30 flex items-center justify-center">
        <Calendar size={16} class="text-blue-600 dark:text-blue-400" />
      </div>
      <p class="text-2xl font-bold text-gray-900 dark:text-gray-100 leading-none">{$isAdmin ? platformTotalMatches : upcomingMatches.length}</p>
      <p class="text-xs text-gray-500 dark:text-gray-400">
        <span class="hidden sm:inline">{$isAdmin ? 'Rachões' : 'Próximos'}</span>
        <span class="sm:hidden">{$isAdmin ? 'Rachões' : 'Próximos'}</span>
      </p>
    </div>
    <div class="card p-4 flex flex-col items-center text-center gap-1.5"
      title="Grupos que você participa">
      <div class="w-8 h-8 rounded-full bg-primary-100 dark:bg-primary-900/30 flex items-center justify-center">
        <Trophy size={16} class="text-primary-600 dark:text-primary-400" />
      </div>
      <p class="text-2xl font-bold text-gray-900 dark:text-gray-100 leading-none">{myGroups.length}</p>
      <p class="text-xs text-gray-500 dark:text-gray-400">
        <span class="hidden sm:inline">Grupos</span>
        <span class="sm:hidden">Grupos</span>
      </p>
    </div>
    {#if $isAdmin}
      <div class="card p-4 flex flex-col items-center text-center gap-1.5"
        title="Jogadores ativos cadastrados">
        <div class="w-8 h-8 rounded-full bg-green-100 dark:bg-green-900/30 flex items-center justify-center">
          <Users size={16} class="text-green-600 dark:text-green-400" />
        </div>
        <p class="text-2xl font-bold text-gray-900 dark:text-gray-100 leading-none">{playerCount}</p>
        <p class="text-xs text-gray-500 dark:text-gray-400">
          <span class="hidden sm:inline">Jogadores</span>
          <span class="sm:hidden">Jogadores</span>
        </p>
      </div>
    {/if}
    <div class="card p-4 flex flex-col items-center text-center gap-1.5"
      title="{$isAdmin ? 'Total de horas de partidas encerradas na plataforma' : 'Horas jogadas em partidas encerradas com presença confirmada'}">
      <div class="w-8 h-8 rounded-full bg-orange-100 dark:bg-orange-900/30 flex items-center justify-center">
        <Clock size={16} class="text-orange-600 dark:text-orange-400" />
      </div>
      <p class="text-2xl font-bold text-gray-900 dark:text-gray-100 leading-none">
        <span class="hidden sm:inline">{fmtPlaytime($isAdmin ? platformMinutesPlayed : minutesPlayed)}</span>
        <span class="sm:hidden">{fmtPlaytimeMobile($isAdmin ? platformMinutesPlayed : minutesPlayed)}</span>
      </p>
      <p class="text-xs text-gray-500 dark:text-gray-400">
        <span class="hidden sm:inline">{$isAdmin ? 'Horas jogadas' : 'Horas jogadas'}</span>
        <span class="sm:hidden">Jogadas</span>
      </p>
    </div>
  </div>

  <div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
    <!-- Matches with tabs — primeiro no mobile -->
    <div class="card">
      <div class="card-header pb-0">
        <div class="flex gap-1 border-b border-gray-200 dark:border-gray-700 -mb-px">
          <button
            class="px-4 py-2 text-sm font-medium border-b-2 transition-colors {matchTab === 'upcoming' ? 'border-primary-600 text-primary-600' : 'border-transparent text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300'}"
            onclick={() => matchTab = 'upcoming'}>
            Próximos Rachões
          </button>
          <button
            class="px-4 py-2 text-sm font-medium border-b-2 transition-colors {matchTab === 'past' ? 'border-primary-600 text-primary-600' : 'border-transparent text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300'}"
            onclick={() => matchTab = 'past'}>
            Últimos Rachões
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
          {@const empty = matchTab === 'past' ? 'Nenhum rachão encerrado ainda.' : 'Nenhum rachão agendado.'}
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
                        Bola rolando
                      </span>
                    {:else}
                      <span class="badge shrink-0 {m.status === 'open' ? 'badge-green' : 'badge-gray'}">
                        {m.status === 'open' ? 'Aberta' : 'Encerrada'}
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
        <h2 class="font-semibold flex items-center gap-2"><Trophy size={16} class="text-primary-600" /> Meus Grupos</h2>
        <a href="/groups" class="text-xs text-primary-600 hover:underline font-medium">Ver todos</a>
      </div>
      <div class="divide-y divide-gray-100 dark:divide-gray-700">
        {#if loading}
          {#each [1,2,3] as _}
            <div class="px-6 py-4 animate-pulse"><div class="h-4 bg-gray-100 dark:bg-gray-700 rounded w-3/4"></div></div>
          {/each}
        {:else if myGroups.length === 0}
          <div class="px-6 py-8 text-center text-gray-400 dark:text-gray-500 text-sm">Você não pertence a nenhum grupo ainda.</div>
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
  <!-- ── Admin-only: Novos Cadastros ──────────────────────── -->
  {#if $isAdmin && signupStats}
    <div class="mt-6">
      <h2 class="text-base font-semibold text-white flex items-center gap-2 mb-3">
        <UserPlus size={16} class="text-primary-400" /> Novos Cadastros
      </h2>

      <div class="grid grid-cols-3 gap-3 mb-4">
        <div class="card p-4 text-center">
          <p class="text-2xl font-bold text-primary-600 dark:text-primary-400">{signupStats.total}</p>
          <p class="text-xs text-gray-500 dark:text-gray-400 mt-0.5">Total</p>
        </div>
        <div class="card p-4 text-center">
          <p class="text-2xl font-bold text-green-600 dark:text-green-400">{signupStats.last_7_days}</p>
          <p class="text-xs text-gray-500 dark:text-gray-400 mt-0.5">Últimos 7 dias</p>
        </div>
        <div class="card p-4 text-center">
          <p class="text-2xl font-bold text-blue-600 dark:text-blue-400">{signupStats.last_30_days}</p>
          <p class="text-xs text-gray-500 dark:text-gray-400 mt-0.5">Últimos 30 dias</p>
        </div>
      </div>

      <div class="card overflow-hidden">
        <div class="px-4 py-3 border-b border-gray-100 dark:border-gray-700 flex items-center justify-between">
          <span class="text-sm font-medium text-gray-700 dark:text-gray-300">Registros recentes</span>
          <a href="/players" class="text-xs text-primary-600 dark:text-primary-400 hover:underline">Ver todos →</a>
        </div>
        {#if signupStats.recent.length === 0}
          <div class="px-4 py-8 text-center text-sm text-gray-400">Nenhum cadastro ainda.</div>
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
                    {p.active ? 'Ativo' : 'Inativo'}
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
