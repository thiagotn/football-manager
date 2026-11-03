<script lang="ts">
  /** Arte do card no hub: três copos americanos no balcão, dois prontos. */
  import Vessel from './Vessel.svelte';
  import { VESSELS } from './content';

  const ART: string[][] = [
    ['laranja', 'laranja', 'laranja', 'laranja'],
    ['uva', 'limao', 'melancia', 'cafe'],
    ['morango', 'morango', 'morango', 'morango'],
  ];
</script>

<div class="art absolute inset-0">
  <div class="absolute inset-x-0 bottom-0 flex items-end justify-center gap-3 pb-3">
    {#each ART as layers, i (i)}
      <div class="w-[52px]" style="transform: translateY({i === 1 ? -6 : 0}px)">
        <Vessel vessel={VESSELS.americano} {layers} />
      </div>
    {/each}
  </div>
</div>

<style>
  .art {
    background:
      radial-gradient(70% 60% at 50% 100%, rgba(255, 214, 150, 0.16), transparent 70%),
      linear-gradient(180deg, transparent 62%, #3b2519 62%, #1f130c 100%),
      url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='44' height='44' viewBox='0 0 44 44'%3E%3Crect width='44' height='44' fill='%23183a69'/%3E%3Cpath d='M22 7C25 16 28 19 37 22C28 25 25 28 22 37C19 28 16 25 7 22C16 19 19 16 22 7Z' fill='%232f68ad'/%3E%3Ccircle cx='22' cy='22' r='3.5' fill='%23d4e3f6' fill-opacity='.6'/%3E%3C/svg%3E")
        0 0 / 44px 44px;
  }
</style>
