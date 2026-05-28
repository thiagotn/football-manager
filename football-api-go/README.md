# football-api-go

<p align="center">
  <a href="https://github.com/thiagotn/football-manager/actions/workflows/api-go.yml"><img src="https://github.com/thiagotn/football-manager/actions/workflows/api-go.yml/badge.svg" alt="Build & Test API Go" /></a>
  <a href="https://codecov.io/gh/thiagotn/football-manager"><img src="https://codecov.io/gh/thiagotn/football-manager/graph/badge.svg?flag=api-go" alt="codecov" /></a>
</p>

Port da API rachao.app em Go — mesmo banco de dados, mesma tabela de JWT, paridade total de endpoints em relação à `football-api/` (Python/FastAPI). Roteado pelo Traefik como `/api/v2`.

## Stack

| Camada | Tecnologia |
|--------|-----------|
| **Router** | [Chi v5](https://github.com/go-chi/chi) |
| **Banco** | PostgreSQL 16 · [pgx/v5](https://github.com/jackc/pgx) + `pgxpool` |
| **Queries** | [sqlc](https://sqlc.dev) — geração de código type-safe a partir de SQL |
| **Auth** | JWT HS256 — mesma `SECRET_KEY` da API Python |
| **Streaming** | Server-Sent Events (SSE) para `/api/v2/chat` |
| **Storage** | Supabase Storage (HTTP direto — sem SDK) |
| **OTP** | Twilio Verify (opcional; sem config aceita qualquer código) |
| **Docs** | [Mintlify](https://mintlify.com) + `openapi.yaml` gerado via `swaggo/swag` |
| **CI/CD** | GitHub Actions → GHCR → VPS |

---

## Desenvolvimento local

> Comandos executados a partir deste diretório (`football-api-go/`).

### 1. Configurar o ambiente

```bash
cp .env.example .env
```

O `.env` já vem configurado para rodar localmente. Twilio é opcional — sem configuração, qualquer código OTP é aceito.

### 2. Subir o Postgres

```bash
docker compose up postgres -d
```

Porta exposta: `5433` (para não colidir com o Postgres da `football-api/` na `5432`).

### 3. Aplicar as migrations

As migrations ficam em `../football-api/migrations/` (banco compartilhado). Requer `golang-migrate`:

```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
export DATABASE_URL="postgres://football:football@localhost:5433/football_dev?sslmode=disable"
make migrate
```

### 4. Rodar o servidor

```bash
make run       # live-reload via air (recomendado)
# ou:
make build && ./tmp/server
```

A API estará disponível em `http://localhost:8001/api/v2` (porta `8001` no Docker Compose local).

### Login inicial (admin)

```
WhatsApp: +5511999990000
Senha:    admin123
```

---

## Comandos úteis

```bash
make run              # Live-reload via air
make build            # Compila o binário em ./tmp/server
make test             # Testes unitários (sem banco)
make test-race        # Testes unitários com race detector
make test-integration # Testes de integração (requer DATABASE_URL)
make test-all         # unit + integration
make lint             # golangci-lint
make generate         # sqlc generate + go generate
make migrate          # Aplica migrations (requer DATABASE_URL)
make docs             # Gera openapi.yaml em mintlify/ via swaggo/swag
make coverage         # Roda testes e abre relatório de cobertura no browser
make deps             # go mod download + tidy
make clean            # Remove ./tmp e coverage.out
```

---

## Estrutura de diretórios

```
football-api-go/
├── cmd/server/
│   └── main.go                 # Entrypoint: config, pool, router, cron jobs
├── internal/
│   ├── config/
│   │   └── config.go           # Leitura de variáveis de ambiente
│   ├── db/
│   │   ├── *.go                # Queries geradas pelo sqlc + helpers manuais
│   │   └── recurrence.go       # Queries específicas para jobs de recorrência
│   ├── handlers/
│   │   ├── auth.go             # Login, registro, OTP, troca de senha
│   │   ├── groups.go           # Grupos, membros, skill, lista de espera
│   │   ├── matches.go          # Partidas, presenças, discover, stats
│   │   ├── players.go          # Jogadores, avatar (Supabase Storage)
│   │   ├── teams.go            # Sorteio de times (snake-draft)
│   │   ├── votes.go            # Votação pós-partida
│   │   ├── finance.go          # Controle financeiro por grupo
│   │   ├── invites.go          # Convites por link
│   │   ├── subscriptions.go    # Planos e assinaturas
│   │   ├── webhooks.go         # Webhooks do Stripe
│   │   ├── push.go             # Web Push (VAPID)
│   │   ├── ranking.go          # Ranking geral da plataforma
│   │   ├── reviews.go          # Avaliações do app
│   │   ├── mcp_tokens.go       # Tokens MCP pessoais
│   │   ├── beta.go             # Beta Android signup
│   │   ├── chat.go             # Assistente IA — SSE + Anthropic direct HTTP
│   │   └── admin.go            # Painel admin (stats, subscriptions, players)
│   ├── middleware/
│   │   ├── auth.go             # Validação JWT + lookup no banco
│   │   ├── admin.go            # RequireAdmin — restringe a super admins
│   │   ├── cors.go             # CORS configurável
│   │   ├── rate_limit.go       # Rate limiting de login
│   │   └── recovery.go         # Panic recovery com resposta JSON
│   ├── server/
│   │   └── router.go           # Montagem do chi.Router com todos os handlers
│   └── services/
│       ├── auth.go             # JWT issue/verify, bcrypt, OTP via Twilio
│       ├── billing_stripe.go   # Checkout, cancelamento de assinatura
│       ├── storage.go          # Upload/remoção de avatares no Supabase
│       ├── twilio.go           # SendOTP / CheckOTP (E.164)
│       └── recurrence.go       # Criação de partidas recorrentes + status sync
├── tests/
│   ├── unit/                   # Testes sem banco (nil pool seguro)
│   └── integration/            # Testes com banco real (auth, groups, matches)
├── mintlify/                   # Documentação Mintlify
│   ├── mint.json               # Config: nav, cores, logo, baseUrl=docs.rachao.app
│   ├── openapi.yaml            # Gerado por `make docs` (swaggo/swag)
│   ├── quickstart.mdx
│   ├── authentication.mdx
│   └── architecture.mdx
├── sql/
│   └── queries/                # Arquivos .sql lidos pelo sqlc
├── .air.toml                   # Config do live-reload
├── .golangci.yml               # Config do linter
├── docker-compose.yml          # Desenvolvimento local (Postgres na 5433, API na 8001)
├── Dockerfile                  # Multi-stage: dev / builder / production (scratch, ~10 MB)
├── Makefile
└── sqlc.yaml
```

---

## API — Endpoints principais

A API expõe ~99 endpoints sob `/api/v2`, estruturalmente equivalentes à `football-api/` em `/api/v1`.

### Públicos (sem autenticação)

| Método | Rota | Descrição |
|--------|------|-----------|
| `GET`  | `/api/v2/health` | Health check |
| `POST` | `/api/v2/auth/login` | Login (retorna JWT) |
| `POST` | `/api/v2/auth/register` | Cadastro |
| `GET`  | `/api/v2/matches/public/{hash}` | Dados da partida pelo hash |
| `GET`  | `/api/v2/matches/{id}/teams` | Times sorteados (público) |
| `GET`  | `/api/v2/ranking` | Ranking da plataforma |
| `GET`  | `/api/v2/matches/discover` | Partidas abertas |

### Autenticados (JWT)

| Método | Rota | Descrição |
|--------|------|-----------|
| `GET`  | `/api/v2/auth/me` | Dados do jogador autenticado |
| `GET`  | `/api/v2/groups` | Meus grupos |
| `POST` | `/api/v2/groups` | Cria grupo |
| `POST` | `/api/v2/groups/{id}/matches` | Cria partida |
| `POST` | `/api/v2/groups/{groupID}/matches/{id}/attendance` | Confirma/recusa presença |
| `POST` | `/api/v2/matches/{id}/votes` | Submete votação pós-partida |
| `POST` | `/api/v2/chat` | Assistente IA (SSE streaming) |
| `GET`  | `/api/v2/mcp-tokens` | Tokens MCP pessoais |

### Admin

| Método | Rota | Descrição |
|--------|------|-----------|
| `GET`  | `/api/v2/admin/stats` | Big numbers da plataforma |
| `GET`  | `/api/v2/admin/subscriptions` | Lista de assinantes |
| `PATCH`| `/api/v2/admin/subscriptions/{id}` | Atualiza plano/status |
| `GET`  | `/api/v2/admin/chat-users` | Controle de acesso ao Assistente IA |

---

## Variáveis de ambiente

| Variável | Obrigatória | Descrição |
|----------|-------------|-----------|
| `DATABASE_URL` | Sim | `postgres://user:pass@host/db?sslmode=...` |
| `SECRET_KEY` | Sim | Chave JWT HS256 — deve ser idêntica à da `football-api/` |
| `APP_ENV` | Não | `development` (padrão) / `production` |
| `PORT` | Não | Porta HTTP (padrão: `8080`) |
| `CORS_ORIGINS` | Não | Origens permitidas (separadas por vírgula) |
| `TWILIO_ACCOUNT_SID` | Não | SID da conta Twilio |
| `TWILIO_AUTH_TOKEN` | Não | Token Twilio |
| `TWILIO_VERIFY_SERVICE_SID` | Não | SID do serviço Twilio Verify |
| `SUPABASE_URL` | Não | URL do projeto Supabase (para Storage de avatares) |
| `SUPABASE_SERVICE_KEY` | Não | Service role key do Supabase |
| `ANTHROPIC_API_KEY` | Não | Chave da API Anthropic (para `/api/v2/chat`) |
| `LLM_MODEL` | Não | Modelo Anthropic (padrão: `claude-haiku-4-5`) |
| `STRIPE_SECRET_KEY` | Não | Chave secreta da API Stripe |
| `STRIPE_WEBHOOK_SECRET` | Não | Segredo para validar webhooks do Stripe |
| `VAPID_PUBLIC_KEY` | Não | Chave pública VAPID para Web Push |
| `VAPID_PRIVATE_KEY` | Não | Chave privada VAPID para Web Push |
| `OTP_BYPASS_CODE` | Não | Código OTP fixo para dev local (ex: `123456`) |

> Sem `TWILIO_*` configurados, qualquer código OTP é aceito (modo bypass automático).

---

## Banco de dados

`football-api-go` usa o mesmo banco PostgreSQL da `football-api/`. As migrations ficam em `../football-api/migrations/` e são aplicadas uma única vez (não há migrator próprio no binário Go — use `make migrate` ou o Python API no startup).

Queries são geradas pelo sqlc a partir de `sql/queries/`. Para regenerar após alterar os `.sql`:

```bash
make generate
```

---

## Testes

### Unitários (sem banco)

```bash
make test
# ou com race detector:
make test-race
```

Handlers são testados com `nil` pool — válido apenas para caminhos que retornam antes de qualquer chamada ao banco (validação de UUID, autorização, parsing de corpo).

### Integração (banco real)

```bash
export DATABASE_URL="postgres://football:football@localhost:5433/football_dev?sslmode=disable"
make test-integration
```

Cobrem fluxos end-to-end: registro → login → criação de grupo → criação de partida.

### CI

O workflow `.github/workflows/api-go.yml` executa automaticamente em todo push para `football-api-go/**`:

```
push → lint → unit-tests → integration-tests → build & push GHCR
```

---

## Docker

### Desenvolvimento (com live-reload)

```bash
docker compose up --build
```

API disponível em `http://localhost:8001/api/v2`. Edições em qualquer arquivo Go disparam rebuild automático via [air](https://github.com/air-verse/air).

### Imagem de produção

Multi-stage build com imagem final `FROM scratch`:

```bash
docker build --target production -t football-api-go .
```

Imagem resultante: ~10 MB (binário estático + certificados CA + timezone data).

---

## Produção

Em produção o serviço sobe como `api-go` no `docker-compose.prod.yml` e é roteado pelo Traefik:

```
api.rachao.app/api/v2  →  api-go:8080   (este serviço)
api.rachao.app/api/v1  →  api:8000      (football-api, Python)
```

O deploy é feito pelo workflow principal `.github/workflows/main.yml` junto com os demais serviços do monorepo.
