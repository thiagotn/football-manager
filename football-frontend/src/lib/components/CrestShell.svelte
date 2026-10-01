<script lang="ts">
  /**
   * CrestShell — moldura do escudo 300×360 (handoff design_handoff_player_crest_card).
   *
   * Só a geometria: moldura metálica de 3px, campo verde com brilho superior e
   * textura diagonal. O conteúdo vem por `children` e se posiciona em absoluto
   * dentro do campo verde. Usado por `PlayerCrestCard` e pela tela de vitória
   * da Paciência.
   */
  import type { Snippet } from 'svelte';

  let { children }: { children: Snippet } = $props();
</script>

<div class="crest-outer relative shrink-0 drop-shadow-[0_20px_40px_rgba(31,154,77,.35)]">
  <div class="crest-inner">
    {@render children()}
  </div>
</div>

<style>
  /* Moldura metálica (3px) */
  .crest-outer {
    width: 300px;
    height: 360px;
    clip-path: path('M14 0 H286 A14 14 0 0 1 300 14 V202 C300 272 228 328 150 360 C72 328 0 272 0 202 V14 A14 14 0 0 1 14 0 Z');
    background: linear-gradient(160deg, #f7e2ad 0%, #e0b95c 28%, #8a6c2a 55%, #f5d585 82%, #c9a34d 100%);
  }
  /* Campo verde */
  .crest-inner {
    position: absolute;
    top: 3px;
    left: 3px;
    width: 294px;
    height: 354px;
    overflow: hidden;
    clip-path: path('M13 0 H281 A13 13 0 0 1 294 13 V198 C294 266 224 322 147 354 C70 322 0 266 0 198 V13 A13 13 0 0 1 13 0 Z');
    background: linear-gradient(158deg, #15422a 0%, #1f9a4d 34%, #0d5028 64%, #07331a 100%);
  }
  /* brilho superior */
  .crest-inner::before {
    content: '';
    position: absolute;
    inset: 0;
    background: radial-gradient(115% 60% at 50% 0%, rgba(255, 255, 255, 0.2), transparent 62%);
    pointer-events: none;
  }
  /* textura diagonal */
  .crest-inner::after {
    content: '';
    position: absolute;
    inset: 0;
    background: repeating-linear-gradient(118deg, rgba(255, 255, 255, 0.055) 0 2px, transparent 2px 10px);
    mix-blend-mode: overlay;
    pointer-events: none;
  }
</style>
