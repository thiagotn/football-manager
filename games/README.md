# games.thiagotn.com

Jogos casuais em PWA — estáticos, offline, instaláveis um a um.

| Jogo | Rota |
|------|------|
| Enche Aí! — puzzle de ordenar sabores | `/enche-ai/` |
| Paciência — Klondike | `/paciencia/` |
| Tetris 3D | `/tetris/` |

Detalhes de arquitetura em [`CLAUDE.md`](CLAUDE.md); produto nos PRDs
[053](../docs/prd/053-jogo-enche-ai.md) (o app e o Enche Aí!) e
[054](../docs/prd/054-migracao-paciencia-tetris-games.md) (a vinda da Paciência
e do Tetris, que moravam no rachao.app).

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
