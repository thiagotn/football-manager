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

/**
 * Distributes `group` (sorted by stars desc) across `nTeams` teams using
 * shuffled-tier round-robin: takes groups of nTeams players (similar stars),
 * shuffles each group, then assigns one player to each team.
 *
 * This ensures similar-skill players go to different teams, but which team
 * gets which is determined by random draw — no systematic bias.
 *
 * Returns overflow (players beyond perTeam * nTeams).
 */
function assignTiers<T extends { stars: number }>(
  group: T[],
  perTeam: number,
  nTeams: number,
  teams: T[][],
): T[] {
  const toDist = group.slice(0, perTeam * nTeams);
  const overflow = group.slice(perTeam * nTeams);

  for (let round = 0; round < perTeam; round++) {
    const tier = shuffle(toDist.slice(round * nTeams, (round + 1) * nTeams));
    for (let i = 0; i < tier.length; i++) {
      teams[i].push(tier[i]);
    }
  }

  return overflow;
}

/**
 * Builds balanced teams from a list of active players.
 *
 * Algorithm — shuffled-tier round-robin per position:
 * 1. GKs: randomly shuffled and assigned 1 per team.
 * 2. Each field position (lat/zag/mei/ata): sorted by stars desc, split
 *    into tiers of nTeams players. Each tier is shuffled before assignment —
 *    similar-skill players go to different teams, but which team gets which
 *    is random (no systematic bias toward any team).
 * 3. Per-position cap: sum of perTeam values cannot exceed field slots per
 *    team, preventing teams from exceeding teamSize.
 * 4. Overflow fills remaining slots using the same shuffled-tier approach.
 * 5. Players beyond total capacity become reserves.
 */
export function buildTeams(
  activePlayers: DrawPlayer[],
  playersPerTeam: number,
  nTeams: number,
  teamNames: string[],
): TeamResult {
  if (nTeams === 0) return { teams: [], reserves: [] };

  const teamSize = playersPerTeam + 1;

  // Shuffle first so equal-star players have random order
  const shuffled = shuffle(activePlayers);

  // Separate by position, sort by stars desc within each group
  const byPos: Partial<Record<Position, DrawPlayer[]>> = {};
  for (const p of shuffled) {
    if (!byPos[p.position]) byPos[p.position] = [];
    byPos[p.position]!.push(p);
  }
  for (const pos of Object.keys(byPos) as Position[]) {
    byPos[pos]!.sort((a, b) => b.stars - a.stars);
  }

  const gks   = byPos['goalkeeper'] ?? [];
  const teamArrays: DrawPlayer[][] = Array.from({ length: nTeams }, () => []);
  const overflow: DrawPlayer[] = [];

  const teams: Team[] = Array.from({ length: nTeams }, (_, i) => ({
    name:          teamNames[i] ?? `Time ${i + 1}`,
    color:         TEAM_COLORS[i % TEAM_COLORS.length],
    players:       [],
    totalStars:    0,
    hasGoalkeeper: false,
  }));

  // Step 1: GKs — 1 per team, randomly assigned
  const gksForTeams = shuffle(gks.slice(0, nTeams));
  for (let i = 0; i < gksForTeams.length; i++) {
    teamArrays[i].push(gksForTeams[i]);
  }
  overflow.push(...gks.slice(nTeams));

  // Step 2: Compute perTeam for each position, capped to fit field slots
  const fieldSlots = teamSize - 1;
  const fieldPositions = ['fullback', 'defender', 'midfielder', 'forward'] as const;

  const perTeamMap: Record<string, number> = {};
  for (const pos of fieldPositions) {
    perTeamMap[pos] = Math.floor((byPos[pos as Position]?.length ?? 0) / nTeams);
  }

  // Reduce the most abundant position until total fits within fieldSlots
  let total = Object.values(perTeamMap).reduce((s, v) => s + v, 0);
  while (total > fieldSlots) {
    const maxPos = fieldPositions.reduce((a, b) =>
      perTeamMap[a] >= perTeamMap[b] ? a : b
    );
    perTeamMap[maxPos]--;
    total--;
  }

  // Step 3: Distribute each position using shuffled tiers
  for (const pos of fieldPositions) {
    const group = byPos[pos as Position] ?? [];
    const leftover = assignTiers(group, perTeamMap[pos], nTeams, teamArrays);
    overflow.push(...leftover);
  }

  // Step 4: Overflow fills remaining slots using shuffled tiers
  overflow.sort((a, b) => b.stars - a.stars);
  const remaining = teamArrays.map(t => teamSize - t.length);

  let idx = 0;
  while (idx < overflow.length) {
    const openTeams = teamArrays
      .map((_, i) => i)
      .filter(i => remaining[i] > 0);
    if (openTeams.length === 0) break;

    const batch = shuffle(overflow.slice(idx, idx + openTeams.length));
    for (let i = 0; i < batch.length; i++) {
      const ti = openTeams[i];
      teamArrays[ti].push(batch[i]);
      remaining[ti]--;
    }
    idx += batch.length;
  }

  const reserves = overflow.slice(idx);

  // Build final Team objects
  for (let i = 0; i < nTeams; i++) {
    for (const p of teamArrays[i]) {
      const isGk = p.position === 'goalkeeper';
      teams[i].players.push({ ...p, isGkSlot: isGk });
      teams[i].totalStars += p.stars;
      if (isGk) teams[i].hasGoalkeeper = true;
    }
  }

  return { teams, reserves };
}
