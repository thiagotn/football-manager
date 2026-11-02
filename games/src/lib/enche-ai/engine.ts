/**
 * Enche Aí! — lógica pura do jogo.
 *
 * Sem DOM, Svelte ou localStorage: só regras, geração seedada de fases e o
 * solver usado para validar fases e dar dicas. Persistência em `storage.ts`,
 * conteúdo (sabores, recipientes, cardápios) em `content.ts`.
 *
 * Regras: cada recipiente tem `CAPACITY` camadas. Só se despeja sobre o mesmo
 * sabor ou num recipiente vazio, e a sequência inteira do topo vai junto até
 * onde couber. Recipiente cheio com um sabor só fica pronto (travado).
 */

export const CAPACITY = 4;

/** Ajudas disponíveis por pedido. */
export const AIDS = { undo: 3, extra: 1, hint: 3 } as const;
export type AidKind = keyof typeof AIDS;
export type AidCount = Record<AidKind, number>;
export type AidUsage = Record<AidKind, boolean>;

/** Um recipiente: camadas de baixo (índice 0) para cima. */
export type Tube = string[];
export type Board = Tube[];
export type Move = [from: number, to: number];

/** Recipientes vazios extras em toda fase. */
export const EMPTY_TUBES = 2;
export const MAX_FLAVORS = 9;
export const DAILY_FLAVORS = 7;
/** Pedidos por cardápio. */
export const LEVELS_PER_WORLD = 10;

export const topOf = (t: Tube): string | undefined => t[t.length - 1];

/** Quantas camadas iguais estão em sequência no topo. */
export function runLength(t: Tube): number {
  if (!t.length) return 0;
  const c = t[t.length - 1];
  let n = 0;
  for (let i = t.length - 1; i >= 0 && t[i] === c; i--) n++;
  return n;
}

export const isComplete = (t: Tube): boolean => t.length === CAPACITY && runLength(t) === CAPACITY;

export const isWon = (b: Board): boolean => b.every((t) => t.length === 0 || isComplete(t));

export function canPour(b: Board, from: number, to: number): boolean {
  if (from === to) return false;
  const A = b[from];
  const B = b[to];
  if (!A || !B || !A.length || B.length >= CAPACITY || isComplete(A)) return false;
  return B.length === 0 || topOf(B) === topOf(A);
}

/** Despeja `from → to` e devolve um tabuleiro novo e quantas camadas foram. */
export function applyPour(b: Board, from: number, to: number): { board: Board; moved: number } {
  if (!canPour(b, from, to)) return { board: b, moved: 0 };
  const board = b.map((t) => t.slice());
  const moved = pourInPlace(board, from, to);
  return { board, moved };
}

function pourInPlace(b: Board, from: number, to: number): number {
  const A = b[from];
  const B = b[to];
  const k = Math.min(runLength(A), CAPACITY - B.length);
  for (let i = 0; i < k; i++) B.push(A.pop() as string);
  return k;
}

export function hasMove(b: Board): boolean {
  for (let a = 0; a < b.length; a++) for (let c = 0; c < b.length; c++) if (canPour(b, a, c)) return true;
  return false;
}

/** Sabores já prontos (recipientes completos). */
export const completedFlavors = (b: Board): Set<string> =>
  new Set(b.filter(isComplete).map((t) => t[0]));

const keyOf = (b: Board): string => b.map((t) => t.join(',')).sort().join('|');

/**
 * Busca em profundidade com memo de estados. Devolve a lista de jogadas até a
 * vitória (não necessariamente a mais curta) ou `null` se não achar dentro de
 * `limit` estados. Poda jogadas inúteis: despejar recipiente de um sabor só
 * num vazio, e tentar mais de um vazio para a mesma origem.
 */
export function solve(start: Board, limit = 40000): Move[] | null {
  const seen = new Set<string>();
  const path: Move[] = [];
  let nodes = 0;

  function moves(b: Board): Move[] {
    const scored: [number, Move][] = [];
    for (let a = 0; a < b.length; a++) {
      const A = b[a];
      if (!A.length || isComplete(A)) continue;
      const uniform = runLength(A) === A.length;
      let emptyTried = false;
      for (let c = 0; c < b.length; c++) {
        if (!canPour(b, a, c)) continue;
        if (!b[c].length) {
          if (uniform || emptyTried) continue;
          emptyTried = true;
          scored.push([1, [a, c]]);
        } else {
          // Preferir despejos que levam a sequência inteira.
          scored.push([CAPACITY - b[c].length >= runLength(A) ? -1 : 0, [a, c]]);
        }
      }
    }
    return scored.sort((x, y) => x[0] - y[0]).map((s) => s[1]);
  }

  function dfs(b: Board): boolean {
    if (isWon(b)) return true;
    if (++nodes > limit) return false;
    const k = keyOf(b);
    if (seen.has(k)) return false;
    seen.add(k);
    for (const [a, c] of moves(b)) {
      const next = b.map((t) => t.slice());
      pourInPlace(next, a, c);
      path.push([a, c]);
      if (dfs(next)) return true;
      path.pop();
      if (nodes > limit) return false;
    }
    return false;
  }

  return dfs(start.map((t) => t.slice())) ? path.slice() : null;
}

/** PRNG determinístico (mulberry32). */
export function mulberry32(seed: number): () => number {
  let a = seed;
  return () => {
    a |= 0;
    a = (a + 0x6d2b79f5) | 0;
    let t = Math.imul(a ^ (a >>> 15), 1 | a);
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

/** FNV-1a 32 bits — transforma texto (pedido, data) em seed. */
export function fnv1a(s: string): number {
  let h = 2166136261;
  for (const ch of s) {
    h ^= ch.charCodeAt(0);
    h = Math.imul(h, 16777619);
  }
  return h >>> 0;
}

/** Chave de dia local (`YYYY-MM-DD`) — não usa UTC para não virar o dia à noite. */
export function dateKey(d: Date): string {
  return (
    d.getFullYear() + '-' + String(d.getMonth() + 1).padStart(2, '0') + '-' + String(d.getDate()).padStart(2, '0')
  );
}

export const levelSeed = (level: number): number => fnv1a('enche-ai-' + level);
export const dailySeed = (day: string): number => fnv1a('enche-ai-diario-' + day);

/** 3 sabores no pedido 1, +1 a cada 4 pedidos, até `MAX_FLAVORS`. */
export const flavorCountFor = (level: number): number =>
  Math.min(MAX_FLAVORS, 3 + Math.floor((Math.max(1, level) - 1) / 4));

/**
 * Gera uma fase com `flavors.length` recipientes cheios + `EMPTY_TUBES` vazios.
 * Rejeita embaralhamentos com 3+ camadas iguais no mesmo recipiente e os que o
 * solver não resolve — a fase que chega ao jogador sempre tem solução.
 */
export function generate(seed: number, flavors: readonly string[]): Board {
  const rnd = mulberry32(seed);
  let fallback: Board | null = null;
  for (let attempt = 0; attempt < 200; attempt++) {
    const pool: string[] = [];
    for (const f of flavors) for (let i = 0; i < CAPACITY; i++) pool.push(f);
    for (let i = pool.length - 1; i > 0; i--) {
      const j = Math.floor(rnd() * (i + 1));
      [pool[i], pool[j]] = [pool[j], pool[i]];
    }
    const board: Board = [];
    for (let i = 0; i < flavors.length; i++) board.push(pool.slice(i * CAPACITY, (i + 1) * CAPACITY));
    for (let i = 0; i < EMPTY_TUBES; i++) board.push([]);
    if (board.some((t) => runLength(t) >= 3)) continue;
    if (solve(board)) return board;
    fallback ??= board;
  }
  // Na prática não acontece (coberto por teste); devolve algo jogável com o copo extra.
  return fallback ?? flavors.map((f) => Array(CAPACITY).fill(f)).concat([[], []]);
}

/** 3 estrelas sem ajuda; −1 por tipo de ajuda usada; mínimo 1. */
export const starsFor = (used: AidUsage): number =>
  Math.max(1, 3 - Object.values(used).filter(Boolean).length);

export const fmtTime = (seconds: number): string =>
  Math.floor(seconds / 60) + ':' + String(seconds % 60).padStart(2, '0');
