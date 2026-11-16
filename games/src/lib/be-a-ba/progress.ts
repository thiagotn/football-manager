/**
 * Bê-á-Bá — progressão: estágios de cada trilha, caixas de domínio (Leitner)
 * e montagem da rodada de 5 desafios. Puro: o sorteio recebe `rnd`.
 */

import { WORDS, type WordLevel } from './words';
import { makeLetterChallenge, type LetterChallenge, type LetterMode, type PlayWord } from './letters';
import { keyOf, makeNumberChallenge, type NumberChallenge, type NumberKind } from './numbers';

export const ROUND_SIZE = 5;
export const MAX_BOX = 5;
/** Rodadas boas seguidas (≥ 4 de 5 sem erro nem dica) para subir de estágio. */
export const GOOD_ROUNDS_TO_ADVANCE = 2;
export const GOOD_ROUND_CLEAN = 4;

export type Trail = 'letters' | 'numbers';
export type Challenge = LetterChallenge | NumberChallenge;
export type Outcome = 'clean' | 'hint' | 'miss';

export type LetterStage = { modes: LetterMode[]; levels: WordLevel[]; extras: number };
export type NumberStage = { kinds: NumberKind[]; max: number };

/** Do mais fácil ao mais difícil — o teclado (L4) só aparece no 3º estágio. */
export const LETTER_STAGES: LetterStage[] = [
  { modes: ['first'], levels: [1, 2], extras: 0 },
  { modes: ['build'], levels: [1, 2], extras: 0 },
  { modes: ['build', 'type'], levels: [1, 2, 3], extras: 1 },
  { modes: ['type'], levels: [2, 3], extras: 1 },
  { modes: ['type', 'build'], levels: [3, 4], extras: 2 },
];

export const NUMBER_STAGES: NumberStage[] = [
  { kinds: ['count'], max: 5 },
  { kinds: ['count', 'compare'], max: 10 },
  { kinds: ['add', 'count', 'compare'], max: 5 },
  { kinds: ['add', 'sub'], max: 10 },
];

/** Caixa de domínio de um item (palavra ou tipo de conta). */
export type BoxEntry = { box: number; seen: number; misses: number };
export type Boxes = Record<string, BoxEntry>;

export type TrailProgress = {
  stage: number;
  /** `true` = o adulto fixou o estágio; não sobe sozinho. */
  locked: boolean;
  goodRounds: number;
  boxes: Boxes;
};

export const emptyTrail = (): TrailProgress => ({ stage: 0, locked: false, goodRounds: 0, boxes: {} });

/** Resultado de um desafio pelos erros e pelo uso de dica. */
export const outcomeOf = (mistakes: number, hinted: boolean): Outcome =>
  mistakes >= 3 ? 'miss' : mistakes === 0 && !hinted ? 'clean' : 'hint';

/** Leitner: sem erro sobe uma caixa, com dica fica, errando muito desce. */
export function recordOutcome(boxes: Boxes, id: string, outcome: Outcome): Boxes {
  const cur = boxes[id] ?? { box: 1, seen: 0, misses: 0 };
  const box =
    outcome === 'clean' ? Math.min(MAX_BOX, cur.box + 1) : outcome === 'miss' ? Math.max(1, cur.box - 1) : cur.box;
  return { ...boxes, [id]: { box, seen: cur.seen + 1, misses: cur.misses + (outcome === 'miss' ? 1 : 0) } };
}

/**
 * Fim de rodada: conta as rodadas boas seguidas e sobe de estágio depois de
 * `GOOD_ROUNDS_TO_ADVANCE`. Rodada boa = ≥ 80% dos desafios sem erro nem dica
 * (4 de 5; no modo Pequenos, 3 de 3). Nunca desce sozinho; estágio fixado
 * pelo adulto não muda.
 */
export function afterRound(
  p: TrailProgress,
  stageCount: number,
  cleanCount: number,
  roundSize = ROUND_SIZE
): { progress: TrailProgress; leveledUp: boolean } {
  if (p.locked) return { progress: p, leveledUp: false };
  const good = cleanCount >= Math.ceil(roundSize * (GOOD_ROUND_CLEAN / ROUND_SIZE));
  const goodRounds = good ? p.goodRounds + 1 : cleanCount <= Math.floor(roundSize / 2) ? 0 : p.goodRounds;
  if (goodRounds >= GOOD_ROUNDS_TO_ADVANCE && p.stage < stageCount - 1) {
    return { progress: { ...p, stage: p.stage + 1, goodRounds: 0 }, leveledUp: true };
  }
  return { progress: { ...p, goodRounds }, leveledUp: false };
}

/**
 * Sorteio ponderado sem repetição: caixa baixa e item nunca visto pesam mais
 * (revisão), mas caixa alta ainda aparece às vezes (confiança).
 */
export function pickWeighted<T>(items: T[], weight: (x: T) => number, n: number, rnd: () => number): T[] {
  const pool = items.slice();
  const out: T[] = [];
  while (out.length < n && pool.length) {
    const total = pool.reduce((s, x) => s + weight(x), 0);
    let r = rnd() * total;
    let i = 0;
    for (; i < pool.length - 1; i++) {
      r -= weight(pool[i]);
      if (r < 0) break;
    }
    out.push(pool.splice(i, 1)[0]);
  }
  return out;
}

const weightOf = (boxes: Boxes) => (id: string) => {
  const b = boxes[id];
  return b ? MAX_BOX + 1 - b.box : MAX_BOX;
};

/**
 * Rodada da trilha das Letras: 5 palavras do estágio, com **uma palavra da
 * família** quando houver (é a que a criança mais quer escrever).
 */
export function letterRound(p: TrailProgress, family: PlayWord[], rnd: () => number): LetterChallenge[] {
  const stage = LETTER_STAGES[Math.min(p.stage, LETTER_STAGES.length - 1)];
  const w = weightOf(p.boxes);
  const fromFamily = family.length ? pickWeighted(family, (x) => w(x.id), 1, rnd) : [];
  const bank: PlayWord[] = WORDS.filter((x) => stage.levels.includes(x.level));
  const fromBank = pickWeighted(bank, (x) => w(x.id), ROUND_SIZE - fromFamily.length, rnd);
  const words = [...fromBank];
  // A palavra da família entra no meio da rodada, não sempre na mesma posição.
  if (fromFamily.length) words.splice(1 + Math.floor(rnd() * (words.length - 1)), 0, fromFamily[0]);
  return words.map((word, i) => {
    // Palavras longas da família no estágio do "primeira letra" seguem nele;
    // no resto, alterna os modos do estágio.
    const mode = stage.modes[(i + Math.floor(rnd() * stage.modes.length)) % stage.modes.length];
    return makeLetterChallenge(mode, word, stage.extras, rnd);
  });
}

/** Rodada da trilha dos Números: 5 contas do estágio, sem repetir conta. */
export function numberRound(p: TrailProgress, rnd: () => number): NumberChallenge[] {
  const stage = NUMBER_STAGES[Math.min(p.stage, NUMBER_STAGES.length - 1)];
  const out: NumberChallenge[] = [];
  const seen = new Set<string>();
  for (let i = 0, guard = 0; out.length < ROUND_SIZE && guard < 200; guard++) {
    const kind = stage.kinds[i % stage.kinds.length];
    const c = makeNumberChallenge(kind, stage.max, rnd);
    if (seen.has(keyOf(c))) continue;
    seen.add(keyOf(c));
    out.push(c);
    i++;
  }
  return out;
}

/** Item do Leitner na trilha dos Números: o tipo de conta (`add`, `sub`…). */
export const numberItemId = (c: NumberChallenge): string => c.kind;
