-- Migration 029: make match number sequential per group instead of global
--
-- Before: matches_number_seq is a single global sequence shared by all groups.
-- After:  number is unique per (group_id, number) and restarts from 1 per group.
--
-- Steps:
--   1. Drop the old global unique index on number
--   2. Recalculate every match's number using ROW_NUMBER() per group (ordered by created_at)
--   3. Add the new unique constraint on (group_id, number)
--   4. Drop the global sequence (no longer used — number is set by the application)

-- 1. Drop old global unique index
DROP INDEX IF EXISTS idx_matches_number;

-- 2. Recalculate all existing match numbers per group
WITH numbered AS (
    SELECT
        id,
        ROW_NUMBER() OVER (PARTITION BY group_id ORDER BY created_at ASC) AS new_number
    FROM matches
)
UPDATE matches
SET number = numbered.new_number
FROM numbered
WHERE matches.id = numbered.id;

-- 3. Add unique constraint per group (idempotent)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'uq_matches_group_number'
    ) THEN
        ALTER TABLE matches
            ADD CONSTRAINT uq_matches_group_number UNIQUE (group_id, number);
    END IF;
END $$;

-- 4. Drop the now-unused global sequence
DROP SEQUENCE IF EXISTS matches_number_seq;
