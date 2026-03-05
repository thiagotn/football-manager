-- Migration 011: Add recurrence_enabled to groups
ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS recurrence_enabled BOOLEAN NOT NULL DEFAULT FALSE;
