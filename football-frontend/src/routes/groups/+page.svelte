<script lang="ts">
  import { groups as groupsApi, subscriptions as subsApi, ApiError } from '$lib/api';
  import type { Group, SubscriptionInfo } from '$lib/api';
  import { isAdmin } from '$lib/stores/auth';
  import { toastSuccess, toastError } from '$lib/stores/toast';
  import Modal from '$lib/components/Modal.svelte';
  import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
  import UpsellModal from '$lib/components/UpsellModal.svelte';
  import { Plus, Trophy, ChevronRight, Trash2, Lock } from 'lucide-svelte';
  import PageBackground from '$lib/components/PageBackground.svelte';

  let groupList: Group[] = $state([]);
  let sub: SubscriptionInfo | null = $state(null);
  let loading = $state(true);
  let loadError = $state('');
  let showCreate = $state(false);
  let form = $state({ name: '', description: '', slug: '' });
  let saving = $state(false);

  let showUpsell = $state(false);
  let upsellMessage = $state('');

  let confirmOpen = $state(false);
  let confirmMessage = $state('');
  let confirmAction = $state<() => void>(() => {});

  // Limite atingido quando groups_used >= groups_limit (e limit não é null)
  let atGroupLimit = $derived(
    !$isAdmin && sub !== null && sub.groups_limit !== null && sub.groups_used >= sub.groups_limit
  );

  function askConfirm(message: string, action: () => void) {
    confirmMessage = message;
    confirmAction = action;
    confirmOpen = true;
  }

  $effect(() => {
    let cancelled = false;
    (async () => {
      try {
        const [data, subData] = await Promise.all([
          groupsApi.list(),
          $isAdmin ? Promise.resolve(null) : subsApi.me(),
        ]);
        if (!cancelled) {
          groupList = data;
          sub = subData;
        }
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

  function openCreateOrUpsell() {
    if (atGroupLimit) {
      upsellMessage = `Você já possui ${sub!.groups_used} grupo${sub!.groups_used !== 1 ? 's' : ''} ativo${sub!.groups_used !== 1 ? 's' : ''}, o máximo do plano Free. Planos com mais grupos estarão disponíveis em breve.`;
      showUpsell = true;
    } else {
      showCreate = true;
    }
  }

  async function createGroup() {
    saving = true;
    try {
      const g = await groupsApi.create({ name: form.name, description: form.description || undefined, slug: form.slug || undefined });
      groupList = [g, ...groupList];
      if (sub) sub = { ...sub, groups_used: sub.groups_used + 1 };
      showCreate = false;
      form = { name: '', description: '', slug: '' };
      toastSuccess('Grupo criado com sucesso!');
    } catch (e) {
      if (e instanceof ApiError && e.status === 403 && e.message === 'PLAN_LIMIT_EXCEEDED') {
        showCreate = false;
        upsellMessage = 'Você atingiu o limite de grupos do plano Free.';
        showUpsell = true;
      } else {
        toastError(e instanceof ApiError ? e.message : 'Erro ao criar grupo');
      }
    }
    saving = false;
  }

  async function deleteGroup(g: Group) {
    askConfirm(`Excluir "${g.name}"? Esta ação não pode ser desfeita.`, async () => {
      try {
        await groupsApi.delete(g.id);
        groupList = groupList.filter(x => x.id !== g.id);
        if (sub) sub = { ...sub, groups_used: Math.max(0, sub.groups_used - 1) };
        toastSuccess('Grupo excluído');
      } catch (e) {
        toastError('Erro ao excluir grupo');
      }
    });
  }
</script>

<svelte:head><title>Grupos — rachao.app</title></svelte:head>

<PageBackground>
<main class="relative z-10 max-w-7xl mx-auto px-4 py-8">
  <div class="flex items-center justify-between mb-6">
    <div>
      <h1 class="text-2xl font-bold text-white flex items-center gap-2">
        <Trophy size={24} class="text-primary-400" /> Grupos
      </h1>
      <p class="text-sm text-gray-300 mt-0.5">Grupos de futebol que você participa</p>
    </div>
    <div class="flex flex-col items-end gap-1">
      {#if !$isAdmin && sub}
        <p class="text-xs {sub.groups_used >= (sub.groups_limit ?? Infinity) ? 'text-red-300' : sub.groups_used >= (sub.groups_limit ?? Infinity) * 0.8 ? 'text-yellow-300' : 'text-gray-400'}">
          {sub.groups_used} de {sub.groups_limit} grupo{sub.groups_limit !== 1 ? 's' : ''}
        </p>
      {/if}
      <button class="btn-primary {atGroupLimit ? 'opacity-80' : ''}" onclick={openCreateOrUpsell}>
        {#if atGroupLimit}
          <Lock size={15} /> Novo Grupo
        {:else}
          <Plus size={16} /> Novo Grupo
        {/if}
      </button>
    </div>
  </div>

  {#if loadError}
    <div class="alert-error mb-4">Erro ao carregar grupos: <strong>{loadError}</strong></div>
  {/if}

  {#if loading}
    <div class="grid sm:grid-cols-2 lg:grid-cols-3 gap-4">
      {#each [1,2,3] as _}
        <div class="card p-6 animate-pulse">
          <div class="h-5 bg-gray-100 dark:bg-gray-700 rounded w-2/3 mb-3"></div>
          <div class="h-3 bg-gray-100 dark:bg-gray-700 rounded w-full"></div>
        </div>
      {/each}
    </div>
  {:else if groupList.length === 0}
    <div class="card p-12 text-center">
      <Trophy size={40} class="text-gray-300 mx-auto mb-3" />
      <p class="text-gray-500">Nenhum grupo encontrado.</p>
      <button class="btn-primary mt-4" onclick={openCreateOrUpsell}>
        {#if atGroupLimit}<Lock size={15} />{:else}<Plus size={16} />{/if}
        Criar primeiro grupo
      </button>
    </div>
  {:else}
    <div class="grid sm:grid-cols-2 lg:grid-cols-3 gap-4">
      {#each groupList as g}
        <div class="card hover:shadow-md transition-shadow">
          <a href="/groups/{g.id}" class="card-body block">
            <div class="flex items-start justify-between">
              <div>
                <h3 class="font-semibold text-gray-900 dark:text-gray-100">{g.name}</h3>
                {#if g.description}<p class="text-sm text-gray-500 dark:text-gray-400 mt-1 line-clamp-2">{g.description}</p>{/if}
              </div>
              <ChevronRight size={18} class="text-gray-400 shrink-0 ml-2 mt-0.5" />
            </div>
            <span class="inline-block mt-3 text-xs font-mono bg-gray-100 dark:bg-gray-700 text-gray-500 dark:text-gray-400 px-2 py-0.5 rounded">{g.slug}</span>
          </a>
          {#if $isAdmin}
            <div class="px-6 pb-4 flex justify-end">
              <button onclick={() => deleteGroup(g)} class="btn-sm btn-ghost text-red-500 hover:bg-red-50">
                <Trash2 size={14} /> Excluir
              </button>
            </div>
          {/if}
        </div>
      {/each}
    </div>
  {/if}
</main>
</PageBackground>

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
      <p class="text-xs text-gray-400 dark:text-gray-500 mt-1">Apenas letras, números e hífens</p>
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

<UpsellModal bind:open={showUpsell} message={upsellMessage} />
