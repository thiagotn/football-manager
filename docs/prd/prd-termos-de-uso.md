# PRD — Termos de Uso
## Rachao.app · Gerenciamento de Grupos e Partidas

| | |
|---|---|
| **Versão** | 1.0 |
| **Status** | Draft |
| **Data** | Março de 2026 |
| **Plataforma** | https://rachao.app |

---

## 1. Visão Geral

### 1.1 Contexto

O rachao.app está em fase Beta e já possui usuários ativos. A plataforma não dispõe atualmente de um Termo de Uso publicado e acessível. Com a evolução prevista para planos pagos e crescimento da base de usuários, a ausência deste documento representa um risco jurídico e de credibilidade para o produto.

### 1.2 Problema

- Não há instrumento jurídico formal que regule a relação entre a plataforma e seus usuários.
- Ausência de regras claras sobre uso aceito, responsabilidades e conduta esperada dos usuários.
- Usuários não são explicitamente informados das condições do período Beta (possibilidade de reset de dados, mudanças de funcionalidades).
- Sem aceite formal, a plataforma fica mais exposta em eventuais conflitos.

### 1.3 Objetivo

Publicar o Termo de Uso do rachao.app e integrar o aceite ao fluxo de cadastro (`/register`) e, retroativamente, para usuários já cadastrados.

---

## 2. Escopo

Este PRD cobre:

- Criação da rota `/termos` com o texto do Termo de Uso;
- Adição de checkbox de aceite obrigatório no formulário de cadastro;
- Exibição de aceite retroativo para usuários já cadastrados que ainda não aceitaram;
- Link para `/termos` no rodapé e nas telas de cadastro e login;
- Armazenamento do registro de aceite (data, versão, IP).

**Fora de escopo:** Política de Privacidade (PRD separado), cookie banner, consentimento de marketing.

---

## 3. Requisitos Funcionais

**RF-01 — Rota pública `/termos`**
Criar página pública com o texto completo do Termo de Uso. Acessível sem login, indexável por motores de busca.

**RF-02 — Aceite obrigatório no cadastro**
O formulário `/register` deve incluir, antes do botão de envio:
- Checkbox: *"Li e aceito os [Termos de Uso]"* (link para `/termos` em nova aba).
- O botão "Criar conta" deve permanecer desabilitado enquanto o checkbox não estiver marcado.

**RF-03 — Aceite retroativo**
Usuários já cadastrados que não possuam registro de aceite devem ser redirecionados para uma tela de aceite ao fazer login. O acesso à plataforma só é liberado após aceite. Não deve ser possível ignorar ou fechar a tela sem aceitar.

**RF-04 — Registro do aceite no backend**
Ao aceitar, o backend deve registrar:
- `player_id`
- `terms_version` (ex.: `"1.0"`)
- `accepted_at` (timestamp)
- `ip_address` (para fins de comprovação)

**RF-05 — Versionamento do Termo**
O sistema deve suportar versionamento. Quando uma nova versão for publicada, usuários devem ser solicitados a aceitar novamente na próxima sessão.

**RF-06 — Link no rodapé e telas chave**
O link "Termos de Uso" deve aparecer em:
- Rodapé da landing page (`/lp`);
- Rodapé do layout principal (pós-login);
- Tela de cadastro (`/register`);
- Tela de login (`/login`).

---

## 4. Requisitos Não Funcionais

**RNF-01 — Leiturabilidade:** o texto deve ser renderizado com tipografia adequada para leitura longa (parágrafos, títulos, espaçamento).

**RNF-02 — Performance:** a página `/termos` deve carregar sem dependência de autenticação, sem dados dinâmicos da API.

**RNF-03 — Imutabilidade do registro:** o registro de aceite nunca deve ser deletado ou sobrescrito — apenas novos registros são criados para novas versões.

---

## 5. Modelagem de Dados

```sql
-- Migration: 018_terms_acceptance.sql
CREATE TABLE terms_acceptance (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    player_id     UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    terms_version VARCHAR(10) NOT NULL,   -- ex: "1.0", "1.1"
    accepted_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ip_address    INET
);

CREATE INDEX idx_terms_acceptance_player ON terms_acceptance (player_id, terms_version);
```

---

## 6. Endpoints da API

| Método | Endpoint | Descrição |
|---|---|---|
| `POST` | `/api/v1/auth/accept-terms` | Registra aceite do termo (auth obrigatória) |
| `GET`  | `/api/v1/auth/terms-status` | Retorna se o usuário aceitou a versão atual |

### 6.1 `POST /auth/accept-terms`

**Request:**
```json
{ "terms_version": "1.0" }
```

**Response 200:**
```json
{ "accepted": true, "terms_version": "1.0", "accepted_at": "2026-03-10T14:00:00Z" }
```

**Erros:**
- `400` — versão não reconhecida
- `409` — aceite já registrado para esta versão

### 6.2 `GET /auth/terms-status`

**Response 200:**
```json
{
  "current_version": "1.0",
  "accepted": true,
  "accepted_at": "2026-03-10T14:00:00Z"
}
```

Se `"accepted": false`, o frontend deve exibir a tela de aceite obrigatório.

---

## 7. Fluxo de Aceite Retroativo

```
Usuário faz login
       ↓
[GET /auth/terms-status]
  accepted = false?
       ↓ sim
[Tela de aceite obrigatório]
  Exibe resumo dos Termos + link para /termos
  Botão "Li e aceito os Termos de Uso"
       ↓
[POST /auth/accept-terms]
  { "terms_version": "1.0" }
       ↓
Redireciona para destino original
```

---

## 8. Interface do Usuário

### 8.1 Checkbox no cadastro

```
┌─────────────────────────────────────────┐
│  [ ] Li e aceito os Termos de Uso       │
│      (link abre em nova aba)            │
│                                         │
│  [ Criar conta grátis ] ← desabilitado  │
│             até checkbox marcado        │
└─────────────────────────────────────────┘
```

### 8.2 Tela de aceite retroativo (modal ou página dedicada)

```
┌─────────────────────────────────────────┐
│  Termos de Uso atualizados              │
│─────────────────────────────────────────│
│  Para continuar usando o rachao.app,    │
│  você precisa aceitar nossos Termos     │
│  de Uso.                                │
│                                         │
│  [Leia os Termos de Uso completos →]    │
│                                         │
│  [ Aceitar e continuar ]                │
└─────────────────────────────────────────┘
```

---

## 9. Arquivos a Criar / Modificar

### Backend
- `football-api/migrations/018_terms_acceptance.sql` — criar
- `football-api/app/models/terms_acceptance.py` — criar
- `football-api/app/db/repositories/terms_repo.py` — criar
- `football-api/app/api/v1/routers/auth.py` — modificar (novos endpoints)
- `football-api/app/schemas/terms.py` — criar

### Frontend
- `football-frontend/src/routes/termos/+page.svelte` — criar (texto estático)
- `football-frontend/src/routes/register/+page.svelte` — modificar (adicionar checkbox)
- `football-frontend/src/lib/components/TermsAcceptanceModal.svelte` — criar
- `football-frontend/src/hooks.server.ts` ou guard de rota — modificar (verificar aceite pós-login)
- `football-frontend/src/lib/api.ts` — modificar (novos endpoints)
- Layout principal e landing page — modificar (adicionar link no rodapé)

---

## 10. Critérios de Aceitação

- [ ] Página `/termos` acessível sem login com texto completo e formatado
- [ ] Link para `/termos` visível no rodapé da LP, rodapé logado, cadastro e login
- [ ] Checkbox de aceite presente e obrigatório no cadastro
- [ ] Botão "Criar conta" desabilitado sem aceite marcado
- [ ] Usuários já cadastrados sem aceite são bloqueados até aceitar
- [ ] Registro de aceite salvo com `player_id`, `version`, `timestamp` e IP
- [ ] Nova versão do termo dispara novo ciclo de aceite

---

## 11. Fora de Escopo

- Política de Privacidade (PRD separado)
- Notificação por e-mail ou WhatsApp sobre atualização de termos
- Histórico de versões anteriores acessível ao usuário
- Exportação do comprovante de aceite

---

*Documento elaborado para uso interno da equipe de produto e engenharia do Rachao.app.*
