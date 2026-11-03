# PRD 053 — Enche Aí! (puzzle de ordenar sabores, PWA em games.thiagotn.com)

- **Status:** 🚧 Implementado em `games/` + card no `/games` do rachao.app, aguardando deploy (2026-11-03)
- **Código:** [`games/`](../../games/) — estado corrente em [`games/CLAUDE.md`](../../games/CLAUDE.md)
- **Protótipo jogável + especificação visual:** [`docs/design/enche-ai/prototipo.html`](../design/enche-ai/prototipo.html) (HTML único, abre direto no navegador)
- **Referência de stack:** Paciência (`/paciencia`, `$lib/paciencia/*`) e Tetris 3D (`/tetris`) do `football-frontend`

## Problema / Objetivo

O `/games` do rachao.app já tem Paciência e Tetris 3D: client-side, offline e sem backend.
Queremos um terceiro jogo casual, rápido de jogar no celular, e começar a separar os jogos
num domínio próprio (`games.thiagotn.com`), onde cada jogo é instalável como PWA independente.

**Enche Aí!** é um puzzle de ordenação (gênero "sort puzzle") com tema de lanchonete:
recipientes com camadas misturadas de bebidas e sobremesas; o jogador despeja de um para outro
até cada recipiente ter um sabor só.

## Decisões de produto

| Decisão | Valor |
|---|---|
| Regra base | Recipiente com 4 camadas. Despeja só sobre o mesmo sabor ou num recipiente vazio; move a sequência inteira do topo até onde couber |
| Copo pronto | 4 camadas iguais → trava, ganha finalização (vapor / canudo + rodela / chantilly + cereja), risca o sabor na comanda, vibra 30 ms (opcional) |
| Vitória | Todos os recipientes prontos ou vazios |
| Sabores | 12, cada um com **cor + textura** (acessibilidade p/ daltonismo; textura pode ser desligada nos ajustes): café (grãos), leite (liso), chocolate (ondas), laranja (polpa), limonada (rodelas), melancia (caroços), maracujá (sementes), menta (listras), morango (pontinhos), açaí (granola), refri de uva (bolhas), raspadinha (gelo) |
| Cardápios (mundos) | Padaria · caneca de vidro (1–10) · Lanchonete · copo americano (11–20) · Casa de Sucos · copo long (21–30) · Sorveteria · taça (31–40). Depois do 40 o ciclo recomeça |
| Dificuldade | `sabores = min(9, 3 + floor((pedido-1)/4))` + 2 recipientes vazios |
| Geração | Seed FNV-1a (`enche-ai-<pedido>`) → embaralha (mulberry32) → rejeita recipientes com 3+ camadas iguais → **solver valida** antes de exibir. Nenhuma fase sem solução chega ao jogador |
| Pedido do dia | Seed por data local (`enche-ai-diario-YYYY-MM-DD`), 7 sabores, cardápio pelo dia da semana — igual para todos, como o desafio diário da Paciência |
| Ajudas (por pedido) | 3 desfazer · 1 copo extra · 3 dicas (dica = 1º passo do solver a partir do estado atual). Sem anúncios, sem compra |
| Estrelas | 3 sem ajuda; −1 por **tipo** de ajuda usada; mínimo 1 |
| Sem saída | Sem jogada válida → toast sugerindo desfazer ou copo extra |
| Telas | Jogo (HUD: início, recomeçar, desfazer, copo extra; título do pedido; pedido do dia; dica; comanda; balcão) · Vitória · Início (continuar, pedido do dia, cardápios, ajustes) |

## Direção visual

- Lanchonete de bairro à noite: parede de **azulejo azul** no topo (onde fica o HUD), **balcão de madeira** como tabuleiro, **comanda de papel** com borda picotada listando os sabores.
- Destaques: mostarda (`#ffc93c`) para contadores/CTA, ketchup (`#e8473b`) para o pedido do dia.
- Tipografia: Lilita One (títulos) + Nunito (texto). Tema único escuro (é um jogo; não segue o tema do sistema).
- Animação de despejo: recipiente de origem levanta, viaja até o destino, inclina 72°, filete da cor do sabor, camadas crescem de baixo para cima (`scaleY` com `transform-box: fill-box`). Respeita `prefers-reduced-motion`.
- Recipientes desenhados em SVG (contorno + `clipPath` interno + camadas como `rect` + `pattern` de textura). Nada de imagem bitmap: o jogo inteiro pesa poucos KB além das fontes.

## Arquitetura / Stack

Mesmo padrão dos jogos atuais — lógica pura em TS, estado em `localStorage`, zero backend —
mas como **app próprio** publicado em `games.thiagotn.com`.

| Camada | Escolha |
|---|---|
| Framework | SvelteKit 2 + Svelte 5 + TypeScript + Tailwind (igual ao `football-frontend`) |
| Build | `@sveltejs/adapter-static`; rotas de jogo com `ssr = false` + `prerender = true` |
| Lógica | `src/lib/enche-ai/engine.ts` — tipos, `canPour`, `pour`, `isDone`, `isWon`, `generate(seed, sabores)`, `solve(estado)`; sem DOM, testável com Vitest |
| Conteúdo | `src/lib/enche-ai/content.ts` — sabores, recipientes (paths SVG), cardápios |
| Estado | `src/lib/enche-ai/storage.ts` — chave `games_enche_ai`: pedido atual, progresso/estrelas por pedido, recordes do pedido do dia, ajustes. Try/catch em toda leitura/escrita (mesmo padrão de `$lib/paciencia/storage.ts`) |
| Componentes | `Vessel.svelte` (SVG do recipiente), `Comanda.svelte`, `WinDialog.svelte` |
| PWA | Service worker **nativo do SvelteKit** (`$service-worker`: precache do build + cache-first). Escolhido no lugar do `vite-plugin-pwa`/Workbox do rachao.app porque, com manifesto por jogo em `static/`, o plugin não agrega nada e só adicionaria dependências |
| Instalação por jogo | Um `manifest.webmanifest` por jogo em `static/<jogo>/` com `id`, `scope` e `start_url` = `/<jogo>/`, linkado no `<svelte:head>` da rota. O hub `/` tem o manifesto próprio. Um único SW na raiz cobre todos |
| i18n | `messages/{pt-BR,en,es}.json`, chaves `enche_ai.*` (mesmo fluxo do CLAUDE.md raiz) |
| Hospedagem | Cloudflare Pages, domínio `games.thiagotn.com`; preview por branch |
| rachao.app | Card novo em `/games` apontando para `https://games.thiagotn.com/enche-ai/` (link externo). Paciência/Tetris podem migrar depois |
| Testes | Vitest: solver resolve todas as fases 1–99 + pedido do dia de 365 datas; regras de `pour`. Playwright (`football-e2e/tests/test_enche_ai.py`): jogada válida, jogada inválida (shake), desfazer, vitória forçada via `localStorage`, persistência |

### Estrutura sugerida

```
games/                         # novo app, irmão de football-frontend
  src/
    lib/
      enche-ai/
        engine.ts
        storage.ts
        content.ts
        Vessel.svelte
    routes/
      +page.svelte             # hub de jogos
      enche-ai/+page.svelte    # tela do jogo (fixed inset-0, sem navbar)
      enche-ai/+page.ts        # ssr = false, prerender = true
    service-worker.ts
  static/
    enche-ai/manifest.webmanifest
    enche-ai/icon-192.png · icon-512.png · maskable-512.png
  messages/ pt-BR.json · en.json · es.json
  svelte.config.js             # adapter-static
  vite.config.ts               # sveltekit() + vitest
```

### Exemplo de manifesto por jogo

```json
{
  "id": "/enche-ai/",
  "name": "Enche Aí!",
  "short_name": "Enche Aí",
  "start_url": "/enche-ai/",
  "scope": "/enche-ai/",
  "display": "standalone",
  "orientation": "portrait",
  "background_color": "#1f130c",
  "theme_color": "#183a69",
  "icons": [
    { "src": "/enche-ai/icon-192.png", "sizes": "192x192", "type": "image/png" },
    { "src": "/enche-ai/icon-512.png", "sizes": "512x512", "type": "image/png" },
    { "src": "/enche-ai/maskable-512.png", "sizes": "512x512", "type": "image/png", "purpose": "maskable" }
  ]
}
```

## Fora de escopo (v1)

- Ranking online / login — o jogo segue sem backend. Se vier, reaproveita a API do rachao.app em PRD próprio.
- Sons (previstos para v2, com botão de mudo).
- Monetização / anúncios.

## Decisões tomadas (2026-11-02)

1. `games/` é pasta neste monorepo (reaproveita `football-e2e` e o fluxo de i18n).
2. Só o Enche Aí! na v1; Paciência e Tetris continuam em `rachao.app/games` (o hub linka para lá).
3. Nome mantido: **Enche Aí!** (en: *Fill It Up!*, es: *¡Llénalo!*).
4. Ícone provisório gerado de `games/scripts/icons/*.svg`.
5. Card no `rachao.app/games` só depois do deploy de `games.thiagotn.com`.

## Pendências

- Criar projeto no Cloudflare Pages + DNS de `games.thiagotn.com` (passos em `games/README.md`).
  Até isso existir, o card do `/games` aponta para um domínio que ainda não responde —
  o merge do card deve acompanhar o deploy.

## Concluído depois da implementação

- Card do Enche Aí! em `football-frontend/src/routes/games/+page.svelte` apontando para
  `https://games.thiagotn.com/enche-ai/` (link externo, `target="_blank"`), com a arte do
  balcão em SVG, chaves `games.enche_ai_*`/`games.tag_puzzle` nos 3 idiomas e teste E2E
  em `football-e2e/tests/test_games.py`. O selo "Novo" passou da Paciência para ele.
