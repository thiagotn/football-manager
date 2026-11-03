<script lang="ts">
  /**
   * Card de um jogo no hub. Os dados vêm do registro (`$lib/games`); a arte de
   * 168px é um componente do próprio jogo, para o card não conhecer nenhum deles.
   */
  import type { Component } from 'svelte';
  import { ArrowRight } from 'lucide-svelte';
  import type { Game, TagTone } from '$lib/games';
  import { t } from '$lib/i18n';

  let { game, Art, accent = '#ffc93c' }: { game: Game; Art: Component; accent?: string } = $props();

  const TONE: Record<TagTone, string> = {
    hot: 'bg-ketchup/20 text-[#ffb3ab]',
    gold: 'bg-mustard/15 text-mustard',
    cool: 'bg-[#2e7cf0]/20 text-[#9ec5ff]',
    neutral: 'bg-white/10 text-white/70',
  };
</script>

<a
  href="/{game.slug}/"
  style="--accent:{accent}"
  class="game-card group flex flex-col overflow-hidden rounded-xl border border-white/10 bg-ink-2 transition-[transform,border-color] duration-150 hover:-translate-y-0.5"
>
  <div class="relative h-[168px] overflow-hidden">
    <Art />
    {#if game.isNew}
      <span
        class="absolute left-3 top-3 inline-flex items-center rounded-full bg-mustard px-2.5 py-0.5 text-xs font-bold text-mustard-ink"
      >
        {$t('hub.badge_new')}
      </span>
    {/if}
  </div>

  <div class="flex flex-col gap-2.5 p-4">
    <div class="flex flex-col gap-0.5">
      <span class="font-display text-[30px] leading-none text-white">{$t(game.nameKey)}</span>
      <span class="text-sm leading-[1.45] text-white/60">{$t(game.descKey)}</span>
    </div>
    <div class="flex flex-wrap gap-1.5">
      {#each game.tags as tag (tag.key)}
        <span class="tag {TONE[tag.tone]}">{$t(tag.key)}</span>
      {/each}
    </div>
    <span
      class="inline-flex h-11 items-center justify-center gap-2 rounded-lg bg-mustard text-[15px] font-extrabold text-mustard-ink transition-colors group-hover:bg-[#ffd666]"
    >
      {$t('hub.play')}
      <ArrowRight size={16} strokeWidth={2.4} />
    </span>
  </div>
</a>

<style>
  .game-card:hover {
    border-color: var(--accent);
  }
</style>
