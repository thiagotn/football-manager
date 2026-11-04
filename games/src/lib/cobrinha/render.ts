/**
 * Desenho da Cobrinha em canvas 2D.
 *
 * Não conhece as regras: recebe o estado do engine e pinta. O fundo (gradiente
 * da lagoa + malha de hexágonos) é desenhado uma vez num canvas fora da tela e
 * só volta a ser gerado quando o tamanho da célula muda — redesenhar centenas
 * de hexágonos a 60fps seria o único gasto real deste jogo.
 */

import { COLS, ROWS, type Game, type Point } from './engine';
import type { Skin } from './skins';

/** Cores da lagoa (espelham os tokens `lagoon` do Tailwind). */
const WATER_TOP = '#0b3d48';
const WATER_BOTTOM = '#05222a';
const HEX_LINE = 'rgba(94, 204, 214, 0.10)';
const WALL = '#2b7f91';
const WALL_EDGE = 'rgba(255, 255, 255, 0.28)';
const FRUIT = '#ffc93c';
const FRUIT_CORE = '#ffe9a8';
const BONUS = '#ff8ad1';

export type DrawOptions = {
  /** Cobra no passo anterior — com ela o movimento é interpolado. */
  prev: Point[] | null;
  /** 0→1 entre dois passos do relógio. */
  alpha: number;
  skin: Skin;
  /** Relógio para os pulsos (ms). */
  time: number;
  /** `false` com `prefers-reduced-motion`: sem pulso e sem interpolação. */
  motion: boolean;
};

export type Renderer = {
  /** Define o lado da célula em CSS px e redesenha o fundo. */
  resize: (cell: number) => void;
  draw: (game: Game, options: DrawOptions) => void;
  dispose: () => void;
};

export function createRenderer(canvas: HTMLCanvasElement): Renderer {
  const context = canvas.getContext('2d');
  if (!context) throw new Error('canvas 2d indisponível');
  // Reatribuído para o tipo perder o `null`: as funções abaixo rodam depois.
  const ctx: CanvasRenderingContext2D = context;

  let cell = 20;
  let dpr = 1;
  let background: HTMLCanvasElement | null = null;

  /** Gradiente + hexágonos, uma vez por tamanho. */
  function paintBackground(): void {
    const w = cell * COLS;
    const h = cell * ROWS;
    const layer = background ?? document.createElement('canvas');
    layer.width = Math.round(w * dpr);
    layer.height = Math.round(h * dpr);
    const c = layer.getContext('2d');
    if (!c) return;
    c.setTransform(dpr, 0, 0, dpr, 0, 0);

    const water = c.createLinearGradient(0, 0, 0, h);
    water.addColorStop(0, WATER_TOP);
    water.addColorStop(1, WATER_BOTTOM);
    c.fillStyle = water;
    c.fillRect(0, 0, w, h);

    // Malha de hexágonos de ponta para cima, como no jogo que inspirou.
    const r = cell * 1.15;
    const stepX = r * Math.sqrt(3);
    const stepY = r * 1.5;
    c.strokeStyle = HEX_LINE;
    c.lineWidth = 1;
    for (let row = -1; row * stepY < h + r; row++) {
      for (let col = -1; col * stepX < w + stepX; col++) {
        const cx = col * stepX + (row % 2 ? stepX / 2 : 0);
        const cy = row * stepY;
        c.beginPath();
        for (let i = 0; i < 6; i++) {
          const angle = (Math.PI / 180) * (60 * i - 90);
          const x = cx + r * Math.cos(angle);
          const y = cy + r * Math.sin(angle);
          if (i === 0) c.moveTo(x, y);
          else c.lineTo(x, y);
        }
        c.closePath();
        c.stroke();
      }
    }
    background = layer;
  }

  /** Retângulo arredondado — os segmentos, os obstáculos e a cabeça. */
  function rounded(x: number, y: number, size: number, radius: number): void {
    ctx.beginPath();
    ctx.roundRect(x, y, size, size, radius);
  }

  /** Centro da célula em px. */
  const center = (p: Point): Point => ({ x: (p.x + 0.5) * cell, y: (p.y + 0.5) * cell });

  /**
   * Posição desenhada do segmento `i`: entre o passo anterior e o atual. Um
   * salto maior que uma célula é travessia de parede — aí não dá para
   * interpolar, senão o segmento atravessa a tela inteira.
   */
  function lerped(snake: Point[], prev: Point[] | null, i: number, alpha: number): Point {
    const now = center(snake[i]);
    if (!prev || !prev[i]) return now;
    const before = center(prev[i]);
    if (Math.abs(now.x - before.x) > cell * 1.5 || Math.abs(now.y - before.y) > cell * 1.5) {
      return now;
    }
    return { x: before.x + (now.x - before.x) * alpha, y: before.y + (now.y - before.y) * alpha };
  }

  function drawFruit(p: Point, time: number, motion: boolean): void {
    const { x, y } = center(p);
    const pulse = motion ? 1 + Math.sin(time / 260) * 0.08 : 1;
    const r = cell * 0.3 * pulse;
    ctx.save();
    ctx.shadowColor = 'rgba(255, 201, 60, 0.55)';
    ctx.shadowBlur = cell * 0.5;
    ctx.fillStyle = FRUIT;
    ctx.beginPath();
    ctx.arc(x, y, r, 0, Math.PI * 2);
    ctx.fill();
    ctx.restore();
    ctx.fillStyle = FRUIT_CORE;
    ctx.beginPath();
    ctx.arc(x - r * 0.3, y - r * 0.3, r * 0.3, 0, Math.PI * 2);
    ctx.fill();
  }

  function drawBonus(p: Point, left: number, time: number, motion: boolean): void {
    const { x, y } = center(p);
    // Perto de acabar, pisca mais depressa.
    const hurry = left < 12 ? 120 : 300;
    const blink = motion ? 0.65 + Math.abs(Math.sin(time / hurry)) * 0.35 : 1;
    ctx.save();
    ctx.globalAlpha = blink;
    ctx.shadowColor = 'rgba(255, 138, 209, 0.6)';
    ctx.shadowBlur = cell * 0.6;
    ctx.fillStyle = BONUS;
    ctx.beginPath();
    ctx.arc(x, y, cell * 0.3, 0, Math.PI * 2);
    ctx.fill();
    ctx.restore();
    ctx.strokeStyle = 'rgba(255,255,255,.75)';
    ctx.lineWidth = Math.max(1.5, cell * 0.07);
    ctx.beginPath();
    ctx.arc(x, y, cell * 0.42, 0, Math.PI * 2);
    ctx.stroke();
  }

  function drawWalls(walls: Point[]): void {
    for (const w of walls) {
      ctx.fillStyle = WALL;
      rounded(w.x * cell + 1, w.y * cell + 1, cell - 2, cell * 0.22);
      ctx.fill();
      ctx.strokeStyle = WALL_EDGE;
      ctx.lineWidth = 1.5;
      ctx.stroke();
      // Risco diagonal: separa a pedra do desenho do fundo num relance.
      ctx.strokeStyle = 'rgba(0,0,0,.22)';
      ctx.beginPath();
      ctx.moveTo(w.x * cell + cell * 0.25, w.y * cell + cell * 0.72);
      ctx.lineTo(w.x * cell + cell * 0.72, w.y * cell + cell * 0.25);
      ctx.stroke();
    }
  }

  function drawSnake(game: Game, options: DrawOptions): void {
    const { skin, prev, motion } = options;
    const alpha = motion ? Math.max(0, Math.min(1, options.alpha)) : 1;
    const size = cell * 0.84;

    if (skin.glow) {
      ctx.save();
      ctx.shadowColor = skin.glow;
      ctx.shadowBlur = cell * 0.7;
    }

    // Da cauda para a cabeça, para a cabeça ficar por cima nas curvas.
    for (let i = game.snake.length - 1; i >= 0; i--) {
      const p = lerped(game.snake, prev, i, alpha);
      ctx.fillStyle = i === 0 ? skin.head : skin.body(i, game.snake.length);
      rounded(p.x - size / 2, p.y - size / 2, size, i === 0 ? cell * 0.34 : cell * 0.26);
      ctx.fill();
      if (i === 0) drawEyes(p, game.dir, skin);
    }

    if (skin.glow) ctx.restore();
  }

  function drawEyes(head: Point, dir: Game['dir'], skin: Skin): void {
    const gap = cell * 0.2;
    const ahead = cell * 0.12;
    const along = { up: { x: 0, y: -1 }, down: { x: 0, y: 1 }, left: { x: -1, y: 0 }, right: { x: 1, y: 0 } }[dir];
    const side = { x: -along.y, y: along.x };
    const r = Math.max(1.6, cell * 0.1);
    for (const sign of [-1, 1]) {
      const x = head.x + along.x * ahead + side.x * gap * sign;
      const y = head.y + along.y * ahead + side.y * gap * sign;
      ctx.fillStyle = '#ffffff';
      ctx.beginPath();
      ctx.arc(x, y, r, 0, Math.PI * 2);
      ctx.fill();
      ctx.fillStyle = skin.eye;
      ctx.beginPath();
      ctx.arc(x + along.x * r * 0.4, y + along.y * r * 0.4, r * 0.5, 0, Math.PI * 2);
      ctx.fill();
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
      paintBackground();
    },

    draw(game, options) {
      ctx.clearRect(0, 0, cell * COLS, cell * ROWS);
      if (background) ctx.drawImage(background, 0, 0, cell * COLS, cell * ROWS);
      drawWalls(game.walls);
      drawFruit(game.fruit, options.time, options.motion);
      if (game.bonus) drawBonus(game.bonus, game.bonusLeft, options.time, options.motion);
      drawSnake(game, options);
    },

    dispose() {
      background = null;
      ctx.setTransform(1, 0, 0, 1, 0, 0);
      ctx.clearRect(0, 0, canvas.width, canvas.height);
    },
  };
}
