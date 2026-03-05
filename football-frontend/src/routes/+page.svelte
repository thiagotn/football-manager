<script lang="ts">
  import { groups, matches } from '$lib/api';
  import type { Group, Match } from '$lib/api';
  import { currentPlayer, isAdmin } from '$lib/stores/auth';
  import { goto } from '$app/navigation';
  import { Trophy, Calendar, Clock, MapPin, ChevronRight } from 'lucide-svelte';

  type MatchWithGroup = Match & { group_name: string; group_slug: string; group_id: string };

  let myGroups: Group[] = $state([]);
  let allMatches: MatchWithGroup[] = $state([]);
  let loading = $state(true);
  let matchTab: 'past' | 'upcoming' = $state('upcoming');

  const today = new Date().toISOString().slice(0, 10);

  let upcomingMatches = $derived(
    allMatches
      .filter(m => m.status === 'open')
      .sort((a, b) => a.match_date.localeCompare(b.match_date))
      .slice(0, 8)
  );

  let pastMatches = $derived(
    allMatches
      .filter(m => m.status === 'closed' || m.match_date < today)
      .sort((a, b) => b.match_date.localeCompare(a.match_date))
      .slice(0, 8)
  );

  $effect(() => {
    let cancelled = false;
    (async () => {
      try {
        const gs = await groups.list();
        if (cancelled) return;
        myGroups = gs;
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
    return new Date(d + 'T00:00').toLocaleDateString('pt-BR', { weekday: 'short', day: '2-digit', month: 'short' });
  }
</script>

<svelte:head><title>Dashboard — rachao.app</title></svelte:head>

<main class="max-w-7xl mx-auto px-4 py-8">
  <div class="mb-8">
    <h1 class="text-2xl font-bold text-gray-900 dark:text-gray-100">
      Olá, {$currentPlayer?.name?.split(' ')[0]} 👋
    </h1>
    <p class="text-gray-500 dark:text-gray-400 text-sm mt-1">Veja seus grupos e próximos rachões.</p>
  </div>

  <!-- Stats row -->
  <div class="grid grid-cols-2 gap-4 mb-8 sm:grid-cols-3">
    <div class="card card-body flex items-center gap-4">
      <div class="w-10 h-10 rounded-full bg-primary-100 dark:bg-primary-900/30 flex items-center justify-center">
        <Trophy size={20} class="text-primary-600 dark:text-primary-400" />
      </div>
      <div>
        <p class="text-2xl font-bold text-gray-900 dark:text-gray-100">{myGroups.length}</p>
        <p class="text-xs text-gray-500 dark:text-gray-400">Grupos</p>
      </div>
    </div>
    <div class="card card-body flex items-center gap-4">
      <div class="w-10 h-10 rounded-full bg-blue-100 dark:bg-blue-900/30 flex items-center justify-center">
        <Calendar size={20} class="text-blue-600 dark:text-blue-400" />
      </div>
      <div>
        <p class="text-2xl font-bold text-gray-900 dark:text-gray-100">{upcomingMatches.length}</p>
        <p class="text-xs text-gray-500 dark:text-gray-400">Próximos rachões</p>
      </div>
    </div>
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
                <div class="w-10 h-10 rounded-lg {m.status === 'open' ? 'bg-green-100 dark:bg-green-900/30' : 'bg-gray-100 dark:bg-gray-700'} flex items-center justify-center shrink-0">
                  <Calendar size={18} class="{m.status === 'open' ? 'text-green-600 dark:text-green-400' : 'text-gray-400 dark:text-gray-500'}" />
                </div>
                <div class="flex-1 min-w-0">
                  <p class="text-sm font-medium text-gray-900 dark:text-gray-100 capitalize">{fmtDate(m.match_date)}
                    <span class="text-xs text-gray-400 dark:text-gray-500 font-normal ml-1">#{m.number}</span>
                  </p>
                  <p class="text-xs text-gray-400 dark:text-gray-500 flex items-center gap-1 mt-0.5">
                    <Clock size={11} />{m.start_time.slice(0,5)}
                    <MapPin size={11} class="ml-1" /><span class="truncate">{m.location}</span>
                  </p>
                  <p class="text-xs text-primary-500 mt-0.5 font-medium">{m.group_name}</p>
                </div>
                <span class="badge {m.status === 'open' ? 'badge-green' : 'badge-gray'}">
                  {m.status === 'open' ? 'Aberta' : 'Encerrada'}
                </span>
              </a>
            {/each}
          {/if}
        {/if}
      </div>
    </div>
  </div>
</main>
