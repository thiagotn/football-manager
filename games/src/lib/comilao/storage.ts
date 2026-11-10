/**
 * Persistência do Comilão em localStorage — partida em andamento, estatísticas
 * de sempre, histórico, desafios diários e ajustes. Não há backend.
 *
 * A partida **é** salva (como no Sudoku, diferente da Cobrinha): três vidas
 * levam minutos e uma ligação no meio não pode custar a corrida. Quem retoma
 * cai sempre na pausa do "Prepare-se" — ver `resumeGame`.
 */

import { COLS, READY_MS, ROWS, type Game, type Pace } from './engine';
import { DEFAULT_SKIN, type SkinKey } from './skins';

const STORAGE_KEY = 'games_comilao';

/** Máximo de partidas guardadas e exibidas. */
export const HISTORY_LIMIT = 60;
export const HISTORY_VISIBLE = 10;

export type HistoryEntry = {
  /** Timestamp do fim da partida. */
  at: number;
  score: number;
  level: number;
  pellets: number;
  ghosts: number;
  daily: string | null;
};

export type Stats = {
  games: number;
  /** Soma de tudo que já foi pontuado — é o que libera os visuais. */
  total: number;
  best: number;
  bestLevel: number;
  pellets: number;
  ghosts: number;
};

/** Melhor resultado do dia; `done` quando a meta foi batida. */
export type DailyRecord = { best: number; done: boolean };

export type Settings = {
  /** Mostrar o direcional na tela. */
  pad: boolean;
  /** Brilho neon nas paredes (desligar ajuda em tela fraca e economiza bateria). */
  glow: boolean;
  /** Ritmo dos fantasmas. */
  pace: Pace;
  skin: SkinKey;
};

export type SavedState = {
  /** Partida em andamento, para o botão Continuar. */
  game: Game | null;
  stats: Stats;
  history: HistoryEntry[];
  /** Por dia (`YYYY-MM-DD`). */
  daily: Record<string, DailyRecord>;
  settings: Settings;
};

export const DEFAULT_SETTINGS: Settings = {
  pad: true,
  glow: true,
  pace: 'normal',
  skin: DEFAULT_SKIN,
};

export const EMPTY_STATS: Stats = {
  games: 0,
  total: 0,
  best: 0,
  bestLevel: 0,
  pellets: 0,
  ghosts: 0,
};

export function emptyState(): SavedState {
  return {
    game: null,
    stats: { ...EMPTY_STATS },
    history: [],
    daily: {},
    settings: { ...DEFAULT_SETTINGS },
  };
}

export function loadState(): SavedState {
  if (typeof localStorage === 'undefined') return emptyState();
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return emptyState();
    const saved = JSON.parse(raw) as Partial<SavedState> | null;
    return {
      game: validGame(saved?.game) ? (saved!.game as Game) : null,
      stats: { ...EMPTY_STATS, ...(saved?.stats ?? {}) },
      history: saved?.history ?? [],
      daily: saved?.daily ?? {},
      settings: { ...DEFAULT_SETTINGS, ...(saved?.settings ?? {}) },
    };
  } catch {
    return emptyState();
  }
}

/**
 * A partida salva veio de um JSON qualquer: sem a grade inteira, sem os quatro
 * fantasmas ou já encerrada, é melhor descartar que montar uma tela quebrada
 * com o formato de uma versão antiga.
 */
function validGame(game: Game | null | undefined): boolean {
  if (!game) return false;
  if (!Array.isArray(game.tiles) || game.tiles.length !== COLS * ROWS) return false;
  if (!Array.isArray(game.ghosts) || game.ghosts.length !== 4) return false;
  if (!game.hero || typeof game.hero.x !== 'number') return false;
  return game.phase !== 'fim' && game.lives > 0;
}

/**
 * Retomar não devolve o jogador no meio do corredor com um fantasma a dois
 * tiles: volta na pausa do "Prepare-se", que é o beijo de misericórdia.
 */
export const resumeGame = (game: Game): Game => ({
  ...game,
  phase: game.phase === 'fim' ? 'fim' : 'pronto',
  phaseLeft: READY_MS,
});

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
    bestLevel: Math.max(stats.bestLevel, g.level),
    pellets: stats.pellets + g.pellets,
    ghosts: stats.ghosts + g.ghostsEaten,
  };
}

/** Prepende a partida ao histórico (mais recente primeiro). */
export function recordMatch(history: HistoryEntry[], g: Game): HistoryEntry[] {
  const entry: HistoryEntry = {
    at: Date.now(),
    score: g.score,
    level: g.level,
    pellets: g.pellets,
    ghosts: g.ghostsEaten,
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
