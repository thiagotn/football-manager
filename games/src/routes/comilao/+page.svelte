<script lang="ts">
  /**
   * Comilão — o clássico do labirinto, no visual do games.thiagotn.com.
   *
   * Rota em tela cheia (`fixed inset-0`), sem Navbar, mobile-first. As abas
   * (Início · Diário · Visuais · Ajustes) ficam por baixo; a partida entra por
   * cima delas. As regras vivem em `$lib/comilao/engine.ts`, o labirinto em
   * `maze.ts`, o desenho em `render.ts` e a persistência em `storage.ts` — aqui
   * só mora o laço do jogo, a entrada do jogador e a interface.
   */
  import {
    ArrowDown,
    ArrowLeft,
    ArrowRight,
    ArrowUp,
    Calendar,
    ChevronLeft,
    House,
    Lock,
    Palette,
    Pause,
    Play,
    Settings as SettingsIcon,
    Target,
    Trophy,
  } from 'lucide-svelte';
  import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
  import LanguageSwitcher from '$lib/components/LanguageSwitcher.svelte';
  import Emblem from '$lib/comilao/Emblem.svelte';
  import {
    COLS,
    GHOST_IDS,
    ROWS,
    dailyGoal,
    dateKey,
    fmtScore,
    newGame,
    reachedGoal,
    tick,
    turn,
    type Dir,
    type Game,
    type Pace,
  } from '$lib/comilao/engine';
  import { createRenderer, POP_MS, type Pop, type Renderer } from '$lib/comilao/render';
  import { GHOST_COLORS, SKINS, nextLock, skinOf, unlockedSkins } from '$lib/comilao/skins';
  import {
    HISTORY_VISIBLE,
    addToStats,
    emptyState,
    loadState,
    recordDaily,
    recordMatch,
    resumeGame,
    saveState,
    streakOf,
    type SavedState,
  } from '$lib/comilao/storage';
  import { locale, t } from '$lib/i18n';

  type Screen = 'home' | 'game' | 'daily' | 'skins' | 'settings';

  type Result = {
    score: number;
    level: number;
    pellets: number;
    ghosts: number;
    record: boolean;
    daily: string | null;
    goal: number;
    goalDone: boolean;
    /** Visuais que esta partida liberou. */
    unlocked: string[];
  };

  /** Passo fixo do engine: a física não depende da taxa de quadros. */
  const STEP = 1000 / 60;
  /** Teto de passos por quadro — quadro perdido não vira avalanche. */
  const MAX_STEPS = 4;

  let screen = $state<Screen>('home');
  let saved = $state<SavedState>(emptyState());
  /** `raw`: o engine devolve um estado novo a cada passo, não há o que observar por dentro. */
  let game = $state.raw<Game | null>(null);
  let paused = $state(false);
  let result = $state<Result | null>(null);
  let confirmReset = $state(false);
  let canvasFailed = $state(false);

  let canvasEl = $state<HTMLCanvasElement | null>(null);
  let boardEl = $state<HTMLDivElement | null>(null);

  let renderer: Renderer | null = null;
  let raf = 0;
  let acc = 0;
  let last = 0;
  let motion = true;
  /** Números que sobem quando o jogador come fantasma ou fruta (só o canvas lê). */
  let pops: Pop[] = [];
  let popId = 0;

  const today = new Date();
  const todayKey = dateKey(today);

  let stats = $derived(saved.stats);
  let settings = $derived(saved.settings);
  let skin = $derived(skinOf(saved.settings.skin));
  let unlocked = $derived(unlockedSkins(saved.stats.total));
  let upcoming = $derived(nextLock(saved.stats.total));
  let resumable = $derived(saved.game);
  let todayRecord = $derived(saved.daily[todayKey]);
  let streak = $derived(streakOf(saved.daily, today));
  let doneDays = $derived(Object.values(saved.daily).filter((d) => d.done).length);
  let dailyBest = $derived(
    Object.values(saved.daily).reduce((best, d) => Math.max(best, d.best), 0)
  );
  let history = $derived(saved.history.slice(0, HISTORY_VISIBLE));

  $effect(() => {
    saved = loadState();
    motion = !window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  });

  function persist(next: SavedState) {
    saved = next;
    saveState(next);
  }

  /** Guarda a partida em andamento (não a cada quadro: só nos marcos). */
  function keep(g: Game | null) {
    persist({ ...saved, game: g && g.phase !== 'fim' ? g : null });
  }

  // ── Partida ───────────────────────────────────────────────────────────────

  function startGame(daily: string | null) {
    game = newGame({ daily, pace: settings.pace });
    paused = false;
    result = null;
    pops = [];
    acc = 0;
    screen = 'game';
    keep(game);
  }

  /** Retomar cai no "Prepare-se": ninguém volta no colo de um fantasma. */
  function continueGame() {
    if (!saved.game) return;
    game = resumeGame(saved.game);
    paused = false;
    result = null;
    pops = [];
    acc = 0;
    screen = 'game';
  }

  function quitGame() {
    if (game && !result) keep(game);
    else keep(null);
    game = null;
    result = null;
    paused = false;
    screen = 'home';
  }

  function togglePause() {
    if (!game || result) return;
    paused = !paused;
    if (paused) keep(game);
  }

  /** Fecha a partida: soma às estatísticas, grava o dia e monta a tela de fim. */
  function finish(g: Game) {
    const bestBefore = stats.best;
    const unlockedBefore = unlockedSkins(stats.total);
    const next: SavedState = {
      game: null,
      stats: addToStats(stats, g),
      history: recordMatch(saved.history, g),
      daily: g.daily ? recordDaily(saved.daily, g.daily, g.score, g.goal) : saved.daily,
      settings,
    };
    persist(next);
    result = {
      score: g.score,
      level: g.level,
      pellets: g.pellets,
      ghosts: g.ghostsEaten,
      record: g.score > bestBefore && g.score > 0,
      daily: g.daily,
      goal: g.goal,
      goalDone: reachedGoal(g),
      unlocked: unlockedSkins(next.stats.total).filter((k) => !unlockedBefore.includes(k)),
    };
  }

  // ── Laço do jogo ──────────────────────────────────────────────────────────

  function fit() {
    if (!boardEl || !renderer) return;
    const box = boardEl.getBoundingClientRect();
    renderer.resize(Math.min(box.width / COLS, box.height / ROWS));
  }

  function frame(now: number) {
    raf = requestAnimationFrame(frame);
    const dt = Math.min(200, now - last);
    last = now;

    let g = game;
    if (!g || !renderer) return;

    if (g.phase !== 'fim' && !paused) {
      acc = Math.min(acc + dt, STEP * MAX_STEPS);
      while (acc >= STEP) {
        acc -= STEP;
        const out = tick(g, STEP);
        g = out.game;
        for (const event of out.events) {
          if (event.kind === 'fantasma' || event.kind === 'fruta') {
            pops = [
              ...pops,
              { id: popId++, text: fmtScore(event.points), x: event.x, y: event.y, born: now },
            ];
          }
          // Marcos valem salvar: morte, virada de fase e fim de partida.
          if (event.kind === 'morreu' || event.kind === 'fase') keep(g);
          if (event.kind === 'fim') finish(g);
        }
      }
      game = g;
      if (pops.length) pops = pops.filter((p) => now - p.born < POP_MS);
    }

    renderer.draw(g, { skin, glow: settings.glow, time: now, motion, pops });
  }

  $effect(() => {
    if (screen !== 'game' || !canvasEl || !boardEl) return;
    try {
      renderer = createRenderer(canvasEl);
    } catch {
      canvasFailed = true;
      return;
    }
    const observer = new ResizeObserver(fit);
    observer.observe(boardEl);
    fit();
    last = performance.now();
    raf = requestAnimationFrame(frame);
    return () => {
      cancelAnimationFrame(raf);
      observer.disconnect();
      renderer?.dispose();
      renderer = null;
    };
  });

  /** Sair da aba pausa e guarda: no celular, a ligação chega no meio da fase. */
  function onHide() {
    if (document.visibilityState !== 'hidden') return;
    if (game && !result) {
      paused = true;
      keep(game);
    }
  }

  // ── Entrada ───────────────────────────────────────────────────────────────

  const KEYS: Record<string, Dir> = {
    ArrowUp: 'up',
    ArrowDown: 'down',
    ArrowLeft: 'left',
    ArrowRight: 'right',
    w: 'up',
    s: 'down',
    a: 'left',
    d: 'right',
  };

  function onKeydown(event: KeyboardEvent) {
    if (screen !== 'game' || !game) return;
    const dir = KEYS[event.key] ?? KEYS[event.key.toLowerCase()];
    if (dir) {
      event.preventDefault();
      press(dir);
      return;
    }
    if (event.key === ' ' || event.key === 'p' || event.key === 'Escape') {
      event.preventDefault();
      togglePause();
    }
  }

  function press(dir: Dir) {
    if (!game || result) return;
    game = turn(game, dir);
    paused = false;
  }

  /** Arraste em qualquer ponto do labirinto: 22px já contam como virada. */
  let swipeFrom: { x: number; y: number } | null = null;

  function onPointerDown(event: PointerEvent) {
    swipeFrom = { x: event.clientX, y: event.clientY };
    if (paused && !result) paused = false;
  }

  function onPointerMove(event: PointerEvent) {
    if (!swipeFrom || !game) return;
    const dx = event.clientX - swipeFrom.x;
    const dy = event.clientY - swipeFrom.y;
    if (Math.hypot(dx, dy) < 22) return;
    press(Math.abs(dx) > Math.abs(dy) ? (dx > 0 ? 'right' : 'left') : dy > 0 ? 'down' : 'up');
    swipeFrom = { x: event.clientX, y: event.clientY };
  }

  // ── Ajustes e visuais ─────────────────────────────────────────────────────

  function update<K extends keyof SavedState['settings']>(key: K, value: SavedState['settings'][K]) {
    persist({ ...saved, settings: { ...settings, [key]: value } });
  }

  function chooseSkin(key: string) {
    if (!unlocked.includes(key as (typeof unlocked)[number])) return;
    update('skin', key as SavedState['settings']['skin']);
  }

  function resetStats() {
    persist({ ...emptyState(), settings: { ...settings, skin: 'synth' } });
  }

  // ── Datas e rótulos ───────────────────────────────────────────────────────

  let weekdays = $derived($t('comilao.weekdays').split(','));

  let todayLabel = $derived(
    today
      .toLocaleDateString($locale, { weekday: 'long', day: 'numeric', month: 'short' })
      .replace(/\./g, '')
  );

  let monthLabel = $derived.by(() => {
    const raw = today
      .toLocaleDateString($locale, { month: 'long', year: 'numeric' })
      .replace(' de ', ' ');
    return raw.charAt(0).toUpperCase() + raw.slice(1);
  });

  let calendar = $derived.by(() => {
    const y = today.getFullYear();
    const m = today.getMonth();
    const startDow = (new Date(y, m, 1).getDay() + 6) % 7; // a semana começa na segunda
    const total = new Date(y, m + 1, 0).getDate();
    const cells: {
      key: string;
      day: number | null;
      date: string;
      today: boolean;
      done: boolean;
      future: boolean;
    }[] = [];
    for (let i = 0; i < startDow; i++) {
      cells.push({ key: `pad-${i}`, day: null, date: '', today: false, done: false, future: false });
    }
    for (let d = 1; d <= total; d++) {
      const key = dateKey(new Date(y, m, d));
      cells.push({
        key,
        day: d,
        date: key,
        today: key === todayKey,
        done: !!saved.daily[key]?.done,
        future: key > todayKey,
      });
    }
    return cells;
  });

  let hudBoxes = $derived([
    { key: 'level', label: $t('comilao.hud_level'), value: String(game?.level ?? 1) },
    { key: 'pellets', label: $t('comilao.hud_pellets'), value: String(game?.pellets ?? 0) },
    { key: 'best', label: $t('comilao.hud_best'), value: fmtScore(stats.best) },
  ]);

  const PACES: { key: Pace; labelKey: string }[] = [
    { key: 'calma', labelKey: 'comilao.pace_calma' },
    { key: 'normal', labelKey: 'comilao.pace_normal' },
    { key: 'veloz', labelKey: 'comilao.pace_veloz' },
  ];

  const TABS = [
    { key: 'home' as const, labelKey: 'comilao.tab_home', icon: House },
    { key: 'daily' as const, labelKey: 'comilao.tab_daily', icon: Calendar },
    { key: 'skins' as const, labelKey: 'comilao.tab_skins', icon: Palette },
    { key: 'settings' as const, labelKey: 'comilao.tab_settings', icon: SettingsIcon },
  ];

  /** O aviso que entra por cima do labirinto, sem tapar o jogo. */
  let banner = $derived(
    !game || result
      ? null
      : game.phase === 'pronto'
        ? game.level > 1
          ? $t('comilao.level_next', { n: game.level })
          : $t('comilao.ready')
        : game.phase === 'fase'
          ? $t('comilao.level_clear')
          : null
  );
</script>

<svelte:head>
  <title>{$t('comilao.title')} — games.thiagotn.com</title>
  <meta name="description" content={$t('comilao.desc')} />
  <meta name="theme-color" content="#140a24" />
  <link rel="manifest" href="/comilao/manifest.webmanifest" />
  <link rel="apple-touch-icon" href="/comilao/apple-touch-icon.png" />
  <meta name="apple-mobile-web-app-title" content={$t('comilao.title')} />
</svelte:head>

<svelte:window onkeydown={onKeydown} />
<svelte:document onvisibilitychange={onHide} />

<div class="comilao fixed inset-0 z-50 flex flex-col overflow-hidden bg-synth text-white">
  {#if screen === 'game' && game}
    <!-- ── Partida ───────────────────────────────────────────────────── -->
    <header
      class="relative z-10 flex-none px-3 pb-2"
      style="padding-top:calc(env(safe-area-inset-top) + 8px)"
    >
      <div class="flex items-center gap-2">
        <button
          type="button"
          onclick={quitGame}
          aria-label={$t('comilao.quit')}
          class="flex h-10 w-10 flex-none items-center justify-center rounded-xl border border-white/15 bg-white/5 text-white/80 transition-colors hover:bg-white/10"
        >
          <ChevronLeft size={20} strokeWidth={2.2} />
        </button>

        <div class="flex min-w-0 flex-1 flex-col items-center">
          <span class="text-[10px] font-bold uppercase tracking-[.18em] text-mustard">
            {game.daily ? $t('comilao.daily_kicker') : $t('comilao.hud_score')}
          </span>
          <span class="font-display text-[34px] leading-none tabular-nums">
            {fmtScore(game.score)}
          </span>
        </div>

        <button
          type="button"
          onclick={togglePause}
          disabled={!!result}
          aria-label={$t('comilao.pause')}
          class="flex h-10 w-10 flex-none items-center justify-center rounded-xl border border-white/15 bg-white/5 text-white/80 transition-colors hover:bg-white/10 disabled:opacity-40"
        >
          {#if paused}<Play size={18} fill="currentColor" />{:else}<Pause size={18} />{/if}
        </button>
      </div>

      <div class="mt-2 grid grid-cols-3 gap-2">
        {#each hudBoxes as box (box.key)}
          <div class="rounded-xl border border-white/10 bg-white/[.04] py-1.5 text-center">
            <div class="text-[9px] font-bold uppercase tracking-[.14em] text-white/50">
              {box.label}
            </div>
            <div class="font-display text-[18px] leading-tight tabular-nums">{box.value}</div>
          </div>
        {/each}
      </div>

      <div class="mt-2 flex items-center gap-2">
        <span class="sr-only">{$t('comilao.hud_lives')}</span>
        <div class="flex flex-1 items-center gap-1.5" aria-label="{$t('comilao.hud_lives')}: {game.lives}">
          {#each Array(Math.min(game.lives, 6)) as _, i (i)}
            <svg width="16" height="16" viewBox="0 0 100 100" aria-hidden="true">
              <path d="M50 50 L95 24 A50 50 0 1 0 95 76 Z" fill={skin.hero} />
            </svg>
          {/each}
          {#if game.lives > 6}
            <span class="text-[11px] font-bold text-white/60">+{game.lives - 6}</span>
          {/if}
        </div>

        {#if game.goal > 0}
          <Target size={14} class="flex-none text-mustard" />
          <div class="h-2 w-28 overflow-hidden rounded-full bg-white/10">
            <div
              class="h-full rounded-full bg-mustard transition-[width] duration-300"
              style="width:{Math.min(100, (game.score / game.goal) * 100)}%"
            ></div>
          </div>
          <span class="flex-none text-[11px] font-bold tabular-nums text-white/70">
            {game.score}/{game.goal}
          </span>
        {/if}
      </div>
    </header>

    <!-- Labirinto -->
    <div
      bind:this={boardEl}
      role="application"
      aria-label={$t('comilao.title')}
      class="relative z-10 min-h-0 flex-1 touch-none select-none p-2"
      onpointerdown={onPointerDown}
      onpointermove={onPointerMove}
      onpointerup={() => (swipeFrom = null)}
      onpointercancel={() => (swipeFrom = null)}
    >
      <div class="flex h-full w-full items-center justify-center">
        {#if canvasFailed}
          <p class="max-w-xs text-center text-sm text-white/70">{$t('comilao.no_canvas')}</p>
        {:else}
          <div class="relative">
            <canvas
              bind:this={canvasEl}
              class="block rounded-2xl shadow-[0_18px_40px_rgba(0,0,0,.5)] ring-1 ring-white/10"
            ></canvas>
            {#if banner}
              <span
                class="pointer-events-none absolute inset-x-0 top-1/2 mx-auto w-max max-w-[85%] -translate-y-1/2 rounded-full bg-black/70 px-4 py-1.5 text-center font-display text-[17px] text-mustard"
              >
                {banner}
              </span>
            {/if}
          </div>
        {/if}
      </div>
    </div>

    <!-- Direcional -->
    {#if settings.pad}
      <div
        class="relative z-10 flex-none pb-2 pt-1"
        style="padding-bottom:calc(env(safe-area-inset-bottom) + 8px)"
      >
        <div class="mx-auto grid w-[168px] grid-cols-3 grid-rows-3 gap-1.5">
          {#each [{ dir: 'up' as const, icon: ArrowUp, cls: 'col-start-2 row-start-1', label: $t('comilao.dir_up') }, { dir: 'left' as const, icon: ArrowLeft, cls: 'col-start-1 row-start-2', label: $t('comilao.dir_left') }, { dir: 'right' as const, icon: ArrowRight, cls: 'col-start-3 row-start-2', label: $t('comilao.dir_right') }, { dir: 'down' as const, icon: ArrowDown, cls: 'col-start-2 row-start-3', label: $t('comilao.dir_down') }] as btn (btn.dir)}
            <button
              type="button"
              class="pad {btn.cls}"
              aria-label={btn.label}
              onpointerdown={(e) => {
                e.preventDefault();
                press(btn.dir);
              }}
            >
              <btn.icon size={22} strokeWidth={2.4} />
            </button>
          {/each}
        </div>
      </div>
    {:else}
      <div class="flex-none" style="height:calc(env(safe-area-inset-bottom) + 8px)"></div>
    {/if}

    <!-- Pausa -->
    {#if paused && !result}
      <div class="overlay">
        <div class="card flex flex-col gap-3 text-center">
          <h2 class="font-display text-[30px] leading-none">{$t('comilao.paused')}</h2>
          <p class="text-sm font-medium text-paper-muted">{$t('comilao.controls_hint')}</p>
          <button class="btn btn-primary" onclick={() => (paused = false)}>
            {$t('comilao.resume')}
          </button>
          <button class="btn btn-ghost" onclick={quitGame}>{$t('comilao.quit')}</button>
        </div>
      </div>
    {/if}

    <!-- Fim de partida -->
    {#if result}
      <div class="overlay">
        <div class="card flex flex-col gap-3">
          <div class="text-center">
            <div class="text-[11px] font-bold uppercase tracking-[.2em] text-ketchup">
              {$t('comilao.over_kicker')}
            </div>
            <h2 class="font-display text-[34px] leading-none">{$t('comilao.over_title')}</h2>
          </div>

          {#if result.record}
            <p class="tag mx-auto bg-mustard text-mustard-ink">{$t('comilao.over_new_record')}</p>
          {/if}
          {#if result.daily}
            <p
              class="text-center text-sm font-bold {result.goalDone
                ? 'text-[#2f8f45]'
                : 'text-paper-muted'}"
            >
              {result.goalDone
                ? $t('comilao.over_goal_done')
                : $t('comilao.over_goal_missed', { goal: result.goal })}
            </p>
          {/if}

          <!-- Esta partida × De sempre -->
          <table class="w-full text-left">
            <thead>
              <tr class="stat-label">
                <th class="font-bold"></th>
                <th class="pb-1 text-right font-bold">{$t('comilao.col_current')}</th>
                <th class="pb-1 text-right font-bold">{$t('comilao.col_alltime')}</th>
              </tr>
            </thead>
            <tbody>
              {#each [{ key: 'score', label: $t('comilao.hud_score'), now: fmtScore(result.score), ever: fmtScore(stats.best) }, { key: 'level', label: $t('comilao.hud_level'), now: String(result.level), ever: String(stats.bestLevel) }, { key: 'pellets', label: $t('comilao.hud_pellets'), now: fmtScore(result.pellets), ever: fmtScore(stats.pellets) }, { key: 'ghosts', label: $t('comilao.hud_ghosts'), now: String(result.ghosts), ever: fmtScore(stats.ghosts) }] as row (row.key)}
                <tr class="border-t border-paper-ink/10">
                  <td class="py-1.5 text-[15px] font-extrabold">{row.label}</td>
                  <td class="stat py-1.5 text-right text-[22px]">{row.now}</td>
                  <td class="stat py-1.5 text-right text-[22px] text-paper-muted">{row.ever}</td>
                </tr>
              {/each}
            </tbody>
          </table>

          {#if result.unlocked.length}
            <p class="text-center text-sm font-bold text-paper-ink">
              {$t('comilao.over_unlocked', {
                skin: result.unlocked.map((k) => $t(skinOf(k).nameKey)).join(', '),
              })}
            </p>
          {/if}

          <button class="btn btn-primary" onclick={() => startGame(result?.daily ?? null)}>
            {$t('comilao.play_again')}
          </button>
          <button class="btn btn-ghost" onclick={quitGame}>{$t('comilao.quit')}</button>
        </div>
      </div>
    {/if}
  {:else}
    <!-- ── Abas ──────────────────────────────────────────────────────── -->
    <div
      class="pointer-events-none absolute inset-x-0 top-0 h-[320px]"
      style="background:radial-gradient(90% 55% at 50% 0%,#3a1f6b 0%,#1f1038 45%,transparent 75%)"
    ></div>
    <div
      class="pointer-events-none absolute inset-0"
      style="background:linear-gradient(180deg,rgba(20,10,36,.35) 0%,rgba(20,10,36,.9) 45%,#140a24 72%)"
    ></div>

    <div
      class="relative z-10 flex-none px-3 pb-1"
      style="padding-top:calc(env(safe-area-inset-top) + 8px)"
    >
      <a
        href="/"
        class="inline-flex items-center gap-1.5 rounded-lg px-2 py-1.5 text-[13px] font-semibold text-white/75 transition-colors hover:bg-white/10 hover:text-white"
      >
        <ChevronLeft size={16} strokeWidth={2.2} /> {$t('comilao.back_to_games')}
      </a>
    </div>

    <div class="relative z-10 min-h-0 flex-1 overflow-y-auto">
      <div class="mx-auto max-w-md">
        {#if screen === 'home'}
          <div class="flex flex-col items-center px-[22px] pb-[26px] pt-[10px] text-center">
            <div class="drop-shadow-[0_16px_30px_rgba(255,61,240,.26)]">
              <Emblem size={132} />
            </div>
            <div
              class="mt-1 bg-[linear-gradient(180deg,#ffe9a8_0%,#ffc93c_52%,#ff3df0_100%)] bg-clip-text pt-1 font-display text-[44px] leading-none tracking-[.01em] text-transparent"
            >
              {$t('comilao.title')}
            </div>

            {#if resumable}
              <div
                class="mt-[18px] flex w-full items-center gap-3.5 rounded-[14px] border border-mustard/35 bg-[linear-gradient(158deg,#2e1a52_0%,#1f1038_55%,#140a24_100%)] py-3.5 pl-4 pr-3.5 text-left"
              >
                <div
                  class="flex h-[46px] w-[46px] flex-none items-center justify-center rounded-xl border border-mustard/45 bg-black/25 text-mustard"
                >
                  <Play size={22} fill="currentColor" />
                </div>
                <div class="flex min-w-0 flex-1 flex-col gap-px">
                  <span class="text-[10.5px] font-bold uppercase tracking-[.16em] text-mustard">
                    {$t('comilao.continue_kicker')}
                  </span>
                  <span class="truncate font-display text-[19px] leading-tight text-white">
                    {$t('comilao.continue_desc', {
                      level: resumable.level,
                      score: fmtScore(resumable.score),
                    })}
                  </span>
                </div>
                <button
                  type="button"
                  onclick={continueGame}
                  class="h-10 flex-none rounded-[10px] bg-mustard px-4 font-display text-[15px] text-mustard-ink transition-colors hover:bg-[#ffd666] active:scale-95"
                >
                  {$t('comilao.continue_game')}
                </button>
              </div>
            {/if}

            <!-- Desafio do dia -->
            <div
              class="relative mt-3.5 flex w-full items-center gap-3.5 overflow-hidden rounded-[14px] border border-mustard/35 bg-[linear-gradient(158deg,#3a1f6b_0%,#2e1a52_45%,#140a24_100%)] py-3.5 pl-4 pr-3.5 text-left"
            >
              <div
                class="relative flex h-[46px] w-[46px] flex-none items-center justify-center rounded-xl border border-mustard/45 bg-black/25 text-mustard"
              >
                <Calendar size={24} />
              </div>
              <div class="relative flex min-w-0 flex-1 flex-col gap-px">
                <span class="text-[10.5px] font-bold uppercase tracking-[.16em] text-mustard">
                  {$t('comilao.daily_today')}
                </span>
                <span class="truncate font-display text-[19px] leading-tight text-white">
                  {todayLabel}
                </span>
                <span class="text-[12px] font-medium text-white/70">
                  {todayRecord?.done
                    ? $t('comilao.daily_status_done', { score: fmtScore(todayRecord.best) })
                    : $t('comilao.daily_status_todo')}
                </span>
              </div>
              <button
                type="button"
                onclick={() => startGame(todayKey)}
                class="relative h-10 flex-none rounded-[10px] bg-mustard px-4 font-display text-[15px] text-mustard-ink transition-colors hover:bg-[#ffd666] active:scale-95"
              >
                {todayRecord ? $t('comilao.replay') : $t('comilao.play')}
              </button>
            </div>

            <button
              type="button"
              onclick={() => startGame(null)}
              class="mt-3.5 flex h-[54px] w-full items-center justify-center gap-2.5 rounded-xl bg-mustard font-display text-[19px] text-mustard-ink shadow-[0_8px_20px_rgba(255,201,60,.3)] transition-colors hover:bg-[#ffd666] active:scale-[.98]"
            >
              <Play size={20} fill="currentColor" />
              {$t('comilao.new_game')}
            </button>

            <div
              class="mt-4 flex flex-wrap justify-center gap-x-[18px] gap-y-1 text-[12px] font-medium text-gray-400"
            >
              <span>
                <b class="font-bold text-white">{fmtScore(stats.best)}</b>
                {$t('comilao.summary_best')}
              </span>
              <span aria-hidden="true">·</span>
              <span>
                <b class="font-bold text-white">{stats.games}</b> {$t('comilao.summary_games')}
              </span>
              <span aria-hidden="true">·</span>
              <span>
                <b class="font-bold text-white">{streak}</b>
                {streak === 1 ? $t('comilao.summary_streak_one') : $t('comilao.summary_streak')}
              </span>
            </div>

            {#if history.length}
              <div class="mt-6 w-full text-left">
                <div class="section-label !text-white/45">{$t('comilao.recent')}</div>
                <ul class="flex flex-col gap-1.5">
                  {#each history as entry (entry.at)}
                    <li
                      class="flex items-center gap-3 rounded-xl border border-white/10 bg-white/[.04] px-3 py-2"
                    >
                      <span
                        class="flex h-7 w-7 flex-none items-center justify-center rounded-lg bg-mustard/15 text-mustard"
                      >
                        {#if entry.daily}<Calendar size={14} />{:else}<Trophy size={14} />{/if}
                      </span>
                      <span class="min-w-0 flex-1 truncate text-[13px] font-semibold text-white/80">
                        {entry.daily ? $t('comilao.entry_daily') : $t('comilao.entry_free')} · {$t(
                          'comilao.entry_level',
                          { n: entry.level }
                        )}
                      </span>
                      <span class="font-display text-[18px] tabular-nums">
                        {fmtScore(entry.score)}
                      </span>
                    </li>
                  {/each}
                </ul>
              </div>
            {/if}
          </div>
        {:else if screen === 'daily'}
          <div class="px-5 pb-[26px] pt-5">
            <div class="text-[11px] font-bold uppercase tracking-[.2em] text-mustard">
              {$t('comilao.daily_kicker')}
            </div>
            <div class="mt-0.5 font-display text-[30px] leading-tight">{monthLabel}</div>
            <p class="mt-1.5 text-[13.5px] leading-[1.45] text-white/70">
              {$t('comilao.daily_note')}
            </p>
            <p class="mt-1.5 text-[13px] font-bold text-mustard">
              {$t('comilao.daily_goal', { goal: fmtScore(dailyGoal(todayKey)) })}
            </p>

            <div class="mt-4 grid grid-cols-3 gap-2.5">
              {#each [{ key: 'done', label: $t('comilao.stat_done'), value: String(doneDays), gold: false }, { key: 'streak', label: $t('comilao.stat_streak'), value: String(streak), gold: true }, { key: 'best', label: $t('comilao.stat_daily_best'), value: dailyBest ? fmtScore(dailyBest) : '—', gold: false }] as box (box.key)}
                <div
                  class="rounded-[14px] border px-2 py-3 text-center {box.gold
                    ? 'border-mustard/45 bg-mustard/5'
                    : 'border-synth-light bg-synth-2/70'}"
                >
                  <div
                    class="font-display text-[26px] leading-none {box.gold ? 'text-mustard' : ''}"
                  >
                    {box.value}
                  </div>
                  <div class="mt-1 text-[10px] font-bold uppercase tracking-[.12em] text-gray-400">
                    {box.label}
                  </div>
                </div>
              {/each}
            </div>

            <div class="mt-3.5 rounded-[14px] border border-synth-light bg-synth-2/85 px-2.5 py-3">
              <div class="grid grid-cols-7 gap-[5px] text-center">
                {#each weekdays as w, i (i)}
                  <span class="pb-1 text-[10px] font-bold tracking-[.08em] text-gray-500">{w}</span>
                {/each}
                {#each calendar as cell (cell.key)}
                  {#if cell.day === null}
                    <span class="aspect-square"></span>
                  {:else}
                    <button
                      type="button"
                      onclick={() => startGame(cell.date)}
                      disabled={cell.future}
                      class="flex aspect-square items-center justify-center rounded-full text-[14px] transition-[filter] hover:brightness-125 disabled:hover:brightness-100
                        {cell.today
                        ? 'border-2 border-mustard'
                        : cell.done
                          ? 'border border-[#6bffb8]'
                          : cell.future
                            ? 'border border-transparent'
                            : 'border border-synth-light'}
                        {cell.done
                        ? 'bg-[#1d5c46] font-bold text-white'
                        : cell.future
                          ? 'text-gray-600'
                          : 'text-gray-200'}
                        {cell.today ? 'font-bold' : ''}"
                    >
                      {cell.day}
                    </button>
                  {/if}
                {/each}
              </div>
            </div>

            <button
              type="button"
              onclick={() => startGame(todayKey)}
              class="mt-4 h-[54px] w-full rounded-xl bg-mustard font-display text-[19px] text-mustard-ink shadow-[0_8px_20px_rgba(255,201,60,.3)] transition-colors hover:bg-[#ffd666]"
            >
              {$t('comilao.daily_play')}
            </button>
          </div>
        {:else if screen === 'skins'}
          <div class="px-5 pb-[26px] pt-5">
            <div class="text-[11px] font-bold uppercase tracking-[.2em] text-mustard">
              {$t('comilao.skins_kicker')}
            </div>
            <div class="mt-0.5 font-display text-[30px] leading-tight">
              {$t('comilao.skins_title')}
            </div>

            {#if upcoming}
              <div class="mt-3 rounded-[14px] border border-synth-light bg-synth-2/70 px-3.5 py-3">
                <div class="flex items-baseline justify-between gap-2">
                  <span class="text-[13px] font-bold text-white/85">
                    {$t('comilao.skins_next', { skin: $t(upcoming.nameKey) })}
                  </span>
                  <span class="text-[12px] font-bold tabular-nums text-mustard">
                    {fmtScore(stats.total)}/{fmtScore(upcoming.need)}
                  </span>
                </div>
                <div class="mt-2 h-2.5 overflow-hidden rounded-full bg-black/40">
                  <div
                    class="h-full rounded-full bg-[linear-gradient(90deg,#ffe29a,#ffc93c)]"
                    style="width:{Math.min(100, (stats.total / upcoming.need) * 100)}%"
                  ></div>
                </div>
              </div>
            {:else}
              <p class="mt-3 text-[13.5px] text-white/70">{$t('comilao.skins_all')}</p>
            {/if}

            <div class="mt-4 grid grid-cols-2 gap-2.5">
              {#each SKINS as item (item.key)}
                {@const open = unlocked.includes(item.key)}
                <button
                  type="button"
                  disabled={!open}
                  onclick={() => chooseSkin(item.key)}
                  class="flex flex-col items-center gap-2 rounded-[14px] border px-3 py-3.5 transition-colors
                    {settings.skin === item.key
                    ? 'border-mustard bg-mustard/10'
                    : open
                      ? 'border-synth-light bg-synth-2/70 hover:bg-synth-2'
                      : 'border-synth-2 bg-black/40'}"
                >
                  <div
                    class="relative flex h-[42px] w-full items-center justify-center overflow-hidden rounded-lg {open
                      ? ''
                      : 'opacity-35'}"
                    style="background:linear-gradient(180deg,{item.floor[0]},{item.floor[1]})"
                  >
                    <!-- Amostra: duas paredes, pastilhas e o herói -->
                    <span
                      class="absolute inset-x-2 top-[9px] h-[2px] rounded-full"
                      style="background:{item.wall[0]};box-shadow:0 0 6px {item.glow}"
                    ></span>
                    <span
                      class="absolute inset-x-2 bottom-[9px] h-[2px] rounded-full"
                      style="background:{item.wall[1]};box-shadow:0 0 6px {item.glow}"
                    ></span>
                    <span
                      class="relative h-[11px] w-[11px] rounded-full"
                      style="background:{item.hero}"
                    ></span>
                    <span
                      class="relative ml-2 h-[5px] w-[5px] rounded-full"
                      style="background:{item.pellet}"
                    ></span>
                    <span
                      class="relative ml-2 h-[8px] w-[8px] rounded-full"
                      style="background:{item.power}"
                    ></span>
                  </div>
                  <span class="flex items-center gap-1.5 text-[13px] font-bold">
                    {#if !open}<Lock size={12} class="text-gray-500" />{/if}
                    {$t(item.nameKey)}
                  </span>
                  <span class="text-[11px] font-semibold text-gray-400">
                    {#if settings.skin === item.key}
                      {$t('comilao.skins_in_use')}
                    {:else if open}
                      {$t('comilao.skins_use')}
                    {:else}
                      {$t('comilao.skins_need', { n: fmtScore(item.need) })}
                    {/if}
                  </span>
                </button>
              {/each}
            </div>

            <!-- Quem é quem: as cores dos fantasmas não mudam com o visual -->
            <div class="section-label !text-white/45">{$t('comilao.ghosts_title')}</div>
            <ul class="flex flex-col gap-1.5">
              {#each GHOST_IDS as id (id)}
                <li
                  class="flex items-center gap-3 rounded-xl border border-white/10 bg-white/[.04] px-3 py-2"
                >
                  <svg width="22" height="22" viewBox="0 0 32 35" class="flex-none" aria-hidden="true">
                    <path
                      d="M0 16a16 16 0 0 1 32 0v19l-5.3-5-5.3 5-5.4-5-5.3 5L0 35Z"
                      fill={GHOST_COLORS[id]}
                    />
                    <circle cx="10" cy="13" r="4.2" fill="#fff" />
                    <circle cx="22" cy="13" r="4.2" fill="#fff" />
                    <circle cx="9" cy="13.6" r="2" fill="#1b1040" />
                    <circle cx="21" cy="13.6" r="2" fill="#1b1040" />
                  </svg>
                  <span class="min-w-0 flex-1">
                    <span class="block text-[13.5px] font-bold">{$t(`comilao.ghost.${id}`)}</span>
                    <span class="block text-[12px] text-white/60">
                      {$t(`comilao.ghost_desc.${id}`)}
                    </span>
                  </span>
                </li>
              {/each}
            </ul>
            <p class="mt-2 text-[12px] leading-[1.45] text-white/45">{$t('comilao.ghosts_note')}</p>
          </div>
        {:else if screen === 'settings'}
          <div class="px-5 pb-[26px] pt-5">
            <div class="text-[11px] font-bold uppercase tracking-[.2em] text-mustard">
              {$t('comilao.tab_settings')}
            </div>
            <div class="mt-0.5 font-display text-[30px] leading-tight">
              {$t('comilao.settings_title')}
            </div>

            <div
              class="mt-4 divide-y divide-synth-light overflow-hidden rounded-[14px] border border-synth-light bg-synth-2/70"
            >
              <div class="flex items-center justify-between gap-3 px-3.5 py-3">
                <div class="min-w-0">
                  <div class="text-[15px] font-bold">{$t('comilao.setting_pace')}</div>
                  <div class="text-[12.5px] text-white/60">{$t('comilao.setting_pace_hint')}</div>
                </div>
                <div class="flex flex-none rounded-[10px] bg-black/35 p-0.5">
                  {#each PACES as option (option.key)}
                    <button
                      type="button"
                      onclick={() => update('pace', option.key)}
                      aria-pressed={settings.pace === option.key}
                      class="rounded-lg px-2.5 py-1.5 text-[12px] font-bold transition-colors
                        {settings.pace === option.key
                        ? 'bg-mustard text-mustard-ink'
                        : 'text-white/70'}"
                    >
                      {$t(option.labelKey)}
                    </button>
                  {/each}
                </div>
              </div>

              {#each [{ key: 'pad' as const, label: $t('comilao.setting_pad'), hint: $t('comilao.setting_pad_hint') }, { key: 'glow' as const, label: $t('comilao.setting_glow'), hint: $t('comilao.setting_glow_hint') }] as item (item.key)}
                <label class="flex cursor-pointer items-center justify-between gap-3 px-3.5 py-3">
                  <span class="min-w-0">
                    <span class="block text-[15px] font-bold">{item.label}</span>
                    <span class="block text-[12.5px] text-white/60">{item.hint}</span>
                  </span>
                  <input
                    type="checkbox"
                    class="h-6 w-6 flex-none accent-mustard"
                    checked={settings[item.key]}
                    onchange={(e) => update(item.key, e.currentTarget.checked)}
                  />
                </label>
              {/each}
            </div>

            <div class="mt-4 flex items-center justify-between gap-3">
              <span class="text-[15px] font-bold">{$t('comilao.language')}</span>
              <LanguageSwitcher tone="dark" />
            </div>

            <button
              type="button"
              onclick={() => (confirmReset = true)}
              class="mt-[22px] h-[46px] w-full rounded-xl border border-red-400/40 text-[14px] font-semibold text-red-300 transition-colors hover:bg-red-900/25"
            >
              {$t('comilao.reset_stats')}
            </button>
          </div>
        {/if}
      </div>
    </div>

    <!-- Abas inferiores -->
    <nav
      class="relative z-10 flex-none border-t border-synth-2 bg-[#0e0718]"
      style="padding-bottom:calc(env(safe-area-inset-bottom) + 6px)"
    >
      <div class="mx-auto grid max-w-lg grid-cols-4">
        {#each TABS as item (item.key)}
          <button
            type="button"
            onclick={() => (screen = item.key)}
            aria-current={screen === item.key ? 'page' : undefined}
            class="flex flex-col items-center gap-[3px] pb-1.5 pt-2.5 text-[11px] font-semibold transition-colors
              {screen === item.key ? 'text-mustard' : 'text-gray-500'}"
          >
            <item.icon size={22} />
            {$t(item.labelKey)}
          </button>
        {/each}
      </div>
    </nav>
  {/if}

  <ConfirmDialog
    bind:open={confirmReset}
    message={$t('comilao.reset_confirm')}
    confirmLabel={$t('comilao.reset_stats')}
    onConfirm={resetStats}
  />
</div>

<style>
  /* Lilita One não tem peso bold — evita o negrito sintético do navegador. */
  .comilao {
    font-synthesis: none;
  }
  .pad {
    display: flex;
    height: 52px;
    align-items: center;
    justify-content: center;
    border-radius: 14px;
    border: 1px solid rgba(255, 255, 255, 0.14);
    background: rgba(255, 255, 255, 0.06);
    color: rgba(255, 255, 255, 0.85);
  }
  .pad:active {
    background: theme('colors.mustard.DEFAULT');
    color: theme('colors.mustard.ink');
  }
</style>
