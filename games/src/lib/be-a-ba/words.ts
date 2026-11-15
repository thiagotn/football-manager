/**
 * Bê-á-Bá — banco de palavras da trilha das Letras (só pt-BR na v1).
 *
 * O nível vem da **estrutura silábica**, não do tamanho: é o que define a
 * dificuldade de leitura. `img` é o código Twemoji do desenho (arquivo em
 * `static/be-a-ba/img/<img>.svg`, copiado por `npm run twemoji`) ou um desenho
 * próprio no mesmo estilo (`mesa`).
 */

export type WordLevel = 1 | 2 | 3 | 4;

export type Word = {
  /** Chave estável (sem acento) — progresso e testes usam ela. */
  id: string;
  /** Grafia em CAIXA ALTA, com acento. */
  text: string;
  img: string;
  level: WordLevel;
};

const w = (text: string, img: string, level: WordLevel): Word => ({ id: plain(text).toLowerCase(), text, img, level });

/** Tira acentos e cedilha: `AVIÃO` → `AVIAO`. */
export function plain(s: string): string {
  return s.normalize('NFD').replace(/[̀-ͯ]/g, '');
}

export const WORDS: Word[] = [
  // Nível 1 — palavras curtas, vogais e sílabas simples
  w('UVA', '1f347', 1),
  w('OVO', '1f95a', 1),
  w('SOL', '2600', 1),
  w('PÃO', '1f35e', 1),
  w('MÃO', '270b', 1),
  w('PÉ', '1f9b6', 1),
  w('LUA', '1f319', 1),
  w('BOI', '1f402', 1),
  w('MAR', '1f30a', 1),
  w('ÍMÃ', '1f9f2', 1),
  w('ANEL', '1f48d', 1),

  // Nível 2 — duas sílabas simples (consoante + vogal)
  w('BOLA', '26bd', 2),
  w('CASA', '1f3e0', 2),
  w('GATO', '1f408', 2),
  w('PATO', '1f986', 2),
  w('MESA', 'mesa', 2),
  w('SAPO', '1f438', 2),
  w('DADO', '1f3b2', 2),
  w('VACA', '1f404', 2),
  w('BOLO', '1f382', 2),
  w('PIPA', '1fa81', 2),
  w('SUCO', '1f9c3', 2),
  w('COCO', '1f965', 2),
  w('MALA', '1f9f3', 2),
  w('FOCA', '1f9ad', 2),
  w('LOBO', '1f43a', 2),
  w('MOTO', '1f3cd', 2),
  w('BEBÊ', '1f476', 2),
  w('SINO', '1f514', 2),
  w('FADA', '1f9da', 2),
  w('RATO', '1f401', 2),
  w('CAMA', '1f6cf', 2),
  w('PERA', '1f350', 2),
  w('MAPA', '1f5fa', 2),
  w('LIMÃO', '1f34b', 2),
  w('BONÉ', '1f9e2', 2),

  // Nível 3 — três sílabas simples
  w('BONECA', '1fa86', 3),
  w('SAPATO', '1f45e', 3),
  w('MACACO', '1f412', 3),
  w('CAVALO', '1f434', 3),
  w('JANELA', '1fa9f', 3),
  w('TOMATE', '1f345', 3),
  w('BANANA', '1f34c', 3),
  w('CAMELO', '1f42b', 3),
  w('BALEIA', '1f433', 3),
  w('CENOURA', '1f955', 3),
  w('PIPOCA', '1f37f', 3),
  w('BATATA', '1f954', 3),
  w('GIRAFA', '1f992', 3),
  w('JACARÉ', '1f40a', 3),
  w('COMETA', '2604', 3),
  w('PANELA', '1f372', 3),
  w('ABACATE', '1f951', 3),

  // Nível 4 — dígrafos, encontros consonantais e palavras longas
  w('CARRO', '1f697', 4),
  w('FLOR', '1f338', 4),
  w('PEIXE', '1f41f', 4),
  w('CHAVE', '1f511', 4),
  w('PRATO', '1f37d', 4),
  w('ELEFANTE', '1f418', 4),
  w('BICICLETA', '1f6b2', 4),
  w('BORBOLETA', '1f98b', 4),
  w('TARTARUGA', '1f422', 4),
  w('ESTRELA', '2b50', 4),
  w('ÁRVORE', '1f333', 4),
  w('AVIÃO', '2708', 4),
  w('CACHORRO', '1f415', 4),
  w('COELHO', '1f407', 4),
  w('PALHAÇO', '1f921', 4),
  w('MORANGO', '1f353', 4),
  w('ABACAXI', '1f34d', 4),
  w('GALINHA', '1f414', 4),
  w('MELANCIA', '1f349', 4),
  w('TREM', '1f682', 4),
  w('LÁPIS', '270f', 4),
];

export const wordsOfLevel = (level: WordLevel): Word[] => WORDS.filter((x) => x.level === level);
