<script>
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { currentPlayer } from '$lib/stores/auth';
  import { groups } from '$lib/api';
  import { formatDate, formatTime } from '$lib/utils.js';
  import { Users, Calendar, UserCircle, TrendingUp, Plus } from 'lucide-svelte';

  let myGroups = [];
  let recentMatches = [];
  let loading = true;

  onMount(async () => {
    try {
      myGroups = await groups.list();
    } finally {
      loading = false;
    }
  });

  const statusLabel = { open: 'Aberta', closed: 'Encerrada', cancelled: 'Cancelada' };
</script>

<svelte:head><title>Dashboard — Football App</title></svelte:head>

<main class="max-w-7xl mx-auto px-4 py-8">
  <!-- Header -->
  <div class="mb-8">
    <h1 class="text-2xl font-bold text-gray-900">
      Olá, {$currentPlayer?.name?.split(' ')[0]} 👋
    </h1>
    <p class="text-gray-500 mt-1">Bem-vindo ao Football App</p>
  </div>

  {#if loading}
    <div class="flex items-center justify-center h-40">
      <div class="text-gray-400">Carregando...</div>
    </div>
  {:else}
    <!-- Stats row -->
    <div class="grid grid-cols-2 lg:grid-cols-3 gap-4 mb-8">
      <div class="card p-5">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-sm text-gray-500">Grupos</p>
            <p class="text-3xl font-bold text-brand-700 mt-1">{myGroups.length}</p>
          </div>
          <div class="p-3 bg-brand-50 rounded-xl"><Users class="text-brand-600" size={22} /></div>
        </div>
      </div>
      <div class="card p-5">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-sm text-gray-500">Partidas</p>
            <p class="text-3xl font-bold text-blue-700 mt-1">{recentMatches.length}</p>
          </div>
          <div class="p-3 bg-blue-50 rounded-xl"><Calendar class="text-blue-600" size={22} /></div>
        </div>
      </div>
      <div class="card p-5 col-span-2 lg:col-span-1">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-sm text-gray-500">Partidas abertas</p>
            <p class="text-3xl font-bold text-orange-600 mt-1">
              {recentMatches.filter(m => m.status === 'open').length}
            </p>
          </div>
          <div class="p-3 bg-orange-50 rounded-xl"><TrendingUp class="text-orange-500" size={22} /></div>
        </div>
      </div>
    </div>

    <div class="grid lg:grid-cols-2 gap-6">
      <!-- My Groups -->
      <div class="card">
        <div class="px-5 py-4 border-b flex items-center justify-between">
          <h2 class="font-semibold text-gray-900">Meus Grupos</h2>
          <a href="/groups/new" class="btn-primary text-xs px-3 py-1.5">
            <Plus size={14} /> Novo
          </a>
        </div>
        {#if myGroups.length === 0}
          <div class="px-5 py-10 text-center text-gray-400">
            <Users size={32} class="mx-auto mb-2 opacity-40" />
            <p class="text-sm">Nenhum grupo ainda</p>
          </div>
        {:else}
          <div class="divide-y">
            {#each myGroups as g}
              <a
                href="/groups/{g.id}"
                class="flex items-center justify-between px-5 py-4 hover:bg-gray-50 transition-colors"
              >
                <div>
                  <p class="font-medium text-gray-900">{g.name}</p>
                  <p class="text-xs text-gray-500 mt-0.5">{g.member_count} membros</p>
                </div>
                <svg class="w-4 h-4 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/>
                </svg>
              </a>
            {/each}
          </div>
        {/if}
      </div>

      <!-- Recent Matches -->
      <div class="card">
        <div class="px-5 py-4 border-b">
          <h2 class="font-semibold text-gray-900">Partidas Recentes</h2>
        </div>
        {#if recentMatches.length === 0}
          <div class="px-5 py-10 text-center text-gray-400">
            <Calendar size={32} class="mx-auto mb-2 opacity-40" />
            <p class="text-sm">Nenhuma partida</p>
          </div>
        {:else}
          <div class="divide-y">
            {#each recentMatches as m}
              <a
                href="/matches/{m.slug}"
                class="flex items-center justify-between px-5 py-4 hover:bg-gray-50 transition-colors"
              >
                <div>
                  <p class="font-medium text-gray-900">{m.title || m.location}</p>
                  <p class="text-xs text-gray-500 mt-0.5">
                    {formatDate(m.match_date)} · {formatTime(m.start_time)}
                  </p>
                  <p class="text-xs text-green-600 mt-0.5">
                    ✅ {m.confirmed_count} confirmados
                  </p>
                </div>
                <span class="badge-{m.status}">{statusLabel[m.status]}</span>
              </a>
            {/each}
          </div>
        {/if}
      </div>
    </div>
  {/if}
</main>
