# ⚽ Football Manager — API + Frontend

Sistema completo para gestão de grupos e partidas de futebol.

## Stack

| Camada | Tecnologia |
|--------|-----------|
| **Backend** | Python 3.12 + FastAPI + SQLAlchemy 2.0 (async) |
| **Banco** | PostgreSQL 16 (compatível com Supabase) |
| **Auth** | JWT (HS256) + bcrypt |
| **Frontend** | SvelteKit 2 + TailwindCSS 3 |
| **Infra** | Docker Compose |

---

## Subindo o projeto

### Pré-requisitos
- Docker Desktop (ou Docker Engine + Compose Plugin)
- Make (opcional)

```bash
# Clone e suba tudo
git clone <repo>
cd football-manager
make up
# ou: docker compose up --build
```

Na primeira vez demora ~2-3 minutos (build das imagens).

### URLs após o `up`:

| Serviço | URL |
|---------|-----|
| **Frontend** | http://localhost:3000 |
| **API** | http://localhost:8000 |
| **API Docs** | http://localhost:8000/docs |
| **Adminer** | `make adminer` → http://localhost:8080 |
| **Postgres** | localhost:5432 |

### Login inicial (admin)

```
WhatsApp: 11999990000
Senha:    admin123
```

> ⚠️ Troque a senha em produção!

---

## Fluxo ponta a ponta

```
1. Admin faz login → recebe JWT
2. Admin cria grupo "Futebol GQC"
3. Admin gera link de convite (expira em 30 min, uso único)
4. Jogador acessa o link → preenche nome, WhatsApp e senha → entra no grupo
5. Admin cria partida (data, hora, local) → sistema gera hash único
6. Partida gera URL pública: http://localhost:3000/match/<hash>
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

## Comandos úteis

```bash
make up           # Sobe tudo com hot-reload
make down         # Para containers
make down-clean   # Para + apaga volumes (DB zerado)
make logs         # Logs do api + frontend
make shell        # Bash no container da API
make db-connect   # psql direto no banco
make adminer      # Abre Adminer (UI do banco)
make health       # Verifica saúde da API
make docs         # Abre Swagger no browser
```

---

## Para produção (Supabase)

1. Crie um projeto em [supabase.com](https://supabase.com)
2. No **SQL Editor** do Supabase, execute as migrations **em ordem**:
   - `migrations/001_initial_schema.sql` — schema base, enums e seed do admin
   - `migrations/002_seed_dev.sql` — dados de exemplo *(omitir em produção)*
   - `migrations/003_match_number.sql` — numeração sequencial de partidas
   - `migrations/004_match_address.sql` — campo de endereço para Google Maps
3. Copie a connection string (Settings → Database → Connection string)
4. Configure as variáveis de ambiente:
   ```
   DATABASE_URL=postgresql+asyncpg://postgres:<senha>@db.<projeto>.supabase.co:5432/postgres
   SECRET_KEY=<openssl rand -hex 32>
   ```
5. Deploy da API em Railway, Render, Fly.io, etc.
6. Build do frontend com `VITE_API_URL=https://sua-api.com/api/v1`

> **Novas migrations futuras:** sempre que um arquivo `migrations/00N_*.sql` for adicionado ao repositório, execute-o no SQL Editor do Supabase antes de fazer o deploy da API correspondente.
