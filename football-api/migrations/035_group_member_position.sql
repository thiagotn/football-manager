-- 035_group_member_position.sql
-- Replace is_goalkeeper (bool) with position (varchar) in group_members.
-- Existing goalkeepers migrate to position='gk'; all others default to 'mei'.

-- 1. Add position column with default 'mei'
ALTER TABLE group_members
  ADD COLUMN IF NOT EXISTS position VARCHAR(3) NOT NULL DEFAULT 'mei';

-- 2. Migrate is_goalkeeper = true → position = 'gk'
UPDATE group_members
  SET position = 'gk'
  WHERE is_goalkeeper = true;

-- 3. Drop is_goalkeeper column
ALTER TABLE group_members
  DROP COLUMN IF EXISTS is_goalkeeper;
