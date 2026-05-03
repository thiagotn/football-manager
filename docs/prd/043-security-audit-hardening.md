# PRD 043 — Auditoria e Hardening de Segurança do rachao.app

**Status:** Em análise  
**Data:** 2026-05-03  
**Autor:** Thiago

---

## TODOs

- [ ] **Ingestão de logs no Grafana Loki** — os logs de auditoria (S-05) já saem em JSON estruturado em produção. Revisar como configurar o Promtail (ou Alloy) para coletar o stdout do container da API e enviar ao Loki já existente em `grafana.rachao.app`. Criar dashboard/alerta para eventos `WARNING` (`auth_login_failed`, `auth_login_rate_limited`, `admin_player_deleted`, `group_member_removed`).

---

## Visão Geral

Este documento registra os resultados de uma auditoria de segurança do repositório rachao.app, motivada por uma revisão de práticas comuns de segurança em APIs REST. O objetivo é identificar gargalos, classificá-los por severidade e propor um plano de remediação priorizado.

A auditoria cobriu: autenticação/autorização, validação de input, gestão de secrets, configuração de CORS e headers HTTP, padrões de acesso ao banco de dados, logs de auditoria, segurança da cadeia de suprimentos (dependências) e Row Level Security no Supabase.

---

## O que está bem implementado

Os controles abaixo estão corretamente implementados e **não requerem ação**:

| Controle | Evidência |
|----------|-----------|
| **IDOR (Insecure Direct Object Reference)** | Todas as rotas com `group_id`/`match_id` verificam membership do player autenticado antes de retornar dados (`group_repo`, `match_repo`). |
| **Validação de input** | Pydantic em todos os endpoints; raw dicts nunca chegam ao banco. |
| **Secrets** | Nenhum secret hardcoded; tudo via variáveis de ambiente. CI/CD usa `${{ secrets.* }}` em todos os jobs. |
| **CORS origins** | Whitelist explícita (`rachao.app`), sem `*`. |
| **Container non-root** | Imagem de produção roda como `appuser`, não root. |
| **Rate limiting do chat** | 20 msgs/hora por usuário implementado via PostgreSQL (sem Redis). |
| **Hashing de senhas e tokens MCP** | Senhas com bcrypt; tokens MCP com hash SHA-256 — nunca armazenados em plaintext. |

---

## Achados por Severidade

### Alto

| ID | Achado | Arquivo(s) | Severidade |
|----|--------|-----------|------------|
| S-01 | ~~Login endpoint sem rate limiting~~ — **resolvido em 2026-05-03** | `routers/auth.py` | ✅ |
| S-02 | ~~Sem headers de segurança HTTP~~ — **resolvido em 2026-05-03** | `main.py` | ✅ |
| S-08 | ~~RLS não habilitado no Supabase~~ — **resolvido em 2026-05-03** | Supabase Dashboard | ✅ |

### Médio-Alto

| ID | Achado | Arquivo(s) | Severidade |
|----|--------|-----------|------------|
| S-03 | JWT com expiração de 24h e sem refresh token — token vazado dá acesso por 24h | `config.py:23` | Médio-Alto |

### Médio

| ID | Achado | Arquivo(s) | Severidade |
|----|--------|-----------|------------|
| S-04 | ~~SQL construído via f-string~~ — **resolvido em 2026-05-03** | `routers/admin.py` | ✅ |
| S-05 | ~~Sem log de auditoria para ações sensíveis~~ — **resolvido em 2026-05-03** | `routers/auth.py`, `players.py`, `groups.py` | ✅ |
| S-06 | ~~Sem dependency vulnerability scanning no CI/CD~~ — **resolvido em 2026-05-03** | `.github/workflows/main.yml` | ✅ |

### Baixo-Médio

| ID | Achado | Arquivo(s) | Severidade |
|----|--------|-----------|------------|
| S-07 | ~~`allow_methods=["*"]` e `allow_headers=["*"]` no CORS~~ — **resolvido em 2026-05-03** | `main.py` | ✅ |

---

## Plano de Remediação

| Prioridade | ID | Ação | Esforço estimado |
|------------|-----|------|-----------------|
| ~~P1~~ | ~~S-08~~ | ~~Habilitar RLS em todas as tabelas no Supabase~~ — **✅ feito em 2026-05-03** | 2h |
| ~~P1~~ | ~~S-07~~ | ~~CORS: restringir methods e headers~~ — **✅ feito em 2026-05-03** | 10 min |
| ~~P1~~ | ~~S-02~~ | ~~Adicionar security headers HTTP ao middleware~~ — **✅ feito em 2026-05-03** | 4h |
| ~~P1~~ | ~~S-06~~ | ~~Adicionar `pip-audit` e `npm audit` ao CI~~ — **✅ feito em 2026-05-03** | 1h |
| ~~P2~~ | ~~S-01~~ | ~~Rate limiting no login (5 req/min por IP)~~ — **✅ feito em 2026-05-03** | 1 dia |
| ~~P2~~ | ~~S-05~~ | ~~Audit logging estruturado para ações sensíveis~~ — **✅ feito em 2026-05-03** | 4h |
| ~~P3~~ | ~~S-04~~ | ~~Refatorar SQL f-string para SQLAlchemy Core~~ — **✅ feito em 2026-05-03** | 1 dia |
| P4 | S-03 | Reduzir JWT para 8h (quick win) | 2h |
| Backlog | S-03 | Implementar refresh token rotativo | 3 dias |

---

## Detalhamento Técnico por Achado

### S-01 — Rate Limiting no Login

**Evidência:** `routers/auth.py:34` — o endpoint `POST /api/v1/auth/login` não possui nenhum mecanismo de throttle. Um atacante pode tentar senhas indefinidamente sem bloqueio.

**Impacto:** Brute force de senhas ou credential stuffing sem qualquer barreira. Combinado com o fato de que o login aceita número de WhatsApp (facilmente enumerável via listas vazadas), o risco é real.

**Remediação — opção A (nova dependência):** Usar `slowapi` (wrapper de `limits` para FastAPI):

```python
# pyproject.toml: adicionar slowapi
from slowapi import Limiter
from slowapi.util import get_remote_address
from slowapi.errors import RateLimitExceeded

limiter = Limiter(key_func=get_remote_address)
app.state.limiter = limiter
app.add_exception_handler(RateLimitExceeded, _rate_limit_handler)

@router.post("/auth/login")
@limiter.limit("5/minute")
async def login(request: Request, body: LoginRequest, db: DB):
    ...
```

**Remediação — opção B (sem nova dependência):** Usar o mesmo padrão do chat rate limit — contador de tentativas por IP no PostgreSQL, com janela de 1 minuto. Elimina a dependência nova mas exige uma coluna/tabela extra.

**Verificação:** Fazer 6 requisições seguidas ao `/auth/login` com o mesmo IP. A 6ª deve retornar `429 Too Many Requests`.

---

### S-02 — Headers de Segurança HTTP

**Evidência:** `main.py` não possui nenhum middleware que injete headers de segurança nas respostas. Uma inspeção com `curl -I https://api.rachao.app` ou ferramenta como securityheaders.com revelaria a ausência completa.

**Impacto:**
- Sem `X-Frame-Options`: app pode ser embutido em iframe (clickjacking)
- Sem `X-Content-Type-Options: nosniff`: MIME-type sniffing em browsers antigos
- Sem `Strict-Transport-Security`: downgrade de HTTPS para HTTP possível em primeira visita
- Sem `Content-Security-Policy`: XSS mais fácil de explorar

**Remediação — opção A (pacote `secure`):**

```python
# pyproject.toml: adicionar secure
from secure import Secure

secure_headers = Secure.with_default_headers()

@app.middleware("http")
async def set_secure_headers(request: Request, call_next):
    response = await call_next(request)
    secure_headers.framework.fastapi(response)
    return response
```

**Remediação — opção B (middleware manual, sem nova dependência):**

```python
@app.middleware("http")
async def set_secure_headers(request: Request, call_next):
    response = await call_next(request)
    response.headers["X-Frame-Options"] = "DENY"
    response.headers["X-Content-Type-Options"] = "nosniff"
    response.headers["X-XSS-Protection"] = "1; mode=block"
    response.headers["Strict-Transport-Security"] = "max-age=31536000; includeSubDomains"
    response.headers["Referrer-Policy"] = "strict-origin-when-cross-origin"
    return response
```

**Verificação:** `curl -I https://api.rachao.app/api/v1/health` — todos os headers acima devem aparecer na resposta.

---

### S-03 — JWT com Expiração de 24h sem Refresh Token

**Evidência:** `config.py:23` — `access_token_expire_minutes: int = 60 * 24`.

**Impacto:** Um token vazado (via log, clipboard, rede) dá ao atacante 24h de acesso pleno à conta da vítima, sem possibilidade de revogação antes do vencimento (o sistema não tem blacklist de tokens).

**Contexto e trade-off:** Para um app mobile B2C com autenticação via WhatsApp OTP, 24h é um trade-off UX aceitável a curto prazo. Reduzir para 15min sem refresh token geraria logout frequente em mobile, o que é pior em termos de adoção. A solução completa (refresh token rotativo) tem custo de implementação de 2-3 dias.

**Remediação — fase 1 (quick win):** Reduzir para 8h. Impacto de UX baixo para uso diário; reduz janela de ataque em 66%.

```python
# config.py
access_token_expire_minutes: int = 60 * 8
```

**Remediação — fase 2 (completa):** Implementar refresh token rotativo:
- Tabela `refresh_tokens` (migration 044): `id`, `player_id`, `token_hash`, `expires_at`, `revoked_at`, `created_at`
- Endpoint `POST /auth/refresh` que valida o refresh token, revoga o antigo e emite par novo
- Access token com 15min de expiração; refresh token com 30 dias
- Frontend armazena o par e chama `/auth/refresh` automaticamente quando recebe `401`

**Verificação:** Após emitir token, aguardar expiração e verificar que requests com o token expirado retornam `401`.

---

### S-04 — SQL Construído via f-string em admin.py

**Evidência:** `routers/admin.py` nas linhas 93, 238, 355 e 361, há padrões do tipo:

```python
conditions = []
if filter_x:
    conditions.append("column = :param")
where = " AND ".join(conditions)
query = text(f"SELECT ... FROM ... WHERE {where}")
```

**Risco real atual:** Baixo — as `conditions` são strings literais construídas no código, não interpolando input do usuário diretamente. O risco de SQL injection imediato é mínimo.

**Risco latente:** Alto. O padrão f-string em queries convida a interpolações futuras de variáveis. Uma mudança de 10 linhas por um dev que não conheça o contexto pode abrir SQL injection real. O código já está "a um passo" do problema.

**Remediação:** Refatorar para SQLAlchemy Core com `and_()`, `or_()`, `select()`:

```python
# Antes
conditions.append("g.status = :status")
params["status"] = filter_status
query = text(f"SELECT ... WHERE {where}")

# Depois
from sqlalchemy import select, and_
stmt = select(Group).where(
    and_(Group.status == filter_status, ...)
)
result = await db.execute(stmt)
```

**Verificação:** Após refatoração, passar um valor com `; DROP TABLE players; --` em qualquer filtro de admin e verificar que é tratado como string literal.

---

### S-05 — Ausência de Audit Logging para Ações Sensíveis

**Evidência:** Login, falha de login, troca de senha, deleção de player, mudança de role e remoção de membro de grupo não geram nenhum registro estruturado além dos logs genéricos de request.

**Impacto:** Impossibilidade de detectar ataques em andamento, investigar incidentes post-hoc, ou demonstrar conformidade com LGPD (que exige rastreabilidade de tratamento de dados pessoais).

**Remediação — nível 1 (sem schema change):** Adicionar `logger.info()` estruturado nos eventos sensíveis, com campos consistentes:

```python
logger.info(
    "auth_login_success player_id=%s ip=%s",
    str(player.id), request.client.host
)
logger.warning(
    "auth_login_failed whatsapp=%s ip=%s",
    body.whatsapp, request.client.host
)
```

Campos mínimos por evento: `event`, `player_id` (quando aplicável), `ip`, `target_id` (quando afeta outro recurso).

**Remediação — nível 2 (tabela dedicada):** Criar tabela `audit_logs` via migration 043:

```sql
CREATE TABLE IF NOT EXISTS audit_logs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    player_id   UUID REFERENCES players(id) ON DELETE SET NULL,
    action      VARCHAR(64) NOT NULL,
    resource_type VARCHAR(32),
    resource_id UUID,
    ip          VARCHAR(45),
    metadata    JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ON audit_logs(player_id);
CREATE INDEX ON audit_logs(action);
CREATE INDEX ON audit_logs(created_at DESC);
```

**Verificação:** Realizar login com credenciais erradas e verificar que o evento aparece nos logs (nível 1) ou na tabela (nível 2).

---

### S-06 — Sem Dependency Vulnerability Scanning no CI

**Evidência:** `.github/workflows/main.yml` não contém nenhum step de `pip-audit` ou `npm audit`. Dependências vulneráveis podem entrar em produção sem alerta.

**Impacto:** Zero visibilidade sobre CVEs em dependências Python (`fastapi`, `sqlalchemy`, `cryptography`, etc.) e npm (`svelte`, `vite`, etc.).

**Remediação:** Adicionar ao job `unit-tests` (Python):

```yaml
- name: Audit Python dependencies
  run: poetry run pip-audit
  working-directory: football-api
```

Adicionar ao job `e2e` ou a um job dedicado (npm):

```yaml
- name: Audit npm dependencies
  run: npm audit --audit-level=high
  working-directory: football-frontend
```

**Verificação:** Introduzir temporariamente uma dependência Python com CVE conhecido e verificar que o CI falha no step de auditoria.

---

### S-07 — CORS com Methods e Headers Abertos

**Evidência:** `main.py:90-91`:

```python
allow_methods=["*"],
allow_headers=["*"],
```

**Impacto:** Qualquer método HTTP e qualquer header customizado são aceitos em requisições cross-origin. Na prática, habilita métodos como `PUT`, `DELETE`, `PATCH`, `OPTIONS` e `TRACE` mesmo que a origem seja legítima mas o método não devesse ser exposto.

**Remediação:**

```python
# main.py
app.add_middleware(
    CORSMiddleware,
    allow_origins=settings.cors_origins,
    allow_credentials=True,
    allow_methods=["GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"],
    allow_headers=["Content-Type", "Authorization"],
)
```

**Verificação:** Enviar uma requisição OPTIONS com `Access-Control-Request-Method: TRACE` e verificar que o browser recebe apenas os métodos permitidos no header `Access-Control-Allow-Methods`.

---

### S-08 — RLS não habilitado no Supabase (PostgREST exposto)

**Contexto:** O projeto usa o Supabase apenas como provedor de PostgreSQL (conexão direta via SQLAlchemy/asyncpg) e para armazenamento de avatares (Storage). O app **não usa PostgREST** intencionalmente. No entanto, o Supabase **habilita o PostgREST por padrão em todo projeto** e não há como desativá-lo no plano gratuito/Pro. Isso significa que o endpoint `https://<projeto>.supabase.co/rest/v1/` está ativo e acessível publicamente.

**Evidência:**
- Nenhuma migration contém `ENABLE ROW LEVEL SECURITY`, `CREATE POLICY` ou `ALTER TABLE ... ENABLE RLS`
- RLS está desabilitado em todas as tabelas: `players`, `groups`, `group_members`, `matches`, `match_attendance`, `match_votes`, `finance_entries`, `subscriptions`, `mcp_tokens`, etc.
- A `anon key` do Supabase é considerada "pública" no modelo de segurança da plataforma — mas sem RLS, ela dá leitura (e potencialmente escrita) irrestrita a todas as tabelas via PostgREST

**Impacto:** Um atacante que obtenha a `anon key` (presente em dashboards, docs, ambientes de staging vazados, ou via engenharia social) pode:
- `GET /rest/v1/players` — listar todos os jogadores com dados pessoais (nome, WhatsApp)
- `GET /rest/v1/subscriptions` — listar planos e status de pagamento de todos os usuários
- `GET /rest/v1/mcp_tokens` — listar tokens MCP (hashes — sem acesso ao valor original, mas confirma existência)
- `POST /rest/v1/players` — inserir dados diretamente no banco, bypassando validações do FastAPI
- Isso viola a LGPD (exposição de dados pessoais sem controle de acesso).

**Remediação:** Habilitar RLS em todas as tabelas **sem criar nenhuma policy**. Sem policies, o comportamento padrão do PostgreSQL com RLS ativo é **negar todo acesso** ao `anon` role — o que é exatamente o que queremos. O backend continua funcionando normalmente porque usa a `service_role`, que **bypassa RLS por design**.

**✅ Executado em 2026-05-03** via SQL Editor do Supabase Dashboard:

```sql
ALTER TABLE avatar_upload_logs     ENABLE ROW LEVEL SECURITY;
ALTER TABLE match_waitlist         ENABLE ROW LEVEL SECURITY;
ALTER TABLE push_subscriptions     ENABLE ROW LEVEL SECURITY;
ALTER TABLE matches                ENABLE ROW LEVEL SECURITY;
ALTER TABLE groups                 ENABLE ROW LEVEL SECURITY;
ALTER TABLE match_teams            ENABLE ROW LEVEL SECURITY;
ALTER TABLE group_members          ENABLE ROW LEVEL SECURITY;
ALTER TABLE players                ENABLE ROW LEVEL SECURITY;
ALTER TABLE match_team_players     ENABLE ROW LEVEL SECURITY;
ALTER TABLE match_vote_flop        ENABLE ROW LEVEL SECURITY;
ALTER TABLE match_vote_top5        ENABLE ROW LEVEL SECURITY;
ALTER TABLE webhook_events         ENABLE ROW LEVEL SECURITY;
ALTER TABLE player_subscriptions   ENABLE ROW LEVEL SECURITY;
ALTER TABLE android_beta_signups   ENABLE ROW LEVEL SECURITY;
ALTER TABLE attendances            ENABLE ROW LEVEL SECURITY;
ALTER TABLE invite_tokens          ENABLE ROW LEVEL SECURITY;
ALTER TABLE match_votes            ENABLE ROW LEVEL SECURITY;
ALTER TABLE match_player_stats     ENABLE ROW LEVEL SECURITY;
ALTER TABLE app_reviews            ENABLE ROW LEVEL SECURITY;
ALTER TABLE schema_migrations      ENABLE ROW LEVEL SECURITY;
ALTER TABLE finance_periods        ENABLE ROW LEVEL SECURITY;
ALTER TABLE finance_payments       ENABLE ROW LEVEL SECURITY;
ALTER TABLE mcp_tokens             ENABLE ROW LEVEL SECURITY;
```

Sem policies criadas — comportamento padrão: **deny-all para `anon`**. O backend continua funcionando porque usa `service_role`, que bypassa RLS por design.

**Por que não adicionar ao código de migration:** Este é um controle de infraestrutura do Supabase. O backend aplica migrations com `service_role`, que bypassa RLS — não faria diferença aplicar via migration. O controle correto é no nível do banco via Dashboard ou Supabase CLI.

**Verificação:**
1. `GET https://<projeto>.supabase.co/rest/v1/players?select=*` com `apikey: <anon_key>` → deve retornar `[]`
2. Supabase Dashboard → Table Editor → todas as tabelas devem exibir ícone de cadeado (RLS enabled)
3. Confirmar que o app continua funcionando normalmente (nenhuma regressão)

---

## Verificação Geral Pós-Remediação

Após implementar todos os itens P1 e P2:

1. Rodar `docker compose run --rm api poetry run pytest tests/unit/ -q` — todos os testes devem passar
2. Testar `curl -I https://api.rachao.app` — verificar headers de segurança presentes
3. Verificar [securityheaders.com](https://securityheaders.com) com a URL da API — nota esperada: A ou A+
4. Rodar `poetry run pip-audit` no diretório `football-api/` — zero vulnerabilidades conhecidas
5. Fazer 6 tentativas de login em sequência — 6ª deve retornar `429`
6. Acessar `https://<projeto>.supabase.co/rest/v1/players?select=*` com `anon key` — deve retornar `[]`
