<script lang="ts">
  import { groups as groupsApi, ApiError } from '$lib/api';
  import type { Group } from '$lib/api';
  import { isAdmin } from '$lib/stores/auth';
  import { toastSuccess, toastError } from '$lib/stores/toast';
  import Modal from '$lib/components/Modal.svelte';
  import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
  import { Plus, Trophy, ChevronRight, Trash2 } from 'lucide-svelte';

  let groupList: Group[] = $state([]);
  let loading = $state(true);
  let loadError = $state('');
  let showCreate = $state(false);
  let form = $state({ name: '', description: '', slug: '' });
  let saving = $state(false);

  let confirmOpen = $state(false);
  let confirmMessage = $state('');
  let confirmAction = $state<() => void>(() => {});

  function askConfirm(message: string, action: () => void) {
    confirmMessage = message;
    confirmAction = action;
    confirmOpen = true;
  }

  $effect(() => {
    let cancelled = false;
    (async () => {
      try {
        const data = await groupsApi.list();
        if (!cancelled) groupList = data;
      } catch (e) {
        if (!cancelled) {
          console.error('[groups] erro ao carregar:', e);
          loadError = e instanceof ApiError ? `${e.status}: ${e.message}` : String(e);
          toastError('Erro ao carregar grupos');
        }
      }
      if (!cancelled) loading = false;
    })();
    return () => { cancelled = true; };
  });

  async function createGroup() {
    saving = true;
    try {
      const g = await groupsApi.create({ name: form.name, description: form.description || undefined, slug: form.slug || undefined });
      groupList = [g, ...groupList];
      showCreate = false;
      form = { name: '', description: '', slug: '' };
      toastSuccess('Grupo criado com sucesso!');
    } catch (e) {
      toastError(e instanceof ApiError ? e.message : 'Erro ao criar grupo');
    }
    saving = false;
  }

  async function deleteGroup(g: Group) {
    askConfirm(`Excluir "${g.name}"? Esta ação não pode ser desfeita.`, async () => {
      try {
        await groupsApi.delete(g.id);
        groupList = groupList.filter(x => x.id !== g.id);
        toastSuccess('Grupo excluído');
      } catch (e) {
        toastError('Erro ao excluir grupo');
      }
    });
  }
</script>

<svelte:head><title>Grupos — rachao.app</title></svelte:head>

<main class="max-w-7xl mx-auto px-4 py-8">
  <div class="flex items-center justify-between mb-6">
    <div>
      <h1 class="text-2xl font-bold text-gray-900 flex items-center gap-2">
        <Trophy size={24} class="text-primary-600" /> Grupos
      </h1>
      <p class="text-sm text-gray-500 mt-0.5">Grupos de futebol que você participa</p>
    </div>
    {#if $isAdmin}
      <button class="btn-primary" onclick={() => showCreate = true}>
        <Plus size={16} /> Novo Grupo
      </button>
    {/if}
  </div>

  {#if loadError}
    <div class="alert-error mb-4">Erro ao carregar grupos: <strong>{loadError}</strong></div>
  {/if}

  {#if loading}
    <div class="grid sm:grid-cols-2 lg:grid-cols-3 gap-4">
      {#each [1,2,3] as _}
        <div class="card p-6 animate-pulse">
          <div class="h-5 bg-gray-100 rounded w-2/3 mb-3"></div>
          <div class="h-3 bg-gray-100 rounded w-full"></div>
        </div>
      {/each}
    </div>
  {:else if groupList.length === 0}
    <div class="card p-12 text-center">
      <Trophy size={40} class="text-gray-300 mx-auto mb-3" />
      <p class="text-gray-500">Nenhum grupo encontrado.</p>
      {#if $isAdmin}<button class="btn-primary mt-4" onclick={() => showCreate = true}><Plus size={16} /> Criar primeiro grupo</button>{/if}
    </div>
  {:else}
    <div class="grid sm:grid-cols-2 lg:grid-cols-3 gap-4">
      {#each groupList as g}
        <div class="card hover:shadow-md transition-shadow">
          <a href="/groups/{g.id}" class="card-body block">
            <div class="flex items-start justify-between">
              <div>
                <h3 class="font-semibold text-gray-900">{g.name}</h3>
                {#if g.description}<p class="text-sm text-gray-500 mt-1 line-clamp-2">{g.description}</p>{/if}
              </div>
              <ChevronRight size={18} class="text-gray-400 shrink-0 ml-2 mt-0.5" />
            </div>
            <p class="text-xs text-gray-400 mt-3">/{g.slug}</p>
          </a>
          {#if $isAdmin}
            <div class="px-6 pb-4 flex justify-end">
              <button onclick={() => deleteGroup(g)} class="btn-icon btn-ghost text-red-500 hover:bg-red-50" title="Excluir grupo">
                <Trash2 size={16} />
              </button>
            </div>
          {/if}
        </div>
      {/each}
    </div>
  {/if}
</main>

<Modal bind:open={showCreate} title="Novo Grupo">
  <form onsubmit={(e) => { e.preventDefault(); createGroup(); }} class="space-y-4">
    <div class="form-group">
      <label class="label" for="name">Nome do Grupo *</label>
      <input id="name" class="input" bind:value={form.name} placeholder="Ex: Futebol GQC" required />
    </div>
    <div class="form-group">
      <label class="label" for="desc">Descrição</label>
      <textarea id="desc" class="input resize-none" rows="2" bind:value={form.description} placeholder="Descrição opcional…"></textarea>
    </div>
    <div class="form-group">
      <label class="label" for="slug">Slug (URL)</label>
      <input id="slug" class="input" bind:value={form.slug} placeholder="futebol-gqc (gerado automaticamente)" />
      <p class="text-xs text-gray-400 mt-1">Apenas letras, números e hífens</p>
    </div>
    <div class="flex gap-3 justify-end pt-2">
      <button type="button" class="btn-secondary" onclick={() => showCreate = false}>Cancelar</button>
      <button type="submit" class="btn-primary" disabled={saving}>{saving ? 'Criando…' : 'Criar Grupo'}</button>
    </div>
  </form>
</Modal>

<ConfirmDialog
  bind:open={confirmOpen}
  message={confirmMessage}
  confirmLabel="Excluir"
  onConfirm={confirmAction}
/>
