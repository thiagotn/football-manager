<script lang="ts">
  import { POS_ABBR, POS_COLOR_CLASSES } from '$lib/team-builder';
  import type { Position } from '$lib/team-builder';
  import { API_TO_POS, POS_TO_API, POS_I18N_KEY as I18N_KEY } from '$lib/positions';
  import { t } from '$lib/i18n';

  const POSITIONS: Position[] = ['goalkeeper', 'defender', 'fullback', 'midfielder', 'forward'];

  let {
    value = $bindable('mei'),
    readonly = false,
    onchange,
  }: {
    value?: string;
    readonly?: boolean;
    onchange?: (apiValue: string) => void;
  } = $props();

  let currentPos = $derived(API_TO_POS[value] ?? 'midfielder');

  function select(pos: Position) {
    if (readonly) return;
    const apiValue = POS_TO_API[pos];
    value = apiValue;
    onchange?.(apiValue);
  }
</script>

<div class="flex gap-1.5 flex-wrap">
  {#each POSITIONS as pos}
    {@const active = currentPos === pos}
    <button
      type="button"
      onclick={() => select(pos)}
      disabled={readonly}
      title={$t(I18N_KEY[pos])}
      class="px-2.5 py-1 rounded text-xs font-bold transition-all border
        {active
          ? POS_COLOR_CLASSES[pos] + ' border-current ring-1 ring-current'
          : 'bg-white/5 text-gray-400 border-white/10 hover:bg-white/10'}
        {readonly ? 'cursor-default' : 'cursor-pointer'}"
    >
      {POS_ABBR[pos]}
    </button>
  {/each}
</div>
