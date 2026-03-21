# PRD — Grupos Públicos e Lista de Espera

**Status:** Proposta
**Data:** 2026-03-19
**Atualizado:** 2026-03-19
**Contexto:** Hoje todos os grupos são fechados — só participam membros convidados pelo admin. Esta feature adiciona visibilidade pública opcional e um mecanismo de entrada por lista de espera, permitindo crescimento orgânico dos grupos.

---

## 1. Problema

- Grupos novos não têm como atrair jogadores sem que o admin convide manualmente cada um
- Jogadores avulsos interessados em um rachão não têm forma de se candidatar
- O admin precisa saber de antemão quem convidar — não há fluxo orgânico de adesão
- Grupos que queiram abrir vagas para desconhecidos não têm ferramenta para isso
- Jogadores sem cadastro que recebem o link de uma partida com vaga disponível não têm caminho claro para participar — a oportunidade de conversão se perde

---

## 2. Objetivo

Permitir que grupos sejam configurados como **públicos**, tornando seu próximo rachão visível para qualquer jogador da plataforma via link direto. Jogadores externos podem se candidatar a uma vaga via **lista de espera**, concordar com os termos do rachão, e ser aceitos pelos admins — entrando automaticamente no grupo com presença confirmada no rachão.

---

## 3. Escopo

### Incluso
- Flag `is_public` no grupo (criação e edição)
- Página pública do grupo (acessível via link, sem necessidade de ser membro)
- Fluxo de entrada na lista de espera com aceite de termos
- Campo de apresentação ("Conte um pouco sobre você") na candidatura
- Fluxo de cadastro integrado para jogadores sem conta que chegam via link de partida
- Redirecionamento automático para a lista de espera pós-cadastro
- Painel de admins para aceite/rejeição de candidatos (com visualização da apresentação)
- Adição automática ao grupo e confirmação de presença ao aceitar candidato
- Notificações push para admins (novo candidato) e candidato (aceito/rejeitado)

### Fora do escopo
- Página de descoberta/busca de grupos públicos (ver seção 9)
- Reputação ou avaliação de jogadores externos
- Limite configurável de candidatos na lista de espera

---

## 4. Regras de Negócio

### 4.1 Visibilidade

| Tipo | Acesso à página pública | Confirmação de presença |
|---|---|---|
| **Público** (`is_public = true`) | Qualquer jogador logado, via link | Membros do grupo + aprovados via lista de espera |
| **Fechado** (`is_public = false`) | Apenas membros do grupo | Apenas membros do grupo |

**Migração:** todos os grupos existentes tornam-se públicos por padrão (`is_public = true`). Admins podem alterar para fechado a qualquer momento.

### 4.2 Lista de espera

- Disponível **somente** para grupos públicos
- O botão "Entrar na fila" é exibido apenas quando:
  - O grupo tem um rachão aberto (`status = open` ou `in_progress`)
  - O jogador está **logado** e **não é membro** do grupo
  - O rachão ainda tem vagas (`confirmed_count < max_players`), ou `max_players` não está definido
- Um jogador só pode estar na fila **uma vez por rachão**
- Se o rachão for encerrado ou a data passar, candidatos pendentes são descartados automaticamente

### 4.3 Aceite de termos e apresentação

Antes de confirmar a entrada na lista de espera, o jogador preenche um modal com duas partes:

**Parte 1 — Termos do rachão (somente leitura):**
- Data, horário de início e término da partida
- Local e endereço
- Tipo de quadra
- Número de jogadores por time (se configurado)
- Limite de vagas (se configurado)
- Valor por partida e/ou mensalidade (se configurados)
- Regras gerais do grupo (campo `notes` do grupo, se preenchido)

**Parte 2 — Apresentação do candidato:**
- Campo de texto livre: **"Conte um pouco sobre você"** (opcional, máx. 500 caracteres)
- Exemplo de placeholder: _"Jogo há 5 anos, posição: meia. Disponível toda quinta."_
- Esse texto é exibido aos admins no painel de revisão para auxiliar na decisão de aceite

**Confirmação:**
- Checkbox obrigatório: **"Li e concordo com as condições acima"**
- Botão: **"Enviar candidatura"**
- O timestamp do aceite e o texto de apresentação são registrados no banco

### 4.4 Aprovação pelo admin

- Qualquer admin do grupo pode aceitar ou rejeitar candidatos
- **Ao aceitar:**
  - O candidato é adicionado ao grupo como membro (`role = player`)
  - A presença no rachão é automaticamente marcada como `confirmed`
  - O candidato recebe push de confirmação
  - A entrada da lista de espera muda para `accepted`
- **Ao rejeitar:**
  - O candidato recebe push de rejeição
  - A entrada muda para `rejected`
  - O candidato não é adicionado ao grupo
- Se o rachão atingir lotação máxima enquanto há candidatos na fila, o aceite de novos candidatos é **bloqueado** com mensagem clara ao admin ("Rachão já está lotado")

### 4.5 Pós-aceitação

- O jogador passa a ser **membro permanente** do grupo
- Nos próximos rachões, ele participa normalmente como membro (recebe convite automático se a recorrência estiver ativa)
- A entrada na lista de espera daquele rachão específico fica com status `accepted` (histórico)

---

## 5. Fluxo de Usuário

### 5.1 Jogador com cadastro — via link da partida ou do grupo

```
1. Recebe link compartilhado pelo admin:
   - Link da partida:  rachao.app/match/[hash]
   - Link do grupo:    rachao.app/groups/[id]
2. Está logado e vê a página com:
   - Detalhes do rachão (data, local, vagas disponíveis)
   - Botão "Quero jogar!" / "Entrar na fila" (se há vagas e não é membro)
3. Clica no botão → modal de candidatura é exibido:
   - Termos do rachão (somente leitura)
   - Campo "Conte um pouco sobre você" (opcional, máx. 500 caracteres)
   - Checkbox: "Li e concordo com as condições acima"
   - Botão: "Enviar candidatura"
4. Entra na fila com status "Aguardando aprovação"
5. Vê confirmação: "Candidatura enviada! Você será notificado quando um admin revisar."
6. Recebe push quando aceito ou rejeitado
```

### 5.2 Jogador sem cadastro — via link da partida

Este é o fluxo de maior oportunidade de conversão: o jogador chega via link da partida, vê vagas disponíveis, mas não tem conta.

```
1. Recebe link da partida: rachao.app/match/[hash]
2. Acessa sem estar logado — a página /match/[hash] é pública
3. Vê os detalhes da partida e as vagas disponíveis
4. Vê um card de chamada à ação:
   "Quer jogar? Crie sua conta grátis e entre na fila de espera."
   Botão: "Criar conta e participar"
5. É redirecionado para: /register?next=/match/[hash]&join_waitlist=1
6. Preenche o cadastro (nome, WhatsApp, senha)
7. Após criação da conta, é redirecionado automaticamente para /match/[hash]
8. A página detecta o parâmetro join_waitlist=1 e abre o modal de candidatura automaticamente
9. Preenche o modal:
   - Termos do rachão
   - Campo "Conte um pouco sobre você" (incentivado: "Ajude o admin a conhecer você")
   - Checkbox de aceite
   - Botão: "Enviar candidatura"
10. Entra na fila com status "Aguardando aprovação"
11. Recebe push quando aceito ou rejeitado
```

**Observação:** se o usuário já tiver uma conta mas não estiver logado, o fluxo usa `/login?next=/match/[hash]&join_waitlist=1` em vez de `/register`.

### 5.3 Admin do grupo

```
1. Recebe push: "⚽ [Nome] quer participar do rachão em [data]"
2. Acessa a página do grupo → aba "Próximos" → badge na lista de espera
3. Vê lista de candidatos com:
   - Nome e apelido do candidato
   - Data/hora da candidatura
   - Texto de apresentação (se preenchido)
4. Por candidato: botões [Aceitar] e [Rejeitar]
5. Ao aceitar:
   - Candidato desaparece da fila
   - Aparece como "Confirmado" na lista de presença do rachão
   - Admin vê toast: "Jogador adicionado ao grupo e confirmado no rachão"
6. Ao rejeitar:
   - Candidato é removido da fila
   - Admin vê toast: "Candidatura rejeitada"
```

---

## 6. Casos de Borda

| Situação | Comportamento |
|---|---|
| Rachão sem `max_players` | Lista de espera disponível sem limite de vagas |
| Rachão atinge lotação enquanto há candidatos na fila | Admin não consegue aceitar novos; vê aviso "Rachão lotado" |
| Rachão é encerrado com candidatos na fila | Candidatos pendentes são ignorados (não notificados — o rachão acabou) |
| Jogador na fila que já era membro do grupo | Impossível — botão não é exibido para membros |
| Jogador tenta entrar na fila duas vezes | API retorna erro; frontend desabilita botão após entrada |
| Grupo muda de público para fechado com candidatos na fila | Candidatos pendentes são descartados; não são notificados |
| Não há próximo rachão aberto | Botão "Entrar na fila" não é exibido; grupo exibe mensagem "Nenhum rachão agendado no momento" |
| Usuário sem conta acessa link da partida (`/match/[hash]`) de grupo fechado | Não vê botão de candidatura; pode criar conta normalmente pelo `/register` sem redirecionamento de waitlist |
| Usuário cria conta via `/register?next=...&join_waitlist=1` mas a partida já foi encerrada quando retorna | Ao carregar `/match/[hash]`, o modal não é aberto (partida `closed`); mensagem: "Este rachão já foi encerrado" |
| Usuário abandona o cadastro no meio do fluxo | Parâmetros `next` e `join_waitlist` se perdem; ao logar depois, vai para home normalmente |
| Usuário chega via `/register?next=...&join_waitlist=1` mas já tem conta | É redirecionado para `/login?next=...&join_waitlist=1` com mensagem "Você já tem uma conta. Faça login para continuar." |

---

## 7. Modelo de Dados

### Alteração na tabela `groups`

```sql
ALTER TABLE groups
  ADD COLUMN is_public BOOLEAN NOT NULL DEFAULT TRUE;
```

### Nova tabela `match_waitlist`

```sql
CREATE TYPE waitlist_status AS ENUM ('pending', 'accepted', 'rejected');

CREATE TABLE match_waitlist (
  id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  match_id     UUID         NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
  player_id    UUID         NOT NULL REFERENCES players(id) ON DELETE CASCADE,
  intro        TEXT,                                -- apresentação do candidato (máx. 500 chars, opcional)
  agreed_at    TIMESTAMPTZ  NOT NULL DEFAULT now(), -- timestamp do aceite dos termos
  status       waitlist_status NOT NULL DEFAULT 'pending',
  reviewed_by  UUID         REFERENCES players(id), -- admin que tomou a ação
  reviewed_at  TIMESTAMPTZ,
  created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
  UNIQUE (match_id, player_id)
);

CREATE INDEX idx_match_waitlist_match   ON match_waitlist (match_id);
CREATE INDEX idx_match_waitlist_player  ON match_waitlist (player_id);
CREATE INDEX idx_match_waitlist_status  ON match_waitlist (status);
```

---

## 8. API — Endpoints

### Novos endpoints

| Método | Rota | Auth | Descrição |
|---|---|---|---|
| `POST` | `/groups/{group_id}/waitlist` | Obrigatório | Entra na lista de espera (jogador externo) |
| `GET` | `/groups/{group_id}/waitlist` | Admin do grupo | Lista candidatos pendentes/histórico |
| `PATCH` | `/groups/{group_id}/waitlist/{entry_id}` | Admin do grupo | Aceita ou rejeita candidato |

### Alterações em endpoints existentes

| Método | Rota | Alteração |
|---|---|---|
| `POST` | `/groups` | Aceita campo `is_public` (boolean, default `true`) |
| `PATCH` | `/groups/{group_id}` | Aceita campo `is_public` |
| `GET` | `/groups/{group_id}` | Expõe `is_public` na resposta |
| `GET` | `/groups/{group_id}/matches` | Página pública passa a ser acessível sem auth se grupo for público |

### Payload `POST /groups/{group_id}/waitlist`

```json
{
  "agreed": true,   // aceite explícito dos termos — obrigatório
  "intro": "Jogo há 5 anos, posição: meia. Disponível toda quinta."  // opcional, máx. 500 chars
}
```

### Payload `PATCH /groups/{group_id}/waitlist/{entry_id}`

```json
{
  "action": "accept"  // ou "reject"
}
```

---

## 9. Frontend — Páginas e Componentes

| Arquivo | Alteração |
|---|---|
| `src/routes/groups/new/+page.svelte` | Adiciona toggle "Visibilidade" (Público / Fechado) |
| `src/routes/groups/[id]/+page.svelte` | Exibe toggle editável de visibilidade; mostra painel de lista de espera para admins; exibe botão "Entrar na fila" para não-membros logados |
| `src/routes/match/[hash]/+page.svelte` | Exibe card de chamada à ação para não-logados com vagas disponíveis; abre modal de candidatura automaticamente se `join_waitlist=1` na URL |
| `src/routes/register/+page.svelte` | Preserva parâmetros `next` e `join_waitlist` na URL após cadastro e redireciona corretamente |
| `src/routes/login/+page.svelte` | Preserva parâmetros `next` e `join_waitlist` na URL após login e redireciona corretamente |
| `src/lib/components/WaitlistModal.svelte` | Modal: termos do rachão + campo "Conte sobre você" + checkbox + botão (novo componente) |
| `src/lib/components/WaitlistPanel.svelte` | Painel admin: lista de candidatos com apresentação e ações Aceitar/Rejeitar (novo componente) |

### Visibilidade do botão "Entrar na fila" (usuário logado)

```
Exibir quando:
  - grupo.is_public === true
  - usuário está logado
  - usuário NÃO é membro do grupo
  - existe um rachão com status 'open' ou 'in_progress'
  - confirmed_count < max_players OU max_players == null
  - usuário ainda não está na fila desse rachão

Desabilitar (com mensagem) quando:
  - rachão está lotado (confirmed_count >= max_players)
  - usuário já está na fila (status 'pending')
```

### Card de chamada à ação (usuário não logado na página /match/[hash])

```
Exibir quando:
  - usuário NÃO está logado
  - grupo do rachão é público (is_public = true)
  - rachão tem status 'open' e há vagas disponíveis

Conteúdo:
  "Quer jogar? Crie sua conta grátis e entre na fila de espera."
  [Criar conta e participar] → /register?next=/match/[hash]&join_waitlist=1
  [Já tenho conta]           → /login?next=/match/[hash]&join_waitlist=1
```

### Abertura automática do modal pós-cadastro/login

```
Na página /match/[hash], ao detectar join_waitlist=1 na URL:
  - Aguardar carregamento do match (status check)
  - Se match.status === 'open' e há vagas → abre WaitlistModal automaticamente
  - Remove join_waitlist=1 da URL (history.replaceState) após abrir o modal
  - Se match já estiver closed → exibe mensagem informativa, não abre modal
```

---

## 10. Notificações Push

| Evento | Destinatário | Título | Corpo |
|---|---|---|---|
| Novo candidato na fila | Todos os admins do grupo | `⚽ Novo candidato — [grupo]` | `[Nome] quer participar do rachão em [data]. Acesse o grupo para revisar.` |
| Candidato aceito | Candidato | `✅ Você foi aceito!` | `Bem-vindo ao grupo [nome]! Sua presença no rachão de [data] foi confirmada.` |
| Candidato rejeitado | Candidato | `❌ Candidatura não aprovada` | `Sua candidatura para o grupo [nome] não foi aprovada desta vez.` |

---

## 11. Feed de Rachões com Vagas (descoberta orgânica)

### 11.1 Contexto

Na versão atual, grupos públicos são encontrados **exclusivamente via link direto** compartilhado pelo admin. Para que o crescimento orgânico de fato aconteça, é necessário que jogadores sem vínculo prévio com um grupo **descubram ativamente rachões próximos com vagas** — sem depender de ninguém compartilhar um link.

Esta seção especifica um feed de descoberta integrado à plataforma, acessível a qualquer jogador logado.

---

### 11.2 Onde exibir

**Opção A — Card na home (dashboard):** seção destacada na tela inicial, logo abaixo do próximo rachão do próprio jogador. Exibe até 3 rachões com vagas de grupos públicos aos quais o jogador não pertence.

**Opção B — Página dedicada `/discover`:** listagem completa, acessível via ícone/tab na navegação principal. Permite rolagem, paginação e filtros.

**Recomendação:** implementar as duas em conjunto — card na home como entry point, página `/discover` como destino ao clicar em "Ver mais".

---

### 11.3 Regras de exibição

Um rachão aparece no feed quando:

- O grupo é público (`is_public = true`)
- O jogador logado **não é membro** do grupo
- O rachão tem `status = 'open'`
- A `match_date` é **hoje ou futura**
- `confirmed_count < max_players` **ou** `max_players` é nulo (sem limite)
- O jogador não está na fila desse rachão (não exibir se já candidatou)

Ordenação padrão: **data mais próxima primeiro**, depois por vagas disponíveis (menos vagas = mais urgência → aparece antes).

---

### 11.4 Informações exibidas por card

Cada card no feed exibe:

| Campo | Origem |
|---|---|
| Nome do grupo | `groups.name` |
| Data e horário | `matches.match_date` + `start_time` |
| Local | `matches.location` |
| Tipo de quadra | `matches.court_type` |
| Vagas disponíveis | `max_players - confirmed_count` (ou "Sem limite") |
| Jogadores por time | `matches.players_per_team` (se definido) |
| Botão de ação | "Quero jogar!" → abre `WaitlistModal` (mesmo fluxo já implementado) |

Campos opcionais (exibidos se preenchidos): endereço, notas/regras do grupo.

---

### 11.5 Filtros (página `/discover`)

| Filtro | Tipo | Observação |
|---|---|---|
| Data | Seletor de período (hoje / esta semana / próximos 30 dias) | Padrão: "esta semana" |
| Tipo de quadra | Multi-select (society, futsal, campo, areia) | Baseado em `court_type` |
| Dia da semana | Multi-select (seg–dom) | Extraído de `match_date` |
| Horário | Faixa (manhã / tarde / noite) | Baseado em `start_time` |
| Com vagas | Toggle | Ligado por padrão |

Filtros de **cidade/bairro** são desejáveis mas dependem de geocodificação — fora do escopo desta versão.

---

### 11.6 Fluxo do usuário

```
1. Jogador logado acessa a home ou /discover
2. Vê card(s) com rachões de grupos públicos com vagas
3. Clica em "Quero jogar!" → WaitlistModal abre (mesmo componente já implementado)
4. Preenche apresentação opcional + aceita termos
5. Entra na fila → feedback "Candidatura enviada!"
6. Admin do grupo recebe push e revisa
7. Jogador recebe push ao ser aceito ou rejeitado
```

Ao ser aceito, o jogador passa a ser membro permanente do grupo e não vê mais esse grupo no feed.

---

### 11.7 Impacto em componentes existentes

- `WaitlistModal.svelte` — reutilizado sem alteração
- `src/routes/+page.svelte` (home/dashboard) — adicionar seção "Rachões com vaga perto de você"
- Nova rota `src/routes/discover/+page.svelte` — listagem completa com filtros
- Nova navegação: ícone/tab "Descobrir" no menu principal (mobile: barra inferior)

---

### 11.8 API

**Novo endpoint:**

```
GET /api/v1/matches/discover
```

- Requer autenticação
- Retorna partidas de grupos públicos com vaga que o jogador não pertence
- Parâmetros de query: `date_from`, `date_to`, `court_type` (multi), `weekday` (multi), `limit`, `offset`
- Response: lista de objetos com dados da partida + grupo + contagem de confirmados/vagas

---

### 11.9 Privacidade e limites

- Jogadores cujo grupo mudou para fechado (`is_public = false`) deixam de aparecer no feed imediatamente
- O feed **não expõe** lista de membros do grupo — apenas as informações públicas da partida
- Admin global não vê o feed de descoberta (role `admin` é isento)
- Grupos com admin inativo por mais de 90 dias podem ser excluídos do feed futuramente (fora do escopo atual)

---

## 12. Checklist de Implementação

### Backend
- [x] Migration: coluna `is_public` na tabela `groups` (default `true`)
- [x] Migration: tabela `match_waitlist` e enum `waitlist_status` (com coluna `intro TEXT`)
- [x] Model `Group`: adicionar campo `is_public`
- [x] Model `MatchWaitlist`: novo model (com campo `intro`)
- [x] Repository `WaitlistRepository`: CRUD + queries por match/player
- [x] `POST /groups/{id}/waitlist`: valida grupo público, rachão aberto, vagas, not already member/in queue — aceita `intro` opcional (máx. 500 chars) — cria entrada + envia push para admins
- [x] `GET /groups/{id}/waitlist`: retorna candidatos com `intro` visível (admin only)
- [x] `GET /groups/{id}/waitlist/me`: retorna entrada do jogador atual no rachão ativo (ou null)
- [x] `PATCH /groups/{id}/waitlist/{entry_id}`: aceita (add member + confirm attendance + push) ou rejeita (push) — bloqueia aceite se rachão lotado
- [x] Atualizar `POST /groups` e `PATCH /groups/{id}` com campo `is_public`
- [x] Expor `is_public` e `group_is_public` no response do `GET /matches/public/{hash}` (para o frontend saber se deve exibir o CTA)

### Frontend
- [x] Toggle "Visibilidade" em `groups/new`
- [x] Toggle editável em `groups/[id]` (configurações do grupo, admin only)
- [x] Badge de visibilidade (Público/Fechado) no cabeçalho da página do grupo
- [x] Exibir próximo rachão na visão pública do grupo para não-membros logados
- [x] Botão "Entrar na fila" com todos os guards de exibição/habilitação (usuários logados)
- [x] `WaitlistModal.svelte`: termos do rachão + campo "Conte sobre você" (textarea, máx. 500 chars, opcional) + checkbox + botão confirmar
- [x] `WaitlistPanel.svelte`: painel admin com lista de candidatos, exibição do `intro` e ações Aceitar/Rejeitar
- [x] Feedback de status para candidato na página do grupo e do rachão ("Aguardando aprovação")
- [x] Card de CTA em `/match/[hash]` para usuários não logados (grupo público + vagas)
- [x] Suporte a `?next=` e `?join_waitlist=1` em `/register` e `/login`
- [x] Abertura automática do `WaitlistModal` em `/match/[hash]` quando `join_waitlist=1` na URL pós-auth

### Testes E2E
- [x] Badge de visibilidade (Público/Fechado) visível na página do grupo
- [x] Modal de edição exibe toggle `is_public`
- [x] Admin vê painel de lista de espera em grupo público com rachão aberto
- [x] Membro/admin não vê botão "Quero jogar!" (apenas não-membros devem ver)
- [x] Usuário não logado vê CTA na página da partida de grupo público
- [x] Link "Criar conta e participar" contém `?join_waitlist=1` e `?next=` corretos
- [ ] Jogador logado → entra na fila via página do grupo → admin aceita → aparece como confirmado
- [ ] Jogador logado → entra na fila via página da partida → admin rejeita → não adicionado ao grupo
- [ ] Jogador sem conta → acessa link da partida → vê CTA → cadastra → modal abre automaticamente → entra na fila
- [ ] Jogador com conta sem login → acessa link da partida → vê CTA → faz login → modal abre automaticamente
- [ ] Tentativa de entrar na fila em grupo fechado → botão não exibido
- [ ] Tentativa de aceitar candidato com rachão lotado → erro claro
- [ ] Partida encerrada ao retornar pós-cadastro → modal não abre, mensagem informativa exibida
