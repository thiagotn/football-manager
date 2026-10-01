<script lang="ts">
  /**
   * /games — seletor de passatempo (handoff design_handoff_paciencia_games).
   *
   * Página intermediária entre a Navbar (Conta › Passatempo) e os dois jogos.
   * A arte de cada card é desenhada em CSS, sem imagem: gramado listrado com o
   * leque de cartas na Paciência, poço neon no Tetris.
   */
  import '@fontsource/bebas-neue/400.css';
  import { ArrowRight, Gamepad2 } from 'lucide-svelte';
  import PageBackground from '$lib/components/PageBackground.svelte';
  import { t } from '$lib/i18n';

  /** Peças do poço do Tetris: cor + células (coluna, linha) na grade de 16px. */
  const TETRIS_PIECES: { color: string; cells: [number, number][] }[] = [
    { color: '#00e5f5', cells: [[0, 9], [1, 9], [2, 9], [3, 9]] },
    { color: '#ffd000', cells: [[8, 8], [9, 8], [8, 9], [9, 9]] },
    { color: '#ee00bb', cells: [[4, 9], [5, 9], [6, 9], [5, 8]] },
    { color: '#00ee77', cells: [[0, 8], [1, 8], [1, 7], [2, 7]] },
    { color: '#ff2244', cells: [[6, 8], [7, 8], [7, 9], [6, 7]] },
    { color: '#3377ff', cells: [[4, 3], [4, 4], [5, 4], [6, 4]] },
  ];

  const blocks = TETRIS_PIECES.flatMap((piece, pi) =>
    piece.cells.map(([x, y], ci) => ({
      key: `${pi}-${ci}`,
      color: piece.color,
      x: x * 16,
      y: y * 16 - 4,
    }))
  );

  /** Leque de 4 cartas da arte da Paciência: verso + A♠ + K♥ + Q♦ (destacada). */
  const FAN = [
    { kind: 'back' as const, left: -90, bottom: 0, rotate: -16 },
    { kind: 'face' as const, left: -46, bottom: 6, rotate: -5, rank: 'A', suit: '♠︎', red: false, gold: false },
    { kind: 'face' as const, left: -12, bottom: 6, rotate: 6, rank: 'K', suit: '♥︎', red: true, gold: false },
    { kind: 'face' as const, left: 26, bottom: 0, rotate: 17, rank: 'Q', suit: '♦︎', red: true, gold: true },
  ];
</script>

<svelte:head>
  <title>{$t('games.title')} — rachao.app</title>
</svelte:head>

<PageBackground>
  <main class="relative z-10 max-w-7xl mx-auto px-4 py-8">
    <div class="flex items-center justify-between mb-6">
      <div>
        <h1 class="text-2xl font-bold text-white flex items-center gap-2">
          <Gamepad2 size={24} class="text-primary-400" /> {$t('games.title')}
        </h1>
        <p class="text-sm text-white/60 mt-0.5">{$t('games.subtitle')}</p>
      </div>
    </div>

    <div class="games-grid grid gap-4">
      <!-- Paciência -->
      <a
        href="/paciencia"
        class="group flex flex-col overflow-hidden rounded-xl border border-gray-700 bg-gray-800 transition-[transform,border-color] duration-150 hover:-translate-y-0.5 hover:border-primary-600"
      >
        <div
          class="relative h-[168px] overflow-hidden"
          style="background:repeating-linear-gradient(180deg,#168641 0 34px,#13793b 34px 68px)"
        >
          <div
            class="absolute inset-0"
            style="background:radial-gradient(120% 90% at 50% 40%,transparent 45%,rgba(7,51,26,.6) 100%)"
          ></div>
          <div class="absolute inset-2 rounded-[3px] border-2 border-white/[.16]"></div>
          <div
            class="absolute left-1/2 top-1/2 -ml-12 -mt-12 h-24 w-24 rounded-full border-2 border-white/[.16]"
          ></div>

          <div class="absolute bottom-[22px] left-1/2 h-0 w-0">
            {#each FAN as card, i (i)}
              {#if card.kind === 'back'}
                <div
                  class="absolute h-[82px] w-[58px] overflow-hidden rounded-[7px] border-[1.5px] border-gold-edge shadow-[0_6px_14px_rgba(0,0,0,.35)]"
                  style="
                    left:{card.left}px;bottom:{card.bottom}px;
                    transform:rotate({card.rotate}deg);transform-origin:50% 100%;
                    background:repeating-linear-gradient(118deg,rgba(255,255,255,.07) 0 2px,transparent 2px 8px),linear-gradient(158deg,#15422a,#1f9a4d 40%,#0d5028 70%,#07331a)"
                >
                  <img
                    src="/logo.png"
                    alt=""
                    class="absolute left-1/2 top-1/2 -ml-5 -mt-5 h-10 w-10 object-cover"
                  />
                </div>
              {:else}
                <div
                  class="absolute h-[82px] w-[58px] rounded-[7px] bg-[linear-gradient(180deg,#fff,#f3f4f6)]
                    {card.gold
                    ? 'border-2 border-gold-400 shadow-[0_0_0_3px_rgba(245,213,133,.4),0_6px_14px_rgba(0,0,0,.35)]'
                    : 'border border-gray-300 shadow-[0_6px_14px_rgba(0,0,0,.35)]'}"
                  style="
                    left:{card.left}px;bottom:{card.bottom}px;
                    transform:rotate({card.rotate}deg);transform-origin:50% 100%;
                    color:{card.red ? '#dc2626' : '#111827'}"
                >
                  <div class="absolute left-1.5 top-[3px] flex flex-col items-center leading-[.85]">
                    <span class="font-bebas text-[24px]">{card.rank}</span>
                    <span class="text-[13px]">{card.suit}</span>
                  </div>
                  <div class="absolute bottom-[3px] right-[5px] text-[28px] leading-none">
                    {card.suit}
                  </div>
                </div>
              {/if}
            {/each}
          </div>

          <span
            class="absolute left-3 top-3 inline-flex items-center rounded-full bg-gold-400 px-2.5 py-0.5 text-xs font-semibold text-rachao-900"
          >
            {$t('games.badge_new')}
          </span>
        </div>

        <div class="flex flex-col gap-2.5 p-4">
          <div class="flex flex-col gap-0.5">
            <span class="font-bebas text-[30px] leading-none tracking-[.02em] text-white">
              {$t('games.paciencia_title')}
            </span>
            <span class="text-sm leading-[1.45] text-gray-400">
              {$t('games.paciencia_desc')}
            </span>
          </div>
          <div class="flex flex-wrap gap-1.5">
            <span class="badge bg-green-900/40 text-green-300">{$t('games.tag_cards')}</span>
            <span class="badge bg-yellow-900/40 text-yellow-300">{$t('games.tag_daily')}</span>
            <span class="badge bg-gray-700 text-gray-300">{$t('games.tag_offline')}</span>
          </div>
          <span
            class="inline-flex h-11 items-center justify-center gap-2 rounded-lg bg-primary-600 text-[15px] font-semibold text-white transition-colors group-hover:bg-primary-700"
          >
            {$t('games.paciencia_cta')}
            <ArrowRight size={16} strokeWidth={2.2} />
          </span>
        </div>
      </a>

      <!-- Tetris 3D -->
      <a
        href="/tetris"
        class="group flex flex-col overflow-hidden rounded-xl border border-gray-700 bg-gray-800 transition-[transform,border-color] duration-150 hover:-translate-y-0.5 hover:border-blue-500"
      >
        <div
          class="relative h-[168px] overflow-hidden"
          style="background:radial-gradient(90% 90% at 50% 30%,#0a1540 0%,#010112 70%)"
        >
          <div
            class="absolute inset-0 opacity-35"
            style="background:radial-gradient(circle,rgba(255,255,255,.5) .7px,transparent 1.2px) 0 0/22px 22px"
          ></div>
          <div
            class="absolute -bottom-px left-1/2 top-4 -ml-20 w-40 border border-b-0 border-[rgba(26,64,144,.6)]"
            style="background:repeating-linear-gradient(90deg,rgba(12,24,56,.6) 0 1px,transparent 1px 16px),repeating-linear-gradient(180deg,rgba(12,24,56,.6) 0 1px,transparent 1px 16px)"
          >
            {#each blocks as b (b.key)}
              <div
                class="absolute m-px h-3.5 w-3.5 rounded-sm"
                style="left:{b.x}px;top:{b.y}px;background:{b.color};box-shadow:0 0 8px {b.color}"
              ></div>
            {/each}
          </div>
          <span
            class="absolute left-3 top-3 inline-flex items-center rounded-full bg-blue-900/60 px-2.5 py-0.5 text-xs font-semibold text-blue-300"
          >
            {$t('games.badge_3d')}
          </span>
        </div>

        <div class="flex flex-col gap-2.5 p-4">
          <div class="flex flex-col gap-0.5">
            <span class="font-bebas text-[30px] leading-none tracking-[.02em] text-white">
              {$t('games.tetris_title')}
            </span>
            <span class="text-sm leading-[1.45] text-gray-400">{$t('games.tetris_desc')}</span>
          </div>
          <div class="flex flex-wrap gap-1.5">
            <span class="badge bg-blue-900/40 text-blue-300">{$t('games.tag_arcade')}</span>
            <span class="badge bg-gray-700 text-gray-300">{$t('games.tag_input')}</span>
          </div>
          <span
            class="inline-flex h-11 items-center justify-center gap-2 rounded-lg border border-gray-600 bg-gray-700 text-[15px] font-semibold text-gray-200 transition-colors group-hover:bg-gray-600"
          >
            {$t('games.tetris_cta')}
            <ArrowRight size={16} strokeWidth={2.2} />
          </span>
        </div>
      </a>
    </div>
  </main>
</PageBackground>

<style>
  /* 1 coluna no mobile, 2–3 no desktop */
  .games-grid {
    grid-template-columns: repeat(auto-fill, minmax(min(100%, 320px), 1fr));
  }
</style>
