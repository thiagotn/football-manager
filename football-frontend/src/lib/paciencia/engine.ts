/**
 * Paciência (Klondike) — lógica pura do jogo.
 *
 * Nenhuma dependência de DOM, Svelte ou localStorage: só regras, baralho e
 * resolução de movimentos. A persistência fica em `storage.ts` e o estado
 * reativo em `src/routes/paciencia/+page.svelte`.
 *
 * Pontuação: descarte→tableau +5 · virar carta +5 · fundação +10 ·
 * fundação→tableau −15 · desfazer −2 · reciclar monte −100 (compra 1) / −20
 * (compra 3). Nunca abaixo de 0.
 */

/** Naipes: 0 ♠ · 1 ♥ · 2 ♦ · 3 ♣ (vermelhos são 1 e 2). */
export type Suit = 0 | 1 | 2 | 3;
export type DrawMode = 1 | 3;

export type Card = {
  /** Estável por baralho (`s * 13 + r`) — usado como chave de lista e alvo de dica/shake. */
  id: number;
  s: Suit;
  /** 1 = A … 13 = K */
  r: number;
  up: boolean;
};

export type Game = {
  stock: Card[];
  waste: Card[];
  /** Uma pilha por naipe, na ordem de `Suit`. */
  found: Card[][];
  tableau: Card[][];
  moves: number;
  score: number;
  elapsed: number;
  won: boolean;
  draw: DrawMode;
  /** `YYYY-MM-DD` quando é o desafio diário; `null` em partida livre. */
  daily: string | null;
};

/** Origem de um toque: monte de descarte, fundação ou coluna do tableau. */
export type MoveSource = 'w' | 'f' | 't';

/** Alvo de uma dica — id de carta ou o monte. */
export type HintTarget = number | 'stock';

export const SUITS = ['♠︎', '♥︎', '♦︎', '♣︎'] as const;
export const RANKS = ['', 'A', '2', '3', '4', '5', '6', '7', '8', '9', '10', 'J', 'Q', 'K'] as const;

export const FOUNDATION_SIZE = 13;
export const UNDO_LIMIT = 200;
/** Intervalo (ms) entre cartas na auto-conclusão. */
export const AUTO_FINISH_STEP_MS = 80;

export const isRed = (c: Card): boolean => c.s === 1 || c.s === 2;

/** `m:ss`, ou `—` quando não há valor (cronômetro desligado / sem recorde). */
export function fmtTime(seconds: number | null | undefined): string {
  if (seconds == null) return '—';
  return Math.floor(seconds / 60) + ':' + String(seconds % 60).padStart(2, '0');
}

/** Chave de dia local (`YYYY-MM-DD`) — não usa UTC para não trocar de dia à noite. */
export function dateKey(d: Date): string {
  return (
    d.getFullYear() +
    '-' +
    String(d.getMonth() + 1).padStart(2, '0') +
    '-' +
    String(d.getDate()).padStart(2, '0')
  );
}

/** FNV-1a — gera a seed do desafio diário a partir da data. */
function fnv1a(s: string): number {
  let h = 2166136261;
  for (const ch of s) {
    h ^= ch.charCodeAt(0);
    h = Math.imul(h, 16777619);
  }
  return h >>> 0;
}

/** mulberry32 — PRNG determinístico, garante a mesma distribuição para todos. */
function mulberry32(seed: number): () => number {
  let a = seed >>> 0;
  return () => {
    a |= 0;
    a = (a + 0x6d2b79f5) | 0;
    let t = Math.imul(a ^ (a >>> 15), 1 | a);
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

/** Distribui um novo jogo. Com `daily` a distribuição é determinística pela data. */
export function deal(draw: DrawMode, daily: string | null): Game {
  const rand = daily ? mulberry32(fnv1a('pac' + daily)) : Math.random;
  const d: Card[] = [];
  for (let s = 0; s < 4; s++) {
    for (let r = 1; r <= 13; r++) d.push({ id: s * 13 + r, s: s as Suit, r, up: false });
  }
  for (let i = d.length - 1; i > 0; i--) {
    const j = Math.floor(rand() * (i + 1));
    [d[i], d[j]] = [d[j], d[i]];
  }
  const tableau: Card[][] = [[], [], [], [], [], [], []];
  for (let i = 0; i < 7; i++) {
    for (let j = 0; j <= i; j++) {
      const c = d.pop()!;
      c.up = j === i;
      tableau[i].push(c);
    }
  }
  return {
    stock: d,
    waste: [],
    found: [[], [], [], []],
    tableau,
    moves: 0,
    score: 0,
    elapsed: 0,
    won: false,
    draw,
    daily,
  };
}

export function cloneGame(g: Game): Game {
  return {
    ...g,
    stock: g.stock.map((c) => ({ ...c })),
    waste: g.waste.map((c) => ({ ...c })),
    found: g.found.map((f) => f.map((c) => ({ ...c }))),
    tableau: g.tableau.map((col) => col.map((c) => ({ ...c }))),
  };
}

/** A carta cabe na fundação do seu naipe? */
export function fitsFoundation(g: Game, c: Card): boolean {
  const f = g.found[c.s];
  const top = f[f.length - 1];
  return (!top && c.r === 1) || (!!top && top.r === c.r - 1);
}

/** A carta cabe sobre `top` (cor alternada, valor −1)? Coluna vazia só aceita rei. */
export function fitsTableau(top: Card | undefined, c: Card): boolean {
  return top ? top.up && isRed(top) !== isRed(c) && top.r === c.r + 1 : c.r === 13;
}

/** Vira a carta exposta no topo de cada coluna (+5 cada). Mutação in-place. */
export function flipExposed(g: Game): void {
  for (const col of g.tableau) {
    const top = col[col.length - 1];
    if (top && !top.up) {
      top.up = true;
      g.score += 5;
    }
  }
}

/** Todas as cartas visíveis e nada no monte/descarte — pré-requisito da auto-conclusão. */
export function allUp(g: Game): boolean {
  return !g.stock.length && !g.waste.length && g.tableau.every((col) => col.every((c) => c.up));
}

export function isWon(g: Game): boolean {
  return g.found.every((f) => f.length === FOUNDATION_SIZE);
}

/**
 * Compra do monte: tira `draw` cartas para o descarte. Com o monte vazio,
 * recicla o descarte pagando a penalidade. Mutação in-place.
 */
export function drawFromStock(g: Game): boolean {
  if (g.stock.length) {
    for (let k = 0; k < g.draw && g.stock.length; k++) {
      const c = g.stock.pop()!;
      c.up = true;
      g.waste.push(c);
    }
  } else if (g.waste.length) {
    g.stock = g.waste.reverse().map((c) => ({ ...c, up: false }));
    g.waste = [];
    g.score -= g.draw === 1 ? 100 : 20;
  } else {
    return false;
  }
  g.moves++;
  return true;
}

/**
 * Resolve o toque numa carta (não há arraste). Ordem de prioridade:
 * 1. carta única do topo que cabe na fundação do naipe;
 * 2. coluna do tableau onde caiba (não vazias primeiro);
 * 3. toque no meio da coluna move a sequência a partir dali;
 * 4. carta da fundação pode voltar ao tableau (−15).
 *
 * Mutação in-place. Devolve o id da carta tocada (para o shake) e se moveu.
 */
export function resolveTap(
  g: Game,
  src: MoveSource,
  col = 0,
  idx = 0
): { moved: boolean; cardId: number | null } {
  let cards: Card[];
  let colArr: Card[] | undefined;

  if (src === 'w') {
    if (!g.waste.length) return { moved: false, cardId: null };
    cards = [g.waste[g.waste.length - 1]];
  } else if (src === 'f') {
    const f = g.found[col];
    if (!f.length) return { moved: false, cardId: null };
    cards = [f[f.length - 1]];
  } else {
    colArr = g.tableau[col];
    if (!colArr[idx]?.up) return { moved: false, cardId: null };
    cards = colArr.slice(idx);
  }

  const c0 = cards[0];
  const remove = () => {
    if (src === 'w') g.waste.pop();
    else if (src === 'f') g.found[col].pop();
    else colArr!.splice(idx);
  };

  if (cards.length === 1 && src !== 'f' && fitsFoundation(g, c0)) {
    remove();
    g.found[c0.s].push(c0);
    g.score += 10;
    g.moves++;
    return { moved: true, cardId: c0.id };
  }

  // Colunas não vazias têm prioridade sobre as vazias.
  const order = [0, 1, 2, 3, 4, 5, 6].sort(
    (a, b) => (g.tableau[a].length ? 0 : 1) - (g.tableau[b].length ? 0 : 1)
  );
  for (const target of order) {
    if (src === 't' && target === col) continue;
    const tc = g.tableau[target];
    const top = tc[tc.length - 1];
    // Um rei que já está na base de uma coluna não troca de coluna vazia.
    if (!top && src === 't' && idx === 0) continue;
    if (fitsTableau(top, c0)) {
      remove();
      tc.push(...cards);
      g.score += src === 'w' ? 5 : src === 'f' ? -15 : 0;
      g.moves++;
      return { moved: true, cardId: c0.id };
    }
  }

  return { moved: false, cardId: c0.id };
}

/**
 * Melhor movimento disponível, como ids a destacar: fundação → tableau → monte.
 * No tableau devolve a carta e o destino.
 */
export function findHint(g: Game): HintTarget[] {
  type Cand = { c: Card; single: boolean; src: 'w' | 't'; ci?: number; fi?: number };
  const cands: Cand[] = [];

  const w = g.waste[g.waste.length - 1];
  if (w) cands.push({ c: w, single: true, src: 'w' });
  g.tableau.forEach((col, ci) => {
    const fi = col.findIndex((c) => c.up);
    if (fi < 0) return;
    cands.push({ c: col[col.length - 1], single: true, src: 't', ci });
    cands.push({ c: col[fi], single: false, src: 't', ci, fi });
  });

  for (const k of cands) {
    if (k.single && fitsFoundation(g, k.c)) return [k.c.id];
  }

  for (const k of cands) {
    // Topo de coluna já foi coberto acima; aqui vale a sequência inteira.
    if (k.single && k.src === 't') continue;
    for (let target = 0; target < 7; target++) {
      if (k.src === 't' && target === k.ci) continue;
      const tc = g.tableau[target];
      const top = tc[tc.length - 1];
      if (!top && k.src === 't' && k.fi === 0) continue;
      if (fitsTableau(top, k.c)) return top ? [k.c.id, top.id] : [k.c.id];
    }
  }

  return ['stock'];
}

/** Um passo da auto-conclusão: leva a menor carta possível para a fundação. */
export function autoFinishStep(g: Game): boolean {
  let best: { c: Card; col: Card[] } | null = null;
  for (const col of g.tableau) {
    if (!col.length) continue;
    const c = col[col.length - 1];
    if (fitsFoundation(g, c) && (!best || c.r < best.c.r)) best = { c, col };
  }
  if (!best) return false;
  best.col.pop();
  g.found[best.c.s].push(best.c);
  g.score += 10;
  g.moves++;
  return true;
}

/** Estrelas da tela de vitória pelo tempo: <4min 5 · <7min 4 · <12min 3 · senão 2. */
export function winStars(elapsed: number): number {
  if (elapsed < 240) return 5;
  if (elapsed < 420) return 4;
  if (elapsed < 720) return 3;
  return 2;
}
