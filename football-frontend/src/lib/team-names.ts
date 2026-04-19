/** Paleta de cores de coletes disponíveis para times do grupo. */
export type BibColor = { slug: string; label: string; hex: string };

export const BIB_COLOR_PALETTE: BibColor[] = [
  { slug: 'laranja',  label: 'Laranja',  hex: '#f97316' },
  { slug: 'azul',     label: 'Azul',     hex: '#3b82f6' },
  { slug: 'verde',    label: 'Verde',    hex: '#22c55e' },
  { slug: 'vermelho', label: 'Vermelho', hex: '#ef4444' },
  { slug: 'amarelo',  label: 'Amarelo',  hex: '#eab308' },
  { slug: 'preto',    label: 'Preto',    hex: '#1f2937' },
  { slug: 'branco',   label: 'Branco',   hex: '#f1f5f9' },
];

/** Banco de nomes de times estilo várzea brasileira (≥ 40 nomes). */
export const TEAM_NAMES: string[] = [
  // Clássicos (do PRD 012)
  "Real Madruga",
  "Barcelusa",
  "Barsemlona",
  "Meia Boca Juniors",
  "Baile de Munique",
  "Varmeiras",
  "Inter de Limão",
  "Manchester Cachaça",
  "Real Matismo",
  "Paysanduba",
  "Horriver Plate",
  "Patético de Madrid",
  "Shakhtar dos Leks",
  "Atlético Piseiro",
  "Mauá City",
  "Espressinho da Mooca",
  // Novos (PRD 033)
  "Leões do Asfalto",
  "Tubarões da Várzea",
  "Dínamo de Boteco",
  "Garotos do Fundão",
  "Titãs do Campo Sujo",
  "Unidos do Barro",
  "Forja FC",
  "Dragões da Periferia",
  "Guerreiros do Baldão",
  "Estrelas do Zé",
  "Cansados do Joelho",
  "Cruzeiro do Bairro",
  "Benficado",
  "Raio que o Parta FC",
  "Seleção do Cervejinho",
  "Os Pesados",
  "Nacional do Piscinão",
  "Herói do Banco",
  "Inter de Fumaça",
  "Rala e Corre FC",
  "Campeões da Arquibancada",
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
