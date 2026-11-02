/**
 * Persistência do Enche Aí! em localStorage — pedido em andamento, progresso,
 * pedidos do dia concluídos e ajustes. Não há backend.
 */

import type { AidCount, AidUsage, Board } from './engine';

const STORAGE_KEY = 'games_enche_ai';

export type CurrentGame = {
  level: number;
  daily: string | null;
  board: Board;
  history: Board[];
  moves: number;
  elapsed: number;
  left: AidCount;
  used: AidUsage;
};

export type DailyRecord = { moves: number; time: number; stars: number };

export type Settings = {
  /** Texturas nos sabores (acessibilidade para daltonismo). */
  patterns: boolean;
  vibrate: boolean;
};

export type SavedState = {
  current: CurrentGame | null;
  /** Próximo pedido normal a jogar (o maior já liberado). */
  level: number;
  /** Melhor nº de estrelas por pedido. */
  stars: Record<number, number>;
  /** Melhor resultado por dia (`YYYY-MM-DD`). */
  daily: Record<string, DailyRecord>;
  settings: Settings;
};

export const DEFAULT_SETTINGS: Settings = { patterns: true, vibrate: true };

export function emptyState(): SavedState {
  return { current: null, level: 1, stars: {}, daily: {}, settings: { ...DEFAULT_SETTINGS } };
}

export function loadState(): SavedState {
  if (typeof localStorage === 'undefined') return emptyState();
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return emptyState();
    const saved = JSON.parse(raw) as Partial<SavedState> | null;
    return {
      current: saved?.current ?? null,
      level: Math.max(1, Number(saved?.level) || 1),
      stars: saved?.stars ?? {},
      daily: saved?.daily ?? {},
      settings: { ...DEFAULT_SETTINGS, ...(saved?.settings ?? {}) },
    };
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

/** Histórico de desfazer guardado no máximo com este tamanho. */
export const HISTORY_LIMIT = 50;
