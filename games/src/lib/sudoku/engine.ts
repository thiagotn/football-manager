/**
 * Sudoku — regras puras.
 *
 * Sem DOM: grade 9×9, lances, anotações, dica e o gerador. O desenho fica em
 * `src/routes/sudoku/+page.svelte`.
 *
 * Cada lance devolve um estado novo (como o `step` da Cobrinha), então o teste
 * compara entrada e saída sem clonar nada na mão.
 *
 * A dificuldade **não** é o número de dados: é a técnica humana que o tabuleiro
 * exige. `hardnessOf` classifica em 1 (só singles), 2 (precisa de par nu ou
 * candidato travado) ou 3 (nem com pares/trincas escondidos fecha), e o gerador
 * sorteia até cair na faixa pedida.
 */

export const SIZE = 9;
export const CELLS = 81;

export type Difficulty = 'facil' | 'medio' | 'dificil' | 'especialista';

export const DIFFICULTIES: Difficulty[] = ['facil', 'medio', 'dificil', 'especialista'];

/** Como a dica foi deduzida (chave i18n `sudoku.tech.<tecnica>`). */
export type Technique = 'cell' | 'house' | 'revelada';

export type Hint = { at: number; digit: number; technique: Technique };

/**
 * Retrato das células **antes** do lance — é o que o desfazer devolve.
 *
 * É uma lista, e não uma célula só, porque escrever um número apaga a anotação
 * daquele número nos vizinhos: sem guardar os vizinhos, o desfazer devolveria o
 * número mas não as anotações que ele comeu.
 */
export type Move = { at: number; before: [at: number, value: number, notes: number][] };

/** Teto da pilha de desfazer — o estado inteiro vai para o localStorage. */
export const UNDO_LIMIT = 150;

export type Game = {
  /** 81 células com o dígito impresso no tabuleiro (0 = vazia). */
  givens: number[];
  /** 81 células com o que está escrito agora (inclui os dados). */
  values: number[];
  /** 81 máscaras de 9 bits com as anotações a lápis. */
  notes: number[];
  solution: number[];
  difficulty: Difficulty;
  undo: Move[];
  hints: number;
  /** Quantas vezes escreveu um dígito diferente da solução. */
  mistakes: number;
  /** Segundos de jogo — quem conta é a tela, o engine só guarda. */
  elapsed: number;
  /** `YYYY-MM-DD` quando é o tabuleiro do dia. */
  daily: string | null;
  seed: number;
  won: boolean;
};

// ── Casas e vizinhos ────────────────────────────────────────────────────────

export const rowOf = (i: number): number => (i / 9) | 0;
export const colOf = (i: number): number => i % 9;
export const boxOf = (i: number): number => ((rowOf(i) / 3) | 0) * 3 + ((colOf(i) / 3) | 0);

/** As 27 casas: 9 linhas, 9 colunas, 9 blocos. */
export const UNITS: number[][] = (() => {
  const units: number[][] = [];
  for (let r = 0; r < 9; r++) units.push(Array.from({ length: 9 }, (_, c) => r * 9 + c));
  for (let c = 0; c < 9; c++) units.push(Array.from({ length: 9 }, (_, r) => r * 9 + c));
  for (let b = 0; b < 9; b++) {
    const r0 = ((b / 3) | 0) * 3;
    const c0 = (b % 3) * 3;
    const cells: number[] = [];
    for (let r = 0; r < 3; r++) for (let c = 0; c < 3; c++) cells.push((r0 + r) * 9 + c0 + c);
    units.push(cells);
  }
  return units;
})();

/** As 20 células que disputam dígito com cada uma (linha + coluna + bloco). */
export const PEERS: number[][] = (() => {
  const peers: number[][] = [];
  for (let i = 0; i < CELLS; i++) {
    const list: number[] = [];
    for (let j = 0; j < CELLS; j++) {
      if (j === i) continue;
      if (rowOf(j) === rowOf(i) || colOf(j) === colOf(i) || boxOf(j) === boxOf(i)) list.push(j);
    }
    peers.push(list);
  }
  return peers;
})();

// ── Máscaras de candidatos (bit 0 = dígito 1) ───────────────────────────────

const ALL = 0b111111111;
const BIT = (d: number) => 1 << (d - 1);

function popcount(mask: number): number {
  let n = 0;
  for (let m = mask; m; m &= m - 1) n++;
  return n;
}

/** Dígitos de uma máscara, em ordem crescente. */
export function digitsOf(mask: number): number[] {
  const out: number[] = [];
  for (let d = 1; d <= 9; d++) if (mask & BIT(d)) out.push(d);
  return out;
}

/** Máscara dos dígitos que ainda cabem na célula, olhando só os vizinhos. */
export function candidateMask(values: number[], at: number): number {
  if (values[at]) return 0;
  let mask = ALL;
  for (const p of PEERS[at]) {
    const v = values[p];
    if (v) mask &= ~BIT(v);
  }
  return mask;
}

/** Dígitos que ainda cabem na célula (vazio se já está preenchida). */
export const candidatesOf = (g: Game, at: number): number[] => digitsOf(candidateMask(g.values, at));

/** Marca as células cujo dígito repete na linha, na coluna ou no bloco. */
export function conflicts(values: number[]): boolean[] {
  const bad = new Array<boolean>(CELLS).fill(false);
  for (let i = 0; i < CELLS; i++) {
    const v = values[i];
    if (!v) continue;
    for (const p of PEERS[i]) {
      if (values[p] === v) {
        bad[i] = true;
        break;
      }
    }
  }
  return bad;
}

/** Quantos de cada dígito ainda faltam (índice 1..9) — é a contagem do teclado. */
export function remainingOf(values: number[]): number[] {
  const left = new Array<number>(10).fill(9);
  left[0] = 0;
  for (const v of values) if (v) left[v]--;
  return left;
}

export const givensCount = (puzzle: number[]): number => puzzle.reduce((n, v) => n + (v ? 1 : 0), 0);

export function isWon(g: Game): boolean {
  for (let i = 0; i < CELLS; i++) if (g.values[i] !== g.solution[i]) return false;
  return true;
}

/** `m:ss` — mesma formatação da Paciência. */
export function fmtTime(seconds: number | null | undefined): string {
  if (seconds == null) return '—';
  return Math.floor(seconds / 60) + ':' + String(seconds % 60).padStart(2, '0');
}

/** 81 caracteres, `.` para vazia — formato curto de fixture e de semeadura. */
export function toLine(grid: number[]): string {
  return grid.map((v) => (v ? String(v) : '.')).join('');
}

export function fromLine(line: string): number[] {
  if (line.length !== CELLS) throw new Error('linha precisa ter 81 caracteres');
  return [...line].map((ch) => (ch >= '1' && ch <= '9' ? Number(ch) : 0));
}

// ── Sorteio determinístico ──────────────────────────────────────────────────

/** PRNG pequeno e estável — é o que faz o tabuleiro do dia ser igual para todos. */
export function mulberry32(seed: number): () => number {
  let a = seed >>> 0;
  return () => {
    a = (a + 0x6d2b79f5) >>> 0;
    let t = Math.imul(a ^ (a >>> 15), 1 | a);
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

export function fnv1a(text: string): number {
  let h = 0x811c9dc5;
  for (let i = 0; i < text.length; i++) {
    h ^= text.charCodeAt(i);
    h = Math.imul(h, 0x01000193);
  }
  return h >>> 0;
}

/** Chave de dia local (`YYYY-MM-DD`) — não usa UTC para não trocar de dia à noite. */
export function dateKey(d: Date): string {
  const p = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`;
}

export const dailySeed = (day: string): number => fnv1a('sudoku-' + day);

/**
 * Dificuldade do tabuleiro do dia: sobe ao longo da semana, como no caderno de
 * passatempos (segunda mais leve, domingo o mais duro).
 */
export function dailyDifficulty(day: string): Difficulty {
  const [y, m, d] = day.split('-').map(Number);
  const dow = new Date(y, m - 1, d).getDay(); // 0 = domingo
  if (dow === 0) return 'especialista';
  if (dow === 1 || dow === 2) return 'facil';
  if (dow === 3 || dow === 4) return 'medio';
  return 'dificil';
}

/** Semente para partida livre — o único lugar com `Math.random`, fora do engine puro. */
export const randomSeed = (): number => (Math.random() * 0x100000000) >>> 0;

/** Embaralha no lugar com o PRNG dado (Fisher-Yates). */
function shuffle<T>(list: T[], rnd: () => number): T[] {
  for (let i = list.length - 1; i > 0; i--) {
    const j = Math.floor(rnd() * (i + 1));
    [list[i], list[j]] = [list[j], list[i]];
  }
  return list;
}

// ── Solver de força (só para contar soluções) ───────────────────────────────

/** Conta soluções até `limit` — é como o gerador garante solução única. */
export function countSolutions(values: number[], limit = 2): number {
  const grid = values.slice();
  return countFrom(grid, limit);
}

function countFrom(grid: number[], limit: number): number {
  let at = -1;
  let best = 10;
  let bestMask = 0;
  for (let i = 0; i < CELLS; i++) {
    if (grid[i]) continue;
    const mask = candidateMask(grid, i);
    const n = popcount(mask);
    if (n === 0) return 0;
    if (n < best) {
      best = n;
      at = i;
      bestMask = mask;
      if (n === 1) break;
    }
  }
  if (at < 0) return 1;
  let found = 0;
  for (const d of digitsOf(bestMask)) {
    grid[at] = d;
    found += countFrom(grid, limit - found);
    grid[at] = 0;
    if (found >= limit) break;
  }
  return found;
}

// ── Solver lógico: é ele que mede a dificuldade e dá a dica ─────────────────

/**
 * Escada de técnicas: 1 só singles · 2 + candidato travado · 3 + pares (nus e
 * escondidos) · 4 + trincas nuas · 5 = não fecha com nada disso.
 */
export type Level = 1 | 2 | 3 | 4 | 5;

type Solved = { values: number[]; solved: boolean; firstStep: Hint | null };

function assign(values: number[], cand: number[], at: number, digit: number): void {
  values[at] = digit;
  cand[at] = 0;
  for (const p of PEERS[at]) cand[p] &= ~BIT(digit);
}

/**
 * Resolve o que der usando só as técnicas até `level`, sem nunca adivinhar.
 * Devolve também o primeiro dígito colocado (e por qual técnica) — é a dica.
 */
export function solveLogically(start: number[], level: Level): Solved {
  const values = start.slice();
  const cand = new Array<number>(CELLS).fill(0);
  for (let i = 0; i < CELLS; i++) cand[i] = values[i] ? 0 : candidateMask(values, i);

  let firstStep: Hint | null = null;
  const note = (at: number, digit: number, technique: Technique) => {
    if (!firstStep) firstStep = { at, digit, technique };
  };

  for (;;) {
    // 1. único candidato possível na célula
    let moved = false;
    for (let i = 0; i < CELLS; i++) {
      if (values[i] || popcount(cand[i]) !== 1) continue;
      const d = digitsOf(cand[i])[0];
      note(i, d, 'cell');
      assign(values, cand, i, d);
      moved = true;
    }
    if (moved) continue;

    // 2. único lugar para o dígito na casa
    for (const unit of UNITS) {
      for (let d = 1; d <= 9; d++) {
        if (unit.some((i) => values[i] === d)) continue;
        let spot = -1;
        let n = 0;
        for (const i of unit) {
          if (!values[i] && cand[i] & BIT(d)) {
            n++;
            spot = i;
          }
        }
        if (n === 1) {
          note(spot, d, 'house');
          assign(values, cand, spot, d);
          moved = true;
        }
      }
    }
    if (moved) continue;

    // 3. eliminações: não colocam dígito, abrem caminho para os singles
    if (level >= 2 && lockedCandidates(values, cand)) continue;
    if (level >= 3 && (nakedPairs(values, cand) || hiddenPairs(values, cand))) continue;
    if (level >= 4 && nakedTriples(values, cand)) continue;
    break;
  }

  return { values, solved: values.every((v) => v > 0), firstStep };
}

/** Duas células da casa com o mesmo par de candidatos tiram esses dois das outras. */
function nakedPairs(values: number[], cand: number[]): boolean {
  let changed = false;
  for (const unit of UNITS) {
    for (let a = 0; a < unit.length; a++) {
      const ia = unit[a];
      if (values[ia] || popcount(cand[ia]) !== 2) continue;
      for (let b = a + 1; b < unit.length; b++) {
        const ib = unit[b];
        if (values[ib] || cand[ib] !== cand[ia]) continue;
        for (const i of unit) {
          if (i === ia || i === ib || values[i]) continue;
          const next = cand[i] & ~cand[ia];
          if (next !== cand[i]) {
            cand[i] = next;
            changed = true;
          }
        }
      }
    }
  }
  return changed;
}

/**
 * Candidato travado: se um dígito só cabe numa linha/coluna dentro do bloco,
 * sai do resto da linha/coluna — e vice-versa (bloco × linha).
 */
function lockedCandidates(values: number[], cand: number[]): boolean {
  let changed = false;
  const strip = (cells: number[], keep: (i: number) => boolean, d: number) => {
    for (const i of cells) {
      if (values[i] || keep(i)) continue;
      const next = cand[i] & ~BIT(d);
      if (next !== cand[i]) {
        cand[i] = next;
        changed = true;
      }
    }
  };

  for (let d = 1; d <= 9; d++) {
    // bloco → linha/coluna
    for (let b = 0; b < 9; b++) {
      const unit = UNITS[18 + b];
      const spots = unit.filter((i) => !values[i] && cand[i] & BIT(d));
      if (spots.length < 2) continue;
      if (spots.every((i) => rowOf(i) === rowOf(spots[0]))) {
        strip(UNITS[rowOf(spots[0])], (i) => boxOf(i) === b, d);
      }
      if (spots.every((i) => colOf(i) === colOf(spots[0]))) {
        strip(UNITS[9 + colOf(spots[0])], (i) => boxOf(i) === b, d);
      }
    }
    // linha/coluna → bloco
    for (let u = 0; u < 18; u++) {
      const unit = UNITS[u];
      const spots = unit.filter((i) => !values[i] && cand[i] & BIT(d));
      if (spots.length < 2) continue;
      const b = boxOf(spots[0]);
      if (spots.every((i) => boxOf(i) === b)) {
        strip(UNITS[18 + b], (i) => unit.includes(i), d);
      }
    }
  }
  return changed;
}

/** Par escondido: dois dígitos que só cabem nas mesmas duas células da casa. */
function hiddenPairs(values: number[], cand: number[]): boolean {
  let changed = false;
  for (const unit of UNITS) {
    for (let d1 = 1; d1 <= 8; d1++) {
      const s1 = unit.filter((i) => !values[i] && cand[i] & BIT(d1));
      if (s1.length !== 2) continue;
      for (let d2 = d1 + 1; d2 <= 9; d2++) {
        const s2 = unit.filter((i) => !values[i] && cand[i] & BIT(d2));
        if (s2.length !== 2 || s2[0] !== s1[0] || s2[1] !== s1[1]) continue;
        const pair = BIT(d1) | BIT(d2);
        for (const i of s1) {
          if (cand[i] !== pair) {
            cand[i] = pair;
            changed = true;
          }
        }
      }
    }
  }
  return changed;
}

/** Trinca nua: três células da casa que, juntas, só aceitam três dígitos. */
function nakedTriples(values: number[], cand: number[]): boolean {
  let changed = false;
  for (const unit of UNITS) {
    const open = unit.filter((i) => !values[i] && popcount(cand[i]) >= 2 && popcount(cand[i]) <= 3);
    for (let a = 0; a < open.length; a++) {
      for (let b = a + 1; b < open.length; b++) {
        for (let c = b + 1; c < open.length; c++) {
          const trio = [open[a], open[b], open[c]];
          const mask = cand[trio[0]] | cand[trio[1]] | cand[trio[2]];
          if (popcount(mask) !== 3) continue;
          for (const i of unit) {
            if (values[i] || trio.includes(i)) continue;
            const next = cand[i] & ~mask;
            if (next !== cand[i]) {
              cand[i] = next;
              changed = true;
            }
          }
        }
      }
    }
  }
  return changed;
}

/** O degrau mais baixo da escada que ainda resolve o tabuleiro inteiro (5 = nenhum). */
export function hardnessOf(puzzle: number[]): Level {
  for (const level of [1, 2, 3, 4] as Level[]) if (solveLogically(puzzle, level).solved) return level;
  return 5;
}

// ── Gerador ─────────────────────────────────────────────────────────────────

/**
 * Faixa de dados (alvo) e o piso de técnica que a dificuldade exige.
 *
 * O piso é um mínimo, não um degrau exato: medindo a geração, tabuleiro com 30
 * dados ou menos quase sempre pede mais do que as quatro técnicas daqui, então
 * exigir o degrau exato só faria o gerador rejeitar à toa. O que separa Difícil
 * de Especialista, então, é o piso **mais** a quantidade de dados na mesa.
 */
const TIERS: Record<
  Difficulty,
  { target: [number, number]; accepts: (h: Level) => boolean; symmetric: boolean }
> = {
  facil: { target: [38, 44], accepts: (h) => h === 1, symmetric: true },
  medio: { target: [30, 34], accepts: (h) => h <= 2, symmetric: true },
  dificil: { target: [26, 30], accepts: (h) => h >= 3, symmetric: false },
  especialista: { target: [22, 25], accepts: (h) => h === 5, symmetric: false },
};

/**
 * Tentativas antes de aceitar o tabuleiro mais duro que apareceu.
 *
 * Medindo: com ~28 dados, 4 de 5 tabuleiros sorteados saem só com singles, logo
 * as faixas duras *precisam* de muitas tentativas — com 14, uma seed em vinte
 * caía no plano B e entregava um Difícil mole. Cada tentativa custa ~2–5 ms.
 */
const MAX_TRIES = 40;

/** Grade completa e válida, sorteada pelo PRNG. */
function fullSolution(rnd: () => number): number[] {
  const grid = new Array<number>(CELLS).fill(0);
  fill(grid, rnd);
  return grid;
}

function fill(grid: number[], rnd: () => number): boolean {
  let at = -1;
  let best = 10;
  let bestMask = 0;
  for (let i = 0; i < CELLS; i++) {
    if (grid[i]) continue;
    const mask = candidateMask(grid, i);
    const n = popcount(mask);
    if (n === 0) return false;
    if (n < best) {
      best = n;
      at = i;
      bestMask = mask;
      if (n === 1) break;
    }
  }
  if (at < 0) return true;
  for (const d of shuffle(digitsOf(bestMask), rnd)) {
    grid[at] = d;
    if (fill(grid, rnd)) return true;
    grid[at] = 0;
  }
  return false;
}

/**
 * Tira células em ordem sorteada enquanto a solução continuar única.
 *
 * Com `symmetric`, tira de dois em dois em posições opostas (giro de 180°) —
 * é o desenho que o sudoku de jornal tem. Nas faixas duras cada remoção vale
 * ouro, então lá a simetria fica de fora.
 */
function carve(solution: number[], rnd: () => number, target: number, symmetric: boolean): number[] {
  const puzzle = solution.slice();
  let givens = CELLS;
  for (const at of shuffle([...Array(CELLS).keys()], rnd)) {
    if (givens <= target) break;
    const twin = CELLS - 1 - at;
    const group = symmetric && twin !== at ? [at, twin] : [at];
    if (group.some((i) => !puzzle[i])) continue;

    const kept = group.map((i) => puzzle[i]);
    for (const i of group) puzzle[i] = 0;
    if (countSolutions(puzzle, 2) === 1) givens -= group.length;
    else group.forEach((i, k) => (puzzle[i] = kept[k]));
  }
  return puzzle;
}

/**
 * Tabuleiro da dificuldade pedida. Gera, classifica e descarta o que não bate
 * com a faixa; se o teto de tentativas estourar, fica com o mais duro que saiu
 * (melhor um Difícil no lugar do Especialista do que travar o celular).
 */
export function generate(difficulty: Difficulty, seed: number): { puzzle: number[]; solution: number[] } {
  const rnd = mulberry32(seed);
  const tier = TIERS[difficulty];
  let fallback: { puzzle: number[]; solution: number[]; hardness: Level } | null = null;

  for (let tries = 0; tries < MAX_TRIES; tries++) {
    const solution = fullSolution(rnd);
    const target = tier.target[0] + Math.floor(rnd() * (tier.target[1] - tier.target[0] + 1));
    const puzzle = carve(solution, rnd, target, tier.symmetric);
    const hardness = hardnessOf(puzzle);
    if (tier.accepts(hardness)) return { puzzle, solution };
    if (!fallback || hardness > fallback.hardness) fallback = { puzzle, solution, hardness };
  }

  return { puzzle: fallback!.puzzle, solution: fallback!.solution };
}

// ── Partida ─────────────────────────────────────────────────────────────────

function gameFrom(
  puzzle: number[],
  solution: number[],
  difficulty: Difficulty,
  seed: number,
  daily: string | null,
): Game {
  return {
    givens: puzzle.slice(),
    values: puzzle.slice(),
    notes: new Array<number>(CELLS).fill(0),
    solution,
    difficulty,
    undo: [],
    hints: 0,
    mistakes: 0,
    elapsed: 0,
    daily,
    seed,
    won: false,
  };
}

export function newGame(difficulty: Difficulty, seed: number): Game {
  const { puzzle, solution } = generate(difficulty, seed);
  return gameFrom(puzzle, solution, difficulty, seed, null);
}

/** Tabuleiro do dia: mesma data ⇒ mesmo tabuleiro em qualquer aparelho. */
export function dailyGame(day: string): Game {
  const seed = dailySeed(day);
  const difficulty = dailyDifficulty(day);
  const { puzzle, solution } = generate(difficulty, seed);
  return gameFrom(puzzle, solution, difficulty, seed, day);
}

/** Empilha o lance respeitando o teto da pilha. */
function pushMove(g: Game, move: Move): Move[] {
  const stack = [...g.undo, move];
  return stack.length > UNDO_LIMIT ? stack.slice(stack.length - UNDO_LIMIT) : stack;
}

/**
 * Escreve um dígito. Tocar o dígito que já está na célula apaga (é o
 * toque-toque que todo sudoku de celular tem). `autoNotes` apaga esse dígito
 * das anotações dos vizinhos.
 */
export function place(g: Game, at: number, digit: number, autoNotes = false): Game {
  if (g.won || g.givens[at]) return g;
  if (g.values[at] === digit) return erase(g, at);

  const values = g.values.slice();
  const notes = g.notes.slice();
  const before: Move['before'] = [[at, g.values[at], g.notes[at]]];
  values[at] = digit;
  notes[at] = 0;

  if (autoNotes) {
    for (const p of PEERS[at]) {
      if (!(notes[p] & BIT(digit))) continue;
      before.push([p, values[p], notes[p]]);
      notes[p] &= ~BIT(digit);
    }
  }

  const next: Game = {
    ...g,
    values,
    notes,
    undo: pushMove(g, { at, before }),
    mistakes: digit === g.solution[at] ? g.mistakes : g.mistakes + 1,
  };
  next.won = isWon(next);
  return next;
}

/** Liga/desliga uma anotação. Célula já preenchida não aceita lápis. */
export function toggleNote(g: Game, at: number, digit: number): Game {
  if (g.won || g.givens[at] || g.values[at]) return g;
  const notes = g.notes.slice();
  const before: Move['before'] = [[at, g.values[at], g.notes[at]]];
  notes[at] ^= BIT(digit);
  return { ...g, notes, undo: pushMove(g, { at, before }) };
}

/** Apaga o dígito e as anotações da célula. */
export function erase(g: Game, at: number): Game {
  if (g.won || g.givens[at] || (!g.values[at] && !g.notes[at])) return g;
  const values = g.values.slice();
  const notes = g.notes.slice();
  const before: Move['before'] = [[at, g.values[at], g.notes[at]]];
  values[at] = 0;
  notes[at] = 0;
  return { ...g, values, notes, undo: pushMove(g, { at, before }) };
}

/** Desfaz o último lance (dica inclusa — ela também empilha). */
export function undo(g: Game): Game {
  if (!g.undo.length) return g;
  const stack = g.undo.slice();
  const last = stack.pop()!;
  const values = g.values.slice();
  const notes = g.notes.slice();
  for (const [at, value, noted] of last.before) {
    values[at] = value;
    notes[at] = noted;
  }
  return { ...g, values, notes, undo: stack, won: false };
}

/** Células onde o jogador escreveu um dígito diferente da solução. */
export function wrongCells(g: Game): number[] {
  const out: number[] = [];
  for (let i = 0; i < CELLS; i++) if (g.values[i] && g.values[i] !== g.solution[i]) out.push(i);
  return out;
}

/**
 * Próxima célula dedutível, com a técnica usada.
 *
 * Roda sobre o tabuleiro **sem os dígitos errados do jogador** — um número
 * errado deixado na grade envenenaria a dedução e a dica viraria mentira.
 * Quando nenhuma técnica aplica, revela a célula mais restrita.
 */
export function hint(g: Game): Hint | null {
  const clean = g.values.map((v, i) => (v === g.solution[i] ? v : 0));
  if (clean.every((v) => v > 0)) return null;

  const step = solveLogically(clean, 4).firstStep;
  if (step) return step;

  let at = -1;
  let best = 10;
  for (let i = 0; i < CELLS; i++) {
    if (clean[i]) continue;
    const n = popcount(candidateMask(clean, i));
    if (n < best) {
      best = n;
      at = i;
    }
  }
  return at < 0 ? null : { at, digit: g.solution[at], technique: 'revelada' };
}

/** Aplica a dica: escreve o dígito e conta a ajuda. */
export function useHint(g: Game, h: Hint): Game {
  const next = place(g, h.at, h.digit, true);
  if (next === g) return g;
  return { ...next, hints: g.hints + 1, mistakes: g.mistakes };
}
