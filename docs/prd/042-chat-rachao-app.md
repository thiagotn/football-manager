# PRD — Assistente de IA do rachao.app

**Produto:** chat.rachao.app  
**Versão:** 1.0  
**Status:** Draft  
**Autor:** Thiago  
**Data:** 2026-04-28  

---

## 1. Visão Geral

### 1.1 Contexto

O rachao.app é uma plataforma SaaS para organização de peladas e rachões. À medida que a base de usuários cresce, cresce também a demanda por suporte contextualizado — dúvidas sobre funcionalidades, fluxos de uso, convites, pagamentos e organização de jogos.

Este PRD descreve o desenvolvimento de um **assistente de IA acessível via `chat.rachao.app`**, capaz de responder perguntas dos usuários com base exclusivamente no contexto do produto rachao.app, utilizando a Anthropic API com acesso ao MCP server proprietário da plataforma.

### 1.2 Objetivo

Reduzir a carga de suporte manual oferecendo um assistente inteligente, contextualizado e controlado pelo administrador da plataforma, disponível de forma opt-in por usuário.

### 1.3 Declaração de Valor

> "Qualquer usuário do rachao.app com acesso habilitado pelo administrador pode conversar com um assistente de IA que conhece o produto de ponta a ponta, sem sair da plataforma."

---

## 2. Escopo

### 2.1 Dentro do Escopo (v1.0)

- Interface de chat acessível via subdomínio dedicado `chat.rachao.app`
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

---

## 4. Requisitos Funcionais

### 4.1 Interface do Chat (`chat.rachao.app`)

| ID | Requisito | Prioridade |
|----|-----------|------------|
| F-01 | A interface deve ser acessível via `https://chat.rachao.app` | Must |
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
| A-06 | O painel admin deve ser acessível apenas para usuários com role `admin_global` | Must |
| A-07 | Deve haver busca/filtro de usuários no painel admin | Should |

### 4.3 Backend — Proxy da Anthropic API

| ID | Requisito | Prioridade |
|----|-----------|------------|
| B-01 | A API key da Anthropic deve estar exclusivamente no backend (nunca exposta ao frontend) | Must |
| B-02 | O endpoint `/api/chat` deve validar a autenticação do usuário antes de processar a requisição | Must |
| B-03 | O endpoint deve validar se o usuário tem acesso habilitado antes de chamar a Anthropic API | Must |
| B-04 | A resposta deve ser entregue via Server-Sent Events (streaming) | Must |
| B-05 | O MCP server `mcp.rachao.app/mcp` deve ser referenciado em todas as requisições | Must |
| B-06 | Deve haver rate limiting por usuário (ex: máximo de 20 mensagens/hora) | Must |
| B-07 | O system prompt deve restringir o assistente ao contexto do rachao.app | Must |
| B-08 | Deve haver logging das requisições (sem conteúdo das mensagens) para monitoramento de custo | Should |

---

## 5. Requisitos Não Funcionais

| ID | Requisito | Meta |
|----|-----------|------|
| NF-01 | Latência para primeira palavra aparecer (TTFW) | < 2 segundos |
| NF-02 | Disponibilidade do serviço | ≥ 99,5% (alinhado ao SLA geral do VPS) |
| NF-03 | O serviço não deve impactar a performance do app principal | Isolamento via subdomínio |
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
https://chat.rachao.app   (SvelteKit — componente de chat)
        │
        │  POST /api/chat  (streaming SSE)
        ▼
https://api.rachao.app    (FastAPI — endpoint proxy)
        │
        ├──► Valida JWT / sessão do usuário
        ├──► Valida flag `chat_enabled` no Supabase
        ├──► Aplica rate limiting (Redis ou Supabase)
        │
        ▼
Anthropic API  ◄──► MCP Server (mcp.rachao.app/mcp)
```

### 6.2 Stack

| Camada | Tecnologia |
|--------|------------|
| Frontend | SvelteKit 5 (rota `chat.rachao.app`) |
| Backend proxy | FastAPI (rota nova em `api.rachao.app`) |
| Banco de dados | Supabase PostgreSQL (campo `chat_enabled` em `profiles`) |
| IA | Anthropic API — `claude-sonnet-4-20250514` |
| Contexto | MCP Server proprietário (`mcp.rachao.app/mcp`) |
| Reverse proxy | Traefik (novo router para `chat.rachao.app`) |
| Deploy | GitHub Actions (pipeline existente) |

### 6.3 Mudanças no Banco de Dados

#### Tabela `profiles` — novo campo

```sql
ALTER TABLE profiles
ADD COLUMN chat_enabled BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN profiles.chat_enabled IS
  'Controla se o usuário tem acesso ao assistente de IA em chat.rachao.app. Gerenciado pelo admin global. Padrão: FALSE.';
```

#### View administrativa (opcional)

```sql
CREATE VIEW admin_chat_access AS
SELECT
  id,
  full_name,
  phone,
  created_at,
  chat_enabled
FROM profiles
ORDER BY created_at DESC;
```

### 6.4 Endpoints de API

#### `POST /api/chat`

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
2. Verifica `chat_enabled = true` em `profiles` para o usuário autenticado
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

#### `GET /api/admin/chat-users`

Lista usuários com status de acesso ao chat. **Apenas admin global.**

**Response:**
```json
{
  "users": [
    { "id": "uuid", "full_name": "João Silva", "phone": "+55...", "chat_enabled": false }
  ]
}
```

#### `PATCH /api/admin/chat-users/:user_id`

Habilita ou desabilita acesso ao chat de um usuário específico. **Apenas admin global.**

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

### 6.6 Configuração do Traefik

Adicionar ao `docker-compose.yml` ou arquivo de rotas Traefik:

```yaml
# Router para chat.rachao.app
- "traefik.http.routers.chat-frontend.rule=Host(`chat.rachao.app`)"
- "traefik.http.routers.chat-frontend.tls=true"
- "traefik.http.routers.chat-frontend.tls.certresolver=letsencrypt"
- "traefik.http.routers.chat-frontend.service=frontend"
```

> O subdomínio `chat.rachao.app` aponta para o mesmo serviço SvelteKit existente, com rota dedicada. Não é necessário um novo container.

---

## 7. Interface do Usuário

### 7.1 Tela de Chat (`chat.rachao.app`)

**Layout:**
- Header simples com logo do rachao.app e nome "Assistente rachao"
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
- Lista de usuários com colunas: Nome, Telefone, Cadastro, Acesso ao Chat
- Toggle (switch) por linha para habilitar/desabilitar instantaneamente
- Campo de busca por nome ou telefone
- Indicador de quantos usuários têm acesso ativo (ex: "3 de 47 usuários com acesso")

---

## 8. Segurança

| Ameaça | Mitigação |
|--------|-----------|
| Exposição da API key | Key exclusivamente no backend via env var |
| Acesso não autorizado | Validação de JWT em toda requisição ao `/api/chat` |
| Uso indevido (spam/custo) | Rate limiting por usuário + flag `chat_enabled` |
| Prompt injection via MCP | MCP server proprietário e controlado |
| CORS | Configurar `allow_origins` no FastAPI apenas para `chat.rachao.app` |
| Escalada de privilégio no admin | Verificação de `role = admin_global` no servidor, nunca só no cliente |

---

## 9. Plano de Rollout

### Fase 1 — Backend e banco (Semana 1)

- [ ] Migration SQL: adicionar `chat_enabled` em `profiles`
- [ ] Implementar endpoint `POST /api/chat` com streaming e validações
- [ ] Implementar endpoints admin `GET` e `PATCH /api/admin/chat-users`
- [ ] Configurar rate limiting
- [ ] Testes de integração com Anthropic API + MCP server
- [ ] Configurar variáveis de ambiente no VPS (`ANTHROPIC_API_KEY`)

### Fase 2 — Frontend (Semana 1-2)

- [ ] Criar rota `/` em `chat.rachao.app` (SvelteKit)
- [ ] Componente `ChatInterface.svelte` com streaming SSE
- [ ] Tela de "acesso indisponível"
- [ ] Painel admin: componente de listagem e toggle de usuários
- [ ] Configurar roteamento Traefik para `chat.rachao.app`
- [ ] Atualizar CI/CD para incluir deploy da nova rota

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
| Usuário sem `chat_enabled` vê tela de bloqueio | Acessar `chat.rachao.app` com usuário padrão |
| Admin habilita usuário e ele consegue acessar imediatamente | Testar toggle no painel e recarregar chat no mesmo momento |
| Assistente recusa perguntas fora do rachao.app | Enviar "qual a capital do Brasil?" e verificar redirecionamento |
| Rate limit bloqueia após N mensagens | Enviar mais de 20 mensagens em sequência e verificar 429 |
| API key não aparece em nenhum response do frontend | Inspecionar network no DevTools |
| Chat funciona corretamente no mobile | Testar em viewport 375px |

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
| Budget mensal máximo de API | A definir | Thiago |
| Critério para habilitar acesso em massa | Manual / automático por plano | Thiago |

---

## Apêndice A — Variáveis de Ambiente Necessárias

```env
# Anthropic
ANTHROPIC_API_KEY=sk-ant-...

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
