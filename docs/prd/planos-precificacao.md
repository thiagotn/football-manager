# Análise de Precificação — Planos de Assinatura
## Rachao.app · Documento Complementar ao PRD de Planos

| | |
|---|---|
| **Versão** | 1.0 |
| **Status** | Referência · Preços a confirmar antes do lançamento |
| **Data** | Março de 2026 |
| **Relacionado** | `docs/prd/planos-assinatura.md` |

---

## 1. Estimativa de Custos Mensais de Operação

| Item | Estimativa BRL/mês |
|---|---|
| VPS (ex: Hetzner CX22 ou equivalente) | ~R$80–150 |
| Domínio | ~R$4 (R$50/ano) |
| E-mail transacional (Resend/Mailgun — free tier → pago) | R$0–50 |
| Backups + monitoramento | ~R$20–40 |
| Stripe Billing (gratuito até ~R$11k/mês processados) | R$0 inicialmente |
| **Total fixo estimado** | **~R$150–250/mês** |

> Para cobrir infraestrutura, são necessários aproximadamente **10–15 assinantes no plano Básico**.

---

## 2. Preços Recomendados

| Plano | Mensal | Anual | Equivalente mensal no anual |
|---|---|---|---|
| **Free** | Grátis | Grátis | — |
| **Básico** | **R$19,90** | **R$191,90** | R$15,99 (~20% de economia) |
| **Pro** | **R$39,90** | **R$383,90** | R$31,99 (~20% de economia) |

> Estes valores não estão confirmados. Devem ser validados com estudo de mercado e definidos no Stripe antes do lançamento. Consulte a seção 14.3 do PRD para criação dos Price IDs.

---

## 3. Justificativa por Plano

### 3.1 Básico — R$19,90/mês

- **Relação custo-benefício clara para o organizador:** um rachão típico com 20 jogadores a R$25 cada movimenta R$500 por partida. Pagar R$20/mês pela ferramenta de gestão representa menos de 4% de uma única pelada — custo imperceptível para quem vê valor no produto.
- **Abaixo do limiar de decisão:** no mercado brasileiro, produtos de R$19,90 costumam ser adquiridos sem muita deliberação. Acima de R$29,90 o processo de decisão se torna mais consciente e exige justificativa.
- **Alinhado ao mercado de SaaS simples no Brasil:** ferramentas de gestão para micro e pequenas operações costumam precificar entre R$19–49/mês.

### 3.2 Pro — R$39,90/mês

- Quem gerencia 10+ grupos é um organizador profissional ou opera uma quadra/futsal comercial. Para esse perfil, R$40/mês é custo operacional irrelevante.
- Mantém proporção ~2× em relação ao Básico, o que é psicologicamente razoável e sinaliza diferença clara de valor.

---

## 4. Impacto das Taxas do Stripe

| Situação | Cobrança bruta | Taxa Stripe* | Receita líquida |
|---|---|---|---|
| Básico mensal | R$19,90 | ~R$1,18 | ~R$18,72 |
| Básico anual | R$191,90 | ~R$7,02 | ~R$184,88 |
| Pro mensal | R$39,90 | ~R$1,86 | ~R$38,04 |
| Pro anual | R$383,90 | ~R$13,55 | ~R$370,35 |

*Estimativa: 3,4% + R$0,50 por transação (cartão nacional). PIX: 1,5%.

> **Planos anuais são significativamente melhores para fluxo de caixa:** menor número de transações, menor custo fixo por transação e recebimento antecipado do valor integral.

---

## 5. Cenários de Sustentabilidade

| Cenário | Composição | Receita bruta/mês |
|---|---|---|
| Break-even de infraestrutura | 10 assinantes Básico | ~R$199 |
| Side income inicial | 30 Básico + 5 Pro | ~R$797 |
| Side income razoável | 50 Básico + 10 Pro | ~R$1.394 |
| Side income confortável | 100 Básico + 20 Pro | ~R$2.788 |

### Estimativa de conversão

A taxa de conversão Free → Pago típica em produtos B2C/SMB é de **3–8%**. Com uma base de 500 usuários Free, espera-se entre 15 e 40 conversões — o suficiente para cobrir custos operacionais confortavelmente.

---

## 6. Diretrizes Estratégicas

### 6.1 Priorizar o plano anual

Incentivar o plano anual com destaque visual na página de planos (`/plans`) e exibição explícita da economia (ex: "Economize R$46,90 por ano"). Planos anuais reduzem churn, melhoram o fluxo de caixa e diminuem o custo proporcional das taxas do gateway.

### 6.2 Não sub-precificar por receio de conversão

R$19,90 não é caro para o público-alvo. Organizar um rachão exige esforço; quem já adotou uma ferramenta para isso tende a pagar pelo valor gerado. Sub-precificar sinaliza baixa qualidade percebida e dificulta reajustes futuros.

### 6.3 Não aumentar preços prematuramente

Com base ainda pequena, preços altos reduzem conversão sem compensar em receita. Preços podem ser ajustados para cima quando houver tração comprovada — usuários existentes devem ser notificados com antecedência e, preferencialmente, mantidos no preço anterior por um período de carência.

### 6.4 Definir os Price IDs no Stripe apenas quando pronto para lançar

Price IDs de produção no Stripe são imutáveis. Criar os produtos e preços certos da primeira vez evita inconsistências entre ambiente de teste e produção. Consulte o checklist da seção 14 do PRD.

---

## 7. Quando Revisar a Precificação

| Gatilho | Ação sugerida |
|---|---|
| Taxa de conversão abaixo de 2% com base > 300 usuários Free | Investigar objeções; considerar reduzir preço do Básico ou oferecer trial |
| Taxa de conversão acima de 15% | Produto claramente sub-precificado; avaliar reajuste |
| Churn mensal acima de 10% nos planos pagos | Investigar causas antes de mexer em preço |
| MRR acima de R$5.000 de forma consistente | Reavaliar custo de infraestrutura e capacidade de suporte |
| Volume processado acima de R$30k/mês | Avaliar migração de gateway (ver seção 9.3 do PRD) |

---

*Documento elaborado como complemento ao PRD de Planos de Assinatura do Rachao.app. Os valores aqui sugeridos são referência e devem ser validados antes da implementação.*
