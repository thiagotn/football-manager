<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import { matches as matchesApi, votes as votesApi, teams as teamsApi, groups as groupsApi, ApiError } from '$lib/api';
  import type { MatchDetail, Attendance, VoteStatusResponse, VoteResultsResponse, TeamsResponse, WaitlistEntry } from '$lib/api';
  import { currentPlayer, isLoggedIn, isAdmin } from '$lib/stores/auth';

  let { data } = $props();
  import { toastSuccess, toastError } from '$lib/stores/toast';
  import { Clock, MapPin, Calendar, CheckCircle, XCircle, Clock3, Link2, Users, Lock, LockOpen, X, Shuffle, ExternalLink, UserPlus } from 'lucide-svelte';
  import PageBackground from '$lib/components/PageBackground.svelte';
  import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
  import VoteForm from '$lib/components/VoteForm.svelte';
  import VoteResults from '$lib/components/VoteResults.svelte';
  import WaitlistModal from '$lib/components/WaitlistModal.svelte';
  import { relativeDate } from '$lib/utils.js';
  import { goto } from '$app/navigation';

  const matchHash = $page.params.hash;
  const COURT_LABELS: Record<string, string> = { campo: 'Campo', sintetico: 'Sintético', terrao: 'Terrão', quadra: 'Quadra' };

  let match: MatchDetail | null = $state(null);
  let loading = $state(true);
  let responding = $state(false);
  let responded = $state(false);
  let lastStatus: 'confirmed' | 'declined' | null = $state(null);
  let isGroupAdmin = $state(false);
  let groupMembers = $state<{ player: { id: string; name: string; nickname: string | null }; role: string }[]>([]);
  let adminResponding = $state<string | null>(null);
  let togglingStatus = $state(false);
  let confirmOpen = $state(false);

  // Teams
  let teamsData = $state<TeamsResponse | null>(null);
  let teamsLoading = $state(false);
  let generatingTeams = $state(false);
  let confirmTeamsOpen = $state(false);

  // Voting
  let voteStatus = $state<VoteStatusResponse | null>(null);
  let voteResults = $state<VoteResultsResponse | null>(null);
  let voteSaving = $state(false);
  let voteSubmitted = $state(false);
  let showVoteModal = $state(true);
  let showResultsPromo = $state(false);
  let closingVote = $state(false);

  $effect(() => {
    const m = match;
    if (!m || !$isLoggedIn) return;
    if (m.status !== 'closed') return;
    votesApi.getStatus(m.id)
      .then(s => {
        voteStatus = s;
        if (s.status === 'closed') {
          if ($isAdmin) return;
          votesApi.getResults(m.id).then(r => {
            voteResults = r;
            if (r.total_voters > 0) {
              showResultsPromo = true;
              showVoteModal = false;
            }
          }).catch(() => {});
        }
      })
      .catch(() => {});
  });

  async function closeVotingEarly() {
    if (!match) return;
    closingVote = true;
    try {
      await votesApi.closeEarly(match.id);
      const s = await votesApi.getStatus(match.id);
      voteStatus = s;
    } catch (e) {
      toastError(e instanceof ApiError ? e.message : 'Erro ao encerrar votação');
    } finally {
      closingVote = false;
    }
  }

  async function submitVote(top5: { player_id: string; position: number }[], flop_player_id: string | null) {
    if (!match) return;
    voteSaving = true;
    try {
      await votesApi.submit(match.id, top5, flop_player_id);
      voteSubmitted = true;
      const s = await votesApi.getStatus(match.id);
      voteStatus = s;
    } catch (e) {
      toastError(e instanceof ApiError ? e.message : 'Erro ao enviar voto');
    } finally {
      voteSaving = false;
    }
  }

  // Jogadores elegíveis para votação (confirmados, exceto o próprio)
  let voteEligible = $derived(
    (match?.attendances ?? []).filter(
      a => a.status === 'confirmed' && a.player.id !== $currentPlayer?.id
    )
  );
  let confirmMessage = $state('');
  let confirmAction = $state<() => void>(() => {});
  let showRsvpBanner = $state(true);

  let confirmed = $derived(match?.attendances.filter(a => a.status === 'confirmed') ?? []);
  let declined  = $derived(match?.attendances.filter(a => a.status === 'declined')  ?? []);
  let pending   = $derived(match?.attendances.filter(a => a.status === 'pending')   ?? []);
  let absentMembers = $derived(
    groupMembers.filter(mb => !match?.attendances.some(a => a.player.id === mb.player.id))
  );
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

  // Polling: atualiza dados a cada 60s enquanto a partida não estiver encerrada.
  // Guards: tab visível + status ativo (open/in_progress). Para ao ficar closed.
  onMount(() => {
    function refresh() {
      if (document.visibilityState !== 'visible') return;
      if (!match || match.status === 'closed') return;
      matchesApi.getByHash(matchHash).then(m => { match = m; }).catch(() => {});
    }
    const timer = setInterval(refresh, 60_000);
    document.addEventListener('visibilitychange', refresh);
    return () => {
      clearInterval(timer);
      document.removeEventListener('visibilitychange', refresh);
    };
  });

  // Carrega times existentes quando a partida é carregada
  $effect(() => {
    const m = match;
    if (!m) return;
    teamsLoading = true;
    teamsApi.get(m.id)
      .then(t => { teamsData = t.teams.length > 0 ? t : null; })
      .catch(() => { teamsData = null; })
      .finally(() => { teamsLoading = false; });
  });

  async function generateTeams() {
    if (!match) return;
    generatingTeams = true;
    try {
      const result = await teamsApi.generate(match.id);
      teamsData = result;
      toastSuccess('Times sorteados!');
    } catch (e) {
      toastError(e instanceof ApiError ? e.message : 'Erro ao montar times');
    } finally {
      generatingTeams = false;
    }
  }

  $effect(() => {
    const player = $currentPlayer;
    const m = match;
    if (!player || !m) { isGroupAdmin = false; return; }
    (async () => {
      try {
        const group = await groupsApi.get(m.group_id);
        if (player.role === 'admin') {
          isGroupAdmin = true;
        } else {
          const member = group.members.find(mb => mb.player.id === player.id);
          isGroupAdmin = member?.role === 'admin';
        }
        if (isGroupAdmin) groupMembers = group.members;
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

  function goBack() {
    if (history.length > 1) {
      history.back();
    } else {
      goto('/');
    }
  }

  // ── Waitlist ──────────────────────────────────────────────────────────────────
  let showWaitlistModal = $state(false);
  let waitlistEntry = $state<WaitlistEntry | null>(null);
  let submittingWaitlist = $state(false);

  // Check if current user is already in waitlist for this match
  $effect(() => {
    const m = match;
    const player = $currentPlayer;
    if (!m || !player || $isAdmin) return;
    if (m.status === 'closed') return;
    // Only check if user is not already in attendances
    const alreadyInAttendances = m.attendances.some(a => a.player.id === player.id);
    if (alreadyInAttendances) return;
    groupsApi.getMyWaitlistEntry(m.group_id)
      .then(e => { waitlistEntry = e; })
      .catch(() => {});
  });

  // Auto-open waitlist modal when join_waitlist=1 is in URL (post-register/login)
  $effect(() => {
    const m = match;
    if (!m || !$isLoggedIn || $isAdmin) return;
    const joinWaitlist = $page.url.searchParams.get('join_waitlist');
    if (joinWaitlist !== '1') return;
    if (m.status !== 'open') return;
    // Check if not already a member (not in attendances)
    const alreadyMember = m.attendances.some(a => a.player.id === $currentPlayer?.id);
    if (alreadyMember) return;
    // Remove the param from URL without navigation
    const url = new URL(window.location.href);
    url.searchParams.delete('join_waitlist');
    history.replaceState({}, '', url.toString());
    // Open modal
    showWaitlistModal = true;
  });

  async function submitWaitlist(data: { agreed: boolean; intro: string }) {
    if (!match) return;
    submittingWaitlist = true;
    try {
      const entry = await groupsApi.joinWaitlist(match.group_id, { agreed: data.agreed, intro: data.intro || undefined });
      waitlistEntry = entry;
      showWaitlistModal = false;
      toastSuccess('Candidatura enviada! Você será notificado quando um admin revisar.');
    } catch (e) {
      toastError(e instanceof ApiError ? e.message : 'Erro ao enviar candidatura');
    } finally {
      submittingWaitlist = false;
    }
  }

  // Derived: show waitlist button for non-member logged users on public groups
  let canJoinWaitlist = $derived(
    !!match &&
    !!$isLoggedIn &&
    !$isAdmin &&
    match.group_is_public &&
    (match.status === 'open' || match.status === 'in_progress') &&
    !match.attendances.some(a => a.player.id === $currentPlayer?.id) &&
    waitlistEntry === null
  );
  let isWaitlistFull = $derived(
    !!match?.max_players && (match?.confirmed_count ?? 0) >= match!.max_players
  );

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
    {#if $isLoggedIn}
      <button
        onclick={goBack}
        class="mb-3 flex items-center gap-1 text-sm font-medium text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 transition-colors">
        ← Voltar
      </button>
    {/if}

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
            <div class="flex items-center gap-1.5">
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
              {#if isGroupAdmin}
                {#if match.status === 'closed'}
                  <button
                    onclick={() => toggleStatus('open')}
                    disabled={togglingStatus}
                    class="p-1 rounded text-white/60 hover:text-white hover:bg-white/10 transition-colors"
                    title="Reabrir partida">
                    <LockOpen size={14} />
                  </button>
                {:else}
                  <button
                    onclick={askCloseMatch}
                    disabled={togglingStatus}
                    class="p-1 rounded text-white/60 hover:text-white hover:bg-white/10 transition-colors"
                    title="Encerrar partida">
                    <Lock size={14} />
                  </button>
                {/if}
              {/if}
            </div>
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

      <!-- CTA for non-logged users on public groups with open spots -->
      {#if !$isLoggedIn && match.group_is_public && match.status === 'open' && (!match.max_players || match.confirmed_count < match.max_players)}
        <div class="card mb-4 p-5 border border-primary-200 dark:border-primary-800 bg-primary-50 dark:bg-primary-900/20">
          <p class="text-sm font-semibold text-primary-800 dark:text-primary-200 mb-1">Quer jogar?</p>
          <p class="text-xs text-primary-600 dark:text-primary-400 mb-4">Crie sua conta grátis e entre na fila de espera do rachão.</p>
          <div class="flex flex-col sm:flex-row gap-2">
            <a
              href="/register?next=/match/{matchHash}&join_waitlist=1"
              class="btn btn-primary flex-1 justify-center text-sm">
              Criar conta e participar
            </a>
            <a
              href="/login?next=/match/{matchHash}&join_waitlist=1"
              class="btn btn-secondary flex-1 justify-center text-sm">
              Já tenho conta
            </a>
          </div>
        </div>
      {/if}

      <!-- Waitlist button for logged-in non-members -->
      {#if $isLoggedIn && !$isAdmin && match.group_is_public && (match.status === 'open' || match.status === 'in_progress') && !match.attendances.some(a => a.player.id === $currentPlayer?.id)}
        <div class="card mb-4 p-4">
          {#if waitlistEntry}
            <div class="flex items-center gap-2 text-sm text-amber-700 dark:text-amber-300">
              <Clock3 size={15} class="shrink-0" />
              <span class="font-medium">Candidatura enviada — aguardando aprovação do admin</span>
            </div>
          {:else if isWaitlistFull}
            <div class="text-sm text-red-500 dark:text-red-400 font-medium text-center py-1">
              ⛔ Rachão lotado — não há vagas disponíveis no momento
            </div>
          {:else}
            <p class="text-sm text-gray-600 dark:text-gray-400 mb-3">Você não é membro deste grupo. Solicite uma vaga para o admin.</p>
            <button
              onclick={() => showWaitlistModal = true}
              class="btn btn-primary w-full justify-center gap-2">
              <UserPlus size={15} /> Quero jogar!
            </button>
          {/if}
        </div>
      {/if}

      <!-- Voting chip (before player list) -->
      {#if voteStatus && match.status === 'closed' && $isLoggedIn && !$isAdmin && mine?.status === 'confirmed' && !showVoteModal}
        <button
          onclick={() => showVoteModal = true}
          class="mb-3 w-full card px-4 py-3 flex items-center justify-between hover:bg-gray-50 dark:hover:bg-gray-700/60 transition-colors text-left">
          <span class="text-sm font-semibold text-gray-700 dark:text-gray-200 flex items-center gap-2">
            🏆 Votação pós-partida
            <span class="text-xs font-normal px-2 py-0.5 rounded-full
              {voteStatus.status === 'open' ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400' :
               voteStatus.status === 'closed' ? 'bg-gray-100 text-gray-500 dark:bg-gray-700 dark:text-gray-400' :
               'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400'}">
              {voteStatus.status === 'open' ? 'Aberta' : voteStatus.status === 'closed' ? 'Encerrada' : 'Em breve'}
            </span>
          </span>
          <span class="text-xs text-primary-600 dark:text-primary-400 font-medium shrink-0">Ver →</span>
        </button>
      {/if}

      <!-- Admin: voting status card with early-close button -->
      {#if voteStatus && match.status === 'closed' && $isAdmin}
        <div class="mb-3 card px-4 py-3 flex items-center justify-between gap-3">
          <span class="text-sm font-semibold text-gray-700 dark:text-gray-200 flex items-center gap-2">
            🏆 Votação
            <span class="text-xs font-normal px-2 py-0.5 rounded-full
              {voteStatus.status === 'open' ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400' :
               voteStatus.status === 'closed' ? 'bg-gray-100 text-gray-500 dark:bg-gray-700 dark:text-gray-400' :
               'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400'}">
              {voteStatus.status === 'open' ? 'Aberta' : voteStatus.status === 'closed' ? 'Encerrada' : 'Em breve'}
            </span>
          </span>
          <span class="text-xs text-gray-400 dark:text-gray-500 shrink-0">
            {voteStatus.voter_count}/{voteStatus.eligible_count} votos
          </span>
          {#if voteStatus.status === 'open'}
            <button
              onclick={closeVotingEarly}
              disabled={closingVote}
              class="btn btn-sm btn-ghost text-red-500 hover:text-red-600 dark:text-red-400 disabled:opacity-40 shrink-0">
              {closingVote ? 'Encerrando…' : 'Encerrar agora'}
            </button>
          {/if}
        </div>
      {/if}

      <!-- Teams card — above player lists -->
      {#if (teamsData && teamsData.teams.length > 0) || (isGroupAdmin && (match.status === 'open' || match.status === 'in_progress'))}
        <div class="card mb-3 overflow-hidden">
          <div class="flex items-center gap-3 px-4 py-3">
            <span class="text-xl">⚽</span>
            <div class="flex-1 min-w-0">
              {#if teamsData && teamsData.teams.length > 0}
                <p class="text-sm font-semibold text-gray-900 dark:text-gray-100">Times sorteados</p>
                <p class="text-xs text-gray-500 dark:text-gray-400">{teamsData.teams.length} times</p>
              {:else}
                <p class="text-sm font-semibold text-gray-900 dark:text-gray-100">Sorteio de times</p>
                <p class="text-xs text-gray-500 dark:text-gray-400">
                  {!match.players_per_team ? 'Configure jogadores por time no grupo' :
                   match.confirmed_count < (match.players_per_team + 1) * 2 ? 'Aguardando confirmações suficientes' :
                   'Pronto para sortear'}
                </p>
              {/if}
            </div>
            <div class="flex items-center gap-2 shrink-0">
              {#if isGroupAdmin && (match.status === 'open' || match.status === 'in_progress')}
                {#if !match.players_per_team || match.confirmed_count < (match.players_per_team + 1) * 2}
                  <button class="btn-sm btn-secondary gap-1 opacity-50" disabled>
                    <Shuffle size={12} /> {teamsData ? 'Remontar' : 'Montar'}
                  </button>
                {:else}
                  <button
                    onclick={() => teamsData ? (confirmTeamsOpen = true) : generateTeams()}
                    disabled={generatingTeams}
                    class="btn-sm btn-primary gap-1">
                    <Shuffle size={12} /> {generatingTeams ? 'Sorteando…' : teamsData ? 'Remontar' : 'Montar'}
                  </button>
                {/if}
              {/if}
              {#if teamsData && teamsData.teams.length > 0}
                <a href="/match/{matchHash}/teams" class="btn-sm btn-secondary gap-1">
                  <ExternalLink size={12} /> Ver Times
                </a>
              {/if}
            </div>
          </div>
        </div>
      {/if}

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
                  {#if voteStatus && voteStatus.status !== 'not_open'}
                    <span class="text-sm shrink-0" title="{voteStatus.voted_player_ids.includes(a.player.id) ? 'Votou' : 'Não votou ainda'}">
                      {voteStatus.voted_player_ids.includes(a.player.id) ? '✅' : '⏳'}
                    </span>
                  {/if}
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

        <!-- Absent members (admin only) -->
        {#if isGroupAdmin && (match.status === 'open' || match.status === 'in_progress') && absentMembers.length > 0}
          <div class="card overflow-hidden">
            <div class="px-4 py-2 bg-blue-50 dark:bg-blue-900/20 border-b border-gray-100 dark:border-gray-700">
              <h3 class="text-sm font-semibold text-blue-700 dark:text-blue-300 flex items-center gap-1.5">
                <UserPlus size={14} /> Adicionar ao rachão ({absentMembers.length})
              </h3>
            </div>
            <ul class="divide-y divide-gray-100 dark:divide-gray-700">
              {#each absentMembers as mb}
                <li class="px-4 py-2 text-sm text-gray-600 dark:text-gray-400 flex items-center gap-2.5">
                  <span class="flex-1 font-medium text-gray-800 dark:text-gray-200">{mb.player.nickname || mb.player.name}</span>
                  <div class="flex gap-1 shrink-0">
                    <button
                      class="text-xs px-2 py-0.5 rounded border border-green-200 text-green-600 hover:bg-green-50 dark:border-green-800 dark:text-green-400 dark:hover:bg-green-900/20 disabled:opacity-40 flex items-center gap-1"
                      onclick={() => respondFor(mb.player.id, 'confirmed')}
                      disabled={adminResponding === mb.player.id}>
                      <CheckCircle size={11} /> Confirmar
                    </button>
                    <button
                      class="text-xs px-2 py-0.5 rounded border border-red-200 text-red-500 hover:bg-red-50 dark:border-red-800 dark:text-red-400 dark:hover:bg-red-900/20 disabled:opacity-40 flex items-center gap-1"
                      onclick={() => respondFor(mb.player.id, 'declined')}
                      disabled={adminResponding === mb.player.id}>
                      <XCircle size={11} /> Recusar
                    </button>
                  </div>
                </li>
              {/each}
            </ul>
          </div>
        {/if}
      </div>
      </div><!-- /relative RSVP wrapper -->


      <!-- Share -->
      <div class="mt-6 pt-5 border-t border-gray-200 dark:border-gray-700 flex gap-3">
        <button onclick={shareWhatsApp} class="flex-1 btn btn-secondary justify-center gap-2">
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="currentColor" class="shrink-0">
            <path d="M17.472 14.382c-.297-.149-1.758-.867-2.03-.967-.273-.099-.471-.148-.67.15-.197.297-.767.966-.94 1.164-.173.199-.347.223-.644.075-.297-.15-1.255-.463-2.39-1.475-.883-.788-1.48-1.761-1.653-2.059-.173-.297-.018-.458.13-.606.134-.133.298-.347.446-.52.149-.174.198-.298.298-.497.099-.198.05-.371-.025-.52-.075-.149-.669-1.612-.916-2.207-.242-.579-.487-.5-.669-.51-.173-.008-.371-.01-.57-.01-.198 0-.52.074-.792.372-.272.297-1.04 1.016-1.04 2.479 0 1.462 1.065 2.875 1.213 3.074.149.198 2.096 3.2 5.077 4.487.709.306 1.262.489 1.694.625.712.227 1.36.195 1.871.118.571-.085 1.758-.719 2.006-1.413.248-.694.248-1.289.173-1.413-.074-.124-.272-.198-.57-.347z"/>
            <path d="M12 0C5.373 0 0 5.373 0 12c0 2.126.558 4.121 1.533 5.853L.036 23.964l6.252-1.639A11.945 11.945 0 0 0 12 24c6.627 0 12-5.373 12-12S18.627 0 12 0zm0 21.818a9.8 9.8 0 0 1-4.998-1.366l-.358-.213-3.712.974 1.014-3.598-.233-.371A9.818 9.818 0 1 1 12 21.818z"/>
          </svg>
          WhatsApp
        </button>
        <button onclick={copyLink} class="flex-1 btn btn-secondary justify-center gap-2">
          <Link2 size={16} /> Copiar link
        </button>
      </div>
      <div class="mt-4 text-center">
        <a href="https://rachao.app" target="_blank" rel="noopener noreferrer" class="text-xs text-gray-400 dark:text-gray-600 hover:text-gray-500 dark:hover:text-gray-400 transition-colors">rachao.app © 2026</a>
      </div>
    {/if}
  </main>
</PageBackground>

<!-- Results promo overlay (blur) -->
{#if showResultsPromo}
  <div class="fixed inset-0 z-50 backdrop-blur-sm bg-black/40 flex items-center justify-center px-6">
    <div class="bg-white dark:bg-gray-800 rounded-2xl shadow-2xl w-full max-w-sm p-6 text-center">
      <p class="text-4xl mb-3">🏆</p>
      <h2 class="text-lg font-bold text-gray-900 dark:text-gray-100 mb-1">Votação encerrada!</h2>
      <p class="text-sm text-gray-500 dark:text-gray-400 mb-5">Confira quem foram os melhores desta partida.</p>
      <a
        href="/match/{matchHash}/results"
        class="btn btn-primary w-full justify-center mb-3">
        Ver resultado completo
      </a>
      <button
        onclick={() => { showResultsPromo = false; }}
        class="btn btn-secondary w-full justify-center">
        <X size={15} /> Fechar
      </button>
    </div>
  </div>
{/if}

<!-- Voting Modal (fixed overlay, bottom sheet on mobile / centered on desktop) -->
{#if voteStatus && match && match.status === 'closed' && $isLoggedIn && !$isAdmin && mine?.status === 'confirmed' && showVoteModal}
  <!-- Backdrop -->
  <div
    class="fixed inset-0 z-50 bg-black/60 flex items-end sm:items-center justify-center"
    role="dialog"
    aria-modal="true"
    onclick={(e) => { if (e.target === e.currentTarget) showVoteModal = false; }}>

    <!-- Card -->
    <div class="bg-white dark:bg-gray-800 w-full sm:max-w-md rounded-t-2xl sm:rounded-2xl shadow-2xl flex flex-col max-h-[90dvh]">

      <!-- Header (sticky) -->
      <div class="flex items-center justify-between px-5 py-4 border-b border-gray-100 dark:border-gray-700 shrink-0">
        <p class="font-bold text-gray-800 dark:text-gray-100 flex items-center gap-2">
          🏆 Votação pós-partida
          <span class="text-xs font-normal px-2 py-0.5 rounded-full
            {voteStatus.status === 'open' ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400' :
             voteStatus.status === 'closed' ? 'bg-gray-100 text-gray-500 dark:bg-gray-700 dark:text-gray-400' :
             'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400'}">
            {voteStatus.status === 'open' ? 'Aberta' : voteStatus.status === 'closed' ? 'Encerrada' : 'Em breve'}
          </span>
        </p>
        <button
          onclick={() => showVoteModal = false}
          class="p-1.5 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 text-gray-400 hover:text-gray-600 dark:hover:text-gray-200 transition-colors"
          aria-label="Fechar">
          <X size={18} />
        </button>
      </div>

      <!-- Body (scrollable) -->
      <div class="overflow-y-auto p-5">
        {#if voteStatus.status === 'not_open'}
          <div class="text-center py-6">
            <p class="text-4xl mb-3">⏳</p>
            <p class="text-base font-semibold text-gray-700 dark:text-gray-200">{voteStatus.time_label}</p>
            <p class="text-sm text-gray-400 dark:text-gray-500 mt-1">
              {voteStatus.vote_open_delay_minutes === 0
                ? 'Imediatamente após o término da partida'
                : `${voteStatus.vote_open_delay_minutes} min após o término da partida`}
            </p>
          </div>

        {:else if voteStatus.status === 'open'}
          {#if voteStatus.current_player_voted || voteSubmitted}
            <div class="text-center py-6">
              <p class="text-4xl mb-3">✅</p>
              <p class="text-base font-semibold text-gray-700 dark:text-gray-200">Voto registrado!</p>
              <p class="text-sm text-gray-400 dark:text-gray-500 mt-1">
                {voteStatus.voter_count} de {voteStatus.eligible_count} jogador{voteStatus.eligible_count !== 1 ? 'es' : ''} votaram
              </p>
              <p class="text-sm text-gray-400 dark:text-gray-500 mt-1">Resultados: {voteStatus.time_label}</p>
            </div>
          {:else if voteEligible.length === 0}
            <p class="text-sm text-center text-gray-400 dark:text-gray-500 py-6">Nenhum jogador elegível para votar.</p>
          {:else}
            <VoteForm eligiblePlayers={voteEligible} onsubmit={submitVote} saving={voteSaving} />
          {/if}

        {:else if voteStatus.status === 'closed'}
          {#if voteResults}
            {#if voteResults.total_voters === 0}
              <div class="text-center py-6">
                <p class="text-3xl mb-3">😶</p>
                <p class="text-sm font-semibold text-gray-600 dark:text-gray-300">Ninguém votou nesta partida</p>
                <p class="text-xs text-gray-400 dark:text-gray-500 mt-1">A votação encerrou sem votos registrados.</p>
              </div>
            {:else}
              <VoteResults results={voteResults} />
            {/if}
          {:else}
            <p class="text-sm text-center text-gray-400 dark:text-gray-500 py-6">Carregando resultados…</p>
          {/if}
        {/if}
      </div>

    </div>
  </div>
{/if}

<ConfirmDialog
  bind:open={confirmOpen}
  message={confirmMessage}
  confirmLabel="Encerrar"
  danger={true}
  onConfirm={confirmAction}
/>

{#if match && showWaitlistModal}
  <WaitlistModal
    bind:open={showWaitlistModal}
    {match}
    submitting={submittingWaitlist}
    onsubmit={submitWaitlist}
    onclose={() => showWaitlistModal = false}
  />
{/if}

<ConfirmDialog
  bind:open={confirmTeamsOpen}
  message="Remontar os times vai substituir o sorteio atual. Continuar?"
  confirmLabel="Remontar"
  danger={false}
  onConfirm={generateTeams}
/>
