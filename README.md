<p align="center">
  <img src="football-frontend/static/logo.png" alt="rachao.app" width="280" />
</p>

<p align="center">
  <a href="https://github.com/thiagotn/football-manager/actions/workflows/main.yml"><img src="https://github.com/thiagotn/football-manager/actions/workflows/main.yml/badge.svg" alt="Build & Deploy Web" /></a>
  <a href="https://github.com/thiagotn/football-manager/actions/workflows/build-twa.yml"><img src="https://github.com/thiagotn/football-manager/actions/workflows/build-twa.yml/badge.svg" alt="Build & Deploy TWA Android" /></a>
  <a href="https://github.com/thiagotn/football-manager/actions/workflows/deploy-monitoring.yml"><img src="https://github.com/thiagotn/football-manager/actions/workflows/deploy-monitoring.yml/badge.svg" alt="Deploy Monitoring Stack" /></a>
  <a href="https://codecov.io/gh/thiagotn/football-manager"><img src="https://codecov.io/gh/thiagotn/football-manager/graph/badge.svg" alt="codecov" /></a>
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
  <a href="#funcionalidades">Funcionalidades</a> ·
  <a href="#servidor-mcp-football-mcp">MCP</a>
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
| **MCP** | Python + FastMCP | Servidor MCP que expõe a API do rachao.app para agentes de IA (Claude). |

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
├── football-e2e/                   # Testes end-to-end
│   ├── conftest.py                 # Fixtures: login, contextos autenticados
│   ├── pages/                      # Page Object Model
│   └── tests/                      # Suites por domínio (auth, groups, matches…)
└── football-mcp/                   # Servidor MCP (Model Context Protocol)
    ├── rachao_mcp/
    │   ├── server.py               # Entrypoint FastMCP — registra tools read/write
    │   ├── auth.py                 # Leitura e validação do RACHAO_TOKEN
    │   ├── client.py               # Cliente HTTP (httpx) para a API
    │   └── tools/                  # Tools por domínio (groups, matches, players, teams)
    ├── tests/                      # Testes unitários do servidor MCP
    ├── Dockerfile                  # Imagem para execução em produção
    ├── Makefile                    # install / dev / register / test
    └── pyproject.toml
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

#### Acesso ao VPS (`main.yml` + `deploy-monitoring.yml`)

| Secret | Descrição |
|--------|-----------|
| `VPS_HOST` | IP público do VPS |
| `VPS_USER` | Usuário SSH (ex: `root`) |
| `VPS_SSH_KEY` | Chave privada SSH (`~/.ssh/id_ed25519`) |
| `VPS_PORT` | Porta SSH (padrão: `22`) |

> Para gerar um par de chaves dedicado ao deploy:
> ```bash
> ssh-keygen -t ed25519 -C "github-deploy" -f ~/.ssh/football_deploy
> ssh-copy-id -i ~/.ssh/football_deploy.pub root@<IP_DO_VPS>
> # Cole o conteúdo de ~/.ssh/football_deploy no secret VPS_SSH_KEY
> ```

#### Banco de dados (`main.yml`)

| Secret | Descrição |
|--------|-----------|
| `DATABASE_URL` | Connection string do Supabase (`postgresql+asyncpg://...`) |
| `SUPABASE_URL` | URL do projeto Supabase (para Storage) |
| `SUPABASE_SERVICE_ROLE_KEY` | Chave de serviço do Supabase |

#### Autenticação / OTP (`main.yml`)

| Secret | Descrição |
|--------|-----------|
| `TWILIO_ACCOUNT_SID` | SID da conta Twilio |
| `TWILIO_AUTH_TOKEN` | Token de autenticação Twilio |
| `TWILIO_VERIFY_SID` | SID do serviço Twilio Verify (OTP via WhatsApp) |

#### Pagamentos — Stripe (`main.yml`)

| Secret | Descrição |
|--------|-----------|
| `STRIPE_SECRET_KEY` | Chave secreta da API Stripe |
| `STRIPE_WEBHOOK_SECRET` | Segredo para validar webhooks do Stripe |
| `STRIPE_PRICE_BASIC_MONTHLY` | ID do preço Stripe — plano Basic mensal |
| `STRIPE_PRICE_BASIC_YEARLY` | ID do preço Stripe — plano Basic anual |
| `STRIPE_PRICE_PRO_MONTHLY` | ID do preço Stripe — plano Pro mensal |
| `STRIPE_PRICE_PRO_YEARLY` | ID do preço Stripe — plano Pro anual |

#### Web Push / VAPID (`main.yml`)

| Secret | Descrição |
|--------|-----------|
| `VAPID_PUBLIC_KEY` | Chave pública VAPID para Web Push |
| `VAPID_PRIVATE_KEY` | Chave privada VAPID para Web Push |
| `VAPID_CLAIMS_EMAIL` | E-mail do remetente VAPID (ex: `mailto:admin@rachao.app`) |

#### MCP (`main.yml`)

| Secret | Descrição |
|--------|-----------|
| `MCP_RACHAO_TOKEN` | Token JWT usado pelo servidor MCP para autenticar na API |
| `MCP_SECRET_KEY` | Chave secreta para assinar tokens MCP internos |

#### Frontend — dados legais (`main.yml`)

Injetados como variáveis de build públicas (`PUBLIC_*`):

| Secret | Descrição |
|--------|-----------|
| `LEGAL_CONTROLLER_NAME` | Nome do controlador de dados (LGPD) |
| `LEGAL_CONTROLLER_DOC` | CPF/CNPJ do controlador |
| `LEGAL_FORUM_CITY` | Foro competente (ex: `São Paulo`) |
| `LEGAL_CONTACT_EMAIL` | E-mail de contato para questões legais |

#### CI (`main.yml`)

| Secret | Descrição |
|--------|-----------|
| `CODECOV_TOKEN` | Token para upload de cobertura de testes no Codecov |

#### Monitoramento (`deploy-monitoring.yml`)

| Secret | Descrição |
|--------|-----------|
| `GRAFANA_ADMIN_USER` | Usuário admin do Grafana (padrão: `admin`) |
| `GRAFANA_ADMIN_PASSWORD` | Senha admin do Grafana |
| `TELEGRAM_BOT_TOKEN` | Token do bot Telegram para alertas |
| `TELEGRAM_CHAT_ID` | Chat ID do Telegram que recebe os alertas |

#### Android / TWA (`build-twa.yml`)

| Secret | Descrição |
|--------|-----------|
| `ANDROID_KEYSTORE_BASE64` | Keystore de assinatura do APK em Base64 |
| `ANDROID_STORE_PASSWORD` | Senha do keystore |
| `ANDROID_KEY_ALIAS` | Alias da chave no keystore |
| `ANDROID_KEY_PASSWORD` | Senha da chave |
| `GOOGLE_PLAY_SERVICE_ACCOUNT_JSON` | JSON da conta de serviço para publicação na Play Store |

#### Variáveis manuais no VPS (`.env.prod`)

Estas não são GitHub secrets — devem ser definidas diretamente em `/opt/football-manager/.env.prod` no VPS:

| Variável | Descrição |
|----------|-----------|
| `ANTHROPIC_API_KEY` | Chave da API Anthropic (assistente IA / chat) |
| `LLM_MODEL` | Modelo a usar (padrão: `claude-haiku-4-5`) |
| `ACME_EMAIL` | E-mail para notificações do Let's Encrypt |

---

### Passo 4 — Fazer o deploy

No GitHub, acesse **Actions → Build & Deploy Web → Run workflow → Run workflow**.

O pipeline executa os seguintes jobs:

```
Run workflow (manual)
       │
       ▼
  Job: changes          (detecta quais paths mudaram — pula jobs desnecessários)
       │
       ├──────────────────────────────────┐
       ▼                                  ▼
  Job: unit-tests       (API)        Job: mcp-tests        (MCP)
  Job: npm-audit        (frontend)
       │
       ▼
  Job: e2e              (stack Docker completa + Playwright)
       │
       ▼
  Job: build
  ├── Build & push API image     → ghcr.io/thiagotn/football-manager-api:latest
  ├── Build & push MCP image     → ghcr.io/thiagotn/football-manager-mcp:latest
  └── Build & push Frontend      → ghcr.io/thiagotn/football-manager-frontend:latest
       │
       ▼
  Job: deploy
  ├── SCP: envia docker-compose.prod.yml + traefik-dynamic.yml + migrations para o VPS
  └── SSH: atualiza .env.prod → docker compose pull → up -d --force-recreate → image prune
```

> O certificado TLS é emitido automaticamente pelo Traefik via Let's Encrypt na primeira vez que o deploy sobe. Aguarde ~30 segundos após o primeiro deploy para o certificado estar ativo.

---

## URLs de produção

### Aplicação

| Serviço | URL | Descrição |
|---------|-----|-----------|
| App | https://rachao.app | PWA principal |
| API | https://api.rachao.app | REST API |
| Swagger | https://api.rachao.app/docs | Documentação interativa (OpenAPI) |
| ReDoc | https://api.rachao.app/redoc | Documentação alternativa |
| MCP | https://mcp.rachao.app | Servidor MCP para agentes de IA |

### Monitoramento

| Serviço | URL | Auth | Descrição |
|---------|-----|------|-----------|
| Grafana | https://grafana.rachao.app | Login nativo | Dashboards de métricas |
| Prometheus | https://prometheus.rachao.app | Basic Auth (`admin` / ver `traefik-dynamic.yml`) | Banco de métricas |
| Uptime Kuma | https://uptime.rachao.app | Login nativo | Monitoramento de uptime |
| Status page | https://status.rachao.app | Pública | Redirect para `/status/rachao` no Uptime Kuma |

---

## Funcionalidades

- Cadastro e autenticação de jogadores (JWT)
- Criação e gerenciamento de grupos de futebol
- Agendamento de partidas com local e horário
- Confirmação de presença por link público (sem login obrigatório)
- Convites por link com expiração (30 min, uso único)
- Controle de administradores por grupo

---

## Servidor MCP (`football-mcp/`)

O **rachao MCP** expõe a API do rachao.app como um servidor [Model Context Protocol](https://modelcontextprotocol.io), permitindo que agentes de IA (como o Claude) interajam com grupos, partidas e jogadores de forma natural.

### Tools disponíveis

| Tool | Tipo | Descrição |
|------|------|-----------|
| `list_groups` | read | Lista todos os grupos do jogador autenticado |
| `get_group` | read | Retorna detalhes de um grupo |
| `get_group_stats` | read | Estatísticas e métricas de um grupo |
| `list_matches` | read | Lista partidas de um grupo |
| `get_match` | read | Retorna detalhes de uma partida pelo hash |
| `discover_matches` | read | Descobre partidas disponíveis para confirmação |
| `list_players` | read | Lista jogadores de um grupo |
| `get_my_stats` | read | Estatísticas pessoais do jogador autenticado |
| `get_ranking` | read | Ranking de jogadores |
| `get_teams` | read | Times sorteados de uma partida |
| `create_match` | write | Cria uma nova partida |
| `update_match` | write | Atualiza dados de uma partida |
| `set_attendance` | write | Confirma ou cancela presença em uma partida |
| `draw_teams` | write | Realiza o sorteio de times de uma partida |

### Variáveis de ambiente

| Variável | Obrigatória | Descrição |
|----------|-------------|-----------|
| `RACHAO_TOKEN` | Sim | JWT de autenticação obtido via login na API |
| `RACHAO_API_URL` | Não | URL base da API (padrão: `https://api.rachao.app/api/v1`) |
| `RACHAO_MCP_READ_ONLY` | Não | `true` para desabilitar tools de escrita |
| `RACHAO_MCP_ALLOWED_TOOLS` | Não | Lista separada por vírgula de tools permitidas |

### Uso local

```bash
cd football-mcp

# 1. Instalar dependências
make install   # ou: uv pip install -e ".[dev]"

# 2. Executar o servidor (stdio)
make dev RACHAO_TOKEN=<jwt>

# 3. Registrar no Claude CLI
make register RACHAO_TOKEN=<jwt>

# 4. Rodar os testes
make test
```

### Integração com Claude CLI

```bash
claude mcp add rachao \
  -e RACHAO_TOKEN="<jwt>" \
  -e RACHAO_API_URL="https://api.rachao.app/api/v1" \
  -- /caminho/para/football-mcp/.venv/bin/python -m rachao_mcp
```

Após registrado, o Claude passa a ter acesso a todos os dados e ações do rachao.app diretamente na conversa.
