# PRD — Termos de Uso
## Rachao.app · Gerenciamento de Grupos e Partidas

| | |
|---|---|
| **Versão** | 1.1 |
| **Status** | Pronto para implementação |
| **Data** | Março de 2026 |
| **Plataforma** | https://rachao.app |

---

## 1. Visão Geral

### 1.1 Contexto

O rachao.app já possui usuários ativos na fase Beta. Não há Termos de Uso publicados. Com planos pagos previstos e crescimento da base, a ausência do documento representa risco jurídico e de credibilidade.

### 1.2 Objetivo

Publicar os Termos de Uso em `/terms` e integrar o aceite ao fluxo de cadastro e, retroativamente, aos usuários já cadastrados. Não requer revisão jurídica prévia para entrar no ar — um texto honesto e preciso já atende o espírito da lei.

> **Nota:** revisão jurídica por advogado especializado antes do lançamento dos planos pagos é recomendada, mas não é pré-requisito para a fase Beta.

---

## 2. Pendências antes de publicar

| # | Ação | Responsável |
|---|---|---|
| 1 | Definir nome completo e CPF/CNPJ do responsável | Fundador |
| 2 | Definir cidade/estado para o foro de eleição | Fundador |
| 3 | Criar e-mail de contato (ex.: `contato@rachao.app`) e garantir monitoramento | Fundador |

Preencher os `[PLACEHOLDERS]` no texto da seção 3 antes de publicar.

---

## 3. Texto dos Termos de Uso

> Este é o texto a ser publicado em `/terms`. Copiar para o componente Svelte substituindo os placeholders.

---

### Termos de Uso — rachao.app

**Última atualização:** março de 2026 · **Versão:** 1.0

---

#### 1. Aceitação

Ao criar uma conta ou usar o rachao.app, você declara ter lido e aceitar estes Termos de Uso. Se não concordar, não utilize a plataforma.

O rachao.app é operado por **[NOME COMPLETO DO RESPONSÁVEL], [CPF ou CNPJ]** ("nós"), com contato em **[contato@rachao.app]**.

A plataforma está em **fase Beta**: funcionalidades podem mudar, ser removidas ou ter dados resetados com aviso mínimo de 48 horas. Não há garantia de disponibilidade contínua (SLA) nesta fase.

---

#### 2. Cadastro e conta

- Você deve informar dados verídicos no cadastro (nome, apelido, número de WhatsApp).
- Cada número de WhatsApp corresponde a uma única conta.
- Você é responsável pela segurança da sua senha e por toda atividade realizada com sua conta.
- Não é permitido criar contas em nome de terceiros sem autorização.
- Menores de 18 anos devem ter consentimento do responsável legal.

---

#### 3. Uso aceito

Você pode usar o rachao.app para:

- Organizar grupos e partidas de futebol amador;
- Confirmar ou recusar presença em partidas;
- Votar nos melhores jogadores e acompanhar estatísticas do grupo.

É expressamente **proibido**:

- Usar a plataforma para fins ilegais ou que violem direitos de terceiros;
- Criar contas falsas, automatizar ações ou usar bots;
- Tentar acessar dados de outros usuários ou grupos sem autorização;
- Enviar conteúdo ofensivo, discriminatório ou abusivo para outros usuários.

---

#### 4. Fase Beta

Durante o período Beta:

- Funcionalidades podem ser alteradas, adicionadas ou removidas sem aviso prévio.
- Dados de toda a plataforma podem ser resetados com aviso mínimo de 48 horas pelo canal de comunicação do produto.
- Não garantimos disponibilidade contínua, tempo de resposta ou ausência de falhas.

Ao aceitar estes termos, você reconhece e aceita essas condições específicas do período Beta.

---

#### 5. Responsabilidades do usuário

Você é responsável por:

- Garantir que as informações fornecidas são verdadeiras e atualizadas;
- Usar a plataforma de acordo com estes Termos e a legislação vigente;
- Qualquer disputa ou conflito decorrente das partidas organizadas pela plataforma — o rachao.app é uma ferramenta de organização, não parte das partidas.

---

#### 6. Limitação de responsabilidade

Na extensão permitida pela lei, o rachao.app não se responsabiliza por:

- Danos indiretos, incidentais ou lucros cessantes;
- Perda de dados durante o período Beta;
- Falhas de disponibilidade ou interrupções do serviço;
- Conflitos entre usuários decorrentes das partidas organizadas.

O serviço é fornecido "no estado em que se encontra" (as-is), sem garantias implícitas de adequação a uma finalidade específica.

---

#### 7. Propriedade intelectual

A marca, logotipo, design e código-fonte do rachao.app são de propriedade exclusiva do controlador. O uso da plataforma não transfere nenhum direito de propriedade intelectual ao usuário.

Os dados inseridos pelos usuários (nomes, resultados, estatísticas) permanecem de propriedade dos respectivos usuários. Ao usar a plataforma, você concede ao rachao.app licença não exclusiva para armazenar e exibir esses dados no contexto do serviço.

---

#### 8. Cancelamento e encerramento

Você pode solicitar a exclusão da sua conta a qualquer momento pelo canal **[contato@rachao.app]**. A conta e seus dados serão removidos conforme os prazos descritos na [Política de Privacidade](/privacidade).

Podemos suspender ou encerrar sua conta caso haja violação destes Termos, com ou sem aviso prévio dependendo da gravidade da infração.

---

#### 9. Alterações nestes Termos

Quando publicarmos uma nova versão dos Termos, você será notificado na próxima vez que acessar a plataforma e precisará aceitar para continuar usando. A data no topo deste documento indica a versão vigente.

---

#### 10. Lei aplicável e foro

Estes Termos são regidos pela legislação brasileira. Fica eleito o foro da comarca de **[CIDADE/ESTADO]** para dirimir quaisquer controvérsias, com renúncia a qualquer outro, por mais privilegiado que seja.

---

#### 11. Contato

**[NOME COMPLETO], [CPF/CNPJ]**
**E-mail:** [contato@rachao.app]
**Plataforma:** https://rachao.app

---

## 4. Requisitos Funcionais

**RF-01 — Rota pública `/terms`**
Página estática com o texto da seção 3. Acessível sem login.

**RF-02 — Aceite obrigatório no cadastro**
Em `/register`, antes do botão de envio:
- Checkbox: *"Li e aceito os [Termos de Uso]"* (link para `/terms` abre em nova aba).
- Botão "Criar conta" permanece desabilitado enquanto o checkbox não estiver marcado.

**RF-03 — Aceite retroativo**
Usuários já cadastrados sem aceite registrado veem um modal não-dismissível na próxima vez que acessam a plataforma. O acesso só é liberado após aceitar. Implementado via campo `terms_accepted` retornado pelo `/auth/me`.

**RF-04 — Registro do aceite**
Ao aceitar, o backend registra: `player_id`, `terms_version`, `accepted_at`, `ip_address`.

**RF-05 — Versionamento**
Quando uma nova versão for publicada (alterar `CURRENT_TERMS_VERSION` no backend), usuários com aceite na versão anterior são tratados como "não aceito" e veem o modal novamente.

**RF-06 — Links no rodapé e telas-chave**
O link "Termos de Uso" deve aparecer em:
- Rodapé da landing page (`/lp`)
- Rodapé do layout principal (pós-login)
- Tela de cadastro (`/register`)
- Tela de login (`/login`)

---

## 5. Modelagem de Dados

```sql
-- Migration: 018_terms_acceptance.sql
CREATE TABLE terms_acceptance (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    player_id     UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    terms_version VARCHAR(10) NOT NULL,   -- ex: "1.0"
    accepted_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ip_address    INET
);

CREATE INDEX idx_terms_acceptance_player ON terms_acceptance (player_id, terms_version);
```

---

## 6. Endpoints da API

| Método | Endpoint | Descrição |
|---|---|---|
| `POST` | `/api/v1/auth/accept-terms` | Registra aceite (auth obrigatória) |

O campo `terms_accepted: bool` é incluído no objeto `Player` retornado por `POST /auth/login` e `GET /auth/me` — sem endpoint extra de status.

### `POST /auth/accept-terms`

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

### Constante no backend

```python
# app/config.py ou app/api/v1/routers/auth.py
CURRENT_TERMS_VERSION = "1.0"
```

O campo `terms_accepted` no objeto Player é calculado via JOIN com `terms_acceptance` ao carregar o usuário:

```sql
EXISTS (
  SELECT 1 FROM terms_acceptance
  WHERE player_id = players.id
    AND terms_version = :current_version
) AS terms_accepted
```

---

## 7. Fluxo de Aceite Retroativo

```
App carrega / usuário faz login
         ↓
authStore.player.terms_accepted = false?
         ↓ sim
[Modal não-dismissível]
  "Para continuar, aceite os Termos de Uso"
  [Leia os Termos completos →] (abre /termos em nova aba)
  [ Aceitar e continuar ]
         ↓
POST /auth/accept-terms { "terms_version": "1.0" }
         ↓
authStore.player.terms_accepted = true → modal fecha
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

### 8.2 Modal de aceite retroativo

```
┌─────────────────────────────────────────┐
│  Termos de Uso                          │
│─────────────────────────────────────────│
│  Para continuar usando o rachao.app,    │
│  você precisa aceitar nossos Termos     │
│  de Uso.                                │
│                                         │
│  [Leia os Termos completos →]           │
│                                         │
│  [ Aceitar e continuar ]                │
└─────────────────────────────────────────┘
```

Não há botão de fechar. O modal cobre a tela inteira (backdrop sem clique para fechar).

---

## 9. Arquivos a Criar / Modificar

### Backend
| Arquivo | Ação |
|---|---|
| `football-api/migrations/018_terms_acceptance.sql` | Criar |
| `football-api/app/models/terms_acceptance.py` | Criar |
| `football-api/app/db/repositories/terms_repo.py` | Criar |
| `football-api/app/api/v1/routers/auth.py` | Modificar — endpoint `accept-terms` + `CURRENT_TERMS_VERSION` |
| `football-api/app/schemas/auth.py` | Modificar — adicionar `terms_accepted: bool` ao PlayerResponse |
| `football-api/app/schemas/terms.py` | Criar — `AcceptTermsRequest`, `AcceptTermsResponse` |

### Frontend
| Arquivo | Ação |
|---|---|
| `football-frontend/src/routes/termos/+page.svelte` | Criar — texto estático da seção 3 |
| `football-frontend/src/routes/register/+page.svelte` | Modificar — checkbox antes do botão |
| `football-frontend/src/lib/components/TermsAcceptanceModal.svelte` | Criar — modal retroativo |
| `football-frontend/src/routes/+layout.svelte` | Modificar — montar modal se `terms_accepted = false` |
| `football-frontend/src/lib/api.ts` | Modificar — `auth.acceptTerms()`, `terms_accepted` no tipo Player |
| `football-frontend/src/routes/+page.svelte` (LP) | Modificar — link no rodapé |
| `football-frontend/src/routes/login/+page.svelte` | Modificar — link no rodapé |

---

## 10. Critérios de Aceitação

- [ ] Preencher todos os `[PLACEHOLDERS]` antes de publicar
- [ ] Página `/terms` acessível sem login, texto formatado para leitura
- [ ] Link para `/terms` no rodapé da LP, rodapé logado, cadastro e login
- [ ] Checkbox de aceite obrigatório no cadastro — botão desabilitado sem marcação
- [ ] Usuários sem aceite veem modal bloqueante na próxima sessão
- [ ] Aceite salvo com `player_id`, `terms_version`, `accepted_at` e IP
- [ ] Campo `terms_accepted` presente no objeto Player da API
- [ ] Nova versão (alterar `CURRENT_TERMS_VERSION`) dispara novo ciclo de aceite

---

## 11. Fora de Escopo

- Política de Privacidade (PRD separado — `prd-politica-de-privacidade.md`)
- Notificação por WhatsApp sobre atualização de termos
- Exportação de comprovante de aceite
- Histórico de versões anteriores visível ao usuário

---

## 12. Recomendação pós-Beta

Antes do lançamento dos planos pagos, revisão jurídica pontual para:
- Validar a cláusula de limitação de responsabilidade
- Adaptar o foro e cláusulas ao modelo comercial (B2C)
- Incluir cláusulas específicas de assinatura e reembolso

---

*Documento elaborado para uso interno da equipe de produto e engenharia do Rachao.app.*
