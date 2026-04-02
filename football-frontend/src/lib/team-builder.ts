export type Position = 'goalkeeper' | 'defender' | 'fullback' | 'midfielder' | 'forward';

export interface DrawPlayer {
  id: string;
  name: string;
  nickname: string;
  stars: number;
  position: Position;
  active: boolean;
}

export interface TeamPlayer extends DrawPlayer {
  isGkSlot: boolean;
}

export interface Team {
  name: string;
  color: string;
  players: TeamPlayer[];
  totalStars: number;
  hasGoalkeeper: boolean;
}

export interface TeamResult {
  teams: Team[];
  reserves: DrawPlayer[];
}

export const TEAM_COLORS = [
  '#e63946', '#2a9d8f', '#e9c46a', '#f4a261',
  '#6a4c93', '#1982c4', '#8ac926', '#ff6b35',
];

export const POS_ABBR: Record<Position, string> = {
  goalkeeper: 'GK',
  defender:   'ZAG',
  fullback:   'LAT',
  midfielder: 'MEI',
  forward:    'ATA',
};

export const POS_COLOR_CLASSES: Record<Position, string> = {
  goalkeeper: 'bg-amber-400/20 text-amber-300',
  defender:   'bg-blue-400/20 text-blue-300',
  fullback:   'bg-cyan-400/20 text-cyan-300',
  midfielder: 'bg-emerald-400/20 text-emerald-300',
  forward:    'bg-red-400/20 text-red-300',
};

function shuffle<T>(arr: T[]): T[] {
  const a = [...arr];
  for (let i = a.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    [a[i], a[j]] = [a[j], a[i]];
  }
  return a;
}

function buildSnakeCycle(nTeams: number): number[] {
  const fwd = Array.from({ length: nTeams }, (_, i) => i);
  return [...fwd, ...[...fwd].reverse()];
}

/**
 * Builds balanced teams from a list of active players.
 *
 * Algorithm:
 * 1. Assign the best GKs to teams (1 per team) via snake draft by stars.
 * 2. Pool remaining players (field + excess GKs) sorted by stars desc.
 * 3. Snake draft the pool into team field slots — guarantees equal distribution.
 * 4. Players beyond total slots become reserves.
 */
export function buildTeams(
  activePlayers: DrawPlayer[],
  playersPerTeam: number,
  nTeams: number,
  teamNames: string[],
): TeamResult {
  const teamSize = playersPerTeam + 1;

  // Shuffle first so equal-star players are randomly ordered across runs
  const shuffled = shuffle(activePlayers);

  const gks = shuffled
    .filter(p => p.position === 'goalkeeper')
    .sort((a, b) => b.stars - a.stars);
  const nonGks = shuffled
    .filter(p => p.position !== 'goalkeeper')
    .sort((a, b) => b.stars - a.stars);

  const assignedGks = gks.slice(0, nTeams);
  const excessGks   = gks.slice(nTeams).sort((a, b) => b.stars - a.stars);

  // Initialize teams with their assigned GK (if any)
  const teams: Team[] = Array.from({ length: nTeams }, (_, i) => ({
    name:          teamNames[i] ?? `Time ${i + 1}`,
    color:         TEAM_COLORS[i % TEAM_COLORS.length],
    players:       assignedGks[i] ? [{ ...assignedGks[i], isGkSlot: true }] : [],
    totalStars:    assignedGks[i]?.stars ?? 0,
    hasGoalkeeper: !!assignedGks[i],
  }));

  // Field pool: non-GKs + demoted excess GKs, sorted by stars desc
  const fieldPool = [...nonGks, ...excessGks].sort((a, b) => b.stars - a.stars);

  // How many field slots each team still needs
  const needs       = teams.map(t => teamSize - t.players.length);
  const totalNeeded = needs.reduce((s, n) => s + n, 0);

  const forDist = fieldPool.slice(0, totalNeeded);
  const reserves = fieldPool.slice(totalNeeded);

  // Snake draft respecting per-team capacity
  const snake     = buildSnakeCycle(nTeams);
  const remaining = [...needs];
  let si = 0;

  for (const player of forDist) {
    // Advance past full teams (safety: max nTeams*2 skips)
    let skips = 0;
    while (remaining[snake[si % snake.length]] === 0 && skips < nTeams * 2) {
      si++;
      skips++;
    }
    const ti = snake[si % snake.length];
    teams[ti].players.push({ ...player, isGkSlot: false });
    teams[ti].totalStars += player.stars;
    remaining[ti]--;
    si++;
  }

  return { teams, reserves };
}
