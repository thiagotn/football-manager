<script lang="ts">
  import { goto } from '$app/navigation';
  import { authStore, isAdmin, currentPlayer } from '$lib/stores/auth';
  import { admin as adminApi, players as playersApi, ApiError } from '$lib/api';
  import type { AdminPlayerItem } from '$lib/api';
  import { toastSuccess, toastError } from '$lib/stores/toast';
  import Modal from '$lib/components/Modal.svelte';
  import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
  import PageBackground from '$lib/components/PageBackground.svelte';
  import { Users, Plus, Search, Eye, EyeOff, Pencil, KeyRound, Trash2, Copy, ChevronLeft, ChevronRight } from 'lucide-svelte';
  import PhoneInput from '$lib/components/PhoneInput.svelte';

  const PAGE_SIZE = 20;

  let items = $state<AdminPlayerItem[]>([]);
  let total = $state(0);
  let page = $state(1);
  let search = $state('');
  let loading = $state(true);
  let error = $state('');

  // Detail modal
  let selected = $state<AdminPlayerItem | null>(null);
  let showDetail = $state(false);

  // Create modal
  let showCreate = $state(false);
  let createForm = $state({ name: '', nickname: '', whatsapp: '', password: '', role: 'player' });
  let showCreatePw = $state(false);
  let creating = $state(false);

  // Edit modal
  let showEdit = $state(false);
  let editForm = $state({ name: '', nickname: '', role: '' });
  let saving = $state(false);

  // Reset password modal
  let showReset = $state(false);
  let generatedPassword = $state('');
  let resetting = $state(false);
  let showResetPw = $state(false);

  // Confirm deactivate
  let confirmOpen = $state(false);

  let totalPages = $derived(Math.max(1, Math.ceil(total / PAGE_SIZE)));

  let searchTimer: ReturnType<typeof setTimeout>;

  function onSearchInput() {
    clearTimeout(searchTimer);
    searchTimer = setTimeout(() => {
      page = 1;
      load();
    }, 350);
  }

  async function load() {
    loading = true;
    error = '';
    try {
      const res = await adminApi.getPlayers({ search: search || undefined, page, page_size: PAGE_SIZE });
      items = res.items;
      total = res.total;
    } catch (e) {
      error = e instanceof ApiError ? e.message : 'Erro ao carregar jogadores.';
    }
    loading = false;
  }

  function goPage(p: number) {
    page = p;
    load();
  }

  let loaded = false;
  $effect(() => {
    if ($authStore.loading) return;
    if (!$isAdmin) { goto('/dashboard', { replaceState: true }); return; }
    if (loaded) return;
    loaded = true;
    load();
  });

  // ── Create ──────────────────────────────────────────────────
  async function create() {
    creating = true;
    try {
      await playersApi.create({ ...createForm, nickname: createForm.nickname || undefined });
      showCreate = false;
      createForm = { name: '', nickname: '', whatsapp: '', password: '', role: 'player' };
      toastSuccess('Jogador criado!');
      page = 1;
      await load();
    } catch (e) { toastError(e instanceof ApiError ? e.message : 'Erro ao criar.'); }
    creating = false;
  }

  // ── Detail ──────────────────────────────────────────────────
  function openDetail(p: AdminPlayerItem) {
    selected = p;
    showDetail = true;
  }

  // ── Edit ────────────────────────────────────────────────────
  function openEdit() {
    if (!selected) return;
    editForm = { name: selected.name, nickname: selected.nickname ?? '', role: selected.role };
    showDetail = false;
    showEdit = true;
  }

  async function saveEdit() {
    if (!selected) return;
    saving = true;
    try {
      await playersApi.update(selected.id, {
        name: editForm.name,
        nickname: editForm.nickname || undefined,
        role: editForm.role,
      });
      const updated = { ...selected, name: editForm.name, nickname: editForm.nickname || null, role: editForm.role };
      selected = updated;
      items = items.map(p => p.id === selected!.id ? { ...p, ...updated } : p);
      showEdit = false;
      showDetail = true;
      toastSuccess('Jogador atualizado!');
    } catch (e) { toastError(e instanceof ApiError ? e.message : 'Erro ao atualizar.'); }
    saving = false;
  }

  // ── Reset password ──────────────────────────────────────────
  function openReset() {
    generatedPassword = '';
    showResetPw = false;
    showDetail = false;
    showReset = true;
  }

  async function doReset() {
    if (!selected) return;
    resetting = true;
    try {
      const res = await playersApi.resetPassword(selected.id);
      generatedPassword = res.temp_password;
    } catch (e) {
      toastError(e instanceof ApiError ? e.message : 'Erro ao resetar senha.');
      showReset = false;
    }
    resetting = false;
  }

  function copyPassword() {
    navigator.clipboard.writeText(generatedPassword);
    toastSuccess('Senha copiada!');
  }

  // ── Deactivate ──────────────────────────────────────────────
  function askDeactivate() {
    showDetail = false;
    confirmOpen = true;
  }

  async function doDeactivate() {
    if (!selected) return;
    try {
      await playersApi.delete(selected.id);
      selected = { ...selected, active: false };
      items = items.map(p => p.id === selected!.id ? { ...p, active: false } : p);
      toastSuccess('Jogador desativado.');
    } catch { toastError('Erro ao desativar.'); }
  }

  // ── Helpers ─────────────────────────────────────────────────
  function fmtDate(iso: string): string {
    return new Date(iso).toLocaleDateString('pt-BR', { day: '2-digit', month: '2-digit', year: 'numeric' });
  }

  function planLabel(plan: string): string {
    return plan === 'free' ? 'Grátis' : plan.charAt(0).toUpperCase() + plan.slice(1);
  }
</script>

<svelte:head><title>Jogadores — rachao.app</title></svelte:head>

<PageBackground>
<main class="relative z-10 max-w-7xl mx-auto px-4 py-8">
  <div class="flex flex-wrap items-center justify-between gap-4 mb-6">
    <div>
      <h1 class="text-2xl font-bold text-white flex items-center gap-2">
        <Users size={24} class="text-primary-400" /> Jogadores
      </h1>
      <p class="text-sm text-white/60 mt-0.5">{total} jogadores cadastrados</p>
    </div>
    <button class="btn-primary" onclick={() => showCreate = true}>
      <Plus size={16} /> Novo Jogador
    </button>
  </div>

  <!-- Search -->
  <div class="relative mb-4 max-w-sm">
    <Search size={16} class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
    <input
      class="input pl-9"
      placeholder="Buscar por nome, apelido ou celular…"
      bind:value={search}
      oninput={onSearchInput}
    />
  </div>

  {#if error}
    <div class="alert-error mb-4">{error}</div>
  {/if}

  {#if loading}
    <div class="card overflow-hidden">
      {#each [1,2,3,4,5] as _}
        <div class="px-6 py-4 border-b border-gray-100 dark:border-gray-700 animate-pulse">
          <div class="h-4 bg-gray-100 dark:bg-gray-700 rounded w-1/3"></div>
        </div>
      {/each}
    </div>
  {:else}
    <div class="card overflow-x-auto">
      <table class="table">
        <thead>
          <tr>
            <th>Jogador</th>
            <th class="hidden sm:table-cell">Celular</th>
            <th class="hidden md:table-cell">Plano</th>
            <th class="hidden sm:table-cell">Cadastro</th>
            <th>Status</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {#each items as p}
            <tr>
              <td>
                <p class="font-medium flex items-center gap-1.5">
                  {p.nickname || p.name}
                  {#if p.role === 'admin'}
                    <span class="inline-flex items-center px-1.5 py-px rounded text-[10px] font-semibold bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400">Admin</span>
                  {/if}
                </p>
                {#if p.nickname}<p class="text-xs text-gray-400">{p.name}</p>{/if}
              </td>
              <td class="font-mono text-xs text-gray-600 hidden sm:table-cell">{p.whatsapp}</td>
              <td class="hidden md:table-cell">
                <span class="badge {p.plan === 'free' ? 'badge-gray' : 'badge-blue'}">
                  {planLabel(p.plan)}
                </span>
              </td>
              <td class="text-xs text-gray-500 hidden sm:table-cell">{fmtDate(p.created_at)}</td>
              <td>
                <span class="badge {p.active ? 'badge-green' : 'badge-red'}">
                  {p.active ? 'Ativo' : 'Inativo'}
                </span>
              </td>
              <td>
                <button
                  onclick={() => openDetail(p)}
                  class="btn-sm btn-ghost flex items-center gap-1 border border-gray-200 dark:border-gray-700"
                >
                  <Eye size={12} /> Detalhes
                </button>
              </td>
            </tr>
          {/each}
          {#if items.length === 0}
            <tr><td colspan="6" class="text-center text-gray-400 py-8">Nenhum jogador encontrado.</td></tr>
          {/if}
        </tbody>
      </table>
    </div>

    <!-- Pagination -->
    {#if totalPages > 1}
      <div class="flex flex-wrap items-center justify-between gap-3 mt-4 text-sm text-gray-400">
        <span class="text-white/70">Página {page} de {totalPages} — {total} registros</span>
        <div class="flex gap-2">
          <button
            onclick={() => goPage(page - 1)}
            disabled={page === 1}
            class="btn-secondary btn-sm flex items-center gap-1 disabled:opacity-40"
          >
            <ChevronLeft size={14} /> Anterior
          </button>
          <button
            onclick={() => goPage(page + 1)}
            disabled={page === totalPages}
            class="btn-secondary btn-sm flex items-center gap-1 disabled:opacity-40"
          >
            Próxima <ChevronRight size={14} />
          </button>
        </div>
      </div>
    {/if}
  {/if}
</main>
</PageBackground>

<!-- Detail modal -->
<Modal bind:open={showDetail} title="Detalhes do Cadastro">
  {#if selected}
    <div class="space-y-4">
      <div class="grid grid-cols-2 gap-3 text-sm">
        <div>
          <p class="text-xs text-gray-400 mb-0.5">Nome</p>
          <p class="font-medium">{selected.name}</p>
        </div>
        <div>
          <p class="text-xs text-gray-400 mb-0.5">Apelido</p>
          <p class="font-medium">{selected.nickname || '—'}</p>
        </div>
        <div>
          <p class="text-xs text-gray-400 mb-0.5">Celular</p>
          <p class="font-mono">{selected.whatsapp}</p>
        </div>
        <div>
          <p class="text-xs text-gray-400 mb-0.5">Perfil</p>
          <span class="badge {selected.role === 'admin' ? 'badge-blue' : 'badge-gray'}">
            {selected.role === 'admin' ? 'Admin' : 'Jogador'}
          </span>
        </div>
        <div>
          <p class="text-xs text-gray-400 mb-0.5">Plano</p>
          <span class="badge {selected.plan === 'free' ? 'badge-gray' : 'badge-blue'}">
            {planLabel(selected.plan)}
          </span>
        </div>
        <div>
          <p class="text-xs text-gray-400 mb-0.5">Grupos</p>
          <p class="font-medium">{selected.total_groups}</p>
        </div>
        <div>
          <p class="text-xs text-gray-400 mb-0.5">Cadastrado em</p>
          <p>{fmtDate(selected.created_at)}</p>
        </div>
        <div>
          <p class="text-xs text-gray-400 mb-0.5">Status</p>
          <span class="badge {selected.active ? 'badge-green' : 'badge-red'}">
            {selected.active ? 'Ativo' : 'Inativo'}
          </span>
        </div>
      </div>

      <div class="border-t border-gray-100 dark:border-gray-700 pt-4 flex flex-wrap gap-2">
        <button onclick={openEdit} class="btn-sm btn-ghost flex items-center gap-1 border border-blue-200 text-blue-600 hover:bg-blue-50 dark:border-blue-800 dark:text-blue-400">
          <Pencil size={14} /> Editar
        </button>
        <button onclick={openReset} class="btn-sm btn-ghost flex items-center gap-1 border border-amber-200 text-amber-600 hover:bg-amber-50 dark:border-amber-800 dark:text-amber-400">
          <KeyRound size={14} /> Resetar Senha
        </button>
        {#if selected.active && selected.id !== $currentPlayer?.id && selected.role !== 'admin'}
          <button onclick={askDeactivate} class="btn-sm btn-ghost flex items-center gap-1 border border-red-200 text-red-500 hover:bg-red-50 dark:border-red-800 dark:text-red-400">
            <Trash2 size={14} /> Desativar
          </button>
        {/if}
      </div>
    </div>
  {/if}
</Modal>

<!-- Create modal -->
<Modal bind:open={showCreate} title="Novo Jogador">
  <form onsubmit={(e) => { e.preventDefault(); create(); }} class="space-y-4">
    <div class="form-group">
      <label class="label">Nome *</label>
      <input class="input" bind:value={createForm.name} required minlength="2" />
    </div>
    <div class="form-group">
      <label class="label">Apelido</label>
      <input class="input" bind:value={createForm.nickname} placeholder="Opcional" />
    </div>
    <div class="form-group">
      <label class="label">Celular *</label>
      <PhoneInput bind:value={createForm.whatsapp} placeholder="11999990000" required />
    </div>
    <div class="form-group">
      <label class="label">Senha *</label>
      <div class="relative">
        <input class="input pr-10" type={showCreatePw ? 'text' : 'password'} bind:value={createForm.password} required minlength="6" />
        <button type="button" onclick={() => showCreatePw = !showCreatePw} class="absolute right-2.5 top-1/2 -translate-y-1/2 text-gray-400">
          {#if showCreatePw}<EyeOff size={16} />{:else}<Eye size={16} />{/if}
        </button>
      </div>
    </div>
    <div class="form-group">
      <label class="label">Perfil</label>
      <select class="input" bind:value={createForm.role}>
        <option value="player">Jogador</option>
        <option value="admin">Administrador</option>
      </select>
    </div>
    <div class="flex gap-3 justify-end pt-2">
      <button type="button" class="btn-secondary" onclick={() => showCreate = false}>Cancelar</button>
      <button type="submit" class="btn-primary" disabled={creating}>{creating ? 'Criando…' : 'Criar'}</button>
    </div>
  </form>
</Modal>

<!-- Edit modal -->
<Modal bind:open={showEdit} title="Editar — {selected?.name ?? ''}">
  <form onsubmit={(e) => { e.preventDefault(); saveEdit(); }} class="space-y-4">
    <div class="form-group">
      <label class="label">Nome *</label>
      <input class="input" bind:value={editForm.name} required minlength="2" />
    </div>
    <div class="form-group">
      <label class="label">Apelido</label>
      <input class="input" bind:value={editForm.nickname} placeholder="Opcional" />
    </div>
    <div class="form-group">
      <label class="label">Perfil</label>
      <select class="input" bind:value={editForm.role}>
        <option value="player">Jogador</option>
        <option value="admin">Administrador</option>
      </select>
    </div>
    <div class="flex gap-3 justify-end pt-2">
      <button type="button" class="btn-secondary" onclick={() => { showEdit = false; showDetail = true; }}>Cancelar</button>
      <button type="submit" class="btn-primary" disabled={saving}>{saving ? 'Salvando…' : 'Salvar'}</button>
    </div>
  </form>
</Modal>

<!-- Reset password modal -->
<Modal bind:open={showReset} title="Resetar Senha — {selected?.name ?? ''}">
  {#if !generatedPassword}
    <p class="text-sm text-gray-600 dark:text-gray-300 mb-4">
      Uma senha temporária será gerada para <strong>{selected?.name}</strong>.
      O jogador precisará alterá-la no próximo acesso.
    </p>
    <div class="flex gap-3 justify-end">
      <button class="btn-secondary" onclick={() => { showReset = false; showDetail = true; }}>Cancelar</button>
      <button class="btn bg-amber-500 hover:bg-amber-600 text-white flex items-center gap-1" onclick={doReset} disabled={resetting}>
        <KeyRound size={15} /> {resetting ? 'Gerando…' : 'Gerar senha temporária'}
      </button>
    </div>
  {:else}
    <p class="text-sm text-gray-600 dark:text-gray-300 mb-3">Senha temporária gerada. Compartilhe com o jogador:</p>
    <div class="flex items-center gap-2 mb-4">
      <code class="flex-1 bg-gray-100 dark:bg-gray-700 rounded-lg px-4 py-3 font-mono text-lg tracking-widest text-center text-gray-900 dark:text-gray-100 select-all">
        {generatedPassword}
      </code>
      <button class="btn-secondary shrink-0 flex items-center gap-1" onclick={copyPassword}>
        <Copy size={15} /> Copiar
      </button>
    </div>
    <p class="text-xs text-amber-700 bg-amber-50 border border-amber-200 rounded-lg px-3 py-2 mb-4">
      ⚠️ O jogador será obrigado a alterar esta senha no próximo acesso.
    </p>
    <div class="flex justify-end">
      <button class="btn-primary" onclick={() => { showReset = false; showDetail = true; }}>Fechar</button>
    </div>
  {/if}
</Modal>

<!-- Confirm deactivate -->
<ConfirmDialog
  bind:open={confirmOpen}
  message="Desativar {selected?.name ?? 'este jogador'}? Ele não conseguirá mais fazer login."
  confirmLabel="Desativar"
  danger={true}
  onConfirm={doDeactivate}
/>
