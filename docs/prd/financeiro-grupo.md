# PRD — Financeiro do Grupo
## Rachao.app · Gerenciamento de Grupos e Partidas

| | |
|---|---|
| **Versão** | 1.2 |
| **Status** | ✅ Implementado — Março 2026 |
| **Data** | Março de 2026 |
| **Plataforma** | https://rachao.app |

---

## 1. Visão Geral

### 1.1 Contexto

Administradores de grupos de rachão gerenciam hoje, fora da plataforma, o controle de pagamentos dos jogadores — seja uma mensalidade fixa ou uma taxa por partida. Esse controle é feito via planilha (Excel/Google Sheets), WhatsApp ou anotação manual, o que gera:

- Esquecimento e inconsistência nos registros
- Falta de visibilidade consolidada do grupo
- Constrangimento ao cobrar jogadores sem histórico claro
- Retrabalho mensal recriando a mesma planilha

### 1.2 Objetivo

Implementar no rachao.app uma seção de **Financeiro** por grupo, onde todos os membros podem visualizar a situação de pagamentos do grupo e o admin registra quem pagou mês a mês.

### 1.3 O que NÃO é este recurso

Este módulo **não é** um gateway de pagamento. Não processa cobranças, não integra com Pix, boleto ou cartão. É um **registro manual de controle financeiro** — o admin marca quem pagou. O dinheiro continua trafegando fora da plataforma.

---

## 2. Tipos de Cobrança

Cada grupo pode operar em um dos três modelos:

| Tipo | Descrição | Exemplo |
|---|---|---|
| **Mensalidade** | Valor fixo por membro por mês, independente de presença | R$ 50/mês por jogador |
| **Por partida** | Valor fixo por partida que o jogador participou | R$ 15 por jogo |
| **Híbrido** | Mensalistas fixos + pagantes por partida (convidados/avulsos) | Titulares mensalistas, convidados pagam por jogo |

---

## 3. Fluxo Principal

```
Admin acessa o grupo → aba "Financeiro"
       ↓
Configura o modelo de cobrança (primeira vez)
  - Tipo: mensalidade / por partida / híbrido
  - Valor padrão por membro
  - Dia de vencimento (para mensalidade)
       ↓
Qualquer membro visualiza o mês atual
  - Lista de todos os membros com status: ✅ Pago | ⏳ Pendente
  - Resumo: total esperado / recebido / pendente
       ↓
Admin marca pagamentos (toggle simples)
  - Toque no membro → alterna entre Pago e Pendente
       ↓
Navegação por meses anteriores (histórico)
```

---

## 4. Requisitos Funcionais

**RF-01 — Configuração financeira do grupo**
O admin configura o modelo de cobrança do grupo: tipo (mensalidade, por partida, híbrido), valor padrão e dia de vencimento. A configuração é por grupo e pode ser editada a qualquer momento.

**RF-02 — Criação de período mensal**
O sistema mantém registros por mês/ano. Um período é criado automaticamente ao acessar o mês atual caso ainda não exista. Ao criar um período, todos os membros ativos do grupo são incluídos com status "Pendente".

**RF-03 — Registro de pagamento**
Ao marcar um membro como pago, o admin informa **apenas o tipo do pagamento**: Mensal ou Avulso. O valor (`amount_due`) é preenchido automaticamente a partir da configuração do grupo (`monthly_amount` ou `per_match_amount`). Não há campos de forma, data ou observação na v1.

Para reverter, o admin toca novamente no membro pago e confirma a ação — o registro retorna a "Pendente".

**RF-04 — Visibilidade para todos os membros**
Todos os membros do grupo podem visualizar a tela de Financeiro — incluindo o status de pagamento de cada membro. Apenas o admin pode alterar o status.

**RF-05 — Resumo financeiro do período**
O topo da tela exibe cards com:
- **Recebido**: soma dos `amount_due` dos membros com status `paid`
- **Pendente**: soma dos `amount_due` dos membros com status `pending`
- **Total esperado**: recebido + pendente
- **Adimplência**: percentual de membros que pagaram

> O "total esperado" não é fixo — ele cresce conforme os membros são marcados, pois o `amount_due` só é definido no momento do pagamento (mensal ou avulso). Membros ainda pendentes têm `amount_due = null` até serem marcados.

**RF-06 — Histórico de períodos**
Todos os membros podem navegar entre meses anteriores para consultar o histórico.

**RF-07 — Valor individual por membro**
O admin pode definir um valor diferente do padrão para um membro específico em um período (ex: jogador que só participou de metade do mês).

**RF-08 — Membros por partida (modo "por partida" e "híbrido")**
No modo por partida, o `amount_due` é calculado com base nas partidas confirmadas do membro no período, cruzando com a tabela de presenças. O admin pode ajustar manualmente.

**RF-09 — Exclusão de membro do período**
O admin pode remover um membro de um período específico (ex: jogador ausente o mês todo que não deve pagar).

---

## 5. Requisitos Não Funcionais

**RNF-01 — Permissões**
- **Visualização**: todos os membros do grupo
- **Edição** (toggle pago/pendente, configuração, exclusão): apenas o admin do grupo

**RNF-02 — Limites por plano**

| Funcionalidade | Grátis | Básico | Pro |
|---|:---:|:---:|:---:|
| Financeiro habilitado | ✅ | ✅ | ✅ |
| Períodos no histórico | 3 meses | 12 meses | Ilimitado |
| Exportar CSV | ❌ | ✅ | ✅ |

> O financeiro é habilitado em todos os planos para não criar barreira de adoção. O histórico estendido e a exportação são o diferencial dos planos pagos.

**RNF-03 — Integridade dos dados**
Deletar um membro do grupo não apaga seu histórico financeiro — os registros ficam marcados com o nome e indicação "membro removido".

**RNF-04 — Performance**
A listagem de um período deve carregar em menos de 500ms mesmo em grupos com 50+ membros.

---

## 6. Modelagem de Dados

```sql
-- Migration: NNN_group_finance.sql

-- Configuração financeira do grupo
CREATE TABLE group_finance_configs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id        UUID NOT NULL UNIQUE REFERENCES groups(id) ON DELETE CASCADE,
    payment_type    VARCHAR(20) NOT NULL DEFAULT 'monthly',
                    -- 'monthly' | 'per_match' | 'hybrid'
    default_amount  INT NOT NULL DEFAULT 0,  -- centavos
    due_day         SMALLINT,                -- dia do mês (1–28), null se per_match
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Período financeiro (mês/ano de um grupo)
CREATE TABLE finance_periods (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id    UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    year        SMALLINT NOT NULL,
    month       SMALLINT NOT NULL,  -- 1–12
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (group_id, year, month)
);

CREATE INDEX idx_finance_periods_group ON finance_periods (group_id, year DESC, month DESC);

-- Configuração financeira do grupo (atualizada)
-- Passa a ter dois valores distintos: mensal e avulso
ALTER TABLE group_finance_configs ADD COLUMN monthly_amount  INT NOT NULL DEFAULT 0;
ALTER TABLE group_finance_configs ADD COLUMN per_match_amount INT NOT NULL DEFAULT 0;
-- default_amount é mantido como fallback/legado

-- Registro de pagamento por membro por período
CREATE TABLE finance_payments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    period_id       UUID NOT NULL REFERENCES finance_periods(id) ON DELETE CASCADE,
    player_id       UUID NOT NULL REFERENCES players(id),
    player_name     VARCHAR(100) NOT NULL,   -- snapshot do nome (RNF-03)
    payment_type    VARCHAR(20),             -- 'monthly' | 'per_match' | null se pending
    amount_due      INT,                     -- centavos; null enquanto pendente, preenchido ao marcar pago
    status          VARCHAR(20) NOT NULL DEFAULT 'pending',
                    -- 'pending' | 'paid' | 'excluded'
    paid_at         TIMESTAMPTZ,             -- preenchido automaticamente ao marcar pago
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (period_id, player_id)
);

CREATE INDEX idx_finance_payments_period ON finance_payments (period_id);
```

> **Nota v1:** `amount_due` e `payment_type` são definidos **no momento em que o admin marca como pago**, com base nos valores da config do grupo. Enquanto `status = 'pending'`, ambos ficam `null`. Os campos `payment_method`, `amount_paid` e `notes` foram intencionalmente omitidos — poderão ser adicionados em v2 sem breaking change.

---

## 7. Endpoints da API

| Método | Endpoint | Descrição |
|---|---|---|
| `GET` | `/api/v1/groups/{id}/finance/config` | Retorna configuração financeira do grupo (inclui `monthly_amount` e `per_match_amount`) |
| `PUT` | `/api/v1/groups/{id}/finance/config` | Cria ou atualiza configuração (admin) |
| `GET` | `/api/v1/groups/{id}/finance/periods` | Lista períodos existentes |
| `POST` | `/api/v1/groups/{id}/finance/periods` | Cria novo período (admin) |
| `GET` | `/api/v1/groups/{id}/finance/periods/{year}/{month}` | Retorna período com pagamentos |
| `PATCH` | `/api/v1/finance/payments/{id}` | Alterna status pago/pendente (admin) |
| `DELETE` | `/api/v1/finance/payments/{id}` | Exclui membro do período (admin) |

### 7.1 `GET /groups/{id}/finance/periods/{year}/{month}`

**Response 200:**
```json
{
  "period_id": "uuid",
  "year": 2026,
  "month": 3,
  "summary": {
    "expected_cents": 150000,
    "received_cents": 90000,
    "pending_cents": 60000,
    "paid_count": 6,
    "pending_count": 4,
    "compliance_pct": 60
  },
  "payments": [
    {
      "id": "uuid",
      "player_id": "uuid",
      "player_name": "João Silva",
      "payment_type": "monthly",
      "amount_due": 5000,
      "status": "paid",
      "paid_at": "2026-03-10T14:32:00Z"
    },
    {
      "id": "uuid",
      "player_id": "uuid",
      "player_name": "Carlos Souza",
      "payment_type": null,
      "amount_due": null,
      "status": "pending",
      "paid_at": null
    }
  ]
}
```

### 7.2 `PATCH /finance/payments/{id}` (admin)

**Marcar como pago** — requer `payment_type`:
```json
{ "status": "paid", "payment_type": "monthly" }
```
ou
```json
{ "status": "paid", "payment_type": "per_match" }
```

**Reverter para pendente:**
```json
{ "status": "pending" }
```

O backend:
- Ao marcar `paid`: preenche `paid_at = NOW()` e busca o `amount_due` na config do grupo (`monthly_amount` ou `per_match_amount` conforme o `payment_type` informado)
- Ao marcar `pending`: limpa `paid_at`, `payment_type` e `amount_due`

---

## 8. Interface do Usuário

### 8.1 Acesso
Nova aba **"Financeiro"** na página do grupo (`/groups/{id}`), visível para **todos os membros**.

### 8.2 Tela principal — Período atual

```
┌──────────────────────────────────────────────────────┐
│  Financeiro — Rachão da Quinta       Março 2026  ← → │
├──────────────────────────────────────────────────────┤
│  ┌──────────┐  ┌──────────┐  ┌──────────┐            │
│  │ Esperado │  │ Recebido │  │ Pendente │            │
│  │ R$ 500   │  │ R$ 300   │  │ R$ 200   │            │
│  └──────────┘  └──────────┘  └──────────┘            │
│                              Adimplência: 60%         │
├──────────────────────────────────────────────────────┤
│  PENDENTE (4)                                         │
│  ⏳ Carlos Souza        R$ 50                         │
│  ⏳ Marcos Lima         R$ 50                         │
│                                                       │
│  PAGO (6)                                             │
│  ✅ João Silva          R$ 50                         │
│  ✅ Pedro Alves         R$ 50                         │
└──────────────────────────────────────────────────────┘
```

- **Membros**: todos visualizam. Admin vê botão de toggle ao lado de cada nome.
- **Marcar pago (admin)**: toque no membro pendente abre um **mini bottom sheet** com dois botões:
  - `Mensal — R$ XX,XX`
  - `Avulso — R$ XX,XX`
  Os valores são carregados da config do grupo. Um toque confirma e fecha.
- **Reverter (admin)**: toque no membro pago exibe confirmação simples ("Desfazer pagamento?").
- **Ordenação**: pendentes primeiro, depois pagos — ambos em ordem alfabética.

### 8.3 Configuração financeira (admin)

Acessível via botão "Configurar" no topo. Campos:
- Valor da mensalidade (R$)
- Valor do avulso / por partida (R$)
- Dia de vencimento (opcional, informativo)

---

## 9. Limites de Plano e Upsell

O upsell ocorre ao tentar navegar para períodos além do limite do plano:

> **Grátis → Básico:** "Histórico disponível nos últimos 3 meses no plano Grátis. Assine o Básico para acessar 12 meses completos."

---

## 10. Fora de Escopo (v1)

- Processamento de pagamentos (Pix, boleto, cartão)
- Detalhes do pagamento: forma, observação, valor pago diferente do esperado — **v2**
- Notificações automáticas por WhatsApp/SMS para inadimplentes — **v2**
- Relatórios avançados / gráficos de evolução — **v2**
- Exportação CSV — **v2**
- Caixa do grupo (controle de despesas, como aluguel da quadra) — **v2**
- Multi-admin — **v2**

---

## 11. Critérios de Aceitação

- [x] Admin configura o modelo de cobrança do grupo
- [x] Período do mês atual é criado automaticamente ao acessar o Financeiro
- [x] Todos os membros ativos aparecem no período com status "Pendente"
- [x] Admin seleciona o tipo (Mensal ou Avulso) ao marcar um membro como pago
- [x] `amount_due` é preenchido automaticamente com o valor da config do grupo conforme o tipo selecionado
- [x] `paid_at` é preenchido automaticamente ao marcar como pago e limpo ao reverter
- [x] Resumo considera apenas membros com `status = paid` no cálculo do "Recebido"
- [x] Todos os membros do grupo visualizam a tela (somente admin edita)
- [x] Resumo (esperado / recebido / pendente / adimplência) é calculado corretamente
- [x] Admin e membros navegam entre meses anteriores
- [x] Membro removido do grupo mantém histórico com nome preservado
- [x] Limite de histórico por plano é respeitado com mensagem de upsell

---

## 12. Considerações de Implementação

### Backend
- `GET /groups/{id}/finance/periods/{year}/{month}` cria o período automaticamente (upsert) se não existir para o mês atual, populando `finance_payments` com todos os membros ativos e `status = 'pending'`. Para meses passados, retorna 404 se não existir.
- `amount_due` e `payment_type` ficam `null` enquanto `status = 'pending'`. São preenchidos apenas ao chamar `PATCH` com `status: "paid"`.
- Ao receber `PATCH` com `status: "paid"`, o backend lê `group_finance_configs` para obter o valor correto (`monthly_amount` ou `per_match_amount`) conforme o `payment_type` informado.
- O `player_name` é snapshot no momento da criação do período — preserva o nome mesmo após saída do grupo.
- `PATCH /finance/payments/{id}` verifica que o `current_player` é admin do grupo associado ao período.

### Frontend
- A aba "Financeiro" é visível para todos os membros; os controles de edição (toggle, configurar) são renderizados condicionalmente por `is_admin`.
- Toggle usa otimistic update: altera o ícone imediatamente e reverte em caso de erro.
- Navegação entre meses usa setas `← →`. Ao tentar ir além do limite do plano, exibe upsell modal.

---

*Documento elaborado para uso interno da equipe de produto e engenharia do Rachao.app.*
