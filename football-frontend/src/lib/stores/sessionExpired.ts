import { writable } from 'svelte/store';

// Sinaliza que uma requisição autenticada recebeu 401 (sessão expirada).
// Observado via $effect no layout raiz para disparar logout + toast + redirect.
export const sessionExpiredStore = writable(false);
