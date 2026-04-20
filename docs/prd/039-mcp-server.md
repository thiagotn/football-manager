# PRD 039 — MCP Server do rachao.app

**Número**: 039  
**Versão**: 1.0  
**Status**: 📋 Proposto  
**Data**: Abril de 2026

---

## O que é MCP

Model Context Protocol (MCP) é um protocolo aberto criado pela Anthropic que permite a assistentes de IA (Claude, GitHub Copilot, Cursor, etc.) se conectarem a ferramentas e fontes de dados externas de forma padronizada. Um **servidor MCP** expõe *tools* (funções chamáveis pela IA) e/ou *resources* (dados legíveis). O cliente (ex: `claude` CLI ou VS Code Copilot) descobre as tools disponíveis e pode invocá-las automaticamente durante uma conversa.

No contexto do rachao.app: em vez de copiar/colar dados da API manualmente, a IA pode perguntar diretamente "quais são os próximos rachões do Grupo Amigos?" e o MCP busca isso automaticamente em `api.rachao.app`.

---

## Problema

Hoje, usar a IA para ajudar a gerenciar o rachao.app exige copiar dados do app e colá-los no contexto do chat manualmente. Não há como a IA consultar grupos, partidas ou jogadores em tempo real. Um servidor MCP mudaria isso: a IA teria acesso live à API do rachao.app como parte natural do fluxo de conversa.

---

## Solução

Criar um servidor MCP (`football-mcp/`) dentro do monorepo que exponha as operações principais da API do rachao.app como tools MCP. O servidor roda localmente via `stdio` (subprocesso da CLI), autentica-se com um JWT do rachao.app configurado como variável de ambiente, e chama `api.rachao.app` em nome do usuário.

---

## Stack técnica

| Componente | Escolha | Motivo |
|-----------|---------|--------|
| Linguagem | **Python 3.11+** | Mesmo stack do backend; reutiliza tipos e lógica |
| SDK MCP | `mcp[cli]` (PyPI, pacote oficial Anthropic) | SDK oficial, suporte stdio + SSE |
| HTTP client | `httpx` | Já usado no backend e nos testes |
| Empacotamento | `pyproject.toml` + `uv` | Consistente com o restante do projeto |
| Transport | **stdio** (padrão) | Mais simples para uso via CLI; zero infraestrutura extra |

---

## Tools a expor

### Grupos
| Tool | Endpoint | Descrição |
|------|----------|-----------|
| `list_groups` | `GET /groups` | Lista todos os grupos do usuário autenticado |
| `get_group` | `GET /groups/{id}` | Detalhes de um grupo: membros, stats, slots de times |
| `get_group_stats` | `GET /groups/{id}/stats` | Artilheiros, assistências, presença por jogador |

### Partidas
| Tool | Endpoint | Descrição |
|------|----------|-----------|
| `list_matches` | `GET /groups/{id}/matches` | Partidas de um grupo |
| `get_match` | `GET /matches/public/{hash}` | Detalhe de uma partida (lista de presença, stats) |
| `create_match` | `POST /groups/{id}/matches` | Criar nova partida |
| `update_match` | `PATCH /groups/{id}/matches/{mid}` | Atualizar partida (data, horário, local, status) |
| `set_attendance` | `POST /groups/{id}/matches/{mid}/attendance` | Confirmar/recusar presença de um jogador |
| `discover_matches` | `GET /matches/discover` | Partidas públicas abertas (sem autenticação obrigatória) |

### Jogadores
| Tool | Endpoint | Descrição |
|------|----------|-----------|
| `list_players` | `GET /players` (admin) ou `GET /groups/{id}/members` | Jogadores de um grupo |
| `get_my_stats` | `GET /players/me/stats/full` | Estatísticas do jogador autenticado |
| `get_ranking` | `GET /ranking` | Ranking geral da plataforma |

### Times
| Tool | Endpoint | Descrição |
|------|----------|-----------|
| `draw_teams` | `POST /groups/{id}/matches/{mid}/teams/draw` | Sortear times para uma partida |
| `get_teams` | `GET /groups/{id}/matches/{mid}/teams` | Times já sorteados |

> **Escopo v1**: somente leitura + ações pontuais (criar partida, set attendance, sortear times). Operações destrutivas (deletar grupo, remover jogador) fora de escopo intencional — exigem confirmação humana.

---

## Estrutura de arquivos

```
football-mcp/
├── pyproject.toml          ← dependências: mcp[cli], httpx
├── README.md
└── rachao_mcp/
    ├── __init__.py
    ├── server.py           ← ponto de entrada: mcp.run()
    ├── client.py           ← wrapper httpx para api.rachao.app
    ├── tools/
    │   ├── groups.py       ← tools de grupos
    │   ├── matches.py      ← tools de partidas
    │   ├── players.py      ← tools de jogadores
    │   └── teams.py        ← tools de times
    └── auth.py             ← lê RACHAO_TOKEN do env
```

### Esqueleto do `server.py`

```python
from mcp.server.fastmcp import FastMCP
from rachao_mcp.tools import groups, matches, players, teams

mcp = FastMCP("rachao.app")

# Registrar tools
mcp.include_tools(groups.tools)
mcp.include_tools(matches.tools)
mcp.include_tools(players.tools)
mcp.include_tools(teams.tools)

if __name__ == "__main__":
    mcp.run()  # stdio por padrão
```

### Esqueleto de uma tool (`matches.py`)

```python
from rachao_mcp.client import api
from mcp.server.fastmcp import Context

async def list_matches(group_id: str) -> list[dict]:
    """Lista as partidas de um grupo (abertas e encerradas)."""
    return await api.get(f"/groups/{group_id}/matches")

async def create_match(
    group_id: str,
    match_date: str,   # YYYY-MM-DD
    start_time: str,   # HH:MM
    location: str,
    notes: str | None = None,
) -> dict:
    """Cria uma nova partida no grupo."""
    return await api.post(f"/groups/{group_id}/matches", json={
        "match_date": match_date,
        "start_time": start_time,
        "location": location,
        "notes": notes,
    })

tools = [list_matches, create_match, ...]
```

---

## Autenticação

### Fluxo

```
Claude CLI / Copilot
    │  (stdio / SSE)
    ▼
rachao-mcp (local)
    │  Authorization: Bearer $RACHAO_TOKEN
    ▼
api.rachao.app  ← JWT validado normalmente pelo backend
```

O servidor MCP **não implementa autenticação própria** — ele apenas repassa o JWT do rachao.app que o usuário configura como variável de ambiente. Esse JWT é obtido fazendo login normal no app (ou via `POST /auth/login`).

### Como obter o token

```bash
# Opção A: via curl
curl -X POST https://api.rachao.app/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"whatsapp": "+55119...", "password": "sua-senha"}'
# → retorna { "access_token": "eyJ..." }

# Opção B: inspecionar localStorage do browser
# DevTools → Application → Local Storage → rachao.app → token
```

### Variáveis de ambiente necessárias

| Variável | Valor | Obrigatória |
|----------|-------|-------------|
| `RACHAO_TOKEN` | JWT do usuário autenticado | Sim |
| `RACHAO_API_URL` | `https://api.rachao.app/api/v1` | Não (default) |

---

## Configuração no Claude Code CLI

### Instalar localmente

```bash
cd football-mcp
pip install -e .   # ou: uv pip install -e .
```

### Registrar o MCP server

```bash
claude mcp add rachao \
  --command "python" \
  --args "-m rachao_mcp" \
  --env RACHAO_TOKEN="eyJ..."
```

Ou editar `~/.claude.json` diretamente:

```json
{
  "mcpServers": {
    "rachao": {
      "command": "python",
      "args": ["-m", "rachao_mcp"],
      "env": {
        "RACHAO_TOKEN": "eyJ...",
        "RACHAO_API_URL": "https://api.rachao.app/api/v1"
      }
    }
  }
}
```

### Uso

```
$ claude
> Quais são os meus próximos rachões esta semana?
[Claude invoca list_groups → list_matches automaticamente]
```

---

## Configuração no GitHub Copilot (VS Code)

No VS Code com a extensão GitHub Copilot (versão que suporta MCP), editar `.vscode/mcp.json` no workspace ou `settings.json` global:

```json
{
  "mcp": {
    "servers": {
      "rachao": {
        "type": "stdio",
        "command": "python",
        "args": ["-m", "rachao_mcp"],
        "env": {
          "RACHAO_TOKEN": "${env:RACHAO_TOKEN}"
        }
      }
    }
  }
}
```

O `${env:RACHAO_TOKEN}` lê do ambiente do shell, evitando hardcodar o token no arquivo de config.

---

## Deploy no VPS (modo remoto — opcional, v2)

Por padrão o MCP roda localmente via stdio. Para disponibilizar remotamente (ex: acessar de qualquer máquina sem instalar nada), é possível rodar no VPS com transporte **SSE (Server-Sent Events)** ou **Streamable HTTP**.

### Quando faz sentido

- Múltiplos usuários consumindo o mesmo servidor
- Uso de clientes que não suportam stdio (agentes web, integrações futuras)
- Centralizar updates: atualiza uma vez no VPS, todos os clientes ganham

### Como rodar com SSE

```python
# server.py — adicionar suporte SSE
mcp.run(transport="sse", host="0.0.0.0", port=8002)
```

```bash
# docker-compose.prod.yml — adicionar serviço
  mcp:
    image: ghcr.io/<owner>/football-manager-mcp:latest
    container_name: football-mcp
    restart: unless-stopped
    ports:
      - "127.0.0.1:8002:8002"   # só localhost; nginx faz o proxy
    environment:
      RACHAO_TOKEN: ${MCP_RACHAO_TOKEN}   # token de serviço (admin ou dedicado)
      RACHAO_API_URL: http://football-api:8000/api/v1   # rede interna Docker
```

### Autenticação do endpoint MCP remoto

O endpoint SSE precisa de proteção, senão qualquer um que souber a URL consegue usar o MCP com o token de serviço.

**Opção recomendada: token estático via header**

Nginx com `proxy_set_header` + validação no servidor MCP:

```nginx
location /mcp/ {
    # Rejeitar se não tiver o header correto
    if ($http_x_mcp_key != "chave-secreta-forte") {
        return 401;
    }
    proxy_pass http://127.0.0.1:8002/;
    proxy_http_version 1.1;
    proxy_set_header Connection "";
}
```

Config no Claude CLI para acessar remotamente:

```json
{
  "mcpServers": {
    "rachao-remote": {
      "url": "https://rachao.app/mcp/sse",
      "headers": {
        "X-Mcp-Key": "chave-secreta-forte"
      }
    }
  }
}
```

> **Nota sobre autenticação por usuário no modo remoto**: no modo remoto com token de serviço, o MCP age como um único usuário (o dono do token). Para múltiplos usuários, cada um precisaria de uma instância separada ou de um mecanismo de passagem de token por header — complexidade considerável. **Para uso pessoal, o modo stdio local é sempre preferível.**

---

## Comparativo: stdio local vs SSE remoto

| Critério | stdio (local) | SSE (VPS) |
|---------|--------------|-----------|
| **Complexidade** | Mínima | Média |
| **Segurança** | Alta (só local) | Requer proteção do endpoint |
| **Usuários** | 1 (você) | N (com mais trabalho) |
| **Latência** | Zero (local) | Rede |
| **Manutenção** | Instalar em cada máquina | Centralizado no VPS |
| **Recomendado para** | Uso pessoal / dev | Equipe / produção |

**Recomendação**: começar com stdio local. Adicionar SSE remoto apenas se houver necessidade de equipe.

---

## Arquivos a criar

| Arquivo | Ação |
|---------|------|
| `football-mcp/pyproject.toml` | Criar |
| `football-mcp/Makefile` | Criar |
| `football-mcp/Dockerfile` | Criar |
| `football-mcp/rachao_mcp/__init__.py` | Criar |
| `football-mcp/rachao_mcp/server.py` | Criar |
| `football-mcp/rachao_mcp/client.py` | Criar |
| `football-mcp/rachao_mcp/auth.py` | Criar |
| `football-mcp/rachao_mcp/tools/groups.py` | Criar |
| `football-mcp/rachao_mcp/tools/matches.py` | Criar |
| `football-mcp/rachao_mcp/tools/players.py` | Criar |
| `football-mcp/rachao_mcp/tools/teams.py` | Criar |
| `football-mcp/README.md` | Criar |
| `football-api/docker-compose.prod.yml` | Modificar (adicionar serviço `mcp`) |
| `football-api/traefik-dynamic.yml` | Modificar (adicionar router, service e middleware `mcp-sse`) |
| `.github/workflows/main.yml` | Modificar (adicionar job `build-mcp`) |

---

## Guardrails — limites do que o MCP pode fazer

Há quatro camadas independentes que se combinam para controlar o que a IA pode executar. Aplicá-las em conjunto garante que nenhum acidente silencioso aconteça.

---

### Camada 1 — Anotações de tool (spec MCP)

O protocolo MCP permite declarar metadados de comportamento em cada tool. O cliente (Claude, Copilot) usa essas anotações para decidir se pede confirmação ao usuário antes de executar.

```python
# tools/matches.py
from mcp.server.fastmcp import FastMCP
from mcp.types import Tool

@mcp.tool(
    annotations={
        "readOnlyHint": True,        # não modifica nada
        "idempotentHint": True,      # seguro chamar múltiplas vezes
    }
)
async def list_matches(group_id: str) -> list[dict]:
    """Lista as partidas de um grupo."""
    ...

@mcp.tool(
    annotations={
        "readOnlyHint": False,
        "destructiveHint": False,    # não é destrutivo, mas modifica estado
        "idempotentHint": False,
    }
)
async def create_match(...) -> dict:
    """Cria uma nova partida. REQUER confirmação do usuário antes de executar."""
    ...

@mcp.tool(
    annotations={
        "readOnlyHint": False,
        "destructiveHint": True,     # sinaliza: esta ação é irreversível
    }
)
async def cancel_match(group_id: str, match_id: str) -> dict:
    """Cancela uma partida (status → closed). Ação irreversível."""
    ...
```

| Anotação | Efeito no cliente |
|----------|-----------------|
| `readOnlyHint: true` | Claude executa sem pedir confirmação |
| `destructiveHint: true` | Claude **sempre** pede confirmação explícita antes |
| `idempotentHint: true` | Claude pode re-executar sem risco em caso de falha |

---

### Camada 2 — Modo read-only via variável de ambiente

Quando `RACHAO_MCP_READ_ONLY=true`, o servidor registra **apenas** as tools de leitura — as tools de escrita simplesmente não existem para o cliente.

```python
# server.py
import os

READ_ONLY = os.getenv("RACHAO_MCP_READ_ONLY", "false").lower() == "true"

mcp.include_tools(groups.read_tools)    # sempre
mcp.include_tools(matches.read_tools)   # sempre
mcp.include_tools(players.read_tools)   # sempre

if not READ_ONLY:
    mcp.include_tools(matches.write_tools)  # create_match, update_match, set_attendance
    mcp.include_tools(teams.write_tools)    # draw_teams
```

Config no `~/.claude.json` para um perfil somente-leitura:

```json
{
  "mcpServers": {
    "rachao-readonly": {
      "command": "python",
      "args": ["-m", "rachao_mcp"],
      "env": {
        "RACHAO_TOKEN": "eyJ...",
        "RACHAO_MCP_READ_ONLY": "true"
      }
    }
  }
}
```

---

### Camada 3 — Allowlist de tools e grupos

Controle granular via variáveis de ambiente: quais tools estão ativas e em quais grupos a IA pode operar.

```python
# server.py
import os

# RACHAO_MCP_ALLOWED_TOOLS=list_groups,list_matches,get_match
# Se não definido, permite todas as tools não-destrutivas
_allowed_raw = os.getenv("RACHAO_MCP_ALLOWED_TOOLS", "")
ALLOWED_TOOLS: set[str] | None = set(_allowed_raw.split(",")) if _allowed_raw else None

# RACHAO_MCP_GROUP_ALLOWLIST=uuid1,uuid2
# Se não definido, permite todos os grupos do token
_groups_raw = os.getenv("RACHAO_MCP_GROUP_ALLOWLIST", "")
GROUP_ALLOWLIST: set[str] | None = set(_groups_raw.split(",")) if _groups_raw else None
```

```python
# client.py — aplicar allowlist em toda chamada que recebe group_id
async def guarded_request(method: str, path: str, **kwargs):
    # Extrair group_id da path se houver
    if GROUP_ALLOWLIST and "/groups/" in path:
        gid = path.split("/groups/")[1].split("/")[0]
        if gid not in GROUP_ALLOWLIST:
            raise PermissionError(
                f"Grupo {gid} não está na allowlist do MCP. "
                f"Grupos permitidos: {GROUP_ALLOWLIST}"
            )
    return await _http_request(method, path, **kwargs)
```

Exemplo de config restrita — IA só pode listar e consultar, e apenas para um grupo específico:

```bash
RACHAO_MCP_ALLOWED_TOOLS=list_groups,list_matches,get_match,get_group_stats
RACHAO_MCP_GROUP_ALLOWLIST=3fa85f64-5717-4562-b3fc-2c963f66afa6
```

---

### Camada 4 — Confirmação explícita no código da tool (Human-in-the-loop)

Para operações de escrita críticas, a tool pode invocar o mecanismo de **sampling/elicitation** do MCP para pausar e pedir confirmação ao usuário antes de prosseguir. Isso é diferente das anotações: a confirmação acontece **dentro da lógica da tool**, não no cliente.

```python
# tools/matches.py
from mcp.server.fastmcp import FastMCP, Context

@mcp.tool()
async def create_match(
    ctx: Context,          # contexto MCP — dá acesso ao elicitation
    group_id: str,
    match_date: str,
    start_time: str,
    location: str,
    notes: str | None = None,
) -> dict:
    """Cria uma nova partida no grupo. Pede confirmação antes de salvar."""

    # Pausar e pedir confirmação humana antes de executar
    confirmation = await ctx.ask(
        f"Confirmar criação de partida?\n"
        f"  Grupo: {group_id}\n"
        f"  Data: {match_date} às {start_time}\n"
        f"  Local: {location}\n"
        f"\nDigite 'sim' para confirmar."
    )

    if confirmation.strip().lower() not in ("sim", "s", "yes", "y"):
        return {"status": "cancelled", "message": "Criação cancelada pelo usuário."}

    return await api.post(f"/groups/{group_id}/matches", json={
        "match_date": match_date,
        "start_time": start_time,
        "location": location,
        "notes": notes,
    })
```

> **Nota**: `ctx.ask()` usa o mecanismo de **MCP Elicitation** (suportado no Claude Code CLI). O fluxo para, exibe a pergunta para o usuário no terminal, e só continua após a resposta. Se o cliente não suportar elicitation, a tool deve tratar a ausência de `Context` com um comportamento seguro (negar por padrão).

---

### Resumo: qual guardrail usar para cada cenário

| Cenário | Guardrail recomendado |
|---------|----------------------|
| IA só deve ler dados, nunca escrever | `RACHAO_MCP_READ_ONLY=true` (camada 2) |
| IA pode escrever, mas deve confirmar ações importantes | Anotações `destructiveHint` + `ctx.ask()` (camadas 1 + 4) |
| Restringir a grupos específicos | `RACHAO_MCP_GROUP_ALLOWLIST=uuid1,uuid2` (camada 3) |
| Expor só um subconjunto de tools | `RACHAO_MCP_ALLOWED_TOOLS=tool1,tool2` (camada 3) |
| Ambiente de teste / staging | `RACHAO_API_URL=https://staging.api.rachao.app/api/v1` |

---

### Classificação das tools por nível de risco

| Tool | Risco | Guardrail obrigatório |
|------|-------|----------------------|
| `list_*`, `get_*`, `discover_*` | Nenhum | `readOnlyHint: true` |
| `set_attendance` | Baixo (reversível) | `idempotentHint: false` |
| `create_match` | Médio | `destructiveHint: false` + `ctx.ask()` |
| `update_match` | Médio | `destructiveHint: false` + `ctx.ask()` |
| `draw_teams` | Médio (recria sorteio) | `ctx.ask()` |
| `cancel_match` (v2) | Alto (irreversível) | `destructiveHint: true` + `ctx.ask()` |
| Deletar grupo/membro | Fora de escopo v1 | Não expor |

---

## Fora de escopo (v1)

- MCP Resources (ex: expor partida como recurso navegável)
- Prompts MCP (templates pré-definidos)
- OAuth2 / autenticação por usuário no modo remoto
- Operações destrutivas (deletar grupo, remover membro)
- Integração com Stripe/financeiro via MCP
- Notificações push via MCP

---

## Desenvolvimento local

### Pré-requisitos

- Python 3.11+
- `uv` instalado (`pip install uv` ou via `brew install uv`)
- Token JWT válido do rachao.app (ver seção Autenticação)

### Makefile — `football-mcp/Makefile`

```makefile
.PHONY: install dev register unregister list check test

RACHAO_API_URL ?= https://api.rachao.app/api/v1

# ── Instalação ────────────────────────────────────────────────

install:
	uv pip install -e .

# ── Execução local (stdio) ────────────────────────────────────

dev:
	@if [ -z "$(RACHAO_TOKEN)" ]; then \
		echo "Erro: defina RACHAO_TOKEN=<jwt>"; exit 1; \
	fi
	RACHAO_TOKEN=$(RACHAO_TOKEN) \
	RACHAO_API_URL=$(RACHAO_API_URL) \
	python -m rachao_mcp

# ── Integração com Claude CLI ─────────────────────────────────

register:
	@if [ -z "$(RACHAO_TOKEN)" ]; then \
		echo "Uso: make register RACHAO_TOKEN=<jwt>"; exit 1; \
	fi
	claude mcp add rachao \
		--command "python" \
		--args "-m rachao_mcp" \
		--env RACHAO_TOKEN="$(RACHAO_TOKEN)" \
		--env RACHAO_API_URL="$(RACHAO_API_URL)"
	@echo ""
	@echo "MCP registrado. Teste com: make list"

unregister:
	claude mcp remove rachao
	@echo "MCP removido."

list:
	claude mcp list

# ── Verificação rápida sem Claude CLI ────────────────────────

check:
	@if [ -z "$(RACHAO_TOKEN)" ]; then \
		echo "Uso: make check RACHAO_TOKEN=<jwt>"; exit 1; \
	fi
	RACHAO_TOKEN=$(RACHAO_TOKEN) \
	RACHAO_API_URL=$(RACHAO_API_URL) \
	python -c "import asyncio; from rachao_mcp.client import api; print(asyncio.run(api.get('/groups')))"

# ── Testes ────────────────────────────────────────────────────

test:
	pytest tests/ -v
```

### Fluxo de teste local (passo a passo)

```bash
# 1. Entrar no diretório e instalar dependências
cd football-mcp
make install

# 2. Registrar o servidor no Claude CLI (só precisa fazer uma vez)
make register RACHAO_TOKEN=eyJ...

# 3. Verificar que aparece na lista
make list
# → rachao  stdio  python -m rachao_mcp

# 4. Abrir o Claude CLI e testar as tools
claude
> Quais são meus grupos no rachao.app?
> Liste as partidas do grupo X

# 5. Para remover quando não precisar mais
make unregister
```

### Variáveis de ambiente para desenvolvimento

Crie um `.env` em `football-mcp/` (não commitar — já deve estar no `.gitignore`):

```bash
RACHAO_TOKEN=eyJ...
RACHAO_API_URL=https://api.rachao.app/api/v1   # ou staging se houver
RACHAO_MCP_READ_ONLY=true                       # seguro para explorar sem risco de escrita
```

Para carregar no shell: `export $(cat .env | xargs)` antes de rodar `make dev` ou `make check`.

---

## Testes unitários

### Stack

| Lib | Papel |
|-----|-------|
| `pytest` + `pytest-asyncio` | Runner e suporte a funções `async` |
| `respx` | Mock de respostas HTTP do `httpx` — simula `api.rachao.app` sem rede real |
| `pytest-mock` | `mocker` fixture para monkeypatching de env vars e módulos |

> **Por que `respx` e não `ASGITransport`?** O MCP é um pacote independente que chama uma API remota via `httpx` — não há app FastAPI local para montar. `respx` intercepta as chamadas HTTP no nível do transport, sem precisar de rede real nem de um servidor rodando.

### Estrutura de arquivos

```
football-mcp/
└── tests/
    ├── conftest.py          ← fixtures: mock_api, env_token
    ├── test_auth.py         ← leitura do token do env
    ├── test_client.py       ← headers, erros HTTP, timeout
    ├── test_guardrails.py   ← read-only mode, group allowlist, tool allowlist
    └── tools/
        ├── test_groups.py
        ├── test_matches.py
        ├── test_players.py
        └── test_teams.py
```

### `tests/conftest.py`

```python
import os
import pytest
import respx
import httpx

@pytest.fixture(autouse=True)
def set_token(monkeypatch):
    monkeypatch.setenv("RACHAO_TOKEN", "test-jwt-token")
    monkeypatch.setenv("RACHAO_API_URL", "https://api.rachao.app/api/v1")

@pytest.fixture
def mock_api():
    """Intercepta todas as chamadas HTTP para api.rachao.app."""
    with respx.mock(base_url="https://api.rachao.app/api/v1") as mock:
        yield mock
```

### Casos obrigatórios por módulo

#### `test_auth.py`
| Caso | Comportamento esperado |
|------|----------------------|
| `RACHAO_TOKEN` definido | `get_token()` retorna o valor |
| `RACHAO_TOKEN` ausente | `RuntimeError` com mensagem "RACHAO_TOKEN não definido" |
| `RACHAO_API_URL` ausente | usa default `https://api.rachao.app/api/v1` |

```python
def test_missing_token_raises(monkeypatch):
    monkeypatch.delenv("RACHAO_TOKEN", raising=False)
    with pytest.raises(RuntimeError, match="RACHAO_TOKEN"):
        from rachao_mcp.auth import get_token
        get_token()
```

---

#### `test_client.py`
| Caso | Comportamento esperado |
|------|----------------------|
| Toda request inclui `Authorization: Bearer <token>` | Header presente em GET e POST |
| API retorna 401 | `PermissionError("Autenticação inválida — verifique RACHAO_TOKEN")` |
| API retorna 404 | `LookupError("Recurso não encontrado")` |
| API retorna 503 / timeout | `RuntimeError("API indisponível")` — não trava o CLI |
| GET bem-sucedido | retorna dict/list parseado do JSON |

```python
@pytest.mark.asyncio
async def test_bearer_header_sent(mock_api):
    mock_api.get("/groups").mock(return_value=httpx.Response(200, json=[]))
    from rachao_mcp.client import api
    await api.get("/groups")
    assert mock_api.calls[0].request.headers["authorization"] == "Bearer test-jwt-token"

@pytest.mark.asyncio
async def test_401_raises_permission_error(mock_api):
    mock_api.get("/groups").mock(return_value=httpx.Response(401))
    from rachao_mcp.client import api
    with pytest.raises(PermissionError, match="RACHAO_TOKEN"):
        await api.get("/groups")

@pytest.mark.asyncio
async def test_503_raises_runtime_error(mock_api):
    mock_api.get("/groups").mock(return_value=httpx.Response(503))
    from rachao_mcp.client import api
    with pytest.raises(RuntimeError, match="indisponível"):
        await api.get("/groups")
```

---

#### `test_guardrails.py`
| Caso | Comportamento esperado |
|------|----------------------|
| `RACHAO_MCP_READ_ONLY=true` | `write_tools` não registrados no servidor |
| `RACHAO_MCP_READ_ONLY=false` (default) | todos os tools registrados |
| `GROUP_ALLOWLIST` definido, group_id na lista | request passa |
| `GROUP_ALLOWLIST` definido, group_id fora da lista | `PermissionError` antes de chamar a API |
| `ALLOWED_TOOLS` definido | tools fora da lista não são registrados |

```python
def test_read_only_excludes_write_tools(monkeypatch):
    monkeypatch.setenv("RACHAO_MCP_READ_ONLY", "true")
    from rachao_mcp import server
    tool_names = {t.name for t in server.mcp.tools}
    assert "create_match" not in tool_names
    assert "draw_teams" not in tool_names
    assert "list_matches" in tool_names

@pytest.mark.asyncio
async def test_group_allowlist_blocks_unauthorized(monkeypatch, mock_api):
    allowed = "aaaaaaaa-0000-0000-0000-000000000000"
    blocked = "bbbbbbbb-0000-0000-0000-000000000000"
    monkeypatch.setenv("RACHAO_MCP_GROUP_ALLOWLIST", allowed)
    from rachao_mcp.client import api
    with pytest.raises(PermissionError, match="allowlist"):
        await api.get(f"/groups/{blocked}/matches")

@pytest.mark.asyncio
async def test_group_allowlist_allows_authorized(monkeypatch, mock_api):
    allowed = "aaaaaaaa-0000-0000-0000-000000000000"
    monkeypatch.setenv("RACHAO_MCP_GROUP_ALLOWLIST", allowed)
    mock_api.get(f"/groups/{allowed}/matches").mock(return_value=httpx.Response(200, json=[]))
    from rachao_mcp.client import api
    result = await api.get(f"/groups/{allowed}/matches")
    assert result == []
```

---

#### `tests/tools/test_groups.py`
| Caso | Comportamento esperado |
|------|----------------------|
| `list_groups` → GET `/groups` | retorna lista de grupos |
| `get_group` → GET `/groups/{id}` | retorna detalhes do grupo |
| `get_group_stats` → GET `/groups/{id}/stats` | retorna stats do grupo |

```python
@pytest.mark.asyncio
async def test_list_groups(mock_api):
    mock_api.get("/groups").mock(return_value=httpx.Response(200, json=[{"id": "abc", "name": "Pelada"}]))
    from rachao_mcp.tools.groups import list_groups
    result = await list_groups()
    assert result[0]["name"] == "Pelada"
```

---

#### `tests/tools/test_matches.py`
| Caso | Comportamento esperado |
|------|----------------------|
| `list_matches(group_id)` → GET `/groups/{id}/matches` | lista correta |
| `create_match(...)` → POST `/groups/{id}/matches` com body correto | retorna partida criada |
| `update_match(...)` → PATCH com apenas campos alterados | body correto |
| `set_attendance(group_id, match_id, player_id, status)` → POST correto | retorna confirmação |
| `discover_matches()` → GET `/matches/discover` sem auth | retorna lista pública |

```python
@pytest.mark.asyncio
async def test_create_match_posts_correct_body(mock_api):
    gid = "group-uuid"
    mock_api.post(f"/groups/{gid}/matches").mock(
        return_value=httpx.Response(201, json={"id": "match-uuid"})
    )
    from rachao_mcp.tools.matches import create_match
    result = await create_match(gid, "2026-05-10", "20:00", "Campo do Zé")
    sent = mock_api.calls[0].request
    import json
    body = json.loads(sent.content)
    assert body["match_date"] == "2026-05-10"
    assert body["start_time"] == "20:00"
    assert body["location"] == "Campo do Zé"
```

---

#### `tests/tools/test_teams.py`
| Caso | Comportamento esperado |
|------|----------------------|
| `draw_teams(group_id, match_id)` → POST correto | retorna times sorteados |
| `get_teams(group_id, match_id)` → GET correto | retorna times existentes |

---

### Rodar os testes

```bash
# Instalar dependências de dev (inclui respx, pytest-asyncio, pytest-mock)
cd football-mcp
make install

# Rodar todos os testes
make test

# Com cobertura
uv run pytest tests/ --cov=rachao_mcp --cov-report=term-missing -q
```

### Adicionar ao `pyproject.toml`

```toml
[project.optional-dependencies]
dev = [
    "pytest>=8.0",
    "pytest-asyncio>=0.24",
    "pytest-mock>=3.14",
    "respx>=0.21",
]

[tool.pytest.ini_options]
asyncio_mode = "auto"
```

---

## Deploy — GitHub Actions

O MCP segue o mesmo pipeline CI/CD da API e do frontend. Um novo job `build-mcp` é adicionado ao workflow existente em `.github/workflows/main.yml`.

### Detecção de mudanças

```yaml
# .github/workflows/main.yml — job detect-changes
- name: Detect changed paths
  uses: dorny/paths-filter@v4
  id: changes
  with:
    filters: |
      api: ['football-api/**']
      frontend: ['football-frontend/**']
      mcp: ['football-mcp/**']        # ← novo
      e2e: ['football-e2e/**']
```

### Job build-mcp

```yaml
build-mcp:
  name: Build & Deploy MCP
  runs-on: ubuntu-latest
  needs: [unit-tests]                 # não faz deploy se os testes falharem
  if: |
    needs.detect-changes.outputs.mcp == 'true' ||
    github.event_name == 'workflow_dispatch'

  steps:
    - uses: actions/checkout@v4

    - name: Log in to GHCR
      uses: docker/login-action@v3
      with:
        registry: ghcr.io
        username: ${{ github.actor }}
        password: ${{ secrets.GITHUB_TOKEN }}

    - name: Set up Docker Buildx
      uses: docker/setup-buildx-action@v3

    - name: Build and push MCP image
      uses: docker/build-push-action@v6
      with:
        context: ./football-mcp
        push: true
        tags: ghcr.io/${{ github.repository_owner }}/football-manager-mcp:latest
        cache-from: type=gha
        cache-to: type=gha,mode=max

    - name: Deploy MCP to VPS
      uses: appleboy/ssh-action@v1
      with:
        host: ${{ secrets.VPS_HOST }}
        username: ${{ secrets.VPS_USER }}
        key: ${{ secrets.VPS_SSH_KEY }}
        script: |
          cd /opt/football-manager
          docker compose -f docker-compose.prod.yml pull mcp
          docker compose -f docker-compose.prod.yml up -d --no-deps mcp
          docker image prune -f
```

### Secrets necessários (já existem no repositório)

| Secret | Uso |
|--------|-----|
| `VPS_HOST` | IP ou hostname do VPS |
| `VPS_USER` | Usuário SSH (`ubuntu` ou similar) |
| `VPS_SSH_KEY` | Chave SSH privada para acesso ao VPS |
| `GITHUB_TOKEN` | Automático — push para GHCR |

### `football-mcp/Dockerfile`

```dockerfile
FROM python:3.11-slim

WORKDIR /app
COPY pyproject.toml ./
RUN pip install --no-cache-dir uv && uv pip install --system -e .

COPY rachao_mcp/ ./rachao_mcp/

ENV PYTHONUNBUFFERED=1
EXPOSE 8080

CMD ["python", "-m", "rachao_mcp"]
```

---

## Infraestrutura VPS — mcp.rachao.app

O projeto já usa **Traefik v3** como reverse proxy com Let's Encrypt integrado (`certificatesresolvers.letsencrypt`). Não é necessário nginx nem Certbot separado — basta adicionar o serviço no Compose e um router no arquivo dinâmico.

### 1. Serviço no `docker-compose.prod.yml`

Adicionar ao final da seção `services` (antes de `volumes:`):

```yaml
  # ── MCP Server ────────────────────────────────────────────────
  mcp:
    image: ghcr.io/thiagotn/football-manager-mcp:latest
    container_name: football-mcp
    restart: unless-stopped
    environment:
      RACHAO_TOKEN: ${MCP_RACHAO_TOKEN}
      RACHAO_API_URL: https://api.rachao.app/api/v1
      MCP_TRANSPORT: sse
      MCP_HOST: 0.0.0.0
      MCP_PORT: 8080
      MCP_SECRET_KEY: ${MCP_SECRET_KEY}
      RACHAO_MCP_READ_ONLY: "false"
    expose:
      - "8080"
    networks:
      - app-net
```

O container **não** expõe a porta 8080 diretamente ao host — o Traefik roteia via `app-net`.

### 2. Router e service no `traefik-dynamic.yml`

Adicionar nas seções correspondentes do arquivo existente em `football-api/traefik-dynamic.yml`:

```yaml
# Em http.routers — adicionar:
    mcp:
      rule: "Host(`mcp.rachao.app`)"
      entrypoints: [websecure]
      tls:
        certResolver: letsencrypt
      middlewares: [mcp-sse]
      service: mcp

# Em http.services — adicionar:
    mcp:
      loadBalancer:
        servers:
          - url: "http://mcp:8080"
        responseForwarding:
          flushInterval: "100ms"   # flush imediato dos eventos SSE

# Em http.middlewares — adicionar:
    mcp-sse:
      headers:
        customResponseHeaders:
          X-Accel-Buffering: "no"  # desativa buffer de resposta para SSE
```

O Traefik emite e renova o certificado TLS para `mcp.rachao.app` automaticamente via ACME HTTP-01 (mesma configuração dos outros subdomínios).

### 3. Variáveis de ambiente no VPS (`.env.prod`)

```bash
MCP_RACHAO_TOKEN=<jwt-do-admin-ou-service-account>
MCP_SECRET_KEY=<string-longa-aleatória-usada-como-chave-de-acesso>
```

### 4. DNS

Adicionar registro A apontando `mcp.rachao.app` para o mesmo IP do VPS — não é necessário servidor separado.

### Arquivos alterados no deploy

| Arquivo | Tipo de mudança |
|---------|----------------|
| `football-api/docker-compose.prod.yml` | Adicionar serviço `mcp` |
| `football-api/traefik-dynamic.yml` | Adicionar router, service e middleware `mcp-sse` |
| `football-api/.env.prod` (VPS) | Adicionar `MCP_RACHAO_TOKEN` e `MCP_SECRET_KEY` |

### Config do cliente remoto (VS Code Copilot / Claude CLI)

```json
// .vscode/mcp.json (SSE remoto)
{
  "servers": {
    "rachao-remote": {
      "type": "sse",
      "url": "https://mcp.rachao.app/sse",
      "headers": {
        "X-Mcp-Key": "<MCP_SECRET_KEY>"
      }
    }
  }
}
```

```bash
# Claude CLI — servidor remoto
claude mcp add rachao-remote \
  --transport sse \
  --url https://mcp.rachao.app/sse \
  --header "X-Mcp-Key=<MCP_SECRET_KEY>"
```

---

## Impacto e uso de recursos

O MCP server é um processo **stateless e idle-first**: não mantém conexão com o banco de dados, não processa filas, e só consome recursos de rede quando uma tool é invocada.

### Estimativa por modo

| Recurso | stdio (local) | SSE (VPS) — idle | SSE (VPS) — tool ativa |
|---------|--------------|-------------------|------------------------|
| RAM | ~40 MB (processo Python) | ~50 MB | ~55 MB |
| CPU | ~0% idle | ~0% idle | < 2% por chamada |
| Rede | 0 (sem rede própria) | 0 | 1–3 req HTTP à API por tool |
| Disco | ~15 MB (imagem slim) | ~15 MB | — |
| Latência por tool | < 100ms (local) | 100–300ms (rede) | depende da API |

### Comparativo com serviços existentes no VPS

| Serviço | RAM atual |
|---------|-----------|
| `football-api` | ~180 MB |
| `football-frontend` (Node SSR) | ~120 MB |
| Grafana | ~200 MB |
| Prometheus | ~80 MB |
| **`mcp` (novo)** | **~50 MB** |

O MCP é o menor serviço da stack. Em um VPS com 2–4 GB de RAM, o impacto é desprezível.

### Tráfego gerado na API

Cada invocação de tool faz **1 chamada HTTP** à `api.rachao.app`. Em uso pessoal típico (< 50 invocações/dia), o overhead sobre a API é irrelevante — equivale a um usuário ativo abrindo o app algumas vezes por dia.

### Conexões SSE

Cada sessão do Claude CLI ou Copilot mantém **uma conexão SSE aberta** enquanto o contexto estiver ativo. A conexão é leve (keep-alive sem dados) e encerrada automaticamente ao fechar o terminal ou a IDE.

### Conclusão

Nenhuma mudança de sizing é necessária no VPS atual. O MCP pode ser adicionado como mais um container no `docker-compose.prod.yml` sem impacto mensurável nos outros serviços.

---

## Verificação

1. `python -m rachao_mcp` inicia sem erro com `RACHAO_TOKEN` válido
2. `claude mcp list` exibe `rachao` como servidor registrado
3. `claude mcp test rachao` retorna tools disponíveis
4. No Claude CLI: perguntar "meus grupos" → Claude invoca `list_groups` e retorna dados reais
5. No Claude CLI: pedir "cria um rachão amanhã às 20h no Campo do Zé" → Claude invoca `list_groups` (para escolher grupo) e `create_match`
6. Token inválido → MCP retorna erro legível ("Autenticação inválida — verifique RACHAO_TOKEN")
7. API offline → MCP retorna erro legível sem travar o CLI
