<script lang="ts">
  /** Hub de games.thiagotn.com — vitrine dos jogos instaláveis. */
  import { ArrowRight, Gamepad2 } from 'lucide-svelte';
  import Patterns from '$lib/enche-ai/Patterns.svelte';
  import Vessel from '$lib/enche-ai/Vessel.svelte';
  import { VESSELS } from '$lib/enche-ai/content';
  import { t } from '$lib/i18n';

  /** Arte do card: três copos americanos, dois prontos e um misturado. */
  const ART: string[][] = [
    ['laranja', 'laranja', 'laranja', 'laranja'],
    ['uva', 'limao', 'melancia', 'cafe'],
    ['morango', 'morango', 'morango', 'morango'],
  ];
</script>

<svelte:head>
  <title>{$t('hub.title')} — games.thiagotn.com</title>
  <meta name="description" content={$t('hub.subtitle')} />
  <meta name="theme-color" content="#0f1622" />
  <link rel="manifest" href="/manifest.webmanifest" />
  <link rel="apple-touch-icon" href="/apple-touch-icon.png" />
</svelte:head>

<Patterns />

<main class="relative z-10 mx-auto max-w-7xl px-4 py-8" style="padding-top: max(2rem, env(safe-area-inset-top))">
  <div class="mb-6 flex items-center justify-between">
    <div>
      <h1 class="flex items-center gap-2 text-2xl font-bold text-white">
        <Gamepad2 size={24} class="text-mustard" />
        {$t('hub.title')}
      </h1>
      <p class="mt-0.5 text-sm text-white/60">{$t('hub.subtitle')}</p>
    </div>
  </div>

  <div class="games-grid grid gap-4">
    <a
      href="/enche-ai/"
      class="group flex flex-col overflow-hidden rounded-xl border border-white/10 bg-ink-2 transition-[transform,border-color] duration-150 hover:-translate-y-0.5 hover:border-mustard"
    >
      <div class="art relative h-[168px] overflow-hidden">
        <div class="absolute inset-x-0 bottom-0 flex items-end justify-center gap-3 pb-3">
          {#each ART as layers, i (i)}
            <div class="w-[52px]" style="transform: translateY({i === 1 ? -6 : 0}px)">
              <Vessel vessel={VESSELS.americano} {layers} />
            </div>
          {/each}
        </div>
        <span class="absolute left-3 top-3 inline-flex items-center rounded-full bg-mustard px-2.5 py-0.5 text-xs font-bold text-mustard-ink">
          {$t('hub.badge_new')}
        </span>
      </div>
      <div class="flex flex-col gap-2.5 p-4">
        <div class="flex flex-col gap-0.5">
          <span class="font-display text-[30px] leading-none text-white">{$t('enche_ai.name')}</span>
          <span class="text-sm leading-[1.45] text-white/60">{$t('enche_ai.desc')}</span>
        </div>
        <div class="flex flex-wrap gap-1.5">
          <span class="tag bg-ketchup/20 text-[#ffb3ab]">{$t('hub.tag_puzzle')}</span>
          <span class="tag bg-mustard/15 text-mustard">{$t('hub.tag_daily')}</span>
          <span class="tag bg-white/10 text-white/70">{$t('hub.tag_offline')}</span>
        </div>
        <span class="inline-flex h-11 items-center justify-center gap-2 rounded-lg bg-mustard text-[15px] font-extrabold text-mustard-ink transition-colors group-hover:bg-[#ffd666]">
          {$t('hub.play')}
          <ArrowRight size={16} strokeWidth={2.4} />
        </span>
      </div>
    </a>
  </div>

  <p class="mt-8 text-sm text-white/50">
    <a href="https://rachao.app/games" class="underline underline-offset-2 hover:text-white">{$t('hub.more')}</a>
  </p>
</main>

<style>
  .games-grid {
    grid-template-columns: repeat(auto-fill, minmax(min(100%, 320px), 1fr));
  }
  .art {
    background:
      radial-gradient(70% 60% at 50% 100%, rgba(255, 214, 150, 0.16), transparent 70%),
      linear-gradient(180deg, transparent 62%, #3b2519 62%, #1f130c 100%),
      url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='44' height='44' viewBox='0 0 44 44'%3E%3Crect width='44' height='44' fill='%23183a69'/%3E%3Cpath d='M22 7C25 16 28 19 37 22C28 25 25 28 22 37C19 28 16 25 7 22C16 19 19 16 22 7Z' fill='%232f68ad'/%3E%3Ccircle cx='22' cy='22' r='3.5' fill='%23d4e3f6' fill-opacity='.6'/%3E%3C/svg%3E") 0 0 / 44px 44px;
  }
  .tag {
    display: inline-flex;
    align-items: center;
    border-radius: 9999px;
    padding: 2px 10px;
    font-size: 12px;
    font-weight: 700;
  }
</style>
