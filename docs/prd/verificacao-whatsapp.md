# PRD — Verificação de Número WhatsApp no Cadastro
## Rachao.app · Gerenciamento de Grupos e Partidas

| | |
|---|---|
| **Versão** | 1.1 |
| **Status** | Draft — decisão de gateway pendente |
| **Data** | Março de 2026 |
| **Plataforma** | https://rachao.app |

---

## 1. Visão Geral

### 1.1 Contexto

O fluxo de auto-cadastro (`POST /api/v1/auth/register`, rota `/register`) permite que qualquer pessoa informe qualquer número de WhatsApp, sem confirmar que é de fato a dona do número. O número de WhatsApp é o identificador único de login na plataforma, o que torna sua validação crítica.

### 1.2 Problema

- Um usuário pode cadastrar o número de WhatsApp de outra pessoa.
- Contas duplicadas ou fraudulentas comprometem a integridade dos grupos.
- Sem verificação, não há garantia de que o número está ativo no WhatsApp — o que é relevante para o uso de convites e futuras notificações via WhatsApp.

### 1.3 Objetivo

Implementar verificação de número via OTP (One-Time Password) no fluxo de cadastro, garantindo que o usuário é dono do número informado antes de criar a conta.

---

## 2. Fluxo Proposto

O cadastro passa a ter **duas etapas**:

### Etapa 1 — Preenchimento do formulário
O usuário preenche nome, apelido (opcional), WhatsApp e senha na rota `/register` e clica em **"Enviar código de verificação"**.

### Etapa 2 — Verificação do OTP
O backend envia um código de 6 dígitos para o número informado (via WhatsApp ou SMS). O usuário digita o código recebido na tela. Após validação, a conta é criada e o usuário é logado automaticamente.

```
[/register — Etapa 1]
  Usuário preenche: nome, WhatsApp, senha
  Clica: "Enviar código"
       ↓
[POST /auth/send-otp]
  Backend gera OTP (6 dígitos, TTL 10 min)
  Envia via gateway (WhatsApp ou SMS)
  Retorna: { status: "pending" }
       ↓
[/register — Etapa 2]
  Usuário digita o código recebido
  Clica: "Confirmar e criar conta"
       ↓
[POST /auth/register]  ← agora requer otp_code no body
  Backend valida OTP
  Cria player + subscription gratuita
  Retorna: JWT (login imediato)
```

> **Nota de segurança:** o OTP deve ser validado no backend antes de qualquer criação de registro. O frontend nunca deve confiar na validação local do código.

---

## 3. Requisitos Funcionais

**RF-01 — Envio de OTP**
O sistema deve enviar um código de 6 dígitos para o número de WhatsApp informado. O código deve expirar em 10 minutos e ser invalidado após uso.

**RF-02 — Validação de OTP no registro**
O endpoint `POST /auth/register` deve receber e validar o OTP antes de criar o player. Se inválido ou expirado, retornar erro `422`.

**RF-03 — Reenvio de código**
O usuário deve poder solicitar o reenvio do código após 60 segundos (cooldown). Máximo de 3 tentativas por número por hora (rate limiting).

**RF-04 — Tentativas inválidas**
Após 5 tentativas incorretas do código, o OTP deve ser invalidado e o usuário deve solicitar um novo envio.

**RF-05 — Feedback visual**
A interface deve exibir:
- Confirmação de envio com os últimos 4 dígitos do número mascarado (ex: `••• ••••• 9000`)
- Contador regressivo para o reenvio
- Mensagem de erro clara em caso de código inválido ou expirado

**RF-06 — Verificação apenas no auto-cadastro**
A verificação de OTP se aplica exclusivamente ao fluxo de `/register`. O fluxo de convite (`/invite/[token]`) e o cadastro via painel admin (`POST /players`) **não requerem OTP** — o convite já é um mecanismo implícito de verificação de acesso.

---

## 4. Requisitos Não Funcionais

**RNF-01 — TTL do OTP:** máximo de 10 minutos.
**RNF-02 — Rate limiting:** máximo de 3 envios por número por hora; máximo de 5 tentativas de validação por OTP.
**RNF-03 — Armazenamento seguro:** o código deve ser armazenado hasheado (bcrypt ou SHA-256 + salt) — nunca em texto puro.
**RNF-04 — Não bloquear número legítimo:** erros de gateway não devem impedir o cadastro indefinidamente — prever fallback ou mensagem de suporte.

---

## 5. Modelagem de Dados

```sql
-- Migration: 016_otp_verifications.sql
CREATE TABLE otp_verifications (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    whatsapp     VARCHAR(20) NOT NULL,
    code_hash    VARCHAR(255) NOT NULL,       -- SHA-256 do código
    expires_at   TIMESTAMPTZ NOT NULL,
    attempts     INT NOT NULL DEFAULT 0,
    verified     BOOLEAN NOT NULL DEFAULT FALSE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_otp_whatsapp ON otp_verifications (whatsapp, expires_at);
```

> **Nota:** se a decisão for pelo Twilio Verify (Opção A), esta tabela **não é necessária** — a Twilio gerencia o estado do OTP internamente. A migration 016 só é necessária nas opções B ou C.

---

## 6. Endpoints da API

| Método | Endpoint | Descrição |
|---|---|---|
| `POST` | `/api/v1/auth/send-otp` | Gera e envia OTP para o número informado |
| `POST` | `/api/v1/auth/register` | Agora requer `otp_code` no body |

### 6.1 `POST /auth/send-otp`

**Request:**
```json
{ "whatsapp": "11999990000" }
```

**Response 200:**
```json
{ "status": "pending", "expires_in_seconds": 600 }
```

**Erros:**
- `409` — número já cadastrado
- `429` — rate limit atingido

### 6.2 `POST /auth/register` (atualizado)

**Request (atualizado):**
```json
{
  "name": "João Silva",
  "whatsapp": "11999990000",
  "password": "senha123",
  "nickname": "Joãozinho",
  "otp_code": "483920"
}
```

**Erros adicionais:**
- `422 OTP_INVALID` — código incorreto
- `422 OTP_EXPIRED` — código expirado
- `422 OTP_MAX_ATTEMPTS` — tentativas esgotadas

---

## 7. Opções de Gateway

> ⚠️ **Decisão pendente.** As opções abaixo foram avaliadas. A escolha deve ser registrada neste PRD antes da implementação.

---

### Opção A — Twilio Verify API
**Status:** ⏳ Pendente de decisão

| Critério | Avaliação |
|---|---|
| Canal principal | WhatsApp Business |
| Fallback automático | ✅ SMS se WhatsApp falhar |
| Custo por verificação (Brasil) | ~US$ 0,07–0,08 (US$ 0,05 verificação + ~US$ 0,02–0,03 SMS Brasil) |
| Gerenciamento do OTP | ✅ Feito pela Twilio (sem tabela local necessária) |
| SDK Python | ✅ `twilio` — maduro e bem documentado |
| Burocracia | Moderada — requer conta Twilio e aprovação de template WhatsApp |
| Confiabilidade | Alta |
| Recomendação | ✅ Melhor custo-benefício para começar |

**Integração:**
```python
from twilio.rest import Client

client = Client(TWILIO_ACCOUNT_SID, TWILIO_AUTH_TOKEN)

# Enviar OTP
client.verify.v2.services(TWILIO_VERIFY_SID) \
    .verifications.create(to=f"+55{whatsapp}", channel="whatsapp")

# Validar OTP
check = client.verify.v2.services(TWILIO_VERIFY_SID) \
    .verification_checks.create(to=f"+55{whatsapp}", code=otp_code)

if check.status != "approved":
    raise ValidationError("OTP_INVALID")
```

> Com Twilio Verify, **a tabela `otp_verifications` não é necessária** — a Twilio gerencia o estado do OTP internamente. A migration 016 só seria necessária nas opções B ou C.

---

### Opção B — Meta Cloud API (WhatsApp Business direto)
**Status:** ⏳ Pendente de decisão

| Critério | Avaliação |
|---|---|
| Canal principal | WhatsApp Business (oficial) |
| Fallback automático | ❌ Não incluso — requer implementação manual |
| Custo por mensagem (Brasil) | Gratuito até 1.000 conversas/mês; ~US$ 0,008/msg após |
| Gerenciamento do OTP | ❌ Manual — requer tabela `otp_verifications` |
| Burocracia | Alta — verificação de empresa Meta, aprovação de template |
| Confiabilidade | Alta (canal oficial) |
| Recomendação | Indicado quando o volume crescer e justificar o onboarding |

---

### Opção C — SMS puro (Twilio SMS / AWS SNS / Sinch)
**Status:** ⏳ Pendente de decisão

| Critério | Avaliação |
|---|---|
| Canal principal | SMS |
| Fallback automático | N/A |
| Custo por SMS (Brasil) | ~US$ 0,02–0,03 |
| Gerenciamento do OTP | ❌ Manual — requer tabela `otp_verifications` |
| Burocracia | Baixa — sem aprovação de template |
| Confirmação de WhatsApp ativo | ❌ Não confirma — apenas que o número existe |
| Recomendação | Opção mais simples; não confirma que o número está no WhatsApp |

---

### Opção D — Z-API / Evolution API ❌ Não recomendado
**Status:** Descartado

Gateways não-oficiais que conectam uma instância real do WhatsApp.
Violam os Termos de Serviço do WhatsApp — conta pode ser banida sem aviso.
**Não recomendado para uso em produção.**

---

## 8. Comparativo das Opções

| Critério | A — Twilio Verify | B — Meta Cloud API | C — SMS puro |
|---|:---:|:---:|:---:|
| Confirma WhatsApp ativo | ✅ | ✅ | ❌ |
| Fallback automático | ✅ | ❌ | N/A |
| Custo por envio (Brasil) | ~US$ 0,075 | ~US$ 0,008 (em volume) | ~US$ 0,025 |
| Gerenciamento de OTP | Twilio | Manual | Manual |
| Facilidade de integração | Alta | Baixa | Alta |
| Burocracia de setup | Moderada | Alta | Baixa |
| **Recomendado para** | **MVP / início** | **Escala** | **Simplicidade** |

---

## 9. Análise de Custos por Volume de Cadastros

O OTP é disparado **apenas no auto-cadastro** (`POST /auth/register`), portanto o custo é diretamente proporcional ao número de novos cadastros por mês — não ao número de usuários ativos. Considerando um multiplicador de **1,15x** para reenvios (RF-03 limita a 3 por número por hora, resultando em ~10–15% de reenvios na prática).

> **Câmbio de referência:** US$ 1 ≈ R$ 5,80

### 9.1 Custo estimado por provedor (para o Brasil)

| Provedor | Modelo | Custo por OTP (Brasil) | Composição |
|---|---|:---:|---|
| **Twilio Verify** (Opção A) | Por verificação bem-sucedida | ~US$ 0,075 | US$ 0,05 (verificação) + ~US$ 0,025 (SMS Brasil) |
| **SMS puro — AWS SNS** (Opção C) | Por mensagem enviada | ~US$ 0,025 | Taxa base AWS + carrier fee Brasil |
| **SMS puro — Twilio SMS** (Opção C) | Por mensagem enviada | ~US$ 0,025–0,030 | Carrier fee Brasil inclusa |
| **SMS puro — Sinch** (Opção C) | Por mensagem enviada | ~US$ 0,020–0,025 | Ligeiramente mais barato que Twilio |
| **Meta Cloud API** (Opção B) | Por conversa | Grátis até 1.000/mês | ~US$ 0,008/msg acima do limite |

### 9.2 Projeções — Opção A (Twilio Verify, ~US$ 0,075/cadastro)

| Fase | Cadastros/mês | Custo USD/mês | Custo BRL/mês | Custo BRL/ano |
|---|:---:|:---:|:---:|:---:|
| MVP / início | 100 | ~US$ 8,63 | ~R$ 50 | ~R$ 600 |
| Crescimento | 500 | ~US$ 43,13 | ~R$ 250 | ~R$ 3.000 |
| Tração | 2.000 | ~US$ 172,50 | ~R$ 1.000 | ~R$ 12.000 |
| Escala | 10.000 | ~US$ 862,50 | ~R$ 5.000 | ~R$ 60.000 |

### 9.3 Projeções — Opção C (SMS puro AWS/Twilio, ~US$ 0,025/cadastro)

| Fase | Cadastros/mês | Custo USD/mês | Custo BRL/mês | Custo BRL/ano |
|---|:---:|:---:|:---:|:---:|
| MVP / início | 100 | ~US$ 2,88 | ~R$ 17 | ~R$ 200 |
| Crescimento | 500 | ~US$ 14,38 | ~R$ 83 | ~R$ 1.000 |
| Tração | 2.000 | ~US$ 57,50 | ~R$ 333 | ~R$ 4.000 |
| Escala | 10.000 | ~US$ 287,50 | ~R$ 1.667 | ~R$ 20.000 |

### 9.4 Interpretação

**Curto prazo (< 500 cadastros/mês):** o custo é irrelevante em qualquer opção — menos de R$ 300/mês mesmo no Twilio Verify. Não é um critério de decisão agora.

**Médio prazo (500–2.000 cadastros/mês):** a diferença entre Verify (~R$ 1.000/mês) e SMS puro (~R$ 333/mês) começa a aparecer, mas ainda é facilmente absorvida com poucos assinantes do plano Básico (R$ 19,90/mês cada).

**Escala (> 5.000 cadastros/mês):** o Twilio Verify oferece descontos por volume a partir desse patamar, e a Meta Cloud API (Opção B) passa a ser economicamente vantajosa. Vale reavaliação nesse momento.

**O custo real de atenção** só ocorre com dezenas de milhares de cadastros mensais — cenário no qual o Rachao já terá receita suficiente para absorver ou migrar de gateway.

### 9.5 Critérios de decisão recomendados (estágio atual)

O custo **não é o critério principal** no estágio atual. Os fatores que importam agora são:

| Critério | Twilio Verify (A) | SMS puro (C) |
|---|:---:|:---:|
| Velocidade de implementação | ✅ Sem tabela local, sem TTL manual | ❌ Requer tabela + hash + rate limit |
| Confirma número no WhatsApp | ✅ Via canal WhatsApp | ❌ Apenas que o número existe |
| Fallback automático WhatsApp → SMS | ✅ | ❌ |
| Complexidade de manutenção | Baixa | Média |
| Custo no MVP | ~R$ 50/mês | ~R$ 17/mês |

**Recomendação:** usar **Twilio Verify** no MVP e reavaliar para Meta Cloud API quando o volume de cadastros ultrapassar 5.000/mês e a operação justificar o processo de onboarding da Meta.

---

## 10. Interface do Usuário

### Etapa 1 — Formulário de cadastro (atual `/register`)
Sem alterações visuais significativas. O botão "Criar conta grátis" passa a ser "Enviar código de verificação".

### Etapa 2 — Verificação do OTP (nova tela / mesmo componente)

```
┌─────────────────────────────────────────┐
│  Verifique seu WhatsApp                 │
│─────────────────────────────────────────│
│  Enviamos um código para                │
│  ••• ••••• 9000                         │
│                                         │
│  [ _ ] [ _ ] [ _ ] [ _ ] [ _ ] [ _ ]   │
│          (6 dígitos)                    │
│                                         │
│  ⏱ Código válido por 10 minutos         │
│                                         │
│  [ Confirmar e criar conta ]            │
│                                         │
│  Não recebeu? Reenviar em 0:45          │
└─────────────────────────────────────────┘
```

---

## 11. Critérios de Aceitação

- [ ] Usuário não consegue criar conta sem inserir OTP válido
- [ ] OTP expira após 10 minutos
- [ ] OTP é invalidado após uso
- [ ] Após 5 tentativas inválidas, OTP é bloqueado
- [ ] Rate limit de 3 envios por número por hora funciona corretamente
- [ ] Reenvio disponível após 60 segundos
- [ ] Número já cadastrado retorna erro 409 no `send-otp` (antes de cobrar o envio)
- [ ] Fluxo de convite e cadastro admin **não** são afetados

---

## 12. Fora de Escopo

- Verificação de número para login (apenas para cadastro)
- Verificação de números já cadastrados (usuários existentes não precisam re-verificar)
- Autenticação por WhatsApp (login via OTP — feature separada)

---

*Documento elaborado para uso interno da equipe de produto e engenharia do Rachao.app.*
