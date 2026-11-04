/**
 * Visuais da cobrinha.
 *
 * Nada de imagem: cada visual é uma função que dá a cor do segmento, então o
 * jogo pesa o mesmo com seis peles ou com uma. Liberam pela **pontuação somada
 * de todas as partidas** (`stats.total`), que é o que a barra de progresso da
 * aba Visuais mostra.
 */

export type SkinKey = 'folha' | 'brasa' | 'fita' | 'gelo' | 'neon' | 'arco';

export type Skin = {
  key: SkinKey;
  /** Chave i18n do nome (`cobrinha.skin.<key>`). */
  nameKey: string;
  /** Pontos acumulados para liberar. */
  need: number;
  head: string;
  eye: string;
  /** Brilho opcional atrás dos segmentos. */
  glow?: string;
  /** Cor do segmento `i` num corpo de `len` segmentos (0 = cabeça). */
  body: (i: number, len: number) => string;
};

/** Interpola dois hexadecimais `#rrggbb`. */
function mix(a: string, b: string, t: number): string {
  const hex = (s: string, i: number) => parseInt(s.slice(1 + i * 2, 3 + i * 2), 16);
  const k = Math.max(0, Math.min(1, t));
  const ch = (i: number) => Math.round(hex(a, i) + (hex(b, i) - hex(a, i)) * k);
  return `rgb(${ch(0)},${ch(1)},${ch(2)})`;
}

/** Posição relativa do segmento no corpo (0 na cabeça, 1 na ponta da cauda). */
const along = (i: number, len: number) => (len <= 1 ? 0 : i / (len - 1));

export const SKINS: Skin[] = [
  {
    key: 'folha',
    nameKey: 'cobrinha.skin.folha',
    need: 0,
    head: '#b6f36a',
    eye: '#10361a',
    body: (i, len) => mix('#8ede4f', '#2f8f45', along(i, len)),
  },
  {
    key: 'brasa',
    nameKey: 'cobrinha.skin.brasa',
    need: 500,
    head: '#ffd166',
    eye: '#4a1206',
    body: (i, len) => mix('#ffb020', '#d1341f', along(i, len)),
  },
  {
    key: 'fita',
    nameKey: 'cobrinha.skin.fita',
    need: 1500,
    head: '#ffdd7a',
    eye: '#3a2a00',
    body: (i) => (Math.floor(i / 2) % 2 === 0 ? '#ffc93c' : '#2b3b57'),
  },
  {
    key: 'gelo',
    nameKey: 'cobrinha.skin.gelo',
    need: 3500,
    head: '#eaf7ff',
    eye: '#0b2f5c',
    body: (i, len) => mix('#bfe9ff', '#2f6fe0', along(i, len)),
  },
  {
    key: 'neon',
    nameKey: 'cobrinha.skin.neon',
    need: 7000,
    head: '#9dfff1',
    eye: '#00312a',
    glow: 'rgba(0, 229, 192, 0.45)',
    body: (i, len) => mix('#00e5c0', '#0a8f9b', along(i, len)),
  },
  {
    key: 'arco',
    nameKey: 'cobrinha.skin.arco',
    need: 12000,
    head: '#ffffff',
    eye: '#2a0a3a',
    body: (i) => `hsl(${(i * 16) % 360} 85% 62%)`,
  },
];

export const DEFAULT_SKIN: SkinKey = 'folha';

export const skinOf = (key: string): Skin => SKINS.find((s) => s.key === key) ?? SKINS[0];

export const isUnlocked = (key: string, total: number): boolean => total >= skinOf(key).need;

export const unlockedSkins = (total: number): SkinKey[] =>
  SKINS.filter((s) => total >= s.need).map((s) => s.key);

/** Próximo visual a liberar — alimenta a barra "faltam X pontos". */
export const nextLock = (total: number): Skin | null =>
  SKINS.find((s) => total < s.need) ?? null;
