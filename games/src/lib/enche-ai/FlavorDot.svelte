<script lang="ts">
  import { FLAVORS } from './content';

  let { id, size = 14 }: { id: string; size?: number } = $props();
  const f = $derived(FLAVORS[id]);
</script>

<svg viewBox="0 0 16 16" width={size} height={size} aria-hidden="true" class="flex-none">
  <g clip-path="url(#ea-clip-dot)">
    <rect width="16" height="16" fill={f.color} />
    <rect class="ea-pat" width="16" height="16" fill="url(#ea-p-{f.pattern})" />
  </g>
  <circle cx="8" cy="8" r="7" fill="none" stroke="rgba(0,0,0,.25)" />
</svg>

<style>
  :global(.ea-no-pat) .ea-pat {
    display: none;
  }
</style>
