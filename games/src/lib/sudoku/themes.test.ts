import { describe, expect, it } from 'vitest';
import { DEFAULT_THEME, THEMES, isUnlocked, nextLock, themeOf, unlockedThemes } from './themes';

describe('visuais do tabuleiro', () => {
  it('as metas sobem e a primeira é de graça', () => {
    expect(THEMES[0].need).toBe(0);
    expect(THEMES[0].key).toBe(DEFAULT_THEME);
    for (let i = 1; i < THEMES.length; i++) {
      expect(THEMES[i].need).toBeGreaterThan(THEMES[i - 1].need);
    }
  });

  it('libera conforme as partidas concluídas', () => {
    expect(unlockedThemes(0)).toEqual(['jornal']);
    expect(unlockedThemes(5)).toEqual(['jornal', 'linho']);
    expect(unlockedThemes(59)).toEqual(['jornal', 'linho', 'cafe', 'menta']);
    expect(unlockedThemes(1000)).toHaveLength(THEMES.length);
  });

  it('isUnlocked bate com a meta do tema', () => {
    expect(isUnlocked('ardosia', 59)).toBe(false);
    expect(isUnlocked('ardosia', 60)).toBe(true);
    expect(isUnlocked('jornal', 0)).toBe(true);
  });

  it('aponta o próximo a liberar', () => {
    expect(nextLock(0)?.key).toBe('linho');
    expect(nextLock(30)?.key).toBe('ardosia');
    expect(nextLock(100)).toBeNull();
  });

  it('chave salva que não existe mais cai no padrão', () => {
    expect(themeOf('xadrez').key).toBe(DEFAULT_THEME);
    expect(themeOf(null).key).toBe(DEFAULT_THEME);
    expect(themeOf('menta').key).toBe('menta');
  });

  it('todo tema tem as onze cores preenchidas', () => {
    for (const t of THEMES) {
      for (const field of ['paper', 'paperAlt', 'line', 'edge', 'given', 'pen', 'note', 'sel', 'peer', 'same', 'bad'] as const) {
        expect(t[field], `${t.key}.${field}`).toMatch(/^#[0-9a-f]{6}$/);
      }
    }
  });
});
