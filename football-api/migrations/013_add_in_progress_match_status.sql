-- Add 'in_progress' value to match_status enum
ALTER TYPE match_status ADD VALUE IF NOT EXISTS 'in_progress';
