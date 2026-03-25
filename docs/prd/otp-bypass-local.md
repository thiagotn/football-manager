# PRD — Bypass de OTP para Desenvolvimento Local

## 1. Contexto

O rachao.app usa o **Twilio Verify** para envio e verificação de códigos OTP em três fluxos:

| Fluxo | Endpoints |
|-------|-----------|
| Cadastro | `POST /auth/send-otp` + `POST /auth/verify-otp` |
| Esqueci minha senha | `POST /auth/forgot-password/send-otp` + `POST /auth/forgot-password/verify-otp` |
| Alterar senha (autenticado) | `POST /auth/send-otp/me` + `POST /auth/verify-otp/me` |

Hoje, o ambiente local (`.env.docker`) já contém credenciais Twilio reais. Isso significa que qualquer chamada OTP durante desenvolvimento — testes manuais, depuração, demonstrações — gera **cobranças reais** na conta Twilio e exige acesso a um número de WhatsApp válido.

A configuração atual de ambiente usa `APP_ENV` com `is_prod` em `app/core/config.py`, mas não há mecanismo de bypass para os serviços externos.

---

## 2. Problema

- Testes manuais do fluxo de cadastro ou recuperação de senha consomem créditos Twilio sem necessidade
- Desenvolver ou depurar qualquer tela que envolva OTP exige ter o celular em mãos
- Impossibilita testar esses fluxos em ambientes de CI ou staging sem conta Twilio configurada
- Risco de acumular custos em iterações rápidas de desenvolvimento/QA

---

## 3. Solução Proposta

Adicionar uma variável de ambiente `OTP_BYPASS_CODE` que, quando definida **em ambiente não-produção**, substitui as chamadas reais ao Twilio por uma implementação local:

- **`send_otp` em bypass:** não chama o Twilio, apenas loga `"[OTP BYPASS] código aceito: {OTP_BYPASS_CODE}"` — simula o envio sem custo
- **`check_otp` em bypass:** retorna `True` se o código enviado for igual ao `OTP_BYPASS_CODE`, `False` caso contrário — sem chamar o Twilio

O comportamento é **transparente para os routers** — nenhuma mudança nos endpoints, schemas ou lógica de negócio.

### Regra de segurança dupla

O bypass só é ativado quando **ambas** as condições são verdadeiras:

1. `OTP_BYPASS_CODE` está definido e não é vazio
2. `APP_ENV != "production"`

Mesmo que alguém acidentalmente coloque `OTP_BYPASS_CODE` em produção, a segunda condição bloqueia o bypass.

---

## 4. Impacto por Camada

### `app/core/config.py`

Adicionar um campo:

```python
otp_bypass_code: str = ""
```

Lido via `OTP_BYPASS_CODE` no `.env`. Padrão vazio = bypass desabilitado.

### `app/services/twilio_verify.py`

Envolver as funções existentes com a checagem de bypass:

```python
import structlog
from app.core.config import settings

log = structlog.get_logger()

async def send_otp(whatsapp: str) -> None:
    if settings.otp_bypass_code and not settings.is_prod:
        log.info("OTP bypass ativo — nenhuma chamada Twilio realizada", whatsapp=whatsapp)
        return
    # ... implementação Twilio atual ...

async def check_otp(whatsapp: str, code: str) -> bool:
    if settings.otp_bypass_code and not settings.is_prod:
        result = code == settings.otp_bypass_code
        log.info("OTP bypass", whatsapp=whatsapp, accepted=result)
        return result
    # ... implementação Twilio atual ...
```

### `.env.docker` (ambiente local)

Adicionar a variável com um código fixo de teste:

```dotenv
OTP_BYPASS_CODE=000000
```

Com isso, ao rodar localmente, o código `000000` funciona em qualquer tela de OTP.

### `.env.example` / `.env.prod.example`

Documentar a variável como **comentada** nos exemplos de produção, para deixar claro que não deve ser definida em prod:

```dotenv
# OTP_BYPASS_CODE=  # Deixar vazio ou ausente em produção — usar apenas em dev/staging
```

---

## 5. Comportamento por Ambiente

| Ambiente | `APP_ENV` | `OTP_BYPASS_CODE` | Comportamento |
|----------|-----------|-------------------|---------------|
| Local (Docker) | `development` | `000000` | Bypass ativo — qualquer OTP `000000` é aceito, Twilio não é chamado |
| Local (sem definir) | `development` | `` (vazio) | Twilio é chamado normalmente |
| Staging (opcional) | `staging` | `123456` | Bypass ativo com código configurável |
| Produção | `production` | qualquer valor | **Bypass bloqueado** pela checagem `is_prod` |

---

## 6. Critérios de Aceitação

- [ ] `app/core/config.py` lê `OTP_BYPASS_CODE` como `otp_bypass_code: str = ""`
- [ ] `send_otp()` em bypass loga e retorna sem chamar Twilio
- [ ] `check_otp()` em bypass retorna `True` somente para o código exato configurado
- [ ] O bypass **nunca** é ativado quando `APP_ENV=production`, independente de `OTP_BYPASS_CODE`
- [ ] `.env.docker` inclui `OTP_BYPASS_CODE=000000` por padrão
- [ ] `.env.example` documenta a variável como comentada/ausente
- [ ] O comportamento de produção (Twilio real) permanece inalterado
- [ ] Os testes unitários existentes de OTP continuam passando

---

## 7. Testes Unitários

Os testes unitários de OTP (`test_auth.py`) usam mocks para `twilio_verify.send_otp` e `twilio_verify.check_otp` — não chamam Twilio nem dependem do bypass. Não há necessidade de novos testes unitários para este PRD.

Opcionalmente, pode-se adicionar testes para o módulo `twilio_verify.py` diretamente:

- `test_send_otp_bypass_skips_twilio` — verifica que Twilio não é chamado quando bypass ativo
- `test_check_otp_bypass_accepts_correct_code` — verifica que o código correto retorna `True`
- `test_check_otp_bypass_rejects_wrong_code` — verifica que código errado retorna `False`
- `test_bypass_disabled_in_production` — verifica que `is_prod=True` desativa o bypass

---

## 8. O que NÃO está no escopo

- **Bypass no frontend:** não há necessidade — o frontend trata o fluxo normalmente; o bypass é transparente
- **OTP em memória / banco:** a solução não armazena OTPs — usa código fixo configurável, mais simples
- **Múltiplos códigos de bypass:** um único código por ambiente é suficiente
- **Remoção das credenciais Twilio do `.env.docker`:** podem ser mantidas para quando for necessário testar o fluxo real localmente (basta remover `OTP_BYPASS_CODE`)
