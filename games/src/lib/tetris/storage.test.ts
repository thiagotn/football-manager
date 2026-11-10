import { afterEach, describe, expect, it, vi } from 'vitest';

import { DEFAULT_SETTINGS, emptyState, loadState, saveState } from './storage';

/** localStorage de mentira: o vitest roda em node, onde ele não existe. */
function fakeStorage(seed: Record<string, string> = {}) {
  const map = new Map(Object.entries(seed));
  return {
    getItem: (key: string) => map.get(key) ?? null,
    setItem: (key: string, value: string) => {
      map.set(key, value);
    },
    removeItem: (key: string) => {
      map.delete(key);
    },
    clear: () => map.clear(),
    key: () => null,
    length: 0,
  } as unknown as Storage;
}

afterEach(() => vi.unstubAllGlobals());

describe('ajustes do tetris', () => {
  it('o 3D vem desligado por padrão', () => {
    expect(DEFAULT_SETTINGS.depth).toBe(false);
    expect(emptyState()).toEqual({ settings: { depth: false }, asked: false });
  });

  it('sem nada salvo, carrega o padrão', () => {
    vi.stubGlobal('localStorage', fakeStorage());
    expect(loadState()).toEqual(emptyState());
  });

  it('guarda e devolve a escolha do jogador', () => {
    vi.stubGlobal('localStorage', fakeStorage());
    saveState({ settings: { depth: true }, asked: true });
    expect(loadState()).toEqual({ settings: { depth: true }, asked: true });
  });

  it('json quebrado não derruba o jogo', () => {
    vi.stubGlobal('localStorage', fakeStorage({ games_tetris: '{isso não é json' }));
    expect(loadState()).toEqual(emptyState());
  });

  it('valor estranho no ajuste cai para desligado', () => {
    vi.stubGlobal('localStorage', fakeStorage({ games_tetris: '{"settings":{"depth":"sim"},"asked":1}' }));
    expect(loadState()).toEqual(emptyState());
  });

  it('sem localStorage, carrega o padrão e salvar não explode', () => {
    vi.stubGlobal('localStorage', undefined);
    expect(loadState()).toEqual(emptyState());
    expect(() => saveState({ settings: { depth: true }, asked: true })).not.toThrow();
  });
});
