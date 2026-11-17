/**
 * Bê-á-Bá — modo **Pequenos** (3 e 4 anos). Puro: sem DOM, voz ou storage.
 *
 * Criança dessa idade ainda não lê nem escreve, então o modo troca escrever
 * por escutar, reconhecer e tocar:
 *
 * - `hear`        — "Cadê a uva?": ouve a palavra e toca na figura certa
 * - `pick_letter` — "Ache a letra igual": letra grande no alto, 2–3 opções
 * - `tap`         — "Vamos contar!": toca em cada objeto e ele fala o número
 * - `find_number` — "Ache o número 3": numeral com bolinhas embaixo
 * - `where`       — "Onde tem 2?": dois grupos, toca no que tem a quantidade
 * - `pick_color`  — só na trilha Inglês (`english.ts`): "Touch red!"
 *
 * Rodadas de 3 desafios, começando com 2 opções.
 */

import { WORDS, plain, type Word } from './words';
import { ITEMS } from './numbers';
import { pickWeighted, MAX_BOX, type Boxes, type TrailProgress } from './progress';

export const LITTLE_ROUND_SIZE = 3;

export type LittleKind = 'hear' | 'pick_letter' | 'pick_color' | 'tap' | 'find_number' | 'where';

export type LittleChallenge =
  /** `article` só no português ("Cadê **a** uva?"); a trilha Inglês não usa. */
  | { kind: 'hear'; word: Word; article?: 'a' | 'o'; options: Word[] }
  | { kind: 'pick_letter'; letter: string; options: string[] }
  /** Só na trilha Inglês ("Touch red!"): `color` e `options` são ids de `EN_COLORS`. */
  | { kind: 'pick_color'; color: string; options: string[] }
  | { kind: 'tap'; item: string; n: number }
  | { kind: 'find_number'; n: number; options: number[] }
  | { kind: 'where'; item: string; n: number; groups: number[] };

/**
 * Palavras concretas que criança de 3 anos reconhece pela figura, com o
 * artigo para a pergunta falada ("Cadê **a** uva?"). Fica fora daqui o que
 * é abstrato ou confunde no desenho (ímã, mapa, cometa, panela…).
 */
export const HEAR_ARTICLES: Record<string, 'a' | 'o'> = {
  uva: 'a', ovo: 'o', sol: 'o', pao: 'o', mao: 'a', pe: 'o', lua: 'a', boi: 'o',
  bola: 'a', casa: 'a', gato: 'o', pato: 'o', sapo: 'o', vaca: 'a', bolo: 'o', pipa: 'a',
  suco: 'o', moto: 'a', bebe: 'o', rato: 'o', cama: 'a', pera: 'a', banana: 'a',
  macaco: 'o', cavalo: 'o', tomate: 'o', baleia: 'a', girafa: 'a', carro: 'o', flor: 'a',
  peixe: 'o', elefante: 'o', cachorro: 'o', coelho: 'o', morango: 'o', arvore: 'a',
  aviao: 'o', trem: 'o', borboleta: 'a', tartaruga: 'a', estrela: 'a', galinha: 'a',
};

export const HEAR_WORDS: Word[] = WORDS.filter((w) => w.id in HEAR_ARTICLES);

export const VOWELS = ['A', 'E', 'I', 'O', 'U'];
/** Consoantes que entram depois das vogais: as de som mais claro e frequentes no começo das palavras. */
export const FIRST_CONSONANTS = ['B', 'M', 'P', 'S', 'L', 'T'];

export type LittleLetterStage = { kinds: ('hear' | 'pick_letter')[]; options: number; consonants: boolean };
export type LittleNumberStage = { kinds: ('tap' | 'find_number' | 'where')[]; max: number; options: number };

export const LITTLE_LETTER_STAGES: LittleLetterStage[] = [
  { kinds: ['hear'], options: 2, consonants: false },
  { kinds: ['hear', 'pick_letter'], options: 2, consonants: false },
  { kinds: ['pick_letter', 'hear'], options: 3, consonants: true },
];

export const LITTLE_NUMBER_STAGES: LittleNumberStage[] = [
  { kinds: ['tap'], max: 3, options: 2 },
  { kinds: ['tap', 'find_number'], max: 3, options: 2 },
  { kinds: ['find_number', 'where', 'tap'], max: 5, options: 3 },
];

const int = (rnd: () => number, min: number, max: number) => min + Math.floor(rnd() * (max - min + 1));

function shuffle<T>(xs: T[], rnd: () => number): T[] {
  const a = xs.slice();
  for (let i = a.length - 1; i > 0; i--) {
    const j = Math.floor(rnd() * (i + 1));
    [a[i], a[j]] = [a[j], a[i]];
  }
  return a;
}

/** `n` números distintos em `[1, max]`, incluindo `answer`, embaralhados. */
function numberOptions(answer: number, max: number, n: number, rnd: () => number): number[] {
  const others = shuffle(
    Array.from({ length: max }, (_, i) => i + 1).filter((x) => x !== answer),
    rnd
  ).slice(0, n - 1);
  return shuffle([answer, ...others], rnd);
}

/** Letras do estágio: vogais; depois as primeiras consoantes e a inicial do nome da criança. */
export function letterPool(stage: LittleLetterStage, childName = ''): string[] {
  const pool = [...VOWELS];
  if (stage.consonants) {
    pool.push(...FIRST_CONSONANTS);
    const initial = plain(childName.trim().charAt(0)).toUpperCase();
    if (/^[A-Z]$/.test(initial) && !pool.includes(initial)) pool.push(initial);
  }
  return pool;
}

/** Chave do Leitner no modo Pequenos (palavra, letra ou tipo de conta). */
export function littleItemId(c: LittleChallenge): string {
  switch (c.kind) {
    case 'hear':
      return `hear:${c.word.id}`;
    case 'pick_letter':
      return `letter:${c.letter}`;
    case 'pick_color':
      return `color:${c.color}`;
    default:
      return c.kind;
  }
}

/** A resposta: id da palavra, a letra, o número ou o índice do grupo (`where`). */
export function littleAnswer(c: LittleChallenge): string | number {
  switch (c.kind) {
    case 'hear':
      return c.word.id;
    case 'pick_letter':
      return c.letter;
    case 'pick_color':
      return c.color;
    case 'tap':
    case 'find_number':
      return c.n;
    case 'where':
      return c.groups.indexOf(c.n);
  }
}

const weightOf = (boxes: Boxes) => (id: string) => {
  const b = boxes[id];
  return b ? MAX_BOX + 1 - b.box : MAX_BOX;
};

/** Rodada da trilha de Letras no modo Pequenos. */
export function littleLetterRound(p: TrailProgress, childName: string, rnd: () => number): LittleChallenge[] {
  const stage = LITTLE_LETTER_STAGES[Math.min(p.stage, LITTLE_LETTER_STAGES.length - 1)];
  const w = weightOf(p.boxes);
  const words = pickWeighted(HEAR_WORDS, (x) => w(`hear:${x.id}`), LITTLE_ROUND_SIZE, rnd);
  const pool = letterPool(stage, childName);
  const letters = pickWeighted(pool, (l) => w(`letter:${l}`), LITTLE_ROUND_SIZE, rnd);

  return Array.from({ length: LITTLE_ROUND_SIZE }, (_, i) => {
    const kind = stage.kinds[(i + Math.floor(rnd() * stage.kinds.length)) % stage.kinds.length];
    if (kind === 'hear') {
      const word = words[i];
      const others = shuffle(HEAR_WORDS.filter((x) => x.id !== word.id), rnd).slice(0, stage.options - 1);
      return { kind, word, article: HEAR_ARTICLES[word.id], options: shuffle([word, ...others], rnd) };
    }
    const letter = letters[i % letters.length];
    const others = shuffle(pool.filter((l) => l !== letter), rnd).slice(0, stage.options - 1);
    return { kind, letter, options: shuffle([letter, ...others], rnd) };
  });
}

/** Rodada da trilha de Números no modo Pequenos (sem repetir a mesma conta). */
export function littleNumberRound(p: TrailProgress, rnd: () => number): LittleChallenge[] {
  const stage = LITTLE_NUMBER_STAGES[Math.min(p.stage, LITTLE_NUMBER_STAGES.length - 1)];
  const out: LittleChallenge[] = [];
  const seen = new Set<string>();
  for (let i = 0, guard = 0; out.length < LITTLE_ROUND_SIZE && guard < 100; guard++) {
    const kind = stage.kinds[i % stage.kinds.length];
    const n = int(rnd, 1, stage.max);
    if (seen.has(`${kind}:${n}`)) continue;
    seen.add(`${kind}:${n}`);
    const item = ITEMS[Math.floor(rnd() * ITEMS.length)];
    if (kind === 'tap') out.push({ kind, item, n });
    else if (kind === 'find_number') out.push({ kind, n, options: numberOptions(n, stage.max, stage.options, rnd) });
    else {
      let other = int(rnd, 1, stage.max - 1);
      if (other >= n) other++;
      out.push({ kind, item, n, groups: shuffle([n, other], rnd) });
    }
    i++;
  }
  return out;
}
