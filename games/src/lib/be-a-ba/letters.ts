/**
 * Bê-á-Bá — regras puras da trilha das Letras. Sem DOM, voz ou storage.
 *
 * Três modos na fase 1 (PRD 058): `first` (com que letra começa), `build`
 * (montar com letras embaralhadas) e `type` (escrever no teclado ABC).
 * Acento e cedilha nunca contam como erro: `A` vale para `Ã`, `C` para `Ç`.
 */

import { plain } from './words';

export type LetterMode = 'first' | 'build' | 'type';

/** Palavra em jogo: do banco ou da família (com foto opcional). */
export type PlayWord = {
  id: string;
  text: string;
  /** Código do desenho; ausente nas palavras da família sem foto. */
  img?: string;
  /** Palavra cadastrada pelos pais. */
  family?: boolean;
  /** Chave da foto no IndexedDB. */
  photo?: string;
};

export type LetterChallenge = {
  kind: 'letter';
  mode: LetterMode;
  word: PlayWord;
  /** `first`: 3 letras (a certa entre elas). */
  options: string[];
  /** `build`: letras da palavra embaralhadas + sobras. */
  tray: string[];
};

export const ALPHABET = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ'.split('');
/** Teclado na tela: ordem ABC (a criança não conhece QWERTY) + Ç. */
export const KEYBOARD = [...ALPHABET, 'Ç'];
export const MAX_WORD_LENGTH = 12;

/** Uma letra digitada vale para a esperada? (`a` = `Á` = `Ã`; `c` = `Ç`) */
export const sameLetter = (expected: string, typed: string): boolean =>
  plain(expected).toUpperCase() === plain(typed).toUpperCase();

/** Letras de uma palavra (cada uma com seu acento). */
export const lettersOf = (text: string): string[] => [...text.normalize('NFC')];

/**
 * Normaliza uma palavra cadastrada pelos pais: CAIXA ALTA, só letras (com
 * acento), sem espaço, até `MAX_WORD_LENGTH`. Devolve `''` se nada sobrar.
 */
export function cleanFamilyWord(raw: string): string {
  return [...raw.normalize('NFC').toLocaleUpperCase('pt-BR')]
    .filter((ch) => /\p{L}/u.test(ch))
    .slice(0, MAX_WORD_LENGTH)
    .join('');
}

function shuffle<T>(xs: T[], rnd: () => number): T[] {
  const a = xs.slice();
  for (let i = a.length - 1; i > 0; i--) {
    const j = Math.floor(rnd() * (i + 1));
    [a[i], a[j]] = [a[j], a[i]];
  }
  return a;
}

/** Letras do alfabeto que não aparecem na palavra — servem de distração. */
function distractors(text: string, n: number, rnd: () => number): string[] {
  const used = new Set(lettersOf(plain(text)));
  return shuffle(ALPHABET.filter((l) => !used.has(l)), rnd).slice(0, n);
}

/** `first`: a primeira letra (sem acento) e duas outras, embaralhadas. */
export function firstLetterOptions(text: string, rnd: () => number): string[] {
  const first = plain(lettersOf(text)[0]);
  return shuffle([first, ...distractors(text, 2, rnd)], rnd);
}

/**
 * `build`: as letras da palavra (sem acento — o acento aparece quando a letra
 * cai no lugar) + `extras` letras sobrando. Nunca devolve a palavra já em ordem.
 */
export function buildTray(text: string, extras: number, rnd: () => number): string[] {
  const letters = lettersOf(plain(text));
  const pool = [...letters, ...distractors(text, extras, rnd)];
  for (let attempt = 0; attempt < 10; attempt++) {
    const tray = shuffle(pool, rnd);
    if (tray.slice(0, letters.length).join('') !== letters.join('') || new Set(letters).size === 1) return tray;
  }
  return pool.slice().reverse();
}

export function makeLetterChallenge(mode: LetterMode, word: PlayWord, extras: number, rnd: () => number): LetterChallenge {
  return {
    kind: 'letter',
    mode,
    word,
    options: mode === 'first' ? firstLetterOptions(word.text, rnd) : [],
    tray: mode === 'build' ? buildTray(word.text, extras, rnd) : [],
  };
}
