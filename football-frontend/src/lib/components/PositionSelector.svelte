<script lang="ts">
  import { POS_ABBR, POS_COLOR_CLASSES } from '$lib/team-builder';
  import type { Position } from '$lib/team-builder';
  import { t } from '$lib/i18n';

  const POSITIONS: Position[] = ['goalkeeper', 'defender', 'fullback', 'midfielder', 'forward'];

  // Map between API value (gk/zag/lat/mei/ata) and team-builder Position type
  const API_TO_POS: Record<string, Position> = {
    gk:  'goalkeeper',
    zag: 'defender',
    lat: 'fullback',
    mei: 'midfielder',
    ata: 'forward',
  };
  const POS_TO_API: Record<Position, string> = {
    goalkeeper: 'gk',
    defender:   'zag',
    fullback:   'lat',
    midfielder: 'mei',
    forward:    'ata',
  };
  const I18N_KEY: Record<Position, string> = {
    goalkeeper: 'position.gk',
    defender:   'position.zag',
    fullback:   'position.lat',
    midfielder: 'position.mei',
    forward:    'position.ata',
  };

  let {
    value = $bindable('mei'),
    readonly = false,
  }: {
    value?: string;
    readonly?: boolean;
  } = $props();

  let currentPos = $derived(API_TO_POS[value] ?? 'midfielder');

  function select(pos: Position) {
    if (readonly) return;
    value = POS_TO_API[pos];
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
