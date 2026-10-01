/**
 * Versos das cartas da Paciência ("Uniforme das cartas", em Ajustes).
 *
 * Cada verso é desenhado como moldura externa + moldura interna de 3px +
 * logo centralizado, com a opacidade do logo variando por tema.
 */

export type CardBackKey = 'escudo' | 'dourado' | 'gramado' | 'noturno';

export type CardBack = {
  /** Chave i18n do nome exibido em Ajustes. */
  labelKey: string;
  /** Borda externa. */
  edge: string;
  /** Linha da moldura interna. */
  line: string;
  /** Fundo da borda externa. */
  outer: string;
  /** Fundo da moldura interna. */
  inner: string;
  /** Opacidade do logo (0 esconde). */
  logo: number;
};

const TEXTURE = 'repeating-linear-gradient(118deg,rgba(255,255,255,.07) 0 2px,transparent 2px 8px)';

export const CARD_BACKS: Record<CardBackKey, CardBack> = {
  escudo: {
    labelKey: 'paciencia.back_escudo',
    edge: '#c9a34d',
    line: 'rgba(245,213,133,.55)',
    outer: '#07331a',
    inner: `${TEXTURE},linear-gradient(158deg,#15422a,#1f9a4d 40%,#0d5028 70%,#07331a)`,
    logo: 0.95,
  },
  dourado: {
    labelKey: 'paciencia.back_dourado',
    edge: '#8a6c2a',
    line: 'rgba(7,51,26,.35)',
    outer: '#c9a34d',
    inner: `${TEXTURE},linear-gradient(160deg,#f7e2ad,#e0b95c 30%,#8a6c2a 60%,#f5d585)`,
    logo: 0.9,
  },
  gramado: {
    labelKey: 'paciencia.back_gramado',
    edge: '#f9fafb',
    line: 'rgba(255,255,255,.5)',
    outer: '#16a34a',
    inner: 'repeating-linear-gradient(90deg,#16a34a 0 6px,#15803d 6px 12px)',
    logo: 0,
  },
  noturno: {
    labelKey: 'paciencia.back_noturno',
    edge: '#3b82f6',
    line: 'rgba(90,165,255,.45)',
    outer: '#010112',
    inner: 'radial-gradient(circle,rgba(90,165,255,.5) 1px,transparent 1.5px) 0 0/6px 6px,#03081c',
    logo: 0,
  },
};

export const CARD_BACK_KEYS = Object.keys(CARD_BACKS) as CardBackKey[];
