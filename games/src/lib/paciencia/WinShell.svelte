<script lang="ts">
  /**
   * Moldura 300×360 da tela de vitória: borda metálica de 3px e campo escuro
   * com brilho superior e textura diagonal. O conteúdo vem por `children` e se
   * posiciona em absoluto dentro do campo.
   *
   * Geometria herdada do card-escudo do rachao.app; as cores são as do games.
   */
  import type { Snippet } from 'svelte';

  let { children }: { children: Snippet } = $props();
</script>

<div class="shell-outer relative shrink-0 drop-shadow-[0_20px_40px_rgba(0,0,0,.5)]">
  <div class="shell-inner">
    {@render children()}
  </div>
</div>

<style>
  /* Moldura mostarda (3px) */
  .shell-outer {
    width: 300px;
    height: 360px;
    clip-path: path('M14 0 H286 A14 14 0 0 1 300 14 V202 C300 272 228 328 150 360 C72 328 0 272 0 202 V14 A14 14 0 0 1 14 0 Z');
    background: linear-gradient(160deg, #ffe29a 0%, #ffc93c 30%, #a9801d 58%, #ffd666 82%, #e0a21f 100%);
  }
  /* Campo de feltro */
  .shell-inner {
    position: absolute;
    top: 3px;
    left: 3px;
    width: 294px;
    height: 354px;
    overflow: hidden;
    clip-path: path('M13 0 H281 A13 13 0 0 1 294 13 V198 C294 266 224 322 147 354 C70 322 0 266 0 198 V13 A13 13 0 0 1 13 0 Z');
    background: linear-gradient(158deg, #1e6b4a 0%, #17533a 34%, #0d3123 70%, #081d15 100%);
  }
  .shell-inner::before {
    content: '';
    position: absolute;
    inset: 0;
    background: radial-gradient(115% 60% at 50% 0%, rgba(255, 255, 255, 0.2), transparent 62%);
    pointer-events: none;
  }
  .shell-inner::after {
    content: '';
    position: absolute;
    inset: 0;
    background: repeating-linear-gradient(118deg, rgba(255, 255, 255, 0.055) 0 2px, transparent 2px 10px);
    mix-blend-mode: overlay;
    pointer-events: none;
  }
</style>
