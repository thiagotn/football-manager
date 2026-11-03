/**
 * Tetris 3D — regras puras.
 *
 * Nenhuma dependência de DOM, three.js ou Svelte: tabuleiro, peças, rotação
 * com kicks, limpeza de linhas e pontuação. A cena 3D fica em `scene.ts` e o
 * laço do jogo em `src/routes/tetris/+page.svelte`.
 *
 * Pontuação: 1/2/3/4 linhas = 100/300/500/800 × nível · hard drop = 2 por
 * linha percorrida. Nível sobe a cada 10 linhas e acelera a queda.
 */

export const BOARD_W = 10;
export const BOARD_H = 20;

/** Célula vazia = 0; célula ocupada guarda a cor da peça que travou ali. */
export type Cell = number;
export type Board = Cell[][];

export type PieceType = 'I' | 'O' | 'T' | 'S' | 'Z' | 'J' | 'L';

/** Posição de um bloco dentro da peça: `[coluna, linha]`. */
type Block = [number, number];

export type Piece = {
  type: PieceType;
  /** Índice da rotação atual (o módulo é feito em `shapeOf`). */
  rot: number;
  col: number;
  row: number;
  color: number;
};

export const TETROMINOES: Record<PieceType, { color: number; shapes: Block[][] }> = {
  I: { color: 0x00e5f5, shapes: [[[0, 0], [1, 0], [2, 0], [3, 0]], [[0, 0], [0, 1], [0, 2], [0, 3]]] },
  O: { color: 0xffd000, shapes: [[[0, 0], [1, 0], [0, 1], [1, 1]]] },
  T: {
    color: 0xee00bb,
    shapes: [
      [[0, 0], [1, 0], [2, 0], [1, 1]],
      [[0, 0], [0, 1], [1, 1], [0, 2]],
      [[1, 0], [0, 1], [1, 1], [2, 1]],
      [[1, 0], [0, 1], [1, 1], [1, 2]],
    ],
  },
  S: { color: 0x00ee77, shapes: [[[1, 0], [2, 0], [0, 1], [1, 1]], [[0, 0], [0, 1], [1, 1], [1, 2]]] },
  Z: { color: 0xff2244, shapes: [[[0, 0], [1, 0], [1, 1], [2, 1]], [[1, 0], [0, 1], [1, 1], [0, 2]]] },
  J: {
    color: 0x3377ff,
    shapes: [
      [[0, 0], [0, 1], [1, 1], [2, 1]],
      [[0, 0], [1, 0], [0, 1], [0, 2]],
      [[0, 0], [1, 0], [2, 0], [2, 1]],
      [[1, 0], [1, 1], [0, 2], [1, 2]],
    ],
  },
  L: {
    color: 0xff7700,
    shapes: [
      [[2, 0], [0, 1], [1, 1], [2, 1]],
      [[0, 0], [0, 1], [0, 2], [1, 2]],
      [[0, 0], [1, 0], [2, 0], [0, 1]],
      [[0, 0], [1, 0], [1, 1], [1, 2]],
    ],
  },
};

export const TYPES = Object.keys(TETROMINOES) as PieceType[];

/** Deslocamentos tentados na rotação, em ordem (0,0 primeiro). */
const KICKS: Block[] = [[0, 0], [1, 0], [-1, 0], [2, 0], [-2, 0]];

export const emptyBoard = (): Board =>
  Array.from({ length: BOARD_H }, () => new Array<Cell>(BOARD_W).fill(0));

export function shapeOf(type: PieceType, rot: number): Block[] {
  const shapes = TETROMINOES[type].shapes;
  return shapes[((rot % shapes.length) + shapes.length) % shapes.length];
}

/** A peça cabe no tabuleiro deslocada de `dc`/`dr` e com rotação `nr`? */
export function isValid(board: Board, piece: Piece, dc = 0, dr = 0, nr?: number): boolean {
  for (const [bc, br] of shapeOf(piece.type, nr ?? piece.rot)) {
    const c = piece.col + bc + dc;
    const r = piece.row + br + dr;
    if (c < 0 || c >= BOARD_W) return false;
    if (r >= BOARD_H) return false;
    // Acima do topo é permitido: é por ali que a peça nasce.
    if (r < 0) continue;
    if (board[r][c]) return false;
  }
  return true;
}

/** Linha em que a peça pousaria caindo direto — a "sombra". */
export function ghostRow(board: Board, piece: Piece): number {
  let drop = 0;
  while (isValid(board, piece, 0, drop + 1)) drop++;
  return piece.row + drop;
}

/** Trava a peça devolvendo um tabuleiro novo (o de entrada não é tocado). */
export function lock(board: Board, piece: Piece): Board {
  const next = board.map((row) => [...row]);
  for (const [bc, br] of shapeOf(piece.type, piece.rot)) {
    const c = piece.col + bc;
    const r = piece.row + br;
    if (r >= 0 && r < BOARD_H && c >= 0 && c < BOARD_W) next[r][c] = piece.color;
  }
  return next;
}

/** Remove as linhas cheias, empurrando o resto para baixo. */
export function clearLines(board: Board): { board: Board; cleared: number } {
  const kept = board.filter((row) => row.some((cell) => cell === 0));
  const cleared = BOARD_H - kept.length;
  const empty = Array.from({ length: cleared }, () => new Array<Cell>(BOARD_W).fill(0));
  return { board: [...empty, ...kept], cleared };
}

/** Rotação com kicks: devolve a peça girada ou `null` se nenhuma posição couber. */
export function rotate(board: Board, piece: Piece): Piece | null {
  const nr = (piece.rot + 1) % TETROMINOES[piece.type].shapes.length;
  for (const [dc, dr] of KICKS) {
    if (isValid(board, piece, dc, dr, nr)) {
      return { ...piece, col: piece.col + dc, row: piece.row + dr, rot: nr };
    }
  }
  return null;
}

/** Coluna de nascimento: a peça entra centralizada. */
export function spawnColumn(type: PieceType): number {
  const width = Math.max(...TETROMINOES[type].shapes[0].map(([c]) => c));
  return Math.floor((BOARD_W - width - 1) / 2);
}

export function newPiece(type: PieceType): Piece {
  return { type, rot: 0, col: spawnColumn(type), row: 0, color: TETROMINOES[type].color };
}

/** `rand` injetável (`Math.random` em produção) para o sorteio ser testável. */
export function randomType(rand: () => number = Math.random): PieceType {
  return TYPES[Math.floor(rand() * TYPES.length)];
}

const LINE_SCORE = [0, 100, 300, 500, 800];

export const scoreForLines = (cleared: number, level: number): number =>
  (LINE_SCORE[cleared] ?? 0) * level;

export const hardDropScore = (fromRow: number, toRow: number): number => (toRow - fromRow) * 2;

export const levelFor = (lines: number): number => Math.floor(lines / 10) + 1;

/** Intervalo de queda por nível, saturando em 80ms. */
export const dropIntervalMs = (level: number): number => Math.max(80, 820 - (level - 1) * 74);

/** `12.3K` a partir de 10 mil, para o HUD não estourar. */
export const fmtScore = (score: number): string =>
  score >= 10_000 ? `${(score / 1000).toFixed(1)}K` : String(score);
