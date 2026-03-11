-- Migration: 018_group_voting_config.sql
-- Adiciona configuração de votação por grupo

ALTER TABLE groups
  ADD COLUMN vote_open_delay_minutes INT NOT NULL DEFAULT 20,
  ADD COLUMN vote_duration_hours     INT NOT NULL DEFAULT 24;

ALTER TABLE groups
  ADD CONSTRAINT chk_vote_open_delay
    CHECK (vote_open_delay_minutes >= 0 AND vote_open_delay_minutes <= 120);

ALTER TABLE groups
  ADD CONSTRAINT chk_vote_duration
    CHECK (vote_duration_hours >= 2 AND vote_duration_hours <= 72);
