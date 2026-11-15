/**
 * Bê-á-Bá — gerador puro dos desafios de números (fase 1 do PRD 058).
 *
 * Todo desafio usa objetos desenhados junto com o número, para a criança
 * contar se precisar. Respostas sempre em 3 opções grandes.
 */

export type NumberKind = 'count' | 'compare' | 'add' | 'sub';

export type NumberChallenge =
  | { kind: 'count'; item: string; n: number; options: number[] }
  | { kind: 'compare'; item: string; left: number; right: number }
  | { kind: 'add'; item: string; a: number; b: number; options: number[] }
  | { kind: 'sub'; item: string; a: number; b: number; options: number[] };

/** Objetos de contagem (códigos Twemoji). Nomes em `be_a_ba.item.<código>`. */
export const ITEMS = ['1f34e', '1f388', '2b50', '1f41f', '1f425', '1f36a', '1f338', '1f697'];

const int = (rnd: () => number, min: number, max: number) => min + Math.floor(rnd() * (max - min + 1));

/** Resposta certa: número, ou o lado (`'left'`/`'right'`) no `compare`. */
export function answerOf(c: NumberChallenge): number | 'left' | 'right' {
  switch (c.kind) {
    case 'count':
      return c.n;
    case 'compare':
      return c.left > c.right ? 'left' : 'right';
    case 'add':
      return c.a + c.b;
    case 'sub':
      return c.a - c.b;
  }
}

/** Três opções distintas em `[min, max]`, a certa entre elas, vizinhas dela. */
export function optionsFor(answer: number, min: number, max: number, rnd: () => number): number[] {
  const near = [answer - 2, answer - 1, answer + 1, answer + 2].filter((x) => x >= min && x <= max);
  const picked: number[] = [];
  while (picked.length < 2 && near.length) picked.push(near.splice(Math.floor(rnd() * near.length), 1)[0]);
  for (let x = min; picked.length < 2 && x <= max; x++) if (x !== answer && !picked.includes(x)) picked.push(x);
  const all = [answer, ...picked];
  for (let i = all.length - 1; i > 0; i--) {
    const j = Math.floor(rnd() * (i + 1));
    [all[i], all[j]] = [all[j], all[i]];
  }
  return all;
}

/** Gera um desafio do tipo pedido com números até `max`. */
export function makeNumberChallenge(kind: NumberKind, max: number, rnd: () => number): NumberChallenge {
  const item = ITEMS[Math.floor(rnd() * ITEMS.length)];
  switch (kind) {
    case 'count': {
      const n = int(rnd, 1, max);
      return { kind, item, n, options: optionsFor(n, 1, max, rnd) };
    }
    case 'compare': {
      const left = int(rnd, 1, max);
      let right = int(rnd, 1, max - 1);
      if (right >= left) right++;
      return { kind, item, left, right };
    }
    case 'add': {
      const a = int(rnd, 1, max - 1);
      const b = int(rnd, 1, max - a);
      return { kind, item, a, b, options: optionsFor(a + b, 0, max, rnd) };
    }
    case 'sub': {
      const a = int(rnd, 2, max);
      const b = int(rnd, 1, a - 1);
      return { kind, item, a, b, options: optionsFor(a - b, 0, max, rnd) };
    }
  }
}

/** Chave do desafio para não repetir a mesma conta numa rodada. */
export function keyOf(c: NumberChallenge): string {
  switch (c.kind) {
    case 'count':
      return `count:${c.n}`;
    case 'compare':
      return `compare:${Math.min(c.left, c.right)}-${Math.max(c.left, c.right)}`;
    default:
      return `${c.kind}:${c.a},${c.b}`;
  }
}
