-- Trim leading/trailing whitespace from player nicknames stored with spaces.
-- Converts nicknames that are only whitespace to NULL.
UPDATE players
SET nickname = NULLIF(TRIM(nickname), '')
WHERE nickname IS NOT NULL
  AND nickname <> TRIM(nickname);
