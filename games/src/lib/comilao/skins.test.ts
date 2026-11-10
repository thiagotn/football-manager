import { describe, expect, it } from 'vitest';
import {
  DEFAULT_SKIN,
  GHOST_COLORS,
  SKINS,
  isUnlocked,
  nextLock,
  skinOf,
  unlockedSkins,
} from './skins';
import { GHOST_IDS } from './engine';

describe('visuais', () => {
  it('as metas sobem na ordem e a primeira é de graça', () => {
    expect(SKINS[0].need).toBe(0);
    expect(SKINS[0].key).toBe(DEFAULT_SKIN);
    for (let i = 1; i < SKINS.length; i++) expect(SKINS[i].need).toBeGreaterThan(SKINS[i - 1].need);
  });

  it('libera pela pontuação somada', () => {
    expect(unlockedSkins(0)).toEqual(['synth']);
    expect(isUnlocked('vapor', 1499)).toBe(false);
    expect(isUnlocked('vapor', 1500)).toBe(true);
    expect(unlockedSkins(1e6)).toHaveLength(SKINS.length);
  });

  it('diz qual é o próximo a liberar', () => {
    expect(nextLock(0)?.key).toBe('vapor');
    expect(nextLock(1e6)).toBeNull();
  });

  it('cai no padrão quando a chave salva não existe mais', () => {
    expect(skinOf('miragem').key).toBe(DEFAULT_SKIN);
    expect(skinOf('gelo').key).toBe('gelo');
  });

  it('todo visual tem as cores que o desenho usa', () => {
    for (const skin of SKINS) {
      expect(skin.wall).toHaveLength(2);
      expect(skin.floor).toHaveLength(2);
      for (const color of [skin.hero, skin.pellet, skin.power, skin.glow, skin.grid]) {
        expect(color).toMatch(/^(#|rgba?\()/);
      }
    }
  });

  it('cada fantasma tem cor própria, igual em todos os visuais', () => {
    expect(Object.keys(GHOST_COLORS).sort()).toEqual([...GHOST_IDS].sort());
    expect(new Set(Object.values(GHOST_COLORS)).size).toBe(GHOST_IDS.length);
  });
});

describe('persistência', () => {
  it('descarta partida salva de formato velho e aceita a boa', async () => {
    const { resumeGame } = await import('./storage');
    const { newGame, READY_MS } = await import('./engine');
    const g = newGame({ seed: 1 });
    const retomada = resumeGame({ ...g, phase: 'jogando', phaseLeft: 0 });
    expect(retomada.phase).toBe('pronto');
    expect(retomada.phaseLeft).toBe(READY_MS);
    expect(retomada.score).toBe(g.score);
  });

  it('soma a partida às estatísticas e ao histórico', async () => {
    const { EMPTY_STATS, addToStats, recordMatch, recordDaily, streakOf } = await import('./storage');
    const { newGame } = await import('./engine');
    const g = { ...newGame({ seed: 1 }), score: 2400, level: 3, pellets: 180, ghostsEaten: 5 };

    const stats = addToStats(EMPTY_STATS, g);
    expect(stats).toEqual({ games: 1, total: 2400, best: 2400, bestLevel: 3, pellets: 180, ghosts: 5 });
    expect(addToStats(stats, { ...g, score: 900 }).best).toBe(2400);
    expect(addToStats(stats, { ...g, score: 900 }).total).toBe(3300);

    expect(recordMatch([], g)[0].score).toBe(2400);

    const daily = recordDaily({}, '2026-11-09', 1500, 1200);
    expect(daily['2026-11-09']).toEqual({ best: 1500, done: true });
    expect(recordDaily(daily, '2026-11-09', 100, 1200)['2026-11-09']).toEqual({ best: 1500, done: true });
    expect(recordDaily({}, '2026-11-09', 100, 1200)['2026-11-09'].done).toBe(false);

    expect(streakOf(daily, new Date(2026, 10, 9))).toBe(1);
    expect(streakOf(daily, new Date(2026, 10, 11))).toBe(0);
  });
});
