# games.thiagotn.com

Jogos casuais em PWA — estáticos, offline, instaláveis um a um. Detalhes de
arquitetura em [`CLAUDE.md`](CLAUDE.md); produto em
[`docs/prd/053-jogo-enche-ai.md`](../docs/prd/053-jogo-enche-ai.md).

```bash
npm install
npm run dev        # http://localhost:3100
npm test && npm run check && npm run build
```

## Deploy — Cloudflare Pages

Criar um projeto Pages ligado a este repositório com:

| Campo | Valor |
|-------|-------|
| Production branch | `main` |
| Root directory | `games` |
| Build command | `npm ci && npm run build` |
| Build output directory | `build` |
| Variável de ambiente | `NODE_VERSION=24` |
| Custom domain | `games.thiagotn.com` (CNAME para `<projeto>.pages.dev`) |

`static/_headers` já define `no-cache` para o service worker e manifestos e
cache longo para `/_app/immutable/*`. Branches ganham preview automático em
`<branch>.<projeto>.pages.dev`.
