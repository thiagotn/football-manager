<script lang="ts">
  import { page } from '$app/stores';
  import { groups as groupsApi, matches as matchesApi, invites, players as playersApi, ApiError } from '$lib/api';
  import type { GroupDetail, Match, Player } from '$lib/api';
  import { currentPlayer, isAdmin } from '$lib/stores/auth';
  import { toastSuccess, toastError, toastInfo } from '$lib/stores/toast';
  import Modal from '$lib/components/Modal.svelte';
  import { Plus, Calendar, Users, Link, Trash2, Clock, MapPin, Copy, UserPlus, ChevronRight, ShieldCheck, ShieldOff, Pencil } from 'lucide-svelte';

  const groupId = $page.params.id;

  let group: GroupDetail | null = $state(null);
  let matchList: Match[] = $state([]);
  let loading = $state(true);
  let tab: 'matches' | 'members' = $state('matches');

  // Modals
  let showMatch = $state(false);
  let showEditMatch = $state(false);
  let showInvite = $state(false);
  let showAddMember = $state(false);
  let showEditGroup = $state(false);

  let inviteLink = $state('');
  const COURT_LABELS: Record<string, string> = { campo: 'Campo', sintetico: 'Sintético', terrao: 'Terrão', quadra: 'Quadra' };
  let matchForm = $state({ match_date: '', start_time: '20:30', end_time: '', location: '', address: '', court_type: '', players_per_team: '', max_players: '', notes: '' });
  let editMatchForm = $state({ match_date: '', start_time: '', end_time: '', location: '', address: '', court_type: '', players_per_team: '', max_players: '', notes: '', status: 'open' });
  let editingMatch: Match | null = $state(null);
  let saving = $state(false);

  let allPlayers: Player[] = $state([]);
  let addMemberId = $state('');

  let editForm = $state({ name: '', description: '', per_match_amount: '', monthly_amount: '' });

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
      toastSuccess('Partida criada!');
    } catch (e) { toastError(e instanceof ApiError ? e.message : 'Erro'); }
    saving = false;
  }

  async function generateInvite() {
    try {
      const inv = await invites.create(groupId);
      const base = window.location.origin;
      inviteLink = `${base}/invite/${inv.token}`;
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
    } catch (e) { toastError(e instanceof ApiError ? e.message : 'Erro'); }
    saving = false;
  }

  async function toggleRole(playerId: string, currentRole: string, name: string) {
    const newRole = currentRole === 'admin' ? 'member' : 'admin';
    const action = newRole === 'admin' ? 'tornar presidente' : 'remover a presidência de';
    if (!confirm(`Deseja ${action} "${name}"?`)) return;
    try {
      await groupsApi.updateMemberRole(groupId, playerId, newRole);
      group = await groupsApi.get(groupId);
      toastSuccess(newRole === 'admin' ? `${name} agora é presidente do grupo` : `${name} voltou a ser membro`);
    } catch (e) { toastError(e instanceof ApiError ? e.message : 'Erro ao alterar papel'); }
  }

  async function removeMember(playerId: string, name: string) {
    if (!confirm(`Remover "${name}" do grupo?`)) return;
    try {
      await groupsApi.removeMember(groupId, playerId);
      group = await groupsApi.get(groupId);
      toastSuccess('Membro removido');
    } catch (e) { toastError('Erro ao remover membro'); }
  }

  async function deleteMatch(m: Match) {
    if (!confirm('Excluir esta partida?')) return;
    try {
      await matchesApi.delete(groupId, m.id);
      matchList = matchList.filter(x => x.id !== m.id);
      toastSuccess('Partida excluída');
    } catch (e) { toastError('Erro ao excluir'); }
  }

  function fmtDate(d: string) {
    return new Date(d + 'T00:00').toLocaleDateString('pt-BR', { weekday: 'long', day: '2-digit', month: 'long' });
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
      toastSuccess('Partida atualizada!');
    } catch (e) { toastError(e instanceof ApiError ? e.message : 'Erro ao salvar'); }
    saving = false;
  }
</script>

<svelte:head><title>{group?.name ?? 'Grupo'} — rachao.app</title></svelte:head>

<main class="max-w-7xl mx-auto px-4 py-8">
  {#if loading}
    <div class="animate-pulse space-y-4">
      <div class="h-8 bg-gray-200 rounded w-1/3"></div>
      <div class="h-4 bg-gray-100 rounded w-1/2"></div>
    </div>
  {:else if group}
    <!-- Header -->
    <div class="flex flex-wrap items-start justify-between gap-4 mb-6">
      <div>
        <h1 class="text-2xl font-bold text-gray-900">{group.name}</h1>
        {#if group.description}<p class="text-gray-500 text-sm mt-1">{group.description}</p>{/if}
        <p class="text-xs text-gray-400 mt-1">{group.total_members} membro{group.total_members !== 1 ? 's' : ''}</p>
        {#if group.per_match_amount != null || group.monthly_amount != null}
          <p class="text-xs text-amber-700 mt-1 font-medium">{fmtPricingParts(group.per_match_amount, group.monthly_amount).join(' · ')}</p>
        {:else}
          <p class="text-xs text-green-600 mt-1">Partida aberta — sem cobrança</p>
        {/if}
      </div>
      {#if isGroupAdmin()}
        <div class="flex flex-wrap gap-2">
          <button class="btn-secondary btn-sm" onclick={openEditGroup}><Pencil size={14} /> Editar</button>
          <button class="btn-secondary btn-sm" onclick={generateInvite}><Link size={14} /> Convidar</button>
          {#if tab === 'members'}
            <button class="btn-secondary btn-sm" onclick={openAddMember}><UserPlus size={14} /> Adicionar</button>
          {:else}
            <button class="btn-primary btn-sm" onclick={() => showMatch = true}><Plus size={14} /> Nova Partida</button>
          {/if}
        </div>
      {/if}
    </div>

    <!-- Tabs -->
    <div class="flex gap-1 border-b border-gray-200 mb-6">
      <button
        class="px-4 py-2 text-sm font-medium border-b-2 transition-colors {tab === 'matches' ? 'border-primary-600 text-primary-600' : 'border-transparent text-gray-500 hover:text-gray-700'}"
        onclick={() => tab = 'matches'}>
        <span class="flex items-center gap-1.5"><Calendar size={14} /> Partidas ({matchList.length})</span>
      </button>
      <button
        class="px-4 py-2 text-sm font-medium border-b-2 transition-colors {tab === 'members' ? 'border-primary-600 text-primary-600' : 'border-transparent text-gray-500 hover:text-gray-700'}"
        onclick={() => tab = 'members'}>
        <span class="flex items-center gap-1.5"><Users size={14} /> Membros ({group.total_members})</span>
      </button>
    </div>

    <!-- Matches tab -->
    {#if tab === 'matches'}
      {#if matchList.length === 0}
        <div class="card p-12 text-center">
          <Calendar size={40} class="text-gray-300 mx-auto mb-3" />
          <p class="text-gray-500">Nenhuma partida agendada.</p>
          {#if isGroupAdmin()}
            <button class="btn-primary mt-4" onclick={() => showMatch = true}><Plus size={16} /> Criar partida</button>
          {/if}
        </div>
      {:else}
        <div class="space-y-3">
          {#each matchList as m}
            <div class="card hover:shadow-md transition-shadow">
              <div class="card-body">
                <div class="flex items-start gap-3">
                  <!-- Number badge -->
                  <div class="w-10 h-10 rounded-xl bg-primary-100 flex items-center justify-center shrink-0 text-primary-700 font-bold text-sm">
                    #{m.number}
                  </div>

                  <!-- Content -->
                  <div class="flex-1 min-w-0">
                    <!-- Date + status badge on same row -->
                    <div class="flex items-start gap-2">
                      <p class="font-semibold text-gray-900 capitalize leading-snug flex-1">{fmtDate(m.match_date)}</p>
                      <span class="badge {m.status === 'open' ? 'badge-green' : 'badge-gray'} shrink-0 mt-0.5">
                        {m.status === 'open' ? 'Aberta' : 'Encerrada'}
                      </span>
                    </div>

                    <!-- Time + Location -->
                    <div class="flex flex-wrap items-center gap-x-3 gap-y-0.5 mt-1 text-sm text-gray-500">
                      <span class="flex items-center gap-1 whitespace-nowrap"><Clock size={12} />{fmtTimeRange(m.start_time, m.end_time)}</span>
                      <span class="flex items-center gap-1 min-w-0"><MapPin size={12} /><span class="truncate">{m.location}</span></span>
                    </div>

                    <!-- Court / players details -->
                    {#if m.court_type || m.players_per_team || m.max_players}
                      <p class="text-xs text-gray-400 mt-0.5">
                        {[
                          m.court_type ? COURT_LABELS[m.court_type] : null,
                          m.players_per_team ? `${m.players_per_team} na linha + gol` : null,
                          m.max_players ? `máx. ${m.max_players}` : null,
                        ].filter(Boolean).join(' · ')}
                      </p>
                    {/if}

                    <!-- Footer: pricing (left) + actions (right) -->
                    <div class="flex items-center justify-between mt-2 pt-2 border-t border-gray-100">
                      <span class="text-xs text-amber-700 font-medium">
                        {#if group && (group.per_match_amount != null || group.monthly_amount != null)}
                          {fmtPricingParts(group.per_match_amount, group.monthly_amount).join(' · ')}
                        {/if}
                      </span>
                      <div class="flex items-center gap-1">
                        <a href="/match/{m.hash}" class="btn-secondary btn-sm" title="Ver partida">
                          <ChevronRight size={14} />
                        </a>
                        {#if isGroupAdmin()}
                          <button onclick={() => openEditMatch(m)} class="btn-ghost btn-sm" title="Editar partida">
                            <Pencil size={14} />
                          </button>
                          <button onclick={() => deleteMatch(m)} class="btn-ghost btn-sm text-red-500 hover:bg-red-50" title="Excluir partida">
                            <Trash2 size={14} />
                          </button>
                        {/if}
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          {/each}
        </div>
      {/if}
    {/if}

    <!-- Members tab -->
    {#if tab === 'members'}
      <div class="card overflow-hidden">
        <table class="table">
          <thead><tr><th>Jogador</th><th>WhatsApp</th><th>Papel</th>{#if isGroupAdmin()}<th></th>{/if}</tr></thead>
          <tbody>
            {#each group.members as m}
              <tr>
                <td>
                  <p class="font-medium text-gray-900">{m.player.name}</p>
                  {#if m.player.nickname}<p class="text-xs text-gray-400">{m.player.nickname}</p>{/if}
                </td>
                <td class="text-gray-500 font-mono text-xs">{m.player.whatsapp}</td>
                <td>
                  <span class="badge {m.role === 'admin' ? 'badge-blue' : 'badge-gray'}">
                    {m.role === 'admin' ? 'Presidente' : 'Membro'}
                  </span>
                </td>
                {#if isGroupAdmin()}
                  <td>
                    {#if m.player.id !== $currentPlayer?.id}
                      <div class="flex gap-1 justify-end">
                        {#if m.role === 'admin'}
                          <button
                            onclick={() => toggleRole(m.player.id, m.role, m.player.name)}
                            title="Remover presidência"
                            class="btn-ghost btn-sm text-gray-400 hover:bg-gray-100">
                            <ShieldOff size={14} />
                          </button>
                        {:else if $isAdmin}
                          <button
                            onclick={() => toggleRole(m.player.id, m.role, m.player.name)}
                            title="Tornar presidente"
                            class="btn-ghost btn-sm text-blue-500 hover:bg-blue-50">
                            <ShieldCheck size={14} />
                          </button>
                        {/if}
                        <button onclick={() => removeMember(m.player.id, m.player.name)}
                          class="btn-ghost btn-sm text-red-500 hover:bg-red-50">
                          <Trash2 size={14} />
                        </button>
                      </div>
                    {/if}
                  </td>
                {/if}
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  {/if}
</main>

<!-- Create match modal -->
<Modal bind:open={showMatch} title="Nova Partida">
  <form onsubmit={(e) => { e.preventDefault(); createMatch(); }} class="space-y-4">
    <div class="form-group">
      <label class="label" for="mdate">Data *</label>
      <input id="mdate" class="input" type="date" bind:value={matchForm.match_date} required />
    </div>
    <div class="grid grid-cols-2 gap-4">
      <div class="form-group">
        <label class="label" for="mtime">Início *</label>
        <input id="mtime" class="input" type="time" bind:value={matchForm.start_time} required />
      </div>
      <div class="form-group">
        <label class="label" for="mendtime">Término <span class="text-gray-400 font-normal">(opcional)</span></label>
        <input id="mendtime" class="input" type="time" bind:value={matchForm.end_time} />
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
<Modal bind:open={showEditMatch} title="Editar Partida">
  <form onsubmit={(e) => { e.preventDefault(); saveEditMatch(); }} class="space-y-4">
    <div class="form-group">
      <label class="label" for="emdate">Data *</label>
      <input id="emdate" class="input" type="date" bind:value={editMatchForm.match_date} required />
    </div>
    <div class="grid grid-cols-2 gap-4">
      <div class="form-group">
        <label class="label" for="emtime">Início *</label>
        <input id="emtime" class="input" type="time" bind:value={editMatchForm.start_time} required />
      </div>
      <div class="form-group">
        <label class="label" for="emendtime">Término <span class="text-gray-400 font-normal">(opcional)</span></label>
        <input id="emendtime" class="input" type="time" bind:value={editMatchForm.end_time} />
      </div>
    </div>
    <div class="form-group">
      <label class="label" for="emloc">Local *</label>
      <input id="emloc" class="input" bind:value={editMatchForm.location} required />
    </div>
    <div class="form-group">
      <label class="label" for="emaddr">Endereço <span class="text-gray-400 font-normal">(opcional)</span></label>
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
            <option value={n}>{n} na linha</option>
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
<Modal bind:open={showInvite} title="Link de Convite">
  <div class="space-y-4">
    <div class="alert-info">⏱ Este link expira em <strong>30 minutos</strong> e só pode ser usado <strong>uma vez</strong>.</div>
    <div class="flex gap-2">
      <input class="input font-mono text-xs" readonly value={inviteLink} />
      <button class="btn-primary shrink-0" onclick={copyLink}><Copy size={16} /></button>
    </div>
    <a
      href="https://wa.me/?text={encodeURIComponent(`Você foi convidado para o grupo *${group?.name}* no Rachão!\n\nClique no link abaixo para criar sua conta e entrar no grupo:\n${inviteLink}`)}"
      target="_blank"
      rel="noopener noreferrer"
      class="btn-secondary w-full flex items-center justify-center gap-2">
      <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="currentColor" class="shrink-0">
        <path d="M17.472 14.382c-.297-.149-1.758-.867-2.03-.967-.273-.099-.471-.148-.67.15-.197.297-.767.966-.94 1.164-.173.199-.347.223-.644.075-.297-.15-1.255-.463-2.39-1.475-.883-.788-1.48-1.761-1.653-2.059-.173-.297-.018-.458.13-.606.134-.133.298-.347.446-.52.149-.174.198-.298.298-.497.099-.198.05-.371-.025-.52-.075-.149-.669-1.612-.916-2.207-.242-.579-.487-.5-.669-.51-.173-.008-.371-.01-.57-.01-.198 0-.52.074-.792.372-.272.297-1.04 1.016-1.04 2.479 0 1.462 1.065 2.875 1.213 3.074.149.198 2.096 3.2 5.077 4.487.709.306 1.262.489 1.694.625.712.227 1.36.195 1.871.118.571-.085 1.758-.719 2.006-1.413.248-.694.248-1.289.173-1.413-.074-.124-.272-.198-.57-.347z"/>
        <path d="M12 0C5.373 0 0 5.373 0 12c0 2.126.558 4.121 1.533 5.853L.036 23.964l6.252-1.639A11.945 11.945 0 0 0 12 24c6.627 0 12-5.373 12-12S18.627 0 12 0zm0 21.818a9.8 9.8 0 0 1-4.998-1.366l-.358-.213-3.712.974 1.014-3.598-.233-.371A9.818 9.818 0 1 1 12 21.818z"/>
      </svg>
      Enviar pelo WhatsApp
    </a>
    <button class="btn-secondary w-full" onclick={() => showInvite = false}>Fechar</button>
  </div>
</Modal>

<!-- Add member modal -->
<Modal bind:open={showAddMember} title="Adicionar Membro">
  <form onsubmit={(e) => { e.preventDefault(); addMember(); }} class="space-y-4">
    <div class="form-group">
      <label class="label" for="pid">Selecionar Jogador</label>
      <select id="pid" class="input" bind:value={addMemberId} required>
        <option value="">— selecione —</option>
        {#each allPlayers.filter(p => !group?.members.some(m => m.player.id === p.id)) as p}
          <option value={p.id}>{p.name} {p.nickname ? `(${p.nickname})` : ''}</option>
        {/each}
      </select>
    </div>
    <div class="flex gap-3 justify-end">
      <button type="button" class="btn-secondary" onclick={() => showAddMember = false}>Cancelar</button>
      <button type="submit" class="btn-primary" disabled={saving}>{saving ? 'Adicionando…' : 'Adicionar'}</button>
    </div>
  </form>
</Modal>

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
        <p class="text-xs text-gray-400 mt-0.5">Deixe vazio se não cobrar por partida</p>
      </div>
      <div class="form-group">
        <label class="label" for="egmonthly">Mensalidade (R$)</label>
        <input id="egmonthly" class="input" type="number" min="0" step="0.01"
          bind:value={editForm.monthly_amount} placeholder="Ex: 75,00" />
        <p class="text-xs text-gray-400 mt-0.5">Deixe vazio se não cobrar mensalidade</p>
      </div>
    </div>
    <div class="flex gap-3 justify-end pt-2">
      <button type="button" class="btn-secondary" onclick={() => showEditGroup = false}>Cancelar</button>
      <button type="submit" class="btn-primary" disabled={saving}>{saving ? 'Salvando…' : 'Salvar'}</button>
    </div>
  </form>
</Modal>
