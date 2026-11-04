/**
 * Cobrinha — regras puras.
 *
 * Sem DOM, sem canvas: grade, movimento, comida, obstáculos e pontuação. O
 * desenho fica em `render.ts` e o laço do jogo em `src/routes/cobrinha/+page.svelte`.
 *
 * Cada passo devolve um estado novo (como o `lock` do Tetris), então o teste
 * compara entrada e saída sem precisar clonar nada na mão.
 *
 * Pontuação: fruta = 10 × nível · fruta bônus = 50 × nível. O nível sobe a cada
 * 5 frutas e encurta o intervalo entre os passos.
 */

export const COLS = 15;
export const ROWS = 19;

/** A cada quantas frutas aparece uma bônus, e por quantos passos ela fica. */
export const BONUS_EVERY = 5;
export const BONUS_STEPS = 40;

export type Dir = 'up' | 'down' | 'left' | 'right';
export type Point = { x: number; y: number };
export type Speed = 'calma' | 'normal' | 'veloz';

/** Evento do passo, para a tela saber o que animar. */
export type StepEvent = 'ate' | 'bonus' | 'died' | null;

export type Game = {
  /** Cabeça primeiro. */
  snake: Point[];
  dir: Dir;
  /** Curvas pedidas e ainda não aplicadas (no máximo 2). */
  queued: Dir[];
  fruit: Point;
  bonus: Point | null;
  /** Passos que restam para a bônus sumir. */
  bonusLeft: number;
  walls: Point[];
  score: number;
  eaten: number;
  over: boolean;
  wrap: boolean;
  speed: Speed;
  /** `YYYY-MM-DD` quando é o desafio do dia. */
  daily: string | null;
  /** Pontos necessários no desafio do dia (0 na partida livre). */
  goal: number;
  /** Estado do PRNG — é o que faz o desafio do dia ser igual para todo mundo. */
  rng: number;
};

const OPPOSITE: Record<Dir, Dir> = { up: 'down', down: 'up', left: 'right', right: 'left' };
const DELTA: Record<Dir, Point> = {
  up: { x: 0, y: -1 },
  down: { x: 0, y: 1 },
  left: { x: -1, y: 0 },
  right: { x: 1, y: 0 },
};

// ── Sorteio determinístico ──────────────────────────────────────────────────
// Cópia curta de propósito: importar do engine do Enche Aí! arrastaria o solver
// dele para o chunk desta rota.

/** PRNG determinístico (mulberry32). */
export function mulberry32(seed: number): () => number {
  let a = seed >>> 0;
  return () => {
    a = (a + 0x6d2b79f5) >>> 0;
    let t = Math.imul(a ^ (a >>> 15), 1 | a);
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

/** FNV-1a — transforma a data (ou qualquer texto) na seed. */
export function fnv1a(s: string): number {
  let h = 0x811c9dc5;
  for (let i = 0; i < s.length; i++) {
    h ^= s.charCodeAt(i);
    h = Math.imul(h, 0x01000193);
  }
  return h >>> 0;
}

/** `YYYY-MM-DD` no fuso local (o dia do jogador, não o UTC). */
export function dateKey(d: Date): string {
  const p = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`;
}

export const dailySeed = (day: string): number => fnv1a('cobrinha-' + day);

/** Meta de pontos do desafio do dia — varia, mas é a mesma para todo mundo. */
export const dailyGoal = (day: string): number => 150 + (dailySeed(day) % 7) * 25;

/** Um número do PRNG, devolvido junto com o estado seguinte. */
function roll(rng: number): { value: number; rng: number } {
  const next = (rng + 0x6d2b79f5) >>> 0;
  let t = Math.imul(next ^ (next >>> 15), 1 | next);
  t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
  return { value: ((t ^ (t >>> 14)) >>> 0) / 4294967296, rng: next };
}

// ── Consultas ───────────────────────────────────────────────────────────────

export const samePoint = (a: Point, b: Point): boolean => a.x === b.x && a.y === b.y;

const has = (list: Point[], p: Point): boolean => list.some((q) => samePoint(q, p));

export const levelFor = (eaten: number): number => Math.floor(eaten / 5) + 1;

const BASE_MS: Record<Speed, number> = { calma: 220, normal: 180, veloz: 145 };
const FLOOR_MS: Record<Speed, number> = { calma: 115, normal: 85, veloz: 65 };

/** Intervalo entre passos: cai 12ms por nível até o piso da velocidade. */
export const tickMs = (level: number, speed: Speed): number =>
  Math.max(FLOOR_MS[speed], BASE_MS[speed] - (level - 1) * 12);

export const scoreForFruit = (level: number): number => 10 * level;
export const scoreForBonus = (level: number): number => 50 * level;

/** `1 615` — separador fino para o placar não virar um borrão de dígitos. */
export const fmtScore = (n: number): string => String(n).replace(/\B(?=(\d{3})+(?!\d))/g, ' ');

type Board = Pick<Game, 'snake' | 'walls' | 'fruit' | 'bonus'>;

/** Células livres: fora da cobra, dos obstáculos e das comidas na mesa. */
function freeCells(b: Board, ignoreFruit: boolean): Point[] {
  const out: Point[] = [];
  for (let y = 0; y < ROWS; y++) {
    for (let x = 0; x < COLS; x++) {
      const p = { x, y };
      if (has(b.snake, p) || has(b.walls, p)) continue;
      if (!ignoreFruit && samePoint(b.fruit, p)) continue;
      if (b.bonus && samePoint(b.bonus, p)) continue;
      out.push(p);
    }
  }
  return out;
}

/** Sorteia uma célula livre; `null` quando a cobra tomou o tabuleiro inteiro. */
function pickCell(b: Board, rng: number, ignoreFruit = false): { cell: Point | null; rng: number } {
  const cells = freeCells(b, ignoreFruit);
  if (cells.length === 0) return { cell: null, rng };
  const { value, rng: next } = roll(rng);
  return { cell: cells[Math.floor(value * cells.length)], rng: next };
}

// ── Montagem da partida ─────────────────────────────────────────────────────

/**
 * Obstáculos do desafio do dia: blocos de 2 a 3 células, longe das bordas e da
 * faixa onde a cobra nasce, para o dia nunca começar perdido.
 */
export function wallsFor(day: string): Point[] {
  const rnd = mulberry32(dailySeed(day) ^ 0x9e3779b9);
  const midY = Math.floor(ROWS / 2);
  const walls: Point[] = [];
  const blocks = 2 + Math.floor(rnd() * 4);
  for (let i = 0; i < blocks; i++) {
    for (let tries = 0; tries < 20; tries++) {
      const horizontal = rnd() < 0.5;
      const len = 2 + Math.floor(rnd() * 2);
      const x = 2 + Math.floor(rnd() * (COLS - 4 - (horizontal ? len : 0)));
      const y = 2 + Math.floor(rnd() * (ROWS - 4 - (horizontal ? 0 : len)));
      const block: Point[] = [];
      for (let k = 0; k < len; k++) block.push(horizontal ? { x: x + k, y } : { x, y: y + k });
      // A faixa da cabeça (3 linhas no meio) fica livre, e os blocos não se encostam.
      const clear = block.every(
        (p) =>
          Math.abs(p.y - midY) > 1 &&
          !walls.some((w) => Math.abs(w.x - p.x) + Math.abs(w.y - p.y) <= 1)
      );
      if (clear) {
        walls.push(...block);
        break;
      }
    }
  }
  return walls;
}

export type NewGameOptions = {
  wrap?: boolean;
  speed?: Speed;
  /** `YYYY-MM-DD` para o desafio do dia; ausente para partida livre. */
  daily?: string | null;
  /** Seed da partida livre (os testes passam; em jogo é sorteada). */
  seed?: number;
};

export function newGame(options: NewGameOptions = {}): Game {
  const daily = options.daily ?? null;
  const seed = daily ? dailySeed(daily) : (options.seed ?? Math.floor(Math.random() * 0xffffffff));
  const y = Math.floor(ROWS / 2);
  const x = Math.floor(COLS / 2);
  const game: Game = {
    snake: [
      { x, y },
      { x: x - 1, y },
      { x: x - 2, y },
    ],
    dir: 'right',
    queued: [],
    fruit: { x: -1, y: -1 },
    bonus: null,
    bonusLeft: 0,
    walls: daily ? wallsFor(daily) : [],
    score: 0,
    eaten: 0,
    over: false,
    // O desafio do dia ignora os ajustes: todo mundo joga a mesma partida.
    wrap: daily ? false : (options.wrap ?? false),
    speed: daily ? 'normal' : (options.speed ?? 'normal'),
    daily,
    goal: daily ? dailyGoal(daily) : 0,
    rng: seed,
  };
  const { cell, rng } = pickCell(game, game.rng, true);
  game.fruit = cell ?? { x: 0, y: 0 };
  game.rng = rng;
  return game;
}

/**
 * Enfileira uma curva. Ignora a repetida e a inversão de 180° — sem a fila,
 * dois toques dentro do mesmo passo fariam a cobra virar para dentro de si.
 */
export function turn(g: Game, dir: Dir): Game {
  if (g.over) return g;
  const last = g.queued.length ? g.queued[g.queued.length - 1] : g.dir;
  if (dir === last || dir === OPPOSITE[last]) return g;
  if (g.queued.length >= 2) return g;
  return { ...g, queued: [...g.queued, dir] };
}

/**
 * A célula mata a cabeça? Fora da grade só mata com a travessia desligada, e a
 * ponta da cauda não conta: ela sai do lugar no mesmo passo.
 */
export function isDeadly(g: Game, p: Point): boolean {
  if (p.x < 0 || p.x >= COLS || p.y < 0 || p.y >= ROWS) return !g.wrap;
  if (has(g.walls, p)) return true;
  return has(g.snake.slice(0, -1), p);
}

const wrapPoint = (p: Point): Point => ({ x: (p.x + COLS) % COLS, y: (p.y + ROWS) % ROWS });

/** Um passo do relógio: anda, come, cresce ou morre. */
export function step(g: Game): { game: Game; event: StepEvent } {
  if (g.over) return { game: g, event: null };

  const queued = [...g.queued];
  const dir = queued.shift() ?? g.dir;
  const delta = DELTA[dir];
  const ahead = { x: g.snake[0].x + delta.x, y: g.snake[0].y + delta.y };
  const head = g.wrap ? wrapPoint(ahead) : ahead;

  if (isDeadly(g, g.wrap ? head : ahead)) {
    return { game: { ...g, dir, queued, over: true }, event: 'died' };
  }

  const ate = samePoint(head, g.fruit);
  const tookBonus = g.bonus !== null && samePoint(head, g.bonus);

  // Comeu: a cauda fica no lugar e a cobra ganha um segmento.
  const snake = [head, ...g.snake];
  if (!ate) snake.pop();

  const next: Game = { ...g, snake, dir, queued };

  if (ate) {
    next.eaten = g.eaten + 1;
    next.score = g.score + scoreForFruit(levelFor(g.eaten));
    const spawned = pickCell(next, next.rng, true);
    next.rng = spawned.rng;
    if (spawned.cell) next.fruit = spawned.cell;
    else next.over = true; // tabuleiro tomado: não há onde pôr a próxima fruta
    // Bônus a cada 5 frutas, se não houver uma na mesa.
    if (!next.over && next.eaten % BONUS_EVERY === 0 && !next.bonus) {
      const extra = pickCell(next, next.rng);
      next.rng = extra.rng;
      if (extra.cell) {
        next.bonus = extra.cell;
        next.bonusLeft = BONUS_STEPS;
      }
    }
  }

  if (tookBonus) {
    next.score = next.score + scoreForBonus(levelFor(next.eaten));
    next.bonus = null;
    next.bonusLeft = 0;
  } else if (g.bonus && next.bonus) {
    // Só envelhece a bônus que já estava na mesa — a que acabou de nascer, não.
    next.bonusLeft = g.bonusLeft - 1;
    if (next.bonusLeft <= 0) {
      next.bonus = null;
      next.bonusLeft = 0;
    }
  }

  return { game: next, event: next.over ? 'died' : tookBonus ? 'bonus' : ate ? 'ate' : null };
}

/** O desafio do dia foi cumprido nesta partida? */
export const reachedGoal = (g: Game): boolean => g.goal > 0 && g.score >= g.goal;
