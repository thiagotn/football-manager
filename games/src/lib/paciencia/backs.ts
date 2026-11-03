/**
 * Versos das cartas da Paciência ("Uniforme das cartas", em Ajustes).
 *
 * Cada verso é desenhado como moldura externa + moldura interna de 3px +
 * emblema centralizado, com a cor e a opacidade do emblema variando por tema.
 */

export type CardBackKey = 'classico' | 'dourado' | 'feltro' | 'noturno';

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
  /** Cor do emblema. */
  emblem: string;
  /** Opacidade do emblema (0 esconde). */
  logo: number;
};

const TEXTURE = 'repeating-linear-gradient(118deg,rgba(255,255,255,.07) 0 2px,transparent 2px 8px)';

export const CARD_BACKS: Record<CardBackKey, CardBack> = {
  classico: {
    labelKey: 'paciencia.back_classico',
    edge: '#ffc93c',
    line: 'rgba(255,201,60,.5)',
    outer: '#0d3123',
    inner: `${TEXTURE},linear-gradient(158deg,#1e6b4a,#17533a 45%,#0d3123)`,
    emblem: '#ffc93c',
    logo: 0.95,
  },
  dourado: {
    labelKey: 'paciencia.back_dourado',
    edge: '#3a2a00',
    line: 'rgba(58,42,0,.35)',
    outer: '#ffc93c',
    inner: `${TEXTURE},linear-gradient(160deg,#ffe29a,#ffc93c 35%,#e0a21f 70%,#ffd666)`,
    emblem: '#3a2a00',
    logo: 0.85,
  },
  feltro: {
    labelKey: 'paciencia.back_feltro',
    edge: '#f9fafb',
    line: 'rgba(255,255,255,.5)',
    outer: '#17533a',
    inner: 'repeating-linear-gradient(90deg,#17533a 0 6px,#0d3123 6px 12px)',
    emblem: '#ffffff',
    logo: 0,
  },
  noturno: {
    labelKey: 'paciencia.back_noturno',
    edge: '#2e7cf0',
    line: 'rgba(90,165,255,.45)',
    outer: '#010112',
    inner: 'radial-gradient(circle,rgba(90,165,255,.5) 1px,transparent 1.5px) 0 0/6px 6px,#03081c',
    emblem: '#9ec5ff',
    logo: 0,
  },
};

export const CARD_BACK_KEYS = Object.keys(CARD_BACKS) as CardBackKey[];
