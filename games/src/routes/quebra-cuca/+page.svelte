<script lang="ts">
  /**
   * Quebra-Cuca — quebra-cabeças de arrastar e soltar para crianças (PRD 059).
   *
   * Telas: Início (4 temas) → cenas do tema → quantas peças → montar. A
   * cena é desenhada uma vez num `<defs>` e cada peça é um `<use>` dela
   * recortado pelo contorno da peça (`$lib/quebra-cuca/jigsaw`). As peças
   * ficam na bandeja embaixo; arrastar para perto do lugar certo encaixa, e
   * longe volta para a bandeja — nunca há "erro".
   */
  import { onMount } from 'svelte';
  import { ArrowLeft, House, Settings as SettingsIcon, X } from 'lucide-svelte';
  import LanguageSwitcher from '$lib/components/LanguageSwitcher.svelte';
  import SceneArt from '$lib/quebra-cuca/SceneArt.svelte';
  import { LOGO, PARTY, STAR, imgUrl } from '$lib/quebra-cuca/art';
  import { SCENES, SCENE_H, SCENE_W, THEMES, scenesOf, type Scene, type ThemeId } from '$lib/quebra-cuca/scenes';
  import {
    SIZES,
    geometry,
    isNear,
    makeEdges,
    mulberry32,
    pieceBox,
    piecePath,
    shuffledOrder,
    sizeOf,
    trayLayout,
    type Size,
  } from '$lib/quebra-cuca/jigsaw';
  import { emptyState, loadState, recordDone, saveState, starsOf, type SavedState } from '$lib/quebra-cuca/storage';
  import { speak, stopSpeaking } from '$lib/speech';
  import { locale, t } from '$lib/i18n';

  type Piece = { i: number; r: number; c: number; d: string; placed: boolean };
  type Game = { scene: Scene; size: Size; pieces: Piece[]; order: number[]; done: boolean };

  /** Folga do encaixe: o centro solto até 40% da célula do lugar certo. */
  const SNAP = 0.4;
  /** Margens da área de jogo (px). */
  const PAD = 12;

  let saved = $state<SavedState>(emptyState());
  let ready = $state(false);
  let theme = $state<ThemeId | null>(null);
  let picking = $state<Scene | null>(null);
  let showSettings = $state(false);
  let game = $state<Game | null>(null);
  let celebrate = $state(false);
  let drag = $state<{ i: number; x: number; y: number; ox: number; oy: number } | null>(null);
  let justPlaced = $state<number | null>(null);
  /** Só anima as peças depois que a área foi medida (senão elas "voam" da posição padrão ao abrir). */
  let settled = $state(false);

  let areaEl = $state<HTMLDivElement>();
  let areaW = $state(360);
  let areaH = $state(640);

  const persist = () => saveState($state.snapshot(saved));
  const say = (text: string) => speak(text, { lang: $locale, enabled: saved.settings.voice });
  const sceneName = (s: Scene) => $t(`quebra_cuca.scene.${s.id}`);

  onMount(() => {
    saved = loadState();
    ready = true;
    return () => stopSpeaking();
  });

  // ───────── Geometria da área de jogo ─────────

  /** Tabuleiro: largura toda (até 620 px), sem passar de 56% da altura. */
  const board = $derived.by(() => {
    const w = Math.max(160, Math.min(areaW - 2 * PAD, 620, ((areaH - 2 * PAD) * 0.56 * SCENE_W) / SCENE_H));
    return { w, h: (w * SCENE_H) / SCENE_W, x: (areaW - w) / 2, y: PAD };
  });
  /** px por unidade da cena. */
  const k = $derived(board.w / SCENE_W);

  const geo = $derived(game ? geometry(game.size) : null);
  const boxW = $derived(geo ? (geo.pw + 2 * geo.tab) * k : 0);
  const boxH = $derived(geo ? (geo.ph + 2 * geo.tab) * k : 0);

  const trayTop = $derived(board.y + board.h + 36);
  const unplaced = $derived(game ? game.order.filter((i) => !game!.pieces[i].placed) : []);
  const tray = $derived(
    trayLayout(unplaced.length, boxW, boxH, { width: areaW - 2 * PAD, height: Math.max(60, areaH - trayTop - PAD) })
  );

  /** Onde a peça está desenhada agora (canto, escala). */
  function placeOf(p: Piece): { x: number; y: number; s: number; z: number } {
    if (drag?.i === p.i) return { x: drag.x, y: drag.y, s: 1, z: 30 };
    if (p.placed && game) {
      const b = pieceBox(p.r, p.c, game.size);
      return { x: board.x + b.x * k, y: board.y + b.y * k, s: 1, z: 1 };
    }
    const slot = tray.slots[unplaced.indexOf(p.i)] ?? { x: 0, y: 0 };
    return { x: PAD + slot.x, y: trayTop + slot.y, s: tray.scale, z: 5 };
  }

  // ───────── Partida ─────────

  function start(scene: Scene, pieces: number) {
    const size = sizeOf(pieces);
    const rnd = mulberry32((Date.now() ^ (scene.id.length * 7919)) >>> 0);
    const edges = makeEdges(size, rnd);
    const list: Piece[] = [];
    for (let r = 0; r < size.rows; r++)
      for (let c = 0; c < size.cols; c++) list.push({ i: list.length, r, c, d: piecePath(r, c, size, edges), placed: false });
    game = { scene, size, pieces: list, order: shuffledOrder(list.length, rnd), done: false };
    settled = false;
    setTimeout(() => (settled = true), 150);
    picking = null;
    celebrate = false;
    drag = null;
    saved.settings.pieces = pieces;
    persist();
    say($t('quebra_cuca.start_say', { name: sceneName(scene) }));
  }

  function quit() {
    stopSpeaking();
    game = null;
    celebrate = false;
    drag = null;
  }

  function finish() {
    if (!game) return;
    game.done = true;
    saved = recordDone(saved, game.scene.id, game.size.pieces);
    persist();
    setTimeout(() => {
      celebrate = true;
      if (game) say($t('quebra_cuca.done_say', { name: sceneName(game.scene) }));
    }, 450);
  }

  // ───────── Arrastar ─────────

  function onDown(event: PointerEvent, p: Piece) {
    if (!game || game.done || p.placed || drag) return;
    event.preventDefault();
    (event.currentTarget as HTMLElement).setPointerCapture(event.pointerId);
    const area = areaEl!.getBoundingClientRect();
    const rect = (event.currentTarget as HTMLElement).getBoundingClientRect();
    const s = placeOf(p).s;
    // O ponto agarrado continua debaixo do dedo quando a peça cresce para o tamanho real.
    const ox = (event.clientX - rect.left) / s;
    const oy = (event.clientY - rect.top) / s;
    drag = { i: p.i, x: event.clientX - area.left - ox, y: event.clientY - area.top - oy, ox, oy };
  }

  function onMove(event: PointerEvent) {
    if (!drag) return;
    const area = areaEl!.getBoundingClientRect();
    drag.x = event.clientX - area.left - drag.ox;
    drag.y = event.clientY - area.top - drag.oy;
  }

  function onUp() {
    if (!drag || !game || !geo) return;
    const p = game.pieces[drag.i];
    const b = pieceBox(p.r, p.c, game.size);
    const dx = drag.x - (board.x + b.x * k);
    const dy = drag.y - (board.y + b.y * k);
    if (isNear(dx, dy, SNAP * Math.min(geo.pw, geo.ph) * k)) {
      p.placed = true;
      justPlaced = p.i;
      setTimeout(() => {
        if (justPlaced === p.i) justPlaced = null;
      }, 500);
      if (game.pieces.every((x) => x.placed)) finish();
    }
    // Longe do lugar: solta e a peça volta sozinha para a bandeja.
    drag = null;
  }

  function nextScene() {
    if (!game) return;
    const list = scenesOf(game.scene.theme);
    const at = list.findIndex((s) => s.id === game!.scene.id); // game.scene é proxy do $state
    const next = list[(at + 1) % list.length];
    start(next, game.size.pieces);
  }

  const nextSize = $derived(game ? SIZES.find((s) => s.pieces > game!.size.pieces) : undefined);
  const doneIn = (id: ThemeId) => scenesOf(id).filter((s) => saved.done[s.id]).length;
</script>

<svelte:head>
  <title>{$t('quebra_cuca.title')} — games.thiagotn.com</title>
  <meta name="description" content={$t('quebra_cuca.desc')} />
  <meta name="theme-color" content="#fff3dc" />
  <link rel="manifest" href="/quebra-cuca/manifest.webmanifest" />
  <link rel="apple-touch-icon" href="/quebra-cuca/apple-touch-icon.png" />
  <meta name="apple-mobile-web-app-title" content={$t('quebra_cuca.title')} />
</svelte:head>

<!-- A cena da partida, desenhada uma vez e reaproveitada por todas as peças. -->
<svg width="0" height="0" class="absolute" aria-hidden="true">
  <defs>
    <g id="qc-scene">{#if game}<SceneArt scene={game.scene} uid="-play" />{/if}</g>
  </defs>
</svg>

<div class="qc fixed inset-0 z-50 flex flex-col overflow-hidden text-puzzle-ink" data-screen={game ? 'play' : theme ? 'theme' : 'home'}>
  {#if !ready}
    <!-- localStorage ainda não lido -->
  {:else if game}
    <!-- ── Montando ───────────────────────────────────────────────────── -->
    <header class="flex flex-none items-center gap-2 px-3 pb-1" style="padding-top:calc(env(safe-area-inset-top) + 8px)">
      <button type="button" class="icon-btn" onclick={quit} aria-label={$t('quebra_cuca.quit')}>
        <X size={22} strokeWidth={2.6} />
      </button>
      <div class="min-w-0 flex-1 text-center">
        <h1 class="truncate font-display text-[22px] leading-tight">{sceneName(game.scene)}</h1>
        <p class="text-xs font-bold text-puzzle-muted">
          {$t('quebra_cuca.progress', { n: game.pieces.filter((p) => p.placed).length, total: game.size.pieces })}
        </p>
      </div>
      <span class="w-11"></span>
    </header>

    <div
      bind:this={areaEl}
      bind:clientWidth={areaW}
      bind:clientHeight={areaH}
      class="relative min-h-0 flex-1 touch-none select-none"
      style="padding-bottom:env(safe-area-inset-bottom)"
      onpointermove={onMove}
      onpointerup={onUp}
      onpointercancel={onUp}
      role="application"
      aria-label={$t('quebra_cuca.board')}
    >
      <!-- Tabuleiro: figura clarinha (se ligada) e o contorno das vagas -->
      <svg
        class="board absolute"
        style="left:{board.x}px;top:{board.y}px;width:{board.w}px;height:{board.h}px"
        viewBox="0 0 {SCENE_W} {SCENE_H}"
        data-board
        aria-hidden="true"
      >
        <rect width={SCENE_W} height={SCENE_H} fill="#fff" />
        {#if saved.settings.ghost || game.done}
          <use href="#qc-scene" opacity={game.done ? 1 : 0.3} style="transition: opacity .5s" />
        {/if}
        {#if !game.done}
          {#each game.pieces as p (p.i)}
            {#if !p.placed}
              <path d={p.d} fill="none" stroke="rgba(59,42,90,.28)" stroke-width={1.6} stroke-dasharray="5 4" />
            {/if}
          {/each}
        {/if}
      </svg>

      {#each game.pieces as p (p.i)}
        {@const at = placeOf(p)}
        {@const b = pieceBox(p.r, p.c, game.size)}
        <div
          class="piece absolute"
          class:placed={p.placed}
          class:settled
          class:dragging={drag?.i === p.i}
          class:pop={justPlaced === p.i}
          class:hidden-done={game.done}
          style="left:{at.x}px;top:{at.y}px;width:{boxW}px;height:{boxH}px;transform:scale({at.s});z-index:{at.z}"
          onpointerdown={(e) => onDown(e, p)}
          data-piece="{p.r}-{p.c}"
          data-placed={p.placed}
          role="button"
          tabindex="-1"
          aria-label={$t('quebra_cuca.piece', { n: p.i + 1 })}
        >
          <svg viewBox="{b.x} {b.y} {b.w} {b.h}" class="h-full w-full overflow-visible" aria-hidden="true">
            <defs><clipPath id="qc-clip-{p.i}"><path d={p.d} /></clipPath></defs>
            <g clip-path="url(#qc-clip-{p.i})"><use href="#qc-scene" /></g>
            <path d={p.d} fill="none" stroke={p.placed ? 'rgba(255,255,255,.55)' : '#fff'} stroke-width={p.placed ? 1 : 2.2} />
          </svg>
        </div>
      {/each}

      {#if !game.done && unplaced.length === game.size.pieces}
        <p class="hint" style="top:{board.y + board.h + 14}px">{$t('quebra_cuca.hint_drag')}</p>
      {/if}
    </div>

    {#if celebrate}
      <div class="confetti" aria-hidden="true">
        {#each Array.from({ length: 36 }, (_, i) => i) as i (i)}
          <span style="--x:{(i * 37) % 100}%;--d:{(i % 9) * 0.16}s;--c:{['#ff7a3d', '#8e24aa', '#ffc93c', '#43b85c', '#e53935', '#2e9bff'][i % 6]};--r:{(i * 53) % 360}deg"></span>
        {/each}
      </div>
      <div class="done-bar" role="dialog" aria-labelledby="qc-done-title">
        <h2 id="qc-done-title" class="flex items-center justify-center gap-2 font-display text-[30px] leading-none">
          <img src={imgUrl(PARTY)} alt="" class="h-9 w-9" /> {$t('quebra_cuca.done_title')}
        </h2>
        <div class="mt-1 flex justify-center gap-1" aria-label={$t('quebra_cuca.stars', { n: starsOf(saved.done[game.scene.id]) })}>
          {#each SIZES as s, i (s.pieces)}
            <img src={imgUrl(STAR)} alt="" class="h-7 w-7 {i < starsOf(saved.done[game.scene.id]) ? '' : 'opacity-25 grayscale'}" />
          {/each}
        </div>
        <div class="mt-3 grid grid-cols-2 gap-2">
          <button type="button" class="big-btn bg-puzzle-pop text-white" onclick={nextScene}>{$t('quebra_cuca.next')}</button>
          {#if nextSize}
            <button type="button" class="big-btn bg-white" onclick={() => game && start(game.scene, nextSize.pieces)}>
              {$t('quebra_cuca.more_pieces', { n: nextSize.pieces })}
            </button>
          {:else}
            <button type="button" class="big-btn bg-white" onclick={() => game && start(game.scene, game.size.pieces)}>
              {$t('quebra_cuca.replay')}
            </button>
          {/if}
          <button type="button" class="big-btn col-span-2 bg-white/70" onclick={quit}>{$t('quebra_cuca.choose_other')}</button>
        </div>
      </div>
    {/if}
  {:else}
    <!-- ── Temas e cenas ──────────────────────────────────────────────── -->
    <header class="flex flex-none items-center gap-2 px-4 pb-2" style="padding-top:calc(env(safe-area-inset-top) + 12px)">
      {#if theme}
        <button type="button" class="icon-btn" onclick={() => (theme = null)} aria-label={$t('quebra_cuca.back')}>
          <ArrowLeft size={22} strokeWidth={2.6} />
        </button>
      {:else}
        <a href="/" class="icon-btn" aria-label={$t('quebra_cuca.all_games')}><House size={20} strokeWidth={2.4} /></a>
      {/if}
      <h1 class="flex min-w-0 flex-1 items-center gap-2 font-display text-[28px] leading-none">
        <img src={imgUrl(theme ? THEMES.find((x) => x.id === theme)!.icon : LOGO)} alt="" class="h-9 w-9 flex-none" />
        <span class="truncate">{theme ? $t(`quebra_cuca.theme.${theme}`) : $t('quebra_cuca.title')}</span>
      </h1>
      <button type="button" class="icon-btn" onclick={() => (showSettings = true)} aria-label={$t('quebra_cuca.settings')}>
        <SettingsIcon size={20} strokeWidth={2.4} />
      </button>
    </header>

    <main class="min-h-0 flex-1 overflow-y-auto px-4 pb-8" style="padding-bottom:calc(env(safe-area-inset-bottom) + 24px)">
      {#if !theme}
        <p class="mb-3 font-bold text-puzzle-muted">{$t('quebra_cuca.tagline')}</p>
        <div class="mx-auto grid max-w-3xl grid-cols-2 gap-3 sm:grid-cols-4">
          {#each THEMES as th (th.id)}
            {@const first = scenesOf(th.id)[0]}
            <button type="button" class="theme-card" style="--c:{th.color}" onclick={() => (theme = th.id)} data-theme={th.id}>
              <svg viewBox="0 0 {SCENE_W} {SCENE_H}" class="w-full rounded-xl" aria-hidden="true"><SceneArt scene={first} uid="-t" /></svg>
              <span class="mt-1.5 flex items-center justify-between gap-1">
                <b class="font-display text-[20px] leading-none">{$t(`quebra_cuca.theme.${th.id}`)}</b>
                <small class="text-xs font-extrabold text-puzzle-muted">{$t('quebra_cuca.scenes_done', { n: doneIn(th.id), total: 3 })}</small>
              </span>
            </button>
          {/each}
        </div>
      {:else}
        <div class="mx-auto grid max-w-3xl grid-cols-1 gap-3 sm:grid-cols-3">
          {#each scenesOf(theme) as scene (scene.id)}
            {@const stars = starsOf(saved.done[scene.id])}
            <button type="button" class="scene-card" onclick={() => (picking = scene)} data-scene={scene.id}>
              <svg viewBox="0 0 {SCENE_W} {SCENE_H}" class="w-full rounded-xl" aria-hidden="true"><SceneArt scene={scene} uid="-s" /></svg>
              <span class="mt-2 flex items-center justify-between gap-2">
                <b class="font-display text-[20px] leading-none">{sceneName(scene)}</b>
                <span class="flex flex-none" aria-label={$t('quebra_cuca.stars', { n: stars })}>
                  {#each SIZES as s, i (s.pieces)}
                    <img src={imgUrl(STAR)} alt="" class="h-5 w-5 {i < stars ? '' : 'opacity-25 grayscale'}" />
                  {/each}
                </span>
              </span>
            </button>
          {/each}
        </div>
      {/if}
    </main>
  {/if}

  <!-- Quantas peças? -->
  {#if picking}
    <div class="overlay" role="presentation" onclick={(e) => { if (e.target === e.currentTarget) picking = null; }}>
      <div class="card text-center" role="dialog" aria-modal="true" aria-labelledby="qc-pick-title">
        <svg viewBox="0 0 {SCENE_W} {SCENE_H}" class="mx-auto mb-3 w-full max-w-[260px] rounded-xl" aria-hidden="true"><SceneArt scene={picking} uid="-p" /></svg>
        <h2 id="qc-pick-title" class="font-display text-[26px] leading-none">{$t('quebra_cuca.choose_pieces')}</h2>
        <div class="mt-4 grid grid-cols-2 gap-2.5">
          {#each SIZES as s (s.pieces)}
            <button
              type="button"
              class="size-btn"
              class:chosen={saved.settings.pieces === s.pieces}
              onclick={() => picking && start(picking, s.pieces)}
              data-pieces={s.pieces}
            >
              <span class="grid gap-[3px]" style="grid-template-columns:repeat({s.cols},10px)" aria-hidden="true">
                {#each Array.from({ length: s.pieces }, (_, i) => i) as i (i)}<i></i>{/each}
              </span>
              {$t('quebra_cuca.pieces', { n: s.pieces })}
            </button>
          {/each}
        </div>
        <button type="button" class="btn btn-ghost mt-3" onclick={() => (picking = null)}>{$t('quebra_cuca.back')}</button>
      </div>
    </div>
  {/if}

  <!-- Ajustes -->
  {#if showSettings}
    <div class="overlay" role="presentation" onclick={(e) => { if (e.target === e.currentTarget) showSettings = false; }}>
      <div class="card" role="dialog" aria-modal="true" aria-labelledby="qc-settings-title">
        <h2 id="qc-settings-title" class="mb-2 font-display text-[26px] leading-none">{$t('quebra_cuca.settings')}</h2>
        <label class="toggle" for="qc-ghost">
          {$t('quebra_cuca.opt_ghost')}
          <input id="qc-ghost" type="checkbox" checked={saved.settings.ghost} onchange={(e) => { saved.settings.ghost = e.currentTarget.checked; persist(); }} />
        </label>
        <label class="toggle" for="qc-voice">
          {$t('quebra_cuca.opt_voice')}
          <input id="qc-voice" type="checkbox" checked={saved.settings.voice} onchange={(e) => { saved.settings.voice = e.currentTarget.checked; persist(); }} />
        </label>
        <div class="toggle">
          <span>{$t('quebra_cuca.language')}</span>
          <LanguageSwitcher />
        </div>
        <p class="mt-3 text-xs font-semibold text-paper-muted">{$t('quebra_cuca.credits')}</p>
        <button type="button" class="btn btn-primary mt-3" onclick={() => (showSettings = false)}>{$t('quebra_cuca.close')}</button>
      </div>
    </div>
  {/if}
</div>

<style>
  .qc {
    background:
      radial-gradient(circle at 1px 1px, rgba(59, 42, 90, 0.07) 1.2px, transparent 1.6px) 0 0 / 18px 18px,
      theme('colors.puzzle.table');
    font-synthesis: none;
  }
  .icon-btn {
    display: grid;
    place-items: center;
    width: 44px;
    height: 44px;
    flex: none;
    border-radius: 14px;
    background: #fff;
    box-shadow: 0 4px 0 rgba(59, 42, 90, 0.12);
  }
  .theme-card,
  .scene-card {
    display: block;
    padding: 8px 8px 10px;
    border-radius: 20px;
    background: #fff;
    text-align: left;
    box-shadow: 0 6px 0 rgba(59, 42, 90, 0.12);
    transition: transform 0.1s;
  }
  .theme-card {
    border-bottom: 6px solid var(--c);
  }
  .theme-card:active,
  .scene-card:active {
    transform: translateY(3px);
  }

  .board {
    border-radius: 10px;
    box-shadow:
      0 0 0 6px theme('colors.puzzle.wood'),
      0 10px 0 6px rgba(59, 42, 90, 0.18);
  }
  .piece {
    transform-origin: 0 0;
    filter: drop-shadow(0 3px 3px rgba(59, 42, 90, 0.35));
    cursor: grab;
    touch-action: none;
  }
  .piece.settled {
    transition:
      left 0.25s ease-out,
      top 0.25s ease-out,
      transform 0.25s ease-out;
  }
  .piece.dragging {
    transition: none;
    filter: drop-shadow(0 10px 10px rgba(59, 42, 90, 0.4));
    cursor: grabbing;
  }
  .piece.placed {
    filter: none;
    cursor: default;
    pointer-events: none;
  }
  .piece.pop svg {
    animation: pop 0.45s cubic-bezier(0.3, 1.6, 0.5, 1);
  }
  .piece.hidden-done {
    opacity: 0;
    transition: opacity 0.5s;
  }
  .hint {
    position: absolute;
    left: 0;
    right: 0;
    text-align: center;
    font-size: 13px;
    font-weight: 800;
    color: theme('colors.puzzle.muted');
    pointer-events: none;
  }

  .size-btn {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 8px;
    padding: 12px 8px;
    border-radius: 16px;
    background: rgba(59, 42, 90, 0.06);
    font-weight: 800;
    border: 3px solid transparent;
  }
  .size-btn.chosen {
    border-color: theme('colors.puzzle.pop');
    background: #fff;
  }
  .size-btn i {
    width: 10px;
    height: 10px;
    border-radius: 3px;
    background: theme('colors.puzzle.pop');
  }

  .done-bar {
    position: absolute;
    left: 50%;
    bottom: calc(env(safe-area-inset-bottom) + 14px);
    z-index: 40;
    width: min(380px, calc(100% - 24px));
    transform: translateX(-50%);
    padding: 16px;
    border-radius: 22px;
    background: theme('colors.paper.DEFAULT');
    box-shadow: 0 14px 30px rgba(59, 42, 90, 0.35);
    text-align: center;
    animation: rise 0.4s ease-out;
  }
  .big-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 52px;
    border-radius: 16px;
    font-family: theme('fontFamily.display');
    font-size: 19px;
    box-shadow: 0 5px 0 rgba(59, 42, 90, 0.18);
  }
  .confetti {
    position: absolute;
    inset: 0;
    z-index: 35;
    overflow: hidden;
    pointer-events: none;
  }
  .confetti span {
    position: absolute;
    top: -20px;
    left: var(--x);
    width: 10px;
    height: 16px;
    border-radius: 3px;
    background: var(--c);
    transform: rotate(var(--r));
    animation: fall 2.6s ease-in var(--d) forwards;
  }
  @keyframes pop {
    0% {
      transform: scale(1);
    }
    40% {
      transform: scale(1.08);
    }
    100% {
      transform: scale(1);
    }
  }
  @keyframes rise {
    from {
      transform: translate(-50%, 30px);
      opacity: 0;
    }
  }
  @keyframes fall {
    to {
      transform: translateY(110vh) rotate(calc(var(--r) + 540deg));
    }
  }
</style>
