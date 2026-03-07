<script lang="ts">
  import { groups, matches, players as playersApi } from '$lib/api';
  import type { Group, Match } from '$lib/api';
  import { currentPlayer, isAdmin } from '$lib/stores/auth';
  import { goto } from '$app/navigation';
  import { Trophy, Calendar, Clock, MapPin, ChevronRight, Users } from 'lucide-svelte';
  import { relativeDate } from '$lib/utils.js';

  type MatchWithGroup = Match & { group_name: string; group_slug: string; group_id: string };

  let myGroups: Group[] = $state([]);
  let allMatches: MatchWithGroup[] = $state([]);
  let loading = $state(true);
  let matchTab: 'past' | 'upcoming' = $state('upcoming');
  let playerCount = $state(0);

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

  $effect(() => {
    let cancelled = false;
    (async () => {
      try {
        const fetchGroups = groups.list();
        const fetchPlayers = $isAdmin ? playersApi.list() : Promise.resolve(null);
        const [gs, pl] = await Promise.all([fetchGroups, fetchPlayers]);
        if (cancelled) return;
        myGroups = gs;
        if (pl) playerCount = pl.filter(p => p.id !== $currentPlayer?.id).length;
        const fetched: MatchWithGroup[] = [];
        await Promise.all(gs.map(async g => {
          const ms = await matches.list(g.id);
          fetched.push(...ms.map(m => ({ ...m, group_name: g.name, group_slug: g.slug, group_id: g.id })));
        }));
        if (!cancelled) allMatches = fetched;
      } catch (e) { console.error('[dashboard] erro:', e); }
      if (!cancelled) loading = false;
    })();
    return () => { cancelled = true; };
  });

  function fmtDate(d: string) {
    return relativeDate(d, { weekday: 'short', day: '2-digit', month: 'short' });
  }
</script>

<svelte:head><title>Dashboard — rachao.app</title></svelte:head>

<div class="min-h-screen relative bg-gray-900"
  style="background-image: url('/background-login.png'); background-size: cover; background-position: center;">
  <div class="absolute inset-0 bg-gradient-to-b from-gray-900/80 via-gray-900/55 to-gray-900/40 pointer-events-none"></div>
  <main class="relative z-10 max-w-7xl mx-auto px-4 py-8">
  <div class="mb-8">
    <h1 class="text-2xl font-bold text-white">
      Olá, {$currentPlayer?.name?.split(' ')[0]} 👋
    </h1>
    <p class="text-gray-300 text-sm mt-1">Veja seus grupos e próximos rachões.</p>
  </div>

  <!-- Stats row -->
  <div class="grid gap-4 mb-8 {$isAdmin ? 'grid-cols-3' : 'grid-cols-2'}">
    <div class="card p-4 flex flex-col items-center text-center gap-1.5">
      <div class="w-8 h-8 rounded-full bg-primary-100 dark:bg-primary-900/30 flex items-center justify-center">
        <Trophy size={16} class="text-primary-600 dark:text-primary-400" />
      </div>
      <p class="text-2xl font-bold text-gray-900 dark:text-gray-100 leading-none">{myGroups.length}</p>
      <p class="text-xs text-gray-500 dark:text-gray-400">Grupos</p>
    </div>
    <div class="card p-4 flex flex-col items-center text-center gap-1.5">
      <div class="w-8 h-8 rounded-full bg-blue-100 dark:bg-blue-900/30 flex items-center justify-center">
        <Calendar size={16} class="text-blue-600 dark:text-blue-400" />
      </div>
      <p class="text-2xl font-bold text-gray-900 dark:text-gray-100 leading-none">{$isAdmin ? allMatches.length : upcomingMatches.length}</p>
      <p class="text-xs text-gray-500 dark:text-gray-400">{$isAdmin ? 'Rachões' : 'Próximos'}</p>
    </div>
    {#if $isAdmin}
      <div class="card p-4 flex flex-col items-center text-center gap-1.5">
        <div class="w-8 h-8 rounded-full bg-green-100 dark:bg-green-900/30 flex items-center justify-center">
          <Users size={16} class="text-green-600 dark:text-green-400" />
        </div>
        <p class="text-2xl font-bold text-gray-900 dark:text-gray-100 leading-none">{playerCount}</p>
        <p class="text-xs text-gray-500 dark:text-gray-400">Jogadores</p>
      </div>
    {/if}
  </div>

  <div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
    <!-- Groups -->
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

    <!-- Matches with tabs -->
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
              <a href="/match/{m.hash}" class="flex items-center gap-3 px-6 py-4 hover:bg-gray-50 dark:hover:bg-gray-700">
                <div class="w-10 h-10 rounded-lg {m.status === 'open' ? 'bg-green-100 dark:bg-green-900/30' : m.status === 'in_progress' ? 'bg-blue-100 dark:bg-blue-900/30' : 'bg-gray-100 dark:bg-gray-700'} flex items-center justify-center shrink-0">
                  <Calendar size={18} class="{m.status === 'open' ? 'text-green-600 dark:text-green-400' : m.status === 'in_progress' ? 'text-blue-600 dark:text-blue-400' : 'text-gray-400 dark:text-gray-500'}" />
                </div>
                <div class="flex-1 min-w-0">
                  <p class="text-sm font-medium text-gray-900 dark:text-gray-100 capitalize">{fmtDate(m.match_date)}
                    <span class="text-xs text-gray-400 dark:text-gray-500 font-normal ml-1">#{m.number}</span>
                  </p>
                  <p class="text-xs text-gray-400 dark:text-gray-500 flex items-center gap-1 mt-0.5">
                    <Clock size={11} />{m.start_time.slice(0,5)}{m.end_time ? ` – ${m.end_time.slice(0,5)}` : ''}
                    <MapPin size={11} class="ml-1" /><span class="truncate">{m.location}</span>
                  </p>
                  <p class="text-xs text-primary-500 mt-0.5 font-medium">{m.group_name}</p>
                </div>
                <span class="badge {m.status === 'open' ? 'badge-green' : m.status === 'in_progress' ? 'badge-blue' : 'badge-gray'}">
                  {m.status === 'open' ? 'Aberta' : m.status === 'in_progress' ? 'Em andamento' : 'Encerrada'}
                </span>
              </a>
            {/each}
          {/if}
        {/if}
      </div>
    </div>
  </div>
  </main>
</div>
