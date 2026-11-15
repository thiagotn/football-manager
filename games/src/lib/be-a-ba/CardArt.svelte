<script lang="ts">
  /** Arte do card no hub: céu de dia, a corujinha e U-V-A montado em peças. */
  import { MASCOT, imgUrl } from './art';

  const TILES = [
    { l: 'U', r: -8, y: 6 },
    { l: 'V', r: 4, y: 0 },
    { l: 'A', r: -3, y: 4 },
  ];
</script>

<div class="art absolute inset-0">
  <img src={imgUrl('2600')} alt="" class="absolute -right-4 -top-5 h-24 w-24 opacity-90" />
  <img src={imgUrl(MASCOT)} alt="" class="absolute bottom-3 left-5 h-[92px] w-[92px]" />
  <img src={imgUrl('1f347')} alt="" class="absolute left-[118px] top-7 h-[60px] w-[60px]" />
  <div class="absolute bottom-6 right-6 flex gap-2">
    {#each TILES as tile (tile.l)}
      <span class="tile" style="transform: translateY({tile.y}px) rotate({tile.r}deg)">{tile.l}</span>
    {/each}
  </div>
  <span class="absolute right-7 top-9 font-display text-[30px] text-white/90 [text-shadow:0_3px_0_rgba(36,52,90,.25)]">1+2</span>
</div>

<style>
  .art {
    background:
      radial-gradient(70% 60% at 100% 0%, rgba(255, 207, 63, 0.6), transparent 60%),
      linear-gradient(180deg, #8fd3ff 0%, #c7ebff 64%, #7ccf5a 64%, #43b85c 100%);
  }
  .tile {
    display: grid;
    place-items: center;
    width: 46px;
    height: 50px;
    border-radius: 12px;
    background: #ffcf3f;
    color: #24345a;
    font-family: theme('fontFamily.display');
    font-size: 30px;
    box-shadow: 0 4px 0 #d9a514;
  }
</style>
