const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8000/api/v1';

export class ApiError extends Error {
  constructor(public status: number, message: string) {
    super(message);
  }
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const token = typeof localStorage !== 'undefined' ? localStorage.getItem('token') : null;

  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...options.headers,
    },
  });

  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    const msg = body?.detail ?? `Erro ${res.status}`;
    throw new ApiError(res.status, Array.isArray(msg) ? msg[0]?.msg ?? String(msg) : String(msg));
  }

  if (res.status === 204) return undefined as T;
  return res.json();
}

const get = <T>(path: string) => request<T>(path, { method: 'GET' });
const post = <T>(path: string, body?: unknown) => request<T>(path, { method: 'POST', body: JSON.stringify(body) });
const patch = <T>(path: string, body?: unknown) => request<T>(path, { method: 'PATCH', body: JSON.stringify(body) });
const del = (path: string) => request<void>(path, { method: 'DELETE' });

// ── Auth ──────────────────────────────────────────────────────
export const auth = {
  login: (whatsapp: string, password: string) =>
    post<{ access_token: string; player_id: string; name: string; role: string; must_change_password: boolean }>('/auth/login', { whatsapp, password }),
  me: () => get<Player>('/auth/me'),
  changePassword: (current_password: string, new_password: string) =>
    post<void>('/auth/change-password', { current_password, new_password }),
};

// ── Types ─────────────────────────────────────────────────────
export type Player = {
  id: string; name: string; nickname: string | null;
  whatsapp: string; role: 'admin' | 'player'; active: boolean;
  must_change_password: boolean;
  created_at: string; updated_at: string;
};
export type PlayerPublic = { id: string; name: string; nickname: string | null; role: string };
export type PlayerMemberView = PlayerPublic & { whatsapp: string };
export type Group = { id: string; name: string; description: string | null; slug: string; per_match_amount: number | null; monthly_amount: number | null; recurrence_enabled: boolean; created_at: string; updated_at: string };
export type GroupMember = { id: string; player: PlayerMemberView; role: 'admin' | 'member'; created_at: string };
export type GroupDetail = Group & { members: GroupMember[]; total_members: number };
export type Match = {
  id: string; number: number; group_id: string; match_date: string; start_time: string; end_time: string | null;
  location: string; address: string | null;
  court_type: 'campo' | 'sintetico' | 'terrao' | 'quadra' | null;
  players_per_team: number | null;
  max_players: number | null;
  notes: string | null; hash: string; status: 'open' | 'in_progress' | 'closed';
  created_at: string; updated_at: string;
};
export type Attendance = { id: string; player: PlayerPublic; status: 'pending' | 'confirmed' | 'declined'; updated_at: string };
export type MatchDetail = Match & { attendances: Attendance[]; confirmed_count: number; declined_count: number; pending_count: number; group_name: string; group_per_match_amount: number | null; group_monthly_amount: number | null };

// ── Players ───────────────────────────────────────────────────
export const players = {
  list: () => get<Player[]>('/players'),
  get: (id: string) => get<Player>(`/players/${id}`),
  create: (data: { name: string; nickname?: string; whatsapp: string; password: string; role?: string }) =>
    post<Player>('/players', data),
  update: (id: string, data: Partial<{ name: string; nickname: string; whatsapp: string; password: string; role: string; active: boolean }>) =>
    patch<Player>(`/players/${id}`, data),
  delete: (id: string) => del(`/players/${id}`),
  resetPassword: (id: string) => post<{ temp_password: string }>(`/players/${id}/reset-password`),
};

// ── Groups ────────────────────────────────────────────────────
export const groups = {
  list: () => get<Group[]>('/groups'),
  get: (id: string) => get<GroupDetail>(`/groups/${id}`),
  create: (data: { name: string; description?: string; slug?: string }) => post<Group>('/groups', data),
  update: (id: string, data: { name?: string; description?: string; per_match_amount?: number | null; monthly_amount?: number | null; recurrence_enabled?: boolean }) => patch<Group>(`/groups/${id}`, data),
  delete: (id: string) => del(`/groups/${id}`),
  addMember: (groupId: string, playerId: string, role = 'member') =>
    post<GroupMember>(`/groups/${groupId}/members`, { player_id: playerId, role }),
  removeMember: (groupId: string, playerId: string) => del(`/groups/${groupId}/members/${playerId}`),
  updateMemberRole: (groupId: string, playerId: string, role: string) =>
    patch<GroupMember>(`/groups/${groupId}/members/${playerId}`, { role }),
};

// ── Matches ───────────────────────────────────────────────────
export const matches = {
  list: (groupId: string) => get<Match[]>(`/groups/${groupId}/matches`),
  get: (groupId: string, matchId: string) => get<MatchDetail>(`/groups/${groupId}/matches/${matchId}`),
  getByHash: (hash: string) => get<MatchDetail>(`/matches/public/${hash}`),
  create: (groupId: string, data: { match_date: string; start_time: string; location: string; notes?: string }) =>
    post<Match>(`/groups/${groupId}/matches`, data),
  update: (groupId: string, matchId: string, data: Partial<{ match_date: string; start_time: string; end_time: string | null; location: string; address: string | null; court_type: string | null; players_per_team: number | null; max_players: number | null; notes: string | null; status: string }>) =>
    patch<Match>(`/groups/${groupId}/matches/${matchId}`, data),
  delete: (groupId: string, matchId: string) => del(`/groups/${groupId}/matches/${matchId}`),
  setAttendance: (groupId: string, matchId: string, playerId: string, status: string) =>
    post<Attendance>(`/groups/${groupId}/matches/${matchId}/attendance`, { player_id: playerId, status }),
};

// ── Invites ───────────────────────────────────────────────────
export const invites = {
  create: (groupId: string) => post<{ id: string; token: string; expires_at: string }>('/invites', { group_id: groupId }),
  getInfo: (token: string) => get<{ valid: boolean; group_name: string; expires_at: string }>(`/invites/${token}`),
  checkWhatsapp: (token: string, whatsapp: string) =>
    get<{ exists: boolean; first_name: string | null }>(`/invites/${token}/check?whatsapp=${encodeURIComponent(whatsapp)}`),
  accept: (token: string, data: { name?: string; nickname?: string; whatsapp: string; password: string }) =>
    post<{ access_token: string; player_id: string; name: string; role: string }>(`/invites/${token}/accept`, data),
};
