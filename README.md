<p align="center">
  <img src="football-frontend/static/logo.png" alt="rachao.app" width="280" />
</p>

<p align="center">
  <a href="https://github.com/thiagotn/football-manager/actions/workflows/main.yml"><img src="https://github.com/thiagotn/football-manager/actions/workflows/main.yml/badge.svg" alt="Build & Deploy Web" /></a>
  <a href="https://github.com/thiagotn/football-manager/actions/workflows/build-twa.yml"><img src="https://github.com/thiagotn/football-manager/actions/workflows/build-twa.yml/badge.svg" alt="Build & Deploy TWA Android" /></a>
  <a href="https://codecov.io/gh/thiagotn/football-manager"><img src="https://codecov.io/gh/thiagotn/football-manager/branch/main/graph/badge.svg" alt="codecov" /></a>
</p>

<p align="center">PWA para gerenciamento de grupos de futebol.</p>

<p align="center">
  <a href="#arquitetura">Arquitetura</a> ·
  <a href="#pré-requisitos">Pré-requisitos</a> ·
  <a href="#executar-localmente">Executar localmente</a> ·
  <a href="#comandos-úteis">Comandos úteis</a> ·
  <a href="#estrutura-do-repositório">Estrutura</a> ·
  <a href="#variáveis-de-ambiente">Variáveis de ambiente</a> ·
  <a href="#deploy-em-produção-vps">Deploy</a> ·
  <a href="#funcionalidades">Funcionalidades</a>
</p>

---

## Arquitetura

<p align="center">
  <img src="docs/arquitetura_rachao_app.svg" alt="Arquitetura rachao.app" />
</p>

### Desenvolvimento local

```
Navegador
    │ http://localhost:3000
    ▼
Frontend (SvelteKit)       porta 3000
    │ http://localhost:8000/api/v1
    ▼
API (FastAPI)              porta 8000
    │
    ▼
PostgreSQL                 porta 5432
```

### Produção (VPS + Traefik + Supabase)

```
Navegador
    │ HTTPS
    ▼
Traefik                    portas 80 / 443  (TLS via Let's Encrypt)
    ├── rachao.app      →  Frontend (SvelteKit)
    └── api.rachao.app  →  API (FastAPI)
                            │
                            ▼
                        Supabase PostgreSQL  (sa-east-1, São Paulo)
```

### Componentes

| Componente | Tecnologia | Descrição |
|---|---|---|
| **Frontend** | SvelteKit 5 + Tailwind CSS | SPA com roteamento client-side. Consome a API via fetch. |
| **API** | FastAPI + SQLAlchemy (async) | REST API com autenticação JWT. Documentação automática em `/docs`. |
| **Banco (local)** | PostgreSQL 16 (Docker) | Instância local para desenvolvimento. |
| **Banco (prod)** | Supabase PostgreSQL | Banco gerenciado na região sa-east-1 (São Paulo). |
| **Traefik** | Traefik v3 | Proxy reverso + TLS automático (produção). |
| **Adminer** | Adminer 4 | Interface web para inspecionar o banco (opcional, via Docker profile). |
| **E2E** | Playwright + pytest (Python) | Testes end-to-end dos cenários principais. Roda em CI a cada push. |

---

## Pré-requisitos

- [Docker](https://docs.docker.com/get-docker/) 24+
- [Docker Compose](https://docs.docker.com/compose/) v2+

---

## Executar localmente

### 1. Configurar variáveis de ambiente

```bash
cp football-api/.env.example football-api/.env.docker
```

O arquivo já vem configurado para o ambiente local. Não é necessário alterar nada para rodar.

### 2. Subir os containers

A partir da **raiz do repositório**:

```bash
make up        # build normal + sobe API, frontend e PostgreSQL
# ou:
docker compose up --build
```

Na primeira execução o Docker irá:
1. Construir as imagens da API e do frontend
2. Iniciar o PostgreSQL e aplicar as migrations automaticamente
3. Iniciar a API e o frontend

### 3. Acessar

| Serviço | URL |
|---|---|
| Frontend | http://localhost:3000 |
| API (REST) | http://localhost:8000/api/v1 |
| Swagger (docs interativa) | http://localhost:8000/docs |
| ReDoc | http://localhost:8000/redoc |

### Login inicial (admin)

```
WhatsApp: 11999990000
Senha:    admin123
```

### Adminer (opcional)

```bash
make adminer
# ou: docker compose --profile tools up adminer -d
```

Acesse http://localhost:8080 e conecte com:
- **Servidor:** `postgres`
- **Usuário:** `postgres`
- **Senha:** `football123`
- **Banco:** `football`

---

## Comandos úteis

### Raiz do repositório

```bash
make up           # Build + sobe API, frontend e PostgreSQL
make rebuild      # Rebuild sem cache (--no-cache) + sobe tudo
make down         # Para todos os containers
make down-clean   # Para e apaga o volume do banco (dados zerados)
make logs         # Logs da API e do frontend em tempo real
```

### Dentro de `football-api/` (comandos adicionais)

```bash
make up-bg        # Sobe em background e exibe logs da API
make shell        # Bash dentro do container da API
make db-connect   # psql direto no banco
make adminer      # Sobe o Adminer (UI do banco)
make health       # Verifica saúde da API
make docs         # Abre o Swagger no browser
make test         # Roda os testes unitários
```

Ou com Docker Compose diretamente (a partir da raiz):

```bash
docker compose up -d --build             # Subir em background
docker compose logs -f api frontend      # Logs da API e frontend
docker compose down                      # Parar tudo
docker compose down -v                   # Parar e apagar volumes (banco zerado)
docker compose build --no-cache          # Rebuild forçado sem cache
```

---

## Estrutura do repositório

```
football-manager/
├── .github/
│   └── workflows/
│       ├── main.yml                # CI/CD: testes unitários → E2E → build → deploy (unificado)
│       └── deploy-monitoring.yml   # Deploy da stack de monitoramento (disparo manual)
├── scripts/
│   └── setup-vps.sh                # Prepara o VPS Ubuntu 24.04 para receber o deploy
├── football-api/                   # Backend
│   ├── app/
│   │   ├── main.py                 # Entrypoint FastAPI
│   │   ├── models/                 # Modelos SQLAlchemy
│   │   ├── routers/                # Endpoints da API
│   │   ├── schemas/                # Schemas Pydantic
│   │   └── core/                   # Config, segurança, DB
│   ├── migrations/                 # Scripts SQL (aplicados automaticamente na 1ª vez)
│   ├── Dockerfile                  # Multi-stage: dev e production
│   ├── docker-compose.yml          # Ambiente local (portas expostas, hot-reload)
│   ├── docker-compose.prod.yml     # Produção (Traefik, imagens do GHCR)
│   ├── .env.example                # Template para desenvolvimento local
│   ├── .env.prod.example           # Template para produção
│   ├── Makefile                    # Atalhos para comandos comuns
│   └── pyproject.toml
├── football-frontend/              # Frontend
│   ├── src/
│   │   ├── routes/                 # Páginas (SvelteKit file-based routing)
│   │   ├── lib/
│   │   │   ├── api.ts              # Client HTTP para a API
│   │   │   ├── stores/             # Estado global (auth, toast)
│   │   │   └── components/         # Componentes reutilizáveis
│   │   └── app.css                 # Estilos globais (Tailwind)
│   └── Dockerfile                  # Multi-stage: builder e production
└── football-e2e/                   # Testes end-to-end
    ├── conftest.py                 # Fixtures: login, contextos autenticados
    ├── pages/                      # Page Object Model
    └── tests/                      # Suites por domínio (auth, groups, matches…)
```

---

## Variáveis de ambiente

### Local (`football-api/.env.docker`)

| Variável | Descrição | Padrão local |
|---|---|---|
| `DATABASE_URL` | String de conexão PostgreSQL | `postgresql+asyncpg://postgres:football123@postgres:5432/football` |
| `SECRET_KEY` | Chave para assinar tokens JWT | `local-dev-secret-key-...` |
| `CORS_ORIGINS` | Origens permitidas pelo CORS | `http://localhost:3000` |
| `APP_ENV` | Ambiente da aplicação | `development` |
| `DEBUG` | Modo debug | `true` |

### Produção (`/opt/football-manager/.env.prod` no VPS)

| Variável | Descrição |
|---|---|
| `DATABASE_URL` | Connection string do Supabase (`postgresql+asyncpg://...`) — injetado via GitHub Actions secret |
| `SECRET_KEY` | Gerado automaticamente pelo `setup-vps.sh` |
| `ACME_EMAIL` | E-mail para notificações do Let's Encrypt |

> O arquivo `.env.prod` nunca é commitado. Ele é criado pelo script de setup diretamente no VPS e atualizado pelo workflow de deploy.

---

## Deploy em produção (VPS)

### Pré-requisitos

- VPS com **Ubuntu 24.04 LTS** (Hostinger KVM ou similar)
- Acesso SSH como root
- Domínio `rachao.app` (e `api.rachao.app`, `www.rachao.app`) com DNS apontando para o IP do VPS
- Repositório clonado localmente com as [secrets do GitHub configuradas](#secrets-do-github)

---

### Passo 1 — Preparar o VPS (executar uma única vez)

Conecte via SSH e execute o script de setup:

```bash
ssh root@<IP_DO_VPS>

# Baixa e executa o script de setup
bash <(curl -fsSL https://raw.githubusercontent.com/thiagotn/football-manager/main/scripts/setup-vps.sh)
```

O script realiza automaticamente:
- Atualização do sistema
- Instalação do **Docker CE** + **Docker Compose v2** (repositório oficial)
- Configuração do **firewall UFW** (libera SSH, 80/tcp e 443/tcp)
- Criação do diretório `/opt/football-manager`
- Geração do arquivo `.env.prod` com `SECRET_KEY` pré-preenchida

---

### Passo 2 — Configurar as variáveis de produção

```bash
nano /opt/football-manager/.env.prod
```

Preencha o campo obrigatório:

```env
ACME_EMAIL=seu@email.com        # para notificações de certificado SSL
```

> `SECRET_KEY` já foi gerada pelo setup. `DATABASE_URL` é injetado automaticamente pelo workflow via GitHub Actions secret.

---

### Passo 3 — Configurar as secrets no GitHub

Acesse **Settings → Secrets and variables → Actions** no repositório e crie:

| Secret | Valor |
|---|---|
| `VPS_HOST` | IP público do VPS |
| `VPS_USER` | `root` (ou usuário com acesso ao Docker) |
| `VPS_SSH_KEY` | Conteúdo da chave privada SSH (`~/.ssh/id_ed25519`) |
| `VPS_PORT` | `22` (padrão) |

> Para gerar um par de chaves dedicado ao deploy:
> ```bash
> ssh-keygen -t ed25519 -C "github-deploy" -f ~/.ssh/football_deploy
> ssh-copy-id -i ~/.ssh/football_deploy.pub root@<IP_DO_VPS>
> # Cole o conteúdo de ~/.ssh/football_deploy no secret VPS_SSH_KEY
> ```

---

### Passo 4 — Fazer o deploy

No GitHub, acesse **Actions → Deploy to Production → Run workflow → Run workflow**.

O pipeline executa os jobs em sequência:

```
Run workflow (manual)
       │
       ▼
  Job: unit-tests  (testes unitários da API)
       │
       ▼
  Job: e2e         (testes E2E com stack completa)
       │
       ▼
  Job: build
  ├── Build API image   → ghcr.io/thiagotn/football-manager-api:latest
  └── Build Frontend    → ghcr.io/thiagotn/football-manager-frontend:latest
       │
       ▼
  Job: deploy
  ├── SCP: envia docker-compose.prod.yml + migrations para o VPS
  └── SSH: docker compose pull → up -d → image prune
```

> Use o input **"Skip tests"** para pular testes e fazer deploy direto (útil para re-deploy sem alterações de código).

> O certificado TLS é emitido automaticamente pelo Traefik via Let's Encrypt na primeira vez que o deploy sobe. Aguarde ~30 segundos após o primeiro deploy para o certificado estar ativo.

---

### Secrets do GitHub

---

## Funcionalidades

- Cadastro e autenticação de jogadores (JWT)
- Criação e gerenciamento de grupos de futebol
- Agendamento de partidas com local e horário
- Confirmação de presença por link público (sem login obrigatório)
- Convites por link com expiração (30 min, uso único)
- Controle de administradores por grupo
