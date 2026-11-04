<script lang="ts">
  /** Arte do card no hub: o canto do tabuleiro de jornal sobre o sepia noturno. */
</script>

<div class="art absolute inset-0">
  <svg viewBox="0 0 320 168" class="absolute inset-0 h-full w-full" aria-hidden="true">
    <defs>
      <!-- Trama do papel, bem de leve, para não virar textura de fundo -->
      <pattern id="sd-grain" width="6" height="6" patternUnits="userSpaceOnUse">
        <path d="M0 6 6 0" stroke="rgba(255,236,196,.05)" stroke-width="1" />
      </pattern>
      <linearGradient id="sd-paper" x1="0" y1="0" x2="0" y2="1">
        <stop offset="0" stop-color="#fbf3df" />
        <stop offset="1" stop-color="#ecdcbd" />
      </linearGradient>
    </defs>

    <rect width="320" height="168" fill="url(#sd-grain)" />

    <!-- A folha entra pela direita, torta, com sombra de papel -->
    <g transform="rotate(-7 232 96)">
      <rect x="150" y="26" width="168" height="168" rx="8" fill="rgba(0,0,0,.35)" />
      <rect x="146" y="22" width="168" height="168" rx="8" fill="url(#sd-paper)" />
      <g stroke="#cdb894" stroke-width="1.4">
        <path d="M174.7 22v168M203.3 22v168M232 22v168M260.7 22v168M289.3 22v168" />
        <path d="M146 50.7h168M146 79.3h168M146 108h168M146 136.7h168M146 165.3h168" />
      </g>
      <g stroke="#8d7b5c" stroke-width="3">
        <path d="M203.3 22v168M260.7 22v168M146 79.3h168M146 136.7h168" />
      </g>

      <g
        font-family="Lilita One, Arial Rounded MT Bold, sans-serif"
        font-size="20"
        text-anchor="middle">
        <text x="160.3" y="44" fill="#2a1f17">5</text>
        <text x="217.7" y="44" fill="#2a1f17">3</text>
        <text x="246.3" y="72" fill="#2a1f17">7</text>
        <text x="189" y="101" fill="#c2410c">8</text>
        <text x="275" y="101" fill="#2a1f17">1</text>
        <text x="160.3" y="129" fill="#2a1f17">4</text>
        <text x="232" y="158" fill="#c2410c">6</text>
        <text x="303.7" y="129" fill="#2a1f17">9</text>
      </g>
    </g>

    <!-- Lápis à esquerda, deitado na mesa -->
    <g transform="rotate(-24 70 118)">
      <rect x="16" y="110" width="96" height="13" rx="2.5" fill="#ffc93c" />
      <rect x="16" y="110" width="96" height="4.5" rx="2" fill="#ffdd7d" />
      <path d="M112 110 128 116.5 112 123Z" fill="#e8dcc0" />
      <path d="M124 114.6 128 116.5 124 118.4Z" fill="#2a1f17" />
      <rect x="6" y="110" width="12" height="13" rx="2.5" fill="#e8473b" />
    </g>
  </svg>
</div>

<style>
  .art {
    background:
      radial-gradient(120% 90% at 18% 10%, rgba(255, 201, 60, 0.16), transparent 58%),
      linear-gradient(180deg, theme('colors.news.light') 0%, theme('colors.news.DEFAULT') 100%);
  }
</style>
