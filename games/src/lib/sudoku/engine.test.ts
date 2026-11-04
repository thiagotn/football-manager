import { describe, expect, it } from 'vitest';
import {
  CELLS,
  DIFFICULTIES,
  PEERS,
  UNITS,
  candidatesOf,
  conflicts,
  dailyDifficulty,
  dailyGame,
  dateKey,
  digitsOf,
  erase,
  fmtTime,
  generate,
  givensCount,
  hardnessOf,
  hint,
  isWon,
  newGame,
  place,
  remainingOf,
  countSolutions,
  fromLine,
  toLine,
  toggleNote,
  undo,
  useHint,
  wrongCells,
  UNDO_LIMIT,
  type Difficulty,
  type Game,
} from './engine';

/** Faixa de dados que cada dificuldade mira (espelha TIERS). */
const RANGE: Record<Difficulty, [number, number]> = {
  facil: [36, 46],
  medio: [29, 35],
  dificil: [25, 32],
  especialista: [21, 28],
};

/** Três seeds por faixa — o suficiente para pegar regressão sem travar o `npm test`. */
const SEEDS = [0x1234abcd, 0x5eed0001, 0x0a0b0c0d];

/** Partida de teste, sempre a mesma: gerar é a parte caríssima. */
const base: Game = newGame('facil', 0x51d0c0);

/** Primeira célula vazia do tabuleiro. */
const firstEmpty = (g: Game): number => g.values.findIndex((v, i) => !g.givens[i] && !v);

describe('casas e vizinhos', () => {
  it('cada célula disputa dígito com 20 outras, sem repetir', () => {
    for (let i = 0; i < CELLS; i++) {
      expect(PEERS[i]).toHaveLength(20);
      expect(new Set(PEERS[i]).size).toBe(20);
      expect(PEERS[i]).not.toContain(i);
    }
  });

  it('as 27 casas cobrem cada célula três vezes', () => {
    expect(UNITS).toHaveLength(27);
    const seen = new Array<number>(CELLS).fill(0);
    for (const unit of UNITS) {
      expect(unit).toHaveLength(9);
      for (const i of unit) seen[i]++;
    }
    expect(seen.every((n) => n === 3)).toBe(true);
  });
});

describe('candidatos e conflitos', () => {
  it('o candidato sai da linha, da coluna e do bloco', () => {
    const values = new Array<number>(CELLS).fill(0);
    values[1] = 1; // mesma linha
    values[18] = 2; // mesma coluna
    values[10] = 3; // mesmo bloco
    const g = { ...base, values, givens: values };
    expect(candidatesOf(g, 0)).toEqual([4, 5, 6, 7, 8, 9]);
  });

  it('célula preenchida não tem candidato', () => {
    expect(candidatesOf(base, base.givens.findIndex((v) => v > 0))).toEqual([]);
  });

  it('o conflito marca as duas células repetidas', () => {
    const values = new Array<number>(CELLS).fill(0);
    values[0] = 5;
    values[4] = 5; // mesma linha
    values[30] = 5; // outra linha, outra coluna, outro bloco
    const bad = conflicts(values);
    expect(bad[0]).toBe(true);
    expect(bad[4]).toBe(true);
    expect(bad[30]).toBe(false);
  });

  it('a contagem do teclado desconta o que já está na mesa', () => {
    const values = new Array<number>(CELLS).fill(0);
    values[0] = 7;
    values[10] = 7;
    const left = remainingOf(values);
    expect(left[7]).toBe(7);
    expect(left[1]).toBe(9);
  });

  it('digitsOf devolve os dígitos da máscara em ordem', () => {
    expect(digitsOf(0b000010101)).toEqual([1, 3, 5]);
    expect(digitsOf(0)).toEqual([]);
  });

  it('a linha de 81 caracteres vai e volta', () => {
    const line = toLine(base.givens);
    expect(line).toHaveLength(81);
    expect(fromLine(line)).toEqual(base.givens);
    expect(() => fromLine('123')).toThrow();
  });
});

describe('gerador', () => {
  it('todo tabuleiro gerado tem solução única', () => {
    for (const difficulty of DIFFICULTIES) {
      const { puzzle } = generate(difficulty, SEEDS[0]);
      expect(countSolutions(puzzle, 2)).toBe(1);
    }
  });

  it('a mesma seed devolve o mesmo tabuleiro', () => {
    for (const difficulty of DIFFICULTIES) {
      const a = generate(difficulty, 0xc0ffee);
      const b = generate(difficulty, 0xc0ffee);
      expect(a.puzzle).toEqual(b.puzzle);
      expect(a.solution).toEqual(b.solution);
    }
  });

  it('seeds diferentes devolvem tabuleiros diferentes', () => {
    const a = generate('facil', 1);
    const b = generate('facil', 2);
    expect(a.puzzle).not.toEqual(b.puzzle);
  });

  it('a quantidade de dados fica na faixa da dificuldade', () => {
    for (const difficulty of DIFFICULTIES) {
      const [min, max] = RANGE[difficulty];
      for (const seed of SEEDS) {
        const dados = givensCount(generate(difficulty, seed).puzzle);
        expect(dados).toBeGreaterThanOrEqual(min);
        expect(dados).toBeLessThanOrEqual(max);
      }
    }
  });

  it('fácil e médio saem só com singles', () => {
    for (const difficulty of ['facil', 'medio'] as Difficulty[]) {
      for (const seed of SEEDS) {
        expect(hardnessOf(generate(difficulty, seed).puzzle)).toBeLessThanOrEqual(2);
      }
    }
  });

  it('difícil e especialista exigem mais do que singles', () => {
    for (const difficulty of ['dificil', 'especialista'] as Difficulty[]) {
      for (const seed of SEEDS) {
        expect(hardnessOf(generate(difficulty, seed).puzzle)).toBeGreaterThanOrEqual(3);
      }
    }
  });

  it('os dados do tabuleiro conferem com a solução', () => {
    const { puzzle, solution } = generate('medio', SEEDS[1]);
    for (let i = 0; i < CELLS; i++) if (puzzle[i]) expect(puzzle[i]).toBe(solution[i]);
    expect(countSolutions(solution, 2)).toBe(1);
  });

  it('a geração não encosta em Math.random', () => {
    const real = Math.random;
    Math.random = () => {
      throw new Error('o gerador tem de ser determinístico');
    };
    try {
      for (const difficulty of DIFFICULTIES) expect(() => generate(difficulty, 99)).not.toThrow();
    } finally {
      Math.random = real;
    }
  });

  it('gerar uma partida de cada dificuldade cabe em 3 s', () => {
    const t0 = Date.now();
    for (const difficulty of DIFFICULTIES) newGame(difficulty, 0xbaba0000 ^ difficulty.length);
    expect(Date.now() - t0).toBeLessThan(3000);
  });
});

describe('tabuleiro do dia', () => {
  it('o mesmo dia devolve o mesmo tabuleiro', () => {
    for (const day of ['2026-11-04', '2026-12-25']) {
      const a = dailyGame(day);
      const b = dailyGame(day);
      expect(a.givens).toEqual(b.givens);
      expect(a.solution).toEqual(b.solution);
      expect(a.seed).toBe(b.seed);
    }
  });

  it('dias diferentes devolvem tabuleiros diferentes', () => {
    expect(dailyGame('2026-11-04').givens).not.toEqual(dailyGame('2026-11-05').givens);
  });

  it('a dificuldade sobe ao longo da semana', () => {
    // 2026-11-02 é segunda-feira.
    expect(dailyDifficulty('2026-11-02')).toBe('facil');
    expect(dailyDifficulty('2026-11-03')).toBe('facil');
    expect(dailyDifficulty('2026-11-04')).toBe('medio');
    expect(dailyDifficulty('2026-11-05')).toBe('medio');
    expect(dailyDifficulty('2026-11-06')).toBe('dificil');
    expect(dailyDifficulty('2026-11-07')).toBe('dificil');
    expect(dailyDifficulty('2026-11-08')).toBe('especialista');
  });

  it('a partida do dia guarda o dia e a dificuldade da data', () => {
    const g = dailyGame('2026-11-08');
    expect(g.daily).toBe('2026-11-08');
    expect(g.difficulty).toBe('especialista');
  });

  it('a chave do dia é local, não UTC', () => {
    expect(dateKey(new Date(2026, 10, 4, 23, 30))).toBe('2026-11-04');
    expect(dateKey(new Date(2026, 0, 1, 0, 5))).toBe('2026-01-01');
  });
});

describe('lances', () => {
  it('escrever preenche a célula e empilha o desfazer', () => {
    const at = firstEmpty(base);
    const next = place(base, at, base.solution[at]);
    expect(next.values[at]).toBe(base.solution[at]);
    expect(next.undo).toHaveLength(1);
    expect(next.mistakes).toBe(0);
  });

  it('não escreve em cima de um dado do tabuleiro', () => {
    const at = base.givens.findIndex((v) => v > 0);
    expect(place(base, at, 5)).toBe(base);
  });

  it('dígito diferente da solução conta erro', () => {
    const at = firstEmpty(base);
    const wrong = (base.solution[at] % 9) + 1;
    expect(place(base, at, wrong).mistakes).toBe(1);
  });

  it('a anotação liga e desliga', () => {
    const at = firstEmpty(base);
    const on = toggleNote(base, at, 4);
    expect(on.notes[at]).toBe(1 << 3);
    expect(toggleNote(on, at, 4).notes[at]).toBe(0);
  });

  it('anotação não entra em célula preenchida', () => {
    const at = firstEmpty(base);
    const filled = place(base, at, 1);
    expect(toggleNote(filled, at, 4)).toBe(filled);
  });

  it('escrever com autoNotes apaga a anotação dos vizinhos', () => {
    const at = firstEmpty(base);
    const digit = base.solution[at];
    const peer = PEERS[at].find((p) => !base.values[p])!;
    const marked = toggleNote(base, peer, digit);
    expect(marked.notes[peer]).not.toBe(0);
    expect(place(marked, at, digit, true).notes[peer]).toBe(0);
  });

  it('apagar limpa dígito e anotações', () => {
    const at = firstEmpty(base);
    const dirty = place(toggleNote(base, at, 2), at, 7);
    const clean = erase(dirty, at);
    expect(clean.values[at]).toBe(0);
    expect(clean.notes[at]).toBe(0);
  });

  it('apagar célula vazia não muda nada', () => {
    expect(erase(base, firstEmpty(base))).toBe(base);
  });

  it('desfazer devolve dígito e anotações anteriores', () => {
    const at = firstEmpty(base);
    const noted = toggleNote(base, at, 3);
    const filled = place(noted, at, 8);
    const back = undo(filled);
    expect(back.values[at]).toBe(0);
    expect(back.notes[at]).toBe(noted.notes[at]);
    expect(back.undo).toHaveLength(1);
  });

  it('desfazer em partida nova não muda nada', () => {
    expect(undo(base)).toBe(base);
  });

  it('tocar no dígito que já está na célula apaga', () => {
    const at = firstEmpty(base);
    const filled = place(base, at, 5);
    expect(place(filled, at, 5).values[at]).toBe(0);
  });

  it('desfazer devolve as anotações que o autoNotes comeu dos vizinhos', () => {
    const at = firstEmpty(base);
    const digit = base.solution[at];
    const peer = PEERS[at].find((p) => !base.values[p])!;
    const marked = toggleNote(base, peer, digit);
    const written = place(marked, at, digit, true);
    expect(written.notes[peer]).toBe(0);
    const back = undo(written);
    expect(back.notes[peer]).toBe(marked.notes[peer]);
    expect(back.values[at]).toBe(0);
  });

  it('a pilha de desfazer para no teto', () => {
    let g = base;
    const open = [...Array(CELLS).keys()].filter((i) => !g.givens[i]);
    for (let n = 0; n < UNDO_LIMIT + 20; n++) {
      const at = open[n % open.length];
      g = toggleNote(g, at, (n % 9) + 1);
    }
    expect(g.undo).toHaveLength(UNDO_LIMIT);
  });

  it('wrongCells aponta só o que o jogador errou', () => {
    const at = firstEmpty(base);
    expect(wrongCells(base)).toEqual([]);
    expect(wrongCells(place(base, at, (base.solution[at] % 9) + 1))).toEqual([at]);
    expect(wrongCells(place(base, at, base.solution[at]))).toEqual([]);
  });

  it('o lance não mexe no estado anterior', () => {
    const snapshot = JSON.stringify(base);
    const at = firstEmpty(base);
    const next = place(base, at, 9);
    expect(JSON.stringify(base)).toBe(snapshot);
    expect(next).not.toBe(base);
  });
});

describe('vitória', () => {
  it('preencher tudo certo ganha', () => {
    let g = base;
    for (let i = 0; i < CELLS; i++) if (!g.givens[i]) g = place(g, i, g.solution[i]);
    expect(g.won).toBe(true);
    expect(isWon(g)).toBe(true);
  });

  it('um dígito errado no fim não ganha', () => {
    let g = base;
    const cells = [...Array(CELLS).keys()].filter((i) => !g.givens[i]);
    for (const i of cells.slice(0, -1)) g = place(g, i, g.solution[i]);
    const last = cells[cells.length - 1];
    g = place(g, last, (g.solution[last] % 9) + 1);
    expect(g.won).toBe(false);
  });

  it('partida ganha não aceita mais lances', () => {
    let g = base;
    for (let i = 0; i < CELLS; i++) if (!g.givens[i]) g = place(g, i, g.solution[i]);
    expect(place(g, firstEmpty(base), 1)).toBe(g);
  });
});

describe('dica', () => {
  it('a dica devolve o dígito certo e a técnica', () => {
    const h = hint(base)!;
    expect(h).not.toBeNull();
    expect(h.digit).toBe(base.solution[h.at]);
    expect(['cell', 'house', 'revelada']).toContain(h.technique);
  });

  it('a dica ignora o dígito errado que o jogador deixou na grade', () => {
    const at = firstEmpty(base);
    const wrong = place(base, at, (base.solution[at] % 9) + 1);
    const h = hint(wrong)!;
    expect(h.digit).toBe(wrong.solution[h.at]);
  });

  it('usar a dica escreve o dígito e conta a ajuda, sem contar erro', () => {
    const h = hint(base)!;
    const next = useHint(base, h);
    expect(next.values[h.at]).toBe(h.digit);
    expect(next.hints).toBe(1);
    expect(next.mistakes).toBe(0);
  });

  it('tabuleiro resolvido não tem mais dica', () => {
    let g = base;
    for (let i = 0; i < CELLS; i++) if (!g.givens[i]) g = place(g, i, g.solution[i]);
    expect(hint(g)).toBeNull();
  });
});

describe('tempo', () => {
  it('formata como m:ss', () => {
    expect(fmtTime(0)).toBe('0:00');
    expect(fmtTime(65)).toBe('1:05');
    expect(fmtTime(600)).toBe('10:00');
    expect(fmtTime(null)).toBe('—');
  });
});
