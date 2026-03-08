<script lang="ts">
  import { page } from '$app/stores';
  import { matches as matchesApi, groups as groupsApi, ApiError } from '$lib/api';
  import type { MatchDetail, Attendance } from '$lib/api';
  import { currentPlayer, isLoggedIn, isAdmin } from '$lib/stores/auth';

  let { data } = $props();
  import { toastSuccess, toastError } from '$lib/stores/toast';
  import { Clock, MapPin, Calendar, CheckCircle, XCircle, Clock3, Link2, Users, Lock, LockOpen, X } from 'lucide-svelte';
  import PageBackground from '$lib/components/PageBackground.svelte';
  import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
  import { relativeDate } from '$lib/utils.js';

  const matchHash = $page.params.hash;
  const COURT_LABELS: Record<string, string> = { campo: 'Campo', sintetico: 'Sintético', terrao: 'Terrão', quadra: 'Quadra' };

  let match: MatchDetail | null = $state(null);
  let loading = $state(true);
  let responding = $state(false);
  let responded = $state(false);
  let lastStatus: 'confirmed' | 'declined' | null = $state(null);
  let isGroupAdmin = $state(false);
  let adminResponding = $state<string | null>(null);
  let togglingStatus = $state(false);
  let confirmOpen = $state(false);
  let confirmMessage = $state('');
  let confirmAction = $state<() => void>(() => {});
  let showRsvpBanner = $state(true);

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

  $effect(() => {
    const player = $currentPlayer;
    const m = match;
    if (!player || !m) { isGroupAdmin = false; return; }
    if (player.role === 'admin') { isGroupAdmin = true; return; }
    (async () => {
      try {
        const group = await groupsApi.get(m.group_id);
        const member = group.members.find(mb => mb.player.id === player.id);
        isGroupAdmin = member?.role === 'admin';
      } catch { isGroupAdmin = false; }
    })();
  });

  function fmtTimeRange(start: string, end: string | null): string {
    const s = start.slice(0, 5);
    if (!end) return s;
    const e = end.slice(0, 5);
    const [sh, sm] = s.split(':').map(Number);
    const [eh, em] = e.split(':').map(Number);
    const mins = (eh * 60 + em) - (sh * 60 + sm);
    if (mins <= 0) return `${s} – ${e}`;
    const h = Math.floor(mins / 60), m = mins % 60;
    const dur = h && m ? `${h}h${String(m).padStart(2, '0')}` : h ? `${h}h` : `${m}min`;
    return `${s} – ${e} (${dur})`;
  }

  function fmtDate(d: string) {
    return relativeDate(d, { weekday: 'long', day: '2-digit', month: 'long', year: 'numeric' });
  }

  function fmtPricingParts(perMatch: number | string | null, monthly: number | string | null): string[] {
    const parts: string[] = [];
    if (perMatch != null) parts.push(`R$ ${Number(perMatch).toFixed(2).replace('.', ',')} avulso`);
    if (monthly != null) parts.push(`R$ ${Number(monthly).toFixed(2).replace('.', ',')} mensal`);
    return parts;
  }

  async function respond(status: 'confirmed' | 'declined') {
    if (!match || !$currentPlayer) return;
    responding = true;
    try {
      await matchesApi.setAttendance(match.group_id, match.id, $currentPlayer.id, status);
      match = await matchesApi.getByHash(matchHash);
      toastSuccess(status === 'confirmed' ? '✅ Presença confirmada!' : '❌ Falta registrada');
      lastStatus = status;
      responded = true;
    } catch (e) { toastError(e instanceof ApiError ? e.message : 'Erro'); }
    responding = false;
  }

  async function respondFor(playerId: string, status: 'confirmed' | 'declined') {
    if (!match) return;
    adminResponding = playerId;
    try {
      await matchesApi.setAttendance(match.group_id, match.id, playerId, status);
      match = await matchesApi.getByHash(matchHash);
      toastSuccess(status === 'confirmed' ? '✅ Presença confirmada!' : '❌ Falta registrada');
    } catch (e) { toastError(e instanceof ApiError ? e.message : 'Erro'); }
    adminResponding = null;
  }

  function fmtDateShare(d: string) {
    const rel = relativeDate(d);
    const start = match!.start_time.slice(0, 5).replace(':', 'h');
    const end = match!.end_time ? ` – ${match!.end_time.slice(0, 5).replace(':', 'h')}` : '';
    const time = `${start}${end}`;
    const isRelativeWord = rel === 'Hoje' || rel === 'Amanhã' || rel === 'Ontem';
    if (isRelativeWord) return `${rel}, ${time}`;
    const dt = new Date(d + 'T00:00');
    const ddmmyyyy = dt.toLocaleDateString('pt-BR', { day: '2-digit', month: '2-digit', year: 'numeric' });
    return `${rel}, ${time} (${ddmmyyyy})`;
  }

  function shareWhatsApp() {
    if (!match) return;
    const confirmedList = match.confirmed_count > 0
      ? confirmed.map((a, i) => `${i + 1} - ${a.player.nickname || a.player.name}`).join('\n')
      : 'Nenhum confirmado ainda';
    const declinedList = match.declined_count > 0
      ? declined.map(a => `- ${a.player.nickname || a.player.name}`).join('\n')
      : 'Nenhum';
    const pendingList = match.pending_count > 0
      ? pending.map(a => `- ${a.player.nickname || a.player.name}`).join('\n')
      : 'Nenhum';
    const confirmedHeader = match.max_players
      ? `Confirmados (${match.confirmed_count}/${match.max_players}):`
      : `Confirmados (${match.confirmed_count}):`;
    const lines = [
      `*Rachão ${match.group_name}*`,
      fmtDateShare(match.match_date),
      `Local: ${match.location}`,
      '',
      confirmedHeader,
      confirmedList,
      '',
      `Não vão (${match.declined_count}):`,
      declinedList,
      '',
      `Pendentes (${match.pending_count}):`,
      pendingList,
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

  async function toggleStatus(newStatus: 'open' | 'closed') {
    if (!match) return;
    togglingStatus = true;
    try {
      await matchesApi.update(match.group_id, match.id, { status: newStatus });
      match = await matchesApi.getByHash(matchHash);
      toastSuccess(newStatus === 'open' ? 'Partida reaberta!' : 'Partida encerrada.');
    } catch (e) { toastError(e instanceof ApiError ? e.message : 'Erro ao atualizar status'); }
    togglingStatus = false;
  }

  function askCloseMatch() {
    confirmMessage = 'Encerrar esta partida? Os jogadores não poderão mais confirmar presença.';
    confirmAction = () => toggleStatus('closed');
    confirmOpen = true;
  }
</script>

<svelte:head>
  {#if data.og}
    <title>{data.og.title} — rachao.app</title>
    <meta property="og:title" content="{data.og.title} — rachao.app" />
    <meta property="og:description" content={data.og.description} />
    <meta property="og:image" content="https://rachao.app/banner-lp.jpg" />
    <meta property="og:url" content="https://rachao.app/match/{$page.params.hash}" />
  {:else}
    <title>{match ? `Partida — ${match.location}` : 'Partida'} — rachao.app</title>
  {/if}
</svelte:head>

<PageBackground>
  <main class="relative z-10 max-w-2xl mx-auto px-4 pt-4 pb-8">
    {#if loading}
      <div class="animate-pulse space-y-4">
        <div class="h-8 bg-gray-200 dark:bg-gray-700 rounded w-2/3"></div>
        <div class="h-4 bg-gray-100 dark:bg-gray-700 rounded w-1/2"></div>
      </div>

    {:else if !match}
      <div class="card p-12 text-center">
        <Calendar size={48} class="text-gray-300 mx-auto mb-4" />
        <h2 class="text-xl font-semibold text-gray-700 dark:text-gray-300">Rachão não encontrado</h2>
        <p class="text-gray-400 dark:text-gray-500 mt-2">O link pode estar errado ou o rachão foi removido.</p>
      </div>

    {:else}
      <!-- Match card -->
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
          <div class="relative">
          <div class="flex items-center justify-between mb-1">
            <p class="text-sm font-bold text-white">
              #{match.number} {match.group_name}
            </p>
            {#if match.status === 'in_progress'}
              <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-semibold bg-red-500/30 text-red-200 border border-red-400/40">
                <span class="w-1.5 h-1.5 rounded-full bg-red-400 animate-pulse"></span>
                Bola rolando
              </span>
            {:else}
              <span class="badge {match.status === 'open' ? 'bg-green-400 text-green-900' : 'bg-gray-400 text-gray-900'}">
                {match.status === 'open' ? 'Aberta' : 'Encerrada'}
              </span>
            {/if}
          </div>
          <h1 class="text-xl font-bold capitalize">{fmtDate(match.match_date)}</h1>
          <div class="flex flex-wrap gap-3 mt-2 text-primary-100 text-sm">
            <span class="flex items-center gap-1.5"><Clock size={14} />{fmtTimeRange(match.start_time, match.end_time)}</span>
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
          {#if match.court_type || match.players_per_team || match.max_players || match.group_per_match_amount != null || match.group_monthly_amount != null}
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
              {#each fmtPricingParts(match.group_per_match_amount, match.group_monthly_amount) as part}
                <span class="bg-amber-500/30 text-amber-200 rounded px-2 py-0.5 font-medium">{part}</span>
              {/each}
            </div>
          {/if}
          {#if match.notes}
            <p class="text-sm text-primary-200 mt-2 bg-primary-800/30 rounded-lg px-3 py-1.5">{match.notes}</p>
          {/if}
          </div><!-- /relative content -->
        </div><!-- /banner header -->

        <!-- Scoreboard summary -->
        <div class="grid grid-cols-3 divide-x divide-gray-100 dark:divide-gray-700">
          <div class="px-3 py-3 text-center">
            <p class="text-xl font-bold text-green-600">
              {match.confirmed_count}{#if match.max_players}<span class="text-sm text-gray-400">/{match.max_players}</span>{/if}
            </p>
            <p class="text-xs text-gray-500 dark:text-gray-400 mt-0.5 flex items-center justify-center gap-1">
              <CheckCircle size={11} />
              {match.max_players && match.confirmed_count >= match.max_players ? 'Lotada!' : 'Confirmados'}
            </p>
          </div>
          <div class="px-3 py-3 text-center">
            <p class="text-xl font-bold text-red-500">{match.declined_count}</p>
            <p class="text-xs text-gray-500 dark:text-gray-400 mt-0.5 flex items-center justify-center gap-1">
              <XCircle size={11} /> Recusaram
            </p>
          </div>
          <div class="px-3 py-3 text-center">
            <p class="text-xl font-bold text-gray-400">{match.pending_count}</p>
            <p class="text-xs text-gray-500 dark:text-gray-400 mt-0.5 flex items-center justify-center gap-1">
              <Clock3 size={11} /> Pendentes
            </p>
          </div>
        </div>
      </div>

      <!-- Players lists -->
      <div class="relative">
        <!-- RSVP overlay -->
        {#if showRsvpBanner && $isLoggedIn && !$isAdmin && !responded && mine?.status === 'pending' && (match.status === 'open' || match.status === 'in_progress')}
          <div class="absolute inset-0 z-10 bg-gray-900/75 rounded-xl flex items-start justify-center pt-6 pb-4">
            <div class="bg-white dark:bg-gray-800 rounded-2xl shadow-xl mx-4 p-5 w-full max-w-sm">
              <div class="flex items-center justify-between mb-4">
                <h3 class="font-semibold text-gray-800 dark:text-gray-100 flex items-center gap-2">
                  <Users size={16} class="text-primary-600" /> Confirme sua presença
                </h3>
                <button
                  onclick={() => showRsvpBanner = false}
                  class="p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-200 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700"
                  aria-label="Fechar">
                  <X size={16} />
                </button>
              </div>
              {#if isFull}
                <p class="text-sm text-red-500 font-medium text-center py-2">⛔ Partida lotada — {match.max_players} jogadores já confirmados.</p>
              {/if}
              <div class="flex gap-3">
                <button class="flex-1 btn btn-primary justify-center" onclick={() => { respond('confirmed'); showRsvpBanner = false; }} disabled={responding || isFull}>
                  <CheckCircle size={16} /> Vou jogar!
                </button>
                <button class="flex-1 btn btn-danger justify-center" onclick={() => { respond('declined'); showRsvpBanner = false; }} disabled={responding}>
                  <XCircle size={16} /> Não posso
                </button>
              </div>
            </div>
          </div>
        {/if}

      <div class="space-y-3">
        <!-- Confirmed -->
        {#if confirmed.length > 0}
          <div class="card overflow-hidden">
            <div class="px-4 py-2 bg-green-50 dark:bg-green-900/20 border-b border-gray-100 dark:border-gray-700">
              <h3 class="text-sm font-semibold text-green-800 dark:text-green-300 flex items-center gap-1.5">
                <CheckCircle size={14} /> Confirmados ({confirmed.length})
              </h3>
            </div>
            <ul class="divide-y divide-gray-100 dark:divide-gray-700">
              {#each confirmed as a, i}
                <li class="px-4 py-2 flex items-center gap-2.5">
                  <span class="w-5 h-5 rounded-full bg-green-100 text-green-700 text-xs flex items-center justify-center font-bold shrink-0">{i+1}</span>
                  <p class="text-sm font-medium text-gray-900 dark:text-gray-100 flex-1">{a.player.nickname || a.player.name}</p>
                  {#if !$isAdmin && a.player.id === $currentPlayer?.id && (match.status === 'open' || match.status === 'in_progress')}
                    <button
                      class="text-xs px-2 py-0.5 rounded border border-red-200 text-red-500 hover:bg-red-50 dark:border-red-800 dark:text-red-400 dark:hover:bg-red-900/20 disabled:opacity-40 flex items-center gap-1 shrink-0"
                      onclick={() => respond('declined')}
                      disabled={responding}>
                      <XCircle size={11} /> Recusar
                    </button>
                  {:else if isGroupAdmin && (match.status === 'open' || match.status === 'in_progress') && a.player.id !== $currentPlayer?.id}
                    <button
                      class="text-xs px-2 py-0.5 rounded border border-red-200 text-red-500 hover:bg-red-50 disabled:opacity-40 shrink-0"
                      onclick={() => respondFor(a.player.id, 'declined')}
                      disabled={adminResponding === a.player.id}>
                      ✕ Recusar
                    </button>
                  {/if}
                </li>
              {/each}
            </ul>
          </div>
        {/if}

        <!-- Declined -->
        {#if declined.length > 0}
          <div class="card overflow-hidden">
            <div class="px-4 py-2 bg-red-50 dark:bg-red-900/20 border-b border-gray-100 dark:border-gray-700">
              <h3 class="text-sm font-semibold text-red-700 dark:text-red-400 flex items-center gap-1.5">
                <XCircle size={14} /> Recusaram ({declined.length})
              </h3>
            </div>
            <ul class="divide-y divide-gray-100 dark:divide-gray-700">
              {#each declined as a}
                <li class="px-4 py-2 text-sm text-gray-600 dark:text-gray-400 flex items-center gap-2.5">
                  <XCircle size={13} class="text-red-400 shrink-0" />
                  <span class="flex-1">{a.player.nickname || a.player.name}</span>
                  {#if !$isAdmin && a.player.id === $currentPlayer?.id && (match.status === 'open' || match.status === 'in_progress')}
                    <button
                      class="text-xs px-2 py-0.5 rounded border border-green-200 text-green-600 hover:bg-green-50 dark:border-green-800 dark:text-green-400 dark:hover:bg-green-900/20 disabled:opacity-40 flex items-center gap-1 shrink-0"
                      onclick={() => respond('confirmed')}
                      disabled={responding || isFull}>
                      <CheckCircle size={11} /> Confirmar
                    </button>
                  {:else if isGroupAdmin && (match.status === 'open' || match.status === 'in_progress') && a.player.id !== $currentPlayer?.id}
                    <button
                      class="text-xs px-2 py-0.5 rounded border border-green-200 text-green-600 hover:bg-green-50 disabled:opacity-40 shrink-0"
                      onclick={() => respondFor(a.player.id, 'confirmed')}
                      disabled={adminResponding === a.player.id}>
                      ✓ Confirmar
                    </button>
                  {/if}
                </li>
              {/each}
            </ul>
          </div>
        {/if}

        <!-- Pending -->
        {#if pending.length > 0}
          <div class="card overflow-hidden">
            <div class="px-4 py-2 border-b border-gray-100 dark:border-gray-700">
              <h3 class="text-sm font-semibold text-gray-600 dark:text-gray-400 flex items-center gap-1.5">
                <Clock3 size={14} /> Aguardando ({pending.length})
              </h3>
            </div>
            <ul class="divide-y divide-gray-100 dark:divide-gray-700">
              {#each pending as a}
                <li class="px-4 py-2 text-sm text-gray-500 dark:text-gray-400 flex items-center gap-2.5">
                  <Clock3 size={13} class="text-gray-400 shrink-0" />
                  <span class="flex-1">{a.player.nickname || a.player.name}</span>
                  {#if !showRsvpBanner && a.player.id === $currentPlayer?.id && !$isAdmin && (match.status === 'open' || match.status === 'in_progress')}
                    <div class="flex gap-1 shrink-0">
                      <button
                        class="text-xs px-2 py-0.5 rounded border border-green-200 text-green-600 hover:bg-green-50 dark:border-green-800 dark:text-green-400 dark:hover:bg-green-900/20 disabled:opacity-40 flex items-center gap-1"
                        onclick={() => respond('confirmed')}
                        disabled={responding || isFull}>
                        <CheckCircle size={11} /> Confirmar
                      </button>
                      <button
                        class="text-xs px-2 py-0.5 rounded border border-red-200 text-red-500 hover:bg-red-50 dark:border-red-800 dark:text-red-400 dark:hover:bg-red-900/20 disabled:opacity-40 flex items-center gap-1"
                        onclick={() => respond('declined')}
                        disabled={responding}>
                        <XCircle size={11} /> Recusar
                      </button>
                    </div>
                  {:else if isGroupAdmin && (match.status === 'open' || match.status === 'in_progress') && a.player.id !== $currentPlayer?.id}
                    <div class="flex gap-1 shrink-0">
                      <button
                        class="text-xs px-2 py-0.5 rounded border border-green-200 text-green-600 hover:bg-green-50 disabled:opacity-40"
                        onclick={() => respondFor(a.player.id, 'confirmed')}
                        disabled={adminResponding === a.player.id}>
                        ✓ Confirmar
                      </button>
                      <button
                        class="text-xs px-2 py-0.5 rounded border border-red-200 text-red-500 hover:bg-red-50 disabled:opacity-40"
                        onclick={() => respondFor(a.player.id, 'declined')}
                        disabled={adminResponding === a.player.id}>
                        ✕ Recusar
                      </button>
                    </div>
                  {/if}
                </li>
              {/each}
            </ul>
          </div>
        {/if}
      </div>
      </div><!-- /relative RSVP wrapper -->

      <!-- Admin status toggle -->
      {#if isGroupAdmin}
        <div class="card card-body mt-3 flex flex-wrap gap-2 justify-end items-center">
          <span class="text-xs text-gray-400 dark:text-gray-500 mr-auto">Ações do administrador</span>
          {#if match.status === 'closed'}
            <button
              onclick={() => toggleStatus('open')}
              disabled={togglingStatus}
              class="btn-sm btn-secondary gap-1">
              <LockOpen size={14} /> Reabrir partida
            </button>
          {:else}
            <button
              onclick={askCloseMatch}
              disabled={togglingStatus}
              class="btn-sm btn-ghost text-red-500 hover:bg-red-50 gap-1">
              <Lock size={14} /> Encerrar partida
            </button>
          {/if}
        </div>
      {/if}

      <!-- Share footer -->
      <div class="mt-6 pt-5 border-t border-gray-200 dark:border-gray-700 flex flex-col sm:flex-row gap-3">
        <button onclick={shareWhatsApp} class="flex-1 btn btn-secondary justify-center gap-2">
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="currentColor" class="shrink-0">
            <path d="M17.472 14.382c-.297-.149-1.758-.867-2.03-.967-.273-.099-.471-.148-.67.15-.197.297-.767.966-.94 1.164-.173.199-.347.223-.644.075-.297-.15-1.255-.463-2.39-1.475-.883-.788-1.48-1.761-1.653-2.059-.173-.297-.018-.458.13-.606.134-.133.298-.347.446-.52.149-.174.198-.298.298-.497.099-.198.05-.371-.025-.52-.075-.149-.669-1.612-.916-2.207-.242-.579-.487-.5-.669-.51-.173-.008-.371-.01-.57-.01-.198 0-.52.074-.792.372-.272.297-1.04 1.016-1.04 2.479 0 1.462 1.065 2.875 1.213 3.074.149.198 2.096 3.2 5.077 4.487.709.306 1.262.489 1.694.625.712.227 1.36.195 1.871.118.571-.085 1.758-.719 2.006-1.413.248-.694.248-1.289.173-1.413-.074-.124-.272-.198-.57-.347z"/>
            <path d="M12 0C5.373 0 0 5.373 0 12c0 2.126.558 4.121 1.533 5.853L.036 23.964l6.252-1.639A11.945 11.945 0 0 0 12 24c6.627 0 12-5.373 12-12S18.627 0 12 0zm0 21.818a9.8 9.8 0 0 1-4.998-1.366l-.358-.213-3.712.974 1.014-3.598-.233-.371A9.818 9.818 0 1 1 12 21.818z"/>
          </svg>
          Compartilhar no WhatsApp
        </button>
        <button onclick={copyLink} class="flex-1 btn btn-secondary justify-center gap-2">
          <Link2 size={16} /> Copiar link
        </button>
      </div>
      <div class="mt-4 rounded-xl bg-gray-900/75 py-4 px-4 flex items-center justify-between">
        {#if $isLoggedIn}
          <button onclick={() => history.back()} class="text-base font-semibold text-white/90 hover:text-white px-2 py-1 rounded-lg hover:bg-white/10 transition-colors">
            ← Voltar
          </button>
        {:else}
          <a href="/" class="text-base font-semibold text-primary-300 hover:text-primary-200 px-2 py-1 rounded-lg hover:bg-white/10 transition-colors">
            Conheça a plataforma →
          </a>
        {/if}
        <a href="https://rachao.app" target="_blank" rel="noopener noreferrer" class="text-sm font-semibold text-white/70 hover:text-white transition-colors">rachao.app</a>
        <span class="text-xs text-white/50">© 2026</span>
      </div>
    {/if}
  </main>
</PageBackground>

<ConfirmDialog
  bind:open={confirmOpen}
  message={confirmMessage}
  confirmLabel="Encerrar"
  danger={true}
  onConfirm={confirmAction}
/>
