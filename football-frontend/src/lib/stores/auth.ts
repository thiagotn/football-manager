import { writable, derived } from 'svelte/store';
import type { Player } from '$lib/api';

type AuthState = {
  token: string | null;
  player: Player | null;
  loading: boolean;
};

// Lê localStorage de forma síncrona para evitar flash de tela no refresh
function readLocalStorage(): Pick<AuthState, 'token' | 'player'> {
  if (typeof window === 'undefined') return { token: null, player: null };
  const token = localStorage.getItem('token');
  if (!token) return { token: null, player: null };
  try {
    const player = JSON.parse(localStorage.getItem('player') ?? 'null');
    return { token, player };
  } catch {
    return { token, player: null };
  }
}

function createAuthStore() {
  const stored = readLocalStorage();

  const { subscribe, set, update } = writable<AuthState>({
    token: stored.token,
    player: stored.player,
    // Ainda está carregando só se temos token mas não temos os dados do player
    loading: !!stored.token && !stored.player,
  });

  return {
    subscribe,
    async init() {
      const token = localStorage.getItem('token');
      if (!token) {
        set({ token: null, player: null, loading: false });
        return;
      }

      // Tenta restaurar player do localStorage
      const playerStr = localStorage.getItem('player');
      let player: Player | null = null;
      if (playerStr) {
        try {
          player = JSON.parse(playerStr);
        } catch {
          localStorage.removeItem('player');
        }
      }

      // Player não encontrado — busca na API
      if (!player) {
        try {
          const { auth } = await import('$lib/api');
          player = await auth.me();
          localStorage.setItem('player', JSON.stringify(player));
        } catch {
          localStorage.removeItem('token');
          set({ token: null, player: null, loading: false });
          return;
        }
      }

      set({ token, player, loading: false });
    },
    login(token: string, player: { player_id: string; name: string; role: string }) {
      const p = { id: player.player_id, name: player.name, role: player.role } as unknown as Player;
      localStorage.setItem('token', token);
      localStorage.setItem('player', JSON.stringify(p));
      set({ token, player: p, loading: false });
    },
    logout() {
      localStorage.removeItem('token');
      localStorage.removeItem('player');
      set({ token: null, player: null, loading: false });
    },
  };
}

export const authStore = createAuthStore();
export const isLoggedIn = derived(authStore, $a => !!$a.token);
export const currentPlayer = derived(authStore, $a => $a.player);
export const isAdmin = derived(authStore, $a => $a.player?.role === 'admin');
