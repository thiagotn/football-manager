import type { DrawPlayer } from './team-builder';

type SeedPlayer = Omit<DrawPlayer, 'id'>;

export const SEED_PLAYERS: SeedPlayer[] = [
  // Goleiros (6)
  { name: 'José Carlos',     nickname: 'Zé Grilo',    stars: 5, position: 'goalkeeper', active: true },
  { name: 'Rodrigo Lima',    nickname: 'Pintado',     stars: 4, position: 'goalkeeper', active: true },
  { name: 'Roberto Cunha',   nickname: 'Burrinho',    stars: 3, position: 'goalkeeper', active: true },
  { name: 'Fábio Souza',     nickname: 'Gauchinho',   stars: 3, position: 'goalkeeper', active: true },
  { name: 'Nilson Pereira',  nickname: 'Nilsão',      stars: 2, position: 'goalkeeper', active: true },
  { name: 'Cláudio Ferraz',  nickname: 'Morcego',     stars: 1, position: 'goalkeeper', active: true },

  // Laterais (7)
  { name: 'Rafael Fino',    nickname: 'Fininho',   stars: 5, position: 'fullback', active: true },
  { name: 'Carlos Zinco',   nickname: 'Zinho',     stars: 4, position: 'fullback', active: true },
  { name: 'Márcio Nunes',   nickname: 'Mosquito',  stars: 3, position: 'fullback', active: true },
  { name: 'Bruno Moraes',   nickname: 'Bolinha',   stars: 3, position: 'fullback', active: true },
  { name: 'Evandro Costa',  nickname: 'Cabelinho', stars: 2, position: 'fullback', active: true },
  { name: 'Sandro Reis',    nickname: 'Sandrim',   stars: 2, position: 'fullback', active: true },
  { name: 'Ailton Figueiredo', nickname: 'Topete', stars: 1, position: 'fullback', active: true },

  // Zagueiros (10)
  { name: 'Alexandre Motta', nickname: 'Ferreirinha', stars: 5, position: 'defender', active: true },
  { name: 'Denis Barbosa',   nickname: 'Dentinho',    stars: 4, position: 'defender', active: true },
  { name: 'Sebastião Ramos', nickname: 'Tião',        stars: 4, position: 'defender', active: true },
  { name: 'Marcelo Assis',   nickname: 'Marcelão',    stars: 3, position: 'defender', active: true },
  { name: 'André Dias',      nickname: 'Dedé',        stars: 3, position: 'defender', active: true },
  { name: 'Valdir Santos',   nickname: 'Vadão',       stars: 3, position: 'defender', active: true },
  { name: 'Augusto Pires',   nickname: 'Guto',        stars: 2, position: 'defender', active: true },
  { name: 'Renato Faria',    nickname: 'Renatin',     stars: 2, position: 'defender', active: true },
  { name: 'Ivaldo Cruz',     nickname: 'Tigrão',      stars: 2, position: 'defender', active: true },
  { name: 'Robson Leal',     nickname: 'Robinho',     stars: 1, position: 'defender', active: true },

  // Meias (15)
  { name: 'Carlos Eduardo',  nickname: 'Carlão',     stars: 5, position: 'midfielder', active: true },
  { name: 'Rogério Silva',   nickname: 'Cabelo',     stars: 5, position: 'midfielder', active: true },
  { name: 'Felipe Mendes',   nickname: 'Pipoca',     stars: 4, position: 'midfielder', active: true },
  { name: 'Wagner Oliveira', nickname: 'Maestro',    stars: 4, position: 'midfielder', active: true },
  { name: 'Branco Santos',   nickname: 'Branquinho', stars: 4, position: 'midfielder', active: true },
  { name: 'Pedro Gomes',     nickname: 'Pelezinho',  stars: 3, position: 'midfielder', active: true },
  { name: 'Francisco Lima',  nickname: 'Índio',      stars: 3, position: 'midfielder', active: true },
  { name: 'Gustavo Torres',  nickname: 'Gato',       stars: 3, position: 'midfielder', active: true },
  { name: 'Leandro Vaz',     nickname: 'Leandrin',   stars: 3, position: 'midfielder', active: true },
  { name: 'Tarcísio Melo',   nickname: 'Tarcis',     stars: 3, position: 'midfielder', active: true },
  { name: 'Hermes Duarte',   nickname: 'Herminho',   stars: 2, position: 'midfielder', active: true },
  { name: 'Raimundo Luz',    nickname: 'Raimundão',  stars: 2, position: 'midfielder', active: true },
  { name: 'Odilon Campos',   nickname: 'Frajola',    stars: 2, position: 'midfielder', active: true },
  { name: 'Adilson Neves',   nickname: 'Didi',       stars: 1, position: 'midfielder', active: true },
  { name: 'Geraldo Brito',   nickname: 'Geraldin',   stars: 1, position: 'midfielder', active: true },

  // Atacantes (12)
  { name: 'Cássio Andrade',  nickname: 'Cascão',     stars: 5, position: 'forward', active: true },
  { name: 'Reinaldo Costa',  nickname: 'Alemão',     stars: 5, position: 'forward', active: true },
  { name: 'Magnus Freitas',  nickname: 'Magrão',     stars: 4, position: 'forward', active: true },
  { name: 'Paulo Pinto',     nickname: 'Pintinho',   stars: 4, position: 'forward', active: true },
  { name: 'Leonardo Cabral', nickname: 'Cabeludo',   stars: 3, position: 'forward', active: true },
  { name: 'Eduardo Rocha',   nickname: 'Perereca',   stars: 3, position: 'forward', active: true },
  { name: 'Mateus Ferreira', nickname: 'Ratão',      stars: 3, position: 'forward', active: true },
  { name: 'Jailson Ramos',   nickname: 'Jailsinho',  stars: 3, position: 'forward', active: true },
  { name: 'Fumácio Borges',  nickname: 'Fumacinha',  stars: 2, position: 'forward', active: true },
  { name: 'Clóvis Mendes',   nickname: 'Clovis',     stars: 2, position: 'forward', active: true },
  { name: 'Gilberto Alves',  nickname: 'Gilbertão',  stars: 2, position: 'forward', active: true },
  { name: 'Waldemar Fonseca',nickname: 'Waldinho',   stars: 1, position: 'forward', active: true },
];

export function seedWithIds(): DrawPlayer[] {
  return SEED_PLAYERS.map((p, i) => ({ ...p, id: String(i) }));
}
