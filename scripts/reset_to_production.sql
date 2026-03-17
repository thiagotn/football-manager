-- ============================================================
-- reset_to_production.sql
-- Zera todos os dados de teste da plataforma, preservando
-- apenas o super admin (player com role='admin' mais antigo).
--
-- ⚠️  ATENÇÃO: operação IRREVERSÍVEL. Faça um backup antes.
-- ⚠️  Execute SOMENTE em ambiente de produção após validação.
-- ============================================================

BEGIN;

DO $$
DECLARE
  v_admin_id   UUID;
  v_admin_name VARCHAR;
BEGIN

  -- 1. Localiza o super admin (primeiro admin cadastrado)
  SELECT id, name
    INTO v_admin_id, v_admin_name
    FROM players
   WHERE role = 'admin'
   ORDER BY created_at ASC
   LIMIT 1;

  IF v_admin_id IS NULL THEN
    RAISE EXCEPTION 'Super admin não encontrado — abortando sem alterações.';
  END IF;

  RAISE NOTICE '>> Super admin identificado: % (id: %)', v_admin_name, v_admin_id;

  -- 2. Remove grupos e tudo que depende deles via CASCADE:
  --    group_members, matches, attendances, match_votes,
  --    match_vote_top5, match_vote_flop, match_teams,
  --    match_team_players, invite_tokens,
  --    finance_periods, finance_payments
  DELETE FROM groups;
  RAISE NOTICE '>> groups e dependências removidos.';

  -- 3. Avaliações de app (independente de grupos)
  DELETE FROM app_reviews;
  RAISE NOTICE '>> app_reviews removido.';

  -- 4. Eventos de webhook (independente de players/grupos)
  DELETE FROM webhook_events;
  RAISE NOTICE '>> webhook_events removido.';

  -- 5. Push subscriptions — remove tudo exceto do super admin
  DELETE FROM push_subscriptions WHERE player_id != v_admin_id;
  RAISE NOTICE '>> push_subscriptions limpo.';

  -- 6. Assinaturas — remove tudo exceto do super admin
  DELETE FROM player_subscriptions WHERE player_id != v_admin_id;
  RAISE NOTICE '>> player_subscriptions limpo.';

  -- 7. Players — remove todos exceto o super admin
  DELETE FROM players WHERE id != v_admin_id;
  RAISE NOTICE '>> players limpo. Super admin preservado.';

  -- 8. Reseta sequences para que novos registros comecem do 1
  --    matches_number_seq: rachão #1 ao criar a primeira partida
  ALTER SEQUENCE matches_number_seq RESTART WITH 1;
  RAISE NOTICE '>> matches_number_seq resetada para 1.';

  --    push_subscriptions_id_seq: id serial da tabela push_subscriptions
  ALTER SEQUENCE push_subscriptions_id_seq RESTART WITH 1;
  RAISE NOTICE '>> push_subscriptions_id_seq resetada para 1.';

  RAISE NOTICE '';
  RAISE NOTICE '✅  Reset concluído com sucesso.';
  RAISE NOTICE '   Super admin mantido: % (id: %)', v_admin_name, v_admin_id;

END $$;

COMMIT;
