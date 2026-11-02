<script lang="ts">
  /**
   * Enche Aí! — tela do jogo (tela cheia, sem navegação).
   *
   * Regras em `$lib/enche-ai/engine.ts`, conteúdo em `content.ts`, persistência
   * em `storage.ts` (localStorage `games_enche_ai`). Move por toque: origem e
   * depois destino. Design: docs/design/enche-ai/prototipo.html.
   */
  import { onMount } from 'svelte';
  import {
    CalendarDays, GlassWater, House, Lightbulb, Lock, RotateCcw, Undo2, X,
  } from 'lucide-svelte';
  import Patterns from '$lib/enche-ai/Patterns.svelte';
  import Vessel from '$lib/enche-ai/Vessel.svelte';
  import FlavorDot from '$lib/enche-ai/FlavorDot.svelte';
  import {
    AIDS, CAPACITY, LEVELS_PER_WORLD, applyPour, canPour, completedFlavors, dateKey, fmtTime, generate,
    hasMove, isComplete, isWon, solve, starsFor, topOf,
    type AidCount, type AidUsage, type Board, type Move,
  } from '$lib/enche-ai/engine';
  import {
    FLAVORS, VESSELS, VIEWBOX_HEIGHT, VIEWBOX_TOP, WORLDS, dailyConfig, levelConfig, worldStart,
    type LevelConfig,
  } from '$lib/enche-ai/content';
  import { HISTORY_LIMIT, emptyState, loadState, saveState, type SavedState } from '$lib/enche-ai/storage';
  import { SUPPORTED_LOCALES, locale, setLocale, t } from '$lib/i18n';

  const LIFT_PX = 18;
  const TILT_DEG = 72;
  const LOCALE_LABEL: Record<string, string> = { 'pt-BR': 'PT', en: 'EN', es: 'ES' };

  let saved = $state<SavedState>(emptyState());
  let cfg = $state<LevelConfig>(levelConfig(1));
  let board = $state<Board>([]);
  let history = $state<Board[]>([]);
  let moves = $state(0);
  let elapsed = $state(0);
  let left = $state<AidCount>({ ...AIDS });
  let used = $state<AidUsage>({ undo: false, extra: false, hint: false });

  let sel = $state<number | null>(null);
  let busy = $state(false);
  let won = $state(false);
  let grow = $state<{ index: number; from: number; key: number } | null>(null);
  let hint = $state<Move | null>(null);
  let toastMsg = $state<string | null>(null);
  let stream = $state<{ left: number; top: number; height: number; color: string } | null>(null);
  let showHome = $state(false);
  let showWin = $state(false);
  let ready = $state(false);

  let vesselEls = $state<HTMLButtonElement[]>([]);
  let rootEl = $state<HTMLDivElement>();
  let growSeq = 0;
  let toastTimer: ReturnType<typeof setTimeout> | undefined;
  let hintTimer: ReturnType<typeof setTimeout> | undefined;
  let reduced = false;

  const vessel = $derived(VESSELS[cfg.world.vessel]);
  const done = $derived(completedFlavors(board));
  const stars = $derived(starsFor(used));
  const today = $derived(dateKey(new Date()));
  const todayRecord = $derived(saved.daily[today]);
  const title = $derived(
    cfg.daily
      ? new Date(cfg.daily + 'T12:00:00').toLocaleDateString($locale, { day: '2-digit', month: 'short' }).replace('.', '')
      : $t('enche_ai.level', { n: cfg.level })
  );
  const continueLabel = $derived(cfg.daily ? $t('enche_ai.daily') : $t('enche_ai.level', { n: cfg.level }));

  // ───────── Ciclo de vida ─────────

  function configFor(level: number, daily: string | null): LevelConfig {
    if (!daily) return levelConfig(level);
    const [y, m, d] = daily.split('-').map(Number);
    return dailyConfig(new Date(y, m - 1, d), level);
  }

  function startGame(c: LevelConfig) {
    cfg = c;
    board = generate(c.seed, c.flavors);
    history = [];
    moves = 0;
    elapsed = 0;
    left = { ...AIDS };
    used = { undo: false, extra: false, hint: false };
    resetTransient();
    persist();
  }

  function resetTransient() {
    sel = null;
    busy = false;
    won = false;
    grow = null;
    hint = null;
    stream = null;
    showWin = false;
    showHome = false;
  }

  function persist() {
    saved.current = won
      ? null
      : {
          level: cfg.level,
          daily: cfg.daily,
          board: $state.snapshot(board),
          history: $state.snapshot(history).slice(-HISTORY_LIMIT),
          moves,
          elapsed,
          left: { ...left },
          used: { ...used },
        };
    saveState($state.snapshot(saved));
  }

  onMount(() => {
    reduced = matchMedia('(prefers-reduced-motion: reduce)').matches;
    saved = loadState();
    const cur = saved.current;
    if (cur) {
      cfg = configFor(cur.level, cur.daily);
      board = cur.board;
      history = cur.history ?? [];
      moves = cur.moves;
      elapsed = cur.elapsed;
      left = { ...AIDS, ...cur.left };
      used = { ...cur.used };
    } else {
      startGame(levelConfig(saved.level));
      // Primeira visita: abre o início com a explicação.
      showHome = Object.keys(saved.stars).length === 0;
    }
    ready = true;

    const tickId = setInterval(() => {
      if (!won && !showHome && document.visibilityState === 'visible') elapsed++;
    }, 1000);
    const flush = () => persist();
    document.addEventListener('visibilitychange', flush);
    window.addEventListener('pagehide', flush);
    return () => {
      clearInterval(tickId);
      document.removeEventListener('visibilitychange', flush);
      window.removeEventListener('pagehide', flush);
      persist();
    };
  });

  // ───────── Interação ─────────

  function toast(msg: string, ms = 2200) {
    toastMsg = msg;
    clearTimeout(toastTimer);
    toastTimer = setTimeout(() => (toastMsg = null), ms);
  }

  function nudge(el: HTMLElement | undefined, keyframes: Keyframe[], duration: number) {
    if (!el || reduced || typeof el.animate !== 'function') return;
    el.animate(keyframes, { duration, easing: 'ease-out' });
  }

  const shake = (i: number) =>
    nudge(vesselEls[i], [
      { transform: 'translateX(0)' }, { transform: 'translateX(-5px)' },
      { transform: 'translateX(5px)' }, { transform: 'translateX(0)' },
    ], 320);

  function onTap(i: number) {
    if (busy || won) return;
    const tube = board[i];
    if (sel === null) {
      if (!tube.length || isComplete(tube)) return shake(i);
      sel = i;
      return;
    }
    if (sel === i) {
      sel = null;
      return;
    }
    if (canPour(board, sel, i)) {
      void pour(sel, i);
      return;
    }
    if (tube.length && !isComplete(tube)) {
      sel = i;
      return;
    }
    shake(i);
    sel = null;
  }

  const wait = (ms: number) => new Promise((r) => setTimeout(r, reduced ? 0 : ms));

  function pushHistory() {
    history = [...history, $state.snapshot(board)].slice(-HISTORY_LIMIT);
  }

  async function pour(a: number, b: number) {
    busy = true;
    hint = null;
    const color = FLAVORS[topOf(board[a]) as string].color;
    const before = board[b].length;
    const wasDone = isComplete(board[b]);
    const { board: next } = applyPour($state.snapshot(board), a, b);
    pushHistory();

    const ea = vesselEls[a];
    const eb = vesselEls[b];
    const root = rootEl?.getBoundingClientRect();

    if (!reduced && root && ea && eb && typeof ea.animate === 'function') {
      const ra = ea.getBoundingClientRect();
      const rb = eb.getBoundingClientRect();
      const lift = sel === a ? -LIFT_PX : 0;
      const H = ra.height;
      const unit = H / VIEWBOX_HEIGHT;
      const mouthFromTop = (vessel.mouth - VIEWBOX_TOP) * unit;
      const acx = ra.left + ra.width / 2;
      const acy = ra.top - lift + H / 2;
      const tcx = rb.left + rb.width / 2;
      const tMouth = rb.top + mouthFromTop;
      const dir = tcx >= acx ? 1 : -1;
      const theta = (TILT_DEG * Math.PI) / 180;
      // Vetor centro→boca depois de girar: posiciona a boca da origem sobre o destino.
      const half = mouthFromTop - H / 2;
      const mx = -half * Math.sin(theta) * dir;
      const my = half * Math.cos(theta);
      const dx = tcx - (acx + mx) - dir * 4;
      const dy = tMouth - 14 - (acy + my);
      const tilted = `translate(${dx}px, ${dy}px) rotate(${dir * TILT_DEG}deg)`;

      ea.style.zIndex = '7';
      const go = ea.animate([{ transform: `translateY(${lift}px)` }, { transform: tilted }], {
        duration: 300, easing: 'cubic-bezier(.3,.7,.4,1)', fill: 'forwards',
      });
      await go.finished;

      const layerH = ((vessel.bottom - vessel.top) / CAPACITY) * unit;
      const surface = rb.top + (vessel.bottom - VIEWBOX_TOP) * unit - next[b].length * layerH;
      stream = {
        left: tcx - root.left - 3.5,
        top: tMouth - 14 - root.top,
        height: Math.max(10, surface - tMouth + 14),
        color,
      };
      board = next;
      grow = { index: b, from: before, key: ++growSeq };
      await wait(440);
      stream = null;
      sel = null;
      const back = ea.animate([{ transform: tilted }, { transform: 'translate(0, 0)' }], {
        duration: 260, easing: 'cubic-bezier(.3,.7,.4,1)', fill: 'forwards',
      });
      await back.finished;
      go.cancel();
      back.cancel();
      ea.style.zIndex = '';
    } else {
      board = next;
      sel = null;
    }

    grow = null;
    moves++;
    busy = false;

    if (!wasDone && isComplete(board[b])) {
      nudge(eb, [{ transform: 'scale(1)' }, { transform: 'scale(1.12)' }, { transform: 'scale(1)' }], 450);
      toast($t('enche_ai.toast_ready', { flavor: $t('enche_ai.flavor.' + board[b][0]) }), 1400);
      if (saved.settings.vibrate) {
        try {
          navigator.vibrate?.(30);
        } catch {
          // sem suporte
        }
      }
    }

    if (isWon(board)) {
      won = true;
      setTimeout(finishWin, reduced ? 0 : 650);
    } else if (!hasMove(board)) {
      toast($t(left.extra ? 'enche_ai.toast_stuck_extra' : 'enche_ai.toast_stuck'), 3200);
    }
    persist();
  }

  function finishWin() {
    const n = starsFor(used);
    if (cfg.daily) {
      const prev = saved.daily[cfg.daily];
      if (!prev || n > prev.stars || (n === prev.stars && moves < prev.moves)) {
        saved.daily[cfg.daily] = { moves, time: elapsed, stars: n };
      }
    } else {
      saved.stars[cfg.level] = Math.max(saved.stars[cfg.level] ?? 0, n);
      saved.level = Math.max(saved.level, cfg.level + 1);
    }
    persist();
    showWin = true;
  }

  function undo() {
    if (busy || won || !history.length || !left.undo) return;
    board = history[history.length - 1];
    history = history.slice(0, -1);
    left.undo--;
    used.undo = true;
    moves++;
    sel = null;
    hint = null;
    persist();
  }

  function extra() {
    if (busy || won || !left.extra) return;
    board = [...board, []];
    history = history.map((h) => [...h, []]);
    left.extra--;
    used.extra = true;
    sel = null;
    toast($t('enche_ai.toast_extra'));
    persist();
  }

  function giveHint() {
    if (busy || won || !left.hint) return;
    const path = solve($state.snapshot(board), 60000);
    if (!path) {
      toast($t(left.extra ? 'enche_ai.toast_no_way_extra' : 'enche_ai.toast_no_way'), 3000);
      return;
    }
    if (!path.length) return;
    left.hint--;
    used.hint = true;
    sel = null;
    hint = path[0];
    toast($t('enche_ai.toast_hint'), 2000);
    clearTimeout(hintTimer);
    hintTimer = setTimeout(() => (hint = null), 2000);
    persist();
  }

  function restart() {
    if (!busy) startGame(configFor(cfg.level, cfg.daily));
  }

  function playDaily() {
    if (busy) return;
    startGame(dailyConfig(new Date(), saved.level));
    toast($t('enche_ai.toast_daily'));
  }

  function nextOrder() {
    startGame(levelConfig(cfg.daily ? saved.level : cfg.level + 1));
  }

  function playWorld(i: number) {
    const from = worldStart(i);
    if (saved.level < from) return;
    startGame(levelConfig(Math.min(saved.level, from + LEVELS_PER_WORLD - 1)));
  }

  function worldStars(i: number): number {
    const from = worldStart(i);
    let n = 0;
    for (let l = from; l < from + LEVELS_PER_WORLD; l++) n += saved.stars[l] ?? 0;
    return n;
  }

  function setOption(key: 'patterns' | 'vibrate', value: boolean) {
    saved.settings[key] = value;
    persist();
  }

  function onKey(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      showHome = false;
      sel = null;
    } else if ((e.key === 'z' || e.key === 'Z') && (e.ctrlKey || e.metaKey)) {
      e.preventDefault();
      undo();
    }
  }

  function tubeLabel(i: number): string {
    const tube = board[i];
    const contents = tube.length
      ? tube.slice().reverse().map((id) => $t('enche_ai.flavor.' + id)).join(', ')
      : $t('enche_ai.tube_empty');
    const prefix = isComplete(tube) ? $t('enche_ai.tube_ready') + ', ' : '';
    return $t('enche_ai.tube_label', { vessel: $t('enche_ai.vessel.' + vessel.id), n: i + 1, contents: prefix + contents });
  }
</script>

<svelte:head>
  <title>{$t('enche_ai.name')}</title>
  <meta name="description" content={$t('enche_ai.desc')} />
  <meta name="theme-color" content="#183a69" />
  <link rel="manifest" href="/enche-ai/manifest.webmanifest" />
  <link rel="apple-touch-icon" href="/enche-ai/apple-touch-icon.png" />
  <meta name="apple-mobile-web-app-title" content={$t('enche_ai.name')} />
</svelte:head>

<svelte:window onkeydown={onKey} />

<Patterns />

<div class="fixed inset-0 z-50 flex justify-center bg-ink" class:ea-no-pat={!saved.settings.patterns}>
  <div
    bind:this={rootEl}
    class="relative flex h-full w-full max-w-[480px] select-none flex-col overflow-hidden bg-wood-dark sm:border-x sm:border-white/10"
  >
    <!-- Parede de azulejo + HUD -->
    <header class="scene px-3.5 pb-9">
      <div class="grid grid-cols-4 gap-2.5">
        <button class="hud-btn" onclick={() => { if (!busy) { sel = null; showHome = true; } }} aria-label={$t('enche_ai.home')} title={$t('enche_ai.home')}>
          <House size={24} strokeWidth={2.4} />
        </button>
        <button class="hud-btn" onclick={restart} aria-label={$t('enche_ai.restart')} title={$t('enche_ai.restart')}>
          <RotateCcw size={24} strokeWidth={2.4} />
        </button>
        <button class="hud-btn" onclick={undo} disabled={!left.undo || !history.length} aria-label={$t('enche_ai.undo')} title={$t('enche_ai.undo')}>
          <Undo2 size={24} strokeWidth={2.4} />
          <span class="count">{left.undo}</span>
        </button>
        <button class="hud-btn" onclick={extra} disabled={!left.extra} aria-label={$t('enche_ai.extra')} title={$t('enche_ai.extra')}>
          <span class="flex items-center gap-0.5"><span class="text-lg font-extrabold leading-none">+</span><GlassWater size={22} strokeWidth={2.4} /></span>
          <span class="count">{left.extra}</span>
        </button>
      </div>

      <div class="mt-4 grid grid-cols-[52px_1fr_52px] items-center gap-2">
        <button class="round-btn !bg-ketchup" onclick={playDaily} aria-label={$t('enche_ai.daily')} title={$t('enche_ai.daily')}>
          <CalendarDays size={24} strokeWidth={2.4} />
        </button>
        <div class="text-center leading-none">
          <span class="mb-1 block text-xs font-extrabold uppercase tracking-[.14em] text-mustard">
            {cfg.daily ? $t('enche_ai.daily') + ' · ' : ''}{$t('enche_ai.world.' + cfg.world.id)}
          </span>
          <h1 class="font-display text-4xl tracking-wide text-white [text-shadow:0_3px_0_rgba(0,0,0,.35)]">{title}</h1>
        </div>
        <button class="round-btn" onclick={giveHint} disabled={!left.hint} aria-label={$t('enche_ai.hint')} title={$t('enche_ai.hint')}>
          <Lightbulb size={24} strokeWidth={2.4} />
          <span class="count">{left.hint}</span>
        </button>
      </div>
    </header>

    <!-- Comanda -->
    <section class="comanda relative z-[2] mx-[18px] -mt-[22px] bg-paper px-3.5 pb-4 pt-2.5 text-paper-ink" aria-label={$t('enche_ai.comanda')}>
      <div class="mb-1.5 flex justify-between border-b border-dashed border-paper-ink/30 pb-1 text-[11px] font-extrabold uppercase tracking-[.14em] text-paper-muted tabular-nums">
        <span>{$t('enche_ai.comanda')}</span>
        <span>{$t('enche_ai.ready_count', { done: done.size, total: cfg.flavors.length })}</span>
      </div>
      <ul class="flex flex-wrap gap-x-3 gap-y-1">
        {#each cfg.flavors as id (id)}
          <li class="inline-flex items-center gap-1.5 whitespace-nowrap text-[13px] font-bold" class:is-done={done.has(id)}>
            <FlavorDot {id} />
            {$t('enche_ai.flavor.' + id)}
            {#if done.has(id)}<span class="font-extrabold text-green-700">✓</span>{/if}
          </li>
        {/each}
      </ul>
    </section>

    <!-- Balcão -->
    <section
      class="counter flex flex-1 flex-wrap content-center justify-center gap-x-2 gap-y-[22px] px-3 pb-5 pt-8"
      style="--vessel-w: {board.length <= 6 ? 'clamp(52px, 15vw, 70px)' : 'clamp(42px, 11.5vw, 56px)'}"
      aria-label={$t('enche_ai.board')}
    >
      {#if ready}
        {#each board as tube, i (i)}
          <button
            bind:this={vesselEls[i]}
            class="vessel"
            class:lifted={sel === i}
            class:glow={hint !== null && (hint[0] === i || hint[1] === i)}
            onclick={() => onTap(i)}
            aria-label={tubeLabel(i)}
            aria-pressed={sel === i}
          >
            <Vessel
              {vessel}
              layers={tube}
              growFrom={grow?.index === i ? grow.from : null}
              growKey={grow?.index === i ? grow.key : 0}
              topping={grow?.index !== i}
            />
          </button>
        {/each}
      {/if}
    </section>

    <footer class="flex items-center justify-between gap-2 border-t border-white/5 bg-wood-dark px-[18px] pt-2.5 text-[13px] text-[#d9c3ae] tabular-nums" style="padding-bottom: max(14px, env(safe-area-inset-bottom))">
      <span>{$t('enche_ai.moves')} <b class="text-white">{moves}</b></span>
      <span class="tracking-widest text-mustard" aria-label={$t('enche_ai.stars_label', { n: stars })}>
        {'★'.repeat(stars)}<span class="text-white/20">{'★'.repeat(3 - stars)}</span>
      </span>
      <span>{$t('enche_ai.time')} <b class="text-white">{fmtTime(elapsed)}</b></span>
    </footer>

    {#if stream}
      <div
        class="stream"
        style="left:{stream.left}px;top:{stream.top}px;height:{stream.height}px;background:{stream.color}"
      ></div>
    {/if}

    {#if toastMsg}
      <div class="toast" role="status">{toastMsg}</div>
    {/if}

    <!-- Vitória -->
    {#if showWin}
      <div class="overlay">
        <div class="card text-center" role="dialog" aria-modal="true" aria-labelledby="win-title">
          <h2 id="win-title" class="font-display text-[32px] leading-tight">
            {$t(cfg.daily ? 'enche_ai.win_daily_title' : 'enche_ai.win_title')}
          </h2>
          <div class="my-1.5 text-[38px] tracking-[.12em] text-[#f2a900]">
            {'★'.repeat(stars)}<span class="text-paper-ink/20">{'★'.repeat(3 - stars)}</span>
          </div>
          <p class="text-sm text-paper-muted">{$t('enche_ai.win_sub_' + stars)}</p>
          <dl class="my-4 grid grid-cols-3 gap-2">
            <div><dt class="stat-label">{$t('enche_ai.moves')}</dt><dd class="stat">{moves}</dd></div>
            <div><dt class="stat-label">{$t('enche_ai.time')}</dt><dd class="stat">{fmtTime(elapsed)}</dd></div>
            <div>
              <dt class="stat-label">{$t('enche_ai.aids')}</dt>
              <dd class="stat">{AIDS.undo - left.undo + AIDS.extra - left.extra + AIDS.hint - left.hint}</dd>
            </div>
          </dl>
          <button class="btn btn-primary" onclick={nextOrder}>
            {$t(cfg.daily ? 'enche_ai.back_to_orders' : 'enche_ai.next')}
          </button>
          <button class="btn btn-ghost mt-2" onclick={restart}>{$t('enche_ai.replay')}</button>
        </div>
      </div>
    {/if}

    <!-- Início -->
    {#if showHome}
      <div class="overlay overflow-y-auto" onclick={(e) => { if (e.target === e.currentTarget) showHome = false; }} role="presentation">
        <div class="card relative" role="dialog" aria-modal="true" aria-labelledby="home-title">
          <button class="absolute right-3 top-3 grid h-9 w-9 place-items-center rounded-full text-paper-muted hover:bg-paper-ink/10" onclick={() => (showHome = false)} aria-label={$t('enche_ai.close')}>
            <X size={20} />
          </button>
          <h2 id="home-title" class="text-center font-display text-[44px] leading-none text-ketchup">{$t('enche_ai.name')}</h2>
          <p class="mb-1 mt-1 text-center text-sm text-paper-muted">{$t('enche_ai.tagline')}</p>
          <p class="mb-3.5 text-center text-[13px] leading-snug text-paper-ink/80">{$t('enche_ai.how_to')}</p>

          <button class="btn btn-primary" onclick={() => (showHome = false)}>
            {moves > 0 ? $t('enche_ai.continue', { label: continueLabel }) : $t('enche_ai.start')}
          </button>
          <button class="btn btn-ghost mt-2" onclick={playDaily}>
            {$t('enche_ai.daily')}{todayRecord ? ' · ' + $t('enche_ai.daily_done', { stars: todayRecord.stars }) : ''}
          </button>

          <h3 class="section-label">{$t('enche_ai.menus')}</h3>
          <div class="grid gap-1.5">
            {#each WORLDS as w, i (w.id)}
              {@const from = worldStart(i)}
              {@const locked = saved.level < from}
              <button class="menu-item" disabled={locked} onclick={() => playWorld(i)}>
                <span class="w-[26px]"><Vessel vessel={VESSELS[w.vessel]} layers={w.order.slice(0, 3)} topping={false} /></span>
                <span class="min-w-0">
                  <b class="block text-[15px]">{$t('enche_ai.world.' + w.id)}</b>
                  <small class="text-xs text-paper-muted">
                    {#if locked}
                      <Lock size={11} class="-mt-0.5 inline" /> {$t('enche_ai.locked', { n: from })}
                    {:else}
                      {$t('enche_ai.levels_range', { from, to: from + LEVELS_PER_WORLD - 1 })}
                    {/if}
                  </small>
                </span>
                <span class="text-xs font-extrabold text-paper-muted tabular-nums">★ {worldStars(i)}/{LEVELS_PER_WORLD * 3}</span>
              </button>
            {/each}
          </div>

          <h3 class="section-label">{$t('enche_ai.settings')}</h3>
          <label class="toggle" for="opt-patterns">
            {$t('enche_ai.opt_patterns')}
            <input id="opt-patterns" type="checkbox" checked={saved.settings.patterns} onchange={(e) => setOption('patterns', e.currentTarget.checked)} />
          </label>
          <label class="toggle" for="opt-vibrate">
            {$t('enche_ai.opt_vibrate')}
            <input id="opt-vibrate" type="checkbox" checked={saved.settings.vibrate} onchange={(e) => setOption('vibrate', e.currentTarget.checked)} />
          </label>
          <div class="toggle">
            <span>{$t('enche_ai.language')}</span>
            <span class="flex gap-1">
              {#each SUPPORTED_LOCALES as l (l)}
                <button
                  class="h-8 rounded-lg px-2.5 text-xs font-extrabold {$locale === l ? 'bg-paper-ink text-paper' : 'bg-paper-ink/10'}"
                  onclick={() => setLocale(l)}
                  aria-pressed={$locale === l}
                >{LOCALE_LABEL[l]}</button>
              {/each}
            </span>
          </div>

          <a href="/" class="mt-3 block text-center text-sm font-bold text-paper-muted underline underline-offset-2">{$t('enche_ai.all_games')}</a>
        </div>
      </div>
    {/if}
  </div>
</div>

<style>
  .scene {
    padding-top: max(14px, env(safe-area-inset-top));
    background-color: #183a69;
    background-image:
      linear-gradient(180deg, rgba(15, 22, 34, 0.15) 0%, rgba(15, 22, 34, 0.55) 70%, rgba(31, 19, 12, 1) 100%),
      url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='44' height='44' viewBox='0 0 44 44'%3E%3Crect width='44' height='44' fill='%23183a69'/%3E%3Cpath d='M22 7C25 16 28 19 37 22C28 25 25 28 22 37C19 28 16 25 7 22C16 19 19 16 22 7Z' fill='%232f68ad'/%3E%3Ccircle cx='22' cy='22' r='3.5' fill='%23d4e3f6' fill-opacity='.6'/%3E%3Cg fill='%232f68ad'%3E%3Ccircle cx='0' cy='0' r='7'/%3E%3Ccircle cx='44' cy='0' r='7'/%3E%3Ccircle cx='0' cy='44' r='7'/%3E%3Ccircle cx='44' cy='44' r='7'/%3E%3C/g%3E%3Crect x='.5' y='.5' width='43' height='43' fill='none' stroke='%23d4e3f6' stroke-opacity='.16'/%3E%3C/svg%3E");
    background-size: auto, 44px 44px;
  }
  .hud-btn,
  .round-btn {
    position: relative;
    display: grid;
    place-items: center;
    height: 48px;
    border-radius: 14px;
    color: #fff;
    background: rgba(214, 228, 248, 0.2);
    box-shadow: inset 0 -3px 0 rgba(0, 0, 0, 0.25), 0 4px 10px rgba(0, 0, 0, 0.25);
    backdrop-filter: blur(3px);
    transition: transform 0.12s, background 0.12s;
  }
  .round-btn {
    width: 52px;
    height: 52px;
    border-radius: 50%;
  }
  .hud-btn:hover,
  .round-btn:hover {
    background: rgba(214, 228, 248, 0.3);
  }
  .hud-btn:active,
  .round-btn:active {
    transform: translateY(2px);
  }
  .hud-btn:disabled,
  .round-btn:disabled {
    opacity: 0.4;
    cursor: default;
  }
  .count {
    position: absolute;
    top: -6px;
    right: -4px;
    min-width: 20px;
    height: 20px;
    padding: 0 5px;
    border-radius: 10px;
    background: #ffc93c;
    color: #3a2a00;
    font-size: 12px;
    font-weight: 800;
    line-height: 20px;
    text-align: center;
    font-variant-numeric: tabular-nums;
    box-shadow: 0 0 0 2px #183a69;
  }

  .comanda {
    transform: rotate(-0.8deg);
    box-shadow: 0 8px 18px rgba(0, 0, 0, 0.35);
    -webkit-mask: radial-gradient(6px at 50% calc(100% + 1px), transparent 98%, #000) 0 0 / 12px 100% repeat-x;
    mask: radial-gradient(6px at 50% calc(100% + 1px), transparent 98%, #000) 0 0 / 12px 100% repeat-x;
  }
  .is-done {
    color: #8a7563;
    text-decoration: line-through;
    text-decoration-thickness: 2px;
  }

  .counter {
    background:
      radial-gradient(70% 45% at 50% 0%, rgba(255, 214, 150, 0.14), transparent 70%),
      repeating-linear-gradient(90deg, rgba(255, 255, 255, 0.022) 0 2px, transparent 2px 11px),
      linear-gradient(180deg, #3b2519, #1f130c);
  }
  .vessel {
    position: relative;
    flex: none;
    width: var(--vessel-w);
    transition: transform 0.18s cubic-bezier(0.3, 0.7, 0.4, 1);
  }
  .vessel::after {
    content: '';
    position: absolute;
    left: 12%;
    right: 12%;
    bottom: -5px;
    height: 8px;
    border-radius: 50%;
    background: rgba(0, 0, 0, 0.45);
    filter: blur(3px);
    z-index: -1;
  }
  .vessel.lifted {
    transform: translateY(-18px);
  }
  .vessel.glow {
    filter: drop-shadow(0 0 8px #ffc93c) drop-shadow(0 0 2px #ffc93c);
  }

  .stream {
    position: absolute;
    z-index: 6;
    width: 7px;
    border-radius: 4px;
    pointer-events: none;
    transform-origin: 50% 0;
    animation: pour-in 0.12s ease-out both;
  }
  @keyframes pour-in {
    from { transform: scaleY(0); }
    to { transform: scaleY(1); }
  }

  .toast {
    position: absolute;
    left: 50%;
    bottom: calc(58px + env(safe-area-inset-bottom));
    z-index: 8;
    width: max-content;
    max-width: calc(100% - 40px);
    transform: translateX(-50%);
    padding: 8px 14px;
    border-radius: 10px;
    background: #fbf4e4;
    color: #3b2a20;
    font-size: 14px;
    font-weight: 700;
    text-align: center;
    box-shadow: 0 8px 20px rgba(0, 0, 0, 0.4);
  }

  .overlay {
    position: absolute;
    inset: 0;
    z-index: 10;
    display: grid;
    place-items: center;
    padding: 20px;
    background: rgba(10, 14, 22, 0.72);
    backdrop-filter: blur(3px);
  }
  .card {
    width: 100%;
    max-width: 360px;
    padding: 22px 20px 20px;
    border-radius: 18px;
    background: #fbf4e4;
    color: #3b2a20;
    box-shadow: 0 20px 40px rgba(0, 0, 0, 0.5);
  }
  .stat-label {
    font-size: 11px;
    font-weight: 800;
    letter-spacing: 0.12em;
    text-transform: uppercase;
    color: #8a7563;
  }
  .stat {
    font-family: 'Lilita One', system-ui, sans-serif;
    font-size: 24px;
    font-variant-numeric: tabular-nums;
  }
  .btn {
    display: flex;
    width: 100%;
    height: 48px;
    align-items: center;
    justify-content: center;
    border-radius: 12px;
    font-size: 16px;
    font-weight: 800;
    box-shadow: inset 0 -3px 0 rgba(0, 0, 0, 0.18);
  }
  .btn-primary {
    background: #ffc93c;
    color: #3a2a00;
  }
  .btn-ghost {
    box-shadow: inset 0 0 0 2px rgba(59, 42, 32, 0.2);
  }
  .section-label {
    margin: 16px 0 6px;
    font-size: 11px;
    font-weight: 800;
    letter-spacing: 0.14em;
    text-transform: uppercase;
    color: #8a7563;
  }
  .menu-item {
    display: grid;
    grid-template-columns: 26px 1fr auto;
    align-items: center;
    gap: 12px;
    width: 100%;
    padding: 6px 10px;
    border-radius: 10px;
    text-align: left;
    background: rgba(59, 42, 32, 0.07);
  }
  .menu-item:hover:not(:disabled) {
    background: rgba(59, 42, 32, 0.13);
  }
  .menu-item:disabled {
    opacity: 0.55;
    cursor: default;
  }
  .toggle {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 10px;
    padding: 6px 2px;
    font-size: 14px;
    font-weight: 700;
  }
  .toggle input {
    width: 20px;
    height: 20px;
    flex: none;
    accent-color: #e8473b;
  }
</style>
