<script lang="ts">
  /**
   * Astronave — nave contra meteoros e inimigos, no visual do games.thiagotn.com.
   *
   * Rota em tela cheia (`fixed inset-0`), sem Navbar, mobile-first. As abas
   * (Início · Diário · Naves · Ajustes) ficam por baixo; a partida entra por
   * cima delas. As regras vivem em `$lib/astronave/engine.ts`, o desenho em
   * `render.ts` e a persistência em `storage.ts` — aqui só mora o laço do jogo,
   * a entrada do jogador e a interface.
   *
   * Dois jeitos de jogar, escolhidos nos Ajustes (ou na pausa):
   *  - `touch`: arrastar em qualquer ponto move a nave e segurar o dedo atira;
   *    sem botões, a arena ocupa a tela toda;
   *  - `pad`: botões de mover e de atirar embaixo da arena.
   * O teclado funciona nos dois.
   */
  import {
    ArrowLeft,
    ArrowRight,
    Calendar,
    ChevronLeft,
    ChevronsUp,
    Crosshair,
    Heart,
    House,
    Lock,
    Pause,
    Play,
    Rocket,
    Settings as SettingsIcon,
    Shield,
    Target,
    Trophy,
  } from 'lucide-svelte';
  import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
  import LanguageSwitcher from '$lib/components/LanguageSwitcher.svelte';
  import Emblem from '$lib/astronave/Emblem.svelte';
  import Ship from '$lib/astronave/Ship.svelte';
  import {
    H,
    MAX_LIVES,
    STEP,
    W,
    dateKey,
    fmtScore,
    newGame,
    reachedGoal,
    tick,
    type Game,
    type GameEvent,
    type Input,
  } from '$lib/astronave/engine';
  import { createRenderer, type Renderer } from '$lib/astronave/render';
  import { DEFAULT_SKIN, SKINS, nextLock, skinOf, unlockedSkins } from '$lib/astronave/skins';
  import {
    HISTORY_VISIBLE,
    addToStats,
    emptyState,
    loadState,
    recordDaily,
    recordMatch,
    saveState,
    streakOf,
    type Controls,
    type SavedState,
  } from '$lib/astronave/storage';
  import { locale, t } from '$lib/i18n';

  type Screen = 'home' | 'game' | 'daily' | 'skins' | 'settings';

  type Result = {
    score: number;
    wave: number;
    kills: number;
    record: boolean;
    daily: string | null;
    goal: number;
    goalDone: boolean;
    /** Naves que esta partida liberou. */
    unlocked: string[];
  };

  let screen = $state<Screen>('home');
  let saved = $state<SavedState>(emptyState());
  let paused = $state(false);
  /** Nada se mexe antes do primeiro comando — ninguém perde antes de pegar no jogo. */
  let started = $state(false);
  let result = $state<Result | null>(null);
  let confirmReset = $state(false);
  let canvasFailed = $state(false);
  /** Número da onda que acabou de começar, enquanto o aviso está na tela. */
  let banner = $state<number | null>(null);

  /**
   * O engine muda o estado no lugar, 60 vezes por segundo: a partida fica fora
   * da reatividade e só este resumo, copiado a cada quadro, chega ao template.
   */
  let hud = $state({
    score: 0,
    lives: MAX_LIVES,
    wave: 1,
    shield: false,
    double: 0,
    over: false,
    daily: null as string | null,
    goal: 0,
  });

  let canvasEl = $state<HTMLCanvasElement | null>(null);
  let boardEl = $state<HTMLDivElement | null>(null);

  let game: Game | null = null;
  let renderer: Renderer | null = null;
  let raf = 0;
  let acc = 0;
  let last = 0;
  let motion = true;
  /** Px de CSS por unidade do campo — converte o arraste do dedo em movimento da nave. */
  let scale = 3;
  let bannerTimer: ReturnType<typeof setTimeout> | undefined;
  let resultTimer: ReturnType<typeof setTimeout> | undefined;

  const today = new Date();
  const todayKey = dateKey(today);

  let stats = $derived(saved.stats);
  let settings = $derived(saved.settings);
  let skin = $derived(skinOf(saved.settings.skin));
  let unlocked = $derived(unlockedSkins(saved.stats.total));
  let upcoming = $derived(nextLock(saved.stats.total));
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
    clearTimeout(resultTimer);
    game = newGame({ daily, seed: Math.floor(Math.random() * 0xffffffff) });
    releaseAll();
    paused = false;
    started = false;
    result = null;
    banner = null;
    acc = 0;
    syncHud(game);
    screen = 'game';
  }

  function quitGame() {
    clearTimeout(resultTimer);
    clearTimeout(bannerTimer);
    game = null;
    result = null;
    paused = false;
    banner = null;
    releaseAll();
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
    const summary: Result = {
      score: g.score,
      wave: g.wave,
      kills: g.kills,
      record: g.score > bestBefore && g.score > 0,
      daily: g.daily,
      goal: g.goal,
      goalDone: reachedGoal(g),
      unlocked: unlockedSkins(next.stats.total).filter((k) => !unlockedBefore.includes(k)),
    };
    // Um instante para a explosão aparecer antes de o placar cobrir a arena.
    resultTimer = setTimeout(() => (result = summary), motion ? 700 : 0);
  }

  function syncHud(g: Game) {
    hud.score = g.score;
    hud.lives = g.lives;
    hud.wave = g.wave;
    hud.shield = g.ship.shield;
    hud.double = Math.ceil(g.ship.doubleLeft);
    hud.over = g.over;
    hud.daily = g.daily;
    hud.goal = g.goal;
  }

  function onEvents(events: GameEvent[]) {
    for (const e of events) {
      if (e.type === 'wave') {
        banner = e.wave;
        clearTimeout(bannerTimer);
        bannerTimer = setTimeout(() => (banner = null), 1500);
      } else if ((e.type === 'hit' && !e.shielded) || e.type === 'died') {
        navigator.vibrate?.(e.type === 'died' ? 180 : 60);
      }
    }
  }

  // ── Laço do jogo ──────────────────────────────────────────────────────────

  function fit() {
    if (!boardEl || !renderer) return;
    const box = boardEl.getBoundingClientRect();
    scale = Math.max(1, Math.min((box.width - 12) / W, (box.height - 6) / H));
    renderer.resize(scale);
  }

  function frame(now: number) {
    raf = requestAnimationFrame(frame);
    const dt = Math.min(0.1, (now - last) / 1000);
    last = now;

    const g = game;
    if (!g || !renderer) return;

    if (!g.over && !paused && started) {
      // Passo fixo: a física não depende da taxa de quadros do aparelho.
      acc += dt;
      let steps = 0;
      while (acc >= STEP && steps < 4 && !g.over) {
        acc -= STEP;
        steps++;
        const events = tick(g, input);
        if (events.length) {
          renderer.fx(events);
          onEvents(events);
        }
      }
      if (steps === 4) acc = 0;
      if (g.over) finish(g);
    }

    syncHud(g);
    if (boardEl) {
      // Para o E2E: o que acontece dentro do canvas não é visível no DOM.
      const data = boardEl.dataset;
      data.shipX = String(Math.round(g.ship.x));
      data.fired = String(g.fired);
    }
    renderer.draw(g, { skin, time: now, motion });
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

  // Sair do app no meio da partida pausa: voltar com a nave já destruída não tem graça.
  $effect(() => {
    const onHide = () => {
      if (document.hidden && screen === 'game' && started && game && !game.over) {
        paused = true;
        releaseAll();
      }
    };
    document.addEventListener('visibilitychange', onHide);
    return () => document.removeEventListener('visibilitychange', onHide);
  });

  // ── Entrada ───────────────────────────────────────────────────────────────

  /** O que o engine lê a cada passo; montado a partir do que está apertado. */
  const input: Input = { axis: 0, targetX: null, fire: false };
  const held = { keyLeft: false, keyRight: false, keyFire: false, padLeft: false, padRight: false, padFire: false, touch: false };
  let drag: { id: number; startX: number; shipX: number } | null = null;

  function refreshInput() {
    const left = held.keyLeft || held.padLeft;
    const right = held.keyRight || held.padRight;
    input.axis = (right ? 1 : 0) - (left ? 1 : 0);
    input.fire = held.keyFire || held.padFire || held.touch;
  }

  function releaseAll() {
    for (const key of Object.keys(held) as (keyof typeof held)[]) held[key] = false;
    drag = null;
    input.targetX = null;
    refreshInput();
  }

  const KEYS: Record<string, 'keyLeft' | 'keyRight' | 'keyFire'> = {
    ArrowLeft: 'keyLeft',
    a: 'keyLeft',
    ArrowRight: 'keyRight',
    d: 'keyRight',
    ' ': 'keyFire',
    ArrowUp: 'keyFire',
    w: 'keyFire',
  };
  const keyOf = (event: KeyboardEvent) => KEYS[event.key] ?? KEYS[event.key.toLowerCase()];

  function onKeydown(event: KeyboardEvent) {
    if (screen !== 'game' || !game || game.over) return;
    const key = keyOf(event);
    if (key) {
      event.preventDefault();
      if (paused) return;
      held[key] = true;
      started = true;
      refreshInput();
      return;
    }
    if (event.key.toLowerCase() === 'p' || event.key === 'Escape') {
      event.preventDefault();
      paused = !paused;
      if (paused) releaseAll();
    }
  }

  function onKeyup(event: KeyboardEvent) {
    const key = keyOf(event);
    if (!key) return;
    held[key] = false;
    refreshInput();
  }

  /**
   * Arraste relativo: a nave anda o mesmo tanto que o dedo, a partir de onde
   * ela estava — o dedo pode ficar em qualquer lugar da arena sem tapar a nave.
   * Enquanto o dedo está na tela, ela atira.
   */
  function onPointerDown(event: PointerEvent) {
    if (!game || game.over || paused || drag) return;
    drag = { id: event.pointerId, startX: event.clientX, shipX: game.ship.x };
    input.targetX = game.ship.x;
    held.touch = true;
    started = true;
    refreshInput();
    boardEl?.setPointerCapture?.(event.pointerId);
  }

  function onPointerMove(event: PointerEvent) {
    if (!drag || drag.id !== event.pointerId) return;
    input.targetX = drag.shipX + (event.clientX - drag.startX) / scale;
  }

  function onPointerUp(event: PointerEvent) {
    if (!drag || drag.id !== event.pointerId) return;
    drag = null;
    input.targetX = null;
    held.touch = false;
    refreshInput();
  }

  /** Botões do modo `pad`: valem enquanto o dedo está em cima. */
  function padDown(key: 'padLeft' | 'padRight' | 'padFire', event: PointerEvent) {
    event.preventDefault();
    if (!game || game.over || paused) return;
    (event.currentTarget as HTMLElement).setPointerCapture?.(event.pointerId);
    held[key] = true;
    started = true;
    refreshInput();
  }

  function padUp(key: 'padLeft' | 'padRight' | 'padFire') {
    held[key] = false;
    refreshInput();
  }

  // ── Ajustes e naves ───────────────────────────────────────────────────────

  function update<K extends keyof SavedState['settings']>(key: K, value: SavedState['settings'][K]) {
    persist({ ...saved, settings: { ...settings, [key]: value } });
  }

  function chooseSkin(key: string) {
    if (!unlocked.includes(key as (typeof unlocked)[number])) return;
    update('skin', key as SavedState['settings']['skin']);
  }

  function resetStats() {
    persist({ ...emptyState(), settings: { ...settings, skin: DEFAULT_SKIN } });
  }

  // ── Datas e rótulos ───────────────────────────────────────────────────────

  let weekdays = $derived($t('astronave.weekdays').split(','));

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

  const CONTROLS: { key: Controls; labelKey: string; hintKey: string }[] = [
    { key: 'touch', labelKey: 'astronave.controls_touch', hintKey: 'astronave.controls_touch_hint' },
    { key: 'pad', labelKey: 'astronave.controls_pad', hintKey: 'astronave.controls_pad_hint' },
  ];

  const PAD = [
    { key: 'padLeft' as const, labelKey: 'astronave.btn_left', icon: ArrowLeft },
    { key: 'padRight' as const, labelKey: 'astronave.btn_right', icon: ArrowRight },
  ];

  const TABS = [
    { key: 'home' as const, labelKey: 'astronave.tab_home', icon: House },
    { key: 'daily' as const, labelKey: 'astronave.tab_daily', icon: Calendar },
    { key: 'skins' as const, labelKey: 'astronave.tab_skins', icon: Rocket },
    { key: 'settings' as const, labelKey: 'astronave.tab_settings', icon: SettingsIcon },
  ];
</script>

<svelte:head>
  <title>{$t('astronave.title')} — games.thiagotn.com</title>
  <meta name="description" content={$t('astronave.desc')} />
  <meta name="theme-color" content="#070b1d" />
  <link rel="manifest" href="/astronave/manifest.webmanifest" />
  <link rel="apple-touch-icon" href="/astronave/apple-touch-icon.png" />
  <meta name="apple-mobile-web-app-title" content={$t('astronave.title')} />
</svelte:head>

<svelte:window onkeydown={onKeydown} onkeyup={onKeyup} onblur={releaseAll} />

<!-- Escolha dos controles: aparece nos Ajustes (fundo escuro) e na pausa (card claro). -->
{#snippet controlsPicker(light = false)}
  <div class="flex items-center justify-between gap-3">
    <span class="text-[15px] font-bold">{$t('astronave.setting_controls')}</span>
    <div class="flex flex-none rounded-[10px] p-0.5 {light ? 'bg-paper-ink/10' : 'bg-black/35'}">
      {#each CONTROLS as option (option.key)}
        <button
          type="button"
          onclick={() => update('controls', option.key)}
          aria-pressed={settings.controls === option.key}
          class="rounded-lg px-3 py-1.5 text-[12.5px] font-bold transition-colors
            {settings.controls === option.key
            ? 'bg-mustard text-mustard-ink'
            : light
              ? 'text-paper-muted'
              : 'text-white/70'}"
        >
          {$t(option.labelKey)}
        </button>
      {/each}
    </div>
  </div>
  <p class="mt-2 text-left text-[12.5px] leading-snug {light ? 'text-paper-muted' : 'text-white/60'}">
    {$t(CONTROLS.find((c) => c.key === settings.controls)?.hintKey ?? CONTROLS[0].hintKey)}
  </p>
{/snippet}

<div class="astronave fixed inset-0 z-50 flex flex-col overflow-hidden bg-ink text-white">
  {#if screen === 'game'}
    <!-- ── Partida ───────────────────────────────────────────────────── -->
    <div class="pointer-events-none absolute inset-0 bg-cosmos"></div>

    <header class="relative z-10 flex-none px-3 pb-1.5" style="padding-top:calc(env(safe-area-inset-top) + 6px)">
      <div class="flex items-center gap-2">
        <button
          type="button"
          onclick={quitGame}
          aria-label={$t('astronave.quit')}
          class="flex h-10 w-10 flex-none items-center justify-center rounded-xl border border-white/15 bg-white/5 text-white/80 transition-colors hover:bg-white/10"
        >
          <ChevronLeft size={20} strokeWidth={2.2} />
        </button>

        <div class="flex min-w-0 flex-1 flex-col items-center">
          <span class="text-[10px] font-bold uppercase tracking-[.18em] text-mustard">
            {hud.daily ? $t('astronave.daily_kicker') : $t('astronave.hud_score')}
          </span>
          <span class="font-display text-[30px] leading-none tabular-nums" data-testid="score">
            {fmtScore(hud.score)}
          </span>
        </div>

        <button
          type="button"
          onclick={() => {
            paused = !paused;
            if (paused) releaseAll();
          }}
          disabled={hud.over}
          aria-label={$t('astronave.pause')}
          class="flex h-10 w-10 flex-none items-center justify-center rounded-xl border border-white/15 bg-white/5 text-white/80 transition-colors hover:bg-white/10 disabled:opacity-40"
        >
          {#if paused}<Play size={18} fill="currentColor" />{:else}<Pause size={18} />{/if}
        </button>
      </div>

      <!-- Vidas · onda · prêmios ativos · recorde, numa linha só para sobrar arena -->
      <div class="mt-1.5 flex items-center justify-between gap-3 px-1 text-[12px] font-bold text-white/75">
        <span
          class="flex items-center gap-1"
          role="img"
          aria-label={$t('astronave.hud_lives', { n: hud.lives })}
          data-testid="lives"
          data-lives={hud.lives}
        >
          {#each Array(MAX_LIVES) as _, i (i)}
            <Heart
              size={15}
              class={i < hud.lives ? 'text-[#ff7aa8]' : 'text-white/20'}
              fill={i < hud.lives ? 'currentColor' : 'none'}
            />
          {/each}
        </span>
        <span class="flex items-center gap-2">
          {#if hud.shield}<Shield size={15} class="text-[#5ee1ff]" />{/if}
          {#if hud.double > 0}
            <span class="flex items-center gap-0.5 tabular-nums text-mustard">
              <ChevronsUp size={15} />{hud.double}s
            </span>
          {/if}
          <span class="uppercase tracking-[.1em]">{$t('astronave.wave_n', { n: hud.wave })}</span>
        </span>
        <span class="tabular-nums">
          <span class="font-semibold uppercase tracking-[.1em] text-white/45">{$t('astronave.hud_best')}</span>
          {fmtScore(Math.max(stats.best, hud.score))}
        </span>
      </div>

      {#if hud.goal > 0}
        <div class="mt-1.5 flex items-center gap-2">
          <Target size={14} class="flex-none text-mustard" />
          <div class="h-2 flex-1 overflow-hidden rounded-full bg-white/10">
            <div
              class="h-full rounded-full bg-mustard transition-[width] duration-300"
              style="width:{Math.min(100, (hud.score / hud.goal) * 100)}%"
            ></div>
          </div>
          <span class="flex-none text-[11px] font-bold tabular-nums text-white/70">
            {hud.score}/{hud.goal}
          </span>
        </div>
      {/if}
    </header>

    <!-- Arena -->
    <div
      bind:this={boardEl}
      role="application"
      aria-label={$t('astronave.title')}
      data-board
      data-controls={settings.controls}
      data-started={started}
      class="relative z-10 min-h-0 flex-1 touch-none select-none"
      onpointerdown={onPointerDown}
      onpointermove={onPointerMove}
      onpointerup={onPointerUp}
      onpointercancel={onPointerUp}
    >
      <div class="flex h-full w-full items-center justify-center">
        {#if canvasFailed}
          <p class="max-w-xs text-center text-sm text-white/70">{$t('astronave.no_canvas')}</p>
        {:else}
          <div class="relative">
            <canvas
              bind:this={canvasEl}
              class="block rounded-2xl shadow-[0_18px_40px_rgba(0,0,0,.45)] ring-1 ring-white/10"
            ></canvas>
            {#if banner && !hud.over}
              <span
                class="pointer-events-none absolute inset-x-0 top-[22%] text-center font-display text-[34px] leading-none text-white drop-shadow-[0_2px_10px_rgba(79,125,255,.9)]"
              >
                {$t('astronave.wave_n', { n: banner })}
              </span>
            {/if}
            {#if !started && !hud.over}
              <span
                class="pointer-events-none absolute inset-x-0 bottom-[14%] mx-auto w-max max-w-[88%] rounded-full bg-black/65 px-3.5 py-1.5 text-center text-[12.5px] font-bold text-white/90"
              >
                {settings.controls === 'pad' ? $t('astronave.start_pad') : $t('astronave.start_touch')}
              </span>
            {/if}
          </div>
        {/if}
      </div>
    </div>

    <!-- Botões (só no modo `pad`) -->
    {#if settings.controls === 'pad'}
      <div
        class="relative z-10 flex flex-none touch-none select-none items-center justify-between gap-3 px-4 pt-2"
        style="padding-bottom:calc(env(safe-area-inset-bottom) + 10px)"
        data-pad
      >
        <div class="flex gap-2.5">
          {#each PAD as btn (btn.key)}
            <button
              type="button"
              class="pad h-[64px] w-[72px]"
              aria-label={$t(btn.labelKey)}
              onpointerdown={(e) => padDown(btn.key, e)}
              onpointerup={() => padUp(btn.key)}
              onpointercancel={() => padUp(btn.key)}
              onlostpointercapture={() => padUp(btn.key)}
              oncontextmenu={(e) => e.preventDefault()}
            >
              <btn.icon size={26} strokeWidth={2.4} />
            </button>
          {/each}
        </div>
        <button
          type="button"
          class="pad pad-fire h-[64px] min-w-[118px] flex-1 gap-2 font-display text-[18px]"
          aria-label={$t('astronave.btn_fire')}
          onpointerdown={(e) => padDown('padFire', e)}
          onpointerup={() => padUp('padFire')}
          onpointercancel={() => padUp('padFire')}
          onlostpointercapture={() => padUp('padFire')}
          oncontextmenu={(e) => e.preventDefault()}
        >
          <Crosshair size={22} strokeWidth={2.4} />
          {$t('astronave.btn_fire')}
        </button>
      </div>
    {:else}
      <div class="flex-none" style="height:calc(env(safe-area-inset-bottom) + 4px)"></div>
    {/if}

    <!-- Pausa -->
    {#if paused && !hud.over}
      <div class="overlay">
        <div class="card flex flex-col gap-3 text-center">
          <h2 class="font-display text-[30px] leading-none">{$t('astronave.paused')}</h2>
          <div class="rounded-xl border border-paper-ink/10 px-3 py-2.5">
            {@render controlsPicker(true)}
          </div>
          <p class="text-[12.5px] font-medium text-paper-muted">{$t('astronave.hint_keys')}</p>
          <button class="btn btn-primary" onclick={() => (paused = false)}>
            {$t('astronave.resume')}
          </button>
          <button class="btn btn-ghost" onclick={quitGame}>{$t('astronave.quit')}</button>
        </div>
      </div>
    {/if}

    <!-- Fim de partida -->
    {#if result}
      <div class="overlay">
        <div class="card flex flex-col gap-3">
          <div class="text-center">
            <div class="text-[11px] font-bold uppercase tracking-[.2em] text-ketchup">
              {$t('astronave.over_kicker')}
            </div>
            <h2 class="font-display text-[34px] leading-none">{$t('astronave.over_title')}</h2>
          </div>

          {#if result.record}
            <p class="tag mx-auto bg-mustard text-mustard-ink">{$t('astronave.over_new_record')}</p>
          {/if}
          {#if result.daily}
            <p class="text-center text-sm font-bold {result.goalDone ? 'text-[#2f8f45]' : 'text-paper-muted'}">
              {result.goalDone
                ? $t('astronave.over_goal_done')
                : $t('astronave.over_goal_missed', { goal: result.goal })}
            </p>
          {/if}

          <!-- Jogo atual × De sempre -->
          <table class="w-full text-left">
            <thead>
              <tr class="stat-label">
                <th class="font-bold"></th>
                <th class="pb-1 text-right font-bold">{$t('astronave.col_current')}</th>
                <th class="pb-1 text-right font-bold">{$t('astronave.col_alltime')}</th>
              </tr>
            </thead>
            <tbody>
              {#each [{ key: 'score', label: $t('astronave.hud_score'), now: fmtScore(result.score), ever: fmtScore(stats.best) }, { key: 'wave', label: $t('astronave.hud_wave'), now: String(result.wave), ever: String(stats.bestWave) }, { key: 'kills', label: $t('astronave.hud_kills'), now: String(result.kills), ever: String(stats.bestKills) }] as row (row.key)}
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
              {$t('astronave.over_unlocked', {
                skin: result.unlocked.map((k) => $t(skinOf(k).nameKey)).join(', '),
              })}
            </p>
          {/if}

          <button class="btn btn-primary" onclick={() => startGame(result?.daily ?? null)}>
            {$t('astronave.play_again')}
          </button>
          <button class="btn btn-ghost" onclick={quitGame}>{$t('astronave.quit')}</button>
        </div>
      </div>
    {/if}
  {:else}
    <!-- ── Abas ──────────────────────────────────────────────────────── -->
    <div
      class="pointer-events-none absolute inset-x-0 top-0 h-[320px]"
      style="background:radial-gradient(90% 55% at 50% 0%,#24337a 0%,#070b1d 45%,transparent 75%)"
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
        <ChevronLeft size={16} strokeWidth={2.2} /> {$t('astronave.back_to_games')}
      </a>
    </div>

    <div class="relative z-10 min-h-0 flex-1 overflow-y-auto">
      <div class="mx-auto max-w-md">
        {#if screen === 'home'}
          <div class="flex flex-col items-center px-[22px] pb-[26px] pt-[10px] text-center">
            <div class="drop-shadow-[0_16px_30px_rgba(79,125,255,.35)]">
              <Emblem size={132} />
            </div>
            <div
              class="mt-1 bg-[linear-gradient(180deg,#ffffff_0%,#8fe9ff_52%,#4f7dff_100%)] bg-clip-text pt-1 font-display text-[44px] leading-none tracking-[.01em] text-transparent"
            >
              {$t('astronave.title')}
            </div>

            <!-- Desafio do dia -->
            <div
              class="relative mt-[18px] flex w-full items-center gap-3.5 overflow-hidden rounded-[14px] border border-mustard/35 bg-[linear-gradient(158deg,#24337a_0%,#111a3d_45%,#070b1d_100%)] py-3.5 pl-4 pr-3.5 text-left"
            >
              <div
                class="relative flex h-[46px] w-[46px] flex-none items-center justify-center rounded-xl border border-mustard/45 bg-black/25 text-mustard"
              >
                <Calendar size={24} />
              </div>
              <div class="relative flex min-w-0 flex-1 flex-col gap-px">
                <span class="text-[10.5px] font-bold uppercase tracking-[.16em] text-mustard">
                  {$t('astronave.daily_today')}
                </span>
                <span class="truncate font-display text-[19px] leading-tight text-white">
                  {todayLabel}
                </span>
                <span class="text-[12px] font-medium text-white/70">
                  {todayRecord?.done
                    ? $t('astronave.daily_status_done', { score: fmtScore(todayRecord.best) })
                    : $t('astronave.daily_status_todo')}
                </span>
              </div>
              <button
                type="button"
                onclick={() => startGame(todayKey)}
                class="relative h-10 flex-none rounded-[10px] bg-mustard px-4 font-display text-[15px] text-mustard-ink transition-colors hover:bg-[#ffd666] active:scale-95"
              >
                {todayRecord ? $t('astronave.replay') : $t('astronave.play')}
              </button>
            </div>

            <button
              type="button"
              onclick={() => startGame(null)}
              class="mt-3.5 flex h-[54px] w-full items-center justify-center gap-2.5 rounded-xl bg-mustard font-display text-[19px] text-mustard-ink shadow-[0_8px_20px_rgba(255,201,60,.3)] transition-colors hover:bg-[#ffd666] active:scale-[.98]"
            >
              <Play size={20} fill="currentColor" />
              {$t('astronave.new_game')}
            </button>

            <div class="mt-4 flex flex-wrap justify-center gap-x-[18px] gap-y-1 text-[12px] font-medium text-gray-400">
              <span>
                <b class="font-bold text-white">{fmtScore(stats.best)}</b>
                {$t('astronave.summary_best')}
              </span>
              <span aria-hidden="true">·</span>
              <span><b class="font-bold text-white">{stats.games}</b> {$t('astronave.summary_games')}</span>
              <span aria-hidden="true">·</span>
              <span>
                <b class="font-bold text-white">{streak}</b>
                {streak === 1 ? $t('astronave.summary_streak_one') : $t('astronave.summary_streak')}
              </span>
            </div>

            {#if history.length}
              <div class="mt-6 w-full text-left">
                <div class="section-label !text-white/45">{$t('astronave.recent')}</div>
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
                          ? $t('astronave.entry_daily')
                          : $t('astronave.entry_free')} · {$t('astronave.entry_wave', { n: entry.wave })}
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
              {$t('astronave.daily_kicker')}
            </div>
            <div class="mt-0.5 font-display text-[30px] leading-tight">{monthLabel}</div>
            <p class="mt-1.5 text-[13.5px] leading-[1.45] text-white/70">
              {$t('astronave.daily_note')}
            </p>

            <div class="mt-4 grid grid-cols-3 gap-2.5">
              {#each [{ key: 'done', label: $t('astronave.stat_done'), value: String(doneDays), gold: false }, { key: 'streak', label: $t('astronave.stat_streak'), value: String(streak), gold: true }, { key: 'best', label: $t('astronave.stat_daily_best'), value: dailyBest ? fmtScore(dailyBest) : '—', gold: false }] as box (box.key)}
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
              {$t('astronave.daily_play')}
            </button>
          </div>
        {:else if screen === 'skins'}
          <div class="px-5 pb-[26px] pt-5">
            <div class="text-[11px] font-bold uppercase tracking-[.2em] text-mustard">
              {$t('astronave.skins_kicker')}
            </div>
            <div class="mt-0.5 font-display text-[30px] leading-tight">{$t('astronave.skins_title')}</div>

            {#if upcoming}
              <div class="mt-3 rounded-[14px] border border-gray-700 bg-gray-800/70 px-3.5 py-3">
                <div class="flex items-baseline justify-between gap-2">
                  <span class="text-[13px] font-bold text-white/85">
                    {$t('astronave.skins_next', { skin: $t(upcoming.nameKey) })}
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
              <p class="mt-3 text-[13.5px] text-white/70">{$t('astronave.skins_all')}</p>
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
                  <div class={open ? '' : 'opacity-35'}><Ship skin={item} size={56} /></div>
                  <span class="flex items-center gap-1.5 text-[13px] font-bold">
                    {#if !open}<Lock size={12} class="text-gray-500" />{/if}
                    {$t(item.nameKey)}
                  </span>
                  <span class="text-[11px] font-semibold text-gray-400">
                    {#if settings.skin === item.key}
                      {$t('astronave.skins_in_use')}
                    {:else if open}
                      {$t('astronave.skins_use')}
                    {:else}
                      {$t('astronave.skins_need', { n: fmtScore(item.need) })}
                    {/if}
                  </span>
                </button>
              {/each}
            </div>
          </div>
        {:else if screen === 'settings'}
          <div class="px-5 pb-[26px] pt-5">
            <div class="text-[11px] font-bold uppercase tracking-[.2em] text-mustard">
              {$t('astronave.tab_settings')}
            </div>
            <div class="mt-0.5 font-display text-[30px] leading-tight">{$t('astronave.settings_title')}</div>

            <div class="mt-4 rounded-[14px] border border-gray-700 bg-gray-800/70 px-3.5 py-3">
              {@render controlsPicker()}
            </div>

            <div class="mt-4 flex items-center justify-between gap-3">
              <span class="text-[15px] font-bold">{$t('astronave.language')}</span>
              <LanguageSwitcher tone="dark" />
            </div>

            <button
              type="button"
              onclick={() => (confirmReset = true)}
              class="mt-[22px] h-[46px] w-full rounded-xl border border-red-400/40 text-[14px] font-semibold text-red-300 transition-colors hover:bg-red-900/25"
            >
              {$t('astronave.reset_stats')}
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
    message={$t('astronave.reset_confirm')}
    confirmLabel={$t('astronave.reset_stats')}
    onConfirm={resetStats}
  />
</div>

<style>
  /* Lilita One não tem peso bold — evita o negrito sintético do navegador. */
  .astronave {
    font-synthesis: none;
  }
  .pad {
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 16px;
    border: 1px solid rgba(255, 255, 255, 0.16);
    background: rgba(255, 255, 255, 0.07);
    color: rgba(255, 255, 255, 0.9);
    -webkit-tap-highlight-color: transparent;
  }
  .pad:active {
    background: rgba(255, 255, 255, 0.22);
  }
  .pad-fire {
    border-color: theme('colors.mustard.DEFAULT');
    background: rgba(255, 201, 60, 0.16);
    color: theme('colors.mustard.DEFAULT');
  }
  .pad-fire:active {
    background: theme('colors.mustard.DEFAULT');
    color: theme('colors.mustard.ink');
  }
</style>
