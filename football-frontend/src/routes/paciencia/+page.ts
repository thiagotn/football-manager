import { redirect } from '@sveltejs/kit';

// Os jogos moram em games.thiagotn.com desde o PRD 054; a rota fica de pé
// como redirect permanente para não quebrar link salvo nem PWA instalado.
// O layout raiz desliga o SSR; aqui ele volta para que o redirect saia como
// 301 de verdade na resposta — melhor para buscador e para link copiado.
export const ssr = true;

export const load = () => redirect(301, 'https://games.thiagotn.com/paciencia/');
