/**
 * Bê-á-Bá — trilha **Inglês**: introdução ao inglês só de ouvido.
 *
 * A criança ouve a palavra em inglês (voz en-US) e toca na figura, na cor ou
 * no número — os mesmos toques do modo Pequenos. **Não há escrita em inglês**:
 * quem está se alfabetizando em português não deve soletrar "apple" ao mesmo
 * tempo. A palavra escrita só aparece no elogio, como exposição.
 *
 * Depois de cada acerto vem a ponte para o português ("Apple! — maçã"), pelas
 * chaves i18n `be_a_ba.en_word.<slug>`. As frases em inglês (perguntas e
 * elogios) são o conteúdo da aula, por isso moram aqui e não no i18n: valem
 * igual em qualquer idioma da interface.
 *
 * Puro: sem DOM, voz ou storage.
 */

import type { LittleChallenge } from './little';
import { MAX_BOX, pickWeighted, type Boxes, type TrailProgress } from './progress';
import { ITEMS } from './numbers';

export const EN_LANG = 'en-US';

export type EnCategory = 'animals' | 'fruits' | 'foods';

export type EnWord = {
  id: string;
  /** A palavra em inglês, como é falada e mostrada no elogio. */
  text: string;
  img: string;
  level: 1;
  category: EnCategory;
  /** "Where **are** the grapes?" */
  plural?: boolean;
};

const e = (text: string, img: string, category: EnCategory, plural = false): EnWord => ({
  id: 'en-' + text.replace(/ /g, '-'),
  text,
  img,
  level: 1,
  category,
  plural,
});

/** Slug da palavra para a chave i18n da ponte (`be_a_ba.en_word.<slug>`). */
export const enSlug = (w: Pick<EnWord, 'id'>): string => w.id.slice(3).replace(/-/g, '_');

export const EN_WORDS: EnWord[] = [
  // Bichos — os primeiros que a criança conhece em casa, na fazenda e no livro
  e('cat', '1f408', 'animals'),
  e('dog', '1f415', 'animals'),
  e('fish', '1f41f', 'animals'),
  e('bird', '1f426', 'animals'),
  e('cow', '1f404', 'animals'),
  e('duck', '1f986', 'animals'),
  e('pig', '1f416', 'animals'),
  e('horse', '1f40e', 'animals'),
  e('sheep', '1f411', 'animals'),
  e('rabbit', '1f407', 'animals'),
  e('frog', '1f438', 'animals'),
  e('lion', '1f981', 'animals'),
  e('elephant', '1f418', 'animals'),
  e('monkey', '1f412', 'animals'),
  e('bear', '1f43b', 'animals'),
  // Frutas
  e('apple', '1f34e', 'fruits'),
  e('banana', '1f34c', 'fruits'),
  e('grapes', '1f347', 'fruits', true),
  e('orange', '1f34a', 'fruits'),
  e('strawberry', '1f353', 'fruits'),
  e('watermelon', '1f349', 'fruits'),
  e('pear', '1f350', 'fruits'),
  e('lemon', '1f34b', 'fruits'),
  e('pineapple', '1f34d', 'fruits'),
  e('cherries', '1f352', 'fruits', true),
  // Comidas simples do dia a dia
  e('bread', '1f35e', 'foods'),
  e('egg', '1f95a', 'foods'),
  e('milk', '1f95b', 'foods'),
  e('cheese', '1f9c0', 'foods'),
  e('cake', '1f382', 'foods'),
  e('cookie', '1f36a', 'foods'),
  e('pizza', '1f355', 'foods'),
  e('ice cream', '1f366', 'foods'),
  e('juice', '1f9c3', 'foods'),
  e('rice', '1f35a', 'foods'),
  e('carrot', '1f955', 'foods'),
  e('corn', '1f33d', 'foods'),
];

export type EnColor = { id: string; hex: string };

/** Na ordem em que entram: as 4 primárias, depois as secundárias, depois as neutras. */
export const EN_COLORS: EnColor[] = [
  { id: 'red', hex: '#e53935' },
  { id: 'blue', hex: '#1e88e5' },
  { id: 'yellow', hex: '#fdd835' },
  { id: 'green', hex: '#43a047' },
  { id: 'orange', hex: '#fb8c00' },
  { id: 'purple', hex: '#8e24aa' },
  { id: 'pink', hex: '#f06292' },
  { id: 'brown', hex: '#8d6e63' },
  { id: 'black', hex: '#212121' },
  { id: 'white', hex: '#ffffff' },
];

export const colorHex = (id: string): string => EN_COLORS.find((c) => c.id === id)?.hex ?? '#ccc';

/** Números em inglês, para a fala da contagem não depender do motor de voz ler dígitos. */
export const EN_NUMBERS = ['zero', 'one', 'two', 'three', 'four', 'five', 'six', 'seven', 'eight', 'nine', 'ten'];

export type EnKind = 'hear' | 'pick_color' | 'find_number' | 'tap';

export type EnStage = {
  kinds: EnKind[];
  categories: EnCategory[];
  colors: number;
  numberMax: number;
  options: number;
};

export const EN_STAGES: EnStage[] = [
  { kinds: ['hear', 'pick_color', 'find_number'], categories: ['animals'], colors: 4, numberMax: 3, options: 2 },
  { kinds: ['hear', 'pick_color', 'hear', 'find_number'], categories: ['animals', 'fruits'], colors: 7, numberMax: 5, options: 3 },
  {
    kinds: ['hear', 'pick_color', 'find_number', 'hear', 'tap'],
    categories: ['animals', 'fruits', 'foods'],
    colors: 10,
    numberMax: 10,
    options: 3,
  },
];

/** Contar tocando vai só até 6: mais que isso não cabe bem na tela do celular. */
const TAP_MAX = 6;

export const EN_PRAISE = ['Great job!', 'Very good!', 'Yes!', 'Well done!'];

/** A pergunta falada (e mostrada) de cada desafio, sempre em inglês. */
export function enPrompt(c: LittleChallenge): string {
  switch (c.kind) {
    case 'hear': {
      const w = c.word as EnWord;
      return `Where ${w.plural ? 'are' : 'is'} the ${w.text}?`;
    }
    case 'pick_color':
      return `Touch ${c.color}!`;
    case 'find_number':
      return `Find number ${EN_NUMBERS[c.n]}!`;
    case 'tap':
      return "Let's count!";
    default:
      return '';
  }
}

/** O que o elogio repete em inglês depois de "Great job!". */
export function enAnswerWord(c: LittleChallenge): string {
  switch (c.kind) {
    case 'hear':
      return c.word.text;
    case 'pick_color':
      return c.color;
    case 'find_number':
    case 'tap':
      return EN_NUMBERS[c.n];
    default:
      return '';
  }
}

const int = (rnd: () => number, min: number, max: number) => min + Math.floor(rnd() * (max - min + 1));

function shuffle<T>(xs: T[], rnd: () => number): T[] {
  const a = xs.slice();
  for (let i = a.length - 1; i > 0; i--) {
    const j = Math.floor(rnd() * (i + 1));
    [a[i], a[j]] = [a[j], a[i]];
  }
  return a;
}

const weightOf = (boxes: Boxes) => (id: string) => {
  const b = boxes[id];
  return b ? MAX_BOX + 1 - b.box : MAX_BOX;
};

export const wordsOfStage = (stage: EnStage): EnWord[] => EN_WORDS.filter((w) => stage.categories.includes(w.category));
export const colorsOfStage = (stage: EnStage): EnColor[] => EN_COLORS.slice(0, stage.colors);

/**
 * Rodada da trilha Inglês com `size` desafios (3 no modo Pequenos, 5 no de
 * 5–7 anos). As figuras erradas saem da **mesma categoria** da certa — é
 * diferenciar "apple" de "pear" que ensina, não de "dog".
 */
export function englishRound(p: TrailProgress, size: number, rnd: () => number): LittleChallenge[] {
  const stage = EN_STAGES[Math.min(p.stage, EN_STAGES.length - 1)];
  const w = weightOf(p.boxes);
  const pool = wordsOfStage(stage);
  const words = pickWeighted(pool, (x) => w(`hear:${x.id}`), size, rnd);
  const colors = pickWeighted(colorsOfStage(stage), (c) => w(`color:${c.id}`), size, rnd);
  const out: LittleChallenge[] = [];
  const seenNumbers = new Set<string>();
  const offset = Math.floor(rnd() * stage.kinds.length);

  for (let i = 0; out.length < size && i < size * 4; i++) {
    const kind = stage.kinds[(i + offset) % stage.kinds.length];
    if (kind === 'hear') {
      const word = words[out.filter((c) => c.kind === 'hear').length % words.length];
      if (out.some((c) => c.kind === 'hear' && c.word.id === word.id)) continue;
      const same = shuffle(pool.filter((x) => x.category === word.category && x.id !== word.id), rnd);
      const others = (same.length >= stage.options - 1 ? same : shuffle(pool.filter((x) => x.id !== word.id), rnd)).slice(
        0,
        stage.options - 1
      );
      out.push({ kind, word, options: shuffle([word, ...others], rnd) });
    } else if (kind === 'pick_color') {
      const color = colors[out.filter((c) => c.kind === 'pick_color').length % colors.length];
      if (out.some((c) => c.kind === 'pick_color' && c.color === color.id)) continue;
      const others = shuffle(colorsOfStage(stage).filter((c) => c.id !== color.id), rnd).slice(0, stage.options - 1);
      out.push({ kind, color: color.id, options: shuffle([color.id, ...others.map((c) => c.id)], rnd) });
    } else if (kind === 'find_number') {
      const n = int(rnd, 1, stage.numberMax);
      if (seenNumbers.has(`f${n}`)) continue;
      seenNumbers.add(`f${n}`);
      const others = shuffle(
        Array.from({ length: stage.numberMax }, (_, k) => k + 1).filter((x) => x !== n),
        rnd
      ).slice(0, stage.options - 1);
      out.push({ kind, n, options: shuffle([n, ...others], rnd) });
    } else {
      const n = int(rnd, 2, Math.min(TAP_MAX, stage.numberMax));
      if (seenNumbers.has(`t${n}`)) continue;
      seenNumbers.add(`t${n}`);
      out.push({ kind, item: ITEMS[Math.floor(rnd() * ITEMS.length)], n });
    }
  }
  return out;
}
