# Football Manager — API

[![Unit Tests](https://github.com/thiagotn/football-manager/actions/workflows/unit-tests.yml/badge.svg)](https://github.com/thiagotn/football-manager/actions/workflows/unit-tests.yml)
[![E2E Tests](https://github.com/thiagotn/football-manager/actions/workflows/e2e.yml/badge.svg)](https://github.com/thiagotn/football-manager/actions/workflows/e2e.yml)
[![Deploy to Production](https://github.com/thiagotn/football-manager/actions/workflows/deploy.yml/badge.svg)](https://github.com/thiagotn/football-manager/actions/workflows/deploy.yml)
[![codecov](https://codecov.io/gh/thiagotn/football-manager/branch/main/graph/badge.svg)](https://codecov.io/gh/thiagotn/football-manager)

Backend da aplicação: REST API construída com FastAPI + SQLAlchemy assíncrono + PostgreSQL.

## Stack

| Camada | Tecnologia |
|--------|-----------|
| **Backend** | Python 3.12 + FastAPI + SQLAlchemy 2.0 (async) |
| **Banco** | PostgreSQL 16 |
| **Auth** | JWT (HS256) + bcrypt |
| **Frontend** | SvelteKit 5 + TailwindCSS |
| **Proxy (prod)** | Traefik v3 + Let's Encrypt |
| **CI/CD** | GitHub Actions → GHCR → VPS |

---

## Desenvolvimento local

> Todos os comandos abaixo devem ser executados a partir deste diretório (`football-api/`).

### 1. Configurar o ambiente

```bash
cp .env.example .env.docker
```

O `.env.docker` já vem configurado para rodar localmente. Não é necessário alterar nada.

### 2. Subir os containers

```bash
make up
# ou: docker compose up --build
```

Na primeira execução o Docker irá:
1. Construir as imagens da API (com hot-reload) e do frontend
2. Iniciar o PostgreSQL e aplicar as migrations automaticamente
3. Iniciar a API e o frontend

### 3. Acessar

| Serviço | URL |
|---------|-----|
| **Frontend** | http://localhost:3000 |
| **API** | http://localhost:8000/api/v1 |
| **Swagger** | http://localhost:8000/docs |
| **ReDoc** | http://localhost:8000/redoc |
| **Adminer** | `make adminer` → http://localhost:8080 |

### 4. Monitoramento (opcional)

```bash
make monitoring
```

| Serviço | URL | Credenciais |
|---------|-----|-------------|
| **Grafana** | http://localhost:3001 | admin / admin |
| **Prometheus** | http://localhost:9090 | — |
| **Uptime Kuma** | http://localhost:3002 | cadastro no 1º acesso |
| **cAdvisor** | http://localhost:8081 | — |

> **Pré-requisito:** a stack principal deve estar rodando (`make up`) antes de subir o monitoramento.

Dashboards recomendados para importar no Grafana (Dashboards → Import → ID):
- `1860` — Node Exporter Full
- `14282` — Docker cAdvisor
- `22676` — FastAPI Observability

### Login inicial (admin)

```
WhatsApp: 11999990000
Senha:    admin123
```

---

## Comandos úteis

```bash
make up               # Sobe tudo com build
make up-bg            # Sobe em background e exibe logs da API
make down             # Para todos os containers
make down-clean       # Para e apaga o volume do banco (dados zerados)
make logs             # Logs da API e do frontend em tempo real
make shell            # Bash dentro do container da API
make db-connect       # psql direto no banco
make adminer          # Sobe o Adminer (UI do banco)
make health           # Verifica saúde da API
make docs             # Abre o Swagger no browser
make test             # Roda os testes
make test-cov         # Roda os testes com relatório de cobertura
make monitoring       # Sobe a stack de monitoramento (Grafana, Prometheus, etc.)
make monitoring-down  # Para a stack de monitoramento
make monitoring-logs  # Logs do Prometheus e Grafana
```

---

## Fluxo ponta a ponta

```
1. Admin faz login → recebe JWT
2. Admin cria grupo "Futebol GQC"
3. Admin gera link de convite (expira em 30 min, uso único)
4. Jogador acessa o link → preenche nome, WhatsApp e senha → entra no grupo
5. Admin cria partida (data, hora, local) → sistema gera hash único
6. Partida gera URL pública: /match/<hash>
7. Jogadores confirmam/recusam presença via frontend
8. Qualquer pessoa pode ver a lista de confirmados via URL pública
```

---

## API — Rotas principais

### Auth
| Método | Rota | Descrição |
|--------|------|-----------|
| `POST` | `/api/v1/auth/login` | Login (retorna JWT) |
| `GET`  | `/api/v1/auth/me` | Dados do usuário logado |

### Jogadores (requer admin)
| Método | Rota | Descrição |
|--------|------|-----------|
| `GET`    | `/api/v1/players` | Lista jogadores |
| `POST`   | `/api/v1/players` | Cria jogador |
| `GET`    | `/api/v1/players/{id}` | Detalhes |
| `PATCH`  | `/api/v1/players/{id}` | Atualiza |
| `DELETE` | `/api/v1/players/{id}` | Desativa (soft delete) |

### Grupos
| Método | Rota | Descrição |
|--------|------|-----------|
| `GET`  | `/api/v1/groups` | Meus grupos |
| `POST` | `/api/v1/groups` | Cria grupo (admin) |
| `GET`  | `/api/v1/groups/{id}` | Detalhes + membros |
| `POST` | `/api/v1/groups/{id}/members` | Adiciona membro |
| `DELETE` | `/api/v1/groups/{id}/members/{pid}` | Remove membro |

### Partidas
| Método | Rota | Descrição |
|--------|------|-----------|
| `GET`  | `/api/v1/groups/{id}/matches` | Lista partidas do grupo |
| `POST` | `/api/v1/groups/{id}/matches` | Cria partida |
| `POST` | `/api/v1/groups/{id}/matches/{mid}/attendance` | Confirma/recusa presença |
| `GET`  | `/api/v1/matches/public/{hash}` | **Público** — dados da partida via hash |

### Convites
| Método | Rota | Descrição |
|--------|------|-----------|
| `POST` | `/api/v1/invites` | Gera convite (admin do grupo) |
| `GET`  | `/api/v1/invites/{token}` | Valida token (público) |
| `POST` | `/api/v1/invites/{token}/accept` | Aceita convite + cria conta |

---

## Migrations

As migrations ficam em `migrations/` e são aplicadas automaticamente pela API no startup (via `app/db/migrate.py`), tanto local quanto em produção. O estado é rastreado na tabela `schema_migrations`.

| Arquivo | Descrição |
|---|---|
| `001_initial_schema.sql` | Schema base, enums e seed do admin |
| `002_seed_dev.sql` | Dados de exemplo (dev only) |
| `003_match_number.sql` | Numeração sequencial de partidas |
| `004_match_address.sql` | Campo de endereço para Google Maps |
| `005_match_venue_fields.sql` | Tipo de quadra e jogadores por time |
| `006_match_max_players.sql` | Limite máximo de jogadores por partida |

> Novas migrations devem ser adicionadas como `00N_descricao.sql`. Em produção, o arquivo é copiado automaticamente pelo workflow de CI/CD e aplicado no próximo start do container de banco.

---

## Produção

O deploy em produção usa **Traefik como proxy reverso** e **GitHub Actions** para CI/CD.

### Como funciona

```
git push → main
    │
    ▼
GitHub Actions (.github/workflows/deploy.yml)
    ├── Build API image   → ghcr.io/thiagotn/football-manager-api:latest
    ├── Build Frontend    → ghcr.io/thiagotn/football-manager-frontend:latest
    │
    ▼
SSH no VPS
    ├── docker compose pull   (baixa as novas imagens)
    └── docker compose up -d  (reinicia os containers)
```

### Configurar para produção

1. Copie o template de variáveis no VPS:

```bash
cp .env.prod.example .env.prod
nano .env.prod
```

2. Suba com o compose de produção:

```bash
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d
```

Consulte o `README.md` da raiz do repositório para o guia completo de deploy.

### Diferenças entre os ambientes

| | Local (`docker-compose.yml`) | Produção (`docker-compose.prod.yml`) |
|---|---|---|
| Acesso | `localhost:3000` / `localhost:8000` | `rachao.app` / `api.rachao.app` |
| TLS | Não | Sim (Let's Encrypt automático) |
| Traefik | Não | Sim |
| API target | `dev` (hot-reload) | `production` (multi-worker) |
| Imagens | Build local | Pull do GHCR |
| Banco | PostgreSQL 16 via Docker (porta 5432 exposta) | Supabase — sa-east-1, São Paulo (sem container local) |
