# Guia: Subir e Testar football-api-go Localmente

> Documento de referência rápida para validar o progresso de implementação da API Go (v2)

## TL;DR — Comece Agora

```bash
cd football-api-go/
cp .env.example .env
docker compose up --build
# API em http://localhost:8001/api/v2
```

---

## 📋 Conteúdo

- [Contexto](#contexto)
- [Pré-requisitos](#pré-requisitos)
- [Opção A: Docker Compose (recomendado)](#opção-a--docker-compose-recomendado)
- [Opção B: Go Nativo + Air](#opção-b--go-nativo--air)
- [Rodando Testes](#rodando-testes)
- [Credenciais Padrão](#credenciais-padrão)
- [Variáveis Opcionais](#variáveis-opcionais)
- [Verificação de Saúde](#verificação-de-saúde)

---

## Contexto

**football-api-go/** é a API Go (v2) do projeto rachao.app, servindo `/api/v2`.

| Atributo | Valor |
|----------|-------|
| Framework | Chi v5 |
| Database | pgx/v5 (pool de conexões) |
| SQL | sqlc (queries tipadas) |
| Auth | JWT HS256 |
| Go Version | 1.24+ |
| Hot Reload | Air |
| Port (Docker) | 8001 |
| Port (Native) | 8080 |
| Database Port | 5433 (isolado) |

Os testes cobrem:
- **Unitários:** sem banco, rápidos (auth, helpers, middleware, lógica pura)
- **Integração/E2E:** banco real, fluxos completos (17+ domínios)

---

## Pré-requisitos

```bash
# Go 1.24+ — verifique a versão
go version

# Air para hot reload (instalação global)
go install github.com/air-verse/air@latest

# golang-migrate (opcional, para rodar migrations manualmente)
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

---

## Opção A — Docker Compose (recomendado)

O `football-api-go/docker-compose.yml` é **completamente independente** do projeto pai. Sobe um Postgres na porta `5433` que não conflita com a API Python na `5432`.

### Passos

```bash
cd football-api-go/

# 1. Copiar env (DATABASE_URL e SECRET_KEY já preenchidos)
cp .env.example .env

# 2. Subir Postgres + API Go com rebuild automático e hot reload
docker compose up --build

# 3. Aguarde a mensagem "API started on :8080"
```

### Resultado

- **API em:** http://localhost:8001/api/v2
- **Postgres em:** localhost:5433
- **Migrations:** aplicadas automaticamente no startup
- **Hot reload:** alterações no `.go` disparam rebuild (se usar `make run`)

### Logs

```bash
# Ver logs apenas da API
docker compose logs api-go -f

# Parar tudo
docker compose down
```

---

## Opção B — Go Nativo + Air

Para desenvolvimento sem Docker (Go instalado localmente).

### Passos

```bash
cd football-api-go/

# 1. Subir apenas o Postgres (porta 5433)
docker compose up postgres -d

# 2. Aguardar Postgres estar saudável
#    (dorme 5 segundos, postgres geralmente sobe rápido)

# 3. Configurar variáveis obrigatórias
export DATABASE_URL="postgres://football:football@localhost:5433/football_dev?sslmode=disable"
export SECRET_KEY="dev-secret-key-change-in-production"

# 4. Aplicar migrations (compartilhadas com football-api/migrations/)
make migrate

# 5. Rodar com live-reload
make run

# 6. Aguarde "API started on :8080"
```

### Resultado

- **API em:** http://localhost:8080/api/v2
- **Postgres em:** localhost:5433
- **Hot reload:** alterações no `.go` disparam rebuild (5s típico via Air)

### Parar

```bash
# Ctrl+C para parar a API
# Depois:
docker compose down postgres
```

---

## Rodando Testes

### ⚡ Quick Start — Testes Integrados

Se você nunca rodou os testes antes, siga **exatamente nesta ordem**:

```bash
# 1. Instalar golang-migrate (primeira vez, leva ~30s)
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# 2. Ir para o diretório da API Go
cd football-api-go/

# 3. Subir Postgres
docker compose up postgres -d
# Aguarde ~10 segundos para inicializar

# 4. Aplicar migrations (uma única vez, leva ~4 minutos)
cd ..  # voltar para raiz
/home/thiagotn/go/bin/migrate -path ./football-api/migrations \
  -database "postgres://football:football@localhost:5433/football_dev?sslmode=disable" up

# 5. Voltar para football-api-go e rodar testes
cd football-api-go/
export DATABASE_URL="postgres://football:football@localhost:5433/football_dev?sslmode=disable"
export SECRET_KEY="dev-secret-key-change-in-production"

# Testes unitários (rápido, sem banco)
make test

# Testes integrados (requer Postgres + migrations)
make test-integration

# Ambos
make test-all
```

**Próximas vezes** (Postgres já rodando, migrations já aplicadas):
```bash
cd football-api-go/
export DATABASE_URL="postgres://football:football@localhost:5433/football_dev?sslmode=disable"
export SECRET_KEY="dev-secret-key-change-in-production"
make test-integration  # ou make test-all
```

---

### Pré-requisito: Aplicar Migrations

**Antes de rodar qualquer teste de integração**, você precisa aplicar as migrations ao banco de dados.

#### Opção 1: Usar golang-migrate globalmente (recomendado)

```bash
# 1. Instalar golang-migrate globalmente (uma única vez)
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# 2. Garantir que Postgres está rodando (se ainda não está)
cd football-api-go/
docker compose up postgres -d

# 3. Aplicar migrations (do diretório raiz do projeto)
cd /caminho/para/football-manager  # voltar para a raiz
/home/thiagotn/go/bin/migrate -path ./football-api/migrations \
  -database "postgres://football:football@localhost:5433/football_dev?sslmode=disable" up

# Esperado: "... migrations applied successfully"
```

#### Opção 2: Via Make (requer golang-migrate instalado)

```bash
cd football-api-go/
docker compose up postgres -d

export DATABASE_URL="postgres://football:football@localhost:5433/football_dev?sslmode=disable"
make migrate
```

### Testes Unitários (sem banco, ~3s)

```bash
cd football-api-go/

# Rodar testes unitários com cobertura
make test

# Com race detector (detecta condições de corrida)
make test-race

# Ver relatório de cobertura em HTML
make coverage
```

**Arquivos de teste:** `tests/unit/`
- `auth_test.go` — login, signup, OTP
- `groups_test.go` — CRUD de grupos
- `matches_test.go` — partidas e sorteio
- `players_test.go` — perfil, avatares
- `middleware_test.go` — JWT, autenticação
- `team_builder_test.go` — algoritmo de sorteio
- `services_pure_test.go` — lógica pura (sem banco)
- E mais...

### Testes de Integração / E2E (com banco, ~30s)

**Pré-requisito:** Migrations devem estar aplicadas (ver seção acima).

Os testes criam e destroem seus próprios dados via API (sem fixtures pré-carregadas). OTP bypass: código `123456` funciona em qualquer ambiente sem Twilio configurado.

```bash
cd football-api-go/

# Garantir Postgres rodando (migrations já aplicadas na etapa anterior)
docker compose up postgres -d

# Configurar variáveis obrigatórias
export DATABASE_URL="postgres://football:football@localhost:5433/football_dev?sslmode=disable"
export SECRET_KEY="dev-secret-key-change-in-production"

# Rodar testes de integração (timeout 120s)
make test-integration

# Ou com verbose
go test ./tests/integration/... -v -timeout 120s
```

**Cobertura:** 17+ arquivos de teste
- Auth (signup, login, OTP, password reset)
- Grupos (CRUD, membros, skills, waitlist)
- Partidas (create, teams, attendance)
- Jogadores (profile, skills, statistics)
- Times (sorteio, draft)
- Financeiro (MRR, limites por plano)
- Votos (rating pós-partida)
- Ranking (global, por grupo)
- Assinaturas (planos)
- E mais...

### Todos os Testes (unit + integration, ~40s)

```bash
cd football-api-go/

# Pré-requisito: migrations aplicadas (ver acima)
# Garantir Postgres rodando
docker compose up postgres -d

# Configurar variáveis
export DATABASE_URL="postgres://football:football@localhost:5433/football_dev?sslmode=disable"
export SECRET_KEY="dev-secret-key-change-in-production"

# Rodar tudo em sequência
make test-all
```

---

## Credenciais Padrão (banco zerado após migrations)

| Campo | Valor |
|-------|-------|
| WhatsApp | `+5511999990000` |
| Senha | `admin123` |
| OTP | `123456` (ou qualquer código sem Twilio) |
| Role | `admin` |

### Como logar

```bash
curl -X POST http://localhost:8001/api/v2/auth/signup \
  -H "Content-Type: application/json" \
  -d '{
    "whatsapp": "+5511999990000",
    "name": "Admin Test",
    "password": "admin123"
  }'

# Retorna token JWT
```

---

## Variáveis Opcionais

Defina apenas se quiser ativar recursos específicos. Sem elas, a API funciona com degradação graceful:

| Variável | Efeito sem ela |
|----------|----------------|
| `TWILIO_ACCOUNT_SID` | OTP bypass automático (aceita qualquer código) |
| `TWILIO_AUTH_TOKEN` | ⬆ mesma situação |
| `TWILIO_FROM_NUMBER` | ⬆ mesma situação |
| `ANTHROPIC_API_KEY` | `/api/v2/chat` retorna erro 503 |
| `STRIPE_SECRET_KEY` | Endpoints de assinatura retornam erro 503 |
| `STRIPE_WEBHOOK_SECRET` | ⬆ mesma situação |
| `SUPABASE_URL` | Upload de avatar falha com 503 |
| `SUPABASE_SERVICE_ROLE_KEY` | ⬆ mesma situação |
| `VAPID_PRIVATE_KEY` | Web Push desabilitado |
| `VAPID_PUBLIC_KEY` | ⬆ mesma situação |

**Obrigatórias:**
- `DATABASE_URL` — conexão ao Postgres
- `SECRET_KEY` — chave para assinar JWT (recomendado 32+ chars)

---

## Verificação de Saúde

### Health Check

```bash
# Via Docker (porta 8001)
curl http://localhost:8001/api/v2/health

# Via Air nativo (porta 8080)
curl http://localhost:8080/api/v2/health

# Retorna (esperado):
# {"status":"ok"}
```

### Swagger / OpenAPI

Se gerado (requer `swag`):

```bash
# Ver documentação interativa
open http://localhost:8001/api/v2/docs

# Arquivo YAML
http://localhost:8001/api/v2/openapi.yaml
```

### Teste de login

```bash
# Fazer login
curl -X POST http://localhost:8001/api/v2/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "whatsapp": "+5511999990000",
    "password": "admin123"
  }'

# Retorna token JWT:
# {"token":"eyJhbGciOiJIUzI1NiIs..."}
```

---

## Targets do Makefile

Para referência rápida, os principais targets disponíveis:

| Target | Descrição |
|--------|-----------|
| `make run` | Subir com live-reload via Air |
| `make build` | Compilar binário otimizado (Linux) |
| `make test` | Testes unitários com cobertura |
| `make test-race` | Testes unitários com race detector |
| `make test-integration` | Testes de integração |
| `make test-all` | Todos os testes |
| `make lint` | Rodar golangci-lint |
| `make generate` | Regenerar código (sqlc, swagger) |
| `make migrate` | Aplicar migrations via golang-migrate |
| `make docs` | Gerar OpenAPI YAML |
| `make coverage` | Abrir relatório HTML de cobertura |
| `make clean` | Remover artefatos (./tmp, coverage.out) |

---

## Troubleshooting

### "Connection refused" ao rodar testes de integração

**Causa:** Postgres não está rodando na porta 5433.

**Solução:**
```bash
docker compose up postgres -d
# Aguarde ~5 segundos para inicializar
```

### "migrate: not found" ao rodar testes

**Causa:** O binário golang-migrate não está no PATH. É necessário instalar ou usar o caminho completo.

**Solução:**
```bash
# Instalar golang-migrate globalmente (uma única vez)
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Testar instalação
which migrate  # deve retornar algo como /home/username/go/bin/migrate

# Se ainda não estiver no PATH, adicione ao .bashrc ou .zshrc:
# export PATH="$PATH:$HOME/go/bin"
```

Depois disso, `make migrate` funcionará corretamente.

### "Migrations not found" ao rodar a API

**Causa:** Migrations estão em `../football-api/migrations/`, não em `./migrations/`.

**Solução:** Já está resolvido na imagem Docker (monta o volume correto). Se rodando nativamente:
```bash
# Garantir que você está na raiz do projeto, não em football-api-go/
cd /caminho/para/football-manager
make -C football-api-go/ migrate
```

### "Database 'football_dev' does not exist"

**Causa:** Postgres subiu mas não criou o banco.

**Solução:**
```bash
# Recrear o container
docker compose down postgres
docker compose up postgres -d
# Aguarde ~10 segundos
# Tente novamente
```

### Hot reload não funciona (Air)

**Solução:**
1. Verifique que `air` está instalado: `air --version`
2. Verifique que `.air.toml` existe no diretório corrente
3. Rode manualmente: `go build -o ./tmp/server ./cmd/server/main.go && ./tmp/server`

### Testes falhando com "too many connections"

**Causa:** Conexões de testes anteriores não foram fechadas.

**Solução:**
```bash
# Recrear banco
docker compose down postgres
docker compose up postgres -d
# Rodar testes novamente
```

---

## Status Atual dos Testes

✅ **Testes unitários:** Rodando sem problemas  
✅ **Migrations:** Todas aplicadas (44 migrações)  
⚠️ **Testes integrados:** Alguns com falhas esperadas (respostas de API podem precisar ajustes)

Os testes integrados que falham indicam endpoints que ainda não implementaram ou precisam revisar a estrutura das respostas. Você pode continuar desenvolvendo — as falhas não indicam problemas com a infraestrutura.

---

## Próximos Passos

1. **Rodar unit tests:** `make test` (valida lógica sem dependências) ✅
2. **Subir API:** `docker compose up --build` (testa integração local)
3. **Rodar integration tests:** `make test-integration` (valida fluxos E2E)
4. **Investigar falhas:** Revise os testes em `tests/integration/` que estão falhando e ajuste as respostas da API conforme necessário
5. **Testar via curl/Postman:** Usar credenciais padrão para logar e explorar

---

## Referências

- [football-api-go/Makefile](./football-api-go/Makefile)
- [football-api-go/docker-compose.yml](./football-api-go/docker-compose.yml)
- [football-api-go/.air.toml](./football-api-go/.air.toml)
- [football-api-go/tests/](./football-api-go/tests/)
- [football-api/migrations/](./football-api/migrations/) — migrations compartilhadas

---

**Última atualização:** 2026-05-26  
**Versão Go:** 1.25  
**Status:** ✅ Migrations aplicadas e testes executando (alguns integrados com falhas esperadas a serem resolvidas)
