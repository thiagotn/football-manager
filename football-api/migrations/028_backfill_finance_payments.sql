-- Migration 028: backfill missing finance_payments for existing members
--
-- Members added to a group after the current month's finance_period was
-- already created were never inserted into finance_payments.
-- This migration inserts the missing rows idempotently using ON CONFLICT DO NOTHING.

INSERT INTO finance_payments (id, period_id, player_id, player_name, status)
SELECT
    gen_random_uuid(),
    fp.id                                   AS period_id,
    gm.player_id,
    COALESCE(p.nickname, p.name)            AS player_name,
    'pending'                               AS status
FROM finance_periods fp
JOIN group_members gm ON gm.group_id = fp.group_id
JOIN players p ON p.id = gm.player_id
WHERE p.role != 'admin'
  AND NOT EXISTS (
      SELECT 1
      FROM finance_payments pay
      WHERE pay.period_id = fp.id
        AND pay.player_id = gm.player_id
  )
ON CONFLICT (period_id, player_id) DO NOTHING;
