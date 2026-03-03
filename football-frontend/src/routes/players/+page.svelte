<script lang="ts">
  import { players as playersApi, ApiError } from '$lib/api';
  import type { Player } from '$lib/api';
  import { toastSuccess, toastError } from '$lib/stores/toast';
  import Modal from '$lib/components/Modal.svelte';
  import { Plus, Users, Pencil, Trash2, Eye, EyeOff, Search, KeyRound, Copy } from 'lucide-svelte';

  let playerList: Player[] = $state([]);
  let loading = $state(true);
  let loadError = $state('');
  let search = $state('');

  let showCreate = $state(false);
  let showEdit = $state(false);
  let showReset = $state(false);
  let editing: Player | null = $state(null);
  let resetTarget: Player | null = $state(null);
  let generatedPassword = $state('');
  let saving = $state(false);
  let resetting = $state(false);
  let showPw = $state(false);

  let form = $state({ name: '', nickname: '', whatsapp: '', password: '', role: 'player' });
  let editForm = $state({ name: '', nickname: '', role: '' });

  let filtered = $derived(
    playerList.filter(p =>
      p.name.toLowerCase().includes(search.toLowerCase()) ||
      p.whatsapp.includes(search) ||
      (p.nickname ?? '').toLowerCase().includes(search.toLowerCase())
    )
  );

  $effect(() => {
    let cancelled = false;
    (async () => {
      try {
        const data = await playersApi.list();
        if (!cancelled) playerList = data;
      } catch (e) {
        if (!cancelled) {
          console.error('[players] erro ao carregar:', e);
          loadError = e instanceof ApiError ? `${e.status}: ${e.message}` : String(e);
          toastError('Erro ao carregar jogadores');
        }
      }
      if (!cancelled) loading = false;
    })();
    return () => { cancelled = true; };
  });

  async function create() {
    saving = true;
    try {
      const p = await playersApi.create({ ...form, nickname: form.nickname || undefined });
      playerList = [p, ...playerList];
      showCreate = false;
      form = { name: '', nickname: '', whatsapp: '', password: '', role: 'player' };
      toastSuccess('Jogador criado!');
    } catch (e) { toastError(e instanceof ApiError ? e.message : 'Erro'); }
    saving = false;
  }

  function openEdit(p: Player) {
    editing = p;
    editForm = { name: p.name, nickname: p.nickname ?? '', role: p.role };
    showEdit = true;
  }

  async function saveEdit() {
    if (!editing) return;
    saving = true;
    try {
      const updated = await playersApi.update(editing.id, {
        name: editForm.name, nickname: editForm.nickname || undefined, role: editForm.role
      });
      playerList = playerList.map(p => p.id === updated.id ? updated : p);
      showEdit = false;
      toastSuccess('Jogador atualizado!');
    } catch (e) { toastError('Erro ao atualizar'); }
    saving = false;
  }

  function openReset(p: Player) {
    resetTarget = p;
    generatedPassword = '';
    showReset = true;
  }

  async function doReset() {
    if (!resetTarget) return;
    resetting = true;
    try {
      const res = await playersApi.resetPassword(resetTarget.id);
      generatedPassword = res.temp_password;
    } catch (e) {
      toastError(e instanceof ApiError ? e.message : 'Erro ao resetar senha');
      showReset = false;
    }
    resetting = false;
  }

  function copyPassword() {
    navigator.clipboard.writeText(generatedPassword);
    toastSuccess('Senha copiada!');
  }

  async function deactivate(p: Player) {
    if (!confirm(`Desativar "${p.name}"?`)) return;
    try {
      await playersApi.delete(p.id);
      playerList = playerList.map(x => x.id === p.id ? { ...x, active: false } : x);
      toastSuccess('Jogador desativado');
    } catch { toastError('Erro ao desativar'); }
  }
</script>

<svelte:head><title>Jogadores — rachao.app</title></svelte:head>

<main class="max-w-7xl mx-auto px-4 py-8">
  <div class="flex flex-wrap items-center justify-between gap-4 mb-6">
    <div>
      <h1 class="text-2xl font-bold text-gray-900 flex items-center gap-2">
        <Users size={24} class="text-primary-600" /> Jogadores
      </h1>
      <p class="text-sm text-gray-500 mt-0.5">{playerList.length} jogadores cadastrados</p>
    </div>
    <button class="btn-primary" onclick={() => showCreate = true}>
      <Plus size={16} /> Novo Jogador
    </button>
  </div>

  {#if loadError}
    <div class="alert-error mb-4">Erro ao carregar jogadores: <strong>{loadError}</strong></div>
  {/if}

  <!-- Search -->
  <div class="relative mb-4 max-w-sm">
    <Search size={16} class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
    <input class="input pl-9" placeholder="Buscar por nome, apelido ou WhatsApp…" bind:value={search} />
  </div>

  {#if loading}
    <div class="card overflow-hidden">
      {#each [1,2,3,4,5] as _}
        <div class="px-6 py-4 border-b border-gray-100 animate-pulse">
          <div class="h-4 bg-gray-100 rounded w-1/3"></div>
        </div>
      {/each}
    </div>
  {:else}
    <div class="card overflow-hidden">
      <table class="table">
        <thead>
          <tr>
            <th>Jogador</th>
            <th>WhatsApp</th>
            <th>Perfil</th>
            <th>Status</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {#each filtered as p}
            <tr>
              <td>
                <p class="font-medium">{p.name}</p>
                {#if p.nickname}<p class="text-xs text-gray-400">{p.nickname}</p>{/if}
              </td>
              <td class="font-mono text-xs text-gray-600">{p.whatsapp}</td>
              <td>
                <span class="badge {p.role === 'admin' ? 'badge-blue' : 'badge-gray'}">
                  {p.role === 'admin' ? 'Admin' : 'Jogador'}
                </span>
              </td>
              <td>
                <span class="badge {p.active ? 'badge-green' : 'badge-red'}">
                  {p.active ? 'Ativo' : 'Inativo'}
                </span>
              </td>
              <td>
                <div class="flex gap-1 justify-end">
                  <button onclick={() => openEdit(p)} class="btn-ghost btn-sm" title="Editar"><Pencil size={14} /></button>
                  <button onclick={() => openReset(p)} class="btn-ghost btn-sm text-amber-600 hover:bg-amber-50" title="Resetar senha"><KeyRound size={14} /></button>
                  {#if p.active}
                    <button onclick={() => deactivate(p)} class="btn-ghost btn-sm text-red-500 hover:bg-red-50" title="Desativar"><Trash2 size={14} /></button>
                  {/if}
                </div>
              </td>
            </tr>
          {/each}
          {#if filtered.length === 0}
            <tr><td colspan="5" class="text-center text-gray-400 py-8">Nenhum jogador encontrado.</td></tr>
          {/if}
        </tbody>
      </table>
    </div>
  {/if}
</main>

<!-- Create modal -->
<Modal bind:open={showCreate} title="Novo Jogador">
  <form onsubmit={(e) => { e.preventDefault(); create(); }} class="space-y-4">
    <div class="form-group">
      <label class="label">Nome *</label>
      <input class="input" bind:value={form.name} required minlength="2" />
    </div>
    <div class="form-group">
      <label class="label">Apelido</label>
      <input class="input" bind:value={form.nickname} placeholder="Opcional" />
    </div>
    <div class="form-group">
      <label class="label">WhatsApp *</label>
      <input class="input" type="tel" bind:value={form.whatsapp} placeholder="11999990000" required />
    </div>
    <div class="form-group">
      <label class="label">Senha *</label>
      <div class="relative">
        <input class="input pr-10" type={showPw ? 'text' : 'password'} bind:value={form.password} required minlength="6" />
        <button type="button" onclick={() => showPw = !showPw} class="absolute right-2.5 top-1/2 -translate-y-1/2 text-gray-400">
          {#if showPw}<EyeOff size={16} />{:else}<Eye size={16} />{/if}
        </button>
      </div>
    </div>
    <div class="form-group">
      <label class="label">Perfil</label>
      <select class="input" bind:value={form.role}>
        <option value="player">Jogador</option>
        <option value="admin">Administrador</option>
      </select>
    </div>
    <div class="flex gap-3 justify-end pt-2">
      <button type="button" class="btn-secondary" onclick={() => showCreate = false}>Cancelar</button>
      <button type="submit" class="btn-primary" disabled={saving}>{saving ? 'Criando…' : 'Criar'}</button>
    </div>
  </form>
</Modal>

<!-- Reset password modal -->
<Modal bind:open={showReset} title="Resetar Senha — {resetTarget?.name ?? ''}">
  {#if !generatedPassword}
    <p class="text-sm text-gray-600 mb-4">
      Uma senha temporária será gerada para <strong>{resetTarget?.name}</strong>.
      O jogador precisará alterá-la no próximo acesso.
    </p>
    <div class="flex gap-3 justify-end">
      <button class="btn btn-secondary" onclick={() => showReset = false}>Cancelar</button>
      <button class="btn bg-amber-500 hover:bg-amber-600 text-white" onclick={doReset} disabled={resetting}>
        <KeyRound size={15} /> {resetting ? 'Gerando…' : 'Gerar senha temporária'}
      </button>
    </div>
  {:else}
    <p class="text-sm text-gray-600 mb-3">Senha temporária gerada. Compartilhe com o jogador:</p>
    <div class="flex items-center gap-2 mb-4">
      <code class="flex-1 bg-gray-100 rounded-lg px-4 py-3 font-mono text-lg tracking-widest text-center text-gray-900 select-all">
        {generatedPassword}
      </code>
      <button class="btn btn-secondary shrink-0" onclick={copyPassword}>
        <Copy size={15} /> Copiar
      </button>
    </div>
    <p class="text-xs text-amber-700 bg-amber-50 border border-amber-200 rounded-lg px-3 py-2 mb-4">
      ⚠️ O jogador será obrigado a alterar esta senha no próximo acesso.
    </p>
    <div class="flex justify-end">
      <button class="btn btn-primary" onclick={() => showReset = false}>Fechar</button>
    </div>
  {/if}
</Modal>

<!-- Edit modal -->
<Modal bind:open={showEdit} title="Editar Jogador">
  <form onsubmit={(e) => { e.preventDefault(); saveEdit(); }} class="space-y-4">
    <div class="form-group">
      <label class="label">Nome *</label>
      <input class="input" bind:value={editForm.name} required minlength="2" />
    </div>
    <div class="form-group">
      <label class="label">Apelido</label>
      <input class="input" bind:value={editForm.nickname} />
    </div>
    <div class="form-group">
      <label class="label">Perfil</label>
      <select class="input" bind:value={editForm.role}>
        <option value="player">Jogador</option>
        <option value="admin">Administrador</option>
      </select>
    </div>
    <div class="flex gap-3 justify-end pt-2">
      <button type="button" class="btn-secondary" onclick={() => showEdit = false}>Cancelar</button>
      <button type="submit" class="btn-primary" disabled={saving}>{saving ? 'Salvando…' : 'Salvar'}</button>
    </div>
  </form>
</Modal>
