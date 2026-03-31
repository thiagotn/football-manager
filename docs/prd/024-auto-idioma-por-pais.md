# PRD — Troca Automática de Idioma pelo Código de País no Login e Cadastro

## 1. Contexto

As telas `/login` e `/register` usam o componente `PhoneInput.svelte`, que exibe um seletor de código de país antes do campo de número. O app já suporta três idiomas (`pt-BR`, `en`, `es`) via `i18n.ts`, com persistência em `localStorage` e detecção inicial via `navigator.language`.

No estado atual, selecionar um país no PhoneInput não tem nenhum efeito sobre o idioma da interface — o usuário precisa usar o seletor de idioma explicitamente após o login.

---

## 2. Problema

### Dor atual
- Um usuário argentino que acessa o app pela primeira vez seleciona `+54 (AR)` no seletor, mas a interface permanece em português
- A detecção automática via `navigator.language` é imprecisa em contextos mobile (especialmente em PWAs e webviews)
- O idioma só muda depois do login, quando o usuário encontra o seletor de idioma no menu — caminho não óbvio

### Impacto
- Fricção desnecessária para usuários não-brasileiros na primeira interação com o app
- O seletor de país já é um sinal explícito de origem — ignorar esse sinal é uma oportunidade perdida de personalização imediata

---

## 3. Solução Proposta

Quando o usuário altera o código de país no `PhoneInput` das telas de login **ou cadastro**, o idioma da interface muda automaticamente com base no país selecionado, de acordo com a tabela de mapeamento abaixo.

### Tabela de mapeamento: país → idioma

| Países | Idioma |
|--------|--------|
| `BR` | `pt-BR` (Português Brasileiro) |
| `ES`, `AR`, `MX`, `CL`, `CO`, `PE`, `UY`, `PY`, `BO`, `VE`, `EC` | `es` (Espanhol) |
| Todos os demais (`US`, `CA`, `GB`, `DE`, `FR`, `IT`, `PT`, `AU`, `JP`, `CN`, `IN`, `NG`, `ZA`) | `en` (Inglês) |

> **Nota:** Portugal (`PT`) mapeia para `en` por não ser suportado como locale separado. Caso o PT seja adicionado futuramente, o mapeamento pode ser atualizado.

### Regra de ativação

A mudança automática de idioma **só ocorre se o usuário não tiver uma preferência salva explicitamente** (via seletor de idioma). Isso evita sobrescrever uma escolha intencional anterior.

A distinção é feita por uma flag de origem no `localStorage`:

| Chave `rachao_locale_source` | Significado |
|-------------------------------|-------------|
| `"auto"` | Locale foi definido automaticamente (por `navigator.language` ou pela troca de país) — pode ser sobrescrito |
| `"user"` | Locale foi definido manualmente pelo usuário via seletor — **não deve ser sobrescrito** |
| ausente | Nenhuma preferência salva — troca automática é permitida |

**Fluxo:**
1. Usuário abre `/login` sem preferência salva → idioma definido por `navigator.language` (source: `"auto"`)
2. Usuário seleciona país no PhoneInput → idioma muda automaticamente (source: `"auto"`)
3. Usuário usa o seletor de idioma explicitamente → idioma muda (source: `"user"`)
4. Na próxima visita, `initLocale()` lê o locale salvo. Se source for `"user"`, respeita. Se `"auto"` ou ausente, roda a detecção normal por `navigator.language`.

---

## 4. Escopo

### Onde se aplica
- **`/login`** — campo de telefone no formulário de login e nos dois campos do fluxo "Esqueci minha senha"
- **`/register`** — campo de telefone no passo de informar o WhatsApp (step `whatsapp`)

A regra de ativação é idêntica em ambas as telas: respeita preferência explícita do usuário (`source: 'user'`), e a função `handleCountryChange` pode ser extraída para um módulo compartilhado ou duplicada em cada página (dado o tamanho reduzido).

### Fora do escopo
- PhoneInput em outros contextos (configuração de conta, etc.)
- Mudança retroativa de idioma após o login
- Adicionar novos locales (`pt-PT`, por exemplo)

---

## 5. Impacto por Camada

### `$lib/i18n.ts`

Adicionar o conceito de `locale_source`:

```typescript
const LOCALE_SOURCE_KEY = 'rachao_locale_source';
type LocaleSource = 'user' | 'auto';

export async function setLocale(newLocale: Locale, source: LocaleSource = 'user'): Promise<void> {
  await loadMessages(newLocale);
  locale.set(newLocale);
  localStorage.setItem(LOCALE_KEY, newLocale);
  localStorage.setItem(LOCALE_SOURCE_KEY, source);
}

export function isLocaleUserChosen(): boolean {
  return localStorage.getItem(LOCALE_SOURCE_KEY) === 'user';
}
```

`initLocale()` não precisa de alteração — ele já lê o `LOCALE_KEY` salvo e o respeita.

### `PhoneInput.svelte`

Adicionar prop de callback para expor o código do país selecionado ao componente pai:

```typescript
interface Props {
  // ... props existentes ...
  oncountrychange?: (countryCode: string) => void;
}
```

Acionar o callback no `$effect` ou via `onchange` do `<select>`:

```svelte
<select
  bind:value={selectedDial}
  onchange={() => oncountrychange?.(selectedCountryCode)}
  ...
>
```

> O `PhoneInput` atualmente trabalha com `dial` (ex: `"55"`) como `value` do select. Para expor o código ISO (ex: `"BR"`), é preciso rastrear o `code` do país selecionado — não apenas o `dial`.

### `/login/+page.svelte` e `/register/+page.svelte`

A mesma lógica de mapeamento se aplica em ambas as páginas. Cada uma importa `setLocale`, `isLocaleUserChosen` e `type Locale` de `$lib/i18n` e define `handleCountryChange` localmente:

```typescript
const SPANISH_COUNTRIES = new Set(['ES','AR','MX','CL','CO','PE','UY','PY','BO','VE','EC']);

function handleCountryChange(countryCode: string) {
  if (isLocaleUserChosen()) return; // respeita preferência explícita

  let newLocale: Locale;
  if (countryCode === 'BR') newLocale = 'pt-BR';
  else if (SPANISH_COUNTRIES.has(countryCode)) newLocale = 'es';
  else newLocale = 'en';

  setLocale(newLocale, 'auto');
}
```

Em `/register`, o `PhoneInput` fica no step `whatsapp` — o callback deve ser passado somente nesse campo:

```svelte
<PhoneInput id="whatsapp" bind:value={whatsapp} placeholder="11999990000" required
  oncountrychange={handleCountryChange} />
```

---

## 6. Detalhe Técnico: PhoneInput usa `dial`, não `code`

O select atual usa `country.dial` como `value`:

```svelte
<option value={country.dial}>
```

Isso significa que `selectedDial` guarda `"55"`, não `"BR"`. Para expor o código ISO ao callback, é necessário:

**Opção 1 (mais simples):** mudar o `value` do `<option>` para `country.code` e derivar o `dial` a partir do país selecionado — mas isso quebra a lógica de composição do número de telefone.

**Opção 2 (sem quebra):** manter `selectedDial` como está, mas adicionar um `$derived` para o país selecionado:

```typescript
let selectedCountry = $derived(COUNTRIES.find(c => c.dial === selectedDial) ?? COUNTRIES[0]);
```

E acionar o callback com `selectedCountry.code`. Atenção: `dial: '1'` é compartilhado por `US` e `CA` — o mapeamento resultaria sempre em `en` para ambos, o que está correto.

---

## 7. Critérios de Aceitação

- [x] Selecionar `BR` no PhoneInput do login **ou cadastro** muda o idioma para `pt-BR`
- [x] Selecionar `AR`, `MX`, `CL`, `CO`, `PE`, `UY`, `PY`, `BO`, `VE` ou `EC` muda o idioma para `es`
- [x] Selecionar `ES` (Espanha) muda o idioma para `es`
- [x] Selecionar qualquer outro país (US, GB, DE, JP, etc.) muda o idioma para `en`
- [x] Se o usuário já tiver trocado o idioma manualmente via seletor, a troca automática **não** ocorre
- [x] A mudança automática persiste na próxima visita como qualquer outra troca de idioma (salva em `localStorage`)
- [x] Após trocar de idioma manualmente, uma nova seleção de país **não** sobrescreve a preferência
- [x] O comportamento **não** se aplica a outros usos do PhoneInput fora de `/login` e `/register`

---

## 8. Status de Internacionalização

### `/login` — já implementado ✅

Todos os campos, labels, botões e mensagens da tela de login já usam `$t()` com traduções completas nos 3 idiomas. O `<title>` foi corrigido para `$t('login.page_title')` na implementação anterior.

### `/register` — completamente internacionalizada ✅

Todos os campos, labels, botões, mensagens de erro e os 3 steps do fluxo de cadastro usam `$t()` com chaves `register.*` traduzidas nos 3 idiomas.

| Elemento | Status |
|----------|--------|
| Todos os labels, botões e mensagens (`register.*`) | ✅ internacionalizados |
| Label "WhatsApp" e nome de marca | ℹ️ hardcoded — correto, é nome de marca |
| `<title>` da página | ✅ usando `$t('register.page_title')` |

---

## 9. O que NÃO está no escopo

- **Detecção de idioma pelo IP/geolocalização** — desnecessário, o país já é informado pelo usuário
- **Locale `pt-PT`** — Portugal permanece mapeado para `en` até que um novo locale seja adicionado
- **Alteração do seletor de idioma pós-login** — o comportamento atual já funciona corretamente
- **PhoneInput em configuração de conta** — o usuário já está autenticado e pode usar o seletor de idioma diretamente
