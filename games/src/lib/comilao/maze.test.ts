/**
 * O labirinto é desenhado à mão — estes testes são o que o aprova.
 *
 * Beco sem saída e pastilha inalcançável são os erros clássicos do gênero:
 * passam no olho e só aparecem quando o jogador morre sem ter tido chance.
 */
import { describe, expect, it } from 'vitest';
import {
  COLS,
  DOOR,
  EMPTY,
  GATE,
  HERO_START,
  HOUSE,
  MAZE,
  PELLET_TOTAL,
  ROWS,
  TUNNEL_ROW,
  at,
  initialTiles,
  isWall,
  walkable,
} from './maze';

const key = (col: number, row: number) => `${col},${row}`;

/** Vizinhos andáveis, já com a travessia lateral. */
function exits(col: number, row: number, ghost = false): { col: number; row: number }[] {
  const out: { col: number; row: number }[] = [];
  for (const [dc, dr] of [
    [0, -1],
    [0, 1],
    [-1, 0],
    [1, 0],
  ]) {
    const next = { col: (col + dc + COLS) % COLS, row: row + dr };
    if (next.row < 0 || next.row >= ROWS) continue;
    if (walkable(next.col, next.row, ghost)) out.push(next);
  }
  return out;
}

/** Tudo que se alcança a partir de uma posição. */
function reach(from: { col: number; row: number }, ghost = false): Set<string> {
  const seen = new Set<string>([key(from.col, from.row)]);
  const queue = [from];
  while (queue.length) {
    const tile = queue.shift()!;
    for (const next of exits(tile.col, tile.row, ghost)) {
      if (seen.has(key(next.col, next.row))) continue;
      seen.add(key(next.col, next.row));
      queue.push(next);
    }
  }
  return seen;
}

describe('labirinto', () => {
  it('tem 19 × 21 tiles', () => {
    expect(MAZE).toHaveLength(ROWS);
    for (const row of MAZE) expect(row).toHaveLength(COLS);
  });

  it('só usa os símbolos do desenho', () => {
    for (const row of MAZE) expect(row).toMatch(/^[#.o \-]+$/);
  });

  it('a borda é fechada, menos as duas bocas da travessia', () => {
    for (let col = 0; col < COLS; col++) {
      expect(isWall(col, 0)).toBe(true);
      expect(isWall(col, ROWS - 1)).toBe(true);
    }
    for (let row = 0; row < ROWS; row++) {
      const open = row === TUNNEL_ROW;
      expect(isWall(0, row)).toBe(!open);
      expect(isWall(COLS - 1, row)).toBe(!open);
    }
  });

  it('é simétrico da esquerda para a direita', () => {
    for (let row = 0; row < ROWS; row++) {
      for (let col = 0; col < COLS; col++) {
        expect(at(col, row)).toBe(at(COLS - 1 - col, row));
      }
    }
  });

  it('o Comilão alcança todas as pastilhas', () => {
    const open = reach(HERO_START);
    const tiles = initialTiles();
    for (let row = 0; row < ROWS; row++) {
      for (let col = 0; col < COLS; col++) {
        if (tiles[row * COLS + col] === EMPTY) continue;
        expect(open.has(key(col, row))).toBe(true);
      }
    }
  });

  it('o Comilão não entra na casa dos fantasmas', () => {
    expect(reach(HERO_START).has(key(HOUSE.col, HOUSE.row))).toBe(false);
  });

  it('nenhum corredor é beco de uma casa', () => {
    // Dentro da casa o fantasma nasce e sai pela porta; fora dela, todo tile
    // tem de ter pelo menos duas saídas, senão é armadilha.
    const house = reach(HOUSE, true);
    const outside = reach(HERO_START);
    for (const tile of outside) {
      const [col, row] = tile.split(',').map(Number);
      if (house.has(tile) && !outside.has(tile)) continue;
      expect(exits(col, row).length, `beco em ${tile}`).toBeGreaterThanOrEqual(2);
    }
  });

  it('a casa tem uma porta só, com saída acima e berço abaixo', () => {
    const doors: string[] = [];
    for (let row = 0; row < ROWS; row++) {
      for (let col = 0; col < COLS; col++) if (at(col, row) === '-') doors.push(key(col, row));
    }
    expect(doors).toEqual([key(DOOR.col, DOOR.row)]);
    expect(walkable(GATE.col, GATE.row)).toBe(true);
    expect(walkable(HOUSE.col, HOUSE.row, true)).toBe(true);
    expect(DOOR.row).toBe(GATE.row + 1);
    expect(HOUSE.row).toBe(DOOR.row + 1);
  });

  it('o fantasma sai da casa e volta para ela', () => {
    const fromHouse = reach(HOUSE, true);
    expect(fromHouse.has(key(GATE.col, GATE.row))).toBe(true);
    expect(reach(GATE, true).has(key(HOUSE.col, HOUSE.row))).toBe(true);
  });

  it('a travessia liga os dois lados', () => {
    expect(walkable(0, TUNNEL_ROW)).toBe(true);
    expect(walkable(COLS - 1, TUNNEL_ROW)).toBe(true);
    expect(exits(0, TUNNEL_ROW).some((e) => e.col === COLS - 1)).toBe(true);
  });

  it('conta as pastilhas e deixa o berço do Comilão limpo', () => {
    const tiles = initialTiles();
    expect(tiles).toHaveLength(COLS * ROWS);
    expect(tiles[HERO_START.row * COLS + HERO_START.col]).toBe(EMPTY);
    expect(PELLET_TOTAL).toBe(tiles.filter((t) => t !== EMPTY).length);
    expect(PELLET_TOTAL).toBeGreaterThan(150);
    expect(tiles.filter((t) => t === 2)).toHaveLength(4);
  });
});
