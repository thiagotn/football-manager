import { describe, expect, it } from 'vitest';
import { SCENES, SCENE_H, SCENE_W, THEMES, scenesOf } from './scenes';
import { LOGO, PARTY, STAR } from './art';
import {
  SIZES, geometry, isNear, makeEdges, mulberry32, pieceBox, piecePath, shuffledOrder, sidesOf, trayLayout,
} from './jigsaw';

const IMG_FILES = new Set(
  Object.keys(import.meta.glob('../../../static/quebra-cuca/img/*.svg')).map((p) => p.split('/').pop()!.replace('.svg', ''))
);

describe('cenas', () => {
  it('4 temas com 3 cenas cada, ids únicos', () => {
    expect(THEMES).toHaveLength(4);
    for (const t of THEMES) expect(scenesOf(t.id)).toHaveLength(3);
    expect(new Set(SCENES.map((s) => s.id)).size).toBe(SCENES.length);
  });

  it('todo desenho usado existe em static/quebra-cuca/img', () => {
    const codes = [...SCENES.flatMap((s) => s.items.map((i) => i.img)), ...THEMES.map((t) => t.icon), LOGO, STAR, PARTY];
    for (const c of codes) expect(IMG_FILES.has(c), c).toBe(true);
  });

  it('todo desenho cabe na tela da cena', () => {
    for (const s of SCENES) {
      for (const i of s.items) {
        expect(i.x - i.s / 2, `${s.id} ${i.img}`).toBeGreaterThanOrEqual(-10);
        expect(i.x + i.s / 2, `${s.id} ${i.img}`).toBeLessThanOrEqual(SCENE_W + 10);
        expect(i.y - i.s / 2, `${s.id} ${i.img}`).toBeGreaterThanOrEqual(-10);
        expect(i.y + i.s / 2, `${s.id} ${i.img}`).toBeLessThanOrEqual(SCENE_H + 10);
      }
    }
  });
});

describe('recorte', () => {
  it('tamanhos: 4, 6, 9 e 12 peças, grade coerente', () => {
    expect(SIZES.map((s) => s.pieces)).toEqual([4, 6, 9, 12]);
    for (const s of SIZES) expect(s.cols * s.rows).toBe(s.pieces);
  });

  it('vizinhas se encaixam: orelha de uma é o buraco da outra', () => {
    for (const size of SIZES) {
      const edges = makeEdges(size, mulberry32(size.pieces));
      for (let r = 0; r < size.rows; r++) {
        for (let c = 0; c < size.cols; c++) {
          const s = sidesOf(r, c, size, edges);
          if (c + 1 < size.cols) expect(s.right).toBe(-sidesOf(r, c + 1, size, edges).left);
          if (r + 1 < size.rows) expect(s.bottom).toBe(-sidesOf(r + 1, c, size, edges).top);
          // Borda do quadro é reta; dentro, sempre orelha ou buraco.
          expect(s.top === 0).toBe(r === 0);
          expect(s.left === 0).toBe(c === 0);
          expect(s.bottom === 0).toBe(r === size.rows - 1);
          expect(s.right === 0).toBe(c === size.cols - 1);
        }
      }
    }
  });

  it('o contorno é fechado, começa no canto da célula e fica dentro da caixa da peça', () => {
    const size = SIZES[3];
    const edges = makeEdges(size, mulberry32(7));
    for (let r = 0; r < size.rows; r++) {
      for (let c = 0; c < size.cols; c++) {
        const d = piecePath(r, c, size, edges);
        expect(d.startsWith('M')).toBe(true);
        expect(d.endsWith('Z')).toBe(true);
        const box = pieceBox(r, c, size);
        const nums = d.match(/-?\d+(\.\d+)?/g)!.map(Number);
        for (let i = 0; i < nums.length; i += 2) {
          expect(nums[i]).toBeGreaterThanOrEqual(box.x - 0.5);
          expect(nums[i]).toBeLessThanOrEqual(box.x + box.w + 0.5);
          expect(nums[i + 1]).toBeGreaterThanOrEqual(box.y - 0.5);
          expect(nums[i + 1]).toBeLessThanOrEqual(box.y + box.h + 0.5);
        }
      }
    }
  });

  it('o centro da caixa da peça é o centro da célula', () => {
    for (const size of SIZES) {
      const { pw, ph } = geometry(size);
      const b = pieceBox(1, 1, size);
      expect(b.x + b.w / 2).toBeCloseTo(1.5 * pw);
      expect(b.y + b.h / 2).toBeCloseTo(1.5 * ph);
    }
  });

  it('mesma seed, mesmo recorte', () => {
    const a = makeEdges(SIZES[2], mulberry32(42));
    const b = makeEdges(SIZES[2], mulberry32(42));
    expect(a).toEqual(b);
  });
});

describe('bandeja e encaixe', () => {
  it('as peças cabem na área, sem se sobrepor', () => {
    for (const n of [1, 4, 6, 9, 12]) {
      const area = { width: 360, height: 230 };
      const { scale, slots } = trayLayout(n, 130, 100, area);
      expect(slots).toHaveLength(n);
      expect(scale).toBeLessThanOrEqual(1);
      for (const s of slots) {
        expect(s.x).toBeGreaterThanOrEqual(-0.01);
        expect(s.x + 130 * scale).toBeLessThanOrEqual(area.width + 0.01);
        expect(s.y + 100 * scale).toBeLessThanOrEqual(area.height + 0.01);
      }
      for (let i = 0; i < n; i++)
        for (let j = i + 1; j < n; j++) {
          const sep = Math.abs(slots[i].x - slots[j].x) >= 130 * scale || Math.abs(slots[i].y - slots[j].y) >= 100 * scale;
          expect(sep).toBe(true);
        }
    }
  });

  it('bandeja apertada encolhe as peças até o mínimo', () => {
    expect(trayLayout(12, 200, 200, { width: 300, height: 100 }).scale).toBeCloseTo(0.3, 1);
  });

  it('encaixa só perto do lugar certo', () => {
    expect(isNear(10, 10, 20)).toBe(true);
    expect(isNear(20, 20, 20)).toBe(false);
  });

  it('a bandeja nunca vem na ordem da grade', () => {
    for (let s = 1; s < 50; s++) {
      const order = shuffledOrder(4, mulberry32(s));
      expect([...order].sort()).toEqual([0, 1, 2, 3]);
      expect(order).not.toEqual([0, 1, 2, 3]);
    }
  });
});
