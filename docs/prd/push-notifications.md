# PRD — Push Notifications

**Status:** Não implementado
**Prioridade:** Média
**Stack:** pywebpush (backend) + Web Push API + Service Worker (frontend)
**Custo:** Zero (Web Push Protocol direto, sem serviço intermediário)

---

## Objetivo

Permitir que jogadores recebam notificações em tempo real sobre eventos relevantes do app (convites, novas partidas, lembretes), mesmo com o app fechado, via PWA já instalada.

---

## Casos de uso

| Evento | Destinatário |
|--------|-------------|
| Convite para grupo recebido | Jogador convidado |
| Nova partida aberta no grupo | Todos os membros do grupo |
| Lembrete X horas antes da partida | Jogadores confirmados |
| Partida encerrada / resultado disponível | Jogadores participantes |
| Remoção do grupo | Jogador removido |

---

## Requisitos funcionais

### Backend

- [ ] Gerar par de chaves VAPID (uma vez, armazenar em variáveis de ambiente `VAPID_PRIVATE_KEY`, `VAPID_PUBLIC_KEY`, `VAPID_CLAIMS_EMAIL`)
- [ ] Instalar `pywebpush`
- [ ] Criar tabela `push_subscriptions` com colunas: `id`, `player_id` (FK), `endpoint`, `p256dh`, `auth`, `user_agent`, `created_at`
- [ ] `POST /push/subscribe` — salva ou atualiza subscrição do jogador autenticado
- [ ] `DELETE /push/subscribe` — remove subscrição (opt-out)
- [ ] `GET /push/vapid-public-key` — retorna chave pública para o frontend
- [ ] Função utilitária `send_push(player_id, title, body, url)` — busca subscrições do jogador e envia via pywebpush; trata erros 410/404 (subscrição expirada) removendo o registro
- [ ] Integrar `send_push` nos eventos: criação de convite, criação de partida, lembrete de partida (via scheduler já existente)

### Frontend

- [ ] Buscar VAPID public key do backend na inicialização
- [ ] Solicitar permissão de notificação ao usuário (fluxo opt-in explícito, com explicação prévia do que será enviado)
- [ ] Subscrever via `pushManager.subscribe()` e enviar objeto de subscrição ao `POST /push/subscribe`
- [ ] Adicionar handler `push` no Service Worker para exibir a notificação com `showNotification()`
- [ ] Adicionar handler `notificationclick` no Service Worker para abrir a URL correta ao clicar
- [ ] Botão de opt-out em `/profile` para cancelar notificações

---

## Requisitos não funcionais

- Compatibilidade: Chrome/Android (todas as versões modernas), Safari/iOS **≥ 16.4 com PWA instalada**
- Falhas de envio (endpoint expirado) devem ser tratadas silenciosamente — remover subscrição inválida sem quebrar o fluxo
- Não bloquear nenhuma requisição existente: envios de push devem ser assíncronos (fire-and-forget ou background task)

---

## Fora do escopo (desta versão)

- Notificações de chat/mensagens em tempo real
- Segmentação ou agendamento avançado de campanhas
- Fila de mensagens (Celery/Redis) — desnecessário na escala atual
- Suporte a browsers sem Service Worker

---

## Arquitetura resumida

```
[Evento no backend] → send_push(player_id, ...)
    → pywebpush → Push Service do browser (FCM / APNs)
        → browser acorda o Service Worker
            → SW exibe showNotification()
                → usuário clica → SW abre URL no app
```

---

## Dependências

- PWA já implementada (Service Worker ativo via vite-plugin-pwa) ✅
- Sistema de autenticação JWT ✅
- Scheduler (APScheduler) para lembretes ✅
- `pywebpush` — adicionar ao `requirements.txt`

---

## Migrações necessárias

```sql
-- push_subscriptions
CREATE TABLE push_subscriptions (
    id SERIAL PRIMARY KEY,
    player_id INTEGER NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    endpoint TEXT NOT NULL,
    p256dh TEXT NOT NULL,
    auth TEXT NOT NULL,
    user_agent TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(player_id, endpoint)
);
```
