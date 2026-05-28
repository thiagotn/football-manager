# PRD 044 — `football-api-go`: Port da API rachao.app para Go

| Campo | Valor |
|---|---|
| **Versão** | 1.1 |
| **Status** | 🚧 Em implementação — revisado para entrega como **ambiente de homologação isolado** (banco-cópia + frontend `beta.rachao.app`) |
| **Autor** | thiagotn |
| **Data** | 2026-05-20 (rev. 2026-05-28) |

---

## 1. Visão Geral

### 1.1 Contexto

A API atual (`football-api/`) é construída em Python/FastAPI com SQLAlchemy async. Ela possui ~95 endpoints, 16 domínios e 43 migrations aplicadas. O objetivo deste PRD é criar uma versão equivalente em Go — o `football-api-go/` — como exploração da stack, benchmarking de desempenho e validação de paridade funcional, sem substituir a API Python existente.

A partir da v1.1, a entrega em produção deixa de ser uma "API paralela no mesmo banco" e passa a ser um **ambiente de homologação completo e isolado**: API Go (`api-go`) + **banco de dados dedicado** (cópia da produção) + **frontend próprio** em `beta.rachao.app`. O ambiente sobe em produção, é navegável de ponta a ponta e tem **zero impacto** nos usuários atuais — produção (`rachao.app` / `/api/v1`) permanece intacta.

### 1.2 Objetivo

Construir `football-api-go/` com paridade total de endpoints em relação à `football-api/`, de modo que o frontend de homologação consuma a API Go alterando apenas o prefixo da URL base (`/api/v1` → `/api/v2`), sem mudança de contrato de dados. A v2 e seu banco-cópia funcionam como um ambiente de homologação/staging desacoplado de produção.

### 1.3 Proposta de valor

- Avaliar ganhos de performance (latência, throughput, uso de memória) versus a stack Python atual
- Ambiente de homologação real (dados reais, copiados), sem risco de afetar produção
- Servir como base para decisão futura de migração parcial ou total de stack
- Documentar o processo de porting como referência para decisões de arquitetura futuras

---

## 2. Escopo

### 2.1 Incluído em v1.1

- Todos os ~97 endpoints dos 16 domínios atuais (paridade total de contrato HTTP)
- **Banco PostgreSQL dedicado** — cópia point-in-time da produção (criado pelo usuário no Supabase), conectado via novo secret `DATABASE_URL_HML`. Schema idêntico ao de produção
- **Frontend de homologação** publicado em `beta.rachao.app` (imagem Docker própria, build apontando para `/api/v2`)
- **Modo aberto de staging** (`APP_ENV=staging`): OTP por bypass code (sem SMS real), billing off no frontend, sem processamento de webhook Stripe; sem controle de acesso por usuário (isolamento é por ambiente)
- Documentação interativa via Mintlify em `docs.rachao.app` com OpenAPI playground e `/llms.txt`
- Testes unitários de handlers + testes de integração com banco real
- Dockerfile multi-stage (dev + production)
- GitHub Actions dedicado (`football-api-go/**`)
- Traefik routing: `/api/v2/` → Go API (sem alterar `/api/v1/` Python) + `beta.rachao.app` → frontend de homologação

### 2.2 Fora de escopo (v1.1)

- Substituir ou deprecar a `football-api/` Python em produção
- Alterar schema de banco em produção
- Sincronização automática contínua entre o banco-cópia e produção (estratégia = re-snapshot manual; ver §3.4 e §15)
- Isolamento do Supabase Storage (avatares) — o ambiente pode reutilizar o bucket de produção (ver §3.11)
- Funcionalidades novas não presentes na API Python
- Otimizações avançadas de Go (caching custom, profiling, etc.)

---

## 3. Decisões de Arquitetura

### 3.1 Framework HTTP — Chi

**Decisão: [Chi](https://github.com/go-chi/chi)**

| Opção | Prós | Contras |
|---|---|---|
| **Chi** ✅ | stdlib-compatible (`net/http`), leve (~1k LOC), composable middlewares | Menos batteries que Gin |
| Gin | Mais popular, binding integrado | Reflection-heavy, contexto próprio (não stdlib) |
| Echo | Performance similar ao Chi | Contexto próprio, menos idiomático |
| Fiber | Altíssima performance | Não usa `net/http` → incompatível com stdlib middleware e `httptest` |

Chi usa `net/http` puro, facilitando testes com `httptest.NewRecorder` sem adaptadores e compondo bem com qualquer middleware da stdlib.

### 3.2 Acesso ao banco — sqlc + pgx/v5

**Decisão: [sqlc](https://sqlc.dev/) + [pgx/v5](https://github.com/jackc/pgx)**

| Opção | Prós | Contras |
|---|---|---|
| **sqlc + pgx/v5** ✅ | Type-safe, sem reflection, SQL explícito, geração de código | Passo de build adicional (`sqlc generate`) |
| GORM | Familiar para devs ORM | Reflection, magia implícita, queries difíceis de debugar |
| sqlx | Mais controle que GORM | Sem type-safety automático |

sqlc gera código Go tipado a partir de arquivos SQL em `sql/queries/`, usando pgx/v5 como driver — o driver PostgreSQL mais performático em Go.

### 3.3 Roteamento de versões

**Decisão: `/api/v2/` no mesmo `api.rachao.app`, mas apontando para banco distinto**

```
https://api.rachao.app/api/v1/...  →  Python API  (container: api)     →  banco PRODUÇÃO
https://api.rachao.app/api/v2/...  →  Go API      (container: api-go)   →  banco CÓPIA (homologação)
https://beta.rachao.app/...        →  frontend de homologação (build com VITE_API_URL=.../api/v2)
https://rachao.app/...             →  frontend de produção (inalterado)
```

A regra Traefik `Host('api.rachao.app') && PathPrefix('/api/v2')` aponta para o container Go. Os dois serviços coexistem no mesmo docker-compose de produção, mas **leem bancos diferentes**: o `api` lê o banco de produção e o `api-go` lê o banco-cópia (via `DATABASE_URL_HML`). O frontend de homologação consome a v2 fixando `VITE_API_URL=https://api.rachao.app/api/v2` no build. Produção nunca chama `/api/v2`, logo o roteamento não tem impacto algum nos usuários atuais.

### 3.4 Banco de dados — dedicado (cópia de produção)

**Decisão: banco PostgreSQL dedicado, cópia point-in-time da produção**

- A API Go conecta a um banco **separado** do de produção, criado pelo usuário no Supabase como cópia do atual, via novo secret **`DATABASE_URL_HML`** (em vez de reutilizar `${DATABASE_URL}`)
- **Isolamento total**: nenhuma escrita da v2 (criar grupo, partida, presença, etc.) toca os dados de produção — garantia de "zero impacto"
- Schema idêntico ao de produção (a cópia nasce com as 43 migrations já aplicadas)
- **Sincronização de schema**: a Go API **não roda migrations** (apenas conecta — ver `internal/db/pool.go`). Migrations futuras aplicadas em produção **não** se propagam ao banco-cópia automaticamente. Estratégia: re-snapshot periódico da produção, ou aplicar manualmente no banco-cópia apenas as migrations novas surgidas em `football-api/migrations/` desde a última cópia (ver decisão D-06 em §13)
- Em ambiente local, o mesmo postgres do docker-compose é usado
- O passo a passo para criar o banco-cópia no Supabase está no **Apêndice §15**

### 3.5 Migrations — reuso das SQL existentes via golang-migrate

As 43 migrations em `football-api/migrations/*.sql` já estão aplicadas em produção e, por consequência, presentes no banco-cópia (que nasce de um snapshot da produção — ver §15). A Go API conecta ao banco-cópia **sem rodar migrations**. A propagação de migrations futuras ao banco-cópia é tratada em §3.4 / D-06. Em ambiente local, `make migrate` aplica as migrations usando [golang-migrate](https://github.com/golang-migrate/migrate) apontando para `../football-api/migrations/`.

### 3.6 Configuração — envconfig

**Decisão: [envconfig](https://github.com/kelseyhightower/envconfig)** + arquivo `.env`

Mesmas variáveis de ambiente da API Python (DATABASE_URL, SECRET_KEY, etc.) — sem novos secrets necessários em produção.

### 3.7 Testes — stdlib + testify + httptest

**Decisão: `testing` stdlib + [testify](https://github.com/stretchr/testify) + `net/http/httptest`**

- Handlers testados com `httptest.NewRecorder` + interfaces de service mockadas
- Integração com banco real via service container no CI (`postgres:16-alpine`)
- Cobertura reportada para Codecov com `flags: api-go`

### 3.8 Controle de acesso — por ambiente (sem flag por usuário)

**Decisão: nenhum gate por usuário; o isolamento é a nível de ambiente**

Como cada ambiente vive isoladamente (produção em `rachao.app`/`/api/v1`/banco de produção; homologação em `beta.rachao.app`/`/api/v2`/banco-cópia), **não há controle de acesso por usuário na v2**. Quem alcança `beta.rachao.app` está, por definição, no ambiente de homologação — o isolamento vem do subdomínio dedicado + banco-cópia, não de uma flag por jogador. O código legado do antigo gate (`api_v2_enabled`: middleware, endpoints admin, coluna, página de gerenciamento) é **removido por limpeza na Fase 7** (§9).

### 3.9 Documentação — Mintlify

**Decisão: [Mintlify](https://mintlify.com/) com OpenAPI interativo em `docs.rachao.app`**

| Opção | Prós | Contras |
|---|---|---|
| **Mintlify** ✅ | OpenAPI playground, `/llms.txt`, search, theming, sem ops de infra | Plano pago para domínio customizado |
| Markdown plain | Zero custo, simples | Sem playground, sem search, sem `/llms.txt` |
| Redoc / Swagger UI | Open-source, OpenAPI nativo | Só referência de API, sem narrativa/setup |

Mintlify é usado por Anthropic, Vercel, Coinbase e Linear. Gera `/llms.txt` automaticamente para compatibilidade com agentes de IA (como o próprio `chat.rachao.app`).

**Setup:**
- Diretório `football-api-go/mintlify/` com `mint.json`, páginas MDX e `openapi.yaml`
- `openapi.yaml` gerado automaticamente com `swaggo/swag` a partir de annotations nos handlers Go (`// @Summary`, `// @Param`, `// @Success`)
- `make docs` = `swag init` → `openapi.yaml` + `mintlify dev` para preview local
- Deploy contínuo: Mintlify Cloud monitora `football-api-go/mintlify/` via GitHub integration
- URL final: `docs.rachao.app` (CNAME para `mintlify.app`)

### 3.10 Frontend de homologação — `beta.rachao.app`

**Decisão: imagem Docker separada do frontend, com build apontando para a v2**

O frontend (`football-frontend`) usa `adapter-node` e lê `VITE_API_URL` via `import.meta.env` (`src/lib/api.ts`), que é **baked em build-time** no bundle. Por isso o ambiente de homologação exige uma **imagem própria**, construída com a URL da v2 — não dá para reaproveitar a imagem de produção em runtime.

| Aspecto | Valor (homologação) |
|---|---|
| **Imagem** | `ghcr.io/${owner}/football-manager-frontend-beta:latest` |
| **Build arg** `VITE_API_URL` | `https://api.rachao.app/api/v2` |
| **Build arg** `VITE_BILLING_ENABLED` | `false` (sem fluxo Stripe na UI) |
| **Build args** `PUBLIC_LEGAL_*` | iguais aos de produção |
| **Runtime** `ORIGIN` | `https://beta.rachao.app` |
| **Runtime** `API_INTERNAL_URL` | `http://api-go:8080/api/v2` (chamadas SSR container-a-container) |
| **Traefik** | router `Host('beta.rachao.app')` → `frontend-beta:3000`, TLS letsencrypt |
| **DNS** | registro `beta.rachao.app` → IP do VPS (pré-requisito do usuário) |

- **Isolamento de sessão automático**: o auth do frontend vive em `localStorage` (`token`, `refresh_token`, `player`), que é **por origin**. Como `beta.rachao.app` e `rachao.app` são origens distintas, as sessões não se misturam — nenhum login de homologação afeta produção.
- **Nota cosmética (não bloqueante)**: existem referências hardcoded a `https://status.rachao.app` (em `+layout.svelte`) e a share URL `https://rachao.app/players/{id}` que, a partir do beta, apontarão para produção. Aceitável; não será tratado nesta entrega.

### 3.11 Modo de homologação e isolamento de side-effects

**Decisão: `APP_ENV=staging` no `api-go`, garantindo "zero impacto em produção"**

| Lever | Efeito |
|---|---|
| `APP_ENV=staging` (≠ `production`) | Marca o ambiente como não-produtivo → habilita o OTP bypass |
| `OTP_BYPASS_CODE` setado | Registro/login no beta sem SMS real (Twilio não dispara) — `services/twilio.go` só bypassa quando `APP_ENV != "production"` |
| `VITE_BILLING_ENABLED=false` no frontend | Sem fluxo de checkout Stripe na UI |
| `STRIPE_WEBHOOK_SECRET` **ausente** no `api-go` | A rota `/api/v2/webhooks/payment` não é registrada → não processa eventos. Além disso, o Stripe só envia webhooks às URLs configuradas no painel (produção `/api/v1`), então `/api/v2` não receberia eventos de qualquer forma — manter unset é cinto + suspensório |
| `CORS_ORIGINS=https://beta.rachao.app` no `api-go` | Permite os fetches client-side do beta para `/api/v2` |
| `FRONTEND_URL=https://beta.rachao.app` no `api-go` | Links/redirects gerados pela API apontam para o beta |
| `SECRET_KEY` igual ao de produção | Sem novo secret; a validade cross-token entre v1/v2 é efeito colateral inofensivo em staging, não um objetivo |

---

## 4. Estrutura de Diretórios

```
football-api-go/
├── cmd/
│   └── server/
│       └── main.go                  # entrypoint: bootstrap, routes, listen
├── internal/
│   ├── config/
│   │   └── config.go                # envconfig struct (mirrors Python Settings)
│   ├── db/
│   │   ├── pool.go                  # pgx/v5 connection pool
│   │   └── queries/                 # código gerado pelo sqlc (não editar)
│   ├── middleware/
│   │   ├── auth.go                  # JWT parsing + MCP tokens → ctx player
│   │   ├── cors.go
│   │   ├── ratelimit.go             # rate limit por IP (login)
│   │   └── recovery.go
│   ├── handlers/                    # um arquivo por domínio
│   │   ├── auth.go                  # login, register, OTP, refresh, me, change-password
│   │   ├── groups.go                # CRUD grupos, membros, stats, waitlist
│   │   ├── matches.go               # CRUD partidas, attendance, discover, public
│   │   ├── players.go               # perfil, stats, avatar, admin
│   │   ├── teams.go                 # sorteio de times (snake-draft)
│   │   ├── votes.go                 # votação pós-partida
│   │   ├── finance.go               # controle financeiro por grupo
│   │   ├── subscriptions.go         # planos e Stripe checkout
│   │   ├── invites.go               # links de convite
│   │   ├── push.go                  # Web Push VAPID
│   │   ├── ranking.go               # ranking público
│   │   ├── reviews.go               # avaliações do app
│   │   ├── mcp_tokens.go            # tokens MCP pessoais
│   │   ├── chat.go                  # SSE + Anthropic API
│   │   ├── beta.go                  # Android beta signup
│   │   ├── webhooks.go              # Stripe webhook (HMAC)
│   │   └── admin.go                 # painel admin
│   ├── services/                    # lógica de negócio isolada dos handlers
│   │   ├── auth_service.go          # JWT HS256, bcrypt, OTP, refresh tokens
│   │   ├── team_builder.go          # algoritmo snake-draft (port de team_builder.py)
│   │   ├── billing.go               # limites de plano por tier
│   │   ├── billing_stripe.go        # Stripe checkout + webhook handlers
│   │   ├── push_service.go          # VAPID web push
│   │   ├── recurrence.go            # geração de partidas recorrentes
│   │   ├── storage.go               # Supabase Storage (avatar upload)
│   │   └── twilio.go                # OTP via Twilio Verify (WhatsApp/SMS)
│   └── apierror/
│       └── errors.go                # tipos padronizados: 404, 403, 409, 422, 429
├── sql/
│   └── queries/                     # arquivos .sql consumidos pelo sqlc
│       ├── players.sql
│       ├── groups.sql
│       ├── matches.sql
│       ├── attendances.sql
│       ├── teams.sql
│       ├── votes.sql
│       ├── finance.sql
│       ├── subscriptions.sql
│       ├── invites.sql
│       ├── push.sql
│       ├── reviews.sql
│       ├── mcp_tokens.sql
│       └── ranking.sql
├── tests/
│   ├── unit/
│   │   ├── helpers_test.go          # fixtures: fake player, mock services
│   │   ├── auth_test.go
│   │   ├── groups_test.go
│   │   ├── matches_test.go
│   │   └── ... (um por handler)
│   └── integration/
│       ├── auth_integration_test.go
│       ├── groups_integration_test.go
│       └── matches_integration_test.go
├── mintlify/
│   ├── mint.json                    # config: nav, colors, logo, baseUrl=docs.rachao.app
│   ├── openapi.yaml                 # gerado por `make docs` (swaggo/swag) — não editar manualmente
│   ├── quickstart.mdx               # pré-requisitos, .env, docker-compose, make run
│   ├── authentication.mdx           # JWT, MCP tokens
│   ├── architecture.mdx             # decisões de stack + diagrama de componentes
│   └── llms.txt                     # gerado automaticamente pelo Mintlify
├── Dockerfile                       # stage dev (air) + stage production (scratch)
├── docker-compose.yml               # postgres:16-alpine + api-go para dev local
├── .air.toml                        # configuração do live-reload (air)
├── sqlc.yaml                        # geração de código do banco
├── Makefile                         # run, test, test-integration, lint, generate, migrate
├── go.mod
├── go.sum
└── .env.example
```

---

## 5. Dependências Go (`go.mod`)

| Pacote | Versão | Propósito |
|---|---|---|
| `go-chi/chi/v5` | v5.x | Framework HTTP |
| `jackc/pgx/v5` | v5.x | Driver PostgreSQL |
| `sqlc-dev/sqlc` | v1.x | Geração de código SQL-safe (tool, não runtime) |
| `golang-jwt/jwt/v5` | v5.x | JWT HS256 |
| `golang.org/x/crypto` | latest | bcrypt |
| `kelseyhightower/envconfig` | v1.x | Configuração via env vars |
| `stretchr/testify` | v1.x | Assertions nos testes |
| `golang-migrate/migrate/v4` | v4.x | Aplicar migrations SQL em dev/ci |
| `stripe/stripe-go/v80` | v80.x | Stripe SDK |
| `joho/godotenv` | v1.x | Carregar `.env` em dev |
| `anthropics/anthropic-sdk-go` | latest | Anthropic API (chat/SSE) |
| `cosmtrek/air` | v1.x | Live-reload em dev (tool) |
| `swaggo/swag` | v1.x | Geração de `openapi.yaml` a partir de annotations nos handlers (tool) |

---

## 6. Inventário de Endpoints (`/api/v2`)

Todos os paths abaixo são espelhados da `football-api/` com prefixo `/api/v2` em vez de `/api/v1`.

### Auth (12 endpoints)
| Método | Path | Auth |
|---|---|---|
| POST | `/auth/login` | público |
| POST | `/auth/send-otp` | público |
| POST | `/auth/verify-otp` | público |
| POST | `/auth/register` | público |
| GET | `/auth/me` | JWT |
| POST | `/auth/forgot-password/send-otp` | público |
| POST | `/auth/forgot-password/verify-otp` | público |
| POST | `/auth/forgot-password/reset` | público |
| POST | `/auth/send-otp/me` | JWT |
| POST | `/auth/verify-otp/me` | JWT |
| POST | `/auth/change-password` | JWT |
| POST | `/auth/refresh` | público |

### Groups (14 endpoints)
| Método | Path | Auth |
|---|---|---|
| GET | `/groups` | JWT |
| POST | `/groups` | JWT |
| GET | `/groups/{id}` | JWT |
| PATCH | `/groups/{id}` | JWT (admin) |
| DELETE | `/groups/{id}` | super-admin |
| GET | `/groups/{id}/members` | JWT |
| POST | `/groups/{id}/members` | JWT (admin) |
| PATCH | `/groups/{id}/members/me` | JWT |
| PATCH | `/groups/{id}/members/{player_id}` | JWT |
| DELETE | `/groups/{id}/members/{player_id}` | JWT (admin) |
| GET | `/groups/{id}/members/lookup` | JWT (admin) |
| POST | `/groups/{id}/members/by-phone` | JWT (admin) |
| GET | `/groups/{id}/stats` | JWT |
| POST | `/groups/{id}/waitlist` | JWT |

### Matches (10 endpoints)
| Método | Path | Auth |
|---|---|---|
| GET | `/matches/discover` | opcional |
| GET | `/matches/public/{hash}` | público |
| GET | `/groups/{id}/matches` | JWT |
| POST | `/groups/{id}/matches` | JWT (admin) |
| GET | `/groups/{id}/matches/{match_id}` | JWT |
| PATCH | `/groups/{id}/matches/{match_id}` | JWT (admin) |
| DELETE | `/groups/{id}/matches/{match_id}` | JWT (admin) |
| GET | `/matches/public/{hash}/player-stats` | público |
| PUT | `/matches/{hash}/player-stats` | JWT (admin) |
| POST | `/groups/{id}/matches/{match_id}/attendance` | JWT |

### Players (12 endpoints)
| Método | Path | Auth |
|---|---|---|
| GET | `/players/me/matches` | JWT |
| GET | `/players/me/stats/full` | JWT |
| GET | `/players/me/stats` | JWT |
| GET | `/players` | super-admin |
| POST | `/players` | super-admin |
| GET | `/players/signups/stats` | super-admin |
| GET | `/players/{id}/public-stats` | público |
| GET | `/players/{id}` | JWT |
| PATCH | `/players/{id}` | JWT |
| POST | `/players/{id}/reset-password` | super-admin |
| PUT | `/players/me/avatar` | JWT |
| DELETE | `/players/me/avatar` | JWT |

### Teams (2 endpoints)
| Método | Path | Auth |
|---|---|---|
| POST | `/matches/{match_id}/teams` | JWT (admin) |
| GET | `/matches/{match_id}/teams` | público |

### Votes (3 endpoints)
| Método | Path | Auth |
|---|---|---|
| GET | `/matches/{match_id}/votes/status` | JWT |
| POST | `/matches/{match_id}/votes` | JWT |
| GET | `/votes/pending` | JWT |

### Finance (3 endpoints)
| Método | Path | Auth |
|---|---|---|
| GET | `/groups/{id}/finance/periods` | JWT |
| GET | `/groups/{id}/finance/periods/{year}/{month}` | JWT |
| PATCH | `/finance/payments/{payment_id}` | JWT |

### Subscriptions (2 endpoints)
| Método | Path | Auth |
|---|---|---|
| GET | `/subscriptions/me` | JWT |
| POST | `/subscriptions` | JWT |

### Invites (4 endpoints)
| Método | Path | Auth |
|---|---|---|
| POST | `/invites` | JWT (admin) |
| GET | `/invites/{token}` | público |
| GET | `/invites/{token}/check` | público |
| POST | `/invites/{token}/accept` | público |

### Push Notifications (3 endpoints)
| Método | Path | Auth |
|---|---|---|
| GET | `/push/vapid-public-key` | público |
| POST | `/push/subscribe` | JWT |
| DELETE | `/push/subscribe` | JWT |

### Ranking (1 endpoint)
| Método | Path | Auth |
|---|---|---|
| GET | `/ranking` | público |

### Reviews (4 endpoints)
| Método | Path | Auth |
|---|---|---|
| GET | `/reviews/me` | JWT |
| PUT | `/reviews/me` | JWT |
| GET | `/reviews/summary` | super-admin |
| GET | `/reviews` | super-admin |

### MCP Tokens (3 endpoints)
| Método | Path | Auth |
|---|---|---|
| POST | `/mcp-tokens` | JWT |
| GET | `/mcp-tokens` | JWT |
| DELETE | `/mcp-tokens/{id}` | JWT |

### Chat (3 endpoints)
| Método | Path | Auth |
|---|---|---|
| POST | `/chat` | JWT |
| GET | `/admin/chat-users` | super-admin |
| PATCH | `/admin/chat-users/{id}` | super-admin |

### Beta (1 endpoint)
| Método | Path | Auth |
|---|---|---|
| POST | `/beta/android-signup` | opcional |

### Webhooks (1 endpoint)
| Método | Path | Auth |
|---|---|---|
| POST | `/webhooks/payment` | HMAC Stripe |

### Admin (10 endpoints)
| Método | Path | Auth |
|---|---|---|
| GET | `/admin/stats` | super-admin |
| GET | `/admin/matches` | super-admin |
| GET | `/admin/groups` | super-admin |
| GET | `/admin/subscriptions/summary` | super-admin |
| GET | `/admin/subscriptions` | super-admin |
| PATCH | `/admin/subscriptions/{player_id}` | super-admin |
| POST | `/admin/subscriptions/{player_id}/cancel` | super-admin |
| GET | `/admin/players` | super-admin |
| DELETE | `/admin/players/{player_id}/avatar` | super-admin |
| GET | `/admin/beta-signups` | super-admin |

**Total: ~97 endpoints** (+ `GET /api/v2/health`)

---

## 7. Requisitos Funcionais

### RF-01 — Paridade de endpoints (Must)
Todos os ~97 endpoints acima devem estar implementados com o mesmo contrato HTTP: método, path, request body, response schema e HTTP status codes idênticos à versão Python.

### RF-02 — Prefixo `/api/v2` (Must)
Todos os routes montados sob `/api/v2/`. A constante `API_PREFIX = "/api/v2"` definida em `config.go`.

### RF-03 — Banco de dados dedicado (Must)
Conectar ao **banco-cópia** (homologação) via `DATABASE_URL_HML`, sem DDL adicional. Schema idêntico ao de produção. Nenhuma operação da v2 pode tocar o banco de produção.

### RF-04 — Formato de JWT compatível (Must)
A Go API usa o mesmo algoritmo (HS256) e `SECRET_KEY` da Python API, de modo que o formato de token é compatível. **Cada ambiente autentica contra o seu próprio banco** (prod vs. cópia), portanto a interoperabilidade de tokens entre v1 e v2 não é mais um objetivo de design — é apenas um efeito colateral inofensivo da chave compartilhada. O middleware de auth deve suportar também MCP tokens com prefixo `rachao_`.

### RF-05 — Documentação Mintlify (Must)
Documentação pública em `docs.rachao.app` com:
- `quickstart.mdx`: pré-requisitos (Go 1.24, Docker, sqlc, golangci-lint), `.env.example`, `make run`, `make test`, `make generate`
- `authentication.mdx`: JWT, MCP tokens
- `architecture.mdx`: decisões de stack, diagrama de componentes, comparativo com Python API
- `openapi.yaml` (gerado via `swaggo/swag`) integrado ao Mintlify com playground interativo para todos os endpoints
- `/llms.txt` gerado automaticamente pelo Mintlify (para compatibilidade com agentes IA)
- `make docs`: comando que roda `swag init` + abre `mintlify dev` localmente

### RF-06 — Testes unitários (Must)
Cada handler com ao menos: 1 teste de caminho feliz + testes dos erros documentados. Cobertura mínima: 70% das linhas dos pacotes `internal/handlers` e `internal/services`.

### RF-07 — Testes de integração (Should)
Domínios `auth`, `groups` e `matches` com testes de integração contra banco real (banco limpo por teste, usando `testing.T.Cleanup`).

### RF-08 — GitHub Actions dedicado (Must)
Workflow `.github/workflows/api-go.yml` acionado por mudanças em `football-api-go/**` ou `workflow_dispatch`. Jobs: `lint` → `unit-tests` → `integration-tests` → `build`.

### RF-09 — Dockerfile multi-stage (Must)
- Stage `dev`: [air](https://github.com/cosmtrek/air) para live-reload
- Stage `production`: binário estático em `scratch` ou `distroless`, usuário não-root, imagem ≤ 30MB

### RF-10 — Traefik routing (Must)
Regras em `traefik-dynamic.yml` e serviços em `docker-compose.prod.yml` para: (a) rotear `Host('api.rachao.app') && PathPrefix('/api/v2')` → container `api-go:8080` (já existe); (b) rotear `Host('beta.rachao.app')` → container `frontend-beta:3000`. O `/api/v1/` e `rachao.app` permanecem inalterados.

### RF-11 — Chat/SSE com Anthropic (Should)
`POST /api/v2/chat` com Server-Sent Events usando o mesmo MCP server existente (`mcp.rachao.app`). Rate limit de 20 mensagens/hora por usuário via coluna `chat_req_count` + `chat_req_window`.

### RF-12 — Makefile (Should)
Comandos: `make run`, `make test`, `make test-integration`, `make lint`, `make generate`, `make migrate`, `make build`, `make docs`.

---

## 8. Requisitos Não-Funcionais

### RNF-01 — Performance
Latência p95 ≤ 50ms para endpoints de leitura simples (benchmark referência da Python API: ~80–120ms). Throughput mínimo: 500 req/s em `GET /api/v2/health` com 1 worker.

### RNF-02 — Compatibilidade de contrato
Campos JSON obrigatórios byte-compatíveis com a Python API. Campos `null` vs. omitidos devem seguir o mesmo contrato (evitar breaking changes nos clientes).

### RNF-03 — Segurança
- bcrypt cost 12 para senhas
- JWT HS256 com `exp` de 15 minutos (access) e rotação de refresh tokens
- HMAC-SHA256 para verificação de webhooks Stripe
- Rate limit no login: 5 tentativas/IP/15 min → 429

### RNF-04 — Imagem Docker final
≤ 30MB (binário estático com CGO desabilitado em `scratch` ou `gcr.io/distroless/static`).

---

## 9. Plano de Rollout

### Fase 1 — Scaffolding e Auth ✅
- [x] Criar `football-api-go/` com estrutura de diretórios completa
- [x] `go.mod` com todas as dependências declaradas
- [x] `config.go` (envconfig, mirrors Python `Settings`)
- [x] Conexão pgx pool + `GET /api/v2/health`
- [x] Middleware: CORS, recovery, rate-limiter por IP
- [x] Middleware de auth: JWT HS256 + MCP tokens `rachao_*`
- [x] Handler `auth.go` — todos os 12 endpoints
- [x] `internal/db/queries.go` — camada de queries hand-crafted (substitui sqlc output para Fase 1)
- [x] `sql/queries/players.sql` + `sql/queries/auth.sql` — anotadas para futura geração sqlc
- [x] Testes unitários de auth (11 testes: login, register, refresh, OTP, get-me)
- [x] Dockerfile (dev + production/scratch)
- [x] `docker-compose.yml` local (postgres + api-go)
- [x] `.air.toml`, `sqlc.yaml`, `.golangci.yml`, `.env.example`, `.gitignore`
- [x] `mintlify/quickstart.mdx` — setup local completo

### Fase 2 — Core domain ✅
- [x] Handler `groups.go` (14 endpoints)
- [x] Handler `matches.go` (10 endpoints)
- [x] Handler `players.go` (12 endpoints)
- [x] Handler `invites.go` (4 endpoints)
- [x] `services/team_builder.go` (port do snake-draft Python)
- [x] Handler `teams.go` (2 endpoints)
- [x] Testes unitários: `groups_test.go`, `matches_test.go`, `players_test.go`, `team_builder_test.go`
- [x] Testes de integração: `auth_integration_test.go`, `groups_integration_test.go`, `matches_integration_test.go`

### Fase 3 — Domínios secundários
- [x] Handler `votes.go` (7 endpoints)
- [x] Handler `finance.go` (3 endpoints)
- [x] Handler `subscriptions.go` + `services/billing_stripe.go`
- [x] Handler `webhooks.go` (HMAC Stripe)
- [x] Handler `push.go` + `services/push_service.go` (VAPID)
- [x] Handler `ranking.go`
- [x] Handler `reviews.go`
- [x] Handler `mcp_tokens.go`
- [x] Handler `beta.go`
- [x] Testes unitários para cada handler

### Fase 4 — Admin e Chat
- [x] Handler `admin.go` (10 endpoints — stats, matches, groups, subscriptions CRUD, players, avatar, beta-signups)
- [x] Handler `chat.go` (SSE + Anthropic direct HTTP com MCP beta)
- [x] `services/twilio.go` (OTP via Twilio Verify)
- [x] `services/storage.go` (Supabase avatar upload/delete)
- [x] `services/recurrence.go` (geração de partidas recorrentes + status sync job)
- [x] Testes dos handlers restantes (93 testes no total)

### Fase 5 — CI/CD e Produção ✅ Fase 5 completa
- [x] `.github/workflows/api-go.yml` (lint + unit + integration + build + push GHCR)
- [x] Adicionar serviço `api-go` em `football-api/docker-compose.prod.yml`
- [x] Adicionar router `/api/v2` em `football-api/traefik-dynamic.yml`
- [x] Push image para GHCR: `ghcr.io/thiagotn/football-manager-api-go` (via api-go.yml)
- [x] Atualizar deploy job em `main.yml` para incluir `api-go` (unit-tests-go job + build step + ANTHROPIC_API_KEY/LLM_MODEL)
- [ ] Anotar handlers Go com `swaggo/swag` (`// @Summary`, `// @Param`, `// @Success`, `// @Router`)
- [x] Configurar `mintlify/mint.json` com branding rachao.app (cores, logo, nav)
- [x] Criar `mintlify/authentication.mdx` e `mintlify/architecture.mdx`
- [ ] `make docs` gera `openapi.yaml` atualizado
- [ ] Conectar repositório ao Mintlify Cloud → deploy em `docs.rachao.app`

### Fase 6 — Ambiente de homologação (banco-cópia + `beta.rachao.app`)
- [ ] Usuário cria o banco-cópia no Supabase (snapshot da produção) e fornece `DATABASE_URL_HML` (ver Apêndice §15)
- [ ] Usuário cria registro DNS `beta.rachao.app` → IP do VPS (Traefik emite cert letsencrypt)
- [ ] Adicionar secrets no GitHub Actions: `DATABASE_URL_HML` e `OTP_BYPASS_CODE`
- [ ] `docker-compose.prod.yml`: alterar serviço `api-go` para usar `DATABASE_URL_HML`, `APP_ENV=staging`, `OTP_BYPASS_CODE`, `CORS_ORIGINS=https://beta.rachao.app`, `FRONTEND_URL=https://beta.rachao.app`; remover `STRIPE_WEBHOOK_SECRET` do `api-go`
- [ ] `docker-compose.prod.yml`: adicionar serviço `frontend-beta`
- [ ] `traefik-dynamic.yml`: adicionar router/serviço `Host('beta.rachao.app')` → `frontend-beta:3000`
- [ ] `main.yml`: injetar `DATABASE_URL_HML` e `OTP_BYPASS_CODE` no `.env.prod`; adicionar build+push da imagem `football-manager-frontend-beta` com os build args da v2 (`VITE_API_URL=https://api.rachao.app/api/v2`, `VITE_BILLING_ENABLED=false`)
- [ ] Validar navegação ponta-a-ponta em `beta.rachao.app` sem afetar produção

### Fase 7 — Remoção do gate `api_v2_enabled` (limpeza de código morto)
Com o controle de acesso agora por ambiente (§3.8), o gate por usuário virou código morto. **Ordem segura:** remover o código primeiro (deploy sem ler a coluna) e só então dropar a coluna do banco.

**`football-api-go` (Go):**
- [ ] Deletar `internal/middleware/api_v2_access.go` e o teste `tests/unit/middleware_test.go`
- [ ] `internal/server/router.go` — remover `apiV2Mw := middleware.ApiV2AccessFor(...)` e sua aplicação ao grupo `/api/v2` (e comentários relacionados)
- [ ] `internal/handlers/admin.go` — remover `GET /admin/api-v2-users` e `PATCH /admin/api-v2-users/{playerID}` (`listApiV2Users`, `toggleApiV2User`, `toggleApiV2Req`) + métodos de store `ListApiV2Users` / `UpdatePlayerApiV2Enabled`
- [ ] `internal/services/auth_service.go` — remover o campo `ApiV2Enabled` dos structs de player/claims
- [ ] `internal/handlers/chat.go` — remover `ApiV2Enabled` do response de `/admin/chat-users`
- [ ] `internal/db/queries.go` — remover campo `ApiV2Enabled`, funções `UpdatePlayerApiV2Enabled` / `ListPlayersForApiV2` / `ListApiV2Users`, tipos `PlayerApiV2Row` / `ApiV2User`, e a coluna `api_v2_enabled` de todos os SELECT/INSERT/scan
- [ ] `sql/queries/players.sql` e `sql/queries/auth.sql` — remover `api_v2_enabled` das queries
- [ ] Limpar referências nos testes: `tests/unit/{phase4,services_pure,helpers}_test.go` e `tests/integration/{setup,auth_integration,admin_integration}_test.go`
- [ ] Docs Go: `README.md`, `mintlify/{architecture,authentication}.mdx`, `mintlify/openapi.yaml`

**`football-frontend`:**
- [ ] Deletar a página `src/routes/admin/api-v2/+page.svelte`
- [ ] `src/routes/admin/+page.svelte` — remover o card/link para `/admin/api-v2`
- [ ] `src/lib/api.ts` — remover `apiV2Admin` e os tipos `ApiV2UserItem` / `ApiV2UsersResponse`
- [ ] Remover chaves i18n relacionadas, se houver (`messages/{pt-BR,en,es}.json`)
- [ ] `football-frontend/CLAUDE.md` — remover menção à rota `/admin/api-v2`

**`football-api` (Python) — banco:**
- [ ] Criar nova migration `NNN_drop_api_v2_enabled.sql` com `ALTER TABLE players DROP COLUMN IF EXISTS api_v2_enabled;` (consultar `football-api/CLAUDE.md` para o próximo número). A `044_api_v2_enabled.sql` permanece como histórico — migrations aplicadas não se editam. **Atenção:** é DDL destrutiva que roda em produção e na cópia; a coluna é não-usada após a Fase 7, então o drop é seguro. Aplicar **depois** do deploy do código acima.

---

## 10. GitHub Actions — `api-go.yml`

```yaml
name: Build & Test API Go

on:
  push:
    paths: ['football-api-go/**']
  workflow_dispatch:

jobs:
  lint:
    name: Lint
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5
      - uses: actions/setup-go@v5
        with: { go-version: '1.24' }
      - uses: golangci/golangci-lint-action@v8
        with: { working-directory: football-api-go }

  unit-tests:
    name: Unit Tests
    runs-on: ubuntu-latest
    needs: lint
    steps:
      - uses: actions/checkout@v5
      - uses: actions/setup-go@v5
        with: { go-version: '1.24' }
      - name: Run unit tests
        working-directory: football-api-go
        env:
          DATABASE_URL: "postgres://fake:fake@localhost/fake"
          SECRET_KEY: "test-secret-key-ci-only"
        run: go test ./internal/... -coverprofile=coverage.out -covermode=atomic
      - uses: codecov/codecov-action@v6
        with:
          token: ${{ secrets.CODECOV_TOKEN }}
          files: football-api-go/coverage.out
          flags: api-go
          fail_ci_if_error: false

  integration-tests:
    name: Integration Tests
    runs-on: ubuntu-latest
    needs: unit-tests
    services:
      postgres:
        image: postgres:16-alpine
        env:
          POSTGRES_DB: rachao_test
          POSTGRES_USER: postgres
          POSTGRES_PASSWORD: postgres
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
    steps:
      - uses: actions/checkout@v5
      - uses: actions/setup-go@v5
        with: { go-version: '1.24' }
      - name: Apply migrations
        run: |
          go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
          migrate -path football-api/migrations \
            -database "postgres://postgres:postgres@localhost/rachao_test?sslmode=disable" up
      - name: Run integration tests
        working-directory: football-api-go
        env:
          DATABASE_URL: "postgres://postgres:postgres@localhost/rachao_test?sslmode=disable"
          SECRET_KEY: "test-secret-key-ci-only"
        run: go test ./tests/integration/... -v -timeout 120s

  build:
    name: Build & Push Image
    runs-on: ubuntu-latest
    needs: [unit-tests, integration-tests]
    permissions:
      contents: read
      packages: write
    steps:
      - uses: actions/checkout@v5
      - uses: docker/setup-buildx-action@v4
      - uses: docker/login-action@v4
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      - uses: docker/build-push-action@v7
        with:
          context: ./football-api-go
          target: production
          push: true
          tags: |
            ghcr.io/${{ github.repository_owner }}/football-manager-api-go:latest
            ghcr.io/${{ github.repository_owner }}/football-manager-api-go:${{ github.sha }}
          cache-from: type=gha,scope=api-go
          cache-to: type=gha,scope=api-go,mode=max
```

---

## 11. Traefik e docker-compose.prod.yml

> O router `api-go` já existe em `traefik-dynamic.yml` e o serviço `api-go` já existe em
> `docker-compose.prod.yml` (Fase 5). A Fase 6 muda o **banco/ambiente** do `api-go` e
> **adiciona** o frontend de homologação.

**`football-api/traefik-dynamic.yml`** — router `api-go` (já existente, inalterado) + novo router `frontend-beta`:

```yaml
http:
  routers:
    api-go:                                   # já existe — inalterado
      rule: "Host(`api.rachao.app`) && PathPrefix(`/api/v2`)"
      service: api-go
      tls:
        certResolver: letsencrypt
    frontend-beta:                            # NOVO
      rule: "Host(`beta.rachao.app`)"
      service: frontend-beta
      tls:
        certResolver: letsencrypt

  services:
    api-go:                                   # já existe — inalterado
      loadBalancer:
        servers:
          - url: "http://api-go:8080"
    frontend-beta:                            # NOVO
      loadBalancer:
        servers:
          - url: "http://frontend-beta:3000"
```

**`football-api/docker-compose.prod.yml`** — `api-go` revisado (aponta para o banco-cópia + modo staging):

```yaml
api-go:
  image: ghcr.io/${GITHUB_REPOSITORY_OWNER:-thiagotn}/football-manager-api-go:latest
  environment:
    APP_ENV: staging                          # ambiente não-produtivo → habilita OTP bypass
    DATABASE_URL: ${DATABASE_URL_HML}         # banco-CÓPIA, não o de produção
    SECRET_KEY: ${SECRET_KEY}
    CORS_ORIGINS: https://beta.rachao.app
    FRONTEND_URL: https://beta.rachao.app
    OTP_BYPASS_CODE: ${OTP_BYPASS_CODE}       # registro/login sem SMS real
    TWILIO_ACCOUNT_SID: ${TWILIO_ACCOUNT_SID}
    TWILIO_AUTH_TOKEN: ${TWILIO_AUTH_TOKEN}
    TWILIO_VERIFY_SID: ${TWILIO_VERIFY_SID}
    SUPABASE_URL: ${SUPABASE_URL}
    SUPABASE_SERVICE_ROLE_KEY: ${SUPABASE_SERVICE_ROLE_KEY}
    ANTHROPIC_API_KEY: ${ANTHROPIC_API_KEY}
    LLM_MODEL: ${LLM_MODEL}
    STRIPE_SECRET_KEY: ${STRIPE_SECRET_KEY}
    # STRIPE_WEBHOOK_SECRET intencionalmente ausente → rota de webhook não é registrada
    VAPID_PRIVATE_KEY: ${VAPID_PRIVATE_KEY}
    VAPID_PUBLIC_KEY: ${VAPID_PUBLIC_KEY}
    VAPID_CLAIMS_EMAIL: ${VAPID_CLAIMS_EMAIL}
  networks: [app-net]
  restart: unless-stopped
```

**`football-api/docker-compose.prod.yml`** — novo serviço `frontend-beta`:

```yaml
frontend-beta:
  image: ghcr.io/${GITHUB_REPOSITORY_OWNER:-thiagotn}/football-manager-frontend-beta:latest
  environment:
    ORIGIN: https://beta.rachao.app
    API_INTERNAL_URL: http://api-go:8080/api/v2
  networks: [app-net]
  restart: unless-stopped
```

> O roteamento fica no arquivo `traefik-dynamic.yml` (file provider), não em labels — seguindo o
> padrão já adotado pelos demais serviços do projeto.

---

## 12. Critérios de Aceite

- [ ] `GET /api/v2/health` retorna `{"status":"ok"}` com HTTP 200
- [ ] `api-go` em produção conecta ao **banco-cópia** (`DATABASE_URL_HML`), não ao de produção
- [ ] Uma escrita feita via v2/beta (ex: criar grupo) **não aparece** no banco/UI de produção (isolamento)
- [ ] Produção permanece intacta: `rachao.app` e `/api/v1` continuam funcionando normalmente
- [ ] `beta.rachao.app` carrega e navega ponta-a-ponta: login via OTP bypass → `POST /api/v2/groups` → `POST /api/v2/groups/{id}/matches` → `POST /api/v2/groups/{id}/matches/{id}/attendance`
- [ ] `go test ./internal/...` passa sem banco real
- [ ] `go test ./tests/integration/...` passa com banco real
- [ ] `golangci-lint` passa sem warnings
- [ ] Imagem Docker production ≤ 30MB
- [ ] GitHub Actions workflow verde (lint + unit + integration + build)
- [ ] Response JSON de `GET /api/v2/groups` é estruturalmente equivalente a `GET /api/v1/groups` para os mesmos dados (logo após a cópia)
- [ ] Qualquer conta existente na base-cópia navega na v2 sem bloqueio por usuário (sem gate)
- [ ] `docs.rachao.app` está acessível com playground Mintlify funcional para ao menos `POST /api/v2/auth/login`, `GET /api/v2/groups` e `GET /api/v2/health`

---

## 13. Decisões em Aberto

| # | Decisão | Opções | Status |
|---|---|---|---|
| D-01 | Incluir `POST /api/v2/chat` (Anthropic SSE) na v1.0? | ✅ Sim / ❌ Deixar para v2.0 | aguardando |
| D-02 | Supabase Storage (avatar) em v1.0? | ✅ Sim / ❌ Endpoint retorna 501 até Fase 4 | aguardando |
| D-03 | Stripe webhooks em v1.0? | ✅ Sim (paridade total) / ❌ Fora do scope inicial | aguardando |
| D-04 | Subir `api-go` em produção durante qual fase? | Fase 1 / Fase 2 / Fase 5/6 | ✅ resolvido — Fase 6, como ambiente de homologação isolado (banco-cópia + `beta.rachao.app`) |
| D-05 | Benchmark formal no PRD de resultado? | Criar PRD 045 de performance / Incluir no README | aguardando |
| D-06 | Sincronização de schema do banco-cópia com produção | Re-snapshot periódico / Aplicar manualmente novas migrations / Automatizar no deploy | aguardando |
| D-07 | Isolar o Supabase Storage (avatares) do `api-go`? | Reusar bucket de prod (cosmético) / Criar bucket próprio no projeto-cópia | aguardando |

---

## 14. Apêndice — Variáveis de Ambiente

Variáveis do `api-go` **no ambiente de homologação** (as marcadas com ★ diferem da Python API):

```env
APP_ENV=staging                                          # ★ ambiente não-produtivo → habilita OTP bypass
DATABASE_URL=${DATABASE_URL_HML}                          # ★ banco-CÓPIA (não o de produção)
SECRET_KEY=<jwt-signing-key>                              #   igual ao de produção
CORS_ORIGINS=https://beta.rachao.app                      # ★
FRONTEND_URL=https://beta.rachao.app                      # ★
OTP_BYPASS_CODE=<código-de-teste>                         # ★ registro/login sem SMS real

# Twilio OTP
TWILIO_ACCOUNT_SID=
TWILIO_AUTH_TOKEN=
TWILIO_VERIFY_SID=

# Supabase Storage (avatares) — reusa o bucket de produção por padrão (ver D-07)
SUPABASE_URL=
SUPABASE_SERVICE_ROLE_KEY=

# Anthropic (chat)
ANTHROPIC_API_KEY=
LLM_MODEL=claude-haiku-4-5

# Stripe
STRIPE_SECRET_KEY=
# STRIPE_WEBHOOK_SECRET= ★ intencionalmente ausente → rota de webhook não é registrada
STRIPE_PRICE_BASIC_MONTHLY=
STRIPE_PRICE_BASIC_YEARLY=
STRIPE_PRICE_PRO_MONTHLY=
STRIPE_PRICE_PRO_YEARLY=

# Web Push VAPID
VAPID_PRIVATE_KEY=
VAPID_PUBLIC_KEY=
VAPID_CLAIMS_EMAIL=admin@rachao.app
```

**Build args do `frontend-beta`** (baked em build-time):

```env
VITE_API_URL=https://api.rachao.app/api/v2
VITE_BILLING_ENABLED=false
PUBLIC_LEGAL_CONTROLLER_NAME=...      # iguais aos de produção
PUBLIC_LEGAL_CONTROLLER_DOC=...
PUBLIC_LEGAL_FORUM_CITY=...
PUBLIC_LEGAL_CONTACT_EMAIL=...
```

**Novos secrets GitHub Actions necessários:** **`DATABASE_URL_HML`** (connection string do banco-cópia, gerada pelo usuário no Supabase — ver §15) e **`OTP_BYPASS_CODE`**. Os demais secrets já existem no repositório.

---

## 15. Apêndice — Cópia do banco no Supabase (passo a passo)

Procedimento executado **pelo usuário** para criar o banco-cópia que alimenta `DATABASE_URL_HML`.

**Premissas:**
- Produção é Supabase (região sa-east-1 / São Paulo). O auth do app é JWT próprio + tabela `players` — **não** usa Supabase Auth. Todos os dados do app vivem no schema **`public`** (o Supabase gere `auth`, `storage`, `extensions`, etc.). Logo, copia-se **apenas `public`**.
- Ter cliente Postgres **16+** instalado (`pg_dump` / `pg_restore` / `psql`) — versão do client ≥ do servidor.

**Passo a passo:**

1. **Criar novo projeto Supabase** (mesma org, região **sa-east-1**), com senha de DB forte. Anotar o `PROJECT_REF` e a senha.

2. **Obter as duas connection strings** (Dashboard → Project Settings → Database → Connection string → URI). Preferir a **Session pooler** (porta 5432, IPv4 — o VPS pode não ter IPv6):
   ```
   PROD_URL="postgresql://postgres.<ref_prod>:<senha>@aws-0-sa-east-1.pooler.supabase.com:5432/postgres"
   HML_URL="postgresql://postgres.<ref_novo>:<senha>@aws-0-sa-east-1.pooler.supabase.com:5432/postgres"
   ```

3. **Dump do schema `public` da produção** (schema + dados), formato custom comprimido:
   ```bash
   pg_dump "$PROD_URL" \
     --schema=public \
     --no-owner --no-privileges --no-publications --no-subscriptions \
     -Fc -f rachao_prod.dump
   ```
   (`--no-owner --no-privileges` evita erros de role/ACL específicos do Supabase.)

4. **Restaurar no projeto novo** (o schema `public` já existe vazio no projeto novo):
   ```bash
   pg_restore --no-owner --no-privileges --no-acl \
     --disable-triggers -d "$HML_URL" rachao_prod.dump
   ```
   Em caso de re-execução, usar `--clean --if-exists` para limpar antes de restaurar.

5. **Validar a cópia** (comparar contagens com a produção):
   ```bash
   psql "$HML_URL" -c "select count(*) from players;" \
                   -c "select count(*) from groups;" \
                   -c "select count(*) from matches;"
   # conferir também que a tabela de controle de migrations reflete as 43 já aplicadas
   ```

6. **Registrar o secret**: usar a `HML_URL` (session pooler, porta 5432) como GitHub secret **`DATABASE_URL_HML`**. Evitar a porta 6543 (transaction pooler) com pgx por causa de prepared statements; se precisar usá-la, anexar `?default_query_exec_mode=simple_protocol`.

**Alternativa (Supabase CLI):**
```bash
supabase db dump --db-url "$PROD_URL" -f schema.sql            # estrutura
supabase db dump --db-url "$PROD_URL" --data-only -f data.sql  # dados
psql "$HML_URL" -f schema.sql && psql "$HML_URL" -f data.sql
```

**Sincronização futura (schema drift):** o banco-cópia é point-in-time. Para acompanhar produção, re-rodar o dump/restore periodicamente, **ou** aplicar manualmente no banco-cópia apenas as migrations novas surgidas em `football-api/migrations/` desde a última cópia (ver D-06).

**Storage (avatares) — nota:** as imagens ficam no Supabase Storage do projeto de produção. Se o `api-go` mantiver `SUPABASE_URL` apontando para produção, avatares aparecem mas uploads no beta gravariam no bucket de prod. Para isolamento total, criar bucket no projeto novo e apontar `SUPABASE_URL` / `SUPABASE_SERVICE_ROLE_KEY` do `api-go` para ele (avatares antigos ficariam quebrados até copiar os objetos). Cosmético — ver D-07.

---

## 16. Impacto na infra do VPS (consumo de recursos)

Ponto de vista sobre o que a subida do beta + v2 representa para o VPS.

### 16.1 O que roda hoje no VPS

`docker-compose.prod.yml` sobe **traefik**, **api** (Python/FastAPI), **frontend** (SvelteKit adapter-node), **api-go** (Go) e **mcp** — atualmente **sem limites de recursos** (`mem_limit`/`cpus` não definidos em nenhum serviço). O **PostgreSQL não roda no VPS** — é Supabase externo. Há ainda a stack de monitoramento (compose separado): Prometheus, Grafana, cAdvisor, node-exporter e Uptime Kuma.

### 16.2 O que efetivamente é novo

- **`api-go` (v2) já está em produção desde a Fase 5.** A Fase 6 **não adiciona um container de API novo** — apenas troca o `DATABASE_URL` do `api-go` para o banco-cópia (externo) e ajusta env (`APP_ENV`, `CORS_ORIGINS`, etc.). Impacto de runtime ≈ nulo.
- **`frontend-beta` é o único container novo no VPS:** um segundo servidor Node SSR (SvelteKit adapter-node), equivalente ao frontend de produção.

### 16.3 Impacto por recurso

| Recurso | Impacto previsto |
|---|---|
| **RAM** | Item principal. Um processo Node SSR (adapter-node) costuma usar **~80–150 MB** em idle. Como é homologação (tráfego baixo, poucos testers), tende ao piso da faixa. O `api-go` (Go) é leve (~15–40 MB) e **já contabilizado**. Estimativa de acréscimo: **~100–150 MB** (só o `frontend-beta`). |
| **CPU** | Negligível em idle; SSR só consome CPU sob request, e o tráfego de homologação é esporádico. Traefik com +1 router/serviço e +1 cert TLS (`beta.rachao.app`): overhead irrelevante. |
| **Disco** | +1 imagem no VPS (`frontend-beta`). Compartilha as camadas-base com a imagem do frontend de produção (mesmo Dockerfile/base alpine) → o incremento real é só a camada da aplicação. O `docker image prune -f` já roda no deploy. Storage da imagem no GHCR é fora do VPS. |
| **Banco de dados** | **Zero carga no VPS** — o banco-cópia vive no Supabase (externo). Sem novo Postgres local, sem disco/IO de banco. O custo recai sobre o plano Supabase do novo projeto. |
| **Rede/egress** | Tráfego de homologação baixo. O `api-go` ↔ Supabase-cópia adiciona conexões de saída, mas em volume pequeno. |
| **CI/CD** | +1 build de imagem (`frontend-beta`) por deploy do frontend → mais minutos de CI e storage no GHCR — **não** no VPS. |

### 16.4 Conclusão e recomendações

- Impacto previsto no VPS é **pequeno e localizado**: essencialmente **+1 processo Node (~100–150 MB RAM)**. Sem novo banco, sem nova API, CPU/disco marginais.
- Como **nenhum container tem `mem_limit`/`cpus` hoje**, vale definir um teto para o `frontend-beta` (ex.: `mem_limit: 256m`) para que um pico em homologação não pressione a RAM de produção — idealmente, aplicar limites a todos os serviços.
- **Gargalo mais provável é RAM**, não CPU/disco. Antes da subida, verificar o headroom de memória do host: a stack de monitoramento + Python API + 2 frontends Node + 2 APIs já somam um consumo relevante. Em VPS pequeno (ex.: 2 GB), considerar limites por container ou avaliar upgrade.
- Já existe observabilidade (cAdvisor + node-exporter + Grafana): **acompanhar RAM/CPU do host e do `frontend-beta` após o deploy** para validar a estimativa acima.
