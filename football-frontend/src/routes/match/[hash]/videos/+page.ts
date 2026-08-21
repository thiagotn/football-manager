import { redirect } from '@sveltejs/kit';
import type { PageLoad } from './$types';

// Rota renomeada para /feed quando passou a aceitar fotos (PRD 052).
// Redirect mantém deeplinks antigos de /videos funcionando.
export const load: PageLoad = ({ params, url }) => {
  redirect(301, `/match/${params.hash}/feed${url.search}`);
};
