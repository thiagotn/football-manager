# Reset para Produção — rachao.app

Guia para zerar todos os dados de teste da plataforma, preservando apenas o super admin.

---

## O que o script faz

O script `scripts/reset_to_production.sql`:

1. Identifica o super admin — o `player` com `role = 'admin'` mais antigo (`created_at ASC`)
2. Remove **todos os grupos** e seus dados dependentes via CASCADE:
   - `group_members`, `matches`, `attendances`
   - `match_votes`, `match_vote_top5`, `match_vote_flop`
   - `match_teams`, `match_team_players`
   - `invite_tokens`, `finance_periods`, `finance_payments`
3. Remove `app_reviews`, `webhook_events`
4. Remove `push_subscriptions` e `player_subscriptions` de todos exceto o super admin
5. Remove todos os `players` exceto o super admin
6. **Reseta as sequences** para que novos registros comecem do zero:
   - `matches_number_seq → 1` — próximo rachão criado será o **#1**
   - `push_subscriptions_id_seq → 1`

Tudo ocorre dentro de uma **transação única** — se qualquer passo falhar, nada é alterado.

---

## ⚠️ Antes de executar

- **Faça um dump de backup** antes de rodar o script em produção
- Confirme quem é o super admin: `SELECT id, name, whatsapp, created_at FROM players WHERE role = 'admin' ORDER BY created_at ASC LIMIT 1;`
- O script é **irreversível** após o `COMMIT`

---

## Opção 1 — Painel do Supabase (mais simples)

1. Acesse [supabase.com](https://supabase.com) → seu projeto → **SQL Editor**
2. Clique em **New query**
3. Cole o conteúdo de `scripts/reset_to_production.sql`
4. Clique em **Run** (ou `Ctrl+Enter`)
5. Verifique as mensagens `NOTICE` no painel de resultados — procure pela linha `✅ Reset concluído com sucesso.`

---

## Opção 2 — Via VPS (usando psql)

### Pré-requisito
A variável `DATABASE_URL` deve estar disponível no ambiente, ou substitua manualmente abaixo.

```bash
# No servidor VPS, acesse o diretório do projeto
cd /root/football-manager   # ajuste o caminho se necessário

# Execute o script diretamente
psql "$DATABASE_URL" -f scripts/reset_to_production.sql
```

Se preferir passar a URL explicitamente:

```bash
psql "postgresql://usuario:senha@host:5432/database?sslmode=require" \
     -f scripts/reset_to_production.sql
```

Para ver as mensagens NOTICE no terminal, certifique-se de que o psql não está suprimindo outputs (`-q` não deve estar ativo).

---

## Opção 3 — Via Docker local (banco local de desenvolvimento)

```bash
# Com o stack rodando (docker compose up)
cd football-api

# A flag -T desabilita a alocação de TTY, necessária ao redirecionar stdin
docker compose exec -T postgres psql -U postgres -d football \
  < ../scripts/reset_to_production.sql
```

Ou copie o arquivo para dentro do container e execute diretamente:

```bash
docker cp ../scripts/reset_to_production.sql \
  $(docker compose ps -q postgres):/tmp/reset.sql

docker compose exec postgres \
  psql -U postgres -d football -f /tmp/reset.sql
```

---

## Verificação pós-execução

Após o script, confirme que apenas o super admin foi mantido:

```sql
-- Deve retornar 1 linha (o super admin)
SELECT id, name, whatsapp, role, created_at FROM players;

-- Devem retornar 0 linhas
SELECT COUNT(*) FROM groups;
SELECT COUNT(*) FROM matches;
SELECT COUNT(*) FROM app_reviews;
```

---

## Dúvidas

- O script identifica o super admin pelo campo `role = 'admin'` com `created_at` mais antigo
- Se houver mais de um admin, apenas o mais antigo é preservado — os outros são removidos junto com os demais players
- O script **não altera** a senha nem nenhum dado do super admin preservado
