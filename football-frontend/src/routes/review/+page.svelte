<script lang="ts">
  import { reviews as reviewsApi, ApiError } from '$lib/api';
  import type { ReviewResponse } from '$lib/api';
  import { toastError } from '$lib/stores/toast';
  import { goto } from '$app/navigation';
  import StarRating from '$lib/components/StarRating.svelte';
  import PageBackground from '$lib/components/PageBackground.svelte';
  import { X } from 'lucide-svelte';

  let rating = $state(0);
  let comment = $state('');
  let existing = $state<ReviewResponse | null>(null);
  let loading = $state(true);
  let saving = $state(false);
  let submitted = $state(false);

  $effect(() => {
    reviewsApi.getMe()
      .then(r => {
        existing = r;
        rating = r.rating;
        comment = r.comment ?? '';
      })
      .catch(e => {
        if (e instanceof ApiError && e.status === 404) {
          // Nunca avaliou — formulário em branco
        }
      })
      .finally(() => { loading = false; });
  });

  async function submit() {
    if (!rating) return;
    saving = true;
    try {
      const r = await reviewsApi.upsert(rating, comment.trim() || null);
      existing = r;
      submitted = true;
    } catch (e) {
      toastError(e instanceof ApiError ? e.message : 'Erro ao salvar avaliação');
    } finally {
      saving = false;
    }
  }

  function formatDate(iso: string) {
    return new Date(iso).toLocaleDateString('pt-BR');
  }
</script>

<svelte:head>
  <title>Avaliar o App — rachao.app</title>
</svelte:head>

<PageBackground>
<main class="relative z-10 max-w-lg mx-auto px-4 py-8">
  <div class="mb-6">
    <h1 class="text-2xl font-bold text-white">Avaliar o App</h1>
    <p class="text-sm text-gray-300 mt-0.5">Sua opinião nos ajuda a melhorar o Rachao</p>
  </div>

  {#if loading}
    <div class="card card-body flex items-center justify-center py-12">
      <span class="text-gray-400 text-sm">Carregando…</span>
    </div>
  {:else}
    <div class="card card-body">
      <div class="flex items-center justify-between mb-4">
        <h2 class="font-semibold text-gray-800 dark:text-gray-200">
          {submitted ? 'Obrigado!' : 'Como você avalia o Rachao?'}
        </h2>
        <button
          type="button"
          onclick={() => goto('/')}
          class="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 p-1 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
          aria-label="Fechar"
        >
          <X size={18} />
        </button>
      </div>

      {#if submitted}
        <div class="text-center py-6">
          <div class="text-5xl mb-4">🙏</div>
          <p class="text-gray-700 dark:text-gray-300 font-medium mb-1">Sua avaliação foi enviada!</p>
          <p class="text-sm text-gray-500 dark:text-gray-400 mb-1">
            Você nos deu
            <span class="font-semibold text-amber-500">{rating === 1 ? '1 estrela' : `${rating} estrelas`}</span>.
          </p>
          <p class="text-sm text-gray-500 dark:text-gray-400 mb-6">Seu feedback nos ajuda a melhorar o Rachao.</p>
          <div class="flex justify-center mb-4">
            <StarRating rating={existing?.rating ?? rating} readonly size={28} />
          </div>
          <button type="button" class="btn-secondary btn-sm" onclick={() => goto('/')}>
            Voltar ao início
          </button>
        </div>
      {:else}
        <form onsubmit={(e) => { e.preventDefault(); submit(); }} class="space-y-6">

          <div class="text-center">
            <div class="flex justify-center">
              <StarRating bind:rating size={40} />
            </div>
            {#if !rating}
              <p class="text-xs text-gray-400 mt-2">Toque em uma estrela para avaliar</p>
            {:else}
              <p class="text-xs text-primary-600 dark:text-primary-400 mt-2 font-medium">
                {rating === 1 ? 'Muito ruim' : rating === 2 ? 'Ruim' : rating === 3 ? 'Regular' : rating === 4 ? 'Bom' : 'Excelente'}
              </p>
            {/if}
          </div>

          <div class="form-group">
            <label class="label" for="comment">
              Comentário <span class="text-gray-400 font-normal">(opcional)</span>
            </label>
            <textarea
              id="comment"
              class="input resize-none"
              rows="4"
              bind:value={comment}
              placeholder="Conte o que achou, sugestões ou o que poderia melhorar…"
              maxlength="500"
              disabled={saving}
            ></textarea>
            <p class="text-xs text-gray-400 mt-1 text-right">{comment.length} / 500</p>
          </div>

          <button
            type="submit"
            class="btn-primary w-full justify-center py-2.5"
            disabled={saving || !rating}
          >
            {saving ? 'Salvando…' : existing ? 'Atualizar avaliação' : 'Enviar avaliação'}
          </button>

          {#if existing}
            <p class="text-xs text-gray-400 text-center">
              Você avaliou em {formatDate(existing.updated_at)} · pode atualizar a qualquer momento
            </p>
          {/if}
        </form>
      {/if}
    </div>
  {/if}
</main>
</PageBackground>
