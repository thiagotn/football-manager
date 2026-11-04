<script lang="ts">
  /**
   * Emblema do jogo: a grade de blocos com um 9 escrito à mão, dentro do anel
   * mostarda que todos os emblemas do hub têm. Aparece no cabeçalho da partida,
   * na tela inicial e nos ícones do PWA.
   */
  let {
    size = 32,
    ring = '#ffc93c',
    paper = '#f6ecd6',
    line = '#8d7b5c',
    ink = '#2a1f17',
    pen = '#c2410c',
  }: { size?: number; ring?: string; paper?: string; line?: string; ink?: string; pen?: string } =
    $props();
</script>

<svg width={size} height={size} viewBox="0 0 100 100" aria-hidden="true">
  <circle cx="50" cy="50" r="46" fill="none" stroke={ring} stroke-width="4" opacity=".55" />

  <!-- A folha de jornal, levemente torta, como recortada do caderno -->
  <g transform="rotate(-4 50 50)">
    <rect x="22" y="22" width="56" height="56" rx="5" fill={paper} />
    <g stroke={line} stroke-width="2.6" stroke-linecap="round">
      <path d="M40.7 24v52M59.3 24v52M24 40.7h52M24 59.3h52" />
    </g>
    <rect
      x="22"
      y="22"
      width="56"
      height="56"
      rx="5"
      fill="none"
      stroke={line}
      stroke-width="3.4"
    />

    <!-- Um dado impresso e um número escrito pelo jogador -->
    <text
      x="31.3"
      y="38"
      font-family="Lilita One, Arial Rounded MT Bold, sans-serif"
      font-size="15"
      text-anchor="middle"
      fill={ink}>5</text>
    <text
      x="68.7"
      y="71"
      font-family="Lilita One, Arial Rounded MT Bold, sans-serif"
      font-size="15"
      text-anchor="middle"
      fill={pen}>9</text>
  </g>
</svg>
