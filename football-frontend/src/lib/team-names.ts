/** Banco de nomes de times estilo várzea brasileira (≥ 40 nomes). */
export const TEAM_NAMES: string[] = [
  // Clássicos (do PRD 012)
  'Real Madruga',
  'Barcelusa',
  'Barsemlona',
  'Meia Boca Juniors',
  'Baile de Munique',
  'Varmeiras',
  'Atecubanos FC',
  'Inter de Limão',
  'Manchester Cachaça',
  'Real Matismo',
  'Paysanduba',
  'Horriver Plate',
  'Patético de Madrid',
  'Shakhtar dos Leks',
  'Espressinho da Mooca',
  // Novos (PRD 033)
  'Leões do Asfalto',
  'Tubarões da Várzea',
  'Dínamo de Boteco',
  'Garotos do Fundão',
  'Titãs do Campo Sujo',
  'Unidos do Barro',
  'Forja FC',
  'Dragões da Periferia',
  'Guerreiros do Baldão',
  'Estrelas do Zé',
  'Cansados do Joelho',
  'Cruzeiro do Bairro',
  'Porto Suado',
  'Benficado',
  'Raio que o Parta FC',
  'Seleção do Cervejinho',
  'Os Pesados',
  'Amigos do Couro',
  'Nacional do Piscinão',
  'Herói do Banco',
  'Clube dos Zueiros',
  'Inter de Fumaça',
  'Rala e Corre FC',
  'Campeões da Arquibancada',
  'Galera do Fundão',
];

/** Retorna os nomes embaralhados para uso em um sorteio. */
export function shuffledNames(): string[] {
  const a = [...TEAM_NAMES];
  for (let i = a.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    [a[i], a[j]] = [a[j], a[i]];
  }
  return a;
}
