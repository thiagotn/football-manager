# Backend — Estado atual

> Este arquivo documenta o **estado corrente** do backend: o que já existe e o que vem a seguir.
> Deve ser atualizado sempre que uma nova migration, router, repo, model, schema ou service for criado.
> Padrões de implementação estão em `CLAUDE.md` na raiz do projeto.

---

## Próxima migration

A última migration criada é `039_group_team_slots.sql`.

**A próxima deve ser numerada `040_`.**

> Sempre verificar com `ls migrations/` antes de criar uma nova, para não pular nem duplicar números.

---

## Routers existentes (`app/api/v1/routers/`)

| Arquivo | Domínio |
|---------|---------|
| `admin.py` | Painel super admin (stats, matches, groups, subscriptions) |
| `auth.py` | Login, cadastro, OTP, troca de senha |
| `finance.py` | Controle financeiro por grupo |
| `groups.py` | CRUD de grupos, membros, skill, waitlist, stats |
| `invites.py` | Convites por link |
| `matches.py` | Partidas, presenças, discover |
| `players.py` | CRUD de jogadores, estatísticas, upload/remoção de avatar (`PUT/DELETE /players/me/avatar`) |
| `push.py` | Web Push (VAPID) |
| `reviews.py` | Avaliações do app |
| `subscriptions.py` | Planos e assinaturas |
| `ranking.py` | Ranking geral da plataforma (público) |
| `beta.py` | Inscrição no beta Android — `POST /beta/android-signup` (público) |
| `teams.py` | Sorteio de times |
| `votes.py` | Votação pós-partida |
| `webhooks.py` | Webhooks do Stripe |

---

## Repositories existentes (`app/db/repositories/`)

`base`, `finance_repo`, `group_repo`, `group_stats_repo`, `invite_repo`, `match_repo`, `match_stats_repo`, `player_repo`, `player_stats_repo`, `ranking_repo`, `review_repo`, `subscription_repo`, `team_repo`, `vote_repo`, `waitlist_repo`

---

## Models existentes (`app/models/`)

`app_review`, `base`, `finance`, `group`, `invite`, `match`, `match_vote`, `player`, `push_subscription`, `subscription`, `team`, `user`, `waitlist`

---

## Schemas existentes (`app/schemas/`)

`admin`, `auth`, `finance`, `group`, `group_stats`, `invite`, `match`, `player`, `player_public`, `player_stats`, `ranking`, `review`, `subscription`, `team`, `vote`

---

## Services existentes (`app/services/`)

| Arquivo | Responsabilidade |
|---------|-----------------|
| `billing.py` | Cálculo de MRR, limites de plano |
| `billing_stripe.py` | Criação de checkout, webhook handlers |
| `push.py` | `send_push(db, player_id, title, body, url)` |
| `recurrence.py` | Geração de próxima partida recorrente |
| `storage.py` | Upload/remoção de avatares no Supabase Storage |
| `team_builder.py` | Algoritmo snake-draft de times equilibrados |
| `twilio_verify.py` | `send_otp(whatsapp)`, `check_otp(whatsapp, code)` — E.164 |
| `voting.py` | `voting_status(match)`, `voting_window(match)` — cálculo lazy |

---

## Como rodar os testes

```bash
docker compose run --rm api poetry run pytest tests/unit/ -q
```

Sempre rodar antes de commitar qualquer mudança no backend.
