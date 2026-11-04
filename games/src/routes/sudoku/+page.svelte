<script lang="ts">
  /**
   * Sudoku — o clássico de lógica, no visual do games.thiagotn.com.
   *
   * Rota em tela cheia (`fixed inset-0`), sem Navbar, mobile-first. As abas
   * (Início · Diário · Visuais · Ajustes) ficam por baixo; a partida entra por
   * cima delas — e, como o jogo é retomável, a aba Início continua viva com o
   * botão Continuar. As regras vivem em `$lib/sudoku/engine.ts`, os visuais em
   * `themes.ts` e a persistência em `storage.ts`; aqui moram a entrada do
   * jogador, o cronômetro e a interface.
   *
   * A casca é escura como o resto do hub (sepia noturno) e o tabuleiro é a
   * folha de papel em cima dela — as cores da folha saem do tema escolhido.
   */
  import {
    Calendar,
    ChevronLeft,
    Eraser,
    House,
    Lightbulb,
    Lock,
    Palette,
    Pause,
    Pencil,
    Play,
    Settings as SettingsIcon,
    Timer,
    Trophy,
    Undo2,
  } from 'lucide-svelte';
  import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
  import LanguageSwitcher from '$lib/components/LanguageSwitcher.svelte';
  import Emblem from '$lib/sudoku/Emblem.svelte';
  import {
    CELLS,
    DIFFICULTIES,
    boxOf,
    colOf,
    conflicts,
    dailyDifficulty,
    dailyGame,
    dateKey,
    digitsOf,
    erase,
    fmtTime,
    hint as nextHint,
    newGame,
    place,
    randomSeed,
    remainingOf,
    rowOf,
    toggleNote,
    undo as undoMove,
    useHint,
    wrongCells,
    type Difficulty,
    type Game,
    type Technique,
  } from '$lib/sudoku/engine';
  import { THEMES, isUnlocked, nextLock, themeOf, type ThemeKey } from '$lib/sudoku/themes';
  import {
    HISTORY_VISIBLE,
    addStart,
    addWin,
    bestDaily,
    emptyState,
    loadState,
    recordDaily,
    recordMatch,
    saveState,
    streakOf,
    totalWins,
    type SavedState,
  } from '$lib/sudoku/storage';
  import { locale, t } from '$lib/i18n';

  type Screen = 'home' | 'game' | 'daily' | 'themes' | 'settings';

  type Result = {
    difficulty: Difficulty;
    time: number;
    hints: number;
    mistakes: number;
    /** Melhores de antes da partida — é a coluna "de sempre" do resumo. */
    bestTime: number;
    bestHints: number;
    bestMistakes: number;
    hadWin: boolean;
    record: boolean;
    daily: string | null;
    /** Visuais que esta partida liberou. */
    unlocked: ThemeKey[];
  };

  const TABS = [
    { key: 'home' as const, labelKey: 'sudoku.tab_home', icon: House },
    { key: 'daily' as const, labelKey: 'sudoku.tab_daily', icon: Calendar },
    { key: 'themes' as const, labelKey: 'sudoku.tab_themes', icon: Palette },
    { key: 'settings' as const, labelKey: 'sudoku.tab_settings', icon: SettingsIcon },
  ];

  const CELL_LIST = [...Array(CELLS).keys()];
  const DIGITS = [1, 2, 3, 4, 5, 6, 7, 8, 9];

  let screen = $state<Screen>('home');
  let saved = $state<SavedState>(emptyState());
  /** `raw`: o engine devolve um estado novo a cada lance, não há o que observar por dentro. */
  let game = $state.raw<Game | null>(null);
  let paused = $state(false);
  let result = $state<Result | null>(null);
  let confirmReset = $state(false);
  /** Gerar um Especialista leva uns quadros — a tela avisa em vez de congelar. */
  let building = $state(false);
  let selected = $state<number | null>(null);
  /** Número escolhido no teclado, quando o ajuste "número primeiro" está ligado. */
  let chosen = $state<number | null>(null);
  let pencil = $state(false);
  let toast = $state('');
  /** Célula que a última dica preencheu — a tela pisca nela. */
  let hinted = $state<number | null>(null);

  let toastTimer = 0;
  /** Próximo tabuleiro, gerado na folga — faz "Novo jogo" ser instantâneo. */
  let warm: { difficulty: Difficulty; game: Game } | null = null;

  const today = new Date();
  const todayKey = dateKey(today);

  let stats = $derived(saved.stats);
  let settings = $derived(saved.settings);
  let theme = $derived(themeOf(saved.settings.theme));
  let wins = $derived(totalWins(saved.stats));
  let upcoming = $derived(nextLock(wins));
  let history = $derived(saved.history.slice(0, HISTORY_VISIBLE));
  let inGame = $derived(screen === 'game' && !!game);
  let todayRecord = $derived(saved.daily[todayKey]);
  let streak = $derived(streakOf(saved.daily, today));
  let doneDays = $derived(Object.values(saved.daily).filter((d) => d.done).length);
  let dailyRecord = $derived(bestDaily(saved.daily));
  let todayDifficulty = $derived(dailyDifficulty(todayKey));

  /** Células em conflito, só quando o ajuste de checagem está ligado. */
  let bad = $derived(
    game && settings.errors ? conflicts(game.values) : new Array<boolean>(CELLS).fill(false),
  );
  let left = $derived(game ? remainingOf(game.values) : new Array<number>(10).fill(9));
  /** Dígito em foco: o da célula escolhida ou o do teclado (modo número primeiro). */
  let focusDigit = $derived(
    selected !== null && game?.values[selected] ? game.values[selected] : chosen,
  );
  let progress = $derived(
    game ? Math.round((game.values.filter((v) => v > 0).length / CELLS) * 100) : 0,
  );

  $effect(() => {
    saved = loadState();
  });

  /** Cronômetro: 1 s por vez, só com a partida à vista e algum lance dado. */
  $effect(() => {
    const id = setInterval(() => {
      if (screen !== 'game' || !game || game.won || paused || !game.undo.length) return;
      game = { ...game, elapsed: game.elapsed + 1 };
    }, 1000);
    return () => clearInterval(id);
  });

  $effect(() => {
    return () => clearTimeout(toastTimer);
  });

  function persist(next: SavedState) {
    saved = next;
    saveState(next);
  }

  function update<K extends keyof SavedState['settings']>(key: K, value: SavedState['settings'][K]) {
    persist({ ...saved, settings: { ...settings, [key]: value } });
  }

  function say(message: string) {
    toast = message;
    clearTimeout(toastTimer);
    toastTimer = setTimeout(() => (toast = ''), 2600) as unknown as number;
  }

  /** Guarda a partida em andamento (o jogador pode fechar a aba a qualquer hora). */
  function keep(g: Game | null) {
    persist({ ...saved, game: g && !g.won ? g : null });
  }

  /**
   * Gera o tabuleiro depois de dois quadros: o primeiro agenda, o segundo roda
   * já com o "Montando o tabuleiro…" pintado na tela.
   */
  function build(make: () => Game, difficulty: Difficulty) {
    building = true;
    screen = 'game';
    requestAnimationFrame(() =>
      requestAnimationFrame(() => {
        const g = make();
        game = g;
        building = false;
        paused = false;
        selected = null;
        chosen = null;
        pencil = false;
        hinted = null;
        result = null;
        persist({ ...saved, game: g, stats: addStart(saved.stats, difficulty) });
        prewarm(difficulty);
      }),
    );
  }

  function startGame(difficulty: Difficulty) {
    const ready = warm && warm.difficulty === difficulty ? warm.game : null;
    warm = null;
    build(() => ready ?? newGame(difficulty, randomSeed()), difficulty);
  }

  function startDaily(day: string) {
    build(() => dailyGame(day), dailyDifficulty(day));
  }

  /** Deixa o próximo tabuleiro pronto enquanto o jogador pensa. */
  function prewarm(difficulty: Difficulty) {
    const idle =
      typeof requestIdleCallback === 'function'
        ? requestIdleCallback
        : (fn: () => void) => setTimeout(fn, 400);
    idle(() => {
      if (warm?.difficulty === difficulty) return;
      warm = { difficulty, game: newGame(difficulty, randomSeed()) };
    });
  }

  function resumeGame() {
    if (!saved.game) return;
    game = saved.game;
    paused = false;
    result = null;
    screen = 'game';
    selected = null;
  }

  function quitGame() {
    keep(game);
    result = null;
    screen = 'home';
  }

  function go(next: Screen) {
    if (inGame) keep(game);
    screen = next;
  }

  function finish(g: Game) {
    const before = saved.stats[g.difficulty];
    const record = before.wins === 0 || g.elapsed < before.best;
    const unlockedBefore = THEMES.filter((item) => wins >= item.need).map((item) => item.key);
    const nextStats = addWin(saved.stats, g);
    const nextWins = totalWins(nextStats);

    persist({
      ...saved,
      game: null,
      stats: nextStats,
      history: recordMatch(saved.history, g),
      daily: g.daily ? recordDaily(saved.daily, g.daily, g.elapsed, g.hints) : saved.daily,
    });

    result = {
      difficulty: g.difficulty,
      time: g.elapsed,
      hints: g.hints,
      mistakes: g.mistakes,
      bestTime: before.best,
      bestHints: before.bestHints,
      bestMistakes: before.bestMistakes,
      hadWin: before.wins > 0,
      record,
      daily: g.daily,
      unlocked: THEMES.filter(
        (item) => nextWins >= item.need && !unlockedBefore.includes(item.key),
      ).map((item) => item.key),
    };
  }

  function commit(next: Game) {
    game = next;
    if (next.won) finish(next);
    else keep(next);
  }

  // ── Entrada do jogador ────────────────────────────────────────────────────

  function tapCell(at: number) {
    if (!game || game.won || paused) return;
    hinted = null;

    // Número primeiro: o toque na célula escreve o dígito já escolhido.
    if (settings.numberFirst && chosen !== null && !game.givens[at]) {
      write(at, chosen);
      selected = at;
      return;
    }
    // Toque na casa sempre escolhe (nada de alternar): "Apagar" e o teclado
    // dependem de ter uma casa escolhida, e perder a seleção sem querer irrita.
    selected = at;
  }

  function tapDigit(digit: number) {
    if (!game || game.won || paused) return;
    hinted = null;

    if (settings.numberFirst) {
      chosen = chosen === digit ? null : digit;
      if (selected !== null && chosen !== null) write(selected, chosen);
      return;
    }
    if (selected === null) {
      say($t('sudoku.pick_cell'));
      return;
    }
    write(selected, digit);
  }

  function write(at: number, digit: number) {
    if (!game || game.givens[at]) return;
    commit(
      pencil ? toggleNote(game, at, digit) : place(game, at, digit, settings.autoNotes),
    );
  }

  function doErase() {
    if (!game || game.won || paused || selected === null) return;
    commit(erase(game, selected));
  }

  function doUndo() {
    if (!game || paused) return;
    hinted = null;
    commit(undoMove(game));
  }

  function doHint() {
    if (!game || game.won || paused) return;
    const wrong = wrongCells(game);
    if (wrong.length) {
      selected = wrong[0];
      say($t('sudoku.hint_wrong'));
      return;
    }
    const h = nextHint(game);
    if (!h) return;
    commit(useHint(game, h));
    selected = h.at;
    hinted = h.at;
    say($t(techniqueKey(h.technique), { digit: h.digit }));
  }

  const techniqueKey = (technique: Technique) => `sudoku.tech.${technique}`;

  function onKeydown(event: KeyboardEvent) {
    if (!inGame || !game) return;
    const key = event.key;

    if (key === ' ' || key === 'p' || key === 'P' || key === 'Escape') {
      event.preventDefault();
      if (!game.won) paused = !paused;
      return;
    }
    if (paused || game.won) return;

    if (key >= '1' && key <= '9') {
      event.preventDefault();
      tapDigit(Number(key));
      return;
    }
    if (key === '0' || key === 'Backspace' || key === 'Delete') {
      event.preventDefault();
      doErase();
      return;
    }
    if (key === 'n' || key === 'N') {
      pencil = !pencil;
      return;
    }
    if (key === 'h' || key === 'H') {
      doHint();
      return;
    }
    if (key === 'z' || key === 'Z') {
      doUndo();
      return;
    }

    const step: Record<string, number> = {
      ArrowUp: -9,
      ArrowDown: 9,
      ArrowLeft: -1,
      ArrowRight: 1,
      w: -9,
      s: 9,
      a: -1,
      d: 1,
    };
    const delta = step[key];
    if (delta === undefined) return;
    event.preventDefault();
    const from = selected ?? 40;
    const to = from + delta;
    if (to < 0 || to >= CELLS) return;
    // Não deixa a seta dar a volta pela borda: a linha tem de continuar a mesma.
    if (Math.abs(delta) === 1 && rowOf(to) !== rowOf(from)) return;
    selected = to;
  }

  function resetStats() {
    persist({ ...emptyState(), settings: { ...settings, theme: 'jornal' } });
    game = null;
    result = null;
    screen = 'home';
    say($t('sudoku.reset_done'));
  }

  // ── Aparência do tabuleiro ───────────────────────────────────────────────

  /** Fundo da célula: seleção, casas em destaque, iguais e o xadrez dos blocos. */
  function cellBackground(at: number): string {
    if (!game) return theme.paper;
    const box = boxOf(at);
    const checker = (((box / 3) | 0) + (box % 3)) % 2 === 0 ? theme.paper : theme.paperAlt;
    if (selected === at) return theme.sel;
    if (!settings.highlight) return checker;
    if (focusDigit && game.values[at] === focusDigit) return theme.same;
    if (
      selected !== null &&
      (rowOf(at) === rowOf(selected) || colOf(at) === colOf(selected) || box === boxOf(selected))
    ) {
      return theme.peer;
    }
    return checker;
  }

  /** Tinta do número: dado, do jogador, ou vermelho quando conflita. */
  function cellInk(at: number): string {
    if (!game) return theme.given;
    if (bad[at]) return theme.bad;
    return game.givens[at] ? theme.given : theme.pen;
  }

  const difficultyKey = (difficulty: Difficulty) => `sudoku.diff_${difficulty}`;

  // ── Calendário do desafio do dia ─────────────────────────────────────────

  let weekdays = $derived($t('sudoku.weekdays').split(','));
  let monthLabel = $derived.by(() => {
    const raw = today.toLocaleDateString($locale, { month: 'long', year: 'numeric' }).replace(' de ', ' ');
    return raw.charAt(0).toUpperCase() + raw.slice(1);
  });
  /** Curto de propósito: o cartão do dia mostra data + dificuldade na mesma linha. */
  let todayLabel = $derived(
    today.toLocaleDateString($locale, { day: 'numeric', month: 'short' }).replace(/\./g, ''),
  );
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
</script>

<svelte:head>
  <title>{$t('sudoku.title')} — games.thiagotn.com</title>
  <meta name="description" content={$t('sudoku.desc')} />
  <meta name="theme-color" content="#171310" />
  <link rel="manifest" href="/sudoku/manifest.webmanifest" />
  <link rel="apple-touch-icon" href="/sudoku/apple-touch-icon.png" />
  <meta name="apple-mobile-web-app-title" content={$t('sudoku.title')} />
</svelte:head>

<svelte:window onkeydown={onKeydown} />

<div class="sudoku fixed inset-0 z-50 flex flex-col overflow-hidden bg-news text-white">
  {#if inGame && game}
    <!-- ── Partida ──────────────────────────────────────────────────────── -->
    <header
      class="relative z-10 flex flex-none items-center gap-2 px-3 pb-2"
      style="padding-top:calc(env(safe-area-inset-top) + 6px)"
    >
      <button
        type="button"
        onclick={quitGame}
        aria-label={$t('sudoku.quit')}
        class="flex h-9 w-9 flex-none items-center justify-center rounded-lg text-white/75 transition-colors hover:bg-white/10 hover:text-white"
      >
        <ChevronLeft size={20} strokeWidth={2.2} />
      </button>

      <div class="min-w-0 flex-1">
        <div class="truncate font-display text-[19px] leading-none">
          {$t(difficultyKey(game.difficulty))}
        </div>
        <div class="mt-0.5 flex items-center gap-2 text-[11.5px] font-semibold text-white/60">
          {#if game.daily}
            <span class="inline-flex items-center gap-1 text-mustard">
              <Calendar size={12} /> {$t('sudoku.hud_daily')}
            </span>
          {/if}
          <span>{$t('sudoku.hud_progress', { n: progress })}</span>
          {#if game.hints}
            <span>
              {game.hints === 1 ? $t('sudoku.hud_hints_one') : $t('sudoku.hud_hints', { n: game.hints })}
            </span>
          {/if}
        </div>
      </div>

      {#if settings.timer}
        <div class="flex flex-none items-center gap-1.5 font-display text-[20px] tabular-nums">
          <Timer size={16} class="text-mustard" />
          {fmtTime(game.elapsed)}
        </div>
      {/if}
      <button
        type="button"
        onclick={() => (paused = !paused)}
        aria-label={$t('sudoku.pause')}
        class="flex h-9 w-9 flex-none items-center justify-center rounded-lg text-white/75 transition-colors hover:bg-white/10 hover:text-white"
      >
        <Pause size={18} />
      </button>
    </header>

    <!-- Tabuleiro, ferramentas e teclado: um bloco só, centrado na vertical -->
    <div
      class="relative z-10 flex min-h-0 flex-1 flex-col items-center justify-center gap-3 px-3"
      style="padding-bottom:calc(env(safe-area-inset-bottom) + 8px)"
    >
      {#if building}
        <p class="text-center text-[14px] font-semibold text-white/70">{$t('sudoku.building')}</p>
      {:else}
        <div
          role="application"
          aria-label={$t('sudoku.title')}
          class="board aspect-square w-full max-w-[min(92vw,460px)] touch-none select-none"
          style="--line:{theme.line};--edge:{theme.edge};background:{theme.paper}"
        >
          {#each CELL_LIST as at (at)}
            {@const digit = game.values[at]}
            <button
              type="button"
              onclick={() => tapCell(at)}
              aria-label={$t('sudoku.cell_label', { row: rowOf(at) + 1, col: colOf(at) + 1 })}
              class="cell"
              class:edge-right={colOf(at) % 3 === 2 && colOf(at) !== 8}
              class:edge-bottom={rowOf(at) % 3 === 2 && rowOf(at) !== 8}
              class:given={!!game.givens[at]}
              class:hinted={hinted === at}
              style="background:{cellBackground(at)};color:{cellInk(at)}"
            >
              {#if digit}
                <span class="digit">{digit}</span>
              {:else if game.notes[at]}
                <span class="notes" style="color:{theme.note}">
                  {#each DIGITS as note (note)}
                    <i>{digitsOf(game.notes[at]).includes(note) ? note : ''}</i>
                  {/each}
                </span>
              {/if}
            </button>
          {/each}
        </div>
      {/if}

      <div class="flex w-full max-w-[min(92vw,460px)] flex-none flex-col gap-2">
        <div class="grid grid-cols-4 gap-2">
          {#each [{ key: 'undo', label: $t('sudoku.tool_undo'), icon: Undo2, on: false, act: doUndo, off: !game.undo.length }, { key: 'pencil', label: $t('sudoku.tool_pencil'), icon: Pencil, on: pencil, act: () => (pencil = !pencil), off: false }, { key: 'erase', label: $t('sudoku.tool_erase'), icon: Eraser, on: false, act: doErase, off: selected === null }, { key: 'hint', label: $t('sudoku.tool_hint'), icon: Lightbulb, on: false, act: doHint, off: false }] as tool (tool.key)}
            <button
              type="button"
              onclick={tool.act}
              disabled={tool.off || paused}
              aria-pressed={tool.key === 'pencil' ? pencil : undefined}
              class="flex h-[52px] flex-col items-center justify-center gap-0.5 rounded-xl border text-[10.5px] font-bold transition-colors disabled:opacity-35
                {tool.on
                ? 'border-mustard bg-mustard/15 text-mustard'
                : 'border-white/12 bg-white/[.05] text-white/80 hover:bg-white/10'}"
            >
              <tool.icon size={19} />
              {tool.label}
            </button>
          {/each}
        </div>

        <div class="grid grid-cols-9 gap-[5px]">
          {#each DIGITS as digit (digit)}
            <button
              type="button"
              onclick={() => tapDigit(digit)}
              disabled={paused || left[digit] === 0}
              class="relative flex h-[58px] flex-col items-center justify-center rounded-xl border font-display text-[26px] leading-none transition-colors disabled:opacity-30
                {chosen === digit
                ? 'border-mustard bg-mustard text-mustard-ink'
                : 'border-white/12 bg-white/[.06] text-white hover:bg-white/12'}"
            >
              {digit}
              <span
                class="mt-0.5 text-[9.5px] font-bold tabular-nums {chosen === digit
                  ? 'text-mustard-ink/70'
                  : 'text-white/45'}"
              >
                {left[digit]}
              </span>
            </button>
          {/each}
        </div>
      </div>
    </div>

    {#if toast}
      <p class="toast">{toast}</p>
    {/if}

    <!-- Pausa -->
    {#if paused && !result}
      <div class="overlay">
        <div class="card flex flex-col gap-3 text-center">
          <h2 class="font-display text-[30px] leading-none">{$t('sudoku.paused')}</h2>
          <p class="text-sm font-semibold text-paper-muted">
            {$t(difficultyKey(game.difficulty))} · {fmtTime(game.elapsed)}
          </p>
          <button class="btn btn-primary" onclick={() => (paused = false)}>
            {$t('sudoku.resume_game')}
          </button>
          <button class="btn btn-ghost" onclick={quitGame}>{$t('sudoku.quit')}</button>
        </div>
      </div>
    {/if}

    <!-- Fim de partida -->
    {#if result}
      <div class="overlay">
        <div class="card flex flex-col gap-3">
          <div class="text-center">
            <div class="text-[11px] font-bold uppercase tracking-[.2em] text-[#2f8f45]">
              {$t('sudoku.over_kicker')}
            </div>
            <h2 class="font-display text-[34px] leading-none">{$t('sudoku.over_title')}</h2>
          </div>

          {#if result.record}
            <p class="tag mx-auto bg-mustard text-mustard-ink">{$t('sudoku.over_record')}</p>
          {/if}
          {#if result.daily}
            <p class="text-center text-sm font-bold text-[#2f8f45]">{$t('sudoku.over_daily_done')}</p>
          {/if}

          <!-- Partida atual × De sempre -->
          <table class="w-full text-left">
            <thead>
              <tr class="stat-label">
                <th class="font-bold"></th>
                <th class="pb-1 text-right font-bold">{$t('sudoku.col_current')}</th>
                <th class="pb-1 text-right font-bold">{$t('sudoku.col_alltime')}</th>
              </tr>
            </thead>
            <tbody>
              {#each [{ key: 'time', label: $t('sudoku.row_time'), now: fmtTime(result.time), ever: result.hadWin ? fmtTime(result.bestTime) : '—' }, { key: 'hints', label: $t('sudoku.row_hints'), now: String(result.hints), ever: result.hadWin ? String(result.bestHints) : '—' }, { key: 'mistakes', label: $t('sudoku.row_mistakes'), now: String(result.mistakes), ever: result.hadWin ? String(result.bestMistakes) : '—' }] as row (row.key)}
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
              {$t('sudoku.over_unlocked', {
                theme: result.unlocked.map((key) => $t(themeOf(key).nameKey)).join(', '),
              })}
            </p>
          {/if}

          <button
            class="btn btn-primary"
            onclick={() =>
              result?.daily ? startDaily(result.daily) : startGame(result!.difficulty)}
          >
            {$t('sudoku.play_again')}
          </button>
          <button class="btn btn-ghost" onclick={quitGame}>{$t('sudoku.quit')}</button>
        </div>
      </div>
    {/if}
  {:else}
    <!-- ── Abas ─────────────────────────────────────────────────────────── -->
    <div
      class="pointer-events-none absolute inset-x-0 top-0 h-[320px]"
      style="background:radial-gradient(90% 55% at 50% 0%,#3a2d23 0%,#241c16 45%,transparent 75%)"
    ></div>
    <div
      class="pointer-events-none absolute inset-0"
      style="background:linear-gradient(180deg,rgba(23,19,16,.35) 0%,rgba(23,19,16,.9) 45%,#171310 72%)"
    ></div>

    <div
      class="relative z-10 flex-none px-3 pb-1"
      style="padding-top:calc(env(safe-area-inset-top) + 8px)"
    >
      <a
        href="/"
        class="inline-flex items-center gap-1.5 rounded-lg px-2 py-1.5 text-[13px] font-semibold text-white/75 transition-colors hover:bg-white/10 hover:text-white"
      >
        <ChevronLeft size={16} strokeWidth={2.2} /> {$t('sudoku.back_to_games')}
      </a>
    </div>

    <div class="relative z-10 min-h-0 flex-1 overflow-y-auto">
      <div class="mx-auto max-w-md">
        {#if screen === 'home'}
          <div class="flex flex-col items-center px-[22px] pb-[26px] pt-[10px] text-center">
            <div class="drop-shadow-[0_16px_30px_rgba(255,201,60,.18)]">
              <Emblem size={124} />
            </div>
            <div
              class="mt-1 bg-[linear-gradient(180deg,#fbf3df_0%,#e8dcc0_52%,#c9b99a_100%)] bg-clip-text pt-1 font-display text-[44px] leading-none tracking-[.01em] text-transparent"
            >
              {$t('sudoku.title')}
            </div>

            {#if saved.game}
              <button
                type="button"
                onclick={resumeGame}
                class="mt-4 flex h-[54px] w-full items-center justify-center gap-2.5 rounded-xl bg-mustard font-display text-[19px] text-mustard-ink shadow-[0_8px_20px_rgba(255,201,60,.3)] transition-colors hover:bg-[#ffd666] active:scale-[.98]"
              >
                <Play size={20} fill="currentColor" />
                {$t('sudoku.resume')} · {$t(difficultyKey(saved.game.difficulty))} · {fmtTime(
                  saved.game.elapsed,
                )}
              </button>
            {/if}

            <!-- Dificuldades -->
            <div class="mt-4 w-full text-left">
              <div class="section-label !mt-0 !text-white/45">{$t('sudoku.pick_difficulty')}</div>
              <ul class="flex flex-col gap-1.5">
                {#each DIFFICULTIES as difficulty (difficulty)}
                  {@const stat = stats[difficulty]}
                  <li>
                    <button
                      type="button"
                      onclick={() => startGame(difficulty)}
                      class="flex w-full items-center gap-3 rounded-xl border border-white/10 bg-white/[.04] px-3.5 py-2.5 text-left transition-colors hover:border-mustard/40 hover:bg-white/[.07]"
                    >
                      <span class="min-w-0 flex-1">
                        <span class="block font-display text-[19px] leading-tight">
                          {$t(difficultyKey(difficulty))}
                        </span>
                        <span class="block text-[11.5px] font-medium text-white/55">
                          {stat.wins
                            ? $t('sudoku.diff_summary', { wins: stat.wins })
                            : $t('sudoku.diff_never')}
                        </span>
                      </span>
                      <span class="flex-none text-right">
                        <span class="block font-display text-[18px] tabular-nums text-mustard">
                          {stat.wins ? fmtTime(stat.best) : '—'}
                        </span>
                        <span class="stat-label !text-white/40">{$t('sudoku.best_time')}</span>
                      </span>
                    </button>
                  </li>
                {/each}
              </ul>
            </div>

            <!-- Tabuleiro do dia -->
            <div
              class="relative mt-4 flex w-full items-center gap-3.5 overflow-hidden rounded-[14px] border border-mustard/35 bg-[linear-gradient(158deg,#3a2d23_0%,#241c16_45%,#171310_100%)] py-3.5 pl-4 pr-3.5 text-left"
            >
              <div
                class="relative flex h-[46px] w-[46px] flex-none items-center justify-center rounded-xl border border-mustard/45 bg-black/25 text-mustard"
              >
                <Calendar size={24} />
              </div>
              <div class="relative flex min-w-0 flex-1 flex-col gap-px">
                <span class="text-[10.5px] font-bold uppercase tracking-[.16em] text-mustard">
                  {$t('sudoku.daily_today')}
                </span>
                <span class="truncate font-display text-[19px] leading-tight text-white">
                  {todayLabel} · {$t(difficultyKey(todayDifficulty))}
                </span>
                <span class="text-[12px] font-medium text-white/70">
                  {todayRecord?.done
                    ? $t('sudoku.daily_status_done', { time: fmtTime(todayRecord.time) })
                    : $t('sudoku.daily_status_todo')}
                </span>
              </div>
              <button
                type="button"
                onclick={() => startDaily(todayKey)}
                class="relative h-10 flex-none rounded-[10px] bg-mustard px-4 font-display text-[15px] text-mustard-ink transition-colors hover:bg-[#ffd666] active:scale-95"
              >
                {todayRecord?.done ? $t('sudoku.replay') : $t('sudoku.play')}
              </button>
            </div>

            <div
              class="mt-4 flex flex-wrap justify-center gap-x-[18px] gap-y-1 text-[12px] font-medium text-gray-400"
            >
              <span><b class="font-bold text-white">{wins}</b> {$t('sudoku.summary_wins')}</span>
              <span aria-hidden="true">·</span>
              <span>
                <b class="font-bold text-white">{streak}</b>
                {streak === 1 ? $t('sudoku.summary_streak_one') : $t('sudoku.summary_streak')}
              </span>
            </div>

            {#if history.length}
              <div class="mt-6 w-full text-left">
                <div class="section-label !text-white/45">{$t('sudoku.recent')}</div>
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
                        {$t(difficultyKey(entry.difficulty))}
                        {#if entry.hints}
                          · {entry.hints === 1
                            ? $t('sudoku.entry_hints_one')
                            : $t('sudoku.entry_hints', { n: entry.hints })}
                        {/if}
                      </span>
                      <span class="font-display text-[18px] tabular-nums">{fmtTime(entry.time)}</span>
                    </li>
                  {/each}
                </ul>
              </div>
            {/if}
          </div>
        {:else if screen === 'daily'}
          <div class="px-5 pb-[26px] pt-5">
            <div class="text-[11px] font-bold uppercase tracking-[.2em] text-mustard">
              {$t('sudoku.daily_kicker')}
            </div>
            <div class="mt-0.5 font-display text-[30px] leading-tight">{monthLabel}</div>
            <p class="mt-1.5 text-[13.5px] text-white/70">{$t('sudoku.daily_note')}</p>

            <div class="mt-4 grid grid-cols-3 gap-2">
              {#each [{ key: 'done', value: String(doneDays), label: $t('sudoku.stat_done'), gold: false }, { key: 'streak', value: String(streak), label: $t('sudoku.stat_streak'), gold: true }, { key: 'best', value: dailyRecord ? fmtTime(dailyRecord) : '—', label: $t('sudoku.stat_daily_best'), gold: false }] as box (box.key)}
                <div
                  class="rounded-[14px] border px-2 py-2.5 text-center {box.gold
                    ? 'border-mustard/45 bg-mustard/5'
                    : 'border-white/10 bg-white/[.05]'}"
                >
                  <div
                    class="font-display text-[22px] leading-none tabular-nums {box.gold
                      ? 'text-mustard'
                      : 'text-white'}"
                  >
                    {box.value}
                  </div>
                  <div class="stat-label mt-1 !text-white/45">{box.label}</div>
                </div>
              {/each}
            </div>

            <div class="mt-3.5 rounded-[14px] border border-white/10 bg-white/[.05] px-2.5 py-3">
              <div class="grid grid-cols-7 gap-[5px] text-center">
                {#each weekdays as weekday, i (i)}
                  <span class="pb-1 text-[10px] font-bold tracking-[.08em] text-gray-500">
                    {weekday}
                  </span>
                {/each}
                {#each calendar as cell (cell.key)}
                  {#if cell.day === null}
                    <span class="aspect-square"></span>
                  {:else}
                    {@const ring = cell.today
                      ? 'border-2 border-mustard'
                      : cell.done
                        ? 'border border-[#2f8f45]'
                        : cell.future
                          ? 'border border-transparent'
                          : 'border border-white/12'}
                    <button
                      type="button"
                      onclick={() => startDaily(cell.date)}
                      disabled={cell.future}
                      class="flex aspect-square items-center justify-center rounded-full text-[14px] transition-[filter] hover:brightness-125 disabled:hover:brightness-100 {ring}
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
              onclick={() => startDaily(todayKey)}
              class="mt-3.5 flex h-[54px] w-full items-center justify-center gap-2.5 rounded-xl bg-mustard font-display text-[19px] text-mustard-ink shadow-[0_8px_20px_rgba(255,201,60,.3)] transition-colors hover:bg-[#ffd666] active:scale-[.98]"
            >
              <Play size={20} fill="currentColor" />
              {todayRecord?.done ? $t('sudoku.daily_replay') : $t('sudoku.daily_play')}
            </button>
          </div>
        {:else if screen === 'themes'}
          <div class="px-5 pb-[26px] pt-5">
            <div class="text-[11px] font-bold uppercase tracking-[.2em] text-mustard">
              {$t('sudoku.themes_kicker')}
            </div>
            <div class="mt-0.5 font-display text-[30px] leading-tight">
              {$t('sudoku.themes_title')}
            </div>

            {#if upcoming}
              <div class="mt-3">
                <div class="flex items-baseline justify-between text-[12.5px] font-semibold">
                  <span class="text-white/70">
                    {$t('sudoku.themes_next', { theme: $t(upcoming.nameKey) })}
                  </span>
                  <span class="tabular-nums text-mustard">{wins}/{upcoming.need}</span>
                </div>
                <div class="mt-1.5 h-2 overflow-hidden rounded-full bg-white/10">
                  <div
                    class="h-full rounded-full bg-mustard transition-[width]"
                    style="width:{Math.min(100, Math.round((wins / upcoming.need) * 100))}%"
                  ></div>
                </div>
              </div>
            {:else}
              <p class="mt-3 text-[13.5px] text-white/70">{$t('sudoku.themes_all')}</p>
            {/if}

            <ul class="mt-4 grid grid-cols-2 gap-2.5">
              {#each THEMES as item (item.key)}
                {@const open = isUnlocked(item.key, wins)}
                <li
                  class="overflow-hidden rounded-[14px] border {settings.theme === item.key
                    ? 'border-mustard'
                    : 'border-white/12'}"
                >
                  <!-- Amostra do tema: quatro células com um dado e um número do jogador -->
                  <div class="grid grid-cols-2 gap-px p-px" style="background:{item.edge}">
                    {#each [{ bg: item.paper, ink: item.given, n: '5' }, { bg: item.paperAlt, ink: item.pen, n: '9' }, { bg: item.sel, ink: item.pen, n: '3' }, { bg: item.paper, ink: item.note, n: '7' }] as sample, i (i)}
                      <span
                        class="flex aspect-square items-center justify-center font-display text-[17px]"
                        style="background:{sample.bg};color:{sample.ink}"
                      >
                        {sample.n}
                      </span>
                    {/each}
                  </div>
                  <div class="flex items-center justify-between gap-1.5 bg-white/[.04] px-2.5 py-2">
                    <span class="truncate text-[13px] font-bold">{$t(item.nameKey)}</span>
                    {#if !open}
                      <span class="flex flex-none items-center gap-1 text-[11px] font-bold text-white/45">
                        <Lock size={11} /> {$t('sudoku.themes_need', { n: item.need })}
                      </span>
                    {:else if settings.theme === item.key}
                      <span class="flex-none text-[11px] font-bold text-mustard">
                        {$t('sudoku.themes_in_use')}
                      </span>
                    {:else}
                      <button
                        type="button"
                        onclick={() => update('theme', item.key)}
                        class="flex-none rounded-lg bg-mustard px-2 py-1 text-[11px] font-bold text-mustard-ink transition-colors hover:bg-[#ffd666]"
                      >
                        {$t('sudoku.themes_use')}
                      </button>
                    {/if}
                  </div>
                </li>
              {/each}
            </ul>
          </div>
        {:else}
          <div class="px-5 pb-[26px] pt-5">
            <div class="text-[11px] font-bold uppercase tracking-[.2em] text-mustard">
              {$t('sudoku.settings_kicker')}
            </div>
            <div class="mt-0.5 font-display text-[30px] leading-tight">
              {$t('sudoku.settings_title')}
            </div>

            <div class="mt-4 flex flex-col gap-2.5">
              {#each [{ key: 'errors' as const, label: $t('sudoku.setting_errors'), note: $t('sudoku.setting_errors_note') }, { key: 'highlight' as const, label: $t('sudoku.setting_highlight'), note: $t('sudoku.setting_highlight_note') }, { key: 'autoNotes' as const, label: $t('sudoku.setting_auto_notes'), note: $t('sudoku.setting_auto_notes_note') }, { key: 'numberFirst' as const, label: $t('sudoku.setting_number_first'), note: $t('sudoku.setting_number_first_note') }, { key: 'timer' as const, label: $t('sudoku.setting_timer'), note: $t('sudoku.setting_timer_note') }] as item (item.key)}
                <label
                  class="flex cursor-pointer items-start gap-3 rounded-xl border border-white/10 bg-white/[.04] px-3.5 py-3"
                >
                  <input
                    type="checkbox"
                    checked={settings[item.key]}
                    onchange={(event) => update(item.key, event.currentTarget.checked)}
                    class="mt-0.5 h-5 w-5 flex-none accent-mustard"
                  />
                  <span class="min-w-0 flex-1">
                    <span class="block text-[14px] font-bold">{item.label}</span>
                    <span class="block text-[12px] text-white/55">{item.note}</span>
                  </span>
                </label>
              {/each}
            </div>

            <div class="section-label !text-white/45">{$t('common.language')}</div>
            <LanguageSwitcher tone="dark" />

            <div class="section-label !text-white/45">{$t('sudoku.settings_data')}</div>
            <button
              type="button"
              onclick={() => (confirmReset = true)}
              class="w-full rounded-xl border border-ketchup/40 bg-ketchup/10 px-3.5 py-3 text-[14px] font-bold text-ketchup transition-colors hover:bg-ketchup/20"
            >
              {$t('sudoku.reset_stats')}
            </button>
            <p class="mt-1.5 text-[12px] text-white/45">{$t('sudoku.reset_note')}</p>
          </div>
        {/if}
      </div>
    </div>

    {#if toast}
      <p class="toast">{toast}</p>
    {/if}

    <nav
      class="relative z-10 flex-none border-t border-white/10 bg-news-2"
      style="padding-bottom:calc(env(safe-area-inset-bottom) + 6px)"
    >
      <div class="mx-auto grid max-w-lg grid-cols-4">
        {#each TABS as item (item.key)}
          <button
            type="button"
            onclick={() => go(item.key)}
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
    message={$t('sudoku.reset_confirm')}
    confirmLabel={$t('sudoku.reset_stats')}
    onConfirm={resetStats}
  />
</div>


<style>
  .sudoku {
    /* As fontes de display não têm peso negrito — sem isso o navegador inventa um. */
    font-synthesis: none;
  }

  .board {
    display: grid;
    grid-template-columns: repeat(9, 1fr);
    border: 3px solid var(--edge);
    border-radius: 8px;
    overflow: hidden;
    box-shadow: 0 18px 40px rgba(0, 0, 0, 0.45);
  }

  .cell {
    position: relative;
    display: flex;
    align-items: center;
    justify-content: center;
    border-right: 1px solid var(--line);
    border-bottom: 1px solid var(--line);
    font-family: theme('fontFamily.display');
    font-size: clamp(16px, 4.6vw, 26px);
    line-height: 1;
    transition: background-color 0.12s;
  }

  /* A última coluna e a última linha não precisam de traço: a moldura já fecha. */
  .cell:nth-child(9n) {
    border-right: 0;
  }
  .cell:nth-child(n + 73) {
    border-bottom: 0;
  }
  .cell.edge-right {
    border-right: 2px solid var(--edge);
  }
  .cell.edge-bottom {
    border-bottom: 2px solid var(--edge);
  }

  /* O número do jogador é um tico menor que o impresso, como no papel. */
  .cell:not(.given) .digit {
    font-size: 0.92em;
  }

  .cell.hinted::after {
    content: '';
    position: absolute;
    inset: 2px;
    border-radius: 4px;
    outline: 2px solid theme('colors.mustard.DEFAULT');
    animation: blink 1.1s ease-out 2;
  }

  @keyframes blink {
    0%,
    100% {
      opacity: 0;
    }
    35% {
      opacity: 1;
    }
  }

  .notes {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    width: 100%;
    height: 100%;
    padding: 1px;
    font-family: theme('fontFamily.sans');
    font-size: clamp(7px, 1.7vw, 10px);
    font-weight: 700;
    line-height: 1;
  }

  .notes i {
    display: flex;
    align-items: center;
    justify-content: center;
    font-style: normal;
  }
</style>
