/**
 * Desenho do Comilão em canvas 2D.
 *
 * Não conhece as regras: recebe o estado do engine e pinta. O fundo (piso,
 * grade do horizonte e o esqueleto das paredes) é desenhado uma vez num canvas
 * fora da tela e só volta a ser gerado quando muda o tamanho do tile ou o
 * visual — repintar 400 arestas com brilho a 60 fps seria o único gasto real
 * deste jogo.
 *
 * As paredes são **traço**, não bloco: o neon da paleta synthwave brilha sem
 * tomar a tela, e a silhueta cheia dos fantasmas continua sendo o que o olho
 * encontra primeiro.
 */

import { COLS, DEATH_MS, ROWS, type Game, type Ghost } from './engine';
import { FRIGHT_BLINK, FRIGHT_COLOR, GHOST_COLORS, type Skin } from './skins';
import { FRUIT_TILE, POWER, isWall, walkable } from './maze';

/** Número que sobe quando o jogador come fantasma ou fruta. */
export type Pop = { id: number; text: string; x: number; y: number; born: number };

export type DrawOptions = {
  skin: Skin;
  /** Brilho neon (ajuste do jogador). */
  glow: boolean;
  /** Relógio para as animações (ms). */
  time: number;
  /** `false` com `prefers-reduced-motion`. */
  motion: boolean;
  pops: Pop[];
};

export type Renderer = {
  /** Define o lado do tile em CSS px e redesenha o fundo. */
  resize: (cell: number) => void;
  draw: (game: Game, options: DrawOptions) => void;
  dispose: () => void;
};

/** Quanto tempo o número comido fica na tela. */
export const POP_MS = 900;

const FRUIT_COLORS = ['#ff4d5e', '#ffd166', '#ff9f45', '#6bffb8', '#ff7ad1', '#2ae0ff', '#c99bff', '#ffffff'];

export function createRenderer(canvas: HTMLCanvasElement): Renderer {
  const context = canvas.getContext('2d');
  if (!context) throw new Error('canvas 2d indisponível');
  // Reatribuído para o tipo perder o `null`: as funções abaixo rodam depois.
  const ctx: CanvasRenderingContext2D = context;

  let cell = 20;
  let dpr = 1;
  let background: HTMLCanvasElement | null = null;
  /** Assinatura do fundo desenhado — muda de visual ou de brilho repinta. */
  let painted = '';

  const px = (v: number): number => v * cell;

  // ── Fundo ─────────────────────────────────────────────────────────────────

  /** Grade em perspectiva: o horizonte synthwave, bem apagado. */
  function paintGrid(c: CanvasRenderingContext2D, w: number, h: number, skin: Skin): void {
    c.strokeStyle = skin.grid;
    c.lineWidth = 1;
    const vanish = { x: w / 2, y: -h * 0.4 };
    for (let i = 0; i <= 14; i++) {
      const x = (w / 14) * i;
      c.beginPath();
      c.moveTo(x, h);
      c.lineTo(vanish.x + (x - vanish.x) * 0.12, vanish.y + (h - vanish.y) * 0.12);
      c.stroke();
    }
    // Linhas horizontais com espaçamento geométrico: dá a sensação de fuga.
    for (let i = 1, y = h; i <= 12 && y > 0; i++) {
      y -= h * 0.22 * 0.82 ** i;
      c.beginPath();
      c.moveTo(0, y);
      c.lineTo(w, y);
      c.stroke();
    }
  }

  /** Arestas de parede que encostam em corredor — o esqueleto do labirinto. */
  function wallEdges(): [number, number, number, number][] {
    const edges: [number, number, number, number][] = [];
    for (let row = 0; row < ROWS; row++) {
      for (let col = 0; col < COLS; col++) {
        if (!isWall(col, row)) continue;
        const x = px(col);
        const y = px(row);
        // Só a face virada para onde se anda: entre duas paredes não há traço.
        if (row > 0 && walkable(col, row - 1, true)) edges.push([x, y, x + cell, y]);
        if (row < ROWS - 1 && walkable(col, row + 1, true)) edges.push([x, y + cell, x + cell, y + cell]);
        if (walkable(col - 1, row, true) && col > 0) edges.push([x, y, x, y + cell]);
        if (walkable(col + 1, row, true) && col < COLS - 1) edges.push([x + cell, y, x + cell, y + cell]);
      }
    }
    return edges;
  }

  function paintBackground(skin: Skin, glow: boolean): void {
    const w = cell * COLS;
    const h = cell * ROWS;
    const layer = background ?? document.createElement('canvas');
    layer.width = Math.round(w * dpr);
    layer.height = Math.round(h * dpr);
    const c = layer.getContext('2d');
    if (!c) return;
    c.setTransform(dpr, 0, 0, dpr, 0, 0);

    const floor = c.createLinearGradient(0, 0, 0, h);
    floor.addColorStop(0, skin.floor[0]);
    floor.addColorStop(1, skin.floor[1]);
    c.fillStyle = floor;
    c.fillRect(0, 0, w, h);
    paintGrid(c, w, h, skin);

    const edges = wallEdges();
    const neon = c.createLinearGradient(0, 0, 0, h);
    neon.addColorStop(0, skin.wall[0]);
    neon.addColorStop(1, skin.wall[1]);

    c.lineCap = 'round';
    // Primeira passada: o brilho. Sem ela a parede é só um risco.
    if (glow) {
      c.save();
      c.shadowColor = skin.glow;
      c.shadowBlur = cell * 0.7;
      c.strokeStyle = skin.rainbow ? skin.wall[0] : neon;
      c.lineWidth = Math.max(2, cell * 0.14);
      c.globalAlpha = 0.55;
      for (const [x1, y1, x2, y2] of edges) {
        c.beginPath();
        c.moveTo(x1, y1);
        c.lineTo(x2, y2);
        c.stroke();
      }
      c.restore();
    }
    // Segunda passada: o traço claro por cima.
    c.lineWidth = Math.max(1.5, cell * 0.1);
    for (const [x1, y1, x2, y2] of edges) {
      // No visual `arco` a cor vem da coluna, então cada aresta tem a sua.
      c.strokeStyle = skin.rainbow ? `hsl(${((x1 / w) * 300 + 190) % 360} 92% 66%)` : neon;
      c.beginPath();
      c.moveTo(x1, y1);
      c.lineTo(x2, y2);
      c.stroke();
    }

    background = layer;
  }

  // ── Elementos do jogo ─────────────────────────────────────────────────────

  function drawPellets(game: Game, skin: Skin, time: number, motion: boolean): void {
    const pulse = motion ? 1 + Math.sin(time / 230) * 0.16 : 1;
    for (let row = 0; row < ROWS; row++) {
      for (let col = 0; col < COLS; col++) {
        const tile = game.tiles[row * COLS + col];
        if (!tile) continue;
        const x = px(col + 0.5);
        const y = px(row + 0.5);
        if (tile === POWER) {
          ctx.save();
          ctx.shadowColor = skin.power;
          ctx.shadowBlur = cell * 0.5;
          ctx.fillStyle = skin.power;
          ctx.beginPath();
          ctx.arc(x, y, cell * 0.22 * pulse, 0, Math.PI * 2);
          ctx.fill();
          ctx.restore();
        } else {
          ctx.fillStyle = skin.pellet;
          ctx.beginPath();
          ctx.arc(x, y, Math.max(1.2, cell * 0.08), 0, Math.PI * 2);
          ctx.fill();
        }
      }
    }
  }

  function drawFruit(game: Game, time: number, motion: boolean): void {
    if (!game.fruit) return;
    const x = px(FRUIT_TILE.col + 0.5);
    const y = px(FRUIT_TILE.row + 0.5);
    // Quase no fim, pisca: é um aviso, não um enfeite.
    const blink = game.fruit.left < 2500 && motion ? 0.45 + Math.abs(Math.sin(time / 130)) * 0.55 : 1;
    const color = FRUIT_COLORS[(game.fruit.kind - 1) % FRUIT_COLORS.length];
    ctx.save();
    ctx.globalAlpha = blink;
    ctx.shadowColor = color;
    ctx.shadowBlur = cell * 0.5;
    ctx.fillStyle = color;
    ctx.beginPath();
    ctx.arc(x - cell * 0.12, y + cell * 0.08, cell * 0.2, 0, Math.PI * 2);
    ctx.arc(x + cell * 0.14, y + cell * 0.12, cell * 0.16, 0, Math.PI * 2);
    ctx.fill();
    ctx.restore();
    // Cabinho e folha.
    ctx.strokeStyle = '#8ede4f';
    ctx.lineWidth = Math.max(1.4, cell * 0.07);
    ctx.beginPath();
    ctx.moveTo(x - cell * 0.1, y - cell * 0.06);
    ctx.quadraticCurveTo(x + cell * 0.08, y - cell * 0.3, x + cell * 0.26, y - cell * 0.28);
    ctx.stroke();
  }

  /** O herói: disco com a boca abrindo na direção em que anda. */
  function drawHero(game: Game, skin: Skin, time: number, motion: boolean): void {
    const dying = game.phase === 'morrendo';
    const progress = dying ? 1 - Math.max(0, game.phaseLeft) / DEATH_MS : 0;
    const radius = cell * 0.46 * (dying ? Math.max(0, 1 - progress * 0.85) : 1);
    // Morrendo, a boca abre até o disco virar nada; vivo, mastiga.
    const open = dying ? 0.08 + progress * 0.92 : motion ? Math.abs(Math.sin(time / 85)) * 0.3 : 0.16;
    const angle = { right: 0, down: Math.PI / 2, left: Math.PI, up: -Math.PI / 2 }[game.hero.dir];

    for (const x of ghostlyCopies(game.hero.x)) {
      const cx = px(x);
      const cy = px(game.hero.y);
      ctx.save();
      ctx.shadowColor = skin.hero;
      ctx.shadowBlur = cell * 0.55;
      ctx.fillStyle = skin.hero;
      ctx.beginPath();
      ctx.moveTo(cx, cy);
      ctx.arc(cx, cy, radius, angle + open * Math.PI, angle - open * Math.PI + Math.PI * 2);
      ctx.closePath();
      ctx.fill();
      ctx.restore();
    }
  }

  /**
   * Posições a desenhar: perto da boca do túnel, o ator aparece nos dois lados
   * para a travessia não ser um salto.
   */
  function ghostlyCopies(x: number): number[] {
    if (x < 1.5) return [x, x + COLS];
    if (x > COLS - 1.5) return [x, x - COLS];
    return [x];
  }

  function drawGhost(ghost: Ghost, game: Game, time: number, motion: boolean): void {
    const frightened = ghost.state === 'assustado';
    const eyesOnly = ghost.state === 'olhos';
    // Susto acabando (abaixo de 1,8 s): pisca branco para avisar.
    const ending = frightened && game.frightLeft < 1800;
    const blinkWhite = ending && motion && Math.floor(time / 180) % 2 === 0;
    const body = frightened ? (blinkWhite ? FRIGHT_BLINK : FRIGHT_COLOR) : GHOST_COLORS[ghost.id];

    const w = cell * 0.84;
    const h = cell * 0.92;
    const wave = motion ? Math.sin(time / 140) * (w / 14) : 0;

    for (const x of ghostlyCopies(ghost.x)) {
      const cx = px(x);
      const cy = px(ghost.y);
      const left = cx - w / 2;
      const right = cx + w / 2;
      const top = cy - h * 0.46;
      const foot = cy + h * 0.4;

      if (!eyesOnly) {
        ctx.save();
        if (!frightened) {
          ctx.shadowColor = body;
          ctx.shadowBlur = cell * 0.4;
        }
        ctx.fillStyle = body;
        ctx.beginPath();
        // Cúpula e laterais.
        ctx.arc(cx, top + w / 2, w / 2, Math.PI, 0);
        ctx.lineTo(right, foot);
        // Barra de baixo em quatro ondas, correndo com o tempo.
        const steps = 4;
        for (let i = 0; i < steps; i++) {
          const sx = right - (w / steps) * i;
          const ex = sx - w / steps;
          const dip = (i % 2 === 0 ? 1 : -1) * (w / steps) * 0.55;
          ctx.quadraticCurveTo((sx + ex) / 2, foot + dip + wave, ex, foot);
        }
        ctx.lineTo(left, top + w / 2);
        ctx.closePath();
        ctx.fill();
        ctx.restore();
      }

      // Olhos: o branco olha para onde o fantasma anda.
      const look = { up: [0, -1], down: [0, 1], left: [-1, 0], right: [1, 0] }[ghost.dir];
      const eyeR = cell * 0.13;
      const eyeY = top + w * 0.46;
      for (const side of [-1, 1]) {
        const ex = cx + side * w * 0.2;
        if (frightened && !eyesOnly) {
          // Assustado não tem olhar: só dois pontos e a boca torta.
          ctx.fillStyle = blinkWhite ? FRIGHT_COLOR : FRIGHT_BLINK;
          ctx.beginPath();
          ctx.arc(ex, eyeY, eyeR * 0.5, 0, Math.PI * 2);
          ctx.fill();
          continue;
        }
        ctx.fillStyle = '#ffffff';
        ctx.beginPath();
        ctx.ellipse(ex, eyeY, eyeR, eyeR * 1.2, 0, 0, Math.PI * 2);
        ctx.fill();
        ctx.fillStyle = '#1b1040';
        ctx.beginPath();
        ctx.arc(ex + look[0] * eyeR * 0.5, eyeY + look[1] * eyeR * 0.6, eyeR * 0.56, 0, Math.PI * 2);
        ctx.fill();
      }

      if (frightened && !eyesOnly) {
        ctx.strokeStyle = blinkWhite ? FRIGHT_COLOR : FRIGHT_BLINK;
        ctx.lineWidth = Math.max(1.2, cell * 0.06);
        ctx.beginPath();
        const mouthY = cy + h * 0.16;
        for (let i = 0; i <= 4; i++) {
          const mx = left + w * 0.16 + (w * 0.68 * i) / 4;
          const my = mouthY + (i % 2 === 0 ? -cell * 0.05 : cell * 0.05);
          if (i === 0) ctx.moveTo(mx, my);
          else ctx.lineTo(mx, my);
        }
        ctx.stroke();
      }
    }
  }

  function drawPops(pops: Pop[], time: number): void {
    if (!pops.length) return;
    ctx.textAlign = 'center';
    ctx.textBaseline = 'middle';
    ctx.font = `700 ${Math.max(10, cell * 0.62)}px Nunito, system-ui, sans-serif`;
    for (const pop of pops) {
      const age = (time - pop.born) / POP_MS;
      if (age < 0 || age > 1) continue;
      ctx.save();
      ctx.globalAlpha = 1 - age;
      ctx.fillStyle = '#eef2f8';
      ctx.fillText(pop.text, px(pop.x), px(pop.y) - age * cell * 1.2);
      ctx.restore();
    }
  }

  return {
    resize(nextCell) {
      cell = Math.max(8, Math.floor(nextCell));
      dpr = Math.min(window.devicePixelRatio || 1, 2);
      canvas.width = Math.round(cell * COLS * dpr);
      canvas.height = Math.round(cell * ROWS * dpr);
      canvas.style.width = `${cell * COLS}px`;
      canvas.style.height = `${cell * ROWS}px`;
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
      painted = '';
    },

    draw(game, options) {
      const signature = `${cell}|${options.skin.key}|${options.glow}`;
      if (signature !== painted) {
        paintBackground(options.skin, options.glow);
        painted = signature;
      }
      ctx.clearRect(0, 0, cell * COLS, cell * ROWS);
      if (background) ctx.drawImage(background, 0, 0, cell * COLS, cell * ROWS);

      drawPellets(game, options.skin, options.time, options.motion);
      drawFruit(game, options.time, options.motion);
      // Os fantasmas vêm antes: morrendo, o herói fica por cima de todos.
      for (const ghost of game.ghosts) {
        if (ghost.state === 'casa' && game.phase === 'pronto' && ghost.wait <= 0) continue;
        drawGhost(ghost, game, options.time, options.motion);
      }
      if (game.phase !== 'fase') drawHero(game, options.skin, options.time, options.motion);
      drawPops(options.pops, options.time);
    },

    dispose() {
      background = null;
      painted = '';
      ctx.setTransform(1, 0, 0, 1, 0, 0);
      ctx.clearRect(0, 0, canvas.width, canvas.height);
    },
  };
}
