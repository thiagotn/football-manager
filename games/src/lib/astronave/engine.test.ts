import { describe, expect, it, vi } from 'vitest';
import {
  BIG_R,
  DOUBLE_SECONDS,
  FIRE_EVERY,
  H,
  IDLE,
  MAX_LIVES,
  PAD_SPEED,
  SCORE,
  SHIP_HALF,
  SHIP_Y,
  SMALL_R,
  STEP,
  W,
  dailyGoal,
  dailySeed,
  enemiesIn,
  enemyCap,
  fmtScore,
  gunnerChance,
  meteorEvery,
  newGame,
  reachedGoal,
  tick,
  type Enemy,
  type Game,
  type GameEvent,
  type Input,
  type Meteor,
} from './engine';

/** Partida sem nada nascendo por um bom tempo — o teste põe o que quiser no campo. */
function quiet(): Game {
  const g = newGame({ seed: 7 });
  g.meteorIn = 999;
  g.spawnIn = 999;
  return g;
}

function run(g: Game, input: Input, seconds: number): GameEvent[] {
  const events: GameEvent[] = [];
  for (let i = 0; i < Math.round(seconds / STEP); i++) events.push(...tick(g, input));
  return events;
}

const meteor = (over: Partial<Meteor> = {}): Meteor => ({
  id: 900,
  x: 50,
  y: 60,
  vx: 0,
  vy: 0,
  r: SMALL_R,
  hp: 1,
  big: false,
  ...over,
});

const enemy = (over: Partial<Enemy> = {}): Enemy => ({
  id: 901,
  kind: 'scout',
  x: 50,
  y: 40,
  homeY: 40,
  vx: 0,
  hp: 1,
  fireIn: 999,
  ...over,
});

const FIRE: Input = { axis: 0, targetX: null, fire: true };

describe('nova partida', () => {
  it('começa no centro, com 3 vidas, na onda 1 e de campo vazio', () => {
    const g = newGame({ seed: 1 });
    expect(g.ship.x).toBe(W / 2);
    expect(g.lives).toBe(MAX_LIVES);
    expect(g.wave).toBe(1);
    expect(g.score).toBe(0);
    expect(g.meteors).toHaveLength(0);
    expect(g.enemies).toHaveLength(0);
    expect(g.toSpawn).toBe(enemiesIn(1));
    expect(g.goal).toBe(0);
  });

  it('a mesma seed dá a mesma partida', () => {
    const a = newGame({ seed: 42 });
    const b = newGame({ seed: 42 });
    run(a, IDLE, 8);
    run(b, IDLE, 8);
    expect(a).toEqual(b);
  });
});

describe('a nave', () => {
  it('anda com o direcional na velocidade dos botões', () => {
    const g = quiet();
    run(g, { axis: 1, targetX: null, fire: false }, 0.5);
    expect(g.ship.x).toBeCloseTo(W / 2 + PAD_SPEED * 0.5, 5);
  });

  it('para nas bordas', () => {
    const g = quiet();
    run(g, { axis: -1, targetX: null, fire: false }, 3);
    expect(g.ship.x).toBe(SHIP_HALF);
    run(g, { axis: 1, targetX: null, fire: false }, 3);
    expect(g.ship.x).toBe(W - SHIP_HALF);
  });

  it('segue o dedo até o alvo e não passa dele', () => {
    const g = quiet();
    run(g, { axis: 0, targetX: 20, fire: false }, 1);
    expect(g.ship.x).toBeCloseTo(20, 5);
  });

  it('o alvo do dedo fora do campo é cortado na borda', () => {
    const g = quiet();
    run(g, { axis: 0, targetX: 500, fire: false }, 2);
    expect(g.ship.x).toBe(W - SHIP_HALF);
  });
});

describe('atirar', () => {
  it('dispara na cadência, não a cada passo', () => {
    const g = quiet();
    run(g, FIRE, 1);
    expect(g.fired).toBe(Math.round(1 / FIRE_EVERY));
  });

  it('sem pedir, não atira', () => {
    const g = quiet();
    run(g, IDLE, 1);
    expect(g.fired).toBe(0);
    expect(g.shots).toHaveLength(0);
  });

  it('o tiro sobe e some no topo', () => {
    const g = quiet();
    tick(g, FIRE);
    expect(g.shots).toHaveLength(1);
    const y0 = g.shots[0].y;
    tick(g, IDLE);
    expect(g.shots[0].y).toBeLessThan(y0);
    run(g, IDLE, 2);
    expect(g.shots).toHaveLength(0);
  });

  it('com o tiro duplo saem dois por disparo, e ele acaba sozinho', () => {
    const g = quiet();
    g.ship.doubleLeft = DOUBLE_SECONDS;
    tick(g, FIRE);
    expect(g.shots).toHaveLength(2);
    expect(g.fired).toBe(1);
    run(g, IDLE, DOUBLE_SECONDS + 0.1);
    tick(g, FIRE);
    expect(g.shots).toHaveLength(1);
  });

  it('acerta o meteoro pequeno: some, pontua e o tiro é gasto', () => {
    const g = quiet();
    g.meteors.push(meteor());
    const events = run(g, FIRE, 1.2);
    expect(g.meteors).toHaveLength(0);
    expect(g.score).toBe(SCORE.small);
    expect(events.some((e) => e.type === 'boom' && e.kind === 'meteor')).toBe(true);
  });

  it('o meteoro grande aguenta 3 tiros e racha em dois pequenos', () => {
    const g = quiet();
    g.meteors.push(meteor({ r: BIG_R, hp: 3, big: true, y: 40 }));
    while (g.meteors.some((m) => m.big)) tick(g, FIRE);
    expect(g.score).toBe(SCORE.big);
    expect(g.meteors).toHaveLength(2);
    expect(g.meteors.every((m) => !m.big && m.r === SMALL_R)).toBe(true);
    expect(g.meteors[0].vx).toBeLessThan(0);
    expect(g.meteors[1].vx).toBeGreaterThan(0);
  });

  it('abate o batedor com um tiro e o artilheiro com três', () => {
    const g = quiet();
    g.enemies.push(enemy());
    run(g, FIRE, 1.2);
    expect(g.enemies).toHaveLength(0);
    expect(g.kills).toBe(1);
    expect(g.score).toBe(SCORE.scout);

    const h = quiet();
    h.enemies.push(enemy({ kind: 'gunner', hp: 3 }));
    tick(h, FIRE);
    run(h, IDLE, 1.2);
    expect(h.enemies[0].hp).toBe(2);
    run(h, FIRE, 1.5);
    expect(h.enemies).toHaveLength(0);
    expect(h.score).toBe(SCORE.gunner);
  });
});

describe('levar dano', () => {
  it('o meteoro que encosta tira uma vida e dá um tempo de invulnerabilidade', () => {
    const g = quiet();
    g.meteors.push(meteor({ y: SHIP_Y - 10, vy: 40 }));
    const events = run(g, IDLE, 0.5);
    expect(g.lives).toBe(MAX_LIVES - 1);
    expect(g.ship.invuln).toBeGreaterThan(0);
    expect(events.some((e) => e.type === 'hit' && !e.shielded)).toBe(true);
  });

  it('invulnerável, a nave atravessa o que vier', () => {
    const g = quiet();
    g.ship.invuln = 1;
    g.meteors.push(meteor({ y: SHIP_Y - 6, vy: 40 }));
    g.bolts.push({ x: 50, y: SHIP_Y - 6, vx: 0, vy: 60 });
    run(g, IDLE, 0.4);
    expect(g.lives).toBe(MAX_LIVES);
  });

  it('o tiro inimigo tira uma vida e some', () => {
    const g = quiet();
    g.bolts.push({ x: 50, y: SHIP_Y - 10, vx: 0, vy: 60 });
    run(g, IDLE, 0.4);
    expect(g.lives).toBe(MAX_LIVES - 1);
    expect(g.bolts).toHaveLength(0);
  });

  it('o tiro que passa ao lado não acerta', () => {
    const g = quiet();
    g.bolts.push({ x: 50 + 8, y: SHIP_Y - 10, vx: 0, vy: 60 });
    run(g, IDLE, 1);
    expect(g.lives).toBe(MAX_LIVES);
  });

  it('o escudo absorve uma pancada e se vai', () => {
    const g = quiet();
    g.ship.shield = true;
    g.bolts.push({ x: 50, y: SHIP_Y - 10, vx: 0, vy: 60 });
    const events = run(g, IDLE, 0.4);
    expect(g.lives).toBe(MAX_LIVES);
    expect(g.ship.shield).toBe(false);
    expect(events.some((e) => e.type === 'hit' && e.shielded)).toBe(true);
  });

  it('perder a vida leva o tiro duplo junto', () => {
    const g = quiet();
    g.ship.doubleLeft = DOUBLE_SECONDS;
    g.bolts.push({ x: 50, y: SHIP_Y - 4, vx: 0, vy: 60 });
    run(g, IDLE, 0.2);
    expect(g.ship.doubleLeft).toBe(0);
  });

  it('na última vida o jogo acaba e para de andar', () => {
    const g = quiet();
    g.lives = 1;
    g.bolts.push({ x: 50, y: SHIP_Y - 4, vx: 0, vy: 60 });
    const events = run(g, IDLE, 0.2);
    expect(g.over).toBe(true);
    expect(g.lives).toBe(0);
    expect(events.filter((e) => e.type === 'died')).toHaveLength(1);
    const frozen = structuredClone(g);
    expect(tick(g, FIRE)).toEqual([]);
    expect(g).toEqual(frozen);
  });
});

describe('prêmios', () => {
  it('cada prêmio faz o que promete', () => {
    const g = quiet();
    g.drops.push({ x: 50, y: SHIP_Y - 4, kind: 'shield' });
    run(g, IDLE, 0.3);
    expect(g.ship.shield).toBe(true);
    expect(g.score).toBe(SCORE.drop);

    g.drops.push({ x: 50, y: SHIP_Y - 4, kind: 'double' });
    run(g, IDLE, 0.3);
    expect(g.ship.doubleLeft).toBeGreaterThan(DOUBLE_SECONDS - 1);

    g.lives = 1;
    g.drops.push({ x: 50, y: SHIP_Y - 4, kind: 'life' });
    run(g, IDLE, 0.3);
    expect(g.lives).toBe(2);
    expect(g.drops).toHaveLength(0);
  });

  it('a vida extra não passa do máximo', () => {
    const g = quiet();
    g.drops.push({ x: 50, y: SHIP_Y - 4, kind: 'life' });
    run(g, IDLE, 0.3);
    expect(g.lives).toBe(MAX_LIVES);
  });

  it('o prêmio que ninguém pega cai fora do campo', () => {
    const g = quiet();
    g.drops.push({ x: 10, y: H - 20, kind: 'shield' });
    run(g, IDLE, 2);
    expect(g.drops).toHaveLength(0);
    expect(g.ship.shield).toBe(false);
  });
});

describe('inimigos e ondas', () => {
  it('o batedor desce até a altura dele, patrulha e atira reto para baixo', () => {
    const g = quiet();
    g.enemies.push(enemy({ y: -5, homeY: 30, vx: 20, fireIn: 0.1, x: 20 }));
    run(g, IDLE, 1.3);
    const e = g.enemies[0];
    expect(e.y).toBe(30);
    expect(e.x).toBeGreaterThan(20);
    expect(g.bolts.length).toBeGreaterThan(0);
    expect(g.bolts[0].vx).toBe(0);
    expect(g.bolts[0].vy).toBeGreaterThan(0);
  });

  it('o artilheiro mira na nave', () => {
    const g = quiet();
    g.ship.x = 80;
    g.enemies.push(enemy({ kind: 'gunner', hp: 3, x: 20, fireIn: 0.01 }));
    tick(g, { axis: 0, targetX: 80, fire: false });
    expect(g.bolts).toHaveLength(1);
    expect(g.bolts[0].vx).toBeGreaterThan(0);
  });

  it('o inimigo bate na borda e volta', () => {
    const g = quiet();
    g.enemies.push(enemy({ x: 90, vx: 30 }));
    run(g, IDLE, 1);
    expect(g.enemies[0].vx).toBeLessThan(0);
    expect(g.enemies[0].x).toBeLessThan(W - 7);
  });

  it('limpar a onda dá bônus, respiro e uma onda maior', () => {
    const g = quiet();
    g.toSpawn = 0;
    const events = tick(g, IDLE);
    expect(events).toContainEqual({ type: 'wave', wave: 2 });
    expect(g.wave).toBe(2);
    expect(g.score).toBe(SCORE.wave);
    expect(g.breakLeft).toBeGreaterThan(0);
    run(g, IDLE, 2.1);
    expect(g.toSpawn).toBe(enemiesIn(2));
  });

  it('a onda não vira enquanto houver inimigo vivo ou por entrar', () => {
    const g = quiet();
    g.toSpawn = 0;
    g.enemies.push(enemy({ x: 5 }));
    run(g, IDLE, 1);
    expect(g.wave).toBe(1);
  });

  it('as ondas apertam, mas com teto', () => {
    expect(enemiesIn(2)).toBeGreaterThan(enemiesIn(1));
    expect(enemiesIn(99)).toBe(12);
    expect(enemyCap(99)).toBe(7);
    expect(gunnerChance(1)).toBe(0);
    expect(gunnerChance(99)).toBe(0.5);
    expect(meteorEvery(1)).toBeGreaterThan(meteorEvery(5));
    expect(meteorEvery(99)).toBe(0.5);
  });

  it('nunca há mais inimigos na tela do que o teto da onda', () => {
    const g = newGame({ seed: 3 });
    g.ship.invuln = 9999;
    let most = 0;
    for (let i = 0; i < 60 * 40; i++) {
      tick(g, IDLE);
      most = Math.max(most, g.enemies.length);
    }
    expect(most).toBeLessThanOrEqual(enemyCap(1));
    expect(most).toBeGreaterThan(0);
  });

  it('tudo que nasce fica dentro do campo na horizontal', () => {
    const g = newGame({ seed: 11 });
    g.ship.invuln = 9999;
    for (let i = 0; i < 60 * 60; i++) {
      tick(g, { axis: i % 400 < 200 ? 1 : -1, targetX: null, fire: true });
      for (const m of g.meteors) {
        expect(m.x).toBeGreaterThan(-1);
        expect(m.x).toBeLessThan(W + 1);
      }
      for (const e of g.enemies) {
        expect(e.x).toBeGreaterThan(0);
        expect(e.x).toBeLessThan(W);
      }
    }
    expect(g.wave).toBeGreaterThan(1);
  });
});

describe('desafio do dia', () => {
  it('a meta fica entre 400 e 800 e depende só da data', () => {
    for (const day of ['2026-01-01', '2026-06-15', '2026-11-27', '2027-02-28']) {
      expect(dailyGoal(day)).toBeGreaterThanOrEqual(400);
      expect(dailyGoal(day)).toBeLessThanOrEqual(800);
      expect(dailyGoal(day)).toBe(dailyGoal(day));
    }
    expect(dailySeed('2026-11-27')).not.toBe(dailySeed('2026-11-28'));
  });

  it('o mesmo dia, jogado igual, dá a mesma partida — e outro dia, outra', () => {
    const play = (day: string) => {
      const g = newGame({ daily: day });
      for (let i = 0; i < 60 * 20 && !g.over; i++) {
        tick(g, { axis: Math.sin(i / 37) > 0 ? 1 : -1, targetX: null, fire: true });
      }
      return g;
    };
    expect(play('2026-11-27')).toEqual(play('2026-11-27'));
    expect(play('2026-11-27')).not.toEqual(play('2026-11-28'));
  });

  it('bater a meta marca o dia', () => {
    const g = newGame({ daily: '2026-11-27' });
    expect(reachedGoal(g)).toBe(false);
    g.score = g.goal;
    expect(reachedGoal(g)).toBe(true);
    expect(reachedGoal(newGame({ seed: 1 }))).toBe(false);
  });

  it('o engine não toca em Math.random', () => {
    const spy = vi.spyOn(Math, 'random');
    const g = newGame({ daily: '2026-11-27' });
    run(g, FIRE, 30);
    expect(spy).not.toHaveBeenCalled();
    spy.mockRestore();
  });
});

describe('equilíbrio', () => {
  /** Segundos até o fim de jogo com uma política de entrada. */
  function survive(seed: number, policy: (g: Game, i: number) => Input, cap = 600): number {
    const g = newGame({ seed });
    let i = 0;
    while (!g.over && g.time < cap) tick(g, policy(g, i++));
    return g.time;
  }

  it('parado no meio sem atirar, a partida acaba em menos de 1 minuto', () => {
    const times = [1, 2, 3, 4, 5, 6, 7, 8].map((seed) => survive(seed, () => IDLE));
    expect(Math.max(...times)).toBeLessThan(60);
    // …mas não de cara: dá tempo de entender o que está acontecendo.
    expect(Math.min(...times)).toBeGreaterThan(6);
  });

  it('quem desvia e atira dura mais do que quem fica parado', () => {
    /** Foge do perigo mais próximo que vem na direção da nave; senão, mira no inimigo. */
    const bot = (g: Game): Input => {
      let danger: { x: number; d: number } | null = null;
      for (const o of [...g.meteors, ...g.bolts]) {
        const dy = SHIP_Y - o.y;
        if (dy < 0 || dy > 45) continue;
        const r = (o as Partial<Meteor>).r ?? 1.3;
        const at = o.x + o.vx * (dy / Math.max(1, o.vy));
        if (Math.abs(at - g.ship.x) < r + 6 && (!danger || dy < danger.d)) danger = { x: at, d: dy };
      }
      if (danger) {
        const away = g.ship.x >= danger.x ? 1 : -1;
        const side = g.ship.x < 12 ? 1 : g.ship.x > W - 12 ? -1 : away;
        return { axis: side, targetX: null, fire: true };
      }
      const prey = g.enemies[0];
      const axis = prey ? Math.sign(prey.x - g.ship.x) : Math.sign(W / 2 - g.ship.x);
      return { axis, targetX: null, fire: true };
    };
    const seeds = [1, 2, 3, 4, 5, 6];
    const idle = seeds.map((s) => survive(s, () => IDLE)).reduce((a, b) => a + b, 0);
    const smart = seeds.map((s) => survive(s, bot)).reduce((a, b) => a + b, 0);
    expect(smart).toBeGreaterThan(idle * 1.5);
  });
});

describe('placar', () => {
  it('separa os milhares com espaço fino', () => {
    expect(fmtScore(0)).toBe('0');
    expect(fmtScore(980)).toBe('980');
    expect(fmtScore(1615)).toBe('1 615');
    expect(fmtScore(1234567)).toBe('1 234 567');
  });
});
