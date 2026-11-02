/**
 * Enche Aí! — conteúdo: sabores, recipientes e cardápios.
 *
 * Nomes visíveis são chaves i18n (`enche_ai.flavor.<id>`, `enche_ai.world.<id>`,
 * `enche_ai.vessel.<id>`), nunca texto literal.
 */

import { DAILY_FLAVORS, LEVELS_PER_WORLD, dailySeed, dateKey, flavorCountFor, levelSeed } from './engine';

export type Topping = 'steam' | 'straw' | 'cream';
export type PatternId =
  | 'beans' | 'plain' | 'waves' | 'pulp' | 'rings' | 'pits'
  | 'specks' | 'stripes' | 'seeds' | 'granola' | 'bubbles' | 'ice';

export type Flavor = {
  id: string;
  color: string;
  pattern: PatternId;
  topping: Topping;
};

export const FLAVORS: Record<string, Flavor> = {
  cafe:       { id: 'cafe',       color: '#3a1f14', pattern: 'beans',   topping: 'steam' },
  leite:      { id: 'leite',      color: '#f1ebdf', pattern: 'plain',   topping: 'steam' },
  chocolate:  { id: 'chocolate',  color: '#8b5a3a', pattern: 'waves',   topping: 'steam' },
  laranja:    { id: 'laranja',    color: '#f7941d', pattern: 'pulp',    topping: 'straw' },
  limao:      { id: 'limao',      color: '#9bd13a', pattern: 'rings',   topping: 'straw' },
  melancia:   { id: 'melancia',   color: '#e02a3c', pattern: 'pits',    topping: 'straw' },
  maracuja:   { id: 'maracuja',   color: '#ffd23f', pattern: 'specks',  topping: 'straw' },
  menta:      { id: 'menta',      color: '#22b8a0', pattern: 'stripes', topping: 'straw' },
  morango:    { id: 'morango',    color: '#f37aa6', pattern: 'seeds',   topping: 'cream' },
  acai:       { id: 'acai',       color: '#4b1a55', pattern: 'granola', topping: 'cream' },
  uva:        { id: 'uva',        color: '#7a4fd6', pattern: 'bubbles', topping: 'straw' },
  raspadinha: { id: 'raspadinha', color: '#2e7cf0', pattern: 'ice',     topping: 'cream' },
};

/**
 * Geometria do recipiente no viewBox `0 -34 100 250`.
 * `mouth` = y da boca; `top`/`bottom` = faixa ocupada pelas 4 camadas;
 * `cx` = centro horizontal (a caneca tem alça à direita).
 */
export type Vessel = {
  id: 'caneca' | 'americano' | 'long' | 'taca';
  mouth: number;
  top: number;
  bottom: number;
  cx: number;
  outline: string;
  clip: string;
  /** Detalhes desenhados por cima do vidro (alça, facetas). */
  details: { d: string; width: number; opacity: number }[];
};

function americanoFacets(): Vessel['details'] {
  const out: Vessel['details'] = [{ d: 'M15.6 62H84.4', width: 1.6, opacity: 0.35 }];
  for (let k = 0; k < 8; k++) {
    const xt = 16.5 + (k + 0.5) * (67.6 / 8);
    const xb = 22.4 + (k + 0.5) * (55.2 / 8);
    out.push({ d: `M${xt.toFixed(1)} 64L${xb.toFixed(1)} 205`, width: 1.4, opacity: 0.16 });
  }
  return out;
}

export const VESSELS: Record<Vessel['id'], Vessel> = {
  caneca: {
    id: 'caneca', mouth: 70, top: 88, bottom: 209, cx: 43,
    outline: 'M10 70H76V196Q76 212 60 212H26Q10 212 10 196Z',
    clip: 'M13 73H73V194Q73 209 58 209H28Q13 209 13 194Z',
    details: [{ d: 'M76 100C99 100 99 170 76 170', width: 7, opacity: 0.78 }],
  },
  americano: {
    id: 'americano', mouth: 34, top: 54, bottom: 207, cx: 50,
    outline: 'M13 34H87L80.5 206Q80 212 74 212H26Q20 212 19.5 206Z',
    clip: 'M16.5 37H83.5L77.6 207H22.4Z',
    details: americanoFacets(),
  },
  long: {
    id: 'long', mouth: 30, top: 46, bottom: 209, cx: 50,
    outline: 'M20 30H80V198Q80 212 66 212H34Q20 212 20 198Z',
    clip: 'M23 33H77V197Q77 209 65 209H35Q23 209 23 197Z',
    details: [],
  },
  taca: {
    id: 'taca', mouth: 30, top: 42, bottom: 141, cx: 50,
    outline:
      'M8 30H92C92 104 72 140 56 146V186C56 194 74 196 80 206Q80 211 75 211H25Q20 211 20 206C26 196 44 194 44 186V146C28 140 8 104 8 30Z',
    clip: 'M11 33H89C89 103 70 137 50 142C30 137 11 103 11 33Z',
    details: [],
  },
};

/** Altura total do viewBox dos recipientes (de -34 a 216). */
export const VIEWBOX_HEIGHT = 250;
export const VIEWBOX_TOP = -34;

export type World = {
  id: 'padaria' | 'lanchonete' | 'sucos' | 'sorveteria';
  vessel: Vessel['id'];
  /** Ordem em que os sabores entram nas fases deste cardápio. */
  order: string[];
};

export const WORLDS: World[] = [
  { id: 'padaria', vessel: 'caneca',
    order: ['cafe', 'leite', 'chocolate', 'laranja', 'morango', 'maracuja', 'limao', 'acai', 'melancia'] },
  { id: 'lanchonete', vessel: 'americano',
    order: ['uva', 'laranja', 'cafe', 'limao', 'melancia', 'leite', 'raspadinha', 'chocolate', 'menta'] },
  { id: 'sucos', vessel: 'long',
    order: ['laranja', 'melancia', 'maracuja', 'limao', 'acai', 'menta', 'morango', 'uva', 'raspadinha'] },
  { id: 'sorveteria', vessel: 'taca',
    order: ['morango', 'acai', 'raspadinha', 'chocolate', 'leite', 'maracuja', 'menta', 'uva', 'cafe'] },
];

export const worldStart = (i: number): number => i * LEVELS_PER_WORLD + 1;

/** Cardápio do pedido; depois do último o ciclo recomeça. */
export const worldIndexFor = (level: number): number =>
  Math.floor((Math.max(1, level) - 1) / LEVELS_PER_WORLD) % WORLDS.length;

export type LevelConfig = {
  level: number;
  /** `YYYY-MM-DD` no pedido do dia; `null` nos pedidos normais. */
  daily: string | null;
  world: World;
  flavors: string[];
  seed: number;
};

export function levelConfig(level: number): LevelConfig {
  const world = WORLDS[worldIndexFor(level)];
  return { level, daily: null, world, flavors: world.order.slice(0, flavorCountFor(level)), seed: levelSeed(level) };
}

/** Pedido do dia: 7 sabores, cardápio pelo dia da semana, mesma fase para todos. */
export function dailyConfig(date: Date, level: number): LevelConfig {
  const day = dateKey(date);
  const world = WORLDS[date.getDay() % WORLDS.length];
  return { level, daily: day, world, flavors: world.order.slice(0, DAILY_FLAVORS), seed: dailySeed(day) };
}
