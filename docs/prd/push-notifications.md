# PRD — Push Notifications

**Status:** Infraestrutura implementada — integração com eventos pendente
**Prioridade:** Média
**Stack:** pywebpush (backend) + Web Push API + Service Worker (frontend)
**Custo:** Zero (Web Push Protocol direto, sem serviço intermediário)

---

## Objetivo

Permitir que jogadores recebam notificações em tempo real sobre eventos relevantes do app (convites, novas partidas, lembretes), mesmo com o app fechado, via PWA já instalada.

---

## Casos de uso

| Evento | Destinatário | Implementado |
|--------|-------------|:---:|
| Convite para grupo recebido | Jogador convidado | ⏳ |
| Nova partida aberta no grupo | Todos os membros do grupo | ⏳ |
| Lembrete X horas antes da partida | Jogadores confirmados | ⏳ |
| Partida encerrada / resultado disponível | Jogadores participantes | ⏳ |
| Remoção do grupo | Jogador removido | ⏳ |

> ⏳ = infraestrutura pronta; chamar `send_push()` no local adequado para ativar.

---

## Requisitos funcionais

### Backend

- [x] Gerar par de chaves VAPID (uma vez, armazenar em `VAPID_PRIVATE_KEY`, `VAPID_PUBLIC_KEY`, `VAPID_CLAIMS_EMAIL` — via `npx web-push generate-vapid-keys`)
- [x] `pywebpush = "^2.0.0"` adicionado ao `pyproject.toml`
- [x] Migração `014_push_subscriptions.sql` — tabela `push_subscriptions` (`id`, `player_id` UUID FK, `endpoint`, `p256dh`, `auth`, `user_agent`, `created_at`)
- [x] Model `app/models/push_subscription.py`
- [x] `GET /api/v1/push/vapid-public-key` — retorna chave pública para o frontend
- [x] `POST /api/v1/push/subscribe` — salva ou atualiza subscrição do jogador autenticado (upsert por endpoint)
- [x] `DELETE /api/v1/push/subscribe` — remove todas as subscrições do jogador (opt-out)
- [x] `app/services/push.py` — função `send_push(db, player_id, title, body, url)`: busca subscrições, envia via pywebpush em thread executor, remove automaticamente subscrições expiradas (404/410); no-op se VAPID não configurado
- [ ] Integrar `send_push` nos eventos: criação de convite, criação de partida, lembrete (via APScheduler)

### Frontend

- [x] `lib/api.ts` — objeto `push` com `getVapidPublicKey`, `subscribe`, `unsubscribe`
- [x] `service-worker.ts` — handler `push` exibe `showNotification()`; handler `notificationclick` abre a URL correta
- [x] `/profile` — seção "Notificações": detecta suporte, solicita permissão, subscreve via `pushManager.subscribe()`, botão de opt-out; exibe aviso se permissão bloqueada no navegador

---

## Requisitos não funcionais

- Compatibilidade: Chrome/Android (todas as versões modernas), Safari/iOS **≥ 16.4 com PWA instalada**
- Falhas de envio (endpoint expirado) tratadas silenciosamente — subscrição inválida removida sem quebrar o fluxo
- Envios assíncronos via `asyncio.to_thread` — não bloqueiam nenhuma requisição existente

---

## Configuração de ambiente

### Gerar chaves VAPID (executar uma vez)
```bash
npx web-push generate-vapid-keys
```

### Variáveis de ambiente (`football-api/.env` local / GitHub Secrets em produção)
```env
VAPID_PRIVATE_KEY=<gerado acima>
VAPID_PUBLIC_KEY=<gerado acima>
VAPID_CLAIMS_EMAIL=seuemail@gmail.com
```

### CI/CD
- Secrets no GitHub Actions: `VAPID_PRIVATE_KEY`, `VAPID_PUBLIC_KEY`, `VAPID_CLAIMS_EMAIL`
- O workflow `deploy.yml` injeta/atualiza automaticamente esses valores no `.env.prod` do VPS a cada deploy
- O `docker-compose.prod.yml` passa as vars ao container da API via `environment`

---

## Fora do escopo (desta versão)

- Notificações de chat/mensagens em tempo real
- Segmentação ou agendamento avançado de campanhas
- Fila de mensagens (Celery/Redis) — desnecessário na escala atual
- Suporte a browsers sem Service Worker

---

## Arquitetura resumida

```
[Evento no backend] → send_push(db, player_id, title, body, url)
    → pywebpush (thread executor) → Push Service do browser (FCM / APNs)
        → browser acorda o Service Worker
            → SW exibe showNotification()
                → usuário clica → SW abre URL no app
```

---

## Arquivos relevantes

| Arquivo | Descrição |
|---------|-----------|
| `football-api/migrations/014_push_subscriptions.sql` | Migration da tabela |
| `football-api/app/models/push_subscription.py` | Model SQLAlchemy |
| `football-api/app/services/push.py` | Utilitário `send_push` |
| `football-api/app/api/v1/routers/push.py` | Endpoints REST |
| `football-api/app/core/config.py` | Campos VAPID nas settings |
| `football-frontend/src/service-worker.ts` | Handlers push/notificationclick |
| `football-frontend/src/lib/api.ts` | Cliente push API |
| `football-frontend/src/routes/profile/+page.svelte` | UI opt-in/opt-out |
| `.github/workflows/deploy.yml` | Injeção de secrets VAPID no VPS |
| `football-api/docker-compose.prod.yml` | Vars VAPID no container API |
