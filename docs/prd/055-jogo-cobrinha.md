# PRD 055 — Cobrinha

- **Status:** ✅ No ar em `games.thiagotn.com` desde 2026-11-03
- **Código:** [`games/src/lib/cobrinha/`](../../games/src/lib/cobrinha/) e [`games/src/routes/cobrinha/`](../../games/src/routes/cobrinha/) — estado corrente em [`games/CLAUDE.md`](../../games/CLAUDE.md)
- **Antecessores:** [PRD 053](053-jogo-enche-ai.md) criou o app `games/`; [PRD 054](054-migracao-paciencia-tetris-games.md) generalizou a infra para N jogos

## Problema / Objetivo

O hub tem um puzzle (Enche Aí!), um jogo de cartas (Paciência) e um arcade de encaixe
(Tetris 3D) — falta o passatempo mais universal de todos, o da cobrinha. O pedido nasceu
das capturas do Snake.io: placar grande, progressão com visuais liberáveis e a tela de fim
de partida comparando **Jogo atual × De sempre**.

A referência é um jogo de arena com dezenas de cobras controladas por servidor. Aqui o app
é estático e offline-first, então o que entra é a **cobrinha clássica em grade** com a
mesma linguagem visual do hub.

## Decisões (2026-11-03)

| # | Decisão | Por quê |
|---|---|---|
| 1 | **Clássico em grade**, não a arena com bots | O app não tem backend; a arena pediria IA para ~25 cobras e um mundo com câmera, muito mais superfície para um jogo de bolso |
| 2 | **Estatísticas, visuais desbloqueáveis e desafio diário** | É o que dá motivo para voltar — e o diário já é a assinatura dos outros dois jogos do hub |
| 3 | **Paleta "lagoa noturna"** | A malha de hexágonos das capturas, mas em teal escuro: o hub e os outros jogos são todos escuros |
| 4 | **Canvas 2D**, sem dependência nova | O jogo todo cabe em ~200 linhas de desenho; nenhum KB a mais no precache |
| 5 | **A cobra só anda depois do primeiro comando** | Começar andando faz o jogador perder antes de pegar no controle |

## Como ficou

| Arquivo | Conteúdo |
|---|---|
| `src/lib/cobrinha/engine.ts` | Regras puras: grade 15×19, `step` imutável, fila de curvas, fruta e bônus, obstáculos, nível, pontuação, seed do dia |
| `src/lib/cobrinha/render.ts` | Canvas 2D: fundo (gradiente + hexágonos) desenhado uma vez fora da tela, cobra interpolada entre dois passos, fruta com pulso |
| `src/lib/cobrinha/skins.ts` | 6 visuais como função de cor por segmento (`folha`, `brasa`, `fita`, `gelo`, `neon`, `arco`) |
| `src/lib/cobrinha/storage.ts` | `games_cobrinha`: estatísticas de sempre, histórico (60), dias do desafio e ajustes |
| `src/lib/cobrinha/{Emblem,CardArt}.svelte` | Marca do jogo e arte do card no hub |
| `src/routes/cobrinha/+page.svelte` | Abas Início · Diário · Visuais · Ajustes, partida em tela cheia, laço do jogo e entrada |

**Regras.** Fruta vale 10 × nível; a cada 5 frutas nasce uma bônus que vale 50 × nível e
some em 40 passos. O nível sobe a cada 5 frutas e encurta o passo (180 ms → 85 ms na
velocidade normal). Morre na parede (ou atravessa, se o ajuste estiver ligado), na própria
cauda e nos obstáculos do desafio do dia.

**Desafio do dia.** A data vira seed (FNV-1a): mesma sequência de frutas, mesmos obstáculos
e mesma meta (`150 + (seed % 7) × 25`) para todo mundo. O dia é marcado quando a meta é
batida; os ajustes do jogador são ignorados na partida diária, senão o resultado não seria
comparável.

**Visuais.** Liberam pela **pontuação somada de todas as partidas** (0 / 500 / 1 500 /
3 500 / 7 000 / 12 000), com a barra de progresso que as capturas têm. Zerar estatísticas
bloqueia tudo de novo e volta ao visual padrão.

**Controles.** Arrastar em qualquer ponto do tabuleiro, direcional de 4 botões (pode ser
escondido nos ajustes) e setas/WASD no teclado. Espaço, `P` ou Esc pausam.

## Integração

- `src/lib/games.ts`: entrada `cobrinha` com o selo **Novo** (que saiu do Enche Aí!), tags
  Clássico · Desafio diário · Funciona offline.
- `tailwind.config.js`: tokens `lagoon` (`DEFAULT`/`2`/`light`).
- i18n: 76 chaves `cobrinha.*` + `hub.tag_classic` nos três idiomas (nome exibido:
  Cobrinha · Snake · Viborita).
- PWA: `static/cobrinha/manifest.webmanifest` (`id`/`scope`/`start_url` = `/cobrinha/`) e os
  quatro PNGs gerados de `scripts/icons/cobrinha{,-maskable}.svg`.

## Testes

- `src/lib/cobrinha/engine.test.ts` — 26 testes: movimento, fila de curvas e inversão
  proibida, crescer ao comer, bônus (nascimento, valor e expiração), morte na parede, na
  cauda e no obstáculo, travessia, ritmo por nível, formatação do placar e **determinismo do
  desafio do dia** (mesma data ⇒ mesma sequência de frutas).
- `src/lib/cobrinha/skins.test.ts` — 6 testes: ordem das metas, liberação por pontuação,
  próximo a liberar e queda para o visual padrão quando a chave salva não existe mais.
- `football-e2e/tests/test_cobrinha.py` — 19 testes: card no hub, abas, a cobra parada até o
  primeiro comando, fim de partida com a tabela Jogo atual × De sempre, persistência,
  pausa, desafio do dia, visual bloqueado × liberado, ajustes, zerar estatísticas,
  manifesto e **offline com o service worker**.
- `.github/workflows/games.yml` passa a rodar os quatro arquivos de E2E.

## Medições

- Build do `games/`: **2,07 MB** no total (`du -sb build`). A rota `/cobrinha/` pesa **22 KB**
  de JS — o jogo é canvas 2D puro, sem dependência nova — mais 207 KB dos quatro PNGs de
  ícone do PWA, que é o grosso do que ela acrescenta ao precache.

## Riscos aceitos

- **Rota em português** (`/cobrinha/`), como `/paciencia/` e `/enche-ai/`. Mudar depois
  custaria um redirect permanente, então fica registrado como decisão.
- **Progresso só no navegador**: limpar os dados do site apaga recordes e visuais. Vale para
  todos os jogos do hub.
- **Sem placar entre jogadores** — o "De sempre" é do próprio aparelho. Um ranking exigiria
  backend, que é justamente o que o `games/` não tem.
