/**
 * Naves da Astronave.
 *
 * Nada de imagem: a nave é um desenho vetorial só (os contornos abaixo, usados
 * tanto no canvas da partida quanto no SVG das abas) e cada visual troca as
 * cores. Liberam pela **pontuação somada de todas as partidas** (`stats.total`).
 */

export type SkinKey = 'pioneira' | 'cometa' | 'brasa' | 'gelo' | 'neon' | 'aurora';

export type Skin = {
  key: SkinKey;
  /** Chave i18n do nome (`astronave.skin.<key>`). */
  nameKey: string;
  /** Pontos acumulados para liberar. */
  need: number;
  hull: string;
  wing: string;
  cockpit: string;
  flame: string;
  /** Cor do tiro. */
  shot: string;
};

/**
 * Contornos da nave numa caixa de −8..8 × −9..11 (nariz para cima, origem no
 * centro). No campo ela é desenhada com `SHIP_SCALE`, o que dá a meia largura
 * de 5 unidades que o engine usa nas bordas.
 */
export const SHIP_HULL = 'M0-9 3-2V5L0 7-3 5V-2Z';
export const SHIP_WINGS = 'M-3 0-8 6V8L-3 5ZM3 0 8 6V8L3 5Z';
export const SHIP_FLAME = 'M-1.7 7 0 11 1.7 7Z';
export const SHIP_COCKPIT = { cx: 0, cy: -2.2, rx: 1.4, ry: 2.3 };
export const SHIP_SCALE = 0.62;

export const SKINS: Skin[] = [
  {
    key: 'pioneira',
    nameKey: 'astronave.skin.pioneira',
    need: 0,
    hull: '#e8eefc',
    wing: '#4f7dff',
    cockpit: '#12224a',
    flame: '#ffc93c',
    shot: '#8fe9ff',
  },
  {
    key: 'cometa',
    nameKey: 'astronave.skin.cometa',
    need: 600,
    hull: '#ffe9a8',
    wing: '#ff9f1c',
    cockpit: '#4a2a00',
    flame: '#ffffff',
    shot: '#ffd166',
  },
  {
    key: 'brasa',
    nameKey: 'astronave.skin.brasa',
    need: 1800,
    hull: '#ffd0c2',
    wing: '#e8473b',
    cockpit: '#3a0a06',
    flame: '#ffb020',
    shot: '#ff8a5c',
  },
  {
    key: 'gelo',
    nameKey: 'astronave.skin.gelo',
    need: 4000,
    hull: '#f2fbff',
    wing: '#7fd6ff',
    cockpit: '#0b2f5c',
    flame: '#bfe9ff',
    shot: '#ffffff',
  },
  {
    key: 'neon',
    nameKey: 'astronave.skin.neon',
    need: 8000,
    hull: '#16203f',
    wing: '#00e5c0',
    cockpit: '#9dfff1',
    flame: '#ff5fd2',
    shot: '#7dffe9',
  },
  {
    key: 'aurora',
    nameKey: 'astronave.skin.aurora',
    need: 14000,
    hull: '#f6e9ff',
    wing: '#b06bff',
    cockpit: '#2a0a3a',
    flame: '#7dffb0',
    shot: '#d9a8ff',
  },
];

export const DEFAULT_SKIN: SkinKey = 'pioneira';

export const skinOf = (key: string): Skin => SKINS.find((s) => s.key === key) ?? SKINS[0];

export const isUnlocked = (key: string, total: number): boolean => total >= skinOf(key).need;

export const unlockedSkins = (total: number): SkinKey[] =>
  SKINS.filter((s) => total >= s.need).map((s) => s.key);

/** Próxima nave a liberar — alimenta a barra "faltam X pontos". */
export const nextLock = (total: number): Skin | null =>
  SKINS.find((s) => total < s.need) ?? null;
