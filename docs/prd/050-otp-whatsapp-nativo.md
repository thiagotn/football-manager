# PRD 050 — OTP nativo via WhatsApp Cloud API (substituição da Twilio)

**Status**: 📋 Proposto — aguardando decisão (pré-requisitos manuais na Meta + decisão de CNPJ)

| | |
|---|---|
| **Versão** | 1.0 |
| **Data** | 2026-07-17 |
| **Substitui** | PRD 003 (Twilio Verify) quando implementado |
| **Relacionados** | PRD 025 (bypass), PRD 026 (autofill), PRD 008 (privacidade/LGPD), PRD 044 §17 (paridade v1/v2) |

---

## 1. Contexto e motivação

O OTP de 6 dígitos (cadastro, recuperação de senha e troca de senha) usa **Twilio Verify v2**
nas duas APIs. Fato levantado na auditoria de 2026-07-17: apesar dos docstrings dizerem
"WhatsApp channel", **o canal real é `channel="sms"`** — a migração para o canal WhatsApp da
Twilio nunca foi concluída (PRD 003 v1.4, pendente de aprovação de template). Ou seja, hoje
pagamos o modelo mais caro (US$0,05/verificação de plataforma + tarifa SMS ≈ **R$0,30–0,45
por OTP**) para entregar por SMS um código de um produto cujo identificador de login já é o
número de WhatsApp.

A integração nativa com a **WhatsApp Business Cloud API** (Meta, oficial) custa
**~R$0,025 por mensagem de autenticação no Brasil** (modelo por mensagem vigente desde
jul/2025; tiers: R$0,025 → R$0,018 acima de 10K/mês → R$0,015 acima de 100K/mês) —
**redução de ~90%+**. Como OTP dispara só em cadastro/recuperação, o custo é proporcional a
novos cadastros, não a usuários ativos.

### Superfície atual (pequena e isolada)

- **v1**: `football-api/app/services/twilio_verify.py` (`send_otp`/`check_otp` via
  `asyncio.to_thread`); chamado por 8 endpoints em `app/api/v1/routers/auth.py`
  (send/verify-otp de cadastro, forgot-password, send/verify-otp/me).
- **v2**: client Twilio inline em `football-api-go/internal/services/auth_service.go`
  (`sendOTPToNumber`/`checkOTP`); `internal/services/twilio.go` é **dead code** (nunca
  instanciado). Divergência de env no `.env.example` (`TWILIO_VERIFY_SERVICE_SID` vs
  `TWILIO_VERIFY_SID` lido pelo config).
- **Envs**: `TWILIO_ACCOUNT_SID`, `TWILIO_AUTH_TOKEN`, `TWILIO_VERIFY_SID`,
  `OTP_BYPASS_CODE` (bypass dev — PRD 025).
- **Estado do OTP**: 100% na Twilio (geração, TTL, tentativas, rate-limit). A integração
  nativa precisa recriar essa camada — base já esboçada no PRD 003 §5.

## 2. Os caminhos avaliados

| # | Caminho | Custo/OTP (BR) | Esforço | Risco | Veredito |
|---|---------|----------------|---------|-------|----------|
| 1 | **Meta WhatsApp Cloud API direta** | ~R$0,025/msg (cai com volume) | Médio-alto (estado de OTP próprio + setup Meta) | Baixo (oficial) | ✅ **Recomendado** |
| 2 | BSP intermediário (360dialog, Infobip, Zenvia…) | Meta + markup (~US$0,003–0,010/msg) e/ou mensalidade | Médio | Baixo | Só se o setup Meta travar |
| 3 | Twilio Verify com `channel="whatsapp"` | US$0,05/verif + template Meta (~R$0,30+) | Trivial (1 linha + template) | Baixo | Não resolve custo; "plano B relâmpago" |
| 4 | APIs não-oficiais (Z-API, Evolution, Baileys) | ~R$0,05–0,10/msg | Baixo | **Viola ToS; banimento do número** | ❌ Descartado (PRD 003 §Opção D) |

Decisões já tomadas: **manter Twilio SMS como fallback temporário** durante o rollout;
CNPJ para verificação Meta **a decidir** (§3 Fase 0 cobre os dois cenários).

## 3. Caminho recomendado em detalhe — Meta WhatsApp Cloud API direta

### Fase 0 — Pré-requisitos manuais (usuário, fora do código; ~3–5 dias úteis)

1. **Meta Business Portfolio** (business.facebook.com) — criar/usar existente.
2. **Decisão CNPJ**:
   - **Com CNPJ** → business verification (2–4 dias): limites escalam 1k→10k→100k
     conversas/dia conforme qualidade; display name "rachao.app" aprovado.
   - **Sem CNPJ** → operar não-verificado: teto de **250 conversas iniciadas/24h**
     (~250 OTPs/dia — suficiente para o volume atual; gargalo só num pico de cadastros).
     Dá para começar assim e verificar depois.
3. **Número de telefone dedicado** para a WABA — **não pode estar ativo no app WhatsApp**
   (número novo/virtual, ou deletar a conta do app antes de registrar). É o remetente dos
   OTPs; sem relação com os números dos usuários.
4. **App Meta** (developers.facebook.com, tipo Business) com o produto WhatsApp; anotar
   `WABA_ID` e `PHONE_NUMBER_ID`.
5. **System User + token permanente** (escopos `whatsapp_business_messaging`,
   `whatsapp_business_management`) — nunca o token temporário de 24h do painel.
6. **Template AUTHENTICATION** (ex.: `otp_login`, idiomas `pt_BR`, `en`, `es`): corpo fixo
   da Meta ("<código> é seu código de verificação"), botão **copy code** (one-tap fica para
   o PRD 026; exige assinatura do app Android), footer com expiração (10 min). Templates de
   autenticação são **auto-aprovados**.
7. **Webhook**: URL pública `POST /api/v1/webhooks/whatsapp` (GET de verificação com
   `hub.challenge`) para status de entrega — insumo do fallback.

### Fase 1 — Backend (regra dual: v1 Python E v2 Go no mesmo PR)

**Nova camada de estado do OTP:**

- **Migration `NNN_otp_verifications.sql`** (próximo número em `football-api/CLAUDE.md`;
  hoje seria `048_`): tabela `otp_verifications(id, whatsapp, code_hash, attempts,
  max_attempts DEFAULT 5, expires_at, verified_at, channel, created_at)` + índice
  `(whatsapp, created_at)`. Idempotente (`IF NOT EXISTS`). Banco compartilhado v1/v2.
- **Regras**: código de 6 dígitos criptograficamente aleatório; armazenar **hash SHA-256**
  (nunca o código em claro); TTL 10 min; máx. 5 tentativas (`OTP_MAX_ATTEMPTS`); cooldown
  de reenvio 60s; rate-limit por número (5 envios/hora) e por IP. Erros mapeados para os
  códigos que o frontend já trata: `OTP_INVALID`, `OTP_EXPIRED`, `OTP_MAX_ATTEMPTS` —
  **o contrato dos endpoints não muda; frontend intocado no fluxo**.

**v1 Python** (`football-api/`):
- Novo `app/services/whatsapp_otp.py`: `send_otp(db, whatsapp)` → gera código, grava hash,
  chama Graph API `POST /{PHONE_NUMBER_ID}/messages` com o template no idioma do usuário;
  `check_otp(db, whatsapp, code)` → valida hash/TTL/tentativas. HTTP via `httpx`
  (dependência existente), timeout explícito.
- **Orquestrador**: `app/services/otp.py` com `OTP_PROVIDER=whatsapp|twilio_sms` e fallback
  **no resend** (não automático em background): falha síncrona da Graph API ou status
  `failed` no webhook → o botão "reenviar" do usuário dispara SMS via Twilio. Bypass
  `OTP_BYPASS_CODE` preservado.
- Webhook em `app/api/v1/routers/webhooks.py` (GET challenge + POST status), validando
  `X-Hub-Signature-256` com o `APP_SECRET`.
- Config: `WHATSAPP_ACCESS_TOKEN`, `WHATSAPP_PHONE_NUMBER_ID`, `WHATSAPP_TEMPLATE_NAME`,
  `WHATSAPP_WEBHOOK_VERIFY_TOKEN`, `WHATSAPP_APP_SECRET`, `OTP_PROVIDER` em
  `app/core/config.py`. `TWILIO_*` permanecem até a Fase 5.
- `auth.py`: substituir o tratamento Twilio-específico (códigos 60200/60203 → HTTP 429)
  por erros do orquestrador; assinaturas dos endpoints inalteradas.

**v2 Go** (`football-api-go/`):
- Remover o dead code `internal/services/twilio.go`.
- Novo `internal/services/whatsapp_otp.go` espelhando a v1 (store via interface para teste,
  padrão `VoteReminderStore`); mesma tabela, mesmas regras.
- `auth_service.go`: `sendOTPToNumber`/`checkOTP` usam o orquestrador; corrigir
  `.env.example` (`TWILIO_VERIFY_SERVICE_SID` → `TWILIO_VERIFY_SID`).
- `config.go`: mesmas envs novas + `OTPProvider`.
- Webhook `/api/v2/webhooks/whatsapp` (mesma validação de assinatura).

**Testes** (obrigatórios antes do commit):
- v1: `tests/unit/services/test_whatsapp_otp.py` (hash/TTL/tentativas/cooldown/bypass,
  Graph API mockada via `pytest-mock`); atualizar `tests/unit/routers/test_auth.py`.
- v2: `tests/unit/` com fake store + fake HTTP; `go test ./...` + `govulncheck`.
- E2E: continuam via bypass (sem mudança).

### Fase 2 — Frontend, i18n e legal (sem mudança de contrato)

- i18n nos 3 arquivos (`messages/{pt-BR,en,es}.json`): "SMS" → "WhatsApp" nas chaves
  `login.phone_hint`, `login.verify_subtitle`, `login.identity_verified`,
  `register.code_sent_to`, `register.phone_hint`, `profile.otp_sent_to`,
  `profile.identity_verified` etc.
- **Política de privacidade (LGPD)**: `privacy.s3.r3.partner` "Provedor de SMS (Twilio)" →
  incluir Meta Platforms como operador do canal WhatsApp (Twilio permanece citada enquanto
  for fallback).
- `autocomplete="one-time-code"` continua (copy-code no WhatsApp + colar); WebOTP/one-tap
  fica no PRD 026.

### Fase 3 — Infra/segredos

- GitHub Secrets novos (`WHATSAPP_*`); workflows `main.yml` e compose files
  (`docker-compose.prod.yml`, `docker-compose.go-dev.yml` segue com bypass).
- Homelab: adicionar as envs ao Secret `rachao-api` (fora do git) — a v2 precisa delas no
  cutover.
- Webhook exige URL pública: `api.rachao.app/api/v1/webhooks/whatsapp` (VPS hoje; homelab
  após o cutover — ingress já expõe `api.rachao.app`).

### Fase 4 — Rollout gradual

1. Deploy com `OTP_PROVIDER=twilio_sms` (comportamento atual; código novo dormindo).
2. Smoke test com `OTP_PROVIDER=whatsapp` em dev/staging (número de teste da Meta envia
   para até 5 números sem custo).
3. Produção → `whatsapp` (fallback SMS no resend ativo). Monitorar: taxa de entrega
   (webhook statuses), quality rating da WABA (painel Meta), frequência do fallback.
4. Critério de sucesso: **≥ 95% dos OTPs entregues por WhatsApp por 2–4 semanas**.

### Fase 5 — Descomissionamento Twilio

- Remover `OTP_PROVIDER`/fallback, código Twilio das duas APIs, envs/secrets/mocks;
  atualizar a política de privacidade (remover Twilio); cancelar o serviço Verify.
- Atualizar PRD 003 (superado por este) e PRD 044 §17 (item de paridade).

## 4. Riscos e mitigação

| Risco | Mitigação |
|---|---|
| Usuário sem WhatsApp ativo no número | Fallback SMS no resend (Fase 4); i18n orienta |
| Limite 250 msgs/dia sem business verification | Suficiente hoje; monitorar; CNPJ destrava 1k+ |
| Quality rating baixo pausa o número | Template auth com texto fixo da Meta = risco mínimo; monitorar painel |
| Token permanente vaza | System user com escopo mínimo; secret fora do git; rotação documentada |
| Preço Meta muda (modelo já mudou em jul/2025) | Custo ~10x menor dá margem; orquestrador volta p/ Twilio via env |
| Estado de OTP próprio tem bugs (segurança) | Hash + TTL + tentativas com testes dedicados nas 2 APIs; rate-limit por número e IP |

## 5. Verificação (quando implementado)

- Unit: suítes novas v1/v2 verdes (`pytest`, `go test ./...`).
- Dev: `OTP_PROVIDER=whatsapp` + número de teste Meta → template real no aparelho, código
  valida, expira em 10 min, 6ª tentativa bloqueia, resend em <60s recusa.
- Webhook: POST de status com assinatura válida/inválida.
- Prod: cadastro real de ponta a ponta; entregas visíveis no painel Meta; custo por
  mensagem no billing.

## 6. Fora de escopo

- One-tap/zero-tap autofill Android (hash do app; PRD 026).
- Notificações de produto via WhatsApp (utility templates) — só OTP.
- Troca de número de WhatsApp do usuário (fluxo inexistente hoje; PRD próprio).
