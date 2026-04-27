# PRD — Autenticação Bearer por Request no Servidor HTTP

**Status:** Proposto  
**Data:** 2026-04-27  
**Repositório:** `football-mcp`

---

## 1. Contexto

O `rachao-mcp` atualmente suporta dois modos de transporte:

- **stdio** — cada usuário executa sua própria instância do processo. O token JWT é passado via variável de ambiente `RACHAO_TOKEN`, lida em `auth.py` com `os.getenv()`.
- **HTTP/SSE** — uma instância única do servidor atende múltiplos clientes simultaneamente (ex.: `https://mcp.rachao.app`).

No modo HTTP, o mecanismo atual de leitura de token via `os.getenv()` é inviável para uso multi-tenant: todos os clientes compartilhariam o mesmo token do processo do servidor, tornando impossível a autenticação individual por usuário.

---

## 2. Problema

> **Como o servidor HTTP identifica qual usuário está fazendo cada chamada de tool, sem vazar o token de um request para outro?**

Requisitos que tornam o problema não-trivial:

- O servidor é assíncrono (asyncio + uvicorn). Múltiplos requests podem estar em execução concorrente.
- Variáveis globais ou de módulo não são seguras para armazenar estado por-request nesse modelo.
- O cliente MCP (ex.: Claude Code) precisa de um mecanismo padrão para enviar credenciais.

---

## 3. Solução Proposta

### 3.1 Visão Geral

Utilizar **`contextvars.ContextVar`** para armazenar o token JWT com escopo isolado por coroutine, combinado com um **middleware Starlette** que extrai o token do header `Authorization: Bearer <jwt>` e o injeta no contexto antes de qualquer execução de tool.

```
Cliente → [Authorization: Bearer <jwt>] → BearerTokenMiddleware → ContextVar._request_token → get_token() → RachaoClient
```

### 3.2 Por que `ContextVar` é a abordagem correta

Em asyncio, cada task/coroutine herda uma **cópia independente** do contexto do ponto onde foi criada. O middleware seta o `ContextVar` antes de chamar `call_next`, garantindo que toda a cadeia de execução daquele request (incluindo as tools do MCP) enxergue o token correto — sem race condition com outros requests simultâneos.

---

## 4. Escopo

### Incluído neste PRD

- Suporte a autenticação Bearer por-request no modo HTTP/SSE.
- Retrocompatibilidade total com o modo stdio (env var continua funcionando).
- Remoção do fail-fast de `get_token()` na inicialização do servidor HTTP.

### Excluído deste PRD

- Sistema de login / emissão de tokens JWT (responsabilidade da API `rachao.app`).
- Rate limiting ou throttling por token.
- Invalidação de sessão / revogação de tokens no MCP.
- Suporte a outros esquemas de autenticação (API Key header, OAuth 2.0 PKCE, etc.).

---

## 5. Requisitos Funcionais

| ID | Requisito |
|----|-----------|
| RF-01 | Em modo HTTP/SSE, o servidor **deve** aceitar o token JWT via header `Authorization: Bearer <token>` em cada request. |
| RF-02 | O token recebido no header **deve** ser usado exclusivamente para o request em questão, sem persistência entre requests. |
| RF-03 | Em modo stdio, o comportamento atual (leitura de `RACHAO_TOKEN` via env var) **deve** ser preservado sem alterações. |
| RF-04 | Se nenhum token for encontrado (nem no header, nem na env var), `get_token()` **deve** lançar `RuntimeError` com mensagem clara. |
| RF-05 | O fail-fast de `get_token()` na inicialização via `create_server()` **deve** ser desabilitado no modo HTTP (token não existe antes dos requests). |
| RF-06 | A env var `RACHAO_TOKEN` **deve** funcionar como fallback no modo HTTP para facilitar deploy single-tenant e testes locais. |

---

## 6. Requisitos Não-Funcionais

| ID | Requisito |
|----|-----------|
| RNF-01 | Zero vazamento de token entre requests concorrentes (isolamento por `ContextVar`). |
| RNF-02 | Nenhuma dependência nova além das já presentes em `pyproject.toml` (Starlette já é transitiva do `mcp[cli]`). |
| RNF-03 | A lógica de extração do token **deve** ser testável de forma unitária, independente do servidor MCP. |

---

## 7. Mudanças Técnicas

### 7.1 `rachao_mcp/auth.py`

Adicionar `ContextVar` e funções auxiliares:

```python
from contextvars import ContextVar
import os

_request_token: ContextVar[str | None] = ContextVar("_request_token", default=None)


def set_request_token(token: str) -> None:
    _request_token.set(token)


def get_token() -> str:
    token = _request_token.get() or os.getenv("RACHAO_TOKEN")
    if not token:
        raise RuntimeError(
            "RACHAO_TOKEN não definido — configure a variável de ambiente "
            "ou envie o header Authorization: Bearer <token>"
        )
    return token
```

### 7.2 `rachao_mcp/middleware.py` *(arquivo novo)*

```python
from starlette.middleware.base import BaseHTTPMiddleware
from starlette.requests import Request
from rachao_mcp.auth import set_request_token


class BearerTokenMiddleware(BaseHTTPMiddleware):
    async def dispatch(self, request: Request, call_next):
        auth = request.headers.get("Authorization", "")
        if auth.startswith("Bearer "):
            set_request_token(auth.removeprefix("Bearer ").strip())
        return await call_next(request)
```

### 7.3 `rachao_mcp/server.py`

- Aplicar `BearerTokenMiddleware` no app HTTP/SSE.
- Condicionar o fail-fast de `get_token()` ao modo stdio.

```python
def create_server() -> FastMCP:
    # Fail-fast apenas em stdio; em HTTP o token chega por request
    if os.getenv("MCP_TRANSPORT", "stdio") == "stdio":
        get_token()
    # ... restante sem alterações


def main() -> None:
    transport = os.getenv("MCP_TRANSPORT", "stdio")

    if transport in ("sse", "http"):
        import uvicorn
        from starlette.applications import Starlette
        from starlette.middleware import Middleware
        from rachao_mcp.middleware import BearerTokenMiddleware

        mcp = create_server()
        base_app = mcp.streamable_http_app() if transport == "http" else mcp.sse_app()

        app = Starlette(middleware=[Middleware(BearerTokenMiddleware)])
        app.mount("/", base_app)

        host = os.getenv("MCP_HOST", "127.0.0.1")
        port = int(os.getenv("MCP_PORT", "8080"))
        uvicorn.run(app, host=host, port=port)
    else:
        mcp = create_server()
        mcp.run()
```

---

## 8. Testes

### Novos testes a implementar

| Arquivo | Caso de teste |
|---------|---------------|
| `tests/test_auth.py` | `set_request_token` → `get_token()` retorna o valor setado |
| `tests/test_auth.py` | `ContextVar` isolado: tokens diferentes em coroutines concorrentes não se cruzam |
| `tests/test_auth.py` | Fallback para env var quando `ContextVar` está vazio |
| `tests/test_middleware.py` *(novo)* | Header `Authorization: Bearer xyz` → `get_token()` retorna `"xyz"` |
| `tests/test_middleware.py` | Request sem header → `ContextVar` permanece `None` |
| `tests/test_middleware.py` | Header malformado (sem `"Bearer "`) → `ContextVar` permanece `None` |
| `tests/test_server.py` *(novo)* | `create_server()` em modo HTTP não chama `get_token()` na inicialização |

### Testes existentes

Nenhum teste existente deve quebrar. O `conftest.py` já define `RACHAO_TOKEN` via `monkeypatch.setenv`, que continua funcionando como fallback.

---

## 9. Configuração do Cliente

Para conectar ao servidor HTTP, o cliente MCP deve enviar o header em cada request:

```json
// .mcp.json (Claude Code) ou equivalente
{
  "mcpServers": {
    "rachao": {
      "url": "https://mcp.rachao.app/sse",
      "headers": {
        "Authorization": "Bearer <seu-jwt>"
      }
    }
  }
}
```

O JWT deve ser obtido pela autenticação normal na plataforma `rachao.app`.

---

## 10. Critérios de Aceitação

- [ ] Dois requests HTTP simultâneos com tokens diferentes recebem respostas autenticadas com seus respectivos tokens, sem cruzamento.
- [ ] Um request sem header `Authorization` retorna erro claro (`RuntimeError`) ao tentar executar qualquer tool que precise de autenticação.
- [ ] `make dev` (modo stdio) continua funcionando sem nenhuma alteração no fluxo atual.
- [ ] `make test` passa 100% após as mudanças, incluindo os novos casos de teste.
- [ ] O `Dockerfile` não requer nenhuma alteração.
