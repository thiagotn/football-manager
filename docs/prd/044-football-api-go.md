# PRD 044 — `football-api-go`: Port da API rachao.app para Go

| Campo | Valor |
|---|---|
| **Versão** | 1.0 |
| **Status** | 🚧 Em implementação (Fase 3 completa) |
| **Autor** | thiagotn |
| **Data** | 2026-05-20 |

---

## 1. Visão Geral

### 1.1 Contexto

A API atual (`football-api/`) é construída em Python/FastAPI com SQLAlchemy async. Ela possui ~95 endpoints, 16 domínios e 43 migrations aplicadas. O objetivo deste PRD é criar uma versão equivalente em Go — o `football-api-go/` — como exploração da stack, benchmarking de desempenho e validação de paridade funcional, sem substituir a API Python existente.

### 1.2 Objetivo

Construir `football-api-go/` com paridade total de endpoints em relação à `football-api/`, de modo que qualquer cliente (frontend, MCP, app mobile) possa chavear entre as duas versões alterando apenas o prefixo da URL base (`/api/v1` → `/api/v2`), sem mudança de contrato de dados.

### 1.3 Proposta de valor

- Avaliar ganhos de performance (latência, throughput, uso de memória) versus a stack Python atual
- Manter o mesmo banco de dados PostgreSQL e schema — zero migração de dados
- Servir como base para decisão futura de migração parcial ou total de stack
- Documentar o processo de porting como referência para decisões de arquitetura futuras

---

## 2. Escopo

### 2.1 Incluído em v1.0

- Todos os ~97 endpoints dos 16 domínios atuais (paridade total de contrato HTTP)
- Mesmo banco PostgreSQL, schema e migrations existentes (migration `044_api_v2_enabled.sql` adicionada)
- Painel admin para controle de acesso à API v2 por usuário (rollout por amostragem)
- Documentação interativa via Mintlify em `docs.rachao.app` com OpenAPI playground e `/llms.txt`
- Testes unitários de handlers + testes de integração com banco real
- Dockerfile multi-stage (dev + production)
- GitHub Actions dedicado (`football-api-go/**`)
- Traefik routing: `/api/v2/` → Go API (sem alterar `/api/v1/` Python)

### 2.2 Fora de escopo (v1.0)

- Substituir ou deprecar a `football-api/` Python em produção
- Migrar dados ou alterar schema de banco
- Novo frontend ou alterações no MCP server
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

**Decisão: `/api/v2/` no mesmo `api.rachao.app`**

```
https://api.rachao.app/api/v1/...  →  Python API  (container: api)
https://api.rachao.app/api/v2/...  →  Go API      (container: api-go)
```

Uma nova regra Traefik `PathPrefix('/api/v2')` aponta para o container Go. Os dois serviços coexistem no mesmo docker-compose de produção e leem o mesmo banco. Para chavear o frontend entre v1 e v2, basta alterar a variável `VITE_API_URL` no build.

### 3.4 Banco de dados — compartilhado

**Decisão: banco compartilhado (mesmo Supabase PostgreSQL)**

- Zero migração de dados; testa paridade real com os mesmos dados
- Em ambiente local, o mesmo postgres do docker-compose é usado

### 3.5 Migrations — reuso das SQL existentes via golang-migrate

As 43 migrations em `football-api/migrations/*.sql` já estão aplicadas em produção. A Go API conecta ao banco sem rodar migrations em produção. Em ambiente local, `make migrate` aplica as migrations usando [golang-migrate](https://github.com/golang-migrate/migrate) apontando para `../football-api/migrations/`.

### 3.6 Configuração — envconfig

**Decisão: [envconfig](https://github.com/kelseyhightower/envconfig)** + arquivo `.env`

Mesmas variáveis de ambiente da API Python (DATABASE_URL, SECRET_KEY, etc.) — sem novos secrets necessários em produção.

### 3.7 Testes — stdlib + testify + httptest

**Decisão: `testing` stdlib + [testify](https://github.com/stretchr/testify) + `net/http/httptest`**

- Handlers testados com `httptest.NewRecorder` + interfaces de service mockadas
- Integração com banco real via service container no CI (`postgres:16-alpine`)
- Cobertura reportada para Codecov com `flags: api-go`

### 3.8 Controle de acesso por usuário — `api_v2_enabled`

**Decisão: flag por usuário controlada pelo super admin, com middleware gate na Go API**

A Go API implementa um middleware `api_v2_access.go` que, após a autenticação JWT, verifica se o player autenticado possui `api_v2_enabled = true` no banco. Se não, retorna `403 {"detail":"API_V2_NOT_ENABLED"}` antes de chegar ao handler.

| Componente | Detalhe |
|---|---|
| **DB column** | `api_v2_enabled BOOLEAN DEFAULT FALSE NOT NULL` na tabela `players` |
| **Migration** | `044_api_v2_enabled.sql` (Python API migrations, aplicada no shared DB) |
| **Middleware** | `internal/middleware/api_v2_access.go` — aplicado em todo o router `/api/v2`, exceto sub-routers públicos |
| **Isenções** | `GET /api/v2/health`, todos os endpoints sem auth (`OptionalPlayer` e públicos), e o super admin (`role = 'admin'`) |
| **Admin endpoints** | `GET /admin/api-v2-users` (lista jogadores + status) e `PATCH /admin/api-v2-users/{player_id}` (toggle) |
| **Frontend** | Página `/admin/api-v2` no SvelteKit — espelho exato de `/admin/chat` |

Esse mecanismo permite rollout por amostragem: o super admin habilita acesso a usuários específicos (ex: equipe interna, beta testers) antes de abrir para todos, sem necessidade de feature flags externas ou variáveis de ambiente por usuário.

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

### RF-03 — Mesmo banco de dados (Must)
Conectar ao mesmo PostgreSQL sem DDL adicional. Schema idêntico ao atual.

### RF-04 — JWT cross-compatível (Must)
Tokens gerados pela Python API (HS256, `SECRET_KEY`) devem ser aceitos pela Go API e vice-versa. O middleware de auth deve suportar também MCP tokens com prefixo `rachao_`.

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
Regra em `traefik-dynamic.yml` e serviço em `docker-compose.prod.yml` para rotear `/api/v2/` → container `api-go:8080`. O `/api/v1/` permanece inalterado.

### RF-11 — Chat/SSE com Anthropic (Should)
`POST /api/v2/chat` com Server-Sent Events usando o mesmo MCP server existente (`mcp.rachao.app`). Rate limit de 20 mensagens/hora por usuário via coluna `chat_req_count` + `chat_req_window`.

### RF-12 — Makefile (Should)
Comandos: `make run`, `make test`, `make test-integration`, `make lint`, `make generate`, `make migrate`, `make build`, `make docs`.

### RF-13 — Controle de acesso por usuário para API v2 (Must)
A Go API deve implementar um gate de acesso por usuário:
- Middleware `api_v2_access.go` aplicado em todo o router `/api/v2`, verificando `api_v2_enabled = true` no player autenticado
- Retorna `403 {"detail":"API_V2_NOT_ENABLED"}` para players autenticados sem o flag ativo
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
- [x] Migration `044_api_v2_enabled.sql` — adicionar `api_v2_enabled BOOLEAN DEFAULT FALSE NOT NULL` à tabela `players` (Python API migrations, shared DB)
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
- [ ] Handler `admin.go` (10 endpoints)
- [ ] Handler `chat.go` (SSE + Anthropic SDK Go)
- [ ] `services/twilio.go` (OTP via Twilio Verify)
- [ ] `services/storage.go` (Supabase avatar upload)
- [ ] `services/recurrence.go` (geração de partidas recorrentes)
- [ ] Testes dos handlers restantes

### Fase 5 — CI/CD e Produção
- [ ] `.github/workflows/api-go.yml` (lint + unit + integration + build + push GHCR)
- [ ] Adicionar serviço `api-go` em `football-api/docker-compose.prod.yml`
- [ ] Adicionar router `/api/v2` em `football-api/traefik-dynamic.yml`
- [ ] Push image para GHCR: `ghcr.io/thiagotn/football-manager-api-go`
- [ ] Atualizar deploy job em `main.yml` para incluir `api-go`
- [x] Implementar middleware `api_v2_access.go` + testes unitários do gate (antecipado — implementado na Fase 1)
- [ ] Implementar `GET /admin/api-v2-users` e `PATCH /admin/api-v2-users/{player_id}` em `handlers/admin.go`
- [ ] Criar página `/admin/api-v2` no SvelteKit frontend (espelho de `/admin/chat`)
- [ ] Anotar handlers Go com `swaggo/swag` (`// @Summary`, `// @Param`, `// @Success`, `// @Router`)
- [ ] Configurar `mintlify/mint.json` com branding rachao.app (cores, logo, nav)
- [ ] Criar `mintlify/authentication.mdx` e `mintlify/architecture.mdx`
- [ ] `make docs` gera `openapi.yaml` atualizado
- [ ] Conectar repositório ao Mintlify Cloud → deploy em `docs.rachao.app`

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

**`football-api/traefik-dynamic.yml`** — adicionar:

```yaml
http:
  routers:
    api-go:
      rule: "Host(`api.rachao.app`) && PathPrefix(`/api/v2`)"
      service: api-go
      tls:
        certResolver: letsencrypt

  services:
    api-go:
      loadBalancer:
        servers:
          - url: "http://api-go:8080"
```

**`football-api/docker-compose.prod.yml`** — adicionar serviço:

```yaml
api-go:
  image: ghcr.io/${GITHUB_REPOSITORY_OWNER}/football-manager-api-go:latest
  environment:
    DATABASE_URL: ${DATABASE_URL}
    SECRET_KEY: ${SECRET_KEY}
    TWILIO_ACCOUNT_SID: ${TWILIO_ACCOUNT_SID}
    TWILIO_AUTH_TOKEN: ${TWILIO_AUTH_TOKEN}
    TWILIO_VERIFY_SID: ${TWILIO_VERIFY_SID}
    SUPABASE_URL: ${SUPABASE_URL}
    SUPABASE_SERVICE_ROLE_KEY: ${SUPABASE_SERVICE_ROLE_KEY}
    ANTHROPIC_API_KEY: ${ANTHROPIC_API_KEY}
    LLM_MODEL: ${LLM_MODEL}
    STRIPE_SECRET_KEY: ${STRIPE_SECRET_KEY}
    STRIPE_WEBHOOK_SECRET: ${STRIPE_WEBHOOK_SECRET}
    VAPID_PRIVATE_KEY: ${VAPID_PRIVATE_KEY}
    VAPID_PUBLIC_KEY: ${VAPID_PUBLIC_KEY}
    VAPID_CLAIMS_EMAIL: ${VAPID_CLAIMS_EMAIL}
  networks: [app-net]
  restart: unless-stopped
  labels:
    - "traefik.enable=true"
    - "traefik.http.routers.api-go.rule=Host(`api.rachao.app`) && PathPrefix(`/api/v2`)"
    - "traefik.http.services.api-go.loadbalancer.server.port=8080"
```

---

## 12. Critérios de Aceite

- [ ] `GET /api/v2/health` retorna `{"status":"ok"}` com HTTP 200
- [ ] Token JWT gerado em `POST /api/v1/auth/login` é aceito em `GET /api/v2/auth/me`
- [ ] Token JWT gerado em `POST /api/v2/auth/login` é aceito em `GET /api/v1/auth/me`
- [ ] Fluxo end-to-end funciona: `POST /api/v2/auth/register` → `POST /api/v2/groups` → `POST /api/v2/groups/{id}/matches` → `POST /api/v2/groups/{id}/matches/{id}/attendance`
- [ ] `go test ./internal/...` passa sem banco real
- [ ] `go test ./tests/integration/...` passa com banco real
- [ ] `golangci-lint` passa sem warnings
- [ ] Imagem Docker production ≤ 30MB
- [ ] GitHub Actions workflow verde (lint + unit + integration + build)
- [ ] Response JSON de `GET /api/v2/groups` é estruturalmente equivalente a `GET /api/v1/groups` para os mesmos dados
- [ ] Request autenticado de player com `api_v2_enabled = false` retorna `403 {"detail":"API_V2_NOT_ENABLED"}` em qualquer endpoint autenticado
- [ ] Endpoints públicos (ex: `GET /api/v2/ranking`, `POST /api/v2/auth/login`) NÃO são bloqueados pelo gate
- [ ] Super admin (`role = 'admin'`) passa pelo gate sem bloqueio independente do flag
- [ ] `PATCH /admin/api-v2-users/{id}` com `{"api_v2_enabled": true}` habilita o acesso e a próxima requisição do player passa a ser aceita
- [ ] `docs.rachao.app` está acessível com playground Mintlify funcional para ao menos `POST /api/v2/auth/login`, `GET /api/v2/groups` e `GET /api/v2/health`

---

## 13. Decisões em Aberto

| # | Decisão | Opções | Status |
|---|---|---|---|
| D-01 | Incluir `POST /api/v2/chat` (Anthropic SSE) na v1.0? | ✅ Sim / ❌ Deixar para v2.0 | aguardando |
| D-02 | Supabase Storage (avatar) em v1.0? | ✅ Sim / ❌ Endpoint retorna 501 até Fase 4 | aguardando |
| D-03 | Stripe webhooks em v1.0? | ✅ Sim (paridade total) / ❌ Fora do scope inicial | aguardando |
| D-04 | Subir `api-go` em produção durante qual fase? | Fase 1 (só `/health`) / Fase 2 (core) / Fase 5 (completo) | aguardando |
| D-05 | Benchmark formal no PRD de resultado? | Criar PRD 045 de performance / Incluir no README | aguardando |

---

## 14. Apêndice — Variáveis de Ambiente

Variáveis necessárias (idênticas à `football-api/`):

```env
DATABASE_URL=postgres://user:pass@host/dbname?sslmode=require
SECRET_KEY=<jwt-signing-key>
APP_ENV=production

# Twilio OTP
TWILIO_ACCOUNT_SID=
TWILIO_AUTH_TOKEN=
TWILIO_VERIFY_SID=

# Supabase Storage (avatares)
SUPABASE_URL=
SUPABASE_SERVICE_ROLE_KEY=

# Anthropic (chat)
ANTHROPIC_API_KEY=
LLM_MODEL=claude-haiku-4-5

# Stripe
STRIPE_SECRET_KEY=
STRIPE_WEBHOOK_SECRET=
STRIPE_PRICE_BASIC_MONTHLY=
STRIPE_PRICE_BASIC_YEARLY=
STRIPE_PRICE_PRO_MONTHLY=
STRIPE_PRICE_PRO_YEARLY=

# Web Push VAPID
VAPID_PRIVATE_KEY=
VAPID_PUBLIC_KEY=
VAPID_CLAIMS_EMAIL=admin@rachao.app
```

**Novos secrets GitHub Actions necessários:** nenhum. Todos os secrets já existem no repositório.
