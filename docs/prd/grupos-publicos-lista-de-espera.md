# PRD — Grupos Públicos e Lista de Espera

**Status:** Proposta
**Data:** 2026-03-19
**Contexto:** Hoje todos os grupos são fechados — só participam membros convidados pelo admin. Esta feature adiciona visibilidade pública opcional e um mecanismo de entrada por lista de espera, permitindo crescimento orgânico dos grupos.

---

## 1. Problema

- Grupos novos não têm como atrair jogadores sem que o admin convide manualmente cada um
- Jogadores avulsos interessados em um rachão não têm forma de se candidatar
- O admin precisa saber de antemão quem convidar — não há fluxo orgânico de adesão
- Grupos que queiram abrir vagas para desconhecidos não têm ferramenta para isso

---

## 2. Objetivo

Permitir que grupos sejam configurados como **públicos**, tornando seu próximo rachão visível para qualquer jogador da plataforma via link direto. Jogadores externos podem se candidatar a uma vaga via **lista de espera**, concordar com os termos do rachão, e ser aceitos pelos admins — entrando automaticamente no grupo com presença confirmada no rachão.

---

## 3. Escopo

### Incluso
- Flag `is_public` no grupo (criação e edição)
- Página pública do grupo (acessível via link, sem necessidade de ser membro)
- Fluxo de entrada na lista de espera com aceite de termos
- Painel de admins para aceite/rejeição de candidatos
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

### 4.3 Aceite de termos

Antes de confirmar a entrada na lista de espera, o jogador vê um resumo com:
- Data, horário de início e término da partida
- Local e endereço
- Tipo de quadra
- Número de jogadores por time (se configurado)
- Limite de vagas (se configurado)
- Valor por partida e/ou mensalidade (se configurados)
- Regras gerais do grupo (campo `notes` do grupo, se preenchido)

O jogador precisa marcar um checkbox — **"Li e concordo com as condições acima"** — para prosseguir. O timestamp do aceite é registrado no banco.

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

### Jogador externo

```
1. Recebe link compartilhado pelo admin (ex: rachao.app/groups/[id])
2. Vê a página pública do grupo:
   - Nome do grupo, tipo de quadra, local
   - Próximo rachão: data, horário, vagas disponíveis (X/Y ou "vagas abertas")
   - Botão "Entrar na fila" (visível se há vagas e usuário está logado)
3. Clica em "Entrar na fila"
4. Modal de termos é exibido:
   - Detalhes do rachão
   - Valores (se houver)
   - Regras do grupo (se houver)
   - Checkbox: "Li e concordo com as condições acima"
   - Botão: "Confirmar candidatura"
5. Entra na fila com status "Aguardando aprovação"
6. Vê mensagem de confirmação: "Sua candidatura foi enviada. Você será notificado quando um admin revisar."
7. Recebe push quando aceito ou rejeitado
```

### Admin do grupo

```
1. Recebe push: "⚽ [Nome] quer participar do rachão em [data]"
2. Acessa a página do grupo → aba "Próximos" → badge na lista de espera
3. Vê lista de candidatos: nome, data de candidatura, status
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
  agreed_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),  -- timestamp do aceite dos termos
  status       waitlist_status NOT NULL DEFAULT 'pending',
  reviewed_by  UUID         REFERENCES players(id),  -- admin que tomou a ação
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
  "agreed": true  // aceite explícito dos termos — campo obrigatório
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
| `src/routes/groups/[id]/+page.svelte` | Exibe toggle editável de visibilidade; mostra painel de lista de espera para admins; exibe botão "Entrar na fila" para não-membros |
| `src/lib/components/WaitlistModal.svelte` | Modal com termos do rachão + checkbox de aceite (novo componente) |
| `src/lib/components/WaitlistPanel.svelte` | Painel admin: lista de candidatos com botões Aceitar/Rejeitar (novo componente) |

### Visibilidade do botão "Entrar na fila"

```
Exibir quando:
  - grupo.is_public === true
  - usuário está logado
  - usuário NÃO é membro do grupo
  - existe um rachão com status 'open'
  - confirmed_count < max_players OU max_players == null
  - usuário ainda não está na fila desse rachão

Desabilitar (com mensagem) quando:
  - rachão está lotado (confirmed_count >= max_players)
  - usuário já está na fila (status 'pending')
```

---

## 10. Notificações Push

| Evento | Destinatário | Título | Corpo |
|---|---|---|---|
| Novo candidato na fila | Todos os admins do grupo | `⚽ Novo candidato — [grupo]` | `[Nome] quer participar do rachão em [data]. Acesse o grupo para revisar.` |
| Candidato aceito | Candidato | `✅ Você foi aceito!` | `Bem-vindo ao grupo [nome]! Sua presença no rachão de [data] foi confirmada.` |
| Candidato rejeitado | Candidato | `❌ Candidatura não aprovada` | `Sua candidatura para o grupo [nome] não foi aprovada desta vez.` |

---

## 11. Descoberta de grupos públicos (escopo futuro)

Nesta versão, grupos públicos são encontrados **exclusivamente via link direto** compartilhado pelo admin. Uma página de descoberta poderá ser construída futuramente com:

- Listagem de grupos públicos com vagas abertas no próximo rachão
- Filtros: cidade, tipo de quadra, dia da semana, horário
- Rota sugerida: `/groups/explore` ou `/discover`

---

## 12. Checklist de Implementação

### Backend
- [ ] Migration: coluna `is_public` na tabela `groups` (default `true`)
- [ ] Migration: tabela `match_waitlist` e enum `waitlist_status`
- [ ] Model `Group`: adicionar campo `is_public`
- [ ] Model `MatchWaitlist`: novo model
- [ ] Repository `WaitlistRepository`: CRUD + queries por match/player
- [ ] `POST /groups/{id}/waitlist`: valida grupo público, rachão aberto, vagas, not already member/in queue — cria entrada + envia push para admins
- [ ] `GET /groups/{id}/waitlist`: retorna candidatos (admin only)
- [ ] `PATCH /groups/{id}/waitlist/{entry_id}`: aceita (add member + confirm attendance + push) ou rejeita (push) — bloqueia aceite se rachão lotado
- [ ] Atualizar `POST /groups` e `PATCH /groups/{id}` com campo `is_public`
- [ ] Ajustar visibilidade do `GET /groups/{id}/matches` para grupos públicos (sem auth)

### Frontend
- [ ] Toggle "Visibilidade" em `groups/new`
- [ ] Toggle editável em `groups/[id]` (configurações do grupo, admin only)
- [ ] Exibir próximo rachão na visão pública do grupo para não-membros
- [ ] Botão "Entrar na fila" com todos os guards de exibição/habilitação
- [ ] `WaitlistModal.svelte`: termos do rachão + checkbox + botão confirmar
- [ ] `WaitlistPanel.svelte`: painel admin com lista de candidatos e ações
- [ ] Feedback de status para candidato na página do grupo ("Aguardando aprovação")

### Testes E2E
- [ ] Jogador externo → entra na fila → admin aceita → aparece como confirmado
- [ ] Jogador externo → entra na fila → admin rejeita → não é adicionado ao grupo
- [ ] Tentativa de entrar na fila sem login → redireciona para login
- [ ] Tentativa de entrar na fila em grupo fechado → botão não exibido
- [ ] Tentativa de aceitar candidato com rachão lotado → erro claro
