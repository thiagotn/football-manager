import { describe, expect, it } from 'vitest';

import { DEFAULT_SKIN, SKINS, isUnlocked, nextLock, skinOf, unlockedSkins } from './skins';

describe('visuais', () => {
  it('o primeiro é o padrão e sai de graça', () => {
    expect(SKINS[0].key).toBe(DEFAULT_SKIN);
    expect(SKINS[0].need).toBe(0);
  });

  it('as metas sobem sem empate', () => {
    const needs = SKINS.map((s) => s.need);
    expect(needs).toEqual([...needs].sort((a, b) => a - b));
    expect(new Set(needs).size).toBe(needs.length);
  });

  it('libera pela pontuação acumulada', () => {
    expect(unlockedSkins(0)).toEqual(['folha']);
    expect(unlockedSkins(500)).toContain('brasa');
    expect(unlockedSkins(499)).not.toContain('brasa');
    expect(unlockedSkins(999999)).toHaveLength(SKINS.length);
    expect(isUnlocked('neon', 7000)).toBe(true);
    expect(isUnlocked('neon', 6999)).toBe(false);
  });

  it('aponta o próximo a liberar, e nada quando todos saíram', () => {
    expect(nextLock(0)?.key).toBe('brasa');
    expect(nextLock(500)?.key).toBe('fita');
    expect(nextLock(999999)).toBeNull();
  });

  it('chave desconhecida cai no padrão — ajuste antigo não quebra o jogo', () => {
    expect(skinOf('cobra-de-ouro').key).toBe(DEFAULT_SKIN);
    expect(skinOf('neon').key).toBe('neon');
  });

  it('todo segmento tem cor, da cabeça à ponta', () => {
    for (const skin of SKINS) {
      for (const len of [1, 3, 40]) {
        for (let i = 0; i < len; i++) {
          expect(skin.body(i, len)).toMatch(/^(#|rgb|hsl)/);
        }
      }
      expect(skin.head).toMatch(/^#/);
      expect(skin.eye).toMatch(/^#/);
    }
  });
});
