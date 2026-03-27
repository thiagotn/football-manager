# PRD — Landing Page de Crescimento Orgânico (Jogadores)

## 1. Contexto

A LP existente (`/lp`) é focada em **organizadores**: gestão de grupos, presenças, sorteio de times. O call-to-action principal é "cadastrar grátis para organizar seu rachão".

Esse ângulo não converte bem para um jogador que chegou via busca orgânica procurando **onde jogar** — ele não quer organizar nada, quer encontrar uma vaga. Para esse perfil, a LP atual fala a língua errada.

O rachao.app já tem os ingredientes certos para capturar esse tráfego:
- Rachões públicos com vagas abertas (`/discover`)
- Ranking geral da plataforma (`/ranking`) — acessível sem login
- Rachão Score — estatísticas pessoais acumuladas por partida

---

## 2. Objetivo

Criar uma nova landing page focada em **aquisição orgânica de jogadores**, coexistindo com a LP atual (`/lp`) que permanece intacta para o público de organizadores. Abrir `/discover` e o Rachão Score para acesso público, tornando-os argumentos concretos de engajamento pré-cadastro.

---

## 3. Decisões definidas

| # | Decisão | Definição |
|---|---------|-----------|
| D1 | Rota | **Nova rota `/jogar`** — LP atual (`/lp`) permanece intacta |
| D2 | Discover público | **Sim** — `/discover` acessível sem login. Exige mudança no frontend (remover redirect) e no backend (endpoint público) |
| D3 | Rachão Score público | **Sim** — stats do jogador visíveis publicamente via URL (ex: `/players/[id]`). Exige novo endpoint e rota pública |

---

## 4. Rota e SEO

| Item | Definição |
|------|-----------|
| Rota | `/jogar` (nova, LP atual `/lp` permanece) |
| Título | `Encontre um rachão perto de você — grátis \| rachao.app` |
| Meta description | `Junte-se a rachões abertos com vagas disponíveis. Cadastro gratuito, sem app. Acumule seu Rachão Score e dispute o ranking da plataforma.` |
| Canonical | `https://rachao.app/jogar` |
| Schema.org | `SportsEvent` + `WebApplication` |

**Palavras-chave alvo:**
- rachão aberto com vaga
- onde jogar futebol society
- pelada aberta cadastro
- futebol society gratuito
- encontrar rachão

---

## 5. Estrutura da página

### Seção 1 — Hero
**Manchete:** "Quer jogar? Tem rachão com vaga pra você."
**Submanchete:** "Encontre partidas abertas perto de você, confirme presença e apareça para jogar. Grátis, sem instalar nada."

**CTAs:**
- Primário: `Ver rachões disponíveis` → `/discover` (público, sem login — D2)
- Secundário: `Criar conta grátis` → `/register`

**Visual:** foto/banner de campo de society em ação. Overlay escuro com o logo.

**Selos de confiança:** `✓ Gratuito · ✓ Sem app · ✓ Partidas acontecendo agora`

---

### Seção 2 — Rachões com vagas (Discover preview)
**Título:** "Partidas abertas agora"
**Subtítulo:** "Grupos públicos que aceitam novos jogadores. Encontre um perto de você."

Mockup visual de 2–3 cards de partida (estilo da tela `/discover`):
```
┌──────────────────────────────┐
│ ⚽ Futebol GQC               │
│ Sex, 4 abr · 20:30 – 22:00  │
│ 📍 BarraSoccer               │
│ ✅ 8/20 confirmados          │
│ [Quero participar]           │
└──────────────────────────────┘
```

**CTA:** `Ver todos os rachões abertos →` → `/discover`

---

### Seção 3 — Ranking da Plataforma
**Título:** "Os melhores do Brasil estão aqui"
**Subtítulo:** "A cada rachão, os jogadores votam nos melhores da partida. Os pontos se acumulam no ranking geral."

Mockup visual do pódio (estilo `/ranking`):
```
        🥇 Thiago (Thiagol)
    🥈 Lucas          🥉 Dudu
   280 pts           241 pts
  312 pts
```

**Destaque:** "Você também vai aparecer nesse ranking. Cadastro gratuito."

**CTA:** `Ver ranking completo →` → `/ranking`

---

### Seção 4 — Rachão Score
**Título:** "Construa sua reputação no campo"
**Subtítulo:** "Cada rachão que você joga conta. Presenças, votos recebidos, sequências — tudo vira estatística no seu perfil público."

Mockup visual do card de stats pessoais:
```
┌──────────────────────────────────┐
│ [Avatar]  Thiago (Thiagol)       │
│           ⭐⭐⭐⭐⭐ Habilidade   │
│                                  │
│  47 rachões · 89% presença       │
│  🔥 12 seguidos                  │
│                                  │
│  🏆 Top 5 em 23 partidas         │
│  😬 Decepção em 2                │
└──────────────────────────────────┘
```

**Destaque:** "Seu Rachão Score é público. Organizadores de grupos fechados podem te encontrar pelo seu histórico."

---

### Seção 5 — Por que é grátis?
**Título:** "Jogar é sempre grátis."
**Subtítulo:** "Quem paga (opcionalmente) são os organizadores que precisam de recursos avançados. Para o jogador, tudo sempre gratuito."

| Para o jogador | |
|----------------|--|
| Encontrar rachões abertos | ✓ Grátis |
| Confirmar presença | ✓ Grátis |
| Votar no melhor da partida | ✓ Grátis |
| Ver Rachão Score | ✓ Grátis |
| Aparecer no Ranking | ✓ Grátis |
| Histórico completo de partidas | ✓ Grátis |

---

### Seção 6 — Como funciona (3 passos)
**Título:** "Em 3 passos você já tá no campo"

1. **Cadastre-se grátis** — WhatsApp + nome, pronto. Sem cartão.
2. **Encontre um rachão** — explore os grupos públicos com vagas abertas.
3. **Apareça e jogue** — confirme presença, chegue na hora e divirta-se.

---

### Seção 7 — CTA Final
**Fundo:** gradiente verde (mesmo padrão da LP atual)
**Título:** "Tá esperando o quê?"
**Subtítulo:** "Tem rachão acontecendo essa semana com vaga pra você."

**CTAs:**
- `Criar conta grátis` → `/register`
- `Ver rachões disponíveis` → `/discover`

---

## 6. Impacto técnico — pré-requisitos de D2 e D3

### D2 — Discover público

**Frontend:**
- Remover `/discover` da lista de rotas que redirecionam para login
- Adicionar `/discover` em `PUBLIC_ROUTES` no `+layout.svelte`
- Exibir `JoinCTABanner` para não logados (já existe)
- Remover ou adaptar funcionalidades que exigem auth (ex: botão "Quero participar" → redireciona para `/register?next=/discover`)

**Backend:**
- Verificar se `GET /api/v1/matches/discover` exige autenticação — se sim, torná-lo público ou criar variante pública sem dados sensíveis

---

### D3 — Rachão Score público

**Backend:**
- Novo endpoint público: `GET /api/v1/players/{id}/public-stats`
- Retorna: nome, nickname, avatar_url, skill_stars, total_matches, attendance_rate, current_streak, top5_count, flop_count
- Não retorna: whatsapp, email, dados sensíveis

**Frontend:**
- Nova rota pública: `/players/[id]` — card de perfil público com Rachão Score
- Adicionar `/players/` em `PUBLIC_ROUTES`
- Link compartilhável: "Ver meu Rachão Score" no perfil do usuário logado
- `JoinCTABanner` para não logados

---

## 7. Impacto técnico total estimado

| Camada | Trabalho |
|--------|---------|
| Migration | Nenhuma |
| Backend | 1 novo endpoint público de stats (`/players/{id}/public-stats`) |
| Frontend LP | Nova rota `/jogar` com mockups estáticos (LP atual `/lp` intocada) |
| Frontend Discover | Liberação de acesso público + adaptação de CTAs |
| Frontend Perfil | Nova rota `/players/[id]` pública + link de compartilhamento no perfil |
| SEO | Meta tags, schema.org, canonical, og:image |

A LP em si pode ser 100% estática (mockups visuais). Os pré-requisitos D2 e D3 exigem trabalho separado mas são independentes entre si e da LP.

---

## 8. Ordem de implementação sugerida

1. **D2** — Liberar `/discover` público (impacto imediato no funil, independe da LP)
2. **D3** — Criar `/players/[id]` público com Rachão Score
3. **LP** — Reescrever `/lp` com nova narrativa, agora com CTAs funcionais para D2 e D3

---

## 9. Fora de escopo (fase 2)

- Dados dinâmicos reais nos mockups da LP (ranking ao vivo, partidas reais)
- Filtro por cidade/bairro no hero
- Depoimentos de jogadores
- Integração com Google Maps
