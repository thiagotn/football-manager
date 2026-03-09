<script lang="ts">
  interface Props {
    rating: number;
    readonly?: boolean;
    size?: number;
    onchange?: (value: number) => void;
  }

  let { rating = $bindable(0), readonly = false, size = 32, onchange }: Props = $props();

  let hovered = $state(0);

  function select(value: number) {
    if (readonly) return;
    rating = value;
    onchange?.(value);
  }
</script>

<div class="flex items-center gap-1" role={readonly ? undefined : 'group'} aria-label={readonly ? `${rating} de 5 estrelas` : 'Selecione uma nota de 1 a 5 estrelas'}>
  {#each [1, 2, 3, 4, 5] as star}
    {#if readonly}
      <span
        style="font-size: {size}px; line-height: 1;"
        class="{(rating >= star) ? 'text-amber-400' : 'text-gray-200 dark:text-gray-600'}"
        aria-hidden="true"
      >★</span>
    {:else}
      <button
        type="button"
        onclick={() => select(star)}
        onmouseenter={() => hovered = star}
        onmouseleave={() => hovered = 0}
        style="font-size: {size}px; line-height: 1;"
        class="transition-colors {((hovered || rating) >= star) ? 'text-amber-400' : 'text-gray-300 dark:text-gray-600'} hover:scale-110 transition-transform"
        aria-label="{star} estrela{star > 1 ? 's' : ''}"
      >★</button>
    {/if}
  {/each}
</div>
