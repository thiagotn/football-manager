# PRD 054 — Paciência e Tetris 3D mudam para games.thiagotn.com

- **Status:** ✅ No ar desde 2026-11-03
- **Código:** [`games/`](../../games/) — estado corrente em [`games/CLAUDE.md`](../../games/CLAUDE.md)
- **Antecessor:** [PRD 053](053-jogo-enche-ai.md) criou o app `games/` e o domínio próprio

## Problema / Objetivo

O PRD 053 tirou os jogos do caminho crítico do rachao.app, mas só o Enche Aí! nasceu lá.
Paciência e Tetris 3D continuavam dentro do `football-frontend`, o que significava: duas
casas para a mesma categoria de produto, jogos que não instalam como app próprio, e o
`three.js` (589 KB) viajando junto do bundle do rachao.app por causa de um jogo.

Este PRD termina a separação: **todos os jogos passam a viver em games.thiagotn.com** e o
rachao.app volta a ser só o app do rachão.

## Decisões (2026-11-03)

| # | Decisão | Por quê |
|---|---|---|
| 1 | **Migrar de vez**, com redirect 301 das rotas antigas | Uma base de código por jogo; os redirects seguram link salvo e PWA instalado |
| 2 | **Sem migração de progresso** | `localStorage` não cruza domínios; uma ponte custaria código dos dois lados por um ganho pequeno |
| 3 | **Re-skin neutro** | Sem marca do rachão: os jogos ganham a paleta e as fontes do hub |
| 4 | **`three` como dependência npm (ESM)** | O `three.min.js` UMD foi removido do pacote na r150 — não havia caminho de só atualizar |
| 5 | **Entrega única** | Jogos, redirects e limpeza no mesmo PR |
| 6 | **O `/games` do rachao.app sai de cena** | Conta › Passatempo aponta direto para o hub; a rota vira redirect |

## O que mudou

### No `games/`

| Área | Mudança |
|---|---|
| Infra | `src/lib/games.ts` (registro dos jogos), `components/GameCard.svelte`, `components/LanguageSwitcher.svelte`, `components/ConfirmDialog.svelte`; as classes compartilhadas (`.btn`, `.card`, `.overlay`, `.tag`, `.toggle`, `.toast`, `.stat*`) subiram para `app.css` |
| Service worker | O fallback de navegação passou a sair de `prerendered` (ordenado do mais específico) — jogo novo não exige mais editar o SW |
| Ícones | `scripts/render-icons.mjs` deriva os alvos da convenção `<slug>.svg` / `<slug>-maskable.svg`, sem tabela fixa |
| Paciência | `src/lib/paciencia/*` (engine, storage, backs, `Emblem`, `WinShell`, `CardArt`) e `routes/paciencia/`; chave do localStorage `games_paciencia` |
| Tetris | `src/lib/tetris/engine.ts` (regras puras + 24 testes), `scene.ts` (three isolado, materiais em pool, `dispose`), `routes/tetris/` |
| i18n | 77 → 186 chaves por idioma (`paciencia.*`, `tetris.*`, `common.*`, novas `hub.*`) |
| Tokens | `felt` (mesa de carteado) e `night` (poço do Tetris) no `tailwind.config.js` |

### No `football-frontend`

- `/games`, `/paciencia` e `/tetris` viram `redirect(301, …)` no padrão de `src/routes/draw/`.
  As três continuam em `PUBLIC_ROUTES` — sem isso o guard do layout mandaria para `/login`
  antes do redirect rodar.
- Navbar: Conta › Passatempo aponta para `https://games.thiagotn.com` (`target="_blank"`);
  saíram `GAME_ROUTES` e as entradas de `getBackHref`.
- Saíram: `src/lib/paciencia/`, `static/tetris-game/`, `static/banners/banner-campo.jpg`,
  98 chaves i18n por idioma, a dependência `three` **e o script `prebuild`** (existia só
  para copiar o UMD), a linha `static/three.min.js` do `.gitignore` e a dependência
  `@fontsource/bebas-neue` com o token `font-bebas`.
- `static/logo.png`, `ConfirmDialog.svelte` e `CrestShell.svelte` **ficam** — são usados por
  outras telas.

## Re-skin — o que virou o quê

| Antes (rachão) | Agora (games) |
|---|---|
| Bebas Neue (`font-bebas`) | Lilita One (`font-display`), com os títulos ~25% menores: a Lilita é bem mais larga |
| `gold-*` | `mustard` / `mustard-ink` |
| `primary-*` (verde de ação) | `mustard` nos CTAs, `felt` nos estados "ganhou/concluído" |
| `logo.png` (cabeçalho, verso das cartas, Início) | `Emblem.svelte` — espadilha num anel |
| `banner-campo.jpg` | gradiente do tema |
| Mesa listrada de gramado + linhas de campo | feltro liso com vinheta |
| Escudo da vitória (`CrestShell`) | `WinShell.svelte` — mesma geometria, moldura mostarda |
| Versos `escudo`/`gramado` | `classico`/`feltro` |
| "Rachão Edition", "Golaço!", "Vestiário", "Campanha" | saíram ou viraram texto neutro |

## Medições

- Build do `games/`: **948 KB → 2,1 MB** (o nó da rota `/tetris/` tem 545 KB com o three
  embutido, com hash e `max-age=31536000`).
- O service worker pré-cacheia tudo, então a primeira visita ao hub baixa o chunk do Tetris
  em segundo plano. Foi a escolha consciente: é o que faz os três jogos funcionarem offline.
  Se o build passar de ~3 MB, o próximo passo é cachear em duas fases (essencial no
  `install`, o resto depois do `activate`).

## Testes

- `games/src/lib/tetris/engine.test.ts`: 24 testes (rotação com kicks, colisão, `lock`
  imutável, limpeza de linhas, pontuação, ritmo de queda, fim de jogo).
- `football-e2e/tests/test_paciencia.py`: 19 testes portados do antigo `test_games.py`,
  incluindo os 5 de arrastar e soltar, mais manifesto e vitória.
- `football-e2e/tests/test_tetris.py`: 10 testes (HUD, teclado, botões táteis, pausa,
  manifesto, console limpo e **offline com o service worker** — o teste que protege a
  decisão de pré-cachear o chunk do three).
- `football-e2e/tests/test_games.py` encolheu para os três redirects + o link da Navbar.
- `.github/workflows/games.yml` roda os três arquivos de E2E.

## Riscos aceitos

- **Progresso perdido** (decisão 2): quem jogava Paciência no rachao.app recomeça do zero.
- **Redirects são para sempre**: 301 fica no cache do navegador; os arquivos de redirect
  não podem ser apagados depois.
- **`games.thiagotn.com` é domínio novo**, sem autoridade de busca. Se virar prioridade,
  cabe um `sitemap.xml` + `robots.txt` no `games/`.

## Ordem de deploy

1. Merge na `main` → o Cloudflare Pages publica o `games/` sozinho.
2. Conferir `games.thiagotn.com/paciencia/` e `/tetris/` no ar (e instalar os dois PWAs).
3. Só então rodar o pipeline do rachao.app (`workflow_dispatch`), que leva os redirects.

Na ordem inversa, o rachao.app redirecionaria para páginas que ainda não existem.
