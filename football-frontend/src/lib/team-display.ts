import type { DrawStrategy } from '$lib/api';

/**
 * Regras de exibição do resultado do sorteio.
 *
 * O sorteio simplificado equilibra **apenas as estrelas** e ignora as posições
 * de linha de propósito. Exibir a posição de cada jogador nesse modo faz o
 * resultado parecer desequilibrado ("por que aquele time ficou com 3 zagueiros
 * e o meu com nenhum?") e gera questionamento sobre um sorteio que está
 * correto pelo critério escolhido.
 *
 * O goleiro é a exceção: é a única posição que o modo simplificado continua
 * garantindo (um por time), então mostrá-la não sugere desequilíbrio nenhum.
 */
export function showsPosition(strategy: DrawStrategy, isGoalkeeper: boolean): boolean {
  return strategy !== 'simple' || isGoalkeeper;
}

/**
 * Estrelas por jogador seguem a mesma lógica: no modo simplificado elas são o
 * critério do sorteio, e expô-las jogador a jogador convida à mesma comparação.
 * O total por time não passa por aqui — ele é justamente a prova de que os
 * times ficaram parelhos, e só o admin do grupo o enxerga.
 */
export function showsPlayerStars(strategy: DrawStrategy): boolean {
  return strategy !== 'simple';
}
