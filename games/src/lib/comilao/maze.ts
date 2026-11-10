/**
 * O labirinto do Comilão — traçado próprio, 19 × 21 tiles.
 *
 * O arcade que inspirou usa 28 × 31. Aqui são 19 × 21 porque o jogo é
 * primariamente de celular: num aparelho de 390 px de largura isso dá ~20 px
 * por tile em vez de ~13, e pastilha de 20 px dá para ver e para acertar.
 *
 * O desenho mora numa lista de strings para poder ser lido e editado como
 * desenho. Quem garante que ele é jogável é o `maze.test.ts`: borda fechada,
 * simetria, toda pastilha alcançável, nenhum beco de uma casa (onde o jogador
 * morre sem ter como escapar) e uma única porta na casa dos fantasmas.
 *
 *   `#` parede · `.` pastilha · `o` pastilhão · ` ` corredor vazio · `-` porta
 */

export const COLS = 19;
export const ROWS = 21;

/** Conteúdo do tile (o que o jogador come). */
export const EMPTY = 0;
export const PELLET = 1;
export const POWER = 2;

export const MAZE: readonly string[] = [
  '###################',
  '#o...............o#',
  '#.###.###.###.###.#',
  '#.................#',
  '#.#.###.#.#.###.#.#',
  '#...###.....###...#',
  '#.#.#####.#####.#.#',
  '#.................#',
  '#.###.##...##.###.#',
  '#.###.###-###.###.#',
  '  ....##   ##....  ',
  '#.###.#######.###.#',
  '#.................#',
  '#.###.###.###.###.#',
  '#.....#.....#.....#',
  '#.#.###.###.###.#.#',
  '#.................#',
  '#.###.###.###.###.#',
  '#.....#.....#.....#',
  '#o...............o#',
  '###################',
];

/** Onde o Comilão nasce (e volta depois de perder uma vida). */
export const HERO_START = { col: 9, row: 14 } as const;
/** Centro da casa dos fantasmas. */
export const HOUSE = { col: 9, row: 10 } as const;
/** A porta: fantasma passa, Comilão não. */
export const DOOR = { col: 9, row: 9 } as const;
/** Primeiro tile livre acima da porta — é por aqui que os fantasmas saem. */
export const GATE = { col: 9, row: 8 } as const;
/** A fruta nasce logo abaixo da casa. */
export const FRUIT_TILE = { col: 9, row: 12 } as const;
/** A linha que atravessa de um lado ao outro. */
export const TUNNEL_ROW = 10;

export const at = (col: number, row: number): string =>
  row < 0 || row >= ROWS ? '#' : MAZE[row][(col + COLS) % COLS];

export const isWall = (col: number, row: number): boolean => at(col, row) === '#';
export const isDoor = (col: number, row: number): boolean => at(col, row) === '-';

/** Dá para andar nesse tile? A porta só deixa fantasma passar. */
export const walkable = (col: number, row: number, ghost = false): boolean => {
  const tile = at(col, row);
  if (tile === '#') return false;
  return tile !== '-' || ghost;
};

/** Tile da linha de travessia (os fantasmas andam devagar aqui). */
export const inTunnel = (col: number, row: number): boolean =>
  row === TUNNEL_ROW && (col <= 5 || col >= 13);

/** Colunas dão a volta; linhas, não. */
export const wrapCol = (col: number): number => (col + COLS) % COLS;

/** Pastilhas e pastilhões do labirinto, num array de 399 posições. */
export function initialTiles(): number[] {
  const tiles = new Array<number>(COLS * ROWS).fill(EMPTY);
  for (let row = 0; row < ROWS; row++) {
    for (let col = 0; col < COLS; col++) {
      const tile = MAZE[row][col];
      if (tile === '.') tiles[row * COLS + col] = PELLET;
      else if (tile === 'o') tiles[row * COLS + col] = POWER;
    }
  }
  // O tile de nascimento fica limpo: ninguém come de graça ao aparecer.
  tiles[HERO_START.row * COLS + HERO_START.col] = EMPTY;
  return tiles;
}

/** Quantas pastilhas (incluindo pastilhões) a fase tem. */
export const PELLET_TOTAL = initialTiles().filter((t) => t !== EMPTY).length;
