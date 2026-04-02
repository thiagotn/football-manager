# PRD — Adição Manual de Jogador pelo Admin do Grupo
## Rachao.app · Admin cria conta + adiciona ao grupo sem link de convite

| | |
|---|---|
| **Versão** | 1.1 |
| **Status** | 📋 Proposto — decisões de produto confirmadas |
| **Data** | Abril de 2026 |
| **Plataforma** | https://rachao.app |

---

## 1. Contexto e Motivação

Hoje, para incluir um novo jogador em um grupo, o admin precisa:
1. Gerar um link de convite.
2. Enviar o link para o jogador via WhatsApp ou outro canal.
3. Aguardar o jogador clicar, preencher o cadastro e confirmar presença.

Esse fluxo tem dois problemas práticos:
- **Jogadores que não se cadastram**: o link é enviado, mas muitos jogadores deixam para depois e nunca completam o fluxo.
- **Atrito desnecessário quando o admin já tem os dados**: em grupos de várzea o presidente já sabe o nome e o WhatsApp de cada jogador — ele quer apenas cadastrá-los sem depender de ação deles.

Esta feature permite que o admin adicione um jogador diretamente, preenchendo nome e WhatsApp. Se o jogador já tem conta, é adicionado ao grupo imediatamente. Se não tem, uma conta é criada pelo admin e o jogador recebe instruções para ativar o acesso na primeira vez que quiser usar o app.

---

## 2. Dois Cenários de Uso

### Cenário A — Jogador já registrado na plataforma

O admin digita o número de WhatsApp e o sistema encontra uma conta existente. O admin confirma a adição com 1 clique, sem nenhum preenchimento adicional.

### Cenário B — Jogador sem conta

O admin digita o WhatsApp (não encontrado no sistema), preenche nome e apelido opcional, e confirma. O sistema cria a conta e adiciona ao grupo. O jogador precisa ativar o acesso pelo fluxo de "esqueci minha senha" quando quiser acessar o app.

---

## 3. Requisitos Funcionais

### RF-01 — Entrada no fluxo

Na página do grupo (`/groups/[id]`), na aba "Jogadores", o admin vê o botão existente de adicionar membro. Este botão é expandido para oferecer a nova opção **"Adicionar por dados"** (em adição ao convite por link que já existe).

Ao clicar, abre um modal de duas etapas.

---

### RF-02 — Etapa 1: busca por WhatsApp

O modal abre com um único campo:

- **WhatsApp** — input com seletor de país (componente `PhoneInput` existente, validação E.164).

Ao preencher e avançar, o frontend consulta:

```
GET /api/v1/groups/{group_id}/members/lookup?whatsapp={numero_e164}
```

Três resultados possíveis:

| Situação | Resposta da API | Comportamento no modal |
|---|---|---|
| Jogador encontrado e já é membro do grupo | `{ status: "already_member", player: {...} }` | Exibe aviso e bloqueia avanço |
| Jogador encontrado, não é membro | `{ status: "found", player: { name, nickname, avatar_url } }` | Vai para Etapa 2-A (confirmação) |
| Número não encontrado | `{ status: "not_found" }` | Vai para Etapa 2-B (dados do novo jogador) |

---

### RF-03 — Etapa 2-A: confirmação (jogador existente)

Exibe o card do jogador encontrado (nome, apelido, avatar se houver) com a mensagem: *"Este jogador já tem conta no rachao.app. Confirmar adição ao grupo?"*

Campos opcionais antes de confirmar:
- **Estrelas** (1–5, padrão 2) — `skill_stars` no `group_members`
- **Goleiro** (toggle) — `is_goalkeeper` no `group_members`

Botão **"Adicionar ao grupo"** → chama `POST /api/v1/groups/{group_id}/members/by-phone`.

---

### RF-04 — Etapa 2-B: dados do novo jogador

Formulário com:

| Campo | Obrigatório | Validação |
|---|---|---|
| Nome completo | Sim | 2–100 caracteres |
| Apelido | Não | máx. 50 caracteres |
| Estrelas (1–5) | Não | padrão 2 |
| Goleiro (toggle) | Não | padrão false |

O WhatsApp já foi preenchido na etapa anterior e é exibido como read-only para confirmação.

Botão **"Criar e adicionar"** → chama `POST /api/v1/groups/{group_id}/members/by-phone`.

---

### RF-05 — Endpoint: lookup de jogador por WhatsApp

**`GET /api/v1/groups/{group_id}/members/lookup`**

Query param: `whatsapp` (string E.164 — ex: `+5511999990000`)

- Acessível apenas pelo admin do grupo ou super admin.
- Normaliza o número antes de buscar (remove caracteres não numéricos além do `+`).
- Não expõe dados sensíveis (sem `password_hash`, sem `email`).

Response:
```json
{
  "status": "found" | "not_found" | "already_member",
  "player": {
    "id": "uuid",
    "name": "Carlos Silva",
    "nickname": "Carlão",
    "avatar_url": null
  }
}
```
O campo `player` é omitido quando `status = "not_found"`.

---

### RF-06 — Endpoint: adicionar membro por telefone

**`POST /api/v1/groups/{group_id}/members/by-phone`**

Request:
```json
{
  "whatsapp": "+5511999990000",
  "name": "Carlos Silva",
  "nickname": "Carlão",
  "skill_stars": 3,
  "is_goalkeeper": false
}
```

- `name` é obrigatório apenas quando o jogador não existe (sistema valida internamente).
- Se o jogador existe: `name` e `nickname` são ignorados (não atualiza dados do jogador).
- Apenas admin do grupo ou super admin pode chamar.

**Lógica interna:**

```
1. Buscar player por whatsapp normalizado
2. Se encontrado:
   a. Se já é membro → ConflictError("Jogador já é membro deste grupo")
   b. Se super admin → ForbiddenError("Super admin não pode ser adicionado como membro")
   c. Verificar limite de plano
   d. add_member(group_id, player.id, MEMBER, skill_stars, is_goalkeeper)
   e. Adicionar como PENDING nas partidas abertas do grupo
   f. Garantir período financeiro do mês corrente
   g. Retornar { member, is_new: false }

3. Se não encontrado:
   a. Validar name (obrigatório)
   b. Verificar limite de plano
   c. Gerar senha temporária aleatória (16 chars, alfanumérico)
   d. player = create(name, nickname, whatsapp, hash(temp_password), must_change_password=True)
   e. sub_repo.get_or_create(player.id)   ← subscription gratuita
   f. add_member(group_id, player.id, MEMBER, skill_stars, is_goalkeeper)
   g. Adicionar como PENDING nas partidas abertas do grupo
   h. Garantir período financeiro do mês corrente
   i. Retornar { member, is_new: true }
```

Response:
```json
{
  "member": {
    "player_id": "uuid",
    "name": "Carlos Silva",
    "nickname": "Carlão",
    "role": "member",
    "skill_stars": 3,
    "is_goalkeeper": false
  },
  "is_new": true
}
```

---

### RF-07 — Feedback de sucesso

Após a chamada bem-sucedida, o modal exibe:

**Jogador existente (`is_new: false`):**
> ✅ **Carlos Silva** adicionado ao grupo.

**Jogador novo (`is_new: true`):**
> ✅ **Carlos Silva** adicionado ao grupo.
>
> Como é o primeiro acesso dele no rachao.app, peça que ele abra o app, informe o número `(11) 99999-0000` e clique em **"Esqueci minha senha"** para criar a própria senha.

O texto do número é exibido formatado (não E.164) para facilitar a comunicação verbal.

> **Decisão confirmada:** o fluxo de ativação de conta para novos jogadores é o "Esqueci minha senha" (OTP existente). Não é necessário link de primeiro acesso dedicado.

---

### RF-08 — Verificação de limite de plano

O mesmo limite de membros aplicado ao fluxo de convite (`PlanLimitError` / `"PLAN_LIMIT_EXCEEDED"`) se aplica aqui. Super admins são isentos.

---

### RF-09 — Atualização da lista de membros

Após o fechamento do modal com sucesso, a lista de membros do grupo é atualizada (re-fetch ou inserção otimista) sem necessidade de recarregar a página.

---

## 4. Requisitos Não-Funcionais

- **Senha temporária**: gerada no backend (nunca exposta na resposta). Comprimento mínimo de 12 caracteres, alfanumérico.
- **`must_change_password = True`**: garante que o jogador seja obrigado a trocar a senha no primeiro login pelo fluxo normal de OTP/redefinição.
- **Idempotência no lookup**: não modifica nenhum dado — apenas consulta.
- **Normalização de WhatsApp**: ambos os endpoints normalizam o número antes de qualquer operação (remover espaços, traços, parênteses; garantir prefixo `+`).
- **Segurança**: o lookup não pode ser usado para enumerar usuários por WhatsApp por pessoas não autorizadas — requer autenticação como admin do grupo.

---

## 5. Modelagem de Dados

Nenhuma migration necessária. A feature reutiliza:
- Tabela `players` — criação de novo player com `must_change_password = True`.
- Tabela `group_members` — com `skill_stars` e `is_goalkeeper` já existentes (PRD 012).
- Tabela `attendances` — adição como `PENDING` nas partidas abertas.
- Tabela `finance_periods` / `finance_member_periods` — período financeiro corrente.

---

## 6. Endpoints da API

| Método | Rota | Auth | Descrição |
|---|---|---|---|
| `GET` | `/api/v1/groups/{group_id}/members/lookup` | Admin do grupo | Busca jogador por WhatsApp |
| `POST` | `/api/v1/groups/{group_id}/members/by-phone` | Admin do grupo | Cria ou encontra jogador e adiciona ao grupo |

Os endpoints existentes (`POST /members` por `player_id`) permanecem inalterados.

---

## 7. Alterações de Frontend

### 7.1 `/groups/[id]` — aba Jogadores

O botão **"Adicionar Membro"** existente (visível apenas para admin do grupo) é **substituído** pelo novo fluxo de adição por dados. O fluxo de convite por link é mantido apenas para o caso de uso diferente (convidar alguém que ainda não tem o WhatsApp cadastrado no app), acessível por outro caminho se já existir.

> **Decisão confirmada:** o botão "Adicionar > Adicionar Membro" na visão do admin do grupo é substituído inteiramente por este novo modal. O fluxo anterior (que exigia `player_id`) é descontinuado no frontend para admins de grupo.

### 7.2 Modal "Adicionar por dados"

Componente novo ou reutilização de `Modal.svelte` com conteúdo multi-etapa:

```
Etapa 1 (sempre)
┌────────────────────────────────────────────┐
│  Adicionar jogador                     [×] │
│                                            │
│  WhatsApp do jogador                       │
│  [PhoneInput ................................]│
│                                            │
│            [Buscar]                        │
└────────────────────────────────────────────┘

Etapa 2-A (jogador encontrado)
┌────────────────────────────────────────────┐
│  Jogador encontrado                        │
│  [Avatar] Carlos Silva (Carlão)            │
│  Já tem conta no rachao.app               │
│                                            │
│  Estrelas: [★★☆☆☆]                        │
│  Goleiro:  [toggle]                        │
│                                            │
│  [← Voltar]     [Adicionar ao grupo]       │
└────────────────────────────────────────────┘

Etapa 2-B (jogador não encontrado)
┌────────────────────────────────────────────┐
│  Novo jogador                              │
│  WhatsApp: +55 11 99999-0000 (read-only)   │
│                                            │
│  Nome completo *                           │
│  [.................................]        │
│  Apelido                                   │
│  [.................................]        │
│  Estrelas: [★★☆☆☆]                        │
│  Goleiro:  [toggle]                        │
│                                            │
│  [← Voltar]     [Criar e adicionar]        │
└────────────────────────────────────────────┘
```

- Em mobile: modal vira bottom sheet (padrão do projeto).
- Loading state no botão de ação durante a chamada à API.
- Exibir erro inline se `ConflictError` ou `PlanLimitError`.

---

## 8. Levantamento de Impacto por Camada

| Camada | Arquivo / Área | Tipo de mudança |
|---|---|---|
| **Backend router** | `app/api/v1/routers/groups.py` | +2 endpoints (`lookup`, `by-phone`) |
| **Backend schema** | `app/schemas/group.py` | `AddMemberByPhoneRequest`, `LookupResponse`, `AddMemberByPhoneResponse` |
| **Backend repo** | `app/db/repositories/group_repo.py` | Reuso do `add_member` existente |
| **Backend repo** | `app/db/repositories/player_repo.py` | Reuso do `get_by_whatsapp` e `create` existentes |
| **Frontend** | `src/routes/groups/[id]/+page.svelte` | Novo botão + lógica de abertura do modal |
| **Frontend** | Novo `src/lib/components/AddMemberModal.svelte` | Modal multi-etapa |
| **Frontend** | `src/lib/api.ts` — namespace `groups` | +2 chamadas (`lookupMember`, `addMemberByPhone`) |
| **i18n** | `messages/pt-BR.json`, `en.json`, `es.json` | Chaves `groups.add_manual.*` |

---

## 9. Testes Unitários (Backend)

Cobrir no mínimo:

| Caso | Resultado esperado |
|---|---|
| Lookup — número encontrado, não é membro | `status: found` + dados do player |
| Lookup — número encontrado, já é membro | `status: already_member` |
| Lookup — número não encontrado | `status: not_found` |
| Lookup — caller não é admin do grupo | 403 |
| Add by phone — player existente, não membro | 201, `is_new: false` |
| Add by phone — player existente, já membro | 409 |
| Add by phone — player novo, name preenchido | 201, `is_new: true`, `must_change_password: true` |
| Add by phone — player novo, sem name | 422 |
| Add by phone — limite de plano atingido | 403 `PLAN_LIMIT_EXCEEDED` |
| Add by phone — caller não é admin do grupo | 403 |

---

## 10. Critérios de Aceitação

- [ ] Admin abre o modal "Adicionar por dados" na aba Jogadores do grupo.
- [ ] Ao digitar um número já registrado, o sistema exibe o nome/apelido do jogador antes de confirmar.
- [ ] Ao digitar um número já registrado que já é membro, o sistema bloqueia o avanço com mensagem clara.
- [ ] Ao digitar um número não registrado, o admin vê o formulário de criação de conta.
- [ ] Confirmar com jogador existente adiciona ao grupo e atualiza a lista sem reload.
- [ ] Confirmar com jogador novo cria a conta com `must_change_password = True` e adiciona ao grupo.
- [ ] Após criar jogador novo, o admin vê instruções claras sobre como o jogador deve criar sua senha.
- [ ] O jogador novo aparece como `PENDING` nas partidas abertas do grupo imediatamente.
- [ ] O limite de plano é verificado e retorna erro amigável quando excedido.
- [ ] `skill_stars` e `is_goalkeeper` definidos pelo admin são salvos corretamente no `group_members`.
- [ ] Em mobile, o modal é exibido como bottom sheet.
- [ ] Todos os textos do modal usam chaves i18n.

---

## 11. Fora de Escopo (v1)

- Envio automático de notificação WhatsApp/SMS ao jogador criado (aguarda canal Twilio aprovado).
- Edição dos dados do jogador pelo admin no ato da criação (nome incorreto após criado requer fluxo de edição separado).
- Adição em lote (múltiplos jogadores de uma vez).
- Definição de senha temporária pelo admin (a senha é sempre gerada automaticamente).
- Busca por nome em vez de WhatsApp.

---

*Documento elaborado para uso interno da equipe de produto e engenharia do Rachao.app.*
