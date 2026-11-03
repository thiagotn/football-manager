<script lang="ts">
  /** Arte do card no hub: o poço do Tetris com peças neon. */
  const PIECES: { color: string; cells: [number, number][] }[] = [
    { color: '#00e5f5', cells: [[0, 9], [1, 9], [2, 9], [3, 9]] },
    { color: '#ffd000', cells: [[8, 8], [9, 8], [8, 9], [9, 9]] },
    { color: '#ee00bb', cells: [[4, 9], [5, 9], [6, 9], [5, 8]] },
    { color: '#00ee77', cells: [[0, 8], [1, 8], [1, 7], [2, 7]] },
    { color: '#ff2244', cells: [[6, 8], [7, 8], [7, 9], [6, 7]] },
    { color: '#3377ff', cells: [[4, 3], [4, 4], [5, 4], [6, 4]] },
  ];

  const blocks = PIECES.flatMap((piece, pi) =>
    piece.cells.map(([x, y], ci) => ({ key: `${pi}-${ci}`, color: piece.color, x: x * 16, y: y * 16 - 4 }))
  );
</script>

<div class="art absolute inset-0">
  <div class="stars absolute inset-0 opacity-35"></div>
  <div class="well absolute -bottom-px left-1/2 top-4 -ml-20 w-40 border border-b-0 border-night-grid/60">
    {#each blocks as b (b.key)}
      <div
        class="absolute m-px h-3.5 w-3.5 rounded-sm"
        style="left:{b.x}px;top:{b.y}px;background:{b.color};box-shadow:0 0 8px {b.color}"
      ></div>
    {/each}
  </div>
</div>

<style>
  .art {
    background: radial-gradient(90% 90% at 50% 30%, theme('colors.night.2') 0%, theme('colors.night.DEFAULT') 70%);
  }
  .stars {
    background: radial-gradient(circle, rgba(255, 255, 255, 0.5) 0.7px, transparent 1.2px) 0 0 / 22px 22px;
  }
  .well {
    background:
      repeating-linear-gradient(90deg, rgba(12, 24, 56, 0.6) 0 1px, transparent 1px 16px),
      repeating-linear-gradient(180deg, rgba(12, 24, 56, 0.6) 0 1px, transparent 1px 16px);
  }
</style>
