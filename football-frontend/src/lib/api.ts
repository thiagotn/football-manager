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
  register: (data: { name: string; whatsapp: string; password: string; nickname?: string }) =>
    post<{ access_token: string; player_id: string; name: string; role: string; must_change_password: boolean }>('/auth/register', data),
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
export type Group = { id: string; name: string; description: string | null; slug: string; per_match_amount: number | null; monthly_amount: number | null; recurrence_enabled: boolean; vote_open_delay_minutes: number; vote_duration_hours: number; created_at: string; updated_at: string };
export type GroupMember = { id: string; player: PlayerMemberView; role: 'admin' | 'member'; skill_stars: number | null; is_goalkeeper: boolean | null; created_at: string };
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
export type MonthlyStatItem = { month: string; matches_confirmed: number; minutes_played: number };
export type RecentMatchItem = { match_date: string; group_name: string; status: 'confirmed' | 'declined' };
export type GroupStatItem = { group_id: string; group_name: string; skill_stars: number; is_goalkeeper: boolean; role: 'admin' | 'member'; matches_confirmed: number };
export type PlayerFullStats = {
  total_matches_confirmed: number;
  total_minutes_played: number;
  total_vote_points: number;
  total_flop_votes: number;
  top1_count: number;
  top5_count: number;
  current_streak: number;
  best_streak: number;
  attendance_rate: number;
  monthly_stats: MonthlyStatItem[];
  recent_matches: RecentMatchItem[];
  groups: GroupStatItem[];
};

export type SignupStats = {
  total: number;
  last_7_days: number;
  last_30_days: number;
  recent: Array<{ id: string; name: string; nickname: string | null; whatsapp: string; active: boolean; created_at: string }>;
};

export const players = {
  list: () => get<Player[]>('/players'),
  get: (id: string) => get<Player>(`/players/${id}`),
  create: (data: { name: string; nickname?: string; whatsapp: string; password: string; role?: string }) =>
    post<Player>('/players', data),
  update: (id: string, data: Partial<{ name: string; nickname: string; whatsapp: string; password: string; role: string; active: boolean }>) =>
    patch<Player>(`/players/${id}`, data),
  delete: (id: string) => del(`/players/${id}`),
  resetPassword: (id: string) => post<{ temp_password: string }>(`/players/${id}/reset-password`),
  myStats: () => get<{ minutes_played: number; platform_minutes_played?: number; platform_total_matches?: number }>('/players/me/stats'),
  myFullStats: () => get<PlayerFullStats>('/players/me/stats/full'),
  signupStats: (limit = 30) => get<SignupStats>(`/players/signups/stats?limit=${limit}`),
};

// ── Groups ────────────────────────────────────────────────────
export type PlayerStatItem = {
  player_id: string;
  display_name: string;
  vote_points: number;
  flop_votes: number;
  minutes_played: number;
};
export type GroupStatsResponse = { players: PlayerStatItem[]; period_label: string };

export const groups = {
  list: () => get<Group[]>('/groups'),
  get: (id: string) => get<GroupDetail>(`/groups/${id}`),
  create: (data: { name: string; description?: string; slug?: string; vote_open_delay_minutes?: number; vote_duration_hours?: number }) => post<Group>('/groups', data),
  update: (id: string, data: { name?: string; description?: string; per_match_amount?: number | null; monthly_amount?: number | null; recurrence_enabled?: boolean; vote_open_delay_minutes?: number; vote_duration_hours?: number }) => patch<Group>(`/groups/${id}`, data),
  delete: (id: string) => del(`/groups/${id}`),
  addMember: (groupId: string, playerId: string, role = 'member') =>
    post<GroupMember>(`/groups/${groupId}/members`, { player_id: playerId, role }),
  removeMember: (groupId: string, playerId: string) => del(`/groups/${groupId}/members/${playerId}`),
  updateMemberRole: (groupId: string, playerId: string, role: string) =>
    patch<GroupMember>(`/groups/${groupId}/members/${playerId}`, { role }),
  updateMemberSkill: (groupId: string, playerId: string, data: { skill_stars?: number; is_goalkeeper?: boolean }) =>
    patch<GroupMember>(`/groups/${groupId}/members/${playerId}`, data),
  getStats: (id: string, params?: { period?: string; month?: string }) => {
    const q = new URLSearchParams();
    if (params?.period) q.set('period', params.period);
    if (params?.month)  q.set('month',  params.month);
    const qs = q.toString();
    return get<GroupStatsResponse>(`/groups/${id}/stats${qs ? '?' + qs : ''}`);
  },
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

// ── Push notifications ────────────────────────────────────────
export const push = {
  getVapidPublicKey: () => get<{ public_key: string }>('/push/vapid-public-key'),
  subscribe: (subscription: PushSubscriptionJSON, userAgent?: string) =>
    post<{ status: string }>('/push/subscribe', {
      endpoint: subscription.endpoint,
      keys: subscription.keys,
      user_agent: userAgent,
    }),
  unsubscribe: () => del('/push/subscribe'),
};

// ── Subscriptions ─────────────────────────────────────────────
export type SubscriptionInfo = {
  plan: string;
  groups_limit: number | null;
  groups_used: number;
  members_limit: number | null;
};

export const subscriptions = {
  me: () => get<SubscriptionInfo>('/subscriptions/me'),
};

// ── Votes ─────────────────────────────────────────────────────
export type VoteStatusResponse = {
  status: 'not_open' | 'open' | 'closed';
  opens_at: string;
  closes_at: string;
  voter_count: number;
  eligible_count: number;
  current_player_voted: boolean;
  time_label: string;
  voted_player_ids: string[];
  vote_open_delay_minutes: number;
};

export type VoteTop5ResultItem = { position: number; player_id: string; name: string; points: number };
export type VoteFlopResultItem = { player_id: string; name: string; votes: number };
export type VoteResultsResponse = {
  top5: VoteTop5ResultItem[];
  flop: VoteFlopResultItem[];
  total_voters: number;
  eligible_voters: number;
};

export type VotePendingItem = {
  match_id: string;
  match_hash: string;
  match_number: number;
  group_name: string;
  time_label: string;
  voter_count: number;
  eligible_count: number;
};

export const votes = {
  getStatus:  (matchId: string) => get<VoteStatusResponse>(`/matches/${matchId}/votes/status`),
  getResults: (matchId: string) => get<VoteResultsResponse>(`/matches/${matchId}/votes/results`),
  submit: (matchId: string, top5: { player_id: string; position: number }[], flop_player_id?: string | null) =>
    post<{ message: string }>(`/matches/${matchId}/votes`, { top5, flop_player_id }),
  getPending: () => get<{ items: VotePendingItem[] }>('/votes/pending'),
};

// ── Reviews ───────────────────────────────────────────────────
export type ReviewResponse = {
  id: string;
  rating: number;
  comment: string | null;
  created_at: string;
  updated_at: string;
};

export type ReviewSummaryResponse = {
  average: number;
  total: number;
  distribution: Record<string, { count: number; percent: number }>;
};

export type ReviewAdminItem = {
  id: string;
  player_id: string;
  player_name: string;
  rating: number;
  comment: string | null;
  created_at: string;
  updated_at: string;
};

export type ReviewListResponse = {
  items: ReviewAdminItem[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
};

const put = <T>(path: string, body?: unknown) => request<T>(path, { method: 'PUT', body: JSON.stringify(body) });

export const reviews = {
  getMe: () => get<ReviewResponse>('/reviews/me'),
  upsert: (rating: number, comment?: string | null) =>
    put<ReviewResponse>('/reviews/me', { rating, comment }),
  summary: () => get<ReviewSummaryResponse>('/reviews/summary'),
  list: (params?: { rating?: string; order_by?: string; page?: number; page_size?: number }) => {
    const q = new URLSearchParams();
    if (params?.rating)    q.set('rating', params.rating);
    if (params?.order_by)  q.set('order_by', params.order_by);
    if (params?.page)      q.set('page', String(params.page));
    if (params?.page_size) q.set('page_size', String(params.page_size));
    const qs = q.toString();
    return get<ReviewListResponse>(`/reviews${qs ? '?' + qs : ''}`);
  },
};

// ── Admin ─────────────────────────────────────────────────────
export type AdminStatsResponse = {
  total_matches: number;
  total_groups: number;
  total_players: number;
  platform_minutes_played: number;
  signups_total: number;
  signups_last_7_days: number;
  signups_last_30_days: number;
  total_reviews: number;
};

export type AdminMatchItem = {
  id: string;
  hash: string;
  number: number;
  group_id: string;
  group_name: string;
  match_date: string;
  start_time: string;
  end_time: string | null;
  location: string;
  status: 'open' | 'in_progress' | 'closed';
};

export type AdminMatchListResponse = { total: number; items: AdminMatchItem[] };

export type AdminGroupItem = {
  id: string;
  name: string;
  description: string | null;
  slug: string;
  total_members: number;
  total_matches: number;
  created_at: string;
};

export type AdminGroupListResponse = { total: number; items: AdminGroupItem[] };

export const admin = {
  getStats: () => get<AdminStatsResponse>('/admin/stats'),
  getMatches: (params?: { status?: string; limit?: number; offset?: number }) => {
    const q = new URLSearchParams();
    if (params?.status) q.set('status', params.status);
    if (params?.limit)  q.set('limit',  String(params.limit));
    if (params?.offset) q.set('offset', String(params.offset));
    const qs = q.toString();
    return get<AdminMatchListResponse>(`/admin/matches${qs ? '?' + qs : ''}`);
  },
  getGroups: (params?: { limit?: number; offset?: number }) => {
    const q = new URLSearchParams();
    if (params?.limit)  q.set('limit',  String(params.limit));
    if (params?.offset) q.set('offset', String(params.offset));
    const qs = q.toString();
    return get<AdminGroupListResponse>(`/admin/groups${qs ? '?' + qs : ''}`);
  },
};

// ── Teams ─────────────────────────────────────────────────────
export type TeamPlayerItem = {
  player_id: string;
  name: string;
  nickname: string | null;
  skill_stars: number;
  is_goalkeeper: boolean;
};

export type TeamItem = {
  id: string;
  name: string;
  color: string | null;
  position: number;
  skill_total: number;
  players: TeamPlayerItem[];
};

export type TeamsResponse = {
  teams: TeamItem[];
  reserves: TeamPlayerItem[];
};

export const teams = {
  generate: (matchId: string) =>
    post<TeamsResponse>(`/matches/${matchId}/teams`),
  get: (matchId: string) =>
    get<TeamsResponse>(`/matches/${matchId}/teams`),
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
