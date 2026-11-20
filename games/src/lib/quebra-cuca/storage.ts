/**
 * Persistência do Quebra-Cuca em localStorage (`games_quebra_cuca`): o maior
 * número de peças já montado em cada cena e os ajustes. A partida em
 * andamento **não** é salva — um quebra-cabeça de criança dura minutos.
 */

import { SIZES } from './jigsaw';

const STORAGE_KEY = 'games_quebra_cuca';

export type Settings = {
  /** Figura clarinha no tabuleiro, como guia. */
  ghost: boolean;
  voice: boolean;
  /** Último número de peças escolhido (volta selecionado). */
  pieces: number;
};

export type SavedState = {
  /** Maior número de peças montado por cena (`0`/ausente = nunca montou). */
  done: Record<string, number>;
  settings: Settings;
};

export const DEFAULT_SETTINGS: Settings = { ghost: true, voice: true, pieces: 4 };

export const emptyState = (): SavedState => ({ done: {}, settings: { ...DEFAULT_SETTINGS } });

export function loadState(): SavedState {
  if (typeof localStorage === 'undefined') return emptyState();
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return emptyState();
    const s = JSON.parse(raw) as Partial<SavedState> | null;
    if (!s || typeof s !== 'object') return emptyState();
    const settings = { ...DEFAULT_SETTINGS, ...(s.settings ?? {}) };
    if (!SIZES.some((z) => z.pieces === settings.pieces)) settings.pieces = DEFAULT_SETTINGS.pieces;
    const done: Record<string, number> = {};
    for (const [id, n] of Object.entries(s.done ?? {})) if (typeof n === 'number' && n > 0) done[id] = n;
    return { done, settings };
  } catch {
    return emptyState();
  }
}

export function saveState(state: SavedState): void {
  if (typeof localStorage === 'undefined') return;
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
  } catch {
    // Cota cheia ou modo privado — o jogo segue sem persistir.
  }
}

/** Registra a cena montada, guardando o maior número de peças. */
export function recordDone(state: SavedState, sceneId: string, pieces: number): SavedState {
  return { ...state, done: { ...state.done, [sceneId]: Math.max(state.done[sceneId] ?? 0, pieces) } };
}

/** Estrelas da cena: 1 a 4 conforme o maior tamanho montado (4, 6, 9, 12 peças). */
export const starsOf = (pieces: number | undefined): number =>
  pieces ? SIZES.findIndex((s) => s.pieces === pieces) + 1 : 0;
