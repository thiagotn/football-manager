/**
 * Registro dos jogos publicados em games.thiagotn.com.
 *
 * O hub monta os cards a partir daqui — jogo novo entra nesta lista. (O
 * service worker não precisa dela: deriva as rotas de `prerendered`.)
 */

/** Tom da etiqueta no card (a cor sai daqui, o texto vem do i18n). */
export type TagTone = 'hot' | 'gold' | 'cool' | 'neutral';

export type GameTag = { key: string; tone: TagTone };

export type Game = {
  /** Pasta da rota, do manifesto e dos ícones: `/<slug>/`. */
  slug: string;
  /** Chaves i18n do nome e da descrição exibidos no card e no `<title>`. */
  nameKey: string;
  descKey: string;
  tags: GameTag[];
  /** `theme-color` da rota e do manifesto. */
  themeColor: string;
  /** Mostra o selo "Novo" no card do hub. */
  isNew?: boolean;
};

export const GAMES: Game[] = [
  {
    slug: 'enche-ai',
    nameKey: 'enche_ai.name',
    descKey: 'enche_ai.desc',
    tags: [
      { key: 'hub.tag_puzzle', tone: 'hot' },
      { key: 'hub.tag_daily', tone: 'gold' },
      { key: 'hub.tag_offline', tone: 'neutral' },
    ],
    themeColor: '#183a69',
    isNew: true,
  },
  {
    slug: 'paciencia',
    nameKey: 'paciencia.title',
    descKey: 'paciencia.desc',
    tags: [
      { key: 'hub.tag_cards', tone: 'cool' },
      { key: 'hub.tag_daily', tone: 'gold' },
      { key: 'hub.tag_offline', tone: 'neutral' },
    ],
    themeColor: '#0d3123',
  },
  {
    slug: 'tetris',
    nameKey: 'tetris.name',
    descKey: 'tetris.desc',
    tags: [
      { key: 'hub.tag_arcade', tone: 'cool' },
      { key: 'hub.tag_input', tone: 'neutral' },
      { key: 'hub.tag_offline', tone: 'neutral' },
    ],
    themeColor: '#010112',
  },
];
