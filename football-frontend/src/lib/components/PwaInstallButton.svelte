<script lang="ts">
  import { pwaInstall } from '$lib/stores/pwaInstall';
  import { Download, Share2, ChevronDown, ChevronUp } from 'lucide-svelte';

  let showIosInstructions = $state(false);
</script>

{#if $pwaInstall.canInstall}
  <button
    onclick={() => pwaInstall.install()}
    class="w-full flex items-center gap-3 px-3 py-3 rounded-xl text-sm font-medium transition-colors text-emerald-400 hover:bg-primary-700 text-left"
  >
    <Download size={18} />
    Instalar App
  </button>

{:else if $pwaInstall.isIos}
  <button
    onclick={() => showIosInstructions = !showIosInstructions}
    class="w-full flex items-center gap-3 px-3 py-3 rounded-xl text-sm font-medium transition-colors text-emerald-400 hover:bg-primary-700 text-left"
  >
    <Download size={18} />
    <span class="flex-1">Instalar App</span>
    {#if showIosInstructions}
      <ChevronUp size={15} class="text-primary-400 shrink-0" />
    {:else}
      <ChevronDown size={15} class="text-primary-400 shrink-0" />
    {/if}
  </button>

  {#if showIosInstructions}
    <div class="mx-3 mb-1 rounded-xl bg-primary-900/70 px-4 py-3 text-xs text-primary-200 space-y-2">
      <p class="font-medium text-primary-100">Para instalar no Safari:</p>
      <ol class="space-y-1.5 list-decimal list-inside leading-relaxed">
        <li>Abra o rachao.app no <strong class="text-primary-100">Safari</strong> — não funciona em outros navegadores no iOS</li>
        <li>Toque em <span class="inline-flex items-center gap-1 text-primary-100 font-medium">Compartilhar <Share2 size={11} class="inline shrink-0" /></span> na barra inferior</li>
        <li>Role e toque em <strong class="text-primary-100">"Adicionar à Tela de Início"</strong></li>
        <li>Confirme tocando em <strong class="text-primary-100">"Adicionar"</strong></li>
      </ol>
    </div>
  {/if}
{/if}
