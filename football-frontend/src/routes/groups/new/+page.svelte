<script lang="ts">
  import { goto } from '$app/navigation';
  import { groups as groupsApi } from '$lib/api';
  import { toastSuccess, toastError } from '$lib/stores/toast';

  let name = $state('');
  let description = $state('');
  let loading = $state(false);
  let error = $state('');

  async function handleCreate() {
    error = '';
    loading = true;
    try {
      const g = await groupsApi.create({ name, description: description || undefined });
      toastSuccess('Grupo criado com sucesso!');
      goto(`/groups/${g.id}`);
    } catch (e: any) {
      error = e.message ?? 'Erro ao criar grupo';
    } finally {
      loading = false;
    }
  }
</script>

<svelte:head><title>Novo Grupo — Joga Bonito</title></svelte:head>

<main class="max-w-xl mx-auto px-4 py-8">
  <div class="mb-6">
    <a href="/groups" class="text-sm text-gray-500 hover:text-gray-700">← Voltar</a>
    <h1 class="text-2xl font-bold text-gray-900 mt-2">Novo Grupo</h1>
  </div>

  <div class="card p-6">
    <form onsubmit={(e) => { e.preventDefault(); handleCreate(); }} class="space-y-4">
      <div>
        <label class="block text-sm font-medium text-gray-700 mb-1">Nome do grupo *</label>
        <input
          type="text"
          bind:value={name}
          class="input"
          placeholder="Ex: Futebol GQC"
          required
          maxlength="100"
        />
      </div>
      <div>
        <label class="block text-sm font-medium text-gray-700 mb-1">Descrição</label>
        <textarea
          bind:value={description}
          class="input resize-none"
          rows="3"
          placeholder="Descrição opcional do grupo"
          maxlength="500"
        ></textarea>
      </div>

      {#if error}
        <div class="bg-red-50 border border-red-200 text-red-700 text-sm px-3 py-2 rounded-lg">{error}</div>
      {/if}

      <div class="flex gap-3 pt-2">
        <a href="/groups" class="btn-secondary flex-1 justify-center">Cancelar</a>
        <button type="submit" class="btn-primary flex-1" disabled={loading || !name.trim()}>
          {loading ? 'Criando...' : 'Criar Grupo'}
        </button>
      </div>
    </form>
  </div>
</main>
