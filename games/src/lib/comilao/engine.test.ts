/**
 * Regras do Comilão. As partidas são rodadas em passos fixos, como na tela, e
 * os testes comparam números exatos — é o que deixa a perseguição auditável.
 */
import { describe, expect, it } from 'vitest';
import {
  CLEAR_MS,
  COLS,
  DEATH_MS,
  FRUITS,
  GHOST_IDS,
  LIFE_EVERY,
  PELLET_TOTAL,
  READY_MS,
  centerOf,
  dailyGame,
  dailyGoal,
  dateKey,
  fmtScore,
  frightMs,
  fruitPoints,
  ghostPoints,
  ghostSpeed,
  heroSpeed,
  newGame,
  reachedGoal,
  targetOf,
  tick,
  tileOf,
  turn,
  type Dir,
  type Event,
  type Game,
  type Ghost,
} from './engine';
import { HERO_START, HOUSE, POWER, initialTiles } from './maze';

/** O mesmo passo que a rota usa. */
const STEP = 1000 / 60;

/** Roda `ms` de jogo. `script` injeta comandos em tempos dados (ms → direção). */
function run(game: Game, ms: number, script: Record<number, Dir> = {}): { game: Game; events: Event[] } {
  const events: Event[] = [];
  let g = game;
  const steps = Math.round(ms / STEP);
  for (let i = 0; i < steps; i++) {
    const at = Math.round(i * STEP);
    for (const [when, dir] of Object.entries(script)) {
      if (at >= Number(when) && at < Number(when) + STEP) g = turn(g, dir);
    }
    const out = tick(g, STEP);
    g = out.game;
    events.push(...out.events);
  }
  return { game: g, events };
}

/** Partida já valendo (sem a pausa do "Prepare-se"), para os testes diretos. */
function playing(options: Parameters<typeof newGame>[0] = { seed: 7 }): Game {
  return { ...newGame(options), phase: 'jogando', phaseLeft: 0 };
}

const put = (g: Game, col: number, row: number, dir: Dir = 'left'): Game => ({
  ...g,
  hero: { x: centerOf(col), y: centerOf(row), dir, next: null },
});

/** Põe um fantasma colado no herói, no estado pedido. */
function withGhost(g: Game, state: Ghost['state'], at = g.hero): Game {
  const ghosts = g.ghosts.map((ghost, i) =>
    i === 0 ? { ...ghost, state, x: at.x, y: at.y, dir: 'right' as Dir, wait: 0 } : { ...ghost, wait: 1e6 }
  );
  return { ...g, ghosts };
}

describe('nova partida', () => {
  it('começa na fase 1, com três vidas e o labirinto cheio', () => {
    const g = newGame({ seed: 1 });
    expect(g.level).toBe(1);
    expect(g.lives).toBe(3);
    expect(g.score).toBe(0);
    expect(g.left).toBe(PELLET_TOTAL);
    expect(g.ghosts).toHaveLength(4);
    expect(g.ghosts.map((x) => x.id)).toEqual(GHOST_IDS);
  });

  it('o Comilão nasce no berço, olhando para a esquerda', () => {
    const g = newGame({ seed: 1 });
    expect(g.hero.x).toBe(centerOf(HERO_START.col));
    expect(g.hero.y).toBe(centerOf(HERO_START.row));
    expect(g.hero.dir).toBe('left');
  });

  it('o caçador já está fora da casa e os outros esperam a vez', () => {
    const g = newGame({ seed: 1 });
    const [cacador, ...resto] = g.ghosts;
    expect(cacador.state).toBe('solto');
    expect(resto.map((x) => x.state)).toEqual(['casa', 'casa', 'casa']);
    expect(resto.map((x) => x.wait > 0)).toEqual([true, true, true]);
  });

  it('ninguém anda durante o "Prepare-se"', () => {
    const g = newGame({ seed: 1 });
    expect(g.phase).toBe('pronto');
    const parado = run(g, READY_MS - 100).game;
    expect(parado.phase).toBe('pronto');
    expect(parado.hero.x).toBe(centerOf(HERO_START.col));
    expect(parado.score).toBe(0);
    expect(run(g, READY_MS + 100).game.phase).toBe('jogando');
  });
});

describe('movimento', () => {
  it('anda para a esquerda comendo o corredor e para na parede', () => {
    const { game, events } = run(playing(), 700);
    // Do berço (col 9) até a parede da col 6: come as pastilhas 8 e 7.
    expect(game.hero.x).toBe(centerOf(7));
    expect(game.score).toBe(20);
    expect(game.left).toBe(PELLET_TOTAL - 2);
    expect(events.filter((e) => e.kind === 'pastilha')).toHaveLength(2);
  });

  it('a curva pedida espera a primeira esquina que couber', () => {
    // Na linha 16, indo para a direita: acima da col 2 é parede, da col 3 não.
    const g = turn(put(playing(), 2, 16, 'right'), 'up');
    expect(g.hero.next).toBe('up');
    const andou = run(g, 400).game;
    expect(andou.hero.dir).toBe('up');
    expect(tileOf(andou.hero.x)).toBe(3);
    expect(andou.hero.y).toBeLessThan(centerOf(16));
  });

  it('recusa a curva contra a parede e segue em frente', () => {
    const g = turn(put(playing(), 9, 16, 'left'), 'up'); // acima da linha 16, col 9, é parede
    const andou = run(g, 200).game;
    expect(andou.hero.dir).toBe('left');
  });

  it('a volta atrás vale na hora, no meio do corredor', () => {
    const meio = run(playing(), 200).game;
    const voltou = run(turn(meio, 'right'), 200).game;
    expect(voltou.hero.dir).toBe('right');
  });

  it('a travessia leva de um lado ao outro do labirinto', () => {
    const g = put(playing(), 1, 10, 'left');
    const atravessou = run(g, 400).game;
    expect(atravessou.hero.x).toBeGreaterThan(COLS - 2);
  });

  it('o Comilão não entra na casa dos fantasmas', () => {
    const g = turn(put(playing(), 9, 8, 'left'), 'down');
    const andou = run(g, 400).game;
    expect(tileOf(andou.hero.y)).toBe(8);
  });

  it('a velocidade sobe com a fase e estaciona', () => {
    expect(heroSpeed(2)).toBeGreaterThan(heroSpeed(1));
    expect(heroSpeed(7)).toBe(heroSpeed(20));
  });
});

describe('pastilhão e susto', () => {
  /** Herói ao lado do pastilhão da esquina, sem pastilha sob os pés. */
  function comPastilhao(level = 1): Game {
    const g = put(playing({ seed: 7, level }), 2, 1, 'left');
    const tiles = [...g.tiles];
    tiles[1 * COLS + 2] = 0;
    return { ...g, tiles, left: g.left - 1 };
  }

  it('assusta os fantasmas soltos e vale 50 pontos', () => {
    const { game, events } = run(comPastilhao(), 300);
    expect(game.score).toBe(50);
    expect(events.some((e) => e.kind === 'pastilhao')).toBe(true);
    expect(game.frightLeft).toBeGreaterThan(0);
    expect(game.ghosts[0].state).toBe('assustado');
  });

  it('o susto encurta a cada fase e acaba na 19', () => {
    expect(frightMs(1)).toBe(7000);
    expect(frightMs(5)).toBeLessThan(frightMs(1));
    expect(frightMs(19)).toBe(0);
    expect(frightMs(30)).toBe(0);
  });

  it('na fase 19 o pastilhão pontua mas não assusta', () => {
    const depois = run(comPastilhao(19), 300).game;
    expect(depois.score).toBe(50);
    expect(depois.frightLeft).toBe(0);
    expect(depois.ghosts[0].state).toBe('solto');
  });

  it('passado o tempo, os fantasmas voltam a caçar', () => {
    const g = run(comPastilhao(), 300).game;
    const depois = run(g, frightMs(1) + 200).game;
    expect(depois.frightLeft).toBe(0);
    expect(depois.ghosts.every((x) => x.state !== 'assustado')).toBe(true);
  });

  it('o combo paga 200, 400, 800 e 1600 no mesmo susto', () => {
    let g = { ...run(comPastilhao(), 300).game, combo: 0 };
    const pontos: number[] = [];
    for (let i = 0; i < 4; i++) {
      const colado = {
        ...g,
        ghosts: g.ghosts.map((ghost, k) =>
          k === i ? { ...ghost, state: 'assustado' as const, x: g.hero.x, y: g.hero.y } : ghost
        ),
      };
      const out = tick(colado, STEP);
      g = out.game;
      const comido = out.events.find((e) => e.kind === 'fantasma');
      if (comido && comido.kind === 'fantasma') pontos.push(comido.points);
    }
    expect(pontos).toEqual([200, 400, 800, 1600]);
    expect(g.ghostsEaten).toBe(4);
  });

  it('o combo zera quando o susto termina', () => {
    const g = { ...run(comPastilhao(), 300).game, combo: 3 };
    expect(run(g, frightMs(1) + 200).game.combo).toBe(0);
    expect(ghostPoints(1)).toBe(200);
    expect(ghostPoints(4)).toBe(1600);
  });

  it('o fantasma comido volta para a casa e renasce', () => {
    const g = withGhost({ ...run(comPastilhao(), 300).game }, 'assustado');
    const comido = tick(g, STEP).game;
    expect(comido.ghosts[0].state).toBe('olhos');
    const voltou = run(comido, 6000).game;
    expect(['casa', 'saindo', 'solto']).toContain(voltou.ghosts[0].state);
  });
});

describe('morte e fim', () => {
  it('fantasma solto tira uma vida e devolve todos ao lugar', () => {
    const g = withGhost(playing(), 'solto');
    const { game, events } = run(g, STEP * 2);
    expect(events.some((e) => e.kind === 'morreu')).toBe(true);
    expect(game.lives).toBe(2);
    expect(game.phase).toBe('morrendo');
    const renasceu = run(game, DEATH_MS + 100).game;
    expect(renasceu.phase).toBe('pronto');
    expect(renasceu.hero.x).toBe(centerOf(HERO_START.col));
  });

  it('fantasma de olhos não machuca', () => {
    const g = withGhost(playing(), 'olhos');
    const depois = run(g, STEP * 2).game;
    expect(depois.lives).toBe(3);
    expect(depois.phase).toBe('jogando');
  });

  it('sem vidas, a partida acaba', () => {
    const g = withGhost({ ...playing(), lives: 1 }, 'solto');
    const morreu = run(g, STEP * 2).game;
    expect(morreu.lives).toBe(0);
    const { game, events } = run(morreu, DEATH_MS + 100);
    expect(game.phase).toBe('fim');
    expect(events.some((e) => e.kind === 'fim')).toBe(true);
    // Partida encerrada não anda mais.
    expect(tick(game, STEP).game).toBe(game);
  });

  it('a vida extra vem a cada 10 000 pontos', () => {
    const g = { ...put(playing(), 9, 14, 'left'), score: LIFE_EVERY - 5 };
    const { game, events } = run(g, 300);
    expect(events.some((e) => e.kind === 'vida')).toBe(true);
    expect(game.lives).toBe(4);
    expect(game.nextLife).toBe(LIFE_EVERY * 2);
  });
});

describe('fase vencida', () => {
  /** Partida com uma pastilha só, na frente do Comilão. */
  function quaseLimpa(level = 1): Game {
    const g = playing({ seed: 5, level });
    const tiles = initialTiles().map(() => 0);
    tiles[14 * COLS + 8] = 1;
    return { ...g, tiles, left: 1, eaten: PELLET_TOTAL - 1 };
  }

  it('zerar as pastilhas vira a fase e enche o labirinto de novo', () => {
    const { game, events } = run(quaseLimpa(), 200);
    expect(game.phase).toBe('fase');
    const nova = run(game, CLEAR_MS + 100);
    expect(nova.game.level).toBe(2);
    expect(nova.game.left).toBe(PELLET_TOTAL);
    expect(nova.game.phase).toBe('pronto');
    expect(nova.events.some((e) => e.kind === 'fase' && e.level === 2)).toBe(true);
    expect(events.some((e) => e.kind === 'pastilha')).toBe(true);
  });

  it('a fase nova é mais rápida e a casa esvazia mais cedo', () => {
    const fase1 = newGame({ seed: 1, level: 1 });
    const fase4 = newGame({ seed: 1, level: 4 });
    expect(heroSpeed(4)).toBeGreaterThan(heroSpeed(1));
    expect(ghostSpeed(4, 'solto', false, 'normal')).toBeGreaterThan(
      ghostSpeed(1, 'solto', false, 'normal')
    );
    expect(fase4.ghosts[3].wait).toBeLessThan(fase1.ghosts[3].wait);
  });
});

describe('fruta', () => {
  it('aparece nos dois terços do caminho e pode ser colhida', () => {
    const third = Math.floor(PELLET_TOTAL / 3);
    const g = { ...put(playing(), 9, 14, 'left'), eaten: third - 1 };
    const comFruta = run(g, 300).game;
    expect(comFruta.fruit).not.toBeNull();

    const colhendo = { ...comFruta, hero: { x: centerOf(HOUSE.col), y: centerOf(HOUSE.row + 2), dir: 'left' as Dir, next: null } };
    const { game, events } = run(colhendo, STEP * 2);
    expect(game.fruit).toBeNull();
    expect(game.basket).toHaveLength(1);
    expect(events.some((e) => e.kind === 'fruta' && e.points === FRUITS[0])).toBe(true);
  });

  it('a fruta some sozinha depois de um tempo', () => {
    const third = Math.floor(PELLET_TOTAL / 3);
    const g = { ...put(playing(), 9, 14, 'left'), eaten: third - 1 };
    const comFruta = run(g, 300).game;
    expect(run(comFruta, 10000).game.fruit).toBeNull();
  });

  it('vale mais a cada fase, até a oitava', () => {
    expect(fruitPoints(1)).toBe(100);
    expect(fruitPoints(8)).toBe(5000);
    expect(fruitPoints(30)).toBe(5000);
  });
});

describe('fantasmas', () => {
  it('cada um persegue uma coisa diferente', () => {
    const base = { ...playing(), wave: 1 }; // onda de caça
    // O ermitão fica longe, senão ele desiste (é a personalidade dele).
    const g = {
      ...base,
      ghosts: base.ghosts.map((x) =>
        x.id === 'ermitao' ? { ...x, state: 'solto' as const, x: centerOf(1), y: centerOf(1) } : x
      ),
    };
    const hero = { col: tileOf(g.hero.x), row: tileOf(g.hero.y) };
    const alvo = (id: string) => targetOf(g, g.ghosts.find((x) => x.id === id)!);

    // O caçador vai no jogador; o emboscador, quatro tiles à frente dele.
    expect(alvo('cacador')).toEqual(hero);
    expect(alvo('emboscador')).toEqual({ col: hero.col - 4, row: hero.row });
    // O flanqueador espelha o caçador através de dois tiles à frente.
    expect(alvo('flanqueador')).not.toEqual(hero);
    // O ermitão está longe daqui, então também persegue.
    expect(alvo('ermitao')).toEqual(hero);
  });

  it('o ermitão desiste quando chega perto', () => {
    const base = { ...playing(), wave: 1 };
    const perto = {
      ...base,
      ghosts: base.ghosts.map((x) =>
        x.id === 'ermitao' ? { ...x, state: 'solto' as const, x: base.hero.x + 1, y: base.hero.y } : x
      ),
    };
    const ermitao = perto.ghosts.find((x) => x.id === 'ermitao')!;
    expect(targetOf(perto, ermitao)).toEqual({ col: 1, row: 19 });
  });

  it('na onda de fuga cada um corre para o seu canto', () => {
    const g = { ...playing(), wave: 0 };
    const cantos = g.ghosts.map((ghost) => targetOf(g, ghost));
    expect(cantos).toEqual([
      { col: COLS - 2, row: 1 },
      { col: 1, row: 1 },
      { col: COLS - 2, row: 19 },
      { col: 1, row: 19 },
    ]);
  });

  it('a troca de onda vira os fantasmas soltos', () => {
    const g = playing();
    const antes = g.ghosts[0].dir;
    const depois = run({ ...g, waveLeft: STEP }, STEP * 2).game;
    expect(depois.wave).toBe(1);
    expect(depois.ghosts[0].dir).not.toBe(antes);
  });

  it('a casa esvazia com o tempo: todo fantasma sai pela porta', () => {
    let g = playing({ seed: 13 });
    const sairam = new Set<string>();
    for (let i = 0; i < 12000 / STEP; i++) {
      g = tick(g, STEP).game;
      for (const ghost of g.ghosts) if (ghost.state === 'solto') sairam.add(ghost.id);
    }
    expect([...sairam].sort()).toEqual([...GHOST_IDS].sort());
  });

  it('andam mais devagar na travessia e mais rápido de olhos', () => {
    expect(ghostSpeed(1, 'solto', true, 'normal')).toBeLessThan(
      ghostSpeed(1, 'solto', false, 'normal')
    );
    expect(ghostSpeed(1, 'assustado', false, 'normal')).toBeLessThan(
      ghostSpeed(1, 'solto', false, 'normal')
    );
    expect(ghostSpeed(1, 'olhos', false, 'normal')).toBeGreaterThan(heroSpeed(1));
  });

  it('o ajuste de ritmo só mexe no fantasma', () => {
    expect(ghostSpeed(1, 'solto', false, 'calma')).toBeLessThan(
      ghostSpeed(1, 'solto', false, 'normal')
    );
    expect(ghostSpeed(1, 'solto', false, 'veloz')).toBeGreaterThan(
      ghostSpeed(1, 'solto', false, 'normal')
    );
  });

  it('nenhum fantasma sai do labirinto ou entra na parede', () => {
    const g = run(playing({ seed: 11 }), 20000, { 1000: 'up', 4000: 'right', 9000: 'down' }).game;
    for (const ghost of g.ghosts) {
      expect(ghost.x).toBeGreaterThanOrEqual(0);
      expect(ghost.x).toBeLessThan(COLS);
      expect(ghost.y).toBeGreaterThan(0);
      expect(ghost.y).toBeLessThan(20);
    }
  });
});

describe('desafio do dia', () => {
  it('o mesmo dia devolve a mesma partida', () => {
    for (const day of ['2026-11-09', '2027-01-01']) {
      const a = dailyGame(day);
      const b = dailyGame(day);
      expect(JSON.stringify(a)).toBe(JSON.stringify(b));
      expect(a.daily).toBe(day);
      expect(a.goal).toBeGreaterThan(0);
    }
  });

  it('o mesmo roteiro de comandos dá o mesmo resultado', () => {
    const script: Record<number, Dir> = { 2000: 'up', 5000: 'left', 8000: 'down', 11000: 'right' };
    const a = run(dailyGame('2026-11-09'), 14000, script).game;
    const b = run(dailyGame('2026-11-09'), 14000, script).game;
    expect(JSON.stringify(a)).toBe(JSON.stringify(b));
    expect(a.score).toBeGreaterThan(0);
  });

  it('dias diferentes têm metas diferentes', () => {
    const metas = new Set(
      ['2026-11-09', '2026-11-10', '2026-11-11', '2026-11-12'].map((d) => dailyGoal(d))
    );
    expect(metas.size).toBeGreaterThan(1);
  });

  it('a meta é batida pela pontuação', () => {
    const g = dailyGame('2026-11-09');
    expect(reachedGoal(g)).toBe(false);
    expect(reachedGoal({ ...g, score: g.goal })).toBe(true);
    expect(reachedGoal({ ...g, goal: 0, score: 999 })).toBe(false);
  });

  it('o desafio do dia ignora o ajuste de ritmo', () => {
    expect(newGame({ daily: '2026-11-09', pace: 'calma' }).pace).toBe('normal');
    expect(newGame({ seed: 1, pace: 'calma' }).pace).toBe('calma');
  });

  it('o jogo não encosta em Math.random', () => {
    const real = Math.random;
    Math.random = () => {
      throw new Error('o jogo tem de ser determinístico');
    };
    try {
      expect(() => run(dailyGame('2026-11-09'), 6000, { 1500: 'up', 3000: 'right' })).not.toThrow();
    } finally {
      Math.random = real;
    }
  });

  it('a data vira a chave do dia no fuso local', () => {
    expect(dateKey(new Date(2026, 10, 9))).toBe('2026-11-09');
    expect(dateKey(new Date(2027, 0, 1))).toBe('2027-01-01');
  });
});

describe('placar', () => {
  it('formata com separador fino', () => {
    expect(fmtScore(0)).toBe('0');
    expect(fmtScore(1615)).toBe('1 615');
    expect(fmtScore(1234567)).toBe('1 234 567');
  });

  it('a pastilha vale 10 e o pastilhão 50', () => {
    const tiles = initialTiles();
    expect(tiles.filter((t) => t === POWER)).toHaveLength(4);
    expect(run(playing(), 150).game.score).toBe(10);
  });
});
