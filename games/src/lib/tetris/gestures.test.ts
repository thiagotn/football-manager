import { describe, expect, it } from 'vitest';
import { LOCK_PX, TAP_MS, createGesture, type GestureAction } from './gestures';

const CELL = 30;

/** Arrasta de (x0,y0) até (x1,y1) em `ms`, com `steps` amostras, e solta. */
function swipe(x0: number, y0: number, x1: number, y1: number, ms: number, steps = 10): GestureAction[] {
  const g = createGesture(CELL, x0, y0, 0);
  const out: GestureAction[] = [];
  for (let i = 1; i < steps; i++) {
    const k = i / steps;
    out.push(...g.move(x0 + (x1 - x0) * k, y0 + (y1 - y0) * k, ms * k));
  }
  out.push(...g.end(x1, y1, ms));
  return out;
}

describe('gestos do Tetris', () => {
  it('toque curto gira; toque longo parado não faz nada', () => {
    expect(createGesture(CELL, 100, 100, 0).end(102, 101, 120)).toEqual(['rotate']);
    expect(createGesture(CELL, 100, 100, 0).end(100, 100, TAP_MS + 200)).toEqual([]);
  });

  it('arrastar para o lado move uma coluna por casa', () => {
    expect(swipe(100, 300, 100 + 3 * CELL + 5, 304, 400)).toEqual(['right', 'right', 'right']);
    expect(swipe(300, 300, 300 - 2 * CELL - 5, 296, 400)).toEqual(['left', 'left']);
  });

  it('ir e voltar no mesmo arraste devolve a peça', () => {
    const g = createGesture(CELL, 200, 300, 0);
    const out = [...g.move(200 + 2 * CELL, 300, 100), ...g.move(200, 300, 200), ...g.end(200, 300, 260)];
    expect(out).toEqual(['right', 'right', 'left', 'left']);
  });

  it('o eixo trava: arraste lateral torto não desce a peça', () => {
    const out = swipe(100, 100, 100 + 4 * CELL, 100 + 2 * CELL, 600);
    expect(out.every((a) => a === 'right')).toBe(true);
    expect(out).toHaveLength(4);
  });

  it('arrastar para baixo devagar desce linha por linha, sem soltar', () => {
    const out = swipe(150, 100, 152, 100 + 4 * CELL + 3, 1200, 20);
    expect(out).toEqual(['down', 'down', 'down', 'down']);
  });

  it('deslizar rápido para baixo solta a peça', () => {
    const out = swipe(150, 100, 150, 100 + 5 * CELL, 120, 6);
    expect(out.at(-1)).toBe('drop');
  });

  it('flick conta mesmo com o dedo saindo um pouco depois do último movimento', () => {
    // Medido no Chromium: o pointerup chega ~50 ms depois do último pointermove, no mesmo ponto.
    const g = createGesture(CELL, 200, 300, 753);
    const out = [
      ...g.move(200, 365, 793),
      ...g.move(200, 430, 862),
      ...g.move(200, 495, 911),
      ...g.move(200, 560, 961),
      ...g.end(200, 560, 1011),
    ];
    expect(out.at(-1)).toBe('drop');
  });

  it('parar o dedo antes de soltar não solta a peça', () => {
    const g = createGesture(CELL, 200, 300, 0);
    const out = [...g.move(200, 400, 40), ...g.move(200, 500, 80), ...g.end(200, 500, 500)];
    expect(out).not.toContain('drop');
  });

  it('flick curto demais (menos de uma casa) não solta', () => {
    const out = swipe(150, 100, 150, 100 + LOCK_PX + 5, 20, 3);
    expect(out).not.toContain('drop');
  });

  it('arrastar para cima não faz nada', () => {
    expect(swipe(150, 400, 150, 100, 100)).toEqual([]);
  });

  it('desacelerar no fim do arraste para baixo não conta como flick', () => {
    const g = createGesture(CELL, 150, 100, 0);
    const out = [...g.move(150, 100 + 3 * CELL, 80), ...g.move(150, 100 + 3 * CELL + 2, 400), ...g.end(150, 100 + 3 * CELL + 2, 420)];
    expect(out).not.toContain('drop');
  });
});
