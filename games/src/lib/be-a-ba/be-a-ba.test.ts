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
    let p = emptyTrail();
    for (let i = 0; i < GOOD_ROUNDS_TO_ADVANCE - 1; i++) p = afterRound(p, 'letters', 5).progress;
    expect(p.stage).toBe(0);
    const up = afterRound(p, 'letters', 4);
    expect(up.leveledUp).toBe(true);
    expect(up.progress.stage).toBe(1);
    expect(afterRound(up.progress, 'letters', 0).progress.stage).toBe(1);
    // Rodada ruim zera a sequência
    p = afterRound(emptyTrail(), 'numbers', 5).progress;
    expect(afterRound(p, 'numbers', 1).progress.goodRounds).toBe(0);
    // Teto
    p = { ...emptyTrail(), stage: NUMBER_STAGES.length - 1, goodRounds: 5 };
    expect(afterRound(p, 'numbers', 5).leveledUp).toBe(false);
    // Fixado pelo adulto
    p = { ...emptyTrail(), locked: true };
    expect(afterRound(afterRound(p, 'letters', 5).progress, 'letters', 5).progress.stage).toBe(0);
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
