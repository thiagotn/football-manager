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
    name: 'Grátis',
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
      'Sorteio automático de times',
      'Votação pós-partida (Top 5 + Decepção)',
      'Estatísticas pessoais por jogador',
    ],
    available: true,
  },
  basic: {
    key: 'basic',
    name: 'Básico',
    price_monthly: 1990,   // centavos → R$ 19,90
    price_yearly: 19900,   // centavos → R$ 199,00 (~2 meses grátis)
    groups: 3,
    members: 50,
    matches_open: -1,
    history_days: 180,
    highlights: [
      'Até 3 grupos ativos',
      'Até 50 jogadores por grupo',
      'Partidas ilimitadas',
      'Histórico de 6 meses',
      'Convites por link e QR Code',
      'Confirmação de presença',
      'Sorteio automático de times',
      'Votação pós-partida (Top 5 + Decepção)',
      'Estatísticas pessoais por jogador',
      'Suporte por e-mail',
    ],
    available: true,
  },
  pro: {
    key: 'pro',
    name: 'Pro',
    price_monthly: 3990,   // centavos → R$ 39,90
    price_yearly: 39900,   // centavos → R$ 399,00 (~2 meses grátis)
    groups: 10,
    members: -1,
    matches_open: -1,
    history_days: -1,
    highlights: [
      'Até 10 grupos ativos',
      'Jogadores ilimitados por grupo',
      'Partidas ilimitadas',
      'Histórico ilimitado',
      'Convites por link e QR Code',
      'Confirmação de presença',
      'Sorteio automático de times',
      'Votação pós-partida (Top 5 + Decepção)',
      'Estatísticas pessoais por jogador',
      'Suporte prioritário',
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

export function formatPrice(plan: PlanConfig, cycle: 'monthly' | 'yearly' = 'monthly'): string {
  if (plan.price_monthly === null) return 'Grátis';
  const cents = cycle === 'yearly' ? (plan.price_yearly ?? plan.price_monthly * 10) : plan.price_monthly;
  return `${formatCents(cents)}${cycle === 'yearly' ? '/ano' : '/mês'}`;
}
