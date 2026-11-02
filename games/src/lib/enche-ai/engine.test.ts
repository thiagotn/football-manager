import { describe, expect, it } from 'vitest';
import {
  CAPACITY, EMPTY_TUBES, applyPour, canPour, completedFlavors, flavorCountFor, generate,
  hasMove, isComplete, isWon, runLength, solve, starsFor, type Board,
} from './engine';
import { FLAVORS, WORLDS, dailyConfig, levelConfig, worldIndexFor } from './content';

const replay = (b: Board, moves: [number, number][]): Board =>
  moves.reduce((acc, [a, c]) => {
    expect(canPour(acc, a, c)).toBe(true);
    return applyPour(acc, a, c).board;
  }, b);

describe('regras', () => {
  it('runLength conta a sequência do topo', () => {
    expect(runLength([])).toBe(0);
    expect(runLength(['a', 'b', 'b'])).toBe(2);
    expect(runLength(['a', 'a', 'a', 'a'])).toBe(4);
  });

  it('só despeja sobre o mesmo sabor ou em vazio, e nunca de um copo pronto', () => {
    const b: Board = [['a', 'b'], ['b'], ['a'], [], ['c', 'c', 'c', 'c'], ['b', 'b', 'b', 'b']];
    expect(canPour(b, 0, 1)).toBe(true);
    expect(canPour(b, 0, 2)).toBe(false);
    expect(canPour(b, 0, 3)).toBe(true);
    expect(canPour(b, 3, 0)).toBe(false); // origem vazia
    expect(canPour(b, 1, 5)).toBe(false); // destino cheio
    expect(canPour(b, 4, 3)).toBe(false); // origem pronta
    expect(canPour(b, 0, 0)).toBe(false);
  });

  it('move a sequência inteira até onde couber', () => {
    const b: Board = [['a', 'b', 'b', 'b'], ['b', 'b']];
    const { board, moved } = applyPour(b, 0, 1);
    expect(moved).toBe(2);
    expect(board).toEqual([['a', 'b'], ['b', 'b', 'b', 'b']]);
    expect(b).toEqual([['a', 'b', 'b', 'b'], ['b', 'b']]); // imutável
  });

  it('detecta pronto, vitória, sem saída e sabores completos', () => {
    expect(isComplete(['x', 'x', 'x', 'x'])).toBe(true);
    expect(isComplete(['x', 'x', 'x'])).toBe(false);
    expect(isWon([['x', 'x', 'x', 'x'], []])).toBe(true);
    expect(isWon([['x', 'x', 'x'], ['x']])).toBe(false);
    expect(hasMove([['a', 'b', 'a', 'b'], ['b', 'a', 'b', 'a']])).toBe(false);
    expect([...completedFlavors([['x', 'x', 'x', 'x'], ['y']])]).toEqual(['x']);
  });

  it('estrelas: 3 sem ajuda, −1 por tipo, mínimo 1', () => {
    expect(starsFor({ undo: false, extra: false, hint: false })).toBe(3);
    expect(starsFor({ undo: true, extra: false, hint: false })).toBe(2);
    expect(starsFor({ undo: true, extra: true, hint: true })).toBe(1);
  });
});

describe('progressão', () => {
  it('sabores sobem de 3 até 9', () => {
    expect(flavorCountFor(1)).toBe(3);
    expect(flavorCountFor(5)).toBe(4);
    expect(flavorCountFor(31)).toBe(9);
    expect(flavorCountFor(200)).toBe(9);
  });

  it('cardápio a cada 10 pedidos, em ciclo', () => {
    expect(worldIndexFor(1)).toBe(0);
    expect(worldIndexFor(11)).toBe(1);
    expect(worldIndexFor(40)).toBe(3);
    expect(worldIndexFor(41)).toBe(0);
  });

  it('todo sabor de cardápio existe e não se repete', () => {
    for (const w of WORLDS) {
      expect(new Set(w.order).size).toBe(w.order.length);
      for (const id of w.order) expect(FLAVORS[id]).toBeDefined();
    }
  });
});

describe('geração', () => {
  it('é determinística pela seed', () => {
    const c = levelConfig(17);
    expect(generate(c.seed, c.flavors)).toEqual(generate(c.seed, c.flavors));
  });

  it('pedidos 1–99 têm o formato certo e solução', () => {
    for (let level = 1; level <= 99; level++) {
      const c = levelConfig(level);
      const b = generate(c.seed, c.flavors);
      expect(b.length).toBe(c.flavors.length + EMPTY_TUBES);
      for (const f of c.flavors) expect(b.flat().filter((x) => x === f).length).toBe(CAPACITY);
      expect(b.some(isComplete)).toBe(false);
      const path = solve(b);
      expect(path, `pedido ${level}`).not.toBeNull();
      expect(isWon(replay(b, path!))).toBe(true);
    }
  });

  it('pedido do dia tem solução por um ano inteiro', () => {
    const d = new Date(2026, 0, 1);
    for (let i = 0; i < 365; i++, d.setDate(d.getDate() + 1)) {
      const c = dailyConfig(d, 1);
      const b = generate(c.seed, c.flavors);
      expect(solve(b), c.daily!).not.toBeNull();
    }
  });
});
