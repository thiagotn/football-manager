/**
 * Persistência da Cobrinha em localStorage — estatísticas de sempre, histórico,
 * desafios diários e ajustes. Não há backend.
 *
 * A partida em andamento não é salva: a cobrinha acaba em segundos, e guardar o
 * meio do caminho só serviria para o jogador voltar e bater numa parede.
 */

import type { Game, Speed } from './engine';
import { levelFor } from './engine';
import { DEFAULT_SKIN, type SkinKey } from './skins';

const STORAGE_KEY = 'games_cobrinha';

/** Máximo de partidas guardadas e exibidas. */
export const HISTORY_LIMIT = 60;
export const HISTORY_VISIBLE = 10;

export type HistoryEntry = {
  /** Timestamp do fim da partida. */
  at: number;
  score: number;
  fruits: number;
  level: number;
  daily: string | null;
};

export type Stats = {
  games: number;
  /** Soma de tudo que já foi pontuado — é o que libera os visuais. */
  total: number;
  best: number;
  bestFruits: number;
  bestLevel: number;
};

/** Melhor resultado do dia; `done` quando a meta foi batida. */
export type DailyRecord = { best: number; done: boolean };

export type Settings = {
  /** Atravessar as paredes em vez de morrer nelas. */
  wrap: boolean;
  /** Mostrar o direcional na tela. */
  pad: boolean;
  speed: Speed;
  skin: SkinKey;
};

export type SavedState = {
  stats: Stats;
  history: HistoryEntry[];
  /** Por dia (`YYYY-MM-DD`). */
  daily: Record<string, DailyRecord>;
  settings: Settings;
};

export const DEFAULT_SETTINGS: Settings = {
  wrap: false,
  pad: true,
  speed: 'normal',
  skin: DEFAULT_SKIN,
};

export const EMPTY_STATS: Stats = { games: 0, total: 0, best: 0, bestFruits: 0, bestLevel: 0 };

export function emptyState(): SavedState {
  return { stats: { ...EMPTY_STATS }, history: [], daily: {}, settings: { ...DEFAULT_SETTINGS } };
}

export function loadState(): SavedState {
  if (typeof localStorage === 'undefined') return emptyState();
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return emptyState();
    const saved = JSON.parse(raw) as Partial<SavedState> | null;
    return {
      stats: { ...EMPTY_STATS, ...(saved?.stats ?? {}) },
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

/** Soma a partida às estatísticas de sempre. */
export function addToStats(stats: Stats, g: Game): Stats {
  const level = levelFor(g.eaten);
  return {
    games: stats.games + 1,
    total: stats.total + g.score,
    best: Math.max(stats.best, g.score),
    bestFruits: Math.max(stats.bestFruits, g.eaten),
    bestLevel: Math.max(stats.bestLevel, level),
  };
}

/** Prepende a partida ao histórico (mais recente primeiro). */
export function recordMatch(history: HistoryEntry[], g: Game): HistoryEntry[] {
  const entry: HistoryEntry = {
    at: Date.now(),
    score: g.score,
    fruits: g.eaten,
    level: levelFor(g.eaten),
    daily: g.daily,
  };
  return [entry, ...history].slice(0, HISTORY_LIMIT);
}

/** Guarda o melhor do dia e marca o desafio como concluído ao bater a meta. */
export function recordDaily(
  daily: Record<string, DailyRecord>,
  day: string,
  score: number,
  goal: number
): Record<string, DailyRecord> {
  const previous = daily[day] ?? { best: 0, done: false };
  return {
    ...daily,
    [day]: { best: Math.max(previous.best, score), done: previous.done || score >= goal },
  };
}

/** Dias concluídos em sequência até hoje (ontem ainda conta: o dia não acabou). */
export function streakOf(daily: Record<string, DailyRecord>, today: Date): number {
  const day = new Date(today);
  let streak = 0;
  // Começa por hoje; se hoje ainda não foi jogado, a sequência vem de ontem.
  for (let i = 0; i < 400; i++) {
    const key = `${day.getFullYear()}-${String(day.getMonth() + 1).padStart(2, '0')}-${String(
      day.getDate()
    ).padStart(2, '0')}`;
    if (daily[key]?.done) streak++;
    else if (i > 0) break;
    day.setDate(day.getDate() - 1);
  }
  return streak;
}
