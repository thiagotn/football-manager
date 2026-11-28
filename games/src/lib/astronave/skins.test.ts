import { describe, expect, it } from 'vitest';

import { SHIP_HALF } from './engine';
import { DEFAULT_SKIN, SHIP_SCALE, SKINS, isUnlocked, nextLock, skinOf, unlockedSkins } from './skins';

describe('naves', () => {
  it('a primeira é a padrão e sai de graça', () => {
    expect(SKINS[0].key).toBe(DEFAULT_SKIN);
    expect(SKINS[0].need).toBe(0);
  });

  it('as metas sobem sem empate', () => {
    const needs = SKINS.map((s) => s.need);
    expect(needs).toEqual([...needs].sort((a, b) => a - b));
    expect(new Set(needs).size).toBe(needs.length);
  });

  it('libera pela pontuação acumulada', () => {
    expect(unlockedSkins(0)).toEqual(['pioneira']);
    expect(unlockedSkins(600)).toContain('cometa');
    expect(unlockedSkins(599)).not.toContain('cometa');
    expect(unlockedSkins(999999)).toHaveLength(SKINS.length);
    expect(isUnlocked('neon', 8000)).toBe(true);
    expect(isUnlocked('neon', 7999)).toBe(false);
  });

  it('aponta a próxima a liberar, e nada quando todas saíram', () => {
    expect(nextLock(0)?.key).toBe('cometa');
    expect(nextLock(600)?.key).toBe('brasa');
    expect(nextLock(999999)).toBeNull();
  });

  it('chave desconhecida cai na padrão — ajuste antigo não quebra o jogo', () => {
    expect(skinOf('nave-de-ouro').key).toBe(DEFAULT_SKIN);
    expect(skinOf('neon').key).toBe('neon');
  });

  it('toda nave tem as cinco cores em hexadecimal', () => {
    for (const skin of SKINS) {
      for (const color of [skin.hull, skin.wing, skin.cockpit, skin.flame, skin.shot]) {
        expect(color).toMatch(/^#[0-9a-f]{6}$/);
      }
    }
  });

  it('o desenho tem a largura que o engine usa nas bordas', () => {
    // A asa vai até x = ±8 no contorno.
    expect(8 * SHIP_SCALE).toBeCloseTo(SHIP_HALF, 1);
  });
});
