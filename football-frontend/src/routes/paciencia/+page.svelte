<script lang="ts">
  /**
   * Paciência Klondike — Rachão Edition (handoff design_handoff_paciencia_games).
   *
   * Rota em tela cheia (`fixed inset-0 z-50`), sem Navbar, mobile-first.
   * Navegação interna por abas (Início · Diário · Campanha · Ajustes); o jogo
   * abre por cima das abas. Toda a lógica de regras vive em
   * `$lib/paciencia/engine.ts` e a persistência em `$lib/paciencia/storage.ts`.
   *
   * A mesa usa geometria proporcional: a carta mede 48×68 na referência de
   * 390px de largura e escala com o container (`cardW`), então todas as medidas
   * internas passam por `s()`.
   */
  import '@fontsource/bebas-neue/400.css';
  import { goto } from '$app/navigation';
  import {
    Calendar,
    CheckCheck,
    ChevronLeft,
    House,
    Lightbulb,
    Play,
    Plus,
    RefreshCcw,
    RotateCcw,
    Settings as SettingsIcon,
    Trophy,
    Undo2,
  } from 'lucide-svelte';
  import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
  import CrestShell from '$lib/components/CrestShell.svelte';
  import { locale, t } from '$lib/i18n';
  import { CARD_BACKS, CARD_BACK_KEYS, type CardBackKey } from '$lib/paciencia/backs';
  import {
    AUTO_FINISH_STEP_MS,
    RANKS,
    SUITS,
    UNDO_LIMIT,
    allUp,
    autoFinishStep,
    cloneGame,
    dateKey,
    deal,
    drawFromStock,
    findHint,
    flipExposed,
    fmtTime,
    isRed,
    isWon,
    resolveTap,
    winStars,
    type Card,
    type DrawMode,
    type Game,
    type HintTarget,
    type MoveSource,
  } from '$lib/paciencia/engine';
  import {
    DEFAULT_SETTINGS,
    HISTORY_VISIBLE,
    loadState,
    recordMatch,
    saveState,
    type DailyRecord,
    type HistoryEntry,
    type Settings,
  } from '$lib/paciencia/storage';

  type Screen = 'home' | 'daily' | 'stats' | 'settings' | 'game';

  // ── Estado ────────────────────────────────────────────────────────────────
  let screen = $state<Screen>('home');
  let game = $state<Game | null>(null);
  let history = $state<HistoryEntry[]>([]);
  let dailyDone = $state<Record<string, DailyRecord>>({});
  let settings = $state<Settings>({ ...DEFAULT_SETTINGS });
  let undoStack = $state<Game[]>([]);
  let shakeId = $state<number | null>(null);
  let hintIds = $state<HintTarget[]>([]);
  let autoRunning = $state(false);
  let loaded = $state(false);
  let confirmReset = $state(false);
  let today = $state(new Date());

  let shakeTimer: ReturnType<typeof setTimeout> | undefined;
  let hintTimer: ReturnType<typeof setTimeout> | undefined;
  let autoTimer: ReturnType<typeof setInterval> | undefined;

  // Hidrata o estado salvo. Sem leituras reativas, roda uma única vez.
  // (`$effect` e não `onMount`: a build de produção do projeto remove as
  // callbacks de `onMount`, e a convenção do repo já prefere `$effect`.)
  $effect(() => {
    const saved = loadState();
    game = saved.game;
    history = saved.history;
    dailyDone = saved.daily;
    settings = saved.settings;
    loaded = true;
  });

  // Cronômetro: conta por segundo e pausa fora da tela do jogo.
  $effect(() => {
    const tick = setInterval(() => {
      const now = new Date();
      if (dateKey(now) !== todayKey) today = now;
      if (screen === 'game' && game && !game.won && game.moves > 0) game.elapsed += 1;
    }, 1000);
    return () => {
      clearInterval(tick);
      clearTimeout(shakeTimer);
      clearTimeout(hintTimer);
      stopAutoFinish();
    };
  });

  // Persistência: partida atual, histórico, desafios diários e ajustes.
  $effect(() => {
    if (!loaded) return;
    saveState({ game, history, daily: dailyDone, settings });
  });

  // ── Ações ─────────────────────────────────────────────────────────────────

  /**
   * Aplica uma mutação ao jogo sobre uma cópia, vira as cartas expostas,
   * detecta a vitória e empilha o estado anterior para o desfazer.
   */
  function commit(mutate: (g: Game) => boolean, noUndo = false): boolean {
    if (!game) return false;
    const prev = cloneGame(game);
    const next = cloneGame(game);
    if (!mutate(next)) return false;

    flipExposed(next);
    next.score = Math.max(0, next.score);
    const justWon = !next.won && isWon(next);
    if (justWon) next.won = true;

    if (!noUndo) undoStack = [...undoStack.slice(-UNDO_LIMIT), prev];
    game = next;
    hintIds = [];

    if (justWon) {
      stopAutoFinish();
      history = recordMatch(history, next, true);
      if (next.daily) {
        const best = dailyDone[next.daily];
        dailyDone = {
          ...dailyDone,
          [next.daily]: {
            time: best ? Math.min(best.time, next.elapsed) : next.elapsed,
            moves: next.moves,
          },
        };
      }
    } else if (!autoRunning && settings.autoFinish && allUp(next)) {
      setTimeout(startAutoFinish, 250);
    }
    return true;
  }

  /** Abandonar uma partida com movimentos registra "Abandonou" no histórico. */
  function startGame(daily: string | null) {
    stopAutoFinish();
    if (game && !game.won && game.moves > 0) history = recordMatch(history, game, false);
    game = deal(settings.draw, daily);
    undoStack = [];
    hintIds = [];
    screen = 'game';
  }

  function go(next: Screen) {
    stopAutoFinish();
    screen = next;
  }

  function onDraw() {
    if (!game || game.won) return;
    commit(drawFromStock);
  }

  function onTap(src: MoveSource, col = 0, idx = 0) {
    if (!game || game.won || autoRunning) return;
    let touched: number | null = null;
    const moved = commit((g) => {
      const res = resolveTap(g, src, col, idx);
      touched = res.cardId;
      return res.moved;
    });
    if (!moved && touched !== null) {
      shakeId = touched;
      clearTimeout(shakeTimer);
      shakeTimer = setTimeout(() => (shakeId = null), 280);
    }
  }

  function onHint() {
    if (!game || game.won) return;
    hintIds = findHint(game);
    clearTimeout(hintTimer);
    hintTimer = setTimeout(() => (hintIds = []), 1400);
  }

  function onUndo() {
    if (!game || game.won || !undoStack.length) return;
    const prev = undoStack[undoStack.length - 1];
    game = {
      ...cloneGame(prev),
      elapsed: game.elapsed,
      moves: game.moves + 1,
      score: Math.max(0, prev.score - 2),
    };
    undoStack = undoStack.slice(0, -1);
    hintIds = [];
  }

  function startAutoFinish() {
    if (!game || game.won || !allUp(game)) return;
    stopAutoFinish();
    autoRunning = true;
    autoTimer = setInterval(() => {
      const moved = commit(autoFinishStep, true);
      if (!moved || game?.won) stopAutoFinish();
    }, AUTO_FINISH_STEP_MS);
  }

  function stopAutoFinish() {
    clearInterval(autoTimer);
    autoRunning = false;
  }

  function resetStats() {
    history = [];
    dailyDone = {};
  }

  function onKeydown(e: KeyboardEvent) {
    if (e.key !== 'Escape') return;
    if (inGame) go('home');
    else goto('/games');
  }

  // ── Derivados ─────────────────────────────────────────────────────────────
  let inGame = $derived(screen === 'game' && !!game);
  let todayKey = $derived(dateKey(today));
  let back = $derived(CARD_BACKS[settings.back] ?? CARD_BACKS.escudo);
  let canUndo = $derived(!!undoStack.length && !!game && !game.won);
  let canAutoFinish = $derived(!!game && !game.won && allUp(game));
  let hasRunningGame = $derived(!!game && !game.won);

  let wins = $derived(history.filter((h) => h.won));
  let winStreak = $derived.by(() => {
    let n = 0;
    for (const h of history) {
      if (!h.won) break;
      n++;
    }
    return n;
  });

  let dailyKeys = $derived(Object.keys(dailyDone));
  let dailyStreak = $derived.by(() => {
    const d = new Date(today);
    if (!dailyDone[dateKey(d)]) d.setDate(d.getDate() - 1);
    let n = 0;
    while (dailyDone[dateKey(d)]) {
      n++;
      d.setDate(d.getDate() - 1);
    }
    return n;
  });
  let dailyBest = $derived(
    dailyKeys.length ? Math.min(...dailyKeys.map((k) => dailyDone[k].time)) : null
  );
  let todayRecord = $derived(dailyDone[todayKey]);

  let calendar = $derived.by(() => {
    const y = today.getFullYear();
    const m = today.getMonth();
    const startDow = (new Date(y, m, 1).getDay() + 6) % 7; // semana começa na segunda
    const total = new Date(y, m + 1, 0).getDate();
    const cells: { key: string; day: number | null; date: string; today: boolean; done: boolean; future: boolean }[] = [];
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
        done: !!dailyDone[key],
        future: key > todayKey,
      });
    }
    return cells;
  });

  let weekdays = $derived($t('paciencia.weekdays').split(','));

  let statCells = $derived([
    { key: 'matches', label: $t('paciencia.stat_matches'), value: String(history.length), highlight: false },
    { key: 'wins', label: $t('paciencia.stat_wins'), value: String(wins.length), highlight: true },
    {
      key: 'rate',
      label: $t('paciencia.stat_rate'),
      value: history.length ? `${Math.round((wins.length / history.length) * 100)}%` : '—',
      highlight: false,
    },
    {
      key: 'best',
      label: $t('paciencia.stat_best_time'),
      value: fmtTime(wins.length ? Math.min(...wins.map((w) => w.time)) : null),
      highlight: false,
    },
    { key: 'streak', label: $t('paciencia.stat_streak'), value: String(winStreak), highlight: false },
    {
      key: 'record',
      label: $t('paciencia.stat_record'),
      value: wins.length ? String(Math.max(...wins.map((w) => w.score))) : '—',
      highlight: false,
    },
  ]);

  // ── Datas por extenso (locale do app) ─────────────────────────────────────
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
  let gameTitle = $derived.by(() => {
    if (!game?.daily) return $t('paciencia.title');
    const d = new Date(`${game.daily}T12:00`);
    return $t('paciencia.challenge_title', {
      date: d.toLocaleDateString($locale, { day: '2-digit', month: '2-digit' }),
    });
  });

  // ── Geometria da mesa ─────────────────────────────────────────────────────
  const PAD = 12;
  const GAP = 5;
  const BASE_W = 48;
  const BASE_H = 68;
  const MIN_CARD_W = 34;
  const MAX_CARD_W = 72;

  let mesaW = $state(0);
  let mesaH = $state(0);

  let cardW = $derived(
    Math.max(MIN_CARD_W, Math.min(MAX_CARD_W, (mesaW - 2 * PAD - 6 * GAP) / 7))
  );
  let cardH = $derived((cardW * BASE_H) / BASE_W);
  let boardW = $derived(7 * cardW + 6 * GAP);
  let originX = $derived(Math.max(PAD, (mesaW - boardW) / 2));
  let topY = $derived(s(14));
  let tabTop = $derived(topY + cardH + s(14));

  /** Converte uma medida da referência (carta 48px) para a escala atual. */
  function s(v: number): number {
    return (v * cardW) / BASE_W;
  }
  function colX(i: number): number {
    return originX + i * (cardW + GAP);
  }

  type Slot = {
    key: string;
    card: Card;
    x: number;
    y: number;
    faceUp: boolean;
    action: (() => void) | null;
  };

  type Placeholder = {
    key: string;
    x: number;
    y: number;
    glyph: string;
    size: number;
    dashed: boolean;
    color: string;
    bg: string;
  };

  let placeholders = $derived.by<Placeholder[]>(() => {
    if (!game) return [];
    const out: Placeholder[] = [];
    for (let suit = 0; suit < 4; suit++) {
      out.push({
        key: `f${suit}`,
        x: colX(3 + suit),
        y: topY,
        glyph: SUITS[suit],
        size: s(26),
        dashed: false,
        color: 'rgba(245,213,133,.6)',
        bg: 'rgba(7,51,26,.35)',
      });
    }
    out.push({
      key: 'stock',
      x: colX(0),
      y: topY,
      glyph: '',
      size: s(15),
      dashed: true,
      color: 'rgba(255,255,255,.4)',
      bg: 'rgba(7,51,26,.25)',
    });
    for (let i = 0; i < 7; i++) {
      out.push({
        key: `t${i}`,
        x: colX(i),
        y: tabTop,
        glyph: 'K',
        size: s(20),
        dashed: true,
        color: 'rgba(255,255,255,.35)',
        bg: 'rgba(7,51,26,.2)',
      });
    }
    return out;
  });

  let slots = $derived.by<Slot[]>(() => {
    if (!game) return [];
    const g = game;
    const out: Slot[] = [];

    // Fundações: mostra a penúltima carta para dar volume à pilha.
    g.found.forEach((pile, suit) => {
      const x = colX(3 + suit);
      if (pile.length > 1) {
        out.push({ key: `fb${suit}`, card: pile[pile.length - 2], x, y: topY, faceUp: true, action: null });
      }
      if (pile.length) {
        const card = pile[pile.length - 1];
        out.push({ key: String(card.id), card, x, y: topY, faceUp: true, action: () => onTap('f', suit) });
      }
    });

    // Descarte: na compra de 3, as cartas visíveis ficam deslocadas.
    const visible = g.waste.slice(-(g.draw === 3 ? 3 : 1));
    visible.forEach((card, i) => {
      const isTop = i === visible.length - 1;
      out.push({
        key: String(card.id),
        card,
        x: colX(1) + i * s(15),
        y: topY,
        faceUp: true,
        action: isTop ? () => onTap('w') : null,
      });
    });

    // Tableau: cartas viradas para baixo avançam 10px, viradas para cima 24px,
    // comprimindo quando a coluna não cabe na altura disponível.
    const avail = mesaH - s(10) - tabTop - cardH;
    g.tableau.forEach((col, ci) => {
      const downs = col.filter((c) => !c.up).length;
      const ups = col.length - downs;
      const upStep =
        ups > 1 ? Math.min(s(24), (avail - downs * s(10)) / (ups - 1)) : s(24);
      let y = tabTop;
      col.forEach((card, i) => {
        out.push({
          key: String(card.id),
          card,
          x: colX(ci),
          y,
          faceUp: card.up,
          action: card.up ? () => onTap('t', ci, i) : null,
        });
        y += card.up ? upStep : s(10);
      });
    });

    return out;
  });

  let suitLabels = $derived([
    $t('paciencia.suit_spades'),
    $t('paciencia.suit_hearts'),
    $t('paciencia.suit_diamonds'),
    $t('paciencia.suit_clubs'),
  ]);
  const cardLabel = (c: Card) => `${RANKS[c.r]} ${suitLabels[c.s]}`;

  const hinted = (c: Card) => hintIds.includes(c.id);
  const shaking = (c: Card) => shakeId === c.id;
</script>

<svelte:head>
  <title>{$t('paciencia.title')} — rachao.app</title>
</svelte:head>

<svelte:window onkeydown={onKeydown} />

{#snippet cardFace(card: Card)}
  <div
    class="absolute inset-0 overflow-hidden bg-[linear-gradient(180deg,#ffffff,#f3f4f6)]"
    style="
      border-radius:{s(6)}px;
      border:{hinted(card) ? '2px solid #f5d585' : shaking(card) ? '2px solid #ef4444' : '1px solid #d1d5db'};
      box-shadow:{hinted(card)
      ? '0 0 0 3px rgba(245,213,133,.45),0 6px 14px rgba(0,0,0,.4)'
      : '0 1px 3px rgba(0,0,0,.35)'};
      color:{isRed(card) ? '#dc2626' : '#111827'};"
  >
    <div
      class="absolute flex flex-col items-center leading-[.85]"
      style="top:{s(2)}px;left:{s(4)}px"
    >
      <span class="font-bebas" style="font-size:{s(19)}px">{RANKS[card.r]}</span>
      <span style="font-size:{s(11)}px">{SUITS[card.s]}</span>
    </div>
    <div class="absolute leading-none" style="right:{s(3)}px;bottom:{s(2)}px;font-size:{s(24)}px">
      {SUITS[card.s]}
    </div>
  </div>
{/snippet}

{#snippet cardBack(shadow: string)}
  <div
    class="absolute inset-0 overflow-hidden"
    style="border-radius:{s(6)}px;border:1.5px solid {back.edge};background:{back.outer};box-shadow:{shadow}"
  >
    <div
      class="absolute"
      style="inset:{s(3)}px;border:1px solid {back.line};border-radius:{s(4)}px;background:{back.inner}"
    ></div>
    {#if back.logo > 0}
      <img
        src="/logo.png"
        alt=""
        class="absolute left-1/2 top-1/2 object-cover"
        style="width:{s(32)}px;height:{s(32)}px;margin:{-s(16)}px 0 0 {-s(16)}px;opacity:{back.logo}"
      />
    {/if}
  </div>
{/snippet}

<div
  class="paciencia fixed inset-0 z-50 flex flex-col overflow-hidden bg-gray-900 text-gray-50"
>
  {#if inGame && game}
    <!-- ── Cabeçalho do jogo ────────────────────────────────────────────── -->
    <header
      class="relative z-10 flex-none border-b-2 border-gold-edge bg-[linear-gradient(180deg,#07331a,#0d5028)] px-2.5 pb-2 pt-0.5"
      style="padding-top:calc(env(safe-area-inset-top) + 2px)"
    >
      <div class="mx-auto grid h-[38px] max-w-lg grid-cols-[44px_1fr_44px] items-center">
        <button
          type="button"
          onclick={() => go('home')}
          aria-label={$t('paciencia.aria_back_home')}
          class="flex h-9 w-9 items-center justify-center rounded-[10px] border-[1.5px] border-white/20 bg-white/[.06] text-white transition-colors hover:bg-white/15"
        >
          <ChevronLeft size={18} strokeWidth={2.2} />
        </button>
        <div class="flex items-center justify-center gap-[7px]">
          <img src="/logo.png" alt="" class="h-[30px] w-[30px] object-cover" />
          <span class="whitespace-nowrap font-bebas text-[21px] uppercase tracking-[.04em]">
            {gameTitle}
          </span>
        </div>
        <span
          class="justify-self-end rounded-md bg-black/30 px-[7px] py-[3px] text-[11px] font-bold text-gold-400"
        >
          ×{game.draw}
        </span>
      </div>
      <div class="mx-auto mt-1 grid max-w-lg grid-cols-3 gap-1.5">
        {#each [{ label: $t('paciencia.hud_time'), value: settings.timer ? fmtTime(game.elapsed) : '—' }, { label: $t('paciencia.hud_moves'), value: String(game.moves) }, { label: $t('paciencia.hud_score'), value: String(game.score) }] as box (box.label)}
          <div class="rounded-lg border border-gold-400/[.22] bg-black/30 py-[3px] text-center">
            <div class="text-[8.5px] font-bold uppercase tracking-[.18em] text-gold-400">
              {box.label}
            </div>
            <div class="font-bebas text-[20px] leading-[1.05] tabular-nums">{box.value}</div>
          </div>
        {/each}
      </div>
    </header>

    <!-- ── Mesa ─────────────────────────────────────────────────────────── -->
    <div
      class="relative min-h-0 flex-1 overflow-hidden"
      style="background:repeating-linear-gradient(180deg,#168641 0 {s(58)}px,#13793b {s(58)}px {s(116)}px)"
      bind:clientWidth={mesaW}
      bind:clientHeight={mesaH}
    >
      <!-- Linhas do campo -->
      <div
        class="pointer-events-none absolute inset-0"
        style="background:radial-gradient(120% 80% at 50% 45%,transparent 50%,rgba(7,51,26,.55) 100%)"
      ></div>
      <div
        class="pointer-events-none absolute border-2 border-white/[.16]"
        style="inset:{s(6)}px;border-radius:{s(4)}px"
      ></div>
      <div
        class="pointer-events-none absolute top-1/2 border-t-2 border-white/[.16]"
        style="left:{s(6)}px;right:{s(6)}px"
      ></div>
      <div
        class="pointer-events-none absolute left-1/2 top-1/2 rounded-full border-2 border-white/[.16]"
        style="width:{s(120)}px;height:{s(120)}px;margin:{-s(60)}px 0 0 {-s(60)}px"
      ></div>
      <div
        class="pointer-events-none absolute left-1/2 border-2 border-b-0 border-white/[.16]"
        style="bottom:{s(6)}px;width:{s(180)}px;height:{s(56)}px;margin-left:{-s(90)}px"
      ></div>

      <!-- Placeholders: fundações, monte e colunas vazias -->
      {#each placeholders as p (p.key)}
        <div
          class="absolute box-border flex items-center justify-center font-bebas"
          style="
            left:{p.x}px;top:{p.y}px;width:{cardW}px;height:{cardH}px;
            border-radius:{s(6)}px;border:2px {p.dashed ? 'dashed' : 'solid'} {p.color};
            background:{p.bg};font-size:{p.size}px;color:{p.color}"
        >
          {p.glyph}
        </div>
      {/each}

      <!-- Monte -->
      <button
        type="button"
        onclick={onDraw}
        aria-label={$t('paciencia.aria_stock')}
        class="absolute transition-transform duration-150"
        style="
          left:{colX(0)}px;top:{topY}px;width:{cardW}px;height:{cardH}px;
          transform:{hintIds.includes('stock') ? `translateY(${-s(6)}px)` : 'none'}"
      >
        {#if game.stock.length}
          {@render cardBack(
            hintIds.includes('stock')
              ? '0 0 0 3px rgba(245,213,133,.5),0 6px 14px rgba(0,0,0,.4)'
              : '0 2px 0 #07331a,0 4px 0 #0d5028,0 5px 8px rgba(0,0,0,.35)'
          )}
          <span
            class="absolute box-border rounded-full bg-gold-400 text-center font-bebas text-[#07331a] shadow-[0_2px_4px_rgba(0,0,0,.35)]"
            style="
              right:{-s(7)}px;bottom:{-s(7)}px;min-width:{s(22)}px;height:{s(22)}px;
              padding:0 {s(5)}px;font-size:{s(13)}px;line-height:{s(22)}px"
          >
            {game.stock.length}
          </span>
        {:else}
          <span class="absolute inset-0 flex items-center justify-center text-white/75">
            <RotateCcw size={Math.round(s(24))} strokeWidth={2.2} />
          </span>
        {/if}
      </button>

      <!-- Cartas -->
      {#each slots as slot (slot.key)}
        {#if slot.action}
          <button
            type="button"
            onclick={slot.action}
            aria-label={cardLabel(slot.card)}
            class="absolute transition-[top,left,transform] duration-150"
            style="
              left:{slot.x}px;top:{slot.y}px;width:{cardW}px;height:{cardH}px;
              transform:{shaking(slot.card)
              ? 'translateX(4px) rotate(3deg)'
              : hinted(slot.card)
                ? `translateY(${-s(6)}px)`
                : 'none'}"
          >
            {@render cardFace(slot.card)}
          </button>
        {:else}
          <div
            class="absolute transition-[top,left] duration-150"
            style="left:{slot.x}px;top:{slot.y}px;width:{cardW}px;height:{cardH}px"
          >
            {#if slot.faceUp}
              {@render cardFace(slot.card)}
            {:else}
              {@render cardBack('0 1px 3px rgba(0,0,0,.35)')}
            {/if}
          </div>
        {/if}
      {/each}
    </div>

    <!-- ── Barra de ações ───────────────────────────────────────────────── -->
    <div
      class="relative z-10 flex-none border-t border-gray-800 bg-[#0b111c] px-2 pt-1.5"
      style="padding-bottom:calc(env(safe-area-inset-bottom) + 10px)"
    >
      <div class="mx-auto grid max-w-lg grid-cols-4">
      <button
        type="button"
        onclick={onUndo}
        disabled={!canUndo}
        class="flex h-[58px] flex-col items-center gap-[3px] text-[11px] font-semibold text-gray-200 disabled:opacity-40"
      >
        <span
          class="flex h-[38px] w-[38px] items-center justify-center rounded-full border-[1.5px] border-white/20 bg-white/[.06]"
        >
          <Undo2 size={18} strokeWidth={2.2} />
        </span>
        {$t('paciencia.action_undo')}
      </button>
      <button
        type="button"
        onclick={onHint}
        class="flex h-[58px] flex-col items-center gap-[3px] text-[11px] font-semibold text-gold-400"
      >
        <span
          class="flex h-[38px] w-[38px] items-center justify-center rounded-full border-[1.5px] border-gold-400/45 bg-gold-400/10"
        >
          <Lightbulb size={18} strokeWidth={2.2} />
        </span>
        {$t('paciencia.action_hint')}
      </button>
      <button
        type="button"
        onclick={() => startGame(game?.daily ?? null)}
        class="flex h-[58px] flex-col items-center gap-[3px] text-[11px] font-semibold text-gray-200"
      >
        <span
          class="flex h-[38px] w-[38px] items-center justify-center rounded-full border-[1.5px] border-white/20 bg-white/[.06]"
        >
          <RefreshCcw size={18} strokeWidth={2.2} />
        </span>
        {$t('paciencia.action_new')}
      </button>
      <button
        type="button"
        onclick={startAutoFinish}
        disabled={!canAutoFinish}
        class="flex h-[58px] flex-col items-center gap-[3px] text-[11px] font-semibold text-primary-400 disabled:opacity-40"
      >
        <span
          class="flex h-[38px] w-[38px] items-center justify-center rounded-full border-[1.5px] border-primary-400/45 bg-primary-400/10"
        >
          <CheckCheck size={18} strokeWidth={2.2} />
        </span>
        {$t('paciencia.action_finish')}
      </button>
      </div>
    </div>

    <!-- ── Vitória ──────────────────────────────────────────────────────── -->
    {#if game.won}
      <div
        class="absolute inset-0 z-20 flex flex-col items-center justify-center gap-[22px] overflow-y-auto p-6"
        style="background:radial-gradient(80% 60% at 50% 40%,rgba(13,80,40,.92),rgba(3,7,18,.94))"
      >
        <div class="max-[339px]:scale-[.92] max-[339px]:origin-top">
          <CrestShell>
            <div class="relative z-10 text-center">
              <div class="flex justify-center gap-1.5 pt-[18px]">
                {#each [1, 2, 3, 4, 5] as i}
                  <svg
                    width="22"
                    height="22"
                    viewBox="0 0 24 24"
                    fill={i <= winStars(game.elapsed) ? '#f5d585' : 'none'}
                    stroke="#f5d585"
                    stroke-width="1.6"
                    stroke-opacity={i <= winStars(game.elapsed) ? 1 : 0.5}
                    aria-hidden="true"
                  >
                    <path
                      d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01z"
                    />
                  </svg>
                {/each}
              </div>
              <div class="mt-1 text-[11px] font-bold uppercase tracking-[.2em] text-gold-400">
                {game.daily ? $t('paciencia.win_kicker_daily') : $t('paciencia.win_kicker')}
              </div>
              <div
                class="mt-3 font-bebas text-[64px] uppercase leading-[.9] text-white [text-shadow:0_4px_14px_rgba(0,0,0,.35)]"
              >
                {$t('paciencia.win_headline')}
              </div>
              <p class="mx-6 mt-1.5 text-[12.5px] font-medium leading-[1.4] text-primary-100">
                {game.daily ? $t('paciencia.win_note_daily') : $t('paciencia.win_note')}
              </p>
              <div class="crest-divider mx-6 mt-4"></div>
              <div class="mx-[22px] mt-2.5 grid grid-cols-3">
                {#each [{ label: $t('paciencia.hud_time'), value: fmtTime(game.elapsed) }, { label: $t('paciencia.hud_moves'), value: String(game.moves) }, { label: $t('paciencia.hud_score'), value: String(game.score) }] as box (box.label)}
                  <div>
                    <div class="font-bebas text-[26px] leading-none">{box.value}</div>
                    <div
                      class="mt-0.5 text-[9px] font-bold uppercase tracking-[.14em] text-gold-400"
                    >
                      {box.label}
                    </div>
                  </div>
                {/each}
              </div>
              <div class="crest-divider mx-6 mt-2.5"></div>
            </div>
          </CrestShell>
        </div>
        <div class="flex w-full max-w-sm flex-col gap-2.5">
          <button
            type="button"
            onclick={() => startGame(null)}
            class="h-[54px] rounded-xl bg-gold-400 font-bebas text-[20px] uppercase tracking-[.06em] text-[#07331a] shadow-[0_8px_20px_rgba(224,185,92,.3)] transition-colors hover:bg-gold-300"
          >
            {$t('paciencia.win_play_again')}
          </button>
          <button
            type="button"
            onclick={() => go('stats')}
            class="h-12 rounded-xl border-[1.5px] border-white/25 text-[15px] font-bold text-white transition-colors hover:bg-white/10"
          >
            {$t('paciencia.win_see_campaign')}
          </button>
        </div>
      </div>
    {/if}
  {:else}
    <!-- ── Telas de abas ────────────────────────────────────────────────── -->
    <div
      class="pointer-events-none absolute inset-0"
      style="background:url('/banners/banner-campo.jpg') center top/cover"
    ></div>
    <div
      class="pointer-events-none absolute inset-0"
      style="background:linear-gradient(180deg,rgba(17,24,39,.55) 0%,rgba(17,24,39,.88) 38%,#111827 70%)"
    ></div>

    <div
      class="relative z-10 flex-none px-3 pb-1"
      style="padding-top:calc(env(safe-area-inset-top) + 8px)"
    >
      <a
        href="/games"
        class="inline-flex items-center gap-1.5 rounded-lg px-2 py-1.5 text-[13px] font-semibold text-white/75 transition-colors hover:bg-white/10 hover:text-white"
      >
        <ChevronLeft size={16} strokeWidth={2.2} /> {$t('paciencia.back_to_games')}
      </a>
    </div>

    <div class="relative z-10 min-h-0 flex-1 overflow-y-auto">
      <div class="mx-auto max-w-md">
        {#if screen === 'home' || screen === 'game'}
          <!-- Início -->
          <div class="flex flex-col items-center px-[22px] pb-[26px] pt-[14px] text-center">
            <img
              src="/logo.png"
              alt="rachao.app"
              class="h-[180px] w-[180px] object-cover drop-shadow-[0_16px_30px_rgba(31,154,77,.45)]"
            />
            <!-- pt-2/-mt-1.5: o background-clip:text usa a caixa do elemento, então
                 sem folga no topo o acento de "PACIÊNCIA" ficaria transparente. -->
            <div
              class="-mt-1.5 bg-[linear-gradient(180deg,#f7e2ad_0%,#e0b95c_50%,#c9a34d_100%)] bg-clip-text pt-2 font-bebas text-[64px] uppercase leading-[.9] tracking-[.01em] text-transparent"
            >
              {$t('paciencia.title')}
            </div>
            <div class="mt-1.5 text-[12px] font-bold uppercase tracking-[.32em] text-primary-400">
              {$t('paciencia.kicker')}
            </div>

            <!-- Desafio do dia -->
            <div
              class="relative mt-[22px] flex w-full items-center gap-3.5 overflow-hidden rounded-[14px] border border-gold-400/35 bg-[linear-gradient(158deg,#15422a_0%,#1f9a4d_40%,#0d5028_72%,#07331a_100%)] py-3.5 pl-4 pr-3.5 text-left"
            >
              <div
                class="pointer-events-none absolute inset-0"
                style="background:repeating-linear-gradient(118deg,rgba(255,255,255,.055) 0 2px,transparent 2px 10px)"
              ></div>
              <div
                class="relative flex h-[46px] w-[46px] flex-none items-center justify-center rounded-xl border border-gold-400/45 bg-black/25 text-gold-400"
              >
                <Calendar size={24} />
              </div>
              <div class="relative flex min-w-0 flex-1 flex-col gap-px">
                <span class="text-[10.5px] font-bold uppercase tracking-[.16em] text-gold-400">
                  {$t('paciencia.daily_today')}
                </span>
                <span class="font-bebas text-[21px] uppercase leading-[1.1] text-white">
                  {todayLabel}
                </span>
                <span class="text-[12px] font-medium text-primary-200">
                  {todayRecord
                    ? $t('paciencia.daily_status_done', { time: fmtTime(todayRecord.time) })
                    : $t('paciencia.daily_status_same')}
                </span>
              </div>
              <button
                type="button"
                onclick={() => startGame(todayKey)}
                class="relative h-10 flex-none rounded-[10px] bg-gold-400 px-4 font-bebas text-[15px] uppercase tracking-[.06em] text-[#07331a] transition-colors hover:bg-gold-300 active:scale-95"
              >
                {todayRecord ? $t('paciencia.replay') : $t('paciencia.play')}
              </button>
            </div>

            <div class="mt-3.5 flex w-full flex-col gap-2.5">
              {#if hasRunningGame && game}
                <button
                  type="button"
                  onclick={() => go('game')}
                  class="flex h-[54px] items-center justify-center gap-2.5 rounded-xl bg-primary-600 font-bebas text-[20px] uppercase tracking-[.06em] text-white shadow-[0_8px_20px_rgba(22,163,74,.35)] transition-colors hover:bg-primary-700 active:scale-[.98]"
                >
                  <Play size={20} fill="currentColor" />
                  {$t('paciencia.resume')} · {fmtTime(game.elapsed)}
                </button>
              {/if}
              <button
                type="button"
                onclick={() => startGame(null)}
                class="flex h-[54px] items-center justify-center gap-2.5 rounded-xl font-bebas text-[20px] uppercase tracking-[.06em] text-white transition-[filter] hover:brightness-110 active:scale-[.98]
                  {hasRunningGame
                  ? 'border-[1.5px] border-white/25 bg-transparent'
                  : 'bg-primary-600 hover:bg-primary-700'}"
              >
                <Plus size={20} strokeWidth={2.4} />
                {$t('paciencia.new_game')}
              </button>
            </div>

            <div class="mt-4 flex flex-wrap justify-center gap-x-[18px] gap-y-1 text-[12px] font-medium text-gray-400">
              <span><b class="font-bold text-white">{wins.length}</b> {$t('paciencia.summary_wins')}</span>
              <span aria-hidden="true">·</span>
              <span><b class="font-bold text-white">{dailyStreak}</b> {$t('paciencia.summary_streak')}</span>
              <span aria-hidden="true">·</span>
              <span>{$t('paciencia.summary_draw')} <b class="font-bold text-white">{settings.draw}</b></span>
            </div>
          </div>
        {:else if screen === 'daily'}
          <!-- Diário -->
          <div class="px-5 pb-[26px] pt-5">
            <div class="text-[11px] font-bold uppercase tracking-[.2em] text-primary-400">
              {$t('paciencia.daily_kicker')}
            </div>
            <div class="mt-0.5 font-bebas text-[42px] uppercase leading-none">{monthLabel}</div>
            <p class="mt-2 text-[13.5px] leading-[1.5] text-gray-300">
              {$t('paciencia.daily_intro')}
            </p>

            <div class="mt-4 grid grid-cols-3 gap-2">
              {#each [{ key: 'done', label: $t('paciencia.daily_done'), value: String(dailyKeys.length), gold: false }, { key: 'streak', label: $t('paciencia.daily_streak'), value: String(dailyStreak), gold: true }, { key: 'best', label: $t('paciencia.daily_best'), value: fmtTime(dailyBest), gold: false }] as box (box.key)}
                <div
                  class="rounded-xl border bg-gray-800/85 py-2.5 text-center
                    {box.gold ? 'border-gold-400/40' : 'border-gray-700'}"
                >
                  <div class="font-bebas text-[28px] leading-none {box.gold ? 'text-gold-400' : ''}">
                    {box.value}
                  </div>
                  <div
                    class="mt-[3px] text-[10px] font-semibold uppercase tracking-[.08em] text-gray-400"
                  >
                    {box.label}
                  </div>
                </div>
              {/each}
            </div>

            <div
              class="mt-3.5 rounded-[14px] border border-gray-700 bg-gray-800/85 px-2.5 py-3"
            >
              <div class="grid grid-cols-7 gap-[5px] text-center">
                {#each weekdays as w, i (i)}
                  <span class="pb-1 text-[10px] font-bold tracking-[.08em] text-gray-500">{w}</span>
                {/each}
                {#each calendar as cell (cell.key)}
                  {#if cell.day === null}
                    <span class="aspect-square"></span>
                  {:else}
                    {@const ring = cell.today
                      ? 'border-2 border-gold-400'
                      : cell.done
                        ? 'border border-primary-600'
                        : cell.future
                          ? 'border border-transparent'
                          : 'border border-gray-700'}
                    <button
                      type="button"
                      onclick={() => startGame(cell.date)}
                      disabled={cell.future}
                      class="flex aspect-square items-center justify-center rounded-full text-[14px] transition-[filter] hover:brightness-125 disabled:hover:brightness-100
                        {ring}
                        {cell.done
                        ? 'bg-primary-600 font-bold text-white'
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
              class="mt-4 h-[54px] w-full rounded-xl bg-primary-600 font-bebas text-[20px] uppercase tracking-[.06em] text-white shadow-[0_8px_20px_rgba(22,163,74,.35)] transition-colors hover:bg-primary-700"
            >
              {todayRecord ? $t('paciencia.daily_cta_replay') : $t('paciencia.daily_cta_play')}
            </button>
          </div>
        {:else if screen === 'stats'}
          <!-- Campanha -->
          <div class="px-5 pb-[26px] pt-5">
            <div class="text-[11px] font-bold uppercase tracking-[.2em] text-primary-400">
              {$t('paciencia.stats_kicker')}
            </div>
            <div class="mt-0.5 font-bebas text-[42px] uppercase leading-none">
              {$t('paciencia.stats_title')}
            </div>

            <div class="mt-4 grid grid-cols-3 gap-2">
              {#each statCells as cell (cell.key)}
                <div
                  class="rounded-xl border bg-gray-800/85 px-1 py-3 text-center
                    {cell.highlight ? 'border-primary-400/40' : 'border-gray-700'}"
                >
                  <div
                    class="font-bebas text-[28px] leading-none {cell.highlight
                      ? 'text-primary-400'
                      : 'text-white'}"
                  >
                    {cell.value}
                  </div>
                  <div
                    class="mt-1 text-[10px] font-semibold uppercase tracking-[.06em] text-gray-400"
                  >
                    {cell.label}
                  </div>
                </div>
              {/each}
            </div>

            <div class="mb-2 mt-[22px] font-bebas text-[20px] uppercase tracking-[.04em]">
              {$t('paciencia.recent')}
            </div>
            {#if !history.length}
              <div
                class="rounded-xl border border-dashed border-gray-700 px-4 py-[22px] text-center text-[13px] text-gray-400"
              >
                {$t('paciencia.no_history')}
              </div>
            {:else}
              <div class="flex flex-col gap-1.5">
                {#each history.slice(0, HISTORY_VISIBLE) as row, i (`${row.at}-${i}`)}
                  <div
                    class="grid grid-cols-[44px_1fr_auto] items-center gap-2.5 rounded-xl border border-gray-700 bg-gray-800/85 px-3 py-2.5"
                  >
                    <span class="text-[13px] font-bold text-gray-400">
                      {new Date(row.at).toLocaleDateString($locale, {
                        day: '2-digit',
                        month: '2-digit',
                      })}
                    </span>
                    <div class="flex min-w-0 flex-col gap-px">
                      <span class="truncate text-[14px] font-semibold">
                        {row.daily ? $t('paciencia.match_daily') : $t('paciencia.match_free')}
                      </span>
                      <span class="truncate text-[12px] text-gray-400">
                        {$t('paciencia.history_detail', {
                          time: fmtTime(row.time),
                          moves: row.moves,
                          score: row.score,
                        })}
                      </span>
                    </div>
                    <span
                      class="inline-flex items-center rounded-full px-2.5 py-0.5 text-[11.5px] font-semibold
                        {row.won
                        ? 'bg-green-900/50 text-primary-300'
                        : 'bg-gray-700 text-gray-300'}"
                    >
                      {row.won ? $t('paciencia.result_win') : $t('paciencia.result_quit')}
                    </span>
                  </div>
                {/each}
              </div>
            {/if}
          </div>
        {:else}
          <!-- Ajustes -->
          <div class="px-5 pb-[26px] pt-5">
            <div class="text-[11px] font-bold uppercase tracking-[.2em] text-primary-400">
              {$t('paciencia.settings_kicker')}
            </div>
            <div class="mt-0.5 font-bebas text-[42px] uppercase leading-none">
              {$t('paciencia.settings_title')}
            </div>

            <div
              class="mt-4 overflow-hidden rounded-[14px] border border-gray-700 bg-gray-800/85"
            >
              <div class="flex items-center justify-between gap-3 px-3.5 py-3">
                <div class="flex min-w-0 flex-col gap-0.5">
                  <span class="text-[15px] font-semibold">{$t('paciencia.draw_label')}</span>
                  <span class="text-[12px] text-gray-400">{$t('paciencia.draw_hint')}</span>
                </div>
                <div class="flex flex-none gap-[3px] rounded-[10px] border border-gray-700 bg-gray-900 p-[3px]">
                  {#each [1, 3] as mode (mode)}
                    <button
                      type="button"
                      onclick={() => (settings.draw = mode as DrawMode)}
                      class="h-8 whitespace-nowrap rounded-[7px] px-3 text-[13px] font-bold transition-colors
                        {settings.draw === mode ? 'bg-primary-600 text-white' : 'text-gray-400'}"
                    >
                      {mode === 1 ? $t('paciencia.draw_1') : $t('paciencia.draw_3')}
                    </button>
                  {/each}
                </div>
              </div>

              {#each [{ key: 'timer' as const, label: $t('paciencia.timer_label'), hint: $t('paciencia.timer_hint') }, { key: 'autoFinish' as const, label: $t('paciencia.autofinish_label'), hint: $t('paciencia.autofinish_hint') }] as row (row.key)}
                <button
                  type="button"
                  onclick={() => (settings[row.key] = !settings[row.key])}
                  role="switch"
                  aria-checked={settings[row.key]}
                  class="flex w-full items-center justify-between gap-3 border-t border-gray-700 px-3.5 py-3 text-left"
                >
                  <div class="flex flex-col gap-0.5">
                    <span class="text-[15px] font-semibold">{row.label}</span>
                    <span class="text-[12px] text-gray-400">{row.hint}</span>
                  </div>
                  <span
                    class="relative h-7 w-12 flex-none rounded-full transition-colors duration-150
                      {settings[row.key] ? 'bg-primary-600' : 'bg-gray-600'}"
                  >
                    <span
                      class="absolute top-[3px] h-[22px] w-[22px] rounded-full bg-white shadow-[0_1px_3px_rgba(0,0,0,.3)] transition-[left] duration-150
                        {settings[row.key] ? 'left-[23px]' : 'left-[3px]'}"
                    ></span>
                  </span>
                </button>
              {/each}
            </div>

            <div class="mb-2.5 mt-[22px] font-bebas text-[20px] uppercase tracking-[.04em]">
              {$t('paciencia.backs_title')}
            </div>
            <div class="grid grid-cols-4 gap-2">
              {#each CARD_BACK_KEYS as key (key)}
                {@const b = CARD_BACKS[key]}
                <button
                  type="button"
                  onclick={() => (settings.back = key as CardBackKey)}
                  class="flex flex-col items-center gap-[7px] rounded-xl border-2 bg-gray-800/85 pb-2 pt-2.5 text-gray-200
                    {settings.back === key ? 'border-primary-400' : 'border-transparent'}"
                >
                  <div
                    class="relative h-[70px] w-[50px] overflow-hidden rounded-md shadow-[0_2px_6px_rgba(0,0,0,.35)]"
                    style="border:1.5px solid {b.edge};background:{b.outer}"
                  >
                    <div
                      class="absolute inset-[3px] rounded"
                      style="border:1px solid {b.line};background:{b.inner}"
                    ></div>
                    {#if b.logo > 0}
                      <img
                        src="/logo.png"
                        alt=""
                        class="absolute left-1/2 top-1/2 -ml-[17px] -mt-[17px] h-[34px] w-[34px] object-cover"
                        style="opacity:{b.logo}"
                      />
                    {/if}
                  </div>
                  <span class="text-[12px] font-semibold">{$t(b.labelKey)}</span>
                </button>
              {/each}
            </div>

            <button
              type="button"
              onclick={() => (confirmReset = true)}
              class="mt-[22px] h-[46px] w-full rounded-xl border border-red-400/40 text-[14px] font-semibold text-red-300 transition-colors hover:bg-red-900/25"
            >
              {$t('paciencia.reset_stats')}
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
      {#each [{ key: 'home' as const, label: $t('paciencia.tab_home'), icon: House }, { key: 'daily' as const, label: $t('paciencia.tab_daily'), icon: Calendar }, { key: 'stats' as const, label: $t('paciencia.tab_campaign'), icon: Trophy }, { key: 'settings' as const, label: $t('paciencia.tab_settings'), icon: SettingsIcon }] as item (item.key)}
        {@const active = screen === item.key || (item.key === 'home' && screen === 'game')}
        <button
          type="button"
          onclick={() => go(item.key)}
          aria-current={active ? 'page' : undefined}
          class="flex flex-col items-center gap-[3px] pb-1.5 pt-2.5 text-[11px] font-semibold transition-colors
            {active ? 'text-primary-400' : 'text-gray-500'}"
        >
          <item.icon size={22} />
          {item.label}
        </button>
      {/each}
      </div>
    </nav>
  {/if}

  <ConfirmDialog
    bind:open={confirmReset}
    message={$t('paciencia.reset_confirm')}
    confirmLabel={$t('paciencia.reset_stats')}
    onConfirm={resetStats}
  />
</div>

<style>
  /* Bebas Neue não tem peso bold — evita o negrito sintético do navegador. */
  .paciencia {
    font-synthesis: none;
  }
  .crest-divider {
    height: 1px;
    background: linear-gradient(90deg, transparent, rgba(245, 213, 133, 0.6), transparent);
  }
</style>
