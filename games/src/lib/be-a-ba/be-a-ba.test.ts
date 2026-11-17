import { describe, expect, it } from 'vitest';
import { WORDS, plain } from './words';
import { NAME_STICKER, STICKERS, nextSticker } from './stickers';
import { MASCOT, MEDAL, PARTY, STAR } from './art';
import {
  ALPHABET, buildTray, cleanFamilyWord, firstLetterOptions, lettersOf, makeLetterChallenge, sameLetter,
} from './letters';
import { ITEMS, answerOf, makeNumberChallenge, optionsFor, type NumberKind } from './numbers';
import {
  GOOD_ROUNDS_TO_ADVANCE, LETTER_STAGES, NUMBER_STAGES, ROUND_SIZE, afterRound, emptyTrail, letterRound,
  numberRound, outcomeOf, pickWeighted, recordOutcome,
} from './progress';
import {
  HEAR_ARTICLES, HEAR_WORDS, LITTLE_LETTER_STAGES, LITTLE_NUMBER_STAGES, LITTLE_ROUND_SIZE, VOWELS, letterPool,
  littleAnswer, littleItemId, littleLetterRound, littleNumberRound,
} from './little';
import { EN_COLORS, EN_STAGES, EN_WORDS, enAnswerWord, enPrompt, enSlug, englishRound, wordsOfStage } from './english';
import { mulberry32 } from '../enche-ai/engine';

// Arquivos de desenho que existem de fato em static/ (só os nomes; nada é carregado).
const IMG_FILES = new Set(
  Object.keys(import.meta.glob('../../../static/be-a-ba/img/*.svg')).map((p) => p.split('/').pop()!.replace('.svg', ''))
);
const rng = (seed = 1) => mulberry32(seed);

describe('conteúdo', () => {
  it('todo desenho usado existe em static/be-a-ba/img', () => {
    const codes = [
      ...WORDS.map((w) => w.img),
      ...STICKERS.map((s) => s.img),
      NAME_STICKER.img,
      ...ITEMS,
      MASCOT, STAR, PARTY, MEDAL,
    ];
    for (const c of codes) expect(IMG_FILES.has(c), c).toBe(true);
  });

  it('palavras: ids únicos, só letras em CAIXA ALTA, todos os níveis com conteúdo', () => {
    expect(new Set(WORDS.map((w) => w.id)).size).toBe(WORDS.length);
    for (const w of WORDS) {
      expect(w.text).toMatch(/^\p{Lu}+$/u);
      expect(w.id).toBe(plain(w.text).toLowerCase());
    }
    for (const level of [1, 2, 3, 4]) expect(WORDS.filter((w) => w.level === level).length).toBeGreaterThanOrEqual(10);
    // Os exemplos do pedido estão no banco (ARTHUR entra como palavra da família).
    for (const id of ['uva', 'carro', 'mesa', 'elefante']) expect(WORDS.some((w) => w.id === id), id).toBe(true);
  });

  it('figurinhas: ids únicos, 24 no álbum, sorteio só entre as que faltam', () => {
    expect(STICKERS).toHaveLength(24);
    expect(new Set(STICKERS.map((s) => s.id)).size).toBe(24);
    const owned = STICKERS.slice(0, 23).map((s) => s.id);
    expect(nextSticker(owned, 0.99)?.id).toBe(STICKERS[23].id);
    expect(nextSticker(STICKERS.map((s) => s.id), 0.5)).toBeNull();
  });
});

describe('letras', () => {
  it('acento e cedilha nunca contam como erro', () => {
    expect(sameLetter('Ã', 'a')).toBe(true);
    expect(sameLetter('Ç', 'C')).toBe(true);
    expect(sameLetter('Ç', 'Ç')).toBe(true);
    expect(sameLetter('É', 'E')).toBe(true);
    expect(sameLetter('B', 'D')).toBe(false);
  });

  it('separa as letras mantendo o acento', () => {
    expect(lettersOf('AVIÃO')).toEqual(['A', 'V', 'I', 'Ã', 'O']);
    expect(lettersOf('AVIÃO')).toEqual(['A', 'V', 'I', 'Ã', 'O']); // NFD vira NFC
  });

  it('limpa a palavra da família', () => {
    expect(cleanFamilyWord(' Arthur ')).toBe('ARTHUR');
    expect(cleanFamilyWord('vovó Ana!')).toBe('VOVÓANA');
    expect(cleanFamilyWord('123')).toBe('');
    expect(cleanFamilyWord('a'.repeat(30))).toHaveLength(12);
  });

  it('primeira letra: 3 opções distintas, a certa sem acento entre elas', () => {
    for (let s = 1; s < 50; s++) {
      const opts = firstLetterOptions('ÍMÃ', rng(s));
      expect(opts).toHaveLength(3);
      expect(new Set(opts).size).toBe(3);
      expect(opts).toContain('I');
      for (const o of opts) expect(ALPHABET).toContain(o);
    }
  });

  it('montar: a bandeja tem as letras da palavra + sobras e nunca vem em ordem', () => {
    for (const w of WORDS) {
      const tray = buildTray(w.text, 2, rng(w.id.length));
      expect(tray).toHaveLength(lettersOf(w.text).length + 2);
      const letters = lettersOf(plain(w.text));
      const left = tray.slice();
      for (const l of letters) left.splice(left.indexOf(l), 1);
      expect(left).toHaveLength(2);
      for (const l of left) expect(letters).not.toContain(l);
      expect(tray.slice(0, letters.length).join('')).not.toBe(letters.join(''));
    }
  });

  it('cada modo preenche só o que usa', () => {
    const word = { id: 'uva', text: 'UVA', img: '1f347' };
    expect(makeLetterChallenge('first', word, 0, rng()).options).toHaveLength(3);
    expect(makeLetterChallenge('build', word, 1, rng()).tray).toHaveLength(4);
    const typed = makeLetterChallenge('type', word, 1, rng());
    expect(typed.options).toEqual([]);
    expect(typed.tray).toEqual([]);
  });
});

describe('números', () => {
  const KINDS: NumberKind[] = ['count', 'compare', 'add', 'sub'];

  it('toda conta gerada fica na faixa e tem a resposta entre as opções', () => {
    for (const max of [5, 10]) {
      for (const kind of KINDS) {
        for (let s = 1; s <= 300; s++) {
          const c = makeNumberChallenge(kind, max, rng(s * 31 + max));
          const ans = answerOf(c);
          if (c.kind === 'compare') {
            expect(c.left).not.toBe(c.right);
            expect(Math.max(c.left, c.right)).toBeLessThanOrEqual(max);
            expect(Math.min(c.left, c.right)).toBeGreaterThanOrEqual(1);
            continue;
          }
          expect(typeof ans).toBe('number');
          expect(c.options).toContain(ans);
          expect(new Set(c.options).size).toBe(3);
          for (const o of c.options) expect(o).toBeGreaterThanOrEqual(0);
          for (const o of c.options) expect(o).toBeLessThanOrEqual(max);
          if (c.kind === 'add') expect(c.a + c.b).toBeLessThanOrEqual(max);
          if (c.kind === 'sub') expect(c.a - c.b).toBeGreaterThanOrEqual(1);
          if (c.kind === 'count') expect(c.n).toBeGreaterThanOrEqual(1);
        }
      }
    }
  });

  it('opções sempre 3 distintas, mesmo na borda da faixa', () => {
    expect(new Set(optionsFor(1, 1, 5, rng())).size).toBe(3);
    expect(new Set(optionsFor(5, 1, 5, rng())).size).toBe(3);
    expect(optionsFor(0, 0, 2, rng()).sort()).toEqual([0, 1, 2]);
  });
});

describe('progresso', () => {
  it('resultado do desafio', () => {
    expect(outcomeOf(0, false)).toBe('clean');
    expect(outcomeOf(0, true)).toBe('hint');
    expect(outcomeOf(2, false)).toBe('hint');
    expect(outcomeOf(3, false)).toBe('miss');
  });

  it('Leitner: sobe sem erro, fica com dica, desce errando, entre 1 e 5', () => {
    let b = recordOutcome({}, 'uva', 'clean');
    expect(b.uva.box).toBe(2);
    b = recordOutcome(b, 'uva', 'hint');
    expect(b.uva.box).toBe(2);
    b = recordOutcome(b, 'uva', 'miss');
    expect(b.uva).toEqual({ box: 1, seen: 3, misses: 1 });
    b = recordOutcome(b, 'uva', 'miss');
    expect(b.uva.box).toBe(1);
    for (let i = 0; i < 10; i++) b = recordOutcome(b, 'uva', 'clean');
    expect(b.uva.box).toBe(5);
  });

  it('sobe de estágio após rodadas boas seguidas; nunca desce; estágio fixo não muda', () => {
    const L = LETTER_STAGES.length;
    let p = emptyTrail();
    for (let i = 0; i < GOOD_ROUNDS_TO_ADVANCE - 1; i++) p = afterRound(p, L, 5).progress;
    expect(p.stage).toBe(0);
    const up = afterRound(p, L, 4);
    expect(up.leveledUp).toBe(true);
    expect(up.progress.stage).toBe(1);
    expect(afterRound(up.progress, L, 0).progress.stage).toBe(1);
    // Rodada ruim zera a sequência
    p = afterRound(emptyTrail(), NUMBER_STAGES.length, 5).progress;
    expect(afterRound(p, NUMBER_STAGES.length, 1).progress.goodRounds).toBe(0);
    // Teto
    p = { ...emptyTrail(), stage: NUMBER_STAGES.length - 1, goodRounds: 5 };
    expect(afterRound(p, NUMBER_STAGES.length, 5).leveledUp).toBe(false);
    // Fixado pelo adulto
    p = { ...emptyTrail(), locked: true };
    expect(afterRound(afterRound(p, L, 5).progress, L, 5).progress.stage).toBe(0);
    // Rodada de 3 (modo Pequenos): boa só com as 3 limpas
    expect(afterRound(emptyTrail(), 3, 3, 3).progress.goodRounds).toBe(1);
    expect(afterRound({ ...emptyTrail(), goodRounds: 1 }, 3, 2, 3).progress.goodRounds).toBe(1);
    expect(afterRound({ ...emptyTrail(), goodRounds: 1 }, 3, 1, 3).progress.goodRounds).toBe(0);
  });

  it('sorteio ponderado não repete e respeita n', () => {
    const out = pickWeighted([1, 2, 3, 4], () => 1, 3, rng());
    expect(out).toHaveLength(3);
    expect(new Set(out).size).toBe(3);
    expect(pickWeighted([1, 2], () => 1, 5, rng())).toHaveLength(2);
  });

  it('rodada das letras: 5 palavras do estágio, sem repetir, modos do estágio', () => {
    LETTER_STAGES.forEach((stage, i) => {
      const round = letterRound({ ...emptyTrail(), stage: i }, [], rng(i + 7));
      expect(round).toHaveLength(ROUND_SIZE);
      expect(new Set(round.map((c) => c.word.id)).size).toBe(ROUND_SIZE);
      for (const c of round) {
        expect(stage.modes).toContain(c.mode);
        expect(stage.levels).toContain(WORDS.find((w) => w.id === c.word.id)!.level);
      }
    });
  });

  it('rodada das letras inclui uma palavra da família quando houver', () => {
    const family = [{ id: 'fam-1', text: 'ARTHUR', family: true }];
    for (let s = 1; s < 20; s++) {
      const round = letterRound({ ...emptyTrail(), stage: 2 }, family, rng(s));
      expect(round).toHaveLength(ROUND_SIZE);
      expect(round.filter((c) => c.word.family)).toHaveLength(1);
      expect(round[0].word.family).toBeFalsy(); // nunca abre a rodada
    }
  });

  it('rodada dos números: 5 contas do estágio, sem repetir', () => {
    NUMBER_STAGES.forEach((stage, i) => {
      for (let s = 1; s < 30; s++) {
        const round = numberRound({ ...emptyTrail(), stage: i }, rng(s));
        expect(round).toHaveLength(ROUND_SIZE);
        for (const c of round) expect(stage.kinds).toContain(c.kind);
      }
    });
  });

  it('palavras dominadas aparecem menos', () => {
    const boxes = Object.fromEntries(WORDS.filter((w) => w.level <= 2 && w.id !== 'uva').map((w) => [w.id, { box: 5, seen: 9, misses: 0 }]));
    let uva = 0;
    for (let s = 1; s <= 200; s++) {
      if (letterRound({ ...emptyTrail(), stage: 0, boxes }, [], rng(s)).some((c) => c.word.id === 'uva')) uva++;
    }
    // Só a UVA não está dominada: aparece em mais que o dobro das rodadas do acaso (5/36 ≈ 14%).
    expect(uva / 200).toBeGreaterThan(0.3);
  });
});

describe('modo Pequenos (3–4 anos)', () => {
  it('toda palavra do "Cadê?" existe no banco e tem artigo', () => {
    expect(HEAR_WORDS.length).toBe(Object.keys(HEAR_ARTICLES).length);
    expect(HEAR_WORDS.length).toBeGreaterThanOrEqual(30);
  });

  it('letras: vogais primeiro; depois consoantes e a inicial do nome', () => {
    expect(letterPool(LITTLE_LETTER_STAGES[0], 'Arthur')).toEqual(VOWELS);
    const later = letterPool(LITTLE_LETTER_STAGES[2], 'Rafael');
    expect(later).toEqual(expect.arrayContaining([...VOWELS, 'B', 'M', 'R']));
    expect(letterPool(LITTLE_LETTER_STAGES[2], 'Ícaro').filter((l) => l === 'I')).toHaveLength(1);
    expect(letterPool(LITTLE_LETTER_STAGES[2], '')).not.toContain('');
  });

  it('rodada de letras: 3 desafios, opções do estágio, resposta entre elas', () => {
    LITTLE_LETTER_STAGES.forEach((stage, i) => {
      for (let s = 1; s < 40; s++) {
        const round = littleLetterRound({ ...emptyTrail(), stage: i }, 'Bia', rng(s));
        expect(round).toHaveLength(LITTLE_ROUND_SIZE);
        for (const c of round) {
          expect(stage.kinds).toContain(c.kind);
          if (c.kind === 'hear') {
            expect(c.options).toHaveLength(stage.options);
            expect(c.options.map((o) => o.id)).toContain(c.word.id);
            expect(new Set(c.options.map((o) => o.id)).size).toBe(stage.options);
            expect(c.article).toBe(HEAR_ARTICLES[c.word.id]);
          } else if (c.kind === 'pick_letter') {
            expect(c.options).toHaveLength(stage.options);
            expect(c.options).toContain(c.letter);
            expect(new Set(c.options).size).toBe(stage.options);
          }
        }
      }
    });
  });

  it('rodada de números: 3 desafios na faixa, sem repetir, resposta certa', () => {
    LITTLE_NUMBER_STAGES.forEach((stage, i) => {
      for (let s = 1; s < 60; s++) {
        const round = littleNumberRound({ ...emptyTrail(), stage: i }, rng(s));
        expect(round).toHaveLength(LITTLE_ROUND_SIZE);
        for (const c of round) {
          expect(stage.kinds).toContain(c.kind);
          if (c.kind === 'tap') expect(c.n).toBeGreaterThanOrEqual(1);
          if (c.kind === 'tap' || c.kind === 'find_number' || c.kind === 'where') expect(c.n).toBeLessThanOrEqual(stage.max);
          if (c.kind === 'find_number') {
            expect(c.options).toHaveLength(stage.options);
            expect(c.options).toContain(c.n);
            expect(littleAnswer(c)).toBe(c.n);
          }
          if (c.kind === 'where') {
            expect(c.groups).toHaveLength(2);
            expect(c.groups[0]).not.toBe(c.groups[1]);
            expect(c.groups[littleAnswer(c) as number]).toBe(c.n);
            for (const g of c.groups) expect(g).toBeGreaterThanOrEqual(1);
          }
        }
      }
    });
  });

  it('chave do Leitner separa palavra, letra e tipo de conta', () => {
    const [hear] = littleLetterRound({ ...emptyTrail(), stage: 0 }, '', rng(3));
    expect(littleItemId(hear)).toMatch(/^hear:/);
    expect(littleItemId({ kind: 'pick_letter', letter: 'A', options: ['A', 'E'] })).toBe('letter:A');
    expect(littleItemId({ kind: 'tap', item: '1f34e', n: 2 })).toBe('tap');
  });
});

describe('trilha Inglês', () => {
  it('toda palavra tem desenho, id único e categoria', () => {
    expect(new Set(EN_WORDS.map((w) => w.id)).size).toBe(EN_WORDS.length);
    for (const w of EN_WORDS) {
      expect(IMG_FILES.has(w.img), w.text).toBe(true);
      expect(w.text).toMatch(/^[a-z ]+$/);
    }
    for (const c of ['animals', 'fruits', 'foods'] as const) {
      expect(EN_WORDS.filter((w) => w.category === c).length).toBeGreaterThanOrEqual(10);
    }
  });

  it('perguntas em inglês, com "are" no plural', () => {
    const word = (id: string) => EN_WORDS.find((w) => w.id === id)!;
    expect(enPrompt({ kind: 'hear', word: word('en-apple'), options: [] })).toBe('Where is the apple?');
    expect(enPrompt({ kind: 'hear', word: word('en-grapes'), options: [] })).toBe('Where are the grapes?');
    expect(enPrompt({ kind: 'pick_color', color: 'red', options: [] })).toBe('Touch red!');
    expect(enPrompt({ kind: 'find_number', n: 3, options: [] })).toBe('Find number three!');
    expect(enAnswerWord({ kind: 'tap', item: '1f34e', n: 7 })).toBe('seven');
    expect(enSlug(word('en-ice-cream'))).toBe('ice_cream');
  });

  it('rodada: tamanho pedido, conteúdo do estágio e figuras erradas da mesma categoria', () => {
    EN_STAGES.forEach((stage, i) => {
      for (const size of [3, 5]) {
        for (let s = 1; s < 60; s++) {
          const round = englishRound({ ...emptyTrail(), stage: i }, size, rng(s * 7 + size));
          expect(round).toHaveLength(size);
          for (const c of round) {
            expect(stage.kinds).toContain(c.kind);
            if (c.kind === 'hear') {
              const w = c.word as (typeof EN_WORDS)[number];
              expect(stage.categories).toContain(w.category);
              expect(c.options).toHaveLength(stage.options);
              expect(new Set(c.options.map((o) => o.id)).size).toBe(stage.options);
              for (const o of c.options) expect((o as typeof w).category).toBe(w.category);
            } else if (c.kind === 'pick_color') {
              const allowed = EN_COLORS.slice(0, stage.colors).map((x) => x.id);
              expect(c.options).toHaveLength(stage.options);
              expect(c.options).toContain(c.color);
              for (const o of c.options) expect(allowed).toContain(o);
            } else if (c.kind === 'find_number') {
              expect(c.options).toContain(c.n);
              for (const o of c.options) expect(o).toBeLessThanOrEqual(stage.numberMax);
            } else if (c.kind === 'tap') {
              expect(c.n).toBeGreaterThanOrEqual(2);
              expect(c.n).toBeLessThanOrEqual(6);
            }
          }
          // A mesma palavra ou cor não se repete na rodada
          const ids = round.map((c) => (c.kind === 'hear' ? c.word.id : c.kind === 'pick_color' ? c.color : null)).filter(Boolean);
          expect(new Set(ids).size).toBe(ids.length);
        }
      }
    });
  });

  it('o 1º estágio é só bichos, 4 cores e números até 3', () => {
    expect(wordsOfStage(EN_STAGES[0]).every((w) => w.category === 'animals')).toBe(true);
    expect(EN_STAGES[0].colors).toBe(4);
    expect(EN_STAGES[0].numberMax).toBe(3);
    expect(littleItemId({ kind: 'pick_color', color: 'red', options: [] })).toBe('color:red');
  });
});
