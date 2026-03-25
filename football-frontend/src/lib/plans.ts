export type PlanKey = 'free' | 'basic' | 'pro';

export interface PlanConfig {
  key: PlanKey;
  name: string;
  price_monthly: number | null;   // null = grátis
  price_yearly: number | null;
  groups: number;                 // -1 = ilimitado
  members: number;                // -1 = ilimitado
  matches_open: number;           // -1 = ilimitado, partidas abertas por grupo
  history_days: number;           // -1 = ilimitado
  highlights: string[];           // bullets exibidos nos cards de plano
  available: boolean;             // false = "em breve"
}

export const PLANS: Record<PlanKey, PlanConfig> = {
  free: {
    key: 'free',
    name: 'plan.free.name',
    price_monthly: null,
    price_yearly: null,
    groups: 1,
    members: 30,
    matches_open: 3,
    history_days: 30,
    highlights: [
      'plan.free.highlight.groups',
      'plan.free.highlight.players',
      'plan.free.highlight.matches',
      'plan.free.highlight.history',
      'plan.highlight.invites',
      'plan.highlight.attendance',
      'plan.highlight.team_draw',
      'plan.highlight.voting',
      'plan.highlight.stats',
      'plan.free.highlight.finance',
    ],
    available: true,
  },
  basic: {
    key: 'basic',
    name: 'plan.basic.name',
    price_monthly: 1990,   // centavos → R$ 19,90
    price_yearly: 19900,   // centavos → R$ 199,00 (~2 meses grátis)
    groups: 3,
    members: 50,
    matches_open: -1,
    history_days: 180,
    highlights: [
      'plan.basic.highlight.groups',
      'plan.basic.highlight.players',
      'plan.basic.highlight.matches',
      'plan.basic.highlight.history',
      'plan.highlight.invites',
      'plan.highlight.attendance',
      'plan.highlight.team_draw',
      'plan.highlight.voting',
      'plan.highlight.stats',
      'plan.basic.highlight.finance',
      'plan.basic.highlight.support',
    ],
    available: true,
  },
  pro: {
    key: 'pro',
    name: 'plan.pro.name',
    price_monthly: 3990,   // centavos → R$ 39,90
    price_yearly: 39900,   // centavos → R$ 399,00 (~2 meses grátis)
    groups: 10,
    members: -1,
    matches_open: -1,
    history_days: -1,
    highlights: [
      'plan.pro.highlight.groups',
      'plan.pro.highlight.players',
      'plan.pro.highlight.matches',
      'plan.pro.highlight.history',
      'plan.highlight.invites',
      'plan.highlight.attendance',
      'plan.highlight.team_draw',
      'plan.highlight.voting',
      'plan.highlight.stats',
      'plan.pro.highlight.finance',
      'plan.pro.highlight.support',
    ],
    available: true,
  },
};

export const PLAN_ORDER: PlanKey[] = ['free', 'basic', 'pro'];

export function getPlan(key: string): PlanConfig {
  return PLANS[key as PlanKey] ?? PLANS.free;
}

/** Formata valor em centavos para exibição: 1990 → "R$ 19,90" */
export function formatCents(cents: number): string {
  return `R$ ${(cents / 100).toFixed(2).replace('.', ',')}`;
}

/**
 * @deprecated Pass a $t function and format in the template using formatCents + plans.per_month/per_year keys.
 * Kept for backward compatibility; hardcoded suffixes are in Portuguese.
 */
export function formatPrice(plan: PlanConfig, cycle: 'monthly' | 'yearly' = 'monthly'): string {
  if (plan.price_monthly === null) return 'Grátis';
  const cents = cycle === 'yearly' ? (plan.price_yearly ?? plan.price_monthly * 10) : plan.price_monthly;
  return `${formatCents(cents)}${cycle === 'yearly' ? '/ano' : '/mês'}`;
}
