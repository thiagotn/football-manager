<script lang="ts">
  /**
   * Emblema do jogo: a cobrinha enrolada num anel. Aparece no cabeçalho da
   * partida, na tela inicial e nos ícones do PWA.
   */
  let {
    size = 32,
    ring = '#ffc93c',
    body = '#8ede4f',
    head = '#b6f36a',
  }: { size?: number; ring?: string; body?: string; head?: string } = $props();
</script>

<svg width={size} height={size} viewBox="0 0 100 100" aria-hidden="true">
  <circle cx="50" cy="50" r="46" fill="none" stroke={ring} stroke-width="4" opacity=".55" />
  <path
    d="M27 78C27 58 55 62 55 45c0-10-10-15-18-10"
    fill="none"
    stroke={body}
    stroke-width="13"
    stroke-linecap="round"
  />
  <circle cx="37" cy="35" r="9.5" fill={head} />
  <circle cx="34.5" cy="32.5" r="2.6" fill="#10361a" />
  <circle cx="41" cy="31.5" r="2.6" fill="#10361a" />
  <circle cx="72" cy="30" r="6" fill={ring} />
</svg>
