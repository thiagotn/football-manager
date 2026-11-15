/**
 * Desenhos fixos da interface do Bê-á-Bá (códigos Twemoji). Ficam aqui para o
 * `npm run twemoji` achar todos os desenhos que o jogo usa.
 */

/** A mascote: a corujinha Lili, que fala as instruções e comemora. */
export const MASCOT = '1f989';
/** Estrela do acerto e da comemoração da rodada. */
export const STAR = '1f31f';
export const PARTY = '1f389';
/** Medalha da subida de estágio. */
export const MEDAL = '1f3c5';

/** Caminho público de um desenho. */
export const imgUrl = (code: string): string => `/be-a-ba/img/${code}.svg`;
