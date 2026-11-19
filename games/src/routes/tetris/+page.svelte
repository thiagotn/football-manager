<script lang="ts">
  /**
   * Tetris 3D — portado do jogo em HTML único do rachao.app (PRD 054).
   *
   * Rota em tela cheia: o canvas é a cena three (`$lib/tetris/scene`), o HUD e
   * os controles são Svelte, e as regras ficam em `$lib/tetris/engine`.
   *
   * O 3D é opcional e **vem desligado**: na primeira partida o jogo pergunta,
   * e depois a chave fica no overlay de início e no de pausa.
   *
   * No celular não há botões: um dedo na arena joga (`$lib/tetris/gestures` —
   * toque gira, arrastar para o lado move, para baixo desce, deslizar rápido
   * solta). Com 3D, **dois dedos** (ou Shift + arrastar no mouse) giram a
   * câmera. No computador, o teclado continua valendo.
   */
  import { onMount } from 'svelte';
  import { ChevronLeft, Pause, Play } from 'lucide-svelte';
  import { t } from '$lib/i18n';
  import {
    TETROMINOES,
    clearLines,
    dropIntervalMs,
    emptyBoard,
    fmtScore,
    ghostRow,
    hardDropScore,
    isValid,
    levelFor,
    lock,
    newPiece,
    randomType,
    rotate,
    scoreForLines,
    shapeOf,
    type Board,
    type Piece,
    type PieceType,
  } from '$lib/tetris/engine';
  import { createScene, type TetrisScene } from '$lib/tetris/scene';
  import { createGesture, type Gesture, type GestureAction } from '$lib/tetris/gestures';
  import { emptyState, loadState, saveState, type SavedState } from '$lib/tetris/storage';

  const ORBIT_SPEED = 0.0085;

  let stage = $state<HTMLDivElement>();
  let canvas = $state<HTMLCanvasElement>();
  let nextCanvas = $state<HTMLCanvasElement>();

  let board = $state<Board>(emptyBoard());
  let piece = $state<Piece | null>(null);
  let nextType = $state<PieceType | null>(null);
  let score = $state(0);
  let level = $state(1);
  let lines = $state(0);
  let running = $state(false);
  let paused = $state(false);
  let started = $state(false);
  let webgl = $state(true);
  let saved = $state<SavedState>(emptyState());
  /** Pergunta do 3D, mostrada uma vez só antes da primeira partida. */
  let asking = $state(false);
  let settings = $derived(saved.settings);

  let scene: TetrisScene | null = null;
  /** Tamanho da casa na tela — o passo do arraste; exposto em `data-cell` para o E2E. */
  let cellPx = $state(30);
  let frame = 0;
  let dropAt = 0;
  let dropMs = dropIntervalMs(1);
  let fastFall: ReturnType<typeof setInterval> | null = null;

  const playable = () => running && !paused && piece;

  // Carga dos ajustes: `$effect`, não `onMount` — a build de produção já
  // engoliu callback de `onMount` neste projeto.
  $effect(() => {
    const state = loadState();
    saved = state;
    scene?.setDepth(state.settings.depth);
  });

  function setDepth(on: boolean) {
    saved = { settings: { ...saved.settings, depth: on }, asked: true };
    saveState(saved);
    scene?.setDepth(on);
  }

  /** Jogar: na primeira vez, a pergunta do 3D vem antes da partida. */
  function play() {
    if (!saved.asked) {
      asking = true;
      return;
    }
    startGame();
  }

  function answer(on: boolean) {
    asking = false;
    setDepth(on);
    startGame();
  }

  function redraw() {
    scene?.draw(board, piece, piece ? ghostRow(board, piece) : null);
  }

  function spawn() {
    const type = nextType ?? randomType();
    nextType = randomType();
    const fresh = newPiece(type);
    if (!isValid(board, fresh)) {
      piece = null;
      running = false;
      redraw();
      return;
    }
    piece = fresh;
    drawNext();
    redraw();
  }

  function lockPiece() {
    if (!piece) return;
    const { board: cleaned, cleared } = clearLines(lock(board, piece));
    board = cleaned;
    if (cleared) {
      score += scoreForLines(cleared, level);
      lines += cleared;
      level = levelFor(lines);
      dropMs = dropIntervalMs(level);
    }
    piece = null;
    spawn();
  }

  function move(dc: number) {
    if (!playable() || !isValid(board, piece!, dc, 0)) return;
    piece = { ...piece!, col: piece!.col + dc };
    redraw();
  }

  function stepDown() {
    if (!playable()) return;
    if (isValid(board, piece!, 0, 1)) {
      piece = { ...piece!, row: piece!.row + 1 };
      redraw();
      return;
    }
    lockPiece();
  }

  function turn() {
    if (!playable()) return;
    const turned = rotate(board, piece!);
    if (!turned) return;
    piece = turned;
    redraw();
  }

  function hardDrop() {
    if (!playable()) return;
    const row = ghostRow(board, piece!);
    score += hardDropScore(piece!.row, row);
    piece = { ...piece!, row };
    lockPiece();
  }

  function startGame() {
    board = emptyBoard();
    piece = null;
    nextType = null;
    score = 0;
    level = 1;
    lines = 0;
    dropMs = dropIntervalMs(1);
    running = true;
    paused = false;
    started = true;
    spawn();
    dropAt = performance.now();
  }

  function togglePause() {
    if (!running) return;
    paused = !paused;
    if (!paused) dropAt = performance.now();
  }

  /** Miniatura da próxima peça, em canvas 2D (64×64, como no original). */
  function drawNext() {
    const ctx = nextCanvas?.getContext('2d');
    if (!ctx || !nextType) return;
    ctx.clearRect(0, 0, 64, 64);
    const blocks = shapeOf(nextType, 0);
    const color = `#${TETROMINOES[nextType].color.toString(16).padStart(6, '0')}`;
    const cols = blocks.map(([c]) => c);
    const rows = blocks.map(([, r]) => r);
    const w = Math.max(...cols) - Math.min(...cols) + 1;
    const h = Math.max(...rows) - Math.min(...rows) + 1;
    const cell = Math.floor(Math.min(52 / w, 52 / h));
    const ox = (64 - cell * w) / 2;
    const oy = (64 - cell * h) / 2;
    ctx.fillStyle = color;
    ctx.shadowColor = color;
    ctx.shadowBlur = 7;
    for (const [c, r] of blocks)
      ctx.fillRect(ox + (c - Math.min(...cols)) * cell + 1, oy + (r - Math.min(...rows)) * cell + 1, cell - 2, cell - 2);
  }

  // ── Entrada ───────────────────────────────────────────────────────────
  function onKeydown(event: KeyboardEvent) {
    if (event.code === 'KeyP') {
      togglePause();
      return;
    }
    if (!playable()) return;
    if (event.code === 'ArrowLeft') move(-1);
    if (event.code === 'ArrowRight') move(1);
    if (event.code === 'ArrowUp') {
      turn();
      event.preventDefault();
    }
    if (event.code === 'ArrowDown' && !fastFall) {
      fastFall = setInterval(stepDown, 55);
      event.preventDefault();
    }
    if (event.code === 'Space') {
      hardDrop();
      event.preventDefault();
    }
  }

  function onKeyup(event: KeyboardEvent) {
    if (event.code !== 'ArrowDown' || !fastFall) return;
    clearInterval(fastFall);
    fastFall = null;
  }

  /** Dedos (ou o mouse) em contato com a arena, por `pointerId`. */
  const pointers = new Map<number, { x: number; y: number }>();
  /** O gesto de um dedo em andamento — vira ações na peça. */
  let gesture: Gesture | null = null;
  /** Dois dedos (ou Shift + mouse) no 3D: o arraste gira a câmera. */
  let orbiting = false;

  function apply(actions: GestureAction[]) {
    for (const action of actions) {
      if (action === 'left') move(-1);
      else if (action === 'right') move(1);
      else if (action === 'down') stepDown();
      else if (action === 'rotate') turn();
      else hardDrop();
    }
  }

  function onCanvasPointerDown(event: PointerEvent) {
    canvas?.setPointerCapture(event.pointerId);
    pointers.set(event.pointerId, { x: event.clientX, y: event.clientY });
    if (settings.depth && (pointers.size >= 2 || event.shiftKey)) {
      // O segundo dedo transforma o gesto em câmera e cancela o da peça.
      gesture = null;
      orbiting = true;
      return;
    }
    if (pointers.size === 1) {
      gesture = createGesture(cellPx, event.clientX, event.clientY, event.timeStamp);
    }
  }

  function onCanvasPointerMove(event: PointerEvent) {
    const prev = pointers.get(event.pointerId);
    if (!prev) return;
    const cur = { x: event.clientX, y: event.clientY };
    pointers.set(event.pointerId, cur);
    if (orbiting) {
      // Com dois dedos, cada um contribui com metade: a câmera segue o meio deles.
      const share = ORBIT_SPEED / pointers.size;
      scene?.orbit((cur.x - prev.x) * share, (cur.y - prev.y) * share);
      return;
    }
    if (gesture && playable()) apply(gesture.move(cur.x, cur.y, event.timeStamp));
  }

  function onCanvasPointerUp(event: PointerEvent) {
    pointers.delete(event.pointerId);
    if (gesture && playable()) apply(gesture.end(event.clientX, event.clientY, event.timeStamp));
    gesture = null;
    if (pointers.size === 0) orbiting = false;
  }

  function onCanvasPointerCancel(event: PointerEvent) {
    pointers.delete(event.pointerId);
    gesture = null;
    if (pointers.size === 0) orbiting = false;
  }

  function resize() {
    if (!stage) return;
    scene?.resize(stage.clientWidth, stage.clientHeight);
    cellPx = scene?.cellPx() ?? cellPx;
  }

  onMount(() => {
    try {
      scene = createScene(canvas!);
    } catch {
      webgl = false;
      return;
    }
    scene.setDepth(saved.settings.depth);
    resize();
    redraw();

    const loop = (now: number) => {
      frame = requestAnimationFrame(loop);
      if (running && !paused && piece && now - dropAt > dropMs) {
        stepDown();
        dropAt = now;
      }
      scene?.render();
    };
    frame = requestAnimationFrame(loop);

    return () => {
      cancelAnimationFrame(frame);
      if (fastFall) clearInterval(fastFall);
      scene?.dispose();
      scene = null;
    };
  });

  let gameOver = $derived(started && !running);
</script>

<svelte:head>
  <title>{$t('tetris.name')} — games.thiagotn.com</title>
  <meta name="description" content={$t('tetris.desc')} />
  <meta name="theme-color" content="#010112" />
  <link rel="manifest" href="/tetris/manifest.webmanifest" />
  <link rel="apple-touch-icon" href="/tetris/apple-touch-icon.png" />
  <meta name="apple-mobile-web-app-title" content={$t('tetris.name')} />
</svelte:head>

<svelte:window onkeydown={onKeydown} onkeyup={onKeyup} onresize={resize} />

<div bind:this={stage} class="stage fixed inset-0 z-50 overflow-hidden bg-night text-white">
  <canvas
    bind:this={canvas}
    class="absolute inset-0 h-full w-full touch-none"
    data-depth={settings.depth}
    data-cell={Math.round(cellPx)}
    data-piece-type={piece?.type ?? ''}
    data-piece-col={piece?.col ?? ''}
    data-piece-row={piece?.row ?? ''}
    data-piece-rot={piece?.rot ?? ''}
    onpointerdown={onCanvasPointerDown}
    onpointermove={onCanvasPointerMove}
    onpointerup={onCanvasPointerUp}
    onpointercancel={onCanvasPointerCancel}
  ></canvas>

  {#if !webgl}
    <div class="absolute inset-0 grid place-items-center p-6 text-center text-sm text-white/80">
      {$t('tetris.no_webgl')}
    </div>
  {/if}

  <!-- ── HUD ──────────────────────────────────────────────────────────── -->
  <div
    class="pointer-events-none absolute inset-x-0 top-0 flex items-start justify-between gap-2 p-3"
    style="padding-top:max(12px,env(safe-area-inset-top))"
  >
    <div class="flex flex-none gap-1.5">
      <a
        href="/"
        aria-label={$t('tetris.all_games')}
        class="pointer-events-auto flex h-9 w-9 flex-none items-center justify-center rounded-lg border border-white/20 bg-white/[.06]"
      >
        <ChevronLeft size={18} strokeWidth={2.2} />
      </a>
      {#if running}
        <button
          type="button"
          aria-label={$t('tetris.aria_pause')}
          onclick={togglePause}
          class="pointer-events-auto flex h-9 w-9 flex-none items-center justify-center rounded-lg border border-white/20 bg-white/[.06]"
        >
          {#if paused}<Play size={16} strokeWidth={2.4} />{:else}<Pause size={16} strokeWidth={2.4} />{/if}
        </button>
      {/if}
    </div>

    <div class="flex items-start gap-2">
      {#each [{ label: $t('tetris.score'), value: fmtScore(score) }, { label: $t('tetris.level'), value: String(level) }, { label: $t('tetris.lines'), value: String(lines) }] as box (box.label)}
        <div class="rounded-lg border border-white/10 bg-ink-2/90 px-2 py-1 text-center">
          <div class="text-[8px] font-bold uppercase tracking-[.14em] text-mustard">{box.label}</div>
          <div class="font-display text-[16px] leading-tight tabular-nums">{box.value}</div>
        </div>
      {/each}
      <div class="hidden rounded-lg border border-white/10 bg-ink-2/90 px-2 py-1 text-center min-[360px]:block">
        <div class="text-[8px] font-bold uppercase tracking-[.14em] text-mustard">{$t('tetris.next')}</div>
        <canvas bind:this={nextCanvas} width="64" height="64" class="h-10 w-10"></canvas>
      </div>
    </div>
  </div>

  <!-- Sem botões: a dica dos gestos fica na folga de baixo do poço. -->
  <p
    class="pointer-events-none absolute inset-x-0 mx-auto max-w-[340px] px-4 text-center text-[10.5px] leading-snug text-white/40"
    style="bottom:calc(8px + env(safe-area-inset-bottom))"
  >
    {settings.depth ? $t('tetris.hint_camera') : $t('tetris.hint_flat')}
    <span class="keys">{$t('tetris.hint_keys')}</span>
  </p>

  <!-- ── Overlay de início / fim ──────────────────────────────────────── -->
  {#if !running && webgl && !asking}
    <div class="overlay">
      <div class="card flex flex-col items-center gap-3 text-center">
        <h1 class="font-display text-[34px] leading-none">
          {gameOver ? $t('tetris.game_over') : $t('tetris.name')}
        </h1>
        {#if gameOver}
          <p class="stat-label">{$t('tetris.final_score')}</p>
          <p class="stat text-[32px]">{score.toLocaleString()}</p>
        {:else}
          <p class="text-sm font-semibold text-paper-muted">{$t('tetris.desc')}</p>
        {/if}
        <!-- Antes da pergunta a chave não aparece: ela é a própria pergunta. -->
        {#if saved.asked}{@render depthSwitch()}{/if}
        <button type="button" class="btn btn-primary {saved.asked ? '' : 'mt-1'}" onclick={play}>
          {gameOver ? $t('tetris.play_again') : $t('tetris.play')}
        </button>
        <a href="/" class="text-xs font-bold uppercase tracking-[.14em] text-paper-muted">
          {$t('tetris.all_games')}
        </a>
      </div>
    </div>
  {/if}

  {#if paused && running}
    <div class="overlay">
      <div class="card flex flex-col items-center gap-3 text-center">
        <h2 class="font-display text-[28px]">{$t('tetris.paused')}</h2>
        {@render depthSwitch()}
        <button type="button" class="btn btn-primary" onclick={togglePause}>{$t('tetris.resume')}</button>
      </div>
    </div>
  {/if}

  <!-- ── Pergunta do 3D (primeira partida) ────────────────────────────── -->
  {#if asking && webgl}
    <div class="overlay">
      <div class="card flex flex-col items-center gap-3 text-center">
        <h2 class="font-display text-[28px] leading-none">{$t('tetris.ask_title')}</h2>
        <p class="text-sm font-semibold text-paper-muted">{$t('tetris.ask_body')}</p>
        <button type="button" class="btn btn-primary mt-1" onclick={() => answer(true)}>
          {$t('tetris.ask_3d')}
        </button>
        <button type="button" class="btn btn-ghost" onclick={() => answer(false)}>
          {$t('tetris.ask_flat')}
        </button>
      </div>
    </div>
  {/if}
</div>

<!-- A chave do 3D, igual no overlay de início e no de pausa. -->
{#snippet depthSwitch()}
  <label class="flex w-full cursor-pointer items-center gap-3 rounded-xl bg-black/[.06] px-3 py-2.5 text-left">
    <input
      type="checkbox"
      class="h-5 w-5 flex-none accent-mustard"
      checked={settings.depth}
      onchange={(event) => setDepth(event.currentTarget.checked)}
    />
    <span class="min-w-0">
      <span class="block text-[14px] font-bold">{$t('tetris.setting_depth')}</span>
      <span class="block text-[11.5px] font-semibold leading-snug text-paper-muted">
        {$t('tetris.setting_depth_hint')}
      </span>
    </span>
  </label>
{/snippet}

<style>
  .stage {
    height: 100dvh;
  }
  /* A linha do teclado só aparece onde há teclado (mouse = ponteiro fino). */
  .keys {
    display: none;
  }
  @media (pointer: fine) {
    .keys {
      display: block;
    }
  }
</style>
