# PRD — Verificação por SMS na Troca de Senha
## Rachao.app · Segurança de Conta

| | |
|---|---|
| **Versão** | 1.0 |
| **Status** | ✅ Implementado — Março 2026 |
| **Data** | Março de 2026 |

---

## 1. Contexto e Viabilidade

### 1.1 Fluxo atual

A troca de senha em `/profile` exige apenas `senha_atual + nova_senha`. Não há verificação adicional de identidade. Se a conta de um usuário for acessada por terceiro (dispositivo compartilhado, sessão não encerrada), qualquer pessoa com acesso à sessão pode trocar a senha sem nenhuma barreira extra.

### 1.2 Infraestrutura já disponível

A integração com **Twilio Verify (canal SMS)** já está implementada e funcionando em produção:

| Endpoint existente | Uso atual |
|---|---|
| `POST /auth/send-otp` | Envio de OTP no **cadastro** (valida que o número não está registrado) |
| `POST /auth/verify-otp` | Verificação do código → retorna `otp_token` assinado (JWT) |

O serviço `twilio_verify.py` expõe `send_otp(whatsapp)` e `check_otp(whatsapp, code)` prontos para reutilização.

### 1.3 Conclusão de viabilidade

✅ **Totalmente viável.** O único trabalho necessário é:
1. Um novo endpoint `POST /auth/send-otp/me` (envia OTP para o WhatsApp do usuário autenticado, sem precisar informar o número).
2. Adaptar o endpoint `POST /auth/change-password` para aceitar `otp_token` como alternativa (ou requisito adicional) à `senha_atual`.

---

## 2. Problema a Resolver

- Troca de senha não exige prova de posse do número de WhatsApp cadastrado.
- Usuário que perde acesso à senha atual não tem como redefinir sem contato com admin.
- Admins precisam gerar senhas temporárias manualmente para esses casos.

---

## 3. Proposta: Dois Casos de Uso

### Caso A — Usuário lembra a senha atual (fluxo normal)
> Mantém o fluxo atual. Nenhuma mudança.

### Caso B — Usuário não lembra a senha atual (novo fluxo)
> Permite redefinir senha via verificação por SMS, sem precisar informar a senha atual.

---

## 4. Fluxo Proposto (Caso B)

```
[Tela de troca de senha]
  → Link: "Não lembro a senha atual"
      ↓
  Código enviado para o WhatsApp cadastrado (número mascarado: ••••••1234)
      ↓
  Usuário insere o código de 6 dígitos
      ↓
  Código validado → formulário de nova senha liberado
      ↓
  Nova senha salva (sem exigir senha atual)
```

### Estados da tela `/profile` (troca de senha):

| Estado | O que o usuário vê |
|---|---|
| Padrão | Campos: senha atual + nova senha + confirmar |
| Clicou "Não lembro" | Campo de senha atual some; botão "Enviar código" aparece |
| Código enviado | Campo para digitar os 6 dígitos; contador de reenvio (60s) |
| Código validado | Campos de nova senha + confirmar liberados; senha atual não é necessária |
| Erro de código | Mensagem "Código inválido ou expirado" |

---

## 5. Alterações Técnicas

### Backend

**Novo endpoint:** `POST /auth/send-otp/me`
- Requer autenticação (`current_player`)
- Envia OTP para `current_player.whatsapp` via Twilio Verify
- Resposta: `{ "message": "OTP enviado" }`
- Reutiliza `send_otp()` de `twilio_verify.py`
- Não precisa verificar se o número já existe (o usuário já está autenticado)

**Adaptação:** `POST /auth/change-password`
- Schema atual: `{ current_password, new_password }`
- Schema novo: `{ current_password?, new_password, otp_token? }`
- Regra: deve vir `current_password` **ou** `otp_token` válido (não ambos obrigatórios)
- Validação do `otp_token`: mesma lógica já usada no cadastro (JWT assinado com claim do WhatsApp do usuário)

### Frontend

- Link "Não lembro a senha atual" abaixo do campo de senha atual
- Troca o modo da tela (estado: `'password' | 'otp-pending' | 'otp-verified'`)
- Componente de input de código OTP (6 dígitos) — pode reutilizar o mesmo do cadastro
- Exibição do número mascarado para confirmar para qual número o código foi enviado

---

## 6. Segurança

| Risco | Mitigação |
|---|---|
| Brute force no código | Twilio bloqueia após 5 tentativas erradas (comportamento padrão do Verify) |
| Reutilização do `otp_token` | Token tem expiração curta (10 min); após uso na troca de senha, não pode ser reutilizado |
| Sessão comprometida + troca de senha | O atacante precisaria também ter acesso ao WhatsApp físico para receber o SMS |

---

## 7. O que NÃO está no escopo desta v1

- Envio de notificação ao usuário informando que a senha foi trocada (push/SMS)
- Encerramento de outras sessões após troca de senha
- Autenticação de dois fatores no login (fluxo diferente)

---

## 8. Impacto nos Planos

Sem impacto — funcionalidade disponível para todos os planos, pois é segurança básica de conta.

---

## 9. Dependências

- Twilio Verify já configurado em produção ✅
- Secrets `TWILIO_ACCOUNT_SID`, `TWILIO_AUTH_TOKEN`, `TWILIO_VERIFY_SID` já injetados no VPS ✅
- Nenhuma nova migração de banco necessária ✅
