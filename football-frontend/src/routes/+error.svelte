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
    <div class="overflow-hidden h-32 sm:h-44">
      <picture>
        <source srcset="/error-404.webp" type="image/webp" />
        <img
          src="/error-404.jpg"
          alt=""
          aria-hidden="true"
          width="1920"
          height="600"
          class="w-full h-full object-cover"
          style="object-position: 25% 55%"
        />
      </picture>
    </div>
    <div class="bg-primary-800 px-6 py-8 text-white text-center">
      <span class="inline-block text-xs font-bold bg-red-500 text-white px-2.5 py-0.5 rounded-full mb-3">
        Erro 404
      </span>
      <h1 class="text-2xl sm:text-3xl font-bold leading-snug">
        A bola saiu pela linha de fundo.
      </h1>
      <p class="text-primary-200 mt-2 text-sm">
        Essa página foi para o vestiário mais cedo que o esperado.
      </p>
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
