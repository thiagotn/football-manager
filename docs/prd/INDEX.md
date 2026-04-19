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

| # | PRD | Status | Notas |
|---|-----|--------|-------|
| 003 | [verificacao-whatsapp.md](003-verificacao-whatsapp.md) | ✅ | OTP via Twilio Verify (SMS). Canal WhatsApp pendente de aprovação Twilio/Meta |
| 025 | [otp-bypass-local.md](025-otp-bypass-local.md) | ✅ | `OTP_BYPASS_CODE` em `.env.docker` para dev local |
| 016 | [troca-de-senha-com-otp.md](016-troca-de-senha-com-otp.md) | ✅ | Troca de senha via OTP quando usuário não lembra a senha atual |
| 023 | [validacao-senha-igual.md](023-validacao-senha-igual.md) | ✅ | Backend 422 + feedback inline no frontend |
| 026 | [otp-leitura-automatica-sms.md](026-otp-leitura-automatica-sms.md) | ⏸ | `autocomplete="one-time-code"` ativo (iOS). WebOTP Android aguarda template Twilio aprovado. `$effect` revertido por interferência no foco dos inputs |

---

## Planos e Assinatura

| # | PRD | Status | Notas |
|---|-----|--------|-------|
| 010 | [planos-precificacao.md](010-planos-precificacao.md) | 📖 | Referência de precificação — preços confirmados: Basic R$19,90/mês · Pro R$39,90/mês |
| 001 | [planos-assinatura.md](001-planos-assinatura.md) | ✅ | Sistema completo de planos e assinatura via Stripe |
| 004 | [minha-conta-plano.md](004-minha-conta-plano.md) | ✅ | `/account/subscription` e `/plans` implementados |
| 014 | [admin-subscriptions.md](014-admin-subscriptions.md) | ✅ | Painel `/admin/subscriptions` — resumo, breakdown, tabela de assinantes, ativação manual |

---

## Perfil do Jogador

| # | PRD | Status | Notas |
|---|-----|--------|-------|
| 027 | [avatar-jogador.md](027-avatar-jogador.md) | 📋 | Upload de foto de perfil com fallback de iniciais e armazenamento no Supabase Storage |

---

## Ferramentas Públicas

| # | PRD | Status | Notas |
|---|-----|--------|-------|
| 033 | [simulador-sorteio-publico.md](033-simulador-sorteio-publico.md) | 📋 | Página pública `/draw` — sorteio de times com posições e estrelas, sem login, sem backend |

---

## Gestão de Membros

| # | PRD | Status | Notas |
|---|-----|--------|-------|
| 034 | [adicionar-jogador-manual.md](034-adicionar-jogador-manual.md) | ✅ | Admin adiciona jogador pelo WhatsApp — cria conta se necessário, sem link de convite |
| 035 | [posicao-jogador-grupo.md](035-posicao-jogador-grupo.md) | 📋 | Substituir flag "goleiro" por seletor de posição (GK/ZAG/LAT/MEI/ATA) em group_members |

---

## Partidas e Grupos

| # | PRD | Status | Notas |
|---|-----|--------|-------|
| 019 | [grupos-publicos-lista-de-espera.md](019-grupos-publicos-lista-de-espera.md) | ✅ | Grupos públicos, lista de espera e feed de descoberta |
| 000 | [recorrencia-heranca-convidados.md](000-recorrencia-heranca-convidados.md) | ✅ | Herança de convidados entre partidas recorrentes |
| 005 | [votacao-pos-partida.md](005-votacao-pos-partida.md) | ✅ | Votação pós-partida completa com aba de Estatísticas |
| 012 | [avaliacao-jogadores-e-montagem-de-times.md](012-avaliacao-jogadores-e-montagem-de-times.md) | ✅ | Notas por grupo, flag de goleiro, sorteio equilibrado e páginas públicas de times e resultados |
| 011 | [configuracao-votacao-por-grupo.md](011-configuracao-votacao-por-grupo.md) | ✅ | Delay configurável por grupo |
| 006 | [avaliacao-app.md](006-avaliacao-app.md) | ✅ | Votação pós-partida v1.5 com banner de pendências e estatísticas com filtro por período |
| 015 | [financeiro-grupo.md](015-financeiro-grupo.md) | ✅ | Controle financeiro por grupo (mensalidade, avulso, histórico) |
| 021 | [timezone-por-grupo.md](021-timezone-por-grupo.md) | ✅ | Timezone configurável por grupo com indicação visual ao usuário |
| 036 | [gols-e-assistencias.md](036-gols-e-assistencias.md) | 📋 | Registro de gols e assistências por partida — somente admin do grupo |
| 037 | [cores-de-coletes.md](037-cores-de-coletes.md) | 📋 | Cores de coletes e nomes customizados por time no grupo — até 5 slots por grupo |

---

## Internacionalização

| # | PRD | Status | Notas |
|---|-----|--------|-------|
| 024 | [auto-idioma-por-pais.md](024-auto-idioma-por-pais.md) | ✅ | Troca automática de idioma via seletor de país em `/login` e `/register` |
| 020 | [internacionalizacao-e-whatsapp-global.md](020-internacionalizacao-e-whatsapp-global.md) | ✅ | WhatsApp E.164 + i18n pt-BR/en/es completo (Paraglide) |

---

## Admin

| # | PRD | Status | Notas |
|---|-----|--------|-------|
| 007 | [admin-home.md](007-admin-home.md) | ✅ | Home `/admin` com big numbers, listagens globais de rachões e grupos |

---

## Infraestrutura e Plataforma

| # | PRD | Status | Notas |
|---|-----|--------|-------|
| 018 | [observabilidade.md](018-observabilidade.md) | ✅ | Grafana + Prometheus + Uptime Kuma + bot Telegram + alertas configurados |
| 002 | [push-notifications.md](002-push-notifications.md) | 🚧 | Infraestrutura (Web Push) implementada. Integração com eventos de negócio pendente |
| 028 | [instalacao-pwa-android.md](028-instalacao-pwa-android.md) | 📋 | Revisão do fluxo de instalação PWA no Android |
| 022 | [migracao-uuid-v7.md](022-migracao-uuid-v7.md) | 📋 | Migração de UUID v4 → v7 nas PKs |
| 013 | [minhas-estatisticas.md](013-minhas-estatisticas.md) | ✅ | Estatísticas por jogador |
| 029 | [ranking-geral.md](029-ranking-geral.md) | ✅ | Ranking geral da plataforma (top e flop) |
| 032 | [publicacao-lojas.md](032-publicacao-lojas.md) | 📋 | TWA Android (Fase 1 aprovada) + Flutter nativo (Fase 2 em avaliação). Inclui arquitetura do monorepo, workflow GitHub Actions e checklist completo |

---

## Marketing e Legal

| # | PRD | Status | Notas |
|---|-----|--------|-------|
| 017 | [seo-landing-page.md](017-seo-landing-page.md) | ✅ | SEO e meta tags da landing page |
| 008 | [politica-de-privacidade.md](008-politica-de-privacidade.md) | ✅ | Publicada em `/privacy` |
| 009 | [termos-de-uso.md](009-termos-de-uso.md) | ✅ | Publicados em `/terms`. Modal de aceite obrigatório implementado |
| 030 | [lp-jogadores-organico.md](030-lp-jogadores-organico.md) | 📋 | Landing page para aquisição orgânica de jogadores |
| 031 | [lp-unificada.md](031-lp-unificada.md) | ✅ | Landing page unificada para organizadores e jogadores |
