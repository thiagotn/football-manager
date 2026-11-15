<script lang="ts">
  /**
   * `n` objetos desenhados, em fileiras de até 5 (como o material dourado e a
   * "caixa de dez" da escola). Os últimos `leaving` vão embora animados — é a
   * subtração ("2 voaram").
   */
  import { imgUrl } from './art';

  let { item, n, leaving = 0, size = 38 }: { item: string; n: number; leaving?: number; size?: number } = $props();

  const rows = $derived(
    Array.from({ length: Math.ceil(n / 5) }, (_, r) => Array.from({ length: Math.min(5, n - r * 5) }, (_, c) => r * 5 + c))
  );
</script>

<div class="objects" style="--s:{size}px" data-count={n}>
  {#each rows as row, r (r)}
    <div class="row">
      {#each row as i (i)}
        <img src={imgUrl(item)} alt="" class:leaving={i >= n - leaving} style="--d:{(i - (n - leaving)) * 120}ms" draggable="false" />
      {/each}
    </div>
  {/each}
</div>

<style>
  .objects {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 6px;
  }
  .row {
    display: flex;
    gap: 6px;
  }
  img {
    width: var(--s);
    height: var(--s);
    user-select: none;
  }
  .leaving {
    animation: leave 0.9s ease-in 0.7s forwards;
    animation-delay: calc(0.7s + var(--d));
  }
  @keyframes leave {
    0% {
      transform: none;
      opacity: 1;
    }
    30% {
      transform: translateY(4px) scale(1.1);
    }
    100% {
      transform: translate(30px, -60px) rotate(25deg) scale(0.6);
      opacity: 0.18;
    }
  }
</style>
