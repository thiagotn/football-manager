# Football Manager

Aplicação web para gerenciamento de grupos de futebol: agendamento de partidas, controle de presenças e convites por link.

---

## Arquitetura

```
┌─────────────────────────────────────────────────────────┐
│                     Navegador                           │
│              http://localhost:3000                      │
└────────────────────────┬────────────────────────────────┘
                         │ HTTP
┌────────────────────────▼────────────────────────────────┐
│              Frontend  (SvelteKit + Tailwind)           │
│                   container: frontend                   │
│                    porta 3000                           │
└────────────────────────┬────────────────────────────────┘
                         │ REST API (HTTP)
┌────────────────────────▼────────────────────────────────┐
│                 API  (FastAPI + Python)                 │
│                   container: api                        │
│              porta 8000  /api/v1/...                    │
│          Autenticação via JWT (Bearer token)            │
└────────────────────────┬────────────────────────────────┘
                         │ asyncpg
┌────────────────────────▼────────────────────────────────┐
│              Banco de Dados  (PostgreSQL 16)            │
│                  container: postgres                    │
│                     porta 5432                         │
└─────────────────────────────────────────────────────────┘
```

### Componentes

| Componente | Tecnologia | Descrição |
|---|---|---|
| **Frontend** | SvelteKit 5 + Tailwind CSS | SPA com roteamento client-side. Consome a API via fetch. |
| **API** | FastAPI + SQLAlchemy (async) | REST API com autenticação JWT. Documentação automática em `/docs`. |
| **Banco** | PostgreSQL 16 | Armazena jogadores, grupos, partidas, presenças e convites. |
| **Adminer** | Adminer 4 | Interface web para inspecionar o banco (opcional, via Docker profile). |

---

## Pré-requisitos

- [Docker](https://docs.docker.com/get-docker/) 24+
- [Docker Compose](https://docs.docker.com/compose/) v2+

---

## Executar localmente

### 1. Configurar variáveis de ambiente

Copie o template da API e ajuste se necessário:

```bash
cp football-api/.env.example football-api/.env.docker
```

> Por padrão o arquivo já está configurado para o ambiente local. Não é necessário alterar nada para rodar.

### 2. Subir os containers

```bash
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
| Documentação interativa (Swagger) | http://localhost:8000/docs |
| Documentação alternativa (ReDoc) | http://localhost:8000/redoc |

### Adminer (opcional)

Para abrir a interface web do banco de dados:

```bash
docker compose --profile tools up adminer
```

Acesse http://localhost:8080 e conecte com:
- **Servidor:** `postgres`
- **Usuário:** `postgres`
- **Senha:** `football123`
- **Banco:** `football`

---

## Comandos úteis

```bash
# Subir em background
docker compose up -d --build

# Ver logs da API em tempo real
docker compose logs -f api

# Parar tudo
docker compose down

# Remover também o volume do banco (apaga todos os dados)
docker compose down -v

# Rebuild forçado sem cache
docker compose build --no-cache
```

---

## Estrutura do repositório

```
football-manager/
├── docker-compose.yml          # Orquestração local dos containers
├── football-api/               # Backend
│   ├── app/
│   │   ├── main.py             # Entrypoint FastAPI
│   │   ├── models/             # Modelos SQLAlchemy
│   │   ├── routers/            # Endpoints da API
│   │   ├── schemas/            # Schemas Pydantic
│   │   └── core/               # Config, segurança, DB
│   ├── migrations/             # Scripts SQL iniciais
│   ├── Dockerfile
│   ├── pyproject.toml
│   └── .env.example            # Template de variáveis de ambiente
└── football-frontend/          # Frontend
    ├── src/
    │   ├── routes/             # Páginas (SvelteKit file-based routing)
    │   ├── lib/
    │   │   ├── api.ts          # Client HTTP para a API
    │   │   ├── stores/         # Estado global (auth, toast)
    │   │   └── components/     # Componentes reutilizáveis
    │   └── app.css             # Estilos globais (Tailwind)
    └── Dockerfile
```

---

## Variáveis de ambiente

### API (`football-api/.env.docker`)

| Variável | Descrição | Exemplo |
|---|---|---|
| `DATABASE_URL` | String de conexão PostgreSQL | `postgresql+asyncpg://user:pass@host/db` |
| `SECRET_KEY` | Chave para assinar tokens JWT | string aleatória de 32+ chars |
| `CORS_ORIGINS` | Origens permitidas pelo CORS | `http://localhost:3000` |
| `APP_ENV` | Ambiente da aplicação | `development` / `production` |
| `DEBUG` | Modo debug | `true` / `false` |

> Gere uma `SECRET_KEY` segura para produção com: `openssl rand -hex 32`

---

## Funcionalidades

- Cadastro e autenticação de jogadores (JWT)
- Criação e gerenciamento de grupos de futebol
- Agendamento de partidas com local e horário
- Confirmação de presença por link público (sem login obrigatório)
- Convites por link com expiração (30 min, uso único)
- Controle de administradores por grupo
