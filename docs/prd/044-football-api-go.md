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
- **Modo aberto de staging** (`APP_ENV=staging`): gate `api_v2_enabled` desligado, OTP por bypass code (sem SMS real), billing off no frontend, sem processamento de webhook Stripe
- Painel admin `/admin/api-v2` para controle de acesso por usuário (opcional no modo aberto)
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

### 3.8 Controle de acesso por usuário — `api_v2_enabled`

**Decisão: gate mantido no código, porém neutralizado no ambiente de homologação aberto**

O gate `api_v2_enabled` foi desenhado para rollout por amostragem **no modelo de banco compartilhado** — o que deixa de se aplicar: o isolamento agora vem do subdomínio dedicado (`beta.rachao.app`) + banco-cópia. No modo aberto (`APP_ENV=staging`) o gate é **bypassed**, e qualquer conta existente na base-cópia navega livremente. O código do gate permanece para o caso de, no futuro, a v2 voltar a apontar para o banco de produção com rollout seletivo.

A Go API implementa um middleware `api_v2_access.go` que, após a autenticação JWT, verifica se o player autenticado possui `api_v2_enabled = true`. Se não, retorna `403 {"detail":"API_V2_NOT_ENABLED"}`. **O bypass passa a valer para qualquer `APP_ENV != "production"`** (ver RF-13) — em homologação roda como `staging`, então o gate não bloqueia.

| Componente | Detalhe |
|---|---|
| **DB column** | `api_v2_enabled BOOLEAN DEFAULT FALSE NOT NULL` na tabela `players` |
| **Migration** | `044_api_v2_enabled.sql` (já presente no schema, portanto no banco-cópia) |
| **Middleware** | `internal/middleware/api_v2_access.go` — aplicado em todo o router `/api/v2`, exceto sub-routers públicos; bypassed quando `APP_ENV != "production"` |
| **Isenções** | `GET /api/v2/health`, todos os endpoints sem auth (`OptionalPlayer` e públicos), e o super admin (`role = 'admin'`) |
| **Admin endpoints** | `GET /admin/api-v2-users` (lista jogadores + status) e `PATCH /admin/api-v2-users/{player_id}` (toggle) — disponíveis, porém opcionais no modo aberto |
| **Frontend** | Página `/admin/api-v2` no SvelteKit — espelho exato de `/admin/chat` |

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
| `APP_ENV=staging` (≠ `production`) | Bypassa o gate `api_v2_enabled` (RF-13) e habilita o OTP bypass |
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
│   │   ├── api_v2_access.go         # gate: 403 se player não tem api_v2_enabled=true
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
│   ├── authentication.mdx           # JWT, MCP tokens, api_v2_enabled
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

### Admin (12 endpoints)
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
| GET | `/admin/api-v2-users` | super-admin |
| PATCH | `/admin/api-v2-users/{player_id}` | super-admin |

**Total: ~99 endpoints** (+ `GET /api/v2/health`)

---

## 7. Requisitos Funcionais

### RF-01 — Paridade de endpoints (Must)
Todos os ~99 endpoints acima devem estar implementados com o mesmo contrato HTTP: método, path, request body, response schema e HTTP status codes idênticos à versão Python.

### RF-02 — Prefixo `/api/v2` (Must)
Todos os routes montados sob `/api/v2/`. A constante `API_PREFIX = "/api/v2"` definida em `config.go`.

### RF-03 — Banco de dados dedicado (Must)
Conectar ao **banco-cópia** (homologação) via `DATABASE_URL_HML`, sem DDL adicional. Schema idêntico ao de produção. Nenhuma operação da v2 pode tocar o banco de produção.

### RF-04 — Formato de JWT compatível (Must)
A Go API usa o mesmo algoritmo (HS256) e `SECRET_KEY` da Python API, de modo que o formato de token é compatível. **Cada ambiente autentica contra o seu próprio banco** (prod vs. cópia), portanto a interoperabilidade de tokens entre v1 e v2 não é mais um objetivo de design — é apenas um efeito colateral inofensivo da chave compartilhada. O middleware de auth deve suportar também MCP tokens com prefixo `rachao_`.

### RF-05 — Documentação Mintlify (Must)
Documentação pública em `docs.rachao.app` com:
- `quickstart.mdx`: pré-requisitos (Go 1.24, Docker, sqlc, golangci-lint), `.env.example`, `make run`, `make test`, `make generate`
- `authentication.mdx`: JWT cross-API, MCP tokens, `api_v2_enabled`
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

### RF-13 — Controle de acesso por usuário para API v2 (Must)
A Go API mantém o gate de acesso por usuário, com bypass por ambiente:
- Middleware `api_v2_access.go` aplicado em todo o router `/api/v2`, verificando `api_v2_enabled = true` no player autenticado
- Retorna `403 {"detail":"API_V2_NOT_ENABLED"}` para players autenticados sem o flag ativo **quando `APP_ENV == "production"`**
- **Bypass por ambiente:** alterar a condição de bypass de `cfg.AppEnv == "development"` para **`cfg.AppEnv != "production"`** (mudança de 1 linha na montagem do middleware), de modo que `APP_ENV=staging` (homologação) já libere o acesso sem depender literalmente de `"development"`
- **Isenções:** endpoints sem autenticação (públicos e `OptionalPlayer`), `GET /api/v2/health`, e super admin (`role = 'admin'` sempre tem acesso)
- Super admin pode habilitar/desabilitar por usuário via `PATCH /admin/api-v2-users/{player_id}`
- Página `/admin/api-v2` no SvelteKit para gerenciamento visual (mesmo padrão de `/admin/chat`)

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
- [x] Middleware `api_v2_access.go` — gate `403 API_V2_NOT_ENABLED` (antecipado da Fase 5)
- [x] Handler `auth.go` — todos os 12 endpoints
- [x] `internal/db/queries.go` — camada de queries hand-crafted (substitui sqlc output para Fase 1)
- [x] `sql/queries/players.sql` + `sql/queries/auth.sql` — anotadas para futura geração sqlc
- [x] Testes unitários de auth (11 testes: login, register, refresh, OTP, get-me)
- [x] Dockerfile (dev + production/scratch)
- [x] `docker-compose.yml` local (postgres + api-go)
- [x] `.air.toml`, `sqlc.yaml`, `.golangci.yml`, `.env.example`, `.gitignore`
- [x] Migration `044_api_v2_enabled.sql` — adicionar `api_v2_enabled BOOLEAN DEFAULT FALSE NOT NULL` à tabela `players` (Python API migrations; faz parte do schema, portanto presente também no banco-cópia)
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
- [x] Handler `admin.go` (12 endpoints — stats, matches, groups, subscriptions CRUD, players, avatar, beta-signups, api-v2-users)
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
- [x] Implementar middleware `api_v2_access.go` + testes unitários do gate (antecipado — implementado na Fase 1)
- [x] `GET /admin/api-v2-users` e `PATCH /admin/api-v2-users/{player_id}` em `handlers/admin.go` (implementado na Fase 4)
- [x] Criar página `/admin/api-v2` no SvelteKit frontend (espelho de `/admin/chat`) + card no painel admin
- [ ] Anotar handlers Go com `swaggo/swag` (`// @Summary`, `// @Param`, `// @Success`, `// @Router`)
- [x] Configurar `mintlify/mint.json` com branding rachao.app (cores, logo, nav)
- [x] Criar `mintlify/authentication.mdx` e `mintlify/architecture.mdx`
- [ ] `make docs` gera `openapi.yaml` atualizado
- [ ] Conectar repositório ao Mintlify Cloud → deploy em `docs.rachao.app`

### Fase 6 — Ambiente de homologação (banco-cópia + `beta.rachao.app`)
- [ ] Usuário cria o banco-cópia no Supabase (snapshot da produção) e fornece `DATABASE_URL_HML` (ver Apêndice §15)
- [ ] Usuário cria registro DNS `beta.rachao.app` → IP do VPS (Traefik emite cert letsencrypt)
- [ ] Adicionar secrets no GitHub Actions: `DATABASE_URL_HML` e `OTP_BYPASS_CODE`
- [ ] Go: ajustar bypass do gate para `cfg.AppEnv != "production"` (1 linha) — ver RF-13
- [ ] `docker-compose.prod.yml`: alterar serviço `api-go` para usar `DATABASE_URL_HML`, `APP_ENV=staging`, `OTP_BYPASS_CODE`, `CORS_ORIGINS=https://beta.rachao.app`, `FRONTEND_URL=https://beta.rachao.app`; remover `STRIPE_WEBHOOK_SECRET` do `api-go`
- [ ] `docker-compose.prod.yml`: adicionar serviço `frontend-beta`
- [ ] `traefik-dynamic.yml`: adicionar router/serviço `Host('beta.rachao.app')` → `frontend-beta:3000`
- [ ] `main.yml`: injetar `DATABASE_URL_HML` e `OTP_BYPASS_CODE` no `.env.prod`; adicionar build+push da imagem `football-manager-frontend-beta` com os build args da v2 (`VITE_API_URL=https://api.rachao.app/api/v2`, `VITE_BILLING_ENABLED=false`)
- [ ] Validar navegação ponta-a-ponta em `beta.rachao.app` sem afetar produção

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
    APP_ENV: staging                          # bypassa gate + habilita OTP bypass
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
- [ ] Em `APP_ENV=production`, request autenticado de player com `api_v2_enabled = false` retorna `403 {"detail":"API_V2_NOT_ENABLED"}`; em `APP_ENV=staging` (homologação) o gate é bypassed e o request passa
- [ ] Endpoints públicos (ex: `GET /api/v2/ranking`, `POST /api/v2/auth/login`) NÃO são bloqueados pelo gate
- [ ] Super admin (`role = 'admin'`) passa pelo gate sem bloqueio independente do flag
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
APP_ENV=staging                                          # ★ bypassa gate + habilita OTP bypass
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

6. **(Opcional — modo aberto)** Como o gate é bypassed via `APP_ENV=staging`, não é necessário; se quiser deixar explícito: `UPDATE players SET api_v2_enabled = true;` no banco-cópia.

7. **Registrar o secret**: usar a `HML_URL` (session pooler, porta 5432) como GitHub secret **`DATABASE_URL_HML`**. Evitar a porta 6543 (transaction pooler) com pgx por causa de prepared statements; se precisar usá-la, anexar `?default_query_exec_mode=simple_protocol`.

**Alternativa (Supabase CLI):**
```bash
supabase db dump --db-url "$PROD_URL" -f schema.sql            # estrutura
supabase db dump --db-url "$PROD_URL" --data-only -f data.sql  # dados
psql "$HML_URL" -f schema.sql && psql "$HML_URL" -f data.sql
```

**Sincronização futura (schema drift):** o banco-cópia é point-in-time. Para acompanhar produção, re-rodar o dump/restore periodicamente, **ou** aplicar manualmente no banco-cópia apenas as migrations novas surgidas em `football-api/migrations/` desde a última cópia (ver D-06).

**Storage (avatares) — nota:** as imagens ficam no Supabase Storage do projeto de produção. Se o `api-go` mantiver `SUPABASE_URL` apontando para produção, avatares aparecem mas uploads no beta gravariam no bucket de prod. Para isolamento total, criar bucket no projeto novo e apontar `SUPABASE_URL` / `SUPABASE_SERVICE_ROLE_KEY` do `api-go` para ele (avatares antigos ficariam quebrados até copiar os objetos). Cosmético — ver D-07.
