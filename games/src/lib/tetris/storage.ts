/**
 * Persistência do Tetris em localStorage — só os ajustes.
 *
 * O jogo não guarda partida nem recorde: o que mora aqui é a escolha entre
 * arena em 3D e arena plana, e a marca de que a pergunta já foi feita — ela
 * aparece uma vez só, na primeira partida do aparelho.
 */

const STORAGE_KEY = 'games_tetris';

export type Settings = {
  /** Arena em 3D: câmera orbital e arraste para girar o poço. */
  depth: boolean;
};

export type SavedState = {
  settings: Settings;
  /** A pergunta do 3D já foi respondida neste aparelho. */
  asked: boolean;
};

/** 3D **desligado** por padrão: quem quiser girar o poço liga na pergunta ou no ajuste. */
export const DEFAULT_SETTINGS: Settings = { depth: false };

export function emptyState(): SavedState {
  return { settings: { ...DEFAULT_SETTINGS }, asked: false };
}

export function loadState(): SavedState {
  if (typeof localStorage === 'undefined') return emptyState();
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return emptyState();
    const saved = JSON.parse(raw) as Partial<SavedState> | null;
    // Comparação com `true` em vez de espalhar o que veio salvo: lixo no
    // localStorage (string, número, null) cai no padrão em vez de virar modo.
    return {
      settings: { depth: saved?.settings?.depth === true },
      asked: saved?.asked === true,
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
