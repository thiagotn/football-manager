import { writable, derived } from 'svelte/store';
import type { Player } from '$lib/api';

type AuthState = {
  token: string | null;
  player: Player | null;
  loading: boolean;
};

function createAuthStore() {
  const { subscribe, set, update } = writable<AuthState>({
    token: null,
    player: null,
    loading: true,
  });

  return {
    subscribe,
    async init() {
      const token = localStorage.getItem('token');
      if (!token) {
        update(s => ({ ...s, loading: false }));
        return;
      }

      // Token exists — try to restore player from localStorage
      const playerStr = localStorage.getItem('player');
      let player: Player | null = null;
      if (playerStr) {
        try {
          player = JSON.parse(playerStr);
        } catch {
          localStorage.removeItem('player');
        }
      }

      // If player data is missing, fetch it from the API
      if (!player) {
        try {
          const { auth } = await import('$lib/api');
          player = await auth.me();
          localStorage.setItem('player', JSON.stringify(player));
        } catch {
          // Token is invalid/expired
          localStorage.removeItem('token');
          update(s => ({ ...s, loading: false }));
          return;
        }
      }

      update(s => ({ ...s, token, player, loading: false }));
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
