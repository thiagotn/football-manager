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
    name: 'Free',
    price_monthly: null,
    price_yearly: null,
    groups: 1,
    members: 30,
    matches_open: 3,
    history_days: 30,
    highlights: [
      '1 grupo ativo',
      'Até 30 jogadores por grupo',
      'Até 3 partidas abertas',
      'Histórico de 30 dias',
      'Convites por link e QR Code',
      'Confirmação de presença',
      'Votação pós-partida (Top 5 + Decepção)',
    ],
    available: true,
  },
  basic: {
    key: 'basic',
    name: 'Básico',
    price_monthly: null,
    price_yearly: null,
    groups: 3,
    members: 50,
    matches_open: -1,
    history_days: 180,
    highlights: [
      'Até 3 grupos ativos',
      'Até 50 jogadores por grupo',
      'Partidas ilimitadas',
      'Histórico de 6 meses',
      'Votação pós-partida (Top 5 + Decepção)',
      'Suporte por e-mail',
    ],
    available: false,
  },
  pro: {
    key: 'pro',
    name: 'Pro',
    price_monthly: null,
    price_yearly: null,
    groups: 10,
    members: -1,
    matches_open: -1,
    history_days: -1,
    highlights: [
      'Até 10 grupos ativos',
      'Jogadores ilimitados por grupo',
      'Partidas ilimitadas',
      'Histórico ilimitado',
      'Votação pós-partida (Top 5 + Decepção)',
      'Suporte prioritário',
    ],
    available: false,
  },
};

export const PLAN_ORDER: PlanKey[] = ['free', 'basic', 'pro'];

export function getPlan(key: string): PlanConfig {
  return PLANS[key as PlanKey] ?? PLANS.free;
}

export function formatPrice(plan: PlanConfig): string {
  if (plan.price_monthly === null) return 'Grátis';
  return `R$ ${plan.price_monthly.toFixed(2).replace('.', ',')}/mês`;
}
