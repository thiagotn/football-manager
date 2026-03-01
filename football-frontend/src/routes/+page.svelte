<script lang="ts">
  import { groups, matches } from '$lib/api';
  import type { Group, Match } from '$lib/api';
  import { currentPlayer, isAdmin } from '$lib/stores/auth';
  import { goto } from '$app/navigation';
  import { Trophy, Calendar, Clock, MapPin, ChevronRight } from 'lucide-svelte';

  type MatchWithGroup = Match & { group_name: string; group_slug: string; group_id: string };

  let myGroups: Group[] = $state([]);
  let recentMatches: MatchWithGroup[] = $state([]);
  let loading = $state(true);

  $effect(() => {
    let cancelled = false;
    (async () => {
      try {
        const gs = await groups.list();
        if (cancelled) return;
        myGroups = gs;
        const allMatches: MatchWithGroup[] = [];
        for (const g of gs.slice(0, 3)) {
          const ms = await matches.list(g.id);
          allMatches.push(...ms.slice(0, 2).map(m => ({ ...m, group_name: g.name, group_slug: g.slug, group_id: g.id })));
        }
        if (!cancelled) recentMatches = allMatches
          .sort((a, b) => b.match_date.localeCompare(a.match_date))
          .slice(0, 5);
      } catch (e) { console.error('[dashboard] erro:', e); }
      if (!cancelled) loading = false;
    })();
    return () => { cancelled = true; };
  });

  function fmtDate(d: string) {
    return new Date(d + 'T00:00').toLocaleDateString('pt-BR', { weekday: 'short', day: '2-digit', month: 'short' });
  }
</script>

<svelte:head><title>Dashboard — Joga Bonito</title></svelte:head>

<main class="max-w-7xl mx-auto px-4 py-8">
  <div class="mb-8">
    <h1 class="text-2xl font-bold text-gray-900">
      Olá, {$currentPlayer?.name?.split(' ')[0]} 👋
    </h1>
    <p class="text-gray-500 text-sm mt-1">Veja seus grupos e próximas partidas.</p>
  </div>

  <!-- Stats row -->
  <div class="grid grid-cols-2 gap-4 mb-8 sm:grid-cols-3">
    <div class="card card-body flex items-center gap-4">
      <div class="w-10 h-10 rounded-full bg-primary-100 flex items-center justify-center">
        <Trophy size={20} class="text-primary-600" />
      </div>
      <div>
        <p class="text-2xl font-bold text-gray-900">{myGroups.length}</p>
        <p class="text-xs text-gray-500">Grupos</p>
      </div>
    </div>
    <div class="card card-body flex items-center gap-4">
      <div class="w-10 h-10 rounded-full bg-blue-100 flex items-center justify-center">
        <Calendar size={20} class="text-blue-600" />
      </div>
      <div>
        <p class="text-2xl font-bold text-gray-900">{recentMatches.length}</p>
        <p class="text-xs text-gray-500">Partidas</p>
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
      <div class="divide-y divide-gray-100">
        {#if loading}
          {#each [1,2,3] as _}
            <div class="px-6 py-4 animate-pulse"><div class="h-4 bg-gray-100 rounded w-3/4"></div></div>
          {/each}
        {:else if myGroups.length === 0}
          <div class="px-6 py-8 text-center text-gray-400 text-sm">Você não pertence a nenhum grupo ainda.</div>
        {:else}
          {#each myGroups.slice(0, 5) as g}
            <a href="/groups/{g.id}" class="flex items-center justify-between px-6 py-4 hover:bg-gray-50">
              <div>
                <p class="font-medium text-sm text-gray-900">{g.name}</p>
                {#if g.description}<p class="text-xs text-gray-400 mt-0.5 truncate max-w-xs">{g.description}</p>{/if}
              </div>
              <ChevronRight size={16} class="text-gray-400" />
            </a>
          {/each}
        {/if}
      </div>
    </div>

    <!-- Recent matches -->
    <div class="card">
      <div class="card-header">
        <h2 class="font-semibold flex items-center gap-2"><Calendar size={16} class="text-blue-600" /> Próximas Partidas</h2>
      </div>
      <div class="divide-y divide-gray-100">
        {#if loading}
          {#each [1,2,3] as _}
            <div class="px-6 py-4 animate-pulse"><div class="h-4 bg-gray-100 rounded w-3/4"></div></div>
          {/each}
        {:else if recentMatches.length === 0}
          <div class="px-6 py-8 text-center text-gray-400 text-sm">Nenhuma partida agendada.</div>
        {:else}
          {#each recentMatches as m}
            <a href="/match/{m.hash}" class="flex items-center gap-3 px-6 py-4 hover:bg-gray-50">
              <div class="w-10 h-10 rounded-lg bg-green-100 flex items-center justify-center shrink-0">
                <Calendar size={18} class="text-green-600" />
              </div>
              <div class="flex-1 min-w-0">
                <p class="text-sm font-medium text-gray-900 capitalize">{fmtDate(m.match_date)}
                  <span class="text-xs text-gray-400 font-normal ml-1">#{m.number}</span>
                </p>
                <p class="text-xs text-gray-400 flex items-center gap-1 mt-0.5">
                  <Clock size={11} />{m.start_time.slice(0,5)}
                  <MapPin size={11} class="ml-1" /><span class="truncate">{m.location}</span>
                </p>
                <p class="text-xs text-primary-500 mt-0.5 font-medium">/{m.group_slug}</p>
              </div>
              <span class="badge {m.status === 'open' ? 'badge-green' : 'badge-gray'}">
                {m.status === 'open' ? 'Aberta' : 'Encerrada'}
              </span>
            </a>
          {/each}
        {/if}
      </div>
    </div>
  </div>
</main>
