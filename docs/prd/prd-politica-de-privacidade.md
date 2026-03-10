# PRD — Política de Privacidade
## Rachao.app · Gerenciamento de Grupos e Partidas

| | |
|---|---|
| **Versão** | 1.0 |
| **Status** | ⚠️ Bloqueado — revisão jurídica pendente |
| **Data** | Março de 2026 |
| **Plataforma** | https://rachao.app |

---

## 1. Visão Geral

### 1.1 Contexto

O rachao.app está em fase Beta e trata dados pessoais de seus usuários: número de WhatsApp, nome, apelido, confirmações de presença, votos e logs de acesso. A Lei Geral de Proteção de Dados (LGPD — Lei nº 13.709/2018) exige que toda plataforma que trata dados pessoais de residentes no Brasil disponibilize uma Política de Privacidade clara, acessível e em conformidade com a lei.

### 1.2 Problema

- A plataforma não possui Política de Privacidade publicada.
- Sem este documento, a coleta de dados pessoais — incluindo número de WhatsApp — ocorre sem o devido respaldo legal de transparência exigido pela LGPD.
- A ausência expõe o produto a risco jurídico, especialmente à medida que a base de usuários cresce e os planos pagos são lançados.
- O texto do documento envolve decisões jurídicas (bases legais, prazos de retenção, transferência internacional de dados) que precisam ser validadas por advogado especializado em direito digital antes da publicação.

### 1.3 Objetivo

Publicar uma Política de Privacidade em conformidade com a LGPD, acessível em `/privacidade`, com link nas telas principais da plataforma e referência no Termo de Uso.

---

## 2. Pré-condição: Revisão Jurídica

> ⚠️ **Este PRD está bloqueado para implementação até que a revisão jurídica seja concluída.**

A Política de Privacidade envolve decisões com implicações legais que não devem ser tomadas unilateralmente pelo time de produto ou engenharia. Antes de iniciar o desenvolvimento, é necessário:

| # | Ação | Responsável | Status |
|---|---|---|---|
| 1 | Contratar ou consultar advogado especializado em LGPD / direito digital | Fundador | ⏳ Pendente |
| 2 | Validar as bases legais utilizadas para cada dado coletado (Art. 7º LGPD) | Jurídico | ⏳ Pendente |
| 3 | Confirmar prazos de retenção de dados adequados ao modelo de negócio | Jurídico | ⏳ Pendente |
| 4 | Avaliar necessidade de nomeação formal de Encarregado (DPO) | Jurídico | ⏳ Pendente |
| 5 | Validar cláusulas de transferência internacional (Twilio, Supabase) conforme Resolução ANPD nº 19/2024 | Jurídico | ⏳ Pendente |
| 6 | Definir e-mail oficial do canal de privacidade (ex.: privacidade@rachao.app) | Fundador | ⏳ Pendente |
| 7 | Confirmar dados do controlador (nome completo e CPF/CNPJ) | Fundador | ⏳ Pendente |
| 8 | Aprovar versão final do texto | Jurídico + Fundador | ⏳ Pendente |

**Estimativa de prazo após início da consulta jurídica:** 1–3 semanas.

---

## 3. Contexto para o Advogado

O texto de referência preparado pela equipe de produto está disponível em `docs/legal/termos-e-privacidade.md` (Parte II). Esse documento foi elaborado com base nos requisitos públicos da LGPD e serve como ponto de partida para a revisão — não como versão final.

### Dados tratados pela plataforma (resumo para a consulta)

| Dado | Como é coletado | Para quê é usado |
|---|---|---|
| Número de WhatsApp | Cadastro obrigatório | Login e envio de OTP de verificação |
| Nome e apelido | Cadastro | Identificação em grupos e partidas |
| Senha (hash bcrypt) | Cadastro | Autenticação |
| Confirmações de presença | Uso da plataforma | Gestão de partidas |
| Votos pós-partida | Uso da plataforma | Ranking e estatísticas do grupo |
| IP e logs de acesso | Geração automática | Segurança e diagnóstico |

### Terceiros que recebem dados

- **Twilio** (ou AWS SNS): recebe o número de WhatsApp para envio do OTP de verificação no cadastro. Servidor fora do Brasil.
- **Supabase**: banco de dados principal (PostgreSQL). Servidores podem estar fora do Brasil.
- **Gateway de pagamento** (Stripe ou Pagar.me — futuro): receberá dados de faturamento quando planos pagos forem lançados.

### Características do produto

- Produto Beta: dados podem ser resetados; funcionalidades podem mudar.
- Público-alvo: adultos brasileiros organizadores de grupos de futebol amador.
- Sem coleta de dados sensíveis (saúde, biometria, opinião política, etc.).
- Sem publicidade ou venda de dados a terceiros.

---

## 4. Escopo (após desbloqueio)

Quando a revisão jurídica for concluída e o texto aprovado, a implementação cobrirá:

- Criação da rota `/privacidade` com texto aprovado;
- Link para `/privacidade` no rodapé da landing page e do layout logado;
- Referência à Política de Privacidade no Termo de Uso (`/termos`);
- Referência à Política de Privacidade no formulário de cadastro (junto ao checkbox de aceite dos Termos);
- Canal de contato de privacidade funcional (e-mail monitorado).

**Fora de escopo:** sistema de gestão de consentimento (cookie banner), portal de exercício de direitos do titular (pode ser fase 2), relatório de impacto à proteção de dados (RIPD).

---

## 5. Requisitos Funcionais (após desbloqueio)

**RF-01 — Rota pública `/privacidade`**
Página pública com texto aprovado, sem dependência de login. Texto estruturado em seções com âncoras navegáveis. Acessível por motores de busca.

**RF-02 — Link no rodapé e telas chave**
O link "Política de Privacidade" deve aparecer em:
- Rodapé da landing page (`/lp`);
- Rodapé do layout principal (pós-login);
- Tela de cadastro (`/register`), junto ao checkbox de aceite dos Termos.

**RF-03 — Referência cruzada com Termos de Uso**
O Termo de Uso deve referenciar e linkar a Política de Privacidade. O texto de cadastro deve mencionar ambos os documentos.

**RF-04 — Canal de privacidade funcional**
O e-mail de contato indicado na Política deve estar ativo e monitorado antes da publicação.

**RF-05 — Versionamento**
O documento deve exibir data de última atualização e versão no topo. Futuras alterações relevantes devem gerar notificação aos usuários.

---

## 6. Arquivos a Criar / Modificar (após desbloqueio)

### Frontend
- `football-frontend/src/routes/privacidade/+page.svelte` — criar (texto estático aprovado)
- Layout da LP e layout logado — modificar (adicionar link no rodapé)
- `football-frontend/src/routes/register/+page.svelte` — modificar (mencionar Política junto ao aceite)
- `football-frontend/src/routes/termos/+page.svelte` — modificar (adicionar link para `/privacidade`)

### Nenhuma alteração de backend é necessária para esta feature.

---

## 7. Critérios de Aceitação (após desbloqueio)

- [ ] Revisão jurídica concluída e texto aprovado
- [ ] Controlador e e-mail de privacidade definidos e preenchidos no texto
- [ ] Página `/privacidade` acessível sem login com texto completo e formatado
- [ ] Link para `/privacidade` visível no rodapé da LP, rodapé logado e tela de cadastro
- [ ] Termos de Uso referenciam a Política de Privacidade com link
- [ ] E-mail de privacidade funcional e monitorado

---

## 8. Observações sobre o Período Beta

Durante o período Beta, recomenda-se incluir na Política de Privacidade — e validar com o advogado — um aviso explícito de que:

- O produto está em fase de testes;
- Dados podem ser resetados sem aviso prévio;
- Funcionalidades e práticas de tratamento de dados podem mudar antes da versão estável.

Esse aviso já está presente no banner de Beta da landing page, mas deve estar refletido formalmente no documento jurídico.

---

## 9. Riscos

| Risco | Impacto | Mitigação |
|---|---|---|
| Publicar texto sem revisão jurídica | Alto — responsabilidade legal por informações incorretas | Manter PRD bloqueado até revisão concluída |
| Atraso na consulta jurídica | Médio — LGPD já está em vigor e a plataforma já trata dados | Priorizar a consulta; considerar advogados especializados em startups/LGPD que oferecem revisões pontuais |
| Custo da consulta jurídica | Baixo | Revisão pontual de um documento de 1–2 páginas tem custo acessível (R$ 500–2.000 estimados) |
| Mudança de terceiros (ex.: troca de gateway) | Baixo | Atualizar a Política sempre que um novo parceiro for adicionado |

---

*Documento elaborado para uso interno da equipe de produto e engenharia do Rachao.app.*
