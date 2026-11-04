<script lang="ts">
  /**
   * Cobrinha — clássico em grade, no visual do games.thiagotn.com.
   *
   * Rota em tela cheia (`fixed inset-0`), sem Navbar, mobile-first. As abas
   * (Início · Diário · Visuais · Ajustes) ficam por baixo; a partida entra por
   * cima delas. As regras vivem em `$lib/cobrinha/engine.ts`, o desenho em
   * `render.ts` e a persistência em `storage.ts` — aqui só mora o laço do jogo,
   * a entrada do jogador e a interface.
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
    Pause,
    Play,
    Shirt,
    Settings as SettingsIcon,
    Target,
    Trophy,
  } from 'lucide-svelte';
  import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
  import LanguageSwitcher from '$lib/components/LanguageSwitcher.svelte';
  import Emblem from '$lib/cobrinha/Emblem.svelte';
  import {
    COLS,
    ROWS,
    dateKey,
    fmtScore,
    levelFor,
    newGame,
    reachedGoal,
    step,
    tickMs,
    turn,
    type Dir,
    type Game,
    type Point,
    type Speed,
  } from '$lib/cobrinha/engine';
  import { createRenderer, type Renderer } from '$lib/cobrinha/render';
  import { SKINS, nextLock, skinOf, unlockedSkins } from '$lib/cobrinha/skins';
  import {
    HISTORY_VISIBLE,
    addToStats,
    emptyState,
    loadState,
    recordDaily,
    recordMatch,
    saveState,
    streakOf,
    type SavedState,
  } from '$lib/cobrinha/storage';
  import { locale, t } from '$lib/i18n';

  type Screen = 'home' | 'game' | 'daily' | 'skins' | 'settings';

  type Result = {
    score: number;
    fruits: number;
    level: number;
    record: boolean;
    daily: string | null;
    goal: number;
    goalDone: boolean;
    /** Visuais que esta partida liberou. */
    unlocked: string[];
  };

  let screen = $state<Screen>('home');
  let saved = $state<SavedState>(emptyState());
  /** `raw`: o engine devolve um estado novo a cada passo, não há o que observar por dentro. */
  let game = $state.raw<Game | null>(null);
  let paused = $state(false);
  /** A cobra só anda depois do primeiro comando — ninguém perde antes de pegar no jogo. */
  let started = $state(false);
  let result = $state<Result | null>(null);
  let confirmReset = $state(false);
  let canvasFailed = $state(false);

  let canvasEl = $state<HTMLCanvasElement | null>(null);
  let boardEl = $state<HTMLDivElement | null>(null);

  let renderer: Renderer | null = null;
  let prevSnake: Point[] | null = null;
  let raf = 0;
  let acc = 0;
  let last = 0;
  let motion = true;

  const today = new Date();
  const todayKey = dateKey(today);

  let stats = $derived(saved.stats);
  let settings = $derived(saved.settings);
  let skin = $derived(skinOf(saved.settings.skin));
  let unlocked = $derived(unlockedSkins(saved.stats.total));
  let upcoming = $derived(nextLock(saved.stats.total));
  let level = $derived(game ? levelFor(game.eaten) : 1);
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

  // ── Partida ───────────────────────────────────────────────────────────────

  function startGame(daily: string | null) {
    game = newGame({ daily, wrap: settings.wrap, speed: settings.speed });
    prevSnake = null;
    paused = false;
    started = false;
    result = null;
    acc = 0;
    screen = 'game';
  }

  function quitGame() {
    game = null;
    result = null;
    paused = false;
    screen = 'home';
  }

  /** Fecha a partida: soma às estatísticas, grava o dia e monta a tela de fim. */
  function finish(g: Game) {
    const bestBefore = stats.best;
    const unlockedBefore = unlockedSkins(stats.total);
    const next: SavedState = {
      ...saved,
      stats: addToStats(stats, g),
      history: recordMatch(saved.history, g),
      daily: g.daily ? recordDaily(saved.daily, g.daily, g.score, g.goal) : saved.daily,
    };
    persist(next);
    result = {
      score: g.score,
      fruits: g.eaten,
      level: levelFor(g.eaten),
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
    const dt = Math.min(120, now - last);
    last = now;

    let g = game;
    if (!g || !renderer) return;

    if (!g.over && !paused && started) {
      const ms = tickMs(levelFor(g.eaten), g.speed);
      acc += dt;
      while (acc >= ms && !g.over) {
        acc -= ms;
        prevSnake = g.snake;
        const next = step(g);
        g = next.game;
        if (next.event === 'died') {
          acc = 0;
          finish(g);
        }
      }
      game = g;
    }

    renderer.draw(g, {
      prev: prevSnake,
      alpha: g.over || paused || !started ? 1 : acc / tickMs(levelFor(g.eaten), g.speed),
      skin,
      time: now,
      motion,
    });
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
      game = turn(game, dir);
      paused = false;
      started = true;
      return;
    }
    if (event.key === ' ' || event.key === 'p' || event.key === 'Escape') {
      event.preventDefault();
      if (!game.over) paused = !paused;
    }
  }

  /** Arraste em qualquer ponto do tabuleiro: 22px já contam como virada. */
  let swipeFrom: { x: number; y: number } | null = null;

  function onPointerDown(event: PointerEvent) {
    swipeFrom = { x: event.clientX, y: event.clientY };
    if (paused) paused = false;
    started = true;
  }

  function onPointerMove(event: PointerEvent) {
    if (!swipeFrom || !game || game.over) return;
    const dx = event.clientX - swipeFrom.x;
    const dy = event.clientY - swipeFrom.y;
    if (Math.hypot(dx, dy) < 22) return;
    const dir: Dir =
      Math.abs(dx) > Math.abs(dy) ? (dx > 0 ? 'right' : 'left') : dy > 0 ? 'down' : 'up';
    game = turn(game, dir);
    swipeFrom = { x: event.clientX, y: event.clientY };
  }

  function press(dir: Dir) {
    if (!game || game.over) return;
    game = turn(game, dir);
    paused = false;
    started = true;
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
    persist({ ...emptyState(), settings: { ...settings, skin: 'folha' } });
  }

  // ── Datas e rótulos ───────────────────────────────────────────────────────

  let weekdays = $derived($t('cobrinha.weekdays').split(','));

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
    { key: 'level', label: $t('cobrinha.hud_level'), value: String(level) },
    { key: 'fruits', label: $t('cobrinha.hud_fruits'), value: String(game?.eaten ?? 0) },
    { key: 'best', label: $t('cobrinha.hud_best'), value: fmtScore(stats.best) },
  ]);

  const SPEEDS: { key: Speed; labelKey: string }[] = [
    { key: 'calma', labelKey: 'cobrinha.speed_calma' },
    { key: 'normal', labelKey: 'cobrinha.speed_normal' },
    { key: 'veloz', labelKey: 'cobrinha.speed_veloz' },
  ];

  const TABS = [
    { key: 'home' as const, labelKey: 'cobrinha.tab_home', icon: House },
    { key: 'daily' as const, labelKey: 'cobrinha.tab_daily', icon: Calendar },
    { key: 'skins' as const, labelKey: 'cobrinha.tab_skins', icon: Shirt },
    { key: 'settings' as const, labelKey: 'cobrinha.tab_settings', icon: SettingsIcon },
  ];
</script>

<svelte:head>
  <title>{$t('cobrinha.title')} — games.thiagotn.com</title>
  <meta name="description" content={$t('cobrinha.desc')} />
  <meta name="theme-color" content="#05222a" />
  <link rel="manifest" href="/cobrinha/manifest.webmanifest" />
  <link rel="apple-touch-icon" href="/cobrinha/apple-touch-icon.png" />
  <meta name="apple-mobile-web-app-title" content={$t('cobrinha.title')} />
</svelte:head>

<svelte:window onkeydown={onKeydown} />

<div class="cobrinha fixed inset-0 z-50 flex flex-col overflow-hidden bg-ink text-white">
  {#if screen === 'game' && game}
    <!-- ── Partida ───────────────────────────────────────────────────── -->
    <header class="relative z-10 flex-none px-3 pb-2" style="padding-top:calc(env(safe-area-inset-top) + 8px)">
      <div class="flex items-center gap-2">
        <button
          type="button"
          onclick={quitGame}
          aria-label={$t('cobrinha.quit')}
          class="flex h-10 w-10 flex-none items-center justify-center rounded-xl border border-white/15 bg-white/5 text-white/80 transition-colors hover:bg-white/10"
        >
          <ChevronLeft size={20} strokeWidth={2.2} />
        </button>

        <div class="flex min-w-0 flex-1 flex-col items-center">
          <span class="text-[10px] font-bold uppercase tracking-[.18em] text-mustard">
            {game.daily ? $t('cobrinha.daily_kicker') : $t('cobrinha.hud_score')}
          </span>
          <span class="font-display text-[34px] leading-none tabular-nums">
            {fmtScore(game.score)}
          </span>
        </div>

        <button
          type="button"
          onclick={() => (paused = !paused)}
          disabled={game.over}
          aria-label={$t('cobrinha.pause')}
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

      {#if game.goal > 0}
        <div class="mt-2 flex items-center gap-2">
          <Target size={14} class="flex-none text-mustard" />
          <div class="h-2 flex-1 overflow-hidden rounded-full bg-white/10">
            <div
              class="h-full rounded-full bg-mustard transition-[width] duration-300"
              style="width:{Math.min(100, (game.score / game.goal) * 100)}%"
            ></div>
          </div>
          <span class="flex-none text-[11px] font-bold tabular-nums text-white/70">
            {game.score}/{game.goal}
          </span>
        </div>
      {/if}
    </header>

    <!-- Tabuleiro -->
    <div
      bind:this={boardEl}
      role="application"
      aria-label={$t('cobrinha.title')}
      class="board relative z-10 min-h-0 flex-1 touch-none select-none p-3"
      onpointerdown={onPointerDown}
      onpointermove={onPointerMove}
      onpointerup={() => (swipeFrom = null)}
      onpointercancel={() => (swipeFrom = null)}
    >
      <div class="flex h-full w-full items-center justify-center">
        {#if canvasFailed}
          <p class="max-w-xs text-center text-sm text-white/70">{$t('cobrinha.no_canvas')}</p>
        {:else}
          <div class="relative">
            <canvas
              bind:this={canvasEl}
              class="block rounded-2xl shadow-[0_18px_40px_rgba(0,0,0,.45)] ring-1 ring-white/10"
            ></canvas>
            {#if !started && !game.over}
              <span
                class="pointer-events-none absolute inset-x-0 bottom-5 mx-auto w-max max-w-[85%] rounded-full bg-black/65 px-3.5 py-1.5 text-center text-[12.5px] font-bold text-white/90"
              >
                {$t('cobrinha.tap_to_start')}
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
          {#each [{ dir: 'up' as const, icon: ArrowUp, cls: 'col-start-2 row-start-1', label: $t('cobrinha.dir_up') }, { dir: 'left' as const, icon: ArrowLeft, cls: 'col-start-1 row-start-2', label: $t('cobrinha.dir_left') }, { dir: 'right' as const, icon: ArrowRight, cls: 'col-start-3 row-start-2', label: $t('cobrinha.dir_right') }, { dir: 'down' as const, icon: ArrowDown, cls: 'col-start-2 row-start-3', label: $t('cobrinha.dir_down') }] as btn (btn.dir)}
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
    {#if paused && !game.over}
      <div class="overlay">
        <div class="card flex flex-col gap-3 text-center">
          <h2 class="font-display text-[30px] leading-none">{$t('cobrinha.paused')}</h2>
          <p class="text-sm font-medium text-paper-muted">{$t('cobrinha.controls_hint')}</p>
          <button class="btn btn-primary" onclick={() => (paused = false)}>
            {$t('cobrinha.resume')}
          </button>
          <button class="btn btn-ghost" onclick={quitGame}>{$t('cobrinha.quit')}</button>
        </div>
      </div>
    {/if}

    <!-- Fim de partida -->
    {#if result}
      <div class="overlay">
        <div class="card flex flex-col gap-3">
          <div class="text-center">
            <div class="text-[11px] font-bold uppercase tracking-[.2em] text-ketchup">
              {$t('cobrinha.over_kicker')}
            </div>
            <h2 class="font-display text-[34px] leading-none">{$t('cobrinha.over_title')}</h2>
          </div>

          {#if result.record}
            <p class="tag mx-auto bg-mustard text-mustard-ink">{$t('cobrinha.over_new_record')}</p>
          {/if}
          {#if result.daily}
            <p class="text-center text-sm font-bold {result.goalDone ? 'text-[#2f8f45]' : 'text-paper-muted'}">
              {result.goalDone
                ? $t('cobrinha.over_goal_done')
                : $t('cobrinha.over_goal_missed', { goal: result.goal })}
            </p>
          {/if}

          <!-- Jogo atual × De sempre -->
          <table class="w-full text-left">
            <thead>
              <tr class="stat-label">
                <th class="font-bold"></th>
                <th class="pb-1 text-right font-bold">{$t('cobrinha.col_current')}</th>
                <th class="pb-1 text-right font-bold">{$t('cobrinha.col_alltime')}</th>
              </tr>
            </thead>
            <tbody>
              {#each [{ key: 'score', label: $t('cobrinha.hud_score'), now: fmtScore(result.score), ever: fmtScore(stats.best) }, { key: 'fruits', label: $t('cobrinha.hud_fruits'), now: String(result.fruits), ever: String(stats.bestFruits) }, { key: 'level', label: $t('cobrinha.hud_level'), now: String(result.level), ever: String(stats.bestLevel) }] as row (row.key)}
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
              {$t('cobrinha.over_unlocked', {
                skin: result.unlocked.map((k) => $t(skinOf(k).nameKey)).join(', '),
              })}
            </p>
          {/if}

          <button class="btn btn-primary" onclick={() => startGame(result?.daily ?? null)}>
            {$t('cobrinha.play_again')}
          </button>
          <button class="btn btn-ghost" onclick={quitGame}>{$t('cobrinha.quit')}</button>
        </div>
      </div>
    {/if}
  {:else}
    <!-- ── Abas ──────────────────────────────────────────────────────── -->
    <div
      class="pointer-events-none absolute inset-x-0 top-0 h-[320px]"
      style="background:radial-gradient(90% 55% at 50% 0%,#0f5a66 0%,#05222a 45%,transparent 75%)"
    ></div>
    <div
      class="pointer-events-none absolute inset-0"
      style="background:linear-gradient(180deg,rgba(15,22,34,.35) 0%,rgba(15,22,34,.9) 45%,#0f1622 72%)"
    ></div>

    <div class="relative z-10 flex-none px-3 pb-1" style="padding-top:calc(env(safe-area-inset-top) + 8px)">
      <a
        href="/"
        class="inline-flex items-center gap-1.5 rounded-lg px-2 py-1.5 text-[13px] font-semibold text-white/75 transition-colors hover:bg-white/10 hover:text-white"
      >
        <ChevronLeft size={16} strokeWidth={2.2} /> {$t('cobrinha.back_to_games')}
      </a>
    </div>

    <div class="relative z-10 min-h-0 flex-1 overflow-y-auto">
      <div class="mx-auto max-w-md">
        {#if screen === 'home'}
          <div class="flex flex-col items-center px-[22px] pb-[26px] pt-[10px] text-center">
            <div class="drop-shadow-[0_16px_30px_rgba(142,222,79,.22)]">
              <Emblem size={132} />
            </div>
            <div
              class="mt-1 bg-[linear-gradient(180deg,#d6ff9a_0%,#8ede4f_52%,#2f8f45_100%)] bg-clip-text pt-1 font-display text-[44px] leading-none tracking-[.01em] text-transparent"
            >
              {$t('cobrinha.title')}
            </div>

            <!-- Desafio do dia -->
            <div
              class="relative mt-[18px] flex w-full items-center gap-3.5 overflow-hidden rounded-[14px] border border-mustard/35 bg-[linear-gradient(158deg,#0f5a66_0%,#0b3d48_45%,#05222a_100%)] py-3.5 pl-4 pr-3.5 text-left"
            >
              <div
                class="relative flex h-[46px] w-[46px] flex-none items-center justify-center rounded-xl border border-mustard/45 bg-black/25 text-mustard"
              >
                <Calendar size={24} />
              </div>
              <div class="relative flex min-w-0 flex-1 flex-col gap-px">
                <span class="text-[10.5px] font-bold uppercase tracking-[.16em] text-mustard">
                  {$t('cobrinha.daily_today')}
                </span>
                <span class="truncate font-display text-[19px] leading-tight text-white">
                  {todayLabel}
                </span>
                <span class="text-[12px] font-medium text-white/70">
                  {todayRecord?.done
                    ? $t('cobrinha.daily_status_done', { score: fmtScore(todayRecord.best) })
                    : $t('cobrinha.daily_status_todo')}
                </span>
              </div>
              <button
                type="button"
                onclick={() => startGame(todayKey)}
                class="relative h-10 flex-none rounded-[10px] bg-mustard px-4 font-display text-[15px] text-mustard-ink transition-colors hover:bg-[#ffd666] active:scale-95"
              >
                {todayRecord ? $t('cobrinha.replay') : $t('cobrinha.play')}
              </button>
            </div>

            <button
              type="button"
              onclick={() => startGame(null)}
              class="mt-3.5 flex h-[54px] w-full items-center justify-center gap-2.5 rounded-xl bg-mustard font-display text-[19px] text-mustard-ink shadow-[0_8px_20px_rgba(255,201,60,.3)] transition-colors hover:bg-[#ffd666] active:scale-[.98]"
            >
              <Play size={20} fill="currentColor" />
              {$t('cobrinha.new_game')}
            </button>

            <div class="mt-4 flex flex-wrap justify-center gap-x-[18px] gap-y-1 text-[12px] font-medium text-gray-400">
              <span>
                <b class="font-bold text-white">{fmtScore(stats.best)}</b>
                {$t('cobrinha.summary_best')}
              </span>
              <span aria-hidden="true">·</span>
              <span><b class="font-bold text-white">{stats.games}</b> {$t('cobrinha.summary_games')}</span>
              <span aria-hidden="true">·</span>
              <span>
                <b class="font-bold text-white">{streak}</b>
                {streak === 1 ? $t('cobrinha.summary_streak_one') : $t('cobrinha.summary_streak')}
              </span>
            </div>

            {#if history.length}
              <div class="mt-6 w-full text-left">
                <div class="section-label !text-white/45">{$t('cobrinha.recent')}</div>
                <ul class="flex flex-col gap-1.5">
                  {#each history as entry (entry.at)}
                    <li
                      class="flex items-center gap-3 rounded-xl border border-white/10 bg-white/[.04] px-3 py-2"
                    >
                      <span class="flex h-7 w-7 flex-none items-center justify-center rounded-lg bg-mustard/15 text-mustard">
                        {#if entry.daily}<Calendar size={14} />{:else}<Trophy size={14} />{/if}
                      </span>
                      <span class="min-w-0 flex-1 truncate text-[13px] font-semibold text-white/80">
                        {entry.daily
                          ? $t('cobrinha.entry_daily')
                          : $t('cobrinha.entry_free')} · {$t('cobrinha.entry_fruits', {
                          n: entry.fruits,
                        })}
                      </span>
                      <span class="font-display text-[18px] tabular-nums">{fmtScore(entry.score)}</span>
                    </li>
                  {/each}
                </ul>
              </div>
            {/if}
          </div>
        {:else if screen === 'daily'}
          <div class="px-5 pb-[26px] pt-5">
            <div class="text-[11px] font-bold uppercase tracking-[.2em] text-mustard">
              {$t('cobrinha.daily_kicker')}
            </div>
            <div class="mt-0.5 font-display text-[30px] leading-tight">{monthLabel}</div>
            <p class="mt-1.5 text-[13.5px] leading-[1.45] text-white/70">
              {$t('cobrinha.daily_note')}
            </p>

            <div class="mt-4 grid grid-cols-3 gap-2.5">
              {#each [{ key: 'done', label: $t('cobrinha.stat_done'), value: String(doneDays), gold: false }, { key: 'streak', label: $t('cobrinha.stat_streak'), value: String(streak), gold: true }, { key: 'best', label: $t('cobrinha.stat_daily_best'), value: dailyBest ? fmtScore(dailyBest) : '—', gold: false }] as box (box.key)}
                <div
                  class="rounded-[14px] border px-2 py-3 text-center {box.gold
                    ? 'border-mustard/45 bg-mustard/5'
                    : 'border-gray-700 bg-gray-800/70'}"
                >
                  <div class="font-display text-[26px] leading-none {box.gold ? 'text-mustard' : ''}">
                    {box.value}
                  </div>
                  <div class="mt-1 text-[10px] font-bold uppercase tracking-[.12em] text-gray-400">
                    {box.label}
                  </div>
                </div>
              {/each}
            </div>

            <div class="mt-3.5 rounded-[14px] border border-gray-700 bg-gray-800/85 px-2.5 py-3">
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
                          ? 'border border-[#2f8f45]'
                          : cell.future
                            ? 'border border-transparent'
                            : 'border border-gray-700'}
                        {cell.done
                        ? 'bg-[#1b5e3a] font-bold text-white'
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
              {$t('cobrinha.daily_play')}
            </button>
          </div>
        {:else if screen === 'skins'}
          <div class="px-5 pb-[26px] pt-5">
            <div class="text-[11px] font-bold uppercase tracking-[.2em] text-mustard">
              {$t('cobrinha.skins_kicker')}
            </div>
            <div class="mt-0.5 font-display text-[30px] leading-tight">{$t('cobrinha.skins_title')}</div>

            {#if upcoming}
              <div class="mt-3 rounded-[14px] border border-gray-700 bg-gray-800/70 px-3.5 py-3">
                <div class="flex items-baseline justify-between gap-2">
                  <span class="text-[13px] font-bold text-white/85">
                    {$t('cobrinha.skins_next', { skin: $t(upcoming.nameKey) })}
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
              <p class="mt-3 text-[13.5px] text-white/70">{$t('cobrinha.skins_all')}</p>
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
                      ? 'border-gray-700 bg-gray-800/70 hover:bg-gray-800'
                      : 'border-gray-800 bg-gray-900/60'}"
                >
                  <div class="flex items-center gap-[3px] {open ? '' : 'opacity-35'}">
                    {#each [0, 1, 2, 3, 4] as i (i)}
                      <span
                        class="h-[18px] w-[18px] rounded-[5px]"
                        style="background:{i === 0 ? item.head : item.body(i, 5)}"
                      ></span>
                    {/each}
                  </div>
                  <span class="flex items-center gap-1.5 text-[13px] font-bold">
                    {#if !open}<Lock size={12} class="text-gray-500" />{/if}
                    {$t(item.nameKey)}
                  </span>
                  <span class="text-[11px] font-semibold text-gray-400">
                    {#if settings.skin === item.key}
                      {$t('cobrinha.skins_in_use')}
                    {:else if open}
                      {$t('cobrinha.skins_use')}
                    {:else}
                      {$t('cobrinha.skins_need', { n: fmtScore(item.need) })}
                    {/if}
                  </span>
                </button>
              {/each}
            </div>
          </div>
        {:else if screen === 'settings'}
          <div class="px-5 pb-[26px] pt-5">
            <div class="text-[11px] font-bold uppercase tracking-[.2em] text-mustard">
              {$t('cobrinha.tab_settings')}
            </div>
            <div class="mt-0.5 font-display text-[30px] leading-tight">{$t('cobrinha.settings_title')}</div>

            <div class="mt-4 divide-y divide-gray-700 overflow-hidden rounded-[14px] border border-gray-700 bg-gray-800/70">
              <div class="flex items-center justify-between gap-3 px-3.5 py-3">
                <div class="min-w-0">
                  <div class="text-[15px] font-bold">{$t('cobrinha.setting_speed')}</div>
                  <div class="text-[12.5px] text-white/60">{$t('cobrinha.setting_speed_hint')}</div>
                </div>
                <div class="flex flex-none rounded-[10px] bg-black/35 p-0.5">
                  {#each SPEEDS as option (option.key)}
                    <button
                      type="button"
                      onclick={() => update('speed', option.key)}
                      aria-pressed={settings.speed === option.key}
                      class="rounded-lg px-2.5 py-1.5 text-[12px] font-bold transition-colors
                        {settings.speed === option.key ? 'bg-mustard text-mustard-ink' : 'text-white/70'}"
                    >
                      {$t(option.labelKey)}
                    </button>
                  {/each}
                </div>
              </div>

              {#each [{ key: 'wrap' as const, label: $t('cobrinha.setting_wrap'), hint: $t('cobrinha.setting_wrap_hint') }, { key: 'pad' as const, label: $t('cobrinha.setting_pad'), hint: $t('cobrinha.setting_pad_hint') }] as item (item.key)}
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
              <span class="text-[15px] font-bold">{$t('cobrinha.language')}</span>
              <LanguageSwitcher tone="dark" />
            </div>

            <button
              type="button"
              onclick={() => (confirmReset = true)}
              class="mt-[22px] h-[46px] w-full rounded-xl border border-red-400/40 text-[14px] font-semibold text-red-300 transition-colors hover:bg-red-900/25"
            >
              {$t('cobrinha.reset_stats')}
            </button>
          </div>
        {/if}
      </div>
    </div>

    <!-- Abas inferiores -->
    <nav
      class="relative z-10 flex-none border-t border-gray-800 bg-[#0b111c]"
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
    message={$t('cobrinha.reset_confirm')}
    confirmLabel={$t('cobrinha.reset_stats')}
    onConfirm={resetStats}
  />
</div>

<style>
  /* Lilita One não tem peso bold — evita o negrito sintético do navegador. */
  .cobrinha {
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
