<script>
  import { page } from '$app/stores';
  import { onMount } from 'svelte';
  import { matches as matchesApi } from '$lib/api';
  import { authStore, isLoggedIn, currentPlayer } from '$lib/stores/auth';
  import { formatDate, formatTime, whatsappLink } from '$lib/utils.js';

  const slug = $page.params.slug;
  let match = null;
  let loading = true;
  let error = '';
  let attending = false;

  onMount(async () => {
    try {
      match = await matchesApi.getByHash(slug);
    } catch (e) {
      error = e.message;
    } finally {
      loading = false;
    }
  });

  async function setAttendance(status) {
    if (!$isLoggedIn) return;
    attending = true;
    try {
      await matchesApi.setAttendance(match.group_id, match.id, $currentPlayer.id, status);
      match = await matchesApi.getByHash(slug);
      window.__toast?.(status === 'confirmed' ? '✅ Presença confirmada!' : '❌ Ausência registrada', status === 'confirmed' ? 'success' : 'info');
    } catch (e) {
      window.__toast?.(e.message, 'error');
    } finally {
      attending = false;
    }
  }

  $: confirmed  = match?.attendances?.filter(a => a.status === 'confirmed')  ?? [];
  $: declined   = match?.attendances?.filter(a => a.status === 'declined')   ?? [];
  $: pending    = match?.attendances?.filter(a => a.status === 'pending')    ?? [];

  $: myAttendance = match?.attendances?.find(a => a.player_id === $currentPlayer?.id);

  function shareUrl() {
    return typeof window !== 'undefined' ? window.location.href : '';
  }

  const statusLabel = { open: 'Aberta', closed: 'Encerrada', cancelled: 'Cancelada' };
</script>

<svelte:head>
  <title>{match?.title ?? match?.location ?? 'Partida'} — Football App</title>
  <meta name="description" content="Visualize os detalhes e confirmações desta partida de futebol" />
</svelte:head>

<!-- Minimal public layout -->
<div class="min-h-screen bg-gray-50">
  <!-- Top bar -->
  <div class="bg-brand-700 text-white px-4 py-3">
    <div class="max-w-2xl mx-auto flex items-center justify-between">
      <a href="/" class="flex items-center gap-2 font-semibold">
        <span class="text-xl">⚽</span> Football App
      </a>
      {#if $isLoggedIn}
        <a href="/dashboard" class="text-sm text-brand-200 hover:text-white transition-colors">Dashboard</a>
      {:else}
        <a href="/login" class="text-sm text-brand-200 hover:text-white transition-colors">Entrar</a>
      {/if}
    </div>
  </div>

  <main class="max-w-2xl mx-auto px-4 py-8">
    {#if loading}
      <div class="text-center py-20 text-gray-400">
        <div class="text-4xl mb-3">⚽</div>
        <p>Carregando partida...</p>
      </div>
    {:else if error}
      <div class="card p-8 text-center">
        <div class="text-4xl mb-3">😕</div>
        <h2 class="text-lg font-semibold text-gray-700">Partida não encontrada</h2>
        <p class="text-gray-500 text-sm mt-1">{error}</p>
      </div>
    {:else if match}
      <!-- Match header card -->
      <div class="card p-6 mb-6">
        <div class="flex items-start justify-between flex-wrap gap-2 mb-4">
          <div>
            <div class="flex items-center gap-2 mb-1">
              <span class="badge-{match.status}">{statusLabel[match.status]}</span>
              {#if match.group_name}
                <span class="text-xs text-gray-400">{match.group_name}</span>
              {/if}
            </div>
            <h1 class="text-xl font-bold text-gray-900">
              {match.title ?? match.location}
            </h1>
          </div>
          <button
            class="btn-secondary text-xs px-2.5 py-1.5"
            on:click={() => navigator.share?.({ url: shareUrl() }) ?? navigator.clipboard.writeText(shareUrl())}
          >
            🔗 Compartilhar
          </button>
        </div>

        <div class="grid grid-cols-2 sm:grid-cols-3 gap-4 text-sm">
          <div>
            <p class="text-xs text-gray-400 uppercase tracking-wide mb-1">Data</p>
            <p class="font-medium">{formatDate(match.match_date)}</p>
          </div>
          <div>
            <p class="text-xs text-gray-400 uppercase tracking-wide mb-1">Horário</p>
            <p class="font-medium">{formatTime(match.start_time)}{match.end_time ? ` – ${formatTime(match.end_time)}` : ''}</p>
          </div>
          <div class="col-span-2 sm:col-span-1">
            <p class="text-xs text-gray-400 uppercase tracking-wide mb-1">Local</p>
            <p class="font-medium">📍 {match.location}</p>
          </div>
        </div>

        {#if match.notes}
          <div class="mt-4 bg-gray-50 rounded-lg px-4 py-3 text-sm text-gray-600">{match.notes}</div>
        {/if}
      </div>

      <!-- Attendance action (if logged in & match open) -->
      {#if $isLoggedIn && match.status === 'open'}
        <div class="card p-5 mb-6">
          <h3 class="font-semibold text-gray-900 mb-3">Sua presença</h3>
          {#if myAttendance}
            <p class="text-sm text-gray-500 mb-3">
              Atual:
              <span class="font-medium {myAttendance.status === 'confirmed' ? 'text-green-600' : myAttendance.status === 'declined' ? 'text-red-500' : 'text-yellow-600'}">
                {myAttendance.status === 'confirmed' ? '✅ Confirmada' : myAttendance.status === 'declined' ? '❌ Recusada' : '⏳ Pendente'}
              </span>
            </p>
          {/if}
          <div class="flex gap-3">
            <button
              class="btn-primary flex-1"
              on:click={() => setAttendance('confirmed')}
              disabled={attending || myAttendance?.status === 'confirmed'}
            >
              ✅ Vou comparecer
            </button>
            <button
              class="btn-danger flex-1"
              on:click={() => setAttendance('declined')}
              disabled={attending || myAttendance?.status === 'declined'}
            >
              ❌ Não vou
            </button>
          </div>
        </div>
      {/if}

      <!-- Summary stats -->
      <div class="grid grid-cols-3 gap-3 mb-6">
        <div class="card p-4 text-center">
          <p class="text-2xl font-bold text-green-600">{confirmed.length}</p>
          <p class="text-xs text-gray-500 mt-1">Confirmados</p>
        </div>
        <div class="card p-4 text-center">
          <p class="text-2xl font-bold text-red-500">{declined.length}</p>
          <p class="text-xs text-gray-500 mt-1">Não vão</p>
        </div>
        <div class="card p-4 text-center">
          <p class="text-2xl font-bold text-yellow-500">{pending.length}</p>
          <p class="text-xs text-gray-500 mt-1">Pendentes</p>
        </div>
      </div>

      <!-- Attendance lists -->
      {#if confirmed.length > 0}
        <div class="card mb-4">
          <div class="px-5 py-3 border-b bg-green-50 rounded-t-xl">
            <h3 class="font-semibold text-green-800">✅ Confirmados ({confirmed.length})</h3>
          </div>
          <div class="divide-y">
            {#each confirmed as a}
              <div class="flex items-center justify-between px-5 py-3">
                <div>
                  <p class="font-medium text-sm">{a.player_nickname ?? a.player_name}</p>
                  {#if a.player_nickname}
                    <p class="text-xs text-gray-400">{a.player_name}</p>
                  {/if}
                </div>
                <a
                  href={whatsappLink(a.player_whatsapp)}
                  target="_blank"
                  class="text-green-500 hover:text-green-700 text-xs"
                >
                  WhatsApp
                </a>
              </div>
            {/each}
          </div>
        </div>
      {/if}

      {#if declined.length > 0}
        <div class="card mb-4">
          <div class="px-5 py-3 border-b bg-red-50 rounded-t-xl">
            <h3 class="font-semibold text-red-700">❌ Não vão ({declined.length})</h3>
          </div>
          <div class="divide-y">
            {#each declined as a}
              <div class="px-5 py-3">
                <p class="font-medium text-sm">{a.player_nickname ?? a.player_name}</p>
              </div>
            {/each}
          </div>
        </div>
      {/if}

      {#if pending.length > 0}
        <div class="card mb-4">
          <div class="px-5 py-3 border-b bg-yellow-50 rounded-t-xl">
            <h3 class="font-semibold text-yellow-700">⏳ Aguardando ({pending.length})</h3>
          </div>
          <div class="divide-y">
            {#each pending as a}
              <div class="px-5 py-3">
                <p class="font-medium text-sm">{a.player_nickname ?? a.player_name}</p>
              </div>
            {/each}
          </div>
        </div>
      {/if}
    {/if}
  </main>
</div>
