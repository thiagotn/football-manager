<script lang="ts">
  import { page } from '$app/stores';
  import { groups as groupsApi, matches as matchesApi, invites, players as playersApi, ApiError } from '$lib/api';
  import type { GroupDetail, Match, Player } from '$lib/api';
  import { currentPlayer, isAdmin } from '$lib/stores/auth';
  import { toastSuccess, toastError, toastInfo } from '$lib/stores/toast';
  import Modal from '$lib/components/Modal.svelte';
  import { Plus, Calendar, Users, Link, Trash2, Clock, MapPin, Copy, UserPlus, ChevronRight } from 'lucide-svelte';

  const groupId = $page.params.id;

  let group: GroupDetail | null = $state(null);
  let matchList: Match[] = $state([]);
  let loading = $state(true);
  let tab: 'matches' | 'members' = $state('matches');

  // Modals
  let showMatch = $state(false);
  let showInvite = $state(false);
  let showAddMember = $state(false);

  let inviteLink = $state('');
  let matchForm = $state({ match_date: '', start_time: '20:30', location: '', notes: '' });
  let saving = $state(false);

  let allPlayers: Player[] = $state([]);
  let addMemberId = $state('');

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
      matchForm = { match_date: '', start_time: '20:30', location: '', notes: '' };
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
</script>

<svelte:head><title>{group?.name ?? 'Grupo'} — Joga Bonito</title></svelte:head>

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
      </div>
      {#if isGroupAdmin()}
        <div class="flex flex-wrap gap-2">
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
              <div class="card-body flex items-center gap-4">
                <div class="w-12 h-12 rounded-xl bg-primary-100 flex items-center justify-center shrink-0 text-primary-700 font-bold text-sm">
                  {new Date(m.match_date + 'T00:00').getDate()}/{new Date(m.match_date + 'T00:00').getMonth() + 1}
                </div>
                <div class="flex-1 min-w-0">
                  <p class="font-medium text-gray-900 capitalize">{fmtDate(m.match_date)}</p>
                  <p class="text-sm text-gray-500 flex flex-wrap gap-3 mt-0.5">
                    <span class="flex items-center gap-1"><Clock size={12} />{m.start_time.slice(0,5)}</span>
                    <span class="flex items-center gap-1"><MapPin size={12} />{m.location}</span>
                  </p>
                </div>
                <div class="flex items-center gap-2 shrink-0">
                  <span class="badge {m.status === 'open' ? 'badge-green' : 'badge-gray'}">
                    {m.status === 'open' ? 'Aberta' : 'Encerrada'}
                  </span>
                  <a href="/match/{m.hash}" class="btn-secondary btn-sm">
                    <ChevronRight size={14} />
                  </a>
                  {#if isGroupAdmin()}
                    <button onclick={() => deleteMatch(m)} class="btn-ghost btn-sm text-red-500 hover:bg-red-50">
                      <Trash2 size={14} />
                    </button>
                  {/if}
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
                <td class="text-gray-500 font-mono text-xs">{m.player.id.slice(0,8)}…</td>
                <td>
                  <span class="badge {m.role === 'admin' ? 'badge-blue' : 'badge-gray'}">
                    {m.role === 'admin' ? 'Admin' : 'Membro'}
                  </span>
                </td>
                {#if isGroupAdmin()}
                  <td>
                    {#if m.player.id !== $currentPlayer?.id}
                      <button onclick={() => removeMember(m.player.id, m.player.name)}
                        class="btn-ghost btn-sm text-red-500 hover:bg-red-50">
                        <Trash2 size={14} />
                      </button>
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
    <div class="grid grid-cols-2 gap-4">
      <div class="form-group">
        <label class="label" for="mdate">Data *</label>
        <input id="mdate" class="input" type="date" bind:value={matchForm.match_date} required />
      </div>
      <div class="form-group">
        <label class="label" for="mtime">Hora *</label>
        <input id="mtime" class="input" type="time" bind:value={matchForm.start_time} required />
      </div>
    </div>
    <div class="form-group">
      <label class="label" for="mloc">Local *</label>
      <input id="mloc" class="input" bind:value={matchForm.location} placeholder="Ex: Arena GQC — Quadra 3" required />
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

<!-- Invite modal -->
<Modal bind:open={showInvite} title="Link de Convite">
  <div class="space-y-4">
    <div class="alert-info">⏱ Este link expira em <strong>30 minutos</strong> e só pode ser usado <strong>uma vez</strong>.</div>
    <div class="flex gap-2">
      <input class="input font-mono text-xs" readonly value={inviteLink} />
      <button class="btn-primary shrink-0" onclick={copyLink}><Copy size={16} /></button>
    </div>
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
