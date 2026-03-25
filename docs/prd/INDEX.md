# Índice de PRDs — rachao.app

Referência rápida de todos os documentos de produto. Atualizar o status aqui ao iniciar ou concluir uma implementação.

## Legenda de status

| Símbolo | Significado |
|---------|-------------|
| ✅ | Implementado e em produção |
| 🚧 | Parcialmente implementado |
| ⏸ | Bloqueado por dependência externa |
| 📋 | Proposto — aguardando decisão ou priorização |
| 📖 | Documento de referência (não é uma feature) |

---

## Autenticação e Conta

| PRD | Status | Notas |
|-----|--------|-------|
| [verificacao-whatsapp.md](verificacao-whatsapp.md) | ✅ | OTP via Twilio Verify (SMS). Canal WhatsApp pendente de aprovação Twilio/Meta |
| [otp-bypass-local.md](otp-bypass-local.md) | ✅ | `OTP_BYPASS_CODE` em `.env.docker` para dev local |
| [troca-de-senha-com-otp.md](troca-de-senha-com-otp.md) | ✅ | Troca de senha via OTP quando usuário não lembra a senha atual |
| [validacao-senha-igual.md](validacao-senha-igual.md) | ✅ | Backend 422 + feedback inline no frontend |
| [otp-leitura-automatica-sms.md](otp-leitura-automatica-sms.md) | ⏸ | `autocomplete="one-time-code"` ativo (iOS). WebOTP Android aguarda template Twilio aprovado. `$effect` revertido por interferência no foco dos inputs |

---

## Planos e Assinatura

| PRD | Status | Notas |
|-----|--------|-------|
| [planos-precificacao.md](planos-precificacao.md) | 📖 | Referência de precificação — preços confirmados: Basic R$19,90/mês · Pro R$39,90/mês |
| [planos-assinatura.md](planos-assinatura.md) | ✅ | Sistema completo de planos e assinatura via Stripe |
| [minha-conta-plano.md](minha-conta-plano.md) | ✅ | `/account/subscription` e `/plans` implementados |
| [admin-subscriptions.md](admin-subscriptions.md) | ✅ | Painel `/admin/subscriptions` — resumo, breakdown, tabela de assinantes, ativação manual |

---

## Partidas e Grupos

| PRD | Status | Notas |
|-----|--------|-------|
| [grupos-publicos-lista-de-espera.md](grupos-publicos-lista-de-espera.md) | ✅ | Grupos públicos, lista de espera e feed de descoberta |
| [recorrencia-heranca-convidados.md](recorrencia-heranca-convidados.md) | ✅ | Herança de convidados entre partidas recorrentes |
| [votacao-pos-partida.md](votacao-pos-partida.md) | ✅ | Votação pós-partida completa com aba de Estatísticas |
| [avaliacao-jogadores-e-montagem-de-times.md](avaliacao-jogadores-e-montagem-de-times.md) | ✅ | Notas por grupo, flag de goleiro, sorteio equilibrado e páginas públicas de times e resultados |
| [configuracao-votacao-por-grupo.md](configuracao-votacao-por-grupo.md) | ✅ | Delay configurável por grupo |
| [avaliacao-app.md](avaliacao-app.md) | ✅ | Votação pós-partida v1.5 com banner de pendências e estatísticas com filtro por período |
| [financeiro-grupo.md](financeiro-grupo.md) | ✅ | Controle financeiro por grupo (mensalidade, avulso, histórico) |
| [timezone-por-grupo.md](timezone-por-grupo.md) | ✅ | Timezone configurável por grupo com indicação visual ao usuário |

---

## Internacionalização

| PRD | Status | Notas |
|-----|--------|-------|
| [auto-idioma-por-pais.md](auto-idioma-por-pais.md) | ✅ | Troca automática de idioma via seletor de país em `/login` e `/register` |
| [internacionalizacao-e-whatsapp-global.md](internacionalizacao-e-whatsapp-global.md) | ✅ | WhatsApp E.164 + i18n pt-BR/en/es completo (Paraglide) |

---

## Admin

| PRD | Status | Notas |
|-----|--------|-------|
| [admin-home.md](admin-home.md) | ✅ | Home `/admin` com big numbers, listagens globais de rachões e grupos |

---

## Infraestrutura e Plataforma

| PRD | Status | Notas |
|-----|--------|-------|
| [observabilidade.md](observabilidade.md) | ✅ | Grafana + Prometheus + Uptime Kuma + bot Telegram + alertas configurados |
| [push-notifications.md](push-notifications.md) | 🚧 | Infraestrutura (Web Push) implementada. Integração com eventos de negócio pendente |
| [instalacao-pwa-android.md](instalacao-pwa-android.md) | 📋 | Revisão do fluxo de instalação PWA no Android |
| [migracao-uuid-v7.md](migracao-uuid-v7.md) | 📋 | Migração de UUID v4 → v7 nas PKs |
| [minhas-estatisticas.md](minhas-estatisticas.md) | ✅ | Estatísticas por jogador |

---

## Marketing e Legal

| PRD | Status | Notas |
|-----|--------|-------|
| [seo-landing-page.md](seo-landing-page.md) | ✅ | SEO e meta tags da landing page |
| [politica-de-privacidade.md](politica-de-privacidade.md) | ✅ | Publicada em `/privacy` |
| [termos-de-uso.md](termos-de-uso.md) | ✅ | Publicados em `/terms`. Modal de aceite obrigatório implementado |
