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
- Sem backend: estado em `localStorage`
- Fontes self-hosted via `@fontsource` (Lilita One + Nunito) — entram no precache

## Rotas

| Rota | Descrição |
|------|-----------|
| `/` | Hub de jogos (cards). Link para `rachao.app/games` (Paciência e Tetris continuam lá) |
| `/enche-ai/` | **Enche Aí!** — puzzle de ordenar sabores. Tela cheia (`fixed inset-0`), `ssr = false`. Overlays internos: Início (cardápios, pedido do dia, ajustes, idioma) e Vitória |

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

E2E: `football-e2e/tests/test_enche_ai.py` (usa `GAMES_BASE_URL`, padrão `http://localhost:4173` do `vite preview`; é pulado se o app não estiver no ar). CI: `.github/workflows/games.yml`.

## Novo jogo — checklist

1. `src/lib/<jogo>/` com `engine.ts` puro + testes, `storage.ts` (chave `games_<jogo>`)
2. `src/routes/<jogo>/+page.svelte` + `+page.ts` (`ssr = false`)
3. `static/<jogo>/manifest.webmanifest` + ícones (192, 512, maskable 512, apple-touch 180)
4. Fallback offline da navegação em `src/service-worker.ts`
5. Card no hub (`src/routes/+page.svelte`), chaves i18n nos 3 idiomas, teste E2E
