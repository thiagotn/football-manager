<script lang="ts">
  import { page } from '$app/stores';
</script>

<svelte:head>
  <title>Erro {$page.status} — rachao.app</title>
</svelte:head>

<div class="min-h-screen bg-gray-50">

  <!-- Beta banner -->
  <div class="bg-yellow-400 text-yellow-900 text-xs font-medium text-center py-1.5 px-4">
    Versao Beta — produto em desenvolvimento.
  </div>

  <!-- Header (idêntico ao da página de partida e FAQ) -->
  <div class="bg-primary-700 text-white py-4 px-4">
    <div class="flex items-center justify-center gap-2">
      <span class="text-xl">⚽</span>
      <span class="font-semibold text-lg tracking-tight">rachao.app</span>
      <span class="text-xs font-bold bg-yellow-400 text-yellow-900 px-1.5 py-0.5 rounded-full">Beta</span>
    </div>
  </div>

  <!-- Banner de erro -->
  {#if $page.status === 404}
    <div class="relative overflow-hidden" style="min-height: 240px;">
      <picture>
        <source srcset="/error-404.webp" type="image/webp" />
        <img
          src="/error-404.jpg"
          alt=""
          aria-hidden="true"
          width="1920"
          height="600"
          class="absolute inset-0 w-full h-full object-cover object-center"
        />
      </picture>
      <div class="absolute inset-0 bg-primary-900/60"></div>
      <div class="relative px-6 py-12 text-white text-center max-w-2xl mx-auto">
        <span class="inline-block text-xs font-bold bg-red-500 text-white px-2.5 py-0.5 rounded-full mb-4">
          Erro 404
        </span>
        <h1 class="text-3xl sm:text-4xl font-bold leading-snug">
          A bola saiu pela<br>linha de fundo.
        </h1>
        <p class="text-primary-200 mt-3 text-sm">
          Essa página foi para o vestiário mais cedo que o esperado.
        </p>
      </div>
    </div>

  {:else}
    <div class="bg-primary-800 px-6 py-12 text-white text-center">
      <span class="inline-block text-xs font-bold bg-red-500 text-white px-2.5 py-0.5 rounded-full mb-4">
        Erro {$page.status}
      </span>
      <h1 class="text-3xl font-bold">O jogo parou.</h1>
      <p class="text-primary-200 mt-3 text-sm">Algo deu errado no meio do campo.</p>
    </div>
  {/if}

  <!-- Card com detalhe e CTA -->
  <main class="max-w-md mx-auto px-4 py-8">
    <div class="card card-body text-center">
      {#if $page.status === 404}
        <p class="text-gray-600 text-sm leading-relaxed">
          A página que você tentou acessar não existe ou foi removida do jogo.<br>
          Confira o endereço ou volte para o início.
        </p>
      {:else}
        <p class="text-gray-600 text-sm leading-relaxed">
          {$page.error?.message ?? 'Ocorreu um erro inesperado. Tente novamente mais tarde.'}
        </p>
      {/if}

      <a href="/lp" class="btn btn-primary mt-5 w-full justify-center">
        Voltar ao início
      </a>
    </div>
  </main>

</div>
