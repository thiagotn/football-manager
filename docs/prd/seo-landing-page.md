# PRD — SEO da Landing Page
## Rachao.app · Aquisição Orgânica

| | |
|---|---|
| **Versão** | 1.0 |
| **Status** | ✅ Concluído |
| **Data** | Março de 2026 |

---

## 1. Contexto

A `/lp` é a principal página de aquisição do produto — é para onde usuários não autenticados são redirecionados ao acessar `/`. Atualmente a página não está indexada de forma otimizada no Google, o que limita o crescimento orgânico.

### 1.1 O que já estava correto (pré-v1)

| Item | Status |
|---|---|
| `lang="pt-BR"` no HTML global | ✅ `app.html` |
| Open Graph completo (type, locale, image, dimensions) | ✅ `app.html` |
| Twitter Card | ✅ `app.html` |
| Viewport e theme-color | ✅ `app.html` |
| Web App Manifest | ✅ `/manifest.webmanifest` |
| Imagem hero em WebP com fallback JPG | ✅ `/lp` |
| `fetchpriority="high"` na imagem acima da dobra | ✅ `/lp` |
| Hierarquia de headings `h2 → h3` coerente | ✅ `/lp` |

### 1.2 Itens implementados na v1

| Item | Impacto | Status |
|---|---|---|
| `<h1>` ausente na página | 🔴 Alto | ✅ Implementado |
| `<link rel="canonical">` ausente | 🔴 Alto | ✅ Implementado (Opção A — aponta para `/`) |
| Meta description genérica (sem palavras-chave) | 🟠 Médio | ✅ Implementado |
| JSON-LD Structured Data ausente | 🟠 Médio | ✅ Implementado (`WebApplication` + `Organization`) |
| `sitemap.xml` ausente | 🟠 Médio | ✅ Implementado e submetido ao Google Search Console |
| `robots.txt` ausente | 🟡 Baixo | ✅ Implementado |
| Imagem OG com proporção fora do padrão (1920×600) | 🟡 Baixo | 🔜 Pendente (fora do escopo v1) |

---

## 2. Problema

O Google não consegue identificar claramente o tema principal da página (sem `<h1>`), pode indexar URLs duplicadas (`/` e `/lp` como conteúdos distintos), e não tem structured data para gerar rich snippets. Isso reduz a visibilidade orgânica para quem busca termos como *"organizar pelada"*, *"controle de presença rachão"*, *"app futebol society"*.

---

## 3. Proposta

### 3.1 `<h1>` — Crítico

A página não tem `<h1>`. O heading principal para o Google deve ser o texto do hero. Solução: transformar o parágrafo principal do hero em `<h1>`, mantendo o logo como imagem decorativa.

**Antes:**
```svelte
<img src="/logo.png" alt="rachao.app" ... />
<p class="text-xl text-primary-100 ...">
  Organize suas partidas de futebol sem complicação.
</p>
```

**Depois:**
```svelte
<img src="/logo.png" alt="rachao.app" ... />
<h1 class="text-xl text-primary-100 max-w-xl mx-auto mb-3">
  Organize suas partidas de futebol sem complicação.
</h1>
```

---

### 3.2 `<link rel="canonical">` — Crítico

`/` redireciona para `/lp` para usuários não autenticados. Sem canonical, o Google pode tratar as duas URLs como conteúdo duplicado. A canonical deve apontar para a URL preferencial — e dado que a LP está em `/lp`, a URL canônica ideal é `/` (a raiz do domínio).

**Opção A (recomendada):** adicionar canonical apontando para `/` em `<svelte:head>` da `/lp`:
```svelte
<link rel="canonical" href="https://rachao.app/" />
```

**Opção B (mais limpa, maior esforço):** mover o conteúdo da LP para `src/routes/+page.svelte` e redirecionar `/lp → /`. A URL raiz já é o destino final de aquisição.

> Para v1, implementar Opção A. Opção B fica como refactor futuro.

---

### 3.3 Meta description com palavras-chave

A descrição atual é genérica e não cobre os termos que o público busca. Atualizar tanto no `svelte:head` da `/lp` quanto no `app.html` (fallback global).

**Atual:**
> "Organize grupos de futebol, convide jogadores e controle presenças em um clique."

**Proposta:**
> "rachao.app — organize sua pelada ou rachão de futebol society sem precisar instalar app. Confirme presenças, sorteie times e acompanhe estatísticas pelo celular."

*Entre 140–160 caracteres, inclui: pelada, rachão, futebol society, confirmar presenças, sorteio de times.*

---

### 3.4 JSON-LD Structured Data

Adicionar dois schemas no `<svelte:head>` da `/lp`:

**`WebApplication`** — para aparecer como app em resultados de busca:
```json
{
  "@context": "https://schema.org",
  "@type": "WebApplication",
  "name": "rachao.app",
  "url": "https://rachao.app",
  "description": "Organize sua pelada ou rachão de futebol. Confirme presenças, sorteie times e acompanhe estatísticas pelo celular.",
  "applicationCategory": "SportsApplication",
  "operatingSystem": "Web",
  "inLanguage": "pt-BR",
  "offers": {
    "@type": "Offer",
    "price": "0",
    "priceCurrency": "BRL",
    "description": "Plano gratuito disponível"
  }
}
```

**`Organization`** — para o painel de conhecimento do Google:
```json
{
  "@context": "https://schema.org",
  "@type": "Organization",
  "name": "rachao.app",
  "url": "https://rachao.app",
  "logo": "https://rachao.app/logo.png",
  "sameAs": []
}
```

---

### 3.5 `sitemap.xml`

Criar `football-frontend/static/sitemap.xml` com as páginas públicas indexáveis:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url>
    <loc>https://rachao.app/</loc>
    <changefreq>monthly</changefreq>
    <priority>1.0</priority>
  </url>
  <url>
    <loc>https://rachao.app/lp</loc>
    <changefreq>monthly</changefreq>
    <priority>0.9</priority>
  </url>
  <url>
    <loc>https://rachao.app/faq</loc>
    <changefreq>monthly</changefreq>
    <priority>0.7</priority>
  </url>
  <url>
    <loc>https://rachao.app/terms</loc>
    <changefreq>yearly</changefreq>
    <priority>0.3</priority>
  </url>
  <url>
    <loc>https://rachao.app/privacy</loc>
    <changefreq>yearly</changefreq>
    <priority>0.3</priority>
  </url>
</urlset>
```

---

### 3.6 `robots.txt`

Criar `football-frontend/static/robots.txt` para bloquear páginas internas de indexação e declarar o sitemap:

```
User-agent: *
Allow: /
Disallow: /admin/
Disallow: /profile/
Disallow: /groups/
Disallow: /match/
Disallow: /players/
Disallow: /account/
Disallow: /invite/
Disallow: /register
Disallow: /login

Sitemap: https://rachao.app/sitemap.xml
```

---

## 4. Fora do Escopo desta v1

| Item | Motivo |
|---|---|
| Mover LP para `/` (Opção B do canonical) | Refactor de rotas — risco de quebrar links existentes |
| Imagem OG 1200×630 (proporção padrão) | Requer geração de novo asset de design |
| FAQ section com schema `FAQPage` | Requer nova seção de conteúdo na LP |
| Internacionalização (`hreflang`) | App é pt-BR only por ora |
| Google Analytics | Infra/configuração fora do escopo de código |

---

## 5. Alterações Técnicas

| Arquivo | Tipo de alteração |
|---|---|
| `src/routes/lp/+page.svelte` | `<h1>` no hero, `<link rel="canonical">`, meta description, JSON-LD em `<svelte:head>` |
| `src/app.html` | Atualizar meta description global (fallback) |
| `static/sitemap.xml` | Novo arquivo |
| `static/robots.txt` | Novo arquivo |

Nenhuma alteração de backend necessária. Nenhuma migração de banco.

---

## 6. Impacto Esperado

| Métrica | Antes | Depois (estimado) |
|---|---|---|
| Indexação da LP no Google | Parcial / sem h1 | Completa e semanticamente correta |
| Rich snippets | Nenhum | WebApplication nos resultados de busca |
| Duplicate content `/` vs `/lp` | Risco presente | Eliminado via canonical |
| Páginas internas indexáveis | Sem controle | Bloqueadas via robots.txt |
| Rastreamento do Googlebot | Sem guia | Orientado via sitemap |

---

## 7. Dependências

- Nenhuma dependência de backend ✅
- Nenhuma nova biblioteca necessária ✅
- Conteúdo da meta description aprovado ✅

---

## 8. Configuração Google Search Console

| Ação | Status |
|---|---|
| Verificação de propriedade (arquivo HTML `google-site-verification-*.html`) | ✅ Concluído |
| Submissão do `sitemap.xml` | ✅ Concluído |
