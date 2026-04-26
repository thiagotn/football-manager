# PRD 040 — MCP Personal Access Tokens

## Contexto

O rachao.app expõe um servidor MCP (`mcp__rachao__*`) que permite que assistentes AI (Claude, etc.) acessem dados da plataforma em nome do usuário. Atualmente não existe controle de autenticação por usuário — qualquer token genérico dá acesso à API.

O objetivo é permitir que cada usuário gere seus próprios **tokens MCP pessoais** dentro da conta, com tempo de expiração configurável, visibilidade dos tokens ativos e possibilidade de revogação. O token gerado é exibido uma única vez (no momento da criação), seguindo o padrão adotado pelo Figma, GitHub e similares.

---

## Comportamento esperado (UX)

### Criação
1. Usuário acessa `/account/mcp-tokens`
2. Clica em "Gerar token"
3. Preenche um formulário com:
   - **Nome** (obrigatório, ex: "Claude Desktop", "VS Code") — identifica o uso
   - **Expiração** — radio/select com 3 opções:
     - 24 horas
     - 7 dias
     - Sem expiração
4. Confirma e o token completo é exibido **uma única vez** em um modal com botão de copiar
5. Aviso explícito: *"Guarde agora. Este token não será exibido novamente."*

### Listagem
- Tabela com: Nome · Prefixo mascarado (`rachao_a1b2c3d4…`) · Criado em · Expira em (ou "Nunca") · Ações
- Tokens expirados aparecem com badge "Expirado" e permanecem listados até revogação manual
- Sem limite de tokens por plano

### Revogação
- Botão "Revogar" por token, com `ConfirmDialog`
- Ao revogar, o token some da listagem

---

## Arquitetura

### Backend

#### Novo modelo — `MCPToken` (`football-api/app/models/mcp_token.py`)

| Coluna | Tipo | Notas |
|--------|------|-------|
| `id` | UUID PK | |
| `player_id` | UUID FK → players | |
| `name` | VARCHAR(100) | Label dado pelo usuário |
| `token_hash` | VARCHAR(64) | SHA-256 do token em hex — nunca armazena plaintext |
| `token_prefix` | VARCHAR(16) | Primeiros 12 chars para exibição mascarada |
| `expires_at` | TIMESTAMPTZ | NULL = sem expiração |
| `created_at` | TIMESTAMPTZ | default now() |
| `last_used_at` | TIMESTAMPTZ | NULL até primeiro uso |
| `revoked_at` | TIMESTAMPTZ | NULL = ativo |

#### Formato do token
```
rachao_<32 bytes hex aleatório>   →  rachao_a1b2c3d4e5f6...  (76 chars total)
```
- Gerado com `secrets.token_hex(32)`
- Hash armazenado: `hashlib.sha256(token.encode()).hexdigest()`
- Prefixo exibido: primeiros 12 chars (`rachao_a1b2c3`)

#### Migration — `040_mcp_tokens.sql`
```sql
CREATE TABLE IF NOT EXISTS mcp_tokens (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
  name VARCHAR(100) NOT NULL,
  token_hash VARCHAR(64) NOT NULL UNIQUE,
  token_prefix VARCHAR(16) NOT NULL,
  expires_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_used_at TIMESTAMPTZ,
  revoked_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_mcp_tokens_player_id ON mcp_tokens(player_id);
CREATE INDEX IF NOT EXISTS idx_mcp_tokens_token_hash ON mcp_tokens(token_hash);
```

#### Novo router — `football-api/app/api/v1/routers/mcp_tokens.py`

| Método | Rota | Descrição |
|--------|------|-----------|
| `POST` | `/api/v1/mcp-tokens` | Cria token — retorna plaintext **uma vez** |
| `GET` | `/api/v1/mcp-tokens` | Lista tokens do usuário (sem plaintext) |
| `DELETE` | `/api/v1/mcp-tokens/{token_id}` | Revoga token |

**POST body:**
```json
{ "name": "Claude Desktop", "expires_in": "24h" | "7d" | null }
```

**POST response** (único momento com o token em plaintext):
```json
{
  "id": "uuid",
  "name": "Claude Desktop",
  "token": "rachao_a1b2c3d4...",
  "token_prefix": "rachao_a1b2c3",
  "expires_at": "2026-04-27T00:00:00Z",
  "created_at": "2026-04-26T..."
}
```

**GET response** (listagem — sem plaintext):
```json
[{
  "id": "uuid",
  "name": "Claude Desktop",
  "token_prefix": "rachao_a1b2c3",
  "expires_at": "...",
  "created_at": "...",
  "last_used_at": "...",
  "is_expired": false
}]
```

**Regras de negócio:**
- Sem limite de tokens por plano — qualquer usuário autenticado pode criar livremente
- Tokens expirados (`expires_at < NOW()`) continuam listados, mas falham na autenticação
- Tokens revogados (`revoked_at IS NOT NULL`) não autenticam

#### Schemas — `football-api/app/schemas/mcp_token.py`
- `MCPTokenCreate`: `name` (str) + `expires_in` (`"24h" | "7d" | None`)
- `MCPTokenCreated`: estende `MCPTokenCreate` + campo `token` (plaintext, retornado uma vez)
- `MCPTokenResponse`: sem campo `token`, com `is_expired` (bool, campo computado)

#### Autenticação MCP (`football-api/app/mcp/`)
O servidor MCP precisa validar o bearer token recebido:
```python
token_hash = hashlib.sha256(bearer_token.encode()).hexdigest()
mcp_token = await db.scalar(
    select(MCPToken).where(
        MCPToken.token_hash == token_hash,
        MCPToken.revoked_at.is_(None),
        or_(MCPToken.expires_at.is_(None), MCPToken.expires_at > datetime.utcnow())
    )
)
if not mcp_token:
    raise UnauthorizedError()
await db.execute(
    update(MCPToken).where(MCPToken.id == mcp_token.id).values(last_used_at=datetime.utcnow())
)
```

> **Atenção:** A localização exata do servidor MCP não foi rastreada na exploração inicial — pode estar em serviço separado ou subdiretório não indexado. Localizar o ponto de autenticação antes de implementar a integração.

#### Testes unitários — `football-api/tests/unit/routers/test_mcp_tokens.py`
- Criar token com expiração 24h
- Criar token com expiração 7d
- Criar token sem expiração
- Listar tokens — plaintext não aparece
- Revogar token próprio
- Tentar revogar token de outro usuário → 403
- Tentar revogar token inexistente → 404

---

### Frontend

#### Nova rota — `football-frontend/src/routes/account/mcp-tokens/`
- `+page.svelte` — página principal
- `+page.ts` — `export const ssr = false`

**Layout:** seguir padrão de `/account/subscription` — `max-w-4xl mx-auto px-4 py-8`, envolvido em `<PageBackground>`

**Header:**
```svelte
<h1 class="text-2xl font-bold text-white flex items-center gap-2">
  <Key size={24} class="text-primary-400" /> Tokens MCP
</h1>
<p class="text-sm text-white/60 mt-0.5">{$t('mcp.subtitle')}</p>
```

**Seções da página:**
1. **Card "Gerar novo token"** — botão que abre `MCPTokenCreateModal`
2. **Card "Tokens"** — tabela responsiva:
   - Mobile: nome + prefixo + expiração em stack vertical, botão revogar
   - Desktop: colunas Nome | Token | Criado em | Expira em | Último uso | Ações
   - Estado vazio com mensagem explicativa
3. **`MCPTokenCreateModal.svelte`** — dois passos no mesmo modal:
   - Passo 1: form (nome + expiração) com botão "Gerar"
   - Passo 2: exibe token completo + botão copiar + aviso de uso único + botão "Entendi"

#### Novo componente — `football-frontend/src/lib/components/MCPTokenCreateModal.svelte`
Props: `bind:open`, `onCreated: (token: MCPTokenCreated) => void`

#### API namespace — `football-frontend/src/lib/api.ts`
Adicionar `mcpTokens`:
```typescript
mcpTokens: {
  list: () => request<MCPTokenResponse[]>('GET', '/mcp-tokens'),
  create: (body: MCPTokenCreate) => request<MCPTokenCreated>('POST', '/mcp-tokens', body),
  revoke: (id: string) => request<void>('DELETE', `/mcp-tokens/${id}`),
}
```

#### i18n — adicionar em `pt-BR.json`, `en.json`, `es.json`
Namespace `mcp.*`:
| Chave | pt-BR |
|-------|-------|
| `mcp.title` | Tokens MCP |
| `mcp.subtitle` | Acesso personalizado ao MCP do rachao.app |
| `mcp.generate` | Gerar token |
| `mcp.token_name` | Nome do token |
| `mcp.token_name_placeholder` | Ex: Claude Desktop |
| `mcp.expiration` | Expiração |
| `mcp.expires_24h` | 24 horas |
| `mcp.expires_7d` | 7 dias |
| `mcp.no_expiration` | Sem expiração |
| `mcp.copy_token` | Copiar token |
| `mcp.token_copied` | Token copiado! |
| `mcp.token_warning` | Guarde agora. Este token não será exibido novamente. |
| `mcp.token_understood` | Entendi |
| `mcp.revoke` | Revogar |
| `mcp.revoke_confirm` | Tem certeza? Este token será invalidado imediatamente. |
| `mcp.expires_at` | Expira em |
| `mcp.never` | Nunca |
| `mcp.last_used` | Último uso |
| `mcp.never_used` | Nunca usado |
| `mcp.expired_badge` | Expirado |
| `mcp.empty_state` | Nenhum token gerado ainda. |

---

## Arquivos a criar/modificar

| Arquivo | Ação |
|---------|------|
| `football-api/app/models/mcp_token.py` | Criar |
| `football-api/app/schemas/mcp_token.py` | Criar |
| `football-api/app/api/v1/routers/mcp_tokens.py` | Criar |
| `football-api/app/api/v1/router.py` | Registrar novo router |
| `football-api/migrations/040_mcp_tokens.sql` | Criar |
| `football-api/app/models/__init__.py` | Importar MCPToken |
| `football-api/tests/unit/routers/test_mcp_tokens.py` | Criar |
| `football-api/CLAUDE.md` | Atualizar próxima migration para 041 |
| `football-frontend/src/routes/account/mcp-tokens/+page.svelte` | Criar |
| `football-frontend/src/routes/account/mcp-tokens/+page.ts` | Criar |
| `football-frontend/src/lib/components/MCPTokenCreateModal.svelte` | Criar |
| `football-frontend/src/lib/api.ts` | Adicionar namespace `mcpTokens` |
| `football-frontend/messages/pt-BR.json` | Adicionar chaves `mcp.*` |
| `football-frontend/messages/en.json` | Adicionar chaves `mcp.*` |
| `football-frontend/messages/es.json` | Adicionar chaves `mcp.*` |
| `docs/prd/INDEX.md` | Adicionar entrada 040 |

---

## Verificação

1. Acessar `/account/mcp-tokens` logado
2. Clicar "Gerar token" → preencher nome "Teste", expiração "7 dias" → confirmar
3. Token completo exibido no modal → copiar → clicar "Entendi"
4. Token aparece na lista com prefixo mascarado e data de expiração correta
5. Autenticar no MCP com o token copiado → deve funcionar
6. Revogar o token → confirmar via `ConfirmDialog` → token some da lista
7. Tentar autenticar com token revogado → deve falhar com 401
8. Criar token "Sem expiração" → coluna "Expira em" mostra "Nunca"
9. Rodar testes unitários: `docker compose run --rm api poetry run pytest tests/unit/routers/test_mcp_tokens.py -v`
