# games/ — Estado atual

> App dos jogos publicados em **games.thiagotn.com** (separado do rachao.app).
> Padrões gerais (i18n, Svelte 5, mobile-first) estão no `CLAUDE.md` da raiz.
> Atualizar este arquivo ao criar rota, jogo ou componente.

## Stack

- SvelteKit 2 + Svelte 5 + TypeScript + Tailwind 3 (mesmas versões do `football-frontend`)
- `@sveltejs/adapter-static` → `build/` 100% estático, tudo pré-renderizado (`trailingSlash: 'always'`)
- Service worker **nativo do SvelteKit** (`src/service-worker.ts`, `$service-worker`): precache do build + cache-first → offline após a 1ª visita.
  Os arquivos de config do Pages (`_headers`, `_redirects`, `_routes.json`) moram em `static/` mas o Pages **não os serve** (404), então ficam fora do precache;
  a instalação cacheia asset por asset (`Promise.allSettled`) para que um 404 não derrube o service worker inteiro — ver `test_service_worker_instala_e_o_jogo_abre_offline`
- Um **manifesto por jogo** em `static/<jogo>/manifest.webmanifest` (`id`/`scope`/`start_url` = `/<jogo>/`), linkado no `<svelte:head>` da rota → cada jogo instala como app próprio. O hub `/` tem `static/manifest.webmanifest`
- Sem backend: estado em `localStorage` (`games_<jogo>`)
- Fontes self-hosted via `@fontsource` (Lilita One + Nunito) — entram no precache.
  A Paciência mantém a **Bebas Neue** que usava no rachao.app (CSS importado na rota, `font-bebas`)
- `three` (ESM, só o Tetris usa) entra no chunk da rota `/tetris/`: ~545 KB com hash e cache longo
- Tokens de cor por jogo no `tailwind.config.js`: `ink`/`mustard`/`paper` (base), `felt` (Paciência), `night` (Tetris), `lagoon` (Cobrinha), `news` (Sudoku), `synth` (Comilão)

## Rotas

| Rota | Descrição |
|------|-----------|
| `/` | Hub de jogos: itera `$lib/games.ts` e monta um `GameCard` por jogo |
| `/enche-ai/` | **Enche Aí!** — puzzle de ordenar sabores. Tela cheia (`fixed inset-0`), `ssr = false`. Overlays internos: Início (cardápios, pedido do dia, ajustes, idioma) e Vitória |
| `/paciencia/` | **Paciência** Klondike (veio do rachao.app no PRD 054). Abas internas: Início · Diário · Partidas · Ajustes. Move por toque ou arraste |
| `/cobrinha/` | **Cobrinha** clássica em grade 15×19 (PRD 055). Abas internas: Início · Diário · Visuais · Ajustes. Arraste, direcional ou teclado |
| `/sudoku/` | **Sudoku** 9×9 em papel de jornal (PRD 056). Abas internas: Início · Diário · Visuais · Ajustes. Toque, teclado numérico na tela e teclado físico |
| `/comilao/` | **Comilão** — labirinto 19×21 em neon synthwave (PRD 057). Abas internas: Início · Diário · Visuais · Ajustes. Arraste, direcional ou teclado |
| `/tetris/` | **Tetris 3D** (veio do rachao.app no PRD 054). Canvas three.js + HUD em Svelte; teclado, botões táteis e, com o 3D ligado, arraste para girar a câmera |

## Compartilhado

| Arquivo | Conteúdo |
|---------|----------|
| `src/lib/games.ts` | Registro dos jogos (slug, chaves i18n, tags, `theme_color`). **Jogo novo entra aqui** — o hub é montado a partir dele |
| `src/lib/components/GameCard.svelte` | Card do hub; recebe o `Game` e o componente de arte do jogo |
| `src/lib/components/LanguageSwitcher.svelte` | PT/EN/ES (`tone="paper"` em card claro, `"dark"` no hub) |
| `src/lib/components/ConfirmDialog.svelte` | Confirmação destrutiva sobre `.overlay`/`.card` |
| `src/app.css` | `@layer components` com as classes de todos os jogos: `.btn`/`.btn-primary`/`.btn-ghost`/`.btn-danger`, `.card`, `.overlay`, `.tag`, `.toggle`, `.toast`, `.stat`, `.stat-label`, `.section-label`, `.menu-item` |

## Paciência (`src/lib/paciencia/`)

| Arquivo | Conteúdo |
|---------|----------|
| `engine.ts` | Regras puras do Klondike: `deal` (seed determinística do desafio diário), `resolveTap`, `draggableCards`/`canDrop`/`applyDrop`, `drawFromStock`, `findHint`, `autoFinishStep`, `winStars` |
| `storage.ts` | localStorage `games_paciencia`: partida, histórico (60), desafios diários e ajustes |
| `backs.ts` | Os 4 versos (`classico`, `dourado`, `feltro`, `noturno`) |
| `Emblem.svelte` | Espadilha num anel — marca do jogo (cabeçalho, verso das cartas, Início) |
| `WinShell.svelte` | Moldura 300×360 da vitória (clip-path + borda mostarda) |
| `CardArt.svelte` | Arte do card no hub |

## Cobrinha (`src/lib/cobrinha/`)

| Arquivo | Conteúdo |
|---------|----------|
| `engine.ts` | Regras puras: `step` imutável, `turn` (fila de 2 curvas, sem inversão), fruta e bônus sorteadas por PRNG guardado no estado, `wallsFor`/`dailyGoal` (desafio do dia), `levelFor`/`tickMs`, `fmtScore` |
| `engine.test.ts` | Vitest: movimento, comer, morte, travessia, ritmo e **determinismo do desafio do dia** |
| `skins.ts` + `skins.test.ts` | Os 6 visuais (cor por segmento) e a liberação por pontuação acumulada (`stats.total`) |
| `storage.ts` | localStorage `games_cobrinha`: estatísticas de sempre, histórico (60), dias do desafio e ajustes. A partida em andamento **não** é salva |
| `render.ts` | Canvas 2D: fundo (gradiente + hexágonos) num canvas fora da tela, cobra interpolada entre passos, fruta/bônus com pulso; `dispose()` ao sair da rota |
| `Emblem.svelte` · `CardArt.svelte` | Marca do jogo e arte do card no hub |

- A cobra só anda depois do primeiro comando (`started`) — ninguém perde antes de pegar no controle
- O desafio do dia ignora os ajustes do jogador (velocidade e travessia), senão o resultado não seria comparável

## Sudoku (`src/lib/sudoku/`)

| Arquivo | Conteúdo |
|---------|----------|
| `engine.ts` | Regras puras: grade de 81 posições, lances imutáveis (`place`/`toggleNote`/`erase`/`undo`), anotações em máscara de 9 bits, `solveLogically` (escada de técnicas), `hardnessOf`, `generate` (sorteio + unicidade + classificação), `hint`, `dailyGame`, `toLine`/`fromLine` |
| `engine.test.ts` | Vitest: unicidade, faixa de dados, classificação por dificuldade, **determinismo do dia** e um teste que quebra se o gerador tocar em `Math.random` |
| `themes.ts` + `themes.test.ts` | Os 6 visuais do tabuleiro (11 cores cada) e a liberação por partidas concluídas |
| `storage.ts` | localStorage `games_sudoku`: **partida em andamento**, estatísticas por dificuldade, histórico (60), dias do desafio e ajustes |
| `Emblem.svelte` · `CardArt.svelte` | Marca do jogo e arte do card no hub |

- Dificuldade é **técnica exigida**, não quantidade de dados: com 24 dados, 19 em 30 tabuleiros saem só com singles. O gerador classifica e descarta o que não bate com a faixa (teto de 40 tentativas, depois aceita o mais duro que apareceu)
- Gerar custa 2–44 ms: a tela mostra "Montando o tabuleiro…" e gera depois de dois `requestAnimationFrame`; o próximo da mesma faixa sai no `requestIdleCallback`
- A partida **é** salva (ao contrário da Cobrinha) — sudoku leva minutos
- O `ConfirmDialog` mora **dentro** da casca `fixed inset-0 z-50`: o `.overlay` é `position:absolute` e, fora dela, fica atrás do jogo

## Comilão (`src/lib/comilao/`)

| Arquivo | Conteúdo |
|---------|----------|
| `maze.ts` | O labirinto 19×21 como desenho legível (`#` parede · `.` pastilha · `o` pastilhão · `-` porta), berço, casa dos fantasmas, travessia e as 198 pastilhas |
| `maze.test.ts` | **É o teste que aprova o traçado**: borda, simetria, flood fill alcançando toda pastilha, nenhum beco de uma saída, porta única, contagem |
| `engine.ts` | Regras puras: posição **contínua** em tiles (não é passo a passo na grade), curva aplicada no centro do tile, 4 personalidades de perseguição, ondas scatter/chase, susto com combo, fruta, vidas e desafio do dia |
| `engine.test.ts` | Vitest: partidas rodadas em passos fixos, com números exatos; **determinismo do dia** e um teste que quebra se o engine tocar em `Math.random` |
| `skins.ts` + `skins.test.ts` | Os 6 visuais do labirinto (parede, brilho, piso, herói, pastilha) e a liberação por pontuação somada. As **cores dos fantasmas moram aqui e são fixas** |
| `render.ts` | Canvas 2D: fundo + esqueleto das paredes num canvas fora da tela (repintado só quando muda o tile, o visual ou o brilho); atores, pastilhas e os números que sobem por quadro; `dispose()` ao sair da rota |
| `storage.ts` | localStorage `games_comilao`: **partida em andamento**, estatísticas de sempre, histórico (60), dias do desafio e ajustes |
| `Emblem.svelte` · `CardArt.svelte` | Marca do jogo e arte do card no hub |

- `tick` é chamado **sempre com `dt` fixo** (16,67 ms, no máximo 4 por quadro): a física não depende da taxa de quadros e o teste compara números exatos
- A curva pedida **fica guardada** até caber — é o que perdoa o toque antecipado no celular
- A partida **é** salva (como no Sudoku), mas retomar cai sempre em `phase: 'pronto'`: voltar no meio de um corredor com um fantasma a dois tiles seria roubo
- Gradiente em SVG de card/ícone precisa de `gradientUnits="userSpaceOnUse"`: em `objectBoundingBox` uma linha horizontal tem caixa de altura zero e **não pinta nada**
- Medido: `tick` custa 0,4 µs; parado no berço o jogador é pego em 12,9 s; um bot de um passo de visão sobrevive 3 min

## Tetris 3D (`src/lib/tetris/`)

| Arquivo | Conteúdo |
|---------|----------|
| `engine.ts` | Regras puras: `shapeOf`, `isValid`, `ghostRow`, `lock` (imutável), `clearLines`, `rotate` (com kicks), `scoreForLines`, `levelFor`, `dropIntervalMs` |
| `engine.test.ts` | Vitest: rotação, colisão, limpeza de linhas, pontuação, fim de jogo |
| `scene.ts` | three.js isolado: poço, luzes, estrelas, 21 materiais criados uma vez e pool de cubos; as duas câmeras (orbital e plana) e `setDepth`; `dispose()` libera tudo ao sair da rota |
| `storage.ts` | localStorage `games_tetris`: só `{ settings: { depth }, asked }` — o jogo não guarda partida nem recorde |
| `storage.test.ts` | Vitest: padrão desligado, ida e volta, json quebrado e valor estranho caindo no padrão |
| `CardArt.svelte` | Arte do card no hub — **não importa three** (senão o chunk vai para o bundle do hub) |

- **O 3D é opcional e vem desligado.** Ligado: câmera em perspectiva orbitando o poço, arraste gira a arena. Desligado: câmera **ortográfica** de frente (`FLAT_MARGIN` enquadra o poço em qualquer tela), `orbit()` não faz nada e a arena não tem como ser movida — o toque só rotaciona a peça
- A pergunta "Jogar em 3D?" aparece **uma vez**, ao tocar Jogar sem `asked` salvo; depois a chave fica no overlay de início e no de pausa
- `canvas[data-depth]` existe para o E2E: o modo é observável no DOM, a câmera não (a trava do arraste foi medida à mão no navegador — arrastar no plano muda 0% dos pixels)

## Enche Aí! (`src/lib/enche-ai/`)

| Arquivo | Conteúdo |
|---------|----------|
| `engine.ts` | Regras puras (`canPour`, `applyPour`, `isComplete`, `isWon`, `hasMove`), gerador seedado (`generate`, FNV-1a + mulberry32), `solve` (DFS com memo — valida fases e dá dicas), `starsFor`, constantes (`CAPACITY=4`, `AIDS`) |
| `engine.test.ts` | Vitest: regras + **pedidos 1–99 e 365 pedidos do dia têm solução** |
| `content.ts` | 12 sabores (cor + textura + finalização), 4 recipientes (geometria SVG), 4 cardápios, `levelConfig`/`dailyConfig` |
| `storage.ts` | localStorage `games_enche_ai`: `current` (partida + histórico), `level` (maior liberado), `stars` por pedido, `daily` por data, `settings` |
| `Vessel.svelte` | SVG do recipiente: vidro + camadas + textura + finalização (vapor/canudo/chantilly); `growFrom` anima o despejo |
| `Patterns.svelte` | `<defs>` compartilhados (texturas `ea-p-*`, clips `ea-clip-*`) — 1 por página |
| `FlavorDot.svelte` | Bolinha do sabor (comanda) |

- Textos: chaves `enche_ai.*` e `hub.*` em `messages/{pt-BR,en,es}.json`; nomes de sabor/cardápio/recipiente em `enche_ai.flavor.<id>`, `enche_ai.world.<id>`, `enche_ai.vessel.<id>`
- Classe `ea-no-pat` no container esconde as texturas (ajuste de acessibilidade)
- Design de referência: `docs/design/enche-ai/prototipo.html` · PRD: `docs/prd/053-jogo-enche-ai.md`

## Comandos

```bash
npm run dev       # http://localhost:3100
npm run check     # svelte-check
npm test          # vitest (engine)
npm run build     # build/ estático
npm run icons     # regera PNGs de scripts/icons/*.svg (precisa de Playwright)
```

E2E: `football-e2e/tests/test_{cobrinha,comilao,enche_ai,paciencia,sudoku,tetris}.py` (usam `GAMES_BASE_URL`, padrão `http://localhost:4173` do `vite preview`; são pulados se o app não estiver no ar). CI: `.github/workflows/games.yml`.

## Novo jogo — checklist

1. `src/lib/<jogo>/` com `engine.ts` puro + testes, `storage.ts` (chave `games_<jogo>`)
2. `src/routes/<jogo>/+page.svelte` + `+page.ts` (só `ssr = false`; `prerender` vem do layout — sem pré-render não há `index.html`, manifesto nem offline)
3. `static/<jogo>/manifest.webmanifest` + `scripts/icons/<jogo>{,-maskable}.svg` + `npm run icons`
   (o Playwright de Node não está instalado aqui: na prática os PNGs saem de um script equivalente com o Playwright de Python do `football-e2e`)
4. Entrada em `src/lib/games.ts` e `CardArt.svelte` do jogo (o hub e o fallback offline do service worker se viram sozinhos)
5. Chaves i18n nos 3 idiomas e teste E2E em `football-e2e/tests/test_<jogo>.py` (fixar `games_locale`)
