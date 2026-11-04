/**
 * Visuais do tabuleiro.
 *
 * Nada de imagem: cada visual é um punhado de cores, então o jogo pesa o mesmo
 * com seis temas ou com um. Liberam por **partidas concluídas** (`wins` somadas
 * de todas as dificuldades), que é o que a barra da aba Visuais mostra.
 *
 * O padrão é o papel de jornal; `ardosia` e `noturno` são as versões escuras,
 * para quem joga no escuro.
 */

export type ThemeKey = 'jornal' | 'linho' | 'cafe' | 'menta' | 'ardosia' | 'noturno';

export type Theme = {
  key: ThemeKey;
  /** Chave i18n do nome (`sudoku.theme.<key>`). */
  nameKey: string;
  /** Partidas concluídas para liberar. */
  need: number;
  /** Fundo da célula e o do bloco alternado (o xadrez 3×3 do tabuleiro). */
  paper: string;
  paperAlt: string;
  /** Linha fina entre células e linha grossa entre blocos. */
  line: string;
  edge: string;
  /** Tinta do número impresso, do número do jogador e das anotações. */
  given: string;
  pen: string;
  note: string;
  /** Célula escolhida, casas em destaque, mesmo dígito e conflito. */
  sel: string;
  peer: string;
  same: string;
  bad: string;
};

export const DEFAULT_THEME: ThemeKey = 'jornal';

export const THEMES: Theme[] = [
  {
    key: 'jornal',
    nameKey: 'sudoku.theme.jornal',
    need: 0,
    paper: '#f6ecd6',
    paperAlt: '#f0e3c5',
    line: '#d6c4a0',
    edge: '#8d7b5c',
    given: '#2a1f17',
    pen: '#c2410c',
    note: '#8a7563',
    sel: '#ffe3a1',
    peer: '#ebdcb8',
    same: '#f7d78a',
    bad: '#d7342a',
  },
  {
    key: 'linho',
    nameKey: 'sudoku.theme.linho',
    need: 5,
    paper: '#f3f1e7',
    paperAlt: '#eae8da',
    line: '#cdcbbb',
    edge: '#7b7a6b',
    given: '#22251f',
    pen: '#1d5fa8',
    note: '#86857a',
    sel: '#d6e7f7',
    peer: '#e4e6dd',
    same: '#c8dff3',
    bad: '#cf3a2b',
  },
  {
    key: 'cafe',
    nameKey: 'sudoku.theme.cafe',
    need: 15,
    paper: '#ecdcc0',
    paperAlt: '#e3d0ad',
    line: '#c9b088',
    edge: '#83684a',
    given: '#30220f',
    pen: '#8a4b16',
    note: '#8b765a',
    sel: '#f1d79f',
    peer: '#e2d0a9',
    same: '#eac785',
    bad: '#b93a2a',
  },
  {
    key: 'menta',
    nameKey: 'sudoku.theme.menta',
    need: 30,
    paper: '#e9f2e4',
    paperAlt: '#dfebd8',
    line: '#c0d3b8',
    edge: '#6f8468',
    given: '#1b2a19',
    pen: '#1f7a4d',
    note: '#7d8d78',
    sel: '#c9eedb',
    peer: '#d9ead2',
    same: '#b8e6ca',
    bad: '#bb3a2a',
  },
  {
    key: 'ardosia',
    nameKey: 'sudoku.theme.ardosia',
    need: 60,
    paper: '#2b3140',
    paperAlt: '#252a37',
    line: '#3d4657',
    edge: '#8d9ab4',
    given: '#e8edf7',
    pen: '#ffc93c',
    note: '#8f9cb5',
    sel: '#3f4c66',
    peer: '#323a4b',
    same: '#4a5a7a',
    bad: '#ff6b5e',
  },
  {
    key: 'noturno',
    nameKey: 'sudoku.theme.noturno',
    need: 100,
    paper: '#101a2b',
    paperAlt: '#0c1522',
    line: '#24304a',
    edge: '#5f7ba8',
    given: '#dce6f7',
    pen: '#ffc93c',
    note: '#7085a8',
    sel: '#20304e',
    peer: '#17233a',
    same: '#2b3f66',
    bad: '#ff6b5e',
  },
];

/** O tema salvo, ou o padrão se a chave não existir mais. */
export function themeOf(key: string | null | undefined): Theme {
  return THEMES.find((t) => t.key === key) ?? THEMES[0];
}

export function isUnlocked(key: ThemeKey, wins: number): boolean {
  return wins >= themeOf(key).need;
}

export function unlockedThemes(wins: number): ThemeKey[] {
  return THEMES.filter((t) => wins >= t.need).map((t) => t.key);
}

/** O próximo tema a liberar (ou `null` quando já liberou todos). */
export function nextLock(wins: number): Theme | null {
  return THEMES.find((t) => wins < t.need) ?? null;
}
