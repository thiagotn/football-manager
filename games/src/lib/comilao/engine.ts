/**
 * Comilão — regras puras.
 *
 * Sem DOM: labirinto, movimento, pastilhas, fantasmas e pontuação. O desenho
 * fica em `render.ts` e o laço do jogo em `src/routes/comilao/+page.svelte`.
 *
 * Diferente da Cobrinha, aqui **não é passo a passo na grade**: Comilão e
 * fantasmas têm posição contínua em unidades de tile e só trocam de direção no
 * centro de um tile. É isso que dá a sensação do arcade — dobrar a esquina
 * rente e ver o fantasma fechar o corredor.
 *
 * `tick` é chamado sempre com o mesmo `dt` (a rota acumula o tempo e chama em
 * passos fixos), então a física não depende da taxa de quadros e o teste pode
 * rodar uma partida inteira comparando números exatos.
 */

import {
  COLS,
  DOOR,
  EMPTY,
  FRUIT_TILE,
  GATE,
  HERO_START,
  HOUSE,
  PELLET_TOTAL,
  POWER,
  ROWS,
  initialTiles,
  inTunnel,
  walkable,
  wrapCol,
} from './maze';

export { COLS, PELLET_TOTAL, ROWS } from './maze';

export type Dir = 'up' | 'down' | 'left' | 'right';
export type GhostId = 'cacador' | 'emboscador' | 'flanqueador' | 'ermitao';
/** `casa` espera a vez · `saindo` sobe pela porta · `olhos` volta depois de ser comido. */
export type GhostState = 'casa' | 'saindo' | 'solto' | 'assustado' | 'olhos';
export type Phase = 'pronto' | 'jogando' | 'morrendo' | 'fase' | 'fim';
export type Pace = 'calma' | 'normal' | 'veloz';
export type Tile = { col: number; row: number };

/** Posição em unidades de tile (o centro do tile `c` é `c + 0.5`). */
export type Actor = {
  x: number;
  y: number;
  dir: Dir;
  /** Curva pedida e ainda não aplicada — entra no próximo centro que der. */
  next: Dir | null;
};

export type Ghost = Actor & {
  id: GhostId;
  state: GhostState;
  /** ms que faltam na casa antes de sair. */
  wait: number;
  /** Último tile em que decidiu o caminho (para decidir uma vez por tile). */
  tile: number;
};

export type Fruit = { kind: number; left: number };

export type Game = {
  level: number;
  /** Cópia mutável do labirinto: 0 vazio · 1 pastilha · 2 pastilhão. */
  tiles: number[];
  /** Pastilhas que faltam na fase. */
  left: number;
  /** Pastilhas comidas na fase (é o que chama a fruta). */
  eaten: number;
  /** Pastilhas comidas na partida inteira. */
  pellets: number;
  /** Fantasmas comidos na partida inteira. */
  ghostsEaten: number;
  hero: Actor;
  ghosts: Ghost[];
  /** Índice na tabela de ondas (scatter/chase). */
  wave: number;
  waveLeft: number;
  /** ms de susto que restam (0 = fantasmas soltos). */
  frightLeft: number;
  /** Fantasmas comidos no susto corrente (0..4). */
  combo: number;
  fruit: Fruit | null;
  /** Frutas colhidas, para a cestinha do HUD. */
  basket: number[];
  lives: number;
  score: number;
  /** Próximo marco de vida extra. */
  nextLife: number;
  phase: Phase;
  /** ms que faltam na pausa da fase corrente (`pronto`, `morrendo`, `fase`). */
  phaseLeft: number;
  /** Estado do PRNG — é o que faz o desafio do dia ser igual para todo mundo. */
  rng: number;
  pace: Pace;
  /** `YYYY-MM-DD` quando é o desafio do dia. */
  daily: string | null;
  /** Pontos necessários no desafio do dia (0 na partida livre). */
  goal: number;
};

/** O que aconteceu no passo — a tela usa para o HUD, o aviso e o som futuro. */
export type Event =
  | { kind: 'pastilha' }
  | { kind: 'pastilhao' }
  | { kind: 'fantasma'; points: number; x: number; y: number }
  | { kind: 'fruta'; points: number; x: number; y: number }
  | { kind: 'vida' }
  | { kind: 'morreu' }
  | { kind: 'fase'; level: number }
  | { kind: 'fim' };

// ── Constantes de ritmo ─────────────────────────────────────────────────────

/** Pausas: "Prepare-se", a morte e a virada de fase. */
export const READY_MS = 1800;
export const DEATH_MS = 1500;
export const CLEAR_MS = 1600;
/** Quanto tempo a fruta fica na mesa. */
export const FRUIT_MS = 9000;
/** Tempo parado na casa depois de ser comido. */
export const HOME_MS = 700;
/** Vida extra a cada tanto de pontos. */
export const LIFE_EVERY = 10000;
export const START_LIVES = 3;

export const DIRS: Dir[] = ['up', 'left', 'down', 'right'];
const DELTA: Record<Dir, Tile> = {
  up: { col: 0, row: -1 },
  down: { col: 0, row: 1 },
  left: { col: -1, row: 0 },
  right: { col: 1, row: 0 },
};
const OPPOSITE: Record<Dir, Dir> = { up: 'down', down: 'up', left: 'right', right: 'left' };

/** Cantos do `scatter`: cada fantasma corre para o seu e perde o jogador de vista. */
const CORNERS: Record<GhostId, Tile> = {
  cacador: { col: COLS - 2, row: 1 },
  emboscador: { col: 1, row: 1 },
  flanqueador: { col: COLS - 2, row: ROWS - 2 },
  ermitao: { col: 1, row: ROWS - 2 },
};

/** Espera de cada fantasma na casa, no começo da fase (ms na fase 1). */
const RELEASE: Record<GhostId, number> = {
  cacador: 0,
  emboscador: 1200,
  flanqueador: 5000,
  ermitao: 9000,
};

/** Ondas: a pressão vem em ciclos, senão não há como respirar. */
const WAVES: { mode: 'scatter' | 'chase'; ms: number }[] = [
  { mode: 'scatter', ms: 7000 },
  { mode: 'chase', ms: 20000 },
  { mode: 'scatter', ms: 7000 },
  { mode: 'chase', ms: 20000 },
  { mode: 'scatter', ms: 5000 },
  { mode: 'chase', ms: 20000 },
  { mode: 'scatter', ms: 5000 },
  { mode: 'chase', ms: Number.POSITIVE_INFINITY },
];

/** Duração do susto por fase — da fase 19 em diante o pastilhão não assusta mais. */
const FRIGHT_MS = [7000, 6000, 5000, 4000, 3000, 3000, 2000, 2000, 1000, 1000, 1000, 1000, 1000, 1000, 1000, 1000, 1000, 1000];

/** Frutas: valor por fase (a oitava vale para todas as seguintes). */
export const FRUITS = [100, 300, 500, 700, 1000, 2000, 3000, 5000];

const PACE: Record<Pace, number> = { calma: 0.9, normal: 1, veloz: 1.1 };

export const POINTS_PELLET = 10;
export const POINTS_POWER = 50;
/** 200 · 400 · 800 · 1600 no mesmo susto. */
export const ghostPoints = (combo: number): number => 200 * 2 ** Math.max(0, combo - 1);

// ── Sorteio determinístico ──────────────────────────────────────────────────
// Cópia curta de propósito: importar do engine de outro jogo arrastaria o chunk
// dele para esta rota.

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

export const dailySeed = (day: string): number => fnv1a('comilao-' + day);

/**
 * Meta de pontos do desafio do dia — varia, mas é a mesma para todo mundo.
 * A faixa saiu de medição: limpar a fase 1 inteira dá ~2 300 pontos, então
 * 1 200–2 400 é meta de partida boa, não de partida perfeita.
 */
export const dailyGoal = (day: string): number => 1200 + (dailySeed(day) % 9) * 150;

/** Um número do PRNG, devolvido junto com o estado seguinte. */
function roll(rng: number): { value: number; rng: number } {
  const next = (rng + 0x6d2b79f5) >>> 0;
  let t = Math.imul(next ^ (next >>> 15), 1 | next);
  t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
  return { value: ((t ^ (t >>> 14)) >>> 0) / 4294967296, rng: next };
}

// ── Consultas ───────────────────────────────────────────────────────────────

export const tileOf = (v: number): number => Math.floor(v);
export const centerOf = (tile: number): number => tile + 0.5;

/** Tile em que o ator está. */
export const tileAt = (a: Actor): Tile => ({ col: tileOf(a.x), row: tileOf(a.y) });

/** `1 615` — separador fino para o placar não virar um borrão de dígitos. */
export const fmtScore = (n: number): string => String(n).replace(/\B(?=(\d{3})+(?!\d))/g, ' ');

/** Velocidade do Comilão em tiles por segundo: sobe com a fase e estaciona. */
export const heroSpeed = (level: number): number => 7.6 + Math.min(level - 1, 6) * 0.3;

/** Velocidade do fantasma: fração da do herói, com o susto e o túnel pesando. */
export function ghostSpeed(level: number, state: GhostState, tunnel: boolean, pace: Pace): number {
  const hero = heroSpeed(level);
  const base = hero * (0.76 + Math.min(level - 1, 8) * 0.03);
  if (state === 'olhos') return hero * 1.7;
  if (state === 'casa' || state === 'saindo') return hero * 0.6;
  if (state === 'assustado') return base * 0.55;
  // O túnel é o esconderijo: lá o fantasma arrasta o pé.
  return base * (tunnel ? 0.45 : PACE[pace]);
}

/** Duração do susto na fase (0 da fase 19 em diante). */
export const frightMs = (level: number): number => FRIGHT_MS[level - 1] ?? 0;

/** Valor da fruta da fase. */
export const fruitPoints = (level: number): number =>
  FRUITS[Math.min(level, FRUITS.length) - 1] ?? FRUITS[FRUITS.length - 1];

export const waveMode = (wave: number): 'scatter' | 'chase' =>
  WAVES[Math.min(wave, WAVES.length - 1)].mode;

/** O desafio do dia foi cumprido nesta partida? */
export const reachedGoal = (g: Game): boolean => g.goal > 0 && g.score >= g.goal;

/** Distância entre dois pontos, contando a volta pela travessia. */
function spanTo(ax: number, ay: number, bx: number, by: number): number {
  const dx = Math.abs(ax - bx);
  const wrapped = Math.min(dx, COLS - dx);
  return Math.hypot(wrapped, ay - by);
}

// ── Montagem da partida ─────────────────────────────────────────────────────

function newGhost(id: GhostId, level: number): Ghost {
  const inHouse = id !== 'cacador';
  // A espera encurta com a fase: no fim, a casa esvazia quase junto.
  const wait = RELEASE[id] * Math.max(0.3, 1 - (level - 1) * 0.15);
  const col = id === 'flanqueador' ? HOUSE.col - 1 : id === 'ermitao' ? HOUSE.col + 1 : HOUSE.col;
  return {
    id,
    x: centerOf(inHouse ? col : GATE.col),
    y: centerOf(inHouse ? HOUSE.row : GATE.row),
    dir: inHouse ? 'up' : 'left',
    next: null,
    state: inHouse ? 'casa' : 'solto',
    wait: inHouse ? wait : 0,
    tile: -1,
  };
}

export const GHOST_IDS: GhostId[] = ['cacador', 'emboscador', 'flanqueador', 'ermitao'];

/** Devolve os atores ao lugar — começo de fase e volta depois de perder vida. */
function resetActors(g: Game): void {
  g.hero = { x: centerOf(HERO_START.col), y: centerOf(HERO_START.row), dir: 'left', next: null };
  g.ghosts = GHOST_IDS.map((id) => newGhost(id, g.level));
  g.wave = 0;
  g.waveLeft = WAVES[0].ms;
  g.frightLeft = 0;
  g.combo = 0;
  g.fruit = null;
  g.phase = 'pronto';
  g.phaseLeft = READY_MS;
}

export type NewGameOptions = {
  /** `YYYY-MM-DD` para o desafio do dia; ausente para partida livre. */
  daily?: string | null;
  pace?: Pace;
  /** Seed da partida livre (os testes passam; em jogo é sorteada). */
  seed?: number;
  /** Fase inicial — só os testes usam. */
  level?: number;
};

export function newGame(options: NewGameOptions = {}): Game {
  const daily = options.daily ?? null;
  const seed = daily ? dailySeed(daily) : (options.seed ?? Math.floor(Math.random() * 0xffffffff));
  const tiles = initialTiles();
  const game: Game = {
    level: options.level ?? 1,
    tiles,
    left: PELLET_TOTAL,
    eaten: 0,
    pellets: 0,
    ghostsEaten: 0,
    hero: { x: 0, y: 0, dir: 'left', next: null },
    ghosts: [],
    wave: 0,
    waveLeft: WAVES[0].ms,
    frightLeft: 0,
    combo: 0,
    fruit: null,
    basket: [],
    lives: START_LIVES,
    score: 0,
    nextLife: LIFE_EVERY,
    phase: 'pronto',
    phaseLeft: READY_MS,
    rng: seed,
    // O desafio do dia ignora os ajustes: todo mundo joga a mesma partida.
    pace: daily ? 'normal' : (options.pace ?? 'normal'),
    daily,
    goal: daily ? dailyGoal(daily) : 0,
  };
  resetActors(game);
  return game;
}

/** A partida do dia — mesma para todo mundo, em qualquer aparelho. */
export const dailyGame = (day: string): Game => newGame({ daily: day });

/** Fase nova: labirinto cheio outra vez, atores no lugar, tudo mais rápido. */
function nextLevel(g: Game): void {
  g.level += 1;
  g.tiles = initialTiles();
  g.left = PELLET_TOTAL;
  g.eaten = 0;
  resetActors(g);
}

// ── Entrada ─────────────────────────────────────────────────────────────────

/**
 * Pede uma curva. Fica guardada até caber: no toque é o que perdoa o jogador
 * que pede a curva antes da esquina.
 */
export function turn(g: Game, dir: Dir): Game {
  if (g.phase === 'fim' || g.phase === 'morrendo') return g;
  return { ...g, hero: { ...g.hero, next: dir } };
}

// ── Movimento ───────────────────────────────────────────────────────────────

const canGo = (a: Actor, dir: Dir, ghost: boolean, doorOk: boolean): boolean => {
  const col = tileOf(a.x);
  const row = tileOf(a.y);
  const d = DELTA[dir];
  const nextRow = row + d.row;
  const nextCol = wrapCol(col + d.col);
  if (!doorOk && nextCol === DOOR.col && nextRow === DOOR.row) return false;
  return walkable(nextCol, nextRow, ghost);
};

/** Posição no eixo em que o ator anda. */
const axisOf = (a: Actor): number => (a.dir === 'left' || a.dir === 'right' ? a.x : a.y);

/** O ator está parado em cima do centro de um tile? */
const onCenterNow = (a: Actor): boolean =>
  Math.abs(axisOf(a) - centerOf(tileOf(axisOf(a)))) < 1e-6;

/** Distância até o próximo centro à frente (sempre maior que zero). */
function toNextCenter(a: Actor): number {
  const pos = axisOf(a);
  const center = centerOf(tileOf(pos));
  const forward = a.dir === 'right' || a.dir === 'down' ? 1 : -1;
  const ahead = (center - pos) * forward;
  // No centro ou já passado dele: o próximo é o do tile seguinte.
  return ahead > 1e-6 ? ahead : ahead + 1;
}

/** Aplica a curva pedida, se ela couber. */
function resolveTurn(a: Actor, ghost: boolean, doorOk: boolean): void {
  if (!a.next) return;
  if (a.next === a.dir) {
    a.next = null;
    return;
  }
  if (canGo(a, a.next, ghost, doorOk)) {
    a.dir = a.next;
    a.next = null;
  }
}

/**
 * Anda `dist` tiles. Curva e parede são resolvidas **em cima de cada centro de
 * tile** — inclusive o centro onde o ator já está, que é o caso de quem parou
 * encostado na parede no quadro anterior e espera a próxima ordem.
 * `onCenter` é onde o fantasma escolhe o caminho (o Comilão não usa).
 */
function advance(
  a: Actor,
  dist: number,
  ghost: boolean,
  doorOk: boolean,
  onCenter?: (a: Actor) => void
): void {
  let left = dist;
  for (let guard = 0; guard < 16; guard++) {
    if (onCenterNow(a)) {
      // Encaixa na grade: o eixo perpendicular nunca sai do meio do corredor.
      a.x = centerOf(tileOf(a.x));
      a.y = centerOf(tileOf(a.y));
      onCenter?.(a);
      resolveTurn(a, ghost, doorOk);
      if (!canGo(a, a.dir, ghost, doorOk)) return; // encostou: para no centro
    }
    if (left <= 1e-9) return;

    const span = toNextCenter(a);
    const go = Math.min(left, span);
    const d = DELTA[a.dir];
    a.x = wrapAxis(a.x + d.col * go);
    a.y += d.row * go;
    left -= go;
    if (go < span - 1e-6) return; // parou antes de chegar ao centro
  }
}

/** A coluna dá a volta; o `% COLS` tem de valer para fração também. */
const wrapAxis = (x: number): number => ((x % COLS) + COLS) % COLS;

// ── Fantasmas ───────────────────────────────────────────────────────────────

/** Para onde esse fantasma está correndo agora. */
export function targetOf(g: Game, ghost: Ghost): Tile {
  if (ghost.state === 'olhos') return { col: HOUSE.col, row: HOUSE.row };
  if (ghost.state === 'assustado') return CORNERS[ghost.id];
  if (waveMode(g.wave) === 'scatter') return CORNERS[ghost.id];

  const hero = tileAt(g.hero);
  const ahead = DELTA[g.hero.dir];
  switch (ghost.id) {
    // Cola no jogador: é a pressão que não alivia.
    case 'cacador':
      return hero;
    // Corta o caminho quatro tiles à frente, em vez de seguir atrás.
    case 'emboscador':
      return { col: hero.col + ahead.col * 4, row: hero.row + ahead.row * 4 };
    // Espelha o caçador através de dois tiles à frente do herói: fecha o outro lado.
    case 'flanqueador': {
      const pivot = { col: hero.col + ahead.col * 2, row: hero.row + ahead.row * 2 };
      const chaser = g.ghosts.find((x) => x.id === 'cacador') ?? ghost;
      return { col: pivot.col * 2 - tileOf(chaser.x), row: pivot.row * 2 - tileOf(chaser.y) };
    }
    // Persegue de longe e desiste de perto: é o imprevisível.
    default:
      return spanTo(ghost.x, ghost.y, g.hero.x, g.hero.y) > 8 ? hero : CORNERS[ghost.id];
  }
}

/**
 * Escolhe a direção no centro do tile: entre as saídas livres (menos a volta
 * atrás), a que mais aproxima do alvo. Empate resolvido em ordem fixa, então a
 * perseguição é determinística e dá para testar.
 */
function chooseDir(g: Game, ghost: Ghost, pick: () => number): Dir {
  const doorOk = ghost.state === 'olhos';
  const options = DIRS.filter(
    (d) => d !== OPPOSITE[ghost.dir] && canGo(ghost, d, true, doorOk)
  );
  if (options.length === 0) return OPPOSITE[ghost.dir];
  if (options.length === 1) return options[0];
  // Assustado, o fantasma não calcula: corre para qualquer lado.
  if (ghost.state === 'assustado') return options[Math.floor(pick() * options.length)];

  const target = targetOf(g, ghost);
  let best = options[0];
  let bestDist = Number.POSITIVE_INFINITY;
  for (const dir of options) {
    const d = DELTA[dir];
    const col = wrapCol(tileOf(ghost.x) + d.col);
    const row = tileOf(ghost.y) + d.row;
    const dist = spanTo(col, row, target.col, target.row);
    if (dist < bestDist - 1e-9) {
      best = dir;
      bestDist = dist;
    }
  }
  return best;
}

/** Sobe pela porta: primeiro alinha no meio da casa, depois sai. */
function leaveHouse(ghost: Ghost, dist: number): void {
  const cx = centerOf(HOUSE.col);
  if (Math.abs(ghost.x - cx) > 1e-6) {
    const go = Math.min(dist, Math.abs(cx - ghost.x));
    ghost.dir = cx > ghost.x ? 'right' : 'left';
    ghost.x += Math.sign(cx - ghost.x) * go;
    return;
  }
  ghost.dir = 'up';
  ghost.y -= dist;
  const out = centerOf(GATE.row);
  if (ghost.y <= out) {
    ghost.y = out;
    ghost.state = 'solto';
    ghost.dir = 'left';
    ghost.next = null;
    ghost.tile = -1;
  }
}

// ── Passo do relógio ────────────────────────────────────────────────────────

const clone = (g: Game): Game => ({
  ...g,
  hero: { ...g.hero },
  ghosts: g.ghosts.map((x) => ({ ...x })),
});

/**
 * Um passo do jogo. `dt` em ms, sempre o mesmo valor (a rota acumula o tempo).
 */
export function tick(g: Game, dt: number): { game: Game; events: Event[] } {
  const events: Event[] = [];
  if (g.phase === 'fim') return { game: g, events };

  const next = clone(g);
  let rng = next.rng;
  const pick = (): number => {
    const r = roll(rng);
    rng = r.rng;
    return r.value;
  };

  // ── Pausas ────────────────────────────────────────────────────────────────
  if (next.phase !== 'jogando') {
    next.phaseLeft -= dt;
    if (next.phaseLeft > 0) {
      next.rng = rng;
      return { game: next, events };
    }
    if (next.phase === 'pronto') {
      next.phase = 'jogando';
      next.phaseLeft = 0;
    } else if (next.phase === 'morrendo') {
      if (next.lives <= 0) {
        next.phase = 'fim';
        events.push({ kind: 'fim' });
        next.rng = rng;
        return { game: next, events };
      }
      resetActors(next);
      next.rng = rng;
      return { game: next, events };
    } else if (next.phase === 'fase') {
      nextLevel(next);
      events.push({ kind: 'fase', level: next.level });
      next.rng = rng;
      return { game: next, events };
    }
  }

  const seconds = dt / 1000;

  // ── Ondas e susto ─────────────────────────────────────────────────────────
  if (next.frightLeft > 0) {
    next.frightLeft -= dt;
    if (next.frightLeft <= 0) {
      next.frightLeft = 0;
      next.combo = 0;
      for (const ghost of next.ghosts) if (ghost.state === 'assustado') ghost.state = 'solto';
    }
  } else {
    next.waveLeft -= dt;
    if (next.waveLeft <= 0) {
      next.wave += 1;
      next.waveLeft = WAVES[Math.min(next.wave, WAVES.length - 1)].ms;
      // Toda troca de onda vira os fantasmas: é o respiro do jogo.
      for (const ghost of next.ghosts) {
        if (ghost.state === 'solto') {
          ghost.dir = OPPOSITE[ghost.dir];
          ghost.next = null;
        }
      }
    }
  }

  // ── Comilão ───────────────────────────────────────────────────────────────
  advance(next.hero, heroSpeed(next.level) * seconds, false, false);

  const col = tileOf(next.hero.x);
  const row = tileOf(next.hero.y);
  const index = row * COLS + col;
  const bite = next.tiles[index];
  if (bite !== EMPTY) {
    // A grade só é copiada quando o jogador come de fato.
    if (next.tiles === g.tiles) next.tiles = g.tiles.slice();
    next.tiles[index] = EMPTY;
    next.left -= 1;
    next.eaten += 1;
    next.pellets += 1;
    next.score += bite === POWER ? POINTS_POWER : POINTS_PELLET;
    if (bite === POWER) {
      const fright = frightMs(next.level);
      next.combo = 0;
      if (fright > 0) {
        next.frightLeft = fright;
        for (const ghost of next.ghosts) {
          if (ghost.state === 'solto') {
            ghost.state = 'assustado';
            ghost.dir = OPPOSITE[ghost.dir];
            ghost.next = null;
          }
        }
      }
      events.push({ kind: 'pastilhao' });
    } else {
      events.push({ kind: 'pastilha' });
    }

    // Fruta nos dois terços do caminho.
    const third = Math.floor(PELLET_TOTAL / 3);
    if (next.eaten === third || next.eaten === third * 2) {
      next.fruit = { kind: Math.min(next.level, FRUITS.length), left: FRUIT_MS };
    }
  }

  if (next.score >= next.nextLife) {
    next.lives += 1;
    next.nextLife += LIFE_EVERY;
    events.push({ kind: 'vida' });
  }

  if (next.left <= 0) {
    next.phase = 'fase';
    next.phaseLeft = CLEAR_MS;
    next.fruit = null;
    next.rng = rng;
    return { game: next, events };
  }

  // ── Fruta ─────────────────────────────────────────────────────────────────
  if (next.fruit) {
    next.fruit = { ...next.fruit, left: next.fruit.left - dt };
    const onFruit =
      tileOf(next.hero.x) === FRUIT_TILE.col && tileOf(next.hero.y) === FRUIT_TILE.row;
    if (onFruit) {
      const points = fruitPoints(next.level);
      next.score += points;
      next.basket = [...next.basket, next.fruit.kind];
      events.push({ kind: 'fruta', points, x: next.hero.x, y: next.hero.y });
      next.fruit = null;
    } else if (next.fruit.left <= 0) {
      next.fruit = null;
    }
  }

  // ── Fantasmas ─────────────────────────────────────────────────────────────
  for (const ghost of next.ghosts) {
    const tunnel = inTunnel(tileOf(ghost.x), tileOf(ghost.y));
    const speed = ghostSpeed(next.level, ghost.state, tunnel, next.pace) * seconds;

    if (ghost.state === 'casa') {
      ghost.wait -= dt;
      if (ghost.wait <= 0) ghost.state = 'saindo';
      continue;
    }
    if (ghost.state === 'saindo') {
      leaveHouse(ghost, speed);
      continue;
    }
    if (ghost.state === 'olhos') {
      advance(ghost, speed, true, true, (a) => {
        const tile = tileOf(a.y) * COLS + tileOf(a.x);
        if (tile === ghost.tile) return;
        ghost.tile = tile;
        a.next = chooseDir(next, ghost, pick);
      });
      if (spanTo(ghost.x, ghost.y, centerOf(HOUSE.col), centerOf(HOUSE.row)) < 0.3) {
        ghost.x = centerOf(HOUSE.col);
        ghost.y = centerOf(HOUSE.row);
        ghost.state = 'casa';
        ghost.wait = HOME_MS;
        ghost.next = null;
      }
      continue;
    }

    advance(ghost, speed, true, false, (a) => {
      const tile = tileOf(a.y) * COLS + tileOf(a.x);
      if (tile === ghost.tile) return;
      ghost.tile = tile;
      a.next = chooseDir(next, ghost, pick);
    });
  }

  // ── Encontros ─────────────────────────────────────────────────────────────
  for (const ghost of next.ghosts) {
    if (ghost.state !== 'solto' && ghost.state !== 'assustado') continue;
    if (spanTo(next.hero.x, next.hero.y, ghost.x, ghost.y) > 0.7) continue;

    if (ghost.state === 'assustado') {
      next.combo += 1;
      next.ghostsEaten += 1;
      const points = ghostPoints(next.combo);
      next.score += points;
      ghost.state = 'olhos';
      ghost.next = null;
      ghost.tile = -1;
      events.push({ kind: 'fantasma', points, x: ghost.x, y: ghost.y });
      continue;
    }

    next.lives -= 1;
    next.phase = 'morrendo';
    next.phaseLeft = DEATH_MS;
    next.fruit = null;
    events.push({ kind: 'morreu' });
    break;
  }

  next.rng = rng;
  return { game: next, events };
}
