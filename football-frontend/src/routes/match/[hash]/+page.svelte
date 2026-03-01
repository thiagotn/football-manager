<script lang="ts">
  import { page } from '$app/stores';
  import { matches as matchesApi, ApiError } from '$lib/api';
  import type { MatchDetail, Attendance } from '$lib/api';
  import { currentPlayer, isLoggedIn } from '$lib/stores/auth';
  import { toastSuccess, toastError } from '$lib/stores/toast';
  import { Clock, MapPin, Calendar, CheckCircle, XCircle, Clock3, Link2, Users } from 'lucide-svelte';

  const matchHash = $page.params.hash;
  const COURT_LABELS: Record<string, string> = { campo: 'Campo', sintetico: 'Sintético', terrao: 'Terrão', quadra: 'Quadra' };

  let match: MatchDetail | null = $state(null);
  let loading = $state(true);
  let responding = $state(false);

  let confirmed = $derived(match?.attendances.filter(a => a.status === 'confirmed') ?? []);
  let declined  = $derived(match?.attendances.filter(a => a.status === 'declined')  ?? []);
  let pending   = $derived(match?.attendances.filter(a => a.status === 'pending')   ?? []);
  let mine      = $derived(match?.attendances.find(a => a.player.id === $currentPlayer?.id));
  let isFull    = $derived(!!match?.max_players && (match?.confirmed_count ?? 0) >= match.max_players && mine?.status !== 'confirmed');

  $effect(() => {
    let cancelled = false;
    (async () => {
      try {
        const m = await matchesApi.getByHash(matchHash);
        if (!cancelled) match = m;
      } catch { if (!cancelled) match = null; }
      if (!cancelled) loading = false;
    })();
    return () => { cancelled = true; };
  });

  function fmtDate(d: string) {
    return new Date(d + 'T00:00').toLocaleDateString('pt-BR', {
      weekday: 'long', day: '2-digit', month: 'long', year: 'numeric'
    });
  }

  async function respond(status: 'confirmed' | 'declined') {
    if (!match || !$currentPlayer) return;
    responding = true;
    try {
      await matchesApi.setAttendance(match.group_id, match.id, $currentPlayer.id, status);
      match = await matchesApi.getByHash(matchHash);
      toastSuccess(status === 'confirmed' ? '✅ Presença confirmada!' : '❌ Falta registrada');
    } catch (e) { toastError(e instanceof ApiError ? e.message : 'Erro'); }
    responding = false;
  }

  function fmtDateShare(d: string) {
    const dt = new Date(d + 'T00:00');
    const weekday = dt.toLocaleDateString('pt-BR', { weekday: 'long' });
    const ddmmyyyy = dt.toLocaleDateString('pt-BR', { day: '2-digit', month: '2-digit', year: 'numeric' });
    const time = match!.start_time.slice(0, 5).replace(':', 'h');
    return `${weekday}, ${time} (${ddmmyyyy})`;
  }

  function shareWhatsApp() {
    if (!match) return;
    const playerList = confirmed.length > 0
      ? confirmed.map((a, i) => `${i + 1} - ${a.player.nickname || a.player.name}`).join('\n')
      : 'Nenhum confirmado ainda';
    const lines = [
      `*${match.group_name}*`,
      fmtDateShare(match.match_date),
      `Local: ${match.location}`,
      '',
      `Confirmados (${confirmed.length}):`,
      playerList,
      '',
      `Nao vao: ${declined.length}`,
      `Pendentes: ${pending.length}`,
      '',
      window.location.href,
    ];
    const text = encodeURIComponent(lines.join('\n'));
    window.open(`https://wa.me/?text=${text}`, '_blank');
  }

  function copyLink() {
    navigator.clipboard.writeText(window.location.href);
    toastSuccess('Link copiado!');
  }
</script>

<svelte:head>
  <title>{match ? `Partida — ${match.location}` : 'Partida'}</title>
</svelte:head>

<div class="min-h-screen bg-gray-50">
  <!-- Header bar -->
  <div class="bg-yellow-400 text-yellow-900 text-xs font-medium text-center py-1.5 px-4">
    Versao Beta — produto em desenvolvimento.
  </div>
  <div class="bg-primary-700 text-white py-4 px-4 flex items-center justify-between">
    <div class="flex items-center gap-2">
      <span class="text-xl">⚽</span>
      <span class="font-semibold">{match?.group_name ?? 'Futebol'}</span>
      <span class="text-xs font-bold bg-yellow-400 text-yellow-900 px-1.5 py-0.5 rounded-full">Beta</span>
    </div>
    <div class="flex items-center gap-2">
      <button onclick={shareWhatsApp} class="btn-ghost btn-sm text-primary-100 hover:text-white hover:bg-primary-600">
        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="currentColor" class="shrink-0">
          <path d="M17.472 14.382c-.297-.149-1.758-.867-2.03-.967-.273-.099-.471-.148-.67.15-.197.297-.767.966-.94 1.164-.173.199-.347.223-.644.075-.297-.15-1.255-.463-2.39-1.475-.883-.788-1.48-1.761-1.653-2.059-.173-.297-.018-.458.13-.606.134-.133.298-.347.446-.52.149-.174.198-.298.298-.497.099-.198.05-.371-.025-.52-.075-.149-.669-1.612-.916-2.207-.242-.579-.487-.5-.669-.51-.173-.008-.371-.01-.57-.01-.198 0-.52.074-.792.372-.272.297-1.04 1.016-1.04 2.479 0 1.462 1.065 2.875 1.213 3.074.149.198 2.096 3.2 5.077 4.487.709.306 1.262.489 1.694.625.712.227 1.36.195 1.871.118.571-.085 1.758-.719 2.006-1.413.248-.694.248-1.289.173-1.413-.074-.124-.272-.198-.57-.347z"/>
          <path d="M12 0C5.373 0 0 5.373 0 12c0 2.126.558 4.121 1.533 5.853L.036 23.964l6.252-1.639A11.945 11.945 0 0 0 12 24c6.627 0 12-5.373 12-12S18.627 0 12 0zm0 21.818a9.8 9.8 0 0 1-4.998-1.366l-.358-.213-3.712.974 1.014-3.598-.233-.371A9.818 9.818 0 1 1 12 21.818z"/>
        </svg>
        WhatsApp
      </button>
      <button onclick={copyLink} class="btn-ghost btn-sm text-primary-100 hover:text-white hover:bg-primary-600">
        <Link2 size={16} /> Copiar link
      </button>
    </div>
  </div>

  <main class="max-w-2xl mx-auto px-4 py-8">
    {#if loading}
      <div class="animate-pulse space-y-4">
        <div class="h-8 bg-gray-200 rounded w-2/3"></div>
        <div class="h-4 bg-gray-100 rounded w-1/2"></div>
      </div>

    {:else if !match}
      <div class="card p-12 text-center">
        <Calendar size={48} class="text-gray-300 mx-auto mb-4" />
        <h2 class="text-xl font-semibold text-gray-700">Partida não encontrada</h2>
        <p class="text-gray-400 mt-2">O link pode estar errado ou a partida foi removida.</p>
      </div>

    {:else}
      <!-- Match card -->
      <div class="card mb-6 overflow-hidden">
        <div class="relative overflow-hidden px-6 py-5 text-white" style="min-height:140px;">
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
          <div class="relative">
          <p class="text-sm font-medium text-primary-200 mb-1">
            <span class="badge {match.status === 'open' ? 'bg-green-400 text-green-900' : 'bg-gray-400 text-gray-900'} mr-2">
              {match.status === 'open' ? 'Aberta' : 'Encerrada'}
            </span>
            {match.group_name}
            <span class="ml-1 opacity-70">· #{match.number}</span>
          </p>
          <h1 class="text-xl font-bold capitalize">{fmtDate(match.match_date)}</h1>
          <div class="flex flex-wrap gap-4 mt-3 text-primary-100 text-sm">
            <span class="flex items-center gap-1.5"><Clock size={14} />{match.start_time.slice(0,5)}</span>
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
                <span class="bg-primary-800/40 rounded px-2 py-0.5">{COURT_LABELS[match.court_type]}</span>
              {/if}
              {#if match.players_per_team}
                <span class="bg-primary-800/40 rounded px-2 py-0.5">{match.players_per_team} na linha + goleiro</span>
              {/if}
              {#if match.max_players}
                <span class="bg-primary-800/40 rounded px-2 py-0.5 {match.confirmed_count >= match.max_players ? 'text-red-300 font-semibold' : ''}">
                  {match.confirmed_count}/{match.max_players} vagas
                </span>
              {/if}
            </div>
          {/if}
          {#if match.notes}
            <p class="text-sm text-primary-200 mt-3 bg-primary-800/30 rounded-lg px-3 py-2">{match.notes}</p>
          {/if}
          </div><!-- /relative content -->
        </div><!-- /banner header -->

        <!-- Scoreboard summary -->
        <div class="grid grid-cols-3 divide-x divide-gray-100">
          <div class="px-4 py-4 text-center">
            <p class="text-2xl font-bold text-green-600">
              {match.confirmed_count}{#if match.max_players}<span class="text-base text-gray-400">/{match.max_players}</span>{/if}
            </p>
            <p class="text-xs text-gray-500 mt-0.5 flex items-center justify-center gap-1">
              <CheckCircle size={11} />
              {match.max_players && match.confirmed_count >= match.max_players ? 'Lotada!' : 'Confirmados'}
            </p>
          </div>
          <div class="px-4 py-4 text-center">
            <p class="text-2xl font-bold text-red-500">{match.declined_count}</p>
            <p class="text-xs text-gray-500 mt-0.5 flex items-center justify-center gap-1">
              <XCircle size={11} /> Recusaram
            </p>
          </div>
          <div class="px-4 py-4 text-center">
            <p class="text-2xl font-bold text-gray-400">{match.pending_count}</p>
            <p class="text-xs text-gray-500 mt-0.5 flex items-center justify-center gap-1">
              <Clock3 size={11} /> Pendentes
            </p>
          </div>
        </div>
      </div>

      <!-- My RSVP (only if logged in and in the match) -->
      {#if $isLoggedIn && match.status === 'open'}
        <div class="card mb-6 card-body">
          <h3 class="font-semibold text-gray-800 mb-3 flex items-center gap-2">
            <Users size={16} class="text-primary-600" /> Sua Confirmação
          </h3>
          {#if mine}
            <p class="text-sm text-gray-500 mb-3">
              Status atual:
              <span class="font-medium {mine.status === 'confirmed' ? 'text-green-600' : mine.status === 'declined' ? 'text-red-500' : 'text-gray-500'}">
                {mine.status === 'confirmed' ? '✅ Confirmado' : mine.status === 'declined' ? '❌ Recusado' : '⏳ Pendente'}
              </span>
            </p>
          {/if}
          {#if isFull}
            <p class="text-sm text-red-500 font-medium text-center py-2">
              ⛔ Partida lotada — {match.max_players} jogadores já confirmados.
            </p>
          {/if}
          <div class="flex gap-3">
            <button
              class="flex-1 btn {mine?.status === 'confirmed' ? 'btn-primary' : 'btn-secondary'}"
              onclick={() => respond('confirmed')} disabled={responding || isFull}>
              <CheckCircle size={16} /> Vou jogar!
            </button>
            <button
              class="flex-1 btn {mine?.status === 'declined' ? 'btn-danger' : 'btn-secondary'}"
              onclick={() => respond('declined')} disabled={responding}>
              <XCircle size={16} /> Não posso
            </button>
          </div>
        </div>
      {/if}

      <!-- Players lists -->
      <div class="space-y-4">
        <!-- Confirmed -->
        {#if confirmed.length > 0}
          <div class="card overflow-hidden">
            <div class="card-header bg-green-50">
              <h3 class="font-semibold text-green-800 flex items-center gap-2">
                <CheckCircle size={16} /> Confirmados ({confirmed.length})
              </h3>
            </div>
            <ul class="divide-y divide-gray-100">
              {#each confirmed as a, i}
                <li class="px-5 py-3 flex items-center gap-3">
                  <span class="w-6 h-6 rounded-full bg-green-100 text-green-700 text-xs flex items-center justify-center font-bold shrink-0">{i+1}</span>
                  <div>
                    <p class="text-sm font-medium text-gray-900">{a.player.nickname || a.player.name}</p>
                    {#if a.player.nickname}<p class="text-xs text-gray-400">{a.player.name}</p>{/if}
                  </div>
                </li>
              {/each}
            </ul>
          </div>
        {/if}

        <!-- Declined -->
        {#if declined.length > 0}
          <div class="card overflow-hidden">
            <div class="card-header bg-red-50">
              <h3 class="font-semibold text-red-700 flex items-center gap-2">
                <XCircle size={16} /> Recusaram ({declined.length})
              </h3>
            </div>
            <ul class="divide-y divide-gray-100">
              {#each declined as a}
                <li class="px-5 py-3 text-sm text-gray-600 flex items-center gap-3">
                  <XCircle size={14} class="text-red-400 shrink-0" />
                  {a.player.nickname || a.player.name}{a.player.nickname ? ` (${a.player.name})` : ''}
                </li>
              {/each}
            </ul>
          </div>
        {/if}

        <!-- Pending -->
        {#if pending.length > 0}
          <div class="card overflow-hidden">
            <div class="card-header">
              <h3 class="font-semibold text-gray-600 flex items-center gap-2">
                <Clock3 size={16} /> Aguardando ({pending.length})
              </h3>
            </div>
            <ul class="divide-y divide-gray-100">
              {#each pending as a}
                <li class="px-5 py-3 text-sm text-gray-500 flex items-center gap-3">
                  <Clock3 size={14} class="text-gray-400 shrink-0" />
                  {a.player.nickname || a.player.name}{a.player.nickname ? ` (${a.player.name})` : ''}
                </li>
              {/each}
            </ul>
          </div>
        {/if}
      </div>
    {/if}
  </main>
</div>
