/**
 * Quebra-Cuca — as cenas dos quebra-cabeças: 4 temas × 3 cenas.
 *
 * Cada cena é desenhada numa tela de `SCENE_W × SCENE_H`: céu em degradê,
 * chão com morrinhos e desenhos Twemoji por cima (`img` = código; arquivo em
 * `static/quebra-cuca/img/<img>.svg`, copiado por `npm run twemoji`). A ordem
 * de `items` é a ordem de pintura: o primeiro fica atrás.
 *
 * **Heróis são genéricos ou do dia a dia** (super-herói de capa, bombeiro,
 * ambulância) — personagens de marca (Marvel, DC…) são propriedade de outros.
 *
 * Nomes nas chaves i18n `quebra_cuca.theme.<tema>` e `quebra_cuca.scene.<id>`.
 */

export const SCENE_W = 400;
export const SCENE_H = 300;

export type ThemeId = 'dinos' | 'herois' | 'fazenda' | 'circo';

export type SceneItem = {
  img: string;
  /** Centro do desenho na tela da cena. */
  x: number;
  y: number;
  /** Lado do desenho (os SVGs da Twemoji são quadrados). */
  s: number;
  /** Espelha na horizontal (bicho olhando para o outro lado). */
  flip?: boolean;
};

export type Scene = {
  id: string;
  theme: ThemeId;
  /** Céu: cor de cima e de baixo. */
  sky: [string, string];
  /** Chão: cor e altura da linha do horizonte; `null` = sem chão (espaço). */
  ground: { color: string; y: number } | null;
  items: SceneItem[];
};

export type Theme = { id: ThemeId; color: string; icon: string };

export const THEMES: Theme[] = [
  { id: 'dinos', color: '#43a047', icon: '1f996' },
  { id: 'herois', color: '#e53935', icon: '1f9b8' },
  { id: 'fazenda', color: '#f59e0b', icon: '1f404' },
  { id: 'circo', color: '#8e24aa', icon: '1f3aa' },
];

const STARS = (pts: [number, number, number][]): SceneItem[] => pts.map(([x, y, s]) => ({ img: '2b50', x, y, s }));

export const SCENES: Scene[] = [
  // ── Dinossauros ────────────────────────────────────────────────────────
  {
    id: 'vale',
    theme: 'dinos',
    sky: ['#7fd3ff', '#dff5ff'],
    ground: { color: '#7ccf5a', y: 205 },
    items: [
      { img: '2600', x: 345, y: 50, s: 64 },
      { img: '1f30b', x: 95, y: 148, s: 140 },
      { img: '1f334', x: 335, y: 148, s: 116 },
      { img: '1f995', x: 255, y: 196, s: 150 },
      { img: '1f996', x: 112, y: 222, s: 124 },
      { img: '1f33f', x: 30, y: 268, s: 52 },
      { img: '1f33f', x: 372, y: 270, s: 46 },
    ],
  },
  {
    id: 'ninho',
    theme: 'dinos',
    sky: ['#ffc46b', '#fff0c9'],
    ground: { color: '#d2a86a', y: 212 },
    items: [
      { img: '2601', x: 205, y: 55, s: 70 },
      { img: '1f334', x: 62, y: 145, s: 124 },
      { img: '1f995', x: 300, y: 178, s: 164, flip: true },
      { img: '1faa8', x: 40, y: 262, s: 60 },
      { img: '1f95a', x: 152, y: 252, s: 46 },
      { img: '1f95a', x: 194, y: 258, s: 42 },
      { img: '1f95a', x: 172, y: 236, s: 40 },
      { img: '1f98e', x: 336, y: 266, s: 58 },
    ],
  },
  {
    id: 'noite',
    theme: 'dinos',
    sky: ['#1b2a5c', '#4d5fa8'],
    ground: { color: '#3f6b3a', y: 215 },
    items: [
      { img: '1f319', x: 335, y: 58, s: 70 },
      ...STARS([
        [55, 48, 24],
        [128, 92, 18],
        [238, 40, 20],
        [385, 125, 16],
        [170, 30, 14],
      ]),
      { img: '1f30b', x: 300, y: 165, s: 124 },
      { img: '1f996', x: 130, y: 202, s: 142 },
      { img: '1f33f', x: 372, y: 270, s: 46 },
      { img: '1f33f', x: 28, y: 272, s: 44 },
    ],
  },

  // ── Heróis ─────────────────────────────────────────────────────────────
  {
    id: 'voo',
    theme: 'herois',
    sky: ['#4aa6ff', '#cfe9ff'],
    ground: { color: '#9aa7b8', y: 252 },
    items: [
      { img: '2601', x: 82, y: 60, s: 64 },
      { img: '26a1', x: 335, y: 58, s: 52 },
      { img: '1f3e2', x: 48, y: 196, s: 112 },
      { img: '1f3e8', x: 150, y: 190, s: 124 },
      { img: '1f3e2', x: 262, y: 196, s: 112 },
      { img: '1f3e8', x: 352, y: 200, s: 100 },
      { img: '1f9b8', x: 214, y: 96, s: 124 },
    ],
  },
  {
    id: 'espaco',
    theme: 'herois',
    sky: ['#0e0f3a', '#3a2a7a'],
    ground: null,
    items: [
      ...STARS([
        [40, 30, 18],
        [150, 60, 14],
        [370, 210, 18],
        [30, 270, 14],
        [200, 280, 16],
        [250, 30, 12],
      ]),
      { img: '1fa90', x: 305, y: 88, s: 112 },
      { img: '1f6f8', x: 80, y: 72, s: 64 },
      { img: '1f680', x: 92, y: 205, s: 112 },
      { img: '1f9b8-200d-2640-fe0f', x: 238, y: 192, s: 134 },
    ],
  },
  {
    id: 'resgate',
    theme: 'herois',
    sky: ['#8fd3ff', '#e6f6ff'],
    ground: { color: '#a0a8b5', y: 228 },
    items: [
      { img: '2600', x: 345, y: 48, s: 56 },
      { img: '1f333', x: 360, y: 178, s: 92 },
      { img: '1f3e0', x: 80, y: 168, s: 128 },
      { img: '1f9d1-200d-1f692', x: 162, y: 238, s: 88 },
      { img: '1f692', x: 262, y: 238, s: 116 },
      { img: '1f691', x: 360, y: 258, s: 72 },
      { img: '1f6a8', x: 210, y: 70, s: 48 },
    ],
  },

  // ── Fazenda ────────────────────────────────────────────────────────────
  {
    id: 'sitio',
    theme: 'fazenda',
    sky: ['#8fd3ff', '#e6f6ff'],
    ground: { color: '#7ccf5a', y: 200 },
    items: [
      { img: '2600', x: 350, y: 50, s: 58 },
      { img: '1f333', x: 205, y: 150, s: 88 },
      { img: '1f3e1', x: 88, y: 158, s: 128 },
      { img: '1f69c', x: 290, y: 215, s: 124 },
      { img: '1f404', x: 150, y: 250, s: 92 },
      { img: '1f33b', x: 30, y: 255, s: 58 },
      { img: '1f33b', x: 372, y: 262, s: 52 },
    ],
  },
  {
    id: 'bichos',
    theme: 'fazenda',
    sky: ['#9bdcff', '#eef9ff'],
    ground: { color: '#86d06a', y: 190 },
    items: [
      { img: '1f308', x: 200, y: 72, s: 150 },
      { img: '1f333', x: 362, y: 162, s: 92 },
      { img: '1f40e', x: 238, y: 170, s: 104 },
      { img: '1f416', x: 88, y: 238, s: 92 },
      { img: '1f414', x: 188, y: 252, s: 72 },
      { img: '1f411', x: 304, y: 238, s: 94 },
    ],
  },
  {
    id: 'horta',
    theme: 'fazenda',
    sky: ['#8fd3ff', '#fff5d6'],
    ground: { color: '#a7783f', y: 205 },
    items: [
      { img: '2600', x: 60, y: 56, s: 60 },
      { img: '2601', x: 300, y: 60, s: 70 },
      { img: '1f9d1-200d-1f33e', x: 202, y: 168, s: 124 },
      { img: '1f955', x: 58, y: 250, s: 62 },
      { img: '1f955', x: 108, y: 262, s: 52 },
      { img: '1f345', x: 160, y: 264, s: 44 },
      { img: '1f407', x: 252, y: 262, s: 58 },
      { img: '1f33d', x: 308, y: 246, s: 70 },
      { img: '1f33d', x: 358, y: 254, s: 62 },
    ],
  },

  // ── Circo ──────────────────────────────────────────────────────────────
  {
    id: 'lona',
    theme: 'circo',
    sky: ['#ffb3d9', '#fff0f7'],
    ground: { color: '#f2c14e', y: 228 },
    items: [
      { img: '1f388', x: 50, y: 82, s: 62 },
      { img: '1f388', x: 352, y: 76, s: 60 },
      { img: '1f388', x: 384, y: 128, s: 44 },
      { img: '1f3a1', x: 62, y: 206, s: 104 },
      { img: '1f3aa', x: 205, y: 142, s: 196 },
      { img: '1f921', x: 334, y: 242, s: 82 },
    ],
  },
  {
    id: 'picadeiro',
    theme: 'circo',
    sky: ['#5b2a86', '#b46bdc'],
    ground: { color: '#d94b4b', y: 232 },
    items: [
      { img: '1f3b6', x: 60, y: 68, s: 50 },
      { img: '1f3b6', x: 342, y: 64, s: 46 },
      { img: '1f939', x: 205, y: 140, s: 108 },
      { img: '1f418', x: 98, y: 206, s: 132 },
      { img: '1f981', x: 302, y: 218, s: 100 },
      { img: '1f9ad', x: 205, y: 268, s: 58 },
    ],
  },
  {
    id: 'parque',
    theme: 'circo',
    sky: ['#8fd3ff', '#ffe6f2'],
    ground: { color: '#7ccf5a', y: 225 },
    items: [
      { img: '1f388', x: 220, y: 66, s: 56 },
      { img: '1f3a1', x: 300, y: 150, s: 152 },
      { img: '1f3a0', x: 108, y: 176, s: 138 },
      { img: '1f37f', x: 56, y: 262, s: 56 },
      { img: '1f3ab', x: 205, y: 262, s: 50 },
      { img: '1f388', x: 380, y: 250, s: 40 },
    ],
  },
];

export const scenesOf = (theme: ThemeId): Scene[] => SCENES.filter((s) => s.theme === theme);
export const sceneById = (id: string): Scene | undefined => SCENES.find((s) => s.id === id);
