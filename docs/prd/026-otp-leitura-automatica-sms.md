# PRD — Leitura Automática do Código OTP via SMS
## Rachao.app · Gerenciamento de Grupos e Partidas

| | |
|---|---|
| **Versão** | 1.0 |
| **Status** | Proposto — aguardando revisão |
| **Data** | Março de 2026 |
| **Plataforma** | https://rachao.app (PWA mobile-first) |

---

## 1. Visão Geral

### 1.1 Contexto

O app usa verificação por OTP via SMS (Twilio Verify) em dois fluxos:
1. **Cadastro** (`/register`, etapa 2) — confirma posse do número antes de criar a conta
2. **Recuperação de senha** (`/login`) — valida identidade antes de permitir nova senha

Em ambos os casos, o usuário recebe um SMS com 6 dígitos e precisa digitá-los manualmente no campo de input da tela.

### 1.2 Problema

Digitar o código manualmente introduz fricção desnecessária:
- Exige troca de contexto (sair do app/browser, abrir notificação do SMS, memorizar 6 dígitos, voltar ao app)
- Em dispositivos que não exibem a notificação na tela, o usuário precisa abrir o app de SMS explicitamente
- Aumenta a taxa de abandono no fluxo de cadastro
- Causa frustração, especialmente quando o SMS chega enquanto o usuário ainda está digitando

### 1.3 Objetivo

Preencher automaticamente o campo OTP assim que o SMS chegar, eliminando a necessidade de troca de contexto manual, usando as APIs nativas disponíveis em cada plataforma.

---

## 2. Solução Técnica

A solução depende da plataforma:

### 2.1 Android — WebOTP API (Chrome / PWA)

A [WebOTP API](https://developer.mozilla.org/en-US/docs/Web/API/WebOTP_API) permite que o browser leia automaticamente um SMS destinado ao domínio registrado e preencha o campo OTP sem interação do usuário.

**Como funciona:**
1. O frontend chama `navigator.credentials.get({ otp: { transport: ['sms'] } })`
2. O Chrome solicita permissão ao usuário — exibe um bottom sheet: _"Usar o código do SMS para rachao.app?"_
3. Se o usuário confirmar (ou se o SMS chegar antes do timeout), o código é retornado automaticamente
4. O campo é preenchido e o formulário pode ser submetido automaticamente

**Suporte:** Chrome 84+ no Android, Edge mobile. **Não suportado no iOS Safari.**

**Requisito de formato do SMS:** o SMS enviado pela Twilio deve terminar com duas linhas específicas:
```
Seu código de verificação é: 483920

@rachao.app #483920
```
A última linha (`@dominio #codigo`) é lida pelo Chrome para verificar que o SMS é destinado a este domínio. Sem esse formato, a WebOTP API não funciona.

**Ajuste necessário no backend:** configurar o template de SMS da Twilio Verify para incluir a linha `@rachao.app #{{code}}` ao final da mensagem.

---

### 2.2 iOS — autocomplete="one-time-code" (Safari / PWA)

O iOS Safari não implementa a WebOTP API, mas suporta o atributo HTML `autocomplete="one-time-code"` nos campos de input.

**Como funciona:**
- Quando um SMS chega com um código numérico, o iOS detecta-o e exibe uma sugestão acima do teclado: _"De Mensagens: 483920"_
- O usuário toca na sugestão (1 toque) — não precisa sair do app nem digitar
- **Não é 100% automático** — exige o toque do usuário, mas elimina a troca de contexto

**Requisito:** o campo input deve ter `autocomplete="one-time-code"`. Nenhuma mudança no SMS é necessária para iOS.

**Suporte:** iOS 12+ / Safari.

---

### 2.3 Fallback

Nos demais casos (desktop, browsers sem suporte, WebOTP API recusada pelo usuário), o campo funciona normalmente — o usuário digita o código manualmente. Nenhuma regressão.

---

## 3. Fluxo por Plataforma

### Android (Chrome/PWA) — preenchimento automático

```
SMS chega no dispositivo
        ↓
Chrome intercepta (verifica domínio @rachao.app)
        ↓
Bottom sheet: "Usar código 483920 para rachao.app?"
        ↓ usuário toca "Usar"  (ou confirmação automática)
        ↓
Campo OTP preenchido → submit automático
        ↓
Fluxo continua normalmente
```

### iOS (Safari/PWA) — sugestão no teclado

```
SMS chega no dispositivo
        ↓
iOS exibe sugestão acima do teclado: "De Mensagens: 483920"
        ↓ usuário toca na sugestão (1 toque)
        ↓
Campo OTP preenchido → usuário toca "Confirmar"
        ↓
Fluxo continua normalmente
```

### Desktop / fallback — digitação manual (sem mudança)

```
SMS chega no dispositivo do usuário
        ↓
Usuário digita os 6 dígitos manualmente
        ↓
Usuário toca "Confirmar"
        ↓
Fluxo continua normalmente
```

---

## 4. Telas Afetadas

| Tela | Fluxo | Campo OTP |
|---|---|---|
| `/register` — etapa 2 | Verificação de número no cadastro | Input `otp_code`, 6 dígitos |
| `/login` — recuperação de senha | Confirmação de identidade antes da nova senha | Input `resetCode`, 6 dígitos |

---

## 5. Requisitos Funcionais

**RF-01 — WebOTP API no Android**
O frontend deve chamar `navigator.credentials.get({ otp: { transport: ['sms'] } })` assim que o campo OTP for exibido. Se a API retornar um código, preenchê-lo no input e submeter o formulário automaticamente.

**RF-02 — autocomplete no iOS**
Todos os inputs de código OTP devem ter `autocomplete="one-time-code"` e `inputmode="numeric"` para ativar a sugestão nativa do iOS.

**RF-03 — Timeout da WebOTP API**
A chamada deve ter timeout de 60 segundos (ou o TTL do OTP). Se o SMS não chegar nesse prazo, o campo permanece disponível para digitação manual — sem bloquear o usuário.

**RF-04 — Cancelamento gracioso**
Se o usuário rejeitar o bottom sheet do Chrome (ou a API lançar `AbortError`), o fluxo cai silenciosamente para digitação manual — sem exibir erros.

**RF-05 — Formato do SMS (backend)**
O template de SMS da Twilio Verify deve ser customizado para incluir a linha `@rachao.app #{{code}}` ao final, ativando a WebOTP API no Android.

**RF-06 — Submit automático no Android**
Após preenchimento via WebOTP, o formulário deve ser submetido automaticamente (sem exigir toque extra do usuário).

**RF-07 — Sem submit automático no iOS**
No iOS, após preenchimento via sugestão, aguardar ação explícita do usuário (toque no botão "Confirmar") — o iOS não garante que o preenchimento ocorreu antes de qualquer auto-submit.

---

## 6. Requisitos Não Funcionais

**RNF-01 — Degradação graceful:** a feature deve ser uma melhoria progressiva — se a API não estiver disponível, o campo funciona normalmente. Nenhuma funcionalidade é bloqueada.

**RNF-02 — Sem mudança na segurança:** o código ainda é validado no backend (Twilio Verify). A leitura automática só elimina a digitação manual — não altera o fluxo de validação.

**RNF-03 — Permissão não persistente:** a WebOTP API solicita permissão por sessão. O app não armazena permissão de leitura de SMS — o Chrome gerencia isso nativamente.

---

## 7. Mudanças Necessárias

### 7.1 Frontend (`/register` e `/login`)

```svelte
<!-- Input OTP — adicionar autocomplete e inputmode -->
<input
  type="text"
  inputmode="numeric"
  autocomplete="one-time-code"
  maxlength="6"
  bind:value={otpCode}
/>
```

```typescript
// Lógica WebOTP — Svelte 5 com $effect reagindo à variável de passo
// onMount NÃO é adequado aqui pois o campo OTP é renderizado condicionalmente
// dentro de um componente maior. $effect rastreia `currentStep` e dispara
// exatamente quando o step muda para 'otp', que é quando o campo fica visível.

let abortController = $state<AbortController | null>(null);

$effect(() => {
  if (currentStep !== 'otp') {
    abortController?.abort(); // cancela se o usuário voltar ao step anterior
    return;
  }
  if (!('OTPCredential' in window)) return;

  abortController = new AbortController();
  const ac = abortController;
  const timeout = setTimeout(() => ac.abort(), 60_000);

  navigator.credentials.get({
    otp: { transport: ['sms'] },
    signal: ac.signal,
  } as CredentialRequestOptions)
    .then((otp) => {
      clearTimeout(timeout);
      if (otp && 'code' in otp) {
        otpCode = (otp as OTPCredential).code;
        handleSubmit(); // submit automático
      }
    })
    .catch(() => {
      // AbortError ou NotSupportedError — silencioso, fallback para digitação manual
    });
});
```

> **Nota:** `onMount` seria suficiente apenas se o step OTP fosse um componente Svelte independente (o `onMount` do filho dispara quando ele é montado no DOM). Como `/register` e `/login` são componentes únicos com renderização condicional por step, `$effect` é a abordagem correta no Svelte 5.

### 7.2 Backend — Template SMS Twilio

> ⚠️ **Limitação identificada:** o Twilio Verify não oferece campo de texto livre para o corpo do SMS — apenas templates pré-aprovados, nenhum deles compatível com o formato `@dominio #codigo` exigido pela WebOTP API. Consulte a seção 12 para opções de resolução.

O formato necessário seria:

```
O seu código de verificação para Rachão App é: {{code}}.

@rachao.app #{{code}}
```

> A linha `@dominio #codigo` **deve ser a última linha do SMS** para a WebOTP API funcionar no Android/Chrome.

---

## 8. Considerações de Privacidade

- A WebOTP API **não dá acesso ao conteúdo completo do SMS** — o browser lê apenas o código da linha `#codigo`, não o restante da mensagem.
- O Chrome exibe um bottom sheet explícito antes de compartilhar o código — o usuário sempre confirma antes do preenchimento automático.
- O iOS só sugere o código via QuickType — não há acesso programático ao SMS.
- Nenhum dado de SMS é armazenado pelo app.

---

## 9. Limitações Conhecidas

| Situação | Comportamento |
|---|---|
| iOS Safari (WebOTP API ausente) | Sugestão via `autocomplete="one-time-code"` — 1 toque do usuário |
| Desktop (Chrome/Firefox) | Digitação manual — sem mudança |
| SMS com formato incorreto (sem `@dominio`) | WebOTP API não detecta o código — digitação manual |
| Usuário rejeita o bottom sheet Android | Digitação manual |
| SMS demora mais de 60s | Timeout da API — digitação manual |
| PWA instalada vs browser | Mesmo comportamento — WebOTP funciona em ambos no Android |

---

## 10. Critérios de Aceitação

> ⚠️ **Implementação revertida em 2026-03-25.** O `$effect` WebOTP foi removido de `/register` e `/login` pois o bottom sheet do Chrome no Android interferia no foco dos inputs de dígitos, quebrando o fluxo de digitação manual. O `autocomplete="one-time-code"` foi mantido (inofensivo, ativa sugestão no iOS). A retomada desta feature depende da aprovação do template SMS pela Twilio (seção 12).

- [ ] No Android/Chrome, ao chegar o SMS, o bottom sheet do Chrome exibe o código e o campo é preenchido automaticamente após confirmação *(bloqueado: template SMS pendente de aprovação Twilio + implementação revertida)*
- [ ] No iOS/Safari, ao tocar na sugestão do teclado, o campo OTP é preenchido *(`autocomplete="one-time-code"` mantido — sugestão aparece, mas sem auto-submit)*
- [ ] Se a WebOTP API lançar `AbortError` ou `NotSupportedError`, nenhum erro é exibido ao usuário
- [x] O campo OTP sempre permite digitação manual como fallback
- [ ] O template do SMS inclui a linha `@rachao.app #codigo` ao final *(aguardando resposta do chamado Twilio)*
- [ ] Telas afetadas: `/register` (etapa 2) e `/login` (recuperação de senha)
- [ ] O submit automático ocorre apenas no Android (WebOTP) — no iOS aguarda ação do usuário

---

## 11. Fora de Escopo

- Login via OTP (autenticação sem senha) — feature separada
- Leitura de SMS em apps nativos (não é PWA/browser — não se aplica)
- Suporte ao Firefox (não implementa WebOTP API — usuário digita manualmente)

---

## 12. Limitação: Twilio Verify não suporta template livre

O Twilio Verify oferece apenas templates pré-aprovados no painel — nenhum deles inclui o formato `@dominio #codigo` exigido pela WebOTP API. Isso bloqueia o preenchimento automático no Android/Chrome.

**Opções disponíveis:**

| Opção | Esforço | Resultado |
|---|---|---|
| **A — Twilio Content Template Builder** | Médio | Criar template customizado via Console → Messaging → Content Template Builder, submetê-lo para aprovação da Twilio e vinculá-lo ao Verify Service. Processo pode levar dias e sem garantia de aprovação. |
| **B — Twilio Programmable SMS** | Alto | Abandonar o Verify e enviar SMS diretamente via API de Mensagens, com controle total do corpo. Exige implementar no backend: geração de código, armazenamento com TTL, hash e rate limiting (o que o Verify faz hoje automaticamente). |
| **C — Aceitar limitação (recomendado agora)** | Nenhum | O código frontend já está pronto e o fallback para digitação manual funciona sem erros. iOS já recebe a melhoria via `autocomplete="one-time-code"`. Android pode ser revisitado ao migrar para Programmable SMS. |

**Decisão atual:** seguir com a **Opção C**. O frontend está implementado e não há regressão — a WebOTP API silenciosamente não detecta o código e o usuário digita normalmente.

---

*Documento elaborado para uso interno da equipe de produto e engenharia do Rachao.app.*
