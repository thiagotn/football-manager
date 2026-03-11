-- Migration: 020_group_member_skill.sql
-- Adiciona nota de habilidade (1-5 estrelas) e flag de goleiro em group_members.
-- Jogadores existentes recebem nota padrão 2 (via DEFAULT).

ALTER TABLE group_members
  ADD COLUMN skill_stars   SMALLINT NOT NULL DEFAULT 2,
  ADD COLUMN is_goalkeeper BOOLEAN  NOT NULL DEFAULT FALSE;

ALTER TABLE group_members
  ADD CONSTRAINT chk_skill_stars
    CHECK (skill_stars >= 1 AND skill_stars <= 5);
