<svelte:head>
  <title>Sem conexão — rachao.app</title>
</svelte:head>

<div class="min-h-screen bg-gray-900 flex flex-col items-center justify-center px-4 text-center">
  <img src="/logo.png" alt="rachao.app" class="w-36 mb-8 opacity-80" />
  <h1 class="text-xl font-bold text-white mb-2">Sem conexão</h1>
  <p class="text-gray-400 text-sm mb-8 max-w-xs">
    Você está offline. Verifique sua conexão e tente novamente.
  </p>
  <button
    onclick={() => window.location.reload()}
    class="bg-primary-600 hover:bg-primary-700 text-white font-medium px-6 py-2.5 rounded-lg transition-colors"
  >
    Tentar novamente
  </button>
</div>
