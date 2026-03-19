<script lang="ts">
  import { onMount } from 'svelte';
  import { pwaInstall } from '$lib/stores/pwaInstall';
  import { X, Download, Share2 } from 'lucide-svelte';

  let dismissed = $state(false);
  let showIosSteps = $state(false);

  onMount(() => {
    dismissed = !!localStorage.getItem('pwa_banner_dismissed');
  });

  function dismiss() {
    dismissed = true;
    localStorage.setItem('pwa_banner_dismissed', '1');
  }

  let visible = $derived(!dismissed && ($pwaInstall.canInstall || $pwaInstall.isIos));
</script>

{#if visible}
  <div
    class="fixed bottom-0 inset-x-0 z-50 bg-primary-900/95 backdrop-blur border-t border-primary-700/60 shadow-2xl"
    style="padding-bottom: env(safe-area-inset-bottom)"
  >
    <div class="max-w-lg mx-auto px-4 py-3">

      <!-- Linha principal -->
      <div class="flex items-center gap-3">
        <img src="/logo.png" alt="rachao.app" class="w-10 h-10 object-contain shrink-0 rounded-xl" />
        <div class="flex-1 min-w-0">
          <p class="text-sm font-semibold text-white leading-tight">rachao.app</p>
          <p class="text-xs text-primary-300 truncate">Adicione à tela inicial para acesso rápido</p>
        </div>

        {#if $pwaInstall.canInstall}
          <button
            onclick={() => pwaInstall.install()}
            class="shrink-0 inline-flex items-center gap-1.5 px-4 py-1.5 bg-primary-500 hover:bg-primary-400 active:bg-primary-600 text-white text-sm font-semibold rounded-lg transition-colors"
          >
            <Download size={13} />
            Instalar
          </button>
        {:else if $pwaInstall.isIos}
          <button
            onclick={() => showIosSteps = !showIosSteps}
            class="shrink-0 inline-flex items-center gap-1.5 px-4 py-1.5 bg-primary-500 hover:bg-primary-400 active:bg-primary-600 text-white text-sm font-semibold rounded-lg transition-colors"
          >
            <Download size={13} />
            {showIosSteps ? 'Fechar' : 'Instalar'}
          </button>
        {/if}

        <button
          onclick={dismiss}
          class="shrink-0 p-1.5 rounded-lg text-primary-400 hover:text-white hover:bg-primary-700 transition-colors"
          aria-label="Fechar"
        >
          <X size={16} />
        </button>
      </div>

      <!-- Passo a passo iOS (expansível) -->
      {#if showIosSteps && $pwaInstall.isIos}
        <div class="mt-3 pt-3 border-t border-primary-700/60 text-xs text-primary-300 space-y-1.5">
          <p class="font-semibold text-primary-100 mb-2">Para instalar no iPhone / iPad:</p>
          <ol class="list-decimal list-inside space-y-1.5 leading-relaxed">
            <li>Abra o rachao.app no <strong class="text-white">Safari</strong> — não funciona em outros navegadores no iOS</li>
            <li>Toque em <span class="inline-flex items-center gap-1 text-white font-medium">Compartilhar <Share2 size={10} class="inline shrink-0" /></span> na barra inferior</li>
            <li>Role e toque em <strong class="text-white">"Adicionar à Tela de Início"</strong></li>
            <li>Confirme tocando em <strong class="text-white">"Adicionar"</strong></li>
          </ol>
        </div>
      {/if}

    </div>
  </div>
{/if}
