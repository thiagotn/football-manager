/**
 * Astronave — regras puras.
 *
 * Sem DOM, sem canvas: a nave, os tiros, os meteoros, os inimigos e a
 * pontuação. O desenho fica em `render.ts` e o laço do jogo em
 * `src/routes/astronave/+page.svelte`.
 *
 * O campo é um retângulo lógico de `W × H` unidades (a tela só escala). A nave
 * corre na horizontal, perto da borda de baixo; tudo o mais desce de cima.
 *
 * `tick` **muda o estado no lugar** (são dezenas de objetos a 60 passos por
 * segundo) e devolve os eventos do passo, para a tela saber o que animar. Deve
 * ser chamado sempre com o mesmo `dt` (`STEP`): a física não depende da taxa
 * de quadros e os testes comparam números exatos.
 *
 * Dois sorteios separados, ambos guardados no estado: `rng` decide o que nasce
 * (igual para todo mundo no desafio do dia) e `rngDrop` decide os prêmios, que
 * dependem de quem o jogador abateu.
 */

export const W = 100;
export const H = 170;
/** Passo fixo da simulação, em segundos. */
export const STEP = 1 / 60;

export const SHIP_Y = H - 12;
/** Meia largura da nave: até onde ela chega nas bordas. */
export const SHIP_HALF = 5;
/** Raio de colisão da nave — menor que o desenho, de propósito. */
export const SHIP_R = 3.2;
export const MAX_LIVES = 3;

/** Velocidade com botões/teclado e teto ao seguir o dedo (unidades/s). */
export const PAD_SPEED = 85;
export const DRAG_SPEED = 260;

export const SHOT_SPEED = 150;
export const FIRE_EVERY = 0.2;
const SHOT_R = 0.9;
const BOLT_R = 1.3;
export const ENEMY_R = 5;
const DROP_R = 3;
const DROP_SPEED = 26;

export const DOUBLE_SECONDS = 10;
const INVULN_AFTER_HIT = 2;
const INVULN_AFTER_SHIELD = 0.8;
const WAVE_BREAK = 2;

export const SMALL_R = 4.2;
export const BIG_R = 8;

export const SCORE = { small: 10, big: 25, scout: 30, gunner: 60, drop: 5, wave: 50 } as const;

export type EnemyKind = 'scout' | 'gunner';
export type DropKind = 'shield' | 'double' | 'life';

export type Shot = { x: number; y: number };
export type Bolt = { x: number; y: number; vx: number; vy: number };
export type Meteor = {
  id: number;
  x: number;
  y: number;
  vx: number;
  vy: number;
  r: number;
  hp: number;
  big: boolean;
};
export type Enemy = {
  id: number;
  kind: EnemyKind;
  x: number;
  y: number;
  /** Altura em que para de descer e passa a patrulhar. */
  homeY: number;
  vx: number;
  hp: number;
  /** Segundos até o próximo disparo. */
  fireIn: number;
};
export type Drop = { x: number; y: number; kind: DropKind };

export type Game = {
  /** Segundos de partida. */
  time: number;
  ship: {
    x: number;
    cooldown: number;
    /** Segundos de invulnerabilidade que restam. */
    invuln: number;
    shield: boolean;
    /** Segundos de tiro duplo que restam. */
    doubleLeft: number;
  };
  lives: number;
  score: number;
  wave: number;
  /** Inimigos abatidos. */
  kills: number;
  /** Disparos feitos (um tiro duplo conta uma vez). */
  fired: number;
  shots: Shot[];
  bolts: Bolt[];
  meteors: Meteor[];
  enemies: Enemy[];
  drops: Drop[];
  /** Inimigos que ainda vão entrar nesta onda. */
  toSpawn: number;
  spawnIn: number;
  meteorIn: number;
  /** Respiro entre ondas (0 = onda em curso). */
  breakLeft: number;
  over: boolean;
  /** `YYYY-MM-DD` quando é o desafio do dia. */
  daily: string | null;
  /** Pontos necessários no desafio do dia (0 na partida livre). */
  goal: number;
  rng: number;
  rngDrop: number;
  nextId: number;
};

/** O que o jogador está pedindo neste passo. */
export type Input = {
  /** −1 (esquerda) a 1 (direita): botões e teclado. */
  axis: number;
  /** Posição desejada da nave (arraste); `null` quando não há dedo na tela. */
  targetX: number | null;
  fire: boolean;
};

export const IDLE: Input = { axis: 0, targetX: null, fire: false };

export type GameEvent =
  | { type: 'shot'; x: number; y: number }
  | { type: 'boom'; x: number; y: number; size: number; kind: 'meteor' | EnemyKind }
  | { type: 'hit'; x: number; y: number; shielded: boolean }
  | { type: 'power'; x: number; y: number; kind: DropKind }
  | { type: 'wave'; wave: number }
  | { type: 'died'; x: number; y: number };

// ── Sorteio determinístico ──────────────────────────────────────────────────
// Cópia curta de propósito (como na Cobrinha): importar de outro jogo arrastaria
// o engine dele para o chunk desta rota.

/** FNV-1a — transforma a data (ou qualquer texto) na seed. */
export function fnv1a(s: string): number {
  let h = 0x811c9dc5;
  for (let i = 0; i < s.length; i++) {
    h ^= s.charCodeAt(i);
    h = Math.imul(h, 0x01000193);
  }
  return h >>> 0;
}

/** Um passo do mulberry32 sobre um estado inteiro: devolve o novo estado e o valor. */
function next(state: number): [number, number] {
  const a = (state + 0x6d2b79f5) >>> 0;
  let t = Math.imul(a ^ (a >>> 15), 1 | a);
  t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
  return [a, ((t ^ (t >>> 14)) >>> 0) / 4294967296];
}

function rand(g: Game): number {
  const [state, value] = next(g.rng);
  g.rng = state;
  return value;
}

function randDrop(g: Game): number {
  const [state, value] = next(g.rngDrop);
  g.rngDrop = state;
  return value;
}

const between = (g: Game, lo: number, hi: number) => lo + rand(g) * (hi - lo);
const clamp = (v: number, lo: number, hi: number) => Math.max(lo, Math.min(hi, v));

// ── Datas e rótulos ─────────────────────────────────────────────────────────

export function dateKey(d: Date): string {
  const p = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`;
}

export const dailySeed = (day: string): number => fnv1a(`astronave:${day}`);

/** Meta do dia: de 400 a 800 pontos, igual para todo mundo. */
export const dailyGoal = (day: string): number => 400 + (dailySeed(day) % 5) * 100;

export const reachedGoal = (g: Game): boolean => g.goal > 0 && g.score >= g.goal;

/** `1 615` — separador fino para o placar não virar um borrão de dígitos. */
export const fmtScore = (n: number): string => String(n).replace(/\B(?=(\d{3})+(?!\d))/g, ' ');

// ── Ritmo das ondas ─────────────────────────────────────────────────────────

/** Quantos inimigos a onda traz e quantos cabem na tela ao mesmo tempo. */
export const enemiesIn = (wave: number): number => Math.min(3 + wave, 12);
export const enemyCap = (wave: number): number => Math.min(3 + Math.floor(wave / 2), 7);
/** Chance de o inimigo ser um artilheiro (mira na nave, aguenta 3 tiros). */
export const gunnerChance = (wave: number): number => clamp((wave - 2) * 0.12, 0, 0.5);
/** Intervalo médio entre meteoros, em segundos. */
export const meteorEvery = (wave: number): number => Math.max(0.5, 1.7 - wave * 0.12);

// ── Partida ─────────────────────────────────────────────────────────────────

export type NewGameOptions = {
  /** `YYYY-MM-DD` para o desafio do dia; ausente para partida livre. */
  daily?: string | null;
  /** Seed da partida livre (os testes passam; em jogo a tela sorteia). */
  seed?: number;
};

export function newGame(options: NewGameOptions = {}): Game {
  const daily = options.daily ?? null;
  const seed = (daily ? dailySeed(daily) : (options.seed ?? 1)) >>> 0;
  return {
    time: 0,
    ship: { x: W / 2, cooldown: 0, invuln: 0, shield: false, doubleLeft: 0 },
    lives: MAX_LIVES,
    score: 0,
    wave: 1,
    kills: 0,
    fired: 0,
    shots: [],
    bolts: [],
    meteors: [],
    enemies: [],
    drops: [],
    toSpawn: enemiesIn(1),
    spawnIn: 1.6,
    meteorIn: 1.2,
    breakLeft: 0,
    over: false,
    daily,
    goal: daily ? dailyGoal(daily) : 0,
    rng: seed,
    rngDrop: (seed ^ 0x9e3779b9) >>> 0,
    nextId: 1,
  };
}

function spawnMeteor(g: Game): void {
  const big = rand(g) < 0.25;
  const r = big ? BIG_R : SMALL_R;
  // Um em cada três vem na direção da nave: ficar parado num canto não é abrigo.
  const aimed = rand(g) < 0.34;
  const x = aimed ? clamp(g.ship.x + between(g, -9, 9), r, W - r) : between(g, r, W - r);
  const speed = between(g, 24, 40) + Math.min(g.wave, 12) * 1.6;
  g.meteors.push({
    id: g.nextId++,
    x,
    y: -r,
    vx: between(g, -7, 7),
    vy: big ? speed * 0.75 : speed,
    r,
    hp: big ? 3 : 1,
    big,
  });
}

function spawnEnemy(g: Game): void {
  const kind: EnemyKind = rand(g) < gunnerChance(g.wave) ? 'gunner' : 'scout';
  const speed = Math.min(14 + g.wave * 1.6, 32);
  g.enemies.push({
    id: g.nextId++,
    kind,
    x: between(g, 10, W - 10),
    y: -ENEMY_R,
    homeY: between(g, 14, 64),
    vx: rand(g) < 0.5 ? -speed : speed,
    hp: kind === 'gunner' ? 3 : 1,
    fireIn: between(g, 1.0, 2.2),
  });
}

function enemyFires(g: Game, e: Enemy): void {
  const speed = Math.min(52 + g.wave * 2.5, 86);
  if (e.kind === 'gunner') {
    // Mira onde a nave está agora: quem continua andando escapa.
    const dx = g.ship.x - e.x;
    const dy = SHIP_Y - e.y;
    const d = Math.hypot(dx, dy) || 1;
    g.bolts.push({ x: e.x, y: e.y + ENEMY_R, vx: (dx / d) * speed, vy: (dy / d) * speed });
  } else {
    g.bolts.push({ x: e.x, y: e.y + ENEMY_R, vx: 0, vy: speed });
  }
  e.fireIn = between(g, 1.3, 2.9) * Math.max(0.55, 1 - g.wave * 0.04);
}

/** A nave levou uma pancada em (x, y): escudo, vida ou fim de jogo. */
function shipHit(g: Game, x: number, y: number, events: GameEvent[]): void {
  if (g.ship.shield) {
    g.ship.shield = false;
    g.ship.invuln = INVULN_AFTER_SHIELD;
    events.push({ type: 'hit', x, y, shielded: true });
    return;
  }
  g.lives -= 1;
  g.ship.doubleLeft = 0;
  if (g.lives <= 0) {
    g.lives = 0;
    g.over = true;
    events.push({ type: 'died', x: g.ship.x, y: SHIP_Y });
    return;
  }
  g.ship.invuln = INVULN_AFTER_HIT;
  events.push({ type: 'hit', x, y, shielded: false });
}

function maybeDrop(g: Game, e: Enemy): void {
  const roll = randDrop(g);
  const chance = e.kind === 'gunner' ? 0.3 : 0.14;
  if (roll >= chance) return;
  const pick = randDrop(g);
  const kind: DropKind =
    g.lives < MAX_LIVES && pick < 0.2 ? 'life' : pick < 0.6 ? 'shield' : 'double';
  g.drops.push({ x: e.x, y: e.y, kind });
}

const touching = (ax: number, ay: number, bx: number, by: number, r: number): boolean => {
  const dx = ax - bx;
  const dy = ay - by;
  return dx * dx + dy * dy < r * r;
};

/** Um passo da simulação. Muda `g` no lugar e devolve o que aconteceu. */
export function tick(g: Game, input: Input, dt: number = STEP): GameEvent[] {
  const events: GameEvent[] = [];
  if (g.over) return events;
  g.time += dt;

  // ── Nave ──
  const ship = g.ship;
  ship.invuln = Math.max(0, ship.invuln - dt);
  ship.doubleLeft = Math.max(0, ship.doubleLeft - dt);
  if (input.targetX !== null) {
    ship.x += clamp(input.targetX - ship.x, -DRAG_SPEED * dt, DRAG_SPEED * dt);
  } else {
    ship.x += clamp(input.axis, -1, 1) * PAD_SPEED * dt;
  }
  ship.x = clamp(ship.x, SHIP_HALF, W - SHIP_HALF);

  ship.cooldown = Math.max(0, ship.cooldown - dt);
  if (input.fire && ship.cooldown <= 0) {
    const y = SHIP_Y - 5;
    if (ship.doubleLeft > 0) g.shots.push({ x: ship.x - 2.6, y }, { x: ship.x + 2.6, y });
    else g.shots.push({ x: ship.x, y });
    ship.cooldown = FIRE_EVERY;
    g.fired += 1;
    events.push({ type: 'shot', x: ship.x, y });
  }

  // ── O que nasce ──
  g.meteorIn -= dt;
  if (g.meteorIn <= 0) {
    spawnMeteor(g);
    g.meteorIn = meteorEvery(g.wave) * between(g, 0.6, 1.4);
  }
  if (g.breakLeft > 0) {
    g.breakLeft -= dt;
    if (g.breakLeft <= 0) {
      g.breakLeft = 0;
      g.toSpawn = enemiesIn(g.wave);
      g.spawnIn = 0.5;
    }
  } else if (g.toSpawn > 0) {
    g.spawnIn -= dt;
    if (g.spawnIn <= 0 && g.enemies.length < enemyCap(g.wave)) {
      spawnEnemy(g);
      g.toSpawn -= 1;
      g.spawnIn = Math.max(0.55, 1.5 - g.wave * 0.08);
    }
  } else if (g.enemies.length === 0) {
    g.score += SCORE.wave * g.wave;
    g.wave += 1;
    g.breakLeft = WAVE_BREAK;
    events.push({ type: 'wave', wave: g.wave });
  }

  // ── Movimento ──
  for (const s of g.shots) s.y -= SHOT_SPEED * dt;
  for (const b of g.bolts) {
    b.x += b.vx * dt;
    b.y += b.vy * dt;
  }
  for (const m of g.meteors) {
    m.x += m.vx * dt;
    m.y += m.vy * dt;
    if ((m.x < m.r && m.vx < 0) || (m.x > W - m.r && m.vx > 0)) m.vx = -m.vx;
  }
  for (const e of g.enemies) {
    if (e.y < e.homeY) {
      e.y = Math.min(e.homeY, e.y + 34 * dt);
    } else {
      e.x += e.vx * dt;
      if ((e.x < 8 && e.vx < 0) || (e.x > W - 8 && e.vx > 0)) e.vx = -e.vx;
      e.fireIn -= dt;
      if (e.fireIn <= 0) enemyFires(g, e);
    }
  }
  for (const d of g.drops) d.y += DROP_SPEED * dt;

  // ── Tiros da nave ──
  const born: Meteor[] = [];
  for (const s of g.shots) {
    if (s.y < -2) continue;
    let used = false;
    for (const m of g.meteors) {
      if (m.hp <= 0 || !touching(s.x, s.y, m.x, m.y, m.r + SHOT_R)) continue;
      used = true;
      m.hp -= 1;
      if (m.hp <= 0) {
        g.score += m.big ? SCORE.big : SCORE.small;
        events.push({ type: 'boom', x: m.x, y: m.y, size: m.r, kind: 'meteor' });
        if (m.big) {
          // O grande racha em dois pequenos, um para cada lado.
          for (const side of [-1, 1]) {
            born.push({
              id: g.nextId++,
              x: m.x + side * 3,
              y: m.y,
              vx: side * 15,
              vy: m.vy * 1.15,
              r: SMALL_R,
              hp: 1,
              big: false,
            });
          }
        }
      }
      break;
    }
    if (!used) {
      for (const e of g.enemies) {
        if (e.hp <= 0 || !touching(s.x, s.y, e.x, e.y, ENEMY_R + SHOT_R)) continue;
        used = true;
        e.hp -= 1;
        if (e.hp <= 0) {
          g.score += SCORE[e.kind];
          g.kills += 1;
          events.push({ type: 'boom', x: e.x, y: e.y, size: ENEMY_R, kind: e.kind });
          maybeDrop(g, e);
        }
        break;
      }
    }
    if (used) s.y = -99;
  }

  // ── O que acerta a nave ──
  const vulnerable = () => ship.invuln <= 0 && !g.over;
  for (const m of g.meteors) {
    if (m.hp <= 0 || !vulnerable()) continue;
    if (touching(m.x, m.y, ship.x, SHIP_Y, m.r + SHIP_R)) {
      m.hp = 0;
      events.push({ type: 'boom', x: m.x, y: m.y, size: m.r, kind: 'meteor' });
      shipHit(g, m.x, m.y, events);
    }
  }
  for (const b of g.bolts) {
    if (!vulnerable()) break;
    if (touching(b.x, b.y, ship.x, SHIP_Y, BOLT_R + SHIP_R)) {
      shipHit(g, b.x, b.y, events);
      b.y = H + 99;
    }
  }
  for (const d of g.drops) {
    if (g.over || !touching(d.x, d.y, ship.x, SHIP_Y, DROP_R + SHIP_HALF)) continue;
    if (d.kind === 'shield') ship.shield = true;
    else if (d.kind === 'double') ship.doubleLeft = DOUBLE_SECONDS;
    else g.lives = Math.min(MAX_LIVES, g.lives + 1);
    g.score += SCORE.drop;
    events.push({ type: 'power', x: d.x, y: d.y, kind: d.kind });
    d.y = H + 99;
  }

  // ── Limpeza ──
  g.shots = g.shots.filter((s) => s.y > -2);
  g.bolts = g.bolts.filter((b) => b.y < H + 4 && b.x > -4 && b.x < W + 4);
  g.meteors = g.meteors.filter((m) => m.hp > 0 && m.y < H + m.r);
  if (born.length) g.meteors.push(...born);
  g.enemies = g.enemies.filter((e) => e.hp > 0);
  g.drops = g.drops.filter((d) => d.y < H + 4);

  return events;
}
