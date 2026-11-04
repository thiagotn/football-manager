import { describe, expect, it } from 'vitest';

import {
  BONUS_STEPS,
  COLS,
  ROWS,
  dailyGoal,
  dailySeed,
  dateKey,
  fmtScore,
  isDeadly,
  levelFor,
  newGame,
  reachedGoal,
  samePoint,
  scoreForBonus,
  scoreForFruit,
  step,
  tickMs,
  turn,
  wallsFor,
  type Dir,
  type Game,
  type Point,
} from './engine';

/** Partida de teste: cobra e comida onde eu quiser, sem sortear nada. */
function game(patch: Partial<Game> = {}): Game {
  const base = newGame({ seed: 1 });
  return { ...base, fruit: { x: -5, y: -5 }, ...patch };
}

/** Aplica N passos e devolve o estado final. */
function run(g: Game, times: number): Game {
  let current = g;
  for (let i = 0; i < times; i++) current = step(current).game;
  return current;
}

const head = (g: Game): Point => g.snake[0];

describe('partida nova', () => {
  it('nasce com três segmentos no meio, andando para a direita', () => {
    const g = newGame({ seed: 7 });
    expect(g.snake).toHaveLength(3);
    expect(head(g)).toEqual({ x: Math.floor(COLS / 2), y: Math.floor(ROWS / 2) });
    expect(g.dir).toBe('right');
    expect(g.over).toBe(false);
    expect(g.score).toBe(0);
  });

  it('a fruta nasce fora da cobra e dentro da grade', () => {
    for (let seed = 0; seed < 40; seed++) {
      const g = newGame({ seed });
      expect(g.snake.some((s) => samePoint(s, g.fruit))).toBe(false);
      expect(g.fruit.x).toBeGreaterThanOrEqual(0);
      expect(g.fruit.x).toBeLessThan(COLS);
      expect(g.fruit.y).toBeGreaterThanOrEqual(0);
      expect(g.fruit.y).toBeLessThan(ROWS);
    }
  });
});

describe('movimento', () => {
  it('anda sem crescer', () => {
    const g = run(game(), 3);
    expect(g.snake).toHaveLength(3);
    expect(head(g)).toEqual({ x: Math.floor(COLS / 2) + 3, y: Math.floor(ROWS / 2) });
  });

  it('o passo devolve um estado novo e não mexe no anterior', () => {
    const before = game();
    const snapshot = JSON.stringify(before);
    const after = step(before).game;
    expect(JSON.stringify(before)).toBe(snapshot);
    expect(after).not.toBe(before);
  });

  it('enfileira até duas curvas e aplica uma por passo', () => {
    let g = game();
    g = turn(g, 'up');
    g = turn(g, 'left');
    expect(g.queued).toEqual(['up', 'left']);

    g = step(g).game;
    expect(g.dir).toBe('up');
    g = step(g).game;
    expect(g.dir).toBe('left');
    expect(g.queued).toEqual([]);
  });

  it('ignora a terceira curva enfileirada', () => {
    let g = turn(turn(game(), 'up'), 'left');
    g = turn(g, 'down');
    expect(g.queued).toHaveLength(2);
  });

  it('ignora a inversão de 180° e a direção repetida', () => {
    const g = game();
    expect(turn(g, 'left').queued).toEqual([]);
    expect(turn(g, 'right').queued).toEqual([]);
    // Também em relação à última curva da fila, não só à direção atual.
    expect(turn(turn(g, 'up'), 'down').queued).toEqual(['up']);
  });
});

describe('comer', () => {
  it('cresce um segmento e pontua pelo nível', () => {
    const g = game({ fruit: { x: Math.floor(COLS / 2) + 1, y: Math.floor(ROWS / 2) } });
    const { game: after, event } = step(g);
    expect(event).toBe('ate');
    expect(after.snake).toHaveLength(4);
    expect(after.eaten).toBe(1);
    expect(after.score).toBe(scoreForFruit(1));
  });

  it('a fruta seguinte nasce fora da cobra', () => {
    const g = game({ fruit: { x: Math.floor(COLS / 2) + 1, y: Math.floor(ROWS / 2) } });
    const after = step(g).game;
    expect(after.snake.some((s) => samePoint(s, after.fruit))).toBe(false);
  });

  it('a quinta fruta põe uma bônus na mesa', () => {
    let g = game({ eaten: 4, fruit: { x: Math.floor(COLS / 2) + 1, y: Math.floor(ROWS / 2) } });
    g = step(g).game;
    expect(g.bonus).not.toBeNull();
    expect(g.bonusLeft).toBe(BONUS_STEPS);
  });

  it('a bônus expira sozinha depois de BONUS_STEPS passos', () => {
    // Travessia ligada e corredor livre: a cobra só anda enquanto o tempo corre.
    const corridor = game({
      wrap: true,
      snake: [{ x: 2, y: 0 }, { x: 1, y: 0 }, { x: 0, y: 0 }],
      fruit: { x: 7, y: 7 },
      bonus: { x: 5, y: 5 },
      bonusLeft: 3,
    });
    expect(run(corridor, 2).bonus).not.toBeNull();
    const expired = run(corridor, 3);
    expect(expired.over).toBe(false);
    expect(expired.bonus).toBeNull();
    expect(expired.bonusLeft).toBe(0);
  });

  it('a bônus vale cinco frutas e some ao ser comida', () => {
    const y = Math.floor(ROWS / 2);
    const x = Math.floor(COLS / 2);
    const g = game({ eaten: 5, bonus: { x: x + 1, y }, bonusLeft: 10 });
    const { game: after, event } = step(g);
    expect(event).toBe('bonus');
    expect(after.score).toBe(scoreForBonus(2));
    expect(after.bonus).toBeNull();
    // Bônus não engorda: só a fruta comum cresce.
    expect(after.snake).toHaveLength(3);
  });
});

describe('morte', () => {
  it('bate na parede quando a travessia está desligada', () => {
    const g = game({ snake: [{ x: COLS - 1, y: 2 }, { x: COLS - 2, y: 2 }] });
    const { game: after, event } = step(g);
    expect(event).toBe('died');
    expect(after.over).toBe(true);
  });

  it('com travessia ligada sai do outro lado', () => {
    const g = game({ wrap: true, snake: [{ x: COLS - 1, y: 2 }, { x: COLS - 2, y: 2 }] });
    const after = step(g).game;
    expect(after.over).toBe(false);
    expect(head(after)).toEqual({ x: 0, y: 2 });
  });

  it('morre na própria cauda ao fechar o círculo', () => {
    // Quadrado 2×2: virar para baixo e para a esquerda leva a cabeça ao corpo.
    let g = game({
      snake: [
        { x: 5, y: 5 },
        { x: 4, y: 5 },
        { x: 4, y: 6 },
        { x: 5, y: 6 },
        { x: 6, y: 6 },
      ],
    });
    g = turn(g, 'down');
    expect(step(g).event).toBe('died');
  });

  it('a ponta da cauda não mata: ela sai do lugar no mesmo passo', () => {
    const g = game({
      snake: [
        { x: 5, y: 5 },
        { x: 4, y: 5 },
        { x: 4, y: 6 },
        { x: 5, y: 6 },
      ],
    });
    expect(isDeadly(g, { x: 5, y: 6 })).toBe(false);
    expect(isDeadly(g, { x: 4, y: 6 })).toBe(true);
  });

  it('morre no obstáculo do desafio do dia', () => {
    const g = game({ walls: [{ x: 6, y: 3 }], snake: [{ x: 5, y: 3 }, { x: 4, y: 3 }] });
    expect(step(g).event).toBe('died');
  });

  it('o passo não faz nada depois do fim', () => {
    const over = game({ over: true });
    const { game: after, event } = step(over);
    expect(event).toBeNull();
    expect(after).toBe(over);
  });
});

describe('ritmo e pontuação', () => {
  it('o nível sobe a cada cinco frutas', () => {
    expect(levelFor(0)).toBe(1);
    expect(levelFor(4)).toBe(1);
    expect(levelFor(5)).toBe(2);
    expect(levelFor(23)).toBe(5);
  });

  it('o intervalo encurta com o nível e trava no piso', () => {
    expect(tickMs(2, 'normal')).toBeLessThan(tickMs(1, 'normal'));
    expect(tickMs(99, 'normal')).toBe(tickMs(100, 'normal'));
    expect(tickMs(1, 'veloz')).toBeLessThan(tickMs(1, 'calma'));
  });

  it('formata o placar com separador fino', () => {
    expect(fmtScore(0)).toBe('0');
    expect(fmtScore(999)).toBe('999');
    expect(fmtScore(1615)).toBe('1 615');
    expect(fmtScore(1234567)).toBe('1 234 567');
  });
});

describe('desafio do dia', () => {
  it('a mesma data gera a mesma partida', () => {
    const a = newGame({ daily: '2026-11-03' });
    const b = newGame({ daily: '2026-11-03' });
    expect(b.fruit).toEqual(a.fruit);
    expect(b.walls).toEqual(a.walls);
    expect(b.goal).toBe(a.goal);
  });

  it('datas diferentes geram partidas diferentes', () => {
    const a = newGame({ daily: '2026-11-03' });
    const b = newGame({ daily: '2026-11-04' });
    expect(dailySeed('2026-11-03')).not.toBe(dailySeed('2026-11-04'));
    expect([a.fruit, a.walls]).not.toEqual([b.fruit, b.walls]);
  });

  it('a sequência de frutas se repete passo a passo', () => {
    const sequence = (): Point[] => {
      let g = newGame({ daily: '2026-12-25' });
      const fruits: Point[] = [g.fruit];
      // Leva a cabeça até a fruta algumas vezes, sempre pelo mesmo caminho.
      for (let i = 0; i < 60; i++) {
        const dirs: Dir[] = g.fruit.x > head(g).x ? ['right'] : ['left'];
        if (g.fruit.y !== head(g).y) dirs.unshift(g.fruit.y > head(g).y ? 'down' : 'up');
        for (const d of dirs) g = turn(g, d);
        const before = g.eaten;
        g = step(g).game;
        if (g.over) break;
        if (g.eaten > before) fruits.push(g.fruit);
      }
      return fruits;
    };
    expect(sequence()).toEqual(sequence());
  });

  it('ignora os ajustes do jogador, para o resultado ser comparável', () => {
    const g = newGame({ daily: '2026-11-03', wrap: true, speed: 'veloz' });
    expect(g.wrap).toBe(false);
    expect(g.speed).toBe('normal');
  });

  it('a meta é batida pela pontuação', () => {
    const g = newGame({ daily: '2026-11-03' });
    expect(reachedGoal(g)).toBe(false);
    expect(reachedGoal({ ...g, score: g.goal })).toBe(true);
    // Partida livre não tem meta.
    expect(reachedGoal({ ...newGame({ seed: 3 }), score: 9999 })).toBe(false);
  });

  it('a meta fica numa faixa jogável', () => {
    for (const day of ['2026-01-01', '2026-06-15', '2026-11-03', '2027-02-28']) {
      expect(dailyGoal(day)).toBeGreaterThanOrEqual(150);
      expect(dailyGoal(day)).toBeLessThanOrEqual(300);
    }
  });

  it('os obstáculos deixam livre a faixa onde a cobra nasce', () => {
    const mid = Math.floor(ROWS / 2);
    for (const day of ['2026-01-01', '2026-03-09', '2026-11-03', '2027-07-21']) {
      const walls = wallsFor(day);
      expect(walls.length).toBeGreaterThan(0);
      for (const w of walls) {
        expect(Math.abs(w.y - mid)).toBeGreaterThan(1);
        expect(w.x).toBeGreaterThanOrEqual(0);
        expect(w.x).toBeLessThan(COLS);
        expect(w.y).toBeGreaterThanOrEqual(0);
        expect(w.y).toBeLessThan(ROWS);
      }
    }
  });
});

describe('data', () => {
  it('usa o dia local, não o UTC', () => {
    expect(dateKey(new Date(2026, 10, 3, 23, 30))).toBe('2026-11-03');
    expect(dateKey(new Date(2026, 0, 9, 0, 5))).toBe('2026-01-09');
  });
});
