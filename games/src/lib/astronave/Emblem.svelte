<script lang="ts">
  /**
   * Emblema do jogo: a nave subindo dentro de um anel, com um meteoro ao lado.
   * Aparece na tela inicial; os ícones do PWA repetem o desenho.
   */
  import { SHIP_COCKPIT, SHIP_FLAME, SHIP_HULL, SHIP_WINGS } from './skins';

  let { size = 32, ring = '#ffc93c' }: { size?: number; ring?: string } = $props();
</script>

<svg width={size} height={size} viewBox="0 0 100 100" aria-hidden="true">
  <circle cx="50" cy="50" r="46" fill="none" stroke={ring} stroke-width="4" opacity=".55" />
  <circle cx="24" cy="30" r="1.6" fill="#fff" opacity=".8" />
  <circle cx="78" cy="64" r="1.3" fill="#fff" opacity=".6" />
  <circle cx="30" cy="72" r="1.1" fill="#fff" opacity=".5" />
  <path d="M68 20l9 3 3 9-6 8-10-1-4-9z" fill="#8a7767" />
  <circle cx="71" cy="29" r="2.4" fill="#6b5b4e" />
  <g transform="translate(46 56) scale(3.1)">
    <path d={SHIP_FLAME} fill="#ffc93c" />
    <path d={SHIP_WINGS} fill="#4f7dff" />
    <path d={SHIP_HULL} fill="#e8eefc" />
    <ellipse {...SHIP_COCKPIT} fill="#12224a" />
  </g>
  <rect x="44.6" y="12" width="2.8" height="9" rx="1.4" fill="#8fe9ff" />
</svg>
