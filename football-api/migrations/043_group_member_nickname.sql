-- Migration 043: add per-group nickname to group_members
ALTER TABLE group_members ADD COLUMN IF NOT EXISTS nickname VARCHAR(50);
