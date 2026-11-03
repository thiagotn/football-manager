<script lang="ts">
  /** Hub de games.thiagotn.com — vitrine dos jogos instaláveis. */
  import { Gamepad2 } from 'lucide-svelte';
  import GameCard from '$lib/components/GameCard.svelte';
  import LanguageSwitcher from '$lib/components/LanguageSwitcher.svelte';
  import Patterns from '$lib/enche-ai/Patterns.svelte';
  import EncheAiArt from '$lib/enche-ai/CardArt.svelte';
  import PacienciaArt from '$lib/paciencia/CardArt.svelte';
  import TetrisArt from '$lib/tetris/CardArt.svelte';
  import { GAMES } from '$lib/games';
  import { t } from '$lib/i18n';

  /** Arte e cor de destaque por jogo — o resto vem do registro. */
  const ART = { 'enche-ai': EncheAiArt, paciencia: PacienciaArt, tetris: TetrisArt } as const;
  const ACCENT: Record<string, string> = {
    'enche-ai': '#ffc93c',
    paciencia: '#1e6b4a',
    tetris: '#2e7cf0',
  };
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
  <div class="mb-6 flex items-start justify-between gap-4">
    <div>
      <h1 class="flex items-center gap-2 text-2xl font-bold text-white">
        <Gamepad2 size={24} class="text-mustard" />
        {$t('hub.title')}
      </h1>
      <p class="mt-0.5 text-sm text-white/60">{$t('hub.subtitle')}</p>
    </div>
    <LanguageSwitcher tone="dark" />
  </div>

  <div class="games-grid grid gap-4">
    {#each GAMES as game (game.slug)}
      <GameCard {game} Art={ART[game.slug as keyof typeof ART]} accent={ACCENT[game.slug]} />
    {/each}
  </div>

  <p class="mt-8 text-sm text-white/50">{$t('hub.more')}</p>
</main>

<style>
  .games-grid {
    grid-template-columns: repeat(auto-fill, minmax(min(100%, 320px), 1fr));
  }
</style>
