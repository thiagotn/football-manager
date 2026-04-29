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

      // Usa o cache para renderizar imediatamente sem flash
      const playerStr = localStorage.getItem('player');
      if (playerStr) {
        try {
          const cached = JSON.parse(playerStr);
          set({ token, player: cached, loading: true });
        } catch {
          localStorage.removeItem('player');
        }
      }

      // Sempre busca dados frescos da API (garante campos como chat_enabled atualizados)
      try {
        const { auth } = await import('$lib/api');
        const player = await auth.me();
        localStorage.setItem('player', JSON.stringify(player));
        set({ token, player, loading: false });
      } catch {
        localStorage.removeItem('token');
        set({ token: null, player: null, loading: false });
      }
    },
    login(token: string, player: { player_id: string; name: string; nickname?: string | null; role: string; must_change_password?: boolean; avatar_url?: string | null; chat_enabled?: boolean }) {
      const p = { id: player.player_id, name: player.name, nickname: player.nickname ?? null, role: player.role, must_change_password: player.must_change_password ?? false, avatar_url: player.avatar_url ?? null, chat_enabled: player.chat_enabled ?? false } as unknown as Player;
      localStorage.setItem('token', token);
      localStorage.setItem('player', JSON.stringify(p));
      set({ token, player: p, loading: false });
    },
    updatePlayer(data: Partial<Player>) {
      update(state => {
        if (!state.player) return state;
        const updatedPlayer = { ...state.player, ...data };
        localStorage.setItem('player', JSON.stringify(updatedPlayer));
        return { ...state, player: updatedPlayer };
      });
    },
    setMustChangePassword(value: boolean) {
      update(state => {
        if (!state.player) return state;
        const updatedPlayer = { ...state.player, must_change_password: value };
        localStorage.setItem('player', JSON.stringify(updatedPlayer));
        return { ...state, player: updatedPlayer };
      });
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
export const isChatEnabled = derived(authStore, $a => $a.player?.chat_enabled === true);
