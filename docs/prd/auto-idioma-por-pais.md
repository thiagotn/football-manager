# PRD — Troca Automática de Idioma pelo Código de País no Login

## 1. Contexto

A tela `/login` usa o componente `PhoneInput.svelte`, que exibe um seletor de código de país antes do campo de número. O app já suporta três idiomas (`pt-BR`, `en`, `es`) via `i18n.ts`, com persistência em `localStorage` e detecção inicial via `navigator.language`.

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

Quando o usuário altera o código de país no `PhoneInput` da tela de login, o idioma da interface muda automaticamente com base no país selecionado, de acordo com a tabela de mapeamento abaixo.

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
- **Apenas `/login`** — o PhoneInput também aparece no fluxo de cadastro (`/register` ou dentro do modal de cadastro), mas ali o usuário já está interagindo com o app e pode já ter trocado o idioma manualmente. O gatilho de país só faz sentido na tela inicial de acesso.

### Fora do escopo
- PhoneInput em outros contextos (registro, configuração de conta)
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

### `/login/+page.svelte`

Adicionar a lógica de mapeamento e chamar `setLocale` quando a prop callback for acionada:

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

- [ ] Selecionar `BR` no PhoneInput do login muda o idioma para `pt-BR`
- [ ] Selecionar `AR`, `MX`, `CL`, `CO`, `PE`, `UY`, `PY`, `BO`, `VE` ou `EC` muda o idioma para `es`
- [ ] Selecionar `ES` (Espanha) muda o idioma para `es`
- [ ] Selecionar qualquer outro país (US, GB, DE, JP, etc.) muda o idioma para `en`
- [ ] Se o usuário já tiver trocado o idioma manualmente via seletor, a troca automática **não** ocorre
- [ ] A mudança automática persiste na próxima visita como qualquer outra troca de idioma (salva em `localStorage`)
- [ ] Após trocar de idioma manualmente, uma nova seleção de país **não** sobrescreve a preferência
- [ ] O comportamento se aplica **apenas à tela `/login`** — outros usos do PhoneInput não são afetados

---

## 8. Status de Internacionalização da Tela de Login

Antes de implementar a troca automática de idioma, é relevante confirmar se a própria tela de login já responde corretamente a uma mudança de locale.

### Conclusão: a tela já está completamente internacionalizada

Todos os campos, labels, botões, mensagens de erro e textos de apoio da tela `/login` já usam `$t()` e possuem traduções completas nos 3 idiomas (`pt-BR`, `en`, `es`). Ao chamar `setLocale()` durante a seleção do país, a interface toda atualiza imediatamente e de forma correta.

| Elemento | Status |
|----------|--------|
| Subtítulo da tela | ✅ `$t('login.subtitle')` |
| Label "Senha" | ✅ `$t('login.password_label')` |
| Botão "Entrar" | ✅ `$t('login.submit')` |
| Fluxo "Esqueci minha senha" (todos os passos) | ✅ internacionalizado |
| Mensagens de erro e validação | ✅ internacionalizadas |
| Label "WhatsApp" (campo de telefone) | ℹ️ hardcoded — é nome de marca, não requer tradução |
| `<title>` da página (`Login — rachao.app`) | ⚠️ hardcoded — ajuste menor recomendado (ver abaixo) |

### Ajuste menor recomendado: `<title>` da página

A tag `<svelte:head><title>Login — rachao.app</title></svelte:head>` está hardcoded. Com a troca de idioma ocorrendo antes do login, o título da aba do navegador seria o único elemento fora do locale. Correção trivial:

```typescript
// Adicionar chave nos 3 arquivos de mensagens:
"login.page_title": "Login — rachao.app"  // igual nos 3 (nome próprio)
```

```svelte
<svelte:head><title>{$t('login.page_title')}</title></svelte:head>
```

> O título em si não precisa de tradução (nome da marca), mas envolvê-lo em `$t()` garante consistência arquitetural e evita que ele fique "fora do sistema" de i18n.

Esse ajuste **faz parte do escopo desta implementação**.

---

## 9. O que NÃO está no escopo

- **Detecção de idioma pelo IP/geolocalização** — desnecessário, o país já é informado pelo usuário
- **Troca de idioma no fluxo de cadastro** — pode ser adicionado em iteração futura
- **Locale `pt-PT`** — Portugal permanece mapeado para `en` até que um novo locale seja adicionado
- **Alteração do seletor de idioma pós-login** — o comportamento atual já funciona corretamente
