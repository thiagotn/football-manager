<script lang="ts">
  /** Arte do card no hub: o Comilão num corredor de neon, com a turma na cola. */
</script>

<div class="art absolute inset-0">
  <svg viewBox="0 0 320 168" class="absolute inset-0 h-full w-full" aria-hidden="true">
    <defs>
      <!--
        `userSpaceOnUse` de propósito: em `objectBoundingBox` uma linha
        horizontal tem caixa de altura zero e o gradiente não pinta nada.
      -->
      <linearGradient id="cm-wall" gradientUnits="userSpaceOnUse" x1="0" y1="24" x2="0" y2="144">
        <stop offset="0" stop-color="#ff3df0" />
        <stop offset="1" stop-color="#2ae0ff" />
      </linearGradient>
      <filter id="cm-glow" x="-30%" y="-30%" width="160%" height="160%">
        <feGaussianBlur stdDeviation="2.6" result="b" />
        <feMerge><feMergeNode in="b" /><feMergeNode in="SourceGraphic" /></feMerge>
      </filter>
    </defs>

    <!-- Grade do horizonte -->
    <g stroke="rgba(150,110,235,.2)" stroke-width="1">
      {#each [0, 1, 2, 3, 4, 5, 6, 7, 8] as i (i)}
        <line x1={i * 40} y1="168" x2={120 + i * 10} y2="54" />
      {/each}
      {#each [58, 74, 96, 126, 164] as y (y)}
        <line x1="0" y1={y} x2="320" y2={y} />
      {/each}
    </g>

    <!-- Corredor: parede de cima, parede de baixo e dois cantos -->
    <g
      stroke="url(#cm-wall)"
      stroke-width="4"
      stroke-linecap="round"
      stroke-linejoin="round"
      filter="url(#cm-glow)"
      fill="none"
    >
      <path d="M18 56v-16h284v16" />
      <path d="M18 112v16h284v-16" />
    </g>

    <!-- Pastilhas na pista -->
    <g fill="#ffe9a8">
      {#each [62, 82, 102] as x (x)}
        <circle cx={x} cy="84" r="3" />
      {/each}
      <circle cx="34" cy="84" r="6.5" fill="#fff6d6" />
    </g>

    <!-- O Comilão, comendo para a esquerda -->
    <path d="M196 84 L168 69 A30 30 0 1 1 168 99 Z" fill="#ffc93c" filter="url(#cm-glow)" />

    <!-- Dois fantasmas na cola -->
    {#each [{ x: 234, c: '#ff3b5c' }, { x: 276, c: '#6bffb8' }] as g (g.x)}
      <g transform="translate({g.x} 66)">
        <path d="M0 16a16 16 0 0 1 32 0v19l-5.3-5-5.3 5-5.4-5-5.3 5L0 35Z" fill={g.c} />
        <circle cx="10" cy="13" r="4.2" fill="#fff" />
        <circle cx="22" cy="13" r="4.2" fill="#fff" />
        <circle cx="8.6" cy="13.6" r="2" fill="#1b1040" />
        <circle cx="20.6" cy="13.6" r="2" fill="#1b1040" />
      </g>
    {/each}
  </svg>
</div>

<style>
  .art {
    background:
      radial-gradient(110% 80% at 50% 110%, rgba(255, 61, 240, 0.22), transparent 62%),
      linear-gradient(180deg, theme('colors.synth.2') 0%, theme('colors.synth.DEFAULT') 100%);
  }
</style>
