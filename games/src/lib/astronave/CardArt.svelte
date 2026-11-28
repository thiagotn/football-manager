<script lang="ts">
  /** Arte do card no hub: a nave atirando entre meteoros, com um batedor ao fundo. */
  import { SHIP_COCKPIT, SHIP_FLAME, SHIP_HULL, SHIP_WINGS } from './skins';

  const STARS = [
    [22, 30, 1.4], [58, 110, 1], [96, 22, 1.1], [130, 70, 1.5], [171, 132, 1], [214, 34, 1.2],
    [236, 96, 1], [268, 20, 1.5], [296, 118, 1.1], [44, 146, 1.2], [148, 14, 1], [300, 62, 1],
  ];
</script>

<div class="art absolute inset-0">
  <svg viewBox="0 0 320 168" class="absolute inset-0 h-full w-full" aria-hidden="true">
    {#each STARS as [x, y, r], i (i)}
      <circle cx={x} cy={y} {r} fill="#fff" opacity={0.35 + (i % 3) * 0.2} />
    {/each}

    <!-- Batedor -->
    <g transform="translate(232 44)">
      <ellipse cx="0" cy="-5" rx="9" ry="8" fill="#b9fff0" />
      <ellipse cx="0" cy="2" rx="20" ry="7.5" fill="#2fc79a" />
      <circle cx="-10" cy="3" r="2" fill="#0d3b34" />
      <circle cx="0" cy="4.5" r="2" fill="#0d3b34" />
      <circle cx="10" cy="3" r="2" fill="#0d3b34" />
    </g>
    <rect x="230" y="62" width="4" height="13" rx="2" fill="#ff6b5c" />

    <!-- Meteoros -->
    <path d="M74 38l20 5 9 18-11 19-22 3-14-15 3-21z" fill="#8a7767" />
    <circle cx="80" cy="58" r="5" fill="#6b5b4e" />
    <circle cx="91" cy="70" r="3" fill="#6b5b4e" />
    <path d="M276 104l11 3 4 10-7 9-12-1-5-10z" fill="#8a7767" />

    <!-- Nave e tiros -->
    <rect x="158.6" y="52" width="3.4" height="14" rx="1.7" fill="#8fe9ff" />
    <rect x="158.6" y="22" width="3.4" height="14" rx="1.7" fill="#8fe9ff" opacity=".7" />
    <g transform="translate(160 112) scale(4.6)">
      <path d={SHIP_FLAME} fill="#ffc93c" />
      <path d={SHIP_WINGS} fill="#4f7dff" />
      <path d={SHIP_HULL} fill="#e8eefc" />
      <ellipse {...SHIP_COCKPIT} fill="#12224a" />
    </g>
  </svg>
</div>

<style>
  .art {
    background:
      radial-gradient(110% 90% at 50% 100%, rgba(79, 125, 255, 0.28), transparent 62%),
      linear-gradient(180deg, theme('colors.cosmos.DEFAULT') 0%, theme('colors.cosmos.2') 100%);
  }
</style>
