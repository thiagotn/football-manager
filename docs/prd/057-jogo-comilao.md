# PRD 057 — Comilão

- **Status:** ✅ No ar desde 2026-11-10
- **Código:** [`games/src/lib/comilao/`](../../games/src/lib/comilao/) e [`games/src/routes/comilao/`](../../games/src/routes/comilao/) — estado corrente em [`games/CLAUDE.md`](../../games/CLAUDE.md)
- **Antecessores:** [PRD 053](053-jogo-enche-ai.md) criou o app `games/`; [PRD 054](054-migracao-paciencia-tetris-games.md) generalizou a infra; [PRD 056](056-jogo-sudoku.md) foi o último jogo novo

## Problema / Objetivo

O hub tinha cinco jogos — puzzle de despejo, cartas, encaixe 3D, cobrinha e sudoku — e
faltava o arcade de labirinto, que é o clássico mais reconhecível de todos. O pedido foi
"um pacman no games.thiagotn.com": app estático, offline-first, sem backend, com desafio
diário e progressão no `localStorage`.

**Pac-Man é marca registrada da Bandai Namco** — o nome, os personagens, os nomes dos
fantasmas e o traçado do labirinto original. O jogo entra então como clone com identidade
própria, como "Cobrinha" é para Snake: nome **Comilão** (en *Muncher*, es *Comilón*),
labirinto desenhado aqui, paleta synthwave e fantasmas desenhados aqui. As **mecânicas**
(comer pastilhas, pastilhão que assusta, quatro perseguidores com personalidades
diferentes, scatter/chase, travessia lateral, fruta) não são protegidas — e são
exatamente o que faz o jogo ser esse jogo.

## Decisões (2026-11-10)

| # | Decisão | Por quê |
|---|---|---|
| 1 | **Nome próprio: Comilão** | Marca registrada. O nome segue o padrão do hub ("Cobrinha", "Enche Aí!") e já diz o que o jogo é |
| 2 | **Um labirinto afinado**, com as fases apertando | Escolha do usuário. Um traçado bem equilibrado joga melhor que quatro medianos, e a dificuldade vem da velocidade e da pressão, não de trocar o desenho |
| 3 | **19 × 21 tiles** (o arcade original é 28 × 31) | Mobile-first: num aparelho de 390 px isso dá ~20 px por tile em vez de ~13. Pastilha de 20 px dá para ver e para acertar |
| 4 | **Paleta synthwave** com paredes de traço, não de bloco | Escolha do usuário. Parede fina com brilho deixa a silhueta cheia dos fantasmas ganhar a leitura; o ajuste **Brilho neon** desliga o glow |
| 5 | **Cores dos fantasmas fora do visual** | São elas que dizem quem cola, quem embosca e quem desiste. Num jogo em que meio segundo decide a vida, isso não é tema trocável |
| 6 | **A partida é salva** (como no Sudoku) | Três vidas levam minutos. Retomar cai sempre no "Prepare-se" — voltar no colo de um fantasma seria roubo |
| 7 | **Passo fixo de 16,67 ms** | A física não depende da taxa de quadros, e o teste roda uma partida inteira comparando números exatos |

## Como ficou

| Arquivo | Conteúdo |
|---|---|
| `src/lib/comilao/maze.ts` | O labirinto 19 × 21 como desenho legível (`#` parede · `.` pastilha · `o` pastilhão · `-` porta), com berço, casa dos fantasmas, travessia e 198 pastilhas |
| `src/lib/comilao/maze.test.ts` | 11 testes que **aprovam ou reprovam o traçado**: borda, simetria, alcance, beco, porta e contagem |
| `src/lib/comilao/engine.ts` | Regras puras: posição contínua em tiles, curva no centro, quatro personalidades de perseguição, ondas scatter/chase, susto com combo, fruta, vidas e desafio do dia |
| `src/lib/comilao/engine.test.ts` | Vitest, 44 testes: movimento, parede, travessia, susto, combo, morte, fase, fruta, alvos, **determinismo do dia** e o teste que quebra se o engine tocar em `Math.random` |
| `src/lib/comilao/skins.ts` + `skins.test.ts` | Os 6 visuais do labirinto e a liberação por pontuação somada; as cores dos fantasmas moram aqui, fixas |
| `src/lib/comilao/render.ts` | Canvas 2D: fundo e esqueleto de parede desenhados uma vez fora da tela, atores e pastilhas por quadro, números que sobem ao comer |
| `src/lib/comilao/storage.ts` | `games_comilao`: partida em andamento, estatísticas de sempre, histórico (60), dias do desafio e ajustes |
| `src/routes/comilao/+page.svelte` | Abas Início · Diário · Visuais · Ajustes, partida em tela cheia, HUD com vidas, arraste, direcional e teclado |

**Regras.** Comilão anda sozinho até a parede; a curva pedida fica guardada e entra na
primeira esquina que couber — é o que perdoa o toque antecipado no celular. Pastilha vale
10, pastilhão 50, fruta de 100 a 5 000 conforme a fase, e fantasma comido paga
200 · 400 · 800 · 1 600 **no mesmo susto**. Três vidas, vida extra a cada 10 000 pontos.
A travessia lateral dá a volta, e é lá que o fantasma arrasta o pé (45 % da velocidade).

**Os quatro fantasmas.** No centro de cada tile cada um escolhe, entre as saídas livres
(menos a volta atrás), a que mais aproxima do **seu** alvo — desempate em ordem fixa, então
a perseguição é determinística e testável:

| Fantasma | Alvo | Como joga |
|---|---|---|
| Caçador | o tile do Comilão | cola no jogador, é a pressão que não alivia |
| Emboscador | 4 tiles à frente do Comilão | corta o caminho em vez de seguir atrás |
| Flanqueador | o caçador espelhado através de 2 tiles à frente do herói | fecha o outro lado do corredor |
| Ermitão | o Comilão quando longe, seu canto quando a menos de 8 tiles | o imprevisível, e é ele que dá o respiro |

As ondas alternam fuga e caça (7 s/20 s no começo, encurtando), e **toda troca de onda
inverte os fantasmas** — é o alívio que torna o jogo jogável. No susto eles ficam lentos,
invertem na hora e sorteiam a curva pelo PRNG guardado no estado.

**Dificuldade.** `heroSpeed` vai de 7,6 a 9,4 tiles/s ao longo das fases e os fantasmas de
0,76 a 1,0 da velocidade do herói. O susto encolhe de 7 s a 1 s e **acaba na fase 19**: dali
em diante o pastilhão ainda pontua e inverte, mas não assusta mais. O ajuste **Ritmo**
(calmo/normal/veloz) mexe só na velocidade dos fantasmas, e o desafio do dia o ignora.

**Desafio do dia.** `fnv1a('comilao-' + dia)` semeia a partida e `dailyGoal` dá a meta de
pontos — mesma partida e mesma meta em qualquer aparelho. A meta fica entre 1 200 e 2 400:
limpar a fase 1 inteira dá ~2 300 pontos, então é meta de partida boa, não de partida
perfeita.

**Visuais.** Liberam por **pontuação somada** (0 / 1 500 / 4 000 / 8 000 / 14 000 / 22 000):
`synth`, `vapor`, `laser`, `brasa`, `gelo` e `arco`. Cada um troca parede, brilho, piso,
herói e pastilha — nunca os fantasmas. A aba Visuais traz a legenda **Quem é quem**,
explicando o que cada cor faz.

**Controles.** Arraste em qualquer ponto do labirinto (22 px já contam), direcional na tela
(pode ser escondido) e teclado: setas ou WASD andam, Espaço/`P`/Esc pausam.

## Integração

- `src/lib/games.ts`: entrada `comilao` com o selo **Novo** (que saiu do Sudoku), tags
  Arcade · Desafio diário · Funciona offline — nenhuma chave `hub.*` nova.
- `tailwind.config.js`: token `synth` (`DEFAULT`/`2`/`light`) para a casca roxa.
- i18n: 96 chaves `comilao.*` nos três idiomas (**Comilão · Muncher · Comilón**), visuais em
  `comilao.skin.<chave>` e fantasmas em `comilao.ghost.<id>`/`comilao.ghost_desc.<id>`.
- PWA: `static/comilao/manifest.webmanifest` (`id`/`scope`/`start_url` = `/comilao/`) e os
  quatro PNGs gerados de `scripts/icons/comilao{,-maskable}.svg`.

## Testes

- `maze.test.ts` — 11 testes. É o teste que **aprova o labirinto**: 19 × 21, borda fechada
  menos as duas bocas da travessia, simetria, *flood fill* alcançando toda pastilha, o
  Comilão sem entrar na casa, **nenhum corredor com uma saída só**, porta única com saída
  acima e berço abaixo, e a contagem batendo. Na primeira execução ele reprovou o traçado:
  o tile acima da porta tinha uma saída só — quem entrasse ali morria encurralado.
- `engine.test.ts` — 44 testes, rodando partidas em passos fixos: anda e para na parede,
  curva guardada, curva recusada, volta atrás, travessia, pastilhão (inclusive a fase 19,
  que não assusta mais), combo 200/400/800/1 600, fantasma comido voltando para casa,
  morte, fim de partida, vida extra, virada de fase, fruta nos dois gatilhos, alvo de cada
  personalidade, inversão na troca de onda, a casa esvaziando, **mesmo dia ⇒ mesma
  partida** e o teste que quebra se o engine encostar em `Math.random`.
- `skins.test.ts` — 8 testes: metas em ordem, liberação, próximo a liberar, queda para o
  padrão, cores presentes, um fantasma por cor, e a persistência (retomar, estatísticas,
  histórico, dia e sequência).
- `football-e2e/tests/test_comilao.py` — 25 testes: card no hub com o selo Novo, as quatro
  abas, "Prepare-se" segurando a partida, andar e pontuar, direcional, pausa, sair e
  retomar caindo no "Prepare-se", **fim de partida** e **fantasma comido** (semeando a
  última vida e o susto no estado salvo, em vez de esperar minutos de jogo), calendário,
  visual bloqueado × liberado, Quem é quem, ajustes, zerar estatísticas pelo
  `ConfirmDialog`, manifesto e **offline com o service worker**.
- `.github/workflows/games.yml` passa a rodar os seis arquivos de E2E.

## Medições

- **`tick` custa 0,4 µs** por quadro neste micro (20 000 passos medidos) — 0,002 % do
  orçamento de 16,7 ms. O gasto real do jogo é o desenho, e o fundo com as 400 arestas de
  parede é pintado **uma vez** por tamanho de tile e visual.
- **Equilíbrio, medido em simulação**: parado no berço, o jogador é pego em **12,9 s**; um
  bot bobo de um passo de visão sobrevive **3 minutos** sem perder vida. Em 60 s de partida
  com comandos aleatórios, **nenhum** ator apareceu dentro de parede.
- **Build do `games/`**: 2,91 MB no total (`du -sb build`), contra 2,39 MB antes. A rota
  `/comilao/` pesa **50 KB** de JS (**17,5 KB** com gzip) — engine, labirinto, desenho e
  tela, sem dependência nova — mais 332 KB dos quatro PNGs de ícone, que é o grosso do
  acréscimo.

## Riscos aceitos

- **Fantasmas sem X-wing nem cadeia de pensamento**: a perseguição é a regra clássica de
  "mais perto do alvo" em cada centro de tile. É forte e legível, mas um jogador de arcade
  veterano vai achar padrões e abusar deles. É o preço de um perseguidor determinístico —
  e é o mesmo preço que o arcade original paga.
- **Neon contra fantasma**: a paleta synthwave põe parede brilhante perto de bicho
  colorido. A mitigação está no desenho (parede de traço fino, fantasma sólido em cor fora
  da faixa das paredes) e no ajuste **Brilho neon**. Se ainda incomodar, trocar o visual
  padrão para `gelo` é mexer num objeto de `skins.ts`.
- **Labirinto único**: quem jogar muito vai decorar o traçado. Foi a escolha de formato; um
  segundo labirinto entra sem mexer no engine, porque `maze.ts` é dado, não código.
- **Progresso só no navegador** e **sem placar entre jogadores**: limpar os dados do site
  apaga recordes, histórico e visuais, e o "de sempre" é do próprio aparelho — o `games/`
  não tem backend. Vale para todos os jogos do hub.
