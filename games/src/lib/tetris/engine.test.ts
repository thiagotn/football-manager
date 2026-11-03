import { describe, expect, it } from 'vitest';

import {
  BOARD_H,
  BOARD_W,
  TETROMINOES,
  TYPES,
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
  spawnColumn,
  type Board,
  type PieceType,
} from './engine';

/** Preenche uma linha inteira, deixando `gap` vazia quando informada. */
function fillRow(board: Board, row: number, gap?: number): Board {
  const next = board.map((r) => [...r]);
  for (let c = 0; c < BOARD_W; c++) if (c !== gap) next[row][c] = 0xffffff;
  return next;
}

describe('formas', () => {
  it('cada tipo tem o número de rotações esperado', () => {
    const rots: Record<PieceType, number> = { I: 2, O: 1, T: 4, S: 2, Z: 2, J: 4, L: 4 };
    for (const type of TYPES) expect(TETROMINOES[type].shapes.length).toBe(rots[type]);
  });

  it('shapeOf é cíclico e aceita rotação fora da faixa', () => {
    expect(shapeOf('T', 4)).toEqual(shapeOf('T', 0));
    expect(shapeOf('I', 3)).toEqual(shapeOf('I', 1));
  });

  it('toda peça tem 4 blocos', () => {
    for (const type of TYPES)
      for (let rot = 0; rot < TETROMINOES[type].shapes.length; rot++)
        expect(shapeOf(type, rot)).toHaveLength(4);
  });

  it('spawnColumn mantém a peça dentro do tabuleiro', () => {
    for (const type of TYPES) {
      const piece = newPiece(type);
      const maxCol = Math.max(...shapeOf(type, 0).map(([c]) => piece.col + c));
      expect(piece.col).toBeGreaterThanOrEqual(0);
      expect(maxCol).toBeLessThan(BOARD_W);
      expect(spawnColumn(type)).toBe(piece.col);
    }
  });
});

describe('isValid', () => {
  it('recusa sair pelas paredes e pelo chão', () => {
    const board = emptyBoard();
    const piece = { ...newPiece('O'), col: 0, row: BOARD_H - 2 };
    expect(isValid(board, piece, -1, 0)).toBe(false);
    expect(isValid(board, piece, 0, 1)).toBe(false);
    expect(isValid(board, { ...piece, col: BOARD_W - 2 }, 1, 0)).toBe(false);
  });

  it('aceita linhas acima do topo (a peça nasce fora da tela)', () => {
    const piece = { ...newPiece('I'), row: -2 };
    expect(isValid(emptyBoard(), piece)).toBe(true);
  });

  it('recusa célula ocupada', () => {
    const board = fillRow(emptyBoard(), BOARD_H - 1);
    const piece = { ...newPiece('O'), row: BOARD_H - 2 };
    expect(isValid(board, piece)).toBe(false);
  });
});

describe('ghostRow', () => {
  it('cai até o chão no tabuleiro vazio', () => {
    const piece = newPiece('O');
    expect(ghostRow(emptyBoard(), piece)).toBe(BOARD_H - 2);
  });

  it('para em cima de uma pilha', () => {
    const board = fillRow(emptyBoard(), BOARD_H - 1);
    expect(ghostRow(board, newPiece('O'))).toBe(BOARD_H - 3);
  });
});

describe('lock', () => {
  it('não altera o tabuleiro recebido', () => {
    const board = emptyBoard();
    const piece = { ...newPiece('O'), row: BOARD_H - 2 };
    const next = lock(board, piece);
    expect(board.flat().every((c) => c === 0)).toBe(true);
    expect(next.flat().filter((c) => c !== 0)).toHaveLength(4);
  });

  it('grava a cor da peça', () => {
    const piece = { ...newPiece('T'), row: BOARD_H - 2 };
    const next = lock(emptyBoard(), piece);
    expect(next.flat().filter((c) => c === TETROMINOES.T.color)).toHaveLength(4);
  });
});

describe('clearLines', () => {
  it('conta e remove as linhas cheias, preservando as de cima', () => {
    let board = emptyBoard();
    board = fillRow(board, BOARD_H - 1);
    board = fillRow(board, BOARD_H - 2);
    board[BOARD_H - 3][0] = 0xabcdef;

    const { board: next, cleared } = clearLines(board);
    expect(cleared).toBe(2);
    expect(next).toHaveLength(BOARD_H);
    expect(next[BOARD_H - 1][0]).toBe(0xabcdef);
    expect(next[0].every((c) => c === 0)).toBe(true);
  });

  it('remove linhas não contíguas', () => {
    let board = emptyBoard();
    board = fillRow(board, 5);
    board = fillRow(board, BOARD_H - 1);
    expect(clearLines(board).cleared).toBe(2);
  });

  it('ignora linha com buraco', () => {
    const board = fillRow(emptyBoard(), BOARD_H - 1, 3);
    expect(clearLines(board).cleared).toBe(0);
  });
});

describe('rotate', () => {
  it('gira no espaço livre', () => {
    const piece = { ...newPiece('T'), row: 5 };
    expect(rotate(emptyBoard(), piece)?.rot).toBe(1);
  });

  it('usa kick para girar colado na parede', () => {
    // I na vertical a duas colunas da borda: só cabe deitada com o kick de -2.
    const piece = { ...newPiece('I'), col: BOARD_W - 2, row: 5, rot: 1 };
    const turned = rotate(emptyBoard(), piece);
    expect(turned).not.toBeNull();
    expect(turned!.col).toBeLessThan(piece.col);
  });

  it('devolve null quando nenhum kick cabe', () => {
    const board = emptyBoard().map((row, r) => (r === 6 ? row.map(() => 0xffffff) : row));
    const piece = { ...newPiece('I'), row: 6, rot: 1 };
    expect(rotate(board, piece)).toBeNull();
  });
});

describe('pontuação e ritmo', () => {
  it('pontua por linhas e nível', () => {
    expect(scoreForLines(1, 1)).toBe(100);
    expect(scoreForLines(4, 3)).toBe(2400);
    expect(scoreForLines(0, 5)).toBe(0);
  });

  it('hard drop paga 2 por linha', () => {
    expect(hardDropScore(2, 10)).toBe(16);
    expect(hardDropScore(5, 5)).toBe(0);
  });

  it('nível sobe a cada 10 linhas', () => {
    expect(levelFor(0)).toBe(1);
    expect(levelFor(9)).toBe(1);
    expect(levelFor(10)).toBe(2);
  });

  it('a queda acelera e satura em 80ms', () => {
    expect(dropIntervalMs(1)).toBe(820);
    expect(dropIntervalMs(2)).toBe(746);
    expect(dropIntervalMs(11)).toBe(80);
    expect(dropIntervalMs(30)).toBe(80);
  });

  it('o placar encurta a partir de 10 mil', () => {
    expect(fmtScore(9999)).toBe('9999');
    expect(fmtScore(12_345)).toBe('12.3K');
  });
});

describe('sorteio', () => {
  it('randomType é determinístico com rand injetado', () => {
    expect(randomType(() => 0)).toBe(TYPES[0]);
    expect(randomType(() => 0.999)).toBe(TYPES[TYPES.length - 1]);
  });
});

describe('fim de jogo', () => {
  it('a peça nova não cabe com o topo cheio', () => {
    let board = emptyBoard();
    for (let r = 0; r < 3; r++) board = fillRow(board, r);
    expect(isValid(board, newPiece('O'))).toBe(false);
  });
});
