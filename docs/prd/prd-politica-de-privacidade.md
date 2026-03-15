# PRD — Política de Privacidade
## Rachao.app · Gerenciamento de Grupos e Partidas

| | |
|---|---|
| **Versão** | 1.2 |
| **Status** | Publicado |
| **Data** | Março de 2026 |
| **Plataforma** | https://rachao.app |

---

## 1. Visão Geral

### 1.1 Contexto

O rachao.app trata dados pessoais de seus usuários: número de WhatsApp, nome, apelido, confirmações de presença, votos e logs de acesso. A LGPD (Lei nº 13.709/2018) exige que toda plataforma que trata dados pessoais de residentes no Brasil disponibilize uma Política de Privacidade clara e acessível.

### 1.2 Objetivo

Publicar uma Política de Privacidade mínima e honesta em `/privacy`, que:
- Descreve exatamente o que a plataforma coleta e para quê
- Informa com quem os dados são compartilhados
- Declara os direitos dos titulares e como exercê-los
- Não exige revisão jurídica prévia para entrar no ar — a melhor proteção legal é ter um documento honesto publicado, não ter nenhum

> **Nota:** revisão jurídica por advogado especializado em LGPD é recomendada periodicamente. A fase Beta foi encerrada; o produto está em operação com planos pagos disponíveis.

---

## 2. Pendências antes de publicar

| # | Ação | Responsável |
|---|---|---|
| 1 | Definir nome completo e CPF/CNPJ do controlador | Fundador |
| 2 | Criar e-mail `privacidade@rachao.app` (ou outro) e garantir que está monitorado | Fundador |
| 3 | Confirmar provedor de banco de dados (Supabase) e região dos servidores | Fundador |
| 4 | Confirmar provedor de SMS/OTP (Twilio, AWS SNS ou outro) | Fundador |

Preencher os `[PLACEHOLDERS]` no texto da seção 3 antes de publicar.

---

## 3. Texto da Política de Privacidade

> Este é o texto a ser publicado em `/privacy`. Copiar tal qual para o componente Svelte, substituindo os placeholders.

---

### Política de Privacidade — rachao.app

**Última atualização:** março de 2026 · **Versão:** 1.1

---

#### 1. Quem somos

O rachao.app é uma plataforma de organização de grupos e partidas de futebol amador, operada por **[NOME COMPLETO DO RESPONSÁVEL], [CPF ou CNPJ]** ("nós", "nosso" ou "controlador"), com canal de contato em **[privacidade@rachao.app]**.

A plataforma está em **fase Beta**: funcionalidades podem mudar e dados podem ser resetados sem aviso prévio durante esse período.

---

#### 2. Quais dados coletamos e para quê

| Dado | Finalidade | Base legal (LGPD, Art. 7º) |
|---|---|---|
| Número de WhatsApp | Identificação da conta, envio de código de verificação no cadastro | Execução de contrato |
| Nome e apelido | Identificação em grupos e partidas | Execução de contrato |
| Senha (armazenada como hash bcrypt — nunca em texto claro) | Autenticação | Execução de contrato |
| Confirmações de presença em partidas | Gestão de partidas e listas de presença | Execução de contrato |
| Votos pós-partida e estatísticas | Ranking e histórico do grupo | Execução de contrato |
| IP e logs de acesso | Segurança, diagnóstico de falhas e prevenção de abusos | Legítimo interesse |

Não coletamos dados sensíveis (saúde, biometria, opinião política, etnia, religião). Não vendemos dados a terceiros. Não utilizamos os dados para publicidade.

---

#### 3. Com quem compartilhamos os dados

| Parceiro | Dado compartilhado | Motivo |
|---|---|---|
| **[PROVEDOR DE SMS/OTP — ex: Twilio]** | Número de WhatsApp | Envio do código de verificação no cadastro |
| **[PROVEDOR DE BANCO DE DADOS — ex: Supabase]** | Todos os dados da plataforma | Armazenamento e operação do serviço |

Ambos os provedores operam como **operadores de dados** sob as nossas instruções e possuem políticas de privacidade e segurança próprias. Servidores podem estar localizados fora do Brasil; ao usar a plataforma, você consente com essa transferência internacional nos termos do Art. 33 da LGPD.

Quando os planos pagos forem lançados, um gateway de pagamento (Stripe ou equivalente) também será adicionado — a política será atualizada nesse momento.

---

#### 4. Por quanto tempo guardamos seus dados

| Dado | Prazo de retenção |
|---|---|
| Dados de conta (nome, WhatsApp, senha) | Enquanto a conta estiver ativa; excluídos em até 30 dias após solicitação de exclusão |
| Histórico de partidas, presenças e votos | Enquanto o grupo ou a conta existir |
| Logs de acesso e IP | 90 dias |

Durante o período Beta, podemos resetar dados da plataforma inteira com aviso mínimo de 48 horas pelo canal de comunicação do produto.

---

#### 5. Seus direitos como titular

De acordo com o Art. 18 da LGPD, você tem direito a:

- **Confirmar** se tratamos seus dados
- **Acessar** os dados que temos sobre você
- **Corrigir** dados incompletos, inexatos ou desatualizados
- **Solicitar a exclusão** dos seus dados (sujeito a obrigações legais de retenção)
- **Revogar o consentimento** a qualquer momento, sem prejuízo da licitude dos tratamentos realizados antes
- **Portabilidade** dos seus dados em formato estruturado (a implementar)
- **Reclamar** à Autoridade Nacional de Proteção de Dados (ANPD) caso considere que seus direitos foram violados

Para exercer qualquer direito, envie um e-mail para **[privacidade@rachao.app]** com assunto "Direitos LGPD" e seu número de WhatsApp cadastrado para identificação. Respondemos em até **15 dias úteis**.

---

#### 6. Segurança

Adotamos as seguintes medidas de segurança:

- Senhas armazenadas exclusivamente como hash bcrypt (nunca em texto claro)
- Comunicação via HTTPS em todo o tráfego
- Acesso ao banco de dados restrito por variáveis de ambiente e credenciais de serviço
- Tokens JWT com expiração configurada

Nenhum sistema é 100% seguro. Em caso de incidente de segurança que afete seus dados, notificaremos os usuários afetados e a ANPD conforme exigido pela lei.

---

#### 7. Cookies e armazenamento local

O rachao.app não utiliza cookies de rastreamento ou publicidade. Utilizamos **localStorage** do navegador exclusivamente para armazenar o token de autenticação da sessão. Esses dados ficam apenas no seu dispositivo e são removidos ao fazer logout.

---

#### 8. Alterações nesta política

Quando realizarmos alterações relevantes nesta política, atualizaremos a data no topo deste documento. Para mudanças que ampliem significativamente o uso dos seus dados, notificaremos os usuários ativos pelo canal de comunicação da plataforma com antecedência mínima de 15 dias.

---

#### 9. Contato

**Controlador:** [NOME COMPLETO], [CPF/CNPJ]
**E-mail de privacidade:** [privacidade@rachao.app]
**Plataforma:** https://rachao.app

---

## 4. Escopo de implementação

- [ ] Preencher todos os `[PLACEHOLDERS]` da seção 3 com dados reais
- [ ] Criar rota pública `/privacy` em `football-frontend/src/routes/privacidade/+page.svelte`
- [ ] Adicionar link "Política de Privacidade" no rodapé da landing page (`/lp`)
- [ ] Adicionar link "Política de Privacidade" no rodapé do layout logado
- [ ] Atualizar `/register` — mencionar a Política junto ao texto de aceite dos Termos
- [ ] Atualizar `/terms` — adicionar link para `/privacy`
- [ ] Garantir que o e-mail de privacidade está ativo e monitorado antes de publicar

---

## 5. Arquivos a criar / modificar

| Arquivo | Ação |
|---|---|
| `football-frontend/src/routes/privacidade/+page.svelte` | Criar — texto estático da seção 3 |
| Layout da LP | Modificar — link no rodapé |
| Layout logado | Modificar — link no rodapé |
| `football-frontend/src/routes/register/+page.svelte` | Modificar — referência à Política no aceite |
| `football-frontend/src/routes/termos/+page.svelte` | Modificar — link para `/privacy` |

Nenhuma alteração de backend é necessária.

---

## 6. Recomendação pós-Beta

Antes do lançamento dos planos pagos, contratar revisão jurídica pontual (advogado especializado em LGPD / startups) para:

- Validar as bases legais declaradas
- Confirmar os prazos de retenção
- Avaliar necessidade formal de DPO
- Adaptar cláusulas de transferência internacional à Resolução ANPD nº 19/2024
- Incluir referência ao gateway de pagamento quando ativo

Custo estimado de revisão pontual: R$ 500–2.000.

---

## 7. Riscos

| Risco | Impacto | Mitigação |
|---|---|---|
| Publicar com placeholders não preenchidos | Alto — texto incompleto expõe o produto | Cheklist da seção 4 antes do deploy |
| Dados do controlador incorretos | Médio — invalida o canal de exercício de direitos | Confirmar CPF/CNPJ antes de publicar |
| Mudança de terceiro (ex.: troca de provedor de banco ou SMS) | Baixo | Atualizar a seção 3 sempre que um parceiro mudar |
| Lançar planos pagos sem revisão jurídica | Médio | Contratar revisão antes do início da cobrança |

---

*Documento elaborado para uso interno da equipe de produto e engenharia do Rachao.app.*
