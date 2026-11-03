<script lang="ts">
  /** Arte do card no hub: leque de quatro cartas sobre o feltro. */
  import Emblem from './Emblem.svelte';

  const FAN = [
    { back: true, left: -90, bottom: 0, rotate: -16, rank: '', suit: '', red: false, gold: false },
    { back: false, left: -46, bottom: 6, rotate: -5, rank: 'A', suit: '♠︎', red: false, gold: false },
    { back: false, left: -12, bottom: 6, rotate: 6, rank: 'K', suit: '♥︎', red: true, gold: false },
    { back: false, left: 26, bottom: 0, rotate: 17, rank: 'Q', suit: '♦︎', red: true, gold: true },
  ];
</script>

<div class="art absolute inset-0">
  <div class="absolute bottom-[22px] left-1/2 h-0 w-0">
    {#each FAN as card, i (i)}
      {#if card.back}
        <div
          class="back absolute h-[82px] w-[58px] overflow-hidden rounded-[7px] border-[1.5px] border-mustard/70"
          style="left:{card.left}px;bottom:{card.bottom}px;transform:rotate({card.rotate}deg);transform-origin:50% 100%"
        >
          <div class="absolute left-1/2 top-1/2 -ml-5 -mt-5">
            <Emblem size={40} opacity={0.9} />
          </div>
        </div>
      {:else}
        <div
          class="absolute h-[82px] w-[58px] rounded-[7px] bg-[linear-gradient(180deg,#fff,#f3f4f6)] {card.gold
            ? 'border-2 border-mustard shadow-[0_0_0_3px_rgba(255,201,60,.35),0_6px_14px_rgba(0,0,0,.35)]'
            : 'border border-black/15 shadow-[0_6px_14px_rgba(0,0,0,.35)]'}"
          style="left:{card.left}px;bottom:{card.bottom}px;transform:rotate({card.rotate}deg);transform-origin:50% 100%;color:{card.red
            ? '#dc2626'
            : '#111827'}"
        >
          <div class="absolute left-1.5 top-[3px] flex flex-col items-center leading-[.85]">
            <span class="font-display text-[22px]">{card.rank}</span>
            <span class="text-[13px]">{card.suit}</span>
          </div>
          <div class="absolute bottom-[3px] right-[5px] text-[28px] leading-none">{card.suit}</div>
        </div>
      {/if}
    {/each}
  </div>
</div>

<style>
  .art {
    background:
      radial-gradient(120% 90% at 50% 25%, rgba(255, 255, 255, 0.08), transparent 60%),
      radial-gradient(130% 100% at 50% 100%, rgba(0, 0, 0, 0.45), transparent 70%),
      theme('colors.felt.DEFAULT');
  }
  .back {
    background:
      repeating-linear-gradient(118deg, rgba(255, 255, 255, 0.07) 0 2px, transparent 2px 8px),
      linear-gradient(158deg, theme('colors.felt.light'), theme('colors.felt.dark'));
  }
</style>
