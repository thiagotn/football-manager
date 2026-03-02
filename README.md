# Football Manager

Aplicação web para gerenciamento de grupos de futebol: agendamento de partidas, controle de presenças e convites por link.

---

## Arquitetura

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

### Produção (VPS + Traefik)

```
Navegador
    │ HTTPS
    ▼
Traefik                    portas 80 / 443  (TLS via Let's Encrypt)
    ├── rachao.app      →  Frontend (SvelteKit)
    └── api.rachao.app  →  API (FastAPI)
                            │
                            ▼
                        PostgreSQL  (sem porta pública)
```

### Componentes

| Componente | Tecnologia | Descrição |
|---|---|---|
| **Frontend** | SvelteKit 5 + Tailwind CSS | SPA com roteamento client-side. Consome a API via fetch. |
| **API** | FastAPI + SQLAlchemy (async) | REST API com autenticação JWT. Documentação automática em `/docs`. |
| **Banco** | PostgreSQL 16 | Armazena jogadores, grupos, partidas, presenças e convites. |
| **Traefik** | Traefik v3 | Proxy reverso + TLS automático (produção). |
| **Adminer** | Adminer 4 | Interface web para inspecionar o banco (opcional, via Docker profile). |

---

## Pré-requisitos

- [Docker](https://docs.docker.com/get-docker/) 24+
- [Docker Compose](https://docs.docker.com/compose/) v2+

---

## Executar localmente

> Todos os comandos abaixo devem ser executados a partir do diretório `football-api/`.

```bash
cd football-api
```

### 1. Configurar variáveis de ambiente

```bash
cp .env.example .env.docker
```

O arquivo já vem configurado para o ambiente local. Não é necessário alterar nada para rodar.

### 2. Subir os containers

```bash
docker compose up --build
# ou com Make:
make up
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

Execute a partir de `football-api/`:

```bash
make up           # Sobe tudo com build
make up-bg        # Sobe em background e exibe logs da API
make down         # Para todos os containers
make down-clean   # Para e apaga o volume do banco (dados zerados)
make logs         # Logs da API e do frontend em tempo real
make shell        # Bash dentro do container da API
make db-connect   # psql direto no banco
make adminer      # Sobe o Adminer (UI do banco)
make health       # Verifica saúde da API
make docs         # Abre o Swagger no browser
make test         # Roda os testes
```

Ou com Docker Compose diretamente:

```bash
docker compose up -d --build      # Subir em background
docker compose logs -f api        # Logs da API
docker compose down               # Parar tudo
docker compose down -v            # Parar e apagar volumes (banco zerado)
docker compose build --no-cache   # Rebuild forçado sem cache
```

---

## Estrutura do repositório

```
football-manager/
├── .github/
│   └── workflows/
│       └── deploy.yml              # CI/CD: build → GHCR → deploy no VPS
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
└── football-frontend/              # Frontend
    ├── src/
    │   ├── routes/                 # Páginas (SvelteKit file-based routing)
    │   ├── lib/
    │   │   ├── api.ts              # Client HTTP para a API
    │   │   ├── stores/             # Estado global (auth, toast)
    │   │   └── components/         # Componentes reutilizáveis
    │   └── app.css                 # Estilos globais (Tailwind)
    └── Dockerfile                  # Multi-stage: builder e production
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

### Produção (`football-api/.env.prod`)

| Variável | Descrição |
|---|---|
| `POSTGRES_DB` | Nome do banco |
| `POSTGRES_USER` | Usuário do banco |
| `POSTGRES_PASSWORD` | Senha do banco |
| `SECRET_KEY` | Gere com `openssl rand -hex 32` |
| `ACME_EMAIL` | E-mail para notificações do Let's Encrypt |

> O arquivo `.env.prod` nunca é commitado. Copie `.env.prod.example` no VPS e preencha os valores.

---

## Funcionalidades

- Cadastro e autenticação de jogadores (JWT)
- Criação e gerenciamento de grupos de futebol
- Agendamento de partidas com local e horário
- Confirmação de presença por link público (sem login obrigatório)
- Convites por link com expiração (30 min, uso único)
- Controle de administradores por grupo
