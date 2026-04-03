<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import { goto } from '$app/navigation';
  import { groups as groupsApi, matches as matchesApi, invites, votes as votesApi, finance as financeApi, ApiError } from '$lib/api';
  import type { GroupDetail, GroupMember, Match, MatchDetail, VoteStatusResponse, PlayerStatItem, FinancePeriod, FinancePayment, WaitlistEntry } from '$lib/api';
  import { currentPlayer, isAdmin, isLoggedIn } from '$lib/stores/auth';
  import { toastSuccess, toastError, toastInfo } from '$lib/stores/toast';
  import Modal from '$lib/components/Modal.svelte';
  import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
  import DatePicker from '$lib/components/DatePicker.svelte';
  import TimePicker from '$lib/components/TimePicker.svelte';
  import { Plus, Calendar, Users, Link, Trash2, Clock, MapPin, Copy, UserPlus, ChevronRight, ShieldCheck, ShieldOff, Pencil, Wallet, CheckCircle2, Circle, Globe, Lock } from 'lucide-svelte';
  import PageBackground from '$lib/components/PageBackground.svelte';
  import StarRating from '$lib/components/StarRating.svelte';
  import AvatarImage from '$lib/components/AvatarImage.svelte';
  import PositionSelector from '$lib/components/PositionSelector.svelte';
  import { POS_ABBR, POS_COLOR_CLASSES } from '$lib/team-builder';
  import type { Position } from '$lib/team-builder';
  import WaitlistModal from '$lib/components/WaitlistModal.svelte';
  import WaitlistPanel from '$lib/components/WaitlistPanel.svelte';
  import AddMemberModal from '$lib/components/AddMemberModal.svelte';
  import { relativeDate, playerDisplayName } from '$lib/utils.js';
  import { t, locale } from '$lib/i18n';
  import { TIMEZONE_OPTIONS, TIMEZONE_GROUPS } from '$lib/timezones';

  const groupId = $page.params.id;

  let group: GroupDetail | null = $state(null);
  let matchList: Match[] = $state([]);
  let loading = $state(true);
  let tab: 'upcoming' | 'past' | 'members' | 'stats' | 'finance' = $state('upcoming');

  // Finance tab
  const _monthNames = ['Janeiro','Fevereiro','Março','Abril','Maio','Junho','Julho','Agosto','Setembro','Outubro','Novembro','Dezembro'];
  let financeYear = $state(new Date().getFullYear());
  let financeMonth = $state(new Date().getMonth() + 1);
  let financePeriod = $state<FinancePeriod | null>(null);
  let financeLoading = $state(false);
  let financeError = $state('');
  let paymentTarget = $state<FinancePayment | null>(null);
  let showPaymentSheet = $state(false);
  let markingPayment = $state(false);

  $effect(() => {
    if (tab !== 'finance') return;
    void financeYear; void financeMonth;
    let cancelled = false;
    financePeriod = null;
    financeLoading = true;
    financeError = '';
    financeApi.getPeriod(groupId, financeYear, financeMonth)
      .then(r => { if (!cancelled) financePeriod = r; })
      .catch(() => { if (!cancelled) financeError = 'Erro ao carregar dados financeiros.'; })
      .finally(() => { if (!cancelled) financeLoading = false; });
    return () => { cancelled = true; };
  });

  function fmtCents(cents: number): string {
    return `R$ ${(cents / 100).toFixed(2).replace('.', ',')}`;
  }

  function openPaymentSheet(p: FinancePayment) {
    paymentTarget = p;
    showPaymentSheet = true;
  }

  async function markPaid(type: 'monthly' | 'per_match') {
    if (!paymentTarget) return;
    markingPayment = true;
    try {
      const updated = await financeApi.markPayment(paymentTarget.id, { status: 'paid', payment_type: type });
      financePeriod = financePeriod ? {
        ...financePeriod,
        payments: financePeriod.payments.map(p => p.id === updated.id ? updated : p),
      } : null;
      if (financePeriod) financePeriod = { ...financePeriod, summary: recalcSummary(financePeriod.payments) };
      showPaymentSheet = false;
    } catch { toastError($t('group.finance_mark_error')); }
    markingPayment = false;
  }

  async function markPending(p: FinancePayment) {
    try {
      const updated = await financeApi.markPayment(p.id, { status: 'pending' });
      financePeriod = financePeriod ? {
        ...financePeriod,
        payments: financePeriod.payments.map(x => x.id === updated.id ? updated : x),
      } : null;
      if (financePeriod) financePeriod = { ...financePeriod, summary: recalcSummary(financePeriod.payments) };
    } catch { toastError($t('group.finance_undo_error')); }
  }

  function recalcSummary(payments: FinancePayment[]) {
    const active = payments.filter(p => p.status !== 'excluded');
    const paid = active.filter(p => p.status === 'paid');
    const pending = active.filter(p => p.status === 'pending');
    const received = paid.reduce((s, p) => s + (p.amount_due ?? 0), 0);
    return {
      received_cents: received,
      paid_count: paid.length,
      pending_count: pending.length,
      total_members: active.length,
      compliance_pct: active.length > 0 ? Math.round(paid.length / active.length * 100) : 0,
    };
  }

  function prevMonth() {
    if (financeMonth === 1) { financeMonth = 12; financeYear--; }
    else financeMonth--;
  }
  function nextMonth() {
    const now = new Date();
    if (financeYear === now.getFullYear() && financeMonth === now.getMonth() + 1) return;
    if (financeMonth === 12) { financeMonth = 1; financeYear++; }
    else financeMonth++;
  }
  function isCurrentMonth() {
    const now = new Date();
    return financeYear === now.getFullYear() && financeMonth === now.getMonth() + 1;
  }

  // Modals
  let showMatch = $state(false);
  let showEditMatch = $state(false);
  let showInvite = $state(false);
  let showEditGroup = $state(false);
  let showAddMemberByPhone = $state(false);

  let inviteLink = $state('');
  let inviteQr = $state('');
  let courtLabels = $derived<Record<string, string>>({
    campo: $t('group.court_campo'),
    sintetico: $t('group.court_sintetico'),
    terrao: $t('group.court_terrao'),
    quadra: $t('group.court_quadra'),
  });
  let matchForm = $state({ match_date: '', start_time: '20:30', end_time: '', location: '', address: '', court_type: '', players_per_team: '', max_players: '', notes: '' });
  let editMatchForm = $state({ match_date: '', start_time: '', end_time: '', location: '', address: '', court_type: '', players_per_team: '', max_players: '', notes: '', status: 'open' });
  let editingMatch: Match | null = $state(null);
  let saving = $state(false);


  let editForm = $state({ name: '', description: '', per_match_amount: '', monthly_amount: '', recurrence_enabled: false, is_public: true, vote_open_delay_minutes: 20, vote_duration_hours: 24, timezone: 'America/Sao_Paulo' });

  // Waitlist
  let waitlistEntries = $state<WaitlistEntry[]>([]);
  let myWaitlistEntry = $state<WaitlistEntry | null>(null);
  let acceptingWaitlist = $state<string | null>(null);
  let rejectingWaitlist = $state<string | null>(null);
  let showWaitlistModal = $state(false);
  let submittingWaitlist = $state(false);

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
      .sort((a, b) => playerDisplayName(a.player.name, a.player.nickname).localeCompare(playerDisplayName(b.player.name, b.player.nickname), 'pt-BR', { sensitivity: 'base' }))
  );
  let roleEditMember = $state<{ id: string; name: string; role: string; skill_stars: number; position: string } | null>(null);
  let skillSaving = $state(false);
  let selectedMember = $state<GroupMember | null>(null);
  let showMemberDetail = $state(false);

  const today = new Date().toISOString().slice(0, 10);
  function matchSortKey(m: { match_date: string; start_time: string }) {
    return `${m.match_date}T${m.start_time}`;
  }
  let upcomingMatches = $derived(matchList.filter(m => m.status === 'open' || m.status === 'in_progress').sort((a, b) => matchSortKey(a).localeCompare(matchSortKey(b))));
  let pastMatches = $derived(matchList.filter(m => m.status === 'closed').sort((a, b) => matchSortKey(b).localeCompare(matchSortKey(a))));

  let confirmOpen = $state(false);
  let confirmMessage = $state('');
  let confirmLabel = $state($t('group.cancel'));
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
    if (perMatch != null) parts.push(`R$ ${Number(perMatch).toFixed(2).replace('.', ',')} ${$t('group.label_per_match')}`);
    if (monthly != null) parts.push(`R$ ${Number(monthly).toFixed(2).replace('.', ',')} ${$t('group.label_monthly')}`);
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
      is_public: group.is_public,
      vote_open_delay_minutes: group.vote_open_delay_minutes ?? 20,
      vote_duration_hours: group.vote_duration_hours ?? 24,
      timezone: group.timezone ?? 'America/Sao_Paulo',
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
        is_public: editForm.is_public,
        vote_open_delay_minutes: editForm.vote_open_delay_minutes,
        vote_duration_hours: editForm.vote_duration_hours,
        timezone: editForm.timezone,
      });
      group = await groupsApi.get(groupId);
      showEditGroup = false;
      toastSuccess($t('group.group_updated'));
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

  // Polling: atualiza lista de rachões a cada 60s.
  onMount(() => {
    function refresh() {
      if (document.visibilityState !== 'visible') return;
      if (tab !== 'upcoming') return;
      if (!matchList.some(m => m.status === 'open' || m.status === 'in_progress')) return;
      matchesApi.list(groupId).then(ms => { matchList = ms; }).catch(() => {});
    }
    const timer = setInterval(refresh, 60_000);
    document.addEventListener('visibilitychange', refresh);
    return () => {
      clearInterval(timer);
      document.removeEventListener('visibilitychange', refresh);
    };
  });

  function isGroupAdmin() {
    if ($isAdmin) return true;
    return group?.members.some(m => m.player.id === $currentPlayer?.id && m.role === 'admin') ?? false;
  }

  function isGroupMember() {
    if ($isAdmin) return true;
    return group?.members.some(m => m.player.id === $currentPlayer?.id) ?? false;
  }

  // Load waitlist data when group is loaded
  $effect(() => {
    const g = group;
    if (!g || !$isLoggedIn) return;
    if (isGroupAdmin()) {
      groupsApi.getWaitlist(groupId)
        .then(entries => { waitlistEntries = entries; })
        .catch(() => {});
    } else if (!isGroupMember()) {
      groupsApi.getMyWaitlistEntry(groupId)
        .then(entry => { myWaitlistEntry = entry; })
        .catch(() => {});
    }
  });

  async function acceptWaitlist(entryId: string) {
    acceptingWaitlist = entryId;
    try {
      await groupsApi.reviewWaitlist(groupId, entryId, 'accept');
      waitlistEntries = waitlistEntries.filter(e => e.id !== entryId);
      const [g, ms] = await Promise.all([groupsApi.get(groupId), matchesApi.list(groupId)]);
      group = g;
      matchList = ms;
      toastSuccess($t('group.accept_waitlist_success'));
    } catch (e) {
      toastError(e instanceof ApiError ? e.message : 'Erro ao aceitar candidato');
    }
    acceptingWaitlist = null;
  }

  async function rejectWaitlist(entryId: string) {
    rejectingWaitlist = entryId;
    try {
      await groupsApi.reviewWaitlist(groupId, entryId, 'reject');
      waitlistEntries = waitlistEntries.filter(e => e.id !== entryId);
      toastSuccess($t('group.reject_waitlist_success'));
    } catch (e) {
      toastError(e instanceof ApiError ? e.message : 'Erro ao rejeitar candidato');
    }
    rejectingWaitlist = null;
  }

  async function submitWaitlist(data: { agreed: boolean; intro: string }) {
    submittingWaitlist = true;
    try {
      const entry = await groupsApi.joinWaitlist(groupId, { agreed: data.agreed, intro: data.intro || undefined });
      myWaitlistEntry = entry;
      showWaitlistModal = false;
      toastSuccess($t('group.waitlist_submitted'));
    } catch (e) {
      toastError(e instanceof ApiError ? e.message : 'Erro ao enviar candidatura');
    }
    submittingWaitlist = false;
  }

  // Get the active match for waitlist modal
  let activeMatchDetail = $state<MatchDetail | null>(null);

  async function openWaitlistModal() {
    const m = upcomingMatches[0];
    if (!m) return;
    try {
      activeMatchDetail = await matchesApi.getByHash(m.hash);
    } catch { activeMatchDetail = null; }
    showWaitlistModal = true;
  }

  async function createMatch() {
    saving = true;
    try {
      const m = await matchesApi.create(groupId, matchForm);
      matchList = [m, ...matchList];
      showMatch = false;
      matchForm = { match_date: '', start_time: '20:30', end_time: '', location: '', address: '', court_type: '', players_per_team: '', max_players: '', notes: '' };
      toastSuccess($t('group.create_match_success'));
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
    toastInfo($t('group.link_copied'));
  }

  async function onMemberAdded() {
    group = await groupsApi.get(groupId);
  }

  async function toggleRole(playerId: string, currentRole: string, name: string) {
    const newRole = currentRole === 'admin' ? 'member' : 'admin';
    const actionLabel = newRole === 'admin' ? $t('group.make_president_label') : $t('group.remove_president_label');
    const msg = newRole === 'admin'
      ? $t('group.make_president_confirm').replace('{name}', name)
      : $t('group.remove_president_confirm').replace('{name}', name);
    askConfirm(msg, actionLabel, async () => {
      try {
        await groupsApi.updateMemberRole(groupId, playerId, newRole);
        group = await groupsApi.get(groupId);
        toastSuccess(newRole === 'admin' ? `${name} agora é presidente do grupo` : `${name} voltou a ser membro`);
      } catch (e) { toastError(e instanceof ApiError ? e.message : 'Erro ao alterar papel'); }
    });
  }

  async function saveSkill(playerId: string, skill_stars: number, position: string) {
    skillSaving = true;
    try {
      await groupsApi.updateMemberSkill(groupId, playerId, { skill_stars, position });
      group = await groupsApi.get(groupId);
    } catch (e) { toastError(e instanceof ApiError ? e.message : 'Erro ao salvar'); }
    skillSaving = false;
  }

  async function removeMember(playerId: string, name: string) {
    askConfirm($t('group.remove_member_confirm').replace('{name}', name), $t('group.remove_member_label'), async () => {
      try {
        await groupsApi.removeMember(groupId, playerId);
        group = await groupsApi.get(groupId);
        toastSuccess($t('group.remove_member_success'));
      } catch (e) { toastError($t('group.remove_member_error')); }
    });
  }

  async function deleteMatch(m: Match) {
    askConfirm($t('group.delete_match_confirm'), $t('group.delete_match_label'), async () => {
      try {
        await matchesApi.delete(groupId, m.id);
        matchList = matchList.filter(x => x.id !== m.id);
        toastSuccess($t('group.delete_match_success'));
      } catch (e) { toastError('Erro ao excluir'); }
    });
  }

  function fmtDate(d: string) {
    const s = relativeDate(d, { weekday: 'long', day: '2-digit', month: 'long' }, $locale, {
      today: $t('date.today'),
      tomorrow: $t('date.tomorrow'),
      yesterday: $t('date.yesterday'),
    });
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
      toastSuccess($t('group.edit_match_success'));
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
            <button class="btn-secondary btn-sm" onclick={openEditGroup}><Pencil size={14} /> {$t('group.edit')}</button>
          </div>
        {/if}
      </div>
      <div class="flex flex-wrap items-center gap-x-4 gap-y-1">
        <span class="text-xs text-gray-300 flex items-center gap-1">
          <Users size={12} /> {nonAdminMembers.length === 1 ? $t('group.player_count_one').replace('{n}', '1') : $t('group.player_count_other').replace('{n}', String(nonAdminMembers.length))}
        </span>
        {#if group.per_match_amount != null || group.monthly_amount != null}
          {#each fmtPricingParts(group.per_match_amount, group.monthly_amount) as part}
            <span class="text-xs text-amber-700 dark:text-amber-400 font-medium">{part}</span>
          {/each}
        {:else}
          <span class="text-xs text-green-600 dark:text-green-400">{$t('group.free')}</span>
        {/if}
        {#if group.recurrence_enabled}
          <span class="text-xs text-primary-600 dark:text-primary-400">{$t('group.weekly_recurrence')}</span>
        {/if}
        <span class="text-xs flex items-center gap-1 {group.is_public ? 'text-green-400' : 'text-gray-400'}">
          {#if group.is_public}<Globe size={11} /> {$t('group.public')}{:else}<Lock size={11} /> {$t('group.private')}{/if}
        </span>
        {#if isGroupAdmin()}
          <div class="flex items-center gap-1.5 text-xs text-gray-300 bg-white/10 rounded-lg px-2.5 py-1 w-full sm:w-auto">
            <span>🗳️</span>
            <span class="text-gray-400">{$t('group.vote_settings')}</span>
            <span class="text-white/30">·</span>
            <span>
              {group.vote_open_delay_minutes === 0 ? $t('group.vote_immediate') : $t('group.vote_delay').replace('{n}', String(group.vote_open_delay_minutes))}
            </span>
            <span class="text-white/30">·</span>
            <span>{$t('group.vote_closes').replace('{n}', String(group.vote_duration_hours))}</span>
            <span class="text-white/30">·</span>
            <span>{$t('group.vote_after_match')}</span>
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
              <p class="text-sm font-semibold text-amber-800 dark:text-amber-200">{$t('group.vote_banner').replace('{number}', String(m.number))}</p>
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
        {$t('group.tab_upcoming').replace('{n}', String(upcomingMatches.length))}
      </button>
      {#if pastMatches.length > 0}
        <button
          class="px-3 py-2 text-sm font-medium border-b-2 transition-colors whitespace-nowrap {tab === 'past' ? 'border-primary-400 text-primary-400' : 'border-transparent text-gray-300 hover:text-white'}"
          onclick={() => tab = 'past'}>
          {$t('group.tab_past').replace('{n}', String(pastMatches.length))}
        </button>
      {/if}
      <button
        class="px-3 py-2 text-sm font-medium border-b-2 transition-colors whitespace-nowrap {tab === 'members' ? 'border-primary-400 text-primary-400' : 'border-transparent text-gray-300 hover:text-white'}"
        onclick={() => tab = 'members'}>
        {$t('group.tab_members').replace('{n}', String(nonAdminMembers.length))}
      </button>
      <button
        class="px-3 py-2 text-sm font-medium border-b-2 transition-colors whitespace-nowrap {tab === 'stats' ? 'border-primary-400 text-primary-400' : 'border-transparent text-gray-300 hover:text-white'}"
        onclick={() => { tab = 'stats'; }}>
        {$t('group.tab_stats')}
      </button>
      {#if group?.monthly_amount || group?.per_match_amount}
      <button
        class="px-3 py-2 text-sm font-medium border-b-2 transition-colors whitespace-nowrap {tab === 'finance' ? 'border-primary-400 text-primary-400' : 'border-transparent text-gray-300 hover:text-white'}"
        onclick={() => { tab = 'finance'; }}>
        {$t('group.tab_finance')}
      </button>
      {/if}
    </div>

    <!-- Próximos / Últimos tabs -->
    {#if tab === 'upcoming' || tab === 'past'}
      {#if isGroupAdmin() && tab === 'upcoming'}
        <div class="flex justify-end mb-4">
          <button class="btn-primary btn-sm" onclick={() => showMatch = true}><Plus size={14} /> {$t('group.new_match')}</button>
        </div>
      {/if}
      {@const subList = tab === 'upcoming' ? upcomingMatches : pastMatches}
      {#if subList.length === 0}
        <div class="card p-12 text-center">
          <Calendar size={40} class="text-gray-300 mx-auto mb-3" />
          <p class="text-gray-500">{tab === 'upcoming' ? $t('group.no_upcoming') : $t('group.no_past')}</p>
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
                      {$t('group.status_in_progress')}
                    </span>
                  {:else}
                    <span class="badge {m.status === 'open' ? 'badge-green' : 'badge-gray'} shrink-0">
                      {m.status === 'open' ? $t('group.status_open') : $t('group.status_closed')}
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
                      m.court_type ? courtLabels[m.court_type] : null,
                      m.players_per_team ? $t('group.line_players').replace('{n}', String(m.players_per_team)) : null,
                      m.max_players ? `máx. ${m.max_players}` : null,
                      ...fmtPricingParts(group.per_match_amount, group.monthly_amount),
                    ].filter(Boolean).join(' · ')}
                  </p>
                {/if}

                <!-- Actions -->
                <div class="flex items-center gap-2 mt-3 pt-3 border-t border-gray-100 dark:border-gray-700">
                  <a href="/match/{m.hash}" class="btn-sm btn-secondary shrink-0">
                    {$t('group.details')} <ChevronRight size={14} />
                  </a>
                  {#if isGroupAdmin()}
                    <button onclick={() => openEditMatch(m)} class="btn-sm btn-ghost shrink-0">
                      <Pencil size={14} /> {$t('group.edit_btn')}
                    </button>
                    <button onclick={() => deleteMatch(m)} class="btn-sm btn-ghost text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20 shrink-0">
                      <Trash2 size={14} /> {$t('group.delete_btn')}
                    </button>
                  {/if}
                </div>
              </div>
            </div>
          {/each}
        </div>
      {/if}
    {/if}

    <!-- Waitlist panel (admin) -->
    {#if tab === 'upcoming' && isGroupAdmin() && group.is_public && upcomingMatches.length > 0}
      <div class="mt-4">
        <WaitlistPanel
          entries={waitlistEntries}
          accepting={acceptingWaitlist}
          rejecting={rejectingWaitlist}
          onaccept={acceptWaitlist}
          onreject={rejectWaitlist}
        />
      </div>
    {/if}

    <!-- Join waitlist (non-member, logged in, public group, active match) -->
    {#if tab === 'upcoming' && $isLoggedIn && !$isAdmin && group.is_public && !isGroupMember() && upcomingMatches.length > 0}
      <div class="card mt-4 p-4">
        {#if myWaitlistEntry}
          <div class="flex items-center gap-2 text-sm text-amber-700 dark:text-amber-300">
            <span>⏳</span>
            <span class="font-medium">{$t('group.waitlist_pending')}</span>
          </div>
        {:else}
          <p class="text-sm text-gray-600 dark:text-gray-400 mb-3">{$t('group.not_member')}</p>
          <button
            onclick={openWaitlistModal}
            class="btn btn-primary w-full justify-center gap-2">
            <UserPlus size={15} /> {$t('group.want_to_play')}
          </button>
        {/if}
      </div>
    {/if}

    <!-- Stats tab -->
    {#if tab === 'stats'}
      <!-- Seletor de período -->
      <div class="flex flex-wrap items-center gap-2 mb-4">
        <div class="flex gap-1">
          <button
            class="px-3 py-1.5 rounded-full text-xs font-medium transition-colors {statsPeriod === 'annual' ? 'bg-primary-500 text-white' : 'bg-white/10 text-gray-300 hover:bg-white/20'}"
            onclick={() => { statsPeriod = 'annual'; }}>
            {$t('group.stats_annual')}
          </button>
          <button
            class="px-3 py-1.5 rounded-full text-xs font-medium transition-colors {statsPeriod === 'monthly' ? 'bg-primary-500 text-white' : 'bg-white/10 text-gray-300 hover:bg-white/20'}"
            onclick={() => { statsPeriod = 'monthly'; }}>
            {$t('group.stats_monthly')}
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
          <p class="text-gray-400 dark:text-gray-500 text-sm">{$t('group.stats_empty')}</p>
        </div>
      {:else}
        <div class="card overflow-x-auto">
          <table class="w-full text-sm">
            <thead>
              <tr class="border-b border-gray-100 dark:border-gray-700 text-xs text-gray-500 dark:text-gray-400">
                <th class="px-4 py-2 text-left w-8">#</th>
                <th class="px-4 py-2 text-left">{$t('group.stats_player')}</th>
                <th class="px-4 py-2 text-right">{$t('group.stats_pts')}</th>
                <th class="px-4 py-2 text-right hidden sm:table-cell">{$t('group.stats_flops')}</th>
                <th class="px-4 py-2 text-right hidden sm:table-cell">{$t('group.stats_hours')}</th>
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
          {$t('group.stats_footer').replace('{period}', statsPeriodLabel)}
        </p>
      {/if}
    {/if}

    <!-- Members tab -->
    {#if tab === 'members'}
      {#if isGroupAdmin()}
        <div class="flex justify-end gap-2 mb-4">
          <button class="btn-secondary btn-sm" onclick={generateInvite}><Link size={14} /> {$t('group.invite_btn')}</button>
          <button class="btn-secondary btn-sm" onclick={() => showAddMemberByPhone = true}><UserPlus size={14} /> {$t('group.add_member_btn')}</button>
        </div>
      {/if}
      <div class="card overflow-hidden divide-y divide-gray-100 dark:divide-gray-700">
        {#if nonAdminMembers.length === 0}
          <div class="px-6 py-10 text-center text-gray-400 dark:text-gray-500 text-sm">
            <Users size={32} class="mx-auto mb-2 opacity-40" />
            <p>{$t('group.no_players')}</p>
            {#if isGroupAdmin()}
              <button class="btn-primary mt-4 btn-sm" onclick={() => showAddMemberByPhone = true}><UserPlus size={14} /> {$t('group.add_player')}</button>
            {/if}
          </div>
        {/if}
        {#each nonAdminMembers as m}
          <div class="flex items-center gap-2 px-4 py-3">
            <AvatarImage name={m.player.name} avatarUrl={m.player.avatar_url} size={36} class="shrink-0" />
            <!-- Info -->
            <div class="flex-1 min-w-0">
              <p class="font-medium text-sm text-gray-900 dark:text-gray-100 truncate">
                {playerDisplayName(m.player.name, m.player.nickname)}
              </p>
              {#if isGroupAdmin() && m.skill_stars != null}
                <div class="mt-0.5">
                  <StarRating rating={m.skill_stars} readonly size={13} />
                </div>
              {/if}
              {#if m.role === 'admin' || m.position}
                <div class="flex items-center gap-1 mt-0.5 flex-wrap">
                  {#if m.role === 'admin'}
                    <span class="inline-flex items-center px-1 py-px rounded text-[10px] font-semibold bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400">{$t('group.role_president')}</span>
                  {/if}
                  {#if m.position}
                    {@const pos = ({gk:'goalkeeper',zag:'defender',lat:'fullback',mei:'midfielder',ata:'forward'} as Record<string,Position>)[m.position]}
                    {#if pos}
                      <span class="inline-flex items-center px-1 py-px rounded text-[10px] font-bold {POS_COLOR_CLASSES[pos]}">{POS_ABBR[pos]}</span>
                    {/if}
                  {/if}
                </div>
              {/if}
            </div>
            <!-- Details button -->
            {#if isGroupAdmin()}
              <button
                onclick={() => { selectedMember = m; showMemberDetail = true; }}
                class="text-xs px-2 py-1 rounded border border-gray-200 text-gray-600 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-400 dark:hover:bg-gray-700 flex items-center gap-1 shrink-0">
                <ChevronRight size={12} /> {$t('group.member_details')}
              </button>
            {/if}
          </div>
        {/each}
      </div>
    {/if}

    <!-- Finance tab -->
    {#if tab === 'finance'}
      <!-- Nav de meses -->
      <div class="flex items-center justify-between mb-4">
        <button onclick={prevMonth} class="p-2 rounded-lg text-gray-300 hover:text-white hover:bg-white/10 transition-colors">←</button>
        <span class="text-sm font-semibold text-white">
          {_monthNames[financeMonth - 1]} {financeYear}
        </span>
        <button onclick={nextMonth} disabled={isCurrentMonth()}
          class="p-2 rounded-lg text-gray-300 hover:text-white hover:bg-white/10 transition-colors disabled:opacity-30 disabled:cursor-not-allowed">→</button>
      </div>

      {#if financeLoading}
        <div class="animate-pulse space-y-3">
          <div class="h-20 bg-white/10 rounded-xl"></div>
          <div class="h-40 bg-white/10 rounded-xl"></div>
        </div>
      {:else if financeError}
        <div class="card p-6 text-center text-red-400">{financeError}</div>
      {:else if financePeriod}
        <!-- Summary cards -->
        <div class="grid grid-cols-2 sm:grid-cols-4 gap-3 mb-5">
          <div class="card p-4 text-center">
            <p class="text-xs text-gray-400 mb-1">{$t('group.finance_received')}</p>
            <p class="text-lg font-bold text-green-500">{fmtCents(financePeriod.summary.received_cents)}</p>
          </div>
          <div class="card p-4 text-center">
            <p class="text-xs text-gray-400 mb-1">{$t('group.finance_pending')}</p>
            <p class="text-lg font-bold text-amber-400">{financePeriod.summary.pending_count !== 1 ? $t('group.finance_pending_players_plural').replace('{n}', String(financePeriod.summary.pending_count)) : $t('group.finance_pending_players').replace('{n}', '1')}</p>
          </div>
          <div class="card p-4 text-center">
            <p class="text-xs text-gray-400 mb-1">{$t('group.finance_paid')}</p>
            <p class="text-lg font-bold text-white">{financePeriod.summary.paid_count}/{financePeriod.summary.total_members}</p>
          </div>
          <div class="card p-4 text-center">
            <p class="text-xs text-gray-400 mb-1">{$t('group.finance_compliance')}</p>
            <p class="text-lg font-bold {financePeriod.summary.compliance_pct >= 80 ? 'text-green-400' : financePeriod.summary.compliance_pct >= 50 ? 'text-amber-400' : 'text-red-400'}">
              {financePeriod.summary.compliance_pct}%
            </p>
          </div>
        </div>

        <!-- Payment list -->
        {#if financePeriod.payments.length === 0}
          <div class="card p-10 text-center">
            <Wallet size={32} class="text-gray-400 mx-auto mb-2" />
            <p class="text-sm text-gray-400">{$t('group.finance_no_players')}</p>
          </div>
        {:else}
          {@const pendingList = financePeriod.payments.filter(p => p.status === 'pending')}
          {@const paidList = financePeriod.payments.filter(p => p.status === 'paid')}

          <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
            <!-- Pendentes -->
            <div class="card overflow-hidden divide-y divide-gray-100 dark:divide-gray-700">
              <div class="px-4 py-2 bg-amber-50/5">
                <span class="text-xs font-semibold text-amber-400 uppercase tracking-wide">{$t('group.finance_pending_section').replace('{n}', String(pendingList.length))}</span>
              </div>
              {#if pendingList.length === 0}
                <div class="px-4 py-6 text-center text-xs text-gray-400">{$t('group.finance_none_pending')}</div>
              {:else}
                {#each pendingList as p (p.id)}
                  <div class="flex items-center gap-3 px-4 py-3">
                    <Circle size={18} class="text-amber-400 shrink-0" />
                    <span class="flex-1 text-sm text-gray-900 dark:text-gray-100">{p.player_name}</span>
                    {#if isGroupAdmin()}
                      <button onclick={() => openPaymentSheet(p)}
                        class="btn-sm btn-primary py-1 text-xs">
                        {$t('group.finance_mark_paid')}
                      </button>
                    {/if}
                  </div>
                {/each}
              {/if}
            </div>

            <!-- Pagos -->
            <div class="card overflow-hidden divide-y divide-gray-100 dark:divide-gray-700">
              <div class="px-4 py-2 bg-green-50/5">
                <span class="text-xs font-semibold text-green-400 uppercase tracking-wide">{$t('group.finance_paid_section').replace('{n}', String(paidList.length))}</span>
              </div>
              {#if paidList.length === 0}
                <div class="px-4 py-6 text-center text-xs text-gray-400">{$t('group.finance_none_paid')}</div>
              {:else}
                {#each paidList as p (p.id)}
                  <div class="flex items-center gap-3 px-4 py-3">
                    <CheckCircle2 size={18} class="text-green-500 shrink-0" />
                    <div class="flex-1 min-w-0">
                      <span class="text-sm text-gray-900 dark:text-gray-100">{p.player_name}</span>
                      <span class="ml-2 text-xs text-gray-400">
                        {p.payment_type === 'monthly' ? $t('group.finance_monthly') : $t('group.finance_per_match')}
                        {p.amount_due != null ? `· ${fmtCents(p.amount_due)}` : ''}
                      </span>
                    </div>
                    {#if isGroupAdmin()}
                      <button onclick={() => markPending(p)}
                        class="text-xs text-gray-400 hover:text-red-400 transition-colors px-2 py-1">
                        {$t('group.finance_undo')}
                      </button>
                    {/if}
                  </div>
                {/each}
              {/if}
            </div>
          </div>
        {/if}
      {/if}
    {/if}
  {/if}
</main>
</PageBackground>

<!-- Payment type bottom sheet -->
{#if showPaymentSheet && paymentTarget}
  <button class="fixed inset-0 z-40 bg-black/50" onclick={() => showPaymentSheet = false} aria-label={$t('aria.close')} type="button"></button>
  <div class="fixed z-50 left-0 right-0 bottom-0 sm:inset-0 sm:flex sm:items-center sm:justify-center pointer-events-none">
    <div class="bg-white dark:bg-gray-800 rounded-t-2xl sm:rounded-2xl shadow-2xl w-full sm:max-w-xs pointer-events-auto">
      <div class="px-5 pt-5 pb-3">
        <h2 class="font-semibold text-gray-900 dark:text-gray-100 text-base">{paymentTarget.player_name}</h2>
        <p class="text-sm text-gray-500 dark:text-gray-400 mt-0.5">{$t('group.payment_type_title')}</p>
      </div>
      <div class="px-5 pb-5 space-y-3">
        <button
          onclick={() => markPaid('monthly')}
          disabled={markingPayment || !group?.monthly_amount}
          class="w-full flex items-center justify-between px-4 py-3 rounded-xl border-2 border-primary-500 bg-primary-50 dark:bg-primary-900/20 hover:bg-primary-100 dark:hover:bg-primary-900/30 transition-colors disabled:opacity-40 disabled:cursor-not-allowed">
          <span class="font-semibold text-primary-700 dark:text-primary-300 text-sm">{$t('group.payment_monthly')}</span>
          <span class="text-primary-600 dark:text-primary-400 font-bold text-sm">
            {group?.monthly_amount != null ? `R$ ${Number(group.monthly_amount).toFixed(2).replace('.', ',')}` : $t('group.finance_not_configured')}
          </span>
        </button>
        <button
          onclick={() => markPaid('per_match')}
          disabled={markingPayment || !group?.per_match_amount}
          class="w-full flex items-center justify-between px-4 py-3 rounded-xl border-2 border-gray-200 dark:border-gray-600 bg-gray-50 dark:bg-gray-700/50 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors disabled:opacity-40 disabled:cursor-not-allowed">
          <span class="font-semibold text-gray-700 dark:text-gray-300 text-sm">{$t('group.payment_per_match')}</span>
          <span class="text-gray-600 dark:text-gray-400 font-bold text-sm">
            {group?.per_match_amount != null ? `R$ ${Number(group.per_match_amount).toFixed(2).replace('.', ',')}` : $t('group.finance_not_configured')}
          </span>
        </button>
        {#if !group?.monthly_amount && !group?.per_match_amount}
          <p class="text-xs text-amber-600 dark:text-amber-400 text-center">{$t('group.finance_configure_hint')}</p>
        {/if}
        <button type="button" onclick={() => showPaymentSheet = false}
          class="w-full text-sm text-gray-400 hover:text-gray-600 dark:hover:text-gray-200 py-2 transition-colors">
          {$t('group.cancel')}
        </button>
      </div>
    </div>
  </div>
{/if}

<!-- Create match modal -->
<Modal bind:open={showMatch} title={$t('group.new_match_title')}>
  <form onsubmit={(e) => { e.preventDefault(); createMatch(); }} class="space-y-4">
    <div class="form-group">
      <label class="label" for="mdate">{$t('group.match_date_label')}</label>
      <DatePicker id="mdate" bind:value={matchForm.match_date} required />
    </div>
    <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
      <div class="form-group">
        <label class="label" for="mtime">{$t('group.match_start_label')}</label>
        <TimePicker id="mtime" bind:value={matchForm.start_time} required />
      </div>
      <div class="form-group">
        <label class="label" for="mendtime">{$t('group.match_end_label')} <span class="text-gray-400 dark:text-gray-500 font-normal">{$t('group.match_optional')}</span></label>
        <TimePicker id="mendtime" bind:value={matchForm.end_time} />
      </div>
    </div>
    <div class="form-group">
      <label class="label" for="mloc">{$t('group.match_location_label')}</label>
      <input id="mloc" class="input" bind:value={matchForm.location} placeholder={$t('group.match_location_placeholder')} required />
    </div>
    <div class="form-group">
      <label class="label" for="maddr">{$t('group.match_address_label')} <span class="text-gray-400 font-normal">{$t('group.match_address_hint')}</span></label>
      <input id="maddr" class="input" bind:value={matchForm.address} placeholder={$t('group.match_address_placeholder')} />
    </div>
    <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
      <div class="form-group">
        <label class="label" for="mcourt">{$t('group.court_type_label')}</label>
        <select id="mcourt" class="input" bind:value={matchForm.court_type}>
          <option value="">{$t('group.court_select')}</option>
          <option value="campo">{$t('group.court_campo')}</option>
          <option value="sintetico">{$t('group.court_sintetico')}</option>
          <option value="terrao">{$t('group.court_terrao')}</option>
          <option value="quadra">{$t('group.court_quadra')}</option>
        </select>
      </div>
      <div class="form-group">
        <label class="label" for="mplayers">{$t('group.players_per_team_label')} <span class="text-gray-400 font-normal">{$t('group.players_per_team_no_gk')}</span></label>
        <select id="mplayers" class="input" bind:value={matchForm.players_per_team}>
          <option value="">{$t('group.court_select')}</option>
          {#each [4, 5, 6, 7, 8, 9, 10] as n}
            <option value={n}>{$t('group.line_players').replace('{n}', String(n))}</option>
          {/each}
        </select>
      </div>
    </div>
    <div class="form-group">
      <label class="label" for="mmaxp">{$t('group.max_players_label')} <span class="text-gray-400 font-normal">{$t('group.max_players_hint')}</span></label>
      <input id="mmaxp" class="input" type="number" min="2" bind:value={matchForm.max_players} placeholder={$t('group.max_players_placeholder')} />
    </div>
    <div class="form-group">
      <label class="label" for="mnotes">{$t('group.notes_label')}</label>
      <textarea id="mnotes" class="input resize-none" rows="2" bind:value={matchForm.notes} placeholder={$t('group.notes_placeholder')}></textarea>
    </div>
    <div class="flex gap-3 justify-end pt-2">
      <button type="button" class="btn-secondary" onclick={() => showMatch = false}>{$t('group.cancel')}</button>
      <button type="submit" class="btn-primary" disabled={saving}>{saving ? $t('group.create_loading') : $t('group.create_btn')}</button>
    </div>
  </form>
</Modal>

<!-- Edit match modal -->
<Modal bind:open={showEditMatch} title={$t('group.edit_match_title')}>
  <form onsubmit={(e) => { e.preventDefault(); saveEditMatch(); }} class="space-y-4">
    <div class="form-group">
      <label class="label" for="emdate">{$t('group.match_date_label')}</label>
      <DatePicker id="emdate" bind:value={editMatchForm.match_date} required />
    </div>
    <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
      <div class="form-group">
        <label class="label" for="emtime">{$t('group.match_start_label')}</label>
        <TimePicker id="emtime" bind:value={editMatchForm.start_time} required />
      </div>
      <div class="form-group">
        <label class="label" for="emendtime">{$t('group.match_end_label')} <span class="text-gray-400 dark:text-gray-500 font-normal">{$t('group.match_optional')}</span></label>
        <TimePicker id="emendtime" bind:value={editMatchForm.end_time} />
      </div>
    </div>
    <div class="form-group">
      <label class="label" for="emloc">{$t('group.match_location_label')}</label>
      <input id="emloc" class="input" bind:value={editMatchForm.location} required />
    </div>
    <div class="form-group">
      <label class="label" for="emaddr">{$t('group.match_address_label')} <span class="text-gray-400 dark:text-gray-500 font-normal">{$t('group.match_optional')}</span></label>
      <input id="emaddr" class="input" bind:value={editMatchForm.address} placeholder={$t('group.match_address_placeholder')} />
    </div>
    <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
      <div class="form-group">
        <label class="label" for="emcourt">{$t('group.court_type_label')}</label>
        <select id="emcourt" class="input" bind:value={editMatchForm.court_type}>
          <option value="">{$t('group.court_select')}</option>
          <option value="campo">{$t('group.court_campo')}</option>
          <option value="sintetico">{$t('group.court_sintetico')}</option>
          <option value="terrao">{$t('group.court_terrao')}</option>
          <option value="quadra">{$t('group.court_quadra')}</option>
        </select>
      </div>
      <div class="form-group">
        <label class="label" for="emplayers">{$t('group.players_per_team_label')}</label>
        <select id="emplayers" class="input" bind:value={editMatchForm.players_per_team}>
          <option value="">{$t('group.court_select')}</option>
          {#each [4, 5, 6, 7, 8, 9, 10] as n}
            <option value="{n}">{$t('group.line_players').replace('{n}', String(n))}</option>
          {/each}
        </select>
      </div>
    </div>
    <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
      <div class="form-group">
        <label class="label" for="emmaxp">{$t('group.max_players_label')}</label>
        <input id="emmaxp" class="input" type="number" min="2" bind:value={editMatchForm.max_players} placeholder={$t('group.max_players_placeholder')} />
      </div>
      <div class="form-group">
        <label class="label" for="emstatus">{$t('group.match_status_label')}</label>
        <select id="emstatus" class="input" bind:value={editMatchForm.status}>
          <option value="open">{$t('group.status_open')}</option>
          <option value="in_progress">{$t('group.status_in_progress')}</option>
          <option value="closed">{$t('group.status_closed')}</option>
        </select>
      </div>
    </div>
    <div class="form-group">
      <label class="label" for="emnotes">{$t('group.notes_label')}</label>
      <textarea id="emnotes" class="input resize-none" rows="2" bind:value={editMatchForm.notes} placeholder={$t('group.notes_placeholder')}></textarea>
    </div>
    <div class="flex gap-3 justify-end pt-2">
      <button type="button" class="btn-secondary" onclick={() => showEditMatch = false}>{$t('group.cancel')}</button>
      <button type="submit" class="btn-primary" disabled={saving}>{saving ? $t('group.save_loading') : $t('group.save_btn')}</button>
    </div>
  </form>
</Modal>

<!-- Invite modal -->
<Modal bind:open={showInvite} title={$t('group.invite_modal_title')}>
  <div class="space-y-4">

    <!-- QR Code -->
    {#if inviteQr}
      <div class="flex flex-col items-center gap-2">
        <div class="bg-white rounded-2xl p-3 shadow-inner border border-gray-100 dark:border-gray-700 inline-block">
          <img src={inviteQr} alt="QR Code de convite" width="220" height="220" class="block" />
        </div>
        <p class="text-xs text-gray-500 dark:text-gray-400 text-center">
          {$t('group.invite_qr_hint')}
        </p>
      </div>
    {/if}

    <div class="alert-info text-xs">{@html $t('group.invite_expires')}</div>

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
      {$t('group.invite_whatsapp')}
    </a>

    <button class="btn-secondary w-full justify-center" onclick={() => showInvite = false}>{$t('group.invite_close')}</button>
  </div>
</Modal>

<AddMemberModal bind:open={showAddMemberByPhone} {groupId} onAdded={onMemberAdded} />

<!-- Member detail modal -->
<Modal bind:open={showMemberDetail} title={$t('group.member_detail_modal')}>
  {#if selectedMember}
    <div class="space-y-4">

      <!-- Avatar + nome -->
      <div class="flex items-center gap-3">
        <AvatarImage name={selectedMember.player.name} avatarUrl={selectedMember.player.avatar_url} size={52} />
        <div>
          <p class="font-semibold text-gray-900 dark:text-gray-100">
            {playerDisplayName(selectedMember.player.name, selectedMember.player.nickname)}
          </p>
          <p class="text-xs text-gray-400">{selectedMember.player.name}</p>
        </div>
      </div>

      <div class="grid grid-cols-2 gap-3 text-sm">
        <div>
          <p class="text-xs text-gray-400 mb-0.5">{$t('group.detail_name')}</p>
          <p class="font-medium">{selectedMember.player.name}</p>
        </div>
        <div>
          <p class="text-xs text-gray-400 mb-0.5">{$t('group.detail_nickname')}</p>
          <p class="font-medium">{selectedMember.player.nickname || '—'}</p>
        </div>
        <div>
          <p class="text-xs text-gray-400 mb-0.5">{$t('group.detail_role')}</p>
          <span class="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-semibold {selectedMember.role === 'admin' ? 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400' : 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-300'}">
            {selectedMember.role === 'admin' ? $t('group.detail_role_president') : $t('group.detail_role_member')}
          </span>
        </div>
        <div>
          <p class="text-xs text-gray-400 mb-0.5">{$t('group.detail_position')}</p>
          {#if selectedMember.position}
            {@const pos = ({gk:'goalkeeper',zag:'defender',lat:'fullback',mei:'midfielder',ata:'forward'} as Record<string,Position>)[selectedMember.position]}
            {#if pos}
              <span class="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-bold {POS_COLOR_CLASSES[pos]}">
                {POS_ABBR[pos]} — {$t(('position.' + selectedMember.position) as any)}
              </span>
            {/if}
          {:else}
            <span class="text-sm text-gray-400">—</span>
          {/if}
        </div>
        {#if selectedMember.skill_stars != null}
          <div class="col-span-2">
            <p class="text-xs text-gray-400 mb-1">{$t('group.detail_skill')}</p>
            <StarRating rating={selectedMember.skill_stars} readonly size={18} />
          </div>
        {/if}
      </div>

      <div class="border-t border-gray-100 dark:border-gray-700 pt-4 flex flex-wrap gap-2">
        <button
          onclick={() => {
            roleEditMember = { id: selectedMember!.player.id, name: selectedMember!.player.name, role: selectedMember!.role, skill_stars: selectedMember!.skill_stars ?? 2, position: selectedMember!.position ?? 'mei' };
            showMemberDetail = false;
          }}
          class="btn-sm btn-ghost flex items-center gap-1 border border-gray-200 text-gray-600 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-400">
          <Pencil size={14} /> {$t('group.edit_skill')}
        </button>
        {#if selectedMember.player.id !== $currentPlayer?.id}
          <button
            onclick={() => {
              toggleRole(selectedMember!.player.id, selectedMember!.role, selectedMember!.player.name);
              showMemberDetail = false;
            }}
            class="btn-sm btn-ghost flex items-center gap-1 border border-amber-200 text-amber-600 hover:bg-amber-50 dark:border-amber-800 dark:text-amber-400">
            {#if selectedMember.role === 'admin'}
              <ShieldOff size={14} /> {$t('group.remove_president')}
            {:else}
              <ShieldCheck size={14} /> {$t('group.make_president')}
            {/if}
          </button>
          <button
            onclick={() => {
              removeMember(selectedMember!.player.id, selectedMember!.player.name);
              showMemberDetail = false;
            }}
            class="btn-sm btn-ghost flex items-center gap-1 border border-red-200 text-red-500 hover:bg-red-50 dark:border-red-800 dark:text-red-400">
            <Trash2 size={14} /> {$t('group.remove_from_group')}
          </button>
        {/if}
      </div>
    </div>
  {/if}
</Modal>

<!-- Member edit bottom sheet -->
{#if roleEditMember}
  <button class="fixed inset-0 z-40 bg-black/40" onclick={() => roleEditMember = null} aria-label={$t('aria.close')} />
  <div class="fixed z-50 bottom-0 inset-x-0 sm:inset-0 sm:flex sm:items-center sm:justify-center sm:p-4 pointer-events-none">
    <div class="bg-white dark:bg-gray-800 rounded-t-2xl sm:rounded-2xl shadow-xl w-full sm:max-w-sm p-6 pointer-events-auto space-y-5">
      <p class="text-gray-800 dark:text-gray-200 font-semibold text-center text-base">{roleEditMember.name}</p>

      <!-- Skill stars -->
      <div>
        <p class="text-xs font-medium text-gray-500 dark:text-gray-400 mb-2">{$t('group.skill_level')}</p>
        <div class="flex justify-center">
          <StarRating
            bind:rating={roleEditMember.skill_stars}
            size={32}
          />
        </div>
      </div>

      <!-- Position selector -->
      <div>
        <p class="text-xs font-medium text-gray-500 dark:text-gray-400 mb-2">{$t('group.detail_position')}</p>
        <div class="flex justify-center">
          <PositionSelector bind:value={roleEditMember.position} />
        </div>
      </div>

      <!-- Save skill button -->
      <button
        class="btn btn-primary w-full justify-center"
        disabled={skillSaving}
        onclick={async () => {
          await saveSkill(roleEditMember!.id, roleEditMember!.skill_stars, roleEditMember!.position);
          roleEditMember = null;
        }}>
        {skillSaving ? $t('group.saving') : $t('group.save_skill')}
      </button>

      <div class="border-t border-gray-100 dark:border-gray-700 pt-3 flex flex-col gap-2">
        {#if roleEditMember.id !== $currentPlayer?.id}
          <button
            class="btn btn-secondary justify-center py-2.5"
            onclick={() => { toggleRole(roleEditMember!.id, roleEditMember!.role, roleEditMember!.name); roleEditMember = null; }}>
            {#if roleEditMember.role === 'admin'}
              <ShieldOff size={15} /> {$t('group.remove_president')}
            {:else}
              <ShieldCheck size={15} /> {$t('group.make_president')}
            {/if}
          </button>
        {/if}
        <button class="btn btn-secondary justify-center py-2.5" onclick={() => roleEditMember = null}>
          {$t('group.cancel')}
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
<Modal bind:open={showEditGroup} title={$t('group.edit_group_title')}>
  <form onsubmit={(e) => { e.preventDefault(); saveEditGroup(); }} class="space-y-4">
    <div class="form-group">
      <label class="label" for="egname">{$t('group.edit_name_label')}</label>
      <input id="egname" class="input" bind:value={editForm.name} required minlength="2" maxlength="100" />
    </div>
    <div class="form-group">
      <label class="label" for="egdesc">{$t('group.edit_desc_label')}</label>
      <textarea id="egdesc" class="input resize-none" rows="2" bind:value={editForm.description} placeholder={$t('group.edit_desc_placeholder')}></textarea>
    </div>
    <div class="grid grid-cols-2 gap-4">
      <div class="form-group">
        <label class="label" for="egpermatch">{$t('group.edit_permatch_label')}</label>
        <input id="egpermatch" class="input" type="number" min="0" step="0.01"
          bind:value={editForm.per_match_amount} placeholder={$t('group.edit_permatch_placeholder')} />
        <p class="text-xs text-gray-400 dark:text-gray-500 mt-0.5">{$t('group.edit_permatch_hint')}</p>
      </div>
      <div class="form-group">
        <label class="label" for="egmonthly">{$t('group.edit_monthly_label')}</label>
        <input id="egmonthly" class="input" type="number" min="0" step="0.01"
          bind:value={editForm.monthly_amount} placeholder={$t('group.edit_permatch_placeholder')} />
        <p class="text-xs text-gray-400 dark:text-gray-500 mt-0.5">{$t('group.edit_permatch_hint')}</p>
      </div>
    </div>
    <div class="form-group">
      <label class="flex items-center gap-3 cursor-pointer select-none">
        <div class="relative">
          <input type="checkbox" class="sr-only peer" bind:checked={editForm.recurrence_enabled} />
          <div class="w-10 h-6 bg-gray-200 dark:bg-gray-600 peer-checked:bg-primary-600 rounded-full transition-colors"></div>
          <div class="absolute top-0.5 left-0.5 w-5 h-5 bg-white dark:bg-gray-200 rounded-full shadow transition-transform peer-checked:translate-x-4"></div>
        </div>
        <span class="text-sm font-medium text-gray-700 dark:text-gray-300">{$t('group.weekly_recurrence')}</span>
      </label>
      <p class="text-xs text-gray-400 dark:text-gray-500 mt-1">
        Quando ativa, uma nova partida é criada automaticamente após o encerramento da atual, herdando os convidados com presença pendente.
      </p>
    </div>
    <div class="form-group">
      <label class="flex items-center gap-3 cursor-pointer select-none">
        <div class="relative">
          <input type="checkbox" class="sr-only peer" bind:checked={editForm.is_public} />
          <div class="w-10 h-6 bg-gray-200 dark:bg-gray-600 peer-checked:bg-primary-600 rounded-full transition-colors"></div>
          <div class="absolute top-0.5 left-0.5 w-5 h-5 bg-white dark:bg-gray-200 rounded-full shadow transition-transform peer-checked:translate-x-4"></div>
        </div>
        <span class="text-sm font-medium text-gray-700 dark:text-gray-300">{$t('new_group.public_title')}</span>
      </label>
      <p class="text-xs text-gray-400 dark:text-gray-500 mt-1">
        Quando ativo, qualquer pessoa com o link pode solicitar entrada no próximo rachão via lista de espera.
      </p>
    </div>
    <div class="border-t border-gray-100 dark:border-gray-700 pt-4">
      <label class="label" for="egtimezone">{$t('new_group.timezone_label')}</label>
      <p class="text-xs text-gray-400 dark:text-gray-500 mb-2">{$t('new_group.timezone_desc')}</p>
      <select id="egtimezone" class="input" bind:value={editForm.timezone}>
        {#each TIMEZONE_GROUPS as tzGroup}
          <optgroup label={tzGroup}>
            {#each TIMEZONE_OPTIONS.filter(tz => tz.group === tzGroup) as tz}
              <option value={tz.value}>{tz.label} ({tz.offset})</option>
            {/each}
          </optgroup>
        {/each}
      </select>
    </div>
    <div class="border-t border-gray-100 dark:border-gray-700 pt-4">
      <p class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">{$t('new_group.vote_settings')}</p>
      <div class="space-y-3">
        <div class="form-group">
          <label class="label" for="egvotedelay">{$t('new_group.vote_delay_label')}</label>
          <select id="egvotedelay" class="input" bind:value={editForm.vote_open_delay_minutes}>
            <option value={0}>{$t('new_group.vote_immediate')}</option>
            <option value={10}>{$t('new_group.vote_10min')}</option>
            <option value={20}>{$t('new_group.vote_20min')}</option>
            <option value={30}>{$t('new_group.vote_30min')}</option>
            <option value={60}>{$t('new_group.vote_1h')}</option>
          </select>
        </div>
        <div class="form-group">
          <label class="label" for="egvotedur">{$t('new_group.vote_duration_label')}</label>
          <select id="egvotedur" class="input" bind:value={editForm.vote_duration_hours}>
            <option value={2}>{$t('new_group.vote_2h')}</option>
            <option value={4}>{$t('new_group.vote_4h')}</option>
            <option value={6}>{$t('new_group.vote_6h')}</option>
            <option value={12}>{$t('new_group.vote_12h')}</option>
            <option value={24}>{$t('new_group.vote_24h')}</option>
            <option value={48}>{$t('new_group.vote_48h')}</option>
            <option value={72}>{$t('new_group.vote_72h')}</option>
          </select>
        </div>
      </div>
    </div>
    <div class="flex gap-3 justify-end pt-2">
      <button type="button" class="btn-secondary" onclick={() => showEditGroup = false}>{$t('group.cancel')}</button>
      <button type="submit" class="btn-primary" disabled={saving}>{saving ? $t('group.save_loading') : $t('group.save_btn')}</button>
    </div>
  </form>
</Modal>

{#if showWaitlistModal && activeMatchDetail}
  <WaitlistModal
    bind:open={showWaitlistModal}
    match={activeMatchDetail}
    submitting={submittingWaitlist}
    onsubmit={submitWaitlist}
    onclose={() => showWaitlistModal = false}
  />
{/if}
