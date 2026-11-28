/**
 * Persistência da Astronave em localStorage — estatísticas de sempre,
 * histórico, desafios diários e ajustes. Não há backend.
 *
 * A partida em andamento não é salva: voltar no meio de uma chuva de meteoros,
 * sem o dedo no lugar, seria perder uma vida de graça.
 */

import type { Game } from './engine';
import { DEFAULT_SKIN, type SkinKey } from './skins';

const STORAGE_KEY = 'games_astronave';

/** Máximo de partidas guardadas e exibidas. */
export const HISTORY_LIMIT = 60;
export const HISTORY_VISIBLE = 10;

export type HistoryEntry = {
  /** Timestamp do fim da partida. */
  at: number;
  score: number;
  wave: number;
  kills: number;
  daily: string | null;
};

export type Stats = {
  games: number;
  /** Soma de tudo que já foi pontuado — é o que libera as naves. */
  total: number;
  best: number;
  bestWave: number;
  bestKills: number;
};

/** Melhor resultado do dia; `done` quando a meta foi batida. */
export type DailyRecord = { best: number; done: boolean };

/**
 * `touch`: só o toque — arrastar move, segurar atira, e a arena usa a tela toda.
 * `pad`: botões de mover e de atirar embaixo da arena.
 */
export type Controls = 'touch' | 'pad';

export type Settings = {
  controls: Controls;
  skin: SkinKey;
};

export type SavedState = {
  stats: Stats;
  history: HistoryEntry[];
  /** Por dia (`YYYY-MM-DD`). */
  daily: Record<string, DailyRecord>;
  settings: Settings;
};

export const DEFAULT_SETTINGS: Settings = { controls: 'touch', skin: DEFAULT_SKIN };

export const EMPTY_STATS: Stats = { games: 0, total: 0, best: 0, bestWave: 0, bestKills: 0 };

export function emptyState(): SavedState {
  return { stats: { ...EMPTY_STATS }, history: [], daily: {}, settings: { ...DEFAULT_SETTINGS } };
}

/** Ajuste salvo com valor estranho cai no padrão em vez de quebrar a tela. */
function cleanSettings(raw: Partial<Settings> | undefined): Settings {
  return {
    controls: raw?.controls === 'pad' ? 'pad' : 'touch',
    skin: typeof raw?.skin === 'string' ? raw.skin : DEFAULT_SKIN,
  };
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
      settings: cleanSettings(saved?.settings),
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
  return {
    games: stats.games + 1,
    total: stats.total + g.score,
    best: Math.max(stats.best, g.score),
    bestWave: Math.max(stats.bestWave, g.wave),
    bestKills: Math.max(stats.bestKills, g.kills),
  };
}

/** Prepende a partida ao histórico (mais recente primeiro). */
export function recordMatch(history: HistoryEntry[], g: Game): HistoryEntry[] {
  const entry: HistoryEntry = {
    at: Date.now(),
    score: g.score,
    wave: g.wave,
    kills: g.kills,
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
