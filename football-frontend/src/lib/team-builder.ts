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

// Display order: GK → LAT → ZAG → MEI → ATA
export const POS_ORDER: Record<Position, number> = {
  goalkeeper: 0,
  fullback:   1,
  defender:   2,
  midfielder: 3,
  forward:    4,
};

export function sortPlayersByPosition<T extends { position: Position }>(players: T[]): T[] {
  return [...players].sort((a, b) => POS_ORDER[a.position] - POS_ORDER[b.position]);
}

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
 * 1. Assign the best GKs (1 per team) via snake draft by stars.
 * 2. For each field position (ZAG, LAT, MEI, ATA): distribute floor(count/nTeams)
 *    players per team via snake draft — guarantees equal positional distribution.
 *    The snake index continues globally so the direction alternates correctly
 *    across position groups, keeping star totals balanced.
 * 3. Overflow (position remainders + excess GKs) fills remaining slots via snake.
 * 4. Players beyond total capacity become reserves.
 */
export function buildTeams(
  activePlayers: DrawPlayer[],
  playersPerTeam: number,
  nTeams: number,
  teamNames: string[],
): TeamResult {
  if (nTeams === 0) return { teams: [], reserves: [] };

  const teamSize   = playersPerTeam + 1;
  const snakeCycle = buildSnakeCycle(nTeams);

  // Shuffle so equal-star players are randomly ordered across runs
  const shuffled = shuffle(activePlayers);

  // --- Step 1: GKs — 1 per team, snake draft by stars desc ---
  const gks = shuffled
    .filter(p => p.position === 'goalkeeper')
    .sort((a, b) => b.stars - a.stars);

  const assignedGks = gks.slice(0, nTeams);
  const excessGks   = gks.slice(nTeams);

  const teams: Team[] = Array.from({ length: nTeams }, (_, i) => ({
    name:          teamNames[i] ?? `Time ${i + 1}`,
    color:         TEAM_COLORS[i % TEAM_COLORS.length],
    players:       [],
    totalStars:    0,
    hasGoalkeeper: false,
  }));

  // Global snake index — kept across all distribution steps for correct alternation
  let si = 0;

  for (const gk of assignedGks) {
    const ti = snakeCycle[si % snakeCycle.length];
    teams[ti].players.push({ ...gk, isGkSlot: true });
    teams[ti].totalStars += gk.stars;
    teams[ti].hasGoalkeeper = true;
    si++;
  }

  // --- Step 2: Field positions — equal distribution per position ---
  const fieldPositions: Position[] = ['defender', 'fullback', 'midfielder', 'forward'];
  const overflow: DrawPlayer[] = [...excessGks].sort((a, b) => b.stars - a.stars);

  for (const pos of fieldPositions) {
    const group = shuffled
      .filter(p => p.position === pos)
      .sort((a, b) => b.stars - a.stars);

    // Cap perTeam so we never exceed any team's remaining field capacity
    const minRemaining = Math.min(...teams.map(t => teamSize - t.players.length));
    const perTeam      = Math.min(Math.floor(group.length / nTeams), minRemaining);

    // Remainder goes to overflow for step 3
    overflow.push(...group.slice(perTeam * nTeams));

    if (perTeam === 0) continue;

    // Snake draft (continuing global si) — alternates direction across groups,
    // which compensates for the GK star spread and keeps totals balanced
    const toDistribute = group.slice(0, perTeam * nTeams);
    for (const player of toDistribute) {
      const ti = snakeCycle[si % snakeCycle.length];
      teams[ti].players.push({ ...player, isGkSlot: false });
      teams[ti].totalStars += player.stars;
      si++;
    }
  }

  // --- Step 3: Overflow fills remaining slots via snake ---
  const remaining   = teams.map(t => teamSize - t.players.length);
  const totalNeeded = remaining.reduce((s, n) => s + n, 0);

  overflow.sort((a, b) => b.stars - a.stars);
  const forDist  = overflow.slice(0, totalNeeded);
  const reserves = overflow.slice(totalNeeded);

  for (const player of forDist) {
    // Advance past full teams (safety: max nTeams*2 skips)
    let skips = 0;
    while (remaining[snakeCycle[si % snakeCycle.length]] === 0 && skips < nTeams * 2) {
      si++;
      skips++;
    }
    const ti = snakeCycle[si % snakeCycle.length];
    teams[ti].players.push({ ...player, isGkSlot: false });
    teams[ti].totalStars += player.stars;
    remaining[ti]--;
    si++;
  }

  return { teams, reserves };
}
