# PRD — Validação de Senha Igual à Atual

## 1. Contexto

O rachao.app possui dois fluxos de troca de senha:

1. **Esqueci minha senha** (`/login`) — público, sem autenticação. Usa OTP via WhatsApp para verificar identidade antes de redefinir a senha.
2. **Alterar Senha** (`/profile`) — autenticado. Aceita ou a senha atual como verificação, ou um OTP enviado ao WhatsApp do usuário.

Em ambos os fluxos, atualmente **não há validação que impeça o usuário de cadastrar a mesma senha que já possui**. O resultado é que a operação é executada com sucesso, recalculando o hash sem nenhum benefício real e podendo gerar confusão ("mudei a senha e continua funcionando com a antiga").

---

## 2. Problema

### Dor atual
- Usuário que esqueceu a senha passa por OTP, digita a senha atual por engano e o sistema aceita sem avisar
- Usuário em `/profile` que tenta "renovar" a senha sem perceber que digitou a mesma coisa recebe confirmação de sucesso
- Sem feedback diferenciado, o usuário pode achar que está seguro tendo "trocado" a senha, quando na prática nada mudou

### Impacto
- Experiência degradada: fluxo de segurança com feedback enganoso
- Confusão: usuário acha que redefiniu a senha quando na prática manteve a mesma
- Fraqueza sutil de segurança: reforça reutilização de senha sem nem alertar o usuário

---

## 3. Solução Proposta

### Validação no backend (fonte da verdade)

A validação deve ocorrer **no backend**, pois é lá que o hash da senha atual está disponível. O frontend não deve receber o hash nem tentar comparar localmente.

#### Endpoint `POST /auth/change-password`

O player autenticado já está disponível via `CurrentPlayer`. A verificação é trivial:

```python
if verify_password(body.new_password, current.password_hash):
    raise HTTPException(status_code=422, detail="SAME_PASSWORD")
```

**Momento de checagem:** após autenticar o requester (via `current_password` ou `otp_token`), e antes de salvar o novo hash.

#### Endpoint `POST /auth/forgot-password/reset`

O player não está autenticado, mas o endpoint já busca o player pelo `whatsapp` informado (para atualizar o hash). A verificação é adicionada nesse mesmo ponto:

```python
player = await repo.get_by_whatsapp(body.whatsapp)
if verify_password(body.new_password, player.password_hash):
    raise HTTPException(status_code=422, detail="SAME_PASSWORD")
```

**Momento de checagem:** após validar o OTP token, antes de salvar o novo hash.

---

## 4. Impacto por Camada

### Backend (`football-api`)

| Arquivo | Mudança |
|---------|---------|
| `app/api/v1/routers/auth.py` | Adicionar checagem `verify_password(new_password, hash)` em `change_password()` e `forgot_password_reset()` |

Nenhuma migration necessária — não há mudança de schema.

### Frontend (`football-frontend`)

Ambos os fluxos já tratam erros da API via `catch`. Basta mapear o detalhe `"SAME_PASSWORD"` para uma mensagem de erro contextual.

| Fluxo | Página | Comportamento esperado |
|-------|--------|------------------------|
| Alterar Senha | `/profile` | Mensagem inline no campo "Nova senha": "A nova senha não pode ser igual à senha atual" |
| Esqueci minha senha | `/login` | Mensagem inline no campo "Nova senha": "A nova senha não pode ser igual à senha atual" |

### i18n (`messages/`)

Nova chave a adicionar nos 3 idiomas (`pt-BR`, `en`, `es`):

```json
"auth.same_password_error": "A nova senha não pode ser igual à senha atual"
```

```json
"auth.same_password_error": "The new password cannot be the same as your current password"
```

```json
"auth.same_password_error": "La nueva contraseña no puede ser igual a la contraseña actual"
```

---

## 5. Comportamento por Cenário

| Cenário | Comportamento atual | Comportamento após fix |
|---------|--------------------|-----------------------|
| Alterar senha para a mesma via senha atual | Sucesso silencioso | Erro 422 + mensagem "A nova senha não pode ser igual à senha atual" |
| Alterar senha para a mesma via OTP | Sucesso silencioso | Erro 422 + mensagem igual |
| Redefinir senha (esqueci) para a mesma | Sucesso silencioso | Erro 422 + mensagem igual |
| Troca de senha por senha diferente | Funciona ✅ | Sem alteração ✅ |

---

## 6. Detalhes Técnicos

### Por que a checagem é segura com bcrypt

O bcrypt é uma função de hash **unidirecional**, mas `verify_password(plain, hashed)` compara o texto puro com o hash existente — o que é exatamente o que já fazemos para autenticar o usuário. Usar `verify_password` para checar igualdade antes de salvar é 100% seguro e não expõe o hash.

### Fluxo de erro no frontend

A função `changePassword()` em `api.ts` já propaga exceções HTTP. O tratamento atual em `/profile` e `/login` usa `catch (e)` e exibe mensagens genéricas. A mudança é mapear o `detail === 'SAME_PASSWORD'` para a mensagem específica:

```typescript
} catch (e: any) {
  if (e?.detail === 'SAME_PASSWORD') {
    passwordError = $t('auth.same_password_error');
  } else {
    passwordError = $t('error.generic');
  }
}
```

---

## 7. Critérios de Aceitação

- [ ] `POST /auth/change-password` retorna `422 { detail: "SAME_PASSWORD" }` quando `new_password` é igual à senha atual (tanto via `current_password` quanto via `otp_token`)
- [ ] `POST /auth/forgot-password/reset` retorna `422 { detail: "SAME_PASSWORD" }` quando `new_password` é igual à senha atual
- [ ] O frontend em `/profile` exibe mensagem de erro inline específica ao receber `SAME_PASSWORD`
- [ ] O frontend em `/login` (fluxo esqueci a senha) exibe mensagem de erro inline específica ao receber `SAME_PASSWORD`
- [ ] As mensagens estão traduzidas nos 3 idiomas (`pt-BR`, `en`, `es`)
- [ ] Senha diferente da atual continua funcionando normalmente

---

## 8. O que NÃO está no escopo

- **Histórico de senhas**: bloquear reutilização das últimas N senhas — exigiria armazenar hashes anteriores
- **Política de complexidade**: exigir maiúsculas, números, símbolos — escopo separado
- **Admin reset**: o endpoint `POST /players/{id}/reset-password` gera senha temporária aleatória — nunca será igual à atual por design, sem necessidade de checagem
