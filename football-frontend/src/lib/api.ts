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
    if (res.status === 401 && typeof window !== 'undefined' && localStorage.getItem('player')) {
      const { sessionExpiredStore } = await import('$lib/stores/sessionExpired');
      sessionExpiredStore.set(true);
    }
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
    post<{ access_token: string; player_id: string; name: string; nickname: string | null; role: string; must_change_password: boolean; avatar_url: string | null }>('/auth/login', { whatsapp, password }),
  sendOtp: (whatsapp: string) =>
    post<{ status: string; expires_in_seconds: number }>('/auth/send-otp', { whatsapp }),
  verifyOtp: (whatsapp: string, otp_code: string) =>
    post<{ otp_token: string }>('/auth/verify-otp', { whatsapp, otp_code }),
  register: (data: { name: string; whatsapp: string; password: string; nickname?: string; otp_token: string }) =>
    post<{ access_token: string; player_id: string; name: string; nickname: string | null; role: string; must_change_password: boolean; avatar_url: string | null }>('/auth/register', data),
  me: () => get<Player>('/auth/me'),
  forgotPasswordSendOtp: (whatsapp: string) =>
    post<{ status: string; expires_in_seconds: number }>('/auth/forgot-password/send-otp', { whatsapp }),
  forgotPasswordVerifyOtp: (whatsapp: string, otp_code: string) =>
    post<{ otp_token: string }>('/auth/forgot-password/verify-otp', { whatsapp, otp_code }),
  forgotPasswordReset: (whatsapp: string, otp_token: string, new_password: string) =>
    post<void>('/auth/forgot-password/reset', { whatsapp, otp_token, new_password }),
  sendOtpMe: () =>
    post<{ status: string; expires_in_seconds: number }>('/auth/send-otp/me', {}),
  verifyOtpMe: (otp_code: string) =>
    post<{ otp_token: string }>('/auth/verify-otp/me', { otp_code }),
  changePassword: (new_password: string, opts: { current_password?: string; otp_token?: string }) =>
    post<void>('/auth/change-password', { new_password, ...opts }),
};

// ── Types ─────────────────────────────────────────────────────
export type Player = {
  id: string; name: string; nickname: string | null;
  whatsapp: string; role: 'admin' | 'player'; active: boolean;
  must_change_password: boolean;
  avatar_url: string | null;
  created_at: string; updated_at: string;
};
export type PlayerPublic = { id: string; name: string; nickname: string | null; role: string; avatar_url: string | null };
export type PlayerMemberView = PlayerPublic & { whatsapp: string };
export type Group = { id: string; name: string; description: string | null; slug: string; per_match_amount: number | null; monthly_amount: number | null; recurrence_enabled: boolean; is_public: boolean; vote_open_delay_minutes: number; vote_duration_hours: number; timezone: string; created_at: string; updated_at: string };
export type GroupMember = { id: string; player: PlayerMemberView; role: 'admin' | 'member'; skill_stars: number | null; is_goalkeeper: boolean | null; created_at: string };
export type GroupDetail = Group & { members: GroupMember[]; total_members: number };
export type WaitlistEntry = { id: string; match_id: string; player_id: string; player_name: string; player_nickname: string | null; intro: string | null; status: 'pending' | 'accepted' | 'rejected'; created_at: string };
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
export type MatchDetail = Match & { attendances: Attendance[]; confirmed_count: number; declined_count: number; pending_count: number; group_name: string; group_per_match_amount: number | null; group_monthly_amount: number | null; group_is_public: boolean; group_timezone: string };
export type PlayerMatchItem = Match & { group_name: string; my_attendance: 'confirmed' | 'declined' | 'pending' | null; group_timezone: string };
export type DiscoverMatch = Match & { group_name: string; confirmed_count: number; spots_left: number | null; group_timezone: string };

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
  myMatches: () => get<PlayerMatchItem[]>('/players/me/matches'),
  signupStats: (limit = 30) => get<SignupStats>(`/players/signups/stats?limit=${limit}`),
  uploadAvatar: (file: File): Promise<Player> => {
    const token = typeof localStorage !== 'undefined' ? localStorage.getItem('token') : null;
    const form = new FormData();
    form.append('file', file);
    return fetch(`${API_BASE}/players/me/avatar`, {
      method: 'PUT',
      headers: token ? { Authorization: `Bearer ${token}` } : {},
      body: form,
    }).then(async res => {
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new ApiError(res.status, body?.detail ?? `Erro ${res.status}`);
      }
      return res.json();
    });
  },
  removeAvatar: () => request<Player>('/players/me/avatar', { method: 'DELETE' }),
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
  create: (data: { name: string; description?: string; slug?: string; is_public?: boolean; vote_open_delay_minutes?: number; vote_duration_hours?: number; timezone?: string }) => post<Group>('/groups', data),
  update: (id: string, data: { name?: string; description?: string; per_match_amount?: number | null; monthly_amount?: number | null; recurrence_enabled?: boolean; is_public?: boolean; vote_open_delay_minutes?: number; vote_duration_hours?: number; timezone?: string }) => patch<Group>(`/groups/${id}`, data),
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
  joinWaitlist: (groupId: string, data: { agreed: boolean; intro?: string }) =>
    post<WaitlistEntry>(`/groups/${groupId}/waitlist`, data),
  getWaitlist: (groupId: string) => get<WaitlistEntry[]>(`/groups/${groupId}/waitlist`),
  getMyWaitlistEntry: (groupId: string) => get<WaitlistEntry | null>(`/groups/${groupId}/waitlist/me`),
  reviewWaitlist: (groupId: string, entryId: string, action: 'accept' | 'reject') =>
    patch<WaitlistEntry>(`/groups/${groupId}/waitlist/${entryId}`, { action }),
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
  discover: (params?: { date_from?: string; date_to?: string; court_type?: string[]; weekday?: number[]; limit?: number; offset?: number }) => {
    const q = new URLSearchParams();
    if (params?.date_from) q.set('date_from', params.date_from);
    if (params?.date_to)   q.set('date_to',   params.date_to);
    params?.court_type?.forEach(ct => q.append('court_type', ct));
    params?.weekday?.forEach(d => q.append('weekday', String(d)));
    if (params?.limit  != null) q.set('limit',  String(params.limit));
    if (params?.offset != null) q.set('offset', String(params.offset));
    const qs = q.toString();
    return get<DiscoverMatch[]>(`/matches/discover${qs ? '?' + qs : ''}`);
  },
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
  status: string;
  groups_limit: number | null;
  groups_used: number;
  members_limit: number | null;
  gateway_customer_id: string | null;
  gateway_sub_id: string | null;
  current_period_end: string | null;
  grace_period_end: string | null;
};

export const subscriptions = {
  me: () => get<SubscriptionInfo>('/subscriptions/me'),
  createCheckout: (plan: string, billing_cycle: 'monthly' | 'yearly') =>
    post<{ checkout_url: string }>('/subscriptions', { plan, billing_cycle }),
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

export type VoteTop5ResultItem = { position: number; player_id: string; name: string; nickname: string | null; points: number };
export type VoteFlopResultItem = { player_id: string; name: string; nickname: string | null; votes: number };
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
  getPublicResults: (hash: string) => get<VoteResultsResponse>(`/matches/public/${hash}/votes/results`),
  submit: (matchId: string, top5: { player_id: string; position: number }[], flop_player_id?: string | null) =>
    post<{ message: string }>(`/matches/${matchId}/votes`, { top5, flop_player_id }),
  getPending: () => get<{ items: VotePendingItem[] }>('/votes/pending'),
  closeEarly: (matchId: string) => post<{ message: string }>(`/matches/${matchId}/votes/close`, {}),
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

export type AdminPlayerItem = {
  id: string;
  name: string;
  nickname: string | null;
  whatsapp: string;
  role: string;
  active: boolean;
  created_at: string;
  plan: string;
  total_groups: number;
  avatar_url: string | null;
};

export type AdminPlayerListResponse = {
  total: number;
  page: number;
  page_size: number;
  items: AdminPlayerItem[];
};

export const admin = {
  getStats: () => get<AdminStatsResponse>('/admin/stats'),
  getPlayers: (params?: { search?: string; page?: number; page_size?: number }) => {
    const q = new URLSearchParams();
    if (params?.search)    q.set('search',    params.search);
    if (params?.page)      q.set('page',      String(params.page));
    if (params?.page_size) q.set('page_size', String(params.page_size));
    const qs = q.toString();
    return get<AdminPlayerListResponse>(`/admin/players${qs ? '?' + qs : ''}`);
  },
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
  getSubscriptionSummary: () => get<AdminSubscriptionSummary>('/admin/subscriptions/summary'),
  getSubscriptions: (params?: { status?: string; plan?: string; page?: number; page_size?: number }) => {
    const q = new URLSearchParams();
    if (params?.status)    q.set('status',    params.status);
    if (params?.plan)      q.set('plan',      params.plan);
    if (params?.page)      q.set('page',      String(params.page));
    if (params?.page_size) q.set('page_size', String(params.page_size));
    const qs = q.toString();
    return get<AdminSubscriptionListResponse>(`/admin/subscriptions${qs ? '?' + qs : ''}`);
  },
  updateSubscription: (playerId: string, data: { plan: string; status: string; billing_cycle: string; reason: string }) =>
    patch<{ status: string; plan: string }>(`/admin/subscriptions/${playerId}`, data),
  cancelSubscription: (playerId: string) =>
    post<{ status: string }>(`/admin/subscriptions/${playerId}/cancel`, {}),
  removePlayerAvatar: (playerId: string) =>
    del(`/admin/players/${playerId}/avatar`),
};

export type AdminSubscriptionBreakdownItem = { plan: string; billing_cycle: string; count: number };

export type AdminSubscriptionSummary = {
  total_players: number;
  active: number;
  free: number;
  past_due: number;
  canceled: number;
  mrr_cents: number;
  breakdown: AdminSubscriptionBreakdownItem[];
};

export type AdminSubscriptionItem = {
  player_id: string;
  player_name: string;
  plan: string;
  billing_cycle: string;
  status: string;
  current_period_end: string | null;
  grace_period_end: string | null;
  gateway_customer_id: string | null;
  gateway_sub_id: string | null;
  created_at: string;
};

export type AdminSubscriptionListResponse = {
  total: number;
  page: number;
  page_size: number;
  items: AdminSubscriptionItem[];
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

// ── Finance ───────────────────────────────────────────────────
export type FinancePayment = {
  id: string;
  player_id: string;
  player_name: string;
  payment_type: 'monthly' | 'per_match' | null;
  amount_due: number | null;
  status: 'pending' | 'paid' | 'excluded';
  paid_at: string | null;
};

export type FinanceSummary = {
  received_cents: number;
  pending_count: number;
  paid_count: number;
  total_members: number;
  compliance_pct: number;
};

export type FinancePeriod = {
  period_id: string;
  year: number;
  month: number;
  summary: FinanceSummary;
  payments: FinancePayment[];
};

export type FinancePeriodItem = { id: string; year: number; month: number };

export const finance = {
  getPeriod: (groupId: string, year: number, month: number) =>
    get<FinancePeriod>(`/groups/${groupId}/finance/periods/${year}/${month}`),
  listPeriods: (groupId: string) =>
    get<FinancePeriodItem[]>(`/groups/${groupId}/finance/periods`),
  markPayment: (paymentId: string, data: { status: 'paid' | 'pending'; payment_type?: 'monthly' | 'per_match' }) =>
    patch<FinancePayment>(`/finance/payments/${paymentId}`, data),
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
