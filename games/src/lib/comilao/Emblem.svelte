<script lang="ts">
  /**
   * Emblema do jogo: o Comilão mordendo, com um fantasma na cola, dentro do
   * anel mostarda que todos os emblemas do hub têm. Aparece no cabeçalho da
   * partida, na tela inicial e nos ícones do PWA.
   */
  let {
    size = 32,
    ring = '#ffc93c',
    hero = '#ffc93c',
    ghost = '#ff3b5c',
  }: { size?: number; ring?: string; hero?: string; ghost?: string } = $props();
</script>

<svg width={size} height={size} viewBox="0 0 100 100" aria-hidden="true">
  <circle cx="50" cy="50" r="46" fill="none" stroke={ring} stroke-width="4" opacity=".55" />

  <!-- Pastilhas no caminho -->
  <circle cx="74" cy="38" r="3.4" fill={ring} opacity=".9" />
  <circle cx="86" cy="38" r="5" fill={ring} opacity=".9" />

  <!-- O Comilão, boca aberta para a direita -->
  <path d="M38 38 L63 25 A26 26 0 1 0 63 51 Z" fill={hero} />

  <!-- O fantasma atrás -->
  <path
    d="M12 62a14 14 0 0 1 28 0v16l-4.7-4.4-4.7 4.4-4.7-4.4-4.7 4.4-4.5-4.2V62Z"
    fill={ghost}
  />
  <circle cx="21" cy="58" r="3.6" fill="#fff" />
  <circle cx="32" cy="58" r="3.6" fill="#fff" />
  <circle cx="22.4" cy="58.6" r="1.7" fill="#1b1040" />
  <circle cx="33.4" cy="58.6" r="1.7" fill="#1b1040" />
</svg>
