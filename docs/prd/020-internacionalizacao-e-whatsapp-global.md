# PRD — Internacionalização e Suporte a WhatsApp Internacional

| | |
|---|---|
| **Status** | ✅ Implementado — Março 2026 |
| **Data** | Março de 2026 |

---

## 1. Contexto

O rachao.app foi construído exclusivamente para o mercado brasileiro. Toda a lógica de telefone assume o formato nacional (DDD + 9 dígitos), a interface está em português e strings de data/hora usam `pt-BR` hardcoded. Este PRD define o escopo técnico e de produto para expandir o suporte a jogadores internacionais, cobrindo dois eixos independentes mas relacionados:

1. **WhatsApp global** — aceitar números de qualquer país em formato E.164
2. **i18n** — traduzir a interface para inglês (en) e espanhol (es), além do português (pt-BR) original

---

## 2. Problema

### 2.1 WhatsApp

- O campo `whatsapp` armazena apenas dígitos (`String(20)`), sem distinção de país
- A validação em `_normalize_whatsapp` aceita 10–13 dígitos e assume implicitamente prefixo brasileiro
- `formatWhatsapp()` formata apenas números de 11 dígitos no padrão `(XX) XXXXX-XXXX`
- `whatsappLink()` força o prefixo `55` caso o número não comece com ele
- O sanitize do frontend remove `55` quando detecta 13 dígitos (autocomplete do browser) — lógica específica para Brasil
- Usuários de outros países não conseguem cadastrar ou fazer login com seus números locais

### 2.2 i18n

- Todos os textos de UI, mensagens de erro, labels e conteúdo dinâmico estão em português hardcoded
- Datas e horas usam `toLocaleDateString('pt-BR', ...)` e strings como "Hoje", "Amanhã", "Ontem" em português
- Notificações push têm mensagens em português no backend
- Mensagens de erro da API retornam em português
- Não existe infraestrutura de tradução no projeto

---

## 3. Objetivos

- Permitir que jogadores de qualquer país se cadastrem e usem o app com seu número WhatsApp local
- Suportar a interface em pt-BR, en e es, com detecção automática do idioma do browser
- Manter compatibilidade retroativa com os dados existentes (números brasileiros sem código de país armazenado)
- Não quebrar a lógica de OTP via Twilio Verify (que já trabalha com E.164 internamente)

---

## 4. Escopo — Fora deste PRD

- Suporte a idiomas além de pt-BR, en e es
- Tradução de conteúdo gerado por usuários (nomes de grupos, observações de partidas)
- Localização de moeda/valores (BRL continua sendo a moeda padrão)
- RTL (right-to-left) para árabe ou hebraico
- Verificação de número via SMS (apenas WhatsApp OTP, já suportado pelo Twilio)

---

## 5. Eixo A — WhatsApp Internacional

### 5.1 Formato de armazenamento

**Decisão proposta:** armazenar o número no formato **E.164** completo (ex: `+5511999990000`, `+447911123456`).

Razões:
- É o padrão universal do WhatsApp/Twilio
- Elimina ambiguidade de país
- `wa.me/{número}` já aceita E.164 sem `+`
- A migração de dados existentes é direta: prefixar com `+55` todos os registros sem `+`

**Alternativa descartada:** armazenar `country_code` e `number` em colunas separadas — adiciona complexidade sem benefício real, pois E.164 já é indivisível e padronizado.

### 5.2 Migration de dados

```sql
-- Prefixar com +55 os números que ainda não têm código de país
UPDATE players
SET whatsapp = '+55' || whatsapp
WHERE whatsapp NOT LIKE '+%';
```

O campo `String(20)` comporta E.164 (máximo de 15 dígitos + `+` = 16 chars). Sem alteração de schema necessária para o tipo, apenas para os dados.

### 5.3 Validação backend

Substituir a lógica atual de `_normalize_whatsapp` por validação E.164:

```
Regra: deve começar com + seguido de 7 a 15 dígitos
Regex: ^\+[1-9]\d{6,14}$
```

A normalização deve:
1. Aceitar entrada com ou sem `+`, com ou sem espaços/hífens
2. Exigir código de país (mínimo 1 dígito após o `+`)
3. Rejeitar números com menos de 7 ou mais de 15 dígitos (após o `+`)
4. Sempre armazenar com `+` no início

### 5.4 Componente de input no frontend

Substituir os campos `type="tel"` atuais por um componente `PhoneInput` com:

- **Seletor de país**: dropdown com bandeira + código (ex: 🇧🇷 +55), buscável
- **Campo numérico**: aceita apenas dígitos após o código de país ser selecionado
- **Default**: 🇧🇷 +55 (mantém UX atual para usuários brasileiros)
- **Autocomplete**: `autocomplete="tel"` — com o código de país separado, o browser preenche corretamente

O componente deve ser compartilhado entre:
- `/register` — cadastro direto
- `/login` — autenticação
- `/invite/[token]` — entrada via convite
- `/players` (admin) — cadastro manual de jogadores
- `/profile` — edição de número

Biblioteca sugerida: `svelte-tel-input` ou implementação própria com lista de países do pacote `libphonenumber-js` (apenas os metadados, sem o parser completo, para reduzir bundle size).

### 5.5 Sanitize e formatação frontend

Substituir o `sanitizePhone()` atual (que remove `55` hardcoded) pela normalização via componente:

- `formatWhatsapp(phone)`: se começa com `+`, exibir no formato internacional (ex: `+44 79 1112 3456`); se é legado de 11 dígitos, manter o formato brasileiro atual
- `whatsappLink(phone)`: se começa com `+`, usar diretamente sem adicionar `55`; caso contrário, manter comportamento atual

### 5.6 OTP (Twilio Verify)

O Twilio Verify já trabalha com E.164 internamente. A mudança é transparente — basta garantir que o número enviado para a API do Twilio já esteja em E.164 (o que o novo formato de armazenamento garante).

### 5.7 Impacto em funcionalidades existentes

| Funcionalidade | Impacto |
|---|---|
| Login | Input precisa do PhoneInput com seletor de país |
| Cadastro via convite | Idem |
| Admin — cadastro manual | Idem |
| Compartilhar via WhatsApp | `whatsappLink()` precisa suportar E.164 |
| Exibição de número (admin) | `formatWhatsapp()` precisa suportar E.164 |
| OTP esqueci senha | Transparente (já usa E.164 no Twilio) |
| Testes E2E | Fixtures de login precisam usar E.164 |
| Dados existentes | Migration SQL para prefixar com `+55` |

---

## 6. Eixo B — Internacionalização (i18n)

### 6.1 Biblioteca

**Proposta: [Paraglide JS](https://inlang.com/m/gerre34r/library-inlang-paraglideJs) via `@inlang/paraglide-sveltekit`**

Razões:
- Desenvolvido especificamente para SvelteKit, com integração via Vite plugin
- Tree-shaking por idioma: apenas as strings do idioma ativo são incluídas no bundle
- Tipagem TypeScript automática (autocompletar em chaves de tradução)
- Routing por idioma: `/en/groups`, `/es/groups` ou subdomínio/cookie (configurável)
- Alternativa: `svelte-i18n` — mais simples de migrar, mas sem tree-shaking e sem tipagem automática

**Estratégia de rota:**
- Opção A: prefixo de idioma na URL (`/en/...`, `/es/...`, `/...` para pt-BR default) — amigável a SEO e compartilhamento de links com idioma preservado
- Opção B: detecção via cookie/localStorage sem mudança de URL — menor impacto em rotas existentes, links compartilhados não carregam idioma

**Recomendação**: Opção B para v1 (menor breaking change), com opção A como evolução futura para SEO.

### 6.2 Idiomas suportados

| Código | Idioma | Região alvo |
|---|---|---|
| `pt-BR` | Português (Brasil) | Brasil — idioma atual, default |
| `en` | English | EUA, UK, internacionais em geral |
| `es` | Español | Argentina, México, Espanha, outros hispânicos |

### 6.3 Detecção de idioma

Ordem de prioridade:
1. Preferência salva no perfil do jogador (persistida no backend)
2. Cookie `locale` (preserva preferência entre sessões sem login)
3. `navigator.language` do browser
4. Fallback: `pt-BR`

### 6.4 Estrutura de arquivos de tradução

```
football-frontend/
└── messages/
    ├── pt-BR.json    # idioma base (já existe como hardcoded, será extraído)
    ├── en.json
    └── es.json
```

Formato de chave:

```json
{
  "nav.groups": "Grupos",
  "nav.discover": "Descobrir Rachões",
  "match.status.open": "Aberta",
  "match.status.closed": "Encerrada",
  "match.confirm": "Vou jogar!",
  "match.decline": "Não posso",
  "date.today": "Hoje",
  "date.tomorrow": "Amanhã",
  "date.yesterday": "Ontem"
}
```

### 6.5 Escopo de tradução — Frontend

Prioridade alta (fluxos críticos):

- Autenticação: login, cadastro, recuperação de senha, convite
- Partidas: confirmação/recusa de presença, detalhes, lista de jogadores
- Grupos: listagem, detalhes, abas (Próximos, Últimos, Membros, Stats, Finanças)
- Notificações toast
- Mensagens de erro e validação de formulário
- Navbar e drawer

Prioridade média:

- Dashboard / home
- Descobrir Rachões
- Perfil do jogador
- FAQ (público e admin)
- Votação pós-partida

Prioridade baixa (pode ser fase 2):

- Termos de Uso e Política de Privacidade (documentos legais — tradução manual separada)
- Painel admin (uso interno, sempre por falantes de português)

### 6.6 Datas e horas

Substituir todas as ocorrências de `toLocaleDateString('pt-BR', ...)` e strings hardcoded por funções que recebem o locale ativo:

```typescript
// Antes
d.toLocaleDateString('pt-BR', { weekday: 'long', day: '2-digit', month: 'long' })
// Depois
d.toLocaleDateString(currentLocale, { weekday: 'long', day: '2-digit', month: 'long' })
```

Strings relativas ("Hoje", "Amanhã", "Ontem") devem ser chaves de tradução.

Nomes de meses em português que estão hardcoded no backend (`_MONTHS_PT` em `group_stats_repo.py`, `recurrence.py`) devem ser removidos do backend — as labels de período devem ser interpretadas e traduzidas no frontend.

### 6.7 Escopo de tradução — Backend

O backend retorna strings em dois contextos:

**1. Notificações push** (`app/services/push.py` e chamadas em routers):
- Título e corpo das notificações estão hardcoded em português
- Proposta: armazenar o `locale` preferido do jogador no banco e selecionar a mensagem correta ao enviar

**2. Mensagens de erro da API** (`detail` nos `HTTPException`):
- A maioria das mensagens de erro é exibida internamente ou pelo admin
- Para erros exibidos ao jogador (ex: "Senha incorreta", "Você não é membro deste grupo"), o cliente deve mapear códigos de erro para strings traduzidas no frontend
- Proposta: padronizar `detail` como códigos de erro em inglês snake_case (ex: `"WRONG_PASSWORD"`, `"NOT_A_MEMBER"`) e traduzir no frontend — isso já é parcialmente feito em alguns endpoints

### 6.8 Seletor de idioma na UI

- Disponível no drawer mobile e no navbar desktop (ícone de globo)
- Persistido como preferência no perfil do jogador (campo `locale` na tabela `players`)
- Disponível também nas páginas públicas sem login (FAQ, landing page, convite) via cookie

### 6.9 Impacto em testes

- Testes E2E: fixtures de texto precisam ser agnósticas ao idioma ou parametrizadas por locale
- Testes unitários: mocks de mensagens de erro do backend precisam usar os novos códigos snake_case

---

## 7. Dependências entre os dois eixos

Os dois eixos são **independentes** e podem ser implementados em paralelo ou em sequência. Porém, há uma interseção:

- O componente `PhoneInput` (Eixo A) terá labels e placeholders que precisam de tradução (Eixo B)
- Se o Eixo B for implementado primeiro, o `PhoneInput` já nasce traduzido
- Se o Eixo A for implementado primeiro, o `PhoneInput` começa em português e recebe i18n depois

**Recomendação**: implementar Eixo A primeiro (impacto mais direto em novos usuários internacionais), com Eixo B logo em seguida.

---

## 8. Riscos e mitigações

| Risco | Probabilidade | Mitigação |
|---|---|---|
| Migration quebrar login de usuários existentes | Alta | Testar migration em staging; garantir que `+55` + número existente faz lookup correto |
| Números duplicados pós-migration (ex: `5511...` e `+5511...` para o mesmo usuário) | Baixa | Checar duplicatas antes da migration e resolver manualmente |
| Bundle size aumentar com lista de países | Média | Usar apenas metadados leves (código + nome + flag emoji), sem libphonenumber completo |
| Tradução incompleta gerar UI mista (pt + en) | Alta | Fallback automático para pt-BR quando chave não traduzida; alertas em CI para chaves faltantes |
| Push notifications em idioma errado | Média | Armazenar locale no perfil; para usuários sem locale definido, usar pt-BR como fallback |
| Testes E2E quebrarem por textos hardcoded | Média | Refatorar assertions de texto para usar seletores por `data-testid` em vez de conteúdo |

---

## 9. Métricas de sucesso

- Zero erros de validação de telefone para números internacionais válidos
- Cobertura de tradução ≥ 95% das strings de UI nos fluxos críticos (autenticação, partidas, grupos)
- Tempo de carregamento do bundle não aumenta mais de 10KB por idioma adicional
- Usuários fora do Brasil conseguem completar cadastro + confirmação de presença sem suporte manual

---

## 10. Fases sugeridas de implementação

### Fase 1 — WhatsApp Internacional (Eixo A) ✅ Implementado
1. ✅ `migrations/030_whatsapp_e164.sql` — prefixar dados existentes com `+55`
2. ✅ `normalize_whatsapp()` no backend (E.164: `^\+[1-9]\d{6,14}$`) em `schemas/player.py`, com validators em `schemas/auth.py` e `schemas/invite.py`
3. ✅ Routers `auth.py` refatorado — `re.sub(r"\D", "")` removido, usa `body.whatsapp` pós-validação
4. ✅ `twilio_verify.py` — removido hardcode `+55`, usa número E.164 diretamente
5. ✅ `PhoneInput.svelte` — seletor de país (26 países) + input numérico; default 🇧🇷 +55; usado em login, register, invite, players admin
6. ✅ `formatWhatsapp()` e `whatsappLink()` em `utils.js` suportam E.164 e legado
7. ✅ Testes unitários atualizados para formato E.164 (`+5511999990001` etc.)

### Fase 2 — Infraestrutura i18n (Eixo B, parte 1) ✅ Implementado
1. ✅ Instalar e configurar Paraglide + `@inlang/paraglide-sveltekit`
2. ✅ Extrair todas as strings hardcoded para `messages/pt-BR.json`
3. ✅ Substituir `'pt-BR'` hardcoded por locale dinâmico em funções de data
4. ✅ Implementar seletor de idioma na UI + persistência em cookie/perfil
5. ✅ Adicionar campo `locale` na tabela `players`

### Fase 3 — Traduções (Eixo B, parte 2) ✅ Implementado
1. ✅ Traduzir fluxos críticos para `en` e `es`
2. ✅ Padronizar códigos de erro da API em snake_case inglês
3. ✅ Implementar push notifications multilíngue
4. ✅ Remover `_MONTHS_PT` hardcoded do backend
5. ✅ Atualizar testes E2E para seletores agnósticos de idioma
