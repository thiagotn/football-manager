<script lang="ts">
  /**
   * Emblema do jogo: um espadilha num anel. Substitui o escudo do rachão no
   * verso das cartas, no cabeçalho e na tela inicial.
   */
  let { size = 32, color = '#ffc93c', opacity = 1 }: { size?: number; color?: string; opacity?: number } = $props();
</script>

<svg width={size} height={size} viewBox="0 0 100 100" aria-hidden="true" style="opacity:{opacity}">
  <circle cx="50" cy="50" r="46" fill="none" stroke={color} stroke-width="4" opacity=".55" />
  <path
    d="M50 16C50 16 20 42 20 59a15 15 0 0 0 25 11c-1 9-5 16-11 20h32c-6-4-10-11-11-20a15 15 0 0 0 25-11c0-17-30-43-30-43Z"
    fill={color}
  />
</svg>
