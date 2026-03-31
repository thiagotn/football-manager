# PRD — Landing Page Unificada

## 1. Contexto

O rachao.app tem hoje duas landing pages independentes:

- **`/lp`** — foco em organizadores: gestão de grupos, presenças, sorteio de times.
- **`/jogar`** — foco em jogadores buscando rachão: discover, ranking, Rachão Score.

As duas páginas têm hero separado, narrativas que não se complementam e duplicam esforço de manutenção. A divisão também cria um problema de roteamento de tráfego: campanha ou link orgânico não entrega a mensagem certa para nenhum dos dois perfis de usuário.

A proposta é **substituir `/lp` por uma única LP unificada** e **descartar `/jogar`**, consolidando ambas as narrativas em um único arquivo mantido.

---

## 2. Objetivo

Reescrever `/lp` como a landing page definitiva do produto, que:

1. Apresenta o produto sob dois ângulos complementares: **organizar** e **jogar**
2. Permite que cada visitante se identifique e encontre os argumentos relevantes sem navegar para outra página
3. Converte melhor ao reduzir a fricção de descoberta de valor
4. Consolida manutenção de SEO, meta tags e copy em um único arquivo

---

## 3. Decisões definidas

| # | Decisão | Definição |
|---|---------|-----------|
| D1 | Rota | **`/lp`** — a nova LP substitui o conteúdo atual da rota, sem criar nova rota |
| D2 | `/jogar` | **Remover** — arquivo descartado, rota deixa de existir |
| D3 | Navegação entre perfis | **Seções com âncora** (`#organizar`, `#jogar`) — indexável por SEO, sem JS extra |
| D4 | Hero | **Dois cards distintos** no hero, um por perfil, cada um com seu CTA |

---

## 4. Rota e SEO

| Item | Definição |
|------|-----------|
| Rota | `/lp` (reescrita in-place) |
| Título | `rachao.app — Organize ou encontre seu rachão de futebol` |
| Meta description | `Gerencie seu rachão ou encontre uma pelada aberta perto de você. Confirme presenças, sorteie times, vote no melhor da partida. Grátis.` |
| Canonical | `https://rachao.app/lp` |
| Schema.org | `WebApplication` + `SportsEvent` |
| OG Image | `background-login.png` (campo de society — já em uso) |

**Palavras-chave alvo:**
- rachão de futebol society
- organizar pelada online
- encontrar rachão com vaga
- confirmar presença pelada
- futebol society gratuito

---

## 5. Linguagem visual

### Problema atual

As LPs atuais (`/lp` e `/jogar`) alternam entre `bg-white` e `bg-gray-50` — fundos planos que resultam numa página sem personalidade, especialmente nas seções intermediárias (Discover, Ranking, Score). O contraste quase inexistente entre seções torna a leitura monótona e não transmite a energia do produto.

### Diretrizes para a nova LP

A página deve ser majoritariamente **escura**, consistente com a identidade visual do app (navbar, PageBackground, match pages). Fundos claros ficam reservados para elementos de destaque pontual (cards, tabelas), não para seções inteiras.

**Paleta de seções:**

| Seção | Fundo | Efeito |
|-------|-------|--------|
| Hero | `background-login.png` + gradiente `from-gray-900/90 to-gray-900/70` | Imagem de campo com overlay |
| Organizadores | `bg-gray-900` ou `bg-primary-950` | Escuro sólido |
| Jogadores | `bg-gray-800` | Ligeiramente mais claro que anterior — separa sem usar branco |
| Por que é grátis | `bg-primary-900` com sutil padrão de hexágonos ou noise SVG | Toque de textura |
| Como funciona | `bg-gray-900` | Escuro consistente |
| Planos | `bg-gray-950` | O mais escuro — faz os cards brancos/coloridos se destacarem |
| CTA Final | Gradiente `from-primary-800 to-primary-950` | Gradiente verde profundo |
| Footer | `bg-gray-950` | Contínuo com o CTA |

**Tipografia sobre fundo escuro:**
- Títulos: `text-white`
- Subtítulos / corpo: `text-white/70` ou `text-gray-300`
- Badges de seção: `text-primary-400` uppercase tracking-widest
- Links CTA secundários: `text-primary-400 hover:text-primary-300`

**Cards sobre fundo escuro:**
- Cards de mock (Discover, Ranking, Score, planos): `bg-gray-800 border border-gray-700` ou `bg-white` com sombra — contraste máximo contra o fundo escuro
- Cards de feature (grid de funcionalidades): `bg-gray-800/60 border border-gray-700/50` com hover sutil

**Separadores entre seções:**
- Evitar `<div class="h-px bg-gray-100">` (invisível em fundo escuro)
- Usar variação de tom entre seções adjacentes como único separador
- Em casos necessários: `border-t border-gray-700/50`

---

## 6. Estrutura da página

### Seção 1 — Hero dual

**Manchete:** "Seu rachão começa aqui."
**Submanchete:** "Para quem organiza e para quem quer jogar — na mesma plataforma, gratuita."

Dois cards lado a lado (desktop) / empilhados (mobile):

```
┌─────────────────────────┐  ┌─────────────────────────┐
│  🏟️  Sou organizador    │  │  ⚽  Quero jogar        │
│                         │  │                         │
│  Gerencie grupos,       │  │  Encontre rachões        │
│  presenças e times.     │  │  abertos perto de você.  │
│                         │  │                         │
│  [Criar meu grupo]      │  │  [Ver rachões abertos]  │
└─────────────────────────┘  └─────────────────────────┘
```

Fundo: `background-login.png` + overlay `from-gray-900/90 to-gray-900/70`.
Logo centralizado acima dos cards, tamanho `w-56 sm:w-72`.
Os dois cards do hero devem ter `bg-white/10 border border-white/20 backdrop-blur-sm` — vidro fosco sobre a imagem de fundo.

**Selos:** `✓ Gratuito · ✓ Sem instalar app · ✓ Direto pelo celular` — texto `text-white/60`, fonte pequena.

---

### Seção 2 — Para organizadores

**Fundo:** `bg-gray-900`
**Âncora:** `#organizar`
**Badge:** "Para organizadores" — `text-primary-400`
**Título:** "Chega de contar resposta por resposta no WhatsApp."
**Subtítulo:** "Crie seu grupo, abra a partida e acompanhe quem confirmou — tudo em um link."

**Funcionalidades em grid 2×3:**

| Feature | Descrição curta |
|---------|----------------|
| 👥 Grupos organizados | Crie grupos para cada turma. Convide via link. |
| ✅ Presenças em tempo real | Veja confirmados, recusados e pendentes na hora. |
| 🎲 Sorteio equilibrado | Times montados por nível de habilidade, com goleiro. |
| 📲 Compartilhamento WhatsApp | Resumo completo da partida em um clique. |
| 🏆 Votação pós-partida | Jogadores elegem o melhor de campo automaticamente. |
| 🔄 Rachão recorrente | Próxima partida criada automaticamente toda semana. |

**Cards de feature:** `bg-gray-800/60 border border-gray-700/50 rounded-2xl` — visíveis sobre o fundo escuro.
**Mock ilustrativo** (à direita no desktop): exemplo de resultado de votação pós-partida em card `bg-gray-800 border border-gray-700`.

**CTA:** `Criar meu grupo grátis →` → `/register` — botão `bg-primary-500 hover:bg-primary-400 text-white`

---

### Seção 3 — Para jogadores

**Fundo:** `bg-gray-800`
**Âncora:** `#jogar`
**Badge:** "Para jogadores" — `text-emerald-400`
**Título:** "Tem rachão com vaga perto de você."
**Subtítulo:** "Explore partidas abertas, confirme presença e acumule seu Rachão Score."

**Três destaques visuais:**

**A) Partidas abertas** — 2–3 mock cards do Discover:
```
⚽ Futebol GQC · Sex 20:30 · 8/20 confirmados  [Quero participar]
⚽ Pelada dos Brothers · Dom 08:00 · 5/22       [Quero participar]
```
CTA link: `Ver todos os rachões abertos →` → `/discover`

**B) Ranking da plataforma** — mock compacto do pódio (🥇🥈🥉 com nomes e pontos).
CTA link: `Ver ranking completo →` → `/ranking`

**C) Rachão Score** — mock do card de stats pessoais: avatar + nome + estrelas + métricas (rachões, presença, streak).
Destaque: "Seu histórico é público. Organizadores podem te encontrar pelo seu Score."

**Mock cards do Discover:** `bg-gray-900 border border-gray-700` — cartões escuros sobre fundo cinza médio, alto contraste.
**Mock pódio e Score card:** idem — fundo escuro, texto branco, métricas em `text-primary-400`.

**CTA principal:** `Criar conta grátis →` → `/register`

---

### Seção 4 — Por que é grátis?

**Fundo:** `bg-primary-900` com noise SVG sutil ou padrão de hexágonos em `opacity-5`
**Título:** "Jogar é sempre grátis. Organizar também começa grátis."
**Subtítulo:** "Quem paga (opcionalmente) são organizadores que precisam de recursos avançados para grupos grandes."

Dois painéis lado a lado em cards `bg-primary-800/50 border border-primary-700/50 rounded-2xl`:

| Para jogadores | Para organizadores |
|----------------|-------------------|
| ✓ Encontrar rachões | ✓ 1 grupo grátis |
| ✓ Confirmar presença | ✓ Até 30 membros |
| ✓ Votar no melhor | ✓ Partidas ilimitadas |
| ✓ Rachão Score | ✓ Sorteio de times |
| ✓ Aparecer no Ranking | ✓ Compartilhamento |
| ✓ Histórico completo | ✓ Votação pós-partida |

---

### Seção 5 — Como funciona (3 passos — dual)

**Fundo:** `bg-gray-900`
**Título:** "Em 3 passos você já tá no campo"

Toggle simples (Svelte `$state`) entre dois fluxos:

**Organizar:**
1. Crie seu grupo — nome, WhatsApp, pronto.
2. Abra uma partida — data, local, horário.
3. Compartilhe o link — jogadores confirmam pelo celular.

**Jogar:**
1. Crie sua conta — WhatsApp + nome, sem cartão.
2. Explore os rachões — filtre por cidade e data.
3. Confirme presença — apareça na hora e jogue.

---

### Seção 6 — Planos

**Título:** "Simples e transparente"
**Subtítulo:** "Comece grátis. Faça upgrade quando seu grupo crescer."

**Fundo geral da seção:** `bg-gray-950`

#### Estrutura visual

Dois blocos verticalmente encadeados sobre `bg-gray-950`:

---

**Bloco A — Cards de preço (topo)**

Três cards `bg-gray-800 border border-gray-700 rounded-2xl` sobre `bg-gray-950`.
O card Básico recebe `border-primary-500 ring-1 ring-primary-500` para destaque.
Sem blur nos planos pagos (já disponíveis):

```
┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
│  Grátis         │  │  Básico  ★      │  │  Pro            │
│  Para começar   │  │  Para grupos    │  │  Para múltiplos │
│                 │  │  que crescem    │  │  grupos         │
│  R$ 0           │  │  R$ 19,90/mês  │  │  R$ 39,90/mês  │
│                 │  │  (R$ 199/ano)   │  │  (R$ 399/ano)  │
│  [Cadastrar]    │  │  [Assinar]      │  │  [Assinar]      │
└─────────────────┘  └─────────────────┘  └─────────────────┘
```

O plano Básico deve ter destaque visual (badge "Mais popular", borda colorida).
Toggle mensal/anual no canto superior direito do bloco — atualiza os preços via `$state`.

---

**Bloco B — Tabela comparativa (abaixo dos cards)**

Tabela sobre `bg-gray-950`, bordas `divide-y divide-gray-800`, cabeçalho com nomes dos planos em `text-white`.
Linhas alternadas `hover:bg-gray-800/30` para facilitar leitura.
Valores destacados (limites que diferem) em `text-white font-semibold`; valores "iguais em todos" em `text-primary-400` (✓); indisponíveis em `text-gray-600` (—).

Tabela com cabeçalho fixo mostrando os três planos, linhas por funcionalidade:

| Funcionalidade | Grátis | Básico | Pro |
|----------------|--------|--------|-----|
| **Grupos** | 1 | 3 | 10 |
| **Membros por grupo** | 30 | 50 | Ilimitado |
| **Partidas abertas simultâneas** | 3 | Ilimitadas | Ilimitadas |
| **Histórico de partidas** | 30 dias | 6 meses | Sem limite |
| Convites por link | ✓ | ✓ | ✓ |
| Confirmação de presença | ✓ | ✓ | ✓ |
| Sorteio equilibrado de times | ✓ | ✓ | ✓ |
| Votação pós-partida | ✓ | ✓ | ✓ |
| Estatísticas do grupo | ✓ | ✓ | ✓ |
| **Controle financeiro** | — | Básico | Avançado |
| **Suporte** | — | Prioritário | Prioritário |

Legenda: ✓ = incluído · — = não disponível · valores em negrito = diferem entre planos

**Comportamento mobile:** tabela com scroll horizontal ou exibição em abas (uma coluna por plano).

---

### Seção 7 — CTA Final

**Fundo:** gradiente `from-primary-800 via-primary-900 to-gray-950` — transição suave do verde para o preto do footer
**Título:** "Tá esperando o quê?"
**Subtítulo:** "Tem rachão acontecendo essa semana. Crie sua conta e apareça."

**Dois CTAs:**
- `Criar conta grátis` → `/register`
- `Ver rachões disponíveis` → `/discover`

---

### Footer

**Fundo:** `bg-gray-950` — contínuo com a seção CTA, sem separação visível.
Mesmo conteúdo do footer da `/lp` atual: logo, copyright, Termos, Privacidade, FAQ, Status, Entrar.

---

## 6. Impacto técnico

| Camada | Trabalho |
|--------|---------|
| Frontend | Reescrever `src/routes/lp/+page.svelte` in-place |
| Frontend | Remover `src/routes/jogar/+page.svelte` |
| Frontend | Seção §5 requer toggle `$state` simples entre os dois fluxos |
| i18n | Consolidar chaves `jogar.*` necessárias em `lp.*` (ou reusar namespace `jogar.*` já existente) |
| SEO | Atualizar meta tags, og:image, canonical, schema.org na `/lp` |
| `+layout.svelte` | Remover `/jogar` de `PUBLIC_ROUTES` |
| `Navbar.svelte` | Sem alteração — `/jogar` não aparece no menu |
| Migration | Nenhuma |
| Backend | Nenhuma — `/discover` e `/players/[id]` já públicos |

---

## 7. O que reaproveitar

| Elemento | Origem | Destino |
|----------|--------|---------|
| Hero com `background-login.png` + gradiente | `/lp` e `/jogar` | Seção 1 |
| Logo `w-56 sm:w-72` | `/jogar` | Seção 1 |
| Mock cards do Discover | `/jogar` | Seção 3-A |
| Mock pódio do Ranking | `/jogar` | Seção 3-B |
| Mock Rachão Score card | `/jogar` | Seção 3-C |
| Grid de funcionalidades do organizador | `/lp` | Seção 2 |
| Mock resultado de votação | `/lp` | Seção 2 (ilustração) |
| Cards de planos | `/lp` | Seção 6 |
| CTA final verde | `/lp` e `/jogar` | Seção 7 |
| Footer | `/lp` | Footer |

Estimativa de reúso: ~75% do conteúdo já existe nas duas LPs.

---

## 8. Fora de escopo

- Dados dinâmicos reais nos mocks (ranking ao vivo, partidas reais)
- Filtro por cidade no hero
- Depoimentos de usuários
- Integração com Google Maps
- A/B test entre LP atual e nova
