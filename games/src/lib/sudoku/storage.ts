/**
 * Persistência do Sudoku em localStorage — partida em andamento, estatísticas
 * por dificuldade, histórico, tabuleiros do dia e ajustes. Não há backend.
 *
 * Diferente da Cobrinha, a partida **é** salva: um sudoku leva minutos e ter de
 * começar de novo porque o celular trocou de aba seria cruel.
 */

import type { Difficulty, Game } from './engine';
import { DIFFICULTIES } from './engine';
import { DEFAULT_THEME, type ThemeKey } from './themes';

const STORAGE_KEY = 'games_sudoku';

/** Máximo de partidas guardadas e exibidas. */
export const HISTORY_LIMIT = 60;
export const HISTORY_VISIBLE = 10;

export type HistoryEntry = {
  /** Timestamp do fim da partida. */
  at: number;
  difficulty: Difficulty;
  /** Segundos. */
  time: number;
  hints: number;
  mistakes: number;
  daily: string | null;
};

export type DifficultyStats = {
  games: number;
  wins: number;
  /** Melhor tempo em segundos (0 = ainda não ganhou). */
  best: number;
  /** Menos dicas e menos erros numa vitória — a coluna "de sempre" do resumo. */
  bestHints: number;
  bestMistakes: number;
  /** Soma dos tempos das vitórias — serve para a média. */
  totalTime: number;
};

export type Stats = Record<Difficulty, DifficultyStats>;

/** Resultado do dia; `done` quando o tabuleiro foi resolvido. */
export type DailyRecord = { done: boolean; time: number; hints: number };

export type Settings = {
  /** Marcar em vermelho o número que conflita com a linha, coluna ou bloco. */
  errors: boolean;
  /** Destacar linha, coluna, bloco e os iguais ao número escolhido. */
  highlight: boolean;
  /** Apagar as anotações dos vizinhos ao escrever um número. */
  autoNotes: boolean;
  /** Escolher o número primeiro e depois pintar as células. */
  numberFirst: boolean;
  /** Mostrar o cronômetro. */
  timer: boolean;
  theme: ThemeKey;
};

export type SavedState = {
  /** Partida em andamento (`null` quando não há). */
  game: Game | null;
  stats: Stats;
  history: HistoryEntry[];
  /** Por dia (`YYYY-MM-DD`). */
  daily: Record<string, DailyRecord>;
  settings: Settings;
};

export const DEFAULT_SETTINGS: Settings = {
  errors: true,
  highlight: true,
  autoNotes: true,
  numberFirst: false,
  timer: true,
  theme: DEFAULT_THEME,
};

export const EMPTY_DIFFICULTY_STATS: DifficultyStats = {
  games: 0,
  wins: 0,
  best: 0,
  bestHints: 0,
  bestMistakes: 0,
  totalTime: 0,
};

export function emptyStats(): Stats {
  return Object.fromEntries(DIFFICULTIES.map((d) => [d, { ...EMPTY_DIFFICULTY_STATS }])) as Stats;
}

export function emptyState(): SavedState {
  return {
    game: null,
    stats: emptyStats(),
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
    const stats = emptyStats();
    for (const d of DIFFICULTIES) {
      stats[d] = { ...EMPTY_DIFFICULTY_STATS, ...(saved?.stats?.[d] ?? {}) };
    }
    return {
      game: validGame(saved?.game) ? saved!.game! : null,
      stats,
      history: saved?.history ?? [],
      daily: saved?.daily ?? {},
      settings: { ...DEFAULT_SETTINGS, ...(saved?.settings ?? {}) },
    };
  } catch {
    return emptyState();
  }
}

/**
 * A partida salva veio de um JSON qualquer: sem as três grades de 81 posições
 * a tela quebraria no primeiro desenho, então aqui ela é descartada.
 */
function validGame(game: Game | null | undefined): boolean {
  if (!game) return false;
  const grids = [game.givens, game.values, game.notes, game.solution];
  return grids.every((grid) => Array.isArray(grid) && grid.length === 81) && !game.won;
}

export function saveState(state: SavedState): void {
  if (typeof localStorage === 'undefined') return;
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
  } catch {
    // Cota cheia ou modo privado — o jogo segue sem persistir.
  }
}

/** Total de vitórias somando as dificuldades — é o que libera os visuais. */
export function totalWins(stats: Stats): number {
  return DIFFICULTIES.reduce((n, d) => n + stats[d].wins, 0);
}

export function totalGames(stats: Stats): number {
  return DIFFICULTIES.reduce((n, d) => n + stats[d].games, 0);
}

/** Conta a partida começada (ganha ou não, é uma tentativa). */
export function addStart(stats: Stats, difficulty: Difficulty): Stats {
  const d = stats[difficulty];
  return { ...stats, [difficulty]: { ...d, games: d.games + 1 } };
}

/** Soma a vitória e guarda os melhores da dificuldade (tempo, dicas e erros). */
export function addWin(stats: Stats, g: Game): Stats {
  const d = stats[g.difficulty];
  const first = d.wins === 0;
  return {
    ...stats,
    [g.difficulty]: {
      games: Math.max(d.games, 1),
      wins: d.wins + 1,
      best: first ? g.elapsed : Math.min(d.best, g.elapsed),
      bestHints: first ? g.hints : Math.min(d.bestHints, g.hints),
      bestMistakes: first ? g.mistakes : Math.min(d.bestMistakes, g.mistakes),
      totalTime: d.totalTime + g.elapsed,
    },
  };
}

/** Prepende a vitória ao histórico (mais recente primeiro). */
export function recordMatch(history: HistoryEntry[], g: Game): HistoryEntry[] {
  const entry: HistoryEntry = {
    at: Date.now(),
    difficulty: g.difficulty,
    time: g.elapsed,
    hints: g.hints,
    mistakes: g.mistakes,
    daily: g.daily,
  };
  return [entry, ...history].slice(0, HISTORY_LIMIT);
}

/** Marca o dia como concluído e guarda o melhor tempo dele. */
export function recordDaily(
  daily: Record<string, DailyRecord>,
  day: string,
  time: number,
  hints: number,
): Record<string, DailyRecord> {
  const previous = daily[day];
  if (previous?.done && previous.time <= time) return daily;
  return { ...daily, [day]: { done: true, time, hints } };
}

/** Dias concluídos em sequência até hoje (ontem ainda conta: o dia não acabou). */
export function streakOf(daily: Record<string, DailyRecord>, today: Date): number {
  const day = new Date(today);
  let streak = 0;
  // Começa por hoje; se hoje ainda não foi resolvido, a sequência vem de ontem.
  for (let i = 0; i < 400; i++) {
    const key = `${day.getFullYear()}-${String(day.getMonth() + 1).padStart(2, '0')}-${String(
      day.getDate(),
    ).padStart(2, '0')}`;
    if (daily[key]?.done) streak++;
    else if (i > 0) break;
    day.setDate(day.getDate() - 1);
  }
  return streak;
}

/** Melhor tempo entre os dias resolvidos (0 quando nenhum foi). */
export function bestDaily(daily: Record<string, DailyRecord>): number {
  const times = Object.values(daily)
    .filter((d) => d.done && d.time > 0)
    .map((d) => d.time);
  return times.length ? Math.min(...times) : 0;
}
