<script lang="ts">
  import { page } from '$app/stores';
  import { goto } from '$app/navigation';
  import { groups as groupsApi, matches as matchesApi, invites, players as playersApi, votes as votesApi, ApiError } from '$lib/api';
  import type { GroupDetail, GroupMember, Match, Player, VoteStatusResponse, PlayerStatItem } from '$lib/api';
  import { currentPlayer, isAdmin, isLoggedIn } from '$lib/stores/auth';
  import { toastSuccess, toastError, toastInfo } from '$lib/stores/toast';
  import Modal from '$lib/components/Modal.svelte';
  import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
  import DatePicker from '$lib/components/DatePicker.svelte';
  import TimePicker from '$lib/components/TimePicker.svelte';
  import { Plus, Calendar, Users, Link, Trash2, Clock, MapPin, Copy, UserPlus, ChevronRight, ShieldCheck, ShieldOff, Pencil } from 'lucide-svelte';
  import PageBackground from '$lib/components/PageBackground.svelte';
  import StarRating from '$lib/components/StarRating.svelte';
  import { relativeDate } from '$lib/utils.js';

  const groupId = $page.params.id;

  let group: GroupDetail | null = $state(null);
  let matchList: Match[] = $state([]);
  let loading = $state(true);
  let tab: 'upcoming' | 'past' | 'members' | 'stats' = $state('upcoming');

  // Modals
  let showMatch = $state(false);
  let showEditMatch = $state(false);
  let showInvite = $state(false);
  let showAddMember = $state(false);
  let showEditGroup = $state(false);

  let inviteLink = $state('');
  let inviteQr = $state('');
  const COURT_LABELS: Record<string, string> = { campo: 'Campo', sintetico: 'Sintético', terrao: 'Terrão', quadra: 'Quadra' };
  let matchForm = $state({ match_date: '', start_time: '20:30', end_time: '', location: '', address: '', court_type: '', players_per_team: '', max_players: '', notes: '' });
  let editMatchForm = $state({ match_date: '', start_time: '', end_time: '', location: '', address: '', court_type: '', players_per_team: '', max_players: '', notes: '', status: 'open' });
  let editingMatch: Match | null = $state(null);
  let saving = $state(false);

  let allPlayers: Player[] = $state([]);
  let addMemberId = $state('');

  let editForm = $state({ name: '', description: '', per_match_amount: '', monthly_amount: '', recurrence_enabled: false, vote_open_delay_minutes: 20, vote_duration_hours: 24 });

  // Stats tab
  let stats = $state<PlayerStatItem[] | null>(null);
  let statsLoading = $state(false);
  let statsPeriod = $state<'annual' | 'monthly'>('annual');
  let statsMonth = $state<string>(new Date().toISOString().slice(0, 7));
  let statsPeriodLabel = $state('');

  // Meses do ano corrente até o mês atual (para o seletor)
  const _now = new Date();
  const _year = _now.getFullYear();
  const _curMonth = _now.toISOString().slice(0, 7);
  const _monthNames = ['Janeiro','Fevereiro','Março','Abril','Maio','Junho','Julho','Agosto','Setembro','Outubro','Novembro','Dezembro'];
  const availableMonths = Array.from({ length: 12 }, (_, i) => {
    const m = String(i + 1).padStart(2, '0');
    return { value: `${_year}-${m}`, label: `${_monthNames[i]} ${_year}` };
  }).filter(m => m.value <= _curMonth);

  $effect(() => {
    if (tab !== 'stats') return;
    void statsPeriod;
    void statsMonth;
    let cancelled = false;
    stats = null;
    statsLoading = true;
    const period = statsPeriod;
    const month = period === 'monthly' ? statsMonth : undefined;
    groupsApi.getStats(groupId, { period, month })
      .then(res => { if (!cancelled) { stats = res.players; statsPeriodLabel = res.period_label; } })
      .catch(() => { if (!cancelled) { stats = []; statsPeriodLabel = ''; } })
      .finally(() => { if (!cancelled) statsLoading = false; });
    return () => { cancelled = true; };
  });

  function fmtMinutes(m: number): string {
    if (m === 0) return '—';
    const h = Math.floor(m / 60);
    const min = m % 60;
    if (h === 0) return `${min}min`;
    if (min === 0) return `${h}h`;
    return `${h}h ${min}min`;
  }

  const MEDALS: Record<number, string> = { 1: '🥇', 2: '🥈', 3: '🥉' };

  let nonAdminMembers = $derived(
    (group?.members.filter(m => m.player.role !== 'admin') ?? [])
      .sort((a, b) => (a.player.nickname || a.player.name).localeCompare(b.player.nickname || b.player.name, 'pt-BR', { sensitivity: 'base' }))
  );
  let roleEditMember = $state<{ id: string; name: string; role: string; skill_stars: number; is_goalkeeper: boolean } | null>(null);
  let skillSaving = $state(false);

  const today = new Date().toISOString().slice(0, 10);
  function matchSortKey(m: { match_date: string; start_time: string }) {
    return `${m.match_date}T${m.start_time}`;
  }
  let upcomingMatches = $derived(matchList.filter(m => m.status === 'open' || m.status === 'in_progress').sort((a, b) => matchSortKey(a).localeCompare(matchSortKey(b))));
  let pastMatches = $derived(matchList.filter(m => m.status === 'closed').sort((a, b) => matchSortKey(b).localeCompare(matchSortKey(a))));

  let confirmOpen = $state(false);
  let confirmMessage = $state('');
  let confirmLabel = $state('Confirmar');
  let confirmAction = $state<() => void>(() => {});

  // Partidas encerradas com votação aberta e pendente para o jogador atual
  let pendingVotes = $state<{ match: Match; status: VoteStatusResponse }[]>([]);
  $effect(() => {
    if (!$isLoggedIn || $isAdmin) { pendingVotes = []; return; }
    const closed = matchList.filter(m => m.status === 'closed').slice(0, 3);
    if (closed.length === 0) { pendingVotes = []; return; }
    Promise.all(
      closed.map(m => votesApi.getStatus(m.id).then(s => ({ match: m, status: s })).catch(() => null))
    ).then(results => {
      pendingVotes = results.filter(
        (r): r is { match: Match; status: VoteStatusResponse } =>
          r !== null && r.status.status === 'open' && !r.status.current_player_voted
      );
    });
  });

  function askConfirm(message: string, label: string, action: () => void) {
    confirmMessage = message;
    confirmLabel = label;
    confirmAction = action;
    confirmOpen = true;
  }

  function fmtPricingParts(perMatch: number | string | null, monthly: number | string | null): string[] {
    const parts: string[] = [];
    if (perMatch != null) parts.push(`R$ ${Number(perMatch).toFixed(2).replace('.', ',')} avulso`);
    if (monthly != null) parts.push(`R$ ${Number(monthly).toFixed(2).replace('.', ',')} mensal`);
    return parts;
  }

  function openEditGroup() {
    if (!group) return;
    editForm = {
      name: group.name,
      description: group.description ?? '',
      per_match_amount: group.per_match_amount != null ? String(group.per_match_amount) : '',
      monthly_amount: group.monthly_amount != null ? String(group.monthly_amount) : '',
      recurrence_enabled: group.recurrence_enabled,
      vote_open_delay_minutes: group.vote_open_delay_minutes ?? 20,
      vote_duration_hours: group.vote_duration_hours ?? 24,
    };
    showEditGroup = true;
  }

  async function saveEditGroup() {
    saving = true;
    try {
      await groupsApi.update(groupId, {
        name: editForm.name,
        description: editForm.description || undefined,
        per_match_amount: editForm.per_match_amount !== '' ? parseFloat(editForm.per_match_amount) : null,
        monthly_amount: editForm.monthly_amount !== '' ? parseFloat(editForm.monthly_amount) : null,
        recurrence_enabled: editForm.recurrence_enabled,
        vote_open_delay_minutes: editForm.vote_open_delay_minutes,
        vote_duration_hours: editForm.vote_duration_hours,
      });
      group = await groupsApi.get(groupId);
      showEditGroup = false;
      toastSuccess('Grupo atualizado!');
    } catch (e) { toastError(e instanceof ApiError ? e.message : 'Erro ao salvar'); }
    saving = false;
  }

  $effect(() => {
    let cancelled = false;
    (async () => {
      try {
        const [g, ms] = await Promise.all([
          groupsApi.get(groupId),
          matchesApi.list(groupId),
        ]);
        if (!cancelled) { group = g; matchList = ms; }
      } catch (e) { if (!cancelled) toastError('Erro ao carregar grupo'); }
      if (!cancelled) loading = false;
    })();
    return () => { cancelled = true; };
  });

  function isGroupAdmin() {
    if ($isAdmin) return true;
    return group?.members.some(m => m.player.id === $currentPlayer?.id && m.role === 'admin') ?? false;
  }

  async function createMatch() {
    saving = true;
    try {
      const m = await matchesApi.create(groupId, matchForm);
      matchList = [m, ...matchList];
      showMatch = false;
      matchForm = { match_date: '', start_time: '20:30', end_time: '', location: '', address: '', court_type: '', players_per_team: '', max_players: '', notes: '' };
      toastSuccess('Rachão criado!');
    } catch (e) { toastError(e instanceof ApiError ? e.message : 'Erro'); }
    saving = false;
  }

  async function generateInvite() {
    try {
      const inv = await invites.create(groupId);
      const base = window.location.origin;
      inviteLink = `${base}/invite/${inv.token}`;
      const QRCode = (await import('qrcode')).default;
      inviteQr = await QRCode.toDataURL(inviteLink, { width: 256, margin: 2, color: { dark: '#111827', light: '#ffffff' } });
      showInvite = true;
    } catch (e) { toastError('Erro ao gerar convite'); }
  }

  function copyLink() {
    navigator.clipboard.writeText(inviteLink);
    toastInfo('Link copiado!');
  }

  async function openAddMember() {
    try { allPlayers = await playersApi.list(); } catch {}
    showAddMember = true;
  }

  async function addMember() {
    if (!addMemberId) return;
    saving = true;
    try {
      await groupsApi.addMember(groupId, addMemberId);
      group = await groupsApi.get(groupId);
      showAddMember = false;
      addMemberId = '';
      toastSuccess('Membro adicionado!');
    } catch (e) {
      if (e instanceof ApiError && e.status === 403 && e.message === 'PLAN_LIMIT_EXCEEDED') {
        toastError('Limite de membros do plano atingido (máx. 30 membros no plano Grátis).');
      } else {
        toastError(e instanceof ApiError ? e.message : 'Erro');
      }
    }
    saving = false;
  }

  async function toggleRole(playerId: string, currentRole: string, name: string) {
    const newRole = currentRole === 'admin' ? 'member' : 'admin';
    const actionLabel = newRole === 'admin' ? 'Tornar Presidente' : 'Remover Presidência';
    const msg = newRole === 'admin'
      ? `Tornar "${name}" presidente do grupo?`
      : `Remover a presidência de "${name}"?`;
    askConfirm(msg, actionLabel, async () => {
      try {
        await groupsApi.updateMemberRole(groupId, playerId, newRole);
        group = await groupsApi.get(groupId);
        toastSuccess(newRole === 'admin' ? `${name} agora é presidente do grupo` : `${name} voltou a ser membro`);
      } catch (e) { toastError(e instanceof ApiError ? e.message : 'Erro ao alterar papel'); }
    });
  }

  async function saveSkill(playerId: string, skill_stars: number, is_goalkeeper: boolean) {
    skillSaving = true;
    try {
      await groupsApi.updateMemberSkill(groupId, playerId, { skill_stars, is_goalkeeper });
      group = await groupsApi.get(groupId);
    } catch (e) { toastError(e instanceof ApiError ? e.message : 'Erro ao salvar'); }
    skillSaving = false;
  }

  async function removeMember(playerId: string, name: string) {
    askConfirm(`Remover "${name}" do grupo?`, 'Remover', async () => {
      try {
        await groupsApi.removeMember(groupId, playerId);
        group = await groupsApi.get(groupId);
        toastSuccess('Membro removido');
      } catch (e) { toastError('Erro ao remover membro'); }
    });
  }

  async function deleteMatch(m: Match) {
    askConfirm('Excluir este rachão?', 'Excluir', async () => {
      try {
        await matchesApi.delete(groupId, m.id);
        matchList = matchList.filter(x => x.id !== m.id);
        toastSuccess('Rachão excluído');
      } catch (e) { toastError('Erro ao excluir'); }
    });
  }

  function fmtDate(d: string) {
    const s = relativeDate(d, { weekday: 'long', day: '2-digit', month: 'long' });
    return s.charAt(0).toUpperCase() + s.slice(1);
  }

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

  function openEditMatch(m: Match) {
    editingMatch = m;
    editMatchForm = {
      match_date: m.match_date,
      start_time: m.start_time.slice(0, 5),
      end_time: m.end_time?.slice(0, 5) ?? '',
      location: m.location,
      address: m.address ?? '',
      court_type: m.court_type ?? '',
      players_per_team: m.players_per_team ? String(m.players_per_team) : '',
      max_players: m.max_players ? String(m.max_players) : '',
      notes: m.notes ?? '',
      status: m.status,
    };
    showEditMatch = true;
  }

  async function saveEditMatch() {
    if (!editingMatch) return;
    saving = true;
    try {
      const updated = await matchesApi.update(groupId, editingMatch.id, {
        match_date: editMatchForm.match_date,
        start_time: editMatchForm.start_time,
        end_time: editMatchForm.end_time || null,
        location: editMatchForm.location,
        address: editMatchForm.address || null,
        court_type: (editMatchForm.court_type as any) || null,
        players_per_team: editMatchForm.players_per_team ? parseInt(editMatchForm.players_per_team) : null,
        max_players: editMatchForm.max_players ? parseInt(editMatchForm.max_players) : null,
        notes: editMatchForm.notes || null,
        status: editMatchForm.status as any,
      });
      matchList = matchList.map(m => m.id === updated.id ? updated : m);
      showEditMatch = false;
      toastSuccess('Rachão atualizado!');
    } catch (e) { toastError(e instanceof ApiError ? e.message : 'Erro ao salvar'); }
    saving = false;
  }
</script>

<svelte:head><title>{group?.name ?? 'Grupo'} — rachao.app</title></svelte:head>

<PageBackground>
<main class="relative z-10 max-w-7xl mx-auto px-4 py-8">
  {#if loading}
    <div class="animate-pulse space-y-4">
      <div class="h-8 bg-gray-200 dark:bg-gray-700 rounded w-1/3"></div>
      <div class="h-4 bg-gray-100 dark:bg-gray-700 rounded w-1/2"></div>
    </div>
  {:else if group}
    <!-- Header -->
    <div class="mb-6 space-y-3">
      <div class="flex items-start justify-between gap-3">
        <div>
          <h1 class="text-2xl font-bold text-white">{group.name}</h1>
          {#if group.description}<p class="text-gray-300 text-sm mt-0.5">{group.description}</p>{/if}
        </div>
        {#if isGroupAdmin()}
          <div class="flex gap-2 shrink-0">
            <button class="btn-secondary btn-sm" onclick={openEditGroup}><Pencil size={14} /> Editar</button>
          </div>
        {/if}
      </div>
      <div class="flex flex-wrap items-center gap-x-4 gap-y-1">
        <span class="text-xs text-gray-300 flex items-center gap-1">
          <Users size={12} /> {nonAdminMembers.length} jogador{nonAdminMembers.length !== 1 ? 'es' : ''}
        </span>
        {#if group.per_match_amount != null || group.monthly_amount != null}
          {#each fmtPricingParts(group.per_match_amount, group.monthly_amount) as part}
            <span class="text-xs text-amber-700 dark:text-amber-400 font-medium">{part}</span>
          {/each}
        {:else}
          <span class="text-xs text-green-600 dark:text-green-400">Gratuito</span>
        {/if}
        {#if group.recurrence_enabled}
          <span class="text-xs text-primary-600 dark:text-primary-400">Recorrência semanal</span>
        {/if}
        {#if isGroupAdmin()}
          <div class="flex items-center gap-1.5 text-xs text-gray-300 bg-white/10 rounded-lg px-2.5 py-1 w-full sm:w-auto">
            <span>🗳️</span>
            <span class="text-gray-400">Votação</span>
            <span class="text-white/30">·</span>
            <span>
              {group.vote_open_delay_minutes === 0 ? 'inicia imediatamente' : `inicia em ${group.vote_open_delay_minutes}min`}
            </span>
            <span class="text-white/30">·</span>
            <span>encerra em {group.vote_duration_hours}h</span>
          </div>
        {/if}
      </div>
    </div>

    <!-- Votação pendente -->
    {#if pendingVotes.length > 0}
      <div class="mb-5 space-y-2">
        {#each pendingVotes as { match: m, status: vs }}
          <a
            href="/match/{m.hash}"
            onclick={(e) => { e.preventDefault(); goto('/match/' + m.hash); }}
            class="flex items-center gap-3 px-4 py-3 rounded-xl bg-amber-50 dark:bg-amber-900/20 border border-amber-300 dark:border-amber-700/60 hover:bg-amber-100 dark:hover:bg-amber-900/30 transition-colors">
            <span class="text-xl shrink-0">🏆</span>
            <div class="flex-1 min-w-0">
              <p class="text-sm font-semibold text-amber-800 dark:text-amber-200">Vote nos melhores do Rachão #{m.number}</p>
              <p class="text-xs text-amber-600 dark:text-amber-400">{vs.time_label} · {vs.voter_count} de {vs.eligible_count} já votaram</p>
            </div>
            <ChevronRight size={16} class="text-amber-600 dark:text-amber-400 shrink-0" />
          </a>
        {/each}
      </div>
    {/if}

    <!-- Tabs -->
    <div class="flex gap-1 border-b border-white/20 mb-6 overflow-x-auto [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
      <button
        class="px-3 py-2 text-sm font-medium border-b-2 transition-colors whitespace-nowrap {tab === 'upcoming' ? 'border-primary-400 text-primary-400' : 'border-transparent text-gray-300 hover:text-white'}"
        onclick={() => tab = 'upcoming'}>
        Próximos ({upcomingMatches.length})
      </button>
      {#if pastMatches.length > 0}
        <button
          class="px-3 py-2 text-sm font-medium border-b-2 transition-colors whitespace-nowrap {tab === 'past' ? 'border-primary-400 text-primary-400' : 'border-transparent text-gray-300 hover:text-white'}"
          onclick={() => tab = 'past'}>
          Últimos ({pastMatches.length})
        </button>
      {/if}
      <button
        class="px-3 py-2 text-sm font-medium border-b-2 transition-colors whitespace-nowrap {tab === 'members' ? 'border-primary-400 text-primary-400' : 'border-transparent text-gray-300 hover:text-white'}"
        onclick={() => tab = 'members'}>
        Jogadores ({nonAdminMembers.length})
      </button>
      <button
        class="px-3 py-2 text-sm font-medium border-b-2 transition-colors whitespace-nowrap {tab === 'stats' ? 'border-primary-400 text-primary-400' : 'border-transparent text-gray-300 hover:text-white'}"
        onclick={() => { tab = 'stats'; }}>
        Estatísticas
      </button>
    </div>

    <!-- Próximos / Últimos tabs -->
    {#if tab === 'upcoming' || tab === 'past'}
      {#if isGroupAdmin() && tab === 'upcoming'}
        <div class="flex justify-end mb-4">
          <button class="btn-primary btn-sm" onclick={() => showMatch = true}><Plus size={14} /> Novo Rachão</button>
        </div>
      {/if}
      {@const subList = tab === 'upcoming' ? upcomingMatches : pastMatches}
      {#if subList.length === 0}
        <div class="card p-12 text-center">
          <Calendar size={40} class="text-gray-300 mx-auto mb-3" />
          <p class="text-gray-500">{tab === 'upcoming' ? 'Nenhum rachão agendado.' : 'Nenhum rachão encerrado ainda.'}</p>
        </div>
      {:else}
        <div class="space-y-3">
          {#each subList as m}
            <div class="card hover:shadow-md transition-shadow">
              <div class="card-body">
                <!-- Date + status -->
                <div class="flex items-start gap-2 mb-1">
                  <div class="flex-1 min-w-0">
                    <p class="font-semibold text-gray-900 dark:text-gray-100 leading-snug">
                      <span class="text-primary-600 dark:text-primary-400 font-bold text-base mr-1">#{m.number}</span>{fmtDate(m.match_date)}
                    </p>
                  </div>
                  {#if m.status === 'in_progress'}
                    <span class="shrink-0 inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-semibold bg-red-500/20 text-red-400 border border-red-500/30">
                      <span class="w-1.5 h-1.5 rounded-full bg-red-400 animate-pulse"></span>
                      Bola rolando
                    </span>
                  {:else}
                    <span class="badge {m.status === 'open' ? 'badge-green' : 'badge-gray'} shrink-0">
                      {m.status === 'open' ? 'Aberta' : 'Encerrada'}
                    </span>
                  {/if}
                </div>

                <!-- Time + Location -->
                <div class="flex flex-wrap items-center gap-x-3 gap-y-0.5 mt-1 text-sm text-gray-500 dark:text-gray-400">
                  <span class="flex items-center gap-1 whitespace-nowrap"><Clock size={12} />{fmtTimeRange(m.start_time, m.end_time)}</span>
                  <span class="flex items-center gap-1 min-w-0"><MapPin size={12} /><span class="truncate">{m.location}</span></span>
                </div>

                <!-- Court / players / pricing details -->
                {#if m.court_type || m.players_per_team || m.max_players || group.per_match_amount != null || group.monthly_amount != null}
                  <p class="text-xs text-gray-400 dark:text-gray-500 mt-1">
                    {[
                      m.court_type ? COURT_LABELS[m.court_type] : null,
                      m.players_per_team ? `${m.players_per_team} na linha + gol` : null,
                      m.max_players ? `máx. ${m.max_players}` : null,
                      ...fmtPricingParts(group.per_match_amount, group.monthly_amount),
                    ].filter(Boolean).join(' · ')}
                  </p>
                {/if}

                <!-- Actions -->
                <div class="flex items-center gap-2 mt-3 pt-3 border-t border-gray-100 dark:border-gray-700">
                  <a href="/match/{m.hash}" class="btn-sm btn-secondary shrink-0">
                    Detalhes <ChevronRight size={14} />
                  </a>
                  {#if isGroupAdmin()}
                    <button onclick={() => openEditMatch(m)} class="btn-sm btn-ghost shrink-0">
                      <Pencil size={14} /> Editar
                    </button>
                    <button onclick={() => deleteMatch(m)} class="btn-sm btn-ghost text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20 shrink-0">
                      <Trash2 size={14} /> Excluir
                    </button>
                  {/if}
                </div>
              </div>
            </div>
          {/each}
        </div>
      {/if}
    {/if}

    <!-- Stats tab -->
    {#if tab === 'stats'}
      <!-- Seletor de período -->
      <div class="flex flex-wrap items-center gap-2 mb-4">
        <div class="flex gap-1">
          <button
            class="px-3 py-1.5 rounded-full text-xs font-medium transition-colors {statsPeriod === 'annual' ? 'bg-primary-500 text-white' : 'bg-white/10 text-gray-300 hover:bg-white/20'}"
            onclick={() => { statsPeriod = 'annual'; }}>
            Anual
          </button>
          <button
            class="px-3 py-1.5 rounded-full text-xs font-medium transition-colors {statsPeriod === 'monthly' ? 'bg-primary-500 text-white' : 'bg-white/10 text-gray-300 hover:bg-white/20'}"
            onclick={() => { statsPeriod = 'monthly'; }}>
            Mensal
          </button>
        </div>
        {#if statsPeriod === 'monthly'}
          <select
            bind:value={statsMonth}
            class="bg-white/10 text-gray-200 text-xs rounded-full px-3 py-1.5 border border-white/20 focus:outline-none focus:ring-1 focus:ring-primary-400">
            {#each availableMonths as m}
              <option value={m.value}>{m.label}</option>
            {/each}
          </select>
        {/if}
      </div>

      {#if statsLoading}
        <div class="animate-pulse space-y-2">
          {#each [1,2,3,4,5] as _}
            <div class="h-10 bg-white/10 rounded-lg"></div>
          {/each}
        </div>
      {:else if !stats || stats.length === 0}
        <div class="card p-12 text-center">
          <p class="text-gray-400 dark:text-gray-500 text-sm">Nenhuma estatística disponível ainda.<br>As estatísticas aparecem após o encerramento das votações.</p>
        </div>
      {:else}
        <div class="card overflow-hidden">
          <table class="w-full text-sm">
            <thead>
              <tr class="border-b border-gray-100 dark:border-gray-700 text-xs text-gray-500 dark:text-gray-400">
                <th class="px-4 py-2 text-left w-8">#</th>
                <th class="px-4 py-2 text-left">Jogador</th>
                <th class="px-4 py-2 text-right">Pts</th>
                <th class="px-4 py-2 text-right hidden sm:table-cell">Decepções</th>
                <th class="px-4 py-2 text-right hidden sm:table-cell">Horas</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-gray-700">
              {#each stats as player, i}
                {@const rank = i + 1}
                <tr class="hover:bg-gray-50 dark:hover:bg-gray-700/40 transition-colors">
                  <td class="px-4 py-2.5 text-center">
                    {#if MEDALS[rank]}
                      <span class="text-base">{MEDALS[rank]}</span>
                    {:else}
                      <span class="text-xs text-gray-400">{rank}</span>
                    {/if}
                  </td>
                  <td class="px-4 py-2.5 font-medium text-gray-900 dark:text-gray-100">{player.display_name}</td>
                  <td class="px-4 py-2.5 text-right font-bold text-primary-600 dark:text-primary-400">{player.vote_points}</td>
                  <td class="px-4 py-2.5 text-right hidden sm:table-cell {player.flop_votes > 0 ? 'text-red-500 dark:text-red-400 font-medium' : 'text-gray-400 dark:text-gray-500'}">
                    {player.flop_votes > 0 ? player.flop_votes : '—'}
                  </td>
                  <td class="px-4 py-2.5 text-right hidden sm:table-cell text-gray-600 dark:text-gray-400">{fmtMinutes(player.minutes_played)}</td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
        <p class="mt-2 text-xs text-gray-300 text-center">
          {statsPeriodLabel} · Pontos acumulados nas votações encerradas · Horas apenas em partidas com término registrado
        </p>
      {/if}
    {/if}

    <!-- Members tab -->
    {#if tab === 'members'}
      {#if isGroupAdmin()}
        <div class="flex justify-end gap-2 mb-4">
          <button class="btn-secondary btn-sm" onclick={generateInvite}><Link size={14} /> Convidar</button>
          <button class="btn-secondary btn-sm" onclick={openAddMember}><UserPlus size={14} /> Adicionar</button>
        </div>
      {/if}
      <div class="card overflow-hidden divide-y divide-gray-100 dark:divide-gray-700">
        {#if nonAdminMembers.length === 0}
          <div class="px-6 py-10 text-center text-gray-400 dark:text-gray-500 text-sm">
            <Users size={32} class="mx-auto mb-2 opacity-40" />
            <p>Nenhum jogador no grupo ainda.</p>
            {#if isGroupAdmin()}
              <button class="btn-primary mt-4 btn-sm" onclick={openAddMember}><UserPlus size={14} /> Adicionar jogador</button>
            {/if}
          </div>
        {/if}
        {#each nonAdminMembers as m}
          <div class="flex items-center gap-2 px-4 py-3">
            <!-- Info -->
            <div class="flex-1 min-w-0">
              <p class="font-medium text-sm text-gray-900 dark:text-gray-100 truncate">
                {m.player.nickname || m.player.name}
              </p>
              {#if isGroupAdmin() && m.skill_stars != null}
                <div class="mt-0.5">
                  <StarRating rating={m.skill_stars} readonly size={13} />
                </div>
              {/if}
              {#if m.role === 'admin' || (isGroupAdmin() && m.is_goalkeeper)}
                <div class="flex items-center gap-1 mt-0.5 flex-wrap">
                  {#if m.role === 'admin'}
                    <span class="inline-flex items-center px-1 py-px rounded text-[10px] font-semibold bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400">Presidente</span>
                  {/if}
                  {#if isGroupAdmin() && m.is_goalkeeper}
                    <span class="inline-flex items-center px-1 py-px rounded text-[10px] font-semibold bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400">Goleiro</span>
                  {/if}
                </div>
              {/if}
            </div>
            <!-- Actions -->
            {#if isGroupAdmin() && m.player.id !== $currentPlayer?.id}
              <div class="flex items-center gap-1 shrink-0">
                <button
                  onclick={() => roleEditMember = { id: m.player.id, name: m.player.name, role: m.role, skill_stars: m.skill_stars ?? 2, is_goalkeeper: m.is_goalkeeper ?? false }}
                  class="text-xs px-2 py-1 rounded border border-gray-200 text-gray-600 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-400 dark:hover:bg-gray-700 flex items-center gap-1 shrink-0">
                  <Pencil size={11} /> Editar
                </button>
                <button
                  onclick={() => removeMember(m.player.id, m.player.name)}
                  class="text-xs px-2 py-1 rounded border border-red-200 text-red-500 hover:bg-red-50 dark:border-red-800 dark:text-red-400 dark:hover:bg-red-900/20 flex items-center gap-1 shrink-0">
                  <Trash2 size={11} /> Remover
                </button>
              </div>
            {/if}
          </div>
        {/each}
      </div>
    {/if}
  {/if}
</main>
</PageBackground>

<!-- Create match modal -->
<Modal bind:open={showMatch} title="Novo Rachão">
  <form onsubmit={(e) => { e.preventDefault(); createMatch(); }} class="space-y-4">
    <div class="form-group">
      <label class="label" for="mdate">Data *</label>
      <DatePicker id="mdate" bind:value={matchForm.match_date} required />
    </div>
    <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
      <div class="form-group">
        <label class="label" for="mtime">Início *</label>
        <TimePicker id="mtime" bind:value={matchForm.start_time} required />
      </div>
      <div class="form-group">
        <label class="label" for="mendtime">Término <span class="text-gray-400 dark:text-gray-500 font-normal">(opcional)</span></label>
        <TimePicker id="mendtime" bind:value={matchForm.end_time} />
      </div>
    </div>
    <div class="form-group">
      <label class="label" for="mloc">Local *</label>
      <input id="mloc" class="input" bind:value={matchForm.location} placeholder="Ex: Arena GQC — Quadra 3" required />
    </div>
    <div class="form-group">
      <label class="label" for="maddr">Endereço <span class="text-gray-400 font-normal">(opcional — para abrir no Maps)</span></label>
      <input id="maddr" class="input" bind:value={matchForm.address} placeholder="Ex: Rua das Flores, 123 — São Paulo, SP" />
    </div>
    <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
      <div class="form-group">
        <label class="label" for="mcourt">Tipo de quadra</label>
        <select id="mcourt" class="input" bind:value={matchForm.court_type}>
          <option value="">— selecione —</option>
          <option value="campo">Campo</option>
          <option value="sintetico">Sintético</option>
          <option value="terrao">Terrão</option>
          <option value="quadra">Quadra</option>
        </select>
      </div>
      <div class="form-group">
        <label class="label" for="mplayers">Jogadores por time <span class="text-gray-400 font-normal">(sem goleiro)</span></label>
        <select id="mplayers" class="input" bind:value={matchForm.players_per_team}>
          <option value="">— selecione —</option>
          {#each [4, 5, 6, 7, 8, 9, 10] as n}
            <option value={n}>{n} na linha</option>
          {/each}
        </select>
      </div>
    </div>
    <div class="form-group">
      <label class="label" for="mmaxp">Máximo de jogadores <span class="text-gray-400 font-normal">(opcional — limita confirmações)</span></label>
      <input id="mmaxp" class="input" type="number" min="2" bind:value={matchForm.max_players} placeholder="Ex: 14" />
    </div>
    <div class="form-group">
      <label class="label" for="mnotes">Observações</label>
      <textarea id="mnotes" class="input resize-none" rows="2" bind:value={matchForm.notes} placeholder="Opcional…"></textarea>
    </div>
    <div class="flex gap-3 justify-end pt-2">
      <button type="button" class="btn-secondary" onclick={() => showMatch = false}>Cancelar</button>
      <button type="submit" class="btn-primary" disabled={saving}>{saving ? 'Criando…' : 'Criar Partida'}</button>
    </div>
  </form>
</Modal>

<!-- Edit match modal -->
<Modal bind:open={showEditMatch} title="Editar Rachão">
  <form onsubmit={(e) => { e.preventDefault(); saveEditMatch(); }} class="space-y-4">
    <div class="form-group">
      <label class="label" for="emdate">Data *</label>
      <DatePicker id="emdate" bind:value={editMatchForm.match_date} required />
    </div>
    <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
      <div class="form-group">
        <label class="label" for="emtime">Início *</label>
        <TimePicker id="emtime" bind:value={editMatchForm.start_time} required />
      </div>
      <div class="form-group">
        <label class="label" for="emendtime">Término <span class="text-gray-400 dark:text-gray-500 font-normal">(opcional)</span></label>
        <TimePicker id="emendtime" bind:value={editMatchForm.end_time} />
      </div>
    </div>
    <div class="form-group">
      <label class="label" for="emloc">Local *</label>
      <input id="emloc" class="input" bind:value={editMatchForm.location} required />
    </div>
    <div class="form-group">
      <label class="label" for="emaddr">Endereço <span class="text-gray-400 dark:text-gray-500 font-normal">(opcional)</span></label>
      <input id="emaddr" class="input" bind:value={editMatchForm.address} placeholder="Ex: Rua das Flores, 123" />
    </div>
    <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
      <div class="form-group">
        <label class="label" for="emcourt">Tipo de quadra</label>
        <select id="emcourt" class="input" bind:value={editMatchForm.court_type}>
          <option value="">— selecione —</option>
          <option value="campo">Campo</option>
          <option value="sintetico">Sintético</option>
          <option value="terrao">Terrão</option>
          <option value="quadra">Quadra</option>
        </select>
      </div>
      <div class="form-group">
        <label class="label" for="emplayers">Jogadores por time</label>
        <select id="emplayers" class="input" bind:value={editMatchForm.players_per_team}>
          <option value="">— selecione —</option>
          {#each [4, 5, 6, 7, 8, 9, 10] as n}
            <option value="{n}">{n} na linha</option>
          {/each}
        </select>
      </div>
    </div>
    <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
      <div class="form-group">
        <label class="label" for="emmaxp">Máximo de jogadores</label>
        <input id="emmaxp" class="input" type="number" min="2" bind:value={editMatchForm.max_players} placeholder="Ex: 14" />
      </div>
      <div class="form-group">
        <label class="label" for="emstatus">Status</label>
        <select id="emstatus" class="input" bind:value={editMatchForm.status}>
          <option value="open">Aberta</option>
          <option value="in_progress">Bola rolando</option>
          <option value="closed">Encerrada</option>
        </select>
      </div>
    </div>
    <div class="form-group">
      <label class="label" for="emnotes">Observações</label>
      <textarea id="emnotes" class="input resize-none" rows="2" bind:value={editMatchForm.notes} placeholder="Opcional…"></textarea>
    </div>
    <div class="flex gap-3 justify-end pt-2">
      <button type="button" class="btn-secondary" onclick={() => showEditMatch = false}>Cancelar</button>
      <button type="submit" class="btn-primary" disabled={saving}>{saving ? 'Salvando…' : 'Salvar'}</button>
    </div>
  </form>
</Modal>

<!-- Invite modal -->
<Modal bind:open={showInvite} title="Convidar Jogador">
  <div class="space-y-4">

    <!-- QR Code -->
    {#if inviteQr}
      <div class="flex flex-col items-center gap-2">
        <div class="bg-white rounded-2xl p-3 shadow-inner border border-gray-100 dark:border-gray-700 inline-block">
          <img src={inviteQr} alt="QR Code de convite" width="220" height="220" class="block" />
        </div>
        <p class="text-xs text-gray-500 dark:text-gray-400 text-center">
          Aponte a câmera do celular para escanear
        </p>
      </div>
    {/if}

    <div class="alert-info text-xs">⏱ Este link expira em <strong>30 minutos</strong> e só pode ser usado <strong>uma vez</strong>.</div>

    <!-- Link + copy -->
    <div class="flex gap-2">
      <input class="input font-mono text-xs" readonly value={inviteLink} />
      <button class="btn-primary shrink-0" onclick={copyLink} title="Copiar link"><Copy size={16} /></button>
    </div>

    <!-- WhatsApp -->
    <a
      href="https://wa.me/?text={encodeURIComponent(`Você foi convidado para o grupo *${group?.name}* no rachao.app!\n\nClique no link abaixo para criar sua conta e entrar no grupo:\n${inviteLink}`)}"
      target="_blank"
      rel="noopener noreferrer"
      class="btn-secondary w-full flex items-center justify-center gap-2">
      <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="currentColor" class="shrink-0">
        <path d="M17.472 14.382c-.297-.149-1.758-.867-2.03-.967-.273-.099-.471-.148-.67.15-.197.297-.767.966-.94 1.164-.173.199-.347.223-.644.075-.297-.15-1.255-.463-2.39-1.475-.883-.788-1.48-1.761-1.653-2.059-.173-.297-.018-.458.13-.606.134-.133.298-.347.446-.52.149-.174.198-.298.298-.497.099-.198.05-.371-.025-.52-.075-.149-.669-1.612-.916-2.207-.242-.579-.487-.5-.669-.51-.173-.008-.371-.01-.57-.01-.198 0-.52.074-.792.372-.272.297-1.04 1.016-1.04 2.479 0 1.462 1.065 2.875 1.213 3.074.149.198 2.096 3.2 5.077 4.487.709.306 1.262.489 1.694.625.712.227 1.36.195 1.871.118.571-.085 1.758-.719 2.006-1.413.248-.694.248-1.289.173-1.413-.074-.124-.272-.198-.57-.347z"/>
        <path d="M12 0C5.373 0 0 5.373 0 12c0 2.126.558 4.121 1.533 5.853L.036 23.964l6.252-1.639A11.945 11.945 0 0 0 12 24c6.627 0 12-5.373 12-12S18.627 0 12 0zm0 21.818a9.8 9.8 0 0 1-4.998-1.366l-.358-.213-3.712.974 1.014-3.598-.233-.371A9.818 9.818 0 1 1 12 21.818z"/>
      </svg>
      Enviar pelo WhatsApp
    </a>

    <button class="btn-secondary w-full justify-center" onclick={() => showInvite = false}>Fechar</button>
  </div>
</Modal>

<!-- Add member modal -->
<Modal bind:open={showAddMember} title="Adicionar Membro">
  {@const available = allPlayers.filter(p => !group?.members.some(m => m.player.id === p.id))}
  <p class="text-sm text-gray-500 dark:text-gray-400 mb-4">
    Selecione um jogador cadastrado no sistema que ainda não faça parte deste grupo.
  </p>
  {#if available.length === 0}
    <div class="text-center py-6 text-gray-400 dark:text-gray-500 text-sm">
      <UserPlus size={32} class="mx-auto mb-2 opacity-40" />
      <p>Todos os jogadores cadastrados já fazem parte deste grupo.</p>
    </div>
    <div class="flex justify-end mt-4">
      <button class="btn-secondary" onclick={() => showAddMember = false}>Fechar</button>
    </div>
  {:else}
    <form onsubmit={(e) => { e.preventDefault(); addMember(); }} class="space-y-4">
      <div class="form-group">
        <label class="label" for="pid">Jogador</label>
        <select id="pid" class="input" bind:value={addMemberId} required>
          <option value="">— selecione —</option>
          {#each available as p}
            <option value={p.id}>{p.name}{p.nickname ? ` (${p.nickname})` : ''}</option>
          {/each}
        </select>
      </div>
      <div class="flex gap-3 justify-end">
        <button type="button" class="btn-secondary" onclick={() => showAddMember = false}>Cancelar</button>
        <button type="submit" class="btn-primary" disabled={saving}>{saving ? 'Adicionando…' : 'Adicionar'}</button>
      </div>
    </form>
  {/if}
</Modal>

<!-- Member edit bottom sheet -->
{#if roleEditMember}
  <button class="fixed inset-0 z-40 bg-black/40" onclick={() => roleEditMember = null} aria-label="Cancelar" />
  <div class="fixed z-50 bottom-0 inset-x-0 sm:inset-0 sm:flex sm:items-center sm:justify-center sm:p-4 pointer-events-none">
    <div class="bg-white dark:bg-gray-800 rounded-t-2xl sm:rounded-2xl shadow-xl w-full sm:max-w-sm p-6 pointer-events-auto space-y-5">
      <p class="text-gray-800 dark:text-gray-200 font-semibold text-center text-base">{roleEditMember.name}</p>

      <!-- Skill stars -->
      <div>
        <p class="text-xs font-medium text-gray-500 dark:text-gray-400 mb-2">Nível de habilidade</p>
        <div class="flex justify-center">
          <StarRating
            bind:rating={roleEditMember.skill_stars}
            size={32}
          />
        </div>
      </div>

      <!-- Goalkeeper toggle -->
      <label class="flex items-center justify-between cursor-pointer select-none">
        <span class="text-sm font-medium text-gray-700 dark:text-gray-300">Goleiro (GK)</span>
        <div class="relative">
          <input type="checkbox" class="sr-only peer" bind:checked={roleEditMember.is_goalkeeper} />
          <div class="w-10 h-6 bg-gray-200 dark:bg-gray-600 peer-checked:bg-primary-600 rounded-full transition-colors"></div>
          <div class="absolute top-0.5 left-0.5 w-5 h-5 bg-white dark:bg-gray-200 rounded-full shadow transition-transform peer-checked:translate-x-4"></div>
        </div>
      </label>

      <!-- Save skill button -->
      <button
        class="btn btn-primary w-full justify-center"
        disabled={skillSaving}
        onclick={async () => {
          await saveSkill(roleEditMember!.id, roleEditMember!.skill_stars, roleEditMember!.is_goalkeeper);
          roleEditMember = null;
        }}>
        {skillSaving ? 'Salvando…' : 'Salvar habilidade'}
      </button>

      <div class="border-t border-gray-100 dark:border-gray-700 pt-3 flex flex-col gap-2">
        <button
          class="btn btn-secondary justify-center py-2.5"
          onclick={() => { toggleRole(roleEditMember!.id, roleEditMember!.role, roleEditMember!.name); roleEditMember = null; }}>
          {#if roleEditMember.role === 'admin'}
            <ShieldOff size={15} /> Remover presidência
          {:else}
            <ShieldCheck size={15} /> Tornar Presidente
          {/if}
        </button>
        <button class="btn btn-secondary justify-center py-2.5" onclick={() => roleEditMember = null}>
          Cancelar
        </button>
      </div>
    </div>
  </div>
{/if}

<!-- Confirm dialog -->
<ConfirmDialog
  bind:open={confirmOpen}
  message={confirmMessage}
  confirmLabel={confirmLabel}
  onConfirm={confirmAction}
/>

<!-- Edit group modal -->
<Modal bind:open={showEditGroup} title="Editar Grupo">
  <form onsubmit={(e) => { e.preventDefault(); saveEditGroup(); }} class="space-y-4">
    <div class="form-group">
      <label class="label" for="egname">Nome *</label>
      <input id="egname" class="input" bind:value={editForm.name} required minlength="2" maxlength="100" />
    </div>
    <div class="form-group">
      <label class="label" for="egdesc">Descrição</label>
      <textarea id="egdesc" class="input resize-none" rows="2" bind:value={editForm.description} placeholder="Opcional…"></textarea>
    </div>
    <div class="grid grid-cols-2 gap-4">
      <div class="form-group">
        <label class="label" for="egpermatch">Valor avulso (R$)</label>
        <input id="egpermatch" class="input" type="number" min="0" step="0.01"
          bind:value={editForm.per_match_amount} placeholder="Ex: 25,00" />
        <p class="text-xs text-gray-400 dark:text-gray-500 mt-0.5">Deixe vazio se não cobrar por partida</p>
      </div>
      <div class="form-group">
        <label class="label" for="egmonthly">Mensalidade (R$)</label>
        <input id="egmonthly" class="input" type="number" min="0" step="0.01"
          bind:value={editForm.monthly_amount} placeholder="Ex: 75,00" />
        <p class="text-xs text-gray-400 dark:text-gray-500 mt-0.5">Deixe vazio se não cobrar mensalidade</p>
      </div>
    </div>
    <div class="form-group">
      <label class="flex items-center gap-3 cursor-pointer select-none">
        <div class="relative">
          <input type="checkbox" class="sr-only peer" bind:checked={editForm.recurrence_enabled} />
          <div class="w-10 h-6 bg-gray-200 dark:bg-gray-600 peer-checked:bg-primary-600 rounded-full transition-colors"></div>
          <div class="absolute top-0.5 left-0.5 w-5 h-5 bg-white dark:bg-gray-200 rounded-full shadow transition-transform peer-checked:translate-x-4"></div>
        </div>
        <span class="text-sm font-medium text-gray-700 dark:text-gray-300">Recorrência semanal</span>
      </label>
      <p class="text-xs text-gray-400 dark:text-gray-500 mt-1">
        Quando ativa, uma nova partida é criada automaticamente após o encerramento da atual, herdando os convidados com presença pendente.
      </p>
    </div>
    <div class="border-t border-gray-100 dark:border-gray-700 pt-4">
      <p class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">Configurações de votação</p>
      <div class="space-y-3">
        <div class="form-group">
          <label class="label" for="egvotedelay">Abertura da votação após o término</label>
          <select id="egvotedelay" class="input" bind:value={editForm.vote_open_delay_minutes}>
            <option value={0}>Imediato (sem atraso)</option>
            <option value={10}>10 minutos</option>
            <option value={20}>20 minutos (padrão)</option>
            <option value={30}>30 minutos</option>
            <option value={60}>1 hora</option>
          </select>
        </div>
        <div class="form-group">
          <label class="label" for="egvotedur">Duração da votação</label>
          <select id="egvotedur" class="input" bind:value={editForm.vote_duration_hours}>
            <option value={2}>2 horas</option>
            <option value={4}>4 horas</option>
            <option value={6}>6 horas</option>
            <option value={12}>12 horas</option>
            <option value={24}>24 horas (padrão)</option>
            <option value={48}>48 horas</option>
            <option value={72}>72 horas</option>
          </select>
        </div>
      </div>
    </div>
    <div class="flex gap-3 justify-end pt-2">
      <button type="button" class="btn-secondary" onclick={() => showEditGroup = false}>Cancelar</button>
      <button type="submit" class="btn-primary" disabled={saving}>{saving ? 'Salvando…' : 'Salvar'}</button>
    </div>
  </form>
</Modal>
