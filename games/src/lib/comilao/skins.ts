/**
 * Visuais do labirinto.
 *
 * Nada de imagem: cada visual é um punhado de cores, então o jogo pesa o mesmo
 * com seis ou com um. Liberam pela **pontuação somada de todas as partidas**
 * (`stats.total`), que é o que a barra da aba Visuais mostra.
 *
 * As cores dos fantasmas **não** entram no visual: são fixas, porque é por elas
 * que o jogador reconhece de longe quem cola, quem embosca e quem desiste. Num
 * jogo em que a leitura de meio segundo decide a vida, isso não é tema.
 */

import type { GhostId } from './engine';

export type SkinKey = 'synth' | 'vapor' | 'laser' | 'brasa' | 'gelo' | 'arco';

export type Skin = {
  key: SkinKey;
  /** Chave i18n do nome (`comilao.skin.<key>`). */
  nameKey: string;
  /** Pontos acumulados para liberar. */
  need: number;
  /** Gradiente da parede, de cima para baixo. */
  wall: [string, string];
  /** Brilho atrás da parede. */
  glow: string;
  /** Linhas da grade do fundo. */
  grid: string;
  /** Gradiente do piso. */
  floor: [string, string];
  hero: string;
  pellet: string;
  power: string;
  /** O `arco` pinta a parede pela coluna, em vez do gradiente. */
  rainbow?: boolean;
};

/** Cores dos fantasmas — fixas em todos os visuais, de propósito. */
export const GHOST_COLORS: Record<GhostId, string> = {
  cacador: '#ff3b5c',
  emboscador: '#ff9f45',
  flanqueador: '#6bffb8',
  ermitao: '#c99bff',
};

/** Fantasma assustado, e o branco do aviso de que o susto vai acabar. */
export const FRIGHT_COLOR = '#2742d8';
export const FRIGHT_BLINK = '#eef2f8';

export const SKINS: Skin[] = [
  {
    key: 'synth',
    nameKey: 'comilao.skin.synth',
    need: 0,
    wall: ['#ff3df0', '#2ae0ff'],
    glow: 'rgba(255, 61, 240, 0.5)',
    grid: 'rgba(120, 90, 220, 0.16)',
    floor: ['#1f1038', '#140a24'],
    hero: '#ffc93c',
    pellet: '#ffe9a8',
    power: '#fff6d6',
  },
  {
    key: 'vapor',
    nameKey: 'comilao.skin.vapor',
    need: 1500,
    wall: ['#ff9ad5', '#7ef0e0'],
    glow: 'rgba(255, 154, 213, 0.45)',
    grid: 'rgba(255, 200, 230, 0.14)',
    floor: ['#2a1540', '#180d2c'],
    hero: '#ffd980',
    pellet: '#ffeef7',
    power: '#ffffff',
  },
  {
    key: 'laser',
    nameKey: 'comilao.skin.laser',
    need: 4000,
    wall: ['#9dff6a', '#2f6fe0'],
    glow: 'rgba(157, 255, 106, 0.42)',
    grid: 'rgba(120, 220, 160, 0.14)',
    floor: ['#102a1f', '#07160f'],
    hero: '#ffe066',
    pellet: '#dcffc4',
    power: '#ffffff',
  },
  {
    key: 'brasa',
    nameKey: 'comilao.skin.brasa',
    need: 8000,
    wall: ['#ffb020', '#d1341f'],
    glow: 'rgba(255, 176, 32, 0.45)',
    grid: 'rgba(255, 150, 80, 0.14)',
    floor: ['#2b1206', '#170803'],
    hero: '#ffe9a8',
    pellet: '#ffd9a8',
    power: '#fff3d0',
  },
  {
    key: 'gelo',
    nameKey: 'comilao.skin.gelo',
    need: 14000,
    wall: ['#eaf7ff', '#5aa9ff'],
    glow: 'rgba(170, 220, 255, 0.4)',
    grid: 'rgba(190, 225, 255, 0.14)',
    floor: ['#0d1c33', '#060e1c'],
    hero: '#ffc93c',
    pellet: '#ffffff',
    power: '#d8f1ff',
  },
  {
    key: 'arco',
    nameKey: 'comilao.skin.arco',
    need: 22000,
    wall: ['#ffffff', '#ffffff'],
    glow: 'rgba(255, 255, 255, 0.28)',
    grid: 'rgba(255, 255, 255, 0.1)',
    floor: ['#1a1030', '#0b0718'],
    hero: '#ffc93c',
    pellet: '#ffffff',
    power: '#ffffff',
    rainbow: true,
  },
];

export const DEFAULT_SKIN: SkinKey = 'synth';

export const skinOf = (key: string): Skin => SKINS.find((s) => s.key === key) ?? SKINS[0];

export const isUnlocked = (key: string, total: number): boolean => total >= skinOf(key).need;

export const unlockedSkins = (total: number): SkinKey[] =>
  SKINS.filter((s) => total >= s.need).map((s) => s.key);

/** Próximo visual a liberar — alimenta a barra "faltam X pontos". */
export const nextLock = (total: number): Skin | null => SKINS.find((s) => total < s.need) ?? null;
