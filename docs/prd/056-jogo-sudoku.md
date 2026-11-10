# PRD 056 — Sudoku

- **Status:** ✅ No ar desde 2026-11-04
- **Código:** [`games/src/lib/sudoku/`](../../games/src/lib/sudoku/) e [`games/src/routes/sudoku/`](../../games/src/routes/sudoku/) — estado corrente em [`games/CLAUDE.md`](../../games/CLAUDE.md)
- **Antecessores:** [PRD 053](053-jogo-enche-ai.md) criou o app `games/`; [PRD 054](054-migracao-paciencia-tetris-games.md) generalizou a infra; [PRD 055](055-jogo-cobrinha.md) foi o último jogo novo

## Problema / Objetivo

O hub tem um puzzle de despejo (Enche Aí!), um jogo de cartas (Paciência), um arcade de
encaixe (Tetris 3D) e o clássico da cobrinha — falta o passatempo de lógica, que é o mais
procurado de todos. O pedido foi "um sudoku na temática dos demais": app estático,
offline-first, sem backend, com desafio diário e progressão no `localStorage`.

## Decisões (2026-11-04)

| # | Decisão | Por quê |
|---|---|---|
| 1 | **Quatro dificuldades geradas na hora** + tabuleiro do dia | É o formato clássico, e cada faixa tem estatística e recorde próprios — "melhor tempo" só quer dizer algo dentro da faixa |
| 2 | **Todos os auxílios**: lápis, desfazer, dica, checagem de erro | Sem anotação e sem desfazer, médio e difícil são injogáveis no celular; cada auxílio pode ser desligado nos ajustes |
| 3 | **Paleta "papel de jornal"** no tabuleiro, casca escura em volta | Escolha do usuário; a casca sepia mantém o jogo parente dos outros, que são todos escuros |
| 4 | **Dificuldade medida por técnica**, não por quantidade de dados | Contagem de dados engana: medindo, 19 de 30 tabuleiros com 24 dados saem só com "singles". O gerador classifica e descarta o que não bate com a faixa |
| 5 | **Sem web worker**: gerar fora de um `requestAnimationFrame` | Medido em 2–44 ms por tabuleiro; um worker custaria chunk extra no precache para economizar uns poucos quadros |
| 6 | **A partida é salva** (diferente da Cobrinha) | Um sudoku leva minutos: trocar de aba não pode custar o tabuleiro |

## Como ficou

| Arquivo | Conteúdo |
|---|---|
| `src/lib/sudoku/engine.ts` | Regras puras: grade 9×9 em arrays de 81, lances imutáveis (`place`/`toggleNote`/`erase`/`undo`), anotações em máscara de 9 bits, solver lógico em escada de técnicas, gerador seedado com unicidade garantida, dica e `dailyGame` |
| `src/lib/sudoku/engine.test.ts` | Vitest, 45 testes: unicidade, classificação por faixa, **determinismo do dia**, lances, desfazer, conflito, vitória e dica |
| `src/lib/sudoku/themes.ts` + `themes.test.ts` | Os 6 visuais do tabuleiro (11 cores cada) e a liberação por partidas concluídas |
| `src/lib/sudoku/storage.ts` | `games_sudoku`: partida em andamento, estatísticas por dificuldade, histórico (60), dias do desafio e ajustes |
| `src/lib/sudoku/{Emblem,CardArt}.svelte` | Marca do jogo (grade no anel mostarda) e arte do card no hub |
| `src/routes/sudoku/+page.svelte` | Abas Início · Diário · Visuais · Ajustes, partida em tela cheia, cronômetro, teclado numérico e entrada por toque e teclado |

**Regras.** Tocar a casa e depois o número (ou o contrário, com *número primeiro* ligado);
tocar o mesmo número de novo apaga. O cronômetro só começa no primeiro lance. Conflito com
linha, coluna ou bloco fica em vermelho; o número do jogador sai em laranja-tinta e o
impresso em preto.

**Dificuldade.** O gerador sorteia a solução, tira células enquanto a solução continuar
única e classifica o resultado pela técnica humana mais fraca que ainda o resolve
(1 singles · 2 candidato travado · 3 pares · 4 trincas · 5 além disso). Fácil e Médio
exigem tabuleiro que saia **só com singles** — a diferença entre eles é quantos dados ficam
(38–44 contra 30–34). Difícil (26–30 dados) exige que os singles **não** bastem e
Especialista (22–25) que nem pares e trincas fechem. Fácil e Médio ainda saem com simetria
de 180°, que é o desenho do sudoku impresso.

**Tabuleiro do dia.** A data vira seed (FNV-1a): mesmo tabuleiro para todo mundo. A
dificuldade sobe ao longo da semana — segunda e terça Fácil, quarta e quinta Médio, sexta e
sábado Difícil, domingo Especialista. O dia é marcado quando o tabuleiro fecha.

**Dica.** Roda o solver sobre o tabuleiro do jogador **sem os dígitos errados** (um número
errado envenenaria a dedução) e devolve a casa, o dígito e a técnica, que a tela mostra em
texto: "Nesta casa só cabe o 7". Quando nenhuma técnica aplica — o que acontece em todo
Especialista cedo ou tarde — revela a casa mais fechada e diz isso com todas as letras. Se
houver número errado na grade, a dica aponta o erro em vez de deduzir.

**Visuais.** Liberam por **partidas concluídas** (0 / 5 / 15 / 30 / 60 / 100), com a barra
de progresso: `jornal`, `linho`, `cafe`, `menta`, `ardosia` e `noturno` — os dois últimos
são tabuleiros escuros, para quem joga no escuro. Zerar estatísticas bloqueia tudo de novo.

**Controles.** Toque no tabuleiro e no teclado de 1 a 9 (com a contagem do que falta de cada
dígito), fila Desfazer · Lápis · Apagar · Dica, e no teclado físico 1–9 escreve,
0/Backspace apaga, setas movem, `N` alterna o lápis, `H` dá dica, `Z` desfaz e Espaço/`P`/Esc
pausam.

## Integração

- `src/lib/games.ts`: entrada `sudoku` com o selo **Novo** (que saiu da Cobrinha), tags
  Puzzle · Desafio diário · Funciona offline — nenhuma chave `hub.*` nova.
- `tailwind.config.js`: tokens `news` (`DEFAULT`/`2`/`light`) para a casca sepia.
- i18n: 95 chaves `sudoku.*` nos três idiomas (nome exibido: Sudoku nos três), nomes dos
  visuais em `sudoku.theme.<chave>` e as frases da dica em `sudoku.tech.<técnica>`.
- PWA: `static/sudoku/manifest.webmanifest` (`id`/`scope`/`start_url` = `/sudoku/`) e os
  quatro PNGs gerados de `scripts/icons/sudoku{,-maskable}.svg`.

## Testes

- `src/lib/sudoku/engine.test.ts` — 45 testes: vizinhos e casas, candidatos e conflitos,
  serialização em linha de 81 caracteres, unicidade dos quatro níveis, faixa de dados,
  **a classificação valendo em cada faixa**, mesma seed ⇒ mesmo tabuleiro, mesmo dia ⇒ mesmo
  tabuleiro, a dificuldade do dia pelo dia da semana, lances (escrever, apagar pelo
  toque-toque, lápis, autoNotes), desfazer (inclusive as anotações que o autoNotes comeu) e
  o teto da pilha, vitória, dica e `fmtTime`. Também há um teste que **quebra se o gerador
  encostar em `Math.random`** — é o que protege o determinismo do desafio do dia.
- `src/lib/sudoku/themes.test.ts` — 6 testes: ordem das metas, liberação por vitórias,
  próximo a liberar e queda para o padrão quando a chave salva não existe mais.
- `football-e2e/tests/test_sudoku.py` — 25 testes: card no hub, abas, partida, escrever,
  apagar, lápis, desfazer, conflito marcado, dica com explicação, pausa, sair e retomar,
  vitória com a tabela Partida × De sempre, calendário do dia, visual bloqueado × liberado,
  ajustes, número primeiro, zerar estatísticas pelo `ConfirmDialog`, manifesto e **offline
  com o service worker**.
- `.github/workflows/games.yml` passa a rodar os cinco arquivos de E2E.

## Medições

- Geração (neste micro, Node 24): **2–3 ms** no Fácil e no Médio, **18 ms** no Difícil e
  **38 ms** no Especialista, com pior caso perto de 65 ms. Num celular médio isso dá algo
  como 0,1–0,3 s, coberto pelo "Montando o tabuleiro…" e pelos dois quadros de espera; o
  próximo tabuleiro da mesma faixa ainda é gerado na folga (`requestIdleCallback`), então
  "Jogar de novo" é instantâneo.
- Build do `games/`: **2,39 MB** no total (`du -sb build`), contra 2,07 MB antes. A rota
  `/sudoku/` pesa **47 KB** de JS (16 KB com gzip) — engine, solver e tela, sem dependência
  nova — mais 252 KB dos quatro PNGs de ícone, que é o grosso do acréscimo.

## Riscos aceitos

- **Difícil e Especialista pedem técnicas que o motor não implementa** (X-wing e cadeias).
  O tabuleiro continua com solução única e dedutível, mas a dica nessas faixas revela a casa
  mais fechada em vez de ensinar a jogada — e avisa isso ao jogador.
- **Tabuleiro claro no hub escuro**: foi a paleta escolhida. A casca sepia em volta é o que
  mantém o jogo parecido com os outros, e quem preferir tudo escuro libera `ardosia` e
  `noturno`.
- **Progresso só no navegador**: limpar os dados do site apaga recordes, histórico e
  visuais. Vale para todos os jogos do hub.
- **Sem placar entre jogadores** — o "de sempre" é do próprio aparelho, porque o `games/`
  não tem backend.
