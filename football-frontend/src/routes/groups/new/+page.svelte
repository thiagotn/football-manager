<script lang="ts">
  import { goto } from '$app/navigation';
  import { groups as groupsApi } from '$lib/api';
  import { toastSuccess, toastError } from '$lib/stores/toast';

  let name = $state('');
  let description = $state('');
  let voteOpenDelay = $state(20);
  let voteDuration = $state(24);
  let loading = $state(false);
  let error = $state('');

  async function handleCreate() {
    error = '';
    loading = true;
    try {
      const g = await groupsApi.create({
        name,
        description: description || undefined,
        vote_open_delay_minutes: voteOpenDelay,
        vote_duration_hours: voteDuration,
      });
      toastSuccess('Grupo criado com sucesso!');
      goto(`/groups/${g.id}`);
    } catch (e: any) {
      error = e.message ?? 'Erro ao criar grupo';
    } finally {
      loading = false;
    }
  }
</script>

<svelte:head><title>Novo Grupo — rachao.app</title></svelte:head>

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

      <div class="border-t border-gray-100 pt-4">
        <p class="text-sm font-medium text-gray-700 mb-3">Configurações de votação</p>
        <div class="space-y-3">
          <div>
            <label class="block text-sm text-gray-600 mb-1">Após o termino da partida, será aberta em</label>
            <select bind:value={voteOpenDelay} class="input">
              <option value={0}>Imediato (sem atraso)</option>
              <option value={10}>10 minutos</option>
              <option value={20}>20 minutos (padrão)</option>
              <option value={30}>30 minutos</option>
              <option value={60}>1 hora</option>
            </select>
          </div>
          <div>
            <label class="block text-sm text-gray-600 mb-1">Duração da votação</label>
            <select bind:value={voteDuration} class="input">
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
