/**
 * Persistência do Bê-á-Bá em localStorage (`games_be_a_ba`) — criança,
 * palavras da família, progresso das trilhas, figurinhas e ajustes. As fotos
 * das palavras da família ficam no IndexedDB (`photos.ts`). Nada sai do aparelho.
 *
 * A rodada em andamento **não** é salva: são 5 desafios curtos.
 */

import { emptyTrail, type TrailProgress } from './progress';

const STORAGE_KEY = 'games_be_a_ba';

export type FamilyWord = {
  id: string;
  /** CAIXA ALTA, só letras (`cleanFamilyWord`). */
  text: string;
  /** Tem foto no IndexedDB (chave = `id`). */
  photo: boolean;
  /** É o nome da criança — dá a figurinha especial na 1ª vez. */
  isChildName?: boolean;
};

export type Settings = {
  voice: boolean;
  /** Velocidade da fala (0.6 – 1.1). */
  rate: number;
};

/** `little` = modo Pequenos (3–4 anos); `kids` = 5–7 anos. `null` = ainda não escolhido. */
export type AgeMode = 'little' | 'kids';

export type SavedState = {
  ageMode: AgeMode | null;
  childName: string;
  family: FamilyWord[];
  letters: TrailProgress;
  numbers: TrailProgress;
  /** Progresso do modo Pequenos — trilhas e estágios próprios. */
  little: { letters: TrailProgress; numbers: TrailProgress };
  /** Figurinhas na ordem em que foram ganhas. */
  stickers: string[];
  stars: number;
  rounds: number;
  /** Segundos jogados por dia (`YYYY-MM-DD`). */
  playtime: Record<string, number>;
  settings: Settings;
};

export const DEFAULT_SETTINGS: Settings = { voice: true, rate: 0.85 };

export function emptyState(): SavedState {
  return {
    ageMode: null,
    childName: '',
    family: [],
    letters: emptyTrail(),
    numbers: emptyTrail(),
    little: { letters: emptyTrail(), numbers: emptyTrail() },
    stickers: [],
    stars: 0,
    rounds: 0,
    playtime: {},
    settings: { ...DEFAULT_SETTINGS },
  };
}

function trailOf(x: unknown): TrailProgress {
  const t = (x ?? {}) as Partial<TrailProgress>;
  return {
    stage: Math.max(0, Number(t.stage) || 0),
    locked: !!t.locked,
    goodRounds: Math.max(0, Number(t.goodRounds) || 0),
    boxes: t.boxes && typeof t.boxes === 'object' ? t.boxes : {},
  };
}

export function loadState(): SavedState {
  if (typeof localStorage === 'undefined') return emptyState();
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return emptyState();
    const s = JSON.parse(raw) as Partial<SavedState> | null;
    if (!s || typeof s !== 'object') return emptyState();
    const little = (s.little ?? {}) as Partial<SavedState['little']>;
    return {
      // Quem já jogava antes do modo Pequenos existir segue no modo de 5–7 anos.
      ageMode: s.ageMode === 'little' || s.ageMode === 'kids' ? s.ageMode : 'kids',
      childName: typeof s.childName === 'string' ? s.childName : '',
      family: Array.isArray(s.family) ? s.family.filter((f) => f && typeof f.text === 'string' && f.text) : [],
      letters: trailOf(s.letters),
      numbers: trailOf(s.numbers),
      little: { letters: trailOf(little.letters), numbers: trailOf(little.numbers) },
      stickers: Array.isArray(s.stickers) ? s.stickers.filter((x) => typeof x === 'string') : [],
      stars: Math.max(0, Number(s.stars) || 0),
      rounds: Math.max(0, Number(s.rounds) || 0),
      playtime: s.playtime && typeof s.playtime === 'object' ? s.playtime : {},
      settings: { ...DEFAULT_SETTINGS, ...(s.settings ?? {}) },
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
