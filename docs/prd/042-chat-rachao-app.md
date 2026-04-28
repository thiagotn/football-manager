# PRD — Assistente de IA do rachao.app

**Produto:** rachao.app/chat  
**Versão:** 1.0  
**Status:** Draft  
**Autor:** Thiago  
**Data:** 2026-04-28  

---

## 1. Visão Geral

### 1.1 Contexto

O rachao.app é uma plataforma SaaS para organização de peladas e rachões. À medida que a base de usuários cresce, cresce também a demanda por suporte contextualizado — dúvidas sobre funcionalidades, fluxos de uso, convites, pagamentos e organização de jogos.

Este PRD descreve o desenvolvimento de um **assistente de IA acessível em `rachao.app/chat`**, capaz de responder perguntas dos usuários com base exclusivamente no contexto do produto rachao.app, utilizando a Anthropic API com acesso ao MCP server proprietário da plataforma.

### 1.2 Objetivo

Reduzir a carga de suporte manual oferecendo um assistente inteligente, contextualizado e controlado pelo administrador da plataforma, disponível de forma opt-in por usuário.

### 1.3 Declaração de Valor

> "Qualquer usuário do rachao.app com acesso habilitado pelo administrador pode conversar com um assistente de IA que conhece o produto de ponta a ponta, sem sair da plataforma."

---

## 2. Escopo

### 2.1 Dentro do Escopo (v1.0)

- Interface de chat acessível em `rachao.app/chat` (rota dedicada no app SvelteKit existente — sem novo subdomínio)
- Assistente restrito ao contexto do rachao.app (sem respostas fora do produto)
- Painel administrativo para habilitar/desabilitar acesso por usuário
- Acesso desabilitado por padrão para todos os usuários
- Integração com MCP server do rachao.app (`mcp.rachao.app/mcp`)
- Autenticação via sessão existente da plataforma (mesmo login)
- Histórico de conversa por sessão (não persistido entre sessões na v1.0)
- Rate limiting para controle de custo de API

### 2.2 Fora do Escopo (v1.0)

- Histórico de conversa persistido entre sessões
- Suporte multi-idioma (somente português)
- Análise de métricas de uso do chat por usuário
- Integração com dados pessoais do usuário autenticado (ex: "meus jogos")
- Exportação de conversas
- Customização do assistente por organizador de pelada

---

## 3. Personas e Casos de Uso

### 3.1 Usuário Final (jogador / organizador)

**Dores:**
- Não sabe como convidar jogadores
- Tem dúvida sobre pagamento e cobrança via Stripe
- Não entende o fluxo de confirmação de presença
- Tem dificuldade em configurar uma pelada recorrente

**Casos de uso principais:**
- "Como funciona a confirmação de presença?"
- "Como faço para cobrar os jogadores pelo app?"
- "Por que meu convite não chegou?"
- "Como cancelo uma pelada agendada?"

### 3.2 Administrador Global da Plataforma (Thiago)

**Necessidades:**
- Controlar quem tem acesso ao assistente sem deploy
- Habilitar acesso para usuários beta/selecionados
- Desabilitar acesso individualmente se necessário
- Ter visibilidade de quais usuários estão com acesso ativo

> Role no sistema: `admin` (única role administrativa — verificada via `AdminPlayer` dependency no backend)

---

## 4. Requisitos Funcionais

### 4.1 Interface do Chat (`rachao.app/chat`)

| ID | Requisito | Prioridade |
|----|-----------|------------|
| F-01 | A interface deve ser acessível via `https://rachao.app/chat` | Must |
| F-02 | O usuário deve estar autenticado na plataforma para acessar o chat | Must |
| F-03 | Usuário sem acesso habilitado deve ver uma tela de "acesso indisponível" clara e amigável | Must |
| F-04 | O chat deve exibir as mensagens em tempo real via streaming (SSE) | Must |
| F-05 | O assistente deve responder apenas sobre o contexto do rachao.app | Must |
| F-06 | O histórico deve ser mantido durante a sessão ativa | Must |
| F-07 | O campo de input deve ser desabilitado enquanto o assistente está respondendo | Must |
| F-08 | Deve haver um botão "Nova conversa" para limpar o histórico da sessão | Should |
| F-09 | O layout deve ser responsivo (mobile-first) | Must |
| F-10 | Mensagens de erro de API devem ser exibidas de forma amigável | Must |

### 4.2 Controle Administrativo

| ID | Requisito | Prioridade |
|----|-----------|------------|
| A-01 | O admin global deve poder listar todos os usuários cadastrados com o status de acesso ao chat | Must |
| A-02 | O admin deve poder habilitar o acesso ao chat individualmente por usuário | Must |
| A-03 | O admin deve poder desabilitar o acesso ao chat individualmente por usuário | Must |
| A-04 | O acesso deve ser **desabilitado por padrão** para todo novo usuário cadastrado | Must |
| A-05 | A alteração de status deve ter efeito imediato (sem necessidade de relogin do usuário) | Must |
| A-06 | O painel admin deve ser acessível apenas para usuários com role `admin` (validado via `AdminPlayer` dependency no backend) | Must |
| A-07 | Deve haver busca/filtro de usuários no painel admin | Should |

### 4.3 Backend — Proxy da Anthropic API

| ID | Requisito | Prioridade |
|----|-----------|------------|
| B-01 | A API key da Anthropic deve estar exclusivamente no backend (nunca exposta ao frontend) | Must |
| B-02 | O endpoint `/api/chat` deve validar a autenticação do usuário antes de processar a requisição | Must |
| B-03 | O endpoint deve validar se o usuário tem acesso habilitado antes de chamar a Anthropic API | Must |
| B-04 | A resposta deve ser entregue via Server-Sent Events (streaming) | Must |
| B-05 | O MCP server `mcp.rachao.app/mcp` deve ser referenciado em todas as requisições | Must |
| B-06 | Deve haver rate limiting por usuário (ex: máximo de 20 mensagens/hora) via tabela PostgreSQL (Redis não está na stack) | Must |
| B-07 | O system prompt deve restringir o assistente ao contexto do rachao.app | Must |
| B-08 | Deve haver logging das requisições (sem conteúdo das mensagens) para monitoramento de custo | Should |

---

## 5. Requisitos Não Funcionais

| ID | Requisito | Meta |
|----|-----------|------|
| NF-01 | Latência para primeira palavra aparecer (TTFW) | < 2 segundos |
| NF-02 | Disponibilidade do serviço | ≥ 99,5% (alinhado ao SLA geral do VPS) |
| NF-03 | O serviço não deve impactar a performance do app principal | Chat em rota dedicada `/chat` — sem novo container ou processo |
| NF-04 | Custo de API deve ser controlado por rate limiting | Máximo R$ X/mês (definir por Thiago) |
| NF-05 | O deploy deve seguir o pipeline CI/CD existente (GitHub Actions) | Must |
| NF-06 | Secrets devem ser gerenciados via variáveis de ambiente no VPS | Must |

---

## 6. Arquitetura Técnica

### 6.1 Visão da Arquitetura

```
Usuário autenticado
        │
        ▼
https://rachao.app/chat   (SvelteKit — rota /chat no app existente)
        │                  (auth via localStorage — mesma origem)
        │  POST /api/v1/chat  (streaming SSE)
        ▼
https://api.rachao.app    (FastAPI — endpoint proxy)
        │
        ├──► Valida JWT / sessão do usuário
        ├──► Valida flag `chat_enabled` na tabela `players` (PostgreSQL)
        ├──► Aplica rate limiting (PostgreSQL)
        │
        ▼
Anthropic API  ◄──► MCP Server (mcp.rachao.app/mcp)
```

### 6.2 Stack

| Camada | Tecnologia |
|--------|------------|
| Frontend | SvelteKit 2 + Svelte 5 (rota `/chat` no app existente — `src/routes/chat/`) |
| Backend proxy | FastAPI (rota nova em `api.rachao.app`) |
| Banco de dados | Supabase PostgreSQL (campo `chat_enabled` em `players`) |
| IA | Anthropic API — `claude-haiku-4-5` (configurável via `LLM_MODEL`) |
| Contexto | MCP Server proprietário (`mcp.rachao.app/mcp`) |
| Reverse proxy | Traefik — **sem alteração** (rota `/chat` serve pelo mesmo router existente) |
| Deploy | GitHub Actions (pipeline existente) |

### 6.3 Mudanças no Banco de Dados

#### Migration: `041_chat_enabled.sql`

```sql
ALTER TABLE players
ADD COLUMN IF NOT EXISTS chat_enabled BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN players.chat_enabled IS
  'Controla se o usuário tem acesso ao assistente de IA em rachao.app/chat. Gerenciado pelo admin. Padrão: FALSE.';
```

> **Nota:** A tabela é `players` (não `profiles`). Migrations ficam em `football-api/migrations/` numeradas sequencialmente. Próximo número disponível: `041`.

#### View administrativa (opcional)

```sql
CREATE VIEW admin_chat_access AS
SELECT
  id,
  name,
  whatsapp,
  created_at,
  chat_enabled
FROM players
ORDER BY created_at DESC;
```

### 6.4 Endpoints de API

> **Convenção:** Todas as rotas usam o prefixo `/api/v1/` (padrão da plataforma). FastAPI usa `{param}` para path params.

#### `POST /api/v1/chat`

Recebe a mensagem do usuário e retorna a resposta em streaming.

**Request:**
```json
{
  "messages": [
    { "role": "user", "content": "Como faço para convidar jogadores?" }
  ]
}
```

**Response:** `text/event-stream` (SSE)

**Fluxo de validação:**
1. Extrai e valida JWT do header `Authorization`
2. Verifica `chat_enabled = true` em `players` para o usuário autenticado
3. Verifica rate limit do usuário
4. Chama Anthropic API com MCP server
5. Faz stream da resposta

**Erros:**
| Código | Situação |
|--------|----------|
| 401 | Usuário não autenticado |
| 403 | Acesso ao chat não habilitado para este usuário |
| 429 | Rate limit atingido |
| 500 | Erro interno ou falha na Anthropic API |

#### `GET /api/v1/admin/chat-users`

Lista usuários com status de acesso ao chat. **Apenas admin (`AdminPlayer` dependency).**

**Response:**
```json
{
  "users": [
    { "id": "uuid", "name": "João Silva", "whatsapp": "+55...", "chat_enabled": false }
  ]
}
```

#### `PATCH /api/v1/admin/chat-users/{user_id}`

Habilita ou desabilita acesso ao chat de um usuário específico. **Apenas admin (`AdminPlayer` dependency).**

**Request:**
```json
{ "chat_enabled": true }
```

### 6.5 System Prompt do Assistente

```
Você é o assistente oficial do rachao.app, uma plataforma para organização 
de peladas e rachões no Brasil.

Seu papel é ajudar usuários com dúvidas sobre funcionalidades, fluxos, 
pagamentos, convites, confirmações de presença e configurações do app.

Regras:
- Responda APENAS sobre o rachao.app e suas funcionalidades.
- Se perguntado sobre qualquer outro assunto, decline educadamente e 
  redirecione para tópicos do app.
- Seja direto, amigável e use linguagem informal brasileira.
- Use as ferramentas disponíveis para buscar informações reais do produto
  quando necessário.
- Nunca invente funcionalidades que não existem no app.
```

### 6.6 Infraestrutura

**Nenhuma alteração de infraestrutura necessária.** A rota `/chat` é servida pelo mesmo container SvelteKit e pelo mesmo router Traefik já configurado para `rachao.app`. Não é necessário novo subdomínio, novo certificado TLS, nova entrada no `traefik-dynamic.yml` nem novo serviço Docker.

---

## 7. Interface do Usuário

### 7.1 Tela de Chat (`rachao.app/chat`)

> **Padrão obrigatório:** Seguir o padrão de layout do app — envolver em `<PageBackground>`, usar `h1 text-2xl font-bold text-white flex items-center gap-2`, ícone Lucide `size={24} class="text-primary-400"`. Ver seção "Frontend — Padrões de Página" no CLAUDE.md.
>
> **i18n obrigatório:** Todo texto visível usa `$t('chat.*')`. Adicionar chaves em `pt-BR.json`, `en.json` e `es.json`.

**Layout:**
- Header com logo do rachao.app e título "Assistente rachao" (ícone sugerido: `<MessageCircle size={24} />`)
- Área de mensagens com scroll automático para o final
- Bolhas de mensagem diferenciadas (usuário à direita, assistente à esquerda)
- Input fixo no rodapé com botão de envio
- Indicador de "digitando..." durante o streaming
- Botão "Nova conversa" no header

**Tela de acesso não habilitado:**
- Mensagem clara: _"O assistente de IA ainda não está disponível para sua conta. Entre em contato com o suporte."_
- Sem formulário de chat visível

### 7.2 Painel Admin — Gestão de Acesso ao Chat

**Localização:** Seção existente de administração global da plataforma

**Componentes:**
- Lista de usuários com colunas: Nome (`name`), WhatsApp (`whatsapp`), Cadastro (`created_at`), Acesso ao Chat (`chat_enabled`)
- Toggle (switch) por linha para habilitar/desabilitar instantaneamente
- Campo de busca por nome ou WhatsApp
- Indicador de quantos usuários têm acesso ativo (ex: "3 de 47 usuários com acesso")

---

## 8. Segurança

| Ameaça | Mitigação |
|--------|-----------|
| Exposição da API key | Key exclusivamente no backend via env var |
| Acesso não autorizado | Validação de JWT em toda requisição ao `/api/v1/chat` |
| Uso indevido (spam/custo) | Rate limiting por usuário + flag `chat_enabled` |
| Prompt injection via MCP | MCP server proprietário e controlado |
| CORS | `rachao.app` já está em `allow_origins` no FastAPI — sem alteração necessária |
| Escalada de privilégio no admin | Verificação de `role = admin` via `AdminPlayer` dependency no servidor, nunca só no cliente |

---

## 9. Plano de Rollout

### Fase 1 — Backend e banco (Semana 1)

- [ ] Migration `041_chat_enabled.sql`: adicionar `chat_enabled` em `players`
- [ ] Implementar endpoint `POST /api/v1/chat` com streaming SSE e validações
- [ ] Implementar endpoints admin `GET` e `PATCH /api/v1/admin/chat-users`
- [ ] Configurar rate limiting via PostgreSQL (tabela de controle de janela por usuário)
- [ ] Testes de integração com Anthropic API + MCP server
- [ ] Configurar variáveis de ambiente no VPS (`ANTHROPIC_API_KEY`)

### Fase 2 — Frontend (Semana 1-2)

- [ ] Criar rota `src/routes/chat/+page.svelte` no app SvelteKit existente
- [ ] Componente `ChatInterface.svelte` com streaming SSE (EventSource nativo do browser)
- [ ] Tela de "acesso indisponível" (exibida quando `chat_enabled = false`)
- [ ] Adicionar link de acesso ao chat no menu/dashboard do app para usuários com acesso
- [ ] Painel admin: componente de listagem e toggle de usuários (em `src/routes/admin/chat/+page.svelte`)
- [ ] Adicionar chaves i18n `chat.*` nos 3 arquivos: `messages/pt-BR.json`, `messages/en.json`, `messages/es.json`
- [ ] Garantir que toda string visível usa `$t('chat.*')` — nunca string literal
- [ ] Layout deve seguir padrão obrigatório: `<PageBackground>`, `h1 text-2xl font-bold text-white`, ícone Lucide `size={24} class="text-primary-400"`
- [ ] Nenhuma alteração de Traefik ou CI/CD necessária

### Fase 3 — Beta interno (Semana 2)

- [ ] Habilitar acesso para Thiago (admin) via painel
- [ ] Testes end-to-end no ambiente de produção
- [ ] Validar custo de API com uso real
- [ ] Ajustar system prompt conforme necessário

### Fase 4 — Abertura gradual (Semana 3+)

- [ ] Habilitar acesso para usuários beta selecionados via painel admin
- [ ] Coletar feedback
- [ ] Decidir critérios para abertura geral (ou manter por convite)

---

## 10. Critérios de Aceite

| Critério | Como verificar |
|----------|----------------|
| Usuário sem `chat_enabled` vê tela de bloqueio | Acessar `rachao.app/chat` com usuário padrão |
| Admin habilita usuário e ele consegue acessar imediatamente | Testar toggle no painel e recarregar `rachao.app/chat` no mesmo momento |
| Assistente recusa perguntas fora do rachao.app | Enviar "qual a capital do Brasil?" e verificar redirecionamento |
| Rate limit bloqueia após N mensagens | Enviar mais de 20 mensagens em sequência e verificar 429 |
| API key não aparece em nenhum response do frontend | Inspecionar network no DevTools |
| Chat funciona corretamente no mobile | Testar `rachao.app/chat` em viewport 375px |

---

## 11. Métricas de Sucesso (pós-lançamento)

- Número de usuários com acesso habilitado
- Taxa de sessões iniciadas vs usuários com acesso (engajamento)
- Custo médio de API por usuário/mês
- Taxa de mensagens respondidas sem erro
- Satisfação informal via feedback dos usuários beta

---

## 12. Decisões em Aberto

| Decisão | Opções | Responsável |
|---------|--------|-------------|
| Definir limite de rate limit | 10, 20 ou 30 mensagens/hora por usuário | Thiago |
| Persistir histórico de conversa entre sessões na v1.0? | Sim (Supabase) / Não (somente sessão) | Thiago |
| Budget mensal máximo de API | Ver seção 13 para estimativas | Thiago |
| Critério para habilitar acesso em massa | Manual / automático por plano | Thiago |
| Escolha do modelo de IA | Ver seção 13 — trade-off entre custo e MCP nativo | Thiago |

---

## 13. Custos e Alternativas de Modelo

### 13.1 Premissas para Estimativa

Cenário de referência: **100 usuários ativos/mês × 50 mensagens cada = 5.000 mensagens/mês**

| Parâmetro | Valor estimado |
|-----------|---------------|
| Tokens de input por mensagem | ~1.500 (system prompt ~300 + histórico ~1.000 + pergunta ~200) |
| Tokens de output por mensagem | ~400 |
| Total de input/mês | ~7,5M tokens |
| Total de output/mês | ~2M tokens |

> Os tokens de input crescem conforme o histórico acumula na sessão. O system prompt fixo se beneficia de **prompt caching** (90% de desconto na Anthropic).

---

### 13.2 Comparativo de Custo Mensal (cenário 100 usuários × 50 msgs)

| Provedor | Modelo | Input/M | Output/M | Custo estimado/mês | Suporte MCP nativo |
|----------|--------|---------|----------|-------------------|--------------------|
| Anthropic | **claude-sonnet-4-6** | $3,00 | $15,00 | **~R$ 290** (~$53) | ✅ Nativo |
| Anthropic | **claude-haiku-4-5** | $1,00 | $5,00 | **~R$ 98** (~$18) | ✅ Nativo |
| Anthropic | claude-haiku-4-5 + cache | $0,10* | $5,00 | **~R$ 55** (~$10) | ✅ Nativo |
| Google | **Gemini 2.5 Flash** | $0,30 | $2,50 | **~R$ 22** (~$4) | ✅ Nativo |
| Google | Gemini 2.5 Flash (free tier) | grátis | grátis | **$0** | ✅ Nativo |
| Groq | Llama 3 70B Tool-Use | $0,59 | $0,79 | **~R$ 10** (~$2) | ⚠️ Parcial |
| Groq | Llama 3 8B Tool-Use | $0,05 | $0,08 | **~R$ 1** (~$0,15) | ⚠️ Parcial |
| Groq | Free tier | grátis | grátis | **$0** | ⚠️ Parcial |

*Tokens do system prompt cacheados (90% off); demais tokens pagos normalmente.  
*Câmbio de referência: US$1 ≈ R$5,80.*

---

### 13.3 Limites dos Planos Gratuitos

| Provedor | Limite free | Suficiente para? |
|----------|-------------|-----------------|
| **Gemini 2.5 Flash** | 10 RPM · 250 req/dia (~7.500/mês) | ✅ Suficiente para beta fechado (< 50 usuários × 50 msgs) |
| **Groq** | 30 RPM · 6.000 tokens/min | ✅ Suficiente para beta fechado, mas sem garantia de SLA |

---

### 13.4 Restrição Crítica: MCP Nativo

O PRD usa o MCP server `mcp.rachao.app/mcp` para fornecer contexto real do produto ao assistente. **Apenas Anthropic e Google Gemini têm suporte nativo ao protocolo MCP** no SDK/API.

| Provedor | Suporte MCP | Observação |
|----------|-------------|-----------|
| **Anthropic Claude** | ✅ Nativo via API | MCP connector direto na chamada, suporte a OAuth |
| **Google Gemini** | ✅ Nativo via SDK | SDKs Python/JS com auto-execução de ferramentas MCP |
| **Groq (Llama)** | ⚠️ Parcial | Suporta function calling, mas **não o protocolo MCP**. Exigiria camada de adaptação no backend para traduzir chamadas MCP em function calls — esforço adicional de implementação |

---

### 13.5 Recomendação por Fase

| Fase | Modelo recomendado | Justificativa |
|------|--------------------|---------------|
| **Beta fechado (< 50 usuários)** | Gemini 2.5 Flash (free tier) ou claude-haiku-4-5 | Custo zero ou mínimo; validar utilidade antes de investir |
| **Crescimento (50–500 usuários)** | claude-haiku-4-5 com prompt caching | MCP nativo + custo controlado (~R$ 50–200/mês) |
| **Escala (500+ usuários)** | Reavaliar: Gemini 2.5 Flash pago (~4× mais barato que Haiku) ou manter Haiku se MCP for crítico | Decisão por Thiago |

> **Sugestão para v1.0:** iniciar com `claude-haiku-4-5` (MCP nativo garantido, custo ~R$ 20–100/mês em beta) e configurar o modelo como variável de ambiente (`LLM_MODEL`) para trocar sem redeploy.

---

## Apêndice A — Variáveis de Ambiente Necessárias

```env
# Anthropic (obrigatório se usar Claude)
ANTHROPIC_API_KEY=sk-ant-...

# Modelo de IA (permite trocar sem redeploy)
# Opções: claude-sonnet-4-6 | claude-haiku-4-5 | gemini-2.5-flash (requer SDK diferente)
LLM_MODEL=claude-haiku-4-5

# Já existentes — reutilizadas
SUPABASE_URL=...
SUPABASE_SERVICE_KEY=...
JWT_SECRET=...
```

---

## Apêndice B — Referências

- Anthropic API Docs: https://docs.anthropic.com
- MCP Server rachao.app: https://mcp.rachao.app/mcp
- API rachao.app: https://api.rachao.app/docs
- Plataforma: https://rachao.app
- Chat: https://rachao.app/chat
