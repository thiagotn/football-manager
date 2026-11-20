<script lang="ts">
  /** Arte do card no hub: a cena do vale com uma peça fora do lugar. */
  import SceneArt from './SceneArt.svelte';
  import { sceneById } from './scenes';
  import { SIZES, makeEdges, mulberry32, pieceBox, piecePath } from './jigsaw';

  const scene = sceneById('vale')!;
  const size = SIZES[1];
  const edges = makeEdges(size, mulberry32(7));
  const hole = piecePath(0, 2, size, edges);
  const box = pieceBox(0, 2, size);
</script>

<div class="art absolute inset-0">
  <svg viewBox="0 0 400 300" preserveAspectRatio="xMidYMid slice" class="absolute inset-0 h-full w-full" aria-hidden="true">
    <defs><clipPath id="qc-card-clip"><path d={hole} /></clipPath></defs>
    <g id="qc-card-scene"><SceneArt {scene} uid="-card" /></g>
    <path d={hole} fill="#fff3dc" stroke="rgba(59,42,90,.35)" stroke-width="2" stroke-dasharray="6 5" />
    <g transform="translate(-34 70) rotate(-12 {box.x + box.w / 2} {box.y + box.h / 2})" style="filter: drop-shadow(0 6px 5px rgba(59,42,90,.45))">
      <g clip-path="url(#qc-card-clip)"><use href="#qc-card-scene" /></g>
      <path d={hole} fill="none" stroke="#fff" stroke-width="3" />
    </g>
  </svg>
</div>

<style>
  .art {
    background: #fff3dc;
  }
</style>
