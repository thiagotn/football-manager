<script lang="ts">
  /** Arte do card no hub: a cobrinha atrás da fruta, sobre a lagoa. */
</script>

<div class="art absolute inset-0">
  <svg viewBox="0 0 320 168" class="absolute inset-0 h-full w-full" aria-hidden="true">
    <defs>
      <pattern id="cb-hex" width="34" height="59" patternUnits="userSpaceOnUse">
        <path
          d="M17 0 34 10v20L17 40 0 30V10Z M34 29.5 51 39.5v20L34 69.5 17 59.5v-20Z"
          fill="none"
          stroke="rgba(94,204,214,.14)"
          stroke-width="1"
        />
      </pattern>
      <linearGradient id="cb-body" x1="0" y1="0" x2="1" y2="1">
        <stop offset="0" stop-color="#b6f36a" />
        <stop offset="1" stop-color="#2f8f45" />
      </linearGradient>
    </defs>

    <rect width="320" height="168" fill="url(#cb-hex)" />

    <!-- Corpo: uma onda que entra pela esquerda e termina na cabeça -->
    <path
      d="M-12 138C36 138 40 94 92 94c52 0 56 44 104 28 22-7 34-24 42-40"
      fill="none"
      stroke="url(#cb-body)"
      stroke-width="26"
      stroke-linecap="round"
    />
    <circle cx="243" cy="70" r="17" fill="#b6f36a" />
    <circle cx="239" cy="64" r="4.2" fill="#10361a" />
    <circle cx="251" cy="66" r="4.2" fill="#10361a" />

    <!-- Fruta -->
    <circle cx="289" cy="40" r="11" fill="#ffc93c" />
    <circle cx="285" cy="36" r="3.5" fill="#ffe9a8" />
  </svg>
</div>

<style>
  .art {
    background:
      radial-gradient(120% 90% at 50% 0%, rgba(94, 204, 214, 0.18), transparent 60%),
      linear-gradient(180deg, theme('colors.lagoon.2') 0%, theme('colors.lagoon.DEFAULT') 100%);
  }
</style>
