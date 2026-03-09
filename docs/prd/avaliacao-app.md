# PRD — Avaliação do App
## Rachao.app · Gerenciamento de Grupos e Partidas

| | |
|---|---|
| **Versão** | 1.1 |
| **Status** | Pronto para implementar |
| **Data** | Março de 2026 |
| **Plataforma** | https://rachao.app |

---

## 1. Visão Geral

### 1.1 Contexto

A plataforma não possui nenhum mecanismo interno de coleta de feedback dos usuários. Qualquer insatisfação ou sugestão se perde — o super-admin não tem visibilidade sobre a percepção da base de usuários.

### 1.2 Objetivo

Permitir que usuários logados (exceto super-admins) avaliem o app com 1 a 5 estrelas e deixem um comentário opcional. As avaliações ficam acessíveis ao super-admin via painel dedicado no menu administrativo.

---

## 2. Regras de Negócio

- Apenas players com `role != 'admin'` (super-admin global) podem avaliar.
- Cada player pode submeter **uma avaliação por vez** — uma nova avaliação substitui a anterior.
- A nota (1–5 estrelas) é obrigatória. O comentário é opcional (máximo 500 caracteres).
- Avaliações não são públicas — visíveis apenas ao super-admin.
- O super-admin não pode avaliar o próprio app.

---

## 3. Requisitos Funcionais

**RF-01 — Acesso via menu**
Adicionar item "Avaliar o App" no menu principal da área logada, visível apenas para não super-admins. No `Navbar.svelte`, usar a flag `playerOnly: true` no array `links` (análoga à `adminOnly` já existente) para ocultar o item de super-admins.

**RF-02 — Formulário de avaliação**
Ao acessar `/review`, exibir formulário com:
- Botão X (fechar) no canto superior direito do card — navega de volta para `/`
- Seletor de 1 a 5 estrelas via `StarRating.svelte` (obrigatório)
- Campo de texto para comentário (opcional, máx. 500 caracteres) com contador visível
- Botão "Enviar avaliação"

Se o player já avaliou anteriormente, o formulário deve vir preenchido com a avaliação existente e o botão deve indicar "Atualizar avaliação".

**RF-03 — Submissão e atualização**
A avaliação é salva (upsert via `INSERT ... ON CONFLICT DO UPDATE`) ao confirmar. O player pode retornar e atualizar a qualquer momento.

**RF-04 — Painel administrativo**
Adicionar item "Avaliações" no menu do super-admin (`adminOnly: true`). A página `/admin/reviews` deve exibir:
- Nota média geral (1 casa decimal) com representação em estrelas
- Total de avaliações recebidas
- Distribuição por estrela (1★ a 5★) com percentual e barra visual
- Lista paginada de avaliações individuais, ordenada da mais recente para a mais antiga, contendo: nome do player, nota, comentário (se houver) e data de envio

**RF-05 — Filtros no painel admin**
Permitir filtrar a lista por nota (ex: mostrar apenas avaliações de 1 e 2 estrelas) e ordenar por data ou nota.

---

## 4. Requisitos Não Funcionais

**RNF-01** — A validação de elegibilidade (não super-admin) deve ocorrer no backend via dependency `AdminPlayer` para endpoints admin-only e verificação de `current.role` para o endpoint de submissão.

**RNF-02** — Comentários são texto puro (sem HTML). O backend deve rejeitar qualquer conteúdo com tags HTML usando uma verificação simples (ex: `bleach.clean` com `tags=[]` ou rejeitar se `<` estiver presente). Não é necessária nenhuma lib adicional — basta strip ou rejeição de `<` no campo.

**RNF-03** — A página do super-admin deve suportar paginação para não degradar com o crescimento da base.

---

## 5. Modelagem de Dados

```sql
-- Migration: 016_app_reviews.sql
-- (ou 017 se 016_otp_verifications.sql for implementado antes — usar o próximo número disponível)

CREATE TABLE app_reviews (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    player_id    UUID NOT NULL UNIQUE REFERENCES players(id) ON DELETE CASCADE,
    rating       SMALLINT NOT NULL CHECK (rating BETWEEN 1 AND 5),
    comment      TEXT CHECK (char_length(comment) <= 500),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_app_reviews_rating ON app_reviews (rating);
CREATE INDEX idx_app_reviews_created ON app_reviews (created_at DESC);
```

> `UNIQUE` em `player_id` garante no banco que existe no máximo uma avaliação por player, simplificando o upsert.

---

## 6. Endpoints da API

| Método | Endpoint | Descrição |
|---|---|---|
| `GET` | `/api/v1/reviews/me` | Retorna a avaliação do player logado (404 se ainda não avaliou) |
| `PUT` | `/api/v1/reviews/me` | Cria ou atualiza a avaliação do player logado |
| `GET` | `/api/v1/reviews` | Lista todas as avaliações — super-admin only |
| `GET` | `/api/v1/reviews/summary` | Resumo agregado (média, total, distribuição) — super-admin only |

### 6.1 `PUT /reviews/me`

**Request:**
```json
{
  "rating": 4,
  "comment": "Muito bom! Só falta poder editar os dados do jogador depois do cadastro."
}
```

**Response 200:**
```json
{
  "id": "uuid",
  "rating": 4,
  "comment": "Muito bom! Só falta poder editar os dados do jogador depois do cadastro.",
  "created_at": "2026-03-09T14:00:00-03:00",
  "updated_at": "2026-03-09T14:00:00-03:00"
}
```

**Erros:**
- `403 FORBIDDEN` — super-admin tentando avaliar

### 6.2 `GET /reviews/summary`

**Response 200:**
```json
{
  "average": 4.2,
  "total": 87,
  "distribution": {
    "5": { "count": 41, "percent": 47.1 },
    "4": { "count": 28, "percent": 32.2 },
    "3": { "count": 10, "percent": 11.5 },
    "2": { "count": 5,  "percent": 5.7  },
    "1": { "count": 3,  "percent": 3.4  }
  }
}
```

### 6.3 `GET /reviews`

Suporta query params: `?rating=1,2&order_by=created_at&page=1&page_size=20`

---

## 7. Interface do Usuário

### 7.1 Menu da área logada (não super-admin)

Adicionar no array `links` do `Navbar.svelte` com flag `playerOnly: true`:

```
⭐  Avaliar o App   →  /review
```

O link só é renderizado quando `!$isAdmin`. Tanto no menu desktop quanto no drawer mobile.

### 7.2 Página de avaliação (`/review`)

```
┌──────────────────────────────────────────┐
│  Como você avalia o Rachao?          [✕] │
│──────────────────────────────────────────│
│                                          │
│        ★  ★  ★  ★  ☆                    │
│      (toque para selecionar)             │
│                                          │
│  Comentário (opcional)                   │
│  ┌────────────────────────────────────┐  │
│  │ Conte o que achou, sugestões ou    │  │
│  │ o que poderia melhorar...          │  │
│  │                                    │  │
│  └────────────────────────────────────┘  │
│  0 / 500 caracteres                      │
│                                          │
│  [ Enviar avaliação ]                    │
└──────────────────────────────────────────┘
```

Se o player já avaliou:
- Estrelas e comentário preenchidos com a avaliação anterior
- Botão exibe "Atualizar avaliação"
- Texto abaixo: "Você avaliou em 08/03/2026 · pode atualizar a qualquer momento"

### 7.3 Menu do super-admin

Adicionar no array `links` do `Navbar.svelte` com flag `adminOnly: true`:

```
📊  Avaliações   →  /admin/reviews
```

### 7.4 Painel de avaliações (`/admin/reviews`)

```
┌──────────────────────────────────────────────────┐
│  Avaliações do App                               │
│──────────────────────────────────────────────────│
│                                                  │
│   4.2 ★★★★☆    87 avaliações                    │
│                                                  │
│   5★  ████████████████░░░  47%  (41)             │
│   4★  ███████████░░░░░░░░  32%  (28)             │
│   3★  ████░░░░░░░░░░░░░░░  12%  (10)             │
│   2★  ██░░░░░░░░░░░░░░░░░   6%  ( 5)             │
│   1★  █░░░░░░░░░░░░░░░░░░   3%  ( 3)             │
│                                                  │
│──────────────────────────────────────────────────│
│  Filtrar por nota: [Todas ▾]  Ordenar: [Mais recentes ▾]
│──────────────────────────────────────────────────│
│                                                  │
│  João Silva          ★★★★☆   09/03/2026          │
│  "Muito bom! Só falta poder editar os dados..."  │
│                                                  │
│  Pedro Alves         ★★★★★   08/03/2026          │
│  (sem comentário)                                │
│                                                  │
│  Lucas Ferreira      ★★☆☆☆   07/03/2026          │
│  "Trava muito no celular Android."               │
│                                                  │
│  [ < Anterior ]  Página 1 de 5  [ Próxima > ]    │
└──────────────────────────────────────────────────┘
```

---

## 8. Arquivos a Criar/Modificar

| Arquivo | Ação | Descrição |
|---|---|---|
| `football-api/migrations/016_app_reviews.sql` | Criar | Tabela `app_reviews` (ajustar número se 016 já estiver ocupado) |
| `football-api/app/models/app_review.py` | Criar | Model SQLAlchemy para `app_reviews` |
| `football-api/app/schemas/review.py` | Criar | Schemas Pydantic: `ReviewUpsertRequest`, `ReviewResponse`, `ReviewSummaryResponse`, `ReviewListResponse` |
| `football-api/app/db/repositories/review_repo.py` | Criar | Repositório: `upsert`, `get_by_player`, `list_all` (paginado + filtro), `get_summary` |
| `football-api/app/api/v1/routers/reviews.py` | Criar | Endpoints `/me` (GET/PUT) e admin (GET lista + summary). Usar `CurrentPlayer` e `AdminPlayer` de `app.core.dependencies` |
| `football-api/app/api/v1/router.py` | Modificar | Importar e registrar `reviews.router` (não `main.py`) |
| `football-frontend/src/routes/review/+page.svelte` | Criar | Página de avaliação do usuário |
| `football-frontend/src/routes/admin/reviews/+page.svelte` | Criar | Painel admin de avaliações |
| `football-frontend/src/lib/api.ts` | Modificar | Adicionar chamadas `reviews.getMe()`, `reviews.upsert()`, `reviews.list()`, `reviews.summary()` |
| `football-frontend/src/lib/components/StarRating.svelte` | Criar | Componente reutilizável de seleção/exibição de estrelas. Props: `rating` (bind), `readonly` (bool), `size` |
| `football-frontend/src/lib/components/Navbar.svelte` | Modificar | Adicionar flag `playerOnly` no array `links`; adicionar link "Avaliar o App" (`playerOnly`) e "Avaliações" (`adminOnly`); adicionar casos `/review` e `/admin/reviews` em `getBackHref` |

---

## 9. Decisões de Arquitetura

- **`router.py`, não `main.py`**: routers são registrados em `football-api/app/api/v1/router.py`. O `main.py` apenas inclui o `api_router` já montado.
- **`playerOnly` flag no Navbar**: analogia à `adminOnly` existente. Renderizar o link apenas quando `!$isAdmin`.
- **Sanitização de comentários**: rejeitar campo `comment` que contenha o caractere `<` no backend (validação Pydantic com `validator` ou `field_validator`). O campo é texto puro — sem suporte a markdown ou HTML.
- **Número da migration**: usar `016` se o PRD de OTP (`016_otp_verifications.sql`) ainda não tiver sido implementado; usar `017` caso contrário. Verificar a última migration em `football-api/migrations/` antes de criar o arquivo.
- **Fetch de dados**: via `$effect` no componente, seguindo o padrão de todas as outras páginas (token em `localStorage`, sem `+page.ts`).

---

## 10. Critérios de Aceitação

- [ ] Botão X no card de avaliação fecha e volta para `/`
- [ ] Item "Avaliar o App" aparece no menu apenas para não super-admins
- [ ] Super-admin não consegue submeter avaliação (erro 403 no backend)
- [ ] Nota de 1 a 5 estrelas é obrigatória — formulário não submete sem ela
- [ ] Comentário aceita até 500 caracteres com contador visível
- [ ] Segunda avaliação substitui a anterior (upsert), sem criar duplicata
- [ ] Formulário carrega preenchido se o player já avaliou anteriormente
- [ ] Item "Avaliações" aparece no menu apenas para super-admins
- [ ] Painel admin exibe média, total e distribuição corretamente
- [ ] Lista de avaliações é paginada e ordenada da mais recente para a mais antiga
- [ ] Filtro por nota funciona corretamente no painel admin
- [ ] Comentário com HTML (`<script>`, `<b>` etc.) é rejeitado com erro de validação

---

## 11. Dependências

- Tabela `players` — para validar `role` e exibir nome no painel admin
- `app/core/dependencies.py` — `CurrentPlayer` e `AdminPlayer`
- `Navbar.svelte` — para adicionar os novos links

---

## 12. Fora de Escopo (desta versão)

- Resposta do super-admin a uma avaliação
- Notificação ao super-admin quando uma avaliação negativa (1–2 estrelas) for recebida
- Exportação das avaliações (CSV)
- Exibição pública de avaliações (ex: landing page)
- Moderação ou ocultação de avaliações individuais

---

*Documento elaborado para uso interno da equipe de produto e engenharia do Rachao.app.*
