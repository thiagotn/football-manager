/**
 * Bê-á-Bá — o álbum de figurinhas. Cada rodada completa dá uma figurinha nova
 * (sorteada entre as que faltam); 4 páginas de 6.
 *
 * `img` é o código Twemoji (`static/be-a-ba/img/<img>.svg`). Os nomes ficam no
 * i18n em `be_a_ba.sticker.<id>`.
 */

export type Sticker = { id: string; img: string };

export const STICKERS: Sticker[] = [
  // Página 1 — bichos
  { id: 'leao', img: '1f981' },
  { id: 'tigre', img: '1f42f' },
  { id: 'panda', img: '1f43c' },
  { id: 'coala', img: '1f428' },
  { id: 'raposa', img: '1f98a' },
  { id: 'polvo', img: '1f419' },
  // Página 2 — fantasia
  { id: 'unicornio', img: '1f984' },
  { id: 'golfinho', img: '1f42c' },
  { id: 'dinossauro', img: '1f996' },
  { id: 'brontossauro', img: '1f995' },
  { id: 'castelo', img: '1f3f0' },
  { id: 'arco_iris', img: '1f308' },
  // Página 3 — jardim
  { id: 'joaninha', img: '1f41e' },
  { id: 'abelha', img: '1f41d' },
  { id: 'papagaio', img: '1f99c' },
  { id: 'pintinho', img: '1f425' },
  { id: 'ursinho', img: '1f9f8' },
  { id: 'sorvete', img: '1f366' },
  // Página 4 — aventura
  { id: 'foguete', img: '1f680' },
  { id: 'disco_voador', img: '1f6f8' },
  { id: 'vulcao', img: '1f30b' },
  { id: 'carrossel', img: '1f3a0' },
  { id: 'violao', img: '1f3b8' },
  { id: 'futebol', img: '26bd' },
];

/** Figurinha especial da primeira vez que a criança escreve o próprio nome. */
export const NAME_STICKER: Sticker = { id: 'meu_nome', img: '270d' };

export const PAGE_SIZE = 6;

/**
 * Próxima figurinha: sorteada entre as que faltam (`rnd` em [0, 1)). Com o
 * álbum completo devolve `null` — a rodada ainda premia com estrelas.
 */
export function nextSticker(owned: readonly string[], rnd: number): Sticker | null {
  const missing = STICKERS.filter((s) => !owned.includes(s.id));
  if (!missing.length) return null;
  return missing[Math.min(missing.length - 1, Math.floor(rnd * missing.length))];
}

export const stickerById = (id: string): Sticker | undefined =>
  id === NAME_STICKER.id ? NAME_STICKER : STICKERS.find((s) => s.id === id);
