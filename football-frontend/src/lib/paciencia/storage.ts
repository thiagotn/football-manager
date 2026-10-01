/**
 * Persistência da Paciência em localStorage — partida atual, histórico,
 * desafios diários concluídos e ajustes. Não há backend.
 */

import type { DrawMode, Game } from './engine';
import type { CardBackKey } from './backs';

const STORAGE_KEY = 'rachao_paciencia';

/** Máximo de partidas guardadas no histórico. */
export const HISTORY_LIMIT = 60;
/** Máximo de partidas exibidas em "Últimas partidas". */
export const HISTORY_VISIBLE = 20;

export type HistoryEntry = {
  /** Timestamp do fim da partida. */
  at: number;
  won: boolean;
  time: number;
  moves: number;
  score: number;
  daily: string | null;
};

export type DailyRecord = { time: number; moves: number };

export type Settings = {
  draw: DrawMode;
  timer: boolean;
  autoFinish: boolean;
  back: CardBackKey;
};

export type SavedState = {
  game: Game | null;
  history: HistoryEntry[];
  /** Melhor resultado por dia (`YYYY-MM-DD`). */
  daily: Record<string, DailyRecord>;
  settings: Settings;
};

export const DEFAULT_SETTINGS: Settings = {
  draw: 1,
  timer: true,
  autoFinish: true,
  back: 'escudo',
};

export function emptyState(): SavedState {
  return { game: null, history: [], daily: {}, settings: { ...DEFAULT_SETTINGS } };
}

export function loadState(): SavedState {
  if (typeof localStorage === 'undefined') return emptyState();
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return emptyState();
    const saved = JSON.parse(raw) as Partial<SavedState> | null;
    return {
      game: saved?.game ?? null,
      history: saved?.history ?? [],
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

/**
 * Prepende a partida ao histórico (mais recente primeiro), respeitando o limite.
 * Abandonar uma partida com movimentos registra `won: false`.
 */
export function recordMatch(history: HistoryEntry[], g: Game, won: boolean): HistoryEntry[] {
  const entry: HistoryEntry = {
    at: Date.now(),
    won,
    time: g.elapsed,
    moves: g.moves,
    score: g.score,
    daily: g.daily,
  };
  return [entry, ...history].slice(0, HISTORY_LIMIT);
}
