import type { DrawPlayer } from './team-builder';

type SeedPlayer = Omit<DrawPlayer, 'id'>;

export const SEED_PLAYERS: SeedPlayer[] = [
  // Goleiros (4): distribuição de estrelas — 5, 4, 3, 2
  { name: 'José Carlos',    nickname: 'Zé Grilo',    stars: 5, position: 'goalkeeper', active: true },
  { name: 'Rodrigo Lima',   nickname: 'Pintado',     stars: 4, position: 'goalkeeper', active: true },
  { name: 'Roberto Cunha',  nickname: 'Burrinho',    stars: 3, position: 'goalkeeper', active: true },
  { name: 'Fábio Souza',    nickname: 'Gauchinho',   stars: 2, position: 'goalkeeper', active: true },

  // Zagueiros (6): 4, 4, 3, 3, 2, 1
  { name: 'Alexandre Motta', nickname: 'Ferreirinha', stars: 4, position: 'defender', active: true },
  { name: 'Denis Barbosa',   nickname: 'Dentinho',    stars: 4, position: 'defender', active: true },
  { name: 'Sebastião Ramos', nickname: 'Tião',        stars: 3, position: 'defender', active: true },
  { name: 'Marcelo Assis',   nickname: 'Marcelão',    stars: 3, position: 'defender', active: true },
  { name: 'André Dias',      nickname: 'Dedé',        stars: 2, position: 'defender', active: true },
  { name: 'Valdir Santos',   nickname: 'Vadão',       stars: 1, position: 'defender', active: true },

  // Laterais (4): 4, 3, 3, 2
  { name: 'Rafael Fino',  nickname: 'Fininho',  stars: 4, position: 'fullback', active: true },
  { name: 'Carlos Zinco', nickname: 'Zinho',    stars: 3, position: 'fullback', active: true },
  { name: 'Márcio Nunes', nickname: 'Mosquito', stars: 3, position: 'fullback', active: true },
  { name: 'Bruno Moraes', nickname: 'Bolinha',  stars: 2, position: 'fullback', active: true },

  // Meias (8): 5, 4, 4, 3, 3, 3, 2, 2
  { name: 'Carlos Eduardo',  nickname: 'Carlão',     stars: 5, position: 'midfielder', active: true },
  { name: 'Rogério Silva',   nickname: 'Cabelo',     stars: 4, position: 'midfielder', active: true },
  { name: 'Felipe Mendes',   nickname: 'Pipoca',     stars: 4, position: 'midfielder', active: true },
  { name: 'Wagner Oliveira', nickname: 'Maestro',    stars: 3, position: 'midfielder', active: true },
  { name: 'Branco Santos',   nickname: 'Branquinho', stars: 3, position: 'midfielder', active: true },
  { name: 'Pedro Gomes',     nickname: 'Pelezinho',  stars: 3, position: 'midfielder', active: true },
  { name: 'Francisco Lima',  nickname: 'Índio',      stars: 2, position: 'midfielder', active: true },
  { name: 'Gustavo Torres',  nickname: 'Gato',       stars: 2, position: 'midfielder', active: true },

  // Atacantes (8): 5, 4, 3, 3, 3, 3, 2, 1
  { name: 'Cássio Andrade',  nickname: 'Cascão',    stars: 5, position: 'forward', active: true },
  { name: 'Reinaldo Costa',  nickname: 'Alemão',    stars: 4, position: 'forward', active: true },
  { name: 'Magnus Freitas',  nickname: 'Magrão',    stars: 3, position: 'forward', active: true },
  { name: 'Paulo Pinto',     nickname: 'Pintinho',  stars: 3, position: 'forward', active: true },
  { name: 'Leonardo Cabral', nickname: 'Cabeludo',  stars: 3, position: 'forward', active: true },
  { name: 'Eduardo Rocha',   nickname: 'Perereca',  stars: 3, position: 'forward', active: true },
  { name: 'Mateus Ferreira', nickname: 'Ratão',     stars: 2, position: 'forward', active: true },
  { name: 'Fumácio Borges',  nickname: 'Fumacinha', stars: 1, position: 'forward', active: true },
];

export function seedWithIds(): DrawPlayer[] {
  return SEED_PLAYERS.map((p, i) => ({ ...p, id: String(i) }));
}
