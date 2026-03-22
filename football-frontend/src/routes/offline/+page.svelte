<script lang="ts">
  import { t } from '$lib/i18n';
</script>

<svelte:head>
  <title>Indisponível — rachao.app</title>
</svelte:head>

<div class="min-h-screen bg-gray-900 flex flex-col items-center justify-center px-6 text-center">
  <img src="/logo-192.png" alt="rachao.app" class="w-20 h-20 rounded-2xl mb-8 opacity-90" />

  <h1 class="text-2xl font-bold text-white mb-2">{$t('offline.title')}</h1>
  <p class="text-gray-400 text-sm mb-1 max-w-xs">
    {$t('offline.desc')}
  </p>
  <p class="text-gray-500 text-xs mb-10 max-w-xs">
    {$t('offline.hint')}
  </p>

  <button
    onclick={() => window.location.reload()}
    class="bg-primary-600 hover:bg-primary-700 active:bg-primary-800 text-white font-semibold px-8 py-3 rounded-xl transition-colors text-sm"
  >
    {$t('offline.retry')}
  </button>
</div>
