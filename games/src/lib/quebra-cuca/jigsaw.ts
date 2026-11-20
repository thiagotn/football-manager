/**
 * Quebra-Cuca — recorte do quebra-cabeça. Puro: sem DOM, só geometria.
 *
 * A cena (`SCENE_W × SCENE_H`) é dividida numa grade; cada divisa entre duas
 * peças vizinhas ganha um **encaixe** (a "orelha" de uma entra no "buraco" da
 * outra), sorteado por seed. Cada peça vira um caminho SVG fechado, nas
 * coordenadas da cena, usado como `clipPath` sobre a cena inteira.
 *
 * A peça ocupa a célula dela mais `tab` de folga em cada lado (onde mora a
 * orelha). Como a folga é simétrica, **o centro da caixa da peça é o centro da
 * célula** — é o que o arrastar usa para conferir o encaixe.
 */

import { SCENE_H, SCENE_W } from './scenes';

export type Size = { pieces: number; cols: number; rows: number };

/** Do mais fácil (3 anos) ao mais difícil (6–7 anos). */
export const SIZES: Size[] = [
  { pieces: 4, cols: 2, rows: 2 },
  { pieces: 6, cols: 3, rows: 2 },
  { pieces: 9, cols: 3, rows: 3 },
  { pieces: 12, cols: 4, rows: 3 },
];

export const sizeOf = (pieces: number): Size => SIZES.find((s) => s.pieces === pieces) ?? SIZES[0];

/**
 * Encaixes da grade. `h[r][c]`: divisa entre a linha `r` e `r+1` na coluna
 * `c`; `v[r][c]`: divisa entre a coluna `c` e `c+1` na linha `r`. `+1` = a
 * peça de cima/da esquerda tem a orelha; `-1` = tem o buraco.
 */
export type Edges = { h: number[][]; v: number[][] };

export function mulberry32(seed: number): () => number {
  let a = seed >>> 0;
  return () => {
    a |= 0;
    a = (a + 0x6d2b79f5) | 0;
    let t = Math.imul(a ^ (a >>> 15), 1 | a);
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

export function makeEdges(size: Size, rnd: () => number): Edges {
  const flip = () => (rnd() < 0.5 ? 1 : -1);
  return {
    h: Array.from({ length: size.rows - 1 }, () => Array.from({ length: size.cols }, flip)),
    v: Array.from({ length: size.rows }, () => Array.from({ length: size.cols - 1 }, flip)),
  };
}

export type Geometry = { pw: number; ph: number; tab: number };

/** Célula e tamanho da orelha (20% do menor lado da célula). */
export function geometry(size: Size): Geometry {
  const pw = SCENE_W / size.cols;
  const ph = SCENE_H / size.rows;
  return { pw, ph, tab: 0.2 * Math.min(pw, ph) };
}

/** Os 4 lados da peça, no sentido horário: `+1` orelha para fora, `-1` buraco, `0` reto (borda). */
export function sidesOf(r: number, c: number, size: Size, edges: Edges) {
  return {
    top: r === 0 ? 0 : -edges.h[r - 1][c],
    right: c === size.cols - 1 ? 0 : edges.v[r][c],
    bottom: r === size.rows - 1 ? 0 : edges.h[r][c],
    left: c === 0 ? 0 : -edges.v[r][c - 1],
  };
}

const f = (n: number) => Math.round(n * 100) / 100;

/**
 * Um lado de `(x0,y0)` a `(x1,y1)`. A orelha é desenhada no sistema do lado:
 * `u` ao longo (0→1) e `v` na normal para fora (horário em SVG = esquerda do
 * sentido), escalada por `tab × dir`.
 */
function side(x0: number, y0: number, x1: number, y1: number, dir: number, tab: number): string {
  if (dir === 0) return `L${f(x1)} ${f(y1)}`;
  const dx = x1 - x0;
  const dy = y1 - y0;
  const len = Math.hypot(dx, dy);
  // Normal "para fora" num contorno horário com y para baixo: (dy, -dx).
  const nx = (dy / len) * tab * dir;
  const ny = (-dx / len) * tab * dir;
  const p = (u: number, v: number) => `${f(x0 + dx * u + nx * v)} ${f(y0 + dy * u + ny * v)}`;
  return [
    `L${p(0.36, 0)}`,
    `C${p(0.42, 0.1)} ${p(0.3, 0.92)} ${p(0.5, 0.95)}`,
    `C${p(0.7, 0.92)} ${p(0.58, 0.1)} ${p(0.64, 0)}`,
    `L${p(1, 0)}`,
  ].join(' ');
}

/** Contorno fechado da peça `(r, c)`, em coordenadas da cena. */
export function piecePath(r: number, c: number, size: Size, edges: Edges): string {
  const { pw, ph, tab } = geometry(size);
  const s = sidesOf(r, c, size, edges);
  const x0 = c * pw;
  const y0 = r * ph;
  const x1 = x0 + pw;
  const y1 = y0 + ph;
  return [
    `M${f(x0)} ${f(y0)}`,
    side(x0, y0, x1, y0, s.top, tab),
    side(x1, y0, x1, y1, s.right, tab),
    side(x1, y1, x0, y1, s.bottom, tab),
    side(x0, y1, x0, y0, s.left, tab),
    'Z',
  ].join(' ');
}

/** Caixa da peça na cena: a célula mais a folga da orelha em cada lado. */
export function pieceBox(r: number, c: number, size: Size): { x: number; y: number; w: number; h: number } {
  const { pw, ph, tab } = geometry(size);
  return { x: c * pw - tab, y: r * ph - tab, w: pw + 2 * tab, h: ph + 2 * tab };
}

/**
 * Bandeja: a maior escala (≤ 1, ≥ 0,3) em que `n` peças de `w × h` cabem em
 * fileiras dentro de `area`. Devolve a escala e o canto de cada vaga,
 * centralizadas na área.
 */
export function trayLayout(
  n: number,
  w: number,
  h: number,
  area: { width: number; height: number },
  gap = 8
): { scale: number; slots: { x: number; y: number }[] } {
  if (n === 0) return { scale: 1, slots: [] };
  let scale = 1;
  let cols = 1;
  for (; scale > 0.3; scale -= 0.02) {
    cols = Math.max(1, Math.floor((area.width + gap) / (w * scale + gap)));
    const rows = Math.ceil(n / cols);
    if (rows * (h * scale + gap) - gap <= area.height) break;
  }
  scale = Math.max(0.3, scale);
  cols = Math.max(1, Math.min(n, Math.floor((area.width + gap) / (w * scale + gap))));
  const rows = Math.ceil(n / cols);
  const cw = w * scale;
  const ch = h * scale;
  const slots: { x: number; y: number }[] = [];
  for (let i = 0; i < n; i++) {
    const row = Math.floor(i / cols);
    const inRow = row === rows - 1 ? n - row * cols : cols;
    const rowW = inRow * cw + (inRow - 1) * gap;
    const col = i % cols;
    slots.push({ x: (area.width - rowW) / 2 + col * (cw + gap), y: row * (ch + gap) });
  }
  return { scale, slots };
}

/** O centro solto está perto o bastante do centro certo? */
export const isNear = (dx: number, dy: number, tolerance: number): boolean => Math.hypot(dx, dy) <= tolerance;

/** Ordem embaralhada das peças na bandeja (nunca a ordem da grade, se der). */
export function shuffledOrder(n: number, rnd: () => number): number[] {
  const order = Array.from({ length: n }, (_, i) => i);
  for (let attempt = 0; attempt < 5; attempt++) {
    for (let i = n - 1; i > 0; i--) {
      const j = Math.floor(rnd() * (i + 1));
      [order[i], order[j]] = [order[j], order[i]];
    }
    if (n < 2 || order.some((v, i) => v !== i)) break;
  }
  return order;
}
